package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type assignmentWorkspacesFake struct {
	mu         sync.Mutex
	copies     map[run.WorkspaceID]*assignmentCopyFake
	prepareErr error
	forks      int
}

func (f *assignmentWorkspacesFake) ForkAssignment(ctx context.Context, parent run.AssignmentWorkspace, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.copies[id] != nil {
		return nil, errors.New("duplicate child copy")
	}
	foundation, err := parent.Commit(ctx, "parent foundation")
	if err != nil {
		return nil, err
	}
	f.forks++
	child := &assignmentCopyFake{id: id, commits: []string{foundation}}
	f.copies[id] = child
	return child, nil
}

type assignmentCopyFake struct {
	id      run.WorkspaceID
	commits []string
	closed  bool
}

func (f *assignmentWorkspacesFake) PrepareAssignment(_ context.Context, spec run.WorkspaceSpec) (run.AssignmentWorkspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.copies == nil {
		f.copies = map[run.WorkspaceID]*assignmentCopyFake{}
	}
	if _, exists := f.copies[spec.ID]; exists {
		return nil, errors.New("duplicate copy")
	}
	copy := &assignmentCopyFake{id: spec.ID, commits: []string{strings.Repeat(string(spec.ID[0]), 40)}}
	f.copies[spec.ID] = copy
	return copy, f.prepareErr
}

func TestAssignmentStartRetainsPartialWorkspace(t *testing.T) {
	work := &assignmentWorkspacesFake{prepareErr: errors.New("workspace initialization incomplete")}
	agents := &assignmentAgentsFake{}
	partial, err := run.StartAssignment(context.Background(), work, agents, run.AssignmentRequest{ID: "agent", Profile: "codex", Home: "home"})
	if err == nil || partial == nil || partial.Workspace() == nil || partial.Workspace().Volume() != "volume-agent" {
		t.Fatalf("partial assignment %+v: %v", partial, err)
	}
	if len(agents.opened) != 0 {
		t.Fatal("Agent started on an incomplete working copy")
	}
}

func TestAssignmentStartsOnPreparedChildCopyWithoutRecreatingIt(t *testing.T) {
	copy := &assignmentCopyFake{id: "child", commits: []string{strings.Repeat("a", 40)}}
	work, agents := &assignmentWorkspacesFake{}, &assignmentAgentsFake{}
	a, err := run.StartAssignment(context.Background(), work, agents, run.AssignmentRequest{ID: "child", Profile: "codex", Home: "home", Prepared: copy})
	if err != nil || a.Workspace() != copy || len(work.copies) != 0 {
		t.Fatalf("prepared copy was recreated: %+v %v", a, err)
	}
	if agents.opened["child"].spec.Workspace != "volume-child" {
		t.Fatal("Agent did not mount its prepared child copy")
	}
	if _, err := run.StartAssignment(context.Background(), work, agents, run.AssignmentRequest{ID: "other", Profile: "codex", Home: "home", Prepared: copy}); err == nil {
		t.Fatal("mismatched prepared copy was accepted")
	}
}

