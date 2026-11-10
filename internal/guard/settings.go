package guard

import (
	"encoding/json"
	"strings"
)

// preToolUse is the PreToolUse hooks of a settings file or of a plugin's hooks
// file: groups of hooks, each run before the tools its matcher names.
type preToolUse []struct {
	Matcher string `json:"matcher"`
	Hooks   []struct {
		Type    string `json:"type"`
		If      string `json:"if"`
		Command string `json:"command"`
	} `json:"hooks"`
}

// readPreToolUse reads the PreToolUse hooks of a settings or hooks file, or says
// that it is not JSON of that shape.
func readPreToolUse(data []byte) (preToolUse, bool) {
	var s struct {
		Hooks struct {
			PreToolUse preToolUse `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if json.Unmarshal(data, &s) != nil {
		return nil, false
	}
	return s.Hooks.PreToolUse, true
}

// HasGuard says whether settings run HookCommand before a command of every
// shell tool that records a verdict, before every tool that writes a file and
// before every tool that starts an agent; a mention elsewhere does not count. A
// handler without a rule runs before every call of its tools, and so covers
// them; one with another rule than the tool's MarkRule covers nothing.
func HasGuard(settings []byte) bool {
	pre, ok := readPreToolUse(settings)
	if !ok {
		return false
	}
	shells, others := strings.Split(ShellTools, "|"), guardedTools()
	covered := map[string]bool{}
	for _, group := range pre {
		for _, h := range group.Hooks {
			if h.Type != "command" || strings.TrimSpace(h.Command) != HookCommand {
				continue
			}
			for _, t := range shells {
				if matches(group.Matcher, t) && (h.If == "" || h.If == MarkRule(t)) {
					covered[t] = true
				}
			}
			for _, t := range others {
				if matches(group.Matcher, t) && h.If == "" {
					covered[t] = true
				}
			}
		}
	}
	return len(covered) == len(shells)+len(others)
}

// matches says whether a group's matcher names a tool: by its name in a list, or
// as every tool, with * or nothing.
func matches(matcher, tool string) bool {
	if matcher == "" || matcher == "*" {
		return true
	}
	for _, t := range strings.Split(matcher, "|") {
		if strings.TrimSpace(t) == tool {
			return true
		}
	}
	return false
}

// guardedTools are the tools the hook runs before on every call.
func guardedTools() []string {
	return append(strings.Split(EditTools, "|"), strings.Split(AgentTools, "|")...)
}

// guardHooks hold the rights of the four roles (DEC-100): only the reviewer
// records verdicts, so that the agent that built does not verify its own work; a
// subagent writes only what its role may; and the builder starts only the
// reviewer. The first group runs only before a shell command that records, with
// one handler for each shell tool, since a rule names one tool; the second runs
// before every tool that writes a file, the third before every tool that starts
// an agent. Each runs in bash, as the plugin's do, since its command is a shell
// command, also on Windows.
func guardHooks() []any {
	hook := func(rule string) map[string]any {
		h := map[string]any{"type": "command", "shell": "bash", "command": HookCommand}
		if rule != "" {
			h["if"] = rule
		}
		return h
	}
	var marks []any
	for _, tool := range strings.Split(ShellTools, "|") {
		marks = append(marks, hook(MarkRule(tool)))
	}
	return []any{
		map[string]any{"matcher": ShellTools, "hooks": marks},
		map[string]any{"matcher": EditTools, "hooks": []any{hook("")}},
		map[string]any{"matcher": AgentTools, "hooks": []any{hook("")}},
	}
}

// GuardedHooks returns the settings' hooks with the guard in place of one an
// earlier markitect wrote, or why it cannot add it without losing what is there.
func GuardedHooks(settings map[string]any) (map[string]any, string) {
	hooks := map[string]any{}
	if v, ok := settings["hooks"]; ok {
		if hooks, ok = v.(map[string]any); !ok {
			return nil, "they are not an object"
		}
	}
	var pre []any
	if v, ok := hooks["PreToolUse"]; ok {
		if pre, ok = v.([]any); !ok {
			return nil, "its PreToolUse hooks are not a list"
		}
	}
	kept := []any{}
	for _, entry := range pre {
		if !ownGuard(entry) {
			kept = append(kept, entry)
		}
	}
	hooks["PreToolUse"] = append(kept, guardHooks()...)
	return hooks, ""
}

// ownGuard says whether a PreToolUse entry is one a markitect wrote: a group
// whose every hook ends in the call of the hook, whatever comes before it. The
// current one replaces it.
func ownGuard(entry any) bool {
	group, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	hooks, ok := group["hooks"].([]any)
	if !ok || len(hooks) == 0 {
		return false
	}
	for _, h := range hooks {
		hook, ok := h.(map[string]any)
		if !ok {
			return false
		}
		if command, _ := hook["command"].(string); !strings.HasSuffix(strings.TrimSpace(command), hookCall) {
			return false
		}
	}
	return true
}
