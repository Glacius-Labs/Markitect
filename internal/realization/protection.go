package realization

import (
	"errors"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var ErrInvalidProtection = errors.New("invalid protection")

// Protection is a fixed exact-file or directory-subtree restriction: files that
// belong to the project's canonical or Markitect control inputs and that no
// Manager may change. No globbing, physical existence or rights are inferred
// from these lexical facts.
type Protection struct {
	file      snapshot.Path
	directory snapshot.DirectoryBoundary
	why       string
}

func ProtectFile(path snapshot.Path, why string) (Protection, error) {
	if path.String() == "" || strings.TrimSpace(why) == "" {
		return Protection{}, ErrInvalidProtection
	}
	return Protection{file: path, why: why}, nil
}

func ProtectDirectory(directory snapshot.DirectoryBoundary, why string) (Protection, error) {
	if directory.String() == "" || strings.TrimSpace(why) == "" {
		return Protection{}, ErrInvalidProtection
	}
	return Protection{directory: directory, why: why}, nil
}

func (p Protection) Why() string                 { return p.why }
func (p Protection) File() (snapshot.Path, bool) { return p.file, p.file.String() != "" }
func (p Protection) Directory() (snapshot.DirectoryBoundary, bool) {
	return p.directory, p.directory.String() != ""
}

// Covers reports whether the exact file or the directory subtree contains path.
func (p Protection) Covers(path snapshot.Path) bool {
	if file, ok := p.File(); ok {
		return file == path
	}
	directory, ok := p.Directory()
	return ok && directory.Contains(path)
}
