package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func loadWorkingTree(root string, s *snapshot.Snapshot, limits Limits) error {
	var total int64
	var walk func(string, string) error
	walk = func(dir, relDir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("read source directory %q: %w", relDir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			rel := name
			if relDir != "" {
				rel = relDir + "/" + name
			}
			if excludedFilePath(rel) {
				continue
			}
			info, err := os.Lstat(filepath.Join(dir, name))
			if err != nil {
				return fmt.Errorf("stat source path %q: %w", rel, err)
			}
			if info.IsDir() {
				if excludedDirectoryPath(rel) {
					continue
				}
				if err := validateRepoPath(rel); err != nil {
					return err
				}
				if isSymlink(info) {
					return fmt.Errorf("refusing to traverse symlink or reparse point %q", rel)
				}
				if err := walk(filepath.Join(dir, name), rel); err != nil {
					return err
				}
				continue
			}
			if isSymlink(info) {
				return fmt.Errorf("refusing to read symlink or reparse point %q", rel)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported non-regular source file %q", rel)
			}
			if err := validateRepoPath(rel); err != nil {
				return err
			}
			if info.Size() > limits.MaxFileBytes {
				return fmt.Errorf("source file %q exceeds per-file limit", rel)
			}
			if err := accountFile(s, limits); err != nil {
				return err
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				return fmt.Errorf("read source file %q: %w", rel, err)
			}
			if int64(len(data)) > limits.MaxFileBytes || total > limits.MaxTotalBytes-int64(len(data)) {
				return fmt.Errorf("source snapshot exceeds byte limit at %q", rel)
			}
			total += int64(len(data))
			s.Files[rel] = data
			mode := snapshot.RegularMode
			if info.Mode().Perm()&0111 != 0 {
				mode = snapshot.ExecutableMode
			}
			s.Modes[rel] = mode
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return err
	}
	return validatePortablePaths(snapshotPaths(s))
}

func excludedDirectoryPath(p string) bool {
	parts := strings.Split(p, "/")
	for _, component := range parts {
		if component == ".git" {
			return true
		}
	}
	if len(parts) > 0 {
		switch parts[0] {
		case "docs", ".agents", ".claude", ".codex":
			return false
		case ".artifacts", ".cache", ".worktrees", ".venv", "__pycache__", "vendor":
			return true
		}
	}
	for _, component := range parts {
		switch component {
		case "bin", "obj", "node_modules", "vendor":
			return true
		}
	}
	return false
}

func excludedFilePath(p string) bool {
	parts := strings.Split(p, "/")
	for _, component := range parts {
		if component == ".git" {
			return true // linked worktrees store a .git file instead of a directory
		}
	}
	if len(parts) < 2 {
		return false
	}
	return excludedDirectoryPath(strings.Join(parts[:len(parts)-1], "/"))
}
