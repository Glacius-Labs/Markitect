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

func where(file string, line int) string {
	if line == 0 {
		return file
	}
	return fmt.Sprintf("%s:%d", file, line)
}
