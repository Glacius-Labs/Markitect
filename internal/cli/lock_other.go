//go:build !windows

package cli

import "path/filepath"

// canonicalPath is the real path of a folder, every symbolic link resolved.
func canonicalPath(dir string) (string, error) { return filepath.EvalSymlinks(dir) }
