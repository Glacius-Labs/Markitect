package run

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// A lock is held while its process lives, and taken over when the process has
// gone, the file stayed empty too long, or it is older than any shell call.
func TestLocksAreTakenOverWhenStale(t *testing.T) {
	dir := testkit.TempDir(t)
	release, err := lock(dir, "w1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock(dir, "w1", 0); err != ErrBusy {
		t.Fatalf("a held lock was taken: %v", err)
	}
	release()
	if release, err = lock(dir, "w1", 0); err != nil {
		t.Fatalf("a released lock was not taken: %v", err)
	}
	release()

	stale := func(name string, content string, age time.Duration) {
		t.Helper()
		file := filepath.Join(dir, name+".lock")
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-age)
		if err := os.Chtimes(file, at, at); err != nil {
			t.Fatal(err)
		}
		release, err := lock(dir, name, 0)
		if err != nil {
			t.Fatalf("%s was not taken over: %v", name, err)
		}
		release()
	}
	stale("gone", "999999999\n", 0)
	stale("empty", "", time.Hour)
	stale("ancient", fmt.Sprintf("%d\n", os.Getpid()), 3*time.Hour)
	// A live process's lock counts.
	live := filepath.Join(dir, "live.lock")
	if err := os.WriteFile(live, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := lock(dir, "live", 0); err != ErrBusy {
		t.Fatalf("a live lock was taken: %v", err)
	}
}

// A changed link is refused when it can leave the folder, on any host.
func TestLeavesTree(t *testing.T) {
	cases := []struct {
		to   string
		want bool
	}{
		{"limit.md", false},
		{"sub/limit.md", false},
		{"../limit.md", false},
		{"./limit.md", false},
		{"../../limit.md", true},
		{"sub/../../limit.md", true},
		{"/dev/shm/good.md", true},
		{"//server/share/good.md", true},
		{`..\..\good.md`, true},
		{"C:/Users/you/.ssh/id_rsa", true},
		{"c:good.md", true},
	}
	for _, c := range cases {
		if got := leavesTree("docs/x.md", c.to); got != c.want {
			t.Errorf("a link to %s: leaves the tree %v, want %v", c.to, got, c.want)
		}
	}
}
