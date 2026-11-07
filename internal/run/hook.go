package run

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SettingsFile is the project's Claude Code settings, and HookCommand the hook in
// them that runs Guard. The agents are subagents of the owner's own session, so
// the hook has to be in the project's settings.
const (
	SettingsFile = ".claude/settings.json"
	HookCommand  = "markitect hook pretool"
)

// GuardedCommands are the commands the hook guards, as the permission rules
// that start it. A rule holds one command, so each gets its own handler.
var GuardedCommands = []string{"Bash(markitect answer*)", "Bash(markitect checklist*)"}

// HasGuard says whether settings run HookCommand before each guarded shell
// command, as a command hook before Bash; a mention elsewhere does not count.
// A handler without a rule runs before every command, and so covers all.
func HasGuard(settings []byte) bool {
	var s struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Type    string `json:"type"`
					If      string `json:"if"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if json.Unmarshal(settings, &s) != nil {
		return false
	}
	covered := map[string]bool{}
	for _, m := range s.Hooks.PreToolUse {
		if m.Matcher != "" && m.Matcher != "*" && m.Matcher != "Bash" {
			continue
		}
		for _, h := range m.Hooks {
			if h.Type != "command" || strings.TrimSpace(h.Command) != HookCommand {
				continue
			}
			for _, rule := range GuardedCommands {
				if h.If == "" || h.If == rule {
					covered[rule] = true
				}
			}
		}
	}
	return len(covered) == len(GuardedCommands)
}

// namingFlags are the flags with which checklist begins work or names a change.
var namingFlags = []string{"--start", "--full", "--base", "--findings"}

// Guard is the project's hook before a shell command. Only the reviewer stores
// answers, so an agent cannot mark its own work as done; and only the owner's
// session begins the work or names a change, so an agent that changed the spec
// cannot start over with its change as the owner's. Its input is what the
// provider tells a hook, which names the agent a subagent's call comes from. It
// needs no Run, since answer refuses anyway where none is open. It returns what
// the hook prints, empty to let the command run.
func Guard(input []byte) []byte {
	var in struct {
		AgentType string         `json:"agent_type"`
		AgentID   string         `json:"agent_id"`
		ToolInput map[string]any `json:"tool_input"`
	}
	if json.Unmarshal(input, &in) != nil {
		return nil
	}
	command, _ := in.ToolInput["command"].(string)
	switch {
	case strings.Contains(command, "markitect answer") && in.AgentType != ReviewerAgent:
		return deny(fmt.Sprintf("Only the %s agent answers the checklist. Start it, and it answers.", ReviewerAgent))
	case in.AgentID != "" && strings.Contains(command, "markitect checklist") && namesWork(command):
		return deny("An agent takes the work as it is: markitect checklist, or markitect checklist --review. Only the owner's session begins the work or names a change.")
	}
	return nil
}

func namesWork(command string) bool {
	for _, flag := range namingFlags {
		if strings.Contains(command, flag) {
			return true
		}
	}
	return false
}

// deny is the hook's answer that refuses the command, and why.
func deny(why string) []byte {
	out, err := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": why,
	}})
	if err != nil {
		return nil
	}
	return out
}
