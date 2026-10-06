package host

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func safeDestination(root, name string) (string, error) {
	if err := validateWritePath(name); err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	// Check the root and every ancestor directly; EvalSymlinks alone does not
	// identify all Windows reparse tags and textual equality misses short names.
	if err := rejectReparseAncestors(absolute); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalPathSpelling(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize output root spelling: %w", err)
	}
	resolvedCanonical, err := canonicalPathSpelling(resolved)
	if err != nil {
		return "", fmt.Errorf("canonicalize resolved output root: %w", err)
	}
	if !samePathSpelling(resolvedCanonical, canonical) {
		return "", fmt.Errorf("output root contains a symlink: %s", root)
	}
	current := absolute
	for _, part := range strings.Split(name, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if isReparsePoint(info) {
			return "", fmt.Errorf("symlink in output path %s", name)
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return "", err
		}
		canonical, err := canonicalPathSpelling(current)
		if err != nil {
			return "", fmt.Errorf("canonicalize output path spelling: %w", err)
		}
		resolvedCanonical, err := canonicalPathSpelling(resolved)
		if err != nil {
			return "", fmt.Errorf("canonicalize resolved output path: %w", err)
		}
		if !samePathSpelling(resolvedCanonical, canonical) {
			return "", fmt.Errorf("reparse point in output path %s", name)
		}
	}
	return current, nil
}

func validateWritePath(name string) error {
	if name == "" || strings.Contains(name, "\\") || strings.Contains(name, ":") || filepath.IsAbs(name) {
		return fmt.Errorf("unsafe output path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return fmt.Errorf("unsafe output path %q", name)
		}
	}
	return nil
}

// writeRoot pins all output mutations to the identity of the approved project
// directory. Path checks remain useful for policy, but never authorize a
// path-based mutation.
type writeRoot struct {
	root     *os.Root
	absolute string
	identity os.FileInfo
}

func openWriteRoot(path string) (*writeRoot, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := rejectReparseAncestors(absolute); err != nil {
		return nil, err
	}
	expected, err := os.Lstat(absolute)
	if err != nil {
		return nil, fmt.Errorf("inspect approved write root: %w", err)
	}
	if !expected.IsDir() || isReparsePoint(expected) {
		return nil, fmt.Errorf("approved write root is not a plain directory: %s", path)
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("open approved write root: %w", err)
	}
	actual, err := root.Stat(".")
	if err != nil {
		root.Close()
		return nil, fmt.Errorf("inspect opened write root: %w", err)
	}
	if !os.SameFile(expected, actual) {
		root.Close()
		return nil, fmt.Errorf("approved write root changed while opening: %s", path)
	}
	return &writeRoot{root: root, absolute: absolute, identity: actual}, nil
}

func (w *writeRoot) Close() error { return w.root.Close() }

func (w *writeRoot) checkPath(name string) error {
	if err := validateWritePath(name); err != nil {
		return err
	}
	current := ""
	parts := strings.Split(name, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := w.root.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if isReparsePoint(info) {
			return fmt.Errorf("symlink or reparse point in output path %s", name)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("non-directory component in output path %s", name)
		}
	}
	return nil
}

func (w *writeRoot) checkIdentity() error {
	if err := rejectReparseAncestors(w.absolute); err != nil {
		return err
	}
	current, err := os.Lstat(w.absolute)
	if err != nil || !os.SameFile(w.identity, current) || isReparsePoint(current) {
		if err != nil {
			return fmt.Errorf("approved write root moved or changed: %w", err)
		}
		return fmt.Errorf("approved write root moved or changed: %s", w.absolute)
	}
	return nil
}

func (w *writeRoot) MkdirAll(name string, mode os.FileMode) error {
	if name == "." || name == "" {
		return nil
	}
	if err := w.checkPath(name); err != nil {
		return err
	}
	if err := w.checkIdentity(); err != nil {
		return err
	}
	if err := w.root.MkdirAll(filepath.FromSlash(name), mode); err != nil {
		return err
	}
	return w.checkPath(name)
}

func (w *writeRoot) Mkdir(name string, mode os.FileMode) error {
	if err := w.checkPath(name); err != nil {
		return err
	}
	if err := w.checkIdentity(); err != nil {
		return err
	}
	return w.root.Mkdir(filepath.FromSlash(name), mode)
}

