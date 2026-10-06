package source

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
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
	identity, _, err := identifyGit(root, selectiveGitOutput)
	return identity, err
}

// LoadSelected reads only the exact regular-file paths listed from a full,
// lowercase commit ID. All requested paths are validated before Git is
// queried, and all metadata is checked before any blob content is read.
func LoadSelected(root, fullCommit string, paths []string) (*SelectedSnapshot, error) {
	return loadSelected(root, fullCommit, paths, selectiveGitOutput, readSelectedBlobs)
}

type gitOutputFunc func(root string, args ...string) ([]byte, error)
type selectedBlobReader func(root string, files []treeFile) (map[string][]byte, error)

const (
	maxSelectedTreePathsPerCommand = 128
	// Leave headroom below CreateProcessW's 32767 UTF-16-code-unit limit.
	maxSelectedTreeCommandUnits = 30000
)

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

	top, gitDirOut, commonDirOut, objectFormatOut, err := identifyGitMetadata(canonicalRoot, run)
	if err != nil {
		return zero, stats, err
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
	objectFormat := strings.TrimSpace(string(objectFormatOut))
	if objectFormat != "sha1" && objectFormat != "sha256" {
		return zero, stats, fmt.Errorf("unsupported Git object format %q", objectFormat)
	}
	identity := GitIdentity{Root: canonicalRoot, GitDir: gitDir, CommonDir: commonDir, ObjectFormat: objectFormat}
	identity.Digest = gitIdentityDigest(identity)
	return identity, gitIdentityStats{root: info, git: gitInfo, common: commonInfo}, nil
}

// identifyGitMetadata obtains the same four identity fields with one Git
// process in the usual case. rev-parse emits newline-delimited values rather
// than a NUL-delimited record; if a repository path itself contains a newline
// (or the output is otherwise ambiguous), fall back to the original one-field
// queries so those paths retain their established handling.
func identifyGitMetadata(root string, run gitOutputFunc) (top, gitDir, commonDir, objectFormat []byte, err error) {
	combined, combinedErr := run(root, "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-object-format")
	if combinedErr == nil {
		if fields, ok := parseGitIdentityMetadata(combined); ok {
			return fields[0], fields[1], fields[2], fields[3], nil
		}
	}

	top, err = run(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify Git worktree root: %w", err)
	}
	gitDir, err = run(root, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify Git directory: %w", err)
	}
	commonDir, err = run(root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify common Git directory: %w", err)
	}
	objectFormat, err = run(root, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify Git object format: %w", err)
	}
	return top, gitDir, commonDir, objectFormat, nil
}

