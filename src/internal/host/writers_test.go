package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func initWriterRepo(t *testing.T, root string) {
	t.Helper()
	runWriterGit(t, root, "init", "-b", "feature/test-writes")
	runWriterGit(t, root, "config", "user.name", "Markitect Test")
	runWriterGit(t, root, "config", "user.email", "markitect-test@example.invalid")
}

func runWriterGit(t *testing.T, root string, args ...string) string {
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

func TestWriteSchemasRefusesUnmanagedTargetAndParentTraversal(t *testing.T) {
	t.Run("unmanaged target", func(t *testing.T) {
		root := tempRoot(t)
		target := filepath.Join(root, "schema/v1alpha1.json")
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		original := []byte("hand-authored schema\n")
		if err := os.WriteFile(target, original, 0644); err != nil {
			t.Fatal(err)
		}
		if err := WriteSchemas(root, map[string][]byte{"schema/v1alpha1.json": []byte("generated\n")}); err == nil || !strings.Contains(err.Error(), "unmanaged schema output") {
			t.Fatalf("WriteSchemas error = %v, want unmanaged-target refusal", err)
		}
		actual, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if string(actual) != string(original) {
			t.Fatal("unmanaged schema target was changed")
		}
	})

	t.Run("parent traversal", func(t *testing.T) {
		root := tempRoot(t)
		if err := WriteSchemas(root, map[string][]byte{"schema/../outside.json": []byte("escaped\n")}); err == nil || !strings.Contains(err.Error(), "unsafe output path") {
			t.Fatalf("WriteSchemas error = %v, want traversal refusal", err)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(root), "outside.json")); !os.IsNotExist(err) {
			t.Fatalf("schema write escaped its root: stat error = %v", err)
		}
	})
}

func TestWriteSchemasRefusesSymlinkPath(t *testing.T) {
	root := tempRoot(t)
	outside := tempRoot(t)
	if err := os.Symlink(outside, filepath.Join(root, "schema")); err != nil {
		t.Skipf("symlink creation is unavailable in this environment: %v", err)
	}
	err := WriteSchemas(root, map[string][]byte{"schema/v1alpha1.json": []byte("generated\n")})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("WriteSchemas error = %v, want symlink-path refusal", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "v1alpha1.json")); !os.IsNotExist(err) {
		t.Fatalf("schema write followed a symlink: stat error = %v", err)
	}
}
