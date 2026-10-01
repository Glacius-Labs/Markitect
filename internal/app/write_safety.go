package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/source"
)

func safeDestination(root, name string) (string, error) {
	if name == "" || strings.Contains(name, "\\") || strings.Contains(name, ":") || filepath.IsAbs(name) {
		return "", fmt.Errorf("unsafe output path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return "", fmt.Errorf("unsafe output path %q", name)
		}
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	// Check the root and every ancestor directly; EvalSymlinks alone does not
	// identify all Windows reparse tags and textual equality misses short names.
	if err := rejectReparseAncestors(absolute); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalPathSpelling(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize output root spelling: %w", err)
	}
	resolvedCanonical, err := canonicalPathSpelling(resolved)
	if err != nil {
		return "", fmt.Errorf("canonicalize resolved output root: %w", err)
	}
	if !samePathSpelling(resolvedCanonical, canonical) {
		return "", fmt.Errorf("output root contains a symlink: %s", root)
	}
	current := absolute
	for _, part := range strings.Split(name, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if isReparsePoint(info) {
			return "", fmt.Errorf("symlink in output path %s", name)
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return "", err
		}
		canonical, err := canonicalPathSpelling(current)
		if err != nil {
			return "", fmt.Errorf("canonicalize output path spelling: %w", err)
		}
		resolvedCanonical, err := canonicalPathSpelling(resolved)
		if err != nil {
			return "", fmt.Errorf("canonicalize resolved output path: %w", err)
		}
		if !samePathSpelling(resolvedCanonical, canonical) {
			return "", fmt.Errorf("reparse point in output path %s", name)
		}
	}
	return current, nil
}

func rejectReparseAncestors(path string) error {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect output root ancestor %s: %w", current, err)
		}
		if isReparsePoint(info) {
			return fmt.Errorf("output root contains a symlink or reparse point: %s", current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

// hasGitMetadata checks the selected directory and its ancestors so nested
// source roots still receive the same branch guard as repository roots.
func hasGitMetadata(root string) bool {
	current := filepath.Clean(root)
	for {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		current = parent
	}
}

func writableBranch(root string) error {
	_, err := writeBranchName(root)
	return err
}

// writeBranchName returns the named, non-protected Git branch for a write.
func writeBranchName(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	branch, err := source.GitOutput(absolute, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("writing canonical sources requires an isolated Git branch: %w", err)
	}
	name := strings.TrimSpace(string(branch))
	if name == "" || strings.EqualFold(name, "master") || strings.EqualFold(name, "main") {
		return "", fmt.Errorf("writing requires an isolated non-protected Git branch; detached HEAD and protected branches are not writable")
	}
	return name, nil
}

// ensureWriteBranch catches a branch switch, including one to a different
// branch at the same HEAD, during a multi-file write.
func ensureWriteBranch(root, expected string) error {
	current, err := writeBranchName(root)
	if err != nil {
		return fmt.Errorf("write branch changed or became protected: %w", err)
	}
	if current != expected {
		return fmt.Errorf("branch changed during write from %q to %q", expected, current)
	}
	return nil
}

func atomicWrite(dest string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(dest), ".markitect-*")
	if err != nil {
		return err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Chmod(temporary, 0644); err != nil {
		return err
	}
	return os.Rename(temporary, dest)
}

func lockWriter(root string) (func(), error) {
	lockPath, err := safeDestination(root, ".artifacts/markitect/write.lock")
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("renderer lock is already present: %w", err)
	}
	lock.Close()
	return func() { os.Remove(lockPath) }, nil
}
