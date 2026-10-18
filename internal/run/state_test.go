package run_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func command(id, kind string) run.Command {
	return run.Command{ID: id, Kind: kind, Assignment: "a", Run: "r", ObservedAt: observed}
}

func commit(t *testing.T, c *run.Coordinator, j *memoryJournal, command run.Command) run.AppendReceipt {
	t.Helper()
	p, err := c.Propose(command)
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Commit(context.Background(), j, p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func running(t *testing.T) (*run.Coordinator, *memoryJournal) {
	return runningWith(t, binding(t))
}

func runningWith(t *testing.T, b run.Binding) (*run.Coordinator, *memoryJournal) {
	t.Helper()
	c, _ := run.NewCoordinator("repo")
	j := journal()
	commit(t, c, j, run.PrepareCommand("prepare", "a", "r", "", observed, b))
	commit(t, c, j, command("start", run.StartRun))
	return c, j
}

func finishIncomplete(t *testing.T, c *run.Coordinator, j *memoryJournal) {
	t.Helper()
	f := command("finish", run.FinishRun)
	f.Outcome = run.Incomplete
	f.Reason = "Retain unfinished work"
	commit(t, c, j, f)
}

func spend(t *testing.T, c *run.Coordinator, j *memoryJournal, kind run.InvocationKind, count int) {
	t.Helper()
	for n := 0; n < count; n++ {
		id := fmt.Sprintf("%s-%d", kind, n)
		commit(t, c, j, reserve(id, kind, run.Assessment, time.Second))
		commit(t, c, j, settle(id, 0))
	}
}

func TestContinuationPreservesAssignmentOwnerRequest(t *testing.T) {
	t.Parallel()
	c, j := running(t)
	finishIncomplete(t, c, j)
	before, rev, calls := c.Budget(), c.Revision(), j.appendCalls
	i := fixture(t)
	i.Source.Guard = "changed-source"
	i.OwnerRequest, _ = run.NewPayload("text/plain", "utf-8", []byte("A materially different owner assignment"))
	b, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Propose(run.PrepareCommand("continue", "a", "r2", "r", observed, b)); !errors.Is(err, run.ErrState) {
		t.Fatal("owner assignment changed without amendment", err)
	}
	if c.Revision() != rev || !reflect.DeepEqual(before, c.Budget()) || j.appendCalls != calls {
		t.Fatal("rejected continuation consumed or appended")
	}
	i = fixture(t)
	i.Source.Guard = "changed-source"
	b, err = run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, run.PrepareCommand("continue", "a", "r2", "r", observed, b))
	if c.Budget().Consumed.Runs != 2 {
		t.Fatal("identical assignment input could not continue")
	}
}

func reserve(id string, kind run.InvocationKind, purpose run.Purpose, maximum time.Duration) run.Command {
	c := command("reserve-"+id, run.ReserveInvocation)
	c.Reservation = run.Reservation{ID: id, Kind: kind, Purpose: purpose, Maximum: maximum}
	return c
}

func settle(id string, elapsed time.Duration) run.Command {
	c := command("settle-"+id, run.SettleInvocation)
	c.Invocation = id
	c.Elapsed = elapsed
	c.Reason = "Host observed completed invocation and elapsed duration"
	return c
}

func TestPreparedAndRunChargeRequireConfirmedAtomicAppend(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"lost", "wrong", "absent", "put"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := run.NewCoordinator("repo")
			j := journal()
			p, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
			if err != nil {
				t.Fatal(err)
			}
			if c.Current().ID != "" || c.Budget().Consumed.Runs != 0 || c.Revision() != 0 {
				t.Fatal("proposal prepared a run")
			}
			j.appendMode = mode
			j.putFailure = mode == "put"
			if _, err := c.Commit(context.Background(), j, p); !errors.Is(err, run.ErrUncertain) {
				t.Fatal(err)
			}
			if c.Current().ID != "" || c.Budget().Consumed.Runs != 0 || c.Revision() != 0 {
				t.Fatal("unconfirmed append affected projection")
			}
			if _, err := c.Propose(command("start", run.StartRun)); !errors.Is(err, run.ErrUncertain) {
				t.Fatal("dependent work admitted")
			}
			if mode == "put" && j.appendCalls != 0 {
				t.Fatal("append followed failed put")
			}
			j.putFailure = false
			j.appendMode = ""
			j.readFailure = true
			if err := c.ReconcilePending(context.Background(), j); !errors.Is(err, run.ErrUncertain) {
				t.Fatal("unavailable store became absence", err)
			}
			j.readFailure = false
			if err := c.ReconcilePending(context.Background(), j); err != nil {
				t.Fatal(err)
			}
			if mode == "lost" || mode == "wrong" {
				if c.Current().State != run.Prepared || c.Budget().Consumed.Runs != 1 || c.Revision() != 2 {
					t.Fatal("matching actual commit not recovered")
				}
			} else {
				if c.Current().ID != "" || c.Budget().Consumed.Runs != 0 {
					t.Fatal("absent commit charged")
				}
				commit(t, c, j, run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
			}
			before := c.Budget()
			calls := j.appendCalls
			commit(t, c, j, run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
			if !reflect.DeepEqual(before, c.Budget()) || j.appendCalls != calls {
				t.Fatal("logical retry charged or reappended")
			}
		})
	}
}

func TestProposalReplayCannotCrossCoordinators(t *testing.T) {
	t.Parallel()
	c, j := running(t)
	replayed, err := c.Propose(command("start", run.StartRun))
	if err != nil {
		t.Fatal(err)
	}
	other, _ := run.NewCoordinator("repo")
	if _, err := other.Commit(context.Background(), j, replayed); !errors.Is(err, run.ErrConflict) {
		t.Fatal("foreign replay accepted", err)
	}
	different := command("start", run.StartRun)
	different.ObservedAt = observed.Add(time.Second)
	if _, err := c.Propose(different); !errors.Is(err, run.ErrConflict) {
		t.Fatal("changed retry accepted", err)
	}
}

func TestContinuationSharesConsumedBudgetAndCannotRewriteFinishedRun(t *testing.T) {
	t.Parallel()
	c, j := running(t)
	exploreNext(t, c, j, stepInputs(t, false), "root")
	finish := command("finish", run.FinishRun)
	finish.Outcome = run.Incomplete
	finish.Reason = "Awaiting model change"
	commit(t, c, j, finish)
	old := c.Current()
	before := c.Budget()
	in := fixture(t)
	in.Source.Guard = "new-source-guard"
	b, err := run.Bind(in)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, run.PrepareCommand("continue", "a", "r2", "r", observed, b))
	if c.Current().State != run.Prepared || c.Budget().Consumed.Runs != 2 || c.Budget().Consumed.Steps != before.Consumed.Steps || c.Budget().Consumed.ExecutionNanoseconds != before.Consumed.ExecutionNanoseconds {
		t.Fatal("continuation reset ledger")
	}
	if old.State != run.Finished || old.Outcome != run.Incomplete {
		t.Fatal("old outcome changed")
	}
	for _, outcome := range []run.Outcome{run.Applied, run.NoChange} {
		finish.Run = "r2"
		finish.ID = "false-" + string(outcome)
		finish.Outcome = outcome
		if _, err := c.Propose(finish); !errors.Is(err, run.ErrInvalid) {
			t.Fatal("unsupported final evidence manufactured", err)
		}
	}
}
