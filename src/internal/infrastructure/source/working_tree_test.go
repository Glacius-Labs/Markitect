package source

import (
	"os"
	"path/filepath"
	"reflect"
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
