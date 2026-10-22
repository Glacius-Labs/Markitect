package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type reviewCopy struct {
	id   run.WorkspaceID
	head string
}

func (w *reviewCopy) ID() run.WorkspaceID                            { return w.id }
func (w *reviewCopy) Volume() string                                 { return string(w.id) }
func (w *reviewCopy) StartingCommit() string                         { return w.head }
func (w *reviewCopy) Commit(context.Context, string) (string, error) { return w.head, nil }
func (w *reviewCopy) ContainsCommit(_ context.Context, id string) (bool, error) {
	return id == w.head, nil
}
func (w *reviewCopy) Close(context.Context) error { return nil }

type reviewCopies struct {
	forkErr error
	forked  bool
}

func freshReviewHome(context.Context, string) error { return nil }

func TestFocusedReviewRefusesExistingHomeBeforeOpeningAgent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commit := strings.Repeat("a", 40)
	agent := &reviewAgent{}
	result, err := run.ReviewAssignment(ctx, &reviewCopies{}, agent, run.AssignmentReview{CreateHome: func(context.Context, string) error { return errors.New("volume already exists") }, ID: "review", Purpose: run.Verify, Source: &reviewCopy{id: "source", head: commit}, Commit: commit, Profile: "review", Context: "Fixed model", Timeout: time.Second, Transcript: filepath.Join(t.TempDir(), "log.jsonl")})
	if err == nil || agent.spec.ID != "" || result.Home != "review-home" || result.HomeReady || result.Workspace == nil || result.AcceptedCommit != "" {
		t.Fatalf("stale home: %+v %v", result, err)
	}
}

func TestFocusedReviewRejectsLateDecisionAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commit := strings.Repeat("a", 40)
	agent := &reviewAgent{stopped: true}
	agent.decide = func(context.Context, run.ToolHandler) { cancel() }
	result, err := run.ReviewAssignment(ctx, &reviewCopies{}, agent, run.AssignmentReview{CreateHome: freshReviewHome, ID: "review", Purpose: run.Verify, Source: &reviewCopy{id: "source", head: commit}, Commit: commit, Profile: "review", Context: "Fixed model", Timeout: time.Second, Transcript: filepath.Join(t.TempDir(), "log.jsonl")})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	late := make(chan run.ToolAnswer, 1)
	go func() {
		late <- agent.handle(context.Background(), run.ModelProblemTool, json.RawMessage(`{"owner":".","obligation":"model/rule","expected":"fixed","observed":"late problem"}`))
	}()
	if answer := <-late; !answer.Refused || len(result.Issues) != 0 || result.AcceptedCommit != "" {
		t.Fatalf("late decision changed a retained result: %+v %+v", answer, result)
	}
}

func (w *reviewCopies) PrepareAssignment(context.Context, run.WorkspaceSpec) (run.AssignmentWorkspace, error) {
	return nil, errors.New("review must fork the fixed Git result")
}
func (w *reviewCopies) ForkAssignmentCommit(_ context.Context, source run.AssignmentWorkspace, commit string, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	w.forked = true
	return &reviewCopy{id: id, head: commit}, w.forkErr
}

type reviewAgent struct {
	spec      run.AssignmentSpec
	handle    run.ToolHandler
	decide    func(context.Context, run.ToolHandler)
	closeErr  error
	stopped   bool
	prompt    string
	code      int
	closeWait chan struct{}
}

func (a *reviewAgent) OpenAssignment(_ context.Context, spec run.AssignmentSpec, handle run.ToolHandler) (run.AssignmentSession, error) {
	a.spec, a.handle = spec, handle
	return a, nil
}
func (a *reviewAgent) Message(ctx context.Context, prompt string, _ time.Duration) (run.TurnResult, error) {
	a.prompt = prompt
	if a.decide != nil {
		a.decide(ctx, a.handle)
	}
	return run.TurnResult{Completed: true, Final: "Ordinary prose summary with no status word or JSON.", Commands: []run.CommandObservation{{Command: "go test ./...", Status: "exited", ExitCode: &a.code, Output: "actual output"}}}, nil
}
func (a *reviewAgent) Close(context.Context) (run.TurnResult, error) {
	if a.closeWait != nil {
		<-a.closeWait
	}
	return run.TurnResult{Stopped: a.stopped}, a.closeErr
}

func TestFocusedReviewReturnsRetainedWorkWhenCloseIgnoresDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commit := strings.Repeat("a", 40)
	agent := &reviewAgent{stopped: true, closeWait: make(chan struct{})}
	defer close(agent.closeWait)
	agent.decide = func(ctx context.Context, h run.ToolHandler) {
		body, _ := json.Marshal(map[string]string{"commit": commit, "reason": "reviewed"})
		h(ctx, run.AcceptResultTool, body)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		result, err := run.ReviewAssignment(ctx, &reviewCopies{}, agent, run.AssignmentReview{CreateHome: freshReviewHome, ID: "review", Purpose: run.Verify, Source: &reviewCopy{id: "source", head: commit}, Commit: commit, Profile: "review", Context: "Fixed model", Timeout: 20 * time.Millisecond, Transcript: filepath.Join(t.TempDir(), "log.jsonl")})
		if err == nil || !strings.Contains(err.Error(), "uncertain") || result.AcceptedCommit != "" || result.Workspace == nil || result.Handoff.ResultCommit == "" || result.Closure.Stopped {
			t.Errorf("blocked close: %+v %v", result, err)
		}
	}()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("review could not return its incomplete work")
	}
}

