//go:build !windows

package codexappserver

import "os"

func redirected(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }
