package run

import (
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
// A cycle with the parent-first order is refused.
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
	revised := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "lending" && !revised && f.ev.index("basis.accepted", "lending/reminders", "", nil) >= 0 {
			c.must(toolReport, map[string]any{"kind": kindRevise, "reason": "the reminder needs the due date"})
			revised = true
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
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
