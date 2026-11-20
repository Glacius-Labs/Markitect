package lang

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Spec is the documents of one spec folder, sorted by path.
type Spec struct {
	Documents []*Document
	byPath    map[string]*Document
}

// NewSpec indexes documents by path and sorts them.
func NewSpec(docs []*Document) *Spec {
	s := &Spec{Documents: append([]*Document(nil), docs...), byPath: make(map[string]*Document, len(docs))}
	sort.SliceStable(s.Documents, func(i, j int) bool { return s.Documents[i].Path < s.Documents[j].Path })
	for _, d := range s.Documents {
		s.byPath[d.Path] = d
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
)

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

// Check checks every document of the spec: its title, its type line, the
// sections its type requires and its links. The result is sorted by path,
// then by line.
func Check(s *Spec) []Diagnostic {
	var out []Diagnostic
	for _, d := range s.Documents {
		add := func(line int, code, format string, args ...any) {
			out = append(out, Diagnostic{Path: d.Path, Line: line, Code: code, Message: fmt.Sprintf(format, args...)})
		}
		switch {
		case d.TitleLine == 0:
			add(max(d.ContentLine, 1), CodeTitle, "missing level-1 title")
		case d.SecondTitleLine != 0:
			add(d.SecondTitleLine, CodeTitle, "second level-1 heading")
		}
		switch {
		case d.TypeLine != 0 && !d.HasType():
			add(d.TypeLine, CodeTypeLine, "type line must be exactly one link to a document")
		case d.HasType():
			t, ok := s.Document(d.TypePath)
			if !ok {
				add(d.TypeLine, CodeTypeUnresolved, "no document at \"%s\"", d.TypeRaw)
				break
			}
			for _, want := range t.Declared() {
				sec, has := d.Section(want.Name)
				switch {
				case !has && !want.Optional:
					add(d.TypeLine, CodeSectionMissing, "missing section \"%s\" required by %s", want.Name, t.Path)
				case has && !sec.HasText:
					add(sec.Line, CodeSectionEmpty, "empty section \"%s\"", want.Name)
				}
			}
		}
		for _, line := range d.StrayTypeLines {
			add(line, CodeTypeLine, "a type line stands only on the first line after the title")
		}
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
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Line < out[j].Line
	})
	return out
}
