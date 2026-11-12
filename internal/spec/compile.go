package spec

import (
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const coreBody = `# Kind

A Kind says what a Definition is and how it is used. Every Definition names
its Kind in its front matter; a Kind is itself a Definition whose Kind is
Kind. Core offers this Kind and Package, and nothing else.
`

const corePackageBody = `# Package

A Package Definition makes the folder it lies in a Package, a module of the
project, and carries the folder's name: members/members.md for the Package
members. It says what the Package is responsible for, so that no two Packages
are responsible for the same thing, and what it keeps to itself. It does not
list what the Package contains or depends on.

It is verified like every Definition: the files the Package owns do what it
is responsible for, and they rely on nothing of another Package that no link
of its Definitions declares.
`

// Compile parses and checks a spec with its Libraries. The returned
// Diagnostics are the compile errors, sorted; the Spec is usable for display
// even when there are errors.
func Compile(src Source) (*Spec, Diagnostics) {
	c := &compiler{
		spec: &Spec{
			Definitions:  map[string]*Definition{},
			Packages:     map[string]*Package{RootPackage: {Path: RootPackage}},
			Libraries:    map[string]*Library{},
			Edges:        map[string]map[string][]Use{},
			PlainFolders: map[string]bool{},
		},
		rawKind: map[string]rawKind{},
	}
	c.spec.Definitions[CoreKind] = &Definition{ID: CoreKind, Library: "core", Kind: CoreKind, Title: CoreKind, File: "core/Kind.md", Data: []byte(coreBody), Words: Words([]byte(coreBody))}
	c.spec.Definitions[CorePackage] = &Definition{ID: CorePackage, Library: "core", Kind: CoreKind, Title: CorePackage, File: "core/Package.md", Data: []byte(corePackageBody), Words: Words([]byte(corePackageBody))}
	for _, lib := range src.Libraries {
		c.loadLibrary(lib)
	}
	c.loadSpec(src)
	c.resolve()
	c.checkCycles()
	for _, p := range c.spec.Packages {
		sort.Strings(p.Children)
		sort.Strings(p.Definitions)
	}
	for _, l := range c.spec.Libraries {
		sort.Strings(l.Definitions)
	}
	return c.spec, c.diags.Sorted()
}

type rawKind struct {
	value string
	line  int
}

type compiler struct {
	spec    *Spec
	diags   Diagnostics
	rawKind map[string]rawKind // Definition ID -> kind as written
	raw     map[string][]RawLink
	pages   map[string]parsed // by file, parsed once
}

func (c *compiler) loadLibrary(lib LibrarySource) {
	if !ValidSegment(lib.Name) || lib.Name == CoreKind || lib.Name == CorePackage {
		c.diags.Add("library.name", lib.Dir, 0, "%q cannot name a Library: use letters, digits, - and _, and neither %q nor %q", lib.Name, CoreKind, CorePackage)
		return
	}
	c.spec.Libraries[lib.Name] = &Library{Name: lib.Name, Version: lib.Version}
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
		c.spec.Libraries[lib.Name].Definitions = append(c.spec.Libraries[lib.Name].Definitions, id)
	}
}

