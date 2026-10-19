package run

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// libraryFinding scripts round 1: library raises one blocking finding about
// itself and everything else is default. repair, when given, runs the Execute
// turns of the repair phases (round 2 on); the first Execute phase is default.
func libraryFinding(repair func(*call) error) func(*call) error {
	return func(c *call) error {
		switch {
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "library obligation", true)})
			return nil
		case c.turn.Kind == ExecuteTurn && c.round() >= 2 && repair != nil:
			return repair(c)
		}
		return c.defaultScript()
	}
}

func TestExecuteTurnWithoutObservedStopFinishesIncomplete(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		c.noStop = true
		return nil
	}
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "without an observed container stop") {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if f.ws.harvests != 0 || f.ws.isClosed("run1-work") || len(out.Turns) != 1 {
		t.Fatalf("captured, closed or went on after uncertain quiescence: %v", turnIDs(out))
	}
	for _, tr := range out.Turns {
		if tr.Kind == ExecuteTurn && tr.Stopped {
			t.Fatal("record claims a stop")
		}
	}
}

func TestVerifyTurnWithoutObservedStopFinishesIncomplete(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.is(1, "library", VerifyTurn) {
			c.noStop = true
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "observed container stop") || len(out.Rounds) != 1 || out.Rounds[0].Green {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
}

func TestUncertainHarvestFinishesIncompleteAndKeepsTheTree(t *testing.T) {
	f := newFixture(t)
	f.ws.uncertain = true
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "capture") {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if f.ws.harvests != 1 || f.ws.isClosed("run1-work") || len(out.Rounds) != 0 || countKind(out, VerifyTurn) != 0 {
		t.Fatalf("harvests %d rounds %d", f.ws.harvests, len(out.Rounds))
	}
	if !out.Candidate.Equal(snapshot.Tree{}) {
		t.Fatal("an incomplete run names a candidate")
	}
}

func TestUnsupportedHarvestEntriesBecomeFindingsOfTheirOwners(t *testing.T) {
	f := newFixture(t)
	f.ws.setExtra("library/odd", "link", "../x")
	f.agents.script = libraryFinding(func(c *call) error {
		f.ws.setExtra()
		c.write("library/new"+c.turn.ID+".go", "x")
		return c.defaultScript()
	})
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 2 || out.Rounds[0].Green || !out.Rounds[1].Green {
		t.Fatalf("%s %s rounds %d", out.Status, out.Reason, len(out.Rounds))
	}
	owners := map[string]string{}
	for _, fd := range out.Findings {
		if fd.RaisedBy != MarkitectFinding {
			continue
		}
		if !fd.Blocking || !fd.Closed || fd.ClosedRound != 2 {
			t.Errorf("finding %+v", fd)
		}
		for _, name := range []string{"library/odd", "link", "../x"} {
			if strings.Contains(fd.Observed, `"`+name+`"`) {
				owners[name] = fd.Owner
			}
		}
	}
	if want := (map[string]string{"library/odd": "library", "link": ".", "../x": "."}); !maps.Equal(owners, want) {
		t.Fatalf("owners %v", owners)
	}
	contains(t, "library verify prompt", f.agents.prompt("run1-r1-library-v1"), "library/odd")
}

func TestVerifyTurnErrorIsKeptInTheRunReason(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.is(1, "library", VerifyTurn) {
			return errTest
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "test failure") || !strings.Contains(out.Reason, `"library"`) {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	for _, tr := range out.Turns {
		if tr.Manager == "library" && tr.Kind == VerifyTurn && !strings.Contains(tr.Reason, "test failure") {
			t.Errorf("turn record %+v", tr)
		}
	}
}

func TestDeadlineStopsRunningTurnsAndKeepsTheSharedTree(t *testing.T) {
	t.Run("execute", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.Deadline = time.Now().Add(150 * time.Millisecond)
		f.agents.script = func(c *call) error {
			<-c.ctx.Done()
			return c.ctx.Err()
		}
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "deadline") || f.ws.harvests != 0 {
			t.Fatalf("%s %s harvests %d", out.Status, out.Reason, f.ws.harvests)
		}
		if len(out.Turns) != 1 || f.ws.isClosed("run1-work") || len(f.ws.order) != 1 {
			t.Fatalf("only Root's first turn runs: %v closed %v", turnIDs(out), f.ws.closed)
		}
		for _, tr := range out.Turns {
			if !tr.Stopped || tr.Reason == "" {
				t.Fatalf("turn %+v", tr)
			}
		}
	})
	t.Run("verify", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.Deadline = time.Now().Add(400 * time.Millisecond)
		f.agents.script = func(c *call) error {
			if c.turn.Kind == VerifyTurn {
				<-c.ctx.Done()
				return c.ctx.Err()
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "deadline") {
			t.Fatalf("%s %s", out.Status, out.Reason)
		}
		if countKind(out, VerifyTurn) != 2 || f.ws.harvests != 1 || !f.ws.isClosed("run1-c1") || f.ws.isClosed("run1-work") {
			t.Fatalf("turns %v closed %v", turnIDs(out), f.ws.closed)
		}
		for _, tr := range out.Turns {
			if tr.Kind == VerifyTurn && (!tr.Stopped || tr.Reason == "") {
				t.Fatalf("turn %+v", tr)
			}
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		f := newFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		f.agents.script = func(c *call) error {
			cancel()
			<-c.ctx.Done()
			return c.ctx.Err()
		}
		out, err := Reconcile(ctx, f.cfg)
		if err != nil || out.Status != Incomplete || !strings.Contains(out.Reason, "cancelled") {
			t.Fatalf("%v %s %s", err, out.Status, out.Reason)
		}
		f.assertEnded(out)
	})
}

func TestTurnAndRoundBudgetsEndIncomplete(t *testing.T) {
	t.Run("turns do not cover an Execute and a Verify turn of every Manager", func(t *testing.T) {
		for _, turns := range []int{3, 7} {
			f := newFixture(t)
			f.cfg.Budget.AgentTurns = turns
			out := f.run()
			if out.Status != Incomplete || len(out.Turns) != 0 || len(f.ws.order) != 0 || !strings.Contains(out.Reason, "budget") {
				t.Fatalf("%d turns: %s %s %v", turns, out.Status, out.Reason, turnIDs(out))
			}
		}
	})
	t.Run("turns end after the first round", func(t *testing.T) {
		f := newFixture(t)
		f.agents.quiet = true
		f.cfg.Budget.AgentTurns = 8
		f.agents.script = libraryFinding(nil)
		out := f.run()
		if out.Status != Incomplete || countKind(out, ExecuteTurn) != 4 || countKind(out, VerifyTurn) != 4 || !strings.Contains(out.Reason, "budget") || f.ws.harvests != 1 {
			t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
		}
	})
	t.Run("turns end during repair", func(t *testing.T) {
		f := newFixture(t)
		f.agents.quiet = true
		f.cfg.Budget.AgentTurns = 13
		f.agents.script = func(c *call) error {
			switch {
			case c.is(1, "library", VerifyTurn):
				c.assess(false, []fnd{finding("library", "a", true)})
				return nil
			case c.is(1, "lending", VerifyTurn):
				c.assess(false, []fnd{finding("lending", "b", true)})
				return nil
			case c.is(2, "library", VerifyTurn):
				c.assess(false, nil) // keeps its finding open
				return nil
			}
			return c.defaultScript()
		}
		out := f.run()
		// Round 1 takes eight turns. The repair phase has room for one Execute turn
		// beside a whole Verify round: lending repairs, the library is refused, round
		// 2 verifies and is not green, and no further Execute turn fits.
		if out.Status != Incomplete || !strings.Contains(out.Reason, "budget") || len(out.Turns) != 13 || countKind(out, ExecuteTurn) != 5 || len(out.Rounds) != 2 || f.ws.harvests != 2 {
			t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
		}
		if executeTurnsOf(out, "lending", 2) != 1 || executeTurnsOf(out, "library", 2) != 0 || f.ev.count("execute.budget_limit") != 2 {
			t.Fatalf("%v %d limit events", turnIDs(out), f.ev.count("execute.budget_limit"))
		}
	})
	t.Run("rounds end without green", func(t *testing.T) {
		f := newFixture(t)
		f.agents.quiet = true
		f.cfg.Budget.Rounds = 2
		f.agents.script = func(c *call) error {
			switch {
			case c.is(1, "lending", VerifyTurn):
				c.assess(false, []fnd{finding("lending/reminders", "r", true)})
				return nil
			case c.turn.Manager == "lending" && c.turn.Kind == VerifyTurn:
				c.assess(false, nil) // keeps its finding open
				return nil
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Incomplete || len(out.Rounds) != 2 || !strings.Contains(out.Reason, "round budget") {
			t.Fatalf("%s %s %d rounds", out.Status, out.Reason, len(out.Rounds))
		}
		if executeTurnsOf(out, "lending/reminders", 2) != 1 || countKind(out, ExecuteTurn) != 5 {
			t.Fatalf("no repair follows the last round: %v", turnIDs(out))
		}
	})
	t.Run("one round runs one Execute phase and no repair", func(t *testing.T) {
		f := newFixture(t)
		f.agents.quiet = true
		f.cfg.Budget.Rounds = 1
		f.agents.script = libraryFinding(nil)
		out := f.run()
		if out.Status != Incomplete || countKind(out, ExecuteTurn) != 4 || len(out.Rounds) != 1 || f.ws.harvests != 1 || !strings.Contains(out.Reason, "round budget") {
			t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
		}
	})
}

func TestBudgetLimitLeadsToCaptureAndVerifyNotToAHalt(t *testing.T) {
	f := newFixture(t)
	f.agents.quiet = true
	f.cfg.Budget.AgentTurns = 10
	// Lending and reminders keep messaging each other: every message starts
	// another Execute turn, until the budget cannot also cover the Verify round.
	f.agents.script = func(c *call) error {
		switch c.turn.Manager {
		case "lending/reminders":
			c.send("parent", "ping")
		case "lending":
			if c.turn.Resume != "" {
				c.send("lending/reminders", "pong")
			}
		}
		return c.defaultScript()
	}
	out := f.run()
	// The refused turns' pending items do not keep the phase open: the stand is
	// captured and the reserved turns run the Verify round, which is green.
	if out.Status != Verified || countKind(out, ExecuteTurn) != 6 || countKind(out, VerifyTurn) != 4 || f.ws.harvests != 1 || len(out.Rounds) != 1 || !out.Rounds[0].Green {
		t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
	}
	if limits := f.ev.kinds("execute.budget_limit"); len(limits) != 1 || limits[0].Round != 1 {
		t.Fatalf("budget limit events %+v", limits)
	}
}

func TestReserveKeepsTheFirstTurnOfAnUnreleasedChild(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.AgentTurns = 8
	libraryAsked := make(chan struct{})
	f.ev.onRecord = func(e Event) {
		if e.Kind == "message" && e.Manager == "library" {
			close(libraryAsked)
		}
	}
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "library", ExecuteTurn):
			c.send("parent", "which basis do I use?")
		case c.is(1, "lending", ExecuteTurn):
			// Lending's first turn runs until the library's message reached Root, so
			// that lending/reminders is not released yet.
			if err := c.wait(libraryAsked); err != nil {
				return err
			}
		}
		return c.defaultScript()
	}
	out := f.run()
	// The library's message would give Root a second turn, which would take the
	// budget of lending/reminders' first turn: it is refused, and the run stays
	// within eight turns and is Verified.
	if out.Status != Verified || executeTurnsOf(out, ".", 1) != 1 || executeTurnsOf(out, "lending/reminders", 1) != 1 || countKind(out, VerifyTurn) != 4 {
		t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
	}
	if limits := f.ev.kinds("execute.budget_limit"); len(limits) != 1 || limits[0].Manager != "." {
		t.Fatalf("budget limit events %+v", limits)
	}
}

