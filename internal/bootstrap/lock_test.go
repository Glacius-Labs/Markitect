package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestStateLockAdmitsOneHolderAtATime(t *testing.T) {
	state := filepath.Join(testkit.TempDir(t), "state", "nested")
	first, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(state, lockFileName)); err != nil || !info.Mode().IsRegular() {
		t.Fatal("lock file missing", err)
	}
	if second, err := lockState(state); !errors.Is(err, errStateLocked) || second != nil {
		t.Fatalf("second Apply took a held state directory: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	// A released lock leaves its file behind; it must not count as held.
	again, err := lockState(state)
	if err != nil {
		t.Fatalf("released lock stayed held: %v", err)
	}
	again.Release()
}

func TestStateLocksOfDifferentDirectoriesAreIndependent(t *testing.T) {
	root := testkit.TempDir(t)
	a, err := lockState(filepath.Join(root, "a"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Release()
	b, err := lockState(filepath.Join(root, "b"))
	if err != nil {
		t.Fatal(err)
	}
	b.Release()
}
