package model

import (
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const coreBody = `# Kind

A Kind says what a Definition is and how it is used. Every Definition names
its Kind in its front matter; a Kind is itself a Definition whose Kind is
Kind. Core offers nothing else.
`

// Compile parses and checks a model with its Libraries. The returned
// Diagnostics are the compile errors, sorted; the Model is usable for display
// even when there are errors.
func Compile(src Source) (*Model, Diagnostics) {
	c := &compiler{
		m: &Model{
			Definitions: map[string]*Definition{},
			Packages:    map[string]*Package{RootPackage: {Path: RootPackage}},
			Libraries:   map[string]*Library{},
			Edges:       map[string]map[string][]Use{},
		},
		rawKind: map[string]rawKind{},
	}
	c.m.Definitions[CoreKind] = &Definition{ID: CoreKind, Library: "core", Kind: CoreKind, Title: CoreKind, File: "core/Kind.md", Data: []byte(coreBody), Words: Words([]byte(coreBody))}
	for _, lib := range src.Libraries {
		c.loadLibrary(lib)
	}
	c.loadModel(src)
	c.resolve()
	c.checkCycles()
	for _, p := range c.m.Packages {
		sort.Strings(p.Children)
		sort.Strings(p.Definitions)
	}
	for _, l := range c.m.Libraries {
		sort.Strings(l.Definitions)
	}
	return c.m, c.diags.Sorted()
}

type rawKind struct {
	value string
	line  int
}

type compiler struct {
	m       *Model
	diags   Diagnostics
	rawKind map[string]rawKind // Definition ID -> kind as written
	raw     map[string][]RawLink
}

func (c *compiler) loadLibrary(lib LibrarySource) {
	if !ValidSegment(lib.Name) || lib.Name == CoreKind {
		c.diags.Add("library.name", lib.Dir, 0, "%q cannot name a Library: use letters, digits, - and _, and not %q", lib.Name, CoreKind)
		return
	}
	c.m.Libraries[lib.Name] = &Library{Name: lib.Name, Version: lib.Version}
	for _, f := range lib.Files {
		file := path.Join(lib.Dir, f.Path)
		name, ok := strings.CutSuffix(f.Path, ".md")
		if !ok || !ValidSegment(name) {
			c.diags.Add("library.file", file, 0, "a Library holds only Kind Definitions, one <Name>.md file each at its top level")
			continue
		}
		id := lib.Name + "/" + name
		d := c.parseDefinition(id, file, f.Data)
		if d == nil {
			continue
		}
		d.Library = lib.Name
		if d.Internal {
			c.diags.Add("library.internal", file, 0, "a Library Definition cannot be internal")
		}
		c.m.Libraries[lib.Name].Definitions = append(c.m.Libraries[lib.Name].Definitions, id)
	}
}

func (c *compiler) loadModel(src Source) {
	// names[parent][lower-case name] collects the spellings and forms (file or
	// folder) of every entry, to find entries that collide.
	type entry struct{ name, form string }
	names := map[string]map[string][]entry{}
	note := func(parent, name, form string) {
		if names[parent] == nil {
			names[parent] = map[string][]entry{}
		}
		key := strings.ToLower(name)
		for _, e := range names[parent][key] {
			if e == (entry{name, form}) {
				return
			}
		}
		names[parent][key] = append(names[parent][key], entry{name, form})
	}
	files := append([]File(nil), src.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for _, f := range files {
		file := path.Join(src.Dir, f.Path)
		id, isMarkdown := strings.CutSuffix(f.Path, ".md")
		if !isMarkdown {
			c.diags.Add("model.file", file, 0, "the model folder holds only Definitions (.md files) and Package folders")
			continue
		}
		if !validPath(id) {
			c.diags.Add("model.name", file, 0, "%q is not a valid Definition path: every folder and file name starts with a letter or digit and uses only letters, digits, - and _", f.Path)
			continue
		}
		segments := strings.Split(id, "/")
		parent := RootPackage
		for _, s := range segments[:len(segments)-1] {
			note(parent, s, "folder")
			child := strings.TrimPrefix(parent+"/"+s, "/")
			if _, ok := c.m.Packages[child]; !ok {
				c.m.Packages[child] = &Package{Path: child, Parent: parent}
				c.m.Packages[parent].Children = append(c.m.Packages[parent].Children, child)
			}
			parent = child
		}
		note(parent, segments[len(segments)-1], "file")
		if _, taken := c.m.Definitions[id]; taken {
			continue // Core's Kind or a Library Kind; reported as reserved below
		}
		d := c.parseDefinition(id, file, f.Data)
		if d == nil {
			continue
		}
		d.Package = parent
		c.m.Packages[parent].Definitions = append(c.m.Packages[parent].Definitions, id)
	}
	for parent, byKey := range names {
		for _, entries := range byKey {
			if len(entries) > 1 {
				var spelled []string
				for _, e := range entries {
					spelled = append(spelled, e.form+" "+e.name)
				}
				sort.Strings(spelled)
				c.diags.Add("model.collision", path.Join(src.Dir, parent), 0, "%s name the same thing; links could not tell them apart", strings.Join(spelled, " and "))
			}
		}
	}
	for _, top := range c.m.Packages[RootPackage].Children {
		c.reserved(src.Dir, top)
	}
	for _, id := range c.m.Packages[RootPackage].Definitions {
		c.reserved(src.Dir, id)
	}
}

func (c *compiler) reserved(dir, name string) {
	if name == CoreKind {
		c.diags.Add("model.reserved", path.Join(dir, name), 0, "%q is Core's Kind; choose another top-level name", name)
	}
	if _, ok := c.m.Libraries[name]; ok {
		c.diags.Add("model.reserved", path.Join(dir, name), 0, "%q is the name of a Library; links to %s/... would be ambiguous", name, name)
	}
}

// parseDefinition reads one Definition file. It returns nil when the file is
// unusable; the reason is then a diagnostic.
func (c *compiler) parseDefinition(id, file string, data []byte) *Definition {
	page, err := ParsePage(data)
	if err != nil {
		c.diags.Add("definition.front_matter", file, page.FrontLine, "%v", err)
		return nil
	}
	d := &Definition{ID: id, File: file, Data: data, Words: Words(page.Body)}
	kind := rawKind{line: max(page.FrontLine, 1)}
	if page.FrontLine == 0 {
		c.diags.Add("definition.front_matter", file, 1, "a Definition starts with front matter that names its kind:\n---\nkind: std/Feature\n---")
	}
	for key, node := range page.Front {
		switch key {
		case "kind":
			if node.Kind != yaml.ScalarNode || node.Tag != "!!str" || strings.TrimSpace(node.Value) == "" {
				c.diags.Add("definition.front_matter", file, node.Line, "kind must be a path such as std/Feature")
				continue
			}
			kind = rawKind{value: strings.TrimSpace(node.Value), line: node.Line}
		case "internal":
			if node.Kind != yaml.ScalarNode || node.Tag != "!!bool" {
				c.diags.Add("definition.front_matter", file, node.Line, "internal must be true or false")
				continue
			}
			d.Internal = node.Value == "true"
		default:
			c.diags.Add("definition.front_matter", file, node.Line, "unknown front matter key %q: a Definition's front matter holds only kind and internal", key)
		}
	}
	if kind.value == "" && page.FrontLine != 0 {
		if _, ok := page.Front["kind"]; !ok {
			c.diags.Add("definition.front_matter", file, page.FrontLine, "the front matter must name the Definition's kind")
		}
	}
	title, line, err := page.Title()
	if err != nil {
		c.diags.Add("definition.title", file, line, "%v", err)
	}
	d.Title = title
	c.m.Definitions[id] = d
	c.rawKind[id] = kind
	if c.raw == nil {
		c.raw = map[string][]RawLink{}
	}
	c.raw[id] = page.Links()
	return d
}

// resolve resolves every Kind and link, checks visibility and direction, and
// records the Package edges.
func (c *compiler) resolve() {
	ids := make([]string, 0, len(c.rawKind))
	for id := range c.rawKind {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		d := c.m.Definitions[id]
		if k := c.rawKind[id]; k.value != "" {
			link := c.resolveLink(d, k.value, k.line, true)
			if link.Target != "" {
				if link.Package || link.Target != CoreKind && c.rawKind[link.Target].value != CoreKind {
					c.diags.Add("kind.not_a_kind", d.File, k.line, "%s is not a Kind: a Kind is a Definition whose kind is Kind", k.value)
				} else {
					d.Kind = link.Target
				}
			}
			d.Links = append(d.Links, link)
		}
		if d.Library != "" && d.Kind != "" && d.Kind != CoreKind {
			c.diags.Add("library.not_a_kind", d.File, c.rawKind[id].line, "a Library holds only Kinds; this Definition's kind must be Kind")
		}
		for _, raw := range c.raw[id] {
			d.Links = append(d.Links, c.resolveLink(d, raw.Target, raw.Line, false))
		}
	}
}

func (c *compiler) resolveLink(d *Definition, raw string, line int, kind bool) Link {
	link := Link{Raw: raw, Line: line, Kind: kind}
	what := "link"
	code := "link.unresolved"
	if kind {
		what, code = "kind", "kind.unresolved"
	}
	target := strings.TrimSuffix(raw, ".md")
	if !validPath(target) {
		c.diags.Add("link.syntax", d.File, line, "%s %q is not a path such as members/member or std/Rule: no leading or trailing /, no .., only letters, digits, - and _ in each name", what, raw)
		return link
	}
	first, _, nested := strings.Cut(target, "/")
	switch {
	case target == CoreKind:
		link.Target = CoreKind
	case c.m.Libraries[first] != nil:
		if _, ok := c.m.Definitions[target]; ok && nested {
			link.Target = target
		} else {
			c.diags.Add(code, d.File, line, "%s %q: Library %s has no Kind of that name", what, raw, first)
			return link
		}
	case d.Library != "":
		c.diags.Add(code, d.File, line, "%s %q: a Library Definition may link only to Libraries the manifest names and to Kind", what, raw)
		return link
	default:
		if def, ok := c.m.Definitions[target]; ok && def.InModel() {
			link.Target = target
		} else if _, ok := c.m.Packages[target]; ok {
			link.Target, link.Package = target, true
		} else {
			c.diags.Add(code, d.File, line, "%s %q resolves to no Definition, Package or Library Kind; model paths start at the model folder", what, raw)
			return link
		}
	}
	if d.Library == "" {
		c.checkDirection(d, link)
	}
	return link
}

// checkDirection applies visibility and the Go-like dependency rules to one
// resolved link of a model Definition and records the edge it makes.
func (c *compiler) checkDirection(d *Definition, link Link) {
	to := link.Target
	target := c.m.Definitions[link.Target]
	if !link.Package {
		if !target.InModel() {
			return // Libraries and Core lie outside the Package graph
		}
		to = target.Package
	}
	from := d.Package
	switch {
	case from == to || Within(from, to):
		// The same Package, or one above: in scope by location.
		return
	case Within(to, from):
		if link.Package {
			return // naming a child Package is no edge
		}
		c.diags.Add("link.downward", d.File, link.Line, "%s links down to %s inside its child Package %s; a parent may name a child Package but not a Definition inside it (move the shared concept up, or let the child link to a need stated here)", PackageName(from), link.Target, to)
		return
	}
	if !link.Package && target.Internal {
		c.diags.Add("link.internal", d.File, link.Line, "%s is internal to %s; only %s and the Packages below it may link to it", link.Target, PackageName(to), PackageName(to))
	}
	if c.m.Edges[from] == nil {
		c.m.Edges[from] = map[string][]Use{}
	}
	c.m.Edges[from][to] = append(c.m.Edges[from][to], Use{From: d.ID, Target: link.Target, Kind: link.Kind, File: d.File, Line: link.Line})
}
