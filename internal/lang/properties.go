package lang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// A property is a section of a type. Its type is what the property type line
// says, the line starting with "Type:" first under the section's heading: links
// to types make it a reference to documents of one of these types or of a type
// that extends one, code spans make it one of fixed values, and no line makes it
// text. The type of a property comes down the chain of extends lines like its
// requiredness.

const (
	// The codes of the checks of properties.
	CodePropertyLine       = "property.line"
	CodePropertyUnresolved = "property.unresolved"
	CodePropertyRetyped    = "property.retyped"
	CodePropertyValue      = "property.value"
	CodeReferenceType      = "reference.type"
)

// readProperties reads the property type line of every level-2 heading: the
// first line of the block right after the heading, when that block is a
// paragraph and the line starts with "Type:". The line is marked as read, so
// that it is neither a stray type line, nor a link, nor prose. A heading of a
// name that an earlier heading has already, in any case, declares nothing: a
// line under it is no property type line but a stray type line.
func (p *parser) readProperties(root ast.Node) {
	p.properties = map[int]PropertyLine{}
	var declared []string
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		h, ok := n.(*ast.Heading)
		if !ok || h.Level != 2 {
			continue
		}
		name := SectionName(p.headingText(h))
		again := false
		for _, d := range declared {
			again = again || strings.EqualFold(d, name)
		}
		if again {
			continue
		}
		declared = append(declared, name)
		para, ok := nextBlock(n.NextSibling()).(*ast.Paragraph)
		if !ok {
			continue
		}
		seg := para.Lines().At(0)
		line := p.lineText(seg)
		if !strings.HasPrefix(line, typePrefix) {
			continue
		}
		at := p.line(seg.Start)
		p.headLines[at] = true
		pl := p.propertyLine(strings.TrimSpace(line[len(typePrefix):]))
		pl.Line = at
		p.properties[p.line(h.Pos())] = pl
	}
}

// propertyLine reads what follows "Type:" on a property type line: one or more
// links to .md paths, or one or more code spans, separated by commas, and
// nothing else. Anything else gives a line without a kind.
func (p *parser) propertyLine(rest string) PropertyLine {
	src := []byte(rest)
	root := markdown.Parser().Parse(text.NewReader(src))
	para, ok := root.FirstChild().(*ast.Paragraph)
	if !ok || para.NextSibling() != nil {
		return PropertyLine{}
	}
	var out PropertyLine
	item := true
	for c := para.FirstChild(); c != nil; c = c.NextSibling() {
		if !item {
			t, ok := c.(*ast.Text)
			if !ok || strings.TrimSpace(string(t.Segment.Value(src))) != "," {
				return PropertyLine{}
			}
			item = true
			continue
		}
		switch c := c.(type) {
		case *ast.Link:
			raw, ok := localPath(string(c.Destination))
			if !ok || out.Kind == PropertyValues {
				return PropertyLine{}
			}
			out.Kind = PropertyRefs
			out.Refs = append(out.Refs, PropertyRef{Raw: raw, Target: resolve(p.doc.Path, raw)})
		case *ast.CodeSpan:
			value := codeText(c, src)
			if strings.TrimSpace(value) == "" || out.Kind == PropertyRefs {
				return PropertyLine{}
			}
			out.Kind = PropertyValues
			out.Values = append(out.Values, value)
		default:
			return PropertyLine{}
		}
		item = false
	}
	if item {
		return PropertyLine{}
	}
	return out
}

// holdsProperties reports whether the property type lines of the document are
// read: a README has none, and neither has a document with a type line, an
// object, whose lines are errors.
func (d *Document) holdsProperties() bool { return !isReadme(d.Path) && d.TypeLine == 0 }

// SectionName returns the name a type declares with a heading: the heading
// without its "(optional)" marker.
func SectionName(heading string) string {
	name, _ := cutOptional(heading)
	return name
}

// propertyLine returns the property type line of the section name in the
// document, when the first section of that name has a valid one and the
// document is of a kind that holds such lines. A later section of the name
// declares nothing.
func (d *Document) propertyLine(name string) (PropertyLine, bool) {
	if !d.holdsProperties() {
		return PropertyLine{}, false
	}
	for _, sec := range d.Sections {
		if strings.EqualFold(SectionName(sec.Heading), name) {
			return sec.Property, sec.Property.Kind != ""
		}
	}
	return PropertyLine{}, false
}

// declares reports whether the document declares a section of that name.
func (d *Document) declares(name string) bool {
	for _, sec := range d.Sections {
		if strings.EqualFold(SectionName(sec.Heading), name) {
			return true
		}
	}
	return false
}

