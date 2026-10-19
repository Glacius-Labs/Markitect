package run_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func extension(id string, caps run.Capacity) run.Command {
	c := command(id, run.ExtendAssignment)
	c.Caps = caps
	c.OwnerInput = "\xef\xbb\xbfExtend these exact caps.\r\n"
	return c
}
func closure(id string) run.Command {
	c := command(id, run.CloseAssignment)
	c.OwnerInput = "Close this assignment; retain its unfinished work.\r\n"
	c.Quiescence = "Host observed no active or uncertain effects after reconciliation."
	return c
}

func TestAssignmentRejectsInvalidExtensionsWithoutMutation(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"equal", "decrease", "ceiling", "negative", "missing input", "oversize input", "invalid UTF8", "extra closure"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := running(t)
			caps := fixture(t).Policy.Assignment
			caps.Steps++
			x := extension("bad", caps)
			switch kind {
			case "equal":
				x.Caps = fixture(t).Policy.Assignment
			case "decrease":
				x.Caps.Runs--
			case "ceiling":
				x.Caps.Steps = fixture(t).Policy.Ceilings.Steps + 1
			case "negative":
				x.Caps.Steps = -1
			case "missing input":
				x.OwnerInput = ""
			case "oversize input":
				x.OwnerInput = strings.Repeat("x", (64<<10)+1)
			case "invalid UTF8":
				x.OwnerInput = string([]byte{0xff})
			case "extra closure":
				x.Quiescence = "unexpected"
			}
			before, rev := c.Assignments(), c.Revision()
			if _, err := c.Propose(x); err == nil {
				t.Fatal("accepted invalid extension")
			}
			if c.Revision() != rev || !reflect.DeepEqual(before, c.Assignments()) {
				t.Fatal("rejection mutated ledger")
			}
		})
	}
}

func TestClosureRequiresFinishedAndExplicitQuiescence(t *testing.T) {
	t.Parallel()
	c, j := running(t)
	close := closure("close")
	if _, err := c.Propose(close); !errors.Is(err, run.ErrState) {
		t.Fatal("closed unfinished run", err)
	}
	commit(t, c, j, sessionPreparation(t, c, "s"))
	u := command("uncertain", run.ObserveUncertain)
	u.Invocation = "open-s"
	u.Reason = "connection lost"
	commit(t, c, j, u)
	if _, err := c.Propose(close); !errors.Is(err, run.ErrState) {
		t.Fatal("closed unknown effect", err)
	}
	stop := command("stop", run.StopRun)
	stop.Reason = "owner stop"
	commit(t, c, j, stop)
	if _, err := c.Propose(close); !errors.Is(err, run.ErrState) {
		t.Fatal("stop request established quiescence", err)
	}
	io, _ := c.SessionChannel(j, "s", run.RefuseTools)
	if _, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("stopped"), Terminated: true, Reason: "Host observed termination"}); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-s", time.Second))
	finishIncomplete(t, c, j)
	close.Quiescence = ""
	if _, err := c.Propose(close); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("empty hold ledger became physical quiescence", err)
	}
	close = closure("close")
	commit(t, c, j, close)
	v := c.Assignments()[0]
	if v.Closure.Outcome != run.Incomplete || v.Closure.Quiescence != close.Quiescence || v.Closure.OwnerInput != close.OwnerInput {
		t.Fatal("closure changed observations", v)
	}
	for _, x := range []run.Command{closure("reclose"), extension("reopen", fixture(t).Policy.Ceilings), command("restart", run.StartRun)} {
		if _, err := c.Propose(x); !errors.Is(err, run.ErrState) {
			t.Fatal("closed assignment mutable", err)
		}
	}
}

