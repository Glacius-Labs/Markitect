package run

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// A turn whose input the provider channel cannot carry is not repeated, which
// would only grow its input: the run ends Incomplete at once, naming why.
func TestAContextLimitEndsTheRunWithoutRetry(t *testing.T) {
	f := newFixture(t)
	turns := 0
	f.agents.script = func(c *call) error {
		turns++
		return fmt.Errorf("%w: the prompt of 2000000 bytes exceeds the 1048576 bytes the provider channel carries", ErrContextLimit)
	}
	r := f.open()
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Status != Incomplete || !strings.Contains(out.Reason, "exceeds the 1048576 bytes") || turns != 1 {
		t.Fatalf("status %s reason %q after %d turns", out.Status, out.Reason, turns)
	}
	if len(out.Problems) != 1 || !strings.Contains(out.Problems[0].Text, "2000000 bytes") {
		t.Fatalf("problems %+v", out.Problems)
	}
}

// An answer above the bound is refused to the Agent with its size, never cut,
// and the lookup's record says so: what it records is what the Agents port
// received.
func TestAnOversizedAnswerIsRefusedAndRecordedAsSuch(t *testing.T) {
	f := newFixture(t)
	huge := strings.Repeat("the full text of a very long Definition. ", MaxToolAnswer/40+1)
	f.cfg.Definition = func(id string) (ModelDefinition, bool) {
		return ModelDefinition{ID: id, Owner: "library", Text: huge}, id == "library/interface"
	}
	var answer ToolAnswer
	var mu sync.Mutex
	f.agents.script = func(c *call) error {
		mu.Lock()
		if c.turn.Manager == "library" && answer.Text == "" {
			answer = c.tool(toolLookUp, map[string]any{"topic": "definition", "id": "library/interface"})
		}
		mu.Unlock()
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	f.finish(r, "")
	if !answer.Refused || !strings.Contains(answer.Text, "exceeds the 262144 bytes") || strings.Contains(answer.Text, "very long Definition") {
		t.Fatalf("answer %q", answer.Text[:min(200, len(answer.Text))])
	}
	events := f.ev.kinds("lookup.definition")
	data, _ := events[0].Data.(map[string]any)
	if len(events) != 1 || data["refused"] != true || data["requested"].(int) <= MaxToolAnswer || data["bytes"] != len(answer.Text) || data["answer"] != textDigest(answer.Text) {
		t.Fatalf("lookup record %+v", events)
	}
}
