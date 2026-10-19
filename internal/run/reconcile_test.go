package run

import (
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func contains(t *testing.T, what, s string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(s, p) {
			t.Errorf("%s lacks %q:\n%s", what, p, s)
		}
	}
}

func TestInitialGreenIsVerifiedWithoutExecute(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", "library")
	out := f.run()
	if out.Status != Verified || !out.Candidate.Equal(f.cfg.Start) {
		t.Fatalf("outcome %s %q", out.Status, out.Reason)
	}
	if countKind(out, ExecuteTurn) != 0 || countKind(out, VerifyTurn) != 4 {
		t.Fatalf("turns %v", turnIDs(out))
	}
	if len(out.Rounds) != 1 || !out.Rounds[0].Green || len(out.Rounds[0].Assessments) != 4 || len(out.Rounds[0].Checks) != 1 || !out.Rounds[0].Checks[0].Passed {
		t.Fatalf("rounds %+v", out.Rounds)
	}
	if !slices.Equal(f.ws.order, []string{"run1-c0", "run1-work"}) || !f.ws.isClosed("run1-c0") || !f.ws.isClosed("run1-work") {
		t.Fatalf("volumes %v closed %v", f.ws.order, f.ws.closed)
	}
	if f.ws.harvests != 0 {
		t.Fatal("harvested without Execute")
	}
	if want := []string{"run1-root-v", "run1-lending-v", "run1-lending-reminders-v", "run1-library-v"}; !slices.Equal(out.Homes, want) {
		t.Fatalf("homes %v", out.Homes)
	}
	for _, tr := range f.agents.turnsOf(VerifyTurn) {
		if !tr.ReadOnly || tr.Workspace != "vol-run1-c0" || tr.Resume != "" || len(tr.Tools) != 1 || tr.Tools[0].Name != toolAssess {
			t.Fatalf("verify turn %+v", tr)
		}
		contains(t, "verify prompt", tr.Prompt, mandate, "/tmp", "blocking", "advisory", "unit: passed", "Decision: three loans.", "Realize the model.")
	}
	if req := f.checks.reqs[0]; req.ID != "run1-r1-unit" || !req.Tree.Equal(f.cfg.Start) || !req.Valid() {
		t.Fatalf("check request %+v", req)
	}
	if out.Usage.InputTokens != 40 || out.Usage.OutputTokens != 20 {
		t.Fatalf("usage %+v", out.Usage)
	}
	for _, kind := range []string{"run.start", "turn.start", "turn.end", "assessment", "check", "round.end", "outcome"} {
		if f.ev.count(kind) == 0 {
			t.Errorf("no %s event", kind)
		}
	}
	if out.Turns[0].ID != "run1-r1-lending-reminders-v1" && out.Turns[0].ID != "run1-r1-library-v1" {
		t.Fatalf("a parent verified first: %v", turnIDs(out))
	}
}

func TestVerifyRunsChildrenBeforeParentsAndBranchesInParallel(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var log []string
	started := map[string]chan struct{}{"library": make(chan struct{}), "lending/reminders": make(chan struct{})}
	note := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	f.agents.script = func(c *call) error {
		m := c.turn.Manager
		note("s:" + m)
		if ch, leaf := started[m]; leaf {
			close(ch)
			// The two independent leaves overlap.
			if err := c.wait(started["library"]); err != nil {
				return err
			}
			if err := c.wait(started["lending/reminders"]); err != nil {
				return err
			}
		}
		defer note("e:" + m)
		return c.defaultScript()
	}
	if out := f.run(); out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	pos := func(s string) int { return slices.Index(log, s) }
	for _, pc := range [][2]string{{"lending/reminders", "lending"}, {"lending", "."}, {"library", "."}} {
		if pos("e:"+pc[0]) > pos("s:"+pc[1]) {
			t.Errorf("%s started before its child %s ended: %v", pc[1], pc[0], log)
		}
	}
}

