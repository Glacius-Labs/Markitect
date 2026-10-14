// Package recordstore owns the confined physical layout and byte publication
// of four operational record kinds. Domain codecs, CAS meanings and lifecycle
// authority belong to its callers. Version cutover requires quiescent writers.
package recordstore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	root     *os.Root
	absolute string
	identity os.FileInfo
	hooks    hooks
}
type hooks struct {
	beforeOpen, beforePublish, afterPublish func() error
	beforeMkdir, beforeLeafOpen             func() error
	beforeTempCreate, afterPublicationClose func() error
	fileSync, directorySync                 func() error
	directoryCreated                        func() error
	afterDeleteMark                         func()
	afterLeaseCreateResult                  func(error)
}
type directory struct {
	root   *os.Root
	file   *os.File
	levels []level
	store  *Store
}
type level struct {
	parent   *os.Root
	leaf     string
	root     *os.Root
	identity os.FileInfo
	private  bool
}

// Only a negative child Lstat beneath an unchanged safe chain is absence.
// Guard failures and disappearing positively observed directories are errors.
type directoryAbsent struct{ cause error }

func (e *directoryAbsent) Error() string { return "record directory absent: " + e.cause.Error() }
func (e *directoryAbsent) Unwrap() error { return e.cause }
func isDirectoryAbsent(err error) bool {
	var absent *directoryAbsent
	return errors.As(err, &absent)
}

func (d *directory) confirmMissingName(leaf string) error {
	if err := d.check(); err != nil {
		return err
	}
	if err := storedName(d.root, leaf); err != nil {
		return err
	}
	return d.check()
}

