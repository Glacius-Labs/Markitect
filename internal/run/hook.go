package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Hook events, as the harness names them.
const (
	HookStop        = "stop"
	HookStopFailure = "stopfailure"
	HookPreTool     = "pretool"
)

// The provider's names of those events.
const (
	HookStopName        = "Stop"
	HookStopFailureName = "StopFailure"
	HookPreToolName     = "PreToolUse"
)

// hookInput is what the provider tells a hook.
type hookInput struct {
	AgentType       string            `json:"agent_type"`
	ToolName        string            `json:"tool_name"`
	ToolInput       map[string]any    `json:"tool_input"`
	BackgroundTasks []json.RawMessage `json:"background_tasks"`
	Error           string            `json:"error"`
	ErrorDetails    string            `json:"error_details"`
}

// Hook answers one hook of a Run. It returns what the hook prints, empty for
// nothing to say. A problem of the hook's own must not stop an agent, so it
// answers nothing when it cannot read the Run.
func (e *Engine) Hook(ctx context.Context, event, run string, input []byte) []byte {
	st, found, err := e.Store.Load(run)
	if err != nil || !found || !st.Open {
		return nil
	}
	var in hookInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil
	}
	switch event {
	case HookStop:
		return e.hookStop(ctx, st, in)
	case HookStopFailure:
		_ = e.Store.Update(run, func(s *RunState) error {
			s.Failure = strings.TrimSpace(in.Error + " " + in.ErrorDetails)
			return nil
		})
	case HookPreTool:
		return e.hookPreTool(ctx, st, in)
	}
	return nil
}

// hookStop sends a session back while its Run is open, unless it is waiting for
// a reviewer or for a submit. The provider's own count of blocks starts again at
// any tool call, so the tool keeps its own.
func (e *Engine) hookStop(ctx context.Context, st *RunState, in hookInput) []byte {
	if len(in.BackgroundTasks) > 0 || e.Store.SubmitRunning(st.Run) {
		return nil
	}
	if st.Blocks >= maxBlocks {
		_ = e.close(ctx, st, StateIncomplete, fmt.Sprintf("the session kept trying to end without markitect submit, %d times since its last call", maxBlocks), nil)
		return nil
	}
	_ = e.Store.Update(st.Run, func(s *RunState) error {
		s.Blocks++
		return nil
	})
	reason := "This run is still open, and nobody can answer you. Call markitect submit and do what it says, or run markitect stop \"why\" if the spec cannot be met."
	out, err := json.Marshal(map[string]string{"decision": "block", "reason": reason})
	if err != nil {
		return nil
	}
	return out
}

// hookPreTool holds the deadline and lets only a reviewer call answer.
func (e *Engine) hookPreTool(ctx context.Context, st *RunState, in hookInput) []byte {
	if e.now().After(st.Deadline) && !e.Store.SubmitRunning(st.Run) {
		why := "the run passed its deadline"
		_ = e.close(ctx, st, StateIncomplete, why, nil)
		out, err := json.Marshal(map[string]any{"continue": false, "stopReason": why})
		if err != nil {
			return nil
		}
		return out
	}
	if in.AgentType == ReviewerAgent {
		return nil
	}
	command, _ := in.ToolInput["command"].(string)
	if !strings.Contains(command, "markitect answer") {
		return nil
	}
	out, err := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": fmt.Sprintf("Only the %s agent answers the checklist. Start it, and it answers.", ReviewerAgent),
	}})
	if err != nil {
		return nil
	}
	return out
}
