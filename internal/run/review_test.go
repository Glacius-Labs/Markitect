package run

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func genuine(owner, obligation string) map[string]any {
	return map[string]any{"owner": owner, "obligation": obligation, "observed": "observed " + obligation, "blocking": true}
}

// Verify runs fresh read-only reviewers on one immutable candidate: a reviewer
// is Done with a genuine finding, the Pass verdict needs correction and the run
// does not verify. The finding stays open across Passes until a fresh reviewer
// of the raising responsibility closes it on a later candidate; omitting it
// does not close it, and an author cannot close it.
func TestVerifyFindingsStayOpenUntilAnIndependentClosure(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	verifies := 0
	var authorClose, sameCandidateClose string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		review := c.turn.Kind == VerifyTurn
		if review && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on your own contribution") {
			verifies++
			switch verifies {
			case 1:
				c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
				return nil
			case 2:
				// A fresh reviewer that omits the finding does not close it.
				c.must(toolReport, c.envelope(kindBasis, nil))
				return nil
			}
		}
		if review && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "integrate") && verifies == 1 {
			a := c.result(map[string]any{"closed": []any{map[string]string{"id": "F1", "reason": "looks fine now"}}})
			sameCandidateClose = a.Text
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		if review && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "integrate") && verifies == 2 {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		if !review && c.turn.Manager == "lending/reminders" && strings.HasPrefix(c.now(), "work on") && verifies >= 1 {
			// The owner corrects its notice, so the next candidate differs.
			c.write("lending/reminders/r.go", fmt.Sprintf("r corrected after %d reviews", verifies))
		}
		if !review && c.turn.Manager == "lending/reminders" && authorClose == "" && strings.HasPrefix(c.now(), "work on") && verifies >= 1 {
			a := c.basis(map[string]any{"closed": []any{map[string]string{"id": "F1", "reason": "fixed"}}})
			authorClose = a.Text
			if !a.Refused {
				t.Errorf("the author closed a reviewer's finding")
			}
		}
		if review && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") && verifies <= 2 {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	for _, spec := range []PassSpec{implementSpec(), verifySpec()} {
		r.Pass(spec)
	}
	first := r.e.passes[1]
	if first.verdict != NeedsCorrection {
		t.Fatalf("first verify verdict %s", first.verdict)
	}
	if !strings.Contains(sameCandidateClose, "raised in this Pass") {
		t.Errorf("a finding closed on the candidate it was raised on: %q", sameCandidateClose)
	}
	for _, spec := range []PassSpec{implementSpec(), verifySpec(), implementSpec(), verifySpec()} {
		if res := r.Pass(spec); res.Stopped {
			t.Fatalf("stopped in %s", spec.Purpose)
		}
	}
	out := f.finish(r, "")
	if out.Status != Verified {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	f1 := out.Findings[0]
	if f1.ID != "F1" || !f1.Closed || f1.ClosedPass != "p6" || f1.RaisedBy != "lending" || !f1.Review {
		t.Errorf("F1 %+v; want it closed only by lending's reviewer in p6", f1)
	}
	if out.Passes[3].Verdict != NeedsCorrection {
		t.Errorf("the reviewer that omitted the finding left the verdict %s", out.Passes[3].Verdict)
	}
	// Each review Pass has fresh homes and reviewers never resume across Passes.
	homes := map[string]map[string]bool{}
	for _, tr := range f.agents.all() {
		if tr.Kind != VerifyTurn {
			continue
		}
		pass := strings.Split(strings.TrimPrefix(tr.ID, "run1-"), "-")[0]
		if homes[tr.Home] == nil {
			homes[tr.Home] = map[string]bool{}
		}
		homes[tr.Home][pass] = true
		if tr.Resume != "" && !strings.HasPrefix(tr.Resume, "conv-run1-"+pass+"-") {
			t.Errorf("review turn %s resumes %s from another Pass", tr.ID, tr.Resume)
		}
	}
	for home, passes := range homes {
		if len(passes) != 1 {
			t.Errorf("review home %s serves Passes %v", home, passes)
		}
	}
}

