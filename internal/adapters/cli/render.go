package cli

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// The renderers write plain text, as Go's tools print theirs; --json gives the
// same as one document.

// renderList prints each Package, then its Definitions indented below it; with
// --deps a Package's coupling, abstractness and distance and what it depends on,
// and each Definition's links. With packagesOnly it prints the Packages alone,
// a dependency as one line with the number of links behind it, so that the
// structure of a large spec fits a screen.
func renderList(pkgs []analysis.ListedPackage, packagesOnly bool) string {
	var b strings.Builder
	for _, p := range pkgs {
		b.WriteString(p.Name)
		if p.Deps != nil {
			fmt.Fprintf(&b, " (Ca %d, Ce %d, instability %.2f, abstractness %.2f, distance %.2f)", p.Deps.Afferent, p.Deps.Efferent, p.Deps.Instability, p.Deps.Abstractness, p.Deps.Distance)
		}
		if packagesOnly {
			fmt.Fprintf(&b, ", %d %s\n", len(p.Definitions), plural(len(p.Definitions), "Definition", "Definitions"))
			if p.Deps != nil {
				for _, d := range p.Deps.DependsOn {
					fmt.Fprintf(&b, "  depends on %s (%d %s)\n", d.Package, len(d.Uses), plural(len(d.Uses), "link", "links"))
				}
			}
			continue
		}
		b.WriteString("\n")
		if p.Deps != nil {
			for _, d := range p.Deps.DependsOn {
				var because []string
				for _, u := range d.Uses {
					verb := "links"
					if u.Via == "kind" {
						verb = "is of Kind"
					}
					because = append(because, fmt.Sprintf("%s %s %s", u.From, verb, u.Target))
				}
				fmt.Fprintf(&b, "  depends on %s: %s\n", d.Package, strings.Join(because, "; "))
			}
		}
		for _, d := range p.Definitions {
			fmt.Fprintf(&b, "  %s\n", definitionLine(d))
			if len(d.Links) > 0 {
				fmt.Fprintf(&b, "    links %s\n", strings.Join(d.Links, ", "))
			}
		}
	}
	return b.String()
}

// renderKinds prints every Kind with the Definitions the spec has of it:
// first those it has some of, with the top-level folders they lie in, then
// those it has none of.
func renderKinds(kinds []analysis.KindCount) string {
	var b strings.Builder
	var none []string
	b.WriteString("Kinds and how many Definitions the spec has of each:\n\n")
	for _, k := range kinds {
		if k.Definitions == 0 {
			none = append(none, k.Kind)
			continue
		}
		fmt.Fprintf(&b, "  %s: %d, in %s\n", k.Kind, k.Definitions, strings.Join(k.Folders, ", "))
	}
	if len(none) > 0 {
		fmt.Fprintf(&b, "\nNo Definition yet: %s\n", strings.Join(none, ", "))
	}
	b.WriteString("\nmarkitect doc KIND prints what a Kind is for.\n")
	return b.String()
}

// renderRead prints the Definitions at or below a folder as one text to
// read, in Markdown: a heading for each folder, each Definition under its
// title with its ID and Kind, and a part one level below its whole.
func renderRead(folder string, entries []analysis.ReadEntry) string {
	var b strings.Builder
	if folder == spec.RootPackage {
		b.WriteString("# The spec\n")
	} else {
		fmt.Fprintf(&b, "# The spec: %s\n", folder)
	}
	fmt.Fprintf(&b, "\n%d %s, as markitect doc --read prints them. The files under .markitect/spec are the spec; this text is a view of them.\n", len(entries), plural(len(entries), "Definition", "Definitions"))
	at := "\x00"
	for _, e := range entries {
		level := min(3+e.Depth, 6)
		if e.Depth == 0 && e.Folder != at {
			at = e.Folder
			if at != spec.RootPackage {
				fmt.Fprintf(&b, "\n## %s\n", at)
			}
		}
		title := e.Title
		if title == "" {
			title = e.ID
		}
		fmt.Fprintf(&b, "\n%s %s\n\n`%s`, %s\n", strings.Repeat("#", level), title, e.ID, e.Kind)
		if text := strings.TrimSpace(headingsBelow(partText(either(e.Shown, e.Text)), level)); text != "" {
			b.WriteString("\n" + text + "\n")
		}
	}
	return b.String()
}

