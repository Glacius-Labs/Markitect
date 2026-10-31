package bootstrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// The desktop creates, opens and clones repositories with the host's Git and
// sets a project up with the command line's own init.
func TestTheDesktopWorksOnRealRepositories(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(testkit.TempDir(t), "library")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := (desktopRepos{}).Init(ctx, dir); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	ok, err := desktopCommands{}.Init(ctx, dir, &out)
	if err != nil || !ok {
		t.Fatalf("init: %v %v\n%s", ok, err, out.String())
	}
	if !strings.Contains(out.String(), "created .markitect/manifest.yaml") {
		t.Errorf("init output:\n%s", out.String())
	}
	if code, checked, errOut := run(t, dir, "check"); code != 0 {
		t.Fatalf("check after the desktop's init: %d\n%s%s", code, checked, errOut)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	ws, err := desktopRepos{}.Open(ctx, filepath.Join(dir, "docs"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := filepath.EvalSymlinks(ws.Root()); got != mustEval(t, dir) {
		t.Errorf("root %s, want %s", ws.Root(), dir)
	}

	source := testkit.NewRepo(t)
	source.Write("README.md", "# Library\n")
	source.Commit("start")
	clone := filepath.Join(testkit.TempDir(t), "copy")
	if err := (desktopRepos{}).Clone(ctx, source.Dir, clone); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(clone, "README.md")); err != nil {
		t.Errorf("clone: %v", err)
	}
	if err := (desktopRepos{}).Clone(ctx, "--upload-pack=x", filepath.Join(testkit.TempDir(t), "x")); err == nil {
		t.Error("an option passed as the URL")
	}
	failed := filepath.Join(testkit.TempDir(t), "failed")
	if err := (desktopRepos{}).Clone(ctx, filepath.Join(testkit.TempDir(t), "missing"), failed); err == nil {
		t.Error("cloned a repository that does not exist")
	}
	if _, err := os.Stat(failed); !os.IsNotExist(err) {
		t.Errorf("a failed clone left %s: %v", failed, err)
	}
	if err := (desktopRepos{}).Clone(ctx, source.Dir, clone); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("clone into an existing folder: %v", err)
	}
}

func mustEval(t *testing.T, path string) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
