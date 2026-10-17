//go:build unix

package checks

import (
	"os"
	"syscall"
)

func redirected(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }

func openExecutable(root *os.Root, name string) (*os.File, error) {
	// Nonblocking avoids a hung open if a regular path is replaced by a FIFO;
	// no-follow refuses a leaf symlink introduced between Lstat and open.
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
}
