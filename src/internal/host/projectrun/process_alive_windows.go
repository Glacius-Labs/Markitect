//go:build windows

package projectrun

import (
	"syscall"
)

func processAlive(pid int) (bool, error) {
	const queryLimitedInformation = 0x1000
	h, err := syscall.OpenProcess(queryLimitedInformation, false, uint32(pid))
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno == 87 {
			return false, nil
		}
		return true, err
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return true, err
	}
	return code == 259, nil
}
