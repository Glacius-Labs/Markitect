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
	opened          map[string]*scheduledSession
	repeatQuestions bool
	crossArea       bool
	cycleConsult    bool
	failID          string
	blockCloseID    string
	unblockClose    chan struct{}
}

type scheduledSession struct {
	spec            run.AssignmentSpec
	handler         run.ToolHandler
	corrected       bool
	asked           bool
	closed          bool
	questions       []string
	timeouts        []time.Duration
	correction      string
	reviews         []string
	repeatQuestions bool
	crossArea       bool
	cycleConsult    bool
	consulted       bool
	fail            bool
	unblockClose    chan struct{}
}

func (a *scheduledAgents) OpenAssignment(_ context.Context, spec run.AssignmentSpec, handler run.ToolHandler) (run.AssignmentSession, error) {
	if a.opened == nil {
		a.opened = map[string]*scheduledSession{}
	}
	s := &scheduledSession{spec: spec, handler: handler, repeatQuestions: a.repeatQuestions, crossArea: a.crossArea, cycleConsult: a.cycleConsult, fail: spec.ID == a.failID}
	if spec.ID == a.blockCloseID {
		s.unblockClose = a.unblockClose
	}
	a.opened[spec.ID] = s
	return s, nil
}

var reviewedResult = regexp.MustCompile(`result ([a-f]{40})`)

func (s *scheduledSession) Message(ctx context.Context, prompt string, timeout time.Duration) (run.TurnResult, error) {
	s.timeouts = append(s.timeouts, timeout)
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
		if s.crossArea {
			body, _ := json.Marshal(map[string]string{"child": "library/lending", "question": "What lending contract should Storage use?"})
			answer := s.handler(ctx, run.AskChildTool, body)
			if answer.Refused {
				return run.TurnResult{}, errors.New(answer.Text)
			}
			return run.TurnResult{Final: answer.Text, Completed: true}, nil
		}
		return run.TurnResult{Final: "Use the public storage contract.", Completed: true}, nil
	case strings.Contains(prompt, "Your parent library asks:"):
		s.consulted = true
		if s.cycleConsult {
			answer := s.handler(ctx, run.AskParentTool, json.RawMessage(`{"question":"Can Library decide for me?"}`))
			if answer.Refused {
				return run.TurnResult{}, errors.New(answer.Text)
			}
		}
		return run.TurnResult{Final: "Use the Lending v1 contract.", Completed: true}, nil
	case strings.Contains(prompt, "Your responsibility"):
		if s.fail {
			return run.TurnResult{Final: "failed work", Commands: []run.CommandObservation{{Command: "go test ./...", Status: "failed", Output: "assertion failed"}}}, errors.New("provider interrupted")
		}
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
			count := 1
			if s.repeatQuestions {
				count = 2
			}
			for i := 0; i < count; i++ {
				question, expected := "Which contract?", "Use the public storage contract."
				if s.crossArea {
					question, expected = "Which lending contract?", "Use the Lending v1 contract."
				}
				body, _ := json.Marshal(map[string]string{"question": question})
				answer := s.handler(ctx, run.AskParentTool, body)
				if answer.Refused || answer.Text != expected {
					return run.TurnResult{}, fmt.Errorf("parent answer: %+v", answer)
				}
				s.asked = true
			}
		}
	case strings.Contains(prompt, "Review your direct child"):
		s.reviews = append(s.reviews, prompt)
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
	exit := 0
	return run.TurnResult{Final: "Completed " + s.spec.ID, Completed: true, Commands: []run.CommandObservation{{Command: "go test ./...", Status: "passed", ExitCode: &exit}}}, nil
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
	result, _, err := run.RunAssignmentTree(ctx, team, work, agents, git, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement the model", MaxMessages: 20, Timeout: time.Second, TranscriptDir: t.TempDir()})
	if err != nil || result.ResultCommit == "" || !git.conflicted || len(git.accepted) != 3 {
		t.Fatalf("conflict correction was not integrated: %+v %v, accepted=%v", result, err, git.accepted)
	}
	if !strings.Contains(agents.opened["trial-3"].correction, "Rebase storage") {
		t.Fatal("the owning child did not receive the parent correction")
	}
}

func TestAssignmentSchedulerRejectsUnboundedInvocation(t *testing.T) {
	_, _, err := run.RunAssignmentTree(context.Background(), []run.Manager{{Path: "."}}, &assignmentWorkspacesFake{}, &scheduledAgents{}, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Work", MaxMessages: 1, Timeout: time.Second, TranscriptDir: t.TempDir()})
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unbounded assignment was allowed: %v", err)
	}
}

