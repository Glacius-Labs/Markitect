package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type queuedAgents struct {
	*concurrentAgents
	reviewGate    <-chan struct{}
	replyGate     <-chan struct{}
	questions     chan string
	resumed       atomic.Bool
	nativeCalls   atomic.Int32
	queued        atomic.Int32
	uncooperative bool
	queryReturned chan struct{}
}
type queuedSession struct {
	*concurrentSession
	owner *queuedAgents
}

func (a *queuedAgents) OpenAssignment(ctx context.Context, spec run.AssignmentSpec, h run.ToolHandler) (run.AssignmentSession, error) {
	s, err := a.concurrentAgents.OpenAssignment(ctx, spec, h)
	if err != nil {
		return nil, err
	}
	return &queuedSession{s.(*concurrentSession), a}, nil
}
func (s *queuedSession) Message(ctx context.Context, prompt string, timeout time.Duration) (run.TurnResult, error) {
	a := s.owner
	a.nativeCalls.Add(1)
	if timeout != time.Second {
		return run.TurnResult{}, errors.New("consultation/resume changed message timeout")
	}
	call := func(name string, args map[string]string) error {
		raw, _ := json.Marshal(args)
		answer := s.handler(ctx, name, raw)
		if answer.Refused {
			return errors.New(answer.Text)
		}
		if name == run.AskChildTool {
			if !strings.Contains(answer.Text, "queued") {
				return errors.New("consultation did not return immediate queue acknowledgement")
			}
			a.queued.Add(1)
		}
		return nil
	}
	if s.manager.Path == "library" && strings.HasPrefix(prompt, "Review your direct child library/second") {
		defer a.enterParent()()
		if !strings.Contains(prompt, "Queued child answer from") {
			a.reviewEntered <- "library/second"
			if err := waitGate(ctx, a.reviewGate); err != nil {
				return run.TurnResult{}, err
			}
			for _, q := range []string{"Q1", "Q2"} {
				if err := call(run.AskChildTool, map[string]string{"child": "library/first", "question": q}); err != nil {
					return run.TurnResult{}, err
				}
			}
			// Even a provisional decision must wait for the queued answers.
			if err := call(run.AcceptChildTool, map[string]string{"child": "library/second"}); err != nil {
				return run.TurnResult{}, err
			}
			return run.TurnResult{Completed: true, Final: "Questions queued; awaiting replies"}, nil
		}
		a.resumed.Store(true)
		if !strings.Contains(prompt, "A1") || !strings.Contains(prompt, "A2") {
			return run.TurnResult{}, errors.New("parent resumed without both ordered replies")
		}
		return run.TurnResult{Completed: true, Final: "Reviewed with both child replies"}, call(run.AcceptChildTool, map[string]string{"child": "library/second"})
	}
	if s.manager.Path == "library/first" && strings.HasPrefix(prompt, "Your parent library asks: Q") {
		question, answer := "Q1", "A1"
		if strings.HasPrefix(prompt, "Your parent library asks: Q2") {
			question, answer = "Q2", "A2"
		}
		a.questions <- question
		if question == "Q2" {
			if a.uncooperative {
				<-a.replyGate
				close(a.queryReturned)
			} else if err := waitGate(ctx, a.replyGate); err != nil {
				return run.TurnResult{Final: "partial queued reply"}, err
			}
		}
		return run.TurnResult{Completed: true, Final: answer}, nil
	}
	return s.concurrentSession.Message(ctx, prompt, timeout)
}

func TestAssignmentBusyChildQuestionsQueueInOrderAndResumeWithoutParentLock(t *testing.T) {
	for _, mode := range []string{"success", "cancel", "close", "close blocked", "allowance"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, base := concurrentFixture()
			ask, firstWork, review, reply := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
			base.askNow = ask
			base.workGates["library/first"] = firstWork
			agents := &queuedAgents{concurrentAgents: base, reviewGate: review, replyGate: reply, questions: make(chan string, 4), uncooperative: mode == "close blocked", queryReturned: make(chan struct{})}
			git := &concurrentIntegration{agents: base, entered: make(chan string, 8), gates: map[string]<-chan struct{}{}}
			options := run.AssignmentSchedule{IDPrefix: "fan", Task: "Implement", MaxMessages: 40, Timeout: time.Second, TranscriptDir: t.TempDir()}
			if mode == "allowance" {
				options.MaxMessages = 8
			}
			done := make(chan concurrentOutcome, 1)
			var tree *run.AssignmentTree
			if strings.HasPrefix(mode, "close") {
				options.CreateHome = func(context.Context, string) error { return nil }
				var err error
				tree, err = run.OpenAssignmentTree(ctx, base.team, &assignmentWorkspacesFake{}, agents, git, options)
				if err != nil {
					t.Fatal(err)
				}
				go func() { result, err := tree.Implement(ctx); done <- concurrentOutcome{result, tree.Flow, err} }()
			} else {
				go func() {
					result, flow, err := run.RunAssignmentTree(ctx, base.team, &assignmentWorkspacesFake{}, agents, git, options)
					done <- concurrentOutcome{result, flow, err}
				}()
			}
			receivePath(t, base.reviewEntered, "library/second")
			close(ask) // First is busy, and now asks the same parent reviewing Second.
			select {
			case <-base.questionCalled:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			close(review)
			receivePath(t, base.questionEntered, "library") // Parent lock was released.
			select {
			case <-base.questionAnswered:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			close(firstWork)
			receivePath(t, agents.questions, "Q1")
			receivePath(t, agents.questions, "Q2")
			if len(git.entered) != 0 || agents.resumed.Load() || base.parentBusy.Load() != 0 {
				t.Fatal("provisional acceptance applied or parent queue remained held while waiting")
			}
			switch mode {
			case "cancel":
				cancel()
			case "close", "close blocked":
				closeErr := tree.Close()
				if mode == "close blocked" {
					if closeErr == nil || !strings.Contains(closeErr.Error(), "question cleanup remains uncertain") {
						t.Fatalf("blocked consultation cleanup claimed settled: %v", closeErr)
					}
				} else if closeErr != nil {
					t.Fatal(closeErr)
				}
			default:
				close(reply)
			}
			outcome := <-done
			if mode == "close blocked" {
				close(reply)
				<-agents.queryReturned
			}
			if mode == "success" {
				if outcome.err != nil || outcome.result.ResultCommit == "" || !agents.resumed.Load() {
					t.Fatalf("queued consultation did not complete: %v", outcome.err)
				}
				if strings.Join(git.order, ",") != "library/second,library/first,library" {
					t.Fatalf("handoff review order changed: %v", git.order)
				}
			} else {
				if outcome.err == nil || outcome.result.ResultCommit != "" || agents.resumed.Load() || len(git.order) != 0 {
					t.Fatalf("cancel/close/allowance accepted or resumed pending review: %v", outcome.err)
				}
				if agents.nativeCalls.Load() != 8 {
					t.Fatalf("cancel/Close/budget allowed another provider Message: %d", agents.nativeCalls.Load())
				}
				if mode == "allowance" && (agents.nativeCalls.Load() != 8 || !strings.Contains(outcome.err.Error(), "allowance exhausted")) {
					t.Fatalf("consultation/resume escaped shared allowance: %d %v", agents.nativeCalls.Load(), outcome.err)
				}
			}
			if agents.queued.Load() != 2 || base.overlapped.Load() || len(base.stopped) != 4 || outcome.flow.Workspace("library/first") == nil {
				t.Fatal("queue lost order, parent serialization, cleanup or retained child copy")
			}
		})
	}
}
