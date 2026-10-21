package run

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func assessSpec() PassSpec {
	return PassSpec{Purpose: Assess, Mandate: "Assess the bound model.", Review: true, Assesses: true, Guidance: "A genuine finding is a material model problem.", Corrections: []Purpose{"model_error", "model_gap", "unclassified"}}
}

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

// libraryGap is the material gap the fixture's library reviewer reports.
var libraryGap = map[string]any{"owner": "library", "obligation": "library/interface", "observed": "the model leaves the maximum name length open", "blocking": true, "correction": "model_gap"}

// raiseGap scripts the library reviewer to report libraryGap with its first
// result and returns whether it did.
func raiseGap(c *call, raised *bool) bool {
	if c.pass() != "p1" || c.turn.Manager != "library" || !strings.HasPrefix(c.now(), "submit your complete result") || *raised {
		return false
	}
	*raised = true
	c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []map[string]any{libraryGap}}))
	return true
}

// An accepted Model assessment releases the work. Its reviewers are fresh and
// read-only, no required check runs and no probe is offered, its clean seal
// verifies nothing, and Root is told that a missing decision is a finding of
// the assessment, not a question beyond the tree.
func TestAnAcceptedAssessmentReleasesTheWork(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", ".")
	var mu sync.Mutex
	var refusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.pass() == "p1" && c.turn.Manager == "." && refusal == "" {
			refusal = c.send("user", sendQuestion, "Which name length?", "").Text
		}
		c.protocol()
		return nil
	}
	r := f.open()
	if res := r.Pass(assessSpec()); res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("assessment %+v", res)
	}
	if n := len(f.checks.reqs); n != 0 {
		t.Fatalf("%d checks ran in the assessment", n)
	}
	if !strings.Contains(refusal, "genuine finding of this assessment") || strings.Contains(refusal, "owner decisions") {
		t.Fatalf("refusal %q", refusal)
	}
	for _, turn := range f.agents.turns {
		if fresh := strings.HasSuffix(turn.ID, "-r1"); turn.Kind != VerifyTurn || !turn.ReadOnly || fresh && (turn.Resume != "" || !strings.Contains(turn.Prompt, "You assess the bound model")) {
			t.Fatalf("assessment turn %+v", turn)
		}
		for _, tool := range turn.Tools {
			if tool.Name == toolCheck {
				t.Fatalf("turn %s offers %s", turn.ID, toolCheck)
			}
		}
	}
	for _, spec := range []PassSpec{implementSpec(), verifySpec()} {
		if res := r.Pass(spec); res.Stopped || res.Verdict != Satisfied {
			t.Fatalf("%s %+v", spec.Purpose, res)
		}
	}
	out := f.finish(r, "")
	if out.Status != Verified || len(out.Passes) != 3 || out.Passes[0].Purpose != Assess || len(out.Passes[0].Checks) != 0 || len(out.Passes[2].Checks) != 1 {
		t.Fatalf("status %s: %s; passes %+v", out.Status, out.Reason, out.Passes)
	}
}

// A clean assessment alone is no verified candidate.
func TestAnAssessmentAloneVerifiesNothing(t *testing.T) {
	f := newFixture(t)
	r := f.open()
	if res := r.Pass(assessSpec()); res.Verdict != Satisfied {
		t.Fatalf("assessment %+v", res)
	}
	if out := f.finish(r, ""); out.Status == Verified {
		t.Fatalf("status %s", out.Status)
	}
}

