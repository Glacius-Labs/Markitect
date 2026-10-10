package guardedwrite

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// testDirectory creates a directory under the package's ignored .cache so the
// path has no symlinked or 8.3-spelled ancestors.
func testDirectory(t *testing.T) string {
	t.Helper()
	if err := os.MkdirAll(".cache", 0755); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(".cache", "guardedwrite-test-")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeTestDirectory(abs); err != nil {
			t.Errorf("remove test directory: %v", err)
		}
	})
	return abs
}

// removeTestDirectory retries Windows access and sharing violations, which
// short-lived Git processes can cause while they release their handles.
func removeTestDirectory(path string) error {
	const (
		windowsAccessDenied     = syscall.Errno(5)
		windowsSharingViolation = syscall.Errno(32)
	)
	deadline := time.Now().Add(2 * time.Second)
	delay := time.Millisecond
	for {
		err := os.RemoveAll(path)
		if err == nil || runtime.GOOS != "windows" || (!errors.Is(err, windowsAccessDenied) && !errors.Is(err, windowsSharingViolation)) || time.Now().Add(delay).After(deadline) {
			return err
		}
		time.Sleep(delay)
		delay = min(2*delay, 100*time.Millisecond)
	}
}

// testRepo returns a repository on branch with one committed README.md.
func testRepo(t *testing.T, branch string) string {
	t.Helper()
	root := testDirectory(t)
	runGit(t, root, "init", "-b", branch)
	runGit(t, root, "config", "user.name", "Markitect Test")
	runGit(t, root, "config", "user.email", "markitect-test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test consumer\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-m", "consumer baseline")
	return root
}

func runGit(t *testing.T, root string, args ...string) string {
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

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
