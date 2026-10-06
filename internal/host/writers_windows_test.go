//go:build windows

package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestSafeDestinationAcceptsShortPathSpelling(t *testing.T) {
	root, shortRoot := windowsTestPath(t)
	if _, err := safeDestination(shortRoot, "output/new.md"); err != nil {
		t.Fatalf("safeDestination rejected valid short root %q: %v", shortRoot, err)
	}
	if _, err := safeDestination(root, "output/new.md"); err != nil {
		t.Fatalf("safeDestination rejected long root %q: %v", root, err)
	}
}

func TestSafeDestinationRetainsReparseRejectionWithShortPath(t *testing.T) {
	root, shortRoot := windowsTestPath(t)
	outside, err := os.MkdirTemp(os.TempDir(), "markitect-writer-outside-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outside)
	link := filepath.Join(root, "linked")
	if err := makeWindowsJunction(link, outside); err != nil {
		t.Skipf("directory reparse point creation unavailable: %v", err)
	}
	defer os.Remove(link)
	if _, err := safeDestination(shortRoot, "linked/output.md"); err == nil || (!strings.Contains(strings.ToLower(err.Error()), "symlink") && !strings.Contains(strings.ToLower(err.Error()), "reparse point")) {
		t.Fatalf("safeDestination through short reparse path was accepted: %v", err)
	}
}

func TestAnchoredAtomicWriteRefusesSwappedJunctionParent(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "markitect-rooted-junction-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	parent := filepath.Join(root, "target")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	outside, err := os.MkdirTemp(os.TempDir(), "markitect-rooted-outside-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outside)
	sentinel := filepath.Join(outside, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	if err := os.Rename(parent, filepath.Join(root, "target-original")); err != nil {
		t.Fatal(err)
	}
	if err := makeWindowsJunction(parent, outside); err != nil {
		t.Skipf("directory junction creation unavailable: %v", err)
	}
	if err := anchored.AtomicWrite("target/escaped.txt", []byte("must not write\n"), 0644); err == nil {
		t.Fatal("anchored writer accepted a swapped junction parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "escaped.txt")); !os.IsNotExist(err) {
		t.Fatalf("write escaped through replacement junction: %v", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("external sentinel changed: bytes=%q error=%v", got, err)
	}
}

func TestAnchoredAtomicWriteRefusesReplacedRootJunction(t *testing.T) {
	base, err := os.MkdirTemp(os.TempDir(), "markitect-rooted-root-swap-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)
	scope := filepath.Join(base, "scope")
	root := filepath.Join(scope, "repo")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	outside, err := os.MkdirTemp(os.TempDir(), "markitect-rooted-root-outside-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outside)
	replacement := filepath.Join(outside, "repo")
	if err := os.Mkdir(replacement, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(replacement, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	if err := os.Rename(scope, filepath.Join(base, "scope-original")); err != nil {
		t.Skipf("platform prevents swapping an ancestor of an open root: %v", err)
	}
	if err := makeWindowsJunction(scope, outside); err != nil {
		t.Skipf("directory junction creation unavailable: %v", err)
	}
	if err := anchored.AtomicWrite("escaped.txt", []byte("must not write\n"), 0644); err == nil {
		t.Fatal("anchored writer accepted a root replaced through a junction ancestor")
	}
	entries, err := os.ReadDir(replacement)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel.txt" {
		t.Fatalf("replacement root received artifact/temp writes: entries=%v error=%v", entries, err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("replacement root sentinel changed: bytes=%q error=%v", got, err)
	}
}

func TestAnchoredAtomicWriteContainsJunctionSwapAtPreOpenBoundary(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "markitect-rooted-race-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	parent := filepath.Join(root, "target")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	outside, err := os.MkdirTemp(os.TempDir(), "markitect-rooted-outside-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outside)
	sentinel := filepath.Join(outside, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	anchored, err := openWriteRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	original := filepath.Join(root, "target-original")
	var hookReached, injected bool
	var injectionErr error
	err = anchored.atomicWriteWithHook("target/escaped.txt", []byte("must not write\n"), 0644, func() error {
		hookReached = true
		if err := os.Rename(parent, original); err != nil {
			injectionErr = err
			return err
		}
		if err := makeWindowsJunction(parent, outside); err != nil {
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
		t.Skipf("directory junction swap could not be injected: %v", injectionErr)
	}
	if err == nil {
		t.Fatal("rooted OpenFile accepted a junction swapped after path and root identity validation")
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

func TestAnchoredAtomicWriteDoesNotFollowInRootJunctionAtPreOpenBoundary(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "markitect-rooted-internal-race-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	parent := filepath.Join(root, "target")
	sibling := filepath.Join(root, "sibling")
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
	anchored, err := openWriteRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	original := filepath.Join(root, "target-original")
	var hookReached, injected bool
	var injectionErr error
	err = anchored.atomicWriteWithHook("target/escaped.txt", []byte("must not write\n"), 0644, func() error {
		hookReached = true
		if err := os.Rename(parent, original); err != nil {
			injectionErr = err
			return err
		}
		if err := makeWindowsJunction(parent, sibling); err != nil {
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
		t.Skipf("in-root junction swap could not be injected: %v", injectionErr)
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

func TestSafeDestinationSupportsLongWindowsPaths(t *testing.T) {
	base, err := os.MkdirTemp(os.TempDir(), "markitect-long-path-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)
	root := filepath.Join(base, strings.Repeat("a", 100), strings.Repeat("b", 100), strings.Repeat("c", 100))
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Skipf("filesystem or Windows configuration does not permit long test paths: %v", err)
	}
	shortRoot, err := windowsShortPath(root)
	if err != nil {
		t.Skipf("GetShortPathNameW cannot represent the long test path: %v", err)
	}
	if _, err := safeDestination(shortRoot, "new.md"); err != nil {
		t.Fatalf("safeDestination rejected supported long path %q: %v", shortRoot, err)
	}
	if _, err := safeDestination(root, "new.md"); err != nil {
		t.Fatalf("safeDestination rejected supported long spelling %q: %v", root, err)
	}
}

func windowsTestPath(t *testing.T) (string, string) {
	t.Helper()
	root, err := os.MkdirTemp(os.TempDir(), "markitect-writer-long-path-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	shortRoot, err := windowsShortPath(root)
	if err != nil {
		t.Skipf("GetShortPathNameW is unavailable: %v", err)
	}
	if !strings.Contains(shortRoot, "~") {
		t.Skipf("filesystem did not provide an 8.3 spelling for %q", root)
	}
	return root, shortRoot
}

func windowsShortPath(longPath string) (string, error) {
	input, err := syscall.UTF16PtrFromString(longPathAPISpelling(longPath))
	if err != nil {
		return "", err
	}
	buffer := make([]uint16, 260)
	n, err := syscall.GetShortPathName(input, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", syscall.EINVAL
	}
	if int(n) >= len(buffer) {
		buffer = make([]uint16, int(n)+1)
		n, err = syscall.GetShortPathName(input, &buffer[0], uint32(len(buffer)))
		if err != nil {
			return "", err
		}
	}
	return syscall.UTF16ToString(buffer[:n]), nil
}

func makeWindowsJunction(link, target string) error {
	command := exec.Command("cmd.exe", "/c", "mklink", "/J", link, target)
	_, junctionErr := command.CombinedOutput()
	if junctionErr == nil {
		return nil
	}
	if err := os.Symlink(target, link); err == nil {
		return nil
	}
	return junctionErr
}
