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
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	modelcoverage "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectcoverage"
)

// ObserveWorking captures the fixed HEAD, index and current filesystem path
// universe. Git ignore rules are not consulted. A root .git directory or file
// is repository administration and is omitted; nested repositories are opaque
// boundaries and are never traversed.
func ObserveWorking(root string, options modelcoverage.Options) (*modelcoverage.Universe, error) {
	return observeWorking(root, options, nil)
}

// ObserveWorkingIn is ObserveWorking for a caller that reads the repository
// through acquisition. The census adds no identity checks of its own; the
// caller confirms the acquisition after its last read.
func ObserveWorkingIn(acquisition *source.Acquisition, options modelcoverage.Options) (*modelcoverage.Universe, error) {
	return observeWorkingIn(acquisition, options, nil)
}

// observeWorking keeps an internal mutation seam immediately before the final
// consistency check so tests can exercise concurrent path changes without
// timing-dependent filesystem races.
func observeWorking(root string, options modelcoverage.Options, beforeRecheck func() error) (*modelcoverage.Universe, error) {
	return acquireCensus(root, func(acquisition *source.Acquisition) (*modelcoverage.Universe, error) {
		return observeWorkingIn(acquisition, options, beforeRecheck)
	})
}

// acquireCensus runs one census as its own repository acquisition.
func acquireCensus(root string, census func(*source.Acquisition) (*modelcoverage.Universe, error)) (*modelcoverage.Universe, error) {
	acquisition, err := source.BeginAcquisition(root)
	if err != nil {
		return nil, err
	}
	universe, err := census(acquisition)
	if err != nil {
		return nil, err
	}
	if err := acquisition.Confirm(); err != nil {
		return nil, err
	}
	return universe, nil
}

func observeWorkingIn(acquisition *source.Acquisition, options modelcoverage.Options, beforeRecheck func() error) (*modelcoverage.Universe, error) {
	identity := acquisition.Identity()
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
	paths := map[string]*modelcoverage.PathState{}
	gitlinks := map[string]bool{}
	if revision != "" {
		if err := addHeadPaths(identity.Root, revision, paths, gitlinks); err != nil {
			return nil, err
		}
	}
	if err := addIndexPaths(identity.Root, paths, gitlinks); err != nil {
		return nil, err
	}
	fileModeEnabled, err := acquisition.FileModeEnabled()
	if err != nil {
		return nil, fmt.Errorf("inspect Git worktree mode policy: %w", err)
	}
	boundaries, worktreeModes, err := walkWorking(identity.Root, gitlinks)
	if err != nil {
		return nil, err
	}
	normalizeWorktreeModes(paths, worktreeModes, fileModeEnabled)
	for name, mode := range worktreeModes {
		state := pathState(paths, name)
		state.Worktree = modelcoverage.FileState{Present: true, Mode: mode}
	}
	for name := range boundaries {
		state := pathState(paths, name)
		state.OpaqueBoundary = true
		state.Worktree = modelcoverage.FileState{Present: true, Mode: "opaque-repository"}
	}
	ignoreBytes, err := readWorkingIgnore(acquisition)
	if err != nil {
		return nil, err
	}
	ignore, err := modelcoverage.DecodeIgnore(ignoreBytes)
	if err != nil {
		return nil, err
	}
	if err := modelcoverage.ValidateOptions(options); err != nil {
		return nil, err
	}
	contentPaths := modelcoverage.SelectWorkingContentPaths(pathStates(paths), ignore, options)
	contentPaths = append(contentPaths, modelcoverage.IgnorePath)
	contentPaths = uniquePaths(contentPaths)
	observed, observeErr := acquisition.ObserveSelectedWorking(contentPaths)
	if observeErr != nil {
		return nil, fmt.Errorf("read repository census bytes: %w", observeErr)
	}
	if len(observed.MissingPaths) > 0 {
		for _, missing := range observed.MissingPaths {
			if missing != modelcoverage.IgnorePath {
				return nil, fmt.Errorf("repository path %q changed during census", missing)
			}
		}
	}
	observedIgnore := observed.Snapshot.Files[modelcoverage.IgnorePath]
	if !bytes.Equal(observedIgnore, ignoreBytes) {
		return nil, fmt.Errorf("repository ignore policy changed during census")
	}
	for name, data := range observed.Snapshot.Files {
		if name == modelcoverage.IgnorePath {
			state := pathState(paths, name)
			state.Worktree = modelcoverage.FileState{Present: true, Mode: observed.Snapshot.Modes[name], Digest: sha256Hex(data)}
			continue
		}
		state := pathState(paths, name)
		state.Worktree = modelcoverage.FileState{Present: true, Mode: observed.Snapshot.Modes[name], Digest: sha256Hex(data)}
	}
	// Fixed HEAD blobs are read separately from the live worktree. This allows
	// a single report to bind the commit layer without treating an unrelated
	// staged or unstaged change as part of that commit.
	headContent := modelcoverage.SelectRevisionContentPaths(pathStates(paths), ignore, options)
	if revision != "" && len(headContent) > 0 {
		selected, loadErr := acquisition.LoadSelected(revision, headContent)
		if loadErr != nil {
			return nil, fmt.Errorf("read fixed HEAD census bytes: %w", loadErr)
		}
		for name, data := range selected.Snapshot.Files {
			state := pathState(paths, name)
			state.Head.Digest = sha256Hex(data)
		}
	}
	if beforeRecheck != nil {
		if err := beforeRecheck(); err != nil {
			return nil, fmt.Errorf("run census recheck mutation: %w", err)
		}
	}
	if err := recheckWorkingUniverse(identity.Root, revision, unbornRef, paths, gitlinks, options); err != nil {
		return nil, err
	}
	result := modelcoverage.NewUniverse(revision, identity.Digest, pathStates(paths), ignoreBytes, options, false)
	result.Snapshot = cloneSnapshot(observed.Snapshot)
	return result, nil
}

