package run_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type handoffIntegrationFake struct {
	mu       sync.Mutex
	accepted []string
	conflict string
	entered  chan struct{}
	release  chan struct{}
	cleanup  error
}

func (f *handoffIntegrationFake) Changes(_ context.Context, _ run.AssignmentWorkspace, handoff run.AssignmentHandoff) (string, error) {
	return "diff for " + handoff.ResultCommit, nil
}

func (f *handoffIntegrationFake) Integrate(_ context.Context, parent, child run.AssignmentWorkspace, handoff run.AssignmentHandoff, previous string) (string, error) {
	if f.entered != nil {
		close(f.entered)
		<-f.release
	}
	if child.ID() == run.WorkspaceID(f.conflict) {
		return "", errors.New("Git conflict")
	}
	f.mu.Lock()
	f.accepted = append(f.accepted, string(parent.ID())+"<-"+string(child.ID())+":"+previous+":"+handoff.ResultCommit)
	f.mu.Unlock()
	return strings.Repeat("f", 40), f.cleanup
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
	if _, err := f.Accept(ctx, "library", "library/lending", corrected.ResultCommit); err == nil {
		t.Fatal("same child commit was integrated twice")
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
	if _, err := f.Return(ctx, "library", "library/storage", "Fix storage integration", time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Submit(ctx, "library"); err == nil {
		t.Fatal("Library submitted while accepted child had an open correction")
	}
	newStorage, err := f.Submit(ctx, "library/storage")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.Changes(ctx, "library", "library/storage"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Accept(ctx, "library", "library/storage", newStorage.ResultCommit); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Submit(ctx, "library"); err != nil {
		t.Fatalf("Library could not hand integrated work to Root: %v", err)
	}
	if len(git.accepted) != 3 || !strings.Contains(git.accepted[0], corrected.ResultCommit) || !strings.Contains(git.accepted[2], storage.ResultCommit+":"+newStorage.ResultCommit) {
		t.Fatalf("child integration history: %v", git.accepted)
	}
}

func TestAssignmentHandoffsRetainsPartialStartWithoutRunnableParent(t *testing.T) {
	ctx := context.Background()
	work := &assignmentWorkspacesFake{prepareErr: errors.New("incomplete copy")}
	f, err := run.NewAssignmentHandoffs([]run.Manager{{Path: ".", Children: []string{"child"}}, {Path: "child", Parent: "."}}, work, &assignmentAgentsFake{}, &handoffIntegrationFake{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.StartRoot(ctx, run.AssignmentRequest{ID: "failed", Profile: "codex", Home: "root"}); err == nil {
		t.Fatal("partial root start appeared complete")
	}
	if retained := f.RetainedWorkspace("."); retained == nil || retained.ID() != "failed" {
		t.Fatalf("partial copy was not retained: %+v", retained)
	}
	if err := f.Delegate(ctx, ".", "child", run.AssignmentRequest{ID: "child", Profile: "codex", Home: "child"}); err == nil {
		t.Fatal("child started under a failed parent")
	}
	work.prepareErr = nil
	if err := f.StartRoot(ctx, run.AssignmentRequest{ID: "retry", Profile: "codex", Home: "root"}); err != nil {
		t.Fatalf("failed parent could not be restarted on a fresh copy: %v", err)
	}
}

func TestAssignmentHandoffsCancelledQuestionWaitsForBusyParent(t *testing.T) {
	ctx := context.Background()
	work, agents := &assignmentWorkspacesFake{}, &assignmentAgentsFake{}
	f, err := run.NewAssignmentHandoffs([]run.Manager{{Path: ".", Children: []string{"child"}}, {Path: "child", Parent: "."}}, work, agents, &handoffIntegrationFake{})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct{ path, id string }{{".", "root"}, {"child", "child"}} {
		path, id := entry.path, entry.id
		request := run.AssignmentRequest{ID: run.WorkspaceID(id), Profile: "codex", Home: id}
		if path == "." {
			err = f.StartRoot(ctx, request)
		} else {
			err = f.Delegate(ctx, ".", path, request)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	agents.opened["root"].onMessage = make(chan struct{})
	busy, stopBusy := context.WithCancel(ctx)
	parentDone := make(chan error, 1)
	go func() { _, err := f.Message(busy, ".", "wait", time.Minute); parentDone <- err }()
	<-agents.opened["root"].onMessage
	questionCtx, cancelQuestion := context.WithCancel(ctx)
	questionDone := make(chan error, 1)
	go func() { _, err := f.Message(questionCtx, "child", "question", time.Minute); questionDone <- err }()
	cancelQuestion()
	select {
	case err := <-questionDone:
		if err == nil {
			t.Fatal("cancelled child question was answered")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled child question stayed blocked behind busy parent")
	}
	stopBusy()
	<-parentDone
}

func TestAssignmentHandoffsSerializesParentGitWorkAndRechecksChild(t *testing.T) {
	ctx := context.Background()
	work, agents := &assignmentWorkspacesFake{}, &assignmentAgentsFake{}
	git := &handoffIntegrationFake{entered: make(chan struct{}), release: make(chan struct{})}
	f, err := run.NewAssignmentHandoffs([]run.Manager{{Path: ".", Children: []string{"child"}}, {Path: "child", Parent: "."}}, work, agents, git)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.StartRoot(ctx, run.AssignmentRequest{ID: "root", Profile: "codex", Home: "root"}); err != nil {
		t.Fatal(err)
	}
	if err := f.Delegate(ctx, ".", "child", run.AssignmentRequest{ID: "child", Profile: "codex", Home: "child"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Message(ctx, "child", "result", time.Second); err != nil {
		t.Fatal(err)
	}
	handoff, err := f.Submit(ctx, "child")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.Changes(ctx, ".", "child"); err != nil {
		t.Fatal(err)
	}
	acceptDone := make(chan error, 1)
	go func() { _, err := f.Accept(ctx, ".", "child", handoff.ResultCommit); acceptDone <- err }()
	<-git.entered
	blocked, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := f.Message(blocked, ".", "parent edit", time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("concurrent parent edit was not held behind integration: %v", err)
	}
	if _, err := f.Submit(blocked, "."); !errors.Is(err, context.Canceled) {
		t.Fatalf("concurrent parent submit was not held behind integration: %v", err)
	}
	if len(agents.opened["root"].messages) != 0 {
		t.Fatal("parent Agent edited its stand during integration")
	}
	if _, err := f.Return(ctx, ".", "child", "correct changed interface", time.Second); err != nil {
		t.Fatal(err)
	}
	close(git.release)
	if err := <-acceptDone; err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("accepted a child returned during Git integration: %v", err)
	}
	if _, err := f.Submit(ctx, "."); err == nil {
		t.Fatal("parent submitted returned child after an integration race")
	}
}

func TestAssignmentHandoffsRecordsIntegratedCommitEvenIfCleanupFails(t *testing.T) {
	ctx := context.Background()
	work, agents := &assignmentWorkspacesFake{}, &assignmentAgentsFake{}
	git := &handoffIntegrationFake{cleanup: errors.New("temporary worktree cleanup failed")}
	f, err := run.NewAssignmentHandoffs([]run.Manager{{Path: ".", Children: []string{"child"}}, {Path: "child", Parent: "."}}, work, agents, git)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.StartRoot(ctx, run.AssignmentRequest{ID: "root", Profile: "codex", Home: "root"}); err != nil {
		t.Fatal(err)
	}
	if err := f.Delegate(ctx, ".", "child", run.AssignmentRequest{ID: "child", Profile: "codex", Home: "child"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Message(ctx, "child", "result", time.Second); err != nil {
		t.Fatal(err)
	}
	handoff, err := f.Submit(ctx, "child")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.Changes(ctx, ".", "child"); err != nil {
		t.Fatal(err)
	}
	result, err := f.Accept(ctx, ".", "child", handoff.ResultCommit)
	if result == "" || !errors.Is(err, git.cleanup) {
		t.Fatalf("integrated result and cleanup error were not both retained: %q %v", result, err)
	}
	if _, err := f.Accept(ctx, ".", "child", handoff.ResultCommit); err == nil || len(git.accepted) != 1 {
		t.Fatalf("cleanup retry reapplied the same child delta: %v %v", err, git.accepted)
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
