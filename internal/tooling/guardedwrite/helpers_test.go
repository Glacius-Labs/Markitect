package guardedwrite

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
)

// testDirectory creates a directory outside the repository whose path has no
// symlinked or 8.3-spelled ancestors. A directory inside the checkout would
// appear as unaccounted working files to checks running in parallel.
func testDirectory(t *testing.T) string {
	t.Helper()
	return testkit.TempDir(t)
}

// testRepo returns a repository on branch with one committed README.md.
func testRepo(t *testing.T, branch string) string {
	t.Helper()
	root := testDirectory(t)
	runGit(t, root, "init", "-b", branch)
	runGit(t, root, "config", "user.name", "Markitect Test")
	runGit(t, root, "config", "user.email", "markitect-test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test consumer\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-m", "consumer baseline")
	return root
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	gitArgs := append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)
	command := exec.Command("git", gitArgs...)
	command.Env = source.CleanGitEnv()
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
