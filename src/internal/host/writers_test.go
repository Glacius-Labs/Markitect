package host

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func initWriterRepo(t *testing.T, root string) {
	t.Helper()
	runWriterGit(t, root, "init", "-b", "feature/test-writes")
	runWriterGit(t, root, "config", "user.name", "Markitect Test")
	runWriterGit(t, root, "config", "user.email", "markitect-test@example.invalid")
}

func runWriterGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	gitArgs := append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)
	command := exec.Command("git", gitArgs...)
	command.Env = source.CleanGitEnv()
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestWriteSchemasRefusesUnmanagedTargetAndParentTraversal(t *testing.T) {
	t.Run("unmanaged target", func(t *testing.T) {
		root := tempRoot(t)
		target := filepath.Join(root, "schema/v1alpha1.json")
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		original := []byte("hand-authored schema\n")
		if err := os.WriteFile(target, original, 0644); err != nil {
			t.Fatal(err)
		}
		if err := WriteSchemas(root, map[string][]byte{"schema/v1alpha1.json": []byte("generated\n")}); err == nil || !strings.Contains(err.Error(), "unmanaged schema output") {
			t.Fatalf("WriteSchemas error = %v, want unmanaged-target refusal", err)
		}
		actual, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if string(actual) != string(original) {
			t.Fatal("unmanaged schema target was changed")
		}
	})

	t.Run("parent traversal", func(t *testing.T) {
		root := tempRoot(t)
		if err := WriteSchemas(root, map[string][]byte{"schema/../outside.json": []byte("escaped\n")}); err == nil || !strings.Contains(err.Error(), "unsafe output path") {
			t.Fatalf("WriteSchemas error = %v, want traversal refusal", err)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(root), "outside.json")); !os.IsNotExist(err) {
			t.Fatalf("schema write escaped its root: stat error = %v", err)
		}
	})
}

func TestWriteSchemasRefusesSymlinkPath(t *testing.T) {
	root := tempRoot(t)
	outside := tempRoot(t)
	if err := os.Symlink(outside, filepath.Join(root, "schema")); err != nil {
		t.Skipf("symlink creation is unavailable in this environment: %v", err)
	}
	err := WriteSchemas(root, map[string][]byte{"schema/v1alpha1.json": []byte("generated\n")})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("WriteSchemas error = %v, want symlink-path refusal", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "v1alpha1.json")); !os.IsNotExist(err) {
		t.Fatalf("schema write followed a symlink: stat error = %v", err)
	}
}

func TestSafeDestinationRejectsCaseOnlySymlinkAncestor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows path comparison is case-insensitive; reparse-point coverage is platform-specific")
	}
	base := t.TempDir()
	targetParent := filepath.Join(base, "parent")
	if err := os.MkdirAll(filepath.Join(targetParent, "repo"), 0755); err != nil {
		t.Fatal(err)
	}
	aliasParent := filepath.Join(base, "Parent")
	if err := os.Symlink(targetParent, aliasParent); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	root := filepath.Join(aliasParent, "repo")
	if _, err := safeDestination(root, "output.md"); err == nil || (!strings.Contains(strings.ToLower(err.Error()), "symlink") && !strings.Contains(strings.ToLower(err.Error()), "reparse")) {
		t.Fatalf("safeDestination accepted case-only symlink ancestor %q: %v", root, err)
	}
}

