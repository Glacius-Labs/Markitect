package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
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
	err = anchored.atomicWriteWithHook("target/escaped.txt", []byte("must not write\n"), 0644, func() error {
		if err := os.Rename(parent, original); err != nil {
			return err
		}
		return os.Symlink(outside, parent)
	})
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
	err = anchored.atomicWriteWithHook("target/escaped.txt", []byte("must not write\n"), 0644, func() error {
		if err := os.Rename(parent, original); err != nil {
			return err
		}
		return os.Symlink(sibling, parent)
	})
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