func TestFocusedReviewReceivesProseButNeedsCommitBoundJudgment(t *testing.T) {
	commit := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name, target    string
		decide, stopped bool
		wantAccepted    bool
	}{
		{"accepted", commit, true, true, true},
		{"wrong commit", strings.Repeat("b", 40), true, true, false},
		{"completion is not acceptance", commit, false, true, false},
		{"unknown stop", commit, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			copies, agent := &reviewCopies{}, &reviewAgent{stopped: tc.stopped}
			if tc.decide {
				agent.decide = func(ctx context.Context, h run.ToolHandler) {
					body, _ := json.Marshal(map[string]string{"commit": tc.target, "reason": "Inspected assertions and the model."})
					h(ctx, run.AcceptResultTool, body)
				}
			}
			result, err := run.ReviewAssignment(ctx, copies, agent, run.AssignmentReview{CreateHome: freshReviewHome, ID: "verify", Purpose: run.Verify, Source: &reviewCopy{id: "implement", head: commit}, Commit: commit, Profile: "review", Context: "Model: preserve the loan limit. Child claims require independent assessment.", Timeout: time.Second, Transcript: filepath.Join(t.TempDir(), "verify.jsonl")})
			if (result.AcceptedCommit == commit) != tc.wantAccepted {
				t.Fatalf("acceptance: %+v error %v", result, err)
			}
			if tc.wantAccepted && err != nil {
				t.Fatal(err)
			}
			if !tc.wantAccepted && err == nil {
				t.Fatal("unresolved review did not return its incomplete error")
			}
			if result.Handoff.Summary == "" || result.Workspace == nil || len(result.Observation.Commands) != 1 {
				t.Fatal("incomplete review lost its received work or actual observations")
			}
			if !copies.forked || agent.spec.Home != "verify-home" || agent.spec.Workspace == "implement" {
				t.Fatal("review reused the implementation copy or home")
			}
			if !strings.Contains(agent.prompt, "actual assertions") || !strings.Contains(agent.prompt, commit) {
				t.Fatal("focused Verify did not receive its mandate or exact commit")
			}
			agent.code = 99
			if *result.Observation.Commands[0].ExitCode != 0 {
				t.Fatal("native observation was not retained independently")
			}
		})
	}
}

func TestFocusedReviewRetainsRequiredMattersAndRejectsContradictoryAccept(t *testing.T) {
	for _, purpose := range []run.Purpose{run.Assess, run.Verify} {
		t.Run(string(purpose), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			commit := strings.Repeat("a", 40)
			agent := &reviewAgent{stopped: true, code: 1}
			agent.decide = func(ctx context.Context, h run.ToolHandler) {
				name, accept := run.RequestCorrectionTool, run.AcceptResultTool
				if purpose == run.Assess {
					name, accept = run.ModelProblemTool, run.AcceptModelTool
				}
				issue, _ := json.Marshal(run.ReviewIssue{Owner: "lending", Obligation: "lending/limit", Expected: "The public boundary must settle the permitted loan limit.", Observed: "The model requires a limit but neither fixes it nor delegates its choice."})
				if answer := h(ctx, name, issue); answer.Refused {
					t.Fatal(answer.Text)
				}
				body, _ := json.Marshal(map[string]string{"commit": commit, "reason": "green suite"})
				if answer := h(ctx, accept, body); !answer.Refused {
					t.Fatal("accepted despite an open required matter")
				}
			}
			result, err := run.ReviewAssignment(ctx, &reviewCopies{}, agent, run.AssignmentReview{CreateHome: freshReviewHome, ID: "review", Purpose: purpose, Source: &reviewCopy{id: "source", head: commit}, Commit: commit, Profile: "review", Context: "Fixed model", Timeout: time.Second, Transcript: filepath.Join(t.TempDir(), "log.jsonl")})
			if err != nil || len(result.Issues) != 1 || result.AcceptedCommit != "" || result.ModelProblem != (purpose == run.Assess) {
				t.Fatalf("required matter: %+v %v", result, err)
			}
			if *result.Observation.Commands[0].ExitCode != 1 {
				t.Fatal("failed ordinary command observation was lost")
			}
		})
	}
}

func TestFocusedReviewRefusesDriftAndRetainsPartialFork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	options := run.AssignmentReview{CreateHome: freshReviewHome, ID: "review", Purpose: run.Verify, Source: &reviewCopy{id: "source", head: "new"}, Commit: "old", Profile: "review", Context: "Fixed model", Timeout: time.Second, Transcript: filepath.Join(t.TempDir(), "log.jsonl")}
	copies := &reviewCopies{}
	if _, err := run.ReviewAssignment(ctx, copies, &reviewAgent{}, options); err == nil || copies.forked {
		t.Fatal("stale source reached a reviewer")
	}
	options.Commit = "new"
	copies.forkErr = errors.New("partial fork")
	result, err := run.ReviewAssignment(ctx, copies, &reviewAgent{}, options)
	if err == nil || result.Workspace == nil || result.AcceptedCommit != "" {
		t.Fatal("partial copy was lost or accepted")
	}
}
