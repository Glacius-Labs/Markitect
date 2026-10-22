package run

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// W's native turns changed code/tests and ended normally without ANY MCP call.
// The native final assessment now suffices for Host submission, never Verify.
func TestNativeCompletedChildSubmitsWorkWithoutAnyMCPCall(t *testing.T) {
	f := probeFixture(t)
	f.cfg.Start = treeOf(t, map[string]string{"go.mod": "module x\n", "library/a.go": "broken boundary", "library/test.go": "weak assertions", "lending/l.go": "l", "lending/reminders/r.go": "r"})
	var nativeTurn string
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" && c.turn.Kind == ExecuteTurn {
			nativeTurn = c.turn.ID
			c.write("library/a.go", "correct inclusive boundary")
			c.write("library/test.go", "assert the inclusive boundary and invalid input")
			bad, good := 2, 0
			c.commands = []CommandObservation{{ID: "cmd-setup", Command: "python missing.py", Status: "completed", ExitCode: &bad}, {ID: "cmd-tests", Command: "python -m unittest", Status: "completed", ExitCode: &good, Output: "two discriminating assertions passed"}}
			c.final = `{"status":"satisfied","summary":"Corrected the boundary and its weak assertions; ready for independent review.","findings":[],"open":[]}`
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	work := r.Pass(implementSpec())
	if work.Stopped || work.Verdict != Satisfied || r.e.verified != nil {
		t.Fatalf("work skipped normal review: %+v", work)
	}
	review := r.Pass(verifySpec())
	out := f.finish(r, "")
	if review.Stopped || out.Status != Verified || len(f.agents.turnsOf("library")) != 3 {
		t.Fatalf("workflow %+v %s: %s", review, out.Status, out.Reason)
	}
	if f.agents.toolCalls[nativeTurn] != 0 {
		t.Fatalf("native child used %d MCP calls", f.agents.toolCalls[nativeTurn])
	}
	counts := map[string]int{}
	var captured bool
	for _, ev := range f.ev.kinds("turn.submitted") {
		counts[ev.Turn]++
		if ev.Turn == nativeTurn {
			data := ev.Data.(map[string]any)
			paths := data["changedPaths"].([]string)
			captured = data["completed"] == true && data["candidate"] != "" && len(paths) == 2 && len(data["commands"].([]CommandObservation)) == 2
		}
	}
	for _, turn := range out.Turns {
		if counts[turn.ID] != 1 {
			t.Fatalf("%s has %d submissions", turn.ID, counts[turn.ID])
		}
	}
	if !captured || len(out.Probes) != 0 {
		t.Fatalf("captured=%v fabricated probe receipts=%d", captured, len(out.Probes))
	}
	checks := out.Passes[1].Checks
	if len(checks) != 1 || checks[0].ID != "run1-p2-tests" || !checks[0].Passed {
		t.Fatalf("required checks not bound: %+v", checks)
	}
	for _, turn := range f.agents.all() {
		for _, tool := range turn.Tools {
			if tool.Name == "report" {
				t.Fatal("mandatory report tool survived")
			}
		}
		if turn.Kind == VerifyTurn && (!turn.ReadOnly || turn.Resume != "" && turn.Manager == "library" && strings.HasSuffix(turn.ID, "r1")) {
			t.Fatal("review independence changed")
		}
		if strings.Contains(turn.Prompt, "ended without a report") {
			t.Fatal("missing-report reminder loop survived")
		}
	}
}

