package host

import (
	"fmt"
	"os"
	"strings"
)

func checkDiskComponentAlias(parent, component, repoPath string) error {
	entries, err := os.ReadDir(parent)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect path %s: %w", repoPath, err)
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), component) && entry.Name() != component {
			return fmt.Errorf("case-insensitive path collision at %q with existing component %q", repoPath, entry.Name())
		}
	}
	return nil
}

func cloneByteMap(input map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(input))
	for key, value := range input {
		result[key] = append([]byte(nil), value...)
	}
	return result
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