// A finding Root confirms (it stays open and undisputed in the sealed Pass)
// ends the run Failed with its class when the assessment seals; nothing is
// reassessed and no work Pass starts.
func TestAConfirmedModelProblemEndsTheRunBeforeAnyWork(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	raised := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if !raiseGap(c, &raised) {
			c.protocol()
		}
		return nil
	}
	r := f.open()
	if res := r.Pass(assessSpec()); res.Stopped || res.Verdict != NeedsCorrection {
		t.Fatalf("assessment %+v", res)
	}
	if r.Reassess() {
		t.Fatal("a confirmed model problem is reassessed")
	}
	if next := r.Pass(implementSpec()); !next.Stopped {
		t.Fatalf("work started after a confirmed model problem: %+v", next)
	}
	out := f.finish(r, "")
	if out.Status != Failed || !strings.HasPrefix(out.Reason, ModelAssessmentReason) || !strings.Contains(out.Reason, "[library, model_gap]") || !strings.Contains(out.Reason, "maximum name length") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	if len(out.Passes) != 1 || len(out.Findings) != 1 || out.Findings[0].Closed || out.Findings[0].Correction != "model_gap" {
		t.Fatalf("passes %+v, findings %+v", out.Passes, out.Findings)
	}
}

// disputeGap scripts Root to dispute the library reviewer's gap when it
// integrates the first assessment.
func disputeGap(c *call) bool {
	if c.pass() != "p1" || c.turn.Manager != "." || !strings.HasPrefix(c.now(), "integrate your children") {
		return false
	}
	for _, id := range c.pendingDecisions() {
		c.accept(id)
	}
	c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "disputed": []any{map[string]any{"id": "F1", "reason": "library/interface fixes the limit by its wording", "evidence": "library/interface, its first statement"}}}))
	return true
}

// A finding Root disputes is not confirmed: a fresh assessment of the unchanged
// stand decides it, and once it is disproved the work proceeds.
func TestADisputedFindingIsReassessedBeforeTheWork(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	raised := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case raiseGap(c, &raised), disputeGap(c):
		case c.pass() == "p2" && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedDisproved, "reason": "the wording of library/interface fixes the limit", "evidence": "library/interface, its first statement"}}}))
		default:
			c.protocol()
		}
		return nil
	}
	r := f.open()
	first := r.Pass(assessSpec())
	if first.Stopped || first.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("first assessment %+v", first)
	}
	if second := r.Pass(assessSpec()); second.Stopped || second.Verdict != Satisfied {
		t.Fatalf("second assessment %+v", second)
	}
	if res := r.Pass(implementSpec()); res.Stopped {
		t.Fatalf("work %+v", res)
	}
	out := f.finish(r, "")
	if len(out.Findings) != 1 || !out.Findings[0].Closed || out.Findings[0].ClosedAs != ClosedDisproved || len(out.Findings[0].Disputes) != 1 {
		t.Fatalf("findings %+v", out.Findings)
	}
}

// A disputed finding that no fresh assessment can decide within the limits
// leaves the run Incomplete, never Failed as a model problem.
func TestAnUndecidedAssessmentIsIncomplete(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.Passes = 1
	var mu sync.Mutex
	raised := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if !raiseGap(c, &raised) && !disputeGap(c) {
			c.protocol()
		}
		return nil
	}
	r := f.open()
	if res := r.Pass(assessSpec()); res.Stopped || res.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("assessment %+v", res)
	}
	if res := r.Pass(assessSpec()); !res.Stopped {
		t.Fatalf("a Pass beyond the budget ran: %+v", res)
	}
	if out := f.finish(r, ""); out.Status != Incomplete || strings.HasPrefix(out.Reason, ModelAssessmentReason) {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}

// A Model assessment is a read-only review Pass of the model.
func TestAModelAssessmentIsAReviewOfTheModel(t *testing.T) {
	for _, spec := range []PassSpec{
		{Purpose: Assess, Mandate: "m", Assesses: true, Corrections: []Purpose{"model_gap"}},
		{Purpose: Assess, Mandate: "m", Assesses: true, Writes: true, Corrections: []Purpose{"model_gap"}},
		{Purpose: Assess, Mandate: "m", Assesses: true, Review: true, Guidance: "g", ArtifactFindingsAdvisory: true, Corrections: []Purpose{"model_gap"}},
	} {
		if validateSpec(spec) == nil {
			t.Errorf("accepted %+v", spec)
		}
	}
	if err := validateSpec(assessSpec()); err != nil {
		t.Fatal(err)
	}
}

