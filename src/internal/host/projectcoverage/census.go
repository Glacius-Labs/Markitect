package projectcoverage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// FromSnapshot builds a pure in-memory candidate universe from the exact
// files supplied by a caller. It performs no filesystem or Git I/O. Unlike
// ObserveWorking, it can account only for paths present in the snapshot; the
// caller should use it for candidate diagnostics, while base/final gates use
// a repository census.
func FromSnapshot(s *snapshot.Snapshot, options Options) (*Universe, error) {
	if s == nil {
		return nil, fmt.Errorf("candidate snapshot is required")
	}
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	paths := make(map[string]*PathState, len(s.Files))
	ignoreBytes := append([]byte(nil), s.Files[IgnorePath]...)
	if _, err := DecodeIgnore(ignoreBytes); err != nil {
		return nil, err
	}
	for name, data := range s.Files {
		if err := validatePath(name); err != nil {
			return nil, err
		}
		mode := s.Modes[name]
		if mode != "100644" && mode != "100755" {
			return nil, fmt.Errorf("candidate path %q has unsupported source mode %q", name, mode)
		}
		if _, exists := paths[name]; exists {
			return nil, fmt.Errorf("candidate repeats path %q", name)
		}
		paths[name] = &PathState{Path: name, Worktree: FileState{Present: true, Mode: mode, Digest: sha256Hex(data)}}
	}
	if err := validatePathSet(paths); err != nil {
		return nil, err
	}
	result := makeUniverse(s.ID, "snapshot", paths, ignoreBytes, options, false)
	result.Snapshot = cloneSnapshot(s)
	return result, nil
}

// ObserveWorking captures the fixed HEAD, index and current filesystem path
// universe. Git ignore rules are not consulted. A root .git directory or file
// is repository administration and is omitted; nested repositories are opaque
// boundaries and are never traversed.
func ObserveWorking(root string, options Options) (*Universe, error) {
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return nil, err
	}
	head, err := source.GitOutput(identity.Root, "rev-parse", "--verify", "HEAD^{commit}")
	revision := ""
	unbornRef := ""
	if err != nil {
		unborn, ref, unbornErr := isUnbornRepository(identity.Root)
		if unbornErr != nil {
			return nil, fmt.Errorf("resolve repository HEAD: %w (check unborn branch: %v)", err, unbornErr)
		}
		if !unborn {
			return nil, fmt.Errorf("resolve repository HEAD: %w", err)
		}
		unbornRef = ref
	} else {
		revision = strings.TrimSpace(string(head))
		if !validCommit(revision, identity.ObjectFormat) {
			return nil, fmt.Errorf("Git returned an invalid full HEAD commit")
		}
	}
	paths := map[string]*PathState{}
	gitlinks := map[string]bool{}
	if revision != "" {
		if err := addHeadPaths(identity.Root, revision, paths, gitlinks); err != nil {
			return nil, err
		}
	}
	if err := addIndexPaths(identity.Root, paths, gitlinks); err != nil {
		return nil, err
	}
	boundaries, worktreeModes, err := walkWorking(identity.Root, gitlinks)
	if err != nil {
		return nil, err
	}
	for name, mode := range worktreeModes {
		state := pathState(paths, name)
		state.Worktree = FileState{Present: true, Mode: mode}
	}
	for name := range boundaries {
		state := pathState(paths, name)
		state.OpaqueBoundary = true
		state.Worktree = FileState{Present: true, Mode: "opaque-repository"}
	}
	ignoreBytes, err := readWorkingIgnore(identity.Root)
	if err != nil {
		return nil, err
	}
	ignore, err := DecodeIgnore(ignoreBytes)
	if err != nil {
		return nil, err
	}
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	contentPaths := selectedContentPaths(paths, ignore, options, true)
	contentPaths = append(contentPaths, IgnorePath)
	contentPaths = uniquePaths(contentPaths)
	observed, observeErr := source.ObserveSelectedWorking(identity.Root, contentPaths)
	if observeErr != nil {
		return nil, fmt.Errorf("read repository census bytes: %w", observeErr)
	}
	if observed.Identity.Digest != identity.Digest {
		return nil, fmt.Errorf("repository identity changed during census")
	}
	if len(observed.MissingPaths) > 0 {
		for _, missing := range observed.MissingPaths {
			if missing != IgnorePath {
				return nil, fmt.Errorf("repository path %q changed during census", missing)
			}
		}
	}
	observedIgnore := observed.Snapshot.Files[IgnorePath]
	if !bytes.Equal(observedIgnore, ignoreBytes) {
		return nil, fmt.Errorf("repository ignore policy changed during census")
	}
	for name, data := range observed.Snapshot.Files {
		if name == IgnorePath {
			state := pathState(paths, name)
			state.Worktree = FileState{Present: true, Mode: observed.Snapshot.Modes[name], Digest: sha256Hex(data)}
			continue
		}
		state := pathState(paths, name)
		state.Worktree = FileState{Present: true, Mode: observed.Snapshot.Modes[name], Digest: sha256Hex(data)}
	}
	// Fixed HEAD blobs are read separately from the live worktree. This allows
	// a single report to bind the commit layer without treating an unrelated
	// staged or unstaged change as part of that commit.
	headContent := selectedHeadContentPaths(paths, ignore, options)
	if revision != "" && len(headContent) > 0 {
		selected, loadErr := source.LoadSelected(identity.Root, revision, headContent)
		if loadErr != nil {
			return nil, fmt.Errorf("read fixed HEAD census bytes: %w", loadErr)
		}
		for name, data := range selected.Snapshot.Files {
			state := pathState(paths, name)
			state.Head.Digest = sha256Hex(data)
		}
	}
	if err := recheckWorkingUniverse(identity.Root, revision, unbornRef, paths, gitlinks); err != nil {
		return nil, err
	}
	result := makeUniverse(revision, identity.Digest, paths, ignoreBytes, options, false)
	result.Snapshot = cloneSnapshot(observed.Snapshot)
	return result, nil
}