func parseGitIdentityMetadata(output []byte) ([4][]byte, bool) {
	var fields [4][]byte
	parts := bytes.Split(output, []byte{'\n'})
	if len(parts) != len(fields)+1 || len(parts[len(parts)-1]) != 0 {
		return fields, false
	}
	for i := range fields {
		if len(parts[i]) == 0 || bytes.ContainsAny(parts[i], "\r") {
			return fields, false
		}
		fields[i] = parts[i]
	}
	return fields, true
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

	pathBatches, err := selectedTreePathBatches(identity.Root, fullCommit, paths)
	if err != nil {
		return nil, err
	}
	entriesByPath := make(map[string]selectedTreeEntry, len(paths))
	for _, batch := range pathBatches {
		args := selectedTreeArgs(fullCommit, batch)
		out, err := run(identity.Root, args...)
		if err != nil {
			return nil, fmt.Errorf("inspect selected paths %q: %w", batch, err)
		}
		entries, err := parseSelectedTreeEntries(out)
		if err != nil {
			return nil, fmt.Errorf("inspect selected paths %q: %w", batch, err)
		}
		for _, entry := range entries {
			batchIndex := sort.SearchStrings(batch, entry.path)
			if batchIndex == len(batch) || batch[batchIndex] != entry.path {
				return nil, fmt.Errorf("Git returned unselected tree entry %q", entry.path)
			}
			if _, duplicate := entriesByPath[entry.path]; duplicate {
				return nil, fmt.Errorf("Git returned duplicate tree entry %q", entry.path)
			}
			entriesByPath[entry.path] = entry
		}
	}

	files := make([]treeFile, 0, len(paths))
	var total int64
	for _, path := range paths {
		file, ok := entriesByPath[path]
		if !ok {
			return nil, fmt.Errorf("selected path %q does not identify exactly one Git tree entry", path)
		}
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

func selectedTreeArgs(fullCommit string, paths []string) []string {
	args := []string{"--literal-pathspecs", "ls-tree", "-z", "--full-tree", "--long", fullCommit, "--"}
	return append(args, paths...)
}

// selectedTreePathBatches keeps Git metadata queries scoped to validated
// literal pathspecs while bounding both argument count and the Windows command
// line. The command length calculation includes gitCommandWithEnv's fixed
// arguments and leaves headroom below CreateProcessW's hard limit.
func selectedTreePathBatches(root, fullCommit string, paths []string) ([][]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve source root for selected metadata: %w", err)
	}
	baseArgs := []string{"git", "--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(abs), "-C", abs}
	baseArgs = append(baseArgs, "--literal-pathspecs", "ls-tree", "-z", "--full-tree", "--long", fullCommit, "--")
	baseUnits := windowsCommandLineUnits(baseArgs)
	if baseUnits > maxSelectedTreeCommandUnits {
		return nil, errors.New("Git command base exceeds selected metadata command-line limit")
	}

	var batches [][]string
	var batch []string
	units := baseUnits
	for _, path := range paths {
		pathUnits := windowsCommandLineArgUnits(path) + 1 // separating space
		if baseUnits+pathUnits > maxSelectedTreeCommandUnits {
			return nil, fmt.Errorf("selected path %q exceeds Git command-line limit", path)
		}
		if len(batch) > 0 && (len(batch) >= maxSelectedTreePathsPerCommand || units+pathUnits > maxSelectedTreeCommandUnits) {
			batches = append(batches, batch)
			batch = nil
			units = baseUnits
		}
		batch = append(batch, path)
		units += pathUnits
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches, nil
}

func windowsCommandLineUnits(args []string) int {
	units := 1 // terminating NUL
	for i, arg := range args {
		if i > 0 {
			units++ // separating space
		}
		units += windowsCommandLineArgUnits(arg)
	}
	return units
}

func windowsCommandLineArgUnits(arg string) int {
	quoted := arg == "" || strings.ContainsAny(arg, " \t\"")
	units := 0
	if quoted {
		units = 2 // surrounding quotes
	}
	backslashes := 0
	for _, r := range arg {
		if r == '\\' {
			backslashes++
			continue
		}
		if r == '"' {
			// Each backslash before a quote is doubled, then the quote is escaped.
			units += backslashes*2 + 1 + utf16.RuneLen(r)
		} else {
			units += backslashes + utf16.RuneLen(r)
		}
		backslashes = 0
	}
	if quoted {
		units += backslashes * 2
	} else {
		units += backslashes
	}
	return units
}

func validateSelectedPaths(paths []string) error {
	seen := make(map[string]struct{}, len(paths))
	for _, selected := range paths {
		if err := validateRepoPath(selected); err != nil {
			return err
		}
		for _, component := range strings.Split(selected, "/") {
			if strings.EqualFold(component, ".git") {
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
			if fields[3] == "-" {
				return nil, fmt.Errorf("Git blob object is unavailable locally for %q", entry.path)
			}
			size, err := strconv.ParseInt(fields[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Git blob size is unavailable locally for %q", entry.path)
			}
			if size < 0 {
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
	if err := loadBlobsWithCommand(root, files, loaded, selectiveGitCommand); err != nil {
		return nil, err
	}
	return loaded.Files, nil
}

func selectiveGitEnvironment() []string {
	env := CleanGitEnv()
	return append(env, "GIT_NO_LAZY_FETCH=1", "GIT_ALLOW_PROTOCOL=", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
}

func selectiveGitCommand(root string, args ...string) *exec.Cmd {
	return gitCommandWithEnv(root, selectiveGitEnvironment(), args...)
}

func selectiveGitOutput(root string, args ...string) ([]byte, error) {
	cmd := selectiveGitCommand(root, args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, err
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
