// Package guardedwrite applies selected working-tree changes only while the
// repository, branch, HEAD and captured file bytes are unchanged. It also owns
// the anchored write root those checks build on; the remaining legacy writers
// in the host root package use that root until they are removed.
package guardedwrite

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// File is the exact working-tree state captured for one selected
// path. Mode contains permission bits only. An absent path is represented by
// Exists=false, an empty Bytes slice, and Mode=0.
type File struct {
	Exists bool
	Bytes  []byte
	Mode   fs.FileMode
}

// Capture binds selected working files to one repository identity,
// branch and HEAD. Its exported fields are inspectable by Host callers; Apply
// rejects changes to the capture after CaptureFiles returns.
type Capture struct {
	Root     string
	Identity source.GitIdentity
	// Head is a full commit ID, or `unborn:<symbolic-ref>` for a new named
	// branch with no commit yet. A later first commit invalidates that capture.
	Head   string
	Branch string
	Files  map[string]File

	rootIdentity os.FileInfo
	seal         string
}

// Change is one selected-path mutation. Delete removes an
// existing regular file; otherwise Bytes and Mode replace or create it.
type Change struct {
	Path   string
	Bytes  []byte
	Mode   fs.FileMode
	Delete bool
}

// Result reports mutations that actually completed. It can be
// non-empty alongside an error because this API is deliberately not a
// multi-file transaction.
type Result struct {
	CompletedPaths []string
}

// CaptureFiles records exact bytes, modes, and missing selected paths
// from a Git worktree on a writable named branch. Selected paths are an exact
// allowlist for the later Apply call.
func CaptureFiles(root string, selectedPaths []string) (*Capture, error) {
	paths, err := normalizeGuardedPaths(selectedPaths)
	if err != nil {
		return nil, err
	}
	writer, err := OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	identity, branch, head, err := guardedGitState(root)
	if err != nil {
		return nil, err
	}
	capture := &Capture{
		Root:         identity.Root,
		Identity:     identity,
		Head:         head,
		Branch:       branch,
		Files:        make(map[string]File, len(paths)),
		rootIdentity: writer.identity,
	}
	for _, name := range paths {
		if _, err := SafeDestination(root, name); err != nil {
			return nil, err
		}
		file, err := readGuardedWriteFile(writer, name)
		if err != nil {
			return nil, fmt.Errorf("capture selected path %s: %w", name, err)
		}
		capture.Files[name] = file
	}
	if err := writer.CheckIdentity(); err != nil {
		return nil, err
	}
	if err := ensureGuardedGitState(root, identity, branch, head); err != nil {
		return nil, err
	}
	capture.seal, err = guardedCaptureSeal(capture)
	if err != nil {
		return nil, err
	}
	return capture, nil
}

// Apply applies only changes to paths selected by capture. It
// acquires the same Host writer lock as other render/init/format operations,
// then rechecks repository identity, branch, HEAD and captured bytes before
// the first mutation and before each changed path.
func Apply(root string, capture *Capture, changes []Change) (Result, error) {
	return applyGuardedWrite(root, capture, changes, nil, nil)
}

// ApplyChecked is Apply with an additional caller-owned
// read-only precondition. The callback runs under the shared writer lock after
// Host rechecks Git identity and captured file state, and before any target is
// changed. Host then rechecks those bindings again before the first mutation.
func ApplyChecked(root string, capture *Capture, changes []Change, validate func() error) (Result, error) {
	return applyGuardedWrite(root, capture, changes, validate, nil)
}