func TestDirectChildDelegationForksCurrentParentGitStand(t *testing.T) {
	ctx := context.Background()
	work, agents, git := &assignmentWorkspacesFake{}, &assignmentAgentsFake{}, &handoffIntegrationFake{}
	flow, err := run.NewAssignmentHandoffs([]run.Manager{{Path: ".", Children: []string{"child"}}, {Path: "child", Parent: "."}}, work, agents, git)
	if err != nil {
		t.Fatal(err)
	}
	if err := flow.StartRoot(ctx, run.AssignmentRequest{ID: "root", Profile: "codex", Home: "root"}); err != nil {
		t.Fatal(err)
	}
	answer, err := flow.Message(ctx, ".", "parent foundation", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	answer.Commands[0].Status = "changed by caller"
	*answer.Commands[0].ExitCode = 99
	if retained := flow.Observations("."); retained[0].Commands[0].Status != "passed" || *retained[0].Commands[0].ExitCode != 0 {
		t.Fatalf("returned response changed retained native observation: %+v", retained)
	}
	if err := flow.DelegateCurrent(ctx, ".", "child", run.AssignmentRequest{ID: "child", Profile: "codex", Home: "child"}); err != nil {
		t.Fatal(err)
	}
	if work.forks != 1 || work.copies["child"].StartingCommit() != work.copies["root"].commits[len(work.copies["root"].commits)-1] || agents.opened["child"].spec.Workspace == agents.opened["root"].spec.Workspace {
		t.Fatalf("child did not receive a separate copy of current parent stand: %+v", work.copies)
	}
}
func (f *assignmentCopyFake) ID() run.WorkspaceID    { return f.id }
func (f *assignmentCopyFake) Volume() string         { return "volume-" + string(f.id) }
func (f *assignmentCopyFake) StartingCommit() string { return f.commits[0] }
func (f *assignmentCopyFake) Commit(context.Context, string) (string, error) {
	id := strings.Repeat(string('a'+rune(len(f.commits))), 40)
	f.commits = append(f.commits, id)
	return id, nil
}
func (f *assignmentCopyFake) ContainsCommit(_ context.Context, id string) (bool, error) {
	for _, commit := range f.commits {
		if commit == id {
			return true, nil
		}
	}
	return false, nil
}
func (f *assignmentCopyFake) Close(context.Context) error { f.closed = true; return nil }

type assignmentAgentsFake struct {
	mu     sync.Mutex
	opened map[string]*assignmentSessionFake
}
type assignmentSessionFake struct {
	spec      run.AssignmentSpec
	handler   run.ToolHandler
	closed    bool
	messages  []string
	onMessage chan struct{}
}

func (f *assignmentAgentsFake) OpenAssignment(_ context.Context, spec run.AssignmentSpec, handler run.ToolHandler) (run.AssignmentSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.opened == nil {
		f.opened = map[string]*assignmentSessionFake{}
	}
	session := &assignmentSessionFake{spec: spec, handler: handler}
	f.opened[spec.ID] = session
	return session, nil
}
func (f *assignmentSessionFake) Message(ctx context.Context, text string, _ time.Duration) (run.TurnResult, error) {
	if text == "wait" {
		close(f.onMessage)
		<-ctx.Done()
		return run.TurnResult{Reason: "cancelled"}, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return run.TurnResult{Reason: "cancelled"}, err
	}
	f.messages = append(f.messages, text)
	final := text
	if text == "question" {
		answer := f.handler(ctx, run.AskParentTool, json.RawMessage(`{"question":"Which contract?"}`))
		if answer.Refused {
			return run.TurnResult{}, errors.New(answer.Text)
		}
		final = answer.Text
	}
	exit := 0
	return run.TurnResult{Final: final, Completed: true, Conversation: f.spec.ID, Commands: []run.CommandObservation{{Command: "go test ./...", Status: "passed", ExitCode: &exit}}}, nil
}

func TestAssignmentConcurrentCloseKeepsAbortIncomplete(t *testing.T) {
	ctx := context.Background()
	work := &assignmentWorkspacesFake{}
	agents := &assignmentAgentsFake{}
	a, err := run.StartAssignment(ctx, work, agents, run.AssignmentRequest{ID: "parallel", Profile: "codex", Home: "home"})
	if err != nil {
		t.Fatal(err)
	}
	session := agents.opened["parallel"]
	session.onMessage = make(chan struct{})
	messageCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { _, err := a.Message(messageCtx, "wait", 0); done <- err }()
	<-session.onMessage
	closed, err := a.Close(ctx)
	if err != nil || !closed.Stopped {
		t.Fatalf("close %+v %v", closed, err)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("message: %v", err)
	}
	if _, err := a.Submit(ctx); err == nil {
		t.Fatal("cancelled work became a complete handoff")
	}
}
func (f *assignmentSessionFake) Close(context.Context) (run.TurnResult, error) {
	f.closed = true
	return run.TurnResult{Stopped: true}, nil
}

func TestAssignmentQuestionHandoffCorrectionAndIndependentCopy(t *testing.T) {
	ctx := context.Background()
	work := &assignmentWorkspacesFake{}
	agents := &assignmentAgentsFake{}
	questions := 0
	start := func(id run.WorkspaceID) *run.Assignment {
		assignment, err := run.StartAssignment(ctx, work, agents, run.AssignmentRequest{
			ID: id, Profile: "codex", Home: "home-" + string(id), Handler: func(context.Context, string, json.RawMessage) run.ToolAnswer { return run.ToolAnswer{} },
			AskParent: func(_ context.Context, q string) (string, error) {
				questions++
				if q != "Which contract?" {
					t.Fatalf("question %q", q)
				}
				return "Use the public interface", nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return assignment
	}
	first := start("agent-a")
	second := start("agent-b")
	if agents.opened["agent-a"].spec.Workspace == agents.opened["agent-b"].spec.Workspace {
		t.Fatal("shared working copy")
	}
	for _, message := range []string{"work", "question", "first delivery"} {
		if result, err := first.Message(ctx, message, time.Second); err != nil || !result.Completed || message == "question" && result.Final != "Use the public interface" {
			t.Fatalf("message %q: %+v %v", message, result, err)
		}
	}
	if questions != 1 {
		t.Fatalf("parent received %d questions, want one", questions)
	}
	firstHandoff, err := first.Submit(ctx)
	if err != nil || firstHandoff.Summary != "first delivery" || firstHandoff.StartingCommit == firstHandoff.ResultCommit {
		t.Fatalf("first handoff %+v %v", firstHandoff, err)
	}
	if _, err := first.Message(ctx, "corrected delivery", time.Second); err != nil {
		t.Fatal(err)
	}
	corrected, err := first.Submit(ctx)
	if err != nil || corrected.Summary != "corrected delivery" || corrected.ResultCommit == firstHandoff.ResultCommit {
		t.Fatalf("corrected handoff %+v %v", corrected, err)
	}
	if retained, _ := first.Workspace().ContainsCommit(ctx, firstHandoff.ResultCommit); !retained {
		t.Fatal("first submitted commit was lost")
	}
	if _, err := second.Message(ctx, "other area", time.Second); err != nil {
		t.Fatal(err)
	}
	if closed, err := first.Close(ctx); err != nil || !closed.Stopped || !agents.opened["agent-a"].closed || work.copies["agent-a"].closed {
		t.Fatal("closing Agent lost workspace or failed to stop container")
	}
	if agents.opened["agent-b"].closed {
		t.Fatal("closing one Agent stopped another")
	}
	if closed, err := second.Close(ctx); err != nil || !closed.Stopped {
		t.Fatalf("second close %+v %v", closed, err)
	}
	final, err := second.Submit(ctx)
	if err != nil || final.Summary != "other area" || final.ResultCommit == "" {
		t.Fatalf("handoff after close %+v %v", final, err)
	}
}

func TestAssignmentCancellationRetainsIncompleteWork(t *testing.T) {
	ctx := context.Background()
	work := &assignmentWorkspacesFake{}
	agents := &assignmentAgentsFake{}
	a, err := run.StartAssignment(ctx, work, agents, run.AssignmentRequest{ID: "cancelled", Profile: "codex", Home: "home", Handler: func(context.Context, string, json.RawMessage) run.ToolAnswer { return run.ToolAnswer{} }})
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := a.Message(cancelled, "unfinished", time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := a.Submit(ctx); err == nil {
		t.Fatal("incomplete provider response was submitted")
	}
	if _, err := a.Close(cancelled); err != nil || work.copies["cancelled"].closed {
		t.Fatalf("partial workspace was discarded: %v", err)
	}
}

func TestWorkspaceRequestsNameAWorkspaceAndBoundObservation(t *testing.T) {
	t.Parallel()
	if !(run.WorkspaceSpec{ID: "w"}).Valid() || (run.WorkspaceSpec{}).Valid() {
		t.Fatal("workspace spec validity")
	}
	if !(run.HarvestRequest{ID: "w", MaxEntries: 1, MaxBytes: 1}).Valid() {
		t.Fatal("bounded harvest refused")
	}
	for name, r := range map[string]run.HarvestRequest{
		"no entries":    {ID: "w", MaxBytes: 1},
		"no bytes":      {ID: "w", MaxEntries: 1},
		"huge entries":  {ID: "w", MaxEntries: 1 << 30, MaxBytes: 1},
		"huge bytes":    {ID: "w", MaxEntries: 1, MaxBytes: 1 << 40},
		"no identifier": {MaxEntries: 1, MaxBytes: 1},
	} {
		if r.Valid() {
			t.Fatal(name, "accepted")
		}
	}
	if !(run.WorkspaceHarvest{}).Complete() || (run.WorkspaceHarvest{Unsupported: []string{"link"}}).Complete() || (run.WorkspaceHarvest{Uncertain: true}).Complete() {
		t.Fatal("an incomplete harvest counted as complete")
	}
}
