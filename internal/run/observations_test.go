package run

import (
	"strings"
	"sync"
	"testing"
)

// resultTurn reports whether the turn submits a complete result, after it
// decided its children's proposals.
func (c *call) resultTurn() bool {
	if now := c.now(); !strings.HasPrefix(now, "submit your complete result") && !strings.HasPrefix(now, "integrate your children") {
		return false
	}
	for _, id := range c.pendingDecisions() {
		if !c.decided[id] {
			c.accept(id)
		}
	}
	return true
}

// suggestion is an optional note a reviewer adds to its result.
func suggestion(owner, obligation string) map[string]any {
	return map[string]any{"kind": string(Suggestion), "owner": owner, "obligation": obligation, "observed": "a clearer name would read better", "blocking": false}
}

// suspicion is a concrete, unresolved conformity question.
func suspicion(owner, obligation string) map[string]any {
	return map[string]any{"kind": string(Suspected), "owner": owner, "obligation": obligation, "expected": "a notice on the due date", "observed": "the notice may be sent a day late; the test does not cover the boundary", "blocking": true}
}

// The declared effect of each kind is enforced, never the truth of the
// classification: a suggestion never blocks, every other kind blocks, a
// deviation or suspicion states its expected outcome, and each Pass knows only
// its own kinds.
func TestFindingKindsHaveTheirDeclaredEffects(t *testing.T) {
	work, assess := implementSpec(), assessSpec()
	no, yes := false, true
	for _, tc := range []struct {
		name string
		spec PassSpec
		in   findingIn
		want Observation
		err  string
	}{
		{"default genuine", work, findingIn{Blocking: &yes, Expected: "e"}, Deviation, ""},
		{"default note", work, findingIn{Blocking: &no}, Suggestion, ""},
		{"default in the assessment", assess, findingIn{Blocking: &yes}, ModelProblem, ""},
		{"suspicion", work, findingIn{Kind: "suspected", Blocking: &yes, Expected: "e"}, Suspected, ""},
		{"a blocking suggestion", work, findingIn{Kind: "suggestion", Blocking: &yes}, "", "a suggestion never blocks"},
		{"a non-blocking deviation", work, findingIn{Kind: "deviation", Blocking: &no, Expected: "e"}, "", "a deviation blocks"},
		{"a deviation without its expected outcome", work, findingIn{Blocking: &yes}, "", "states the outcome its obligation requires"},
		{"a suspicion without its expected outcome", work, findingIn{Kind: "suspected", Blocking: &yes}, "", "states the outcome its obligation requires"},
		{"a model problem outside the assessment", work, findingIn{Kind: "model", Blocking: &yes}, "", "kind must be one of"},
		{"a deviation in the assessment", assess, findingIn{Kind: "deviation", Blocking: &yes, Expected: "e"}, "", "kind must be one of"},
	} {
		got, _, err := observationOf(tc.spec, 0, tc.in)
		if tc.err == "" && (err != nil || got != tc.want) || tc.err != "" && (err == nil || !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: %s %v", tc.name, got, err)
		}
	}
}

// Suggestions alone complete a Verify: they are recorded as notes, name no
// correction and start no work Pass, however often they are repeated.
func TestSuggestionsAloneLetTheRunVerify(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "lending" && c.resultTurn() {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"findings": []any{suggestion("lending", "lending/borrow"), suggestion("lending/reminders", "lending/reminders/notice")}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	if res.Stopped || res.Verdict != Satisfied || len(r.Corrections()) != 0 {
		t.Fatalf("verify %+v, corrections %v", res, r.Corrections())
	}
	out := f.finish(r, "")
	if out.Status != Verified || len(out.Findings) != 2 {
		t.Fatalf("status %s: %s; findings %+v", out.Status, out.Reason, out.Findings)
	}
	for _, x := range out.Findings {
		if x.Kind != Suggestion || x.Blocking || x.Correction != "" {
			t.Errorf("finding %+v", x)
		}
	}
}

// A genuine deviation is retained: a suggestion about the same obligation
// neither closes nor softens it, and only a suspicion can be closed as
// confirmed.
func TestAGenuineDeviationIsRetained(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var confirmRefusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.pass() == "p2" && c.turn.Manager == "lending" && c.resultTurn():
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.pass() == "p4" && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on"):
			confirmRefusal = c.basis(map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedConfirmed, "reason": "now a note", "evidence": "x"}}, "findings": []any{suggestion("lending/reminders", "lending/reminders/notice")}}).Text
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{suggestion("lending/reminders", "lending/reminders/notice")}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	if res := r.Pass(verifySpec()); res.Verdict != NeedsCorrection || len(r.Corrections()) != 1 || r.Corrections()[0] != Implement {
		t.Fatalf("first verify %+v, corrections %v", res, r.Corrections())
	}
	r.Pass(implementSpec())
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.pass() == "p4" && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on") && confirmRefusal == "" {
			confirmRefusal = c.basis(map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedConfirmed, "reason": "now a note", "evidence": "x"}}, "findings": []any{suggestion("lending/reminders", "lending/reminders/notice")}}).Text
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{suggestion("lending/reminders", "lending/reminders/notice")}}))
			return nil
		}
		if c.pass() == "p4" && c.turn.Manager == "lending" && c.resultTurn() {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	res := r.Pass(verifySpec())
	if !strings.Contains(confirmRefusal, "only a suspicion is confirmed") {
		t.Fatalf("closing a deviation as confirmed: %q", confirmRefusal)
	}
	if res.Verdict != NeedsCorrection {
		t.Fatalf("a suggestion softened the deviation: %+v", res)
	}
	out := f.finish(r, "")
	if out.Findings[0].Closed || out.Findings[0].Kind != Deviation {
		t.Fatalf("F1 %+v", out.Findings[0])
	}
}

