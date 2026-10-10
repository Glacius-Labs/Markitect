// Package source loads fixed or provisional snapshots from either the current
// working tree or a committed Git tree.
package source

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
)

// Limits bounds memory and filesystem use while loading and materializing a
// source snapshot. Load uses DefaultLimits; callers with larger trusted inputs
// can use LoadWithLimits.
type Limits struct {
	MaxFiles      int
	MaxFileBytes  int64
	MaxTotalBytes int64
}

// DefaultMaxFiles, DefaultMaxFileBytes and DefaultMaxTotalBytes bound
// accidental or hostile input while allowing typical documentation trees.
const (
	DefaultMaxFiles      = 100_000
	DefaultMaxFileBytes  = 64 << 20
	DefaultMaxTotalBytes = 512 << 20
)

// DefaultLimits returns the package's default resource bounds.
func DefaultLimits() Limits {
	return Limits{MaxFiles: DefaultMaxFiles, MaxFileBytes: DefaultMaxFileBytes, MaxTotalBytes: DefaultMaxTotalBytes}
}

// Load reads the working tree when revision is empty, or the immutable Git
// commit named by revision otherwise.
func Load(root, revision string) (*snapshot.Snapshot, error) {
	return LoadWithLimits(root, revision, DefaultLimits())
}

// LoadWithLimits is Load with caller-selected resource limits.
func LoadWithLimits(root, revision string, limits Limits) (*snapshot.Snapshot, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	if root == "" {
		return nil, errors.New("source root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve source root: %w", err)
	}
	if err := rejectSymlinkAncestors(abs); err != nil {
		return nil, fmt.Errorf("source root: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat source root: %w", err)
	}
	if isSymlink(info) || !info.IsDir() {
		return nil, fmt.Errorf("source root must be a real directory: %s", abs)
	}

	s := &snapshot.Snapshot{ID: revision, Provisional: revision == "", Files: map[string][]byte{}, Modes: map[string]string{}}
	if revision == "" {
		if err := loadWorkingTree(abs, s, limits); err != nil {
			return nil, err
		}
		return s, nil
	}
	commit, err := resolveCommit(abs, revision)
	if err != nil {
		return nil, err
	}
	s.ID = commit
	if err := loadCommit(abs, commit, s, limits); err != nil {
		return nil, err
	}
	return s, nil
}

func validateLimits(l Limits) error {
	if l.MaxFiles <= 0 || l.MaxFileBytes <= 0 || l.MaxTotalBytes <= 0 || l.MaxFileBytes > l.MaxTotalBytes {
		return errors.New("invalid source limits")
	}
	return nil
}

func accountFile(s *snapshot.Snapshot, limits Limits) error {
	if len(s.Files) >= limits.MaxFiles {
		return errors.New("source snapshot exceeds file-count limit")
	}
	return nil
}
