// Package state is a project of the rebuilt tool, over a file system rooted
// at the project folder: the layout of .markitect/, the project loaded with
// its spec, the files of the project without what its ignore files exclude,
// the hash of a file, and the state: its rows, what status lists and the
// rows a mark leaves. It reads only the file system it is handed, so that two projects,
// such as two states to compare, load the same way.
//
// The rules are in docs/guides/project-reference.md.
package state

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// The layout of .markitect/, by paths below the project folder.
const (
	Dir            = ".markitect"
	VersionFile    = ".markitect/version"
	SpecFile       = ".markitect/spec"
	AttributesFile = ".markitect/.gitattributes"
	IgnoreFile     = ".markitect/ignore"
	// DefaultSpec is the spec folder init names when it is given none.
	DefaultSpec = "docs/spec"
)

// attributes has Git keep the lines of both sides of a merge of the state and
// the history.
const attributes = "state.jsonl merge=union\nhistory.jsonl merge=union\n"

// Layout is what .markitect/ says of a project: the tool version it pins and
// its spec folder, a path below the project folder.
type Layout struct {
	Version    string
	SpecFolder string
}

// File is one file to write, by its path below the project folder.
type File struct {
	Path string
	Data []byte
}

// InitFiles are the files init writes into .markitect/ for a project that
// pins version and keeps its spec in spec, a path SpecPath accepts.
func InitFiles(version, spec string) []File {
	return []File{
		{VersionFile, []byte(version + "\n")},
		{SpecFile, []byte(spec + "\n")},
		{AttributesFile, []byte(attributes)},
		{IgnoreFile, nil},
	}
}

// SpecPath turns a spec folder, as given on the command line or written in
// .markitect/spec, into a clean path below the project folder with forward
// slashes. It refuses a path that is empty, absolute, has a drive, leaves the
// project folder, is the project folder itself, or lies in .markitect/ or a
// .git folder, in any case.
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
	// The reserved names count in any case, as a file system that ignores
	// case reads them.
	for i, part := range strings.Split(p, "/") {
		switch {
		case i == 0 && strings.EqualFold(part, Dir):
			return "", fmt.Errorf("the spec folder %s lies in %s/", arg, Dir)
		case strings.EqualFold(part, ".git"):
			return "", fmt.Errorf("the spec folder %s lies in a .git folder", arg)
		}
	}
	return p, nil
}

// ReadLayout reads .markitect/version and .markitect/spec. Each holds one
// line; a byte order mark, white space and the line end around it are
// dropped, so a checkout with CRLF reads the same.
func ReadLayout(fsys fs.FS) (Layout, error) {
	version, err := readLine(fsys, VersionFile)
	if err != nil {
		return Layout{}, err
	}
	specLine, err := readLine(fsys, SpecFile)
	if err != nil {
		return Layout{}, err
	}
	spec, err := SpecPath(specLine)
	if err != nil {
		return Layout{}, fmt.Errorf("%s: %w", SpecFile, err)
	}
	return Layout{Version: version, SpecFolder: spec}, nil
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
	Layout
	// FS is the project folder, SpecFS the spec folder in it.
	FS, SpecFS fs.FS
	Spec       *lang.Spec
	// SpecAliases are further paths of the spec folder below the project
	// folder, where a symbolic link or a junction on the way to it leads.
	SpecAliases []string
}

// InSpec says whether a file of the walk belongs to the spec folder: it lies
// in it or in one of its aliases, or it is a link on the way to it.
func (p *Project) InSpec(file string) bool {
	for _, folder := range append([]string{p.SpecFolder}, p.SpecAliases...) {
		if strings.HasPrefix(file, folder+"/") || file == folder || strings.HasPrefix(p.SpecFolder, file+"/") {
			return true
		}
	}
	return false
}

// Load reads the layout of the project at fsys and its spec, whose folder
// must exist.
func Load(fsys fs.FS) (*Project, error) {
	layout, err := ReadLayout(fsys)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(fsys, layout.SpecFolder)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("the spec folder %s, which %s names, is no folder", layout.SpecFolder, SpecFile)
	}
	sub, err := fs.Sub(fsys, layout.SpecFolder)
	if err != nil {
		return nil, err
	}
	spec, err := lang.Load(sub)
	if err != nil {
		return nil, err
	}
	return &Project{Layout: layout, FS: fsys, SpecFS: sub, Spec: spec}, nil
}