func applyGuardedWrite(root string, capture *Capture, changes []Change, validate func() error, beforeTarget func(string) error) (Result, error) {
	result := Result{CompletedPaths: []string{}}
	if capture == nil || capture.rootIdentity == nil || len(capture.seal) == 0 {
		return result, errors.New("guarded apply requires an intact Host capture")
	}
	seal, err := guardedCaptureSeal(capture)
	if err != nil || seal != capture.seal {
		return result, errors.New("guarded write capture changed after observation")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return result, err
	}
	// Git canonicalizes the captured root, while Windows callers may retain
	// an equivalent 8.3 or extended-prefix spelling. Normalize both spellings;
	// the opened-directory identity and reparse checks below remain required.
	rootSpelling, err := CanonicalPathSpelling(rootAbs)
	if err != nil {
		return result, fmt.Errorf("canonicalize guarded apply root spelling: %w", err)
	}
	capturedSpelling, err := CanonicalPathSpelling(capture.Root)
	if err != nil {
		return result, fmt.Errorf("canonicalize captured repository root spelling: %w", err)
	}
	if !SamePathSpelling(rootSpelling, capturedSpelling) {
		return result, errors.New("guarded apply root differs from captured repository root")
	}
	normalizedChanges, err := normalizeGuardedChanges(changes, capture.Files)
	if err != nil {
		return result, err
	}
	writer, err := OpenRoot(root)
	if err != nil {
		return result, err
	}
	defer writer.Close()
	if !os.SameFile(capture.rootIdentity, writer.identity) {
		return result, errors.New("guarded apply root identity changed since capture")
	}
	if err := ensureGuardedGitState(root, capture.Identity, capture.Branch, capture.Head); err != nil {
		return result, err
	}
	if err := compareGuardedFiles(writer, capture.Files); err != nil {
		return result, err
	}
	var unlock func()
	if _, selectedProjectManifest := capture.Files[".markitect/project.yaml"]; selectedProjectManifest {
		// Initialization captures the still-missing manifest, so select the
		// project lock explicitly before its existence can drive LockWriter.
		unlock, err = writer.lockWriterAt(".markitect", "write.lock")
	} else {
		unlock, err = writer.LockWriter()
	}
	if err != nil {
		return result, err
	}
	defer unlock()
	if err := writer.CheckIdentity(); err != nil {
		return result, err
	}
	if !os.SameFile(capture.rootIdentity, writer.identity) {
		return result, errors.New("guarded apply root identity changed while acquiring lock")
	}
	if err := ensureGuardedGitState(root, capture.Identity, capture.Branch, capture.Head); err != nil {
		return result, err
	}
	if err := compareGuardedFiles(writer, capture.Files); err != nil {
		return result, fmt.Errorf("selected file changed while acquiring the writer lock: %w", err)
	}
	if validate != nil {
		if err := validate(); err != nil {
			return result, fmt.Errorf("guarded write precondition failed: %w", err)
		}
		seal, err := guardedCaptureSeal(capture)
		if err != nil || seal != capture.seal {
			return result, errors.New("guarded write capture changed during precondition validation")
		}
		if err := writer.CheckIdentity(); err != nil {
			return result, err
		}
		if !os.SameFile(capture.rootIdentity, writer.identity) {
			return result, errors.New("guarded apply root identity changed during precondition validation")
		}
		if err := ensureGuardedGitState(root, capture.Identity, capture.Branch, capture.Head); err != nil {
			return result, err
		}
		if err := compareGuardedFiles(writer, capture.Files); err != nil {
			return result, fmt.Errorf("selected file changed during precondition validation: %w", err)
		}
	}
	expectedFiles := cloneGuardedWriteFiles(capture.Files)
	for _, change := range normalizedChanges {
		if beforeTarget != nil {
			if err := beforeTarget(change.Path); err != nil {
				return result, err
			}
		}
		if err := writer.CheckIdentity(); err != nil {
			return result, err
		}
		if err := ensureGuardedGitState(root, capture.Identity, capture.Branch, capture.Head); err != nil {
			return result, err
		}
		if err := compareGuardedFiles(writer, expectedFiles); err != nil {
			return result, fmt.Errorf("selected files changed during guarded apply: %w", err)
		}
		before := expectedFiles[change.Path]
		current, err := readGuardedWriteFile(writer, change.Path)
		if err != nil {
			return result, fmt.Errorf("recheck selected path %s: %w", change.Path, err)
		}
		if !equalGuardedWriteFile(current, before) {
			return result, fmt.Errorf("selected path changed during guarded apply: %s", change.Path)
		}
		if change.Delete {
			if !current.Exists {
				continue
			}
			if err := writer.RemoveRegular(change.Path); err != nil {
				if WasPublished(err) {
					result.CompletedPaths = append(result.CompletedPaths, change.Path)
				}
				return result, fmt.Errorf("delete selected path %s: %w", change.Path, err)
			}
			result.CompletedPaths = append(result.CompletedPaths, change.Path)
			expectedFiles[change.Path] = File{}
			continue
		}
		if current.Exists && bytes.Equal(current.Bytes, change.Bytes) && current.Mode == change.Mode.Perm() {
			continue
		}
		if err := writer.AtomicWrite(change.Path, change.Bytes, change.Mode.Perm()); err != nil {
			if WasPublished(err) {
				result.CompletedPaths = append(result.CompletedPaths, change.Path)
			}
			return result, fmt.Errorf("write selected path %s: %w", change.Path, err)
		}
		result.CompletedPaths = append(result.CompletedPaths, change.Path)
		written, err := readGuardedWriteFile(writer, change.Path)
		if err != nil {
			return result, fmt.Errorf("verify completed path %s: %w", change.Path, err)
		}
		if !written.Exists || !bytes.Equal(written.Bytes, change.Bytes) {
			return result, fmt.Errorf("completed path changed immediately after write: %s", change.Path)
		}
		expectedFiles[change.Path] = written
	}
	return result, nil
}

