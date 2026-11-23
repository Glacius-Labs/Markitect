package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// IgnoreFile is the file of a project that leaves out what does not belong to
// its own files, by its path below the project folder.
const IgnoreFile = ".markitect/ignore"

// Ignore is the ignore file of a project: the files and folders that diff does
// not look at. Besides what it names, every file and folder whose name starts
// with a dot, the spec folder, and links and junctions are always left out. The
// zero value, a pointer to it and nil leave out nothing more.
type Ignore struct {
	rules []ignoreRule
}

type ignoreRule struct {
	pattern lang.PathPattern
	// anywhere is set for a pattern of one name, which matches a file or folder
	// of that name at any depth.
	anywhere bool
}

// ParseIgnore reads an ignore file: one pattern per line, blank lines and lines
// that start with # skipped, white space at the end of a line dropped, a byte
// order mark at the start dropped. A pattern is a path pattern; one without a /,
// other than at its end, matches a file or folder of that name at any depth,
// and one that ends with / matches only folders. A line that is no valid
// pattern is an error that names the file and the line.
func ParseIgnore(data []byte) (*Ignore, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	ig := &Ignore{}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := lang.ParsePathPattern(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: bad pattern \"%s\": %v", IgnoreFile, i+1, line, err)
		}
		ig.rules = append(ig.rules, ignoreRule{pattern: p, anywhere: len(p.Segments) == 1})
	}
	return ig, nil
}

// ReadIgnore reads the ignore file of the project at fsys; a project without
// one leaves out nothing more.
func ReadIgnore(fsys fs.FS) (*Ignore, error) {
	data, err := fs.ReadFile(fsys, IgnoreFile)
	if errors.Is(err, fs.ErrNotExist) {
		return &Ignore{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", IgnoreFile, unwrapPath(err))
	}
	return ParseIgnore(data)
}

// leaves reports whether the file or folder at rel, a path below the project
// folder, is left out by the ignore file.
func (ig *Ignore) leaves(rel string, folder bool) bool {
	if ig == nil {
		return false
	}
	for _, r := range ig.rules {
		if r.pattern.Folder && !folder {
			continue
		}
		if r.anywhere {
			if ok, _ := path.Match(r.pattern.Segments[0], path.Base(rel)); ok {
				return true
			}
		} else if r.pattern.Is(rel) {
			return true
		}
	}
	return false
}

// fileTable maps the files of a project that are looked at to their sizes, by
// their path below the project folder.
type fileTable map[string]int64

// folders are folders below a project folder that are left out of its files.
// A path names one when it is equal to it, or, on a file system that does not
// regard case, when it differs only in case and lies at the same place: the
// folder that is meant is the one on the disk, whatever case the name was
// written in.
type folders struct {
	names []string
	infos []fs.FileInfo
}

func newFolders(fsys fs.FS, names ...string) *folders {
	f := &folders{}
	for _, name := range names {
		if name == "" || name == "." {
			continue
		}
		info, _ := fs.Stat(fsys, name)
		f.names = append(f.names, name)
		f.infos = append(f.infos, info)
	}
	return f
}

// holds reports whether the folder at p, an entry of the walk, is one of them.
func (f *folders) holds(p string, e fs.DirEntry) bool {
	for i, name := range f.names {
		if p == name {
			return true
		}
		if f.infos[i] == nil || !strings.EqualFold(p, name) {
			continue
		}
		if info, err := e.Info(); err == nil && os.SameFile(f.infos[i], info) {
			return true
		}
	}
	return false
}

// listFiles walks the project at fsys. It leaves out what starts with a dot,
// the folders named, which are the spec folder and a project that lies inside
// this one, links and junctions, which it never follows, special files and what
// the ignore file names; a folder that is left out is left out with everything
// below it.
func listFiles(fsys fs.FS, ig *Ignore, leave ...string) (fileTable, error) {
	files := fileTable{}
	out := newFolders(fsys, leave...)
	err := fs.WalkDir(fsys, ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", p, unwrapPath(err))
		}
		if p == "." {
			return nil
		}
		skip := func() error {
			if e.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		switch {
		case strings.HasPrefix(e.Name(), "."):
			return skip()
		case e.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0:
			return nil
		case e.IsDir():
			if out.holds(p, e) || ig.leaves(p, true) {
				return fs.SkipDir
			}
			return nil
		case !e.Type().IsRegular() || ig.leaves(p, false):
			return nil
		}
		info, err := e.Info()
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", p, unwrapPath(err))
		}
		files[p] = info.Size()
		return nil
	})
	return files, err
}

// unwrapPath drops the operation and the path from a file system error, which
// the message names itself.
func unwrapPath(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// chunk is how much of each file is read at a time.
const chunk = 64 << 10

// comparer compares files byte for byte, with two buffers it reuses.
type comparer struct{ a, b []byte }

func newComparer() *comparer { return &comparer{make([]byte, chunk), make([]byte, chunk)} }

// same compares the file at rel in two file systems byte for byte, in chunks,
// so that neither is loaded as a whole. The sizes are those of the files, which
// differ for files that cannot be the same.
func (c *comparer) same(oldFS, newFS fs.FS, rel string, oldSize, newSize int64) (bool, error) {
	if oldSize != newSize {
		return false, nil
	}
	a, err := oldFS.Open(rel)
	if err != nil {
		return false, fmt.Errorf("cannot read %s in the old project: %w", rel, unwrapPath(err))
	}
	defer a.Close()
	b, err := newFS.Open(rel)
	if err != nil {
		return false, fmt.Errorf("cannot read %s in the new project: %w", rel, unwrapPath(err))
	}
	defer b.Close()
	for {
		na, errA := io.ReadFull(a, c.a)
		nb, errB := io.ReadFull(b, c.b)
		for _, r := range []struct {
			side string
			err  error
		}{{"old", errA}, {"new", errB}} {
			if r.err != nil && r.err != io.EOF && r.err != io.ErrUnexpectedEOF {
				return false, fmt.Errorf("cannot read %s in the %s project: %w", rel, r.side, unwrapPath(r.err))
			}
		}
		if na != nb || !bytes.Equal(c.a[:na], c.b[:nb]) {
			return false, nil
		}
		if na < chunk {
			return true, nil
		}
	}
}