// ObserveRevision captures only the named immutable Git tree. Index and live
// worktree state are deliberately absent from the result.
func ObserveRevision(root, revision string, options modelcoverage.Options) (*modelcoverage.Universe, error) {
	return acquireCensus(root, func(acquisition *source.Acquisition) (*modelcoverage.Universe, error) {
		return ObserveRevisionIn(acquisition, revision, options)
	})
}

// ObserveRevisionIn is ObserveRevision for a caller that reads the repository
// through acquisition, as ObserveWorkingIn is for ObserveWorking.
func ObserveRevisionIn(acquisition *source.Acquisition, revision string, options modelcoverage.Options) (*modelcoverage.Universe, error) {
	identity := acquisition.Identity()
	resolved, err := source.GitOutput(identity.Root, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve revision %q: %w", revision, err)
	}
	full := strings.TrimSpace(string(resolved))
	if !validCommit(full, identity.ObjectFormat) {
		return nil, fmt.Errorf("Git returned an invalid full commit")
	}
	paths := map[string]*modelcoverage.PathState{}
	gitlinks := map[string]bool{}
	if err := addHeadPaths(identity.Root, full, paths, gitlinks); err != nil {
		return nil, err
	}
	ignoreBytes := []byte(nil)
	if state := paths[modelcoverage.IgnorePath]; state != nil && state.Head.Present {
		selected, loadErr := acquisition.LoadSelected(full, []string{modelcoverage.IgnorePath})
		if loadErr != nil {
			return nil, loadErr
		}
		ignoreBytes = append([]byte(nil), selected.Snapshot.Files[modelcoverage.IgnorePath]...)
	}
	ignore, err := modelcoverage.DecodeIgnore(ignoreBytes)
	if err != nil {
		return nil, err
	}
	if err := modelcoverage.ValidateOptions(options); err != nil {
		return nil, err
	}
	contentPaths := modelcoverage.SelectRevisionContentPaths(pathStates(paths), ignore, options)
	// Capture the exact fixed-revision bytes used by both classification and
	// Host model compilation in one selected-snapshot read.
	allContent := append([]string(nil), contentPaths...)
	if state := paths[modelcoverage.IgnorePath]; state != nil && state.Head.Present {
		allContent = append(allContent, modelcoverage.IgnorePath)
	}
	allContent = uniquePaths(allContent)
	selected, err := acquisition.LoadSelected(full, allContent)
	if err != nil {
		return nil, fmt.Errorf("capture fixed revision census snapshot: %w", err)
	}
	for name, data := range selected.Snapshot.Files {
		if name == modelcoverage.IgnorePath {
			continue
		}
		state := pathState(paths, name)
		state.Head.Digest = sha256Hex(data)
	}
	result := modelcoverage.NewUniverse(full, identity.Digest, pathStates(paths), ignoreBytes, options, true)
	result.Snapshot = cloneSnapshot(selected.Snapshot)
	return result, nil
}

func addHeadPaths(root, revision string, paths map[string]*modelcoverage.PathState, gitlinks map[string]bool) error {
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
		if err := modelcoverage.ValidateRepositoryPath(name); err != nil {
			return err
		}
		if mode == "160000" || kind == "commit" {
			gitlinks[name] = true
			state := pathState(paths, strings.TrimSuffix(name, "/")+"/")
			state.Head = modelcoverage.FileState{Present: true, Mode: "opaque-repository", Digest: fields[2]}
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
		state.Head = modelcoverage.FileState{Present: true, Mode: mode}
	}
	return validateAcquiredPathSet(paths)
}

