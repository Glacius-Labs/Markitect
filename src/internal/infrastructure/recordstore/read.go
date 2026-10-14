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

// ReadWithDirectoryInspection performs the direct-child inspections of a
// closed record's parent and grandparent, then reads the leaf under the same
// held directory chain. It has no domain codec, cache, or write behavior.
func (s *Store) ReadWithDirectoryInspection(n Name, limit int64) (File, error) {
	p, err := n.Path()
	if err != nil {
		return File{}, err
	}
	parent := path.Dir(p)
	grandparent := path.Dir(parent)
	if grandparent == "." || grandparent == parent || !strings.HasPrefix(grandparent, RootPath+"/") {
		return File{}, errors.New("record path lacks an inspectable parent chain")
	}
	grandparentName, err := directoryName(grandparent)
	if err != nil {
		return File{}, err
	}
	parentName, err := directoryName(parent)
	if err != nil {
		return File{}, err
	}
	grandparentRelative := grandparent
	if err := s.CheckCutover(n.Kind); err != nil {
		return File{}, err
	}
	d, err := s.openDirectory(grandparentRelative, false, true)
	if isDirectoryAbsent(err) {
		return s.readWithFreshDirectoryInspection(n, grandparentName, parentName, limit)
	}
	if err != nil {
		return File{}, err
	}
	defer func() {
		if d != nil {
			d.close()
		}
	}()
	if len(d.levels) < 2 {
		return File{}, errors.New("record parent chain is too short")
	}
	if err := inspectHeldDirectory(d, len(d.levels)-1, grandparent); err != nil {
		return File{}, err
	}

	if err := s.CheckCutover(n.Kind); err != nil {
		return File{}, err
	}
	if err := checkHeldReadDirectory(d, len(d.levels), false); err != nil {
		return File{}, err
	}
	leafDirectory := path.Base(parent)
	if err := extendHeldReadDirectory(d, leafDirectory); isDirectoryAbsent(err) {
		d.close()
		d = nil
		return s.readWithFreshDirectoryInspection(n, grandparentName, parentName, limit)
	} else if err != nil {
		return File{}, err
	}
	if err := inspectHeldDirectory(d, len(d.levels)-1, parent); err != nil {
		return File{}, err
	}

	if err := s.CheckCutover(n.Kind); err != nil {
		return File{}, err
	}
	if err := checkHeldReadDirectory(d, len(d.levels), true); err != nil {
		return File{}, err
	}
	return readLeaf(d, path.Base(p), limit)
}

type directoryParts struct {
	kind Kind
	id   string
	role string
}