func normalizeGuardedPaths(input []string) ([]string, error) {
	if len(input) == 0 {
		return nil, errors.New("guarded capture requires at least one selected path")
	}
	paths := append([]string(nil), input...)
	seen := make(map[string]struct{}, len(paths))
	for _, name := range paths {
		if err := validateWritePath(name); err != nil {
			return nil, err
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("duplicate selected path %q", name)
		}
		seen[name] = struct{}{}
	}
	if err := source.ValidateIncludedPaths(paths); err != nil {
		return nil, fmt.Errorf("selected paths are unsafe or have portable aliases: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func normalizeGuardedChanges(changes []Change, selected map[string]File) ([]Change, error) {
	result := append([]Change(nil), changes...)
	paths := make([]string, 0, len(result))
	seen := make(map[string]struct{}, len(result))
	for i := range result {
		change := &result[i]
		if err := validateWritePath(change.Path); err != nil {
			return nil, err
		}
		if _, ok := selected[change.Path]; !ok {
			return nil, fmt.Errorf("guarded change is outside the captured selection: %s", change.Path)
		}
		if _, exists := seen[change.Path]; exists {
			return nil, fmt.Errorf("duplicate guarded change %q", change.Path)
		}
		seen[change.Path] = struct{}{}
		paths = append(paths, change.Path)
		if change.Delete {
			if len(change.Bytes) != 0 || change.Mode != 0 {
				return nil, fmt.Errorf("delete change must not include bytes or mode: %s", change.Path)
			}
			continue
		}
		if change.Mode.Perm() == 0 || change.Mode&^fs.FileMode(0777) != 0 {
			return nil, fmt.Errorf("guarded write requires permission bits from 0001 through 0777: %s", change.Path)
		}
		change.Bytes = append([]byte(nil), change.Bytes...)
	}
	if err := source.ValidateIncludedPaths(paths); err != nil {
		return nil, fmt.Errorf("guarded changes have unsafe or portable path aliases: %w", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func guardedGitState(root string) (source.GitIdentity, string, string, error) {
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return source.GitIdentity{}, "", "", fmt.Errorf("identify guarded write repository: %w", err)
	}
	branch, err := BranchName(root)
	if err != nil {
		return source.GitIdentity{}, "", "", err
	}
	head, err := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		headRef, refErr := source.GitOutput(root, "symbolic-ref", "--quiet", "HEAD")
		if refErr != nil {
			return source.GitIdentity{}, "", "", fmt.Errorf("read guarded write HEAD: %w", err)
		}
		ref := strings.TrimSpace(string(headRef))
		if ref != "refs/heads/"+branch {
			return source.GitIdentity{}, "", "", fmt.Errorf("unborn guarded write HEAD %q does not match named branch %q", ref, branch)
		}
		if _, branchRefErr := source.GitOutput(root, "show-ref", "--verify", "--quiet", ref); branchRefErr == nil {
			return source.GitIdentity{}, "", "", fmt.Errorf("named branch ref %s exists but guarded HEAD does not resolve to a commit: %w", ref, err)
		} else if !gitExitedWithCode(branchRefErr, 1) {
			return source.GitIdentity{}, "", "", fmt.Errorf("verify guarded unborn branch ref %s: %w", ref, branchRefErr)
		}
		return identity, branch, "unborn:" + ref, nil
	}
	return identity, branch, strings.TrimSpace(string(head)), nil
}

func gitExitedWithCode(err error, code int) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == code
}

func ensureGuardedGitState(root string, expectedIdentity source.GitIdentity, expectedBranch, expectedHead string) error {
	identity, branch, head, err := guardedGitState(root)
	if err != nil {
		return err
	}
	if identity != expectedIdentity {
		return errors.New("Git repository identity changed since guarded capture")
	}
	if err := EnsureBranch(root, expectedBranch); err != nil {
		return err
	}
	if branch != expectedBranch {
		return fmt.Errorf("branch changed during guarded write from %q to %q", expectedBranch, branch)
	}
	if head != expectedHead {
		return fmt.Errorf("HEAD changed during guarded write from %s to %s", expectedHead, head)
	}
	return nil
}

func readGuardedWriteFile(writer *Root, name string) (File, error) {
	if err := writer.checkPath(name); err != nil {
		return File{}, err
	}
	before, err := writer.Lstat(name)
	if os.IsNotExist(err) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	if !before.Mode().IsRegular() || IsReparsePoint(before) {
		return File{}, fmt.Errorf("selected path is not a regular file: %s", name)
	}
	contents, err := writer.ReadFile(name)
	if err != nil {
		return File{}, err
	}
	after, err := writer.Lstat(name)
	if err != nil {
		return File{}, err
	}
	if !after.Mode().IsRegular() || IsReparsePoint(after) || !os.SameFile(before, after) || before.Size() != after.Size() || before.Mode().Perm() != after.Mode().Perm() {
		return File{}, fmt.Errorf("selected file changed while being captured: %s", name)
	}
	return File{Exists: true, Bytes: contents, Mode: after.Mode().Perm()}, nil
}

func compareGuardedFiles(writer *Root, expected map[string]File) error {
	paths := make([]string, 0, len(expected))
	for name := range expected {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		current, err := readGuardedWriteFile(writer, name)
		if err != nil {
			return fmt.Errorf("recheck selected path %s: %w", name, err)
		}
		if !equalGuardedWriteFile(current, expected[name]) {
			return fmt.Errorf("selected path changed since capture: %s (exists %t/%t, mode %04o/%04o)", name, current.Exists, expected[name].Exists, current.Mode.Perm(), expected[name].Mode.Perm())
		}
	}
	return nil
}

func equalGuardedWriteFile(left, right File) bool {
	return left.Exists == right.Exists && left.Mode.Perm() == right.Mode.Perm() && bytes.Equal(left.Bytes, right.Bytes)
}

func cloneGuardedWriteFiles(files map[string]File) map[string]File {
	clone := make(map[string]File, len(files))
	for name, file := range files {
		file.Bytes = append([]byte(nil), file.Bytes...)
		clone[name] = file
	}
	return clone
}

func guardedCaptureSeal(capture *Capture) (string, error) {
	data, err := json.Marshal(struct {
		Root     string
		Identity source.GitIdentity
		Head     string
		Branch   string
		Files    map[string]File
	}{capture.Root, capture.Identity, capture.Head, capture.Branch, capture.Files})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
