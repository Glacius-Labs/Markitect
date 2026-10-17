//go:build !windows

package gitrepo

import "os"

func redirected(info os.FileInfo) bool { return info != nil && info.Mode()&os.ModeSymlink != 0 }
