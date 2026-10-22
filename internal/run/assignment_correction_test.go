package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type finalCorrectionAgents struct{ scheduledAgents }
type finalCorrectionSession struct{ *scheduledSession }

func (a *finalCorrectionAgents) OpenAssignment(ctx context.Context, spec run.AssignmentSpec, h run.ToolHandler) (run.AssignmentSession, error) {
	s, err := a.scheduledAgents.OpenAssignment(ctx, spec, h)
	if err != nil {
		return nil, err
	}
	return &finalCorrectionSession{s.(*scheduledSession)}, nil
}
func (s *finalCorrectionSession) Message(ctx context.Context, prompt string, timeout time.Duration) (run.TurnResult, error) {
	var child, task string
	switch {
	case strings.HasPrefix(prompt, "Independent Verify returned"):
		child, task = "library", "Route final lending defect"
	case prompt == "Route final lending defect":
		child, task = "library/lending", "Repair final lending defect"
	case prompt == "Repair final lending defect":
		s.correction = prompt
		return run.TurnResult{Completed: true, Final: "Corrected the actual loan-limit defect and tested the boundary."}, nil
	default:
		return s.scheduledSession.Message(ctx, prompt, timeout)
	}
	body, _ := json.Marshal(map[string]string{"child": child, "correction": task})
	if answer := s.handler(ctx, run.ReturnChildTool, body); answer.Refused {
		return run.TurnResult{}, errors.New(answer.Text)
	}
	return run.TurnResult{Completed: true, Final: "The owning child must correct the model-derived lending defect."}, nil
}

func TestOpenAssignmentTreeRoutesFinalCorrectionWithoutRestartingUnaffectedAgents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}, ExecuteProfile: "exec"},
		{Path: "library/lending", Parent: "library", ExecuteProfile: "exec"},
		{Path: "library/storage", Parent: "library", ExecuteProfile: "exec"},
	}
	work, agents, git := &assignmentWorkspacesFake{}, &finalCorrectionAgents{}, &handoffIntegrationFake{}
	homes := make(map[string]bool)
	tree, err := run.OpenAssignmentTree(ctx, team, work, agents, git, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement the fixed model", MaxMessages: 30, Timeout: time.Second, TranscriptDir: t.TempDir(), CreateHome: func(_ context.Context, home string) error {
		if homes[home] {
			return errors.New("home already exists")
		}
		homes[home] = true
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	first, err := tree.Implement(ctx)
	if err != nil {
		t.Fatal(err)
	}
	storageBefore := tree.Flow.Observations("library/storage")
	storageCopy := tree.Flow.Workspace("library/storage")
	for id, s := range agents.opened {
		if s.closed {
			t.Fatalf("%s closed before independent Verify could return a correction", id)
		}
	}
	corrected, err := tree.Correct(ctx, "lending/limit: the public borrowing boundary still violates the modeled limit.")
	if err != nil {
		t.Fatal(err)
	}
	if first.ResultCommit == corrected.ResultCommit || len(agents.opened) != 4 || work.forks != 3 || len(homes) != 4 || len(tree.Homes()) != 4 {
		t.Fatal("final correction restarted assignments or failed to produce a new Root result")
	}
	if tree.Flow.Workspace("library/storage") != storageCopy || len(tree.Flow.Observations("library/storage")) != len(storageBefore) {
		t.Fatal("unaffected Storage was rerun or replaced")
	}
	if agents.opened["trial-2"].correction != "Repair final lending defect" || len(git.accepted) != 5 {
		t.Fatalf("correction was not reviewed/integrated upward: %v", git.accepted)
	}
	if err := tree.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := tree.Correct(ctx, "another defect"); err == nil {
		t.Fatal("corrected a closed assignment")
	}
	for id, s := range agents.opened {
		if !s.closed {
			t.Fatalf("%s was not closed", id)
		}
	}
}

func TestOpenAssignmentTreeRetainsHomeFailureWithoutOpeningAnAgent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	agents := &finalCorrectionAgents{}
	tree, err := run.OpenAssignmentTree(ctx, []run.Manager{{Path: ".", ExecuteProfile: "exec"}}, &assignmentWorkspacesFake{}, agents, &handoffIntegrationFake{}, run.AssignmentSchedule{IDPrefix: "trial", Task: "Implement", MaxMessages: 2, Timeout: time.Second, TranscriptDir: t.TempDir(), CreateHome: func(context.Context, string) error { return errors.New("home already exists") }})
	if err == nil || tree == nil || len(tree.Homes()) != 1 || len(agents.opened) != 0 {
		t.Fatalf("home failure was lost: %v", err)
	}
	if err := tree.Close(); err != nil {
		t.Fatal(err)
	}
}
