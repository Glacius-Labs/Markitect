package snapshot

import "strings"

// DirectoryBoundary is a fixed lexical area, either the repository root or a
// named relative directory. Its zero value is invalid. It neither represents a
// Tree entry nor proves that a directory exists, resolves aliases or grants
// authority. Only adapters can establish physical filesystem facts.
type DirectoryBoundary struct {
	path Path
	root bool
}

// RootDirectory explicitly selects the repository-wide lexical area. This does
// not make '.' a valid Path or a canonical model Package.
func RootDirectory() DirectoryBoundary { return DirectoryBoundary{root: true} }

// DirectoryAt retains a validated Path as a named directory boundary without
// changing its spelling. A zero Path is invalid, never an implicit root.
func DirectoryAt(path Path) (DirectoryBoundary, error) {
	if path.value == "" {
		return DirectoryBoundary{}, ErrInvalidPath
	}
	return DirectoryBoundary{path: path}, nil
}

func (d DirectoryBoundary) IsRoot() bool { return d.root }

// String returns '.' for the explicit repository root, the exact named path,
// or empty for the invalid zero value.
func (d DirectoryBoundary) String() string {
	if d.root {
		return "."
	}
	return d.path.String()
}

// Contains includes the named boundary itself and exact slash descendants.
// Root contains every valid relative Path. Invalid values never match.
func (d DirectoryBoundary) Contains(path Path) bool {
	if path.value == "" {
		return false
	}
	return d.root || d.path.value != "" && (d.path == path || strings.HasPrefix(path.value, d.path.value+"/"))
}

// ContainsDescendant excludes the named boundary itself. A new file under a
// creation directory must not replace that directory. All valid Paths are
// strict descendants of the explicit repository root.
func (d DirectoryBoundary) ContainsDescendant(path Path) bool {
	return d.Contains(path) && (d.root || d.path != path)
}

// ContainsDirectory checks an inclusive directory-area relationship. Named
// directories may equal their ceiling; only the explicit root contains root.
// Invalid boundaries never match, even each other.
func (d DirectoryBoundary) ContainsDirectory(other DirectoryBoundary) bool {
	if other.root {
		return d.root
	}
	return d.Contains(other.path)
}
