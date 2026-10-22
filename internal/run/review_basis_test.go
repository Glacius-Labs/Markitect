package run

import (
	"strings"
	"sync"
	"testing"
)

// Scripted Agents establish direct exposure, participation and identity, not
// semantic review quality. No reviewer submits a preliminary basis.
func TestReviewReceivesCandidateDirectlyWithoutBasis(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	first := map[string]bool{}
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if !first[c.turn.Manager] {
			first[c.turn.Manager] = true
			if c.turn.Kind != VerifyTurn || !c.turn.ReadOnly || c.turn.Resume != "" || c.turn.Home == "" {
				t.Errorf("not a fresh independent reviewer: %+v", c.turn)
			}
			f.ws.mu.Lock()
			_, candidate := f.ws.byName[c.turn.Workspace].files["library/a.go"]
			f.ws.mu.Unlock()
			if !candidate {
				t.Error("candidate withheld from first review Turn")
			}
			if strings.Contains(c.turn.Prompt, "MODEL-FIRST BASIS") {
				t.Error("obsolete basis barrier")
			}
			c.must(toolLookUp, map[string]any{"topic": "definition", "id": "library/interface"})
			c.must(toolLookUp, map[string]any{"topic": "checks"})
		}
		for _, id := range c.pendingDecisions() {
			c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": id})
		}
		if strings.HasPrefix(c.now(), "your result") || strings.HasPrefix(c.now(), "your complete result is accepted") {
			return nil
		}
		if a := c.result(nil); a.Refused {
			if !strings.Contains(a.Text, "children's results") {
				t.Errorf("direct result: %+v", a)
			}
			for _, m := range f.cfg.Team {
				if m.Path == c.turn.Manager {
					for _, child := range m.Children {
						if f.ev.index("result.accepted", child, "", nil) < 0 {
							c.must(toolReport, map[string]any{"kind": kindWait, "for": child, "reason": "integrate child result"})
							return nil
						}
					}
				}
			}
		}
		return nil
	}
	r := f.open()
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if res.Stopped || out.Status != Verified {
		t.Fatalf("direct review: %+v %s", res, out.Reason)
	}
	if len(first) != len(teamPaths) || len(f.ev.kinds("basis.accepted")) != 0 {
		t.Fatalf("participation %v or unnecessary basis", first)
	}
	for _, p := range out.Passes {
		for _, m := range p.Managers {
			if m.Basis != "" || m.Result == "" {
				t.Fatalf("report %+v", m)
			}
		}
	}
}
