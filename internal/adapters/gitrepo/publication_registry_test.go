package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestPublicationAllowsValidAddedAndDeletedRegistryShards(t *testing.T) {
	r, previous, git, _ := runFixture(t)
	empty := "apiVersion: markitect.registry/v1\nartifacts: []\n"
	r.Write(".markitect/registry/b/registry.yaml", empty)
	r.Commit("Capture an empty intermediate Package registry")
	ctx := context.Background()
	tree, err := withDirectories(ctx, r.Dir, r.Git("rev-parse", "HEAD^{tree}"), []string{".markitect/model/b/empty"})
	if err != nil {
		t.Fatal(err)
	}
	head, err := localCommand(ctx, r.Dir, []byte("Retain empty Package\n"), "commit-tree", tree, "-p", r.Git("rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	r.Git("reset", "--mixed", head)
	repository := adapter(t)
	captured, err := project.NewPreparationService(project.NewModelService(repository, repository)).Capture(ctx, project.CaptureRequest{Selection: previous.Selection()}, model.Limits{})
	attempt, ok := captured.Prepared()
	if err != nil || !ok {
		t.Fatal(captured.Status(), err)
	}
	prepared, err := project.Prepare(attempt)
	if err != nil {
		t.Fatal(err)
	}
	input, err := git.Capture(ctx, "registry-fixture", prepared)
	if err != nil {
		t.Fatal(err)
	}
	service, err := git.Open(ctx, run.Request{ID: "registry-realization", Project: "registry-fixture", Prepared: prepared, Input: input})
	if err != nil {
		t.Fatal(err)
	}
	copy, err := service.Copy(ctx, ".", run.Commit{ID: input.Commit, Tree: input.Tree})
	if err != nil {
		t.Fatal(err)
	}
	dir := copy.Location().Directory
	if err = os.Remove(filepath.Join(dir, ".markitect/registry/b/registry.yaml")); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(dir, ".markitect/registry/b/empty"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, ".markitect/registry/b/empty/registry.yaml"), []byte(empty), 0600); err != nil {
		t.Fatal(err)
	}
	fixed, err := copy.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := copy.Inspect(ctx, fixed)
	if err != nil {
		t.Fatal(err)
	}
	if err = prepared.ValidateCandidate(checked, "."); err != nil {
		t.Fatal(err)
	}
	ref, err := service.Retain(ctx, copy, fixed)
	if err != nil {
		t.Fatal(err)
	}
	if err = copy.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err = service.Close(ctx); err != nil {
		t.Fatal(err)
	}
	records, _ := runstore.New(git.StateRoot)
	verified, err := (&run.Executor{Sessions: &controlledManagers{crossed: make(chan struct{})}, Workspaces: git, Records: records}).Execute(ctx, run.Request{ID: "registry-verify", Project: "registry-fixture", Prepared: prepared, Input: ref, Mandate: run.Verify, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || verified.Technical != run.Completed || verified.Decision != run.Approve {
		t.Fatal(verified.Technical, verified.Decision, err)
	}
	published, err := (&Publisher{Repository: repository, Store: git}).Publish(ctx, apply.PublicationRequest{Prepared: prepared, Approval: verified})
	if err != nil || !published.Readback || len(published.CompletedPaths) != 2 {
		t.Fatal(published, err)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, ".markitect/registry/b/registry.yaml")); !os.IsNotExist(err) {
		t.Fatal("old selected shard survived", err)
	}
	if data, err := os.ReadFile(filepath.Join(r.Dir, ".markitect/registry/b/empty/registry.yaml")); err != nil || string(data) != empty {
		t.Fatal("new shard not published", err)
	}
}

// Nothing stale is published: even a bound closed approval of a candidate that
// still links a deleted Definition is refused before any target write.
func TestPublicationRefusesACandidateWithAStaleRegistryLink(t *testing.T) {
	r, prepared, git, input := runFixture(t, deletedDefinition)
	head := r.Git("rev-parse", "HEAD")
	// This Publisher seam fixture supplies a bound closed approval; a real
	// Verify cannot approve this candidate, because Root's check refuses it.
	approval := run.Result{ID: "approved", Project: "fixture", Binding: prepared.Binding(), Mandate: run.Verify, Technical: run.Completed, Decision: run.Approve, Input: input, Candidate: &input}
	for _, m := range prepared.Topology().Managers() {
		approval.Handoffs = append(approval.Handoffs, run.Handoff{Manager: m.Package})
		approval.Cleanup = append(approval.Cleanup, run.Closure{Resource: "session:" + m.Package, Closed: true}, run.Closure{Resource: "copy:" + m.Package, Closed: true})
	}
	approval.Cleanup = append(approval.Cleanup, run.Closure{Resource: "manager-workers", Closed: true}, run.Closure{Resource: "git-service", Closed: true})
	published, err := (&Publisher{Repository: adapter(t), Store: git}).Publish(context.Background(), apply.PublicationRequest{Prepared: prepared, Approval: approval})
	if !errors.Is(err, project.ErrCandidateInvalid) || !strings.Contains(err.Error(), "registry.stale_link") || !strings.Contains(err.Error(), "out/gone.txt") || published.Readback || len(published.CompletedPaths) != 0 {
		t.Fatal("publication accepted a stale link", published, err)
	}
	if r.Git("rev-parse", "HEAD") != head {
		t.Fatal("refused publication changed the target")
	}
	if data, err := os.ReadFile(filepath.Join(r.Dir, "out/gone.txt")); err != nil || string(data) != "realizes a deleted Definition\n" {
		t.Fatal("refused publication touched the target", err)
	}
}

func TestCustomManifestPublicationUsesV1LockAndProtectedTargetStartsNoRun(t *testing.T) {
	r, previous, git, _ := runFixture(t)
	custom := filepath.Join(r.Dir, "custom")
	if err := os.Mkdir(custom, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"model.yaml", "model-lock.yaml", "model"} {
		if err := os.Rename(filepath.Join(r.Dir, ".markitect", name), filepath.Join(custom, name)); err != nil {
			t.Fatal(err)
		}
	}
	r.Commit("Select a nonconventional model manifest")
	ctx := context.Background()
	tree, err := withDirectories(ctx, r.Dir, r.Git("rev-parse", "HEAD^{tree}"), []string{"custom/model/b/empty"})
	if err != nil {
		t.Fatal(err)
	}
	head, err := localCommand(ctx, r.Dir, []byte("Retain custom empty Package\n"), "commit-tree", tree, "-p", r.Git("rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	r.Git("reset", "--mixed", head)
	selection := previous.Selection()
	selection.Local.ManifestPath = fixturePath(t, "custom/model.yaml")
	repository := adapter(t)
	preparation := project.NewPreparationService(project.NewModelService(repository, repository))
	prepare := func() project.Prepared {
		captured, err := preparation.Capture(ctx, project.CaptureRequest{Selection: selection}, model.Limits{})
		attempt, ok := captured.Prepared()
		if err != nil || !ok {
			t.Fatal(captured.Status(), err)
		}
		prepared, err := project.Prepare(attempt)
		if err != nil {
			t.Fatal(err)
		}
		return prepared
	}
	prepared := prepare()
	ref, err := git.Capture(ctx, "custom-manifest", prepared)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := runstore.New(git.StateRoot)
	verified, err := (&run.Executor{Sessions: &controlledManagers{crossed: make(chan struct{})}, Workspaces: git, Records: store}).Execute(ctx, run.Request{ID: "custom-verify", Project: "custom-manifest", Prepared: prepared, Input: ref, Mandate: run.Verify, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	publisher := &Publisher{Repository: repository, Store: git}
	if err = publisher.Preflight(ctx, prepared); err != nil {
		t.Fatal(err)
	}
	published, err := publisher.Publish(ctx, apply.PublicationRequest{Prepared: prepared, Approval: verified})
	if err != nil || !published.Readback || !published.NoChange {
		t.Fatal(published, err)
	}
	if _, err = os.Stat(filepath.Join(r.Dir, ".artifacts")); !os.IsNotExist(err) {
		t.Fatal("custom layout used legacy lock administration", err)
	}
	if _, err = os.Stat(filepath.Join(r.Dir, ".markitect/state/write.lock")); !os.IsNotExist(err) {
		t.Fatal("V1 cooperating lock survived", err)
	}
	// Apply uses the same existing write policy before opening any Manager.
	r.Git("branch", "-m", "main")
	prepared = prepare()
	executor := &applyExecutor{t: t, git: git, store: store}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: publisher}
	report, err := command.Execute(ctx, run.Request{ID: "protected", Project: "custom-manifest", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100)})
	if err == nil || !strings.Contains(err.Error(), "non-protected") || len(executor.requests) != 0 || report.Publication != nil {
		t.Fatal("protected target was not an early preflight refusal", err, len(executor.requests))
	}
}
