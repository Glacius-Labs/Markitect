package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type scheduledAgents struct {
	opened map[string]*scheduledSession
}

type scheduledSession struct {
	spec       run.AssignmentSpec
	handler    run.ToolHandler
	corrected  bool
	asked      bool
	closed     bool
	questions  []string
	correction string
}

func (a *scheduledAgents) OpenAssignment(_ context.Context, spec run.AssignmentSpec, handler run.ToolHandler) (run.AssignmentSession, error) {
	if a.opened == nil {
		a.opened = map[string]*scheduledSession{}
	}
	s := &scheduledSession{spec: spec, handler: handler}
	a.opened[spec.ID] = s
	return s, nil
}

var reviewedResult = regexp.MustCompile(`result ([a-f]{40})`)

func (s *scheduledSession) Message(ctx context.Context, prompt string, _ time.Duration) (run.TurnResult, error) {
	call := func(name string, args any) error {
		body, _ := json.Marshal(args)
		answer := s.handler(ctx, name, body)
		if answer.Refused {
			return fmt.Errorf("%s: %s", name, answer.Text)
		}
		return nil
	}
	switch {
	case strings.Contains(prompt, "Your direct child library/storage asks:"):
		s.questions = append(s.questions, prompt)
		return run.TurnResult{Final: "Use the public storage contract.", Completed: true}, nil
	case strings.Contains(prompt, "Your responsibility"):
		switch s.spec.ID {
		case "trial-0":
			if err := call(run.DelegateChildTool, map[string]string{"child": "library", "task": "Build the public application"}); err != nil {
				return run.TurnResult{}, err
			}
		case "trial-1":
			for _, child := range []string{"library/lending", "library/storage"} {
				if err := call(run.DelegateChildTool, map[string]string{"child": child, "task": "Build " + child}); err != nil {
					return run.TurnResult{}, err
				}
			}
		case "trial-3":
			answer := s.handler(ctx, run.AskParentTool, json.RawMessage(`{"question":"Which contract?"}`))
			if answer.Refused || answer.Text != "Use the public storage contract." {
				return run.TurnResult{}, fmt.Errorf("parent answer: %+v", answer)
			}
			s.asked = true
		}
	case strings.Contains(prompt, "Review your direct child"):
		found := reviewedResult.FindStringSubmatch(prompt)
		if len(found) != 2 {
			return run.TurnResult{}, fmt.Errorf("no fixed result in review prompt")
		}
		child := "library"
		if strings.Contains(prompt, "library/lending") {
			child = "library/lending"
		}
		if strings.Contains(prompt, "library/storage") {
			child = "library/storage"
		}
		if child == "library/lending" && !s.corrected {
			s.corrected = true
			if err := call(run.ReturnChildTool, map[string]string{"child": child, "correction": "Repair the lending limit test"}); err != nil {
				return run.TurnResult{}, err
			}
		} else if err := call(run.AcceptChildTool, map[string]string{"child": child, "commit": found[1]}); err != nil {
			return run.TurnResult{}, err
		}
	case strings.Contains(prompt, "Git found a real conflict"):
		if err := call(run.ReturnChildTool, map[string]string{"child": "library/storage", "correction": "Rebase storage against the current parent interface"}); err != nil {
			return run.TurnResult{}, err
		}
	case strings.Contains(prompt, "Repair the lending limit test"), strings.Contains(prompt, "Rebase storage against"):
		s.correction = prompt
	}
	return run.TurnResult{Final: "Completed " + s.spec.ID, Completed: true}, nil
}

type conflictOnceIntegration struct {
	handoffIntegrationFake
	conflicted bool
}

func (g *conflictOnceIntegration) Integrate(ctx context.Context, parent, child run.AssignmentWorkspace, handoff run.AssignmentHandoff, previous string) (string, error) {
	if child.ID() == "trial-3" && !g.conflicted {
		g.conflicted = true
		return "", run.ErrIntegrationConflict
	}
	return g.handoffIntegrationFake.Integrate(ctx, parent, child, handoff, previous)
}

func TestAssignmentSchedulerRoutesGitConflictToOwningChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}, ExecuteProfile: "exec"},
		{Path: "library/lending", Parent: "library", ExecuteProfile: "exec"},
		{Path: "library/storage", Parent: "library", ExecuteProfile: "exec"},
	}
	work, agents, git := &assignmentWorkspacesFake{}, &scheduledAgents{}, &conflictOnceIntegration{}
	result, _, err := run.RunAssignmentTree(ctx, team, work, agents, git, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement the model", MaxMessages: 20, Timeout: time.Second})
	if err != nil || result.ResultCommit == "" || !git.conflicted || len(git.accepted) != 3 {
		t.Fatalf("conflict correction was not integrated: %+v %v, accepted=%v", result, err, git.accepted)
	}
	if !strings.Contains(agents.opened["trial-3"].correction, "Rebase storage") {
		t.Fatal("the owning child did not receive the parent correction")
	}
}

func TestAssignmentSchedulerRejectsUnboundedInvocation(t *testing.T) {
	_, _, err := run.RunAssignmentTree(context.Background(), []run.Manager{{Path: "."}}, &assignmentWorkspacesFake{}, &scheduledAgents{}, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Work", MaxMessages: 1, Timeout: time.Second})
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unbounded assignment was allowed: %v", err)
	}
}

func (s *scheduledSession) Close(context.Context) (run.TurnResult, error) {
	s.closed = true
	return run.TurnResult{Stopped: true}, nil
}

func TestAssignmentSchedulerDelegatesAnswersReviewsReturnsAndIntegrates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, Role: "project", ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}, Role: "application", ExecuteProfile: "exec"},
		{Path: "library/lending", Parent: "library", Role: "loans", ExecuteProfile: "exec"},
		{Path: "library/storage", Parent: "library", Role: "storage", ExecuteProfile: "exec"},
	}
	work, agents, git := &assignmentWorkspacesFake{}, &scheduledAgents{}, &handoffIntegrationFake{}
	result, flow, err := run.RunAssignmentTree(ctx, team, work, agents, git, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement the model", MaxMessages: 20, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResultCommit == "" || result.Summary != "Completed trial-0" || flow == nil {
		t.Fatalf("Root handoff: %+v", result)
	}
	if work.forks != 3 || len(git.accepted) != 3 {
		t.Fatalf("not every required result was integrated: forks=%d accepted=%v", work.forks, git.accepted)
	}
	if !agents.opened["trial-3"].asked || len(agents.opened["trial-1"].questions) != 1 {
		t.Fatal("storage question did not reach its direct parent")
	}
	if agents.opened["trial-2"].correction == "" {
		t.Fatal("lending correction was not delivered to its existing Agent")
	}
	for id, agent := range agents.opened {
		if !agent.closed {
			t.Fatalf("Agent %s was not closed", id)
		}
	}
}
