//go:build windows

package pathspell

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Canonical expands Windows 8.3 path components using the OS API, matching
// the spelling accepted by Markitect's explicit-record and source-root gates.
func Canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	input, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return "", err
	}
	buffer := make([]uint16, 260)
	for {
		n, callErr := syscall.GetLongPathName(input, &buffer[0], uint32(len(buffer)))
		if callErr != nil {
			return "", fmt.Errorf("GetLongPathNameW: %w", callErr)
		}
		if n == 0 {
			return "", fmt.Errorf("GetLongPathNameW returned an empty path")
		}
		if int(n) >= len(buffer) {
			buffer = make([]uint16, int(n)+1)
			continue
		}
		canonical := filepath.Clean(syscall.UTF16ToString(buffer[:n]))
		if _, err := os.Lstat(canonical); err != nil {
			return "", err
		}
		return canonical, nil
	}
}
