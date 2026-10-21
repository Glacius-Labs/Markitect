package run

import (
	"strings"
	"sync"
	"testing"
)

func assessSpec() PassSpec {
	return PassSpec{Purpose: Assess, Mandate: "Assess the bound model.", Assesses: true, Corrections: []Purpose{Assess}}
}

// libraryGap is the material gap the fixture's library reviewer reports.
var libraryGap = map[string]any{"owner": "library", "obligation": "library/interface", "observed": "the model leaves the maximum name length open", "blocking": true}

// An accepted model assessment releases the work: it runs read-only through
// the same base process, and the work Pass follows.
func TestAnAcceptedAssessmentReleasesTheWork(t *testing.T) {
	f := newFixture(t)
	r := f.open()
	if res := r.Pass(assessSpec()); res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("assessment %+v", res)
	}
	if res := r.Pass(implementSpec()); res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("work %+v", res)
	}
	out := f.finish(r, "")
	if len(out.Passes) != 2 || out.Passes[0].Purpose != Assess || out.Passes[1].Purpose != Implement {
		t.Fatalf("passes %+v", out.Passes)
	}
	for _, turn := range f.agents.turns {
		if strings.Contains(turn.Prompt, "Pass p1: assess") && (!turn.ReadOnly || writable(turn, "library/a.go")) {
			t.Errorf("assessment turn %s can write", turn.ID)
		}
	}
}

// A material problem that the tree upholds up to Root ends the run Failed
// when the assessment seals; no work Pass starts after it.
func TestAConfirmedModelProblemEndsTheRunBeforeAnyWork(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	raised := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit your complete result") && !raised {
			raised = true
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []map[string]any{libraryGap}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(assessSpec())
	if res.Verdict != NeedsCorrection {
		t.Fatalf("assessment %+v", res)
	}
	if next := r.Pass(implementSpec()); !next.Stopped {
		t.Fatalf("work started after a confirmed model problem: %+v", next)
	}
	out := f.finish(r, "")
	if out.Status != Failed || !strings.HasPrefix(out.Reason, ModelAssessmentReason) || !strings.Contains(out.Reason, "maximum name length") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	for _, turn := range f.agents.turns {
		if !strings.Contains(turn.Prompt, "Pass p1: assess") && !strings.Contains(turn.Prompt, "standing") && strings.Contains(turn.Prompt, "Pass p2") {
			t.Errorf("turn %s ran after the assessment", turn.ID)
		}
	}
}

// A parent challenges a finding it judges unsupported by returning it; a
// withdrawn finding never takes effect, and the work proceeds.
func TestAChallengedFindingDoesNotEndTheRun(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	raised, returned := false, false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit your complete result") && !raised {
			raised = true
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []map[string]any{libraryGap}}))
			return nil
		}
		if c.turn.Manager == "." && !returned {
			for _, id := range c.pendingDecisions() {
				if strings.Contains(c.turn.Prompt, "maximum name length") {
					returned = true
					if c.decided == nil {
						c.decided = map[string]bool{}
					}
					c.decided[id] = true
					c.must(toolReport, map[string]any{"kind": kindReturn, "proposal": id, "reason": "library/interface fixes the limit by its wording"})
				}
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	if res := r.Pass(assessSpec()); res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("assessment %+v", res)
	}
	if res := r.Pass(implementSpec()); res.Stopped {
		t.Fatalf("work %+v", res)
	}
	out := f.finish(r, "")
	if !returned || len(out.Findings) != 0 {
		t.Fatalf("returned %v, findings %+v", returned, out.Findings)
	}
}

// A model assessment is read-only work of the base process.
func TestAModelAssessmentIsReadOnlyWork(t *testing.T) {
	for _, spec := range []PassSpec{
		{Purpose: Assess, Mandate: "m", Assesses: true, Writes: true, Corrections: []Purpose{Assess}},
		{Purpose: Assess, Mandate: "m", Assesses: true, Review: true, Guidance: "g", Corrections: []Purpose{Assess}},
		{Purpose: Assess, Mandate: "m", Assesses: true, Plans: true, Corrections: []Purpose{Assess}},
		{Purpose: Assess, Mandate: "m", Assesses: true, BindWaits: true, Corrections: []Purpose{Assess}},
	} {
		if validateSpec(spec) == nil {
			t.Errorf("accepted %+v", spec)
		}
	}
	if err := validateSpec(assessSpec()); err != nil {
		t.Fatal(err)
	}
}
