package projectrun

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func safeRepoPath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\:\x00") || strings.HasPrefix(value, "/") {
		return false
	}
	if path.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return false
		}
		if runtime.GOOS == "windows" {
			base := strings.ToUpper(strings.TrimSuffix(segment, path.Ext(segment)))
			switch base {
			case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
				return false
			}
		}
	}
	return true
}

func rejectReparsePath(root, relative string) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect project root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("project root must be a real directory")
	}
	current := root
	for _, segment := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) && current == filepath.Join(root, ".markitect") {
			return fmt.Errorf("%s does not exist", relative)
		}
		if statErr != nil {
			return fmt.Errorf("inspect %s: %w", relative, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s contains a symlink", relative)
		}
		if current != filepath.Join(root, filepath.FromSlash(relative)) && !info.IsDir() {
			return fmt.Errorf("%s parent must be a directory", relative)
		}
	}
	return nil
}
