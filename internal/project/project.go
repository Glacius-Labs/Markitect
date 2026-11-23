// Package project is a Markitect project, over a file system rooted at the
// project folder: the file .markitect/spec that makes a folder a project and
// names its spec folder, the project loaded with its spec, the ignore file that
// leaves out what does not belong to its files, and the difference between two
// projects: their specs, and for each module its files. It reads only the file
// systems it is handed, so that two projects, such as the two that diff
// compares, load the same way.
//
// The rules are in docs/guides/project-reference.md.
package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// The file that makes a folder a project, by its path below the project
// folder. Its one line names the spec folder; whatever else lies in .markitect/
// is no concern of the tool.
const (
	Dir      = ".markitect"
	SpecFile = ".markitect/spec"
)

// SpecPath turns a spec folder, as written in .markitect/spec, into a clean
// path below the project folder with forward slashes. It refuses a path that
// is empty, absolute, has a drive, leaves the project folder, is the project
// folder itself, or lies in .markitect/, in any case.
func SpecPath(arg string) (string, error) {
	p := strings.TrimSpace(strings.ReplaceAll(arg, "\\", "/"))
	switch {
	case p == "":
		return "", errors.New("the spec folder is empty")
	case strings.HasPrefix(p, "/") || len(p) >= 2 && p[1] == ':':
		return "", fmt.Errorf("the spec folder %s is not a path below the project folder", arg)
	}
	p = path.Clean(p)
	if p == "." || p == ".." || strings.HasPrefix(p, "../") {
		return "", fmt.Errorf("the spec folder %s is not a path below the project folder", arg)
	}
	// The reserved name counts in any case, as a file system that ignores
	// case reads it.
	if first, _, _ := strings.Cut(p, "/"); strings.EqualFold(first, Dir) {
		return "", fmt.Errorf("the spec folder %s lies in %s/", arg, Dir)
	}
	return p, nil
}

// ReadSpecFolder reads .markitect/spec, which holds one line: the spec folder.
// A byte order mark, white space and the line end around it are dropped, so a
// checkout with CRLF reads the same.
func ReadSpecFolder(fsys fs.FS) (string, error) {
	line, err := readLine(fsys, SpecFile)
	if err != nil {
		return "", err
	}
	spec, err := SpecPath(line)
	if err != nil {
		return "", fmt.Errorf("%s: %w", SpecFile, err)
	}
	return spec, nil
}

// readLine reads a file of one line.
func readLine(fsys fs.FS, name string) (string, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	line := strings.TrimSpace(string(data))
	switch {
	case line == "":
		return "", fmt.Errorf("%s: empty", name)
	case strings.ContainsAny(line, "\r\n"):
		return "", fmt.Errorf("%s: more than one line", name)
	}
	return line, nil
}

// Project is a project loaded from a file system rooted at its folder.
type Project struct {
	// FS is the project folder.
	FS fs.FS
	// SpecFolder is the spec folder, a path below the project folder.
	SpecFolder string
	// SpecFS is the spec folder.
	SpecFS fs.FS
	Spec   *lang.Spec
}

// Load reads .markitect/spec of the project at fsys and its spec, whose
// folder must exist.
func Load(fsys fs.FS) (*Project, error) {
	folder, err := ReadSpecFolder(fsys)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(fsys, folder)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("the spec folder %s, which %s names, is no folder", folder, SpecFile)
	}
	sub, err := fs.Sub(fsys, folder)
	if err != nil {
		return nil, err
	}
	spec, err := lang.Load(sub)
	if err != nil {
		return nil, err
	}
	return &Project{FS: fsys, SpecFolder: folder, SpecFS: sub, Spec: spec}, nil
}
