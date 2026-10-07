//go:build !windows

package inventory

import "os"

func isLink(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink != 0
}
