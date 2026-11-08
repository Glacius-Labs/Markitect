package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// The commands on the spec besides check: vet analyses its design, list shows
// what it holds and what depends on what, and doc shows one Definition or
// Package. They run over a spec with errors too, and say so on standard error.

// vet prints the warnings of the spec's design as diagnostics, and one line that
// counts its notes; with --strict the notes too. It exits 1 only with --strict and
// a warning: notes never change the exit code (DEC-102). --json always carries
// both.
func (c CLI) vet(_ context.Context, args []string) (int, error) {
	var o options
	var strict bool
	fs := c.flags("vet", &o)
	fs.BoolVar(&strict, "strict", false, "list the notes too, and exit 1 when there is a warning")
	if positional, err := parse(fs, args); err != nil {
		return Misused, err
	} else if len(positional) != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	ws, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	r := analysis.Vet(p, analysis.History{Revisions: revisionsOf(ws)})
	if o.json {
		ds := diagnostics(severityWarning, r.Warnings)
		for _, n := range r.Notes {
			ds = append(ds, diagnostic{Severity: severityNote, Code: n.Code, Message: n.Message, File: n.File, Line: n.Line, Targets: n.Targets, Structure: n.Structure})
		}
		if err := c.printJSON(struct {
			envelope
			Diagnostics []diagnostic `json:"diagnostics"`
		}{newEnvelope("vet"), ds}); err != nil {
			return Misused, err
		}
	} else {
		for _, d := range r.Warnings {
			fmt.Fprintln(c.Stdout, d.String())
		}
		switch {
		case strict:
			for _, n := range r.Notes {
				fmt.Fprintln(c.Stdout, n.String())
			}
		case len(r.Notes) > 0:
			fmt.Fprintln(c.Stdout, notesLine(r.Notes))
		}
	}
	if strict && len(r.Warnings) > 0 {
		return Failed, nil
	}
	return OK, nil
}

// revisionsOf reads from the project's record how often each Definition's own
// text changed between verdicts; nothing when there is no record. A line it cannot
// read counts nothing here: status names it.
func revisionsOf(ws Workspace) map[string]int {
	data, err := ws.Read(project.ReviewFile)
	if err != nil {
		return nil
	}
	return run.ParseRecord(data).Revisions()
}

// notesLine is the one line plain vet prints for its notes: how many unlinked
// mentions it found within the Package structure and in how many Definitions, as
// the measurement record counts them, how many more go against or beyond the
// structure, and how many other notes there are.
func notesLine(notes []analysis.Note) string {
	var mentions, sources, apart, other int
	for _, n := range notes {
		switch {
		case n.Code != "vet.unlinked_mention":
			other++
		case n.Structure == "":
			mentions += len(n.Targets)
			sources++
		default:
			apart += len(n.Targets)
		}
	}
	var parts []string
	if mentions > 0 {
		parts = append(parts, fmt.Sprintf("%d unlinked %s in %d %s", mentions, plural(mentions, "mention", "mentions"), sources, plural(sources, "Definition", "Definitions")))
	}
	switch {
	case apart > 0 && mentions > 0:
		parts = append(parts, fmt.Sprintf("%d more against or beyond the Package structure", apart))
	case apart > 0:
		parts = append(parts, fmt.Sprintf("%d unlinked %s against or beyond the Package structure", apart, plural(apart, "mention", "mentions")))
	}
	switch {
	case other > 0 && len(parts) > 0:
		parts = append(parts, fmt.Sprintf("%d other %s", other, plural(other, "note", "notes")))
	case other > 0:
		parts = append(parts, fmt.Sprint(other))
	}
	return "notes: " + strings.Join(parts, ", ") + "; markitect vet --strict lists them"
}

