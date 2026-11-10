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

// HasGuard says whether settings run HookCommand before a shell command that
// records a verdict, before every tool that writes a file and before every tool
// that starts an agent; a mention elsewhere does not count. A handler without a
// rule runs before every call of its tools, and so covers them.
func HasGuard(settings []byte) bool {
	pre, ok := readPreToolUse(settings)
	if !ok {
		return false
	}
	mark, covered := false, map[string]bool{}
	for _, m := range pre {
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
			for _, t := range guardedTools() {
				if all || tools[t] {
					covered[t] = true
				}
			}
		}
	}
	return mark && len(covered) == len(guardedTools())
}

// guardedTools are the tools the hook runs before on every call.
func guardedTools() []string {
	return append(strings.Split(EditTools, "|"), strings.Split(AgentTools, "|")...)
}

// GuardsEveryTool says whether a plugin's hooks run the guard before the shell
// and before every tool that writes a file or starts an agent. An older plugin,
// whose hooks ran it only before some of them, does not count.
func GuardsEveryTool(data []byte) bool {
	pre, ok := readPreToolUse(data)
	if !ok {
		return false
	}
	covered := map[string]bool{}
	for _, group := range pre {
		guards := false
		for _, hook := range group.Hooks {
			guards = guards || strings.Contains(hook.Command, "hook pretool")
		}
		if !guards {
			continue
		}
		for _, tool := range strings.Split(group.Matcher, "|") {
			covered[strings.TrimSpace(tool)] = true
		}
	}
	for _, tool := range strings.Split("Bash|"+EditTools+"|"+AgentTools, "|") {
		if !covered[tool] {
			return false
		}
	}
	return true
}

// guardHooks hold the rights of the four roles (DEC-100): only the reviewer
// records verdicts, so that the agent that built does not verify its own work; a
// subagent writes only what its role may; and the builder starts only the
// reviewer. The first runs only before a shell command that records, the second
// before every tool that writes a file, the third before every tool that starts
// an agent. Each runs in bash, as the plugin's do, since its command is a shell
// command, also on Windows.
func guardHooks() []any {
	hook := func(rule string) []any {
		h := map[string]any{"type": "command", "shell": "bash", "command": HookCommand}
		if rule != "" {
			h["if"] = rule
		}
		return []any{h}
	}
	return []any{
		map[string]any{"matcher": "Bash", "hooks": hook(MarkRule)},
		map[string]any{"matcher": EditTools, "hooks": hook("")},
		map[string]any{"matcher": AgentTools, "hooks": hook("")},
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

// ownGuard says whether a PreToolUse entry is one an earlier markitect wrote: a
// group whose every hook runs HookCommand. The current one replaces it.
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
		if !ok || hook["command"] != HookCommand {
			return false
		}
	}
	return true
}
