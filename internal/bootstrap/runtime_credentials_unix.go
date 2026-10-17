//go:build unix

package bootstrap

import (
	"os"
	"syscall"
)

func credentialRedirected(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }

func openCredentialFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
}
