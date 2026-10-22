package run

import (
	"slices"
	"strings"
	"sync"
	"testing"
)

func withWorker(f *fixture, manager string, roles ...Helper) {
	for i := range f.cfg.Team {
		if f.cfg.Team[i].Path == manager {
			f.cfg.Team[i].Helpers = roles
			f.cfg.Team[i].MaxHelpers = 2
		}
	}
}

// A Worker gets an explicit existing subset of its Manager's own area in its
// own container; while it works the subset is read-only for the Manager, an
// overlapping or foreign subset is refused, and its result reaches the Manager
// without completing it.
func TestWorkerWritesOnlyItsDelegatedSubset(t *testing.T) {
	f := newFixture(t)
	f.cfg.Start = treeOf(t, map[string]string{"go.mod": "module x\n", "library/a.go": "a", "library/cmd/main.go": "m", "lending/l.go": "l", "lending/reminders/r.go": "r"})
	withWorker(f, "library", Helper{Role: "worker", Profile: "work", Writes: true}, Helper{Role: "architect", Profile: "arch"})
	var mu sync.Mutex
	delegated := false
	var foreign, overlap, readOnly, submitWhileWorking string
	var managerMounts []PathMount
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.helper() && c.turn.Profile == "work":
			if !writable(c.turn, "library/cmd/main.go") || writable(c.turn, "library/a.go") || writable(c.turn, "lending/l.go") {
				t.Errorf("worker mounts %+v", c.turn.Mounts)
			}
			c.write("library/cmd/main.go", "worker wrote it")
		case c.turn.Manager == "library" && !c.helper() && strings.HasPrefix(c.now(), "work on") && !delegated:
			foreign = c.tool(toolDelegate, map[string]any{"role": "worker", "task": "t", "paths": []string{"lending"}}).Text
			readOnly = c.tool(toolDelegate, map[string]any{"role": "architect", "task": "t", "paths": []string{"library/cmd"}}).Text
			c.must(toolDelegate, map[string]any{"role": "worker", "task": "write the command", "paths": []string{"library/cmd"}})
			overlap = c.tool(toolDelegate, map[string]any{"role": "worker", "task": "t", "paths": []string{"library/cmd/main.go"}}).Text
			submitWhileWorking = c.basis(nil).Text
			delegated = true
			return nil
		case c.turn.Manager == "library" && !c.helper() && strings.Contains(c.turn.Prompt, "Your worker h1"):
			managerMounts = c.turn.Mounts
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v: %s", res.Stopped, out.Reason)
	}
	if !strings.Contains(foreign, "not in your own writable area") || !strings.Contains(readOnly, "works read-only") || !strings.Contains(overlap, "overlaps") || !strings.Contains(submitWhileWorking, "helpers are still working") {
		t.Errorf("refusals: %q | %q | %q | %q", foreign, readOnly, overlap, submitWhileWorking)
	}
	if content, _ := f.ws.file("run1-work", "library/cmd/main.go"); content != "worker wrote it" {
		t.Errorf("worker content %q", content)
	}
	if managerMounts == nil || !writable(Turn{ReadOnly: true, Mounts: managerMounts}, "library/a.go") {
		t.Errorf("the Manager lost its own area after the helper: %+v", managerMounts)
	}
	var helperTurn *TurnRecord
	for i := range out.Turns {
		if out.Turns[i].Role == "worker" {
			helperTurn = &out.Turns[i]
		}
	}
	if helperTurn == nil || helperTurn.Pass != "p1" || helperTurn.Manager != "library" {
		t.Fatalf("helper turn %+v", helperTurn)
	}
	if !slicesContains(out.Homes, "run1-p1-library-h1-home") {
		t.Errorf("helper home not owned for removal: %v", out.Homes)
	}
}

func slicesContains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// While a helper holds a subset, a Manager turn mounts it read-only.
func TestManagerTurnsMountHelperSubsetsReadOnly(t *testing.T) {
	m := &member{mgr: &mgr{Manager: Manager{Path: "library", Write: WritePlan{Mounts: []PathMount{{Path: "library", Writable: true}}}}}, p: &pass{spec: implementSpec()}, stage: stageOwn}
	m.helpers = []*helperRun{{paths: []PathMount{{Path: "library/cmd", Writable: true}}}}
	e := &engine{}
	readOnly, mounts := e.mountsLocked(m)
	turn := Turn{ReadOnly: readOnly, Mounts: mounts}
	if !writable(turn, "library/a.go") || writable(turn, "library/cmd/main.go") {
		t.Errorf("mounts %+v", mounts)
	}
	m.stage = stageDelegated
	if readOnly, mounts := e.mountsLocked(m); !readOnly || len(mounts) != 0 {
		t.Errorf("a delegating parent's area is read-only: %v %+v", readOnly, mounts)
	}
}

