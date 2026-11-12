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
	// Shown is the text as doc prints it: each link by its label, or by the
	// title of what it names.
	Shown string `json:"-"`
	// Whole is the Definition this one is a part of. Parts are its own parts
	// and the parts of other things it links, with their text, each folder's
	// together, in the order it links them and then by ID.
	Whole string    `json:"whole,omitempty"`
	Parts []DocPart `json:"parts"`
	// Fields are the labelled fields of the text, each with what it links.
	Fields []DocField `json:"fields,omitempty"`
	// BoundBy are the Definitions that bind this one without a link: those of
	// a Kind that declares binds, in its Package or a Package above it, parts
	// left out since their whole stands for them. Nothing binds a Definition
	// that no file realizes (DEC-113).
	BoundBy []ListedDefinition `json:"bound_by,omitempty"`
	// Declares is what a Kind declares about its Definitions: their fields
	// and the folders of their parts.
	Declares   *DocDeclares `json:"declares,omitempty"`
	LinkedFrom []Ref        `json:"linked_from"`
	RealizedBy []Realizer   `json:"realized_by"`
	Record     *DocRecord   `json:"record,omitempty"`
}

// DocField is one labelled field of a Definition's text with the Definitions
// and Packages it links.
type DocField struct {
	Label string   `json:"label"`
	Links []string `json:"links"`
}

// DocDeclares is what a Kind declares. Fields or Parts is null when the Kind
// declares nothing about them, and empty when it declares that there are
// none.
type DocDeclares struct {
	Fields []DocDeclaredField `json:"fields"`
	Parts  []DocDeclaredPart  `json:"parts"`
	// Links are the Kinds a link of a Definition may name where a field does
	// not say for itself; null when the Kind does not say.
	Links []string `json:"links"`
	// Unrealized says that no file realizes a Definition of the Kind,
	// Binds that one binds its Package and those below it without a link,
	// and Last that one is built and verified last, on the whole project.
	Unrealized bool `json:"unrealized,omitempty"`
	Binds      bool `json:"binds,omitempty"`
	Last       bool `json:"last,omitempty"`
}

// DocDeclaredField is one field a Kind declares. Links is null when a link
// in the field may name anything.
type DocDeclaredField struct {
	Name     string   `json:"name"`
	Required bool     `json:"required"`
	Links    []string `json:"links"`
}

// DocDeclaredPart is one folder of parts a Kind declares; the folder "."
// stands for the thing's own folder.
type DocDeclaredPart struct {
	Folder string   `json:"folder"`
	Kinds  []string `json:"kinds"`
}

