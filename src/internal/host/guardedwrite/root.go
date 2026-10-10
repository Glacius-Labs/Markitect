package guardedwrite

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
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

// WasPublished reports whether err means the mutation was published even
// though the named destination changed afterwards.
func WasPublished(err error) bool {
	var published *publishedWriteError
	return errors.As(err, &published)
}

// SafeDestination returns the absolute path for a repository-relative output
// name after rejecting unsafe names and any symlink or reparse point on the
// root, its ancestors or the existing part of the output path.
func SafeDestination(root, name string) (string, error) {
	if err := validateWritePath(name); err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	// Check the root and every ancestor directly; EvalSymlinks alone does not
	// identify all Windows reparse tags and textual equality misses short names.
	if err := RejectReparseAncestors(absolute); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	canonical, err := CanonicalPathSpelling(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize output root spelling: %w", err)
	}
	resolvedCanonical, err := CanonicalPathSpelling(resolved)
	if err != nil {
		return "", fmt.Errorf("canonicalize resolved output root: %w", err)
	}
	if !SamePathSpelling(resolvedCanonical, canonical) {
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
		if IsReparsePoint(info) {
			return "", fmt.Errorf("symlink in output path %s", name)
		}
		if err := requireStoredName(os.DirFS(filepath.Dir(current)), ".", part); err != nil {
			return "", fmt.Errorf("unsafe output path %s: %w", name, err)
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return "", err
		}
		canonical, err := CanonicalPathSpelling(current)
		if err != nil {
			return "", fmt.Errorf("canonicalize output path spelling: %w", err)
		}
		resolvedCanonical, err := CanonicalPathSpelling(resolved)
		if err != nil {
			return "", fmt.Errorf("canonicalize resolved output path: %w", err)
		}
		if !SamePathSpelling(resolvedCanonical, canonical) {
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
		// Like Git's core.protectNTFS, refuse git~1 too: it is the usual
		// Windows 8.3 short name of .git.
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") || strings.EqualFold(part, "git~1") {
			return fmt.Errorf("unsafe output path %q", name)
		}
	}
	return nil
}

// requireStoredName refuses an existing path component that Windows resolved
// through another spelling: an 8.3 short name such as GIT~1 for .git, or a
// case variant. Lexical checks see only the requested text, so a write through
// such an alias would reach a path they never authorized. Only stored names
// appear in a directory listing.
func requireStoredName(directory fs.FS, parent, part string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	entries, err := fs.ReadDir(directory, parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == part {
			return nil
		}
	}
	return fmt.Errorf("%q names an existing entry stored under another name", part)
}

// Root pins all output mutations to the identity of the approved project
// directory. Path checks remain useful for policy, but never authorize a
// path-based mutation.
type Root struct {
	root     *os.Root
	absolute string
	identity os.FileInfo
}

// OpenRoot opens path as a write root after checking that it and its
// ancestors are plain directories.
func OpenRoot(path string) (*Root, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := RejectReparseAncestors(absolute); err != nil {
		return nil, err
	}
	expected, err := os.Lstat(absolute)
	if err != nil {
		return nil, fmt.Errorf("inspect approved write root: %w", err)
	}
	if !expected.IsDir() || IsReparsePoint(expected) {
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
	return &Root{root: root, absolute: absolute, identity: actual}, nil
}

func (w *Root) Close() error { return w.root.Close() }

func (w *Root) checkPath(name string) error {
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
		if IsReparsePoint(info) {
			return fmt.Errorf("symlink or reparse point in output path %s", name)
		}
		parent := "."
		if i > 0 {
			parent = strings.Join(parts[:i], "/")
		}
		if err := requireStoredName(w.root.FS(), parent, part); err != nil {
			return fmt.Errorf("unsafe output path %s: %w", name, err)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("non-directory component in output path %s", name)
		}
	}
	return nil
}

// CheckIdentity reports an error when the root directory has moved, been
// replaced or become a symlink or reparse point since it was opened.
func (w *Root) CheckIdentity() error {
	if err := RejectReparseAncestors(w.absolute); err != nil {
		return err
	}
	current, err := os.Lstat(w.absolute)
	if err != nil || !os.SameFile(w.identity, current) || IsReparsePoint(current) {
		if err != nil {
			return fmt.Errorf("approved write root moved or changed: %w", err)
		}
		return fmt.Errorf("approved write root moved or changed: %s", w.absolute)
	}
	return nil
}

