package recordstore

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path"
	"time"
)

// Lease keeps the approved parent and exact leaf alive through release.
type Lease struct {
	file      *os.File
	directory *directory
	identity  os.FileInfo
	leaf      string
	exclusive bool
	unlock    func() error
	data      []byte
}

// LeaseBusy identifies only an actual terminal lease-leaf create collision.
// Ancestor creation collisions and other IO failures are not held leases.
type LeaseBusy struct{ Cause error }

func (e *LeaseBusy) Error() string { return "record lease is held: " + e.Cause.Error() }
func (e *LeaseBusy) Unwrap() error { return e.Cause }

func (l *Lease) Close() error {
	err := l.Check()
	if l.unlock != nil {
		err = errors.Join(err, l.unlock())
	}
	if l.exclusive {
		if err == nil {
			err = errors.Join(err, removeOwned(l.directory, l.leaf, l.identity))
		}
	}
	err = errors.Join(err, l.file.Close())
	l.directory.close()
	return err
}
func (l *Lease) Check() error {
	if err := l.directory.check(); err != nil {
		return err
	}
	i, err := l.directory.root.Lstat(l.leaf)
	if err != nil {
		return err
	}
	if platformIsReparse(i) || !os.SameFile(l.identity, i) {
		return errors.New("record lease name identity changed")
	}
	if l.exclusive {
		current, err := readLeaf(l.directory, l.leaf, int64(len(l.data))+1)
		if err != nil {
			return err
		}
		if !bytes.Equal(current.Bytes, l.data) {
			return errors.New("record lease bytes changed")
		}
	} else {
		held, err := l.file.Stat()
		if err != nil || held.Size() != 0 {
			return errors.New("kernel lease must remain empty")
		}
	}
	return nil
}
func leaseName(kind Kind, id, leaf string) (Name, error) {
	n := Name{Kind: kind, ID: id, Role: leaf}
	if _, err := n.Path(); err != nil {
		return Name{}, err
	}
	valid := kind == Run && id == "" && leaf == ".write.lock" || kind == Briefing && leaf == ".briefings.lock" || kind == Brownfield && (leaf == "session.lock" || leaf == "manager-runs.lock")
	if !valid {
		return Name{}, errors.New("unsupported lease role")
	}
	return n, nil
}
func (s *Store) Exclusive(kind Kind, id, leaf string, data []byte, wait time.Duration, dead func([]byte) (bool, error)) (*Lease, error) {
	if kind != Run && kind != Briefing {
		return nil, errors.New("exclusive leases support only Run and Briefing")
	}
	if kind != Run && dead != nil {
		return nil, errors.New("dead-owner recovery is restricted to the Runtime Run lease")
	}
	if wait < 0 || kind == Briefing && wait > 10*time.Second {
		return nil, errors.New("lease wait exceeds the approved bound")
	}
	n, err := leaseName(kind, id, leaf)
	if err != nil {
		return nil, err
	}
	if err = s.CheckWrite(kind); err != nil {
		return nil, err
	}
	p, _ := n.Path()
	d, err := s.openDirectory(path.Dir(p), true, true)
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			d.close()
		}
	}()
	parent, err := platformWritableParent(d.file)
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	deadline := time.Now().Add(wait)
	for {
		if err = d.check(); err != nil {
			return nil, err
		}
		f, err := platformCreate(d.root, parent, leaf)
		if errors.Is(err, fs.ErrExist) && dead != nil {
			prior, e := readLeaf(d, leaf, 1<<20)
			if e != nil {
				return nil, e
			}
			remove, e := dead(prior.Bytes)
			if e != nil {
				return nil, e
			}
			if remove {
				if e = removeOwned(d, leaf, prior.identity, prior.Bytes); e != nil {
					return nil, e
				}
				continue
			}
		}
		if errors.Is(err, fs.ErrExist) && wait > 0 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				return nil, &LeaseBusy{Cause: err}
			}
			return nil, err
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, &PublicationError{Path: p, Outcome: Published, Cause: err}
		}
		l := &Lease{file: f, directory: d, identity: info, leaf: leaf, exclusive: true, data: append([]byte(nil), data...)}
		if err = f.Chmod(0600); err == nil {
			_, err = f.Write(data)
		}
		if err == nil {
			err = f.Sync()
		}
		if err == nil {
			err = platformSyncDir(parent)
		}
		if err == nil {
			err = l.Check()
		}
		if err == nil {
			held, openErr := platformLeaseHandle(d, parent, f, leaf, info)
			if openErr != nil {
				err = openErr
			} else {
				f = held
				l.file = held
				err = l.Check()
			}
		}
		if err != nil {
			_ = f.Close()
			return nil, &PublicationError{Path: p, Outcome: Published, Cause: err}
		}
		success = true
		return l, nil
	}
}

