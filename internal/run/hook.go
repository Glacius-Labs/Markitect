package run

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SettingsFile is the project's Claude Code settings, and HookCommand the hook in
// them that runs AnswerGuard. A Run's agents are subagents of the owner's own
// session, so the hook has to be in the project's settings.
const (
	SettingsFile = ".claude/settings.json"
	HookCommand  = "markitect hook pretool"
)

// HasAnswerHook says whether settings run HookCommand before every shell command,
// as a command hook before Bash; a mention elsewhere does not count.
func HasAnswerHook(settings []byte) bool {
	var s struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Type    string `json:"type"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if json.Unmarshal(settings, &s) != nil {
		return false
	}
	for _, m := range s.Hooks.PreToolUse {
		if m.Matcher != "" && m.Matcher != "*" && m.Matcher != "Bash" {
			continue
		}
		for _, h := range m.Hooks {
			if h.Type == "command" && strings.TrimSpace(h.Command) == HookCommand {
				return true
			}
		}
	}
	return false
}

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
