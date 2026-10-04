package source

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// GitIdentity identifies the repository location and object format used to
// acquire a fixed Git snapshot. Remote configuration is deliberately omitted:
// it is informational, mutable, and may contain credentials.
type GitIdentity struct {
	Root         string `yaml:"root"`
	GitDir       string `yaml:"gitDir"`
	CommonDir    string `yaml:"commonDir"`
	ObjectFormat string `yaml:"objectFormat"`
	Digest       string `yaml:"digest"`
}

// SelectedSnapshot binds an exact selected-file snapshot to the Git
// repository identity from which it was acquired.
type SelectedSnapshot struct {
	Identity GitIdentity
	Snapshot *snapshot.Snapshot
}

// IdentifyGit requires root to be the real top-level directory of a Git
// repository and returns its canonical worktree and Git-directory identity.
func IdentifyGit(root string) (GitIdentity, error) {
	identity, _, err := identifyGit(root, GitOutput)
	return identity, err
}

// LoadSelected reads only the exact regular-file paths listed from a full,
// lowercase commit ID. All requested paths are validated before Git is
// queried, and all metadata is checked before any blob content is read.
func LoadSelected(root, fullCommit string, paths []string) (*SelectedSnapshot, error) {
	return loadSelected(root, fullCommit, paths, GitOutput, readSelectedBlobs)
}

type gitOutputFunc func(root string, args ...string) ([]byte, error)
type selectedBlobReader func(root string, files []treeFile) (map[string][]byte, error)

type gitIdentityStats struct {
	root   os.FileInfo
	git    os.FileInfo
	common os.FileInfo
}