func Open(root string) (*Store, error) {
	absolute, err := filepath.Abs(root)
	if err != nil || strings.TrimSpace(root) == "" {
		return nil, errors.New("record repository root required")
	}
	for p := absolute; ; p = filepath.Dir(p) {
		i, e := os.Lstat(p)
		if e != nil {
			return nil, e
		}
		if !i.IsDir() || platformIsReparse(i) {
			return nil, errors.New("record root ancestor is not a real directory")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	opened, err := r.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		r.Close()
		return nil, errors.New("record root changed during open")
	}
	return &Store{root: r, absolute: absolute, identity: opened}, nil
}
func (s *Store) Close() error          { return s.root.Close() }
func (s *Store) Identity() os.FileInfo { return s.identity }
func (s *Store) CheckIdentity() error {
	i, err := os.Lstat(s.absolute)
	if err != nil {
		return fmt.Errorf("observe record repository root: %w", err)
	}
	if !i.IsDir() || platformIsReparse(i) || !os.SameFile(s.identity, i) {
		return errors.New("record repository root identity changed")
	}
	return nil
}

// CheckRootBinding validates another spelling of the same physical root. It
// does not open or substitute a fresh Store; borrowed IO keeps this owner.
func (s *Store) CheckRootBinding(root string) error {
	if err := s.CheckIdentity(); err != nil {
		return err
	}
	absolute, err := filepath.Abs(root)
	if err != nil || strings.TrimSpace(root) == "" {
		return errors.New("record repository root required")
	}
	for p := absolute; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if !info.IsDir() || platformIsReparse(info) {
			return errors.New("borrowed record root ancestor is not a real directory")
		}
		if p == absolute && !os.SameFile(s.identity, info) {
			return errors.New("borrowed record root physical identity differs")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return s.CheckIdentity()
}
func (d *directory) close() {
	if d.file != nil {
		d.file.Close()
	}
	for i := len(d.levels) - 1; i >= 0; i-- {
		d.levels[i].root.Close()
	}
}
func (d *directory) check() error {
	if err := d.store.CheckIdentity(); err != nil {
		return err
	}
	for _, l := range d.levels {
		info, err := l.parent.Lstat(l.leaf)
		if err != nil {
			return fmt.Errorf("observe record parent %q: %w", l.leaf, err)
		}
		if platformIsReparse(info) || !info.IsDir() || !os.SameFile(l.identity, info) {
			return errors.New("record parent name identity changed")
		}
		if l.private {
			if err := platformMode(info, 0700); err != nil {
				return err
			}
		}
	}
	return nil
}
func storedName(r *os.Root, leaf string) error {
	f, err := r.Open(".")
	if err != nil {
		return fmt.Errorf("open stored-name parent for %q: %w", leaf, err)
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return fmt.Errorf("read stored-name parent for %q: %w", leaf, err)
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name(), leaf) && e.Name() != leaf {
			return errors.New("record path aliases a differently spelled stored name")
		}
	}
	return nil
}
func (s *Store) openDirectory(relative string, create, private bool) (result *directory, resultErr error) {
	d := &directory{root: s.root, store: s}
	success := false
	created := false
	defer func() {
		if resultErr != nil {
			resultErr = fmt.Errorf("open record directory %q: %w", relative, resultErr)
		}
		if created && resultErr != nil && !WasPublished(resultErr) && !OutcomeUnknown(resultErr) {
			resultErr = &PublicationError{Path: relative, Outcome: Published, Cause: resultErr}
		}
		if !success {
			d.close()
		}
	}()
	if err := s.CheckIdentity(); err != nil {
		return nil, err
	}
	if relative != "." && relative != "" {
		for idx, leaf := range strings.Split(relative, "/") {
			if !safeComponent(leaf) && leaf != ".markitect" {
				return nil, errors.New("unsafe record directory component")
			}
			if err := d.check(); err != nil {
				return nil, err
			}
			if err := storedName(d.root, leaf); err != nil {
				return nil, err
			}
			info, err := d.root.Lstat(leaf)
			if errors.Is(err, fs.ErrNotExist) && !create {
				if checkErr := d.confirmMissingName(leaf); checkErr != nil {
					return nil, checkErr
				}
				return nil, &directoryAbsent{cause: err}
			}
			collided := false
			if errors.Is(err, fs.ErrNotExist) && create {
				if err := d.check(); err != nil {
					return nil, err
				}
				parent, openErr := d.root.Open(".")
				if openErr != nil {
					return nil, fmt.Errorf("open creation parent for %q: %w", leaf, openErr)
				}
				writer, writerErr := platformWritableParent(parent)
				parent.Close()
				if writerErr != nil {
					return nil, writerErr
				}
				if s.hooks.beforeMkdir != nil {
					if err = s.hooks.beforeMkdir(); err != nil {
						writer.Close()
						return nil, err
					}
				}
				if err = d.check(); err != nil {
					writer.Close()
					return nil, err
				}
				err = platformMkdir(d.root, writer, leaf)
				writer.Close()
				collided = errors.Is(err, fs.ErrExist) && !WasPublished(err) && !OutcomeUnknown(err)
				if err != nil && !collided {
					return nil, err
				}
				if collided {
					if err = d.check(); err != nil {
						return nil, err
					}
					if err = storedName(d.root, leaf); err != nil {
						return nil, err
					}
				} else {
					created = true
				}
				if !collided && s.hooks.directoryCreated != nil {
					if err = s.hooks.directoryCreated(); err != nil {
						return nil, err
					}
				}
				info, err = d.root.Lstat(leaf)
			}
			if err != nil {
				return nil, fmt.Errorf("observe directory child %q: %w", leaf, err)
			}
			if !info.IsDir() || platformIsReparse(info) {
				return nil, errors.New("unsafe record parent directory")
			}
			if private && idx >= 1 {
				if err := platformMode(info, 0700); err != nil {
					return nil, err
				}
			}
			if s.hooks.beforeOpen != nil {
				if err := s.hooks.beforeOpen(); err != nil {
					return nil, err
				}
			}
			child, err := d.root.OpenRoot(leaf)
			if err != nil {
				return nil, fmt.Errorf("open directory child %q: %w", leaf, err)
			}
			actual, err := child.Stat(".")
			if err != nil || !os.SameFile(info, actual) {
				child.Close()
				return nil, errors.New("record directory changed while opening")
			}
			if private && idx >= 1 {
				if err := platformMode(actual, 0700); err != nil {
					child.Close()
					return nil, err
				}
			}
			d.levels = append(d.levels, level{parent: d.root, leaf: leaf, root: child, identity: actual, private: private && idx >= 1})
			d.root = child
			if collided {
				if err = syncObservedAncestor(d); err != nil {
					return nil, err
				}
			}
		}
	}
	var err error
	d.file, err = d.root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("open held record directory: %w", err)
	}
	if err = d.check(); err != nil {
		return nil, err
	}
	success = true
	return d, nil
}