func TestFindingsDriveParallelExecuteCaptureAndGreen(t *testing.T) {
	f := newFixture(t)
	libStarted, remStarted := make(chan struct{}), make(chan struct{})
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "library composes the program", true)})
		case c.is(1, "lending/reminders", VerifyTurn):
			c.assess(false, []fnd{finding("lending/reminders", "reminders assess overdue loans", true)})
		case c.turn.Kind == ExecuteTurn && c.turn.Manager == "library":
			close(libStarted)
			if err := c.wait(remStarted); err != nil { // both owners work at the same time
				return err
			}
			c.write("library/new.go", "package library")
			return nil // no finish: counts as done
		case c.turn.Kind == ExecuteTurn && c.turn.Manager == "lending/reminders":
			close(remStarted)
			if err := c.wait(libStarted); err != nil {
				return err
			}
			c.write("lending/reminders/new.go", "package reminders")
			c.finish("waiting")
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	for _, p := range []string{"library/new.go", "lending/reminders/new.go"} {
		if _, ok := out.Candidate.Lookup(mustPath(t, p)); !ok {
			t.Errorf("candidate lacks %s", p)
		}
	}
	if countKind(out, ExecuteTurn) != 2 || len(out.Rounds) != 2 || out.Rounds[0].Green || !out.Rounds[1].Green {
		t.Fatalf("turns %v rounds %+v", turnIDs(out), out.Rounds)
	}
	for _, fd := range out.Findings {
		if !fd.Closed || fd.ClosedRound != 2 || fd.ClosedReason != "resolved" {
			t.Errorf("finding %+v", fd)
		}
	}
	status := map[string]string{}
	for _, tr := range out.Turns {
		if tr.Kind == ExecuteTurn {
			status[tr.Manager] = tr.Status
		}
	}
	if status["library"] != "" || status["lending/reminders"] != "waiting" {
		t.Fatalf("statuses %v", status)
	}
	for _, tr := range f.agents.turnsOf(ExecuteTurn) {
		if tr.ReadOnly || tr.Workspace != "vol-run1-work" || tr.Resume != "" || !strings.HasPrefix(tr.ID, "run1-r1-") || !strings.HasSuffix(tr.ID, "-x1") {
			t.Errorf("execute turn %+v", tr)
		}
		contains(t, "execute prompt", tr.Prompt, mandate, "Other Managers work in it at the same time", "Go is available", "Finding F", "parent: ")
	}
	var round2 []string
	for _, tr := range f.agents.turnsOf(VerifyTurn) {
		if strings.Contains(tr.ID, "-r2-") {
			round2 = append(round2, tr.Workspace)
		}
	}
	if len(round2) != 4 || round2[0] != "vol-run1-c1" {
		t.Fatalf("round 2 volumes %v", round2)
	}
	if !slices.Equal(f.ws.order, []string{"run1-c0", "run1-work", "run1-c1"}) || !f.ws.isClosed("run1-c0") || !f.ws.isClosed("run1-c1") || !f.ws.isClosed("run1-work") {
		t.Fatalf("volumes %v closed %v", f.ws.order, f.ws.closed)
	}
	if out.Usage.InputTokens != 10*int64(len(out.Turns)) {
		t.Fatalf("usage %+v for %d turns", out.Usage, len(out.Turns))
	}
	if want := []string{"run1-root-v", "run1-lending-v", "run1-lending-reminders-x", "run1-lending-reminders-v", "run1-library-x", "run1-library-v"}; !slices.Equal(out.Homes, want) {
		t.Fatalf("homes %v", out.Homes)
	}
	if f.ev.count("finding.raised") != 2 || f.ev.count("finding.closed") != 2 || f.ev.count("candidate") != 1 {
		t.Fatalf("events: raised %d closed %d candidate %d", f.ev.count("finding.raised"), f.ev.count("finding.closed"), f.ev.count("candidate"))
	}
}

func TestOwnersStartExecuteInRoundOneWhileOtherVerifyTurnsRun(t *testing.T) {
	f := newFixture(t)
	libExecuting := make(chan struct{})
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "library obligation", true)})
		case c.is(1, "lending", VerifyTurn):
			// Only an early Execute start of library lets this turn end.
			if err := c.wait(libExecuting); err != nil {
				return err
			}
			return c.defaultScript()
		case c.turn.Kind == ExecuteTurn:
			close(libExecuting)
			if !strings.Contains(c.turn.Workspace, "work") {
				t.Errorf("execute in %s", c.turn.Workspace)
			}
			c.write("library/early.go", "x")
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	if out := f.run(); out.Status != Verified || len(out.Rounds) != 2 {
		t.Fatalf("%s %s rounds %d", out.Status, out.Reason, len(out.Rounds))
	}
}

