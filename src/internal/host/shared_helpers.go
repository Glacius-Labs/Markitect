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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