// EstablishKernelLeaf is creator-only setup, before the first Session record.
func (s *Store) EstablishKernelLeaf(id, leaf string) error {
	n, err := leaseName(Brownfield, id, leaf)
	if err != nil {
		return err
	}
	if err = s.CheckWrite(Brownfield); err != nil {
		return err
	}
	p, _ := n.Path()
	d, err := s.openDirectory(path.Dir(p), true, true)
	if err != nil {
		return err
	}
	defer d.close()
	existing, err := readLeaf(d, leaf, 1)
	if err != nil {
		return err
	}
	if existing.Exists {
		if len(existing.Bytes) != 0 {
			return errors.New("kernel lease leaf must remain empty")
		}
		return nil
	}
	parent, err := platformWritableParent(d.file)
	if err != nil {
		return err
	}
	defer parent.Close()
	f, err := platformCreate(d.root, parent, leaf)
	if err != nil {
		return err
	}
	defer f.Close()
	created := func(err error) error {
		if err != nil {
			return &PublicationError{Path: p, Outcome: Published, Cause: err}
		}
		return nil
	}
	if err = f.Chmod(0600); err != nil {
		return created(err)
	}
	if s.hooks.fileSync != nil {
		if err = s.hooks.fileSync(); err != nil {
			return created(err)
		}
	}
	if err = f.Sync(); err != nil {
		return created(err)
	}
	if s.hooks.directorySync != nil {
		if err = s.hooks.directorySync(); err != nil {
			return created(err)
		}
	}
	if err = platformSyncDir(parent); err != nil {
		return created(err)
	}
	return created(d.check())
}

// Kernel opens an established empty leaf without creation or byte mutation.
func (s *Store) Kernel(id, leaf string) (*Lease, error) {
	n, err := leaseName(Brownfield, id, leaf)
	if err != nil {
		return nil, err
	}
	if err = s.CheckCutover(Brownfield); err != nil {
		return nil, err
	}
	p, _ := n.Path()
	d, err := s.openDirectory(path.Dir(p), false, true)
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			d.close()
		}
	}()
	for _, name := range []string{"session.lock", "manager-runs.lock"} {
		state, err := readLeaf(d, name, 1)
		if err != nil {
			return nil, err
		}
		if !state.Exists || len(state.Bytes) != 0 {
			return nil, errors.New("Brownfield established kernel lease missing or nonempty")
		}
	}
	prior, err := readLeaf(d, leaf, 1)
	if err != nil {
		return nil, err
	}
	f, err := d.root.OpenFile(leaf, os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !os.SameFile(prior.identity, info) {
		f.Close()
		return nil, errors.New("kernel lease identity changed during open")
	}
	unlock, err := platformLock(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	l := &Lease{file: f, directory: d, identity: info, leaf: leaf, unlock: unlock}
	if err = l.Check(); err != nil {
		err = errors.Join(err, l.Close())
		success = true
		return nil, err
	}
	success = true
	return l, nil
}
