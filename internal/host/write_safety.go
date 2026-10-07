package host

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// publishedWriteError reports that the bytes were atomically published into
// the pinned destination directory, but the directory's current pathname no
// longer resolves to that same directory. Callers must preserve the write in
// their partial-write report; rolling it back could delete a replacement.
type publishedWriteError struct {
	Path  string
	Cause error
}

func (e *publishedWriteError) Error() string {
	return fmt.Sprintf("object at %s was created or published, but its named destination changed: %v", e.Path, e.Cause)
}

func (e *publishedWriteError) Unwrap() error { return e.Cause }

func writeWasPublished(err error) bool {
	var published *publishedWriteError
	return errors.As(err, &published)
}

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

func (w *writeRoot) Mkdir(name string, mode os.FileMode) error {
	return w.mkdirWithHook(name, mode, nil)
}

func (w *writeRoot) mkdirWithHook(name string, mode os.FileMode, afterCreate func() error) error {
	if err := validateWritePath(name); err != nil {
		return err
	}
	parentName, leaf := splitWritePath(name)
	parent, closeParent, err := w.openDirectory(parentName, false, 0)
	if err != nil {
		return err
	}
	defer closeParent(parent)
	if err := w.checkIdentity(); err != nil {
		return err
	}
	if err := parent.Mkdir(leaf, mode); err != nil {
		return err
	}
	if afterCreate != nil {
		if err := afterCreate(); err != nil {
			return &publishedWriteError{Path: name, Cause: err}
		}
	}
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		return &publishedWriteError{Path: name, Cause: err}
	}
	return nil
}

func (w *writeRoot) CreateExclusive(name string, mode os.FileMode) (*os.File, error) {
	return w.createExclusiveWithHook(name, mode, nil)
}

func (w *writeRoot) createExclusiveWithHook(name string, mode os.FileMode, afterCreate func() error) (*os.File, error) {
	if err := validateWritePath(name); err != nil {
		return nil, err
	}
	parentName, leaf := splitWritePath(name)
	parent, closeParent, err := w.openDirectory(parentName, false, 0)
	if err != nil {
		return nil, err
	}
	defer closeParent(parent)
	if err := w.checkIdentity(); err != nil {
		return nil, err
	}
	file, err := parent.OpenFile(leaf, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return nil, err
	}
	if afterCreate != nil {
		if err := afterCreate(); err != nil {
			_ = file.Close()
			return nil, &publishedWriteError{Path: name, Cause: err}
		}
	}
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		_ = file.Close()
		return nil, &publishedWriteError{Path: name, Cause: err}
	}
	return file, nil
}

func splitWritePath(name string) (string, string) {
	dir, leaf := filepath.Split(filepath.FromSlash(name))
	dir = filepath.ToSlash(filepath.Clean(dir))
	if dir == "." {
		dir = ""
	}
	return dir, leaf
}

func (w *writeRoot) checkNamedDirectoryIdentity(name string, expected *os.Root) error {
	if err := w.checkIdentity(); err != nil {
		return err
	}
	actualRoot, closeActual, err := w.openDirectory(name, false, 0)
	if err != nil {
		return err
	}
	defer closeActual(actualRoot)
	expectedInfo, err := expected.Stat(".")
	if err != nil {
		return err
	}
	actualInfo, err := actualRoot.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(expectedInfo, actualInfo) {
		return fmt.Errorf("output parent changed during write: %s", name)
	}
	return nil
}

func (w *writeRoot) openDirectory(name string, create bool, mode os.FileMode) (*os.Root, func(*os.Root) error, error) {
	if name == "" || name == "." {
		return w.root, func(*os.Root) error { return nil }, nil
	}
	if err := validateWritePath(name); err != nil {
		return nil, nil, err
	}
	if err := w.checkIdentity(); err != nil {
		return nil, nil, err
	}
	var current *os.Root = w.root
	owned := false
	closeCurrent := func() {
		if owned {
			_ = current.Close()
		}
	}
	for _, part := range strings.Split(name, "/") {
		info, err := current.Lstat(part)
		if os.IsNotExist(err) && create {
			err = current.Mkdir(part, mode)
			if err == nil || os.IsExist(err) {
				info, err = current.Lstat(part)
			}
		}
		if err != nil {
			closeCurrent()
			return nil, nil, err
		}
		if isReparsePoint(info) || !info.IsDir() {
			closeCurrent()
			return nil, nil, fmt.Errorf("non-directory or reparse point in output parent %s", name)
		}
		next, err := current.OpenRoot(part)
		if err != nil {
			closeCurrent()
			return nil, nil, err
		}
		actual, err := next.Stat(".")
		if err != nil || !os.SameFile(info, actual) {
			next.Close()
			closeCurrent()
			if err != nil {
				return nil, nil, err
			}
			return nil, nil, fmt.Errorf("output parent changed while opening: %s", name)
		}
		closeCurrent()
		current = next
		owned = true
	}
	return current, func(root *os.Root) error {
		if root == w.root {
			return nil
		}
		return root.Close()
	}, nil
}

func (w *writeRoot) ReadFile(name string) ([]byte, error) {
	return w.readFileWithHooks(name, nil, nil)
}