func (c *compiler) loadSpec(src Source) {
	// names[folder][lower-case name] collects the spellings and forms (file or
	// folder) of every entry, to find entries that collide.
	type entry struct{ name, form string }
	names := map[string]map[string][]entry{}
	note := func(folder, name, form string) {
		if names[folder] == nil {
			names[folder] = map[string][]entry{}
		}
		key := strings.ToLower(name)
		for _, e := range names[folder][key] {
			if e == (entry{name, form}) {
				return
			}
		}
		names[folder][key] = append(names[folder][key], entry{name, form})
	}
	files := append([]File(nil), src.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	// A folder is a namespace. It is a Package when it holds its anchor: a
	// Definition of kind Package that carries the folder's name.
	anchored := map[string]bool{}
	for _, f := range files {
		id, ok := strings.CutSuffix(f.Path, ".md")
		if !ok || !ValidPath(id) || !c.declaresPackage(path.Join(src.Dir, f.Path), f.Data) {
			continue
		}
		file := path.Join(src.Dir, f.Path)
		switch folder := FolderOf(id); {
		case folder == RootPackage:
			c.diags.Add("spec.package", file, 0, "a Package Definition makes the folder it lies in a Package; the spec folder is the Root Package already")
		case id != AnchorOf(folder):
			c.diags.Add("spec.package", file, 0, "a Package Definition carries the name of its folder, so this one is %s.md", AnchorOf(folder))
		default:
			anchored[folder] = true
		}
	}
	top := map[string]bool{}
	for _, f := range files {
		file := path.Join(src.Dir, f.Path)
		id, isMarkdown := strings.CutSuffix(f.Path, ".md")
		if !isMarkdown {
			c.diags.Add("spec.file", file, 0, "the spec folder holds only Definitions (.md files) and folders")
			continue
		}
		if !ValidPath(id) {
			c.diags.Add("spec.name", file, 0, "%q is not a valid Definition path: every folder and file name starts with a letter or digit and uses only letters, digits, - and _", f.Path)
			continue
		}
		segments := strings.Split(id, "/")
		folder, pkg := RootPackage, RootPackage
		for _, s := range segments[:len(segments)-1] {
			note(folder, s, "folder")
			if folder == RootPackage {
				top[s] = true
			}
			folder = strings.TrimPrefix(folder+"/"+s, "/")
			if !anchored[folder] {
				c.spec.PlainFolders[folder] = true
				continue
			}
			if _, ok := c.spec.Packages[folder]; !ok {
				c.spec.Packages[folder] = &Package{Path: folder, Parent: pkg, Anchor: AnchorOf(folder)}
				c.spec.Packages[pkg].Children = append(c.spec.Packages[pkg].Children, folder)
			}
			pkg = folder
		}
		name := segments[len(segments)-1]
		note(folder, name, "file")
		if folder == RootPackage {
			top[name] = true
		}
		if _, taken := c.spec.Definitions[id]; taken {
			what := "a Library Kind"
			if id == CoreKind || id == CorePackage {
				what = "Core's " + id
			}
			c.diags.Add("spec.reserved", file, 0, "%s is already the ID of %s; choose another name", id, what)
			continue
		}
		d := c.parseDefinition(id, file, f.Data)
		if d == nil {
			continue
		}
		d.Package = pkg
		c.spec.Packages[pkg].Definitions = append(c.spec.Packages[pkg].Definitions, id)
	}
	for folder, byKey := range names {
		for _, entries := range byKey {
			if len(entries) > 1 {
				var spelled []string
				for _, e := range entries {
					spelled = append(spelled, e.form+" "+e.name)
				}
				sort.Strings(spelled)
				c.diags.Add("spec.collision", path.Join(src.Dir, folder), 0, "%s name the same thing; links could not tell them apart", strings.Join(spelled, " and "))
			}
		}
	}
	topNames := make([]string, 0, len(top))
	for name := range top {
		topNames = append(topNames, name)
	}
	sort.Strings(topNames)
	for _, name := range topNames {
		c.reserved(src.Dir, name)
	}
}

// declaresPackage reports whether a Definition file names Core's Package as
// its kind. Front matter that cannot be read is searched for the line itself,
// so that a broken anchor stays one error and its folder a Package.
func (c *compiler) declaresPackage(file string, data []byte) bool {
	page, err := c.page(file, data)
	if err == nil {
		node := page.Front["kind"]
		return node != nil && node.Kind == yaml.ScalarNode && isCorePackage(node.Value)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "kind:"); ok {
			return isCorePackage(strings.Trim(strings.TrimSpace(value), `"'`))
		}
	}
	return false
}

func isCorePackage(kind string) bool {
	return strings.TrimSuffix(strings.TrimSpace(kind), ".md") == CorePackage
}

// page parses a file once: loadSpec reads the kind of every file before it
// reads the Definitions.
func (c *compiler) page(file string, data []byte) (Page, error) {
	if p, ok := c.pages[file]; ok {
		return p.page, p.err
	}
	page, err := ParsePage(data)
	if c.pages == nil {
		c.pages = map[string]parsed{}
	}
	c.pages[file] = parsed{page, err}
	return page, err
}

type parsed struct {
	page Page
	err  error
}

func (c *compiler) reserved(dir, name string) {
	if name == CoreKind || name == CorePackage {
		c.diags.Add("spec.reserved", path.Join(dir, name), 0, "%q is Core's %s; choose another top-level name", name, name)
	}
	if _, ok := c.spec.Libraries[name]; ok {
		c.diags.Add("spec.reserved", path.Join(dir, name), 0, "%q is the name of a Library; links to %s/... would be ambiguous", name, name)
	}
}