func TestExecuteStaysPausedWhileLaterRoundsVerify(t *testing.T) {
	f := newFixture(t)
	var verifying atomic.Int32
	f.agents.script = func(c *call) error {
		switch {
		case c.turn.Kind == ExecuteTurn:
			if n := verifying.Load(); n != 0 {
				t.Errorf("execute started while %d verify turns run", n)
			}
			c.write("library/x"+c.turn.ID+".go", "x")
			return c.defaultScript()
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "first", true)})
			return nil
		case c.is(2, "library", VerifyTurn):
			verifying.Add(1)
			defer verifying.Add(-1)
			time.Sleep(20 * time.Millisecond)
			c.assess(false, []fnd{finding("library", "second", true)}, c.myOpen()...)
			return nil
		case c.turn.Kind == VerifyTurn && c.round() == 2:
			verifying.Add(1)
			defer verifying.Add(-1)
			time.Sleep(30 * time.Millisecond)
			return c.defaultScript()
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 3 {
		t.Fatalf("%s %s rounds %d", out.Status, out.Reason, len(out.Rounds))
	}
}

func TestMessagesFollowTheTreeAndAnswersCarryNewItems(t *testing.T) {
	f := newFixture(t)
	remSent := make(chan struct{})
	var refusals []string
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "lending", VerifyTurn):
			c.assess(false, []fnd{finding("lending", "lending obligation", true)})
		case c.turn.Manager == "lending" && c.turn.Kind == ExecuteTurn:
			for _, to := range []string{"library", ".", "owner", "nobody"} {
				a := c.send(to, "hello")
				if !a.Refused {
					t.Errorf("send to %q accepted", to)
				}
				refusals = append(refusals, a.Text)
			}
			if a := c.send("lending/reminders", "please add the notice"); a.Refused {
				t.Errorf("send to child refused: %s", a.Text)
			}
			if err := c.wait(remSent); err != nil {
				return err
			}
			a := c.send("lending/reminders", "and once more")
			contains(t, "send answer", a.Text, "Message from your child lending/reminders (overdue notices): need loan status", "New items for you")
			if a.Refused {
				t.Errorf("refused: %s", a.Text)
			}
			if a := c.finish("done"); a.Refused {
				t.Errorf("finish refused")
			}
			if a := c.finish("done"); !a.Refused {
				t.Error("second finish accepted")
			}
			if a := c.send("lending/reminders", "late"); !a.Refused || !strings.Contains(a.Text, "finished") {
				t.Errorf("send after finish: %+v", a)
			}
		case c.turn.Manager == "lending/reminders" && c.turn.Kind == ExecuteTurn:
			if c.turn.Resume == "" {
				contains(t, "first reminders prompt", c.turn.Prompt, "Message from your parent lending (loans): please add the notice")
				if a := c.send("parent", "need loan status"); a.Refused {
					t.Errorf("send to parent refused: %s", a.Text)
				}
				if a := c.send(".", "x"); !a.Refused {
					t.Error("send to grandparent accepted")
				}
				close(remSent)
			} else {
				contains(t, "resumed reminders prompt", c.turn.Prompt, "You continue", "and once more")
				if strings.Contains(c.turn.Prompt, mandate) {
					t.Error("resumed turn repeats the mandate")
				}
				if c.turn.Resume != "conv-run1-r1-lending-reminders-x1" {
					t.Errorf("resume %q", c.turn.Resume)
				}
			}
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	contains(t, "refusal", refusals[0], `"parent"`, `"lending/reminders"`)
	contains(t, "owner refusal", refusals[2], `"parent"`)
	if f.ev.count("message") != 3 {
		t.Fatalf("%d messages recorded", f.ev.count("message"))
	}
	if n := countKind(out, ExecuteTurn); n != 3 {
		t.Fatalf("execute turns %v", turnIDs(out))
	}
}

