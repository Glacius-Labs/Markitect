//go:build unix

package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestStateHeldProbesWithoutCreatingAnythingOrBlockingAStart(t *testing.T) {
	state := filepath.Join(testkit.TempDir(t), "state")
	if held, err := stateHeld(state); err != nil || held {
		t.Fatalf("missing state directory: %v %v", held, err)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("the probe created the state directory")
	}
	lock, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	if held, err := stateHeld(state); err != nil || !held {
		t.Fatalf("held state: %v %v", held, err)
	}
	lock.Release()
	if held, err := stateHeld(state); err != nil || held {
		t.Fatalf("released state: %v %v", held, err)
	}
	again, err := lockState(state)
	if err != nil {
		t.Fatalf("the probe left the lock held: %v", err)
	}
	again.Release()
}

func TestLockStateWithinWaitsForABriefHolderAndGivesUpOtherwise(t *testing.T) {
	state := testkit.TempDir(t)
	held, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		held.Release()
	}()
	lock, err := lockStateWithin(context.Background(), state, 5*time.Second)
	if err != nil {
		t.Fatalf("a brief holder made the start fail: %v", err)
	}
	defer lock.Release()
	if _, err := lockStateWithin(context.Background(), state, 50*time.Millisecond); !errors.Is(err, errStateLocked) {
		t.Fatalf("a lasting holder was not reported: %v", err)
	}
}
