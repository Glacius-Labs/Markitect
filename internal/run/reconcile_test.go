package run

import (
	"fmt"
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

func TestUnchangedStartIsVerifiedInRoundOneAfterTheFirstExecutePhase(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", "library")
	var executing, ended atomic.Int32
	var accounted []int32
	f.cfg.Account = func(snapshot.Tree) []Finding {
		accounted = append(accounted, executing.Load(), ended.Load())
		return nil
	}
	f.agents.script = func(c *call) error {
		if c.turn.Kind == ExecuteTurn {
			executing.Add(1)
			defer func() { executing.Add(-1); ended.Add(1) }()
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || !out.Candidate.Equal(f.cfg.Start) {
		t.Fatalf("outcome %s %q", out.Status, out.Reason)
	}
	// Every Manager takes part in Execute, even though nothing changes, and no
	// Account, check or Verify turn runs before the Execute phase has ended.
	for _, m := range teamPaths {
		if executeTurnsOf(out, m, 1) == 0 {
			t.Errorf("%s ran no Execute turn", m)
		}
	}
	if len(accounted) != 2 || accounted[0] != 0 || accounted[1] < 4 {
		t.Fatalf("Account saw (running, ended) Execute turns %v", accounted)
	}
	if out.Turns[0].ID != "run1-r1-root-x1" || countKind(out, VerifyTurn) != 4 {
		t.Fatalf("turns %v", turnIDs(out))
	}
	lastExecute, firstVerify := -1, len(out.Turns)
	for i, tr := range out.Turns {
		if tr.Kind == ExecuteTurn {
			lastExecute = i
		} else {
			firstVerify = min(firstVerify, i)
		}
	}
	if lastExecute > firstVerify {
		t.Fatalf("a Verify turn ended before an Execute turn: %v", turnIDs(out))
	}
	if len(out.Rounds) != 1 || !out.Rounds[0].Green || len(out.Rounds[0].Assessments) != 4 || len(out.Rounds[0].Checks) != 1 || !out.Rounds[0].Checks[0].Passed {
		t.Fatalf("rounds %+v", out.Rounds)
	}
	// There is no candidate 0: the work volume holds the start and the first
	// candidate is the captured tree, which equals the start here.
	if !slices.Equal(f.ws.order, []string{"run1-work", "run1-c1"}) || !f.ws.isClosed("run1-c1") || f.ws.isClosed("run1-work") || f.ws.harvests != 1 {
		t.Fatalf("volumes %v closed %v harvests %d: the caller owns the work volume", f.ws.order, f.ws.closed, f.ws.harvests)
	}
	if want := []string{"run1-root-x", "run1-root-v", "run1-lending-x", "run1-lending-v", "run1-lending-reminders-x", "run1-lending-reminders-v", "run1-library-x", "run1-library-v"}; !slices.Equal(out.Homes, want) {
		t.Fatalf("homes %v", out.Homes)
	}
	for _, tr := range f.agents.turnsOf(VerifyTurn) {
		if !tr.ReadOnly || tr.Workspace != "vol-run1-c1" || tr.Resume != "" || len(tr.Tools) != 1 || tr.Tools[0].Name != toolAssess {
			t.Fatalf("verify turn %+v", tr)
		}
		contains(t, "verify prompt", tr.Prompt, mandate, "/tmp", "blocking", "advisory", "unit: passed", "Decision: three loans.", "Realize the model.")
	}
	for _, tr := range f.agents.turnsOf(ExecuteTurn) {
		if tr.ReadOnly || tr.Workspace != "vol-run1-work" {
			t.Fatalf("execute turn %+v", tr)
		}
	}
	if req := f.checks.reqs[0]; req.ID != "run1-r1-unit" || !req.Tree.Equal(f.cfg.Start) || !req.Valid() {
		t.Fatalf("check request %+v", req)
	}
	if out.Usage.InputTokens != 10*int64(len(out.Turns)) {
		t.Fatalf("usage %+v for %d turns", out.Usage, len(out.Turns))
	}
	for _, kind := range []string{"run.start", "manager.release", "turn.start", "turn.end", "turn.finish", "candidate", "assessment", "check", "round.end", "outcome"} {
		if f.ev.count(kind) == 0 {
			t.Errorf("no %s event", kind)
		}
	}
	if ev := f.ev.kinds("candidate"); len(ev) != 1 || ev[0].Round != 1 {
		t.Fatalf("candidate events %+v", ev)
	}
	if out.Turns[len(out.Turns)-1].Manager != "." || out.Turns[len(out.Turns)-1].Kind != VerifyTurn {
		t.Fatalf("Root verified before its children: %v", turnIDs(out))
	}
}

func TestExecuteChangesBecomeTheCandidateOfRoundOne(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", "library")
	f.agents.script = func(c *call) error {
		if c.is(1, "library", ExecuteTurn) {
			c.write("library/new.go", "package library")
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 1 || out.Candidate.Equal(f.cfg.Start) {
		t.Fatalf("%s %s rounds %d", out.Status, out.Reason, len(out.Rounds))
	}
	if _, ok := out.Candidate.Lookup(mustPath(t, "library/new.go")); !ok {
		t.Fatal("candidate lacks the new file")
	}
	if req := f.checks.reqs[0]; !req.Tree.Equal(out.Candidate) {
		t.Fatal("the check did not run on the candidate")
	}
	if f.ev.count("write.observed") != 0 {
		t.Fatal("every Manager ran, so nothing is observed")
	}
}

func TestVerifyRunsChildrenBeforeParentsAndBranchesInParallel(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var log []string
	started := map[string]chan struct{}{"library": make(chan struct{}), "lending/reminders": make(chan struct{})}
	note := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	f.agents.script = func(c *call) error {
		if c.turn.Kind != VerifyTurn {
			return c.defaultScript()
		}
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

func TestChildrenStartAfterTheirParentsFirstTurnAndSiblingsRunInParallel(t *testing.T) {
	f := newFixture(t)
	f.agents.quiet = true
	var mu sync.Mutex
	var log []string
	note := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	libStarted, lendStarted := make(chan struct{}), make(chan struct{})
	f.agents.script = func(c *call) error {
		if c.turn.Kind != ExecuteTurn {
			return c.defaultScript()
		}
		m := c.turn.Manager
		note("start:" + m)
		defer note("end:" + m)
		switch m {
		case ".":
			// A message to a child that has not started is queued for its first turn.
			for to, text := range map[string]string{"lending": "the module exists", "library": "use the module"} {
				if a := c.send(to, text); a.Refused {
					t.Errorf("send to %s refused: %s", to, a.Text)
				}
			}
			time.Sleep(30 * time.Millisecond)
		case "lending":
			close(lendStarted)
			// The siblings run at the same time.
			if err := c.wait(libStarted); err != nil {
				return err
			}
			time.Sleep(30 * time.Millisecond)
		case "library":
			close(libStarted)
			if err := c.wait(lendStarted); err != nil {
				return err
			}
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || countKind(out, ExecuteTurn) != 4 {
		t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
	}
	mu.Lock()
	defer mu.Unlock()
	if log[0] != "start:." {
		t.Fatalf("Root does not start first: %v", log)
	}
	before := func(a, b string) {
		t.Helper()
		if i, j := slices.Index(log, a), slices.Index(log, b); i < 0 || j < 0 || i > j {
			t.Errorf("%s must come before %s: %v", a, b, log)
		}
	}
	before("end:.", "start:lending")
	before("end:.", "start:library")
	before("end:lending", "start:lending/reminders")

	// One release per Manager, recorded with the parent that released it.
	released := map[string]any{}
	for _, ev := range f.ev.kinds("manager.release") {
		if _, dup := released[ev.Manager]; dup {
			t.Errorf("release of %s recorded twice", ev.Manager)
		}
		data, _ := ev.Data.(map[string]any)
		released[ev.Manager] = data["parent"]
	}
	if len(released) != 4 || released["."] != nil || released["lending"] != "." || released["library"] != "." || released["lending/reminders"] != "lending" {
		t.Fatalf("release events %v", released)
	}

	// The initial item tells a parent to supply its children's foundation and a
	// child that its parent's foundation is in place; queued messages follow it.
	rootPrompt := f.agents.prompt("run1-r1-root-x1")
	contains(t, "Root prompt", rootPrompt, mandate, "Carry out your responsibility now", "Your children start when this turn ends", "First establish, within your own area, the foundation they need", "assess it against your obligations and your children's known needs", "the concrete basis they can use")
	if strings.Contains(rootPrompt, "Your parent's foundation") {
		t.Errorf("Root has no parent:\n%s", rootPrompt)
	}
	libPrompt := f.agents.prompt("run1-r1-library-x1")
	contains(t, "library prompt", libPrompt, "Your parent's foundation is in place", "Message from your parent . (whole library): use the module")
	if strings.Contains(libPrompt, "Your children start") || strings.Index(libPrompt, "Carry out your responsibility") > strings.Index(libPrompt, "Message from your parent") {
		t.Errorf("library prompt:\n%s", libPrompt)
	}
	contains(t, "lending prompt", f.agents.prompt("run1-r1-lending-x1"), "Your parent's foundation is in place", "Your children start when this turn ends", "the module exists")
	remPrompt := f.agents.prompt("run1-r1-lending-reminders-x1")
	contains(t, "reminders prompt", remPrompt, "Your parent's foundation is in place")
	if strings.Contains(remPrompt, "Your children start") {
		t.Errorf("reminders has no children:\n%s", remPrompt)
	}
}

func TestChildrenAreReleasedWhenTheirParentsFirstTurnFailsOrTheParentIsDisabled(t *testing.T) {
	for name, failures := range map[string]int{"first turn fails once": 1, "parent disabled": 2} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.agents.quiet = true
			var mu sync.Mutex
			var log []string
			note := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
			var rootTurns atomic.Int32
			f.agents.script = func(c *call) error {
				if c.turn.Kind == ExecuteTurn && c.turn.Manager == "." {
					note("start:.")
					defer note("end:.")
					time.Sleep(30 * time.Millisecond)
					if int(rootTurns.Add(1)) <= failures {
						return errTest
					}
				} else if c.turn.Kind == ExecuteTurn {
					note("start:" + c.turn.Manager)
				}
				return c.defaultScript()
			}
			out := f.run()
			if out.Status != Verified {
				t.Fatalf("%s %s", out.Status, out.Reason)
			}
			mu.Lock()
			defer mu.Unlock()
			end := slices.Index(log, "end:.")
			for _, m := range []string{"lending", "library", "lending/reminders"} {
				if i := slices.Index(log, "start:"+m); i < 0 || i < end {
					t.Errorf("%s must start after the first turn of Root ended: %v", m, log)
				}
				if executeTurnsOf(out, m, 1) != 1 {
					t.Errorf("%s ran %d Execute turns", m, executeTurnsOf(out, m, 1))
				}
			}
			if n := executeTurnsOf(out, ".", 1); n != 2 {
				t.Errorf("Root ran %d Execute turns", n)
			}
			if disabled := f.ev.count("manager.disabled"); disabled != failures-1 {
				t.Errorf("%d disabled events", disabled)
			}
		})
	}
}

func TestParentIntegratesItsQuietSubtreeInOneTurn(t *testing.T) {
	f := newFixture(t)
	libraryReported, remindersStarted := make(chan struct{}), make(chan struct{})
	f.ev.onRecord = func(e Event) {
		if e.Kind == "child.report" && e.Manager == "library" {
			close(libraryReported)
		}
	}
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "lending", ExecuteTurn) && c.turn.Resume == "":
			// Lending's first turn outlasts the library's report to Root.
			if err := c.wait(libraryReported); err != nil {
				return err
			}
		case c.is(1, "lending/reminders", ExecuteTurn):
			close(remindersStarted)
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	// Root's first turn, then one turn with every report, after lending had
	// integrated reminders; no turn per report.
	if n := executeTurnsOf(out, ".", 1); n != 2 {
		t.Fatalf("Root ran %d Execute turns: %v", n, turnIDs(out))
	}
	contains(t, "Root's integration prompt", f.agents.prompt("run1-r1-root-x2"),
		"Your child library (program) reports done", "Your child lending (loans) reports done")
	select {
	case <-remindersStarted:
	default:
		t.Fatal("reminders never ran")
	}
	var rootStart, lendingEnd int
	for i, ev := range f.ev.events {
		switch {
		case ev.Kind == "turn.start" && ev.Turn == "run1-r1-root-x2":
			rootStart = i
		case ev.Kind == "turn.end" && ev.Turn == "run1-r1-lending-x2":
			lendingEnd = i
		}
	}
	if rootStart == 0 || lendingEnd == 0 || rootStart < lendingEnd {
		t.Fatalf("Root's integration turn started at event %d, before lending's integration ended at %d", rootStart, lendingEnd)
	}
}

func TestDoneSummariesOfChildrenReachTheirParent(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Kind == ExecuteTurn && c.turn.Manager != "." {
			status := "done"
			if c.turn.Manager == "library" {
				status = "waiting"
			}
			c.tool(toolFinish, map[string]any{"status": status, "summary": "summary of " + c.turn.Manager})
			return nil
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	// Reminders reported done to Lending, which ran again with that report.
	if n := executeTurnsOf(out, "lending", 1); n != 2 {
		t.Fatalf("lending ran %d Execute turns: %v", n, turnIDs(out))
	}
	contains(t, "second lending prompt", f.agents.prompt("run1-r1-lending-x2"), "You continue", "Your child lending/reminders (overdue notices) reports done: summary of lending/reminders")
	// Lending reported done to Root; library only waited and reported nothing.
	var rootItems strings.Builder
	for _, tr := range f.agents.turnsOf(ExecuteTurn) {
		if tr.Manager == "." {
			rootItems.WriteString(tr.Prompt)
		}
	}
	contains(t, "Root prompts", rootItems.String(), "Your child lending (loans) reports done: summary of lending")
	if strings.Contains(rootItems.String(), "library (program) reports done") {
		t.Error("a waiting child reported done")
	}
	if reports := f.ev.kinds("child.report"); len(reports) < 2 {
		t.Fatalf("child.report events %+v", reports)
	}
}

func TestFindingsDriveParallelRepairCaptureAndGreen(t *testing.T) {
	f := newFixture(t)
	libStarted, remStarted := make(chan struct{}), make(chan struct{})
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "library composes the program", true)})
		case c.is(1, "lending/reminders", VerifyTurn):
			c.assess(false, []fnd{finding("lending/reminders", "reminders assess overdue loans", true)})
		case c.is(2, "library", ExecuteTurn):
			close(libStarted)
			if err := c.wait(remStarted); err != nil { // both owners work at the same time
				return err
			}
			c.write("library/new.go", "package library")
			return nil // no finish: counts as done
		case c.is(2, "lending/reminders", ExecuteTurn):
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
	// The repair phase runs the two owners and Root: the library ended without a
	// finish call, which counts as done and reports to Root, while reminders
	// waited and reported nothing to lending.
	if executeTurnsOf(out, "library", 2) != 1 || executeTurnsOf(out, "lending/reminders", 2) != 1 || executeTurnsOf(out, "lending", 2) != 0 || executeTurnsOf(out, ".", 2) != 1 {
		t.Fatalf("turns %v", turnIDs(out))
	}
	rootRepair := ""
	for _, tr := range out.Turns {
		if tr.Manager == "." && tr.Kind == ExecuteTurn && tr.Round == 2 {
			rootRepair = tr.ID
		}
	}
	if prompt := f.agents.prompt(rootRepair); !strings.Contains(prompt, "library (program) reports done: no summary was given") {
		t.Errorf("Root's repair prompt %q", prompt)
	}
	if len(out.Rounds) != 2 || out.Rounds[0].Green || !out.Rounds[1].Green {
		t.Fatalf("rounds %+v", out.Rounds)
	}
	for _, fd := range out.Findings {
		if !fd.Closed || fd.ClosedRound != 2 || fd.ClosedReason != "resolved" {
			t.Errorf("finding %+v", fd)
		}
	}
	status := map[string]string{}
	for _, tr := range out.Turns {
		if tr.Kind == ExecuteTurn && tr.Round == 2 {
			status[tr.Manager] = tr.Status
		}
	}
	if status["library"] != "" || status["lending/reminders"] != "waiting" {
		t.Fatalf("statuses %v", status)
	}
	for _, tr := range f.agents.turnsOf(ExecuteTurn) {
		if tr.ReadOnly || tr.Workspace != "vol-run1-work" {
			t.Errorf("execute turn %+v", tr)
		}
		switch {
		case tr.Resume == "":
			contains(t, "initial prompt", tr.Prompt, mandate, "Other Managers work in it at the same time", "Run the project's build and tests yourself", "Carry out your responsibility now", "Your contacts for the tool send")
			if !strings.HasPrefix(tr.ID, "run1-r1-") || !strings.HasSuffix(tr.ID, "-x1") {
				t.Errorf("first turn %+v", tr)
			}
		case strings.HasPrefix(tr.ID, "run1-r2-"):
			if tr.Manager == "." {
				contains(t, "Root's report prompt", tr.Prompt, "You continue", "reports done")
				continue
			}
			contains(t, "repair prompt", tr.Prompt, "You continue", "Finding F")
			if !strings.HasSuffix(tr.ID, "-x2") {
				t.Errorf("repair turn %+v", tr)
			}
		}
	}
	var round2 []string
	for _, tr := range f.agents.turnsOf(VerifyTurn) {
		if strings.Contains(tr.ID, "-r2-") {
			round2 = append(round2, tr.Workspace)
		}
	}
	if len(round2) != 4 || round2[0] != "vol-run1-c2" {
		t.Fatalf("round 2 volumes %v", round2)
	}
	if !slices.Equal(f.ws.order, []string{"run1-work", "run1-c1", "run1-c2"}) || !f.ws.isClosed("run1-c1") || !f.ws.isClosed("run1-c2") || f.ws.isClosed("run1-work") {
		t.Fatalf("volumes %v closed %v: the caller owns the work volume", f.ws.order, f.ws.closed)
	}
	if out.Usage.InputTokens != 10*int64(len(out.Turns)) {
		t.Fatalf("usage %+v for %d turns", out.Usage, len(out.Turns))
	}
	if f.ev.count("finding.raised") != 2 || f.ev.count("finding.closed") != 2 || f.ev.count("candidate") != 2 {
		t.Fatalf("events: raised %d closed %d candidate %d", f.ev.count("finding.raised"), f.ev.count("finding.closed"), f.ev.count("candidate"))
	}
}

func TestExecuteStaysPausedWhileVerifyRuns(t *testing.T) {
	f := newFixture(t)
	var verifying atomic.Int32
	f.agents.script = func(c *call) error {
		if c.turn.Kind == VerifyTurn {
			verifying.Add(1)
			defer verifying.Add(-1)
			time.Sleep(10 * time.Millisecond)
		}
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
			c.assess(false, []fnd{finding("library", "second", true)}, c.myOpen()...)
			return nil
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
	f.agents.quiet = true
	lendingAgain, remSent := make(chan struct{}), make(chan struct{})
	var refusals []string
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "lending", ExecuteTurn) && strings.HasSuffix(c.turn.ID, "-x1"):
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
			return c.defaultScript()
		case c.turn.Manager == "lending" && c.turn.Kind == ExecuteTurn:
			// The second turn starts with the message of the child.
			contains(t, "resumed lending prompt", c.turn.Prompt, "You continue", "Message from your child lending/reminders (overdue notices): need loan status")
			close(lendingAgain)
			if err := c.wait(remSent); err != nil {
				return err
			}
			a := c.send("lending/reminders", "and once more")
			contains(t, "send answer", a.Text, "Message from your child lending/reminders (overdue notices): second question", "New items for you")
			if a.Refused {
				t.Errorf("refused: %s", a.Text)
			}
			// The last finish of a turn counts.
			if a := c.finish("waiting"); a.Refused {
				t.Errorf("finish refused")
			}
			if a := c.finish("done"); a.Refused {
				t.Errorf("second finish refused: %+v", a)
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
				if err := c.wait(lendingAgain); err != nil {
					return err
				}
				// Lending runs: the message is queued for its running turn.
				if a := c.send("parent", "second question"); a.Refused {
					t.Errorf("send to parent refused: %s", a.Text)
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
	if f.ev.count("message") != 4 {
		t.Fatalf("%d messages recorded", f.ev.count("message"))
	}
	if n, m := executeTurnsOf(out, "lending", 1), executeTurnsOf(out, "lending/reminders", 1); n != 2 || m != 2 {
		t.Fatalf("execute turns %v", turnIDs(out))
	}
	for _, tr := range out.Turns {
		if tr.ID == "run1-r1-lending-x2" && tr.Status != "done" {
			t.Errorf("lending's second turn ended %q, not with its last finish", tr.Status)
		}
	}
}

func TestRootContactsAreChildrenAndOwner(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", ExecuteTurn) && c.turn.Resume == "":
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
		case c.is(1, "library", ExecuteTurn):
			contains(t, "library prompt", c.turn.Prompt, "Message from your parent . (whole library): build the program")
		}
		return c.defaultScript()
	}
	if out := f.run(); out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
}

func TestOwnerQuestionIsAnsweredAndDeliveredToRoot(t *testing.T) {
	f := newFixture(t)
	var delivered atomic.Bool
	f.agents.script = func(c *call) error {
		switch {
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn && c.turn.Resume == "":
			a := c.send("owner", "which loan limit?")
			if a.Refused {
				t.Errorf("refused: %s", a.Text)
			}
			contains(t, "ask answer", a.Text, "Q1")
			time.AfterFunc(60*time.Millisecond, func() { f.owner.answer("Q1", "three") })
			c.finish("waiting")
		case c.turn.Manager == "." && c.turn.Kind == ExecuteTurn:
			if strings.Contains(c.turn.Prompt, "Owner answer to your question Q1") {
				contains(t, "resumed root prompt", c.turn.Prompt, "which loan limit?", "three")
				delivered.Store(true)
			}
			return c.defaultScript()
		case c.is(1, ".", VerifyTurn):
			contains(t, "root verify prompt", c.turn.Prompt, "Answered owner questions", "Answer: three")
			return c.defaultScript()
		default:
			return c.defaultScript()
		}
		return nil
	}
	out := f.run()
	if out.Status != Verified || !delivered.Load() {
		t.Fatalf("%s %s delivered %v", out.Status, out.Reason, delivered.Load())
	}
	if len(out.Questions) != 1 || !out.Questions[0].Answered || out.Questions[0].Answer != "three" || out.Questions[0].Text != "which loan limit?" {
		t.Fatalf("questions %+v", out.Questions)
	}
	if len(f.owner.questions()) != 1 || f.ev.count("question") != 1 || f.ev.count("answer") != 1 {
		t.Fatal("question or answer not recorded")
	}
}

func TestOwnerQuestionWithoutOwnerIsRefused(t *testing.T) {
	f := newFixture(t)
	f.cfg.Owner = nil
	f.agents.script = func(c *call) error {
		if c.is(1, ".", ExecuteTurn) && c.turn.Resume == "" {
			a := c.send("owner", "which loan limit?")
			if !a.Refused {
				t.Error("question accepted without an owner")
			}
			contains(t, "refusal", a.Text, "owner decisions")
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Questions) != 0 {
		t.Fatalf("%s %s %+v", out.Status, out.Reason, out.Questions)
	}
}

func TestRootEndsTheRunWithAModelDefect(t *testing.T) {
	toolNames := func(turn Turn) (names []string) {
		for _, tool := range turn.Tools {
			names = append(names, tool.Name)
		}
		return names
	}
	t.Run("offered to Root only", func(t *testing.T) {
		f := newFixture(t)
		f.cfg.DefectTool = true
		f.agents.script = func(c *call) error {
			if c.turn.Kind == ExecuteTurn {
				offered := slices.Contains(toolNames(c.turn), toolDefect)
				if offered != (c.turn.Manager == ".") {
					t.Errorf("%s: model_defect offered = %v", c.turn.ID, offered)
				}
				if c.turn.Manager == "library" {
					if a := c.tool(toolDefect, map[string]any{"text": "x"}); !a.Refused || !strings.Contains(a.Text, "Unknown tool") {
						t.Errorf("a child reported a model defect: %+v", a)
					}
				}
			}
			return c.defaultScript()
		}
		if out := f.run(); out.Status != Verified {
			t.Fatalf("%s %s", out.Status, out.Reason)
		}
	})
	t.Run("not offered without the option", func(t *testing.T) {
		f := newFixture(t)
		f.agents.script = func(c *call) error {
			if slices.Contains(toolNames(c.turn), toolDefect) {
				t.Errorf("%s: model_defect offered without DefectTool", c.turn.ID)
			}
			return c.defaultScript()
		}
		f.run()
	})
	t.Run("the report ends the run", func(t *testing.T) {
		// As in Apply: no owner channel, the model defect is Root's only way out.
		f := newFixture(t)
		f.cfg.DefectTool, f.cfg.Owner = true, nil
		f.agents.script = func(c *call) error {
			if c.is(1, ".", ExecuteTurn) {
				contains(t, "Root's prompt", c.turn.Prompt, "A defect of the model itself goes to the owner only through the tool model_defect, which ends the run.")
				if strings.Contains(c.turn.Prompt, "owner (the human") {
					t.Error("Root's contacts offer the owner without an owner channel")
				}
				for _, tool := range c.turn.Tools {
					if tool.Name == toolSend && strings.Contains(tool.Description, "owner") {
						t.Errorf("send offers the owner: %s", tool.Description)
					}
				}
				if a := c.send("owner", "which limit?"); !a.Refused || !strings.Contains(a.Text, "model_defect") {
					t.Errorf("a question to the owner was not refused towards model_defect: %+v", a)
				}
				if a := c.tool(toolDefect, map[string]any{"text": "lending/loan-policy says four and five"}); a.Refused {
					t.Errorf("model defect refused: %s", a.Text)
				}
				return nil
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Incomplete || out.Reason != ModelDefectReason+"lending/loan-policy says four and five" {
			t.Fatalf("%s %q", out.Status, out.Reason)
		}
		if len(out.Rounds) != 0 || countKind(out, VerifyTurn) != 0 || f.ws.harvests != 0 {
			t.Fatalf("the run went on after the report: %v", turnIDs(out))
		}
		if events := f.ev.kinds("model.defect"); len(events) != 1 || events[0].Manager != "." {
			t.Fatalf("model.defect events %+v", events)
		}
	})
}

func TestUnansweredOwnerQuestionKeepsTheRunIncomplete(t *testing.T) {
	f := newFixture(t)
	f.cfg.Budget.Deadline = time.Now().Add(400 * time.Millisecond)
	f.agents.script = func(c *call) error {
		if c.is(1, ".", ExecuteTurn) && c.turn.Resume == "" {
			c.send("owner", "which loan limit?")
			c.finish("waiting")
			return nil
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Incomplete || !strings.Contains(out.Reason, "deadline") {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if len(out.Questions) != 1 || out.Questions[0].Answered {
		t.Fatalf("questions %+v", out.Questions)
	}
	if f.ws.harvests != 0 || f.ws.isClosed("run1-work") || len(out.Rounds) != 0 {
		t.Fatal("the incomplete shared tree must be kept and not captured")
	}
}

func TestOwnerThatCannotBeReachedRefusesTheQuestion(t *testing.T) {
	f := newFixture(t)
	f.owner.askErr = errTest
	f.agents.script = func(c *call) error {
		if c.is(1, ".", ExecuteTurn) && c.turn.Resume == "" {
			if a := c.send("owner", "q"); !a.Refused {
				t.Error("accepted")
			}
		}
		return c.defaultScript()
	}
	if out := f.run(); out.Status != Verified || len(out.Questions) != 0 {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
}

func TestEarlierAnsweredQuestionsContinueTheOwnerChannel(t *testing.T) {
	f := newFixture(t)
	// Q1 was refused in the earlier run and kept its number.
	f.cfg.Questions = []Question{{ID: "Q2", Text: "which loan limit?", Answer: "three", Answered: true}}
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, ".", ExecuteTurn) && c.turn.Resume == "":
			contains(t, "root execute prompt", c.turn.Prompt, "Answered owner questions", "which loan limit?", "Answer: three")
			a := c.send("owner", "which loan period?")
			contains(t, "ask answer", a.Text, "Q3")
			time.AfterFunc(60*time.Millisecond, func() { f.owner.answer("Q3", "fourteen days") })
			c.finish("waiting")
			return nil
		case c.is(1, ".", VerifyTurn):
			contains(t, "root verify prompt", c.turn.Prompt, "Answer: three", "Answer: fourteen days")
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Questions) != 2 || out.Questions[1].ID != "Q3" || !out.Questions[1].Answered {
		t.Fatalf("%s %s %+v", out.Status, out.Reason, out.Questions)
	}
	if asked := f.owner.questions(); len(asked) != 1 || asked[0].ID != "Q3" {
		t.Fatalf("the owner was asked %+v", asked)
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
			// A later valid report replaces an earlier one of the same turn; only the
			// last one is applied when the turn ends.
			c.assess(true, []fnd{finding("library", "placeholder", false)})
			if a := c.assess(true, []fnd{finding("library", "advisory depth", false)}); a.Refused || !strings.Contains(a.Text, "replaces") {
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
		case c.turn.Kind == ExecuteTurn && c.turn.Manager == "lending/reminders" && c.round() >= 2:
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
	if n := executeTurnsOf(out, "lending/reminders", 2); n != 2 {
		t.Fatalf("the open finding is delivered again each round: %v", turnIDs(out))
	}
	var advisory []Finding
	for _, fd := range out.Findings {
		if !fd.Blocking {
			advisory = append(advisory, fd)
		}
	}
	if len(advisory) != 1 || advisory[0].RaisedBy != "library" || advisory[0].Owner != "library" || advisory[0].Obligation != "advisory depth" {
		t.Fatalf("advisory findings %+v", advisory)
	}
}

func TestItemsOfFindingsClosedByTheRaiserLapse(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "lending", VerifyTurn):
			c.assess(false, []fnd{finding("lending/reminders", "reminders obligation", true)})
		case c.is(1, "library", VerifyTurn):
			c.assess(false, []fnd{finding("library", "first", true)})
		case c.is(2, "library", VerifyTurn):
			// lending closes F1 in the same round; library keeps the round red.
			c.assess(false, []fnd{finding("library", "second", true)}, c.myOpen()...)
		case c.turn.Kind == ExecuteTurn && c.round() >= 2:
			c.write("library/x"+c.turn.ID+".go", "x")
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
	// Reminders repaired once, after round 1. F1 was closed in round 2 after
	// its item for the next phase was delivered, so no second turn follows.
	if n := executeTurnsOf(out, "lending/reminders", 2); n != 1 {
		t.Fatalf("a closed finding triggers an Execute turn: %v", turnIDs(out))
	}
	for _, tr := range out.Turns {
		if tr.Manager == "lending/reminders" && tr.Kind == ExecuteTurn && tr.Round >= 2 && !strings.HasPrefix(tr.ID, "run1-r2-") {
			t.Fatalf("reminders repaired in %s", tr.ID)
		}
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
	for _, tr := range g.agents.turnsOf(ExecuteTurn) {
		if !strings.HasPrefix(tr.ID, "run1-r1-") {
			t.Fatalf("work continued: %s", tr.ID)
		}
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
		case c.is(2, "library", ExecuteTurn):
			contains(t, "library prompt", c.turn.Prompt, `Required check "unit"`, "FAIL: boom", "raised by markitect")
			c.write("library/ok.go", "x")
		case c.is(2, "lending", ExecuteTurn):
			contains(t, "lending prompt", c.turn.Prompt, "lending/stray.go is unexplained", "[blocking]")
			c.write("lending/fixed.go", "x")
		case c.is(1, "library", VerifyTurn):
			contains(t, "verify prompt", c.turn.Prompt, "unit: FAILED, exit code 1", "FAIL: boom")
		case c.is(1, "lending", VerifyTurn):
			contains(t, "verify prompt", c.turn.Prompt, "Structural findings of your area", "lending/stray.go")
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
	if executeTurnsOf(out, "library", 2) != 1 || executeTurnsOf(out, "lending", 2) != 1 || executeTurnsOf(out, "lending/reminders", 2) != 0 {
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
		case c.is(2, "library", ExecuteTurn):
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

func TestEnforcedWritePlansSetMountsAndReadOnlyPerExecuteTurn(t *testing.T) {
	f := newFixture(t)
	f.cfg.Team[0].Write = WritePlan{Enforced: true, Mounts: []PathMount{{Path: "tools", Writable: true}}}
	f.cfg.Team[1].Write = WritePlan{Enforced: true, Base: true, Mounts: []PathMount{{Path: "lending/reminders", Writable: false}}}
	f.cfg.Team[2].Write = WritePlan{Enforced: true, Mounts: []PathMount{{Path: "lending/reminders", Writable: true}}}
	// An unenforced plan keeps the fully writable workspace and adds no directory.
	f.cfg.Team[3].Write = WritePlan{Mounts: []PathMount{{Path: "ignored", Writable: true}}}
	files := map[string]string{".": "tools/gen.go", "lending": "lending/n.go", "lending/reminders": "lending/reminders/n.go", "library": "library/n.go"}
	f.agents.script = func(c *call) error {
		if c.turn.Kind == ExecuteTurn {
			c.write(files[c.turn.Manager], "x")
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 1 {
		t.Fatalf("%s %s %v", out.Status, out.Reason, turnIDs(out))
	}
	want := map[string]struct {
		readOnly bool
		mounts   []PathMount
		note     string
	}{
		".":                 {true, []PathMount{{Path: "tools", Writable: true}}, "Outside your writable areas /work is read-only and a write there fails. Your writable areas: tools."},
		"lending":           {false, []PathMount{{Path: "lending/reminders"}}, "These areas of /work are read-only and a write there fails: lending/reminders."},
		"lending/reminders": {true, []PathMount{{Path: "lending/reminders", Writable: true}}, "Your writable areas: lending/reminders."},
		"library":           {false, nil, ""},
	}
	for _, tr := range f.agents.turnsOf(ExecuteTurn) {
		w := want[tr.Manager]
		if tr.ReadOnly != w.readOnly || !slices.Equal(tr.Mounts, w.mounts) {
			t.Errorf("%s: read-only %t mounts %v, want %t %v", tr.Manager, tr.ReadOnly, tr.Mounts, w.readOnly, w.mounts)
		}
		if tr.Resume != "" {
			continue // later turns resume the conversation and repeat no note
		}
		if w.note != "" {
			contains(t, tr.Manager+" prompt", tr.Prompt, w.note, "/tmp")
		} else if strings.Contains(tr.Prompt, "read-only") {
			t.Errorf("unenforced prompt mentions read-only:\n%s", tr.Prompt)
		}
	}
	for _, tr := range f.agents.turnsOf(VerifyTurn) {
		if !tr.ReadOnly || len(tr.Mounts) != 0 {
			t.Errorf("verify turn %s: read-only %t mounts %v", tr.ID, tr.ReadOnly, tr.Mounts)
		}
	}
	if got := f.ws.dirs["run1-work"]; !slices.Equal(got, []string{"lending/reminders", "tools"}) {
		t.Errorf("work volume directories %v", got)
	}
	if got := f.ws.dirs["run1-c1"]; len(got) != 0 {
		t.Errorf("candidate volume has directories %v", got)
	}
}

func TestWriteNoteNamesReadOnlyMountsAndTheWritableOverlaysInsideThem(t *testing.T) {
	root := WritePlan{Enforced: true, Base: true, Mounts: []PathMount{
		{Path: ".markitect"}, {Path: ".markitect/registry", Writable: true}, {Path: ".markitect/registry/lending"}, {Path: "realization/lending"},
	}}
	contains(t, "Root note", writeNote(root),
		"read-only and a write there fails: .markitect, .markitect/registry/lending, realization/lending",
		"Writable again within them: .markitect/registry")
	if got := writeNote(WritePlan{Enforced: true, Base: true, Mounts: []PathMount{{Path: "a"}}}); strings.Contains(got, "Writable again") {
		t.Errorf("no writable overlay to name: %q", got)
	}
}

func TestChildTurnWithoutFinishReportsDoneToItsParent(t *testing.T) {
	f := newFixture(t)
	f.agents.quiet = true
	f.agents.script = func(c *call) error {
		if c.is(1, "lending/reminders", ExecuteTurn) {
			return nil // no finish call
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	contains(t, "lending's report prompt", f.agents.prompt("run1-r1-lending-x2"), "Your child lending/reminders (overdue notices) reports done: no summary was given")
	reports := f.ev.kinds("child.report")
	if len(reports) != 1 || reports[0].Manager != "lending/reminders" {
		t.Fatalf("child.report events %+v", reports)
	}
}

func TestStartFindingsAreTheOwnersFirstWork(t *testing.T) {
	f := newFixture(t)
	f.cfg.StartFindings = []Finding{
		{Owner: "lending/reminders", Obligation: "file.unexplained", Observed: "lending/reminders/notes.txt is not registered"},
		{Owner: "gone", Obligation: "registry.unknown_owner", Observed: "a shard of a removed Package"},
	}
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "lending/reminders", ExecuteTurn):
			contains(t, "reminders' first prompt", c.turn.Prompt, "Carry out your responsibility now", "Finding F1 [blocking], owner lending/reminders, raised by markitect in round 0.", "lending/reminders/notes.txt is not registered")
			if strings.Index(c.turn.Prompt, "Finding F1") < strings.Index(c.turn.Prompt, "Carry out your responsibility now") {
				t.Error("the start finding comes before the initial item")
			}
		case c.is(1, ".", ExecuteTurn) && c.turn.Resume == "":
			contains(t, "Root's first prompt", c.turn.Prompt, "Finding F2 [blocking], owner .", "reported for unknown owner \"gone\"")
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 1 {
		t.Fatalf("%s %s rounds %d", out.Status, out.Reason, len(out.Rounds))
	}
	// Like every structural finding they hold for the stand they were observed on:
	// round 1 observes the candidate afresh.
	for _, fd := range out.Findings {
		if fd.Round != 0 || fd.RaisedBy != MarkitectFinding || !fd.Blocking || !fd.Closed || fd.ClosedRound != 1 {
			t.Errorf("start finding %+v", fd)
		}
	}
	if len(out.Findings) != 2 {
		t.Fatalf("findings %+v", out.Findings)
	}
}

func TestManyStartFindingsOfOneKindBecomeOne(t *testing.T) {
	var many []Finding
	for i := range maxListedNames + 5 {
		many = append(many, Finding{Owner: "library", Obligation: "file.unexplained", Observed: fmt.Sprintf("library/vendor/f%02d.go is not registered", i)})
	}
	few := Finding{Owner: "library", Obligation: "registry.missing_file", Observed: "library/gone.go is registered but absent"}
	got := groupedStart(append(many, few))
	if len(got) != 2 || got[1] != few {
		t.Fatalf("grouped %+v", got)
	}
	contains(t, "merged finding", got[0].Observed, fmt.Sprintf("%d findings of this kind; the first %d:", maxListedNames+5, maxListedNames), "library/vendor/f00.go", fmt.Sprintf("library/vendor/f%02d.go", maxListedNames-1))
	if strings.Contains(got[0].Observed, fmt.Sprintf("f%02d.go", maxListedNames)) || got[0].Owner != "library" || got[0].Obligation != "file.unexplained" {
		t.Fatalf("merged finding %+v", got[0])
	}
}

func TestToolsStayUsableAfterFinishAndTheLastFinishCounts(t *testing.T) {
	f := newFixture(t)
	f.agents.quiet = true
	f.agents.script = func(c *call) error {
		if c.is(1, "lending/reminders", ExecuteTurn) && c.turn.Resume == "" {
			if a := c.finish("done"); a.Refused {
				t.Errorf("finish refused: %s", a.Text)
			}
			if a := c.send("parent", "a note after finish"); a.Refused {
				t.Errorf("send after finish refused: %s", a.Text)
			}
			// The last call counts: waiting reports nothing to the parent.
			if a := c.finish("waiting"); a.Refused {
				t.Errorf("second finish refused: %s", a.Text)
			}
			return nil
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	for _, r := range f.ev.kinds("child.report") {
		if r.Manager == "lending/reminders" {
			t.Errorf("a turn that last finished waiting reported done: %+v", r)
		}
	}
	contains(t, "lending's prompt", f.agents.prompt("run1-r1-lending-x2"), "Message from your child lending/reminders (overdue notices): a note after finish")
}

func TestAssessKeepsTheLastValidReportOfTheTurn(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.is(1, "library", VerifyTurn):
			if a := c.assess(false, []fnd{finding("library", "first", true)}); a.Refused {
				t.Errorf("assess refused: %s", a.Text)
			}
			// An invalid call later in the turn leaves the valid report in place.
			if a := c.tool(toolAssess, map[string]any{"summary": "s", "satisfied": true, "extra": 1}); !a.Refused {
				t.Errorf("invalid assess accepted: %+v", a)
			}
			return nil
		case c.is(2, "library", VerifyTurn):
			open := c.myOpen()
			if len(open) != 1 {
				t.Fatalf("open findings of library %v", open)
			}
			c.assess(true, nil, open...)
			// The replacing report keeps the finding open: the closure above is not applied.
			if a := c.assess(false, nil); a.Refused || !strings.Contains(a.Text, "replaces") {
				t.Errorf("replacing assess %+v", a)
			}
			return nil
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 3 || out.Rounds[1].Green {
		t.Fatalf("%s %s rounds %+v", out.Status, out.Reason, out.Rounds)
	}
	var first Finding
	for _, fd := range out.Findings {
		if fd.Obligation == "first" {
			first = fd
		}
	}
	if first.RaisedBy != "library" || first.Round != 1 || !first.Closed || first.ClosedRound != 3 {
		t.Fatalf("finding %+v", first)
	}
	if n := f.ev.count("assessment"); n != 4*3 {
		t.Errorf("%d assessment events, want one per Manager and round", n)
	}
}
