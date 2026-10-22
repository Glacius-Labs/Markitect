package run

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

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
	work, review, assess := implementSpec(), verifySpec(), assessSpec()
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
		{"suspicion", review, findingIn{Kind: "suspected", Blocking: &yes, Expected: "e"}, Suspected, ""},
		{"a suspicion in a work Pass", work, findingIn{Kind: "suspected", Blocking: &yes, Expected: "e"}, "", "kind must be one of"},
		{"a blocking suggestion", work, findingIn{Kind: "suggestion", Blocking: &yes}, "", "a suggestion never blocks"},
		{"a non-blocking deviation", work, findingIn{Kind: "deviation", Blocking: &no, Expected: "e"}, "", "a deviation blocks"},
		{"a deviation without its expected outcome", work, findingIn{Blocking: &yes}, "", "states the outcome its obligation requires"},
		{"a suspicion without its expected outcome", review, findingIn{Kind: "suspected", Blocking: &yes}, "", "states the outcome its obligation requires"},
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

// A finding a review of this unchanged candidate disproved is not raised
// again by another reviewer or in other words.
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
			if len(f.ev.kinds("finding.recorded")) == 0 {
				c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "basis"})
				return nil
			}
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
	if !strings.Contains(reopened, "F1 disproved this obligation") || !strings.Contains(reopened, "no new evidence") {
		t.Fatalf("a reworded repeat was accepted: %q", reopened)
	}
	if out := f.finish(r, ""); second.Verdict != Satisfied || out.Status != Verified || len(out.Findings) != 1 {
		t.Fatalf("second verify %s, status %s: %s, findings %d", second.Verdict, out.Status, out.Reason, len(out.Findings))
	}
}

// Next to a deviation of the same owner, a suspicion still is no work: the
// correction Pass the deviation starts hands its owner the deviation only.
func TestASuspicionNextToADeviationIsNotWorkedOn(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.pass() == "p2" && c.turn.Manager == "lending" && c.resultTurn() {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []any{genuine("lending/reminders", "lending/reminders/notice"), suspicion("lending/reminders", "lending/reminders/boundary")}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	if v := r.Pass(verifySpec()); v.Verdict != NeedsCorrection || !slices.Equal(r.Corrections(), []Purpose{Implement}) {
		t.Fatalf("verify %+v, corrections %v", v, r.Corrections())
	}
	r.Pass(implementSpec())
	seen := false
	for _, turn := range f.agents.turns {
		if strings.Contains(turn.ID, "-p3-") && turn.Manager == "lending/reminders" && strings.Contains(turn.Prompt, "Finding F1") {
			seen = true
			if strings.Contains(turn.Prompt, "Finding F2") {
				t.Fatalf("the correction Pass hands the suspicion to its owner:\n%s", turn.Prompt)
			}
		}
	}
	if !seen {
		t.Fatal("the correction Pass does not hand the deviation to its owner")
	}
	f.finish(r, "")
}

// Only a disproof settles a finding: after its raiser closed it as resolved
// on a changed candidate, another reviewer may raise the same obligation
// there.
func TestAResolvedFindingCanBeRaisedAgain(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var again ToolAnswer
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.pass() == "p2" && c.turn.Manager == "lending" && c.resultTurn():
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.pass() == "p3" && c.turn.Manager == "lending/reminders" && strings.HasPrefix(c.now(), "work on"):
			c.write("lending/reminders/r.go", "r2")
		case c.pass() == "p4" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") && again.Text == "":
			if again = c.result(map[string]any{"verdict": string(NeedsCorrection), "findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}); !again.Refused {
				return nil
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	r.Pass(verifySpec())
	r.Pass(implementSpec())
	r.Pass(verifySpec())
	out := f.finish(r, "")
	if again.Refused || len(out.Findings) != 2 || out.Findings[0].ClosedAs != ClosedResolved {
		t.Fatalf("raise again %q; findings %+v", again.Text, out.Findings)
	}
}

// A suspicion is confirmed only by a deviation of its own obligation.
func TestAConfirmationNamesTheSameObligation(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var refusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.pass() == "p2" && c.turn.Manager == "lending" && c.resultTurn():
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []any{suspicion("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.pass() == "p3" && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on") && refusal == "":
			refusal = c.basis(map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedConfirmed, "reason": "another defect", "evidence": "r.go"}}, "findings": []any{genuine("lending/reminders", "lending/reminders/other")}}).Text
			// The suspicion stays open for the next review.
			c.must(toolReport, c.envelope(kindBasis, nil))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	if v := r.Pass(verifySpec()); v.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("verify %+v", v)
	}
	r.Pass(verifySpec())
	f.finish(r, "")
	if !strings.Contains(refusal, "of the same owner and obligation") {
		t.Fatalf("a deviation of another obligation confirmed the suspicion: %q", refusal)
	}
}

