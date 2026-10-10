//go:build windows

package projectadoption

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32Lock     = syscall.NewLazyDLL("kernel32.dll")
	lockFileExProc   = kernel32Lock.NewProc("LockFileEx")
	unlockFileExProc = kernel32Lock.NewProc("UnlockFileEx")
	moveFileExProc   = kernel32Lock.NewProc("MoveFileExW")
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
	movefileReplaceExisting = 0x00000001
	movefileWriteThrough    = 0x00000008
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
	var overlapped syscall.Overlapped
	result, _, callErr := lockFileExProc.Call(file.Fd(), lockfileExclusiveLock|lockfileFailImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if result == 0 {
		_ = file.Close()
		return nil, fmt.Errorf("process lock is held or unavailable: %w", callErr)
	}
	pathInfo, err = os.Lstat(path)
	if err != nil || !pathInfo.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, pathInfo) {
		_, _, _ = unlockFileExProc.Call(file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
		_ = file.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("process lock path changed while acquiring lock: %q", path)
	}
	return func() {
		_, _, _ = unlockFileExProc.Call(file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
		_ = file.Close()
	}, nil
}

func atomicReplaceFile(source, destination string) error {
	sourcePtr, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPtr, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	result, _, callErr := moveFileExProc.Call(uintptr(unsafe.Pointer(sourcePtr)), uintptr(unsafe.Pointer(destinationPtr)), movefileReplaceExisting|movefileWriteThrough)
	if result == 0 {
		if callErr != nil {
			return callErr
		}
		return syscall.GetLastError()
	}
	return nil
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
