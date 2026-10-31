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