// ObserveRevision captures only the named immutable Git tree. Index and live
// worktree state are deliberately absent from the result.
func ObserveRevision(root, revision string, options Options) (*Universe, error) {
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return nil, err
	}
	resolved, err := source.GitOutput(identity.Root, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve revision %q: %w", revision, err)
	}
	full := strings.TrimSpace(string(resolved))
	if !validCommit(full, identity.ObjectFormat) {
		return nil, fmt.Errorf("Git returned an invalid full commit")
	}
	paths := map[string]*PathState{}
	gitlinks := map[string]bool{}
	if err := addHeadPaths(identity.Root, full, paths, gitlinks); err != nil {
		return nil, err
	}
	ignoreBytes := []byte(nil)
	if state := paths[IgnorePath]; state != nil && state.Head.Present {
		selected, loadErr := source.LoadSelected(identity.Root, full, []string{IgnorePath})
		if loadErr != nil {
			return nil, loadErr
		}
		ignoreBytes = append([]byte(nil), selected.Snapshot.Files[IgnorePath]...)
	}
	ignore, err := DecodeIgnore(ignoreBytes)
	if err != nil {
		return nil, err
	}
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	contentPaths := selectedHeadContentPaths(paths, ignore, options)
	// Capture the exact fixed-revision bytes used by both classification and
	// Host model compilation in one selected-snapshot read.
	allContent := append([]string(nil), contentPaths...)
	if state := paths[IgnorePath]; state != nil && state.Head.Present {
		allContent = append(allContent, IgnorePath)
	}
	allContent = uniquePaths(allContent)
	selected, err := source.LoadSelected(identity.Root, full, allContent)
	if err != nil {
		return nil, fmt.Errorf("capture fixed revision census snapshot: %w", err)
	}
	for name, data := range selected.Snapshot.Files {
		if name == IgnorePath {
			continue
		}
		state := pathState(paths, name)
		state.Head.Digest = sha256Hex(data)
	}
	result := makeUniverse(full, identity.Digest, paths, ignoreBytes, options, true)
	result.Snapshot = cloneSnapshot(selected.Snapshot)
	return result, nil
}

