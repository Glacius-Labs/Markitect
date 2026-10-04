//go:build !windows

package githubcli

import "os"

func isPlatformReparsePoint(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink != 0
}
