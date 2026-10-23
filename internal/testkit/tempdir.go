package testkit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// cleanupDeadline bounds how long cleanup retries a directory that another
// process still holds or writes to, such as a git process that has not exited
// yet or a Windows virus scanner.
const cleanupDeadline = 10 * time.Second

// TempDir creates a short temporary directory and removes it when the test
// ends. Unlike t.TempDir, the path does not contain the test name, so deep
// fixture trees stay below Windows path limits. The path is returned in its
// canonical spelling, without symbolic links or Windows short names, and
// removal is retried while files are still in use.
func TempDir(t testing.TB) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "mk")
	if err != nil {
		t.Fatalf("create temporary directory: %v", err)
	}
	t.Cleanup(func() {
		if err := removeAll(dir); err != nil {
			t.Errorf("remove temporary directory: %v", err)
		}
	})
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve temporary directory: %v", err)
	}
	return canonical
}

// removeAll removes path, retrying with backoff until cleanupDeadline when
// removal fails, and returns the last error.
func removeAll(path string) error {
	deadline := time.Now().Add(cleanupDeadline)
	delay := 10 * time.Millisecond
	for {
		err := os.RemoveAll(path)
		if err == nil || time.Now().After(deadline) {
			return err
		}
		time.Sleep(delay)
		delay = min(2*delay, 500*time.Millisecond)
	}
}
