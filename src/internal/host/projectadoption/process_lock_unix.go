//go:build !windows

package projectadoption

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func acquireProcessFileLock(path string) (func(), error) {
	before, err := os.Lstat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && (!before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf("unsafe process lock file %q", path)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	pathInfo, err := os.Lstat(path)
	if err != nil || !pathInfo.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, pathInfo) {
		_ = file.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("process lock path changed during open: %q", path)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("process lock is held: %w", err)
	}
	pathInfo, err = os.Lstat(path)
	if err != nil || !pathInfo.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, pathInfo) {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("process lock path changed while acquiring lock: %q", path)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

func atomicReplaceFile(source, destination string) error {
	if err := os.Rename(source, destination); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func ensureNoIncompleteAtomicWrite(dir, basename string) error {
	for _, suffix := range []string{".tmp", ".previous"} {
		path := dir + string(os.PathSeparator) + "." + basename + suffix
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("ledger publication has an unreconciled %s file", suffix)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}
