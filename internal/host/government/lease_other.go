//go:build !windows

package government

import (
	"errors"
	"os"
	"syscall"
)

func lockLeaseFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
func unlockLeaseFile(file *os.File) error { return syscall.Flock(int(file.Fd()), syscall.LOCK_UN) }
func verifyLeaseFileLock(file *os.File) error {
	if file == nil {
		return errors.New("lease file is closed")
	}
	return nil
}
