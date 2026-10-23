package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Exercise actual scheduler messages through the native codec, rather than
// estimating wire size from raw text. The Run package tests delivery separately.
func TestAssignmentReviewSchedulerMessagesFitCodexFrames(t *testing.T) {
	for _, tc := range []struct {
		name, diff, summary string
	}{
		{"utf8-boundary", strings.Repeat("a", (128<<10)-1) + "✓" + strings.Repeat("b", 300<<10) + "END", "child result"},
		{"json-escaping", strings.Repeat("&", 300<<10) + "END", strings.Repeat("&", 140<<10) + "\nOPEN MATTER: admission-limit regression remains; this handoff is not conforming."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			agents := &reviewTransportAgents{codec: testCodec(t, 1<<20), summary: tc.summary}
			git := &reviewTransportGit{diff: tc.diff}
			result, _, err := run.RunAssignmentTree(ctx, []run.Manager{
				{Path: ".", Children: []string{"child"}, ExecuteProfile: "exec"},
				{Path: "child", Parent: ".", ExecuteProfile: "exec"},
			}, reviewTransportWorkspaces{}, agents, git, run.AssignmentSchedule{
				IDPrefix: "review", Task: "Implement", MaxMessages: 20, Timeout: time.Second, TranscriptDir: t.TempDir(),
			})
			if err != nil || result.ResultCommit == "" || git.integrated != 1 {
				t.Fatalf("scheduler/codec review failed: result=%+v integrated=%d err=%v", result, git.integrated, err)
			}
			var diff, summary strings.Builder
			parts, summaries, decisions := 0, 0, 0
			for _, prompt := range agents.root.prompts {
				switch {
				case strings.HasPrefix(prompt, "Fixed child summary from"):
					summaries++
					summary.WriteString(reviewTransportPart(t, prompt, ":\n", "\nRead this part."))
				case strings.HasPrefix(prompt, "Fixed Git difference from direct child"):
					parts++
					diff.WriteString(reviewTransportPart(t, prompt, ":\n", "\nRead this part."))
				case strings.HasPrefix(prompt, "Review your direct child"):
					decisions++
					_, final, found := strings.Cut(prompt, "\nActual Git changes, final part")
					if !found {
						t.Fatal("review decision lacks final Git difference")
					}
					diff.WriteString(reviewTransportPart(t, final, ":\n", "\nInspect conformance"))
				}
			}
			if parts < 2 || decisions != 1 || diff.String() != tc.diff {
				t.Fatalf("wire delivery changed Git difference: parts=%d decisions=%d bytes=%d/%d", parts, decisions, diff.Len(), len(tc.diff))
			}
			if tc.name == "json-escaping" && (summaries < 2 || summary.String() != tc.summary) {
				t.Fatalf("wire delivery lost child summary/open matter: parts=%d bytes=%d/%d", summaries, summary.Len(), len(tc.summary))
			}
		})
	}
}

func reviewTransportPart(t *testing.T, text, start, end string) string {
	t.Helper()
	_, rest, found := strings.Cut(text, start)
	last := strings.LastIndex(rest, end)
	if !found || last < 0 {
		t.Fatal("malformed scheduler review part")
	}
	return rest[:last]
}

type reviewTransportAgents struct {
	codec   *Codec
	summary string
	root    *reviewTransportSession
}

func (a *reviewTransportAgents) OpenAssignment(_ context.Context, spec run.AssignmentSpec, handler run.ToolHandler) (run.AssignmentSession, error) {
	s := &reviewTransportSession{agents: a, root: spec.ID == "review-0", handler: handler}
	if s.root {
		a.root = s
	}
	return s, nil
}

type reviewTransportSession struct {
	agents  *reviewTransportAgents
	root    bool
	handler run.ToolHandler
	prompts []string
}

func (s *reviewTransportSession) Message(ctx context.Context, prompt string, _ time.Duration) (run.TurnResult, error) {
	if !s.root {
		exit := 0
		commands := make([]run.CommandObservation, 5)
		for i := range commands {
			commands[i] = run.CommandObservation{Command: "check", Status: "passed", ExitCode: &exit, Output: strings.Repeat("&", 4096)}
		}
		return run.TurnResult{Completed: true, Final: s.agents.summary, Commands: commands}, nil
	}
	wire, err := s.agents.codec.EncodeTurnStart(IntegerRequestID(1), TurnStartParams{Thread: "review-thread", Text: prompt, Effort: "high"})
	if err != nil {
		return run.TurnResult{}, err
	}
	var request struct {
		Params struct{ Input []struct{ Text string } }
	}
	if err := json.Unmarshal(wire, &request); err != nil {
		return run.TurnResult{}, err
	}
	if len(request.Params.Input) != 1 || request.Params.Input[0].Text != prompt {
		return run.TurnResult{}, errors.New("Codex frame changed scheduler prompt")
	}
	s.prompts = append(s.prompts, request.Params.Input[0].Text)
	var tool string
	var args json.RawMessage
	switch {
	case strings.HasPrefix(prompt, "Your responsibility"):
		tool, args = run.DelegateChildTool, json.RawMessage(`{"child":"child","task":"Implement child"}`)
	case strings.HasPrefix(prompt, "Review your direct child"):
		tool, args = run.AcceptChildTool, json.RawMessage(`{"child":"child"}`)
	}
	if tool != "" {
		if answer := s.handler(ctx, tool, args); answer.Refused {
			return run.TurnResult{}, errors.New(answer.Text)
		}
	}
	return run.TurnResult{Completed: true, Final: "reviewed"}, nil
}

func (*reviewTransportSession) Close(context.Context) (run.TurnResult, error) {
	return run.TurnResult{Stopped: true}, nil
}

// These controlled Git ports only supply identities for the transport test;
// real Git behavior is covered by the Git adapter's own tests.
type reviewTransportWorkspaces struct{}

func (reviewTransportWorkspaces) PrepareAssignment(_ context.Context, spec run.WorkspaceSpec) (run.AssignmentWorkspace, error) {
	return reviewTransportWorkspace{spec.ID}, nil
}
func (reviewTransportWorkspaces) ForkAssignmentCommit(_ context.Context, _ run.AssignmentWorkspace, _ string, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	return reviewTransportWorkspace{id}, nil
}
func (reviewTransportWorkspaces) UpdateAssignment(context.Context, run.AssignmentWorkspace, run.AssignmentWorkspace, string, string) error {
	return nil
}

type reviewTransportWorkspace struct{ id run.WorkspaceID }

func (w reviewTransportWorkspace) ID() run.WorkspaceID  { return w.id }
func (w reviewTransportWorkspace) Volume() string       { return string(w.id) }
func (reviewTransportWorkspace) StartingCommit() string { return strings.Repeat("a", 40) }
func (reviewTransportWorkspace) Commit(context.Context, string) (string, error) {
	return strings.Repeat("b", 40), nil
}
func (reviewTransportWorkspace) ContainsCommit(_ context.Context, commit string) (bool, error) {
	return commit == strings.Repeat("a", 40) || commit == strings.Repeat("b", 40), nil
}
func (reviewTransportWorkspace) Close(context.Context) error { return nil }

type reviewTransportGit struct {
	diff       string
	integrated int
}

func (g *reviewTransportGit) Changes(context.Context, run.AssignmentWorkspace, run.AssignmentHandoff) (string, error) {
	return g.diff, nil
}
func (g *reviewTransportGit) Integrate(_ context.Context, _, _ run.AssignmentWorkspace, handoff run.AssignmentHandoff, _ string) (string, error) {
	g.integrated++
	return handoff.ResultCommit, nil
}
