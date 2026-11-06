package rungit_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

func open(t *testing.T) (*rungit.Repo, *testkit.Repo) {
	t.Helper()
	r := testkit.NewRepo(t)
	r.Git("config", "user.name", "Owner")
	r.Git("config", "user.email", "owner@example.invalid")
	r.Git("config", "core.autocrlf", "false")
	r.Write("README.md", "# Project\n")
	r.Write(".markitect/manifest.yaml", "libraries: {}\n")
	r.Write(".gitignore", "build/\n.claude/worktrees/\n")
	r.Commit("first")
	host, err := rungit.Open(context.Background(), r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	return host, r
}

// Open finds the common Git directory from a linked worktree too, so that every
// command of a Run sees the same store.
func TestOpenFindsTheCommonDirectory(t *testing.T) {
	host, r := open(t)
	wt := filepath.Join(r.Dir, ".claude", "worktrees", "r1")
	r.Git("worktree", "add", "-q", "-b", "worktree-r1", wt, "HEAD")
	inner, err := rungit.Open(context.Background(), wt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(filepath.ToSlash(inner.Dir), filepath.ToSlash(host.Dir)) {
		t.Fatalf("%s and %s", inner.Dir, host.Dir)
	}
}

// Uncommitted leaves out the spec, which apply commits, and the provider's own
// local files.
func TestUncommittedCountsOnlyTheWork(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	r.Write(".markitect/manifest.yaml", "libraries: {std: 1.0.0}\n")
	r.Write(".claude/settings.local.json", "{}\n")
	r.Write(".claude/worktrees/x/file", "x\n")
	r.Write("build/out", "ignored\n")
	got, err := host.Uncommitted(ctx, r.Dir)
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
	r.Write("README.md", "# Changed\n")
	r.Write("new.txt", "new\n")
	if err := os.MkdirAll(filepath.Join(r.Dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = host.Uncommitted(ctx, filepath.Join(r.Dir, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "README.md,new.txt" {
		t.Fatalf("from a subfolder: %v", got)
	}
}

// CommitSpec commits only the spec, with the user's identity, and is quiet when
// nothing changed.
func TestCommitSpec(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	before := r.Git("rev-parse", "HEAD")
	if got, err := host.CommitSpec(ctx, r.Dir, "spec"); err != nil || got != before {
		t.Fatalf("%s %v", got, err)
	}
	r.Write(".markitect/manifest.yaml", "libraries: {std: 1.0.0}\n")
	r.Write("README.md", "# Being edited\n")
	got, err := host.CommitSpec(ctx, r.Dir, "spec")
	if err != nil || got == before {
		t.Fatalf("%s %v", got, err)
	}
	if author := r.Git("log", "-1", "--format=%an"); author != "Owner" {
		t.Fatalf("author %s", author)
	}
	if files := r.Git("show", "--name-only", "--format=", "HEAD"); files != ".markitect/manifest.yaml" {
		t.Fatalf("committed %s", files)
	}
	if status := r.Git("status", "--porcelain"); !strings.Contains(status, "README.md") {
		t.Fatalf("the other edit went: %s", status)
	}
}

// Candidate takes what Git would take, new and deleted files included, and leaves
// the worktree, its index and HEAD as they are.
func TestCandidateChangesNothing(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	head := r.Git("rev-parse", "HEAD")
	if got, err := host.Candidate(ctx, r.Dir, "m"); err != nil || got != head {
		t.Fatalf("no change: %s %v", got, err)
	}
	r.Write("new.txt", "new\n")
	r.Write("build/out", "ignored\n")
	if err := os.Remove(filepath.Join(r.Dir, "README.md")); err != nil {
		t.Fatal(err)
	}
	before := r.Git("status", "--porcelain")
	cand, err := host.Candidate(ctx, r.Dir, "m")
	if err != nil || cand == head {
		t.Fatalf("%s %v", cand, err)
	}
	if after := r.Git("status", "--porcelain"); after != before {
		t.Fatalf("status changed: %q to %q", before, after)
	}
	if got := r.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD moved to %s", got)
	}
	files := r.Git("ls-tree", "-r", "--name-only", cand)
	if !strings.Contains(files, "new.txt") || strings.Contains(files, "build/out") || strings.Contains(files, "README.md") {
		t.Fatalf("the candidate holds: %s", files)
	}
}

// A worktree of a Run is never part of the work, even if nothing ignores it.
func TestCandidateLeavesWorktreesOut(t *testing.T) {
	host, r := open(t)
	r.Write(".gitignore", "build/\n")
	r.Commit("worktrees not ignored")
	wt := filepath.Join(r.Dir, ".claude", "worktrees", "r1")
	r.Git("worktree", "add", "-q", "-b", "worktree-r1", wt, "HEAD")
	r.Write("new.txt", "new\n")
	cand, err := host.Candidate(context.Background(), r.Dir, "m")
	if err != nil {
		t.Fatal(err)
	}
	if files := r.Git("ls-tree", "-r", "--name-only", cand); strings.Contains(files, "worktrees") || !strings.Contains(files, "new.txt") {
		t.Fatalf("the candidate holds: %s", files)
	}
}

// Advance moves a checked-out branch only forward and only from where it was,
// keeps uncommitted edits, and moves a branch nobody has checked out by its ref.
func TestAdvance(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	head := r.Git("rev-parse", "HEAD")
	next, err := host.Write(ctx, head, map[string][]byte{"result.md": []byte("x\n")}, "m\n")
	if err != nil {
		t.Fatal(err)
	}
	r.Write("notes.txt", "being edited\n")
	if err := host.Advance(ctx, "main", next, "0000000"); err == nil {
		t.Fatal("moved from the wrong commit")
	}
	if err := host.Advance(ctx, "main", next, head); err != nil {
		t.Fatal(err)
	}
	if got := r.Git("rev-parse", "HEAD"); got != next {
		t.Fatalf("at %s", got)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "result.md")); err != nil {
		t.Fatalf("the working tree did not follow: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "notes.txt")); err != nil {
		t.Fatalf("the edit went: %v", err)
	}
	// A branch nobody has checked out.
	r.Git("branch", "other", head)
	if err := host.Advance(ctx, "other", next, head); err != nil {
		t.Fatal(err)
	}
	if got := r.Git("rev-parse", "other"); got != next {
		t.Fatalf("other at %s", got)
	}
	if err := host.Advance(ctx, "other", head, head); err == nil {
		t.Fatal("a ref moved from the wrong commit")
	}
}

// Work already in the folder stays, the files it lacks arrive, and an edit of the
// owner's own to a file the move changes stops it before anything moves.
func TestAdvanceOverWorkInTheFolder(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	head := r.Git("rev-parse", "HEAD")
	r.Write("work.go", "package work\n")
	cand, err := host.Candidate(ctx, r.Dir, "the work\n")
	if err != nil {
		t.Fatal(err)
	}
	next, err := host.Write(ctx, cand, map[string][]byte{"result.md": []byte("landed\n")}, "result\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Advance(ctx, "main", next, head); err != nil {
		t.Fatal(err)
	}
	if got := r.Git("rev-parse", "HEAD"); got != next {
		t.Fatalf("at %s", got)
	}
	if status := r.Git("status", "--porcelain"); status != "" {
		t.Fatalf("not clean: %s", status)
	}
	if data, err := os.ReadFile(filepath.Join(r.Dir, "result.md")); err != nil || string(data) != "landed\n" {
		t.Fatalf("%q %v", data, err)
	}

	again, err := host.Write(ctx, next, map[string][]byte{"result.md": []byte("again\n")}, "again\n")
	if err != nil {
		t.Fatal(err)
	}
	r.Write("result.md", "my own edit\n")
	if err := host.Advance(ctx, "main", again, next); err == nil || !strings.Contains(err.Error(), "result.md") {
		t.Fatalf("moved over an edit: %v", err)
	}
	if got := r.Git("rev-parse", "HEAD"); got != next {
		t.Fatalf("moved anyway, to %s", got)
	}
}

// The files change before the branch moves, so a landing stopped by Git's own lock
// runs again from the start; one stopped after the branch moved only has the index
// left to bring up.
func TestAdvanceRunsAgainAfterALock(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	head := r.Git("rev-parse", "HEAD")
	r.Write("work.go", "package work\n")
	cand, err := host.Candidate(ctx, r.Dir, "the work\n")
	if err != nil {
		t.Fatal(err)
	}
	next, err := host.Write(ctx, cand, map[string][]byte{"result.md": []byte("landed\n")}, "result\n")
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(r.Dir, ".git", "index.lock")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := host.Advance(ctx, "main", next, head); err == nil {
		t.Fatal("landed through Git's lock")
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if got := r.Git("rev-parse", "HEAD"); got == next {
		// The branch moved before the index: only the index is left.
		if err := host.Advance(ctx, "main", next, next); err != nil {
			t.Fatal(err)
		}
	} else if err := host.Advance(ctx, "main", next, head); err != nil {
		t.Fatal(err)
	}
	if got := r.Git("rev-parse", "HEAD"); got != next {
		t.Fatalf("at %s", got)
	}
	if status := r.Git("status", "--porcelain"); status != "" {
		t.Fatalf("not clean: %s", status)
	}
}

// A link in the work counts as the work, as Git stores it.
func TestAdvanceOverALink(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	head := r.Git("rev-parse", "HEAD")
	r.Write("target.go", "package work\n")
	if err := os.Symlink("target.go", filepath.Join(r.Dir, "link.go")); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}
	cand, err := host.Candidate(ctx, r.Dir, "the work\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Advance(ctx, "main", cand, head); err != nil {
		t.Fatal(err)
	}
	if status := r.Git("status", "--porcelain"); status != "" {
		t.Fatalf("not clean: %s", status)
	}
}

// Tip names one branch, also when others share its folder, and nothing for one that
// does not exist.
func TestTip(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	head := r.Git("rev-parse", "HEAD")
	r.Git("branch", "feature/a")
	r.Git("branch", "feature/b")
	if got, err := host.Tip(ctx, "feature/a"); err != nil || got != head {
		t.Fatalf("%q %v", got, err)
	}
	if got, err := host.Tip(ctx, "feature"); err != nil || got != "" {
		t.Fatalf("a folder of branches: %q %v", got, err)
	}
	if got, err := host.CountCommits(ctx, head, head); err != nil || got != 0 {
		t.Fatalf("%d %v", got, err)
	}
}

// DeleteBranch removes a branch, and a second call finds nothing to do.
func TestDeleteBranch(t *testing.T) {
	host, r := open(t)
	ctx := context.Background()
	r.Git("branch", "markitect/runs/r1")
	if err := host.DeleteBranch(ctx, "markitect/runs/r1"); err != nil {
		t.Fatal(err)
	}
	if got, err := host.Tip(ctx, "markitect/runs/r1"); err != nil || got != "" {
		t.Fatalf("it remains: %q %v", got, err)
	}
	if err := host.DeleteBranch(ctx, "markitect/runs/r1"); err != nil {
		t.Fatalf("a second delete: %v", err)
	}
}
