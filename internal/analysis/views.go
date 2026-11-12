package analysis

import (
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// The views of list and doc: what they show of a loaded project, as the command
// line renders it in text and JSON.

// ListedDefinition is one Definition as list and doc show it.
type ListedDefinition struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	File  string `json:"file"`
	// Internal says that only its own Package and those below may link to it;
	// the public Definitions of a Package are what it publishes.
	Internal bool `json:"internal"`
	// Links are the Definitions and Packages its body links to, with --deps.
	Links []string `json:"links,omitempty"`
}

// ListedPackage is one Package as list shows it.
type ListedPackage struct {
	Path        string             `json:"path"`
	Name        string             `json:"name"`
	Parent      string             `json:"parent,omitempty"`
	Definitions []ListedDefinition `json:"definitions"`
	Deps        *PackageDeps       `json:"deps,omitempty"`
}

// PackageDeps is what a Package depends on, its coupling, and its abstractness and
// distance from the main sequence, with --deps. They inform; vet warns about none
// of the last two yet (DEC-102).
type PackageDeps struct {
	Afferent     int          `json:"afferent"`
	Efferent     int          `json:"efferent"`
	Instability  float64      `json:"instability"`
	Abstractness float64      `json:"abstractness"`
	Distance     float64      `json:"distance"`
	DependsOn    []Dependency `json:"depends_on"`
}

// Dependency is one Package another depends on, and the links that make it.
type Dependency struct {
	Package string `json:"package"`
	Uses    []Use  `json:"uses"`
}

// Use is one link that makes a dependency.
type Use struct {
	From   string `json:"from"`
	Target string `json:"target"`
	// Via is "link", or "kind" when From is of Target as its Kind.
	Via  string `json:"via"`
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
}

// ListPackages is the spec as list shows it; with deps, what each Package and
// Definition depends on.
func ListPackages(p *project.Project, deps bool) []ListedPackage {
	sp := p.Spec
	g := BuildGraph(p)
	nodes := map[string]PackageNode{}
	for _, n := range g.Packages {
		nodes[n.Path] = n
	}
	edges := map[string][]Edge{}
	for _, e := range g.Edges {
		edges[e.From] = append(edges[e.From], e)
	}
	out := []ListedPackage{}
	for _, path := range sp.SortedPackages() {
		pkg := sp.Packages[path]
		lp := ListedPackage{Path: path, Name: spec.PackageName(path), Definitions: []ListedDefinition{}}
		if path != spec.RootPackage {
			lp.Parent = spec.PackageName(pkg.Parent)
		}
		for _, id := range pkg.Definitions {
			lp.Definitions = append(lp.Definitions, listDefinition(sp.Definitions[id], deps))
		}
		if deps {
			n := nodes[path]
			d := &PackageDeps{Afferent: n.Afferent, Efferent: n.Efferent, Instability: n.Instability, Abstractness: n.Abstractness, Distance: n.Distance, DependsOn: []Dependency{}}
			for _, e := range edges[path] {
				dep := Dependency{Package: spec.PackageName(e.To)}
				for _, u := range e.Uses {
					via := "link"
					if u.Kind {
						via = "kind"
					}
					dep.Uses = append(dep.Uses, Use{From: u.From, Target: u.Target, Via: via, File: u.File, Line: u.Line})
				}
				d.DependsOn = append(d.DependsOn, dep)
			}
			lp.Deps = d
		}
		out = append(out, lp)
	}
	return out
}

// listDefinition is one Definition as list shows it; with links, the Definitions
// and Packages its body links to.
func listDefinition(d *spec.Definition, links bool) ListedDefinition {
	ld := ListedDefinition{ID: d.ID, Title: d.Title, Kind: d.Kind, File: d.File, Internal: d.Internal}
	if links {
		seen := map[string]bool{}
		for _, l := range d.Links {
			if l.Kind || l.Target == "" || l.Target == d.ID || seen[l.Target] {
				continue
			}
			seen[l.Target] = true
			ld.Links = append(ld.Links, l.Target)
		}
		sort.Strings(ld.Links)
	}
	return ld
}

// DocRecord is a Definition's record, as status sees it. The record is not
// analysis's to read, so the caller fills it in.
type DocRecord struct {
	Item    string   `json:"item"`
	State   string   `json:"state"`
	Verdict string   `json:"verdict,omitempty"`
	Why     string   `json:"why,omitempty"`
	Changes []string `json:"changes,omitempty"`
}

// Realizer is one registry row that names a Definition.
type Realizer struct {
	File string `json:"file"`
	Page string `json:"page"`
	Line int    `json:"line,omitempty"`
}

