package cli

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// The renderers write plain text, as Go's tools print theirs; --json gives the
// same as one document.

// renderList prints each Package, then its Definitions indented below it; with
// --deps a Package's coupling, abstractness and distance and what it depends on,
// and each Definition's links.
func renderList(pkgs []analysis.ListedPackage) string {
	var b strings.Builder
	for _, p := range pkgs {
		b.WriteString(p.Name)
		if p.Deps != nil {
			fmt.Fprintf(&b, " (Ca %d, Ce %d, instability %.2f, abstractness %.2f, distance %.2f)", p.Deps.Afferent, p.Deps.Efferent, p.Deps.Instability, p.Deps.Abstractness, p.Deps.Distance)
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
	fmt.Fprintf(&b, "\n%s", d.Text)
	if len(d.Links) > 0 {
		fmt.Fprintf(&b, "\nLinks to: %s\n", strings.Join(d.Links, ", "))
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
