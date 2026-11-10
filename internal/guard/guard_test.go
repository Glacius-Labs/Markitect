package guard_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/guard"
)

func has(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Fatalf("%q not in:\n%s", w, text)
		}
	}
}

func hookInput(t *testing.T, v map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Only the reviewer records a verdict: the agent that built does not verify its
// own work, and neither does the owner's session through an agent.
func TestGuardLetsOnlyTheReviewerRecord(t *testing.T) {
	call := func(agentID, agent, command string) string {
		return string(guard.Guard(hookInput(t, map[string]any{
			"agent_id": agentID, "agent_type": agent, "tool_name": "Bash", "tool_input": map[string]any{"command": command},
		})))
	}
	for _, command := range []string{
		"markitect mark fees/fee@0a1b2c ok -e src/fees.go",
		"cd . && markitect mark fees/fee@0a1b2c fail -m wrong",
		"markitect  mark\tfees/fee@0a1b2c ok",
		`bash -c "markitect mark fees/fee@0a1b2c ok"`,
		"markitect.exe mark fees/fee@0a1b2c ok",
		"MARKITECT Mark fees/fee@0a1b2c ok",
		"/usr/local/bin/markitect mark fees/fee@0a1b2c ok",
		"./markitect mark fees/fee@0a1b2c ok",
		`& "C:\Tools\markitect.exe" mark fees/fee@0a1b2c ok`,
		"markitect.cmd mark fees/fee@0a1b2c ok",
	} {
		has(t, call("a1", guard.BuilderAgent, command), `"permissionDecision":"deny"`, guard.ReviewerAgent)
		has(t, call("", "", command), `"permissionDecision":"deny"`)
		for _, reviewer := range []string{guard.ReviewerAgent, "Markitect-Reviewer"} {
			if out := call("a2", reviewer, command); out != "" {
				t.Fatalf("refused %s %q: %s", reviewer, command, out)
			}
		}
	}
	for _, command := range []string{"markitect status", "markitect status --all", "go test ./...", "cat .claude/agents/markitect-reviewer.md", "echo markitect marked", "markitect markdown x", "cat docs/markitect-mark.md", "markitect.cmd status"} {
		if out := call("a1", guard.BuilderAgent, command); out != "" {
			t.Fatalf("refused %q: %s", command, out)
		}
	}
	if out := string(guard.Guard([]byte("not json"))); out != "" {
		t.Fatalf("%s", out)
	}
}

// A subagent writes only what its role may (DEC-100): the builder, and any other
// subagent, code and the registry pages; the architect the spec and the registry
// pages; the reviewer and the auditor nothing; nobody the record, which mark
// writes, its Git attributes, which init writes, or a file that steers the
// agents. The owner's own session is free.
func TestGuardHoldsEachRolesRightToWrite(t *testing.T) {
	call := func(agentID, agent, tool, key, file string) string {
		return string(guard.Guard(hookInput(t, map[string]any{
			"agent_id": agentID, "agent_type": agent, "tool_name": tool, "tool_input": map[string]any{key: file},
		})))
	}
	files := map[string][]string{
		"spec": {
			"/p/.markitect/spec/fees/fee.md",
			"/p/.markitect/manifest.yaml",
			"/p/.markitect/ignore",
			`C:\p\.markitect\spec\fees\fee.md`,
			".markitect/registry/../spec/fee.md",
			"/p/.Markitect/Spec/fee.md",
			// Any other file of Markitect's folder is the spec's to settle.
			"/p/.markitect/other/fee.md",
			"/p/.markitect/registry/notes.txt",
			"/p/.markitect/spec/.gitattributes",
		},
		"registry": {
			"/p/.markitect/registry.md",
			"/p/.markitect/registry/fees.md",
			`C:\p\.markitect\registry\members\loans.md`,
		},
		"record": {"/p/.markitect/review.sum", `C:\p\.MARKITECT\review.sum`},
		"setup":  {"/p/.markitect/.gitattributes", `C:\p\.Markitect\.gitattributes`},
		"harness": {
			"/p/.claude/settings.json",
			`C:\p\.claude\settings.local.json`,
			"/p/.claude/agents/markitect-reviewer.md",
			"/p/.claude/skills/markitect-build/SKILL.md",
			"/p/.agents/skills/markitect-architect/reference.md",
			"/p/.claude/workflows/markitect-parallel.js",
			"/p/.markitect/harness.yaml",
			`C:\p\.Markitect\Harness.yaml`,
			"/p/.claude/worktrees/agent-a1/.claude/settings.json",
			"/p/AGENTS.md",
			"/p/CLAUDE.md",
			"/p/CLAUDE.local.md",
			"/p/docs/AGENTS.md",
			"/p/.claude/hooks/check.sh",
			`C:\p\.claude\rules\go\style.md`,
			"/p/.claude/worktrees/agent-a1/.claude/hooks/check.sh",
		},
		"code": {
			"/p/src/fees.go",
			"/p/docs/markitect.md",
			"/p/.markitect-notes/x.md",
			"/p/.claude/worktrees/agent-a1/src/fees.go",
			"/p/.claude/agents/my-own.md",
			"/p/.claude/workflows/my-own.js",
			"/p/src/harness.yaml",
			"/p/.claude/commands/settings.json",
			"/p/docs/agents-guide.md",
			"/p/docs/agents.md",
			"/p/docs/claude.local.md",
			"/p/src/hooks/check.sh",
			"/p/rules/go.md",
			"/p/.gitattributes",
		},
	}
	may := map[string]map[string]bool{
		guard.BuilderAgent:   {"code": true, "registry": true},
		"Markitect-Builder":  {"code": true, "registry": true},
		"general-purpose":    {"code": true, "registry": true},
		"":                   {"code": true, "registry": true},
		guard.ArchitectAgent: {"spec": true, "registry": true},
		guard.ReviewerAgent:  {},
		guard.AuditorAgent:   {},
	}
	for agent, rights := range may {
		for kind, list := range files {
			for _, file := range list {
				for _, tool := range []string{"Edit", "Write", "MultiEdit"} {
					out := call("a1", agent, tool, "file_path", file)
					if rights[kind] && out != "" {
						t.Fatalf("refused %q its %s %s: %s", agent, kind, file, out)
					}
					if !rights[kind] && !strings.Contains(out, `"permissionDecision":"deny"`) {
						t.Fatalf("let %q write the %s %s", agent, kind, file)
					}
				}
				if out := call("", agent, "Write", "file_path", file); out != "" {
					t.Fatalf("refused the owner's session %s: %s", file, out)
				}
			}
		}
	}
	has(t, call("a1", guard.BuilderAgent, "Edit", "file_path", "/p/.markitect/spec/fee.md"), "the registry pages")
	has(t, call("a1", guard.BuilderAgent, "Edit", "file_path", "/p/.markitect/review.sum"), "markitect mark")
	has(t, call("a1", guard.BuilderAgent, "Edit", "file_path", "/p/.claude/settings.json"), "steers the agents")
	has(t, call("a1", guard.ArchitectAgent, "Edit", "file_path", "/p/.markitect/.gitattributes"), "init and update")
	has(t, call("a1", guard.ArchitectAgent, "Write", "file_path", "/p/.markitect/harness.yaml"), "steers the agents")
	has(t, call("a1", guard.ArchitectAgent, "Edit", "file_path", "/p/src/fees.go"), "the builder writes")
	has(t, call("a1", guard.AuditorAgent, "Edit", "file_path", "/p/src/fees.go"), "changes no file")
	has(t, call("a1", guard.BuilderAgent, "NotebookEdit", "notebook_path", "/p/.markitect/x.ipynb"), `"permissionDecision":"deny"`)
}

// The builder starts only the reviewer, the other roles start no agent, and no
// subagent starts the builder or the architect, which only the owner's session
// starts (DEC-100). Subagent type lists in an agent's tools hold only for an agent
// run as the main thread, so the hook holds this.
func TestGuardHoldsWhoStartsWhom(t *testing.T) {
	call := func(agentID, agent, tool, start string) string {
		input := map[string]any{"prompt": "go"}
		if start != "" {
			input["subagent_type"] = start
		}
		return string(guard.Guard(hookInput(t, map[string]any{
			"agent_id": agentID, "agent_type": agent, "tool_name": tool, "tool_input": input,
		})))
	}
	allowed := []struct{ id, agent, start string }{
		{"a1", guard.BuilderAgent, guard.ReviewerAgent},
		{"a1", "general-purpose", "Explore"},
		{"a1", "general-purpose", ""},
		{"a1", "general-purpose", guard.ReviewerAgent},
		{"", "", guard.BuilderAgent},
		{"", "", guard.ArchitectAgent},
	}
	for _, c := range allowed {
		for _, tool := range []string{"Agent", "Task"} {
			if out := call(c.id, c.agent, tool, c.start); out != "" {
				t.Fatalf("refused %q starting %q: %s", c.agent, c.start, out)
			}
		}
	}
	refused := []struct{ agent, start, why string }{
		{guard.BuilderAgent, "", guard.ReviewerAgent},
		{guard.BuilderAgent, "general-purpose", guard.ReviewerAgent},
		{guard.BuilderAgent, guard.ArchitectAgent, guard.ReviewerAgent},
		{guard.BuilderAgent, guard.BuilderAgent, guard.ReviewerAgent},
		{guard.ReviewerAgent, guard.ReviewerAgent, "starts no agent"},
		{guard.ArchitectAgent, "Explore", "starts no agent"},
		{guard.AuditorAgent, "", "starts no agent"},
		{"general-purpose", guard.ArchitectAgent, "owner's session"},
		{"general-purpose", guard.BuilderAgent, "owner's session"},
		{"general-purpose", "Markitect-Architect", "owner's session"},
		{"Markitect-Builder", "general-purpose", guard.ReviewerAgent},
	}
	for _, c := range refused {
		for _, tool := range []string{"Agent", "Task"} {
			has(t, call("a1", c.agent, tool, c.start), `"permissionDecision":"deny"`, c.why)
		}
	}
	// A workflow starts agents of every role, so only the owner's session runs one
	// (DEC-103).
	for _, agent := range []string{guard.BuilderAgent, guard.ReviewerAgent, guard.ArchitectAgent, guard.AuditorAgent, "general-purpose", ""} {
		has(t, call("a1", agent, "Workflow", ""), `"permissionDecision":"deny"`, "owner's session starts a workflow")
	}
	if out := call("", "", "Workflow", ""); out != "" {
		t.Fatalf("refused the owner's session a workflow: %s", out)
	}
}

func TestHasGuard(t *testing.T) {
	group := func(matcher string, rules ...string) string {
		var handlers []string
		for _, rule := range rules {
			handlers = append(handlers, `{"type": "command", "if": "`+rule+`", "command": "`+guard.HookCommand+`"}`)
		}
		return `{"matcher": "` + matcher + `", "hooks": [` + strings.Join(handlers, ", ") + `]}`
	}
	settings := func(groups ...string) string {
		return `{"hooks": {"PreToolUse": [` + strings.Join(groups, ", ") + `]}}`
	}
	bash, powershell := guard.MarkRule("Bash"), guard.MarkRule("PowerShell")
	shells, edits, agents := group(guard.ShellTools, bash, powershell), group(guard.EditTools, ""), group(guard.AgentTools, "")
	for s, want := range map[string]bool{
		settings(shells, edits, agents): true,
		settings(group("", "")):         true,
		settings(group("*", "")):        true,
		settings(group("Bash", bash), group("PowerShell", powershell), edits, agents):                              true,
		settings(group("Bash|PowerShell", ""), group("Write|Edit|NotebookEdit|MultiEdit|Task|Agent|Workflow", "")): true,
		// Before the Bash tool only, a verdict recorded through PowerShell gets past.
		settings(group("Bash", bash), edits, agents):                 false,
		settings(group(guard.ShellTools, bash), edits, agents):       false,
		settings(group(guard.ShellTools, bash, bash), edits, agents): false,
		settings(group("*", bash, powershell)):                       false,
		settings(shells, edits, group("Agent|Task", "")):             false,
		settings(shells, edits):                                      false,
		settings(shells, edits, group("Agent", "")):                  false,
		settings(shells):        false,
		settings(edits, agents): false,
		settings(shells, group("Edit|Write", ""), agents):                                       false,
		settings(group(guard.ShellTools, "Bash(markitect status*)", powershell), edits, agents): false,
		settings(shells, edits, group(guard.AgentTools, "Agent(markitect-*)")):                  false,
		`{"hooks": {"PostToolUse": [` + group("", "") + `]}}`:                                   false,
		`{"note": "` + guard.HookCommand + `"}`:                                                 false,
		strings.ReplaceAll(settings(shells, edits, agents), guard.HookCommand, "my-own-check"):  false,
		"not json": false,
	} {
		if got := guard.HasGuard([]byte(s)); got != want {
			t.Errorf("%s: %v", s, got)
		}
	}
}

// The guard takes the place of the hooks any earlier markitect wrote, whatever
// stood before the call in them, and keeps the owner's own.
func TestGuardedHooksReplaceAnEarlierGuard(t *testing.T) {
	var settings map[string]any
	if err := json.Unmarshal([]byte(`{"hooks": {"PreToolUse": [
		{"matcher": "Bash", "hooks": [{"type": "command", "command": "markitect hook pretool"}]},
		{"matcher": "Edit|Write", "hooks": [{"type": "command", "command": "command -v markitect >/dev/null 2>&1 || exit 0; MARKITECT_HOOK=project markitect hook pretool"}]},
		{"matcher": "Bash", "hooks": [{"type": "command", "command": "my-own-check"}, {"type": "command", "command": "markitect hook pretool"}]},
		{"matcher": "Write", "hooks": [{"type": "command", "command": "echo markitect hook pretool ran"}]}
	]}}`), &settings); err != nil {
		t.Fatal(err)
	}
	hooks, why := guard.GuardedHooks(settings)
	if why != "" {
		t.Fatal(why)
	}
	settings["hooks"] = hooks
	out, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if pre := hooks["PreToolUse"].([]any); len(pre) != 5 {
		t.Fatalf("want the owner's two groups and the guard's three:\n%s", out)
	}
	has(t, string(out), "my-own-check", "echo markitect hook pretool ran", guard.MarkRule("PowerShell"))
	if strings.Contains(string(out), "MARKITECT_HOOK") || !guard.HasGuard(out) {
		t.Fatalf("%s", out)
	}
}
