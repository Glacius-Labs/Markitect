//go:build windows

package checks

import (
	"os"
	"syscall"
)

func redirected(info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func openExecutable(root *os.Root, name string) (*os.File, error) { return root.Open(name) }
