package lang

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Spec is the documents of one spec folder, sorted by path.
type Spec struct {
	Documents []*Document
	byPath    map[string]*Document
	types     map[string]bool // the paths of the documents that are types
	modules   map[string]bool // the folders that hold a document directly; "." is the spec folder
}

// NewSpec indexes documents by path and sorts them. A type line or an extends
// line that names a README names no type: the document keeps the line and
// HeadReadme holds the path. A type is a document that a type line, an extends
// line or a property type line names, that has an extends line or that has a
// property type line; a README is none, and the property type lines of a
// document with a type line are errors that name nothing.
func NewSpec(docs []*Document) *Spec {
	s := &Spec{Documents: append([]*Document(nil), docs...), byPath: make(map[string]*Document, len(docs)), types: map[string]bool{}, modules: map[string]bool{}}
	sort.SliceStable(s.Documents, func(i, j int) bool { return s.Documents[i].Path < s.Documents[j].Path })
	for _, d := range s.Documents {
		s.byPath[d.Path] = d
		s.modules[path.Dir(d.Path)] = true
	}
	for _, d := range s.Documents {
		d.readHeadReadme(s)
	}
	for _, d := range s.Documents {
		if d.HasType() {
			s.types[d.TypePath] = true
		}
		if d.HasExtends() {
			s.types[d.ExtendsPath] = true
		}
		if d.HasExtends() {
			s.types[d.Path] = true
		}
		if d.holdsProperties() {
			for _, sec := range d.Sections {
				if sec.Property.Kind == "" {
					continue
				}
				s.types[d.Path] = true
				for _, r := range sec.Property.Refs {
					if !isReadmeDocument(s, r.Target) {
						s.types[r.Target] = true
					}
				}
			}
		}
	}
	return s
}

// Document returns the document at a path below the spec folder. Paths are
// compared exactly, case included.
func (s *Spec) Document(p string) (*Document, bool) {
	d, ok := s.byPath[p]
	return d, ok
}

// Load reads every document of the spec at the root of fsys: every file at
// any depth whose name ends in .md, in any case, after at least one other
// character. Other files are ignored.
func Load(fsys fs.FS) (*Spec, error) {
	var docs []*Document
	err := fs.WalkDir(fsys, ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() || !IsDocumentName(e.Name()) {
			return nil
		}
		if !e.Type().IsRegular() {
			info, err := fs.Stat(fsys, p)
			if err != nil || !info.Mode().IsRegular() {
				return nil
			}
		}
		src, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		docs = append(docs, Parse(p, src))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return NewSpec(docs), nil
}

// IsDocumentName reports whether a file name names a document: it ends in .md,
// in any case, and something other than a leading dot comes before it.
func IsDocumentName(name string) bool {
	i := strings.LastIndexByte(name, '.')
	return i > 0 && strings.EqualFold(name[i:], ".md")
}

// The codes of Check.
const (
	CodeTitle          = "title"
	CodeTypeUnresolved = "type.unresolved"
	CodeTypeLine       = "type.line"
	CodeSectionMissing = "section.missing"
	CodeSectionEmpty   = "section.empty"
	CodeLinkUnresolved = "link.unresolved"
	CodeLinkInternal   = "link.internal"
	CodeModuleReadme   = "module.readme"
	CodeReadmeType     = "readme.type"
	CodeReadmePattern  = "readme.pattern"

	CodeExtendsUnresolved = "extends.unresolved"
	CodeExtendsLine       = "extends.line"
	CodeExtendsCycle      = "extends.cycle"
	CodeExtendsLoosens    = "extends.loosens"
)

// checkTitle checks that the document starts with one level-1 title.
func checkTitle(d *Document) []Diagnostic {
	switch {
	case d.TitleLine == 0:
		return []Diagnostic{{Path: d.Path, Line: max(d.ContentLine, 1), Code: CodeTitle, Message: "missing level-1 title"}}
	case d.SecondTitleLine != 0:
		return []Diagnostic{{Path: d.Path, Line: d.SecondTitleLine, Code: CodeTitle, Message: "second level-1 heading"}}
	}
	return nil
}

// checkSections checks the sections of the type t, its own and those of every
// type up its chain: a required one missing is reported at line, and one the
// document has must hold text.
func (s *Spec) checkSections(d, t *Document, line int) []Diagnostic {
	var out []Diagnostic
	for _, want := range s.ChainSections(t) {
		sec, has := d.Section(want.Name)
		switch {
		case !has && !want.Optional:
			out = append(out, Diagnostic{Path: d.Path, Line: line, Code: CodeSectionMissing, Message: fmt.Sprintf("missing section \"%s\" required by %s", want.Name, want.RequiredBy)})
		case has && !sec.HasText:
			out = append(out, Diagnostic{Path: d.Path, Line: sec.Line, Code: CodeSectionEmpty, Message: fmt.Sprintf("empty section \"%s\"", want.Name)})
		}
	}
	return out
}

// Diagnostic is one finding of Check, at a line of a document.
type Diagnostic struct {
	Path    string
	Line    int
	Code    string
	Message string
}

// String renders the diagnostic as path:line: code: message.
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", d.Path, d.Line, d.Code, d.Message)
}

// Check checks every document of the spec: its title, its type line or
// extends line, the sections its type requires and its links, and every module
// of it: its README and what its links may reach. The result is sorted by path,
// then by line.
func Check(s *Spec) []Diagnostic {
	out := s.checkModules()
	shown := s.shownByModule()
	for _, d := range s.Documents {
		add := func(line int, code, format string, args ...any) {
			out = append(out, Diagnostic{Path: d.Path, Line: line, Code: code, Message: fmt.Sprintf(format, args...)})
		}
		out = append(out, checkTitle(d)...)
		switch {
		case d.TypeLine != 0 && !d.HasType() && d.HeadReadme == "":
			add(d.TypeLine, CodeTypeLine, "type line must be exactly one link to a document")
		case d.HasType():
			t, ok := s.Document(d.TypePath)
			if !ok {
				add(d.TypeLine, CodeTypeUnresolved, "no document at \"%s\"", d.TypeRaw)
				break
			}
			out = append(out, s.checkSections(d, t, d.TypeLine)...)
			out = append(out, s.checkProperties(d, t)...)
		}
		for _, line := range d.StrayTypeLines {
			if d.ExtendsLine != 0 && d.inHead(line) {
				add(line, CodeTypeLine, bothLines)
			} else {
				add(line, CodeTypeLine, "a type line stands only on the first line after the title")
			}
		}
		out = append(out, s.checkExtends(d)...)
		out = append(out, s.checkPropertyLines(d)...)
		out = append(out, s.checkRetyped(d)...)
		out = append(out, s.checkReadmeType(d)...)
		out = append(out, checkPatterns(d)...)
		out = append(out, checkRealized(d)...)
		out = append(out, s.checkInternal(d, shown)...)
		seen := map[string]bool{}
		for _, l := range d.Links {
			if seen[l.Target] {
				continue
			}
			seen[l.Target] = true
			if _, ok := s.Document(l.Target); !ok {
				add(l.Line, CodeLinkUnresolved, "no document at \"%s\"", l.Raw)
			}
		}
	}
	out = append(out, s.checkModuleCycles()...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Line < out[j].Line
	})
	return out
}