func (w *Root) Mkdir(name string, mode os.FileMode) error {
	return w.mkdirWithHook(name, mode, nil)
}

func (w *Root) mkdirWithHook(name string, mode os.FileMode, afterCreate func() error) error {
	if err := validateWritePath(name); err != nil {
		return err
	}
	parentName, leaf := splitWritePath(name)
	parent, closeParent, err := w.openDirectory(parentName, false, 0)
	if err != nil {
		return err
	}
	defer closeParent(parent)
	if err := w.CheckIdentity(); err != nil {
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

func (w *Root) CreateExclusive(name string, mode os.FileMode) (*os.File, error) {
	return w.createExclusiveWithHook(name, mode, nil)
}

func (w *Root) createExclusiveWithHook(name string, mode os.FileMode, afterCreate func() error) (*os.File, error) {
	if err := validateWritePath(name); err != nil {
		return nil, err
	}
	parentName, leaf := splitWritePath(name)
	parent, closeParent, err := w.openDirectory(parentName, false, 0)
	if err != nil {
		return nil, err
	}
	defer closeParent(parent)
	if err := w.CheckIdentity(); err != nil {
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

func (w *Root) checkNamedDirectoryIdentity(name string, expected *os.Root) error {
	if err := w.CheckIdentity(); err != nil {
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

func (w *Root) openDirectory(name string, create bool, mode os.FileMode) (*os.Root, func(*os.Root) error, error) {
	if name == "" || name == "." {
		return w.root, func(*os.Root) error { return nil }, nil
	}
	if err := validateWritePath(name); err != nil {
		return nil, nil, err
	}
	if err := w.CheckIdentity(); err != nil {
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
		if IsReparsePoint(info) || !info.IsDir() {
			closeCurrent()
			return nil, nil, fmt.Errorf("non-directory or reparse point in output parent %s", name)
		}
		// Check after any create: a component missing from checkPath's view
		// may since have resolved to an existing entry through an alias.
		if err := requireStoredName(current.FS(), ".", part); err != nil {
			closeCurrent()
			return nil, nil, fmt.Errorf("unsafe output parent %s: %w", name, err)
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

func (w *Root) ReadFile(name string) ([]byte, error) {
	return w.readFileWithHooks(name, nil, nil)
}

func (w *Root) readFileWithHooks(name string, beforeOpen, beforeAccept func() error) ([]byte, error) {
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
	if IsReparsePoint(expected) || !expected.Mode().IsRegular() {
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
	if IsReparsePoint(opened) || !opened.Mode().IsRegular() || opened.Size() != expected.Size() || !os.SameFile(expected, opened) {
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
	if IsReparsePoint(current) || !current.Mode().IsRegular() || current.Size() != expected.Size() || !os.SameFile(opened, current) {
		return nil, fmt.Errorf("output file identity changed while reading: %s", name)
	}
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		return nil, err
	}
	return data, nil
}

func (w *Root) Lstat(name string) (os.FileInfo, error) {
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
	if IsReparsePoint(info) {
		return nil, fmt.Errorf("symlink or reparse point in output path %s", name)
	}
	return info, nil
}

// RemoveRegular removes one regular file through its pinned parent directory.
// It refuses directories, links and reparse points, and reports a published
// removal if the named parent changes after the mutation.
func (w *Root) RemoveRegular(name string) error {
	if err := w.checkPath(name); err != nil {
		return err
	}
	parentName, leaf := splitWritePath(name)
	parent, closeParent, err := w.openDirectory(parentName, false, 0)
	if err != nil {
		return err
	}
	defer closeParent(parent)
	info, err := parent.Lstat(leaf)
	if err != nil {
		return err
	}
	if IsReparsePoint(info) || !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to remove non-regular file or reparse point %s", name)
	}
	if err := w.CheckIdentity(); err != nil {
		return err
	}
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		return err
	}
	current, err := parent.Lstat(leaf)
	if err != nil {
		return err
	}
	if IsReparsePoint(current) || !current.Mode().IsRegular() || !os.SameFile(info, current) {
		return fmt.Errorf("output file changed before removal: %s", name)
	}
	if err := parent.Remove(leaf); err != nil {
		return err
	}
	if err := w.checkNamedDirectoryIdentity(parentName, parent); err != nil {
		return &publishedWriteError{Path: name, Cause: err}
	}
	return nil
}

func (w *Root) AtomicWrite(name string, data []byte, mode os.FileMode) error {
	return w.atomicWriteWithHooks(name, data, mode, nil, nil)
}

// atomicWriteWithHook exposes the final pre-open boundary to focused tests so
// a directory swap can be injected in the exact former path-based race window.
func (w *Root) atomicWriteWithHook(name string, data []byte, mode os.FileMode, beforeOpen func() error) error {
	return w.atomicWriteWithHooks(name, data, mode, beforeOpen, nil)
}

func (w *Root) atomicWriteWithHooks(name string, data []byte, mode os.FileMode, beforeOpen, beforeRename func() error) error {
	if err := w.checkPath(name); err != nil {
		return err
	}
	parentName, leaf := splitWritePath(name)
	if err := w.CheckIdentity(); err != nil {
		return err
	}
	parent, closeParent, err := w.openDirectory(parentName, true, 0755)
	if err != nil {
		return err
	}
	defer closeParent(parent)
	if info, err := parent.Lstat(leaf); err == nil && IsReparsePoint(info) {
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
	if err := w.CheckIdentity(); err != nil {
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

func (w *Root) LockWriter() (func(), error) {
	lockDirectory := ".artifacts/markitect"
	manifest, err := w.Lstat(".markitect/project.yaml")
	if err == nil {
		if IsReparsePoint(manifest) || !manifest.Mode().IsRegular() {
			return nil, errors.New("project manifest must be a regular file before selecting its writer lock")
		}
		lockDirectory = ".markitect"
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect project manifest before selecting writer lock: %w", err)
	}
	return w.lockWriterAt(lockDirectory, "write.lock")
}

// lockWriterAt acquires the shared write lock at an explicit project-relative
// directory. It uses the same pinned-root and parent-identity checks as the
// compatibility LockWriter path.
func (w *Root) lockWriterAt(lockDirectory, leaf string) (func(), error) {
	if err := validateWritePath(lockDirectory); err != nil {
		return nil, fmt.Errorf("unsafe writer lock directory: %w", err)
	}
	if err := validateWritePath(leaf); err != nil || strings.Contains(leaf, "/") {
		return nil, fmt.Errorf("unsafe writer lock filename %q", leaf)
	}
	lockPath := lockDirectory + "/" + leaf
	parent, closeParent, err := w.openDirectory(lockDirectory, true, 0755)
	if err != nil {
		return nil, err
	}
	if err := w.CheckIdentity(); err != nil {
		_ = closeParent(parent)
		return nil, err
	}
	lock, err := parent.OpenFile(leaf, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		_ = closeParent(parent)
		return nil, fmt.Errorf("renderer lock is already present: %w", err)
	}
	lockInfo, err := lock.Stat()
	if err != nil {
		_ = lock.Close()
		_ = closeParent(parent)
		return nil, &publishedWriteError{Path: lockPath, Cause: err}
	}
	if err := w.checkNamedDirectoryIdentity(lockDirectory, parent); err != nil {
		_ = lock.Close()
		_ = closeParent(parent)
		return nil, &publishedWriteError{Path: lockPath, Cause: err}
	}
	var releaseOnce sync.Once
	return func() {
		releaseOnce.Do(func() {
			if w.CheckIdentity() == nil && w.checkNamedDirectoryIdentity(lockDirectory, parent) == nil {
				if current, err := parent.Lstat(leaf); err == nil && !IsReparsePoint(current) && os.SameFile(lockInfo, current) {
					_ = parent.Remove(leaf)
				}
			}
			_ = lock.Close()
			_ = closeParent(parent)
		})
	}, nil
}

// RejectReparseAncestors rejects path when it or any ancestor is a symlink or
// reparse point.
func RejectReparseAncestors(path string) error {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect output root ancestor %s: %w", current, err)
		}
		if IsReparsePoint(info) {
			return fmt.Errorf("output root contains a symlink or reparse point: %s", current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

// HasGitMetadata checks the selected directory and its ancestors so nested
// source roots still receive the same branch guard as repository roots.
func HasGitMetadata(root string) bool {
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

// BranchName returns the named, non-protected Git branch for a write.
func BranchName(root string) (string, error) {
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

// EnsureBranch catches a branch switch, including one to a different
// branch at the same HEAD, during a multi-file write.
func EnsureBranch(root, expected string) error {
	current, err := BranchName(root)
	if err != nil {
		return fmt.Errorf("write branch changed or became protected: %w", err)
	}
	if current != expected {
		return fmt.Errorf("branch changed during write from %q to %q", expected, current)
	}
	return nil
}
