package run

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

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

// Children start only after their parent's basis is accepted, and a
// grandchild only after its own parent's basis was accepted by Root: never on a
// mere turn exit.
func TestChildrenStartOnlyOnAnAcceptedParentBasis(t *testing.T) {
	f := newFixture(t)
	r := f.open()
	res := r.Pass(prepareSpec())
	f.finish(r, "")
	if res.Stopped {
		t.Fatalf("stopped")
	}
	accepted := func(manager string) int { return f.ev.index("basis.accepted", manager, "", nil) }
	released := func(manager string) int { return f.ev.index("manager.release", manager, "", nil) }
	firstTurn := func(manager string) int { return f.ev.index("turn.start", manager, "", nil) }
	for _, pair := range [][2]string{{".", "lending"}, {".", "library"}, {"lending", "lending/reminders"}} {
		parent, child := pair[0], pair[1]
		if !(accepted(parent) >= 0 && accepted(parent) < released(child) && released(child) < firstTurn(child)) {
			t.Errorf("%s started at %d, released at %d, before its parent's basis acceptance at %d", child, firstTurn(child), released(child), accepted(parent))
		}
	}
	// The grandchild's release needs lending's basis accepted by Root's decision.
	decision := f.ev.index("basis.accepted", "lending", "by", ".")
	if decision < 0 || decision > released("lending/reminders") {
		t.Errorf("lending's basis was not accepted by Root before its child started: %d vs %d", decision, released("lending/reminders"))
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
func TestReturnedBasisStaysWithItsOwnerAndReleasesNoChild(t *testing.T) {
	f := newFixture(t)
	returned := false
	var mu sync.Mutex
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "." && !returned {
			for _, id := range c.pendingDecisions() {
				if strings.Contains(c.turn.Prompt, "lending submitted its basis "+id) || strings.Contains(c.turn.Prompt, "lending (loans) submitted its basis "+id) {
					c.must(toolReport, map[string]any{"kind": kindReturn, "proposal": id, "reason": "the loan interface is missing"})
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
	if ret < 0 || rel < ret {
		t.Errorf("the grandchild was released at %d, before or without the return at %d", rel, ret)
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

// A turn that ends without a report is incomplete work: it is returned with a
// note, and repeated empty turns end the run Incomplete, never Done.
func TestTurnEndWithoutReportIsNeverCompletion(t *testing.T) {
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
	if !res.Stopped || out.Status != Incomplete || !strings.Contains(out.Reason, `"library"`) {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	for _, m := range out.Passes[0].Managers {
		if m.Manager == "library" && m.State == Done || m.Manager == "." && m.State == Done {
			t.Errorf("%s is Done without a report", m.Manager)
		}
	}
	if !strings.Contains(promptOf(f, "library", "ended without a report"), "Your turn ended without a report") {
		t.Errorf("no note reached library")
	}
}

// A complete result covers every own obligation; a missing one is refused.
func TestResultMustCoverEveryOwnObligation(t *testing.T) {
	f := newFixture(t)
	var refusal string
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit your complete result") && refusal == "" {
			a := c.tool(toolReport, map[string]any{"kind": kindResult, "summary": "done", "verdict": "satisfied", "obligations": []any{}})
			refusal = a.Text
			if !a.Refused {
				t.Errorf("a result without coverage was accepted")
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	f.finish(r, "")
	if !strings.Contains(refusal, "library/interface") {
		t.Errorf("refusal %q does not name the missing obligation", refusal)
	}
}

// A submission whose area fails the File check is returned to its owner, and
// the check composes the accepted stand with the submitter's own area only:
// another Manager's unaccepted live work never enters it.
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
				c.must(toolReport, map[string]any{"kind": kindProgress, "summary": "half done"})
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
