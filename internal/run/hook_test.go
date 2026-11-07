package run_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

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
		return string(run.Guard(hookInput(t, map[string]any{
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
	} {
		has(t, call("a1", run.BuilderAgent, command), `"permissionDecision":"deny"`, run.ReviewerAgent)
		has(t, call("", "", command), `"permissionDecision":"deny"`)
		if out := call("a2", run.ReviewerAgent, command); out != "" {
			t.Fatalf("refused the reviewer %q: %s", command, out)
		}
	}
	for _, command := range []string{"markitect status", "markitect status --all", "go test ./...", "cat .claude/agents/markitect-reviewer.md", "echo markitect marked", "markitect markdown x"} {
		if out := call("a1", run.BuilderAgent, command); out != "" {
			t.Fatalf("refused %q: %s", command, out)
		}
	}
	if out := string(run.Guard([]byte("not json"))); out != "" {
		t.Fatalf("%s", out)
	}
}

// A subagent writes only what its role may (DEC-100): the builder, and any other
// subagent, code and the registry pages; the architect the spec and the registry
// pages; the reviewer and the auditor nothing; nobody the record, which mark
// writes. The owner's own session is free.
func TestGuardHoldsEachRolesRightToWrite(t *testing.T) {
	call := func(agentID, agent, tool, key, file string) string {
		return string(run.Guard(hookInput(t, map[string]any{
			"agent_id": agentID, "agent_type": agent, "tool_name": tool, "tool_input": map[string]any{key: file},
		})))
	}
	files := map[string][]string{
		"spec": {
			"/p/.markitect/model/fees/fee.md",
			"/p/.markitect/manifest.yaml",
			"/p/.markitect/ignore",
			`C:\p\.markitect\model\fees\fee.md`,
			".markitect/registry/../model/fee.md",
			"/p/.Markitect/model/fee.md",
			"/p/.markitect/registry/notes.txt",
		},
		"registry": {
			"/p/.markitect/registry.md",
			"/p/.markitect/registry/fees.md",
			`C:\p\.markitect\registry\members\loans.md`,
		},
		"record": {"/p/.markitect/review.sum", `C:\p\.MARKITECT\review.sum`},
		"harness": {
			"/p/.claude/settings.json",
			`C:\p\.claude\settings.local.json`,
			"/p/.claude/agents/markitect-reviewer.md",
			"/p/.claude/skills/markitect-build/SKILL.md",
			"/p/.agents/skills/markitect-design/reference.md",
			"/p/.claude/worktrees/agent-a1/.claude/settings.json",
			"/p/AGENTS.md",
			"/p/CLAUDE.md",
			"/p/docs/AGENTS.md",
		},
		"code": {
			"/p/src/fees.go",
			"/p/docs/markitect.md",
			"/p/.markitect-notes/x.md",
			"/p/.claude/worktrees/agent-a1/src/fees.go",
			"/p/.claude/agents/my-own.md",
			"/p/.claude/commands/settings.json",
			"/p/docs/agents-guide.md",
			"/p/docs/agents.md",
		},
	}
	may := map[string]map[string]bool{
		run.BuilderAgent:    {"code": true, "registry": true},
		"Markitect-Builder": {"code": true, "registry": true},
		"general-purpose":   {"code": true, "registry": true},
		"":                  {"code": true, "registry": true},
		run.ArchitectAgent:  {"spec": true, "registry": true},
		run.ReviewerAgent:   {},
		run.AuditorAgent:    {},
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
	has(t, call("a1", run.BuilderAgent, "Edit", "file_path", "/p/.markitect/model/fee.md"), "the registry pages")
	has(t, call("a1", run.BuilderAgent, "Edit", "file_path", "/p/.markitect/review.sum"), "markitect mark")
	has(t, call("a1", run.BuilderAgent, "Edit", "file_path", "/p/.claude/settings.json"), "steers the agents")
	has(t, call("a1", run.ArchitectAgent, "Edit", "file_path", "/p/src/fees.go"), "the builder writes")
	has(t, call("a1", run.AuditorAgent, "Edit", "file_path", "/p/src/fees.go"), "changes no file")
	has(t, call("a1", run.BuilderAgent, "NotebookEdit", "notebook_path", "/p/.markitect/x.ipynb"), `"permissionDecision":"deny"`)
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
		return string(run.Guard(hookInput(t, map[string]any{
			"agent_id": agentID, "agent_type": agent, "tool_name": tool, "tool_input": input,
		})))
	}
	allowed := []struct{ id, agent, start string }{
		{"a1", run.BuilderAgent, run.ReviewerAgent},
		{"a1", "general-purpose", "Explore"},
		{"a1", "general-purpose", ""},
		{"a1", "general-purpose", run.ReviewerAgent},
		{"", "", run.BuilderAgent},
		{"", "", run.ArchitectAgent},
	}
	for _, c := range allowed {
		for _, tool := range []string{"Agent", "Task"} {
			if out := call(c.id, c.agent, tool, c.start); out != "" {
				t.Fatalf("refused %q starting %q: %s", c.agent, c.start, out)
			}
		}
	}
	refused := []struct{ agent, start, why string }{
		{run.BuilderAgent, "", run.ReviewerAgent},
		{run.BuilderAgent, "general-purpose", run.ReviewerAgent},
		{run.BuilderAgent, run.ArchitectAgent, run.ReviewerAgent},
		{run.BuilderAgent, run.BuilderAgent, run.ReviewerAgent},
		{run.ReviewerAgent, run.ReviewerAgent, "starts no agent"},
		{run.ArchitectAgent, "Explore", "starts no agent"},
		{run.AuditorAgent, "", "starts no agent"},
		{"general-purpose", run.ArchitectAgent, "owner's session"},
		{"general-purpose", run.BuilderAgent, "owner's session"},
		{"general-purpose", "Markitect-Architect", "owner's session"},
		{"Markitect-Builder", "general-purpose", run.ReviewerAgent},
	}
	for _, c := range refused {
		for _, tool := range []string{"Agent", "Task"} {
			has(t, call("a1", c.agent, tool, c.start), `"permissionDecision":"deny"`, c.why)
		}
	}
}

func TestHasGuard(t *testing.T) {
	group := func(matcher, rule string) string {
		return `{"matcher": "` + matcher + `", "hooks": [{"type": "command", "if": "` + rule + `", "command": "` + run.HookCommand + `"}]}`
	}
	settings := func(groups ...string) string {
		return `{"hooks": {"PreToolUse": [` + strings.Join(groups, ", ") + `]}}`
	}
	for s, want := range map[string]bool{
		settings(group("Bash", run.MarkRule), group(run.EditTools, ""), group(run.AgentTools, "")): true,
		settings(group("", "")): true,
		settings(group("Bash", ""), group("Write|Edit|NotebookEdit|MultiEdit|Task|Agent", "")):                       true,
		settings(group("Bash", run.MarkRule), group(run.EditTools, "")):                                              false,
		settings(group("Bash", run.MarkRule), group(run.EditTools, ""), group("Agent", "")):                          false,
		settings(group("Bash", run.MarkRule)):                                                                        false,
		settings(group(run.EditTools, ""), group(run.AgentTools, "")):                                                false,
		settings(group("Bash", run.MarkRule), group("Edit|Write", ""), group(run.AgentTools, "")):                    false,
		settings(group("Bash", "Bash(markitect status*)"), group(run.EditTools, ""), group(run.AgentTools, "")):      false,
		settings(group("Bash", run.MarkRule), group(run.EditTools, ""), group(run.AgentTools, "Agent(markitect-*)")): false,
		`{"hooks": {"PostToolUse": [` + group("", "") + `]}}`:                                                        false,
		`{"note": "` + run.HookCommand + `"}`:                                                                        false,
		"not json":                                                                                                   false,
	} {
		if got := run.HasGuard([]byte(s)); got != want {
			t.Errorf("%s: %v", s, got)
		}
	}
}