func (w *writeRoot) readFileWithHooks(name string, beforeOpen, beforeAccept func() error) ([]byte, error) {
	if err := validateWritePath(name); err != nil {
		return nil, err
	}
	parentName, leaf := splitWritePath(name)
	parent, closeParent, err := w.openDirectory(parentName, false, 0)
	if err != nil {
		return nil, err
	}
	defer closeParent(parent)
	expected, err := parent.Lstat(leaf)
	if err != nil {
		return nil, err
	}
	if isReparsePoint(expected) || !expected.Mode().IsRegular() {
		return nil, fmt.Errorf("non-regular file or reparse point in output path %s", name)
	}
	if beforeOpen != nil {
		if err := beforeOpen(); err != nil {
			return nil, err
		}
	}
	file, err := parent.Open(leaf)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if isReparsePoint(opened) || !opened.Mode().IsRegular() || opened.Size() != expected.Size() || !os.SameFile(expected, opened) {
		return nil, fmt.Errorf("output file identity changed while opening: %s", name)
	}
	data, err := io.ReadAll(io.LimitReader(file, expected.Size()+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != expected.Size() {
		return nil, fmt.Errorf("selected file size changed while reading: %s", name)
	}
	readInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if readInfo.Size() != expected.Size() || !os.SameFile(opened, readInfo) {
		return nil, fmt.Errorf("selected file size or identity changed while reading: %s", name)
	}
	if beforeAccept != nil {
		if err := beforeAccept(); err != nil {
			return nil, err
		}
	}
	current, err := parent.Lstat(leaf)
	if err != nil {
		return nil, err
	}
	if isReparsePoint(current) || !current.Mode().IsRegular() || current.Size() != expected.Size() || !os.SameFile(opened, current) {
		return nil, fmt.Errorf("output file identity changed while reading: %s", name)
	}
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		return nil, err
	}
	return data, nil
}

func (w *writeRoot) Lstat(name string) (os.FileInfo, error) {
	if err := validateWritePath(name); err != nil {
		return nil, err
	}
	parentName, leaf := splitWritePath(name)
	parent, closeParent, err := w.openDirectory(parentName, false, 0)
	if err != nil {
		return nil, err
	}
	defer closeParent(parent)
	info, err := parent.Lstat(leaf)
	if err != nil {
		return nil, err
	}
	if isReparsePoint(info) {
		return nil, fmt.Errorf("symlink or reparse point in output path %s", name)
	}
	return info, nil
}

func (w *writeRoot) AtomicWrite(name string, data []byte, mode os.FileMode) error {
	return w.atomicWriteWithHooks(name, data, mode, nil, nil)
}

// atomicWriteWithHook exposes the final pre-open boundary to focused tests so
// a directory swap can be injected in the exact former path-based race window.
func (w *writeRoot) atomicWriteWithHook(name string, data []byte, mode os.FileMode, beforeOpen func() error) error {
	return w.atomicWriteWithHooks(name, data, mode, beforeOpen, nil)
}

func (w *writeRoot) atomicWriteWithHooks(name string, data []byte, mode os.FileMode, beforeOpen, beforeRename func() error) error {
	if err := w.checkPath(name); err != nil {
		return err
	}
	parentName, leaf := splitWritePath(name)
	if err := w.checkIdentity(); err != nil {
		return err
	}
	parent, closeParent, err := w.openDirectory(parentName, true, 0755)
	if err != nil {
		return err
	}
	defer closeParent(parent)
	if info, err := parent.Lstat(leaf); err == nil && isReparsePoint(info) {
		return fmt.Errorf("symlink or reparse point in output path %s", name)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	var temporary string
	var file *os.File
	hookCalled := false
	for attempts := 0; attempts < 8; attempts++ {
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return err
		}
		temporary = ".markitect-" + hex.EncodeToString(random[:])
		if !hookCalled && beforeOpen != nil {
			if err := beforeOpen(); err != nil {
				return err
			}
			hookCalled = true
		}
		created, openErr := parent.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if openErr == nil {
			file = created
			if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
				_ = file.Close()
				_ = parent.Remove(temporary)
				return err
			}
			break
		}
		if !os.IsExist(openErr) {
			return openErr
		}
	}
	if file == nil {
		return fmt.Errorf("could not allocate temporary output for %s", name)
	}
	cleanup := func() { _ = parent.Remove(temporary) }
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
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		cleanup()
		return err
	}
	if err := w.checkIdentity(); err != nil {
		cleanup()
		return err
	}
	if beforeRename != nil {
		if err := beforeRename(); err != nil {
			cleanup()
			return err
		}
	}
	if err := parent.Rename(temporary, leaf); err != nil {
		cleanup()
		return err
	}
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		return &publishedWriteError{Path: name, Cause: err}
	}
	return nil
}

func (w *writeRoot) LockWriter() (func(), error) {
	parent, closeParent, err := w.openDirectory(".artifacts/markitect", true, 0755)
	if err != nil {
		return nil, err
	}
	if err := w.checkIdentity(); err != nil {
		_ = closeParent(parent)
		return nil, err
	}
	lock, err := parent.OpenFile("write.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		_ = closeParent(parent)
		return nil, fmt.Errorf("renderer lock is already present: %w", err)
	}
	lockInfo, err := lock.Stat()
	if err != nil {
		_ = lock.Close()
		_ = closeParent(parent)
		return nil, &publishedWriteError{Path: ".artifacts/markitect/write.lock", Cause: err}
	}
	if err := w.checkNamedDirectoryIdentity(".artifacts/markitect", parent); err != nil {
		_ = lock.Close()
		_ = closeParent(parent)
		return nil, &publishedWriteError{Path: ".artifacts/markitect/write.lock", Cause: err}
	}
	var releaseOnce sync.Once
	return func() {
		releaseOnce.Do(func() {
			if w.checkIdentity() == nil && w.checkNamedDirectoryIdentity(".artifacts/markitect", parent) == nil {
				if current, err := parent.Lstat("write.lock"); err == nil && !isReparsePoint(current) && os.SameFile(lockInfo, current) {
					_ = parent.Remove("write.lock")
				}
			}
			_ = lock.Close()
			_ = closeParent(parent)
		})
	}, nil
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
