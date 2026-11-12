package spec

import (
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// DirectParts is the folder name under which a Kind declares the parts that
// lie directly in a thing's folder, beside the Definition of its name.
const DirectParts = "."

// Schema is what a Kind declares about the Definitions of that Kind: their
// labelled fields and the folders of their parts. What a Kind does not
// declare it leaves to its text, and the compiler does not check it.
type Schema struct {
	// Fields are the fields a Definition of the Kind may have, in the order
	// declared. HasFields tells a Kind that declares that there are none
	// from a Kind that declares nothing about fields.
	Fields    []FieldDecl
	HasFields bool
	// Parts are the folders a thing of the Kind has its parts in.
	Parts    []PartDecl
	HasParts bool
	// Links are the Kinds a link of a Definition of the Kind may name, in
	// its own sentences and in every field that does not say for itself;
	// Limited is false when the Kind does not say.
	Links   []string
	Limited bool
	// Unrealized says that no file realizes a Definition of the Kind: it
	// states something given or wanted, which nothing is built for.
	Unrealized bool
	// Binds says that a Definition of the Kind binds its Package and every
	// Package below it without being linked, as a Rule does.
	Binds bool
	// Last says that a Definition of the Kind is built and verified last: it
	// is judged on the whole project, as a quality is.
	Last bool
	// Broken says that the declaration has an error of its own: Definitions
	// are then not held to it, so that one error stays one.
	Broken bool
}

// FieldDecl is one field a Kind declares.
type FieldDecl struct {
	Name     string
	Required bool
	// Links are the Kinds a link in the field may name. Limited is false
	// when the Kind does not say, and then any link may stand there.
	Links   []string
	Limited bool
}

// PartDecl is one folder of parts a Kind declares, with the Kinds of what
// lies in it. Folder is a folder name, or DirectParts.
type PartDecl struct {
	Folder string
	Kinds  []string
}

// Field returns the declaration of a field, or nil.
func (s *Schema) Field(name string) *FieldDecl {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	return nil
}

// Part returns the declaration of a folder of parts, or nil.
func (s *Schema) Part(folder string) *PartDecl {
	for i := range s.Parts {
		if s.Parts[i].Folder == folder {
			return &s.Parts[i]
		}
	}
	return nil
}

// Field is one labelled field of a Definition's text as written.
type Field struct {
	Label string
	Line  int
}

// schemaRef is one Kind a declaration names, to check once every Definition
// is read.
type schemaRef struct {
	raw, file string
	line      int
	schema    *Schema
}

// yamlBool reads a YAML boolean, however it is capitalised.
func yamlBool(n *yaml.Node) (value, ok bool) {
	if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" {
		return false, false
	}
	return value, n.Decode(&value) == nil
}

