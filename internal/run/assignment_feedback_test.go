package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

const run04OwnCommit = "f830d7600000000000000000000000000000000000"
const run04Handoff = "95838480000000000000000000000000000000000"

type fixedHandoffCopy struct {
	run.AssignmentWorkspace
	raw *assignmentCopyFake
}

func (c *fixedHandoffCopy) Commit(ctx context.Context, message string) (string, error) {
	if message == "Agent assignment result" {
		c.raw.commits = append(c.raw.commits, run04Handoff)
		return run04Handoff, nil
	}
	return c.AssignmentWorkspace.Commit(ctx, message)
}

type fixedHandoffCopies struct{ assignmentWorkspacesFake }

func (w *fixedHandoffCopies) ForkAssignmentCommit(ctx context.Context, source run.AssignmentWorkspace, commit string, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	copy, err := w.assignmentWorkspacesFake.ForkAssignmentCommit(ctx, source, commit, id)
	if err == nil && id == "fan-2" {
		raw := copy.(*assignmentCopyFake)
		raw.commits = append(raw.commits, run04OwnCommit)
		return &fixedHandoffCopy{copy, raw}, nil
	}
	return copy, err
}
func (w *fixedHandoffCopies) UpdateAssignment(ctx context.Context, target, source run.AssignmentWorkspace, commit, previous string) error {
	if c, ok := target.(*fixedHandoffCopy); ok {
		target = c.AssignmentWorkspace
	}
	return w.assignmentWorkspacesFake.UpdateAssignment(ctx, target, source, commit, previous)
}

type feedbackAgents struct {
	*concurrentAgents
	mode            string
	reviewCalls     atomic.Int32
	delegationCalls atomic.Int32
	feedbackSeen    atomic.Bool
	commitRefused   atomic.Bool
	leafPrompts     atomic.Int32
	nativeCalls     atomic.Int32
}
type feedbackSession struct {
	*concurrentSession
	owner *feedbackAgents
}