// A plan's dependency becomes a start condition of the later Pass: library
// starts its Implement work only after lending's accepted Implement basis.
// A cycle with the integration order is refused.
func TestPlanDependenciesBecomeStartConditions(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var cycle string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.planning() && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			pl := samplePlan()
			pl["dependencies"] = []any{map[string]any{"owner": ".", "need": "the result of Root", "pass": "implement", "stage": "result"}}
			cycle = c.basis(map[string]any{"plan": pl}).Text
			pl["dependencies"] = []any{map[string]any{"owner": "lending", "need": "the loan type", "pass": "implement", "stage": "basis"}}
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"plan": pl}))
			return nil
		}
		if c.planning() && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit") {
			pl := samplePlan()
			pl["dependencies"] = []any{map[string]any{"owner": "lending", "need": "the loan type", "pass": "implement", "stage": "basis"}}
			c.must(toolReport, c.envelope(kindResult, map[string]any{"plan": pl}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(planSpec())
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped {
		t.Fatalf("stopped: %s", out.Reason)
	}
	if !strings.Contains(cycle, "cycle") {
		t.Errorf("cycle refusal %q", cycle)
	}
	// Root's basis releases library, but its productive work waits for
	// lending's accepted basis: its first submission follows that acceptance.
	var accepted, released, submitted = -1, -1, -1
	for i, ev := range f.ev.events {
		if ev.Pass != "p2" {
			continue
		}
		switch {
		case ev.Kind == "basis.accepted" && ev.Manager == "lending" && accepted < 0:
			accepted = i
		case ev.Kind == "manager.release" && ev.Manager == "library" && released < 0:
			released = i
		case ev.Kind == "proposal.submitted" && ev.Manager == "library" && submitted < 0:
			submitted = i
		}
	}
	if released < 0 || accepted < 0 || submitted < accepted {
		t.Errorf("library released at %d, submitted at %d, lending's basis accepted at %d", released, submitted, accepted)
	}
	if !strings.Contains(promptOf(f, "library", "Your dependency: Accepted basis"), "lending") {
		t.Errorf("library did not receive its dependency's basis")
	}
}

// A revised basis explicitly invalidates its consumers: the children start
// again on the revised basis.
func TestRevisedBasisResetsItsConsumers(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	revised, bound := false, false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "lending/reminders" && !bound && strings.HasPrefix(c.now(), "work on") {
			bound = true
			a := c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "basis"})
			if !strings.Contains(a.Text, "recorded dependency") {
				return nil
			}
		}
		if c.turn.Manager == "lending" && !revised && f.ev.index("basis.accepted", "lending/reminders", "", nil) >= 0 {
			c.must(toolReport, map[string]any{"kind": kindRevise, "reason": "the reminder needs the due date"})
			revised = true
			return nil
		}
		tolerant(c)
		return nil
	}
	r := f.open()
	res := r.Pass(bindingSpec())
	out := f.finish(r, "")
	if res.Stopped || !revised {
		t.Fatalf("stopped %v revised %v: %s", res.Stopped, revised, out.Reason)
	}
	if f.ev.index("basis.invalidated", "lending", "", nil) < 0 || f.ev.index("proposal.invalidated", "lending/reminders", "", nil) < 0 {
		t.Errorf("no explicit invalidation recorded")
	}
	n := 0
	for _, ev := range f.ev.kinds("manager.release") {
		if ev.Manager == "lending/reminders" {
			n++
		}
	}
	if n != 2 {
		t.Errorf("lending/reminders released %d times; want again after the revision", n)
	}
	if !strings.Contains(promptOf(f, "lending/reminders", "was reset"), "Re-establish") {
		t.Errorf("the child was not told about the reset")
	}
}

// A parent reopens a Done child's result before the Pass closes; the child
// integrates again.
func TestParentReopensAChildResult(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	reopened := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") && !reopened {
			c.must(toolReport, map[string]any{"kind": kindReopen, "child": "library", "reason": "its output does not match lending's"})
			reopened = true
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v: %s", res.Stopped, out.Reason)
	}
	if f.ev.index("result.reopened", "library", "", nil) < 0 || len(f.ev.kinds("result.accepted")) != 5 {
		t.Errorf("reopen not recorded or not re-accepted: %d results", len(f.ev.kinds("result.accepted")))
	}
}