func TestRootContactsAreChildrenAndOwner(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", VerifyTurn):
			c.assess(false, []fnd{finding(".", "root obligation", true)})
			return nil
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn:
			for _, to := range []string{"lending/reminders", "parent", "."} {
				if a := c.send(to, "x"); !a.Refused {
					t.Errorf("root sent to %q", to)
				}
			}
			a := c.send("library", "build the program")
			if a.Refused {
				t.Errorf("refused: %s", a.Text)
			}
			return c.defaultScript()
		case c.turn.Manager == "library" && c.turn.Kind == ExecuteTurn:
			contains(t, "library prompt", c.turn.Prompt, "Message from your parent . (whole library): build the program")
			return c.defaultScript()
		}
		return c.defaultScript()
	}
	if out := f.run(); out.Status != Verified || countKind(out, ExecuteTurn) != 2 {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
}

func TestOwnerQuestionIsAnsweredAndDeliveredToRoot(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", VerifyTurn):
			c.assess(false, []fnd{finding(".", "root obligation", true)})
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn && c.turn.Resume == "":
			a := c.send("owner", "which loan limit?")
			if a.Refused {
				t.Errorf("refused: %s", a.Text)
			}
			contains(t, "ask answer", a.Text, "Q1")
			time.AfterFunc(60*time.Millisecond, func() { f.owner.answer("Q1", "three") })
			c.finish("waiting")
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn:
			contains(t, "resumed root prompt", c.turn.Prompt, "Owner answer to your question Q1", "which loan limit?", "three")
			return c.defaultScript()
		case c.is(2, ".", VerifyTurn):
			contains(t, "root verify prompt", c.turn.Prompt, "Answered owner questions", "Answer: three")
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if len(out.Questions) != 1 || !out.Questions[0].Answered || out.Questions[0].Answer != "three" || out.Questions[0].Text != "which loan limit?" {
		t.Fatalf("questions %+v", out.Questions)
	}
	if len(f.owner.questions()) != 1 || f.ev.count("question") != 1 || f.ev.count("answer") != 1 {
		t.Fatal("question or answer not recorded")
	}
	if n := countKind(out, ExecuteTurn); n != 2 {
		t.Fatalf("root ran %d execute turns", n)
	}
}

func TestOwnerQuestionWithoutOwnerIsRefused(t *testing.T) {
	f := newFixture(t)
	f.cfg.Owner = nil
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", VerifyTurn):
			c.assess(false, []fnd{finding(".", "root obligation", true)})
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn:
			a := c.send("owner", "which loan limit?")
			if !a.Refused {
				t.Error("question accepted without an owner")
			}
			contains(t, "refusal", a.Text, "owner decisions")
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Verified || len(out.Questions) != 0 {
		t.Fatalf("%s %s %+v", out.Status, out.Reason, out.Questions)
	}
}

func TestUnansweredOwnerQuestionKeepsTheRunIncomplete(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.Deadline = time.Now().Add(400 * time.Millisecond)
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", VerifyTurn):
			c.assess(false, []fnd{finding(".", "root obligation", true)})
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn:
			c.send("owner", "which loan limit?")
			c.finish("waiting")
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "deadline") {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if len(out.Questions) != 1 || out.Questions[0].Answered {
		t.Fatalf("questions %+v", out.Questions)
	}
	if f.ws.harvests != 0 || f.ws.isClosed("run1-work") {
		t.Fatal("the incomplete shared tree must be kept and not captured")
	}
}