// Property is the type of a property of a type.
type Property struct {
	// Kind is PropertyRefs for a reference, PropertyValues for fixed values, and
	// "" for text.
	Kind string
	// Refs are the paths of the types a reference may point to, as written on
	// the line, resolved below the spec folder.
	Refs []string
	// Values are the fixed values, in order.
	Values []string
	// From is the path of the type whose line gives the type, "" for text.
	From string
}

// PropertyType returns the type of the property name of the type t: the line of
// the nearest type of its chain that has one for a section of that name, text
// when none has. A type that declares the section again without a line keeps
// the type of the one above.
func (s *Spec) PropertyType(t *Document, name string) Property {
	return propertyOf(s.Chain(t), name)
}

func propertyOf(chain []*Document, name string) Property {
	for _, level := range chain {
		if pl, ok := level.propertyLine(name); ok {
			prop := Property{Kind: pl.Kind, Values: pl.Values, From: level.Path}
			for _, r := range pl.Refs {
				prop.Refs = append(prop.Refs, r.Target)
			}
			return prop
		}
	}
	return Property{}
}

// same reports whether two property types are the same: text, the same set of
// types, or the same set of values.
func (p Property) same(o Property) bool {
	if p.Kind != o.Kind {
		return false
	}
	return strings.Join(uniqueSorted(p.Refs), "\x00") == strings.Join(uniqueSorted(o.Refs), "\x00") &&
		strings.Join(uniqueSorted(p.Values), "\x00") == strings.Join(uniqueSorted(o.Values), "\x00")
}

func uniqueSorted(list []string) []string {
	out := append([]string(nil), list...)
	sort.Strings(out)
	n := 0
	for i, x := range out {
		if i == 0 || x != out[i-1] {
			out[n] = x
			n++
		}
	}
	return out[:n]
}

// describe says what the property type is, for a message.
func (p Property) describe() string {
	switch p.Kind {
	case PropertyRefs:
		return "a reference to " + strings.Join(p.Refs, ", ")
	case PropertyValues:
		return "one of " + quoteValues(p.Values)
	}
	return "text"
}

func quoteValues(values []string) string { return QuoteValues(values) }

// QuoteValue writes a fixed value as a code span that reads back as the value:
// the delimiter is longer than any run of backticks in it, and a value that
// starts or ends with a backtick is padded with a space at each end.
func QuoteValue(v string) string {
	longest, run := 0, 0
	for _, c := range v {
		if c == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", longest+1)
	if strings.HasPrefix(v, "`") || strings.HasSuffix(v, "`") {
		v = " " + v + " "
	}
	return fence + v + fence
}

// QuoteValues writes fixed values as code spans, separated by commas.
func QuoteValues(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = QuoteValue(v)
	}
	return strings.Join(quoted, ", ")
}

// checkPropertyLines checks the property type lines of a document: where they
// stand, what they say and which documents they name.
func (s *Spec) checkPropertyLines(d *Document) []Diagnostic {
	var out []Diagnostic
	add := func(line int, code, format string, args ...any) {
		out = append(out, Diagnostic{Path: d.Path, Line: line, Code: code, Message: fmt.Sprintf(format, args...)})
	}
	for _, sec := range d.Sections {
		pl := sec.Property
		switch {
		case pl.Line == 0:
		case isReadme(d.Path):
			add(pl.Line, CodePropertyLine, "a README has no property type line")
		case d.TypeLine != 0:
			add(pl.Line, CodePropertyLine, "a document with a type line has no property type line")
		case pl.Kind == "":
			add(pl.Line, CodePropertyLine, "property type line must be links to documents or code spans, separated by commas")
		default:
			seen := map[string]bool{}
			for _, r := range pl.Refs {
				if seen[r.Target] {
					continue
				}
				seen[r.Target] = true
				switch target, ok := s.Document(r.Target); {
				case !ok:
					add(pl.Line, CodePropertyUnresolved, "no document at \"%s\"", r.Raw)
				case isReadme(target.Path):
					add(pl.Line, CodeReadmeType, "\"%s\" is a README, which is no type", target.Path)
				}
			}
		}
	}
	return out
}

