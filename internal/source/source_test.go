package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadPinnedCommitIgnoresWorkingTreeChanges(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "-q")
	gitTest(t, root, "config", "user.email", "source-test@example.invalid")
	gitTest(t, root, "config", "user.name", "Source Test")
	writeTestFile(t, root, "keep.txt", "committed")
	writeTestFile(t, root, "delete.txt", "gone")
	writeTestFile(t, root, "vendor/generated.txt", "excluded")
	writeTestFile(t, root, ".worktrees/local/generated.txt", "excluded")
	gitTest(t, root, "add", "keep.txt", "delete.txt", "vendor/generated.txt", ".worktrees/local/generated.txt")
	gitTest(t, root, "commit", "-qm", "initial")
	revision := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))

	writeTestFile(t, root, "keep.txt", "dirty working copy")
	if err := os.Remove(filepath.Join(root, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "added.txt", "uncommitted")

	s, err := Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	if s.Provisional {
		t.Fatal("pinned snapshot marked provisional")
	}
	if s.Revision != revision {
		t.Fatalf("revision = %q, want full commit %q", s.Revision, revision)
	}
	if got := string(s.Files["keep.txt"]); got != "committed" {
		t.Fatalf("pinned content = %q, want committed blob", got)
	}
	if _, ok := s.Files["delete.txt"]; !ok {
		t.Fatal("pinned snapshot lost a committed file deleted from working tree")
	}
	if _, ok := s.Files["added.txt"]; ok {
		t.Fatal("pinned snapshot included an uncommitted file")
	}
	if _, ok := s.Files["vendor/generated.txt"]; ok {
		t.Fatal("pinned snapshot included an excluded directory")
	}
	if _, ok := s.Files[".worktrees/local/generated.txt"]; ok {
		t.Fatal("pinned snapshot included an excluded nested worktree")
	}
}

func TestLoadPinnedCommitIgnoresAmbientGitRepositoryOverrides(t *testing.T) {
	selected := t.TempDir()
	gitTest(t, selected, "init", "-q", "-b", "selected")
	gitTest(t, selected, "config", "user.email", "source-test@example.invalid")
	gitTest(t, selected, "config", "user.name", "Source Test")
	writeTestFile(t, selected, "selected.txt", "selected repository")
	gitTest(t, selected, "add", "selected.txt")
	gitTest(t, selected, "commit", "-qm", "selected")
	selectedRevision := strings.TrimSpace(gitTest(t, selected, "rev-parse", "HEAD"))

	foreign := t.TempDir()
	gitTest(t, foreign, "init", "-q", "-b", "foreign")
	gitTest(t, foreign, "config", "user.email", "source-test@example.invalid")
	gitTest(t, foreign, "config", "user.name", "Source Test")
	writeTestFile(t, foreign, "foreign.txt", "foreign repository")
	gitTest(t, foreign, "add", "foreign.txt")
	gitTest(t, foreign, "commit", "-qm", "foreign")
	foreignRevision := strings.TrimSpace(gitTest(t, foreign, "rev-parse", "HEAD"))
	gitTest(t, selected, "fetch", foreign, "foreign")
	gitTest(t, selected, "update-ref", "refs/replace/"+selectedRevision, foreignRevision)

	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))
	t.Setenv("GIT_WORK_TREE", foreign)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(foreign, ".git", "index"))
	t.Setenv("GIT_OBJECT_DIRECTORY", filepath.Join(foreign, ".git", "objects"))
	t.Setenv("GIT_CONFIG_PARAMETERS", "'core.repositoryformatversion=999'")

	snapshot, err := Load(selected, selectedRevision)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Revision != selectedRevision {
		t.Fatalf("revision = %q, want selected repo revision %q", snapshot.Revision, selectedRevision)
	}
	if got := string(snapshot.Files["selected.txt"]); got != "selected repository" {
		t.Fatalf("selected blob = %q, want selected repository contents", got)
	}
	if _, ok := snapshot.Files["foreign.txt"]; ok {
		t.Fatal("pinned snapshot was redirected to the ambient repository")
	}
}

func TestCleanGitEnvDropsGitVariablesCaseInsensitively(t *testing.T) {
	t.Setenv("gIt_DiR", "foreign")
	t.Setenv("GIT_CONFIG_PARAMETERS", "injected")
	t.Setenv("MARKITECT_TEST_VALUE", "preserved")
	values := make(map[string]string)
	for _, entry := range CleanGitEnv() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[strings.ToUpper(key)] = value
		}
	}
	for key := range values {
		if strings.HasPrefix(key, "GIT_") {
			t.Errorf("Git variable %s survived environment cleanup", key)
		}
	}
	if values["MARKITECT_TEST_VALUE"] != "preserved" {
		t.Fatal("non-Git environment variable was not preserved")
	}
}

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

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestUnsafePathsRejectedBeforeMaterialize(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "escape.txt")
	s := &Snapshot{
		Files: map[string][]byte{"../escape.txt": []byte("no")},
		Modes: map[string]string{"../escape.txt": "100644"},
	}
	err := Materialize(s, filepath.Join(t.TempDir(), "destination"))
	if err == nil || !strings.Contains(err.Error(), "unsafe repository path") {
		t.Fatalf("Materialize error = %v, want unsafe-path rejection", err)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("escaped file unexpectedly exists (stat error: %v)", err)
	}
}

func TestValidateRepoPathRejectsWindowsUnsafeNames(t *testing.T) {
	for _, p := range []string{"C:/outside", "a\\b", "CON.txt", "nested/NUL", "trailing.", "trailing ", "../x", "a/./b"} {
		if err := validateRepoPath(p); err == nil {
			t.Errorf("validateRepoPath(%q) succeeded", p)
		}
	}
}

func TestValidatePortablePathsRejectsCaseCollisions(t *testing.T) {
	for _, paths := range [][]string{
		{"README.md", "Readme.md"},
		{"Area/one.txt", "area/two.txt"},
		{"config", "Config/nested.yaml"},
	} {
		if err := validatePortablePaths(paths); err == nil || !strings.Contains(err.Error(), "case-insensitive path collision") {
			t.Errorf("validatePortablePaths(%q) error = %v", paths, err)
		}
	}
	if err := validatePortablePaths([]string{"Area/one.txt", "Area/two.txt"}); err != nil {
		t.Fatalf("valid shared directory rejected: %v", err)
	}
}

func TestValidateIncludedPathsAppliesSnapshotBoundaries(t *testing.T) {
	for _, p := range []string{".artifacts/report/README.md", ".ARTIFACTS/report/README.md", "vendor/pkg/README.md", "app/node_modules/README.md"} {
		if err := ValidateIncludedPaths([]string{p}); err == nil || !strings.Contains(err.Error(), "excluded") {
			t.Errorf("ValidateIncludedPaths(%q) error = %v, want exclusion", p, err)
		}
	}
	for _, paths := range [][]string{
		{"docs/area/README.md", "docs/Area/other.md"},
		{"markitect.yaml", "Markitect.yaml"},
	} {
		if err := ValidateIncludedPaths(paths); err == nil || !strings.Contains(err.Error(), "case-insensitive path collision") {
			t.Errorf("ValidateIncludedPaths(%q) error = %v, want portable collision", paths, err)
		}
	}
	if err := ValidateIncludedPaths([]string{"docs/area/README.md", "markitect.yaml"}); err != nil {
		t.Fatalf("valid prospective paths rejected: %v", err)
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

func gitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = CleanGitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