func TestIncompleteExitsCloseTheCandidateVolume(t *testing.T) {
	t.Run("the budget ends in the Verify round", func(t *testing.T) {
		f := newFixture(t)
		f.agents.quiet = true
		f.cfg.Budget.AgentTurns = 9
		f.agents.script = func(c *call) error {
			if (c.is(1, "lending/reminders", VerifyTurn) || c.is(1, "library", VerifyTurn)) && strings.HasSuffix(c.turn.ID, "-v1") {
				return nil // no assessment: the retries take the turns the round has not got
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "the agent turn budget ended") || !slices.Contains(f.ws.order, "run1-c1") {
			t.Fatalf("%s %s %v %v", out.Status, out.Reason, f.ws.order, turnIDs(out))
		}
		if !f.ws.isClosed("run1-c1") {
			t.Fatal("candidate volume stays")
		}
	})
	t.Run("run stops right after capture", func(t *testing.T) {
		f := newFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f.ev.onRecord = func(e Event) {
			if e.Kind == "candidate" {
				cancel()
			}
		}
		out, err := Reconcile(ctx, f.cfg)
		if err != nil || out.Status != Incomplete {
			t.Fatalf("%v %s %s", err, out.Status, out.Reason)
		}
		if !f.ws.isClosed("run1-c1") {
			t.Fatal("candidate volume stays")
		}
	})
}

func TestEvidenceFailureFinishesIncomplete(t *testing.T) {
	for name, edit := range map[string]func(*recEvidence){
		"start":    func(r *recEvidence) { r.failKind = "run.start" },
		"early":    func(r *recEvidence) { r.failAfter = 6 },
		"turn end": func(r *recEvidence) { r.failKind = "turn.end" },
		"finding":  func(r *recEvidence) { r.failKind = "finding.raised" },
		"outcome":  func(r *recEvidence) { r.failKind = "outcome" },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			edit(f.ev)
			if name == "finding" {
				f.agents.script = libraryFinding(nil)
			}
			out := f.run()
			if out.Status != Incomplete || !strings.Contains(out.Reason, "evidence could not be recorded") {
				t.Fatalf("%s %s", out.Status, out.Reason)
			}
			if !out.Candidate.Equal(snapshot.Tree{}) {
				t.Fatal("candidate named although evidence failed")
			}
		})
	}
}

