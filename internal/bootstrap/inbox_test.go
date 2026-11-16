package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	runs "github.com/Glacius-Labs/Markitect/internal/run"
)

// The inbox through the command line (DEC-118): add opens an item once, the
// list and status --json count the open ones, those for the owner apart, and
// close closes one; the hook lets a subagent close only an item an agent may
// settle; an audit's end counts what it opened, by kind.
func TestInboxCommands(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	add := func(args ...string) string {
		t.Helper()
		var doc struct {
			Opened bool
			Item   struct{ ID string }
		}
		out := mustRun(t, r.Dir, append(append([]string{"inbox", "add"}, args...), "--json")...)
		if json.Unmarshal([]byte(out), &doc) != nil || doc.Item.ID == "" {
			t.Fatalf("inbox add --json: %s", out)
		}
		return doc.Item.ID
	}
	if out := mustRun(t, r.Dir, "inbox"); out != "The inbox holds no open item.\n" {
		t.Fatalf("an empty inbox: %s", out)
	}
	mustRun(t, r.Dir, "run", "start", "audit")
	question := add("--source", "audit", "--kind", "question", "--settle", "owner", "-m", "Is a member a person or a household?", "--place", "project", "--ask", "Which one?")
	link := add("--source", "audit", "--kind", "missing-link", "--settle", "agent", "-m", "Big names the project without a link.", "--place", "big,project")
	if again := add("--source", "audit", "--kind", "question", "--settle", "owner", "-m", "Is a member a person or a household?", "--place", "project"); again != question {
		t.Fatalf("an item open already was opened again: %s %s", again, question)
	}
	mustRun(t, r.Dir, "run", "end", "done")
	ev := runs.ParseHistory([]byte(read(t, r.Dir, ".markitect/history.jsonl"))).Events
	if end := ev[len(ev)-1]; end.Found["question"] != 1 || end.Found["missing-link"] != 1 || len(end.Found) != 2 {
		t.Fatalf("an audit's end: %+v", end.Found)
	}
	out := mustRun(t, r.Dir, "inbox")
	wantAll(t, out, "The inbox holds 2 open items, 1 for the owner:", question+"  for the owner  question, from audit  at project\n  Is a member a person or a household?\n  asks: Which one?\n", link+"  for the agent  missing-link, from audit  at big, project\n")
	var status struct {
		Inbox struct {
			Open     int
			ForOwner int `json:"for_owner"`
		}
	}
	if _, out, _ := run(t, r.Dir, "status", "--json"); json.Unmarshal([]byte(out), &status) != nil || status.Inbox.Open != 2 || status.Inbox.ForOwner != 1 {
		t.Fatalf("status --json: %s", out)
	}
	// What the inbox does not take is refused.
	for _, args := range [][]string{
		{"inbox", "add", "--source", "audit", "--kind", "gap", "--settle", "nobody", "-m", "x"},
		{"inbox", "add", "--source", "audit", "--kind", "gap", "--settle", "owner"},
		{"inbox", "add", "--source", "audit", "--kind", "gap", "--settle", "owner", "-m", "x", "--settled"},
		{"inbox", "close", link},
		{"inbox", "close", link, "--settled", "--dropped", "-m", "x"},
		{"inbox", "close", link, "--settled", "-m", "x", "--kind", "gap"},
		{"inbox", "list"},
		{"inbox", "--source", "audit"},
	} {
		if code, _, _ := run(t, r.Dir, args...); code != 2 {
			t.Fatalf("markitect %s: exit %d", strings.Join(args, " "), code)
		}
	}
	// The hook lets a subagent close an item for an agent, and refuses one for
	// the owner; the owner's session closes any.
	hook := func(agentID, id string) string {
		t.Helper()
		input, _ := json.Marshal(map[string]any{"cwd": r.Dir, "agent_id": agentID, "agent_type": "markitect-architect", "tool_name": "Bash", "tool_input": map[string]any{"command": "markitect inbox close " + id + " --settled -m done"}})
		var out, errOut bytes.Buffer
		if code := newCLI(bytes.NewReader(input), &out, &errOut).Run(context.Background(), []string{"hook", "pretool"}); code != 0 {
			t.Fatalf("hook exit %d: %s", code, errOut.String())
		}
		return out.String()
	}
	if out := hook("a1", link); out != "" {
		t.Fatalf("a subagent may not close an item for an agent: %s", out)
	}
	if out := hook("a1", question); !strings.Contains(out, "waits for the owner") {
		t.Fatalf("a subagent may close an item for the owner: %s", out)
	}
	if out := hook("", question); out != "" {
		t.Fatalf("the owner's session may not close an item: %s", out)
	}
	if out := mustRun(t, r.Dir, "inbox", "close", link, "--settled", "-m", "linked from big"); out != "Closed "+link+": settled.\n" {
		t.Fatalf("inbox close: %s", out)
	}
	if code, out, _ := run(t, r.Dir, "inbox", "close", link, "--dropped", "-m", "again"); code != 1 || !strings.Contains(out, "closed already") {
		t.Fatalf("inbox close twice: %d %s", code, out)
	}
	var all struct {
		Counts struct{ Open int }
		Items  []struct {
			ID     string
			Closed *struct{ How string }
		}
	}
	if out := mustRun(t, r.Dir, "inbox", "--all", "--json"); json.Unmarshal([]byte(out), &all) != nil || all.Counts.Open != 1 || len(all.Items) != 2 || all.Items[1].Closed == nil || all.Items[1].Closed.How != "settled" {
		t.Fatalf("inbox --all --json: %s", out)
	}
}