func identifyGit(root string, run gitOutputFunc) (GitIdentity, gitIdentityStats, error) {
	var zero GitIdentity
	var stats gitIdentityStats
	if root == "" {
		return zero, stats, errors.New("source root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return zero, stats, fmt.Errorf("resolve source root: %w", err)
	}
	if err := rejectSymlinkAncestors(abs); err != nil {
		return zero, stats, fmt.Errorf("source root: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return zero, stats, fmt.Errorf("stat source root: %w", err)
	}
	if isSymlink(info) || !info.IsDir() {
		return zero, stats, fmt.Errorf("source root must be a real directory: %s", abs)
	}
	canonicalRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return zero, stats, fmt.Errorf("canonicalize source root: %w", err)
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return zero, stats, fmt.Errorf("resolve canonical source root: %w", err)
	}
	canonicalRoot = filepath.Clean(canonicalRoot)

	top, err := run(canonicalRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return zero, stats, fmt.Errorf("identify Git worktree root: %w", err)
	}
	canonicalTop, err := canonicalExistingPath(strings.TrimSpace(string(top)))
	if err != nil {
		return zero, stats, fmt.Errorf("canonicalize Git worktree root: %w", err)
	}
	topInfo, err := os.Stat(canonicalTop)
	if err != nil || !topInfo.IsDir() || !os.SameFile(info, topInfo) {
		return zero, stats, fmt.Errorf("source root must be the Git worktree top-level directory: %s", canonicalRoot)
	}
	// Use Git's spelling of the top-level path. On Windows this also expands
	// short 8.3 path aliases returned by callers.
	canonicalRoot = canonicalTop
	gitDirOut, err := run(canonicalRoot, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return zero, stats, fmt.Errorf("identify Git directory: %w", err)
	}
	commonDirOut, err := run(canonicalRoot, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return zero, stats, fmt.Errorf("identify common Git directory: %w", err)
	}
	gitDir, err := canonicalExistingPath(strings.TrimSpace(string(gitDirOut)))
	if err != nil {
		return zero, stats, fmt.Errorf("canonicalize Git directory: %w", err)
	}
	commonDir, err := canonicalExistingPath(strings.TrimSpace(string(commonDirOut)))
	if err != nil {
		return zero, stats, fmt.Errorf("canonicalize common Git directory: %w", err)
	}
	gitInfo, err := os.Stat(gitDir)
	if err != nil || !gitInfo.IsDir() {
		return zero, stats, fmt.Errorf("Git directory is not a readable directory: %s", gitDir)
	}
	commonInfo, err := os.Stat(commonDir)
	if err != nil || !commonInfo.IsDir() {
		return zero, stats, fmt.Errorf("common Git directory is not a readable directory: %s", commonDir)
	}
	objectFormatOut, err := run(canonicalRoot, "rev-parse", "--show-object-format")
	if err != nil {
		return zero, stats, fmt.Errorf("identify Git object format: %w", err)
	}
	objectFormat := strings.TrimSpace(string(objectFormatOut))
	if objectFormat != "sha1" && objectFormat != "sha256" {
		return zero, stats, fmt.Errorf("unsupported Git object format %q", objectFormat)
	}
	identity := GitIdentity{Root: canonicalRoot, GitDir: gitDir, CommonDir: commonDir, ObjectFormat: objectFormat}
	identity.Digest = gitIdentityDigest(identity)
	return identity, gitIdentityStats{root: info, git: gitInfo, common: commonInfo}, nil
}

func canonicalExistingPath(value string) (string, error) {
	if value == "" {
		return "", errors.New("path is empty")
	}
	resolved, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func samePath(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo) {
		return true
	}
	if filepath.VolumeName(left) != "" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func gitIdentityDigest(identity GitIdentity) string {
	h := sha256.New()
	for _, value := range []string{identity.Root, identity.GitDir, identity.CommonDir, identity.ObjectFormat} {
		_, _ = fmt.Fprintf(h, "%d:", len(value))
		_, _ = h.Write([]byte(value))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func loadSelected(root, fullCommit string, requested []string, run gitOutputFunc, readBlobs selectedBlobReader) (*SelectedSnapshot, error) {
	paths := append([]string(nil), requested...)
	if err := validateSelectedPaths(paths); err != nil {
		return nil, fmt.Errorf("validate selected paths: %w", err)
	}
	sort.Strings(paths)
	if len(paths) > DefaultMaxFiles {
		return nil, errors.New("selected snapshot exceeds file-count limit")
	}
	identity, initialStats, err := identifyGit(root, run)
	if err != nil {
		return nil, err
	}
	if len(fullCommit) != 40 && len(fullCommit) != 64 {
		return nil, errors.New("selected source requires a full lowercase commit ID")
	}
	if fullCommit != strings.ToLower(fullCommit) {
		return nil, errors.New("selected source requires a lowercase commit ID")
	}
	if identity.ObjectFormat == "sha1" && len(fullCommit) != 40 || identity.ObjectFormat == "sha256" && len(fullCommit) != 64 {
		return nil, errors.New("commit ID length does not match Git object format")
	}
	if _, err := hex.DecodeString(fullCommit); err != nil {
		return nil, errors.New("selected source requires a full hexadecimal commit ID")
	}
	resolved, err := run(identity.Root, "rev-parse", "--verify", "--end-of-options", fullCommit+"^{commit}")
	if err != nil {
		return nil, fmt.Errorf("verify selected commit: %w", err)
	}
	if strings.TrimSpace(string(resolved)) != fullCommit {
		return nil, errors.New("Git resolved selected commit to a different ID")
	}

	files := make([]treeFile, 0, len(paths))
	var total int64
	for _, path := range paths {
		// --literal-pathspecs makes caller paths data, never Git pathspec syntax.
		out, err := run(identity.Root, "--literal-pathspecs", "ls-tree", "-z", "--full-tree", "--long", fullCommit, "--", path)
		if err != nil {
			return nil, fmt.Errorf("inspect selected path %q: %w", path, err)
		}
		entries, err := parseSelectedTreeEntries(out)
		if err != nil {
			return nil, fmt.Errorf("inspect selected path %q: %w", path, err)
		}
		if len(entries) != 1 || entries[0].path != path {
			return nil, fmt.Errorf("selected path %q does not identify exactly one Git tree entry", path)
		}
		file := entries[0]
		if file.mode == "120000" {
			return nil, fmt.Errorf("Git symlink is not a source file: %q", path)
		}
		if file.mode == "160000" || file.kind == "commit" {
			return nil, fmt.Errorf("Git submodule is not a source file: %q", path)
		}
		if file.kind == "tree" || file.mode == "040000" {
			return nil, fmt.Errorf("selected Git path is a tree, not a file: %q", path)
		}
		if file.kind != "blob" || (file.mode != snapshot.RegularMode && file.mode != snapshot.ExecutableMode) {
			return nil, fmt.Errorf("unsupported Git tree entry %q (mode %s, type %s)", path, file.mode, file.kind)
		}
		if file.size > DefaultMaxFileBytes {
			return nil, fmt.Errorf("source file %q exceeds per-file limit", path)
		}
		if total > DefaultMaxTotalBytes-file.size {
			return nil, fmt.Errorf("selected source exceeds byte limit at %q", path)
		}
		total += file.size
		files = append(files, file.treeFile)
	}
	contents, err := readBlobs(identity.Root, files)
	if err != nil {
		return nil, err
	}
	resolvedSnapshot := &snapshot.Snapshot{ID: fullCommit, Files: make(map[string][]byte, len(files)), Modes: make(map[string]string, len(files))}
	for _, file := range files {
		data, ok := contents[file.path]
		if !ok || int64(len(data)) != file.size {
			return nil, fmt.Errorf("Git blob content missing or changed for %q", file.path)
		}
		if got := gitBlobOID(identity.ObjectFormat, data); got != file.oid {
			return nil, fmt.Errorf("Git blob content hash mismatch for %q", file.path)
		}
		resolvedSnapshot.Files[file.path] = data
		resolvedSnapshot.Modes[file.path] = file.mode
	}
	if err := confirmGitIdentity(identity, initialStats, run); err != nil {
		return nil, err
	}
	return &SelectedSnapshot{Identity: identity, Snapshot: resolvedSnapshot}, nil
}

func validateSelectedPaths(paths []string) error {
	seen := make(map[string]struct{}, len(paths))
	for _, selected := range paths {
		if err := validateRepoPath(selected); err != nil {
			return err
		}
		for _, component := range strings.Split(selected, "/") {
			if component == ".git" {
				return fmt.Errorf("selected path %q enters Git metadata", selected)
			}
		}
		if strings.ContainsAny(selected, "*?[]") {
			return fmt.Errorf("selected path %q contains pathspec metacharacters", selected)
		}
		if _, exists := seen[selected]; exists {
			return fmt.Errorf("duplicate selected path %q", selected)
		}
		seen[selected] = struct{}{}
	}
	return validatePortablePaths(paths)
}

func gitBlobOID(objectFormat string, data []byte) string {
	framed := make([]byte, 0, len(data)+32)
	framed = append(framed, "blob "+strconv.Itoa(len(data))+"\x00"...)
	framed = append(framed, data...)
	switch objectFormat {
	case "sha1":
		digest := sha1.Sum(framed)
		return hex.EncodeToString(digest[:])
	case "sha256":
		digest := sha256.Sum256(framed)
		return hex.EncodeToString(digest[:])
	default:
		return ""
	}
}

type selectedTreeEntry struct {
	treeFile
	kind string
}

func parseSelectedTreeEntries(output []byte) ([]selectedTreeEntry, error) {
	var entries []selectedTreeEntry
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return nil, errors.New("malformed git ls-tree record")
		}
		fields := strings.Fields(string(header))
		if len(fields) != 4 {
			return nil, errors.New("malformed git ls-tree header")
		}
		entry := selectedTreeEntry{treeFile: treeFile{path: string(name), mode: fields[0], oid: fields[2]}, kind: fields[1]}
		if fields[1] == "blob" {
			size, err := strconv.ParseInt(fields[3], 10, 64)
			if err != nil || size < 0 {
				return nil, fmt.Errorf("invalid Git blob size for %q", entry.path)
			}
			entry.size = size
		} else if fields[3] != "-" {
			return nil, fmt.Errorf("unexpected Git tree entry size for %q", entry.path)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func readSelectedBlobs(root string, files []treeFile) (map[string][]byte, error) {
	loaded := &snapshot.Snapshot{Files: make(map[string][]byte, len(files)), Modes: make(map[string]string, len(files))}
	if err := loadBlobs(root, files, loaded); err != nil {
		return nil, err
	}
	return loaded.Files, nil
}

func confirmGitIdentity(before GitIdentity, beforeStats gitIdentityStats, run gitOutputFunc) error {
	after, afterStats, err := identifyGit(before.Root, run)
	if err != nil {
		return fmt.Errorf("recheck Git repository identity: %w", err)
	}
	if before != after || !os.SameFile(beforeStats.root, afterStats.root) || !os.SameFile(beforeStats.git, afterStats.git) || !os.SameFile(beforeStats.common, afterStats.common) {
		return errors.New("Git repository identity changed during selected snapshot acquisition")
	}
	return nil
}