func (a *feedbackAgents) OpenAssignment(ctx context.Context, spec run.AssignmentSpec, h run.ToolHandler) (run.AssignmentSession, error) {
	base, err := a.concurrentAgents.OpenAssignment(ctx, spec, h)
	if err != nil {
		return nil, err
	}
	for _, tool := range spec.Tools {
		if tool.Name == run.AcceptChildTool {
			var schema struct {
				Properties map[string]any `json:"properties"`
			}
			if json.Unmarshal(tool.InputSchema, &schema) != nil {
				return nil, errors.New("invalid acceptance schema")
			}
			if _, exists := schema.Properties["commit"]; exists {
				return nil, errors.New("accept_child still offers model commit selection")
			}
		}
	}
	return &feedbackSession{base.(*concurrentSession), a}, nil
}
func (s *feedbackSession) Message(ctx context.Context, prompt string, timeout time.Duration) (run.TurnResult, error) {
	a, path := s.owner, s.manager.Path
	a.nativeCalls.Add(1)
	if timeout != time.Second {
		return run.TurnResult{}, errors.New("decision feedback changed the existing timeout")
	}
	call := func(name string, args map[string]string) run.ToolAnswer {
		raw, _ := json.Marshal(args)
		return s.handler(ctx, name, raw)
	}
	if strings.HasPrefix(prompt, "Your responsibility") && len(s.manager.Children) > 0 {
		if !strings.Contains(prompt, "separate Markitect Manager with its own container and working copy") || !strings.Contains(prompt, "Only delegate_child delegates through the Markitect Host") || !strings.Contains(prompt, "do not substitute a Codex") {
			return run.TurnResult{}, errors.New("required Markitect child confused with native provider subagent")
		}
	}
	if strings.HasPrefix(prompt, "Your responsibility") && len(s.manager.Children) == 0 {
		if strings.Contains(prompt, "delegate_child") || strings.Contains(prompt, "sibling interfaces") || strings.Contains(prompt, "Required direct children") {
			return run.TurnResult{}, errors.New("leaf received parent delegation mandate")
		}
		for _, tool := range s.spec.Tools {
			if tool.Name == run.DelegateChildTool || tool.Name == run.AcceptChildTool || tool.Name == run.ReturnChildTool || tool.Name == run.AskChildTool {
				return run.TurnResult{}, errors.New("leaf received child-coordination tools")
			}
		}
		a.leafPrompts.Add(1)
	}
	if path == "library" && strings.HasPrefix(prompt, "Your responsibility") && a.mode == "delegation" {
		n := a.delegationCalls.Add(1)
		if n == 1 {
			call(run.DelegateChildTool, map[string]string{"child": "library/first", "task": "first"})
			call(run.DelegateChildTool, map[string]string{"child": "library/first", "task": "duplicate"})
			return run.TurnResult{Completed: true, Final: "Go with invalid duplicate"}, nil
		}
		if !strings.Contains(prompt, "Host decision feedback:") {
			return run.TurnResult{}, errors.New("invalid delegation lacked concrete normal follow-up")
		}
		a.feedbackSeen.Store(true)
	}
	if path == "library" && strings.HasPrefix(prompt, "Your responsibility") && a.mode == "pre-go consultation" {
		if !strings.Contains(prompt, "before calling delegate_child") || !strings.Contains(prompt, "finish this response with Go") || !strings.Contains(prompt, "Only then does the Host commit your stand") {
			return run.TurnResult{}, errors.New("Go boundary was not explained to the parent")
		}
		for _, child := range []string{"library/first", "library/second"} {
			answer := call(run.DelegateChildTool, map[string]string{"child": child, "task": child})
			if answer.Refused || !strings.Contains(answer.Text, "finish this response with Go") {
				return run.TurnResult{}, errors.New("delegation did not explain the Go boundary")
			}
		}
		answer := call(run.AskChildTool, map[string]string{"child": "library/first", "question": "Can you start?"})
		if !answer.Refused || !strings.Contains(answer.Text, "starts only after you finish this response with Go") {
			return run.TurnResult{}, errors.New("pre-Go consultation did not return actionable feedback")
		}
		a.feedbackSeen.Store(true)
		return run.TurnResult{Completed: true, Final: "Go after all child tasks"}, nil
	}
	if path == "library" && strings.HasPrefix(prompt, "Review your direct child library/first") && a.mode != "delegation" {
		n := a.reviewCalls.Add(1)
		if n == 1 || a.mode == "exhaust" {
			switch a.mode {
			case "commit", "same response":
				answer := call(run.AcceptChildTool, map[string]string{"child": "library/first", "commit": run04OwnCommit})
				if !answer.Refused || !strings.Contains(answer.Text, "omit commit") {
					return run.TurnResult{}, errors.New("wrong Run04 commit was not returned as tool feedback")
				}
				a.commitRefused.Store(true)
				if a.mode == "same response" {
					a.feedbackSeen.Store(true)
					answer = call(run.AcceptChildTool, map[string]string{"child": "library/first"})
					if answer.Refused {
						return run.TurnResult{}, errors.New(answer.Text)
					}
					return run.TurnResult{Completed: true, Final: "Corrected tool call within the same response"}, nil
				}
			case "other child":
				call(run.AcceptChildTool, map[string]string{"child": "library/second"})
			case "diagnostic bounds":
				for range 100 {
					answer := call(run.AcceptChildTool, map[string]string{"child": "library/first", strings.Repeat("unknown", 10000): "value"})
					if !answer.Refused || len(answer.Text) > 1024 || !strings.Contains(answer.Text, "[diagnostic truncated]") {
						return run.TurnResult{}, errors.New("tool diagnostic exceeded its byte bound or hid truncation")
					}
				}
			case "multiple":
				call(run.AcceptChildTool, map[string]string{"child": "library/first"})
				call(run.ReturnChildTool, map[string]string{"child": "library/first", "correction": "conflicting decision"})
			}
			return run.TurnResult{Completed: true, Final: "Native response with invalid decision"}, nil
		}
		if !strings.Contains(prompt, "Host decision feedback:") {
			return run.TurnResult{}, errors.New("invalid review decision lacked concrete normal follow-up")
		}
		if a.mode == "diagnostic bounds" {
			_, diagnostic, _ := strings.Cut(prompt, "\nTool feedback: ")
			if len(diagnostic) > 9*1024 || strings.Count(diagnostic, "[diagnostic truncated]") != 7 || !strings.Contains(diagnostic, "[additional tool refusals omitted]") {
				return run.TurnResult{}, errors.New("stored tool refusal diagnostics exceeded count/byte bound or omitted no marker")
			}
		}
		a.feedbackSeen.Store(true)
	}
	return s.concurrentSession.Message(ctx, prompt, timeout)
}

