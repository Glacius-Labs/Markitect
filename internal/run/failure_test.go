package run

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// libraryFinding scripts round 1: library raises one blocking finding about
// itself, everything else is default, and run executes the Execute turns.
func libraryFinding(execute func(*call) error) func(*call) error {
	return func(c *call) error {
		switch {
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "library obligation", true)})
			return nil
		case c.turn.Kind == ExecuteTurn && execute != nil:
			return execute(c)
		}
		return c.defaultScript()
	}
}

func TestExecuteTurnWithoutObservedStopFinishesIncomplete(t *testing.T) {
	f := newFixture(t)
	f.agents.script = libraryFinding(func(c *call) error {
		c.noStop = true
		return nil
	})
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "without an observed container stop") {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if f.ws.harvests != 0 || f.ws.isClosed("run1-work") {
		t.Fatal("captured or closed after uncertain quiescence")
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
	f.agents.script = libraryFinding(func(c *call) error {
		c.write("library/new.go", "x")
		return c.defaultScript()
	})
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "capture") {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if f.ws.harvests != 1 || f.ws.isClosed("run1-work") || len(out.Rounds) != 1 {
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
		if c.round() == 2 {
			f.ws.setExtra()
		}
		c.write("library/new"+c.turn.ID+".go", "x")
		return c.defaultScript()
	})
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 3 || out.Rounds[1].Green {
		t.Fatalf("%s %s rounds %d", out.Status, out.Reason, len(out.Rounds))
	}
	owners := map[string]string{}
	for _, fd := range out.Findings {
		if fd.RaisedBy != MarkitectFinding {
			continue
		}
		if !fd.Blocking || !fd.Closed || fd.ClosedRound != 3 {
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
	contains(t, "library verify prompt", f.agents.prompt("run1-r2-library-v1"), "library/odd")
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
		if tr.Manager == "library" && !strings.Contains(tr.Reason, "test failure") {
			t.Errorf("turn record %+v", tr)
		}
	}
}

func TestDeadlineStopsRunningTurnsAndKeepsTheSharedTree(t *testing.T) {
	t.Run("verify", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.Deadline = time.Now().Add(150 * time.Millisecond)
		f.agents.script = func(c *call) error {
			<-c.ctx.Done()
			return c.ctx.Err()
		}
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "deadline") {
			t.Fatalf("%s %s", out.Status, out.Reason)
		}
		if len(out.Turns) != 2 || !f.ws.isClosed("run1-c0") || f.ws.isClosed("run1-work") {
			t.Fatalf("turns %v closed %v", turnIDs(out), f.ws.closed)
		}
		for _, tr := range out.Turns {
			if !tr.Stopped || tr.Reason == "" {
				t.Fatalf("turn %+v", tr)
			}
		}
	})
	t.Run("execute", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.Deadline = time.Now().Add(300 * time.Millisecond)
		f.agents.script = libraryFinding(func(c *call) error {
			<-c.ctx.Done()
			return c.ctx.Err()
		})
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "deadline") || f.ws.harvests != 0 {
			t.Fatalf("%s %s harvests %d", out.Status, out.Reason, f.ws.harvests)
		}
		if countKind(out, ExecuteTurn) != 1 || f.ws.isClosed("run1-work") {
			t.Fatalf("turns %v", turnIDs(out))
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
	t.Run("turns do not cover a Verify round", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.AgentTurns = 3
		out := f.run()
		if out.Status != Incomplete || len(out.Turns) != 0 || len(f.ws.order) != 0 || !strings.Contains(out.Reason, "budget") {
			t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
		}
	})
	t.Run("turns end after the first round", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.AgentTurns = 4
		f.agents.script = libraryFinding(nil)
		out := f.run()
		if out.Status != Incomplete || countKind(out, ExecuteTurn) != 0 || countKind(out, VerifyTurn) != 4 || !strings.Contains(out.Reason, "budget") || f.ws.harvests != 0 {
			t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
		}
	})
	t.Run("turns end during repair", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.AgentTurns = 9
		f.agents.script = func(c *call) error {
			switch {
			case c.is(1, "library", VerifyTurn):
				c.assess(false, []fnd{finding("library", "a", true)})
				return nil
			case c.is(1, "lending", VerifyTurn):
				c.assess(false, []fnd{finding("lending", "b", true)})
				return nil
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "budget") || len(out.Turns) > 9 || len(out.Turns) < 5 {
			t.Fatalf("%s %s %d turns", out.Status, out.Reason, len(out.Turns))
		}
	})
	t.Run("rounds end without green", func(t *testing.T) {
		f := newFixture(t)
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
		if countKind(out, ExecuteTurn) != 1 {
			t.Fatalf("no repair follows the last round: %v", turnIDs(out))
		}
	})
	t.Run("one round never starts Execute", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.Rounds = 1
		f.agents.script = libraryFinding(nil)
		out := f.run()
		if out.Status != Incomplete || countKind(out, ExecuteTurn) != 0 || f.ws.harvests != 0 {
			t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
		}
	})
}

