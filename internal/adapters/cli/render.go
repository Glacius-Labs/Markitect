package cli

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// The renderers write Markdown, so that every output reads in an editor,
// Obsidian or GitHub.

func renderRefs(target string, refs []analysis.Ref) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# References to [[%s]]\n\n", target)
	if len(refs) == 0 {
		b.WriteString("Nothing links to it and no file realizes it.\n")
		return b.String()
	}
	b.WriteString("| From | Via | Where |\n|---|---|---|\n")
	for _, r := range refs {
		from := "[[" + r.From + "]]"
		if r.Via == "registry" {
			from = "`" + r.From + "`"
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", from, r.Via, where(r.File, r.Line))
	}
	return b.String()
}

func renderGraph(g analysis.Graph) string {
	var b strings.Builder
	b.WriteString("# Package graph\n\nArrows are explicit dependencies; every Package also depends on its parent by location.\n\n```mermaid\nflowchart TD\n")
	ids := map[string]string{}
	for i, n := range g.Packages {
		ids[n.Path] = fmt.Sprintf("p%d", i)
		fmt.Fprintf(&b, "  %s[\"%s\"]\n", ids[n.Path], model.PackageName(n.Path))
	}
	for _, n := range g.Packages {
		if n.Path != model.RootPackage {
			fmt.Fprintf(&b, "  %s -.- %s\n", ids[n.Parent], ids[n.Path])
		}
	}
	for _, e := range g.Edges {
		fmt.Fprintf(&b, "  %s --> %s\n", ids[e.From], ids[e.To])
	}
	b.WriteString("```\n\n")
	b.WriteString(packageTable(g))
	if len(g.Edges) > 0 {
		b.WriteString("\n## Dependencies\n\n| From | To | Because |\n|---|---|---|\n")
		for _, e := range g.Edges {
			var because []string
			for _, u := range e.Uses {
				verb := "links"
				if u.Kind {
					verb = "is of Kind"
				}
				because = append(because, fmt.Sprintf("[[%s]] %s [[%s]]", u.From, verb, u.Target))
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", model.PackageName(e.From), model.PackageName(e.To), strings.Join(because, "; "))
		}
	}
	return b.String()
}

func packageTable(g analysis.Graph) string {
	var b strings.Builder
	b.WriteString("| Package | Definitions | Ca | Ce | Instability |\n|---|---|---|---|---|\n")
	for _, n := range g.Packages {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %.2f |\n", model.PackageName(n.Path), n.Definitions, n.Afferent, n.Efferent, n.Instability)
	}
	return b.String()
}

func renderLint(warnings model.Diagnostics, g analysis.Graph) string {
	var b strings.Builder
	b.WriteString("# Lint\n\n")
	if len(warnings) == 0 {
		b.WriteString("No warnings.\n")
	} else {
		fmt.Fprintf(&b, "%d warnings:\n\n", len(warnings))
		for _, w := range warnings {
			fmt.Fprintf(&b, "- %s\n", w.String())
		}
	}
	b.WriteString("\n## Coupling\n\nCa: Packages that depend on it. Ce: Packages it depends on. Instability: Ce / (Ca + Ce).\n\n")
	b.WriteString(packageTable(g))
	return b.String()
}

func renderImpact(base string, im analysis.Impact) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact since %s\n\n", short(base))
	if im.Full {
		b.WriteString("Full: every Definition is reached and every Manager works.\n\n")
	}
	if len(im.Packages) == 0 {
		b.WriteString("Nothing changed; nothing to do.\n")
		return b.String()
	}
	if len(im.Definitions) > 0 {
		b.WriteString("## Changed Definitions\n\n")
		for _, c := range im.Definitions {
			fmt.Fprintf(&b, "- [[%s]] %s\n", c.Path, c.Change)
		}
		b.WriteString("\n")
	}
	if len(im.Files) > 0 {
		b.WriteString("## Changed files\n\n")
		for _, c := range im.Files {
			fmt.Fprintf(&b, "- `%s` %s\n", c.Path, c.Change)
		}
		b.WriteString("\n")
	}
	if len(im.ReachedFiles) > 0 {
		b.WriteString("## Reached files\n\n| File | Owner | Why |\n|---|---|---|\n")
		for _, f := range im.ReachedFiles {
			var why []string
			if f.Change != "" {
				why = append(why, f.Change)
			}
			for _, id := range f.For {
				why = append(why, "realizes [["+id+"]]")
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", f.File, model.PackageName(f.Owner), strings.Join(why, ", "))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Checklists\n\n")
	for _, c := range im.Checklists {
		fmt.Fprintf(&b, "### %s\n\n", model.PackageName(c.Package))
		if c.Empty() {
			b.WriteString("No items of its own; it reviews and integrates its children.\n\n")
			continue
		}
		items(&b, "Own", c.Own)
		items(&b, "Inherited", c.Inherited)
		if len(c.Findings) > 0 {
			b.WriteString("Findings:\n\n")
			for _, f := range c.Findings {
				fmt.Fprintf(&b, "- [[%s]]: observed %s; expected %s\n", f.Definition, f.Observed, f.Expected)
			}
			b.WriteString("\n")
		}
		if len(c.Stale) > 0 {
			b.WriteString("Stale realization to clean up:\n\n")
			for _, s := range c.Stale {
				fmt.Fprintf(&b, "- %s\n", staleText(s.Code, s.File, s.Target, where(s.PageFile, s.Line)))
			}
			b.WriteString("\n")
		}
	}
	if len(im.Notes) > 0 {
		b.WriteString("## Side notes\n\n")
		for _, n := range im.Notes {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}
	return b.String()
}

func items(b *strings.Builder, title string, list []analysis.Item) {
	if len(list) == 0 {
		return
	}
	fmt.Fprintf(b, "%s:\n\n", title)
	for _, it := range list {
		fmt.Fprintf(b, "- [[%s]]: %s\n", it.Definition, strings.Join(it.Reasons, "; "))
	}
	b.WriteString("\n")
}

func staleText(code, file, target, at string) string {
	switch {
	case code == "stale.definition":
		return fmt.Sprintf("`%s` links [[%s]], which no longer exists (%s)", file, target, at)
	case code == "stale.file":
		return fmt.Sprintf("`%s` no longer exists (%s)", file, at)
	case file == "":
		return fmt.Sprintf("the page %s belongs to a removed Package", at)
	default:
		return fmt.Sprintf("`%s` is listed only on the page of a removed Package (%s)", file, at)
	}
}

func where(file string, line int) string {
	if line == 0 {
		return file
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