func addHeadPaths(root, revision string, paths map[string]*PathState, gitlinks map[string]bool) error {
	out, err := source.GitOutput(root, "ls-tree", "-r", "-z", "--full-tree", "--long", revision)
	if err != nil {
		return fmt.Errorf("list fixed Git tree: %w", err)
	}
	for _, record := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if record == "" {
			continue
		}
		header, name, ok := strings.Cut(record, "\t")
		if !ok {
			return errors.New("malformed Git tree record")
		}
		fields := strings.Fields(header)
		if len(fields) < 3 {
			return fmt.Errorf("malformed Git tree header for %q", name)
		}
		mode, kind := fields[0], fields[1]
		if err := validatePath(name); err != nil {
			return err
		}
		if mode == "160000" || kind == "commit" {
			gitlinks[name] = true
			state := pathState(paths, strings.TrimSuffix(name, "/")+"/")
			state.Head = FileState{Present: true, Mode: "opaque-repository", Digest: fields[2]}
			state.OpaqueBoundary = true
			continue
		}
		if mode == "120000" {
			return fmt.Errorf("Git symlink is not a supported repository file: %q", name)
		}
		if kind != "blob" || mode != "100644" && mode != "100755" {
			return fmt.Errorf("unsupported Git tree entry %q (mode %s, kind %s)", name, mode, kind)
		}
		state := pathState(paths, name)
		state.Head = FileState{Present: true, Mode: mode}
	}
	return validatePathSet(paths)
}

func addIndexPaths(root string, paths map[string]*PathState, gitlinks map[string]bool) error {
	out, err := source.GitOutput(root, "ls-files", "--stage", "-z")
	if err != nil {
		return fmt.Errorf("list Git index: %w", err)
	}
	for _, record := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if record == "" {
			continue
		}
		header, name, ok := strings.Cut(record, "\t")
		if !ok {
			return errors.New("malformed Git index record")
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			return fmt.Errorf("malformed Git index header for %q", name)
		}
		if fields[2] != "0" {
			return fmt.Errorf("Git index has unresolved conflict stage %s for %q", fields[2], name)
		}
		if err := validatePath(name); err != nil {
			return err
		}
		mode := fields[0]
		if mode == "160000" {
			gitlinks[name] = true
			state := pathState(paths, strings.TrimSuffix(name, "/")+"/")
			state.Index = FileState{Present: true, Mode: "opaque-repository", Digest: fields[1]}
			state.OpaqueBoundary = true
			continue
		}
		if mode == "120000" {
			return fmt.Errorf("Git index symlink is not a supported repository file: %q", name)
		}
		if mode != "100644" && mode != "100755" {
			return fmt.Errorf("unsupported Git index mode %s for %q", mode, name)
		}
		state := pathState(paths, name)
		state.Index = FileState{Present: true, Mode: mode, Digest: fields[1]}
	}
	return validatePathSet(paths)
}

func walkWorking(root string, gitlinks map[string]bool) (map[string]bool, map[string]string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	rootInfo, err := os.Lstat(rootAbs)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("repository root must be a real directory")
	}
	boundaries := map[string]bool{}
	files := map[string]string{}
	visited := 0
	var walk func(string, string) error
	walk = func(absDir, relDir string) error {
		entries, readErr := os.ReadDir(absDir)
		if readErr != nil {
			return fmt.Errorf("read repository directory %q: %w", relDir, readErr)
		}
		for _, entry := range entries {
			name := entry.Name()
			rel := name
			if relDir != "" {
				rel = relDir + "/" + name
			}
			if relDir == "" && name == ".git" {
				info, statErr := os.Lstat(filepath.Join(absDir, name))
				if statErr != nil || info.Mode()&fs.ModeSymlink != 0 {
					return fmt.Errorf("refusing repository .git symlink or reparse point")
				}
				continue
			}
			if err := validatePath(rel); err != nil {
				return err
			}
			info, statErr := os.Lstat(filepath.Join(absDir, name))
			if statErr != nil {
				return fmt.Errorf("stat repository path %q: %w", rel, statErr)
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				return fmt.Errorf("refusing repository symlink or reparse point %q", rel)
			}
			if info.IsDir() {
				if gitlinks[rel] || hasGitMetadata(filepath.Join(absDir, name)) {
					boundaries[rel+"/"] = true
					continue
				}
				if err := walk(filepath.Join(absDir, name), rel); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported non-regular repository path %q", rel)
			}
			visited++
			if visited > source.DefaultMaxFiles {
				return fmt.Errorf("repository census exceeds %d files", source.DefaultMaxFiles)
			}
			mode := "100644"
			if info.Mode().Perm()&0111 != 0 {
				mode = "100755"
			}
			files[rel] = mode
		}
		return nil
	}
	if err := walk(rootAbs, ""); err != nil {
		return nil, nil, err
	}
	if err := validatePortableNames(files); err != nil {
		return nil, nil, err
	}
	return boundaries, files, nil
}