func TestUnsupportedHarvestEntriesAreAggregatedPerOwner(t *testing.T) {
	f := newFixture(t)
	var names []string
	for i := range 100 {
		names = append(names, fmt.Sprintf("library/link%03d", i), fmt.Sprintf("lending/link%03d", i))
	}
	names = append(names, "../bad")
	f.ws.setExtra(names...)
	f.agents.script = libraryFinding(func(c *call) error {
		f.ws.setExtra()
		c.write("library/new"+c.turn.ID+".go", "x")
		return c.defaultScript()
	})
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	got := map[string]string{}
	for _, fd := range out.Findings {
		if fd.RaisedBy != MarkitectFinding {
			continue
		}
		if _, dup := got[fd.Owner]; dup {
			t.Fatalf("two findings for owner %s", fd.Owner)
		}
		got[fd.Owner] = fd.Observed
	}
	if len(got) != 3 {
		t.Fatalf("findings %v", got)
	}
	contains(t, "library finding", got["library"], "100 unsupported entries", `"library/link000"`, `"library/link019"`, "and 80 more")
	contains(t, "lending finding", got["lending"], "100 unsupported entries", "and 80 more")
	contains(t, "root finding", got["."], "1 unsupported entries", `"../bad"`)
	if strings.Contains(got["library"], `"library/link020"`) || strings.Count(got["library"], `"library/link`) != maxListedNames {
		t.Fatalf("library finding lists too much: %s", got["library"])
	}
}