func (w *writeRoot) CreateExclusive(name string, mode os.FileMode) (*os.File, error) {
	if err := w.checkPath(name); err != nil {
		return nil, err
	}
	if err := w.checkIdentity(); err != nil {
		return nil, err
	}
	return w.root.OpenFile(filepath.FromSlash(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
}

func (w *writeRoot) ReadFile(name string) ([]byte, error) {
	if err := w.checkPath(name); err != nil {
		return nil, err
	}
	return w.root.ReadFile(filepath.FromSlash(name))
}

func (w *writeRoot) Lstat(name string) (os.FileInfo, error) {
	if err := w.checkPath(name); err != nil {
		return nil, err
	}
	return w.root.Lstat(filepath.FromSlash(name))
}

func (w *writeRoot) AtomicWrite(name string, data []byte, mode os.FileMode) error {
	if err := w.checkPath(name); err != nil {
		return err
	}
	parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(name)))
	if parent == "." {
		parent = ""
	} else if err := w.MkdirAll(parent, 0755); err != nil {
		return err
	}
	if err := w.checkPath(name); err != nil {
		return err
	}
	if err := w.checkIdentity(); err != nil {
		return err
	}
	var temporary string
	var file *os.File
	for attempts := 0; attempts < 8; attempts++ {
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return err
		}
		temporary = filepath.Join(parent, ".markitect-"+hex.EncodeToString(random[:]))
		created, openErr := w.root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if openErr == nil {
			file = created
			break
		}
		if !os.IsExist(openErr) {
			return openErr
		}
	}
	if file == nil {
		return fmt.Errorf("could not allocate temporary output for %s", name)
	}
	cleanup := func() { _ = w.root.Remove(temporary) }
	if _, err := file.Write(data); err != nil {
		file.Close()
		cleanup()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		cleanup()
		return err
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return err
	}
	if err := w.checkPath(name); err != nil {
		cleanup()
		return err
	}
	if err := w.checkIdentity(); err != nil {
		cleanup()
		return err
	}
	if err := w.root.Rename(temporary, filepath.FromSlash(name)); err != nil {
		cleanup()
		return err
	}
	return nil
}

func (w *writeRoot) LockWriter() (func(), error) {
	const lockName = ".artifacts/markitect/write.lock"
	if err := w.MkdirAll(".artifacts/markitect", 0755); err != nil {
		return nil, err
	}
	if err := w.checkIdentity(); err != nil {
		return nil, err
	}
	lock, err := w.root.OpenFile(lockName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("renderer lock is already present: %w", err)
	}
	if err := lock.Close(); err != nil {
		_ = w.root.Remove(lockName)
		return nil, err
	}
	return func() { _ = w.root.Remove(lockName) }, nil
}

func rejectReparseAncestors(path string) error {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect output root ancestor %s: %w", current, err)
		}
		if isReparsePoint(info) {
			return fmt.Errorf("output root contains a symlink or reparse point: %s", current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

// hasGitMetadata checks the selected directory and its ancestors so nested
// source roots still receive the same branch guard as repository roots.
func hasGitMetadata(root string) bool {
	current := filepath.Clean(root)
	for {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		current = parent
	}
}

func writableBranch(root string) error {
	_, err := writeBranchName(root)
	return err
}

// writeBranchName returns the named, non-protected Git branch for a write.
func writeBranchName(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	branch, err := source.GitOutput(absolute, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("writing canonical sources requires an isolated Git branch: %w", err)
	}
	name := strings.TrimSpace(string(branch))
	if name == "" || strings.EqualFold(name, "master") || strings.EqualFold(name, "main") {
		return "", fmt.Errorf("writing requires an isolated non-protected Git branch; detached HEAD and protected branches are not writable")
	}
	return name, nil
}

// ensureWriteBranch catches a branch switch, including one to a different
// branch at the same HEAD, during a multi-file write.
func ensureWriteBranch(root, expected string) error {
	current, err := writeBranchName(root)
	if err != nil {
		return fmt.Errorf("write branch changed or became protected: %w", err)
	}
	if current != expected {
		return fmt.Errorf("branch changed during write from %q to %q", expected, current)
	}
	return nil
}
