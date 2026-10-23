package run

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// look_up answers structural facts with their source and refuses unknown
// subjects: Definitions with their owner, owners of paths, states, accepted
// plans and the method guidance.
func TestLookUpAnswersStructuralFactsAndMethods(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	answers := map[string]string{}
	looked := make(chan struct{})
	f.agents.script = func(c *call) error {
		if c.pass() == "p2" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "work on") {
			if err := c.wait(looked); err != nil {
				return err
			}
		}
		mu.Lock()
		defer mu.Unlock()
		if c.pass() == "p2" && c.turn.Manager == "lending" && len(answers) == 0 {
			look := func(key string, args map[string]any) {
				answers[key] = c.tool(toolLookUp, args).Text
			}
			look("own", map[string]any{"topic": "definition", "id": "lending/borrow"})
			look("foreign", map[string]any{"topic": "definition", "id": "library/interface"})
			look("missing", map[string]any{"topic": "definition", "id": "lending/none"})
			look("owner", map[string]any{"topic": "owner", "path": "lending/reminders/x.go"})
			look("state", map[string]any{"topic": "state"})
			look("plan", map[string]any{"topic": "plan", "manager": "library"})
			look("basis", map[string]any{"topic": "basis"})
			look("method", map[string]any{"topic": "method", "id": "reviewing"})
			look("bad method", map[string]any{"topic": "method", "id": "guessing"})
			close(looked)
		}
		c.protocol()
		return nil
	}
	const rootDuty = "As Root you end the run for a defective model."
	f.cfg.RootGuidance = rootDuty
	r := f.open()
	r.Pass(planSpec())
	r.Pass(prepareSpec())
	answers["accepted basis"] = r.e.lookUp(r.e.cur.byPath["lending"], nil, json.RawMessage(`{"topic":"basis"}`)).Text
	f.finish(r, "")
	// The standing context of a fresh conversation names the methods.
	if first := f.agents.turnsOf("lending")[0]; !strings.Contains(first.Prompt, "Method guidance with look_up topic method") || strings.Contains(first.Prompt, rootDuty) {
		t.Errorf("the standing context does not name the methods, or a child carries Root's guidance:\n%s", first.Prompt)
	}
	// Root's standing context carries the Root guidance of the run.
	if first := f.agents.turnsOf(".")[0]; !strings.Contains(first.Prompt, rootDuty) {
		t.Errorf("Root's standing context lacks its guidance:\n%s", first.Prompt)
	}
	for key, want := range map[string]string{
		"own":            "your own obligation",
		"foreign":        "context owned by library",
		"missing":        "no Definition of the bound model",
		"owner":          "owned by lending/reminders",
		"state":          "Pass p2 (prepare)",
		"plan":           "Accepted plan",
		"basis":          "No accepted basis",
		"accepted basis": "Accepted basis",
		"method":         "authority of this run",
		"bad method":     "id names a method",
	} {
		if !strings.Contains(answers[key], want) {
			t.Errorf("%s: %q lacks %q", key, answers[key], want)
		}
	}
}