func TestNativeEndCannotInventSemanticOrTechnicalSuccess(t *testing.T) {
	for _, tc := range []struct {
		name, final              string
		nativeIncomplete, noStop bool
		failure                  error
	}{
		{name: "announcement", final: "Submitted for Verify; all tests green."},
		{name: "empty"},
		{name: "open required point", final: `{"status":"satisfied","summary":"done","findings":[],"open":["required boundary remains unknown"]}`},
		{name: "blocked", final: `{"status":"blocked","summary":"prerequisite unresolved","findings":[],"open":["unknown"]}`},
		{name: "no native end", nativeIncomplete: true, final: `{"status":"satisfied","summary":"done","findings":[],"open":[]}`},
		{name: "native error", failure: errTest, final: `{"status":"satisfied","summary":"done","findings":[],"open":[]}`},
		{name: "timeout", failure: context.DeadlineExceeded, final: `{"status":"satisfied","summary":"done","findings":[],"open":[]}`},
		{name: "unknown stop", noStop: true, final: `{"status":"satisfied","summary":"done","findings":[],"open":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.agents.script = func(c *call) error {
				if c.turn.Manager == "library" {
					c.final = tc.final
					c.nativeIncomplete = tc.nativeIncomplete
					c.noStop = tc.noStop
					return tc.failure
				}
				c.protocol()
				return nil
			}
			r := f.open()
			result := r.Pass(implementSpec())
			out := f.finish(r, "")
			if !result.Stopped || out.Status != Incomplete {
				t.Fatalf("%s promoted to success: %+v %s", tc.name, result, out.Status)
			}
			for _, m := range out.Passes[0].Managers {
				if m.Manager == "library" && m.State == Done {
					t.Fatal("unknown child became Done")
				}
			}
		})
	}
}

func TestClaimedTestAndReceiptIDsDoNotCreateExecutionEvidence(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" && c.turn.Kind == ExecuteTurn {
			c.final = `{"status":"satisfied","summary":"Inspected the static fixture; claimed tests passed","findings":[],"open":[],"support":[{"receipt":"R999"}],"tests":["claimed-check"]}`
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	result := r.Pass(implementSpec())
	out := f.finish(r, "implementation only")
	if result.Stopped || out.Status == Verified || len(out.Probes) != 0 {
		t.Fatalf("claim manufactured verification: %+v", out)
	}
	for _, ev := range f.ev.kinds("proposal.submitted") {
		if ev.Manager == "library" {
			env := ev.Data.(map[string]any)["envelope"].(envelope)
			if len(env.Support) != 0 || len(env.Commands) != 0 || len(env.Checks) != 0 {
				t.Fatalf("claim created Host observations: %+v", env)
			}
		}
	}
}

// The forward sequence Plan, Prepare, Implement and Verify runs through one
// base process: every Manager, Root and the leaves included, submits a basis
// and a complete result, and the Run verifies the candidate.
func TestForwardPassesUseOneBaseProcessForEveryPurpose(t *testing.T) {
	f := newFixture(t)
	out, results := f.apply()
	if out.Status != Verified {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	if len(results) != 4 || len(out.Passes) != 4 {
		t.Fatalf("passes %d, results %d", len(out.Passes), len(results))
	}
	for i, purpose := range []Purpose{Plan, Prepare, Implement, Verify} {
		p := out.Passes[i]
		if p.Purpose != purpose || !p.Sealed || p.Verdict != Satisfied || p.ID != results[i].Record.ID {
			t.Fatalf("pass %d: %+v", i, p)
		}
		for _, m := range p.Managers {
			if m.State != Done || m.Basis == "" || m.Result == "" || m.Verdict != Satisfied {
				t.Errorf("%s in %s: %+v", m.Manager, p.ID, m)
			}
		}
	}
	ids := map[string]bool{}
	for _, p := range out.Passes {
		if ids[p.ID] {
			t.Errorf("repeated pass identity %s", p.ID)
		}
		ids[p.ID] = true
	}
	// Every Manager took part with at least a basis and a result turn per Pass.
	for _, m := range teamPaths {
		for _, p := range out.Passes {
			n := 0
			for _, tr := range out.Turns {
				if tr.Manager == m && tr.Pass == p.ID && (tr.Role == "manager" || tr.Role == "reviewer") {
					n++
				}
			}
			if n < 2 {
				t.Errorf("%s had %d turns in %s", m, n, p.ID)
			}
		}
	}
	if !out.Candidate.Equal(f.cfg.Start) {
		t.Errorf("an unchanged stand verifies as the start")
	}
}

// Every Manager receives its assignment without an unrelated parent preapproval.
func TestChildrenStartWithoutParentBasis(t *testing.T) {
	f := newFixture(t)
	r := f.open()
	res := r.Pass(prepareSpec())
	f.finish(r, "")
	if res.Stopped {
		t.Fatal("stopped")
	}
	for _, pair := range [][2]string{{".", "lending"}, {".", "library"}, {"lending", "lending/reminders"}} {
		released := f.ev.index("manager.release", pair[1], "", nil)
		accepted := f.ev.index("basis.accepted", pair[0], "", nil)
		if released < 0 || accepted < released {
			t.Fatalf("child %s waited for parent: release %d basis %d", pair[1], released, accepted)
		}
	}
}

// A child's submitted proposal wakes its waiting parent before the child is
// Done, so the parent reviews basis and result without a deadlock.
func TestProposalWakesTheParentBeforeTheChildIsDone(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var parentDecisions []string
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "lending" && len(c.pendingDecisions()) > 0 {
			mu.Lock()
			parentDecisions = append(parentDecisions, c.pendingDecisions()...)
			mu.Unlock()
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if len(parentDecisions) != 2 {
		t.Fatalf("lending decided %v; want the basis and the result of its child", parentDecisions)
	}
	if got := out.Passes[0].Managers; got[2].Manager != "lending/reminders" || got[2].Basis != parentDecisions[0] || got[2].Result != parentDecisions[1] {
		t.Errorf("decisions %v, records %+v", parentDecisions, got[2])
	}
}

// A returned basis goes back to its owner within the Pass and releases no
// descendant; the owner submits again.
func TestReturnedOptionalBasisDoesNotBlockIndependentChild(t *testing.T) {
	f := newFixture(t)
	returned := false
	var mu sync.Mutex
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "." && !returned {
			for _, id := range c.pendingDecisions() {
				if strings.Contains(c.turn.Prompt, "lending submitted its basis "+id) || strings.Contains(c.turn.Prompt, "lending (loans) submitted its basis "+id) {
					c.must(testAssessment, map[string]any{"kind": kindReturn, "proposal": id, "reason": "the loan interface is missing"})
					returned = true
				}
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(prepareSpec())
	out := f.finish(r, "")
	if res.Stopped || !returned {
		t.Fatalf("stopped %v, returned %v: %s", res.Stopped, returned, out.Reason)
	}
	ret := f.ev.index("basis.returned", "lending", "", nil)
	rel := f.ev.index("manager.release", "lending/reminders", "", nil)
	if ret < 0 || rel < 0 || rel > ret {
		t.Errorf("independent grandchild release %d waited for optional basis return %d", rel, ret)
	}
	if n := len(f.ev.kinds("basis.accepted")); n != 4 {
		t.Errorf("%d accepted bases; want 4 after the resubmission", n)
	}
	if !strings.Contains(promptOf(f, "lending", "the loan interface is missing"), "returned your basis") {
		t.Errorf("lending never saw the return")
	}
}

func promptOf(f *fixture, manager, text string) string {
	for _, tr := range f.agents.turnsOf(manager) {
		if strings.Contains(tr.Prompt, text) {
			return tr.Prompt
		}
	}
	return ""
}

// Missing final semantic content stays unknown. One targeted correction is
// allowed before a visible Incomplete outcome; no report call is requested.
func TestMissingFinalAssessmentRemainsIncompleteAfterOneCorrection(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" {
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if !res.Stopped || out.Status != Incomplete || !strings.Contains(out.Reason, `library`) {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	for _, m := range out.Passes[0].Managers {
		if m.Manager == "library" && m.State == Done || m.Manager == "." && m.State == Done {
			t.Errorf("%s is Done without a final assessment", m.Manager)
		}
	}
	if len(f.agents.turnsOf("library")) != 2 || !strings.Contains(promptOf(f, "library", "Correct only"), "Your work was captured") {
		t.Errorf("no note reached library")
	}
}

// A concise result still needs assessed evidence, not a bare success label.

func TestSubmissionGateChecksOnlyTheSubmittersStableArea(t *testing.T) {
	f := newFixture(t)
	f.cfg.FileCheck = func(tree snapshot.Tree) []Finding {
		var out []Finding
		if _, ok := tree.Lookup(mustPath(t, "library/new.go")); ok {
			if _, ok := tree.Lookup(mustPath(t, "library/registry")); !ok {
				out = append(out, Finding{Owner: "library", Obligation: "file.unexplained", Observed: "library/new.go is not registered"})
			}
		}
		if _, ok := tree.Lookup(mustPath(t, "lending/wip.go")); ok {
			out = append(out, Finding{Owner: "lending", Obligation: "file.unexplained", Observed: "lending/wip.go is not registered"})
		}
		return out
	}
	libraryDone := make(chan struct{})
	var once sync.Once
	f.agents.script = func(c *call) error {
		switch {
		case c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on your own contribution"):
			c.write("library/new.go", "package library")
			if strings.Contains(c.turn.Prompt, "did not pass the File check") {
				c.write("library/registry", "new.go")
			}
		case c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on your own contribution"):
			if !strings.Contains(c.turn.Prompt, "Continue your work") {
				c.write("lending/wip.go", "work in progress")
				c.must(testAssessment, map[string]any{"kind": kindProgress, "summary": "half done"})
				return nil
			}
			if err := c.wait(libraryDone); err != nil {
				return err
			}
			c.ag.ws.remove(c.turn.Workspace, "lending/wip.go")
		}
		c.protocol()
		if c.turn.Manager == "." && f.ev.index("result.accepted", "library", "", nil) >= 0 {
			once.Do(func() { close(libraryDone) })
		}
		return nil
	}
	r := f.open()
	res := r.Pass(prepareSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
	checked := f.ev.kinds("proposal.checked")
	failed := 0
	for _, ev := range checked {
		data := ev.Data.(map[string]any)
		if data["passed"] == false {
			failed++
			if ev.Manager != "library" {
				t.Errorf("%s failed its check; the live work of another area leaked into it", ev.Manager)
			}
		}
	}
	if failed != 1 {
		t.Errorf("%d failed checks; want library's first basis", failed)
	}
	if content, ok := f.ws.file("run1-work", "library/registry"); !ok || content != "new.go" {
		t.Errorf("the registration was not made")
	}
}

// A real timer cancels an active scripted writer. Its observed end is still
// required; the timer neither seals work nor manufactures a Verify result.
func TestWorkCutoffCancelsActiveWorkWithoutSpendingTheVerifyReserve(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.WorkDeadline = time.Now().Add(80 * time.Millisecond)
	f.agents.script = func(c *call) error { <-c.ctx.Done(); return c.ctx.Err() }
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if !res.Stopped || out.Status != Incomplete || !strings.Contains(out.Reason, "work deadline") {
		t.Fatalf("result %+v, outcome %s: %s", res, out.Status, out.Reason)
	}
	for _, tr := range out.Turns {
		if !tr.Stopped {
			t.Fatalf("unobserved stop: %+v", tr)
		}
	}
	if len(out.Passes) != 1 || out.Passes[0].Sealed {
		t.Fatalf("unfinished work was sealed: %+v", out.Passes)
	}
}

func TestVerifyMayUseItsReservedWindowAfterTheWorkCutoff(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.WorkDeadline = time.Now().Add(-time.Second)
	r := f.open()
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if res.Stopped || out.Status != Verified {
		t.Fatalf("Verify lost its reserve: %s %s", out.Status, out.Reason)
	}
}
