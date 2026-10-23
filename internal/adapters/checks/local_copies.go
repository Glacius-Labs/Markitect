package checks

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// LocalCopies owns disposable directories on the engine host. The engine must
// separately establish that its bind transport names these same directories.
type LocalCopies struct {
	root   string
	mu     sync.Mutex
	copies map[CopyID]CopyObservation
}

func NewLocalCopies(parent string) (*LocalCopies, error) {
	if !filepath.IsAbs(parent) || filepath.Clean(parent) != parent {
		return nil, fmt.Errorf("check copies require an explicit absolute parent")
	}
	root, err := os.MkdirTemp(parent, "markitect-checks-")
	if err != nil {
		return nil, err
	}
	return &LocalCopies{root: root, copies: map[CopyID]CopyObservation{}}, nil
}

func (c *LocalCopies) Materialize(ctx context.Context, req CopySpec) (CopyObservation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	o := CopyObservation{ID: req.ID}
	if err := ctx.Err(); err != nil {
		return o, err
	}
	if req.ID == "" || filepath.Base(string(req.ID)) != string(req.ID) || strings.ContainsAny(string(req.ID), "/\\:") || req.ID == "." || req.ID == ".." {
		return o, fmt.Errorf("invalid disposable check identity")
	}
	if _, exists := c.copies[req.ID]; exists {
		return o, fmt.Errorf("disposable check identity already exists")
	}
	o.Directory = filepath.Join(c.root, string(req.ID))
	c.copies[req.ID] = o
	if err := os.MkdirAll(c.root, 0700); err != nil {
		return o, err
	}
	if err := os.Mkdir(o.Directory, 0700); err != nil {
		return o, err
	}
	for _, e := range req.Tree.Entries() {
		if err := ctx.Err(); err != nil {
			return o, err
		}
		// Windows host permissions do not represent the POSIX executable bit.
		if runtime.GOOS == "windows" && e.Mode() == snapshot.Executable {
			return o, fmt.Errorf("exact executable mode requires a POSIX check materializer")
		}
		file := filepath.Join(o.Directory, filepath.FromSlash(e.Path().String()))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return o, err
		}
		mode := os.FileMode(0600)
		if e.Mode() == snapshot.Executable {
			mode = 0700
		}
		if err := os.WriteFile(file, e.Bytes(), mode); err != nil {
			return o, err
		}
	}
	var entries []snapshot.Entry
	err := filepath.WalkDir(o.Directory, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular check input")
		}
		rel, err := filepath.Rel(o.Directory, p)
		if err != nil {
			return err
		}
		path, err := snapshot.NewPath(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		mode := snapshot.Regular
		if info.Mode()&0111 != 0 {
			mode = snapshot.Executable
		}
		e, err := snapshot.NewEntry(path, mode, data)
		if err == nil {
			entries = append(entries, e)
		}
		return err
	})
	if err != nil {
		return o, err
	}
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		return o, err
	}
	if !tree.Equal(req.Tree) {
		return o, fmt.Errorf("disposable check readback differs from fixed tree")
	}
	o.Tree = tree.Digest()
	c.copies[req.ID] = o
	return o, nil
}

func (c *LocalCopies) Close(_ context.Context, id CopyID) (CopyObservation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	o, exists := c.copies[id]
	if !exists {
		return CopyObservation{ID: id}, fmt.Errorf("unknown disposable check identity")
	}
	if o.Directory != filepath.Join(c.root, string(id)) {
		return o, fmt.Errorf("check directory ownership mismatch")
	}
	if err := os.RemoveAll(o.Directory); err != nil {
		return o, err
	}
	if _, err := os.Lstat(o.Directory); !os.IsNotExist(err) {
		return o, fmt.Errorf("check directory absence unconfirmed")
	}
	o.Closed = true
	c.copies[id] = o
	allClosed := true
	for _, copy := range c.copies {
		allClosed = allClosed && copy.Closed
	}
	if allClosed {
		if err := os.Remove(c.root); err != nil && !os.IsNotExist(err) {
			return o, err
		}
	}
	return o, nil
}
