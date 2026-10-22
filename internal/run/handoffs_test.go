package run_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type handoffIntegrationFake struct {
	accepted []string
	conflict string
}

func (f *handoffIntegrationFake) Changes(_ context.Context, _ run.AssignmentWorkspace, handoff run.AssignmentHandoff) (string, error) {
	return "diff for " + handoff.ResultCommit, nil
}

func (f *handoffIntegrationFake) Integrate(_ context.Context, parent, child run.AssignmentWorkspace, handoff run.AssignmentHandoff, previous string) (string, error) {
	if child.ID() == run.WorkspaceID(f.conflict) {
		return "", errors.New("Git conflict")
	}
	f.accepted = append(f.accepted, string(parent.ID())+"<-"+string(child.ID())+":"+previous+":"+handoff.ResultCommit)
	return strings.Repeat("f", 40), nil
}

func TestAssignmentHandoffsRouteQuestionCorrectionAndRequiredChildren(t *testing.T) {
	ctx := context.Background()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}},
		{Path: "library/lending", Parent: "library"},
		{Path: "library/storage", Parent: "library"},
	}
	work, agents, git := &assignmentWorkspacesFake{}, &assignmentAgentsFake{}, &handoffIntegrationFake{}
	f, err := run.NewAssignmentHandoffs(team, work, agents, git)
	if err != nil {
		t.Fatal(err)
	}
	request := func(id string) run.AssignmentRequest {
		return run.AssignmentRequest{ID: run.WorkspaceID(id), Profile: "codex", Home: "home-" + id}
	}
	if err := f.StartRoot(ctx, request("root")); err != nil {
		t.Fatal(err)
	}
	if err := f.Delegate(ctx, ".", "library", request("library")); err != nil {
		t.Fatal(err)
	}
	if err := f.Delegate(ctx, "library", "library/lending", request("lending")); err != nil {
		t.Fatal(err)
	}
	if err := f.Delegate(ctx, "library", "library/storage", request("storage")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Message(ctx, "library/storage", "question", time.Second); err != nil {
		t.Fatal(err)
	}
	if got := agents.opened["library"].messages[0]; !strings.Contains(got, "library/storage asks: Which contract?") {
		t.Fatalf("question skipped parent: %q", got)
	}
	for _, path := range []string{"library/lending", "library/storage"} {
		if _, err := f.Message(ctx, path, "first result", time.Second); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Submit(ctx, path); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.Submit(ctx, "library"); err == nil {
		t.Fatal("Library submitted without required child integration")
	}
	changes, first, err := f.Changes(ctx, "library", "library/lending")
	if err != nil || !strings.Contains(changes, first.ResultCommit) {
		t.Fatalf("fixed child result was not inspectable: %q %+v %v", changes, first, err)
	}
	if _, err := f.Accept(ctx, "library", "library/lending", "uninspected"); err == nil {
		t.Fatal("accepted a result other than the inspected commit")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := f.Return(cancelled, "library", "library/lending", "Correct the loan limit test", time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled correction: %v", err)
	}
	if _, retained, err := f.Changes(ctx, "library", "library/lending"); err != nil || retained.ResultCommit != first.ResultCommit {
		t.Fatalf("failed correction hid the fixed result: %+v %v", retained, err)
	}
	if _, err := f.Accept(ctx, "library", "library/lending", first.ResultCommit); err == nil {
		t.Fatal("returned work was accepted after its correction failed")
	}
	if _, err := f.Return(ctx, "library", "library/lending", "Correct the loan limit test", time.Second); err != nil {
		t.Fatal(err)
	}
	corrected, err := f.Submit(ctx, "library/lending")
	if err != nil || corrected.ResultCommit == first.ResultCommit {
		t.Fatalf("correction failed to produce a new fixed result: %+v %v", corrected, err)
	}
	if _, err := f.Accept(ctx, "library", "library/lending", first.ResultCommit); err == nil {
		t.Fatal("accepted stale pre-correction result")
	}
	if _, _, err := f.Changes(ctx, "library", "library/lending"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Accept(ctx, "library", "library/lending", corrected.ResultCommit); err != nil {
		t.Fatal(err)
	}
	_, storage, err := f.Changes(ctx, "library", "library/storage")
	if err != nil {
		t.Fatal(err)
	}
	git.conflict = "storage"
	if _, err := f.Accept(ctx, "library", "library/storage", storage.ResultCommit); err == nil {
		t.Fatal("integration conflict was hidden")
	}
	if _, err := f.Submit(ctx, "library"); err == nil {
		t.Fatal("conflicted child was treated as integrated")
	}
	git.conflict = ""
	if _, err := f.Accept(ctx, "library", "library/storage", storage.ResultCommit); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Submit(ctx, "library"); err != nil {
		t.Fatalf("Library could not hand integrated work to Root: %v", err)
	}
	if len(git.accepted) != 2 || !strings.Contains(git.accepted[0], corrected.ResultCommit) {
		t.Fatalf("child integration history: %v", git.accepted)
	}
}

func TestAssignmentHandoffsRejectsMissingOrUnrelatedManagers(t *testing.T) {
	work, agents, git := &assignmentWorkspacesFake{}, &assignmentAgentsFake{}, &handoffIntegrationFake{}
	for _, team := range [][]run.Manager{
		{{Path: ".", Children: []string{"library"}}},
		{{Path: "."}, {Path: "library", Parent: "."}},
		{{Path: ".", Children: []string{"library", "library"}}, {Path: "library", Parent: "."}},
	} {
		if _, err := run.NewAssignmentHandoffs(team, work, agents, git); err == nil {
			t.Fatalf("accepted malformed required team: %+v", team)
		}
	}
}