func TestEarlyExecuteKeepsRoomForTheOpenRoundOneVerifyTurns(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.AgentTurns = 8
	libReported := make(chan struct{})
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "library obligation", true)})
			close(libReported)
			return nil
		case c.is(1, "lending/reminders", VerifyTurn):
			if err := c.wait(libReported); err != nil {
				return err
			}
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "budget") || countKind(out, ExecuteTurn) != 0 || len(out.Turns) != 4 {
		t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
	}
}

func TestIncompleteExitsCloseTheCandidateVolume(t *testing.T) {
	t.Run("no budget for another round", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.Budget.AgentTurns = 9
		libReported := make(chan struct{})
		f.agents.script = func(c *call) error {
			switch {
			case c.is(1, "library", VerifyTurn):
				c.assess(false, []fnd{finding("library", "library obligation", true)})
				close(libReported)
				return nil
			case c.is(1, "lending/reminders", VerifyTurn) && strings.HasSuffix(c.turn.ID, "-v1"):
				// No assessment: the retry takes one more turn of the budget.
				return c.wait(libReported)
			case c.turn.Kind == ExecuteTurn:
				c.write("library/n.go", "x")
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "cannot cover another Verify round") || !slices.Contains(f.ws.order, "run1-c1") {
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
		f.agents.script = libraryFinding(func(c *call) error {
			c.write("library/n.go", "x")
			return c.defaultScript()
		})
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

func TestExecuteFailuresAreRetriedOnceThenTheManagerIsStopped(t *testing.T) {
	t.Run("repeated", func(t *testing.T) {
		f := newFixture(t)
		f.agents.script = libraryFinding(func(c *call) error { return errTest })
		out := f.run()
		if out.Status != Incomplete || !strings.Contains(out.Reason, "stopped after repeated failed turns") {
			t.Fatalf("%s %s", out.Status, out.Reason)
		}
		if countKind(out, ExecuteTurn) != 2 || f.ws.harvests != 0 || f.ev.count("manager.disabled") != 1 {
			t.Fatalf("turns %v harvests %d", turnIDs(out), f.ws.harvests)
		}
		contains(t, "second prompt", f.agents.prompt("run1-r1-library-x2"), "Note: your previous turn ended with an error", "test failure")
	})
	t.Run("recovers", func(t *testing.T) {
		f := newFixture(t)
		f.agents.script = libraryFinding(func(c *call) error {
			if strings.HasSuffix(c.turn.ID, "-x1") {
				return errTest
			}
			c.write("library/n.go", "x")
			return c.defaultScript()
		})
		out := f.run()
		if out.Status != Verified || countKind(out, ExecuteTurn) != 2 {
			t.Fatalf("%s %s", out.Status, out.Reason)
		}
	})
	t.Run("first turn without a conversation gets its items again", func(t *testing.T) {
		f := newFixture(t)
		f.agents.noConversation = true
		f.agents.script = libraryFinding(func(c *call) error {
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
		contains(t, "second prompt", f.agents.prompt("run1-r1-library-x2"), mandate, "Finding F1", "Note: your previous turn ended with an error")
	})
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
			if c.turn.Kind == ExecuteTurn {
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
		"check owner":     func(c *Config) { c.Checks = []Check{{Name: "u", Owner: "ghost", Request: checkRequestFor(t)}} },
		"check request":   func(c *Config) { c.Checks = []Check{{Name: "u", Owner: ".", Request: CheckRequest{}}} },
		"no runner": func(c *Config) {
			c.Checks = []Check{{Name: "u", Owner: ".", Request: checkRequestFor(t)}}
			c.Runner = nil
		},
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