// parseSchema reads what a Kind declares in its front matter: fields, a
// mapping of labels, parts, a mapping of folders, links, the Kinds a link may
// name, and realized, which is false for a Kind whose Definitions no file
// realizes, binds, which is true for a Kind whose Definitions bind their
// Package and those below it without a link, and last, which is true for a
// Kind whose Definitions are judged on the whole project. Lines of the top nodes are file
// lines already; the lines of nodes below count from the first front matter
// line.
func (c *compiler) parseSchema(file string, frontLine int, fields, parts, links, realized, binds, last *yaml.Node) *Schema {
	s := &Schema{}
	line := func(n *yaml.Node) int {
		if n == fields || n == parts || n == links || n == realized || n == binds || n == last {
			return n.Line
		}
		return n.Line + frontLine
	}
	bad := func(n *yaml.Node, format string, args ...any) {
		s.Broken = true
		c.diags.Add("kind.schema", file, line(n), format, args...)
	}
	// kinds reads one Kind path or a list of them.
	kinds := func(n *yaml.Node, what string) []string {
		items := []*yaml.Node{n}
		if n.Kind == yaml.SequenceNode {
			items = n.Content
		}
		var out []string
		for _, item := range items {
			raw := strings.TrimSuffix(strings.TrimSpace(item.Value), ".md")
			if item.Kind != yaml.ScalarNode || item.Tag != "!!str" || !ValidPath(raw) {
				bad(item, "%s names Kinds by path, such as std/Rule or a list [std/Rule, std/Goal]", what)
				continue
			}
			out = append(out, raw)
			c.schemaRefs = append(c.schemaRefs, schemaRef{raw: raw, file: file, line: line(item), schema: s})
		}
		return out
	}
	if links != nil {
		s.Limited = true
		if links.Tag != "!!null" {
			s.Links = kinds(links, "links")
		}
	}
	if binds != nil {
		value, ok := yamlBool(binds)
		if !ok {
			bad(binds, "binds is true or false: true for a Kind whose Definitions bind their Package and those below it without a link")
		}
		s.Binds = ok && value
	}
	if last != nil {
		value, ok := yamlBool(last)
		if !ok {
			bad(last, "last is true or false: true for a Kind whose Definitions are built and verified last, on the whole project")
		}
		s.Last = ok && value
	}
	if realized != nil {
		value, ok := yamlBool(realized)
		if !ok {
			bad(realized, "realized is true or false: false for a Kind whose Definitions no file realizes")
		}
		s.Unrealized = ok && !value
	}
	if fields != nil {
		s.HasFields = true
		if fields.Kind != yaml.MappingNode && fields.Tag != "!!null" {
			bad(fields, "fields is a mapping of field labels, such as Satisfies: {required: true, links: [std/Goal]}")
		}
		for i := 0; fields.Kind == yaml.MappingNode && i+1 < len(fields.Content); i += 2 {
			key, value := fields.Content[i], fields.Content[i+1]
			decl := FieldDecl{Name: strings.TrimSpace(key.Value)}
			if key.Kind != yaml.ScalarNode || decl.Name == "" || strings.ContainsAny(decl.Name, "*:\n") {
				bad(key, "a field is named by its label as written between ** and :, such as Satisfies")
				continue
			}
			if s.Field(decl.Name) != nil {
				bad(key, "the field %q is declared twice", decl.Name)
				continue
			}
			switch {
			case value.Tag == "!!null":
			case value.Kind == yaml.MappingNode:
				said := map[string]bool{}
				for j := 0; j+1 < len(value.Content); j += 2 {
					k, v := value.Content[j], value.Content[j+1]
					if said[k.Value] {
						bad(k, "the field %q says %s twice", decl.Name, k.Value)
						continue
					}
					said[k.Value] = true
					switch k.Value {
					case "required":
						required, ok := yamlBool(v)
						if !ok {
							bad(v, "required is true or false")
							continue
						}
						decl.Required = required
					case "links":
						decl.Limited = true
						if v.Tag != "!!null" {
							decl.Links = kinds(v, "links")
						}
					default:
						bad(k, "a field declares only required and links, not %q", k.Value)
					}
				}
			default:
				bad(value, "the field %q is declared with nothing after it, or with {required: true, links: [std/Goal]}", decl.Name)
			}
			s.Fields = append(s.Fields, decl)
		}
	}
	if parts != nil {
		s.HasParts = true
		if parts.Kind != yaml.MappingNode && parts.Tag != "!!null" {
			bad(parts, "parts is a mapping of folders to the Kinds of what lies in them, such as conditions: std/Condition")
		}
		for i := 0; parts.Kind == yaml.MappingNode && i+1 < len(parts.Content); i += 2 {
			key, value := parts.Content[i], parts.Content[i+1]
			folder := strings.TrimSpace(key.Value)
			if key.Kind != yaml.ScalarNode || folder != DirectParts && !ValidSegment(folder) {
				bad(key, "parts are declared by the folder they lie in, such as conditions, or %q for the thing's own folder", DirectParts)
				continue
			}
			if s.Part(folder) != nil {
				bad(key, "the folder %q is declared twice", folder)
				continue
			}
			s.Parts = append(s.Parts, PartDecl{Folder: folder, Kinds: kinds(value, "parts")})
		}
	}
	return s
}

// checkSchemaRefs checks that every Kind a declaration names is a Kind.
func (c *compiler) checkSchemaRefs() {
	for _, r := range c.schemaRefs {
		if r.raw == CoreKind || r.raw == CorePackage {
			continue
		}
		if _, ok := c.spec.Definitions[r.raw]; !ok || c.rawKind[r.raw].value != CoreKind {
			r.schema.Broken = true
			c.diags.Add("kind.schema", r.file, r.line, "%q names no Kind; a declaration names each Kind by its full path, such as std/Rule", r.raw)
		}
	}
}

// checkSchemas holds every Definition of the spec against what its Kind
// declares, and every part against what the Kind of its whole declares.
func (c *compiler) checkSchemas() {
	for _, id := range c.spec.OwnDefinitions() {
		d := c.spec.Definitions[id]
		if s := c.schemaOf(d); s != nil {
			if s.HasFields {
				c.checkFields(d, s)
			}
			c.checkLinks(d, s)
		}
		whole := c.spec.Definitions[d.Whole]
		if whole == nil {
			continue
		}
		s := c.schemaOf(whole)
		if s == nil || !s.HasParts {
			continue
		}
		folder := PartFolder(id, whole.ID)
		decl := s.Part(folder)
		switch {
		case decl == nil:
			c.diags.Add("part.folder", d.File, 0, "%s is a part of %s, and a %s has parts only %s", id, thingName(whole.ID), whole.Kind, partPlaces(s))
		case d.Kind != "" && !contains(decl.Kinds, d.Kind):
			c.diags.Add("part.kind", d.File, c.rawKind[id].line, "what lies %s of a %s is %s; this is a %s", partPlace(folder), whole.Kind, oneOf(decl.Kinds), d.Kind)
		}
	}
}

func (c *compiler) checkFields(d *Definition, s *Schema) {
	seen := map[string]bool{}
	for _, f := range d.Fields {
		switch {
		case s.Field(f.Label) == nil:
			c.diags.Add("field.unknown", d.File, f.Line, "a %s has no field %q; it has %s", d.Kind, f.Label, fieldNames(s))
		case seen[f.Label]:
			c.diags.Add("field.repeated", d.File, f.Line, "the field %q stands twice; state it once", f.Label)
		}
		seen[f.Label] = true
	}
	for _, decl := range s.Fields {
		if decl.Required && !seen[decl.Name] {
			c.diags.Add("field.missing", d.File, c.rawKind[d.ID].line, "a %s states %q: add a line that starts with **%s:**", d.Kind, decl.Name, decl.Name)
		}
	}
}

