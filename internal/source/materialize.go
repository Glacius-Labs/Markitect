package source

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Materialize writes a snapshot into destination, which must be absent or an
// empty real directory. Every path is validated before anything is written.
func Materialize(s *snapshot.Snapshot, destination string) error {
	return MaterializeWithLimits(s, destination, DefaultLimits())
}

// MaterializeWithLimits is Materialize with caller-selected resource limits.
func MaterializeWithLimits(s *snapshot.Snapshot, destination string, limits Limits) error {
	if s == nil {
		return errors.New("snapshot is nil")
	}
	if destination == "" {
		return errors.New("destination is empty")
	}
	if err := validateLimits(limits); err != nil {
		return err
	}
	if len(s.Files) > limits.MaxFiles {
		return errors.New("snapshot exceeds file-count limit")
	}
	var total int64
	paths := make([]string, 0, len(s.Files))
	for p, data := range s.Files {
		if err := validateRepoPath(p); err != nil {
			return err
		}
		mode := s.Modes[p]
		if mode != snapshot.RegularMode && mode != snapshot.ExecutableMode {
			return fmt.Errorf("unsupported snapshot mode %q for %q", mode, p)
		}
		if int64(len(data)) > limits.MaxFileBytes || total > limits.MaxTotalBytes-int64(len(data)) {
			return fmt.Errorf("snapshot exceeds byte limit at %q", p)
		}
		total += int64(len(data))
		paths = append(paths, p)
	}
	if err := validatePortablePaths(paths); err != nil {
		return err
	}
	sort.Strings(paths)
	dest, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	if err := rejectSymlinkAncestors(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	info, err := os.Lstat(dest)
	if err != nil {
		return fmt.Errorf("stat destination: %w", err)
	}
	if isSymlink(info) || !info.IsDir() {
		return errors.New("destination must be a real directory")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return fmt.Errorf("read destination: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("destination directory is not empty")
	}
	for _, p := range paths {
		if err := writeMaterializedFile(dest, p, s.Files[p], s.Modes[p]); err != nil {
			return err
		}
	}
	return nil
}

func writeMaterializedFile(dest, repoPath string, data []byte, mode string) error {
	parts := strings.Split(repoPath, "/")
	current := dest
	for _, component := range parts[:len(parts)-1] {
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0755); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create destination directory for %q: %w", repoPath, err)
		}
		info, err := os.Lstat(current)
		if err != nil || isSymlink(info) || !info.IsDir() {
			return fmt.Errorf("unsafe destination directory for %q", repoPath)
		}
	}
	file := filepath.Join(current, parts[len(parts)-1])
	perm := os.FileMode(0644)
	if mode == snapshot.ExecutableMode {
		perm = 0755
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("create destination file %q: %w", repoPath, err)
	}
	_, writeErr := io.Copy(f, bytes.NewReader(data))
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("write destination file %q: %w", repoPath, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close destination file %q: %w", repoPath, closeErr)
	}
	return nil
}

func rejectSymlinkAncestors(p string) error {
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
		if isSymlink(info) {
			return fmt.Errorf("path traverses symlink or reparse point: %s", current)
		}
	}
	return nil
}

func validateRepoPath(p string) error {
	if p == "" || !utf8.ValidString(p) || strings.ContainsRune(p, 0) || strings.Contains(p, `\`) || strings.Contains(p, ":") || path.IsAbs(p) {
		return fmt.Errorf("unsafe repository path %q", p)
	}
	clean := path.Clean(p)
	if clean != p || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe repository path %q", p)
	}
	for _, component := range strings.Split(p, "/") {
		if component == "" || component == "." || component == ".." || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
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

func snapshotPaths(s *snapshot.Snapshot) []string {
	paths := make([]string, 0, len(s.Files))
	for p := range s.Files {
		paths = append(paths, p)
	}
	return paths
}

// validatePortablePaths rejects names that collapse to the same path on a
// case-insensitive filesystem, including a file/directory collision in any
// parent component.
func validatePortablePaths(paths []string) error {
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
		if err := validateRepoPath(p); err != nil {
			return err
		}
		folded := strings.ToLower(p)
		if excludedFilePath(p) || excludedDirectoryPath(p) || excludedFilePath(folded) || excludedDirectoryPath(folded) {
			return fmt.Errorf("repository path %q is excluded from Markitect source", p)
		}
	}
	return validatePortablePaths(paths)
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
