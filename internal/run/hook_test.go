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

// No subagent writes under .markitect except the registry pages, which the
// builder keeps; the owner's own session is free.
func TestGuardKeepsSubagentsOutOfMarkitectsFolder(t *testing.T) {
	call := func(agentID, tool, key, file string) string {
		return string(run.Guard(hookInput(t, map[string]any{
			"agent_id": agentID, "agent_type": run.BuilderAgent, "tool_name": tool, "tool_input": map[string]any{key: file},
		})))
	}
	for _, file := range []string{
		"/p/.markitect/model/fees/fee.md",
		"/p/.markitect/review.sum",
		"/p/.markitect/manifest.yaml",
		`C:\p\.markitect\model\fees\fee.md`,
		".markitect/registry/../model/fee.md",
		"/p/.Markitect/model/fee.md",
		"/p/.markitect/registry/notes.txt",
	} {
		for _, tool := range []string{"Edit", "Write", "MultiEdit"} {
			has(t, call("a1", tool, "file_path", file), `"permissionDecision":"deny"`, "the registry pages")
		}
		if out := call("", "Write", "file_path", file); out != "" {
			t.Fatalf("refused the owner's session %s: %s", file, out)
		}
	}
	has(t, call("a1", "NotebookEdit", "notebook_path", "/p/.markitect/x.ipynb"), `"permissionDecision":"deny"`)
	for _, file := range []string{
		"/p/.markitect/registry.md",
		"/p/.markitect/registry/fees.md",
		`C:\p\.markitect\registry\members\loans.md`,
		"/p/src/fees.go",
		"/p/docs/markitect.md",
		"/p/.markitect-notes/x.md",
	} {
		if out := call("a1", "Edit", "file_path", file); out != "" {
			t.Fatalf("refused %s: %s", file, out)
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
		settings(group("Bash", run.MarkRule), group(run.EditTools, "")): true,
		settings(group("", "")): true,
		settings(group("Bash", ""), group("Write|Edit|NotebookEdit|MultiEdit", "")):  true,
		settings(group("Bash", run.MarkRule)):                                        false,
		settings(group(run.EditTools, "")):                                           false,
		settings(group("Bash", run.MarkRule), group("Edit|Write", "")):               false,
		settings(group("Bash", "Bash(markitect status*)"), group(run.EditTools, "")): false,
		`{"hooks": {"PostToolUse": [` + group("", "") + `]}}`:                        false,
		`{"note": "` + run.HookCommand + `"}`:                                        false,
		"not json":                                                                   false,
	} {
		if got := run.HasGuard([]byte(s)); got != want {
			t.Errorf("%s: %v", s, got)
		}
	}
}
