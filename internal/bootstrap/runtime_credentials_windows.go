//go:build windows

package bootstrap

import (
	"os"
	"syscall"
)

func credentialRedirected(info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func openCredentialFile(root *os.Root, name string) (*os.File, error) { return root.Open(name) }