// A failed turn is retried with its items and a note; its submission does not
// stand. Repeated failures stop the required Manager and the run.
func TestFailedTurnsAreRetriedThenStopTheRun(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	fails := 0
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "library" {
			fails++
			if a := c.basis(nil); a.Refused {
				t.Errorf("basis refused: %s", a.Text)
			}
			return errTest
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Status != Incomplete || !strings.Contains(out.Reason, `manager "library" stopped after 2 failed turns`) || fails != 2 {
		t.Fatalf("status %s fails %d: %s", out.Status, fails, out.Reason)
	}
	withdrawn := 0
	for _, ev := range f.ev.kinds("proposal.withdrawn") {
		if ev.Manager == "library" {
			withdrawn++
		}
	}
	if withdrawn != 2 {
		t.Errorf("%d failed submissions of library were withdrawn; want 2", withdrawn)
	}
}

// A turn whose container stop was not observed makes quiescence uncertain.
func TestUnobservedStopEndsTheRunIncomplete(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" {
			c.noStop = true
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Status != Incomplete || !strings.Contains(out.Reason, "without an observed container stop") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}

// Budgets are finite: the agent turn budget and the Pass budget end the run
// Incomplete.
func TestBudgetsEndTheRunIncomplete(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.AgentTurns = 9
	r := f.open()
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Status != Incomplete || !strings.Contains(out.Reason, "agent turn budget of 9 ended") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	f = newFixture(t)
	f.cfg.Budget.Passes = 1
	r = f.open()
	r.Pass(planSpec())
	res := r.Pass(prepareSpec())
	out = f.finish(r, "")
	if !res.Stopped || out.Status != Incomplete || !strings.Contains(out.Reason, "Pass budget of 1 ended") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}

// A report of an earlier, sealed Pass is refused: every repetition has a new
// identity and its own state.
func TestStaleReportsOfASealedPassAreRefused(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var stale ToolHandler
	var staleAnswer string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.pass() == "p1" && c.turn.Manager == "library" && stale == nil {
			stale = c.h
		}
		if c.pass() == "p2" && c.turn.Manager == "library" && staleAnswer == "" && stale != nil {
			old := &call{t: t, ctx: c.ctx, turn: c.turn, h: stale, ag: c.ag}
			staleAnswer = old.basis(nil).Text
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Passes[0].ID == out.Passes[1].ID || !strings.Contains(staleAnswer, "no longer accepts reports") {
		t.Errorf("passes %s %s, stale answer %q", out.Passes[0].ID, out.Passes[1].ID, staleAnswer)
	}
}

// The existing cycle guard includes the implicit explicit-dependency and integration
// edges, which the Haiku Depot trace needed (regression carried over from draft
// PR #536): a result needs every child's result without a declared dependency.
func TestPlanRejectsImplicitIntegrationCycles(t *testing.T) {
	for _, tc := range []struct {
		name, manager, owner, stage string
		refused                     bool
	}{
		{"parent result", "lending/reminders", "lending", "result", true},
		{"ancestor result", "lending/reminders", ".", "result", true},
		{"descendant basis", "lending", "lending/reminders", "basis", true},
		{"descendant result", "lending", "lending/reminders", "result", true},
		{"parent basis", "lending/reminders", "lending", "basis", false},
		{"ancestor basis", "lending/reminders", ".", "basis", false},
		{"other branch result", "library", "lending", "result", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			var mu sync.Mutex
			attempted := false
			f.agents.script = func(c *call) error {
				mu.Lock()
				defer mu.Unlock()
				if !attempted && c.turn.Manager == tc.manager && strings.HasPrefix(c.now(), "work on") {
					attempted = true
					pl := samplePlan()
					pl["dependencies"] = []any{map[string]any{"owner": tc.owner, "need": "required contract", "pass": "implement", "stage": tc.stage}}
					a := c.basis(map[string]any{"plan": pl})
					if a.Refused != tc.refused || tc.refused && !strings.Contains(a.Text, "cycle") {
						t.Errorf("dependency admitted/refused incorrectly: %+v", a)
					}
					if !a.Refused {
						return nil
					}
				}
				c.protocol()
				return nil
			}
			r := f.open()
			res := r.Pass(planSpec())
			out := f.finish(r, "bounded Plan regression")
			if !attempted || res.Stopped {
				t.Fatalf("attempted %v, stopped %v: %s", attempted, res.Stopped, out.Reason)
			}
		})
	}
}

// A Manager that needs another Manager's accepted result first waits for it:
// no turn of it starts while that result is missing, and its acceptance wakes it
// with the accepted submission. Old behavior: only a question or a child could
// be waited for, so the Haiku Depot integrator reported the same progress for 45
// turns, each re-queued at once.
func TestWaitForAnotherManagerWakesOnItsAcceptance(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var waited ToolAnswer
	woken := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") && waited.Text == "" {
			waited = c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "reason": "the program integrates the lending result"})
			return nil
		}
		if c.turn.Manager == "library" && strings.Contains(c.turn.Prompt, "The submission you waited for is accepted") {
			woken = true
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied || !woken || !strings.Contains(waited.Text, "accepted result of lending") {
		t.Fatalf("stopped %v, verdict %s, woken %v, wait %q: %s", res.Stopped, res.Verdict, woken, waited.Text, out.Reason)
	}
	waitAt := f.ev.index("report.wait", "library", "", nil)
	acceptedAt := f.ev.index("result.accepted", "lending", "", nil)
	if waitAt < 0 || acceptedAt < waitAt {
		t.Fatalf("wait %d, lending result %d", waitAt, acceptedAt)
	}
	f.ev.mu.Lock()
	defer f.ev.mu.Unlock()
	for _, ev := range f.ev.events[waitAt:acceptedAt] {
		if ev.Kind == "turn.start" && ev.Manager == "library" {
			t.Errorf("library worked while its wait was unchanged: %+v", ev)
		}
	}
}