type bindingIntegration struct {
	*concurrentIntegration
	mu           sync.Mutex
	firstResults []string
}

func (g *bindingIntegration) Integrate(ctx context.Context, parent, child run.AssignmentWorkspace, h run.AssignmentHandoff, previous string) (string, error) {
	if child.ID() == "fan-2" {
		g.mu.Lock()
		g.firstResults = append(g.firstResults, h.ResultCommit)
		g.mu.Unlock()
	}
	return g.concurrentIntegration.Integrate(ctx, parent, child, h, previous)
}

func TestAssignmentDecisionFeedbackBindsInspectedHandoffAndRecoversWithinBudget(t *testing.T) {
	for _, mode := range []string{"commit", "same response", "other child", "missing", "multiple", "delegation", "pre-go consultation", "exhaust", "allowance", "diagnostic bounds"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, base := concurrentFixture()
			if mode == "allowance" {
				base.workGates["library/second"] = make(chan struct{})
			}
			agents := &feedbackAgents{concurrentAgents: base, mode: mode}
			git := &bindingIntegration{concurrentIntegration: &concurrentIntegration{agents: base, entered: make(chan string, 8), gates: map[string]<-chan struct{}{}}}
			maxMessages := 40
			if mode == "allowance" {
				maxMessages = 5
			}
			result, flow, err := run.RunAssignmentTree(ctx, base.team, &fixedHandoffCopies{}, agents, git, run.AssignmentSchedule{IDPrefix: "fan", Task: "Implement", MaxMessages: maxMessages, Timeout: time.Second, TranscriptDir: t.TempDir()})
			if mode == "allowance" {
				if err == nil || !strings.Contains(err.Error(), "allowance exhausted") || agents.nativeCalls.Load() != 5 || len(git.firstResults) != 0 {
					t.Fatalf("decision feedback reset invocation allowance: %v %d", err, agents.nativeCalls.Load())
				}
				return
			}
			if mode == "exhaust" {
				if err == nil || !strings.Contains(err.Error(), "after two feedback messages") || agents.reviewCalls.Load() != 3 || result.ResultCommit != "" {
					t.Fatalf("invalid decisions retried without finite bound: %v %d", err, agents.reviewCalls.Load())
				}
				if len(git.firstResults) != 0 {
					t.Fatal("applied an invalid decision")
				}
				return
			}
			if err != nil || result.ResultCommit == "" || !agents.feedbackSeen.Load() {
				t.Fatalf("decision did not recover via ordinary feedback: %v", err)
			}
			if len(git.firstResults) != 1 || git.firstResults[0] != run04Handoff {
				t.Fatalf("Host accepted model-selected or unreviewed commit: %v", git.firstResults)
			}
			if len(git.order) != 3 || agents.leafPrompts.Load() != 2 {
				t.Fatalf("invalid action caused extra integration or omitted leaf work: %v", git.order)
			}
			if mode == "commit" && !agents.commitRefused.Load() {
				t.Fatal("Run04 own commit was not refused while fixed handoff remained available")
			}
			if mode == "same response" && agents.reviewCalls.Load() != 1 {
				t.Fatal("valid same-response correction caused an unnecessary Host follow-up")
			}
			for _, r := range flow.Results() {
				if r.Manager == "library/first" && r.Handoff.ResultCommit != run04Handoff {
					t.Fatal("fixed handoff changed")
				}
			}
		})
	}
}
