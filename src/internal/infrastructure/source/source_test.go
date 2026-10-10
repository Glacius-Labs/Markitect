package source

import (
	"os"
	"path/filepath"
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
	if s.ID != revision {
		t.Fatalf("snapshot ID = %q, want full commit %q", s.ID, revision)
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
	if snapshot.ID != selectedRevision {
		t.Fatalf("snapshot ID = %q, want selected repo revision %q", snapshot.ID, selectedRevision)
	}
	if got := string(snapshot.Files["selected.txt"]); got != "selected repository" {
		t.Fatalf("selected blob = %q, want selected repository contents", got)
	}
	if _, ok := snapshot.Files["foreign.txt"]; ok {
		t.Fatal("pinned snapshot was redirected to the ambient repository")
	}
}

func TestLoadPinnedCommitSkipsGitlinksInExcludedDirectories(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	gitTest(t, root, "update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("a", 40)+",vendor/lib")
	gitTest(t, root, "update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("b", 40)+",node_modules")
	gitTest(t, root, "commit", "-qm", "excluded gitlinks")

	working, err := Load(root, "")
	if err != nil {
		t.Fatalf("working Load: %v", err)
	}
	fixed, err := Load(root, "HEAD")
	if err != nil {
		t.Fatalf("Load(HEAD) with excluded gitlinks: %v", err)
	}
	for _, p := range []string{"vendor/lib", "node_modules"} {
		if _, ok := working.Files[p]; ok {
			t.Fatalf("working tree included excluded %s", p)
		}
		if _, ok := fixed.Files[p]; ok {
			t.Fatalf("pinned snapshot included excluded %s", p)
		}
	}
	if got := string(fixed.Files["seed.txt"]); got != "seed" {
		t.Fatalf("pinned seed.txt = %q, want committed blob", got)
	}
}

func TestLoadPinnedCommitRejectsIncludedGitlinkAsSubmodule(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	gitTest(t, root, "update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("a", 40)+",lib")
	gitTest(t, root, "commit", "-qm", "gitlink")
	_, err := Load(root, "HEAD")
	if err == nil || !strings.Contains(err.Error(), "submodule") {
		t.Fatalf("Load(HEAD) error = %v, want submodule rejection", err)
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
