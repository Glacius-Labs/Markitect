//go:build !windows

package recordstore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

func platformIsReparse(i os.FileInfo) bool { return i.Mode()&os.ModeSymlink != 0 }
func platformMode(i os.FileInfo, mode os.FileMode) error {
	if i.Mode().Perm() != mode {
		return fmt.Errorf("record object requires mode %04o, observed %04o", mode, i.Mode().Perm())
	}
	return nil
}
func platformWritableParent(f *os.File) (*os.File, error) {
	fd, err := syscall.Dup(int(f.Fd()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), f.Name()), nil
}
func platformCheckWriter(f *os.File) error { return f.Sync() }
func platformSyncDir(f *os.File) error     { return f.Sync() }
func platformMkdir(r *os.Root, parent *os.File, leaf string) error {
	if err := r.Mkdir(leaf, 0700); err != nil {
		return err
	}
	named, err := r.Lstat(leaf)
	if err != nil || !named.IsDir() || platformIsReparse(named) {
		return &PublicationError{leaf, Published, errors.New("created directory name changed")}
	}
	f, err := r.Open(leaf)
	if err != nil {
		return &PublicationError{leaf, Published, err}
	}
	defer f.Close()
	held, err := f.Stat()
	if err != nil || !os.SameFile(named, held) {
		return &PublicationError{leaf, Published, errors.New("created directory identity changed")}
	}
	if err = f.Chmod(0700); err == nil {
		err = f.Sync()
	}
	if err == nil {
		err = parent.Sync()
	}
	if err != nil {
		return &PublicationError{leaf, Published, err}
	}
	return nil
}
func platformCreate(r *os.Root, _ *os.File, leaf string) (*os.File, error) {
	return r.OpenFile(leaf, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
}
func platformLeaseHandle(_ *directory, _ *os.File, f *os.File, _ string, _ os.FileInfo) (*os.File, error) {
	return f, nil
}
func platformPublish(r *os.Root, _ *os.File, f *os.File, tmp, leaf string, replace bool) (bool, error) {
	if replace {
		if err := r.Rename(tmp, leaf); err != nil {
			return false, errors.Join(errAmbiguous, err)
		}
		return true, nil
	}
	if err := r.Link(tmp, leaf); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, err
		}
		return false, errors.Join(errAmbiguous, err)
	}
	identity, err := f.Stat()
	if err != nil {
		return true, err
	}
	named, err := r.Lstat(tmp)
	if err != nil || !os.SameFile(identity, named) {
		return true, errors.New("published record retained; temporary name changed")
	}
	return true, r.Remove(tmp)
}
func platformRemoveOwned(d *directory, name string, identity os.FileInfo) error {
	// POSIX has no conditional-inode unlink. Supported mutators honor the
	// owning lease; observed substitution is refused before the name operation.
	current, err := d.root.Lstat(name)
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() || platformIsReparse(current) || !os.SameFile(current, identity) {
		return errors.New("cleanup retains replaced record leaf")
	}
	if err := platformMode(current, 0600); err != nil {
		return err
	}
	if err := d.check(); err != nil {
		return err
	}
	return d.root.Remove(name)
}
func platformLock(f *os.File) (func() error, error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, err
	}
	return func() error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }, nil
}