// libraryExecute runs the Execute turns of library with run and every other
// turn by default.
func libraryExecute(run func(*call) error) func(*call) error {
	return func(c *call) error {
		if c.turn.Kind == ExecuteTurn && c.turn.Manager == "library" {
			return run(c)
		}
		return c.defaultScript()
	}
}

func TestExecuteFailuresAreRetriedOnceThenTheManagerIsStopped(t *testing.T) {
	t.Run("repeated", func(t *testing.T) {
		f := newFixture(t)
		f.agents.script = func(c *call) error {
			if c.turn.Kind == ExecuteTurn && c.turn.Manager == "library" {
				return errTest
			}
			return libraryFinding(nil)(c)
		}
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "stopped after repeated failed turns") {
			t.Fatalf("%s %s", out.Status, out.Reason)
		}
		if executeTurnsOf(out, "library", 1) != 2 || len(out.Rounds) != 1 || f.ev.count("manager.disabled") != 1 {
			t.Fatalf("turns %v rounds %d", turnIDs(out), len(out.Rounds))
		}
		contains(t, "second prompt", f.agents.prompt("run1-r1-library-x2"), "Note: your previous turn ended with an error", "test failure")
	})
	t.Run("recovers", func(t *testing.T) {
		f := newFixture(t)
		f.agents.script = libraryExecute(func(c *call) error {
			if strings.HasSuffix(c.turn.ID, "-x1") {
				return errTest
			}
			c.write("library/n.go", "x")
			return c.defaultScript()
		})
		out := f.run()
		if out.Status != Verified || executeTurnsOf(out, "library", 1) != 2 {
			t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
		}
	})
	for name, noConversation := range map[string]bool{"without a conversation": true, "with a conversation": false} {
		t.Run("failed first turn "+name+" gets mandate and items again", func(t *testing.T) {
			f := newFixture(t)
			f.agents.noConversation = noConversation
			f.agents.script = libraryExecute(func(c *call) error {
				if strings.HasSuffix(c.turn.ID, "-x1") {
					return errTest
				}
				c.write("library/n.go", "x")
				return c.defaultScript()
			})
			out := f.run()
			if out.Status != Verified {
				t.Fatalf("%s %s", out.Status, out.Reason)
			}
			contains(t, "second prompt", f.agents.prompt("run1-r1-library-x2"), mandate, "Library brief.", "Carry out your responsibility now", "Note: your previous turn ended with an error")
			var resume []string
			for _, tr := range f.agents.turnsOf(ExecuteTurn) {
				if tr.Manager == "library" {
					resume = append(resume, tr.Resume)
				}
			}
			if len(resume) != 2 || noConversation != (resume[1] == "") {
				t.Fatalf("resume %q", resume)
			}
		})
	}
}