// A turn of the assessment that fails technically ends the run with its own
// classification, never as a confirmed model problem.
func TestATechnicalFailureInTheAssessmentIsNoModelProblem(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.pass() == "p1" && c.turn.Manager == "library" {
			return errors.New("the provider connection broke")
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(assessSpec())
	for !res.Stopped && res.Verdict == NeedsCorrection && r.Reassess() {
		res = r.Pass(assessSpec())
	}
	out := f.finish(r, "the model assessment did not reach a clean Pass")
	if out.Status != Incomplete || !strings.Contains(out.Reason, "the provider connection broke") || strings.HasPrefix(out.Reason, ModelAssessmentReason) {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}

// A dispute counts only in the Pass it was raised in: a finding disputed in
// one assessment and upheld, undisputed, in the next is confirmed.
func TestAFindingUpheldInTheNextAssessmentIsConfirmed(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	raised := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case raiseGap(c, &raised), disputeGap(c):
		case c.pass() == "p2" && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			// The fresh reviewer keeps its finding: it neither closes nor softens it.
			c.must(toolReport, c.envelope(kindBasis, nil))
		case c.pass() == "p2" && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit your complete result"):
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
		default:
			c.protocol()
		}
		return nil
	}
	r := f.open()
	if first := r.Pass(assessSpec()); first.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("first assessment %+v", first)
	}
	if second := r.Pass(assessSpec()); second.Verdict != NeedsCorrection || r.Reassess() {
		t.Fatalf("second assessment %+v", second)
	}
	if out := f.finish(r, ""); out.Status != Failed || !strings.HasPrefix(out.Reason, ModelAssessmentReason) || !strings.Contains(out.Reason, "F1 [library, model_gap]") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}

// A candidate verified just before an assessment stays verified only through
// clean assessments of the same stand, also after a reassessment.
func TestAVerifiedCandidateStaysVerifiedThroughItsAssessment(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	raised := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.pass() == "p3" && c.turn.Manager == "library" && c.resultTurn() && !raised:
			raised = true
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []map[string]any{libraryGap}}))
		case c.pass() == "p3" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate your children"):
			for _, id := range c.pendingDecisions() {
				c.accept(id)
			}
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "disputed": []any{map[string]any{"id": "F1", "reason": "the wording fixes the limit", "evidence": "library/interface"}}}))
		case c.pass() == "p4" && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedDisproved, "reason": "the wording fixes the limit", "evidence": "library/interface"}}}))
		default:
			c.protocol()
		}
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	if v := r.Pass(verifySpec()); v.Verdict != Satisfied {
		t.Fatalf("verify %+v", v)
	}
	if a := r.Pass(assessSpec()); a.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("first assessment %+v", a)
	}
	if a := r.Pass(assessSpec()); a.Verdict != Satisfied {
		t.Fatalf("second assessment %+v", a)
	}
	if out := f.finish(r, ""); out.Status != Verified {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}

// The Failed reason names each confirmed finding with a shortened text and a
// neutral cut mark; the complete finding stays in the record.
func TestTheAssessmentReasonShortensLongFindings(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	raised := false
	long := strings.Repeat("the model leaves the maximum name length open; ", 30)
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.pass() == "p1" && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit your complete result") && !raised {
			raised = true
			gap := map[string]any{"owner": "library", "obligation": "library/interface", "observed": long, "blocking": true, "correction": "model_gap"}
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "findings": []any{gap}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(assessSpec())
	out := f.finish(r, "")
	if out.Status != Failed || !strings.Contains(out.Reason, " [...]") || strings.Contains(out.Reason, "probe") || len(out.Reason) > 1000 || out.Findings[0].Observed != strings.TrimSpace(long) {
		t.Fatalf("reason %q", out.Reason)
	}
}
