//go:build !windows

package guardedwrite

import (
	"os"
	"path/filepath"
)

// CanonicalPathSpelling returns the cleaned path; only Windows has alternate
// spellings to expand.
func CanonicalPathSpelling(path string) (string, error) {
	return filepath.Clean(path), nil
}

// SamePathSpelling reports whether two canonical spellings name the same path.
func SamePathSpelling(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

// IsReparsePoint reports whether info describes a symlink or, on Windows, any
// reparse point.
func IsReparsePoint(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink != 0
}