// checkLinks holds the links of a Definition to what its Kind declares: a
// link in a field that names its Kinds to those, and every other link to the
// Kinds the Kind names for all of them.
func (c *compiler) checkLinks(d *Definition, s *Schema) {
	for _, l := range d.Links {
		if l.Kind || l.Target == "" {
			continue
		}
		kind := CorePackage
		if !l.Package {
			kind = c.spec.Definitions[l.Target].Kind
		}
		if kind == "" {
			continue
		}
		switch decl := s.Field(l.Field); {
		case decl != nil && decl.Limited:
			if !contains(decl.Links, kind) {
				c.diags.Add("field.target", d.File, l.Line, "%q of a %s links %s; %s is a %s", l.Field, d.Kind, oneOf(decl.Links), l.Target, kind)
			}
		case s.Limited && !contains(s.Links, kind):
			c.diags.Add("link.target", d.File, l.Line, "a %s links %s; %s is a %s", d.Kind, oneOf(s.Links), l.Target, kind)
		}
	}
}

// schemaOf returns what the Kind of a Definition declares, or nil.
func (c *compiler) schemaOf(d *Definition) *Schema {
	if k := c.spec.Definitions[d.Kind]; k != nil && k.Schema != nil && !k.Schema.Broken {
		return k.Schema
	}
	return nil
}

// declared returns what the Kind of a Definition declares, or nil: nil too
// when the declaration has an error of its own, which holds no Definition.
func (s *Spec) declared(d *Definition) *Schema {
	if k := s.Definitions[d.Kind]; k != nil && k.Schema != nil && !k.Schema.Broken {
		return k.Schema
	}
	return nil
}

// VerifiedLast says whether a Definition is built and verified last, on the
// whole project: its Kind declares last, or declares that no file realizes
// its Definitions, which can then only be judged on what the others built.
func (s *Spec) VerifiedLast(id string) bool {
	d := s.Definitions[id]
	if d == nil {
		return false
	}
	schema := s.declared(d)
	return schema != nil && (schema.Last || schema.Unrealized)
}

// Binding returns, by Package path, what binds the Package and every Package
// below it without a link (DEC-110, DEC-113): each unit whose Definition lies
// in the Package itself and whose Kind declares binds, sorted. A Package that
// has none is left out.
func (s *Spec) Binding() map[string][]string {
	out := map[string][]string{}
	for id := range s.Units() {
		d := s.Definitions[id]
		if schema := s.declared(d); schema != nil && schema.Binds {
			out[d.Package] = append(out[d.Package], id)
		}
	}
	for _, ids := range out {
		sort.Strings(ids)
	}
	return out
}

// Bound says whether what binds a Definition's Package binds the Definition:
// whether a file realizes the unit it belongs to, so that what is said of what
// is built holds for it (DEC-113). Nothing binds a Definition of a Kind that
// declares that no file realizes it, such as a Goal, nor a Kind.
func (s *Spec) Bound(id string) bool {
	d := s.Definitions[s.UnitOf(id)]
	if d == nil || !d.Own() || d.Kind == CoreKind {
		return false
	}
	schema := s.declared(d)
	return schema == nil || !schema.Unrealized
}

// PartFolder returns the folder a part lies in below its whole, as a Kind
// declares it: conditions for lend/conditions/in-stock, and DirectParts for
// what lies beside the whole. A part with parts of its own lies in a folder
// of its name, which does not count.
func PartFolder(id, whole string) string {
	folder := FolderOf(strings.TrimPrefix(id, FolderOf(whole)+"/"))
	if id == AnchorOf(FolderOf(id)) {
		folder = FolderOf(folder)
	}
	if folder == "" {
		return DirectParts
	}
	first, _, _ := strings.Cut(folder, "/")
	return first
}

func partPlace(folder string) string {
	if folder == DirectParts {
		return "directly in the folder"
	}
	return "under " + folder + "/"
}

func partPlaces(s *Schema) string {
	if len(s.Parts) == 0 {
		return "nowhere: it has none"
	}
	var places []string
	for _, p := range s.Parts {
		places = append(places, partPlace(p.Folder))
	}
	return strings.Join(places, ", ")
}

func fieldNames(s *Schema) string {
	if len(s.Fields) == 0 {
		return "none"
	}
	var names []string
	for _, f := range s.Fields {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}

// thingName names a Definition as a link does: a thing with parts by its
// folder.
func thingName(id string) string {
	if folder := FolderOf(id); folder != RootPackage && id == AnchorOf(folder) {
		return folder
	}
	return id
}

func oneOf(kinds []string) string {
	switch len(kinds) {
	case 0:
		return "nothing"
	case 1:
		return "a " + kinds[0]
	}
	sorted := append([]string(nil), kinds...)
	sort.Strings(sorted)
	return "one of " + strings.Join(sorted, ", ")
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
