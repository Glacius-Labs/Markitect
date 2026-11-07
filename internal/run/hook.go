package run

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// ReviewerAgent is the agent type of the reviewer, the one agent that records
// verdicts; BuilderAgent the builder's. The harness defines both.
const (
	ReviewerAgent = "markitect-reviewer"
	BuilderAgent  = "markitect-builder"
)

// SettingsFile is the project's Claude Code settings, and HookCommand the hook in
// them that runs Guard. The agents are subagents of the owner's own session, so
// the hook has to be in the project's settings.
const (
	SettingsFile = ".claude/settings.json"
	HookCommand  = "markitect hook pretool"
)

// MarkRule is the permission rule that starts the hook before a shell command
// that records a verdict.
const MarkRule = "Bash(markitect mark*)"

// EditTools are the tools that write files, as the matcher of the hook that
// keeps subagents out of Markitect's folder.
const EditTools = "Edit|Write|MultiEdit|NotebookEdit"

// HasGuard says whether settings run HookCommand before a shell command that
// records a verdict, and before every tool that writes a file; a mention elsewhere
// does not count. A handler without a rule runs before every call of its tools,
// and so covers them.
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
	mark, edits := false, map[string]bool{}
	for _, m := range s.Hooks.PreToolUse {
		tools := map[string]bool{}
		all := m.Matcher == "" || m.Matcher == "*"
		for _, t := range strings.Split(m.Matcher, "|") {
			tools[strings.TrimSpace(t)] = true
		}
		for _, h := range m.Hooks {
			if h.Type != "command" || strings.TrimSpace(h.Command) != HookCommand {
				continue
			}
			if (all || tools["Bash"]) && (h.If == "" || h.If == MarkRule) {
				mark = true
			}
			if h.If != "" {
				continue
			}
			for _, t := range strings.Split(EditTools, "|") {
				if all || tools[t] {
					edits[t] = true
				}
			}
		}
	}
	return mark && len(edits) == len(strings.Split(EditTools, "|"))
}

// markCommand finds a call of markitect mark in a shell command.
var markCommand = regexp.MustCompile(`(^|[\s;&|("'` + "`" + `])markitect(\.exe)?\s+mark(\s|$)`)

// Guard is the project's hook before a tool call. It guards the record and the
// spec, not the process:
//
//   - only the reviewer runs markitect mark, so the agent that built does not
//     verify its own work;
//   - no subagent writes or edits under Markitect's folder except the registry
//     pages, which the builder keeps, so a builder cannot weaken a Definition
//     during the work or write lines into the record. The owner's own session is
//     no subagent and stays free.
//
// Its input is what the provider tells a hook, which names the agent a subagent's
// call comes from. It returns what the hook prints, empty to let the call run. It
// reads the call as text, so it guards against mistakes, not against an agent
// that hides a command.
func Guard(input []byte) []byte {
	var in struct {
		AgentType string         `json:"agent_type"`
		AgentID   string         `json:"agent_id"`
		ToolName  string         `json:"tool_name"`
		ToolInput map[string]any `json:"tool_input"`
	}
	if json.Unmarshal(input, &in) != nil {
		return nil
	}
	if command, ok := in.ToolInput["command"].(string); ok && markCommand.MatchString(command) && in.AgentType != ReviewerAgent {
		return deny(fmt.Sprintf("Only the %s agent records verdicts, so the agent that built does not verify its own work. Start it, and it verifies and records.", ReviewerAgent))
	}
	if in.AgentID == "" {
		return nil
	}
	for _, key := range []string{"file_path", "notebook_path"} {
		file, _ := in.ToolInput[key].(string)
		if file != "" && guardedFile(file) {
			return deny(fmt.Sprintf("%s is the owner's: an agent changes nothing under %s except the registry pages. If the spec is wrong, say so and leave it as it is.", file, project.Dir))
		}
	}
	return nil
}

// guardedFile says whether a file lies in Markitect's folder of a project and is
// not a registry page, whatever folder the path starts from.
func guardedFile(file string) bool {
	segments := strings.Split(path.Clean(strings.ReplaceAll(file, `\`, "/")), "/")
	for i, s := range segments {
		if !strings.EqualFold(s, project.Dir) {
			continue
		}
		rel := strings.Join(segments[i:], "/")
		lower := strings.ToLower(rel)
		registry := strings.ToLower(project.RegistryDir) + "/"
		page := lower == strings.ToLower(project.RootPage) || (strings.HasPrefix(lower, registry) && strings.HasSuffix(lower, ".md"))
		return !page
	}
	return false
}

// deny is the hook's answer that refuses the call, and why.
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
