//go:build windows

package source

import (
	"os"
	"syscall"
)

func isSymlink(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	if data, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
	}
	return false
}