func directoryName(p string) (directoryParts, error) {
	if !strings.HasPrefix(p, RootPath+"/") {
		return directoryParts{}, errors.New("directory path is outside the record store")
	}
	parts := strings.Split(strings.TrimPrefix(p, RootPath+"/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return directoryParts{}, errors.New("record directory path is empty")
	}
	kind, id, role := Kind(parts[0]), "", ""
	if len(parts) > 1 {
		id = parts[1]
	}
	if len(parts) > 2 {
		role = strings.Join(parts[2:], "/")
	}
	if _, err := directoryPath(kind, id, role); err != nil {
		return directoryParts{}, err
	}
	return directoryParts{kind: kind, id: id, role: role}, nil
}

func (s *Store) readWithFreshDirectoryInspection(n Name, grandparent, parent directoryParts, limit int64) (File, error) {
	if _, err := s.List(grandparent.kind, grandparent.id, grandparent.role); err != nil {
		return File{}, err
	}
	if _, err := s.List(parent.kind, parent.id, parent.role); err != nil {
		return File{}, err
	}
	return s.Read(n, limit)
}

func checkHeldReadDirectory(d *directory, hookLevels int, fullCheck bool) error {
	if d == nil || d.store == nil || len(d.levels) == 0 {
		return errors.New("held record directory is unavailable")
	}
	for index, level := range d.levels {
		if err := checkHeldReadPrefix(d, index); err != nil {
			return err
		}
		if err := storedName(level.parent, level.leaf); err != nil {
			return err
		}
		observed, err := level.parent.Lstat(level.leaf)
		if err != nil {
			return fmt.Errorf("observe held record parent %q: %w", level.leaf, err)
		}
		if !observed.IsDir() || platformIsReparse(observed) || !os.SameFile(level.identity, observed) {
			return errors.New("held record parent name identity changed")
		}
		if level.private {
			if err := platformMode(observed, 0700); err != nil {
				return err
			}
		}
		if index < hookLevels && d.store.hooks.beforeOpen != nil {
			if err := d.store.hooks.beforeOpen(); err != nil {
				return err
			}
		}
		held, err := level.root.Stat(".")
		if err != nil {
			return fmt.Errorf("stat held record directory %q: %w", level.leaf, err)
		}
		if !os.SameFile(level.identity, held) {
			return errors.New("held record directory identity changed")
		}
		if level.private {
			if err := platformMode(held, 0700); err != nil {
				return err
			}
		}
	}
	if fullCheck {
		return d.check()
	}
	return nil
}

// checkHeldReadPrefix replays directory.check for only the path components
// opened before the next component in the original openDirectory walk.
func checkHeldReadPrefix(d *directory, count int) error {
	if count < 0 || count > len(d.levels) {
		return errors.New("held directory prefix is invalid")
	}
	if err := d.store.CheckIdentity(); err != nil {
		return err
	}
	for _, level := range d.levels[:count] {
		info, err := level.parent.Lstat(level.leaf)
		if err != nil {
			return fmt.Errorf("observe record parent %q: %w", level.leaf, err)
		}
		if platformIsReparse(info) || !info.IsDir() || !os.SameFile(level.identity, info) {
			return errors.New("record parent name identity changed")
		}
		if level.private {
			if err := platformMode(info, 0700); err != nil {
				return err
			}
		}
	}
	return nil
}

func inspectHeldDirectory(d *directory, index int, relative string) error {
	if d == nil || d.file == nil || index < 0 || index >= len(d.levels) {
		return errors.New("held directory inspection is unavailable")
	}
	root := d.levels[index].root
	entries, err := d.file.ReadDir(-1)
	if err != nil {
		return fmt.Errorf("list record directory %q: %w", relative, err)
	}
	for _, entry := range entries {
		info, err := root.Lstat(entry.Name())
		if err != nil {
			return fmt.Errorf("observe listed record %q/%q: %w", relative, entry.Name(), err)
		}
		if platformIsReparse(info) {
			return errors.New("record list contains reparse entry")
		}
	}
	return d.check()
}
func extendHeldReadDirectory(d *directory, leaf string) (resultErr error) {
	if err := d.check(); err != nil {
		return err
	}
	if err := storedName(d.root, leaf); err != nil {
		return err
	}
	info, err := d.root.Lstat(leaf)
	if errors.Is(err, fs.ErrNotExist) {
		if checkErr := d.confirmMissingName(leaf); checkErr != nil {
			return checkErr
		}
		return &directoryAbsent{cause: err}
	}
	if err != nil {
		return fmt.Errorf("observe record directory child %q: %w", leaf, err)
	}
	if !info.IsDir() || platformIsReparse(info) {
		return errors.New("unsafe record parent directory")
	}
	private := len(d.levels) >= 1
	if private {
		if err := platformMode(info, 0700); err != nil {
			return err
		}
	}
	if d.store.hooks.beforeOpen != nil {
		if err := d.store.hooks.beforeOpen(); err != nil {
			return err
		}
	}

	child, err := d.root.OpenRoot(leaf)
	if err != nil {
		return fmt.Errorf("open directory child %q: %w", leaf, err)
	}
	actual, err := child.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		child.Close()
		return errors.New("record directory changed while opening")
	}
	if private {
		if err := platformMode(actual, 0700); err != nil {
			child.Close()
			return err
		}
	}
	if d.file != nil {
		_ = d.file.Close()
		d.file = nil
	}
	parentRoot := d.root
	d.levels = append(d.levels, level{parent: parentRoot, leaf: leaf, root: child, identity: actual, private: private})
	d.root = child
	d.file, err = d.root.Open(".")
	if err != nil {
		return fmt.Errorf("open held record directory: %w", err)
	}
	return d.check()
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