func TestOwnerThatCannotBeReachedRefusesTheQuestion(t *testing.T) {
	f := newFixture(t)
	f.owner.askErr = errTest
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", VerifyTurn):
			c.assess(false, []fnd{finding(".", "root obligation", true)})
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn:
			if a := c.send("owner", "q"); !a.Refused {
				t.Error("accepted")
			}
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	if out := f.run(); out.Status != Verified || len(out.Questions) != 0 {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
}

func TestFindingsStayOpenUntilTheRaiserClosesThem(t *testing.T) {
	f := newFixture(t)
	lendingReported := make(chan struct{})
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "lending", VerifyTurn):
			c.assess(false, []fnd{finding("lending/reminders", "reminders obligation", true)})
			close(lendingReported)
		case c.is(1, "library", VerifyTurn):
			if err := c.wait(lendingReported); err != nil {
				return err
			}
			// Invalid reports are refused with a reason and may be repeated.
			bad := []struct {
				name string
				call func() ToolAnswer
				want string
			}{
				{"foreign owner", func() ToolAnswer { return c.assess(false, []fnd{finding("lending", "x", true)}) }, "descendants"},
				{"unknown closure", func() ToolAnswer { return c.assess(true, nil, "F99") }, "not a finding you raised"},
				{"satisfied but blocking", func() ToolAnswer { return c.assess(true, []fnd{finding("library", "x", true)}) }, "satisfied is true"},
				{"unsatisfied without blocking", func() ToolAnswer { return c.assess(false, []fnd{finding("library", "x", false)}) }, "satisfied is false"},
				{"no blocking flag", func() ToolAnswer {
					return c.tool(toolAssess, map[string]any{"summary": "s", "satisfied": true, "findings": []any{map[string]any{"owner": "library", "obligation": "o", "observed": "b"}}})
				}, "blocking is required"},
				{"no summary", func() ToolAnswer { return c.tool(toolAssess, map[string]any{"summary": " ", "satisfied": true}) }, "summary"},
				{"unknown field", func() ToolAnswer {
					return c.tool(toolAssess, map[string]any{"summary": "s", "satisfied": true, "extra": 1})
				}, "not valid"},
				{"unknown tool", func() ToolAnswer { return c.tool("finish", map[string]any{}) }, "Unknown tool"},
			}
			for _, b := range bad {
				a := b.call()
				if !a.Refused || !strings.Contains(a.Text, b.want) {
					t.Errorf("%s: %+v", b.name, a)
				}
			}
			c.assess(true, []fnd{finding("library", "advisory depth", false)})
			if a := c.assess(true, nil); !a.Refused || !strings.Contains(a.Text, "already reported") {
				t.Errorf("second assess %+v", a)
			}
		case c.is(2, "library", VerifyTurn):
			// F1 was raised by lending; library cannot close it.
			if a := c.assess(true, nil, "F1"); !a.Refused {
				t.Error("closed another Manager's finding")
			}
			return c.defaultScript()
		case c.is(2, "lending", VerifyTurn):
			contains(t, "lending prompt", c.turn.Prompt, "Open findings you raised earlier", "Finding F1 [blocking]")
			// A fresh reviewer that does not close it keeps it open.
			c.assess(false, nil)
		case c.is(2, "lending/reminders", VerifyTurn):
			contains(t, "reminders prompt", c.turn.Prompt, "Open findings addressed to you", "Finding F1")
			return c.defaultScript()
		case c.is(3, "lending", VerifyTurn):
			return c.defaultScript() // closes F1
		case c.turn.Kind == ExecuteTurn:
			contains(t, "execute prompt", c.turn.Prompt, "Finding F1 [blocking]")
			c.write("lending/reminders/n"+c.turn.ID, "x")
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 3 {
		t.Fatalf("%s %s rounds %d", out.Status, out.Reason, len(out.Rounds))
	}
	if out.Rounds[1].Green {
		t.Fatal("round 2 is green with an open finding")
	}
	var f1 Finding
	for _, fd := range out.Findings {
		if fd.ID == "F1" {
			f1 = fd
		}
	}
	if !f1.Closed || f1.ClosedRound != 3 || f1.RaisedBy != "lending" || f1.Owner != "lending/reminders" {
		t.Fatalf("F1 %+v", f1)
	}
	if n := countKind(out, ExecuteTurn); n != 2 {
		t.Fatalf("the open finding is delivered again each round: %v", turnIDs(out))
	}
	var advisory Finding
	for _, fd := range out.Findings {
		if !fd.Blocking {
			advisory = fd
		}
	}
	if advisory.RaisedBy != "library" || advisory.Owner != "library" {
		t.Fatalf("advisory finding %+v", advisory)
	}
}