// A suspicion blocks a clean Verify but starts no work: a fresh Verify of the
// unchanged candidate investigates it. Left unresolved, it ends the run
// Incomplete at the Pass budget; confirmed by a deviation, it hands the work a
// correction.
func TestASuspicionIsInvestigatedNotWorkedOn(t *testing.T) {
	for _, confirm := range []bool{false, true} {
		f := newFixture(t)
		f.cfg.Budget.Passes = 4
		var mu sync.Mutex
		f.agents.script = func(c *call) error {
			mu.Lock()
			defer mu.Unlock()
			switch {
			case c.pass() == "p2" && c.turn.Manager == "lending" && c.resultTurn():
				c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []any{suspicion("lending/reminders", "lending/reminders/notice")}}))
				return nil
			case c.pass() != "p2" && c.turn.Kind == VerifyTurn && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on"):
				extra := map[string]any{}
				if confirm {
					extra["closed"] = []any{map[string]any{"id": "F1", "as": ClosedConfirmed, "reason": "the boundary probe shows the late notice", "evidence": "r.go sends it the day after the due date"}}
					extra["findings"] = []any{genuine("lending/reminders", "lending/reminders/notice")}
				}
				c.must(toolReport, c.envelope(kindBasis, extra))
				return nil
			case c.pass() != "p2" && c.turn.Kind == VerifyTurn && c.turn.Manager == "lending" && c.resultTurn():
				c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
				return nil
			}
			c.protocol()
			return nil
		}
		r := f.open()
		r.Pass(implementSpec())
		first := r.Pass(verifySpec())
		if first.Verdict != NeedsCorrection || len(r.Corrections()) != 0 || !r.Reassess() {
			t.Fatalf("confirm %v: first verify %+v, corrections %v", confirm, first, r.Corrections())
		}
		second := r.Pass(verifySpec())
		switch {
		case confirm && (second.Verdict != NeedsCorrection || len(r.Corrections()) != 1 || r.Reassess()):
			t.Fatalf("confirmed: %+v, corrections %v", second, r.Corrections())
		case !confirm && (second.Verdict != NeedsCorrection || len(r.Corrections()) != 0 || !r.Reassess()):
			t.Fatalf("unresolved: %+v, corrections %v", second, r.Corrections())
		}
		if confirm {
			f.finish(r, "")
			continue
		}
		r.Pass(verifySpec())
		if res := r.Pass(verifySpec()); !res.Stopped {
			t.Fatalf("the investigation ran past the Pass budget: %+v", res)
		}
		if out := f.finish(r, ""); out.Status != Incomplete || out.Findings[0].Closed {
			t.Fatalf("status %s: %s", out.Status, out.Reason)
		}
	}
}

// A finding a review of this unchanged candidate settled is not raised again
// by another reviewer or in other words: only a probe receipt of the Pass is
// new evidence.
func TestASettledFindingIsNotReopenedWithoutNewEvidence(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var reopened string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind != VerifyTurn {
			c.protocol()
			return nil
		}
		switch {
		case c.pass() == "p2" && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.pass() == "p2" && c.turn.Manager == "lending/reminders" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"disputed": []any{map[string]any{"id": "F1", "reason": "the notice is sent on the due date", "evidence": "r.go and its test"}}}))
			return nil
		case c.pass() == "p3" && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedDisproved, "reason": "the timing conforms", "evidence": "r.go and its test on this candidate"}}}))
			return nil
		case c.pass() == "p3" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") && reopened == "":
			again := genuine("lending/reminders", "Lending/Reminders/Notice ")
			again["observed"] = "the reminder timing looks wrong to me"
			reopened = c.result(map[string]any{"verdict": string(NeedsCorrection), "findings": []any{again}}).Text
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	if first := r.Pass(verifySpec()); first.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("first verify %+v", first)
	}
	second := r.Pass(verifySpec())
	if !strings.Contains(reopened, "F1 settled this obligation") || !strings.Contains(reopened, "no new evidence") {
		t.Fatalf("a reworded repeat was accepted: %q", reopened)
	}
	if out := f.finish(r, ""); second.Verdict != Satisfied || out.Status != Verified || len(out.Findings) != 1 {
		t.Fatalf("second verify %s, status %s: %s, findings %d", second.Verdict, out.Status, out.Reason, len(out.Findings))
	}
}
