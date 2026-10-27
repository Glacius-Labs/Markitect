package release

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/guardedwrite"
)

func loadWorkingTree(root string, s *Input, limits Limits) error {
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
			if guardedwrite.ExcludedFile(rel) {
				continue
			}
			info, err := os.Lstat(filepath.Join(dir, name))
			if err != nil {
				return fmt.Errorf("stat source path %q: %w", rel, err)
			}
			if info.IsDir() {
				if guardedwrite.ExcludedDirectory(rel) {
					continue
				}
				if err := guardedwrite.ValidateRepoPath(rel); err != nil {
					return err
				}
				if guardedwrite.IsReparsePoint(info) {
					return fmt.Errorf("refusing to traverse symlink or reparse point %q", rel)
				}
				if err := walk(filepath.Join(dir, name), rel); err != nil {
					return err
				}
				continue
			}
			if guardedwrite.IsReparsePoint(info) {
				return fmt.Errorf("refusing to read symlink or reparse point %q", rel)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported non-regular source file %q", rel)
			}
			if err := guardedwrite.ValidateRepoPath(rel); err != nil {
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
			mode := RegularMode
			if info.Mode().Perm()&0111 != 0 {
				mode = ExecutableMode
			}
			s.Modes[rel] = mode
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return err
	}
	if err := guardedwrite.ValidatePortablePaths(snapshotPaths(s)); err != nil {
		return err
	}
	return applyIndexModes(root, s)
}

// applyIndexModes gives tracked regular files their Git index mode when Git
// ignores executable bits (core.filemode=false, as Git for Windows sets it),
// so a clean checkout has its commit's modes on every platform. Untracked and
// unmerged files, and roots outside a Git working tree, keep filesystem modes.
func applyIndexModes(root string, s *Input) error {
	enabled, err := GitFileModeEnabled(root)
	if err == nil && enabled {
		return nil
	}
	var index []byte
	if err == nil {
		index, err = guardedwrite.GitOutput(root, "ls-files", "--stage", "-z")
	}
	if err != nil {
		inside, insideErr := guardedwrite.GitOutput(root, "rev-parse", "--is-inside-work-tree")
		if insideErr != nil || strings.TrimSpace(string(inside)) != "true" {
			return nil
		}
		return fmt.Errorf("inspect Git index modes: %w", err)
	}
	// ls-files run in root lists only entries beneath it, relative to it.
	for _, record := range bytes.Split(index, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		fields := strings.Fields(string(header))
		if !ok || len(fields) != 3 {
			return errors.New("malformed Git index mode record")
		}
		repoPath, mode := string(name), fields[0]
		if _, loaded := s.Files[repoPath]; loaded && fields[2] == "0" && (mode == RegularMode || mode == ExecutableMode) {
			s.Modes[repoPath] = mode
		}
	}
	return nil
}

// GitFileModeEnabled returns Git's effective core.filemode setting. When the
// setting is absent, Git defaults it on for all platforms.
func GitFileModeEnabled(root string) (bool, error) {
	output, err := guardedwrite.GitOutput(root, "config", "--bool", "--get", "core.filemode")
	if err == nil {
		switch strings.TrimSpace(string(output)) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf("Git returned an invalid core.filemode value %q", strings.TrimSpace(string(output)))
		}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, err
}
