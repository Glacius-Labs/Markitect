package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Walk returns every file of the project at fsys, by its path below the
// project folder with forward slashes, sorted. It leaves out every entry named
// .git, .markitect/ at the top, and what the project's .gitignore files or
// .markitect/ignore exclude. A folder that is another repository, whose .git
// is a folder with a HEAD or a file naming one, is one entry and is not
// walked; a symbolic link is one entry and is not followed.
func Walk(fsys fs.FS) ([]string, error) {
	own, err := readIgnore(fsys, IgnoreFile)
	if err != nil {
		return nil, err
	}
	w := &walker{fsys: fsys, own: parseIgnore("", own), git: map[string][]pattern{}}
	var files []string
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		dir := d.IsDir()
		skip := func() error {
			if dir {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() == ".git" || p == Dir {
			return skip()
		}
		excluded, err := w.excluded(p, dir)
		if err != nil {
			return err
		}
		if excluded {
			return skip()
		}
		if dir && !nestedRepository(fsys, p) {
			return nil
		}
		files = append(files, p)
		return skip()
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// repositoryEntry is what another repository, one entry of the walk, reads as.
const repositoryEntry = "a repository of its own\n"

// Content reads a file the walk lists: a symbolic link as where it points,
// with forward slashes, and another repository as one fixed line, as Git
// keeps them.
func Content(fsys fs.FS, name string) ([]byte, error) {
	info, err := fs.Lstat(fsys, name)
	if err != nil {
		return nil, err
	}
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		to, err := fs.ReadLink(fsys, name)
		return []byte(filepath.ToSlash(to)), err
	case info.IsDir():
		if nestedRepository(fsys, name) {
			return []byte(repositoryEntry), nil
		}
		return nil, fmt.Errorf("%s: a folder", name)
	}
	return fs.ReadFile(fsys, name)
}

// Hash is the SHA-256 of a file's content as "sha256:" and 64 hexadecimal
// digits. A text file, valid UTF-8 without a NUL byte, hashes with each CRLF
// as LF, so that a checkout on Windows and one on Linux agree; any other file
// hashes as it is.
func Hash(data []byte) string {
	if utf8.Valid(data) && bytes.IndexByte(data, 0) < 0 {
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// HashFile is the Hash of a file the walk lists, read as Content reads it.
func HashFile(fsys fs.FS, name string) (string, error) {
	data, err := Content(fsys, name)
	if err != nil {
		return "", err
	}
	return Hash(data), nil
}

// nestedRepository says whether a folder is a repository or worktree of its
// own: its .git is a folder with a HEAD, or a file that names one.
func nestedRepository(fsys fs.FS, dir string) bool {
	git := path.Join(dir, ".git")
	info, err := fs.Stat(fsys, git)
	switch {
	case err != nil:
		return false
	case info.IsDir():
		_, err := fs.Stat(fsys, path.Join(git, "HEAD"))
		return err == nil
	}
	data, err := fs.ReadFile(fsys, git)
	return err == nil && strings.HasPrefix(string(data), "gitdir:")
}

// walker holds the ignore files one walk reads.
type walker struct {
	fsys fs.FS
	// own is .markitect/ignore; git holds the patterns of the .gitignore file
	// of each folder read so far.
	own []pattern
	git map[string][]pattern
}

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
		line = strings.TrimSuffix(line, "\r")
		// Trailing spaces go, unless a backslash keeps one.
		for strings.HasSuffix(line, " ") && !strings.HasSuffix(line, `\ `) {
			line = line[:len(line)-1]
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := pattern{base: base}
		if strings.HasPrefix(line, "!") {
			p.negate, line = true, line[1:]
		} else if strings.HasPrefix(line, `\!`) || strings.HasPrefix(line, `\#`) {
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			p.dirOnly, line = true, strings.TrimSuffix(line, "/")
		}
		p.anchored = strings.Contains(line, "/")
		p.glob = strings.TrimPrefix(line, "/")
		if p.glob != "" {
			out = append(out, p)
		}
	}
	return out
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
		return globMatch(p.glob, path.Base(below))
	}
	return matchParts(strings.Split(p.glob, "/"), strings.Split(below, "/"))
}

// matchParts matches a path against a pattern part by part. A part "**"
// stands for any number of parts, none included; a trailing "**" for one or
// more, so that it holds what is inside a folder and not the folder.
func matchParts(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			if len(pat) == 1 {
				return len(name) > 0
			}
			for i := 0; i <= len(name); i++ {
				if matchParts(pat[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 || !globMatch(pat[0], name[0]) {
			return false
		}
		pat, name = pat[1:], name[1:]
	}
	return len(name) == 0
}

// globMatch matches one part of a path as Git does, with case: "*" and "?"
// within the part, "[a-z]", "[!x]" for "not x", and a backslash escaping the
// character after it. path.Match is used, never filepath.Match, which takes a
// backslash for a separator on Windows.
func globMatch(glob, name string) bool {
	var b strings.Builder
	for i := 0; i < len(glob); i++ {
		switch {
		case glob[i] == '\\' && i+1 < len(glob):
			b.WriteString(glob[i : i+2])
			i++
		case glob[i] == '[' && i+1 < len(glob) && glob[i+1] == '!':
			b.WriteString("[^")
			i++
		default:
			b.WriteByte(glob[i])
		}
	}
	ok, _ := path.Match(b.String(), name)
	return ok
}
