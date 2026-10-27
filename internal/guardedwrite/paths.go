package guardedwrite

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RejectSymlinkAncestors rejects p when p or one of its existing ancestors is
// a symlink or reparse point. Missing components are skipped; unlike
// RejectReparseAncestors, p need not exist yet.
func RejectSymlinkAncestors(p string) error {
	vol := filepath.VolumeName(p)
	current := vol + string(filepath.Separator)
	rest := strings.TrimPrefix(p, current)
	if vol == "" {
		current = string(filepath.Separator)
		rest = strings.TrimPrefix(p, current)
	}
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("stat destination ancestor: %w", err)
		}
		if IsReparsePoint(info) {
			return fmt.Errorf("path traverses symlink or reparse point: %s", current)
		}
	}
	return nil
}

// ValidateRepoPath rejects a repository path that is not a clean relative
// path with portable components, including Windows device names and the
// usual 8.3 short name of .git.
func ValidateRepoPath(p string) error {
	if p == "" || !utf8.ValidString(p) || strings.ContainsRune(p, 0) || strings.Contains(p, `\`) || strings.Contains(p, ":") || path.IsAbs(p) {
		return fmt.Errorf("unsafe repository path %q", p)
	}
	clean := path.Clean(p)
	if clean != p || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe repository path %q", p)
	}
	for _, component := range strings.Split(p, "/") {
		// Like Git's core.protectNTFS, refuse git~1, the usual Windows 8.3
		// short name of .git, on every platform.
		if component == "" || component == "." || component == ".." || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") || strings.EqualFold(component, "git~1") {
			return fmt.Errorf("unsafe repository path %q", p)
		}
		for _, r := range component {
			if r < 32 || strings.ContainsRune(`<>"|?*`, r) {
				return fmt.Errorf("unsafe repository path %q", p)
			}
		}
		base := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
		if windowsReservedName(base) {
			return fmt.Errorf("unsafe Windows device path %q", p)
		}
	}
	return nil
}

// RequireStoredName refuses an existing path component that Windows resolved
// through another spelling: an 8.3 short name such as GIT~1 for .git, or a
// case variant. Lexical checks see only the requested text, so a write through
// such an alias would reach a path they never authorized. Only stored names
// appear in a directory listing. It is a no-op outside Windows.
func RequireStoredName(directory fs.FS, parent, part string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	entries, err := fs.ReadDir(directory, parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == part {
			return nil
		}
	}
	return fmt.Errorf("%q names an existing entry stored under another name", part)
}

// StoredNames applies RequireStoredName to write paths below one directory.
// Writers of copies use it where they write, after creating parent
// directories, because an alias can only be resolved on disk. It remembers
// the directories it has proven, so writing a whole tree lists each directory
// once; use one value per tree while nothing else renames its entries.
type StoredNames struct {
	directory fs.FS
	proven    map[string]bool
}

// NewStoredNames returns the check for paths below directory.
func NewStoredNames(directory fs.FS) *StoredNames {
	return &StoredNames{directory: directory, proven: map[string]bool{}}
}

// Require refuses the slash-separated name when one of its existing
// components is stored under another spelling. It checks up to the first
// component that does not exist yet and is a no-op outside Windows.
func (s *StoredNames) Require(name string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	parts := strings.Split(name, "/")
	for i, part := range parts {
		prefix := strings.Join(parts[:i+1], "/")
		if s.proven[prefix] {
			continue
		}
		if _, err := fs.Lstat(s.directory, prefix); errors.Is(err, fs.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		parent := "."
		if i > 0 {
			parent = strings.Join(parts[:i], "/")
		}
		if err := RequireStoredName(s.directory, parent, part); err != nil {
			return fmt.Errorf("unsafe path %s: %w", name, err)
		}
		if i < len(parts)-1 {
			s.proven[prefix] = true
		}
	}
	return nil
}

// ValidatePortablePaths rejects names that collapse to the same path on a
// case-insensitive filesystem, including a file/directory collision in any
// parent component.
func ValidatePortablePaths(paths []string) error {
	seen := make(map[string]struct {
		name string
		dir  bool
	}, len(paths)*2)
	for _, p := range paths {
		parts := strings.Split(p, "/")
		for i := range parts {
			name := strings.Join(parts[:i+1], "/")
			isDir := i < len(parts)-1
			key := foldPath(name)
			if prior, ok := seen[key]; ok && (prior.name != name || prior.dir != isDir) {
				return fmt.Errorf("case-insensitive path collision between %q and %q", prior.name, name)
			}
			seen[key] = struct {
				name string
				dir  bool
			}{name: name, dir: isDir}
		}
	}
	return nil
}

// ValidateIncludedPaths validates prospective repository paths using the same
// safety, inclusion, and portable-name rules as source snapshots. Callers pass
// existing snapshot paths together with new paths they plan to add.
func ValidateIncludedPaths(paths []string) error {
	for _, p := range paths {
		if err := ValidateRepoPath(p); err != nil {
			return err
		}
		folded := strings.ToLower(p)
		if ExcludedFile(p) || ExcludedDirectory(p) || ExcludedFile(folded) || ExcludedDirectory(folded) {
			return fmt.Errorf("repository path %q is excluded from Markitect source", p)
		}
	}
	return ValidatePortablePaths(paths)
}

func foldPath(p string) string {
	var folded strings.Builder
	for _, r := range p {
		min := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < min {
				min = next
			}
		}
		folded.WriteRune(min)
	}
	return folded.String()
}

func windowsReservedName(base string) bool {
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return true
	}
	return false
}

// ExcludedDirectory reports whether source snapshots leave out the directory
// at repository path p: Git metadata, build output, caches and vendored
// dependencies, except below the protected documentation and provider trees.
func ExcludedDirectory(p string) bool {
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

// ExcludedFile reports whether source snapshots leave out the file at
// repository path p: a .git file or a file below an excluded directory.
func ExcludedFile(p string) bool {
	parts := strings.Split(p, "/")
	for _, component := range parts {
		if component == ".git" {
			return true // linked worktrees store a .git file instead of a directory
		}
	}
	if len(parts) < 2 {
		return false
	}
	return ExcludedDirectory(strings.Join(parts[:len(parts)-1], "/"))
}
