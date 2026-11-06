package run

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AnswerGuard is the project's hook before a shell command: only the reviewer
// stores answers, so an agent cannot mark its own work as done. Its input is what
// the provider tells a hook, which names the agent a subagent's call comes from.
// It needs no Run, since answer refuses anyway where none is open. It returns
// what the hook prints, empty to let the command run.
func AnswerGuard(input []byte) []byte {
	var in struct {
		AgentType string         `json:"agent_type"`
		ToolInput map[string]any `json:"tool_input"`
	}
	if json.Unmarshal(input, &in) != nil || in.AgentType == ReviewerAgent {
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
