package state

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"strings"
)

// excluded says whether the .gitignore files or .markitect/ignore exclude a
// path whose folders the walk took. The two are judged apart, so a negation
// in one never brings back what the other excludes.
func (w *walker) excluded(p string, dir bool) (bool, error) {
	if lastMatch(w.own, p, dir) {
		return true, nil
	}
	// The files of the folders from the top down, so that deeper ones win.
	parts := strings.Split(p, "/")
	var all []pattern
	for i := 0; i < len(parts); i++ {
		folder := strings.Join(parts[:i], "/")
		ps, ok := w.git[folder]
		if !ok {
			data, err := readIgnore(w.fsys, path.Join(folder, ".gitignore"))
			if err != nil {
				return false, err
			}
			ps = parseIgnore(folder, data)
			w.git[folder] = ps
		}
		all = append(all, ps...)
	}
	return lastMatch(all, p, dir), nil
}

// lastMatch says whether the last pattern that matches p excludes it.
func lastMatch(patterns []pattern, p string, dir bool) bool {
	excluded := false
	for _, pt := range patterns {
		if pt.matches(p, dir) {
			excluded = !pt.negate
		}
	}
	return excluded
}

// readIgnore reads an ignore file. A missing one, a folder and a symbolic
// link, which Git does not follow, are none; any other error fails.
func readIgnore(fsys fs.FS, name string) ([]byte, error) {
	info, err := fs.Lstat(fsys, name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	case !info.Mode().IsRegular():
		return nil, nil
	}
	return fs.ReadFile(fsys, name)
}

// pattern is one line of an ignore file.
type pattern struct {
	// base is the folder of the ignore file, "" at the top.
	base    string
	glob    string
	negate  bool
	dirOnly bool
	// anchored patterns match the path below base; the others match the name
	// at any depth below it.
	anchored bool
}

// parseIgnore reads an ignore file in the folder base, as Git reads a
// .gitignore file.
func parseIgnore(base string, data []byte) []pattern {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var out []pattern
	for _, line := range strings.Split(string(data), "\n") {
		line = trimTrailingSpaces(strings.TrimSuffix(line, "\r"))
		if line == "" || line[0] == '#' {
			continue
		}
		p := pattern{base: base}
		if line[0] == '!' {
			p.negate, line = true, line[1:]
		}
		if strings.HasSuffix(line, "/") {
			p.dirOnly, line = true, line[:len(line)-1]
		}
		p.anchored = strings.Contains(line, "/")
		p.glob = strings.TrimPrefix(line, "/")
		if line != "" {
			out = append(out, p)
		}
	}
	return out
}

// trimTrailingSpaces drops the spaces at the end of a line unless a backslash
// escapes the first of them, as Git does; an escaped backslash escapes
// nothing more.
func trimTrailingSpaces(line string) string {
	lastSpace := -1
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ':
			if lastSpace < 0 {
				lastSpace = i
			}
		case '\\':
			i++
			if i == len(line) {
				return line
			}
			lastSpace = -1
		default:
			lastSpace = -1
		}
	}
	if lastSpace >= 0 {
		return line[:lastSpace]
	}
	return line
}

func (p pattern) matches(rel string, dir bool) bool {
	if p.dirOnly && !dir {
		return false
	}
	below := rel
	if p.base != "" {
		var ok bool
		if below, ok = strings.CutPrefix(rel, p.base+"/"); !ok {
			return false
		}
	}
	if !p.anchored {
		return wildmatch(p.glob, path.Base(below), false)
	}
	return wildmatch(p.glob, below, true)
}