// tolerant follows the base process like protocol, for tests in which a
// revision resets or unsettles Managers while their turns run: a decision or
// submission such a change made stale is refused, and the change delivers the
// work again, so those refusals are expected.
func tolerant(c *call) {
	for _, id := range c.pendingDecisions() {
		if !c.decided[id] {
			c.accept(id)
		}
	}
	switch now := c.now(); {
	case strings.HasPrefix(now, "work on your own contribution"):
		c.basis(nil)
	case strings.HasPrefix(now, "submit your complete result"), strings.HasPrefix(now, "integrate your children"):
		c.result(nil)
	}
}

// bindingSpec is an Implement Pass of a workflow without a planning Pass.
func bindingSpec() PassSpec {
	spec := implementSpec()
	spec.BindWaits = true
	return spec
}

// Without a planning Pass, a wait declares the dependency: the accepted
// submission it waited for, or one already accepted when the wait is reported,
// is bound to the waiter's work. A revision of that basis resets the waiter,
// which starts again only on the newly accepted version, recorded as rebound.
// Old behavior: only a plan's dependencies were consumers, so work built on a
// revised basis through a wait stayed accepted.
func TestWaitBindsTheAcceptedSubmissionAndItsRevisionResetsTheWaiter(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	waited, revised := false, false
	var again ToolAnswer
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.turn.Manager == "library" && !waited && strings.HasPrefix(c.now(), "work on"):
			waited = true
			// Lending's basis may already be accepted: then the wait binds at once.
			if a := c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "basis", "reason": "the program uses the loan contract"}); strings.Contains(a.Text, "recorded dependency") {
				again = a
				c.basis(nil)
			}
			return nil
		case c.turn.Manager == "library" && again.Text == "" && strings.Contains(c.turn.Prompt, "The submission you waited for is accepted"):
			again = c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "basis"})
			c.basis(nil)
			return nil
		case c.turn.Manager == "lending" && !revised && f.ev.index("dependency.bound", "library", "on", "lending") >= 0 &&
			(strings.HasPrefix(c.now(), "your children work") || strings.HasPrefix(c.now(), "integrate")):
			revised = true
			c.must(toolReport, map[string]any{"kind": kindRevise, "reason": "the loan contract needs the due date"})
			return nil
		}
		tolerant(c)
		return nil
	}
	r := f.open()
	res := r.Pass(bindingSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied || !revised {
		t.Fatalf("stopped %v, verdict %s, revised %v: %s", res.Stopped, res.Verdict, revised, out.Reason)
	}
	// However the turns interleave, the binding moves from lending's first
	// accepted basis to its revised one.
	var boundTo, accepted []any
	for _, ev := range f.ev.kinds("dependency.bound") {
		if data, _ := ev.Data.(map[string]any); ev.Manager == "library" && data["on"] == "lending" {
			boundTo = append(boundTo, data["proposal"])
		}
	}
	for _, ev := range f.ev.kinds("basis.accepted") {
		if data, _ := ev.Data.(map[string]any); ev.Manager == "lending" {
			accepted = append(accepted, data["proposal"])
		}
	}
	if len(accepted) != 2 || len(boundTo) < 2 || boundTo[0] != accepted[0] || boundTo[len(boundTo)-1] != accepted[1] {
		t.Errorf("library bound to %v; lending's accepted bases %v", boundTo, accepted)
	}
	bound := f.ev.index("dependency.bound", "library", "stage", "basis")
	revisedAt := f.ev.index("basis.invalidated", "lending", "", nil)
	if bound < 0 || revisedAt < bound {
		t.Fatalf("bound at %d, revised at %d", bound, revisedAt)
	}
	f.ev.mu.Lock()
	defer f.ev.mu.Unlock()
	reset, reaccepted, resubmitted := -1, -1, -1
	for i, ev := range f.ev.events[revisedAt:] {
		switch {
		case reset < 0 && ev.Kind == "manager.release" && ev.Manager == "library":
			reset = i
		case reaccepted < 0 && ev.Kind == "basis.accepted" && ev.Manager == "lending":
			reaccepted = i
		case resubmitted < 0 && reaccepted >= 0 && ev.Kind == "proposal.submitted" && ev.Manager == "library":
			resubmitted = i
		}
	}
	if reset < 0 || reaccepted < reset || resubmitted < reaccepted {
		t.Fatalf("library reset %d, lending re-accepted %d, library resubmitted %d", reset, reaccepted, resubmitted)
	}
	// Reset, library waits for the revised basis: it submits nothing before that
	// acceptance (a turn already running at the reset is refused).
	for _, ev := range f.ev.events[revisedAt+reset : revisedAt+reaccepted] {
		if ev.Kind == "proposal.submitted" && ev.Manager == "library" {
			t.Errorf("library submitted before its revised dependency was accepted: %+v", ev)
		}
	}
}

