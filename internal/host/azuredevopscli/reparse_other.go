//go:build !windows

package azuredevopscli

import "os"

func isPlatformReparsePoint(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink != 0
}