// A failed required check on the candidate is a genuine finding: the review
// cannot be clean, and the next candidate's check result decides afresh.
func TestFailedRequiredCheckBlocksTheReview(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", ".")
	f.checks.fail = failingIn("unit", "p2")
	f.agents.script = func(c *call) error {
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") && c.pass() == "p2" {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	for _, spec := range []PassSpec{implementSpec(), verifySpec(), implementSpec(), verifySpec()} {
		r.Pass(spec)
	}
	out := f.finish(r, "")
	if out.Status != Verified || out.Passes[1].Verdict != NeedsCorrection || out.Passes[3].Verdict != Satisfied {
		t.Fatalf("status %s, verdicts %s %s: %s", out.Status, out.Passes[1].Verdict, out.Passes[3].Verdict, out.Reason)
	}
	var check *Finding
	for i := range out.Findings {
		if out.Findings[i].RaisedBy == MarkitectFinding {
			check = &out.Findings[i]
		}
	}
	if check == nil || check.Owner != "." || !check.Closed || check.ClosedPass != "p4" || check.Correction != Implement {
		t.Errorf("check finding %+v", check)
	}
	// Root's correction pass received the finding.
	if !strings.Contains(promptOf(f, ".", "Required check \"unit\""), "genuine") {
		t.Errorf("the check finding never reached Root's correction work")
	}
}

// A Satisfied verdict contradicting an open genuine finding within the
// responsibility is refused; NeedsCorrection without any finding too.
func TestVerdictMustMatchTheOpenFindings(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var satisfied, needs string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("library", "library/interface")}}))
			return nil
		}
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit") {
			satisfied = c.result(nil).Text
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "lending/reminders" && strings.HasPrefix(c.now(), "submit") {
			needs = c.result(map[string]any{"verdict": string(NeedsCorrection)}).Text
		}
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if res.Verdict != NeedsCorrection || out.Status != Incomplete {
		t.Fatalf("verdict %s, status %s", res.Verdict, out.Status)
	}
	if !strings.Contains(satisfied, "stay open") || !strings.Contains(needs, "no genuine finding") {
		t.Errorf("refusals %q / %q", satisfied, needs)
	}
}

// A work Pass's exit gate is strict: the captured stand must be exactly the
// accepted stand and pass the Model check and the File check; a failure ends
// the run Failed, without a repair Pass.
func TestPassExitGateFailureEndsTheRunWithoutRepair(t *testing.T) {
	f := newFixture(t)
	calls := 0
	var mu sync.Mutex
	f.cfg.ModelCheck = func(tree snapshot.Tree) []Finding {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if _, ok := tree.Lookup(mustPath(t, "library/b.go")); ok {
			return []Finding{{Owner: ".", Obligation: "model.invalid", Observed: "the model does not resolve"}}
		}
		return nil
	}
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			c.write("library/b.go", "b")
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	if !res.Stopped {
		t.Fatalf("the pass completed")
	}
	res = r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Status != Failed || !strings.Contains(out.Reason, "Model check or File check failed at the end of p1") || len(out.Passes) != 1 {
		t.Fatalf("status %s, passes %d: %s", out.Status, len(out.Passes), out.Reason)
	}
}

// A change that no accepted submission covers is caught at the Pass exit.
func TestUnacceptedChangeAtPassExitFails(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		c.protocol()
		if c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") {
			// Writes after the turn's submission are taken into its check; a
			// write that bypasses the accepted versions is simulated directly.
			c.ag.ws.write(c.turn.Workspace, "lending/stray.go", "stray")
		}
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Status != Failed || !strings.Contains(out.Reason, "differs from its accepted stand") || !strings.Contains(out.Reason, "lending/stray.go") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}