func TestAnchoredAtomicWriteRefusesRootSwap(t *testing.T) {
	base := t.TempDir()
	rootPath := filepath.Join(base, "repo")
	if err := os.Mkdir(rootPath, 0755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()

	movedRoot := filepath.Join(base, "repo-original")
	if err := os.Rename(rootPath, movedRoot); err != nil {
		t.Skipf("platform prevents swapping an open root handle: %v", err)
	}
	if err := os.Symlink(outside, rootPath); err != nil {
		t.Skipf("directory symlink unavailable for deterministic root-swap test: %v", err)
	}
	if err := anchored.AtomicWrite("escaped.txt", []byte("must not write\n"), 0644); err == nil {
		t.Fatal("anchored writer accepted a swapped repository root")
	}
	if _, err := os.Stat(filepath.Join(outside, "escaped.txt")); !os.IsNotExist(err) {
		t.Fatalf("write escaped through replacement root: %v", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("external sentinel changed: bytes=%q error=%v", got, err)
	}
}

func TestAnchoredAtomicWriteRefusesSwappedParent(t *testing.T) {
	rootPath := tempRoot(t)
	parent := filepath.Join(rootPath, "target")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	outside := tempRoot(t)
	sentinel := filepath.Join(outside, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	if err := os.Rename(parent, filepath.Join(rootPath, "target-original")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, parent); err != nil {
		t.Skipf("directory symlink unavailable for deterministic parent-swap test: %v", err)
	}
	if err := anchored.AtomicWrite("target/escaped.txt", []byte("must not write\n"), 0644); err == nil {
		t.Fatal("anchored writer accepted a swapped parent directory")
	}
	if _, err := os.Stat(filepath.Join(outside, "escaped.txt")); !os.IsNotExist(err) {
		t.Fatalf("write escaped through replacement parent: %v", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("external sentinel changed: bytes=%q error=%v", got, err)
	}
}

func TestAnchoredAtomicWriteContainsSwapAtPreOpenBoundary(t *testing.T) {
	rootPath := tempRoot(t)
	parent := filepath.Join(rootPath, "target")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	outside := tempRoot(t)
	sentinel := filepath.Join(outside, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	original := filepath.Join(rootPath, "target-original")
	var hookReached, injected bool
	var injectionErr error
	err = anchored.atomicWriteWithHook("target/escaped.txt", []byte("must not write\n"), 0644, func() error {
		hookReached = true
		if err := os.Rename(parent, original); err != nil {
			injectionErr = err
			return err
		}
		if err := os.Symlink(outside, parent); err != nil {
			injectionErr = err
			return err
		}
		injected = true
		return nil
	})
	if !hookReached {
		t.Fatalf("pre-open race hook did not execute: %v", err)
	}
	if !injected {
		t.Skipf("directory swap could not be injected: %v", injectionErr)
	}
	if err == nil {
		t.Fatal("rooted OpenFile accepted a parent swapped after path and root identity validation")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel.txt" {
		t.Fatalf("external directory received artifact/temp writes: entries=%v error=%v", entries, err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("external sentinel changed: bytes=%q error=%v", got, err)
	}
	originalEntries, err := os.ReadDir(original)
	if err != nil || len(originalEntries) != 0 {
		t.Fatalf("race wrote a temporary artifact before rooted OpenFile: entries=%v error=%v", originalEntries, err)
	}
}

func TestAnchoredAtomicWriteDoesNotFollowInRootAliasAtPreOpenBoundary(t *testing.T) {
	rootPath := tempRoot(t)
	parent := filepath.Join(rootPath, "target")
	sibling := filepath.Join(rootPath, "sibling")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(sibling, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	original := filepath.Join(rootPath, "target-original")
	var hookReached, injected bool
	var injectionErr error
	err = anchored.atomicWriteWithHook("target/escaped.txt", []byte("must not write\n"), 0644, func() error {
		hookReached = true
		if err := os.Rename(parent, original); err != nil {
			injectionErr = err
			return err
		}
		if err := os.Symlink(sibling, parent); err != nil {
			injectionErr = err
			return err
		}
		injected = true
		return nil
	})
	if !hookReached {
		t.Fatalf("pre-open race hook did not execute: %v", err)
	}
	if !injected {
		t.Skipf("in-root alias swap could not be injected: %v", injectionErr)
	}
	if err == nil {
		t.Fatal("rooted writer accepted a parent identity change during temporary creation")
	}
	entries, err := os.ReadDir(sibling)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel.txt" {
		t.Fatalf("in-root sibling received artifact/temp writes: entries=%v error=%v", entries, err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("in-root sibling sentinel changed: bytes=%q error=%v", got, err)
	}
	originalEntries, err := os.ReadDir(original)
	if err != nil || len(originalEntries) != 0 {
		t.Fatalf("temporary artifact remained in pinned original parent: entries=%v error=%v", originalEntries, err)
	}
}

func TestAnchoredAtomicWriteRejectsInRootAliasAtPreRenameBoundary(t *testing.T) {
	rootPath := tempRoot(t)
	parent := filepath.Join(rootPath, "target")
	sibling := filepath.Join(rootPath, "sibling")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(sibling, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	original := filepath.Join(rootPath, "target-original")
	var hookReached, injected bool
	var injectionErr error
	err = anchored.atomicWriteWithHooks("target/escaped.txt", []byte("must not write\n"), 0644, nil, func() error {
		hookReached = true
		if err := os.Rename(parent, original); err != nil {
			injectionErr = err
			return err
		}
		if err := os.Symlink(sibling, parent); err != nil {
			injectionErr = err
			return err
		}
		injected = true
		return nil
	})
	if !hookReached {
		t.Fatalf("pre-rename race hook did not execute: %v", err)
	}
	if !injected {
		t.Skipf("in-root alias swap could not be injected: %v", injectionErr)
	}
	var published *publishedWriteError
	if !errors.As(err, &published) || published.Path != "target/escaped.txt" {
		t.Fatalf("expected a retained-publication error after destination parent identity changed, got %v", err)
	}
	entries, err := os.ReadDir(sibling)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel.txt" {
		t.Fatalf("in-root sibling received artifact/temp writes: entries=%v error=%v", entries, err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("in-root sibling sentinel changed: bytes=%q error=%v", got, err)
	}
	originalOutput := filepath.Join(original, "escaped.txt")
	got, err = os.ReadFile(originalOutput)
	if err != nil || string(got) != "must not write\n" {
		t.Fatalf("published bytes were rolled back or lost from the pinned original parent: bytes=%q error=%v", got, err)
	}
}

func TestLockWriterReleasePreservesReplacementLock(t *testing.T) {
	rootPath, err := os.MkdirTemp(os.TempDir(), "markitect-lock-replacement-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(rootPath)
	lockDir := filepath.Join(rootPath, ".artifacts", "markitect")
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		t.Fatal(err)
	}
	root, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	release, err := root.LockWriter()
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(lockDir, "write.lock")
	if err := os.Remove(lock); err != nil {
		t.Skipf("platform prevents replacing an open lock file: %v", err)
	}
	if err := os.WriteFile(lock, []byte("replacement lock\n"), 0600); err != nil {
		t.Fatal(err)
	}
	release()
	got, err := os.ReadFile(lock)
	if err != nil || string(got) != "replacement lock\n" {
		t.Fatalf("release removed or changed replacement lock: bytes=%q error=%v", got, err)
	}
}

func TestWriteRootReadRejectsLeafIdentitySwapBeforeOpen(t *testing.T) {
	rootPath := tempRoot(t)
	target := filepath.Join(rootPath, "selected.txt")
	sibling := filepath.Join(rootPath, "sibling.txt")
	if err := os.WriteFile(target, []byte("selected bytes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sibling, []byte("sibling secret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	original := filepath.Join(rootPath, "selected-original.txt")
	var hookReached, injected bool
	var injectionErr error
	data, err := anchored.readFileWithHooks("selected.txt", func() error {
		hookReached = true
		if err := os.Rename(target, original); err != nil {
			injectionErr = err
			return err
		}
		if err := os.Link(sibling, target); err != nil {
			injectionErr = err
			return err
		}
		injected = true
		return nil
	}, nil)
	if !hookReached {
		t.Fatalf("pre-open leaf-swap hook did not execute: %v", err)
	}
	if !injected {
		t.Skipf("leaf identity swap could not be injected: %v", injectionErr)
	}
	if err == nil || data != nil {
		t.Fatalf("read accepted bytes after the selected leaf identity changed: data=%q error=%v", data, err)
	}
}

func TestWriteRootReadRejectsLeafIdentitySwapBeforeAccept(t *testing.T) {
	rootPath := tempRoot(t)
	target := filepath.Join(rootPath, "selected.txt")
	sibling := filepath.Join(rootPath, "sibling.txt")
	if err := os.WriteFile(target, []byte("selected bytes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sibling, []byte("sibling secret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	original := filepath.Join(rootPath, "selected-original.txt")
	var hookReached, injected bool
	var injectionErr error
	data, err := anchored.readFileWithHooks("selected.txt", nil, func() error {
		hookReached = true
		if err := os.Rename(target, original); err != nil {
			injectionErr = err
			return err
		}
		if err := os.Link(sibling, target); err != nil {
			injectionErr = err
			return err
		}
		injected = true
		return nil
	})
	if !hookReached {
		t.Fatalf("pre-accept leaf-swap hook did not execute: %v", err)
	}
	if !injected {
		t.Skipf("leaf identity swap could not be injected: %v", injectionErr)
	}
	if err == nil || data != nil {
		t.Fatalf("read accepted bytes after the selected leaf identity changed: data=%q error=%v", data, err)
	}
}

func TestWriteRootReadRejectsAppendAtAcceptanceBoundary(t *testing.T) {
	rootPath, err := os.MkdirTemp(os.TempDir(), "markitect-read-append-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(rootPath)
	target := filepath.Join(rootPath, "selected.txt")
	if err := os.WriteFile(target, []byte("selected bytes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	root, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	var injected bool
	data, err := root.readFileWithHooks("selected.txt", nil, func() error {
		f, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.Write([]byte("late append\n")); err != nil {
			return err
		}
		injected = true
		return nil
	})
	if !injected {
		t.Skipf("file append could not be injected at acceptance boundary: %v", err)
	}
	if err == nil || data != nil {
		t.Fatalf("read accepted changed-size contents: data=%q error=%v", data, err)
	}
}

func TestAnchoredAtomicWriteRetainsRootLevelPublicationAfterRootMove(t *testing.T) {
	base, err := os.MkdirTemp(os.TempDir(), "markitect-root-level-publication-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)
	rootPath := filepath.Join(base, "repo")
	if err := os.Mkdir(rootPath, 0755); err != nil {
		t.Fatal(err)
	}
	root, err := openWriteRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	original := filepath.Join(base, "repo-original")
	var moved bool
	err = root.atomicWriteWithHooks("escaped.txt", []byte("published\n"), 0644, nil, func() error {
		if err := os.Rename(rootPath, original); err != nil {
			return err
		}
		moved = true
		return nil
	})
	if !moved {
		t.Skipf("platform prevents moving the open root directory: %v", err)
	}
	var published *publishedWriteError
	if !errors.As(err, &published) || published.Path != "escaped.txt" {
		t.Fatalf("expected retained root-level publication after root move, got %v", err)
	}
	got, err := os.ReadFile(filepath.Join(original, "escaped.txt"))
	if err != nil || string(got) != "published\n" {
		t.Fatalf("root-level published bytes were removed or lost: bytes=%q error=%v", got, err)
	}
}
