package run

import (
	"strings"
	"sync"
	"testing"
)

// planWith is a Plan Pass script in which lending's plan names a dependency
// for the Implement Pass.
func dependsOn(c *call, manager, owner, stage string) bool {
	if !c.planning() || c.turn.Manager != manager {
		return false
	}
	pl := samplePlan()
	pl["dependencies"] = []any{map[string]any{"owner": owner, "need": "its accepted interface", "pass": "implement", "stage": stage}}
	switch {
	case strings.HasPrefix(c.now(), "work on"):
		c.must(toolReport, c.envelope(kindBasis, map[string]any{"plan": pl}))
	case strings.HasPrefix(c.now(), "submit"), strings.HasPrefix(c.now(), "integrate"):
		c.must(toolReport, c.envelope(kindResult, map[string]any{"plan": pl}))
	default:
		return false
	}
	return true
}

// After its parent's basis released it, lending waits for library's accepted
// interface but answers library's routed question in a communication-only turn:
// read-only, without submitting or releasing its child. The dependency stays,
// lending waits again, and it starts its productive work only after library's
// basis is accepted.
func TestBlockedManagerAnswersReadOnlyAndKeepsItsDependency(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var blockedTurn *Turn
	var blockedSubmit, blockedProgress string
	asked := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if dependsOn(c, "lending", "library", "basis") {
			return nil
		}
		if c.planning() {
			c.protocol()
			return nil
		}
		switch {
		case c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") && !asked:
			c.must(toolSend, map[string]any{"to": "parent", "kind": sendQuestion, "text": "which loan fields does lending expose?"})
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q1"})
			asked = true
			return nil
		case c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "Question Q1 from your child library"):
			c.must(toolSend, map[string]any{"to": "lending", "kind": sendQuestion, "text": "library asks which loan fields you expose", "about": "Q1"})
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q2"})
			for _, id := range c.pendingDecisions() {
				c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": id})
			}
			return nil
		case c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "Answer to your question Q2"):
			c.must(toolSend, map[string]any{"to": "library", "kind": sendAnswer, "text": "copy, member and due date", "about": "Q1"})
		case c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "your own work waits for"):
			turn := c.turn
			blockedTurn = &turn
			blockedSubmit = c.basis(nil).Text
			blockedProgress = c.tool(toolReport, map[string]any{"kind": kindProgress, "summary": "x"}).Text
			c.must(toolSend, map[string]any{"to": "parent", "kind": sendAnswer, "text": "copy, member and due date", "about": "Q2"})
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(planSpec())
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v: %s", res.Stopped, out.Reason)
	}
	if blockedTurn == nil {
		t.Fatalf("lending never got a communication-only turn")
	}
	if !blockedTurn.ReadOnly || len(blockedTurn.Mounts) != 0 {
		t.Errorf("the communication-only turn could write: %+v", blockedTurn.Mounts)
	}
	if !strings.Contains(blockedSubmit, "waits for") || !strings.Contains(blockedProgress, "waits for") {
		t.Errorf("productive reports were admitted while blocked: %q / %q", blockedSubmit, blockedProgress)
	}
	// After the answer lending waits again for its dependency; its productive
	// release and its child's release follow library's accepted basis.
	libraryBasis := -1
	waitedAgain, unblocked, childReleased := -1, -1, -1
	answered := -1
	for i, ev := range f.ev.events {
		if ev.Pass != "p2" {
			continue
		}
		data, _ := ev.Data.(map[string]any)
		switch {
		case ev.Kind == "question.answered" && ev.Manager == "lending":
			answered = i
		case ev.Kind == "manager.state_changed" && ev.Manager == "lending" && answered >= 0 && waitedAgain < 0 && data["to"] == Waiting && strings.Contains(data["waitingFor"].(string), "library"):
			waitedAgain = i
		case ev.Kind == "basis.accepted" && ev.Manager == "library" && libraryBasis < 0:
			libraryBasis = i
		case ev.Kind == "manager.unblocked" && ev.Manager == "lending":
			unblocked = i
		case ev.Kind == "manager.release" && ev.Manager == "lending/reminders" && childReleased < 0:
			childReleased = i
		}
	}
	if answered < 0 || waitedAgain < answered || libraryBasis < 0 || unblocked < libraryBasis || childReleased < unblocked {
		t.Errorf("answered %d, waited again %d, library basis %d, unblocked %d, child released %d", answered, waitedAgain, libraryBasis, unblocked, childReleased)
	}
	if f.agents.maxActive < 1 {
		t.Errorf("no turn ran")
	}
}

