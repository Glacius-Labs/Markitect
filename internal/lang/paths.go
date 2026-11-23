package lang

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// A path pattern names files of a project by their path below the project
// folder, with / between folders. Each folder name and the file name is matched
// as path.Match matches a name: * for any run of characters, ? for one
// character and a class such as [a-z], none of them across a /; everything
// else, case included, matches itself. A pattern that ends with / names a
// folder and everything below it.

// PathPattern is a path pattern that is well-formed.
type PathPattern struct {
	// Text is the pattern as written.
	Text string
	// Segments are the names between the slashes, each a path.Match pattern.
	Segments []string
	// Folder is whether the pattern ended with /.
	Folder bool
}

// ParsePathPattern reads a path pattern. It refuses one that is empty, has a
// backslash, a drive, is absolute, has an empty or a . or .. folder name, or
// holds a segment that path.Match calls malformed; the error says which.
func ParsePathPattern(text string) (PathPattern, error) {
	switch {
	case strings.TrimSpace(text) == "":
		return PathPattern{}, errors.New("a pattern is not empty")
	case strings.Contains(text, "\\"):
		return PathPattern{}, errors.New("a pattern separates folders with /, not \\")
	case len(text) >= 2 && text[1] == ':':
		return PathPattern{}, errors.New("a pattern is relative to the project folder and has no drive")
	case strings.HasPrefix(text, "/"):
		return PathPattern{}, errors.New("a pattern is relative to the project folder, not absolute")
	}
	body, folder := strings.CutSuffix(text, "/")
	segments := strings.Split(body, "/")
	for _, s := range segments {
		switch s {
		case "..":
			return PathPattern{}, errors.New("a pattern stays below the project folder, without ..")
		case "", ".":
			return PathPattern{}, errors.New("a pattern has no empty folder name and no .")
		}
		if !validPattern(s) {
			return PathPattern{}, errors.New("a pattern uses the wildcards *, ? and [a-z]")
		}
	}
	return PathPattern{Text: text, Segments: segments, Folder: folder}, nil
}

// dotName is the reason a Realized in pattern is refused that has a folder or
// file name starting with a dot: the project's own files never include such a
// name, so the pattern can never name anything.
const dotName = "a name that starts with a dot is always left out, so a pattern with one names nothing"

// hasDotName reports whether a name of the pattern starts with a dot.
func (p PathPattern) hasDotName() bool {
	for _, s := range p.Segments {
		if strings.HasPrefix(s, ".") {
			return true
		}
	}
	return false
}

// Names reports whether the pattern names the file at rel, a path below the
// project folder with /: its whole path matches, or, for a pattern that ends
// with /, a folder it names holds the file at any depth.
func (p PathPattern) Names(rel string) bool {
	names := strings.Split(rel, "/")
	if p.Folder {
		// The file is below the folder, so its own name is no folder name.
		return len(names) > len(p.Segments) && matchSegments(p.Segments, names[:len(p.Segments)])
	}
	return len(names) == len(p.Segments) && matchSegments(p.Segments, names)
}

// Is reports whether the whole path rel, the path of a file or of a folder,
// matches the pattern segment by segment, whether or not it ends with /.
func (p PathPattern) Is(rel string) bool {
	names := strings.Split(rel, "/")
	return len(names) == len(p.Segments) && matchSegments(p.Segments, names)
}

func matchSegments(patterns, names []string) bool {
	for i, pattern := range patterns {
		if ok, _ := path.Match(pattern, names[i]); !ok {
			return false
		}
	}
	return true
}

// realizedPatterns returns the well-formed patterns of the sections Realized in
// of a README, in the order of their lines.
func realizedPatterns(readme *Document) []PathPattern {
	var out []PathPattern
	for _, c := range readme.CodeSpans {
		if !strings.EqualFold(c.Section, realizedSection) {
			continue
		}
		if p, err := ParsePathPattern(c.Text); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// checkRealized reports the code spans of the sections Realized in of a README
// that are no path pattern.
func checkRealized(d *Document) []Diagnostic {
	if !isReadme(d.Path) {
		return nil
	}
	var out []Diagnostic
	for _, c := range d.CodeSpans {
		if !strings.EqualFold(c.Section, realizedSection) {
			continue
		}
		p, err := ParsePathPattern(c.Text)
		if err == nil && p.hasDotName() {
			err = errors.New(dotName)
		}
		if err != nil {
			out = append(out, Diagnostic{Path: d.Path, Line: c.Line, Code: CodeReadmePattern,
				Message: fmt.Sprintf("bad pattern \"%s\": %v", c.Text, err)})
		}
	}
	return out
}

// SpecModule is the module that is the spec folder itself.
const SpecModule = specFolder

// Modules returns the folders of the spec that are modules, the spec folder
// first, which holds a document directly, and the others by path.
func (s *Spec) Modules() []string {
	out := make([]string, 0, len(s.modules))
	for m := range s.modules {
		out = append(out, m)
	}
	SortModules(out)
	return out
}

// RealizedIn returns the path patterns of the sections Realized in of the
// README of a module, those that are well-formed; a module without a README
// has none.
func (s *Spec) RealizedIn(module string) []PathPattern {
	readme, ok := s.byPath[readmeOf(module)]
	if !ok {
		return nil
	}
	return realizedPatterns(readme)
}