// headingsBelow moves the headings of a text below a heading of a level,
// so that a section of a Definition reads as a part of it: a heading of
// level n becomes one of level+n-1, and none goes deeper than six. Lines in
// a fenced block stay as they are.
func headingsBelow(text string, level int) string {
	lines := strings.Split(text, "\n")
	fence := ""
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) > 3 {
			continue
		}
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, fence[:1]+" ") == "" {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			continue
		}
		n := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
		if n == 0 || n > 6 || (len(trimmed) > n && trimmed[n] != ' ') {
			continue
		}
		lines[i] = strings.Repeat("#", min(level+n-1, 6)) + trimmed[n:]
	}
	return strings.Join(lines, "\n")
}

// definitionLine is a Definition on one line: its ID, title and Kind.
func definitionLine(d analysis.ListedDefinition) string {
	return fmt.Sprintf("%s: %s (%s)", d.ID, d.Title, d.Kind)
}

// renderDocDefinition prints a Definition as doc shows it.
func renderDocDefinition(d analysis.DocDefinition) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", definitionLine(d.ListedDefinition))
	switch {
	case d.Library != "":
		fmt.Fprintf(&b, "Library %s\n", d.Library)
	default:
		fmt.Fprintf(&b, "Package %s, %s\n", d.Package, d.File)
	}
	if d.Whole != "" {
		fmt.Fprintf(&b, "Part of %s\n", thing(d.Whole))
	}
	fmt.Fprintf(&b, "\n%s", either(d.Shown, d.Text))
	// A thing is read whole: its parts follow, each group under the name of
	// its folder, a part's title before its text. A part of another thing,
	// which the Definition links, names the thing it is a part of.
	group := "\x00"
	for _, part := range d.Parts {
		if part.Group != group {
			group = part.Group
			if group != "" {
				fmt.Fprintf(&b, "\n%s:\n", group)
			}
		}
		if part.Of != "" {
			fmt.Fprintf(&b, "\n  %s (%s), a part of %s\n", part.Title, part.ID, thing(part.Of))
		} else {
			fmt.Fprintf(&b, "\n  %s (%s)\n", part.Title, part.ID)
		}
		for _, line := range strings.Split(strings.TrimSpace(partText(either(part.Shown, part.Text))), "\n") {
			if line == "" {
				b.WriteString("\n")
				continue
			}
			fmt.Fprintf(&b, "    %s\n", line)
		}
	}
	// What a Kind declares stands in its front matter, which the text above
	// leaves out.
	if s := d.Declares; s != nil {
		if s.Fields != nil {
			b.WriteString("\nDeclares the fields:\n")
			for _, f := range s.Fields {
				line := f.Name
				if f.Required {
					line += ", required"
				}
				switch {
				case f.Links == nil:
				case len(f.Links) == 0:
					line += ", links nothing"
				default:
					line += ", links " + strings.Join(f.Links, ", ")
				}
				fmt.Fprintf(&b, "  %s\n", line)
			}
			if len(s.Fields) == 0 {
				b.WriteString("  none\n")
			}
		}
		if s.Parts != nil {
			b.WriteString("\nDeclares the parts:\n")
			for _, p := range s.Parts {
				where := "under " + p.Folder + "/"
				if p.Folder == spec.DirectParts {
					where = "in the folder itself"
				}
				fmt.Fprintf(&b, "  %s: %s\n", where, strings.Join(p.Kinds, ", "))
			}
			if len(s.Parts) == 0 {
				b.WriteString("  none\n")
			}
		}
		switch {
		case s.Links == nil:
		case len(s.Links) == 0:
			b.WriteString("\nDeclares that it links nothing.\n")
		default:
			fmt.Fprintf(&b, "\nDeclares that it links only: %s\n", strings.Join(s.Links, ", "))
		}
		if s.Binds {
			b.WriteString("\nDeclares that a Definition of this Kind binds its Package and those below it without a link.\n")
		}
		if s.Last {
			b.WriteString("\nDeclares that a Definition of this Kind is built and verified last, on the whole project.\n")
		}
		if s.Unrealized {
			b.WriteString("\nDeclares that no file realizes a Definition of this Kind.\n")
		}
	}
	if len(d.Links) > 0 {
		fmt.Fprintf(&b, "\nLinks to: %s\n", strings.Join(d.Links, ", "))
	}
	if len(d.BoundBy) > 0 {
		b.WriteString("\nBound by, without a link, as everything in its Package:\n")
		for _, o := range d.BoundBy {
			fmt.Fprintf(&b, "  %s\n", definitionLine(o))
		}
	}
	if len(d.LinkedFrom) > 0 {
		b.WriteString("\nLinked from:\n")
		for _, r := range d.LinkedFrom {
			fmt.Fprintf(&b, "  %s (%s), %s\n", r.From, r.Via, where(r.File, r.Line))
		}
	}
	if len(d.RealizedBy) > 0 {
		b.WriteString("\nRealized by:\n")
		for _, r := range d.RealizedBy {
			fmt.Fprintf(&b, "  %s, %s\n", r.File, where(r.Page, r.Line))
		}
	}
	if r := d.Record; r != nil {
		fmt.Fprintf(&b, "\nRecord: %s, %s", r.Item, r.State)
		switch {
		case len(r.Changes) > 0:
			fmt.Fprintf(&b, ": %s", strings.Join(r.Changes, ", "))
		case r.Why != "":
			fmt.Fprintf(&b, ": %s", r.Why)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// thing names a whole as a link names it: by its folder, when the folder
// carries its name.
func thing(id string) string {
	if folder := spec.FolderOf(id); id == spec.AnchorOf(folder) && folder != spec.RootPackage {
		return folder
	}
	return id
}

// either returns the text as doc shows it, or the text as written when a
// caller gave only that.
func either(shown, text string) string {
	if shown != "" {
		return shown
	}
	return text
}

// partText is a part's text without its title, which doc prints above it:
// a first line that starts with "# ", or one underlined with "=".
func partText(text string) string {
	first, rest, _ := strings.Cut(text, "\n")
	if strings.HasPrefix(first, "# ") {
		return rest
	}
	if under, after, _ := strings.Cut(rest, "\n"); under != "" && strings.Trim(under, "=") == "" {
		return after
	}
	return text
}

// renderDocPackage prints a Package as doc shows it.
func renderDocPackage(p analysis.DocPackage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Package %s", p.Name)
	if p.Parent != "" {
		fmt.Fprintf(&b, ", in %s", p.Parent)
	}
	b.WriteString("\n")
	if p.Page != "" {
		fmt.Fprintf(&b, "Registry page %s", p.Page)
		if len(p.Paths) > 0 {
			fmt.Fprintf(&b, ", paths %s", strings.Join(p.Paths, ", "))
		}
		b.WriteString("\n")
	}
	if len(p.Children) > 0 {
		fmt.Fprintf(&b, "Packages below: %s\n", strings.Join(p.Children, ", "))
	}
	// The anchor's text says what the Package is, as go doc opens with a
	// package's comment.
	if p.Text != "" {
		b.WriteString("\n" + either(p.Shown, p.Text))
	}
	// The public Definitions are what the Package publishes, as go doc shows a
	// package's exported names; the internal ones follow apart.
	for _, internal := range []bool{false, true} {
		heading := "\nPublic Definitions:\n"
		if internal {
			heading = "\nInternal Definitions:\n"
		}
		for _, d := range p.Definitions {
			if d.Internal != internal {
				continue
			}
			b.WriteString(heading)
			heading = ""
			fmt.Fprintf(&b, "  %s\n", definitionLine(d))
		}
	}
	if len(p.LinkedFrom) > 0 {
		b.WriteString("\nLinked from:\n")
		for _, r := range p.LinkedFrom {
			fmt.Fprintf(&b, "  %s (%s), %s\n", r.From, r.Via, where(r.File, r.Line))
		}
	}
	if len(p.Files) > 0 {
		b.WriteString("\nFiles:\n")
		for _, f := range p.Files {
			fmt.Fprintf(&b, "  %s\n", f)
		}
	}
	return b.String()
}

func where(file string, line int) string {
	if line == 0 {
		return file
	}
	return fmt.Sprintf("%s:%d", file, line)
}
