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