// Without BindWaits, as in Verify, Extract and a staged workflow, a wait binds
// nothing: a wait on an accepted submission is refused as before and a revision
// resets only children and declared dependents.
func TestWaitsBindNothingWithoutBindWaits(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	waited := false
	var again ToolAnswer
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.turn.Manager == "library" && !waited && strings.HasPrefix(c.now(), "work on"):
			waited = true
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "basis"})
			return nil
		case c.turn.Manager == "library" && again.Text == "" && strings.Contains(c.turn.Prompt, "The submission you waited for is accepted"):
			again = c.tool(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "basis"})
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
	if !again.Refused || !strings.Contains(again.Text, "already accepted") || len(f.ev.kinds("dependency.bound")) != 0 {
		t.Errorf("a wait bound without BindWaits: %+v, %d bindings", again, len(f.ev.kinds("dependency.bound")))
	}
	if err := validateSpec(PassSpec{Purpose: Verify, Mandate: "m", Review: true, Guidance: "g", BindWaits: true, Corrections: corrections}); err == nil {
		t.Errorf("a review Pass accepted BindWaits")
	}
}

// A Manager never binds its own ancestors or descendants: the parent-first and
// integration order already relate them. Before, a parent bound to its child's
// result would be reset with its whole subtree when it reopened that child.
func TestWaitsBindNoAncestorOrDescendant(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var own ToolAnswer
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "lending" && own.Text == "" && strings.HasPrefix(c.now(), "integrate") {
			own = c.tool(toolReport, map[string]any{"kind": kindWait, "for": "lending/reminders", "stage": "result"})
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(bindingSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
	if !own.Refused || len(f.ev.kinds("dependency.bound")) != 0 {
		t.Errorf("a parent bound its child's result: %+v", own)
	}
}

// A Manager whose complete result alone built on a revised basis keeps its own
// basis and its children's work: its accepted result is reopened and it
// integrates again on the new version. Before the re-review fix, such a binding
// reset the whole Manager, invalidating its unchanged basis and its subtree.
func TestRevisionReopensOnlyAResultBuiltOnIt(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	lendingWaits, libraryWaited, revised := 0, false, false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		now := c.now()
		switch {
		case c.turn.Manager == "lending" && strings.HasPrefix(now, "integrate") && lendingWaits < 2 && !strings.Contains(c.turn.Prompt, "The submission you waited for is accepted"):
			// Its result integrates library's contract, at first and again after
			// the revision, where it first tries to submit at once.
			lendingWaits++
			if lendingWaits == 2 {
				if a := c.result(nil); !a.Refused {
					return nil
				} else if !strings.Contains(a.Text, "not accepted again yet") {
					t.Errorf("unexpected refusal: %s", a.Text)
				}
			}
			if a := c.tool(toolReport, map[string]any{"kind": kindWait, "for": "library", "stage": "basis"}); !a.Refused && strings.Contains(a.Text, "recorded dependency") {
				c.result(nil)
			}
			return nil
		case c.turn.Manager == "library" && !libraryWaited && strings.HasPrefix(now, "submit your complete result"):
			libraryWaited = true
			// Lending's result may already be accepted: then it binds at once.
			if a := c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "result"}); strings.Contains(a.Text, "recorded dependency") {
				revised = true
				c.must(toolReport, map[string]any{"kind": kindRevise, "reason": "the program's contract changes"})
			}
			return nil
		case c.turn.Manager == "library" && !revised && strings.Contains(c.turn.Prompt, "The submission you waited for is accepted"):
			revised = true
			c.must(toolReport, map[string]any{"kind": kindRevise, "reason": "the program's contract changes"})
			return nil
		case c.turn.Manager == "library" && revised && strings.HasPrefix(now, "submit your complete result") && !strings.Contains(c.turn.Prompt, "The submission you waited for is accepted") && !strings.Contains(c.turn.Prompt, "builds on is accepted"):
			// Its result built on lending's reopened result: it is refused until
			// lending's new result is accepted.
			if a := c.result(nil); a.Refused {
				if !strings.Contains(a.Text, "not accepted again yet") {
					t.Errorf("unexpected refusal: %s", a.Text)
				}
				if w := c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "result"}); strings.Contains(w.Text, "recorded dependency") {
					c.result(nil)
				}
			}
			return nil
		}
		tolerant(c)
		return nil
	}
	r := f.open()
	res := r.Pass(bindingSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied || !revised {
		t.Fatalf("stopped %v, verdict %s, revised %v: %s", res.Stopped, res.Verdict, revised, out.Reason)
	}
	if f.ev.index("dependency.bound", "lending", "serves", "result") < 0 || f.ev.index("result.reopened", "lending", "by", "markitect") < 0 {
		t.Fatalf("lending's result was not bound and reopened")
	}
	for _, m := range []string{"lending", "lending/reminders"} {
		if i := f.ev.index("proposal.invalidated", m, "", nil); i >= 0 {
			t.Errorf("%s lost work that never used library: %+v", m, f.ev.events[i])
		}
	}
	releases := 0
	for _, ev := range f.ev.kinds("manager.release") {
		if ev.Manager == "lending/reminders" {
			releases++
		}
	}
	if releases != 1 {
		t.Errorf("lending/reminders released %d times; its parent's basis never changed", releases)
	}
	// After the reopening, lending's next result is accepted only on library's
	// new basis, however the turns interleave.
	reopened := f.ev.index("result.reopened", "lending", "by", "markitect")
	f.ev.mu.Lock()
	defer f.ev.mu.Unlock()
	rebasis, again := -1, -1
	for i, ev := range f.ev.events[reopened:] {
		switch {
		case rebasis < 0 && ev.Kind == "basis.accepted" && ev.Manager == "library":
			rebasis = i
		case again < 0 && ev.Kind == "result.accepted" && ev.Manager == "lending":
			again = i
		}
	}
	if rebasis < 0 || again < rebasis {
		t.Errorf("lending's result accepted at %d, library's new basis at %d after the reopening", again, rebasis)
	}
}

