package source

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/guardedwrite"
)

// Materialize writes a snapshot into destination, which must be absent or an
// empty real directory. Every path is validated before anything is written.
func Materialize(s *Input, destination string) error {
	return MaterializeWithLimits(s, destination, DefaultLimits())
}

// MaterializeWithLimits is Materialize with caller-selected resource limits.
func MaterializeWithLimits(s *Input, destination string, limits Limits) error {
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
	paths := snapshotPaths(s)
	for _, p := range paths {
		data := s.Files[p]
		if err := guardedwrite.ValidateRepoPath(p); err != nil {
			return err
		}
		mode := s.Modes[p]
		if mode != RegularMode && mode != ExecutableMode {
			return fmt.Errorf("unsupported snapshot mode %q for %q", mode, p)
		}
		if int64(len(data)) > limits.MaxFileBytes || total > limits.MaxTotalBytes-int64(len(data)) {
			return fmt.Errorf("snapshot exceeds byte limit at %q", p)
		}
		total += int64(len(data))
	}
	if err := guardedwrite.ValidatePortablePaths(paths); err != nil {
		return err
	}
	dest, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	if err := guardedwrite.RejectSymlinkAncestors(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	info, err := os.Lstat(dest)
	if err != nil {
		return fmt.Errorf("stat destination: %w", err)
	}
	if guardedwrite.IsReparsePoint(info) || !info.IsDir() {
		return errors.New("destination must be a real directory")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return fmt.Errorf("read destination: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("destination directory is not empty")
	}
	// The path checks above are lexical, and Windows resolves an 8.3 short
	// name such as MARKIT~1 to an entry written earlier. The listing goes
	// through an opened root because os.DirFS cannot list a \\?\ root.
	opened, err := os.OpenRoot(dest)
	if err != nil {
		return fmt.Errorf("open destination: %w", err)
	}
	defer opened.Close()
	stored := guardedwrite.NewStoredNames(opened.FS())
	for _, p := range paths {
		if err := writeMaterializedFile(stored, dest, p, s.Files[p], s.Modes[p]); err != nil {
			return err
		}
	}
	return nil
}

func writeMaterializedFile(stored *guardedwrite.StoredNames, dest, repoPath string, data []byte, mode string) error {
	parts := strings.Split(repoPath, "/")
	current := dest
	for _, component := range parts[:len(parts)-1] {
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0755); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create destination directory for %q: %w", repoPath, err)
		}
		info, err := os.Lstat(current)
		if err != nil || guardedwrite.IsReparsePoint(info) || !info.IsDir() {
			return fmt.Errorf("unsafe destination directory for %q", repoPath)
		}
	}
	if err := stored.Require(repoPath); err != nil {
		return err
	}
	file := filepath.Join(current, parts[len(parts)-1])
	perm := os.FileMode(0644)
	if mode == ExecutableMode {
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

// snapshotPaths returns the snapshot paths sorted so that validation
// diagnostics do not depend on map iteration order.
func snapshotPaths(s *Input) []string {
	paths := make([]string, 0, len(s.Files))
	for p := range s.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