// An Extract artifact finding keeps its kind but stays advisory: it blocks
// nothing, names no correction and is never refused as a repeat.
func TestAnAdvisoryArtifactFindingNamesNoCorrection(t *testing.T) {
	f := newFixture(t)
	f.cfg.ResolveDefinition = func(snapshot.Tree, string) bool { return true }
	spec := verifySpec()
	spec.ArtifactFindingsAdvisory = true
	f.agents.script = func(c *call) error {
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "lending" && c.resultTurn() {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"findings": []any{map[string]any{"subject": "artifact", "definition": "lending/borrow", "owner": "lending", "obligation": "lending/borrow", "expected": "five loans at most", "observed": "borrow admits a sixth loan", "blocking": true}}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	if v := r.Pass(spec); v.Verdict != Satisfied || len(r.Corrections()) != 0 {
		t.Fatalf("verify %+v, corrections %v", v, r.Corrections())
	}
	out := f.finish(r, "")
	if len(out.Findings) != 1 || out.Findings[0].Blocking || out.Findings[0].Correction != "" || out.Findings[0].Kind != Deviation || out.Findings[0].Subject != subjectArtifact {
		t.Fatalf("findings %+v", out.Findings)
	}
}

// After a disproof on the unchanged candidate, only a failed probe receipt of
// the Pass is new evidence for raising the finding again; a passing receipt is
// none.
func TestOnlyAFailedReceiptReopensADisprovedFinding(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	var bare, passing, failing ToolAnswer
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind != VerifyTurn {
			c.protocol()
			return nil
		}
		switch {
		case c.pass() == "p2" && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("library", "library/interface")}}))
			return nil
		case c.pass() == "p2" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate"):
			for _, id := range c.pendingDecisions() {
				c.accept(id)
			}
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "disputed": []any{map[string]any{"id": "F1", "reason": "the interface conforms", "evidence": "library/a.go"}}}))
			return nil
		case c.pass() == "p3" && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedDisproved, "reason": "the interface conforms", "evidence": "library/a.go on this candidate"}}}))
			return nil
		case c.pass() == "p3" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") && failing.Text == "":
			for _, id := range c.pendingDecisions() {
				c.accept(id)
			}
			bare = c.result(map[string]any{"verdict": string(NeedsCorrection), "findings": []any{genuine("library", "library/interface")}})
			ok := receiptOf(t, c.must(toolCheck, probe("library/interface", "a plain name", "probe_plain.py", "PASS")))
			withPass := genuine("library", "library/interface")
			withPass["receipts"] = []string{ok}
			passing = c.result(map[string]any{"verdict": string(NeedsCorrection), "findings": []any{withPass}})
			bad := receiptOf(t, c.must(toolCheck, probe("library/interface", "a hyphen name", "probe_names.py", "FAIL")))
			withFail := genuine("library", "library/interface")
			withFail["receipts"] = []string{bad}
			if failing = c.result(map[string]any{"verdict": string(NeedsCorrection), "findings": []any{withFail}}); !failing.Refused {
				return nil
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	if first := r.Pass(verifySpec()); first.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("first verify %+v", first)
	}
	r.Pass(verifySpec())
	f.finish(r, "")
	if !bare.Refused || !passing.Refused || !strings.Contains(passing.Text, "failed probe receipt") {
		t.Fatalf("a repeat without new evidence was accepted: %q / %q", bare.Text, passing.Text)
	}
	if failing.Refused {
		t.Fatalf("a repeat with a failed receipt of the Pass was refused: %q", failing.Text)
	}
}
