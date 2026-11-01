package repo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

func TestWorkingTreeAndCommit(t *testing.T) {
	ctx := context.Background()
	r := testkit.NewRepo(t)
	r.Write(".gitignore", "build/\n")
	r.Write("a.txt", "a\n")
	r.Write("gone.txt", "g\n")
	r.Write(".markitect/model/x.md", "x\n")
	base := r.Commit("base")

	r.Write("a.txt", "changed\n")
	r.Write("new.txt", "n\n")
	r.Write("build/out.bin", "ignored\n")
	r.Write(".markitect/model/x.md", "changed\n")
	if err := os.Remove(filepath.Join(r.Dir, "gone.txt")); err != nil {
		t.Fatal(err)
	}

	root, err := Toplevel(ctx, filepath.Join(r.Dir, ".markitect"))
	if err != nil {
		t.Fatal(err)
	}
	ws := WorkingTree{Dir: root, Ctx: ctx}
	files, err := ws.Files()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(files, ","); got != ".gitignore,.markitect/model/x.md,a.txt,new.txt" {
		t.Fatalf("working tree files = %s", got)
	}
	if !ws.GitIgnored("build/out.bin") || ws.GitIgnored("a.txt") {
		t.Fatal("GitIgnored is wrong")
	}

	c, err := ResolveCommit(ctx, root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if c.ID() != base {
		t.Fatalf("HEAD = %s, want %s", c.ID(), base)
	}
	data, err := c.Read("a.txt")
	if err != nil || string(data) != "a\n" {
		t.Fatalf("base a.txt = %q, %v", data, err)
	}
	changed, err := c.Changed()
	if err != nil {
		t.Fatal(err)
	}
	want := []analysis.Change{{Path: "a.txt", Change: analysis.Modified}, {Path: "gone.txt", Change: analysis.Removed}, {Path: "new.txt", Change: analysis.Added}}
	if len(changed) != len(want) {
		t.Fatalf("changed = %+v", changed)
	}
	for i := range want {
		if changed[i] != want[i] {
			t.Fatalf("changed = %+v", changed)
		}
	}
	if _, err := ResolveCommit(ctx, root, "no-such-rev"); err == nil {
		t.Fatal("an unknown revision resolved")
	}
	if _, err := Toplevel(ctx, testkit.TempDir(t)); err == nil {
		t.Fatal("a folder outside Git has a top level")
	}
}

func TestLastLandedIsTheNewestCommitThatAddsAResult(t *testing.T) {
	ctx := context.Background()
	r := testkit.NewRepo(t)
	r.Write("a.txt", "a\n")
	first := r.Commit("first")
	if _, ok, err := LastLanded(ctx, r.Dir, first); err != nil || ok {
		t.Fatalf("before any landing: %v %v", ok, err)
	}
	r.Write(".markitect/results/run-1.md", "# Result\n")
	landed := r.Commit("land run 1")
	r.Write(".markitect/results/run-1.md", "# Result, edited\n")
	r.Write("a.txt", "b\n")
	tip := r.Commit("later")
	if id, ok, err := LastLanded(ctx, r.Dir, tip); err != nil || !ok || id != landed {
		t.Fatalf("LastLanded = %s %v %v, want %s", id, ok, err, landed)
	}
}

func TestUncommittedListsOnlyMarkitectsFolder(t *testing.T) {
	ctx := context.Background()
	r := testkit.NewRepo(t)
	r.Write(".gitignore", ".markitect/cache/\n")
	r.Write(".markitect/manifest.yaml", "a\n")
	r.Write("a.txt", "a\n")
	// With nothing committed, every file counts.
	if files, err := Uncommitted(ctx, r.Dir); err != nil || len(files) != 1 || files[0] != ".markitect/manifest.yaml" {
		t.Fatalf("before the first commit: %v %v", files, err)
	}
	r.Commit("first")
	if files, err := Uncommitted(ctx, r.Dir); err != nil || len(files) != 0 {
		t.Fatalf("after the commit: %v %v", files, err)
	}
	r.Write(".markitect/manifest.yaml", "b\n")
	r.Write(".markitect/model/a b.md", "new\n")
	r.Write(".markitect/cache/x", "ignored\n")
	r.Write("a.txt", "b\n")
	files, err := Uncommitted(ctx, r.Dir)
	if err != nil || strings.Join(files, "|") != ".markitect/manifest.yaml|.markitect/model/a b.md" {
		t.Fatalf("after changes: %q %v", files, err)
	}
}

func TestModelHistoryAndSnapshotsOfItsCommits(t *testing.T) {
	ctx := context.Background()
	r := testkit.NewRepo(t)
	if commits, err := ModelHistory(ctx, r.Dir, 10); err != nil || len(commits) != 0 {
		t.Fatalf("a repository without a commit: %v %v", commits, err)
	}
	r.Write("README.md", "# x\n")
	r.Commit("readme")
	r.Write(".markitect/model/x.md", "# X\n\nTwo\nlines.\n")
	r.Write(".markitect/model/bin.md", string([]byte{0, 1, '\n', 2}))
	model := r.Commit("the model")
	r.Write(".markitect/results/run-1.md", "# Result\n")
	landed := r.Commit("land run 1")
	r.Write("src/a.go", "package a\n")
	r.Commit("code only")

	commits, err := ModelHistory(ctx, r.Dir, 10)
	if err != nil || len(commits) != 2 || commits[0].ID != landed || !commits[0].Landed || commits[1].ID != model || commits[1].Landed || commits[1].Subject != "the model" || commits[1].When.IsZero() {
		t.Fatalf("history %+v %v", commits, err)
	}
	if limited, _ := ModelHistory(ctx, r.Dir, 1); len(limited) != 1 {
		t.Errorf("the limit gave %d", len(limited))
	}

	s, err := Snapshot(ctx, r.Dir, model)
	if err != nil {
		t.Fatal(err)
	}
	files, _ := s.Files()
	if strings.Join(files, ",") != ".markitect/model/bin.md,.markitect/model/x.md,README.md" {
		t.Errorf("files %v", files)
	}
	if data, err := s.Read(".markitect/model/x.md"); err != nil || string(data) != "# X\n\nTwo\nlines.\n" {
		t.Errorf("x.md %q %v", data, err)
	}
	if data, err := s.Read(".markitect/model/bin.md"); err != nil || string(data) != string([]byte{0, 1, '\n', 2}) {
		t.Errorf("bin.md %q %v", data, err)
	}
	if _, err := s.Read("README.md"); err == nil {
		t.Error("a file outside Markitect's folder was read")
	}
}

func TestASnapshotReadsOnlyWhatALoadReads(t *testing.T) {
	ctx := context.Background()
	r := testkit.NewRepo(t)
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\n")
	r.Write(".markitect/model/with space.md", "# Spaces\n")
	r.Write(".markitect/registry/a.md", "# Registry\n")
	r.Write(".markitect/libraries/own/Kind.md", "# Kind\n")
	r.Write(".markitect/tool/source.zip", "a whole archive\n")
	r.Write(".markitect/results/run-1.md", "# Result\n")
	head := r.Commit("the model")
	// A submodule below the model has no content to read.
	r.Git("update-index", "--add", "--cacheinfo", "160000,"+head+",.markitect/model/sub")
	r.Git("commit", "-m", "a submodule")
	id := strings.TrimSpace(r.Git("rev-parse", "HEAD"))

	s, err := Snapshot(ctx, r.Dir, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{".markitect/manifest.yaml", ".markitect/model/with space.md", ".markitect/registry/a.md", ".markitect/libraries/own/Kind.md"} {
		if _, err := s.Read(f); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	for _, f := range []string{".markitect/tool/source.zip", ".markitect/results/run-1.md", ".markitect/model/sub"} {
		if _, err := s.Read(f); err == nil {
			t.Errorf("%s was read", f)
		}
	}
	files, _ := s.Files()
	if !strings.Contains(strings.Join(files, ","), ".markitect/tool/source.zip") {
		t.Errorf("a file left unread is not listed: %v", files)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ModelHistory(cancelled, r.Dir, 10); err == nil {
		t.Error("a request that ended read the history as empty")
	}
}