// DocPart is one part as doc shows it with a Definition. Group is the folder
// it lies in below its whole: conditions for .../lend/conditions/in-stock,
// and for a part with parts of its own the folder its folder lies in. Of
// is its whole when that is another Definition than the one shown: a part of
// another thing, which the Definition links.
type DocPart struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Group string `json:"group"`
	Of    string `json:"of,omitempty"`
	Text  string `json:"text"`
	Shown string `json:"-"`
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
	Shown       string             `json:"-"` // the anchor's text as doc prints it
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
	dd.Shown = shown(p.Spec, d.Data)
	for _, f := range d.Fields {
		field := DocField{Label: f.Label, Links: []string{}}
		for _, l := range d.Links {
			if l.Field == f.Label && !l.Kind && l.Target != "" {
				field.Links = append(field.Links, l.Target)
			}
		}
		dd.Fields = append(dd.Fields, field)
	}
	if d.Own() && p.Spec.Bound(d.ID) {
		unit, binding := p.Spec.UnitOf(d.ID), p.Spec.Binding()
		for at := d.Package; ; at = p.Spec.Parent(at) {
			for _, id := range binding[at] {
				// Neither the Definition's own thing nor a thing it is a part of,
				// and no part, since its whole stands for it.
				if other := p.Spec.Definitions[id]; other.Whole == "" && id != unit && !p.Spec.IsPartOf(d.ID, id) {
					dd.BoundBy = append(dd.BoundBy, listDefinition(other, false))
				}
			}
			if at == spec.RootPackage {
				break
			}
		}
		sort.Slice(dd.BoundBy, func(i, j int) bool { return dd.BoundBy[i].ID < dd.BoundBy[j].ID })
	}
	if s := d.Schema; s != nil {
		dd.Declares = &DocDeclares{Unrealized: s.Unrealized, Binds: s.Binds, Last: s.Last}
		if s.Limited {
			dd.Declares.Links = append([]string{}, s.Links...)
		}
		if s.HasFields {
			dd.Declares.Fields = []DocDeclaredField{}
		}
		for _, f := range s.Fields {
			declared := DocDeclaredField{Name: f.Name, Required: f.Required}
			if f.Limited {
				declared.Links = append([]string{}, f.Links...)
			}
			dd.Declares.Fields = append(dd.Declares.Fields, declared)
		}
		if s.HasParts {
			dd.Declares.Parts = []DocDeclaredPart{}
		}
		for _, part := range s.Parts {
			dd.Declares.Parts = append(dd.Declares.Parts, DocDeclaredPart{Folder: part.Folder, Kinds: append([]string{}, part.Kinds...)})
		}
	}
	rank := map[string]int{}
	for i, l := range d.Links {
		if _, seen := rank[l.Target]; !seen && l.Target != "" {
			rank[l.Target] = i + 1
		}
	}
	add := func(id, whole, of string) {
		part := p.Spec.Definitions[id]
		group := spec.FolderOf(strings.TrimPrefix(id, spec.FolderOf(whole)+"/"))
		if id == spec.AnchorOf(spec.FolderOf(id)) {
			group = spec.FolderOf(group) // a part with parts of its own: beside its siblings
		}
		dd.Parts = append(dd.Parts, DocPart{ID: id, Title: part.Title, Kind: part.Kind, Group: group, Of: of, Text: body(part.Data), Shown: shown(p.Spec, part.Data)})
	}
	shownHere := map[string]bool{d.ID: true}
	for _, id := range p.Spec.Parts(d.ID) {
		shownHere[id] = true
		add(id, d.ID, "")
	}
	// The parts of other things that the Definition links are read with it: a
	// Condition several Operations share lies with what it is about. A part
	// does not show its own thing this way, neither its wholes nor the other
	// parts of the thing it belongs to.
	top := d.ID
	for at := d.Whole; at != ""; at = p.Spec.Definitions[at].Whole {
		top = at
	}
	for _, l := range d.Links {
		t := p.Spec.Definitions[l.Target]
		if l.Kind || l.Package || t == nil || !t.Own() || t.Whole == "" || shownHere[l.Target] || l.Target == top || p.Spec.IsPartOf(l.Target, top) {
			continue
		}
		shownHere[l.Target] = true
		add(l.Target, t.Whole, t.Whole)
	}
	// What lies directly in the thing's folder comes first, under no name.
	// Then the folders in the order the Definition links their parts, a folder
	// below another right after it, and the others by name; in a folder, the
	// parts in the order of the links and then by ID.
	first := map[string]int{}
	for _, part := range dd.Parts {
		topFolder, _, _ := strings.Cut(part.Group, "/")
		if r := rank[part.ID]; r != 0 && (first[topFolder] == 0 || r < first[topFolder]) {
			first[topFolder] = r
		}
	}
	sort.SliceStable(dd.Parts, func(i, j int) bool {
		a, b := dd.Parts[i], dd.Parts[j]
		if a.Group != b.Group {
			if (a.Group == "") != (b.Group == "") {
				return a.Group == ""
			}
			ta, _, _ := strings.Cut(a.Group, "/")
			tb, _, _ := strings.Cut(b.Group, "/")
			if ta != tb {
				fa, fb := first[ta], first[tb]
				if (fa == 0) != (fb == 0) {
					return fa != 0
				}
				if fa != fb {
					return fa < fb
				}
			}
			return a.Group < b.Group
		}
		ra, rb := rank[a.ID], rank[b.ID]
		if (ra == 0) != (rb == 0) {
			return ra != 0
		}
		if ra != rb {
			return ra < rb
		}
		return a.ID < b.ID
	})
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
		dp.Anchor, dp.Text, dp.Shown = a.ID, body(a.Data), shown(p.Spec, a.Data)
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

// shown returns a Definition's text as doc prints it for reading: a link is
// shown by its label, or by the title of what it names, a Package by the
// title of its anchor. The links are the ones the compiler reads, so a link
// in code stays as written, and so does a link that names nothing.
func shown(sp *spec.Spec, data []byte) string {
	page, err := spec.ParsePage(project.Normalize(data))
	if err != nil {
		return body(data)
	}
	out := append([]byte(nil), page.Body...)
	links := page.Links()
	for i := len(links) - 1; i >= 0; i-- {
		l := links[i]
		written := string(page.Body[l.Start:l.End])
		_, label, labelled := strings.Cut(strings.TrimSuffix(written, "]]"), "|")
		show := strings.TrimSpace(label)
		if !labelled || show == "" {
			target := strings.TrimSuffix(l.Target, ".md")
			switch d, pkg := sp.Lookup(target), sp.Packages[target]; {
			case d != nil && d.Title != "":
				show = d.Title
			case pkg != nil && sp.Definitions[pkg.Anchor] != nil && sp.Definitions[pkg.Anchor].Title != "":
				show = sp.Definitions[pkg.Anchor].Title
			default:
				continue
			}
		}
		out = append(out[:l.Start], append([]byte(show), out[l.End:]...)...)
	}
	return strings.TrimSpace(string(out)) + "\n"
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
