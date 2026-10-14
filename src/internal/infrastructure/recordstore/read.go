package recordstore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// File is a raw preimage, distinct from all domain digest/CAS meanings.
type File struct {
	Exists   bool
	Bytes    []byte
	Mode     fs.FileMode
	identity os.FileInfo
}

// ErrReadConflict is a safe regular-leaf identity transition, never a root,
// parent, type, mode, byte-bound, data or generic IO failure. Reads do not retry.
var ErrReadConflict = errors.New("record leaf identity changed during observation")

func validateReadLeaf(i os.FileInfo, limit int64) error {
	if i == nil || !i.Mode().IsRegular() || platformIsReparse(i) {
		return errors.New("record leaf is not regular")
	}
	if err := platformMode(i, 0600); err != nil {
		return err
	}
	if limit > 0 && i.Size() > limit {
		return errors.New("record exceeds caller byte bound")
	}
	return nil
}

func readConflict(d *directory, leaf string, limit int64, prior, observed os.FileInfo) error {
	if err := validateReadLeaf(observed, limit); err != nil {
		return err
	}
	if err := d.check(); err != nil {
		return err
	}
	if err := storedName(d.root, leaf); err != nil {
		return err
	}
	named, err := d.root.Lstat(leaf)
	if err != nil {
		return fmt.Errorf("observe conflict leaf %q: %w", leaf, err)
	}
	if err := validateReadLeaf(named, limit); err != nil {
		return err
	}
	if err := d.check(); err != nil {
		return err
	}
	if os.SameFile(prior, observed) && os.SameFile(prior, named) {
		return errors.New("record changed without a safe leaf identity transition")
	}
	return ErrReadConflict
}

func readLeaf(d *directory, leaf string, limit int64) (result File, resultErr error) {
	defer func() {
		if resultErr != nil {
			parts := make([]string, 0, len(d.levels)+1)
			for _, level := range d.levels {
				parts = append(parts, level.leaf)
			}
			parts = append(parts, leaf)
			resultErr = fmt.Errorf("read record leaf %q: %w", strings.Join(parts, "/"), resultErr)
		}
	}()
	if err := d.check(); err != nil {
		return File{}, err
	}
	if err := storedName(d.root, leaf); err != nil {
		return File{}, err
	}
	i, err := d.root.Lstat(leaf)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, d.confirmMissingName(leaf)
	}
	if err != nil {
		return File{}, fmt.Errorf("observe leaf: %w", err)
	}
	if err := validateReadLeaf(i, limit); err != nil {
		return File{}, err
	}
	if d.store.hooks.beforeLeafOpen != nil {
		if err := d.store.hooks.beforeLeafOpen(); err != nil {
			return File{}, err
		}
	}
	f, err := d.root.Open(leaf)
	if err != nil {
		return File{}, fmt.Errorf("open leaf: %w", err)
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return File{}, fmt.Errorf("observe opened leaf: %w", err)
	}
	if err := validateReadLeaf(opened, limit); err != nil {
		return File{}, err
	}
	if !os.SameFile(i, opened) {
		return File{}, readConflict(d, leaf, limit, i, opened)
	}
	var reader io.Reader = f
	if limit > 0 {
		reader = io.LimitReader(f, limit+1)
	}
	var data []byte
	// Empty persistent kernel leaves are validated by facts, without a DATA
	// read that conflicts with the unchanged Windows byte-range lease.
	if opened.Size() != 0 {
		data, err = io.ReadAll(reader)
		if err != nil {
			return File{}, fmt.Errorf("read leaf bytes: %w", err)
		}
	}
	if limit > 0 && int64(len(data)) > limit {
		return File{}, errors.New("record exceeds caller byte bound")
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(opened, after) || opened.Size() != after.Size() || int64(len(data)) != after.Size() {
		return File{}, errors.New("record changed during read")
	}
	if err := validateReadLeaf(after, limit); err != nil {
		return File{}, err
	}
	named, err := d.root.Lstat(leaf)
	if err != nil {
		return File{}, fmt.Errorf("observe leaf after read: %w", err)
	}
	if err := validateReadLeaf(named, limit); err != nil {
		return File{}, err
	}
	if !os.SameFile(after, named) {
		return File{}, readConflict(d, leaf, limit, after, named)
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
	if isDirectoryAbsent(err) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	defer d.close()
	return readLeaf(d, path.Base(p), limit)
}

// ReadLegacyBrownfieldManagerLedger only observes this historical basename.
// It adds no writable role, codec, migration or Manager authority.
func (s *Store) ReadLegacyBrownfieldManagerLedger(id string, limit int64) (File, error) {
	if !validID(Brownfield, id) {
		return File{}, errors.New("invalid Brownfield session ID")
	}
	if err := s.CheckCutover(Brownfield); err != nil {
		return File{}, err
	}
	p, _ := directoryPath(Brownfield, id, "")
	d, err := s.openDirectory(p, false, true)
	if isDirectoryAbsent(err) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	defer d.close()
	markers := func() error {
		if err := d.check(); err != nil {
			return err
		}
		for _, suffix := range []string{".tmp", ".previous"} {
			leaf := ".manager-runs.json" + suffix
			if err := storedName(d.root, leaf); err != nil {
				return err
			}
			if _, err := d.root.Lstat(leaf); err == nil {
				return fmt.Errorf("ledger publication has an unreconciled %s file", suffix)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("observe legacy marker %q/%q: %w", p, leaf, err)
			}
		}
		return d.check()
	}
	if err := markers(); err != nil {
		return File{}, err
	}
	file, err := readLeaf(d, "manager-runs.json", limit)
	if err != nil {
		return File{}, err
	}
	if err := markers(); err != nil {
		return File{}, err
	}
	return file, nil
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
	if isDirectoryAbsent(err) {
		return []fs.DirEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer d.close()
	entries, err := d.file.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("list record directory %q: %w", p, err)
	}
	for _, e := range entries {
		info, err := d.root.Lstat(e.Name())
		if err != nil {
			return nil, fmt.Errorf("observe listed record %q/%q: %w", p, e.Name(), err)
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
	return a.Exists == b.Exists && (!a.Exists || a.Mode == b.Mode && bytes.Equal(a.Bytes, b.Bytes) && a.identity != nil && b.identity != nil && os.SameFile(a.identity, b.identity))
}