// A question to a Manager that its parent's basis has not released yet is
// refused: it would wait on nobody.
func TestQuestionToAnUnreleasedManagerIsRefused(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var refusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "." && refusal == "" {
			refusal = c.send("lending", sendQuestion, "what do you need?", "").Text
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(planSpec())
	f.finish(r, "")
	if !strings.Contains(refusal, "has not started in this Pass") {
		t.Errorf("refusal %q", refusal)
	}
}

// An erroneous finding is independently disputed with evidence; only that
// dispute keeps the review from being clean, so a new complete Verify Pass with
// a new identity assesses the same unchanged candidate, where a fresh reviewer
// of the raising responsibility disproves it with evidence. The sealed earlier
// Verify stays as it was; no artifact changes and no work Pass runs.
func TestFalseFindingIsDisprovedOnTheUnchangedCandidate(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", ".")
	var mu sync.Mutex
	var resolvedRefusal, authorDispute, checkDispute string
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
			// The finding's owner is not an ancestor of its raiser: no dispute.
			authorDispute = c.basis(map[string]any{"disputed": []any{map[string]any{"id": "F1", "reason": "fine", "evidence": "x"}}}).Text
		case c.pass() == "p2" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate"):
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection), "disputed": []any{map[string]any{"id": "F1", "reason": "the notice is sent on the due date, as lending/reminders/notice states", "evidence": "lending/reminders/r.go sends it at the due date; its test covers it"}}}))
			return nil
		case c.pass() == "p3" && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on"):
			resolvedRefusal = c.basis(map[string]any{"closed": []any{map[string]any{"id": "F1", "reason": "fixed"}}}).Text
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"closed": []any{map[string]any{"id": "F1", "as": ClosedDisproved, "reason": "the notice timing conforms; the finding misread the model", "evidence": "lending/reminders/r.go and its test on this candidate"}}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	first := r.Pass(verifySpec())
	if first.Verdict != NeedsCorrection {
		t.Fatalf("first Verify %s", first.Verdict)
	}
	if !r.Reassess() {
		t.Fatalf("a disputed-only review must be reassessed on the same candidate")
	}
	second := r.Pass(verifySpec())
	out := f.finish(r, "")
	if second.Stopped || second.Verdict != Satisfied || out.Status != Verified {
		t.Fatalf("second Verify %s, status %s: %s", second.Verdict, out.Status, out.Reason)
	}
	if out.Passes[1].ID == out.Passes[2].ID || out.Passes[1].Tree != out.Passes[2].Tree || out.Passes[1].Verdict != NeedsCorrection || !out.Passes[1].Sealed {
		t.Errorf("passes %+v / %+v", out.Passes[1], out.Passes[2])
	}
	f1 := out.Findings[0]
	if !f1.Closed || f1.ClosedAs != ClosedDisproved || f1.ClosedEvidence == "" || f1.ClosedPass != "p3" || len(f1.Disputes) != 1 || f1.Disputes[0].By != "." || f1.Candidate == "" {
		t.Errorf("F1 %+v", f1)
	}
	if !strings.Contains(resolvedRefusal, "unchanged candidate") || !strings.Contains(authorDispute, "not raised by a reviewer below you") {
		t.Errorf("refusals %q / %q", resolvedRefusal, authorDispute)
	}
	if len(out.Candidate.Entries()) == 0 || !out.Candidate.Equal(f.cfg.Start) {
		t.Errorf("an artifact changed")
	}
	_ = checkDispute
}

// A failed required check cannot be disputed away, and a disputed finding the
// fresh reviewer keeps leaves no second reassessment on the same candidate.
func TestDisputesCannotWaiveChecksOrRepeatOnACandidate(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", ".")
	f.checks.fail = failingIn("unit", "p2")
	var mu sync.Mutex
	var checkDispute string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.pass() == "p2" && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") {
			checkDispute = c.result(map[string]any{"verdict": string(NeedsCorrection), "disputed": []any{map[string]any{"id": "F1", "reason": "flaky", "evidence": "it passed locally"}}}).Text
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	r.Pass(verifySpec())
	if r.Reassess() {
		t.Errorf("a failed required check was reassessed instead of corrected")
	}
	f.finish(r, "")
	if !strings.Contains(checkDispute, "failed required check") {
		t.Errorf("refusal %q", checkDispute)
	}
}