func TestPortErrors(t *testing.T) {
	t.Run("materialize", func(t *testing.T) {
		f := newFixture(t)
		f.ws.matErr = errTest
		out, err := Reconcile(context.Background(), f.cfg)
		if !errors.Is(err, errTest) || out.Status != Failed || len(out.Turns) != 0 {
			t.Fatalf("%v %s %s", err, out.Status, out.Reason)
		}
	})
	t.Run("a failed work materialization closes the work volume", func(t *testing.T) {
		f := newFixture(t)
		f.ws.matErrFor = map[WorkspaceID]error{"run1-work": errTest}
		out, err := Reconcile(context.Background(), f.cfg)
		if !errors.Is(err, errTest) || out.Status != Failed || len(out.Turns) != 0 || !f.ws.isClosed("run1-work") {
			t.Fatalf("%v %s work closed %v", err, out.Status, f.ws.isClosed("run1-work"))
		}
	})
	t.Run("a failed candidate materialization closes its volume", func(t *testing.T) {
		f := newFixture(t)
		f.ws.matErrFor = map[WorkspaceID]error{"run1-c1": errTest}
		out, err := Reconcile(context.Background(), f.cfg)
		if !errors.Is(err, errTest) || out.Status != Failed || countKind(out, VerifyTurn) != 0 || !f.ws.isClosed("run1-c1") || f.ws.isClosed("run1-work") {
			t.Fatalf("%v %s c1 closed %v work closed %v", err, out.Status, f.ws.isClosed("run1-c1"), f.ws.isClosed("run1-work"))
		}
	})
	t.Run("a check that cannot run is a failed check", func(t *testing.T) {
		f := newFixture(t)
		f.addCheck("unit", "library")
		f.checks.fail = func(r CheckRequest) (CheckReceipt, error, bool) {
			if r.ID != "run1-r1-unit" {
				return CheckReceipt{}, nil, false
			}
			return CheckReceipt{ID: r.ID}, errTest, true
		}
		f.agents.script = libraryFinding(nil)
		out := f.run()
		if out.Status != Verified || out.Rounds[0].Checks[0].Passed || !strings.Contains(out.Rounds[0].Checks[0].Reason, "could not run") {
			t.Fatalf("%s %s %+v", out.Status, out.Reason, out.Rounds[0].Checks)
		}
		f.assertEnded(out)
	})
	t.Run("uncertain or timed out check is a failed check", func(t *testing.T) {
		f := newFixture(t)
		f.addCheck("unit", "library")
		f.checks.fail = func(r CheckRequest) (CheckReceipt, error, bool) {
			if r.ID != "run1-r1-unit" {
				return CheckReceipt{}, nil, false
			}
			return CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true, TimedOut: true, Uncertain: true, Reason: "limit"}, nil, true
		}
		f.agents.script = libraryFinding(nil)
		out := f.run()
		if out.Status != Verified || out.Rounds[0].Checks[0].Passed || !strings.Contains(out.Rounds[0].Checks[0].Reason, "timed out") {
			t.Fatalf("%s %s %+v", out.Status, out.Reason, out.Rounds[0].Checks)
		}
	})
	t.Run("account finding for an unknown owner goes to Root", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Account = func(tree snapshot.Tree) []Finding {
			if tree.EntryCount() > len(f.cfg.Start.Entries()) {
				return nil
			}
			return []Finding{{Owner: "nobody", Obligation: "o", Observed: "x"}}
		}
		f.agents.script = func(c *call) error {
			if c.turn.Kind == ExecuteTurn && c.round() >= 2 {
				if c.turn.Manager != "." {
					t.Errorf("executed by %s", c.turn.Manager)
				}
				c.write("go.sum", "")
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Verified || out.Findings[0].Owner != "." {
			t.Fatalf("%s %s %+v", out.Status, out.Reason, out.Findings)
		}
	})
}

