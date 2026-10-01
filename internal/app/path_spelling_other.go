//go:build !windows

package app

import (
	"os"
	"path/filepath"
)

func canonicalPathSpelling(path string) (string, error) {
	return filepath.Clean(path), nil
}

func samePathSpelling(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func isReparsePoint(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink != 0
}