func TestVerifyTurnWithoutAssessmentIsRetriedOnce(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.is(1, "library", VerifyTurn) && strings.HasSuffix(c.turn.ID, "-v1") {
			return nil
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || countKind(out, VerifyTurn) != 5 || !slices.Contains(turnIDs(out), "run1-r1-library-v2") {
		t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
	}
	for _, tr := range out.Turns {
		if tr.ID == "run1-r1-library-v1" && tr.Status != "" {
			t.Fatalf("status %q", tr.Status)
		}
	}
	contains(t, "retry prompt", f.agents.prompt("run1-r1-library-v2"), "ended without a valid assess")

	g := newFixture(t)
	g.agents.script = func(c *call) error {
		if c.is(1, "library", VerifyTurn) {
			return nil
		}
		return c.defaultScript()
	}
	out = g.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "did not report") || len(out.Rounds) != 1 || out.Rounds[0].Green {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if g.ws.harvests != 0 || countKind(out, ExecuteTurn) != 0 {
		t.Fatal("work continued")
	}
}

func TestCheckFailuresAndAccountFindingsGoToTheirOwners(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", "library")
	f.checks.fail = func(r CheckRequest) (CheckReceipt, error, bool) {
		if _, ok := r.Tree.Lookup(mustPath(t, "library/ok.go")); ok {
			return CheckReceipt{}, nil, false
		}
		return failing("unit", 1)(r)
	}
	f.cfg.Account = func(tree snapshot.Tree) []Finding {
		if _, ok := tree.Lookup(mustPath(t, "lending/fixed.go")); ok {
			return nil
		}
		return []Finding{{Owner: "lending", Obligation: "every file is registered", Observed: "lending/stray.go is unexplained"}}
	}
	f.agents.script = func(c *call) error {
		switch {
		case c.turn.Manager == "library" && c.turn.Kind == ExecuteTurn:
			contains(t, "library prompt", c.turn.Prompt, `Required check "unit"`, "FAIL: boom", "raised by markitect")
			c.write("library/ok.go", "x")
		case c.turn.Manager == "lending" && c.turn.Kind == ExecuteTurn:
			contains(t, "lending prompt", c.turn.Prompt, "lending/stray.go is unexplained", "[blocking]")
			c.write("lending/fixed.go", "x")
		case c.is(1, "library", VerifyTurn):
			contains(t, "verify prompt", c.turn.Prompt, "unit: FAILED, exit code 1", "FAIL: boom")
			return c.defaultScript()
		case c.is(1, "lending", VerifyTurn):
			contains(t, "verify prompt", c.turn.Prompt, "Structural findings of your area", "lending/stray.go")
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 2 {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if r := out.Rounds[0]; r.Green || len(r.Checks) != 1 || r.Checks[0].Passed || r.Checks[0].ExitCode != 1 || !strings.Contains(r.Checks[0].Output, "boom") || !out.Rounds[1].Checks[0].Passed {
		t.Fatalf("rounds %+v", out.Rounds)
	}
	if countKind(out, ExecuteTurn) != 2 {
		t.Fatalf("turns %v", turnIDs(out))
	}
	if len(out.Findings) != 2 {
		t.Fatalf("findings %+v", out.Findings)
	}
	for _, fd := range out.Findings {
		if fd.RaisedBy != MarkitectFinding || !fd.Blocking || !fd.Closed || fd.ClosedRound != 2 {
			t.Errorf("finding %+v", fd)
		}
	}
	if f.checks.reqs[1].ID != "run1-r2-unit" {
		t.Fatalf("second check %s", f.checks.reqs[1].ID)
	}
}

func TestWritesOutsideTheRunningManagersAreObserved(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", VerifyTurn):
			c.assess(false, []fnd{finding("library", "library obligation", true)})
		case c.turn.Manager == "library" && c.turn.Kind == ExecuteTurn:
			c.write("library/a.go", "changed")
			c.write("lending/l.go", "changed by someone else")
			return c.defaultScript()
		case c.is(2, "lending", VerifyTurn):
			contains(t, "lending prompt", c.turn.Prompt, "although you ran no Execute turn", "lending/l.go")
			return c.defaultScript()
		case c.is(2, "library", VerifyTurn):
			if strings.Contains(c.turn.Prompt, "although you ran no Execute turn") {
				t.Error("the writer's own area is reported")
			}
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	obs := f.ev.kinds("write.observed")
	if len(obs) != 1 || obs[0].Manager != "lending" {
		t.Fatalf("observations %+v", obs)
	}
}
