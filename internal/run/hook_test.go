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

// Only the reviewer answers: an agent cannot mark its own work as done.
func TestGuard(t *testing.T) {
	call := func(agent, command string) string {
		return string(run.Guard(hookInput(t, map[string]any{
			"agent_type": agent, "tool_name": "Bash", "tool_input": map[string]any{"command": command},
		})))
	}
	has(t, call("", "markitect answer <<'EOF'\nx\nEOF"), `"permissionDecision":"deny"`, run.ReviewerAgent)
	has(t, call(run.BuilderAgent, "cd . && markitect answer f.txt"), `"permissionDecision":"deny"`)
	if out := call(run.ReviewerAgent, "markitect answer f.txt"); out != "" {
		t.Fatalf("refused the reviewer: %s", out)
	}
	if out := call(run.BuilderAgent, "go test ./..."); out != "" {
		t.Fatalf("%s", out)
	}
	if out := call(run.BuilderAgent, `markitect gate --spec-problem "the limit is 3 or 5"`); out != "" {
		t.Fatalf("refused the builder's report: %s", out)
	}
	if out := string(run.Guard([]byte("not json"))); out != "" {
		t.Fatalf("%s", out)
	}
}

// Only the owner's session begins the work or names a change: an agent that
// changed the spec cannot start over with its change as the owner's.
func TestGuardKeepsAgentsToTheWorkAsItIs(t *testing.T) {
	call := func(agentID, agentType, command string) string {
		return string(run.Guard(hookInput(t, map[string]any{
			"agent_id": agentID, "agent_type": agentType, "tool_name": "Bash", "tool_input": map[string]any{"command": command},
		})))
	}
	for _, command := range []string{"markitect checklist --start", "cd . && markitect checklist --full", "markitect checklist --base x.json", "markitect checklist --findings f.md", "markitect checklist -start", "markitect  checklist\t-base=x.json"} {
		has(t, call("a1", run.BuilderAgent, command), `"permissionDecision":"deny"`, "Only the owner's session begins the work")
		if out := call("", "", command); out != "" {
			t.Fatalf("refused the owner's session %q: %s", command, out)
		}
	}
	for _, command := range []string{"markitect checklist", "markitect checklist --review", "markitect gate"} {
		if out := call("a2", run.ReviewerAgent, command); out != "" {
			t.Fatalf("refused an agent %q: %s", command, out)
		}
	}
}

func TestHasGuard(t *testing.T) {
	hook := func(event, matcher string, ifs ...string) string {
		var handlers []string
		for _, rule := range ifs {
			handlers = append(handlers, `{"type": "command", "if": "`+rule+`", "command": "`+run.HookCommand+`"}`)
		}
		return `{"hooks": {"` + event + `": [{"matcher": "` + matcher + `", "hooks": [` + strings.Join(handlers, ", ") + `]}]}}`
	}
	answer, checklist := run.GuardedCommands[0], run.GuardedCommands[1]
	for settings, want := range map[string]bool{
		hook("PreToolUse", "Bash", ""):                 true,
		hook("PreToolUse", "", answer, checklist):      true,
		hook("PreToolUse", "Bash", answer):             false,
		hook("PreToolUse", "Edit", ""):                 false,
		hook("PostToolUse", "Bash", answer, checklist): false,
		`{"note": "` + run.HookCommand + `"}`:          false,
		"not json":                                     false,
	} {
		if got := run.HasGuard([]byte(settings)); got != want {
			t.Errorf("%s: %v", settings, got)
		}
	}
}
