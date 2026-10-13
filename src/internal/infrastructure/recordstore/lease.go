package recordstore

import (
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
}

func (l *Lease) Close() error {
	var err error
	if l.unlock != nil {
		err = l.unlock()
	}
	if l.exclusive {
		err = errors.Join(err, removeOwned(l.directory, l.leaf, l.identity))
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
				if e = removeOwned(d, leaf, prior.identity); e != nil {
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
			return nil, err
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, err
		}
		l := &Lease{file: f, directory: d, identity: info, leaf: leaf, exclusive: true}
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
		if err != nil {
			_ = l.Close()
			success = true
			return nil, err
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
	if err = f.Chmod(0600); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = platformSyncDir(parent); err != nil {
		return err
	}
	return d.check()
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
		_ = l.Close()
		success = true
		return nil, err
	}
	success = true
	return l, nil
}
