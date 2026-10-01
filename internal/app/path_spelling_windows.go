//go:build windows

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// canonicalPathSpelling expands Windows 8.3 components without resolving
// reparse points. safeDestination compares this spelling with EvalSymlinks
// after independently rejecting symlink/reparse ancestors.
func canonicalPathSpelling(path string) (string, error) {
	path = longPathAPISpelling(filepath.Clean(path))
	input, err := syscall.UTF16PtrFromString(path)
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
		return filepath.Clean(syscall.UTF16ToString(buffer[:n])), nil
	}
}

func samePathSpelling(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func isReparsePoint(info os.FileInfo) bool {
	if info == nil || info.Mode()&os.ModeSymlink != 0 {
		return info != nil && info.Mode()&os.ModeSymlink != 0
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func longPathAPISpelling(path string) string {
	if strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\\.\`) {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(path, `\\`)
	}
	if filepath.IsAbs(path) && filepath.VolumeName(path) != "" {
		return `\\?\` + path
	}
	return path
}