func TestClosedAssignmentAndFreshLedgerSurviveSameStreamRestoreAndReplay(t *testing.T) {
	t.Parallel()
	c, j := running(t)
	b := binding(t)
	driveNext(t, c, j, stepInputs(t, false), "root")
	finishIncomplete(t, c, j)
	extend := extension("extend", fixture(t).Policy.Ceilings)
	commit(t, c, j, extend)
	// The exact basis can continue once because a new grant was recorded.
	commit(t, c, j, run.PrepareCommand("continue", "a", "r2", "r", observed, b))
	f := command("finish2", run.FinishRun)
	f.Run = "r2"
	f.Reason = "retain"
	f.Outcome = run.Incomplete
	commit(t, c, j, f)
	if _, err := c.Propose(run.PrepareCommand("no-reset", "a", "r3", "r2", observed, b)); !errors.Is(err, run.ErrState) {
		t.Fatal("one grant reset multiple runs", err)
	}
	if _, err := c.Propose(run.PrepareCommand("premature", "b", "new-run", "", observed, b)); !errors.Is(err, run.ErrState) {
		t.Fatal("new assignment bypassed closure", err)
	}
	close := closure("close")
	close.Run = "r2"
	oldReceipt := commit(t, c, j, close)
	closed := c.Assignments()[0]
	if _, err := c.Propose(run.PrepareCommand("reopen", "a", "r3", "", observed, b)); !errors.Is(err, run.ErrState) {
		t.Fatal("reopened closed identity", err)
	}
	i := fixture(t)
	i.OwnerRequest, _ = run.NewPayload("text/plain", "utf-8", []byte("A fresh ordinary request."))
	newBasis, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	rev := c.Revision()
	commit(t, c, j, run.PrepareCommand("ordinary", "b", "new-run", "", observed, newBasis))
	views := c.Assignments()
	if c.Revision() <= rev || len(views) != 2 || !reflect.DeepEqual(views[0], closed) || views[1].OwnerRequest != i.OwnerRequest.Ref() || views[1].CapRevision != 1 || len(views[1].Extensions) != 0 || len(c.Invocations()) != 0 || c.Budget().Consumed.Runs != 1 || c.Budget().Consumed.Steps != 0 || c.Budget().Remaining.Steps != i.Policy.Assignment.Steps {
		t.Fatal("ordinary assignment inherited prior ledger/grant", views)
	}
	// Mutating returned slices cannot alter closed or active retained history.
	views[0].Runs[0].Reason = "tamper"
	views[0].Extensions[0].OwnerInput = "tamper"
	views[0].Invocations[0].Elapsed = 0
	views[1].Runs[0].ID = "tamper"
	if !reflect.DeepEqual(c.Assignments()[0], closed) || c.Current().ID != "new-run" {
		t.Fatal("view mutated history")
	}
	c2 := restored(t, &readJournal{store: j})
	if !reflect.DeepEqual(c.Assignments(), c2.Assignments()) {
		t.Fatal("restore changed assignment history")
	}
	before, head, calls := c2.Assignments(), c2.Revision(), j.appendCalls
	if got := commit(t, c2, j, close); got != oldReceipt {
		t.Fatal("closed replay changed receipt")
	}
	if c2.Revision() != head || j.appendCalls != calls || !reflect.DeepEqual(before, c2.Assignments()) {
		t.Fatal("old replay altered fresh assignment")
	}
	changed := close
	changed.OwnerInput = "different decision"
	if _, err := c2.Propose(changed); !errors.Is(err, run.ErrConflict) {
		t.Fatal("changed old decision reused ID", err)
	}
	start := command("start-new", run.StartRun)
	start.Assignment = "b"
	start.Run = "new-run"
	commit(t, c2, j, start)
	x := sessionPreparation(t, c2, "new-session")
	// Constructor proof includes the attempted historical invocation identity.
	item, _ := c2.Next()
	profile, tools := payload(t, `{"profile":"synthetic recorded metadata"}`), payload(t, `{"tools":[]}`)
	tools, err = run.ToolCatalog(item.Phase)
	if err != nil {
		t.Fatal(err)
	}
	x, err = c2.PrepareTaskSessionCommand(meta("new-command"), item, x.Session.Plan, run.Reservation{ID: "invoke-root", Kind: run.SessionStart, Purpose: item.Purpose, Maximum: time.Minute}, profile, tools)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c2.Propose(x); !errors.Is(err, run.ErrConflict) {
		t.Fatal("reused historic effect identity", err)
	}
}

func TestLifecycleCodecPreservesPriorCommandShapeAndExactRawInput(t *testing.T) {
	t.Parallel()
	c := command("start", run.StartRun)
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	// New optional fields must not alter existing v1 byte identity/history.
	for _, name := range []string{"Caps", "OwnerInput", "Quiescence", "Manager"} {
		if bytes.Contains(data, []byte(name)) {
			t.Fatal("changed prior command shape", string(data))
		}
	}
	co, j := running(t)
	extend := extension("extend", fixture(t).Policy.Ceilings)
	p, err := co.Propose(extend)
	if err != nil {
		t.Fatal(err)
	}
	b, err := run.DecodeBatch(p.Batch().Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if b.Events()[0].Body.OwnerInput != extend.OwnerInput {
		t.Fatal("raw owner input normalized")
	}
	_, err = co.Commit(context.Background(), j, p)
	if err != nil {
		t.Fatal(err)
	}
}