func addIndexPaths(root string, paths map[string]*modelcoverage.PathState, gitlinks map[string]bool) error {
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
		if err := modelcoverage.ValidateRepositoryPath(name); err != nil {
			return err
		}
		mode := fields[0]
		if mode == "160000" {
			gitlinks[name] = true
			state := pathState(paths, strings.TrimSuffix(name, "/")+"/")
			state.Index = modelcoverage.FileState{Present: true, Mode: "opaque-repository", Digest: fields[1]}
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
		state.Index = modelcoverage.FileState{Present: true, Mode: mode, Digest: fields[1]}
	}
	return validateAcquiredPathSet(paths)
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
			if err := modelcoverage.ValidateRepositoryPath(rel); err != nil {
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
	if err := validateAcquiredPathNames(files); err != nil {
		return nil, nil, err
	}
	return boundaries, files, nil
}

func hasGitMetadata(directory string) bool {
	info, err := os.Lstat(filepath.Join(directory, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

func readWorkingIgnore(acquisition *source.Acquisition) ([]byte, error) {
	observed, err := acquisition.ObserveSelectedWorking([]string{modelcoverage.IgnorePath})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", modelcoverage.IgnorePath, err)
	}
	if len(observed.MissingPaths) != 0 {
		return nil, nil
	}
	return append([]byte(nil), observed.Snapshot.Files[modelcoverage.IgnorePath]...), nil
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

func recheckWorkingUniverse(root, revision, unbornRef string, original map[string]*modelcoverage.PathState, gitlinks map[string]bool, options modelcoverage.Options) error {
	head, err := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		unborn, ref, unbornErr := isUnbornRepository(root)
		if unbornErr != nil || revision != "" || !unborn || ref != unbornRef {
			return fmt.Errorf("repository HEAD changed during census: %w", err)
		}
	} else if strings.TrimSpace(string(head)) != revision {
		return fmt.Errorf("repository HEAD changed during census")
	}
	paths := map[string]*modelcoverage.PathState{}
	links := map[string]bool{}
	if revision != "" {
		if err := addHeadPaths(root, revision, paths, links); err != nil {
			return err
		}
	}
	if err := addIndexPaths(root, paths, links); err != nil {
		return err
	}
	fileModeEnabled, err := source.GitFileModeEnabled(root)
	if err != nil {
		return fmt.Errorf("recheck Git worktree mode policy: %w", err)
	}
	boundaries, modes, err := walkWorking(root, links)
	if err != nil {
		return err
	}
	normalizeWorktreeModes(paths, modes, fileModeEnabled)
	for name, mode := range modes {
		state := pathState(paths, name)
		state.Worktree = modelcoverage.FileState{Present: true, Mode: mode}
	}
	for name := range boundaries {
		state := pathState(paths, name)
		state.OpaqueBoundary = true
		state.Worktree = modelcoverage.FileState{Present: true, Mode: "opaque-repository"}
	}
	if !sameCensusMetadata(original, paths, options) || !sameLinks(gitlinks, links) {
		return fmt.Errorf("repository membership, mode, HEAD, or index changed during census")
	}
	return nil
}

func normalizeWorktreeModes(paths map[string]*modelcoverage.PathState, worktreeModes map[string]string, fileModeEnabled bool) {
	if fileModeEnabled {
		return
	}
	for name := range worktreeModes {
		if state := paths[name]; state != nil && state.Index.Present {
			worktreeModes[name] = state.Index.Mode
		}
	}
}

func sameCensusMetadata(left, right map[string]*modelcoverage.PathState, options modelcoverage.Options) bool {
	for name, a := range left {
		if modelcoverage.IsOperationalPath(name, options) {
			continue
		}
		b := right[name]
		if b == nil || a.Head.Present != b.Head.Present || a.Head.Mode != b.Head.Mode ||
			a.Index.Present != b.Index.Present || a.Index.Mode != b.Index.Mode || a.Index.Digest != b.Index.Digest ||
			a.Worktree.Present != b.Worktree.Present || a.Worktree.Mode != b.Worktree.Mode || a.OpaqueBoundary != b.OpaqueBoundary {
			return false
		}
	}
	for name := range right {
		if modelcoverage.IsOperationalPath(name, options) {
			continue
		}
		if _, exists := left[name]; !exists {
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

func pathState(paths map[string]*modelcoverage.PathState, name string) *modelcoverage.PathState {
	state := paths[name]
	if state == nil {
		state = &modelcoverage.PathState{Path: name}
		paths[name] = state
	}
	return state
}

func pathStates(paths map[string]*modelcoverage.PathState) []modelcoverage.PathState {
	result := make([]modelcoverage.PathState, 0, len(paths))
	for _, state := range paths {
		result = append(result, *state)
	}
	return result
}

func validateAcquiredPathSet(paths map[string]*modelcoverage.PathState) error {
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	return modelcoverage.ValidateRepositoryPaths(names)
}

func validateAcquiredPathNames(files map[string]string) error {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	return modelcoverage.ValidateRepositoryPaths(names)
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