// Two siblings may build on each other's accepted bases when one of them needs
// the other only for its result: a revision then resets both without a
// deadlock (the review's mutual-binding scenario). A real cycle among bindings
// and the explicit-dependency and integration order is refused before it is bound,
// whether or not its events already happened.
func TestBindingsNeverCloseACycle(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	stage, remindersWaited := 0, false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.turn.Manager == "lending/reminders" && !remindersWaited && strings.HasPrefix(c.now(), "work on"):
			// Its parent decides it only after library's basis is accepted.
			remindersWaited = true
			if a := c.must(toolReport, map[string]any{"kind": kindWait, "for": "library", "stage": "basis"}); strings.Contains(a.Text, "recorded dependency") {
				c.basis(nil)
			}
			return nil
		case stage == 0 && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			stage = 1
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "lending", "stage": "basis"})
			return nil
		case stage == 1 && c.turn.Manager == "lending" && f.ev.index("basis.accepted", "library", "", nil) >= 0 &&
			(strings.HasPrefix(c.now(), "your children work") || strings.HasPrefix(c.now(), "integrate")):
			stage = 2
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "library", "stage": "basis"})
			c.must(toolReport, map[string]any{"kind": kindRevise, "reason": "the loan contract changes"})
			return nil
		}
		tolerant(c)
		return nil
	}
	r := f.open()
	res := r.Pass(bindingSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied || stage != 2 {
		t.Fatalf("stopped %v, verdict %s, stage %d: %s", res.Stopped, res.Verdict, stage, out.Reason)
	}
	if f.ev.index("dependency.bound", "lending", "serves", "result") < 0 {
		t.Errorf("lending's result binding on library's basis was not recorded")
	}

	// White box: the cycle check covers events that already happened.
	g := newFixture(t)
	gr := g.open()
	defer g.finish(gr, "white box")
	e := gr.e
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.newPassLocked(bindingSpec())
	lending, reminders, library := p.byPath["lending"], p.byPath["lending/reminders"], p.byPath["library"]
	basisOf := func(m string) waitTarget { return waitTarget{manager: m, stage: stageNameBasis} }
	resultOf := func(m string) waitTarget { return waitTarget{manager: m, stage: stageNameResult} }
	for _, tc := range []struct {
		name   string
		preset map[*member][]binding
		m      *member
		b      binding
		cycle  bool
	}{
		{"one basis on another branch", nil, library, binding{target: basisOf("lending"), serves: eventBasis}, false},
		{"mutual bases, one for a result", map[*member][]binding{library: {{target: basisOf("lending"), serves: eventBasis}}}, lending, binding{target: basisOf("library"), serves: eventResult}, false},
		{"mutual bases", map[*member][]binding{library: {{target: basisOf("lending/reminders"), serves: eventBasis}}}, reminders, binding{target: basisOf("library"), serves: eventBasis}, true},
		{"a basis on a result that needs it", map[*member][]binding{lending: {{target: basisOf("library"), serves: eventResult}}}, library, binding{target: resultOf("lending"), serves: eventBasis}, true},
	} {
		for _, x := range p.members {
			x.bound = tc.preset[x]
		}
		if cycle := e.bindCycleLocked(tc.m, tc.b); (cycle != "") != tc.cycle {
			t.Errorf("%s: cycle %q, want one: %v", tc.name, cycle, tc.cycle)
		}
	}

	// A wait on an accepted basis binds it at once, recorded as kind bind.
	for _, x := range p.members {
		x.bound = nil
	}
	lending.basis = &proposal{id: "P1", kind: kindBasis, member: lending, status: proposalAccepted}
	library.stage = stageOwn
	ts := &turnState{m: library, id: "wb"}
	e.mu.Unlock()
	a := e.report(library, ts, []byte(`{"kind":"wait","for":"lending","stage":"basis"}`))
	e.mu.Lock()
	if a.Refused || !strings.Contains(a.Text, "recorded dependency") || len(ts.reports) != 1 || ts.reports[0] != kindBind || ts.wait != "" {
		t.Errorf("immediate binding: %+v, reports %v, wait %q", a, ts.reports, ts.wait)
	}
	if len(library.bound) != 1 || library.bound[0] != (binding{target: basisOf("lending"), serves: eventBasis, proposal: "P1"}) {
		t.Errorf("library bound %+v", library.bound)
	}
}

