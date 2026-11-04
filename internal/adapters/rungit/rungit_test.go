package rungit_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.invalid", "-c", "core.autocrlf=false"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, file, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// An uncommitted spec shows from any folder of the repository, with its whole
// path, and nothing else counts.
func TestUncommittedSpecShowsFromASubfolder(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	write(t, root, ".markitect/model/a.md", "a\n")
	write(t, root, "src/main.go", "package main\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "start")
	ctx := context.Background()
	if dirty, err := rungit.UncommittedSpec(ctx, root); err != nil || len(dirty) != 0 {
		t.Fatalf("a clean spec: %v, %v", dirty, err)
	}
	write(t, root, "src/main.go", "package main\n\n// changed\n")
	if dirty, err := rungit.UncommittedSpec(ctx, root); err != nil || len(dirty) != 0 {
		t.Fatalf("a change outside the spec: %v, %v", dirty, err)
	}
	write(t, root, ".markitect/model/a.md", "changed\n")
	write(t, root, ".markitect/model/b.md", "new\n")
	dirty, err := rungit.UncommittedSpec(ctx, filepath.Join(root, "src"))
	if err != nil || strings.Join(dirty, " ") != ".markitect/model/a.md .markitect/model/b.md" {
		t.Fatalf("from a subfolder: %q, %v", dirty, err)
	}
}

// Links names a commit's symbolic links and where they point.
func TestLinksReadsACommitsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	write(t, root, "a.txt", "a\n")
	git(t, root, "add", "-A")
	link := strings.TrimSpace(hashStdin(t, root, "/dev/shm/good.md"))
	git(t, root, "update-index", "--add", "--cacheinfo", "120000,"+link+",docs/limit.md")
	git(t, root, "commit", "-q", "-m", "a link")
	head := git(t, root, "rev-parse", "HEAD")
	repo, err := rungit.Open(context.Background(), filepath.Join(t.TempDir(), "host"), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Fetch(context.Background(), "main"); err != nil {
		t.Fatal(err)
	}
	links, err := repo.Links(context.Background(), head)
	if err != nil || len(links) != 1 || links["docs/limit.md"] != "/dev/shm/good.md" {
		t.Fatalf("%v, %v", links, err)
	}
}

func hashStdin(t *testing.T, dir, content string) string {
	t.Helper()
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
