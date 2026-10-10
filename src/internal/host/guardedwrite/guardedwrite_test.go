package guardedwrite

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestGuardedWriteCapturesAndAppliesCreateReplaceAndDelete(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	if err := os.WriteFile(filepath.Join(root, "delete.txt"), []byte("remove me\n"), 0644); err != nil {
		t.Fatal(err)
	}
	capture, err := CaptureFiles(root, []string{"README.md", "new/created.txt", "delete.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if capture.Head == "" || capture.Branch != "feature/guarded" || capture.Files["new/created.txt"].Exists {
		t.Fatalf("capture did not bind expected Git and missing-file state: %+v", capture)
	}
	result, err := Apply(root, capture, []Change{
		{Path: "README.md", Bytes: []byte("updated\n"), Mode: 0644},
		{Path: "new/created.txt", Bytes: []byte("created\n"), Mode: 0755},
		{Path: "delete.txt", Delete: true},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
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

func TestGuardedWriteRejectsDifferentRepositoryRoot(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	other := testRepo(t, "feature/guarded")
	capture, err := CaptureFiles(root, []string{"new.txt"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(other, capture, []Change{{Path: "new.txt", Bytes: []byte("must not write\n"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "root differs") || len(result.CompletedPaths) != 0 {
		t.Fatalf("different-root result=%+v error=%v", result, err)
	}
	for _, path := range []string{root, other} {
		if _, err := os.Lstat(filepath.Join(path, "new.txt")); !os.IsNotExist(err) {
			t.Fatalf("different-root attempt changed %s: %v", path, err)
		}
	}
}

func TestGuardedWriteCreatesBelowMissingMarkitectDirectories(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	capture, err := CaptureFiles(root, []string{".markitect/drafts/new.json"})
	if err != nil {
		t.Fatalf("capture under missing .markitect parent: %v", err)
	}
	if capture.Files[".markitect/drafts/new.json"].Exists {
		t.Fatal("capture unexpectedly found a file below the missing .markitect directory")
	}
	result, err := Apply(root, capture, []Change{{Path: ".markitect/drafts/new.json", Bytes: []byte("{\"draft\":true}\n"), Mode: 0644}})
	if err != nil {
		t.Fatalf("apply under missing .markitect parent: %v", err)
	}
	if !reflect.DeepEqual(result.CompletedPaths, []string{".markitect/drafts/new.json"}) {
		t.Fatalf("completed paths = %v", result.CompletedPaths)
	}
	if got := string(mustRead(t, filepath.Join(root, ".markitect", "drafts", "new.json"))); got != "{\"draft\":true}\n" {
		t.Fatalf("created bytes = %q", got)
	}
}

func TestGuardedWriteBindsUnbornBranchAndFirstCommitStalesCapture(t *testing.T) {
	root := testDirectory(t)
	runGit(t, root, "init", "-b", "codex/new")
	capture, err := CaptureFiles(root, []string{".markitect/drafts/new.json"})
	if err != nil {
		t.Fatalf("capture in a new repository: %v", err)
	}
	if capture.Head != "unborn:refs/heads/codex/new" {
		t.Fatalf("captured Head = %q, want explicit unborn branch sentinel", capture.Head)
	}
	result, err := Apply(root, capture, []Change{{Path: ".markitect/drafts/new.json", Bytes: []byte("draft\n"), Mode: 0644}})
	if err != nil || !reflect.DeepEqual(result.CompletedPaths, []string{".markitect/drafts/new.json"}) {
		t.Fatalf("apply in new repository: result=%+v err=%v", result, err)
	}
	stale, err := CaptureFiles(root, []string{"later.txt"})
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "-A")
	runGit(t, root, "-c", "user.name=Markitect Test", "-c", "user.email=markitect-test@example.invalid", "commit", "-m", "first commit")
	result, err = Apply(root, stale, []Change{{Path: "later.txt", Bytes: []byte("must not write\n"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "HEAD changed") {
		t.Fatalf("apply after first commit error = %v, want stale-unborn refusal", err)
	}
	if len(result.CompletedPaths) != 0 {
		t.Fatalf("stale unborn capture reported writes: %v", result.CompletedPaths)
	}
	if _, err := os.Lstat(filepath.Join(root, "later.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale unborn capture created later.txt: %v", err)
	}
}

func TestGuardedWriteRejectsExistingBranchRefWithMissingCommit(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	runGit(t, root, "update-ref", "-d", "refs/heads/feature/guarded")
	ref := filepath.Join(root, ".git", "refs", "heads", "feature", "guarded")
	if err := os.MkdirAll(filepath.Dir(ref), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ref, []byte(strings.Repeat("1", 40)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CaptureFiles(root, []string{"README.md"}); err == nil || !strings.Contains(err.Error(), "verify guarded unborn branch ref") {
		t.Fatalf("CaptureFiles error = %v, want broken existing-ref refusal", err)
	}
}

func TestGuardedWriteUsesProjectLockForMissingProjectManifest(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	capture, err := CaptureFiles(root, []string{".markitect/project.yaml", ".markitect/drafts/init.json"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ApplyChecked(root, capture, []Change{
		{Path: ".markitect/project.yaml", Bytes: []byte("kind: Project\n"), Mode: 0644},
		{Path: ".markitect/drafts/init.json", Bytes: []byte("{}\n"), Mode: 0644},
	}, func() error {
		if _, err := os.Stat(filepath.Join(root, ".markitect", "write.lock")); err != nil {
			return errors.New("project initialization did not hold the .markitect lock")
		}
		if _, err := os.Stat(filepath.Join(root, ".artifacts")); !os.IsNotExist(err) {
			return errors.New("project initialization created a legacy .artifacts directory")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ApplyChecked: %v", err)
	}
	if !reflect.DeepEqual(result.CompletedPaths, []string{".markitect/drafts/init.json", ".markitect/project.yaml"}) {
		t.Fatalf("completed paths = %v", result.CompletedPaths)
	}
	if _, err := os.Stat(filepath.Join(root, ".artifacts")); !os.IsNotExist(err) {
		t.Fatalf("project initialization left .artifacts outside Markitect: %v", err)
	}
}

func TestLegacyWriterUsesProjectLockAfterProjectManifestExists(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	manifest := filepath.Join(root, ".markitect", "project.yaml")
	if err := os.MkdirAll(filepath.Dir(manifest), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte("kind: Project\n"), 0644); err != nil {
		t.Fatal(err)
	}
	writer, err := OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	unlock, err := writer.LockWriter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "write.lock")); err != nil {
		t.Fatalf("project lock missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".artifacts")); !os.IsNotExist(err) {
		t.Fatalf("new project lock created legacy .artifacts directory: %v", err)
	}
	unlock()
}

func TestGuardedWriteRejectsStaleBytesAndNewFile(t *testing.T) {
	t.Run("changed selected bytes", func(t *testing.T) {
		root := testRepo(t, "feature/guarded")
		capture, err := CaptureFiles(root, []string{"README.md"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("concurrent edit\n"), 0644); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, capture, []Change{{Path: "README.md", Bytes: []byte("overwrite\n"), Mode: 0644}})
		if err == nil || !strings.Contains(err.Error(), "changed since capture") {
			t.Fatalf("Apply error = %v, want stale-byte refusal", err)
		}
		if len(result.CompletedPaths) != 0 || string(mustRead(t, filepath.Join(root, "README.md"))) != "concurrent edit\n" {
			t.Fatalf("stale apply changed selected file: result=%+v", result)
		}
	})

	t.Run("planned target appeared", func(t *testing.T) {
		root := testRepo(t, "feature/guarded")
		capture, err := CaptureFiles(root, []string{"new.txt"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("concurrent create\n"), 0644); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, capture, []Change{{Path: "new.txt", Bytes: []byte("must not replace\n"), Mode: 0644}})
		if err == nil || !strings.Contains(err.Error(), "changed since capture") {
			t.Fatalf("Apply error = %v, want appeared-target refusal", err)
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
	root := testRepo(t, "feature/guarded")
	path := filepath.Join(root, "mode.txt")
	if err := os.WriteFile(path, []byte("mode\n"), 0755); err != nil {
		t.Fatal(err)
	}
	capture, err := CaptureFiles(root, []string{"mode.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if capture.Files["mode.txt"].Mode.Perm() != 0755 {
		t.Fatalf("captured mode = %04o, want 0755", capture.Files["mode.txt"].Mode.Perm())
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	result, err := Apply(root, capture, []Change{{Path: "mode.txt", Bytes: []byte("new\n"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "changed since capture") {
		t.Fatalf("Apply error = %v, want stale-mode refusal", err)
	}
	if len(result.CompletedPaths) != 0 || string(mustRead(t, path)) != "mode\n" {
		t.Fatalf("stale mode capture changed file: result=%+v", result)
	}
}

func TestGuardedWriteRejectsBranchSwitch(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	capture, err := CaptureFiles(root, []string{"README.md"})
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "branch", "feature/other")
	runGit(t, root, "checkout", "feature/other")
	result, err := Apply(root, capture, []Change{{Path: "README.md", Bytes: []byte("must not write\n"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "branch changed") {
		t.Fatalf("Apply error = %v, want branch-switch refusal", err)
	}
	if len(result.CompletedPaths) != 0 || string(mustRead(t, filepath.Join(root, "README.md"))) != "test consumer\n" {
		t.Fatalf("branch-switched apply changed file: result=%+v", result)
	}
}

func TestGuardedWriteRejectsUnsafeAndUnselectedPaths(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	for _, selected := range [][]string{{"../escape"}, {"a.txt", "A.txt"}, {`nested\\file.txt`}} {
		if _, err := CaptureFiles(root, selected); err == nil {
			t.Fatalf("CaptureFiles(%q) unexpectedly succeeded", selected)
		}
	}
	capture, err := CaptureFiles(root, []string{"README.md"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(root, capture, []Change{{Path: "unselected.txt", Bytes: []byte("escape selection"), Mode: 0644}})
	if err == nil || !strings.Contains(err.Error(), "outside the captured selection") {
		t.Fatalf("Apply error = %v, want unselected-path refusal", err)
	}
	if len(result.CompletedPaths) != 0 {
		t.Fatalf("unselected change reported completed paths: %v", result.CompletedPaths)
	}
}

func TestGuardedWriteRejectsNonAdjacentPortableAliases(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	for _, selected := range [][]string{
		{"A.txt", "B.txt", "a.txt"},
		{"FileA/one.txt", "filea/two.txt"},
	} {
		if _, err := CaptureFiles(root, selected); err == nil {
			t.Fatalf("CaptureFiles(%q) accepted portable aliases", selected)
		}
	}
	for _, changes := range [][]Change{
		{{Path: "A.txt", Bytes: []byte("a"), Mode: 0644}, {Path: "B.txt", Bytes: []byte("b"), Mode: 0644}, {Path: "a.txt", Bytes: []byte("alias"), Mode: 0644}},
		{{Path: "FileA/one.txt", Bytes: []byte("a"), Mode: 0644}, {Path: "filea/two.txt", Bytes: []byte("alias"), Mode: 0644}},
	} {
		selected := make(map[string]File, len(changes))
		for _, change := range changes {
			selected[change.Path] = File{}
		}
		if _, err := normalizeGuardedChanges(changes, selected); err == nil {
			t.Fatalf("normalizeGuardedChanges(%v) accepted portable aliases", changes)
		}
	}
}

func TestGuardedWriteReportsPartialCompletion(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	for name, contents := range map[string]string{"a.txt": "old a\n", "z.txt": "old z\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	capture, err := CaptureFiles(root, []string{"a.txt", "z.txt"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := applyGuardedWrite(root, capture, []Change{
		{Path: "a.txt", Bytes: []byte("new a\n"), Mode: 0644},
		{Path: "z.txt", Bytes: []byte("new z\n"), Mode: 0644},
	}, nil, func(path string) error {
		if path == "z.txt" {
			return os.WriteFile(filepath.Join(root, path), []byte("concurrent z\n"), 0644)
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "changed during guarded apply") {
		t.Fatalf("Apply error = %v, want partial stale-selection refusal", err)
	}
	if want := []string{"a.txt"}; !reflect.DeepEqual(result.CompletedPaths, want) {
		t.Fatalf("completed paths = %v, want %v", result.CompletedPaths, want)
	}
	if string(mustRead(t, filepath.Join(root, "a.txt"))) != "new a\n" || string(mustRead(t, filepath.Join(root, "z.txt"))) != "concurrent z\n" {
		t.Fatal("partial completion report does not match actual filesystem state")
	}
}

func TestGuardedWriteCheckedRunsReadOnlyPreconditionUnderLock(t *testing.T) {
	root := testRepo(t, "feature/guarded")
	capture, err := CaptureFiles(root, []string{"README.md", "generated.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new-entry.md"), []byte("new inventory member\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ApplyChecked(root, capture, []Change{{Path: "generated.txt", Bytes: []byte("generated\n"), Mode: 0644}}, func() error {
		if _, err := os.Stat(filepath.Join(root, ".artifacts", "markitect", "write.lock")); err != nil {
			return errors.New("precondition did not run under the shared write lock")
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == "new-entry.md" {
				return errors.New("inventory membership changed since review")
			}
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "inventory membership changed") {
		t.Fatalf("ApplyChecked error = %v, want stale-inventory refusal", err)
	}
	if len(result.CompletedPaths) != 0 {
		t.Fatalf("failed precondition reported completed paths: %v", result.CompletedPaths)
	}
	if _, err := os.Lstat(filepath.Join(root, "generated.txt")); !os.IsNotExist(err) {
		t.Fatalf("write ran despite stale inventory: %v", err)
	}
}

func TestGuardedWriteCheckedRechecksCaptureAndSelectedBytesAfterCallback(t *testing.T) {
	t.Run("capture mutation", func(t *testing.T) {
		root := testRepo(t, "feature/guarded")
		capture, err := CaptureFiles(root, []string{"README.md"})
		if err != nil {
			t.Fatal(err)
		}
		result, err := ApplyChecked(root, capture, []Change{{Path: "README.md", Bytes: []byte("must not write\n"), Mode: 0644}}, func() error {
			capture.Files["README.md"] = File{Exists: false}
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "capture changed during precondition") {
			t.Fatalf("ApplyChecked error = %v, want capture-tampering refusal", err)
		}
		if len(result.CompletedPaths) != 0 || string(mustRead(t, filepath.Join(root, "README.md"))) != "test consumer\n" {
			t.Fatalf("changed capture applied a write: result=%+v", result)
		}
	})

	t.Run("selected bytes changed", func(t *testing.T) {
		root := testRepo(t, "feature/guarded")
		capture, err := CaptureFiles(root, []string{"README.md"})
		if err != nil {
			t.Fatal(err)
		}
		result, err := ApplyChecked(root, capture, []Change{{Path: "README.md", Bytes: []byte("must not overwrite\n"), Mode: 0644}}, func() error {
			return os.WriteFile(filepath.Join(root, "README.md"), []byte("callback side effect\n"), 0644)
		})
		if err == nil || !strings.Contains(err.Error(), "changed during precondition validation") {
			t.Fatalf("ApplyChecked error = %v, want selected-byte recheck refusal", err)
		}
		if len(result.CompletedPaths) != 0 || string(mustRead(t, filepath.Join(root, "README.md"))) != "callback side effect\n" {
			t.Fatalf("selected-byte callback side effect was overwritten: result=%+v", result)
		}
	})
}
