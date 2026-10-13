package recordstore

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
)

// File is a raw preimage, distinct from all domain digest/CAS meanings.
type File struct {
	Exists   bool
	Bytes    []byte
	Mode     fs.FileMode
	identity os.FileInfo
}

func readLeaf(d *directory, leaf string, limit int64) (File, error) {
	if err := d.check(); err != nil {
		return File{}, err
	}
	if err := storedName(d.root, leaf); err != nil {
		return File{}, err
	}
	i, err := d.root.Lstat(leaf)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	if !i.Mode().IsRegular() || platformIsReparse(i) {
		return File{}, errors.New("record leaf is not regular")
	}
	if err := platformMode(i, 0600); err != nil {
		return File{}, err
	}
	if limit > 0 && i.Size() > limit {
		return File{}, errors.New("record exceeds caller byte bound")
	}
	f, err := d.root.Open(leaf)
	if err != nil {
		return File{}, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(i, opened) {
		return File{}, errors.New("record leaf changed during open")
	}
	var reader io.Reader = f
	if limit > 0 {
		reader = io.LimitReader(f, limit+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return File{}, err
	}
	if limit > 0 && int64(len(data)) > limit {
		return File{}, errors.New("record exceeds caller byte bound")
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(opened, after) || opened.Size() != after.Size() || int64(len(data)) != after.Size() {
		return File{}, errors.New("record changed during read")
	}
	named, err := d.root.Lstat(leaf)
	if err != nil || !os.SameFile(after, named) || platformIsReparse(named) {
		return File{}, errors.New("record name changed during read")
	}
	if err := d.check(); err != nil {
		return File{}, err
	}
	return File{Exists: true, Bytes: data, Mode: after.Mode().Perm(), identity: after}, nil
}
func (s *Store) Read(n Name, limit int64) (File, error) {
	p, err := n.Path()
	if err != nil {
		return File{}, err
	}
	if err = s.CheckCutover(n.Kind); err != nil {
		return File{}, err
	}
	d, err := s.openDirectory(path.Dir(p), false, true)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	defer d.close()
	return readLeaf(d, path.Base(p), limit)
}
func (s *Store) List(kind Kind, id, role string) ([]fs.DirEntry, error) {
	if err := s.CheckCutover(kind); err != nil {
		return nil, err
	}
	p, err := directoryPath(kind, id, role)
	if err != nil {
		return nil, err
	}
	d, err := s.openDirectory(p, false, true)
	if errors.Is(err, fs.ErrNotExist) {
		return []fs.DirEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer d.close()
	entries, err := d.file.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		info, err := d.root.Lstat(e.Name())
		if err != nil {
			return nil, err
		}
		if platformIsReparse(info) {
			return nil, errors.New("record list contains reparse entry")
		}
	}
	if err := d.check(); err != nil {
		return nil, err
	}
	return entries, nil
}
func equalPreimage(a, b File) bool {
	return a.Exists == b.Exists && (!a.Exists || a.Mode == b.Mode && bytes.Equal(a.Bytes, b.Bytes) && (a.identity == nil || os.SameFile(a.identity, b.identity)))
}