// A concurrent creator owns the creation outcome. A writer admitting its
// already-created ancestor still validates and flushes child then parent.
func syncObservedAncestor(d *directory) error {
	if err := d.check(); err != nil {
		return err
	}
	last := d.levels[len(d.levels)-1]
	for _, root := range []*os.Root{last.root, last.parent} {
		observed, err := root.Open(".")
		if err != nil {
			return err
		}
		writer, err := platformWritableParent(observed)
		observed.Close()
		if err != nil {
			return err
		}
		if err = d.check(); err == nil && d.store.hooks.directorySync != nil {
			err = d.store.hooks.directorySync()
		}
		if err == nil {
			err = platformSyncDir(writer)
		}
		err = errors.Join(err, writer.Close())
		if err != nil {
			return err
		}
	}
	return d.check()
}
func (s *Store) CheckWrite(kind Kind) error {
	if !validKind(kind) {
		return errors.New("unknown record kind")
	}
	if err := s.CheckCutover(kind); err != nil {
		return err
	}
	d, err := s.openDirectory(".", false, false)
	if err != nil {
		return err
	}
	defer d.close()
	f, err := platformWritableParent(d.file)
	if err != nil {
		return err
	}
	defer f.Close()
	return platformCheckWriter(f)
}
func (s *Store) Ensure(kind Kind, id, role string) error {
	if err := s.CheckWrite(kind); err != nil {
		return err
	}
	p, err := directoryPath(kind, id, role)
	if err != nil {
		return err
	}
	d, err := s.openDirectory(p, true, true)
	if err != nil {
		return err
	}
	defer d.close()
	return d.check()
}

// CreateContainer refuses an existing ID even when its first record is absent.
func (s *Store) CreateContainer(kind Kind, id string) error {
	if !validID(kind, id) {
		return errors.New("invalid container ID")
	}
	if err := s.CheckWrite(kind); err != nil {
		return err
	}
	p, _ := directoryPath(kind, "", "")
	d, err := s.openDirectory(p, true, true)
	if err != nil {
		return err
	}
	defer d.close()
	if err = storedName(d.root, id); err != nil {
		return err
	}
	if _, err = d.root.Lstat(id); err == nil {
		return fs.ErrExist
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	parent, err := platformWritableParent(d.file)
	if err != nil {
		return err
	}
	defer parent.Close()
	if err = d.check(); err != nil {
		return err
	}
	if err = platformMkdir(d.root, parent, id); err != nil {
		return err
	}
	if err = d.check(); err != nil {
		return &PublicationError{Path: p + "/" + id, Outcome: Published, Cause: err}
	}
	return nil
}

// CheckCutover observes old names only. It does not decode, import or mutate
// historical authority, and cannot fence an unsupported older binary.
func (s *Store) CheckCutover(kind Kind) error {
	old := ""
	switch kind {
	case Run:
		old = ".markitect/runs"
	case Exploration:
		old = ".markitect/state/explorations"
	case Briefing:
		old = ".markitect/state/briefings"
	case Brownfield:
		old = ".markitect/drafts/brownfield"
	default:
		return errors.New("unknown record kind")
	}
	d, err := s.openDirectory(old, false, false)
	if isDirectoryAbsent(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer d.close()
	entries, err := d.file.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if kind == Run {
			info, err := d.root.Lstat(e.Name())
			if err != nil {
				return err
			}
			if platformIsReparse(info) {
				return errors.New("unsafe old run entry")
			}
			if e.Name() == "private" && info.IsDir() {
				continue
			}
			if info.Mode().IsRegular() && safeComponent(e.Name()) && strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			if info.IsDir() && hex32.MatchString(e.Name()) {
				child, err := s.openDirectory(old+"/"+e.Name(), false, false)
				if err != nil {
					return err
				}
				children, err := child.file.ReadDir(-1)
				if err != nil {
					child.close()
					return err
				}
				transport := len(children) > 0
				for _, c := range children {
					i, er := child.root.Lstat(c.Name())
					packet := strings.TrimSuffix(strings.TrimPrefix(c.Name(), "external-response-"), ".json")
					if er != nil {
						child.close()
						return er
					}
					if !i.Mode().IsRegular() || platformIsReparse(i) || c.Name() != "external-response-"+packet+".json" || !hex32.MatchString(packet) {
						transport = false
					}
				}
				child.close()
				if transport {
					continue
				}
			}
		}
		return fmt.Errorf("prior record layout present: %s at %s/%s; no new operation started; original bytes retained", kind, old, e.Name())
	}
	return d.check()
}