// A wait that closes a cycle with the accepted plans' dependencies could never
// be satisfied: it is refused with the cycle named, so the Manager submits what
// does not need it first and the Pass completes. Before, nothing named such a
// wait and the Haiku Depot run spent its turns on it.
func TestWaitThatClosesACycleIsRefused(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var refused ToolAnswer
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.planning() && c.turn.Manager == "library" && (strings.HasPrefix(c.now(), "work on") || strings.HasPrefix(c.now(), "submit")) {
			pl := samplePlan()
			pl["dependencies"] = []any{map[string]any{"owner": "lending", "need": "the loan result", "pass": "implement", "stage": "result"}}
			kind := kindBasis
			if strings.HasPrefix(c.now(), "submit") {
				kind = kindResult
			}
			c.must(toolReport, c.envelope(kind, map[string]any{"plan": pl}))
			return nil
		}
		if !c.planning() && c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on") && refused.Text == "" {
			refused = c.tool(toolReport, map[string]any{"kind": kindWait, "for": "library", "stage": "result"})
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(planSpec())
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if !refused.Refused || !strings.Contains(refused.Text, "closes a cycle") {
		t.Fatalf("the cyclic wait was not refused with its cycle: %+v", refused)
	}
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
}

// A progress report that repeats an earlier one of the Pass is no progress: it
// is not continued automatically, and repeated it ends the run Incomplete within
// the empty-turn limit instead of spending the turn budget.
func TestRepeatedProgressIsNoProgress(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "library" && (strings.HasPrefix(c.now(), "work on") || strings.Contains(c.turn.Prompt, "repeats an earlier one")) {
			c.must(toolReport, map[string]any{"kind": kindProgress, "summary": "still integrating the parts"})
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if !res.Stopped || out.Status != Incomplete || !strings.Contains(out.Reason, "without the report its work needs") {
		t.Fatalf("stopped %v, status %s: %s", res.Stopped, out.Status, out.Reason)
	}
	if n := len(f.agents.turnsOf("library")); n != maxEmptyTurns+2 {
		t.Errorf("library ran %d turns", n)
	}
}

func TestReportActionsExposeOnlyTheirOwnFields(t *testing.T) {
	for _, spec := range []PassSpec{implementSpec(), verifySpec(), planSpec()} {
		s := reportSchema(spec)
		if _, broad := s["properties"]; broad {
			t.Fatal("broad report envelope exposed")
		}
		for _, action := range s["oneOf"].([]schema) {
			fields := action["properties"].(schema)
			kind := fields["kind"].(schema)["enum"].([]string)[0]
			for name := range fields {
				if name != "kind" && !slices.Contains(reportFields[kind], name) {
					t.Fatalf("%s exposed %s", kind, name)
				}
			}
			if kind == kindProgress && len(fields) != 2 {
				t.Fatalf("progress fields: %+v", fields)
			}
			if kind == kindAccept && len(fields) != 2 {
				t.Fatalf("accept fields: %+v", fields)
			}
			for _, identity := range []string{"run", "pass", "manager", "turn", "stand", "candidate", "children"} {
				if _, ok := fields[identity]; ok {
					t.Fatalf("agent asked for Host identity %s", identity)
				}
			}
			if !spec.Plans {
				if _, ok := fields["plan"]; ok {
					t.Fatal("work/review requested plan payload")
				}
			}
		}
	}
}

// A new dependency version may be accepted before the old consumer Turn ends.
// Its pending report still belongs to the invalidated version and must not stand.
func TestResetWithdrawsRunningSubmissionAfterDependencyReacceptance(t *testing.T) {
	f := newFixture(t)
	r := f.open()
	defer f.finish(r, "bounded reset regression")
	e := r.e
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.newPassLocked(bindingSpec())
	owner, consumer := p.byPath["lending"], p.byPath["library"]
	old := &proposal{id: "old-contract", kind: kindBasis, member: owner, status: proposalAccepted}
	owner.basis = old
	consumer.stage, consumer.running = stageOwn, true
	consumer.bound = []binding{{target: waitTarget{manager: owner.Path, stage: stageNameBasis}, serves: eventBasis, proposal: old.id}}
	ts := &turnState{m: consumer, id: "old-turn"}
	if a := e.submit(consumer, ts, reportIn{Kind: kindBasis, Summary: "built on old contract"}); a.Refused {
		t.Fatal(a.Text)
	}
	submitted := ts.proposal
	owner.basis = nil
	e.resetLocked(consumer, "contract was revised")
	if consumer.stage != stageBlocked {
		t.Fatalf("missing real prerequisite did not block: %s", consumer.stage)
	}
	owner.basis = &proposal{id: "new-contract", kind: kindBasis, member: owner, status: proposalAccepted}
	e.releaseLocked(consumer, "new contract accepted")
	if consumer.stage != stageOwn {
		t.Fatalf("new contract did not release consumer: %s", consumer.stage)
	}
	e.turnEndedLocked(consumer, ts)
	if submitted.status != proposalWithdrawn || consumer.pending != nil || consumer.basis != nil || len(consumer.items) == 0 {
		t.Fatalf("stale report survived: %s pending=%v basis=%v items=%d", submitted.status, consumer.pending, consumer.basis, len(consumer.items))
	}
	consumer.running = false
}