// listedDefinition is one Definition as list and doc show it.
type listedDefinition struct {
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

// listedPackage is one Package as list shows it.
type listedPackage struct {
	Path        string             `json:"path"`
	Name        string             `json:"name"`
	Parent      string             `json:"parent,omitempty"`
	Definitions []listedDefinition `json:"definitions"`
	Deps        *packageDeps       `json:"deps,omitempty"`
}

// packageDeps is what a Package depends on, its coupling, and its abstractness and
// distance from the main sequence, with --deps. They inform; vet warns about none
// of the last two yet (DEC-102).
type packageDeps struct {
	Afferent     int          `json:"afferent"`
	Efferent     int          `json:"efferent"`
	Instability  float64      `json:"instability"`
	Abstractness float64      `json:"abstractness"`
	Distance     float64      `json:"distance"`
	DependsOn    []dependency `json:"depends_on"`
}

// dependency is one Package another depends on, and the links that make it.
type dependency struct {
	Package string `json:"package"`
	Uses    []use  `json:"uses"`
}

// use is one link that makes a dependency.
type use struct {
	From   string `json:"from"`
	Target string `json:"target"`
	// Via is "link", or "kind" when From is of Target as its Kind.
	Via  string `json:"via"`
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
}

// list prints the Packages and their Definitions; with --deps, what each
// depends on.
func (c CLI) list(_ context.Context, args []string) (int, error) {
	var o options
	var deps bool
	fs := c.flags("list", &o)
	fs.BoolVar(&deps, "deps", false, "add what each Package and Definition depends on, and each Package's coupling, abstractness and distance")
	if positional, err := parse(fs, args); err != nil {
		return Misused, err
	} else if len(positional) != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	pkgs := listPackages(p, deps)
	if o.json {
		if err := c.printJSON(struct {
			envelope
			Packages []listedPackage `json:"packages"`
		}{newEnvelope("list"), pkgs}); err != nil {
			return Misused, err
		}
		return OK, nil
	}
	fmt.Fprint(c.Stdout, renderList(pkgs))
	return OK, nil
}

// listPackages is the model as list shows it.
func listPackages(p *project.Project, deps bool) []listedPackage {
	m := p.Model
	g := analysis.BuildGraph(p)
	nodes := map[string]analysis.PackageNode{}
	for _, n := range g.Packages {
		nodes[n.Path] = n
	}
	edges := map[string][]analysis.Edge{}
	for _, e := range g.Edges {
		edges[e.From] = append(edges[e.From], e)
	}
	out := []listedPackage{}
	for _, path := range m.SortedPackages() {
		pkg := m.Packages[path]
		lp := listedPackage{Path: path, Name: model.PackageName(path), Definitions: []listedDefinition{}}
		if path != model.RootPackage {
			lp.Parent = model.PackageName(pkg.Parent)
		}
		for _, id := range pkg.Definitions {
			lp.Definitions = append(lp.Definitions, listDefinition(m.Definitions[id], deps))
		}
		if deps {
			n := nodes[path]
			d := &packageDeps{Afferent: n.Afferent, Efferent: n.Efferent, Instability: n.Instability, Abstractness: n.Abstractness, Distance: n.Distance, DependsOn: []dependency{}}
			for _, e := range edges[path] {
				dep := dependency{Package: model.PackageName(e.To)}
				for _, u := range e.Uses {
					via := "link"
					if u.Kind {
						via = "kind"
					}
					dep.Uses = append(dep.Uses, use{From: u.From, Target: u.Target, Via: via, File: u.File, Line: u.Line})
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
func listDefinition(d *model.Definition, links bool) listedDefinition {
	ld := listedDefinition{ID: d.ID, Title: d.Title, Kind: d.Kind, File: d.File, Internal: d.Internal}
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

// docRecord is a Definition's record, as status sees it.
type docRecord struct {
	Item    string   `json:"item"`
	State   string   `json:"state"`
	Verdict string   `json:"verdict,omitempty"`
	Why     string   `json:"why,omitempty"`
	Changes []string `json:"changes,omitempty"`
}

// realizer is one registry row that names a Definition.
type realizer struct {
	File string `json:"file"`
	Page string `json:"page"`
	Line int    `json:"line,omitempty"`
}

// docDefinition is one Definition as doc shows it.
type docDefinition struct {
	listedDefinition
	Package    string         `json:"package"`
	Library    string         `json:"library,omitempty"`
	Text       string         `json:"text"`
	LinkedFrom []analysis.Ref `json:"linked_from"`
	RealizedBy []realizer     `json:"realized_by"`
	Record     *docRecord     `json:"record,omitempty"`
}

// docPackage is one Package as doc shows it.
type docPackage struct {
	Path        string             `json:"path"`
	Name        string             `json:"name"`
	Parent      string             `json:"parent,omitempty"`
	Children    []string           `json:"children"`
	Page        string             `json:"page,omitempty"`
	Paths       []string           `json:"paths"`
	Definitions []listedDefinition `json:"definitions"`
	LinkedFrom  []analysis.Ref     `json:"linked_from"`
	Files       []string           `json:"files"`
}

// doc prints a Definition with its text, what links to it, what it links to,
// the files that realize it and its record; or a Package with its public and
// its internal Definitions apart, its registry page and its files. A target
// that is neither exits 1.
func (c CLI) doc(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("doc", &o)
	positional, err := parse(fs, args)
	if err != nil {
		return Misused, err
	}
	if len(positional) != 1 {
		return Misused, fmt.Errorf("names one Definition or Package, such as members/member or members")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	target := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(positional[0], "[["), "]]"), ".md")
	if d := p.Model.Definitions[target]; d != nil {
		dd := c.docDefinition(ctx, o, p, d)
		if o.json {
			return OK, c.printJSON(struct {
				envelope
				Definition docDefinition `json:"definition"`
			}{newEnvelope("doc"), dd})
		}
		fmt.Fprint(c.Stdout, renderDocDefinition(dd))
		return OK, nil
	}
	if target == "Root" || target == "." {
		target = model.RootPackage
	}
	if pkg := p.Model.Packages[target]; pkg != nil {
		dp := docPackageOf(p, pkg)
		if o.json {
			return OK, c.printJSON(struct {
				envelope
				Package docPackage `json:"package"`
			}{newEnvelope("doc"), dp})
		}
		fmt.Fprint(c.Stdout, renderDocPackage(dp))
		return OK, nil
	}
	var ds model.Diagnostics
	ds.Add("doc.unknown", "", 0, "%s is neither a Definition nor a Package of the spec", target)
	if o.json {
		return Failed, c.printJSON(struct {
			envelope
			Diagnostics []diagnostic `json:"diagnostics"`
		}{newEnvelope("doc"), diagnostics(severityError, ds)})
	}
	fmt.Fprintf(c.Stderr, "markitect doc: %s\n", ds[0].String())
	return Failed, nil
}

// docDefinition gathers what doc shows of a Definition.
func (c CLI) docDefinition(ctx context.Context, o options, p *project.Project, d *model.Definition) docDefinition {
	dd := docDefinition{listedDefinition: listDefinition(d, true), Package: model.PackageName(d.Package), Library: d.Library, Text: body(d.Data), LinkedFrom: []analysis.Ref{}, RealizedBy: []realizer{}}
	if d.Library != "" {
		dd.Package = ""
	}
	refs, _ := analysis.Refs(p, d.ID)
	for _, r := range refs {
		if r.Via == "registry" {
			dd.RealizedBy = append(dd.RealizedBy, realizer{File: r.From, Page: r.File, Line: r.Line})
			continue
		}
		dd.LinkedFrom = append(dd.LinkedFrom, r)
	}
	if d.InModel() && c.Record != nil {
		if st, err := c.Record.Status(ctx, run.StatusRequest{Dir: o.repo, All: true}); err == nil {
			for _, it := range st.Items {
				if it.Definition != d.ID {
					continue
				}
				r := &docRecord{Item: it.Definition + "@" + it.Fingerprint, State: it.State, Verdict: it.Verdict, Why: it.Why}
				for _, ch := range it.Changes {
					r.Changes = append(r.Changes, ch.String())
				}
				dd.Record = r
			}
		}
	}
	return dd
}

// docPackageOf gathers what doc shows of a Package.
func docPackageOf(p *project.Project, pkg *model.Package) docPackage {
	dp := docPackage{Path: pkg.Path, Name: model.PackageName(pkg.Path), Children: []string{}, Paths: []string{}, Definitions: []listedDefinition{}, LinkedFrom: []analysis.Ref{}, Files: []string{}}
	if pkg.Path != model.RootPackage {
		dp.Parent = model.PackageName(pkg.Parent)
	}
	dp.Children = append(dp.Children, pkg.Children...)
	for _, id := range pkg.Definitions {
		dp.Definitions = append(dp.Definitions, listDefinition(p.Model.Definitions[id], false))
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
	if pkg.Path != model.RootPackage {
		if refs, err := analysis.Refs(p, pkg.Path); err == nil {
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