// checkRetyped reports each property that a derived type types differently
// than the types up its chain: a line where they have none, or another line.
func (s *Spec) checkRetyped(d *Document) []Diagnostic {
	if !d.HasExtends() || !d.holdsProperties() {
		return nil
	}
	above := s.Chain(d)[1:]
	var out []Diagnostic
	seen := map[string]bool{}
	for _, sec := range d.Sections {
		name := SectionName(sec.Heading)
		key := strings.ToLower(name)
		if sec.Property.Kind == "" || seen[key] {
			continue
		}
		seen[key] = true
		mine := propertyOf([]*Document{d}, name)
		up, from, ok := propertyAbove(above, name)
		if !ok || mine.same(up) {
			continue
		}
		out = append(out, Diagnostic{Path: d.Path, Line: sec.Property.Line, Code: CodePropertyRetyped,
			Message: fmt.Sprintf("property \"%s\" is %s in %s, not %s", name, up.describe(), from, mine.describe())})
	}
	return out
}

// propertyAbove returns the type of the property name that the types above
// declare, and the path of the type that gives it: the nearest line, or text
// from the nearest type that declares the section; ok is false when none does.
func propertyAbove(above []*Document, name string) (Property, string, bool) {
	for _, level := range above {
		if _, ok := level.propertyLine(name); ok {
			prop := propertyOf([]*Document{level}, name)
			return prop, level.Path, true
		}
	}
	for _, level := range above {
		if level.declares(name) {
			return Property{}, level.Path, true
		}
	}
	return Property{}, "", false
}

// declaredTypes returns the types a reference property may point to: the
// documents its line links, but for a link that names no document or a README,
// which is reported where the line stands.
func (s *Spec) declaredTypes(prop Property) []*Document {
	var out []*Document
	for _, p := range prop.Refs {
		if t, ok := s.Document(p); ok && !isReadme(t.Path) {
			out = append(out, t)
		}
	}
	return out
}

// ofAny reports whether the document is an object of one of the types or of a
// type that extends one.
func (s *Spec) ofAny(d *Document, types []*Document) bool {
	if !d.HasType() {
		return false
	}
	typ, ok := s.Document(d.TypePath)
	if !ok {
		return false
	}
	for _, c := range s.Chain(typ) {
		for _, t := range types {
			if c == t {
				return true
			}
		}
	}
	return false
}

// referenceLinks returns the links of the section sec of the object d, which
// are the references of a reference property.
func referenceLinks(d *Document, sec Section) []Link {
	var out []Link
	for _, l := range d.Links {
		if l.SectionLine == sec.Line {
			out = append(out, l)
		}
	}
	return out
}

// checkProperties checks the sections of the object d against the property
// types of its type t and of every type up its chain: the links of a reference
// property are objects of the declared types, and the first line of a section of
// fixed values is one of them.
func (s *Spec) checkProperties(d, t *Document) []Diagnostic {
	var out []Diagnostic
	add := func(line int, code, format string, args ...any) {
		out = append(out, Diagnostic{Path: d.Path, Line: line, Code: code, Message: fmt.Sprintf(format, args...)})
	}
	for _, want := range s.ChainSections(t) {
		prop := s.PropertyType(t, want.Name)
		sec, has := d.Section(want.Name)
		if !has || prop.Kind == "" {
			continue
		}
		switch prop.Kind {
		case PropertyValues:
			if !sec.HasText {
				continue
			}
			found := false
			for _, v := range prop.Values {
				found = found || v == sec.First
			}
			if !found {
				add(max(sec.FirstLine, sec.Line), CodePropertyValue, "the first line of section \"%s\" is \"%s\", not one of %s", want.Name, sec.First, quoteValues(prop.Values))
			}
		case PropertyRefs:
			types := s.declaredTypes(prop)
			if len(types) == 0 {
				continue
			}
			var paths []string
			for _, t := range types {
				paths = append(paths, t.Path)
			}
			seen := map[string]bool{}
			for _, l := range referenceLinks(d, sec) {
				target, ok := s.Document(l.Target)
				if !ok || seen[l.Target] || s.ofAny(target, types) {
					continue
				}
				seen[l.Target] = true
				add(l.Line, CodeReferenceType, "section \"%s\" refers to documents of %s, but \"%s\" is %s", want.Name, typesPhrase(paths), l.Target, s.whatIs(target))
			}
		}
	}
	return out
}

// typesPhrase says which types a reference property accepts.
func typesPhrase(refs []string) string {
	if len(refs) == 1 {
		return refs[0] + " or a type that extends it"
	}
	return "one of " + strings.Join(refs, ", ") + " or a type that extends one of them"
}

// whatIs says what a document is that is no object of the declared types.
func (s *Spec) whatIs(d *Document) string {
	switch {
	case isReadme(d.Path):
		return "a README"
	case d.HasType():
		return "a document of " + d.TypePath
	case s.IsType(d):
		return "a type"
	}
	return "a document without a type"
}
