package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallRefusesPartialPinSetAndUnmanagedCollision(t *testing.T) {
	t.Run("partial pin", func(t *testing.T) {
		root := installTestRepo(t, "feature/partial")
		writeFixture(t, root, map[string][]byte{"markitect.lock.yaml": []byte("hand-authored\n")})
		if _, err := Install(root, installTestBundle(t, "f", "0.1.0-rc.4", "candidate"), false); err == nil || !strings.Contains(err.Error(), "partial") {
			t.Fatalf("partial pin error = %v", err)
		}
		data, err := os.ReadFile(filepath.Join(root, "markitect.lock.yaml"))
		if err != nil || string(data) != "hand-authored\n" {
			t.Fatalf("partial pin changed: %q, %v", data, err)
		}
	})

	t.Run("ancestor collision", func(t *testing.T) {
		root := installTestRepo(t, "feature/collision")
		if err := os.WriteFile(filepath.Join(root, "scripts"), []byte("unmanaged file"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(root, installTestBundle(t, "1", "0.1.0-rc.4", "candidate"), false); err == nil {
			t.Fatal("unmanaged file at scripts ancestor was accepted")
		}
		data, err := os.ReadFile(filepath.Join(root, "scripts"))
		if err != nil || string(data) != "unmanaged file" {
			t.Fatalf("unmanaged ancestor changed: %q, %v", data, err)
		}
	})
}

func TestInstallRefusesModifiedBootstrapAndChangedPlanInputs(t *testing.T) {
	t.Run("modified committed bootstrap", func(t *testing.T) {
		root := installTestRepo(t, "feature/modified")
		bundle := installTestBundle(t, "2", "0.1.0-rc.3", "old")
		writeBundleToRoot(t, root, bundle, true)
		commitInstallPins(t, root, "install old release")
		bootstrap := filepath.Join(root, "scripts", "run-markitect.go")
		if err := os.WriteFile(bootstrap, []byte("modified bootstrap\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(root, installTestBundle(t, "3", "0.1.0-rc.4", "new"), false); err == nil || !strings.Contains(err.Error(), "unchanged committed") {
			t.Fatalf("modified bootstrap error = %v", err)
		}
	})

	t.Run("destination changes after plan", func(t *testing.T) {
		root := installTestRepo(t, "feature/concurrent-edit")
		bundle := installTestBundle(t, "4", "0.1.0-rc.4", "new")
		plan, err := Install(root, bundle, false)
		if err != nil || plan.Kind != "install" {
			t.Fatalf("initial plan = (%+v, %v)", plan, err)
		}
		writeFixture(t, root, map[string][]byte{"tools/markitect/release.yaml": []byte("unexpected\n")})
		if _, err := Install(root, bundle, true); err == nil {
			t.Fatal("install accepted target changes made after the plan")
		}
		if got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(releaseManifestPath))); err != nil || string(got) != "unexpected\n" {
			t.Fatalf("changed manifest was overwritten: %q, %v", got, err)
		}
	})
}

func TestInstallWriteRequiresNonProtectedBranchAndSharedLock(t *testing.T) {
	t.Run("protected branch", func(t *testing.T) {
		root := installTestRepo(t, "feature/protected")
		runWriterGit(t, root, "checkout", "-b", "main")
		if _, err := Install(root, installTestBundle(t, "5", "0.1.0-rc.4", "new"), true); err == nil || !strings.Contains(err.Error(), "non-protected") {
			t.Fatalf("protected branch error = %v", err)
		}
	})

	t.Run("switch to protected branch at same commit", func(t *testing.T) {
		root := installTestRepo(t, "feature/branch-race")
		bundle := installTestBundle(t, "5", "0.1.0-rc.4", "new")
		_, state, err := buildInstallPlan(root, bundle)
		if err != nil {
			t.Fatal(err)
		}
		state.branch = "feature/branch-race"
		before := state.head.ID
		runWriterGit(t, root, "branch", "main")
		runWriterGit(t, root, "checkout", "main")
		after := runWriterGit(t, root, "rev-parse", "HEAD")
		if after != before {
			t.Fatalf("test branch switch changed commit: before %s, after %s", before, after)
		}
		if err := ensureInstallStateUnchanged(root, state); err == nil || !strings.Contains(strings.ToLower(err.Error()), "branch") {
			t.Fatalf("same-SHA protected branch switch error = %v", err)
		}
	})

	t.Run("existing shared writer lock", func(t *testing.T) {
		root := installTestRepo(t, "feature/locked")
		lock := filepath.Join(root, ".artifacts", "markitect", "write.lock")
		if err := os.MkdirAll(filepath.Dir(lock), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lock, []byte("held\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(root, installTestBundle(t, "6", "0.1.0-rc.4", "new"), true); err == nil || !strings.Contains(err.Error(), "renderer lock is already present") {
			t.Fatalf("lock error = %v", err)
		}
	})
}

func TestInstallRejectsSymlinkedPinPath(t *testing.T) {
	root := installTestRepo(t, "feature/symlink")
	outside := tempRoot(t)
	if err := os.Symlink(outside, filepath.Join(root, "scripts")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := Install(root, installTestBundle(t, "7", "0.1.0-rc.4", "new"), false); err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("symlinked destination error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "run-markitect.go")); !os.IsNotExist(err) {
		t.Fatalf("installer wrote through symlink: %v", err)
	}
}
