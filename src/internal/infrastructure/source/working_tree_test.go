package source

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestWorkingTreeDigestTracksAddedAndDeletedFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "one.txt", "one")
	first, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "two.txt", "two")
	second, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() == second.Digest() {
		t.Fatal("adding a file did not change digest")
	}
	if err := os.Remove(filepath.Join(root, "two.txt")); err != nil {
		t.Fatal(err)
	}
	third, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if second.Digest() == third.Digest() {
		t.Fatal("deleting a file did not change digest")
	}
	if first.Digest() != third.Digest() {
		t.Fatal("restoring the original file set did not restore digest")
	}
}

func TestWorkingTreeExclusionsAndMaterialize(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/readme.md", "hello")
	writeTestFile(t, root, ".git", "gitdir: elsewhere")
	for _, dir := range []string{"vendor", "node_modules", "bin", "obj", ".artifacts", ".cache", ".worktrees", ".venv", "__pycache__"} {
		writeTestFile(t, root, filepath.Join(dir, "nested", "hidden.txt"), "ignored")
	}
	s, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Files) != 1 || string(s.Files["docs/readme.md"]) != "hello" {
		t.Fatalf("unexpected files in snapshot: %#v", s.Files)
	}

	dest := filepath.Join(t.TempDir(), "materialized")
	if err := Materialize(s, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "docs", "readme.md"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("materialized content = %q, err=%v", got, err)
	}
	if err := Materialize(s, dest); err == nil {
		t.Fatal("materializing into a nonempty directory succeeded")
	}
}

func TestProtectedDocumentationAndProviderTreesMatchAcrossSnapshotModes(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "-q")
	gitTest(t, root, "config", "user.email", "source-test@example.invalid")
	gitTest(t, root, "config", "user.name", "Source Test")
	visible := map[string]string{
		"docs/general/rules/vendor/policy.md": "policy",
		"docs/customers/x/obj/agent.md":       "customer agent",
		"docs/modules/node_modules/readme.md": "module docs",
		".agents/vendor/rules.md":             "agent rules",
		".claude/obj/config.md":               "claude config",
		".codex/bin/provider.md":              "codex config",
	}
	for p, content := range visible {
		writeTestFile(t, root, p, content)
	}
	for _, p := range []string{
		"vendor/dependency.go",
		".cache/build/index",
		".artifacts/output/report.yaml",
		"bin/generated/tool",
		"nested/obj/generated/cache.bin",
		"app/node_modules/pkg/index.js",
	} {
		writeTestFile(t, root, p, "excluded")
	}

	working, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range visible {
		if got := string(working.Files[p]); got != want {
			t.Errorf("working tree lost %q: got %q", p, got)
		}
	}
	for _, p := range []string{"vendor/dependency.go", ".cache/build/index", ".artifacts/output/report.yaml", "bin/generated/tool", "nested/obj/generated/cache.bin", "app/node_modules/pkg/index.js"} {
		if _, ok := working.Files[p]; ok {
			t.Errorf("working tree included excluded path %q", p)
		}
	}

	gitTest(t, root, "add", "-A")
	gitTest(t, root, "commit", "-qm", "initial")
	revision := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	fixed, err := Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(working.Files, fixed.Files) {
		t.Fatalf("fixed and working snapshots apply different inclusion policies\nworking=%v\nfixed=%v", mapKeys(working.Files), mapKeys(fixed.Files))
	}
}

func TestWorkingTreeCollisionDiagnosticDoesNotDependOnMapOrder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "Docs/a.md", "a")
	writeTestFile(t, root, "docs/b.md", "b")
	if _, err := os.Stat(filepath.Join(root, "Docs", "b.md")); err == nil {
		t.Skip("case-insensitive filesystem cannot hold both Docs and docs")
	}
	requireSameError(t, func() error { _, err := Load(root, ""); return err }, `between "Docs" and "docs"`)
}

func TestWorkingTreeTakesTrackedModesFromIndexWhenGitIgnoresFileMode(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	// Git for Windows sets this; setting it here also covers other platforms.
	gitTest(t, root, "config", "core.filemode", "false")
	writeTestFile(t, root, "tool.sh", "#!/bin/sh\n")
	gitTest(t, root, "add", "tool.sh")
	gitTest(t, root, "update-index", "--chmod=+x", "tool.sh")
	gitTest(t, root, "commit", "-qm", "executable tool")
	if status := strings.TrimSpace(gitTest(t, root, "status", "--porcelain")); status != "" {
		t.Fatalf("precondition: worktree not clean: %q", status)
	}
	working, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := Load(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if working.Modes["tool.sh"] != "100755" || working.Digest() != fixed.Digest() {
		t.Fatalf("clean worktree mode %s (digest %s) != HEAD mode %s (digest %s) for tool.sh", working.Modes["tool.sh"], working.Digest(), fixed.Modes["tool.sh"], fixed.Digest())
	}

	writeTestFile(t, root, "untracked.sh", "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(root, "untracked.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	withUntracked, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "100644"
	if runtime.GOOS != "windows" {
		want = "100755"
	}
	if withUntracked.Modes["untracked.sh"] != want || withUntracked.Modes["tool.sh"] != "100755" {
		t.Fatalf("modes = %#v, want untracked filesystem mode %s and tracked index mode 100755", withUntracked.Modes, want)
	}
}

func TestLoadRejectsWorkingTreeSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	writeTestFile(t, root, "target.txt", "data")
	if err := os.Symlink(target, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := Load(root, ""); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Load error = %v, want symlink rejection", err)
	}
}