// With ArtifactFindingsAdvisory every finding states its subject; an artifact
// finding names a Definition of the candidate and never blocks.
func TestAdvisoryArtifactFindingsNeverBlock(t *testing.T) {
	f := newFixture(t)
	f.cfg.ResolveDefinition = func(_ snapshot.Tree, id string) bool { return id == "library/interface" }
	var mu sync.Mutex
	var refusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			bad := c.basis(map[string]any{"findings": []any{genuine("library", "x")}})
			refusal = bad.Text
			art := genuine("library", "library/interface")
			art["subject"], art["definition"] = subjectArtifact, "library/interface"
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{art}}))
			return nil
		}
		c.protocol()
		return nil
	}
	spec := verifySpec()
	spec.ArtifactFindingsAdvisory = true
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(spec)
	out := f.finish(r, "")
	if res.Verdict != Satisfied || out.Status != Verified {
		t.Fatalf("verdict %s, status %s: %s", res.Verdict, out.Status, out.Reason)
	}
	if !strings.Contains(refusal, "subject must be") {
		t.Errorf("refusal %q", refusal)
	}
	if len(out.Findings) != 1 || out.Findings[0].Blocking || out.Findings[0].Subject != subjectArtifact || out.Findings[0].Definition != "library/interface" {
		t.Errorf("findings %+v", out.Findings)
	}
}

// A suite total cannot replace evidence for a particular Definition. Both
// basis and result claims are checked, and the parent receives the exact
// observation it is asked to assess. This tests protocol admission, not the
// semantic truth of the scripted observations.
func TestReviewRequiresPerDefinitionObservations(t *testing.T) {
	for _, status := range []string{coverDone, coverPreserved} {
		t.Run(status, func(t *testing.T) {
			f := newFixture(t)
			var mu sync.Mutex
			refusals, parentSaw := 0, false
			f.agents.script = func(c *call) error {
				mu.Lock()
				defer mu.Unlock()
				if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && (strings.HasPrefix(c.now(), "work on") || strings.HasPrefix(c.now(), "submit")) {
					kind := kindBasis
					if strings.HasPrefix(c.now(), "submit") {
						kind = kindResult
					}
					cov := c.coverage()
					cov[0]["status"] = status
					delete(cov[0], "evidence")
					args := c.envelope(kind, map[string]any{"obligations": cov, "evidence": []string{"51 tests passed"}})
					if a := c.tool(toolReport, args); !a.Refused || !strings.Contains(a.Text, "requires evidence") {
						t.Errorf("unsubstantiated claim admitted: %+v", a)
					}
					refusals++
					for _, field := range []string{"claim", "method", "expected", "observed"} {
						ev := map[string]string{"claim": "permitted unusual input reaches domain", "method": "public command with assertions", "expected": "accepted", "observed": "accepted"}
						ev[field] = " "
						cov[0]["evidence"] = []any{ev}
						if a := c.tool(toolReport, args); !a.Refused || !strings.Contains(a.Text, field) {
							t.Errorf("blank %s admitted: %+v", field, a)
						}
						refusals++
					}
					cov[0]["evidence"] = []observation{{Claim: "permitted unusual input reaches domain", Method: "public command with assertions", Expected: "accepted", Observed: "accepted"}}
					c.must(toolReport, args)
					return nil
				}
				if c.turn.Kind == VerifyTurn && c.turn.Manager == "." && proposalOf(c, kindBasis, "library") != "" {
					parentSaw = strings.Contains(c.turn.Prompt, "Claim: permitted unusual input reaches domain") && strings.Contains(c.turn.Prompt, "Method: public command with assertions") && strings.Contains(c.turn.Prompt, "Expected: accepted") && strings.Contains(c.turn.Prompt, "Observed: accepted")
				}
				c.protocol()
				return nil
			}
			r := f.open()
			r.Pass(implementSpec())
			res := r.Pass(verifySpec())
			out := f.finish(r, "")
			if res.Stopped || out.Status != Verified || refusals != 10 || !parentSaw {
				t.Fatalf("status %s, refused %d, parent saw evidence %v: %s", out.Status, refusals, parentSaw, out.Reason)
			}
		})
	}
}