func (s *scheduledSession) Close(context.Context) (run.TurnResult, error) {
	if s.unblockClose != nil {
		<-s.unblockClose
	}
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
	result, flow, err := run.RunAssignmentTree(ctx, team, work, agents, git, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement the model", MaxMessages: 20, Timeout: time.Second, TranscriptDir: t.TempDir()})
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
	if !strings.Contains(agents.opened["trial-1"].reviews[0], "go test ./...: passed") {
		t.Fatal("parent did not receive observed child commands for review")
	}
	observed := flow.Observations("library/lending")
	*observed[0].Commands[0].ExitCode = 99
	if code := *flow.Observations("library/lending")[0].Commands[0].ExitCode; code != 0 {
		t.Fatalf("caller changed retained native exit code to %d", code)
	}
	for id, agent := range agents.opened {
		if !agent.closed {
			t.Fatalf("Agent %s was not closed", id)
		}
	}
}

func TestAssignmentSchedulerCountsDirectParentAnswers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}, ExecuteProfile: "exec"},
		{Path: "library/lending", Parent: "library", ExecuteProfile: "exec"},
		{Path: "library/storage", Parent: "library", ExecuteProfile: "exec"},
	}
	agents := &scheduledAgents{repeatQuestions: true}
	_, flow, err := run.RunAssignmentTree(ctx, team, &assignmentWorkspacesFake{}, agents, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement", MaxMessages: 8, Timeout: time.Second, TranscriptDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "assignment message allowance exhausted") {
		t.Fatalf("repeated question bypassed common allowance: %v", err)
	}
	if got := len(agents.opened["trial-1"].questions); got != 1 {
		t.Fatalf("parent answered %d questions beyond allowance", got)
	}
	for _, timeout := range agents.opened["trial-1"].timeouts {
		if timeout != time.Second {
			t.Fatalf("parent answer used a different timeout: %s", timeout)
		}
	}
	if flow.Workspace("library/storage") == nil || len(flow.Observations("library/storage")) != 1 {
		t.Fatal("budget failure lost the child's retained copy or partial result")
	}
}

func TestAssignmentSchedulerConsultsSiblingThroughCommonParent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}, ExecuteProfile: "exec"},
		{Path: "library/lending", Parent: "library", ExecuteProfile: "exec"},
		{Path: "library/storage", Parent: "library", ExecuteProfile: "exec"},
	}
	agents := &scheduledAgents{crossArea: true}
	result, flow, err := run.RunAssignmentTree(ctx, team, &assignmentWorkspacesFake{}, agents, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement", MaxMessages: 20, Timeout: time.Second, TranscriptDir: t.TempDir()})
	if err != nil || result.ResultCommit == "" || !agents.opened["trial-2"].consulted || !agents.opened["trial-3"].asked {
		t.Fatalf("cross-area question was not routed through Library to Lending and back: %+v %v", result, err)
	}
	observed := flow.Observations("library/lending")
	if len(observed) != 3 || observed[2].Final != "Use the Lending v1 contract." {
		t.Fatalf("Lending consultation was not retained as an Agent message: %+v", observed)
	}
	agents = &scheduledAgents{crossArea: true}
	_, _, err = run.RunAssignmentTree(ctx, team, &assignmentWorkspacesFake{}, agents, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement", MaxMessages: 8, Timeout: time.Second, TranscriptDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "assignment message allowance exhausted") {
		t.Fatalf("sibling consultation bypassed common allowance: %v", err)
	}
	agents = &scheduledAgents{crossArea: true, cycleConsult: true}
	_, _, err = run.RunAssignmentTree(ctx, team, &assignmentWorkspacesFake{}, agents, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement", MaxMessages: 20, Timeout: time.Second, TranscriptDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "already answering in this question chain") {
		t.Fatalf("consultation recursed into its waiting parent: %v", err)
	}
}

func TestAssignmentSchedulerRetainsFailedChildCopyAndObservation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}, ExecuteProfile: "exec"},
		{Path: "library/lending", Parent: "library", ExecuteProfile: "exec"},
		{Path: "library/storage", Parent: "library", ExecuteProfile: "exec"},
	}
	agents := &scheduledAgents{failID: "trial-2"}
	_, flow, err := run.RunAssignmentTree(ctx, team, &assignmentWorkspacesFake{}, agents, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement", MaxMessages: 8, Timeout: time.Second, TranscriptDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "provider interrupted") {
		t.Fatalf("child failure was hidden: %v", err)
	}
	if flow.Workspace("library/lending") == nil || flow.Workspace("library/lending").StartingCommit() == "" || flow.Transcript("library/lending") != agents.opened["trial-2"].spec.Transcript {
		t.Fatal("failed child's copy or transcript path is not findable")
	}
	observed := flow.Observations("library/lending")
	if len(observed) != 1 || observed[0].Commands[0].Output != "assertion failed" {
		t.Fatalf("partial command observation was discarded: %+v", observed)
	}
	for id, agent := range agents.opened {
		if !agent.closed {
			t.Fatalf("failed run left Agent %s open", id)
		}
	}
}

func TestAssignmentSchedulerAttemptsOtherStopsWhenOneBlocks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}, ExecuteProfile: "exec"},
		{Path: "library/lending", Parent: "library", ExecuteProfile: "exec"},
		{Path: "library/storage", Parent: "library", ExecuteProfile: "exec"},
	}
	unblock := make(chan struct{})
	defer close(unblock)
	agents := &scheduledAgents{blockCloseID: "trial-0", unblockClose: unblock}
	result, flow, err := run.RunAssignmentTree(ctx, team, &assignmentWorkspacesFake{}, agents, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement", MaxMessages: 20, Timeout: 50 * time.Millisecond, TranscriptDir: t.TempDir()})
	if result.ResultCommit == "" || err == nil || !strings.Contains(err.Error(), "close outcome uncertain") {
		t.Fatalf("blocking stop was not reported with retained result: %+v %v", result, err)
	}
	if !agents.opened["trial-1"].closed || !agents.opened["trial-2"].closed || !agents.opened["trial-3"].closed || flow.Workspace(".") == nil {
		t.Fatal("blocked Root stop prevented sibling stop or lost its copy")
	}
}
