package projectconfig

import (
	"fmt"
	"path"
	"runtime"
	"strings"
)

const MaxInstructionFiles = 128

// ValidateInstructionPaths checks portable, collision-free relative paths for runtime instruction files.
func ValidateInstructionPaths(paths []string) error {
	seen := map[string]string{}
	for _, value := range paths {
		if !safeInstructionPath(value) {
			return fmt.Errorf("candidate contains unsafe repository path %q", value)
		}
		key := strings.ToLower(value)
		if prior, exists := seen[key]; exists && prior != value {
			return fmt.Errorf("candidate paths %q and %q collide on case-insensitive filesystems", prior, value)
		}
		seen[key] = value
	}
	return nil
}

func safeInstructionPath(value string) bool {
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