// DocDefinition is one Definition as doc shows it.
type DocDefinition struct {
	ListedDefinition
	Package string `json:"package"`
	Library string `json:"library,omitempty"`
	Text    string `json:"text"`
	// Whole is the Definition this one is a part of, and Parts are its own
	// parts with their text, in the order it links them and then by ID.
	Whole      string     `json:"whole,omitempty"`
	Parts      []DocPart  `json:"parts"`
	LinkedFrom []Ref      `json:"linked_from"`
	RealizedBy []Realizer `json:"realized_by"`
	Record     *DocRecord `json:"record,omitempty"`
}

// DocPart is one part of a Definition as doc shows it. Group is the folder it
// lies in below its whole: conditions for .../lend/conditions/in-stock.
type DocPart struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Group string `json:"group"`
	Text  string `json:"text"`
}

// DocPackage is one Package as doc shows it.
type DocPackage struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Parent   string   `json:"parent,omitempty"`
	Children []string `json:"children"`
	// Anchor is the Definition that says what the Package is, and Text its
	// text without front matter; Root has none.
	Anchor      string             `json:"anchor,omitempty"`
	Text        string             `json:"text,omitempty"`
	Page        string             `json:"page,omitempty"`
	Paths       []string           `json:"paths"`
	Definitions []ListedDefinition `json:"definitions"`
	LinkedFrom  []Ref              `json:"linked_from"`
	Files       []string           `json:"files"`
}

// DocDefinitionOf gathers what doc shows of a Definition but its record.
func DocDefinitionOf(p *project.Project, d *spec.Definition) DocDefinition {
	dd := DocDefinition{ListedDefinition: listDefinition(d, true), Package: spec.PackageName(d.Package), Library: d.Library, Text: body(d.Data), Whole: d.Whole, Parts: []DocPart{}, LinkedFrom: []Ref{}, RealizedBy: []Realizer{}}
	if d.Library != "" {
		dd.Package = ""
	}
	below := spec.FolderOf(d.ID) + "/"
	for _, id := range p.Spec.Parts(d.ID) {
		part := p.Spec.Definitions[id]
		dd.Parts = append(dd.Parts, DocPart{ID: id, Title: part.Title, Kind: part.Kind, Group: spec.FolderOf(strings.TrimPrefix(id, below)), Text: body(part.Data)})
	}
	// The parts of one folder stay together, in the order they came.
	sort.SliceStable(dd.Parts, func(i, j int) bool { return dd.Parts[i].Group < dd.Parts[j].Group })
	refs, _ := Refs(p, d.ID)
	for _, r := range refs {
		if r.Via == "registry" {
			dd.RealizedBy = append(dd.RealizedBy, Realizer{File: r.From, Page: r.File, Line: r.Line})
			continue
		}
		dd.LinkedFrom = append(dd.LinkedFrom, r)
	}
	return dd
}

// DocPackageOf gathers what doc shows of a Package.
func DocPackageOf(p *project.Project, pkg *spec.Package) DocPackage {
	dp := DocPackage{Path: pkg.Path, Name: spec.PackageName(pkg.Path), Children: []string{}, Paths: []string{}, Definitions: []ListedDefinition{}, LinkedFrom: []Ref{}, Files: []string{}}
	if pkg.Path != spec.RootPackage {
		dp.Parent = spec.PackageName(pkg.Parent)
	}
	dp.Children = append(dp.Children, pkg.Children...)
	if a := p.Spec.Definitions[pkg.Anchor]; a != nil {
		dp.Anchor, dp.Text = a.ID, body(a.Data)
	}
	for _, id := range pkg.Definitions {
		dp.Definitions = append(dp.Definitions, listDefinition(p.Spec.Definitions[id], false))
	}
	if p.Registry != nil {
		if page := p.Registry.Pages[pkg.Path]; page != nil {
			dp.Page = page.File
			for _, pat := range page.Patterns {
				dp.Paths = append(dp.Paths, pat.Text)
			}
		}
		for file, owner := range p.Registry.Owner {
			if owner == pkg.Path {
				dp.Files = append(dp.Files, file)
			}
		}
		sort.Strings(dp.Files)
	}
	if pkg.Path != spec.RootPackage {
		if refs, err := Refs(p, pkg.Path); err == nil {
			dp.LinkedFrom = append(dp.LinkedFrom, refs...)
		}
	}
	return dp
}

// body is a Definition's text without its front matter.
func body(data []byte) string {
	text := string(project.Normalize(data))
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		if i := strings.Index(rest, "\n---\n"); i >= 0 {
			text = rest[i+len("\n---\n"):]
		}
	}
	return strings.TrimSpace(text) + "\n"
}
