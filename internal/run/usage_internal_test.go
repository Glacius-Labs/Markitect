package run

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
)

// A Result counts every start's tokens, also those of a start the Run ended,
// and says when a total is a lower bound.
func TestTheResultCountsTheTokensOfEndedStarts(t *testing.T) {
	r := &Result{Starts: []AgentStart{
		{Manager: ".", Role: agentport.Maker, Reason: agentport.Finished, Usage: agentport.Usage{InputTokens: 100, OutputTokens: 10}},
		{Manager: "materials", Role: agentport.Maker, Reason: agentport.Cancelled, Usage: agentport.Usage{InputTokens: 50, OutputTokens: 2, Partial: true}},
	}}
	(&Engine{}).stamp(r)
	if r.Usage != (agentport.Usage{InputTokens: 150, OutputTokens: 12, Partial: true}) {
		t.Fatalf("usage %+v", r.Usage)
	}
	if page := string(Page(r)); !strings.Contains(page, "at least 150 in, 12 out: a start reported no total, for example one the Run ended") {
		t.Fatalf("page:\n%s", page)
	}
	r.Starts = r.Starts[:1]
	(&Engine{}).stamp(r)
	if page := string(Page(r)); r.Usage.Partial || strings.Contains(page, "at least") || !strings.Contains(page, "100 in, 10 out") {
		t.Fatalf("a complete total %+v:\n%s", r.Usage, page)
	}
	if b, _ := json.Marshal(r.Usage); string(b) != `{"inputTokens":100,"outputTokens":10}` {
		t.Fatalf("a complete total's JSON: %s", b)
	}
}
