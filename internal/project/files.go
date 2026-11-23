package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// fileTable maps the files of a project that are looked at to their sizes, by
// their path below the project folder.
type fileTable map[string]int64

// Folder is a folder below a project folder that is left out of its files. It
// is named by its Path below the project folder, with /, or by Info, the folder
// as it is on the disk, which holds whatever names lead to it, upper or lower
// case and short names included. Either may be empty.
type Folder struct {
	Path string
	Info fs.FileInfo
}

// folders are the folders left out of the files of a project. A path names one
// when it is equal to its Path, or when it is the folder of its Info on the
// disk; the Path may also be written in another case, on a file system that does
// not regard case, which os.SameFile tells.
type folders struct {
	list []Folder
}

func newFolders(fsys fs.FS, leave []Folder) *folders {
	f := &folders{}
	for _, l := range leave {
		if l.Path == "." || l.Path == "" && l.Info == nil {
			continue
		}
		if l.Info == nil {
			l.Info, _ = fs.Stat(fsys, l.Path)
		}
		f.list = append(f.list, l)
	}
	return f
}

// holds reports whether the folder at p, an entry of the walk, is one of them.
func (f *folders) holds(p string, e fs.DirEntry) bool {
	for _, l := range f.list {
		if p == l.Path {
			return true
		}
		if l.Info == nil || l.Path != "" && !strings.EqualFold(p, l.Path) {
			continue
		}
		if info, err := e.Info(); err == nil && os.SameFile(l.Info, info) {
			return true
		}
	}
	return false
}

// listFiles walks the project at fsys and returns its files that are looked at.
// Left out are the folder .markitect/ at the top, the folders named, which are
// the spec folder and a project that lies inside this one, links and junctions,
// which are never followed, special files, and what the ignore file leaves out;
// a folder that is left out is not entered. A name that starts with a dot is a
// name like any other.
func listFiles(fsys fs.FS, ig *Ignore, leave ...Folder) (fileTable, error) {
	files := fileTable{}
	out := newFolders(fsys, leave)
	err := fs.WalkDir(fsys, ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", p, unwrapPath(err))
		}
		if p == "." {
			return nil
		}
		switch {
		case strings.EqualFold(p, Dir):
			if e.IsDir() {
				return fs.SkipDir
			}
			return nil
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
