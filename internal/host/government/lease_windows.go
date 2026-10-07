//go:build windows

package government

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	leaseKernel32     = syscall.NewLazyDLL("kernel32.dll")
	leaseLockFileEx   = leaseKernel32.NewProc("LockFileEx")
	leaseUnlockFileEx = leaseKernel32.NewProc("UnlockFileEx")
)

type leaseOverlapped struct {
	Internal     uintptr
	InternalHigh uintptr
	Offset       uint32
	OffsetHigh   uint32
	HEvent       syscall.Handle
}

func lockLeaseFile(file *os.File) error {
	var overlapped leaseOverlapped
	overlapped.Offset = 0x7fffffff
	const lockfileExclusiveLock = 0x00000002
	const lockfileFailImmediately = 0x00000001
	r1, _, err := leaseLockFileEx.Call(file.Fd(), lockfileExclusiveLock|lockfileFailImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if r1 == 0 {
		return err
	}
	return nil
}

func unlockLeaseFile(file *os.File) error {
	if file == nil {
		return nil
	}
	var overlapped leaseOverlapped
	overlapped.Offset = 0x7fffffff
	r1, _, err := leaseUnlockFileEx.Call(file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if r1 == 0 {
		return err
	}
	return nil
}

func verifyLeaseFileLock(file *os.File) error {
	if file == nil {
		return errors.New("lease file is closed")
	}
	return nil
}
