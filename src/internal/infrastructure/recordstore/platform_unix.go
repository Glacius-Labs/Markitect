//go:build !windows

package recordstore

import (
	"fmt"
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
	f, err := r.Open(leaf)
	if err != nil {
		return &PublicationError{leaf, Published, err}
	}
	defer f.Close()
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
func platformPublish(r *os.Root, _ *os.File, f *os.File, tmp, leaf string, replace bool) (bool, error) {
	if err := f.Close(); err != nil {
		return false, err
	}
	if replace {
		if err := r.Rename(tmp, leaf); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := r.Link(tmp, leaf); err != nil {
		return false, err
	}
	return true, r.Remove(tmp)
}
func platformLock(f *os.File) (func() error, error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, err
	}
	return func() error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }, nil
}
