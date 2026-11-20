package testkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTempDirIsShortAndCanonical(t *testing.T) {
	dir := TempDir(t)
	tempRoot, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// One short entry ("mk" and at most ten digits) directly below the
	// canonical spelling of the temporary directory.
	if parent, base := filepath.Dir(dir), filepath.Base(dir); parent != tempRoot || !strings.HasPrefix(base, "mk") || len(base) > 12 {
		t.Fatalf("TempDir = %q, want a short entry below %q", dir, tempRoot)
	}
}

func TestRemoveAllRetriesWhileAFileIsOpen(t *testing.T) {
	dir, err := os.MkdirTemp("", "mk")
	if err != nil {
		t.Fatal(err)
	}
	held, err := os.Create(filepath.Join(dir, "held"))
	if err != nil {
		t.Fatal(err)
	}
	// On Windows the open handle blocks removal until it is closed.
	go func() {
		time.Sleep(200 * time.Millisecond)
		held.Close()
	}()
	if err := removeAll(dir); err != nil {
		t.Fatalf("removeAll: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("directory still exists: %v", err)
	}
}