func hasGitMetadata(directory string) bool {
	info, err := os.Lstat(filepath.Join(directory, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

func readWorkingIgnore(root string) ([]byte, error) {
	observed, err := source.ObserveSelectedWorking(root, []string{IgnorePath})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", IgnorePath, err)
	}
	if len(observed.MissingPaths) != 0 {
		return nil, nil
	}
	return append([]byte(nil), observed.Snapshot.Files[IgnorePath]...), nil
}

func isUnbornRepository(root string) (bool, string, error) {
	ref, err := source.GitOutput(root, "symbolic-ref", "--quiet", "HEAD")
	if err != nil {
		return false, "", nil
	}
	name := strings.TrimSpace(string(ref))
	if !strings.HasPrefix(name, "refs/heads/") || strings.ContainsAny(name, "\r\n") {
		return false, "", nil
	}
	_, err = source.GitOutput(root, "show-ref", "--verify", "--quiet", name)
	if err == nil {
		return false, "", nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, name, nil
	}
	return false, "", fmt.Errorf("inspect symbolic HEAD ref %q: %w", name, err)
}

func recheckWorkingUniverse(root, revision, unbornRef string, original map[string]*PathState, gitlinks map[string]bool) error {
	head, err := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		unborn, ref, unbornErr := isUnbornRepository(root)
		if unbornErr != nil || revision != "" || !unborn || ref != unbornRef {
			return fmt.Errorf("repository HEAD changed during census: %w", err)
		}
	} else if strings.TrimSpace(string(head)) != revision {
		return fmt.Errorf("repository HEAD changed during census")
	}
	paths := map[string]*PathState{}
	links := map[string]bool{}
	if revision != "" {
		if err := addHeadPaths(root, revision, paths, links); err != nil {
			return err
		}
	}
	if err := addIndexPaths(root, paths, links); err != nil {
		return err
	}
	boundaries, modes, err := walkWorking(root, links)
	if err != nil {
		return err
	}
	for name, mode := range modes {
		state := pathState(paths, name)
		state.Worktree = FileState{Present: true, Mode: mode}
	}
	for name := range boundaries {
		state := pathState(paths, name)
		state.OpaqueBoundary = true
		state.Worktree = FileState{Present: true, Mode: "opaque-repository"}
	}
	if !sameCensusMetadata(original, paths) || !sameLinks(gitlinks, links) {
		return fmt.Errorf("repository membership, mode, HEAD, or index changed during census")
	}
	return nil
}

func sameCensusMetadata(left, right map[string]*PathState) bool {
	if len(left) != len(right) {
		return false
	}
	for name, a := range left {
		b := right[name]
		if b == nil || a.Head.Present != b.Head.Present || a.Head.Mode != b.Head.Mode ||
			a.Index.Present != b.Index.Present || a.Index.Mode != b.Index.Mode || a.Index.Digest != b.Index.Digest ||
			a.Worktree.Present != b.Worktree.Present || a.Worktree.Mode != b.Worktree.Mode || a.OpaqueBoundary != b.OpaqueBoundary {
			return false
		}
	}
	return true
}

func sameLinks(left, right map[string]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for name := range left {
		if !right[name] {
			return false
		}
	}
	return true
}

func cloneSnapshot(input *snapshot.Snapshot) *snapshot.Snapshot {
	if input == nil {
		return nil
	}
	result := &snapshot.Snapshot{ID: input.ID, Provisional: input.Provisional, Files: make(map[string][]byte, len(input.Files)), Modes: make(map[string]string, len(input.Modes))}
	for name, data := range input.Files {
		result.Files[name] = append([]byte(nil), data...)
	}
	for name, mode := range input.Modes {
		result.Modes[name] = mode
	}
	return result
}

func uniquePaths(values []string) []string {
	set := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !set[value] {
			set[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func pathState(paths map[string]*PathState, name string) *PathState {
	state := paths[name]
	if state == nil {
		state = &PathState{Path: name}
		paths[name] = state
	}
	return state
}

func makeUniverse(revision, identity string, paths map[string]*PathState, ignoreBytes []byte, options Options, fixed bool) *Universe {
	result := &Universe{Revision: revision, IdentityDigest: identity, FixedRevision: fixed, IgnoreBytes: append([]byte(nil), ignoreBytes...), IgnoreBytesDigest: sha256Hex(ignoreBytes)}
	for _, state := range paths {
		copy := *state
		result.Paths = append(result.Paths, copy)
	}
	sort.Slice(result.Paths, func(i, j int) bool { return result.Paths[i].Path < result.Paths[j].Path })
	result.Digest = universeDigest(result, options)
	return result
}

func selectedContentPaths(paths map[string]*PathState, ignore IgnoreFile, options Options, working bool) []string {
	var result []string
	for name, state := range paths {
		if state.OpaqueBoundary || isIgnored(name, ignore) || isTransitional(name, options) || isOperational(name, options) || name == IgnorePath || !state.Worktree.Present {
			continue
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func selectedHeadContentPaths(paths map[string]*PathState, ignore IgnoreFile, options Options) []string {
	var result []string
	for name, state := range paths {
		if state.OpaqueBoundary || isIgnored(name, ignore) || isTransitional(name, options) || isOperational(name, options) || name == IgnorePath || !state.Head.Present {
			continue
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func validatePath(name string) error {
	cleanName := strings.TrimSuffix(name, "/")
	if cleanName == "" || !utf8.ValidString(cleanName) || strings.ContainsAny(cleanName, "\\:\x00") || strings.HasPrefix(cleanName, "/") || path.Clean(cleanName) != cleanName {
		return fmt.Errorf("unsafe or non-normalized repository path %q", name)
	}
	for _, part := range strings.Split(cleanName, "/") {
		if !safePathComponent(part) {
			return fmt.Errorf("repository path %q contains a reserved or traversal component", name)
		}
	}
	return nil
}

func safePathComponent(part string) bool {
	if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") || strings.TrimRight(part, " .") != part || strings.ContainsAny(part, "<>:\\|?*\x00") {
		return false
	}
	for _, r := range part {
		if unicode.IsControl(r) {
			return false
		}
	}
	base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}

func validatePathSet(paths map[string]*PathState) error {
	seen := map[string]string{}
	terminals := map[string]string{}
	for name := range paths {
		clean := strings.TrimSuffix(name, "/")
		terminalKey := strings.ToLower(clean)
		if previous, ok := terminals[terminalKey]; ok && previous != name {
			return fmt.Errorf("repository contains colliding file and repository-boundary paths %q and %q", previous, name)
		}
		terminals[terminalKey] = name
		parts := strings.Split(clean, "/")
		for i := range parts {
			actual := strings.Join(parts[:i+1], "/")
			key := strings.ToLower(actual)
			if previous, ok := seen[key]; ok && previous != actual {
				return fmt.Errorf("repository contains case-colliding paths %q and %q", previous, actual)
			}
			seen[key] = actual
		}
	}
	return nil
}

func validateUniversePaths(paths []PathState) error {
	set := make(map[string]*PathState, len(paths))
	for _, state := range paths {
		if strings.HasSuffix(state.Path, "/") && !state.OpaqueBoundary {
			return fmt.Errorf("repository file path %q unexpectedly ends with a slash", state.Path)
		}
		if err := validatePath(state.Path); err != nil {
			return err
		}
		if _, exists := set[state.Path]; exists {
			return fmt.Errorf("candidate contains duplicate path %q", state.Path)
		}
		set[state.Path] = &PathState{Path: state.Path}
	}
	return validatePathSet(set)
}

func validatePortableNames(files map[string]string) error {
	paths := make(map[string]*PathState, len(files))
	for name := range files {
		paths[name] = &PathState{Path: name}
	}
	return validatePathSet(paths)
}

func validCommit(commit, objectFormat string) bool {
	want := 40
	if objectFormat == "sha256" {
		want = 64
	}
	if len(commit) != want || strings.ToLower(commit) != commit {
		return false
	}
	for _, r := range commit {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
