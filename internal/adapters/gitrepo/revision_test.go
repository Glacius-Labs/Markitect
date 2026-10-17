package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestResolveRevisionExplicitCommitRefs(t *testing.T) {
	repo := testkit.NewRepo(t)
	repo.Write("file.txt", "first\n")
	first := repo.Commit("first")
	repo.Git("branch", "selected-branch", first)
	repo.Git("tag", "selected-tag", first)
	repo.Git("tag", "-a", "annotated-tag", "-m", "selected annotated tag", first)
	repo.Write("file.txt", "second\n")
	head := repo.Commit("second")
	// Resolving a ref neither requires a clean tree nor captures working bytes.
	repo.Write("file.txt", "uncommitted\n")
	repo.Write("untracked.txt", "untracked\n")
	r := adapter(t)
	for _, tc := range []struct{ ref, want string }{
		{"HEAD", head},
		{"selected-branch", first},
		{"selected-tag", first},
		{"annotated-tag", first},
		{first, first},
	} {
		t.Run(tc.ref, func(t *testing.T) {
			got, err := r.ResolveRevision(context.Background(), repo.Dir, tc.ref)
			if err != nil || got != tc.want {
				t.Fatalf("ResolveRevision(%q) = %q, %v; want %q", tc.ref, got, err, tc.want)
			}
		})
	}
	if got := repo.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD changed to %q", got)
	}
	for path, want := range map[string]string{"file.txt": "uncommitted\n", "untracked.txt": "untracked\n"} {
		got, err := os.ReadFile(filepath.Join(repo.Dir, path))
		if err != nil || string(got) != want {
			t.Fatalf("working file %s = %q, %v", path, got, err)
		}
	}
}

func TestResolveRevisionRejectsInvalidSelection(t *testing.T) {
	repo := testkit.NewRepo(t)
	repo.Write("subdir/file.txt", "source\n")
	repo.Commit("source")
	nonGit := testkit.TempDir(t)
	r := adapter(t)
	for _, tc := range []struct {
		name, repository, ref string
		unsafe                bool
	}{
		{"empty ref", repo.Dir, "", true},
		{"option ref", repo.Dir, "--all", true},
		{"missing ref", repo.Dir, "refs/heads/does-not-exist", false},
		{"tree object", repo.Dir, "HEAD^{tree}", false},
		{"empty repository", "", "HEAD", true},
		{"missing repository", filepath.Join(nonGit, "missing"), "HEAD", false},
		{"non Git directory", nonGit, "HEAD", false},
		{"repository subdirectory", filepath.Join(repo.Dir, "subdir"), "HEAD", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.ResolveRevision(context.Background(), tc.repository, tc.ref)
			if err == nil || got != "" {
				t.Fatalf("invalid selection resolved to %q, %v", got, err)
			}
			if tc.unsafe && !errors.Is(err, ErrUnsafe) {
				t.Fatalf("error = %v; want ErrUnsafe", err)
			}
		})
	}
}

func TestResolveRevisionPreservesContextFailure(t *testing.T) {
	repo := testkit.NewRepo(t)
	repo.Write("file.txt", "source\n")
	repo.Commit("source")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, expire := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer expire()
	r := adapter(t)
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"canceled", canceled, context.Canceled},
		{"deadline", expired, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.ResolveRevision(tc.ctx, repo.Dir, "HEAD")
			if got != "" || !errors.Is(err, tc.want) {
				t.Fatalf("ResolveRevision = %q, %v; want empty SHA and %v", got, err, tc.want)
			}
		})
	}
}

func TestResolveRevisionRefusesChangedIdentityOrCancellationAfterResolution(t *testing.T) {
	for _, kind := range []string{"git-directory-replacement", "cancel-before-confirm"} {
		t.Run(kind, func(t *testing.T) {
			repo := testkit.NewRepo(t)
			repo.Write("file.txt", "original\n")
			repo.Commit("original")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := adapter(t)
			observed := false
			r.observe = func(event, path string) {
				if event != "before-confirm" || observed {
					return
				}
				if filepath.Clean(path) != filepath.Clean(repo.Dir) {
					t.Fatalf("confirmation path = %q; want %q", path, repo.Dir)
				}
				observed = true
				if kind == "cancel-before-confirm" {
					cancel()
					return
				}
				replacement := testkit.NewRepo(t)
				replacement.Write("file.txt", "replacement\n")
				replacement.Commit("replacement")
				gitDir := filepath.Join(repo.Dir, ".git")
				if err := os.Rename(gitDir, gitDir+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(replacement.Dir, ".git"), gitDir); err != nil {
					t.Fatal(err)
				}
			}
			got, err := r.ResolveRevision(ctx, repo.Dir, "HEAD")
			want := ErrChanged
			if kind == "cancel-before-confirm" {
				want = context.Canceled
			}
			if !observed || got != "" || !errors.Is(err, want) {
				t.Fatalf("ResolveRevision = %q, %v; observed=%v; want empty SHA and %v", got, err, observed, want)
			}
		})
	}
}
