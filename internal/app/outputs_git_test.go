package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestWriteOutputsUsesSelectedRepositoryForProtectedBranchGuard(t *testing.T) {
	root := tempRoot(t)
	gitInRepo(t, root, "init", "-b", "main")
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}

	foreign := t.TempDir()
	gitInRepo(t, foreign, "init", "-b", "feature/foreign")
	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))

	if _, err := WriteOutputs(root, p); err == nil || !strings.Contains(err.Error(), "non-protected branch") {
		t.Fatalf("WriteOutputs error = %v, want protected-branch refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/general/skills/entry.md")); !os.IsNotExist(err) {
		t.Fatalf("protected main branch received generated output (stat error: %v)", err)
	}
}

func TestWriteMigrationUsesSelectedRepositoryForProtectedBranchGuard(t *testing.T) {
	root := tempRoot(t)
	gitInRepo(t, root, "init", "-b", "main")
	writeFixture(t, root, map[string][]byte{"docs/general/notes.md": []byte("Legacy owner source.\n")})
	snapshot, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}

	foreign := t.TempDir()
	gitInRepo(t, foreign, "init", "-b", "feature/foreign")
	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))

	if _, err := WriteMigration(root, snapshot, migrationProject(t, "")); err == nil || !strings.Contains(err.Error(), "isolated non-protected Git branch") {
		t.Fatalf("WriteMigration error = %v, want protected-branch refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "markitect.yaml")); !os.IsNotExist(err) {
		t.Fatalf("migration wrote on protected main branch (stat error: %v)", err)
	}
}

func gitInRepo(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)...)
	command.Env = source.CleanGitEnv()
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