// parseDefinition reads one Definition file. It returns nil when the file is
// unusable; the reason is then a diagnostic.
func (c *compiler) parseDefinition(id, file string, data []byte) *Definition {
	page, err := c.page(file, data)
	if err != nil {
		// Keep the Definition, so that links to it still resolve and one
		// broken file reports one error.
		c.diags.Add("definition.front_matter", file, page.FrontLine, "%v", err)
		d := &Definition{ID: id, File: file, Data: data}
		c.spec.Definitions[id] = d
		c.rawKind[id] = rawKind{}
		return d
	}
	d := &Definition{ID: id, File: file, Data: data, Words: Words(page.Body)}
	kind := rawKind{line: max(page.FrontLine, 1)}
	if page.FrontLine == 0 {
		c.diags.Add("definition.front_matter", file, 1, "a Definition starts with front matter that names its kind, such as kind: std/Feature between two lines ---")
	}
	for key, node := range page.Front {
		switch key {
		case "kind":
			if node.Kind != yaml.ScalarNode || node.Tag != "!!str" || strings.TrimSpace(node.Value) == "" {
				c.diags.Add("definition.front_matter", file, node.Line, "kind must be a path such as std/Feature")
				continue
			}
			kind = rawKind{value: strings.TrimSuffix(strings.TrimSpace(node.Value), ".md"), line: node.Line}
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
	c.spec.Definitions[id] = d
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
		d := c.spec.Definitions[id]
		if k := c.rawKind[id]; k.value != "" {
			link := c.resolveLink(d, k.value, k.line, true)
			if link.Target != "" {
				if link.Package || link.Target != CoreKind && link.Target != CorePackage && c.rawKind[link.Target].value != CoreKind {
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
	if !ValidPath(target) {
		c.diags.Add("link.syntax", d.File, line, "%s %q is not a path such as members/member or std/Rule: no leading or trailing /, no .., only letters, digits, - and _ in each name", what, raw)
		return link
	}
	first, _, nested := strings.Cut(target, "/")
	switch {
	case target == CoreKind || target == CorePackage:
		link.Target = target
	case c.spec.Libraries[first] != nil:
		if _, ok := c.spec.Definitions[target]; ok && nested {
			link.Target = target
		} else {
			c.diags.Add(code, d.File, line, "%s %q: Library %s has no Kind of that name", what, raw, first)
			return link
		}
	case d.Library != "":
		c.diags.Add(code, d.File, line, "%s %q: a Library Definition may link only to Libraries the manifest names and to Core's Kind and Package", what, raw)
		return link
	default:
		if def, ok := c.spec.Definitions[target]; ok && def.Own() {
			link.Target = target
		} else if _, ok := c.spec.Packages[target]; ok {
			link.Target, link.Package = target, true
		} else if c.spec.PlainFolders[target] {
			instead := "link a Definition in it or the Package above"
			if kind {
				instead = "name a Kind in it"
			}
			if own := c.spec.Definitions[AnchorOf(target)]; own != nil && own.Own() && !kind {
				// A thing with parts: the folder holds the Definition that names it.
				c.diags.Add(code, d.File, line, "%s %q names a folder, which is no Package; the Definition of that name is %s", what, raw, AnchorOf(target))
				return link
			}
			c.diags.Add(code, d.File, line, "%s %q names a folder that is no Package: %s; %s", what, raw, c.spec.WhyNoPackage(target), instead)
			return link
		} else if strings.EqualFold(target, CoreKind) || strings.EqualFold(target, CorePackage) {
			c.diags.Add(code, d.File, line, "%s %q resolves to nothing; Core's two Kinds are written %s and %s", what, raw, CoreKind, CorePackage)
			return link
		} else {
			c.diags.Add(code, d.File, line, "%s %q resolves to no Definition, Package or Library Kind; paths start at the spec folder", what, raw)
			return link
		}
	}
	if d.Library == "" {
		c.checkDirection(d, link)
	}
	return link
}

// checkDirection applies visibility and the Go-like dependency rules to one
// resolved link of a Definition of the spec and records the edge it makes.
func (c *compiler) checkDirection(d *Definition, link Link) {
	to := link.Target
	target := c.spec.Definitions[link.Target]
	if !link.Package {
		if !target.Own() {
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
	if c.spec.Edges[from] == nil {
		c.spec.Edges[from] = map[string][]Use{}
	}
	c.spec.Edges[from][to] = append(c.spec.Edges[from][to], Use{From: d.ID, Target: link.Target, Kind: link.Kind, File: d.File, Line: link.Line})
}
