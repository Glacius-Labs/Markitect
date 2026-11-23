package lang

import (
	"fmt"
	"strings"
)

// Chain returns the type t followed by the types up its chain, the nearest
// first: t, its base, its base's base, and so on. The chain ends at a type
// without a valid extends line, at a base that is no document and where it
// comes back to a type it already holds, so every type stands in it once.
func (s *Spec) Chain(t *Document) []*Document {
	chain := []*Document{t}
	seen := map[string]bool{t.Path: true}
	for cur := t; cur.HasExtends(); {
		base, ok := s.Document(cur.ExtendsPath)
		if !ok || seen[base.Path] {
			break
		}
		seen[base.Path] = true
		chain = append(chain, base)
		cur = base
	}
	return chain
}

// ChainSection is one section of a type, declared by the type or by a type up
// its chain.
type ChainSection struct {
	// Name is the heading without its "(optional)" marker, as the most derived
	// type that declares the section writes it.
	Name string
	// Optional is false when any type of the chain requires the section.
	Optional bool
	// DeclaredBy is the path of the most derived type that declares the
	// section, and RequiredBy that of the most derived type that requires it,
	// "" for an optional section.
	DeclaredBy, RequiredBy string
}

// ChainSections returns the sections of the type t: its own and those of every
// type up its chain. A section declared at more than one level, in any case,
// counts once: it is required when any level requires it, and the most derived
// level that declares it describes it. The base's sections come first, each
// level in its own order, then what the derived types add.
func (s *Spec) ChainSections(t *Document) []ChainSection {
	chain := s.Chain(t)
	var out []ChainSection
	for i := len(chain) - 1; i >= 0; i-- {
		level := chain[i]
	next:
		for _, d := range level.Declared() {
			for k := range out {
				if strings.EqualFold(out[k].Name, d.Name) {
					out[k].Name, out[k].DeclaredBy = d.Name, level.Path
					if !d.Optional {
						out[k].Optional, out[k].RequiredBy = false, level.Path
					}
					continue next
				}
			}
			c := ChainSection{Name: d.Name, Optional: d.Optional, DeclaredBy: level.Path}
			if !d.Optional {
				c.RequiredBy = level.Path
			}
			out = append(out, c)
		}
	}
	return out
}

// IsType reports whether d is a type: a document that another document names
// in its type line or its extends line, or one whose extends line is a link.
func (s *Spec) IsType(d *Document) bool { return s.types[d.Path] }

// Objects returns the documents whose type is t or a type that extends t,
// directly or further down, in path order.
func (s *Spec) Objects(t *Document) []*Document {
	var out []*Document
	for _, d := range s.Documents {
		if !d.HasType() {
			continue
		}
		typ, ok := s.Document(d.TypePath)
		if !ok {
			continue
		}
		for _, c := range s.Chain(typ) {
			if c == t {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

// inCycle reports whether the chain of d comes back to d.
func (s *Spec) inCycle(d *Document) bool {
	seen := map[string]bool{d.Path: true}
	for cur := d; cur.HasExtends(); {
		base, ok := s.Document(cur.ExtendsPath)
		if !ok {
			return false
		}
		if base == d {
			return true
		}
		if seen[base.Path] {
			return false
		}
		seen[base.Path] = true
		cur = base
	}
	return false
}

// bothLines is the message of the second of a type line and an extends line in
// the head block.
const bothLines = "a document has a type line or an extends line, not both"

// inHead reports whether a line stands in the head block of the document.
func (d *Document) inHead(line int) bool {
	return d.HeadFirst != 0 && line >= d.HeadFirst && line <= d.HeadLast
}

// checkExtends checks the extends line of a type: the line, its base, the
// cycle and the sections the type loosens.
func (s *Spec) checkExtends(d *Document) []Diagnostic {
	add := func(line int, code, format string, args ...any) Diagnostic {
		return Diagnostic{Path: d.Path, Line: line, Code: code, Message: fmt.Sprintf(format, args...)}
	}
	var out []Diagnostic
	switch {
	case d.ExtendsLine != 0 && !d.HasExtends():
		out = append(out, add(d.ExtendsLine, CodeExtendsLine, "extends line must be exactly one link to a document"))
	case d.HasExtends():
		if _, ok := s.Document(d.ExtendsPath); !ok {
			out = append(out, add(d.ExtendsLine, CodeExtendsUnresolved, "no document at \"%s\"", d.ExtendsRaw))
			break
		}
		if s.inCycle(d) {
			out = append(out, add(d.ExtendsLine, CodeExtendsCycle, "the chain of extends lines comes back to itself"))
		}
		out = append(out, s.loosened(d)...)
	}
	for _, line := range d.StrayExtendsLines {
		if d.TypeLine != 0 && d.inHead(line) {
			out = append(out, add(line, CodeExtendsLine, bothLines))
		} else {
			out = append(out, add(line, CodeExtendsLine, "an extends line stands only on the first line after the title"))
		}
	}
	return out
}

// loosened reports each section that the type d marks optional and a type up
// its chain requires, at d's heading of the section.
func (s *Spec) loosened(d *Document) []Diagnostic {
	above := s.Chain(d)[1:]
	var out []Diagnostic
	for _, own := range d.Declared() {
		if !own.Optional {
			continue
		}
		for _, up := range above {
			if !requires(up, own.Name) {
				continue
			}
			line := d.ExtendsLine
			for _, sec := range d.Sections {
				if name, _ := cutOptional(sec.Heading); strings.EqualFold(name, own.Name) {
					line = sec.Line
					break
				}
			}
			out = append(out, Diagnostic{Path: d.Path, Line: line, Code: CodeExtendsLoosens,
				Message: fmt.Sprintf("section \"%s\" is required by %s", own.Name, up.Path)})
			break
		}
	}
	return out
}

// requires reports whether the type t declares the section name as required.
func requires(t *Document, name string) bool {
	for _, d := range t.Declared() {
		if !d.Optional && strings.EqualFold(d.Name, name) {
			return true
		}
	}
	return false
}
