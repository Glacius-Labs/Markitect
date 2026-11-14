package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	runs "github.com/Glacius-Labs/Markitect/internal/run"
)

// A driver tells the history that a run starts, that a slice starts and that
// the run ends; the tool adds the time, the list, the harness configuration and
// what the record says then (DEC-114). A run without an end leaves the history
// readable. What the hook refuses lands in the history too.
func TestRunEventsAndRefusals(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	events := func() []runs.Event {
		t.Helper()
		return runs.ParseHistory([]byte(read(t, r.Dir, ".markitect/history.jsonl"))).Events
	}
	if out := mustRun(t, r.Dir, "run", "start", "build", "--strategy", "sequential"); !strings.Contains(out, "Run started: build, 1 item on the list in 1 slice.") {
		t.Fatalf("run start: %s", out)
	}
	start := events()[0]
	if start.Type != runs.EventRunStart || start.Kind != "build" || start.Strategy != "sequential" || start.Trigger != "owner" || start.Items == nil || *start.Items != 1 || *start.Slices != 1 || start.Roles["reviewer"].Model != "sonnet" || len(start.Roles) != 4 {
		t.Fatalf("run start: %+v", start)
	}
	mustRun(t, r.Dir, "run", "slice")
	verifyAll(t, r.Dir)
	if out := mustRun(t, r.Dir, "run", "end", "done", "--rounds", "1"); !strings.Contains(out, "Run ended: done; 1 verified, 0 left") {
		t.Fatalf("run end: %s", out)
	}
	ev := events()
	var types []string
	for _, e := range ev {
		types = append(types, e.Type)
	}
	if got := strings.Join(types, " "); got != "run-start run-slice verdict run-end" {
		t.Fatalf("events: %s", got)
	}
	end := ev[3]
	if end.Outcome != "done" || end.Rounds == nil || *end.Rounds != 1 || end.Counts == nil || end.Counts.Verified != 1 || end.Counts.Left != 0 {
		t.Fatalf("run end: %+v %+v", end, end.Counts)
	}
	// A verdict belongs to the run whose start and end enclose its time.
	if v := ev[2]; v.Time.Before(ev[0].Time) || v.Time.After(end.Time) {
		t.Fatalf("the verdict lies outside its run: %v %v %v", ev[0].Time, v.Time, end.Time)
	}
	// What the driver may not say is refused, and nothing is appended.
	for _, args := range [][]string{
		{"run"},
		{"run", "begin"},
		{"run", "start", "deploy"},
		{"run", "start", "build", "--trigger", "cron"},
		{"run", "start", "build", "--strategy", "auto"},
		{"run", "end", "fine"},
		{"run", "end", "done", "--rounds", "-1"},
		{"run", "slice", ","},
		{"run", "slice", "--rounds", "2"},
		{"run", "end", "done", "--trigger", "owner"},
	} {
		if code, _, _ := run(t, r.Dir, args...); code != 2 {
			t.Fatalf("markitect %s: exit %d", strings.Join(args, " "), code)
		}
	}
	// A run that never ends leaves the history readable.
	mustRun(t, r.Dir, "run", "start", "review", "--trigger", "claude")
	if ev := events(); len(ev) != 5 || ev[4].Type != runs.EventRunStart || ev[4].Kind != "review" {
		t.Fatalf("an open run: %+v", ev)
	}
	// The hook appends what it refuses to the history of the project the call
	// came from, and nothing for a call it lets run.
	hook := func(agent, file string) string {
		t.Helper()
		input, _ := json.Marshal(map[string]any{"cwd": r.Dir, "agent_id": "a1", "agent_type": agent, "tool_name": "Edit", "tool_input": map[string]any{"file_path": filepath.Join(r.Dir, file)}})
		var out, errOut bytes.Buffer
		if code := newCLI(bytes.NewReader(input), &out, &errOut).Run(context.Background(), []string{"hook", "pretool"}); code != 0 {
			t.Fatalf("hook exit %d: %s", code, errOut.String())
		}
		return out.String()
	}
	if out := hook("markitect-builder", "src/main.go"); out != "" || len(events()) != 5 {
		t.Fatalf("a call the hook lets run: %s", out)
	}
	if out := hook("markitect-builder", ".markitect/review.sum"); !strings.Contains(out, "deny") {
		t.Fatalf("the hook let the builder write the record: %s", out)
	}
	if ev := events(); len(ev) != 6 || ev[5].Type != runs.EventRefusal || ev[5].Agent != "markitect-builder" || ev[5].Tool != "Edit" || !strings.HasSuffix(filepath.ToSlash(ev[5].Target), ".markitect/review.sum") || !strings.Contains(ev[5].Why, "markitect mark") {
		t.Fatalf("the refusal: %+v", ev[len(ev)-1])
	}
}
