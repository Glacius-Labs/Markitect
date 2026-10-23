//go:build !windows

package source

import "os"

func isSymlink(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink != 0
}
