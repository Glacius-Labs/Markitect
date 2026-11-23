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
// character and a class such as [a-z], none of them across a /, and a leading
// dot is a character like any other; everything else, case included, matches
// itself. A folder name that is ** alone matches folders: **/ at the start in
// any folder, /**/ in the middle zero or more folders, /** at the end everything
// inside. A pattern that ends with / names a folder and everything below it.
// The ignore file of a project reads its patterns in the same language.

// PathPattern is a path pattern that is well-formed.
type PathPattern struct {
	// Text is the pattern as written.
	Text string
	// Segments are the names between the slashes, each a path.Match pattern or **.
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
	segments, err := PatternSegments(body)
	if err != nil {
		return PathPattern{}, err
	}
	return PathPattern{Text: text, Segments: segments, Folder: folder}, nil
}

// PatternSegments splits a pattern without its slashes at the start and the end
// into its names and checks each: none is empty, . or .., and each is ** or one
// that path.Match reads.
func PatternSegments(body string) ([]string, error) {
	segments := strings.Split(body, "/")
	for _, s := range segments {
		switch s {
		case "..":
			return nil, errors.New("a pattern stays below the project folder, without ..")
		case "", ".":
			return nil, errors.New("a pattern has no empty folder name and no .")
		}
		if !validPattern(s) {
			return nil, errors.New("a pattern uses the wildcards *, ? and [a-z]")
		}
	}
	return segments, nil
}

// MatchSegments reports whether the names, the folders and the file of a path,
// match the pattern segments: each name matches its segment as path.Match says,
// and a segment ** matches zero or more names, or, as the last segment, one or
// more.
func MatchSegments(pattern, names []string) bool {
	if len(pattern) == 0 {
		return len(names) == 0
	}
	if pattern[0] == "**" {
		if len(pattern) == 1 {
			return len(names) > 0
		}
		for i := 0; i <= len(names); i++ {
			if MatchSegments(pattern[1:], names[i:]) {
				return true
			}
		}
		return false
	}
	if len(names) == 0 {
		return false
	}
	if ok, _ := path.Match(pattern[0], names[0]); !ok {
		return false
	}
	return MatchSegments(pattern[1:], names[1:])
}

// Names reports whether the pattern names the file at rel, a path below the
// project folder with /: its whole path matches, or, for a pattern that ends
// with /, a folder it names holds the file at any depth.
func (p PathPattern) Names(rel string) bool {
	names := strings.Split(rel, "/")
	if p.Folder {
		// The file is below the folder, so its own name is no folder name.
		for k := 1; k < len(names); k++ {
			if MatchSegments(p.Segments, names[:k]) {
				return true
			}
		}
		return false
	}
	return MatchSegments(p.Segments, names)
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
		if _, err := ParsePathPattern(c.Text); err != nil {
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