func plan(mounts ...PathMount) WritePlan { return WritePlan{Enforced: true, Mounts: mounts} }

func TestInvalidConfigurationIsRefusedBeforeAnyEffect(t *testing.T) {
	for name, edit := range map[string]func(*Config){
		"run id":         func(c *Config) { c.RunID = "a b" },
		"empty run id":   func(c *Config) { c.RunID = "" },
		"no team":        func(c *Config) { c.Team = nil },
		"root not first": func(c *Config) { c.Team[0], c.Team[3] = c.Team[3], c.Team[0] },
		"duplicate":      func(c *Config) { c.Team = append(c.Team, c.Team[3]) },
		"unknown parent": func(c *Config) { c.Team[3].Parent = "ghost" },
		"unlisted child": func(c *Config) { c.Team[0].Children = []string{"lending"} },
		"phantom child":  func(c *Config) { c.Team[0].Children = append(c.Team[0].Children, "ghost") },
		"own parent": func(c *Config) {
			c.Team[3].Parent = "library"
			c.Team[3].Children = append(c.Team[3].Children, "library")
		},
		"own child":      func(c *Config) { c.Team[3].Children = []string{"library"} },
		"root own child": func(c *Config) { c.Team[0].Children = append(c.Team[0].Children, ".") },
		"no profile":     func(c *Config) { c.Team[3].VerifyProfile = "" },
		"same slug": func(c *Config) {
			c.Team = append(c.Team, Manager{Path: "lending-reminders", Parent: ".", ExecuteProfile: "e", VerifyProfile: "v"})
			c.Team[0].Children = append(c.Team[0].Children, "lending-reminders")
		},
		"bad name":        func(c *Config) { c.Team[3].Path = "lib rary"; c.Team[0].Children[1] = "lib rary" },
		"no deadline":     func(c *Config) { c.Budget.Deadline = time.Time{} },
		"no turn timeout": func(c *Config) { c.Budget.TurnTimeout = 0 },
		"no turns":        func(c *Config) { c.Budget.AgentTurns = 0 },
		"no rounds":       func(c *Config) { c.Budget.Rounds = 0 },
		"no harvest cap":  func(c *Config) { c.MaxBytes = 0 },
		"no account":      func(c *Config) { c.Account = nil },
		"no owner of":     func(c *Config) { c.OwnerOf = nil },
		"no agents":       func(c *Config) { c.Agents = nil },
		"no workspaces":   func(c *Config) { c.Workspaces = nil },
		"no evidence":     func(c *Config) { c.Evidence = nil },
		"mount absolute":  func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "/library"}) },
		"mount parent":    func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "../library"}) },
		"mount inner dot": func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "a/../library"}) },
		"mount root":      func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "."}) },
		"mount empty":     func(c *Config) { c.Team[3].Write = plan(PathMount{}) },
		"mount backslash": func(c *Config) { c.Team[3].Write = plan(PathMount{Path: `library\x`}) },
		"mount drive":     func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "C:/library"}) },
		"mount duplicate": func(c *Config) {
			c.Team[3].Write = plan(PathMount{Path: "library", Writable: true}, PathMount{Path: "library"})
		},
		"mount is a file":      func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "go.mod"}) },
		"mount below a file":   func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "library/a.go/x"}) },
		"descendant mount 1st": func(c *Config) { c.Team[3].Write = plan(PathMount{Path: "library/x"}, PathMount{Path: "library"}) },
		"check owner":          func(c *Config) { c.Checks = []Check{{Name: "u", Owner: "ghost", Request: checkRequestFor(t)}} },
		"check request":        func(c *Config) { c.Checks = []Check{{Name: "u", Owner: ".", Request: CheckRequest{}}} },
		"no runner": func(c *Config) {
			c.Checks = []Check{{Name: "u", Owner: ".", Request: checkRequestFor(t)}}
			c.Runner = nil
		},
		"earlier question open": func(c *Config) { c.Questions = []Question{{ID: "Q1", Text: "q"}} },
		"earlier questions out of order": func(c *Config) {
			c.Questions = []Question{{ID: "Q2", Text: "q", Answer: "a", Answered: true}, {ID: "Q1", Text: "q", Answer: "a", Answered: true}}
		},
		"earlier question malformed": func(c *Config) { c.Questions = []Question{{ID: "Q01", Text: "q", Answer: "a", Answered: true}} },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			edit(&f.cfg)
			out, err := Reconcile(context.Background(), f.cfg)
			if !errors.Is(err, ErrInvalid) || out.Status != Failed || out.Reason == "" {
				t.Fatalf("%v %s %q", err, out.Status, out.Reason)
			}
			if len(f.agents.turns) != 0 || len(f.ws.order) != 0 || len(f.ev.events) != 0 {
				t.Fatal("effects before validation")
			}
		})
	}
}
