package gitrepo

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestExternalModelInputsSurviveMutationRemovalAndFreshClosedRuns(t *testing.T) {
	r, previous, g, _ := runFixture(t)
	external := t.TempDir()
	original := filepath.Join(r.Dir, ".markitect/dependencies/core")
	err := filepath.WalkDir(original, func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(original, file)
		if err != nil {
			return err
		}
		target := filepath.Join(external, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	r.Git("rm", "-r", "--", ".markitect/dependencies/core")
	r.Commit("Keep explicitly selected dependency external")
	ctx := context.Background()
	tree := r.Git("rev-parse", "HEAD^{tree}")
	tree, err = withDirectories(ctx, r.Dir, tree, []string{".markitect/model/b/empty"})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := localCommand(ctx, r.Dir, []byte("Retain local empty Package\n"), "commit-tree", tree, "-p", r.Git("rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	r.Git("reset", "--mixed", commit)
	selection := previous.Selection()
	selection.Dependencies[0].Boundary = external
	repo := adapter(t)
	capture, err := project.NewPreparationService(project.NewModelService(repo, repo)).Capture(ctx, project.CaptureRequest{Selection: selection}, model.Limits{})
	attempt, ok := capture.Prepared()
	if err != nil || !ok {
		t.Fatal("external capture", capture.Status(), err)
	}
	p, err := project.Prepare(attempt)
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := p.Context("a")
	if !ok {
		t.Fatal("Manager context absent")
	}
	if _, found := selected.Definition(model.Identity{Package: "markitect/core", ID: "Kind"}); !found {
		t.Fatal("external imported Definition absent from actual context")
	}
	ref, err := g.Capture(ctx, "external-fixture", p)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(external, "model.yaml"), []byte("mutated external manifest"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.RemoveAll(external); err != nil {
		t.Fatal(err)
	}
	for _, mandate := range []run.Mandate{run.Assessment, run.Verify} {
		freshGit := &LocalGit{StateRoot: g.StateRoot, TempRoot: g.TempRoot}
		store, _ := runstore.New(g.StateRoot)
		executor := run.Executor{Workspaces: freshGit, Sessions: &controlledManagers{crossed: make(chan struct{})}, Records: store}
		result, err := executor.Execute(ctx, run.Request{MaxManagers: 5, ID: string(mandate), Project: "external-fixture", Prepared: p, Input: ref, Mandate: mandate, Allowance: run.NewAllowance(runDeadline(t), 80), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
		if err != nil || result.Technical != run.Completed || result.Candidate == nil {
			t.Fatal(result, err)
		}
		destination := t.TempDir()
		if err = freshGit.Reopen(ctx, *result.Candidate, destination); err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(filepath.Join(destination, ".markitect/dependencies/core")); !os.IsNotExist(err) {
			t.Fatal("external dependency was vendored into product", err)
		}
	}
	stored, err := g.candidatePath(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(stored, "inputs.json"), []byte(`{"Version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if service, err := g.Open(ctx, run.Request{Project: "external-fixture", Prepared: p, Input: ref}); err == nil || service != nil {
		t.Fatal("tampered retained authority started a Run service")
	}
}
