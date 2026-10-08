package host

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestGuardedWriteCapturesAndAppliesCreateReplaceAndDelete(t *testing.T) {
	root := installTestRepo(t, "feature/guarded")
	if err := os.WriteFile(filepath.Join(root, "delete.txt"), []byte("remove me\n"), 0644); err != nil {
		t.Fatal(err)
	}
	capture, err := CaptureGuardedWrite(root, []string{"README.md", "new/created.txt", "delete.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if capture.Head == "" || capture.Branch != "feature/guarded" || capture.Files["new/created.txt"].Exists {
		t.Fatalf("capture did not bind expected Git and missing-file state: %+v", capture)
	}
	result, err := ApplyGuardedWrite(root, capture, []GuardedWriteChange{
		{Path: "README.md", Bytes: []byte("updated\n"), Mode: 0644},
		{Path: "new/created.txt", Bytes: []byte("created\n"), Mode: 0755},
		{Path: "delete.txt", Delete: true},
	})
	if err != nil {
		t.Fatalf("ApplyGuardedWrite: %v", err)
	}
	if want := []string{"README.md", "delete.txt", "new/created.txt"}; !reflect.DeepEqual(result.CompletedPaths, want) {
		t.Fatalf("completed paths = %v, want %v", result.CompletedPaths, want)
	}
	if got := string(mustRead(t, filepath.Join(root, "README.md"))); got != "updated\n" {
		t.Fatalf("replacement = %q", got)
	}
	created := filepath.Join(root, "new", "created.txt")
	if got := string(mustRead(t, created)); got != "created\n" {
		t.Fatalf("created bytes = %q", got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(created)
		if err != nil || info.Mode().Perm() != 0755 {
			t.Fatalf("created mode = %v, err=%v; want 0755", info, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, "delete.txt")); !os.IsNotExist(err) {
		t.Fatalf("deleted path still exists or returned unexpected error: %v", err)
	}
}

func TestGuardedWriteRejectsStaleBytesAndNewFile(t *testing.T) {
	t.Run("changed selected bytes", func(t *testing.T) {
		root := installTestRepo(t, "feature/guarded")
		capture, err := CaptureGuardedWrite(root, []string{"README.md"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("concurrent edit\n"), 0644); err != nil {
			t.Fatal(err)
		}
		result, err := ApplyGuardedWrite(root, capture, []GuardedWriteChange{{Path: "README.md", Bytes: []byte("overwrite\n"), Mode: 0644}})
		if err == nil || !strings.Contains(err.Error(), "changed since capture") {
			t.Fatalf("ApplyGuardedWrite error = %v, want stale-byte refusal", err)
		}
		if len(result.CompletedPaths) != 0 || string(mustRead(t, filepath.Join(root, "README.md"))) != "concurrent edit\n" {
			t.Fatalf("stale apply changed selected file: result=%+v", result)
		}
	})

	t.Run("planned target appeared", func(t *testing.T) {
		root := installTestRepo(t, "feature/guarded")
		capture, err := CaptureGuardedWrite(root, []string{"new.txt"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("concurrent create\n"), 0644); err != nil {
			t.Fatal(err)
		}
		result, err := ApplyGuardedWrite(root, capture, []GuardedWriteChange{{Path: "new.txt", Bytes: []byte("must not replace\n"), Mode: 0644}})
		if err == nil || !strings.Contains(err.Error(), "changed since capture") {
			t.Fatalf("ApplyGuardedWrite error = %v, want appeared-target refusal", err)
		}
		if len(result.CompletedPaths) != 0 || string(mustRead(t, filepath.Join(root, "new.txt"))) != "concurrent create\n" {
			t.Fatalf("appeared target was replaced: result=%+v", result)
		}
	})
}

func TestGuardedWriteCapturesAndChecksModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file modes do not expose executable permission bits")
	}
	root := installTestRepo(t, "feature/guarded")
	path := filepath.Join(root, "mode.txt")
	if err := os.WriteFile(path, []byte("mode\n"), 0755); err != nil {
		t.Fatal(err)
	}
	capture, err := CaptureGuardedWrite(root, []string{"mode.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if capture.Files["mode.txt"].Mode.Perm() != 0755 {
		t.Fatalf("captured mode = %04o, want 0755", capture.Files["mode.txt"].Mode.Perm())
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ApplyGuardedWrite(root, capture, []GuardedWriteChange{{Path: "mode.txt", Bytes: []byte("new\n"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "changed since capture") {
		t.Fatalf("ApplyGuardedWrite error = %v, want stale-mode refusal", err)
	}
	if len(result.CompletedPaths) != 0 || string(mustRead(t, path)) != "mode\n" {
		t.Fatalf("stale mode capture changed file: result=%+v", result)
	}
}

func TestGuardedWriteRejectsBranchSwitch(t *testing.T) {
	root := installTestRepo(t, "feature/guarded")
	capture, err := CaptureGuardedWrite(root, []string{"README.md"})
	if err != nil {
		t.Fatal(err)
	}
	runWriterGit(t, root, "branch", "feature/other")
	runWriterGit(t, root, "checkout", "feature/other")
	result, err := ApplyGuardedWrite(root, capture, []GuardedWriteChange{{Path: "README.md", Bytes: []byte("must not write\n"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "branch changed") {
		t.Fatalf("ApplyGuardedWrite error = %v, want branch-switch refusal", err)
	}
	if len(result.CompletedPaths) != 0 || string(mustRead(t, filepath.Join(root, "README.md"))) != "test consumer\n" {
		t.Fatalf("branch-switched apply changed file: result=%+v", result)
	}
}

func TestGuardedWriteRejectsUnsafeAndUnselectedPaths(t *testing.T) {
	root := installTestRepo(t, "feature/guarded")
	for _, selected := range [][]string{{"../escape"}, {"a.txt", "A.txt"}, {`nested\\file.txt`}} {
		if _, err := CaptureGuardedWrite(root, selected); err == nil {
			t.Fatalf("CaptureGuardedWrite(%q) unexpectedly succeeded", selected)
		}
	}
	capture, err := CaptureGuardedWrite(root, []string{"README.md"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ApplyGuardedWrite(root, capture, []GuardedWriteChange{{Path: "unselected.txt", Bytes: []byte("escape selection"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "outside the captured selection") {
		t.Fatalf("ApplyGuardedWrite error = %v, want unselected-path refusal", err)
	}
	if len(result.CompletedPaths) != 0 {
		t.Fatalf("unselected change reported completed paths: %v", result.CompletedPaths)
	}
}

func TestGuardedWriteReportsPartialCompletion(t *testing.T) {
	root := installTestRepo(t, "feature/guarded")
	for name, contents := range map[string]string{"a.txt": "old a\n", "z.txt": "old z\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	capture, err := CaptureGuardedWrite(root, []string{"a.txt", "z.txt"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := applyGuardedWrite(root, capture, []GuardedWriteChange{
		{Path: "a.txt", Bytes: []byte("new a\n"), Mode: 0644},
		{Path: "z.txt", Bytes: []byte("new z\n"), Mode: 0644},
	}, func(path string) error {
		if path == "z.txt" {
			return os.WriteFile(filepath.Join(root, path), []byte("concurrent z\n"), 0644)
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "changed during guarded apply") {
		t.Fatalf("ApplyGuardedWrite error = %v, want partial stale-selection refusal", err)
	}
	if want := []string{"a.txt"}; !reflect.DeepEqual(result.CompletedPaths, want) {
		t.Fatalf("completed paths = %v, want %v", result.CompletedPaths, want)
	}
	if string(mustRead(t, filepath.Join(root, "a.txt"))) != "new a\n" || string(mustRead(t, filepath.Join(root, "z.txt"))) != "concurrent z\n" {
		t.Fatal("partial completion report does not match actual filesystem state")
	}
}
