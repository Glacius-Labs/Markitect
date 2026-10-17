package run_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
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

func TestContinuationRequiresRemainingStructuralReviewFloor(t *testing.T) {
	for _, resource := range []string{"starts", "steps", "time", "runs", "overrun"} {
		t.Run(resource, func(t *testing.T) {
			i := fixture(t)
			if resource == "runs" {
				i.Policy.Assignment.Runs = 1
			}
			b, err := run.Bind(i)
			if err != nil {
				t.Fatal(err)
			}
			c, j := runningWith(t, b)
			switch resource {
			case "starts":
				spend(t, c, j, run.SessionStart, 5) // 3 remain; new basis needs 4.
			case "steps":
				spend(t, c, j, run.AgentStep, 7)
			case "time":
				commit(t, c, j, reserve("check", run.ProjectCheck, run.Assessment, 10*time.Minute))
				commit(t, c, j, settle("check", 10*time.Minute))
			case "overrun":
				commit(t, c, j, reserve("overrun", run.AgentStep, run.Assessment, time.Second))
				commit(t, c, j, settle("overrun", 2*time.Second))
			}
			finishIncomplete(t, c, j)
			before, rev, calls := c.Budget(), c.Revision(), j.appendCalls
			i.Source.Guard = "changed-source"
			next, err := run.Bind(i)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Propose(run.PrepareCommand("continue", "a", "r2", "r", observed, next)); !errors.Is(err, run.ErrBudget) {
				t.Fatal("insufficient structural capacity prepared", err)
			}
			if c.Revision() != rev || !reflect.DeepEqual(before, c.Budget()) || j.appendCalls != calls {
				t.Fatal("rejected continuation consumed or appended")
			}
		})
	}
}

func TestContinuationCanUseStructuralFloorBelowConfiguredReserve(t *testing.T) {
	i := fixture(t)
	i.Policy.Reserve = run.Capacity{SessionStarts: 6, Steps: 6, ExecutionTime: 5 * time.Minute}
	b, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	c, j := runningWith(t, b)
	spend(t, c, j, run.SessionStart, 4)
	spend(t, c, j, run.AgentStep, 6)
	commit(t, c, j, reserve("check", run.ProjectCheck, run.Assessment, 6*time.Minute))
	commit(t, c, j, settle("check", 6*time.Minute))
	finishIncomplete(t, c, j)
	if c.Budget().Remaining.SessionStarts != 4 || c.Budget().Remaining.Steps != 4 || c.Budget().Remaining.ExecutionTime != 4*time.Minute {
		t.Fatal("fixture did not distinguish floor from configured reserve", c.Budget())
	}
	i.Source.Guard = "changed-source"
	b, err = run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, run.PrepareCommand("continue", "a", "r2", "r", observed, b))
	start := command("start-r2", run.StartRun)
	start.Run = "r2"
	commit(t, c, j, start)
	write := reserve("write-r2", run.AgentStep, run.Writing, time.Second)
	write.Run = "r2"
	if _, err := c.Propose(write); !errors.Is(err, run.ErrBudget) {
		t.Fatal("writing crossed configured reserve", err)
	}
	review := reserve("review-r2", run.AgentStep, run.Assessment, time.Second)
	review.Run = "r2"
	commit(t, c, j, review)
}

func TestZeroReviewContinuationHasZeroStructuralFloor(t *testing.T) {
	i := fixtureWith(t, func(input *model.Input) {
		var entries []model.Entry
		for _, e := range input.Root.Entries {
			if e.Path == "model.yaml" || e.Path == "model-lock.yaml" {
				entries = append(entries, e)
			}
		}
		input.Root.Entries = append(entries, model.Entry{Path: "model", Kind: model.Directory})
	}, nil)
	// This model has no artifact responsibilities. Keep only its actual inputs.
	i.Registry = realization.Registry{}
	var entries []snapshot.Entry
	for _, d := range i.Documents {
		entries = append(entries, d.File)
	}
	i.Census = tree(t, entries)
	b, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	if b.RequiredReviews() != 0 {
		t.Fatal("fixture still needs reviews")
	}
	c, j := runningWith(t, b)
	spend(t, c, j, run.SessionStart, 8)
	spend(t, c, j, run.AgentStep, 10)
	commit(t, c, j, reserve("check", run.ProjectCheck, run.Assessment, 10*time.Minute))
	commit(t, c, j, settle("check", 10*time.Minute))
	finishIncomplete(t, c, j)
	i.Source.Guard = "changed-source"
	b, err = run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, run.PrepareCommand("continue", "a", "r2", "r", observed, b))
	if c.Budget().Consumed.Runs != 2 {
		t.Fatal("zero review floor blocked continuation")
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

func TestWritingProtectsReserveAndAssessmentCanSpendIt(t *testing.T) {
	c, j := running(t)
	for i := 0; i < 6; i++ {
		id := fmt.Sprint(i)
		commit(t, c, j, reserve(id, run.AgentStep, run.Writing, time.Second))
		commit(t, c, j, settle(id, 0))
	}
	if c.Budget().Remaining.Steps != 4 {
		t.Fatal("unexpected step balance", c.Budget())
	}
	if _, err := c.Propose(reserve("writer", run.AgentStep, run.Writing, time.Second)); !errors.Is(err, run.ErrBudget) {
		t.Fatal("writing spent completion reserve", err)
	}
	commit(t, c, j, reserve("review", run.AgentStep, run.Assessment, time.Minute))
	if c.Budget().Held.Steps != 1 || c.Budget().Consumed.Steps != 6 {
		t.Fatal("reservation not held separately", c.Budget())
	}
	commit(t, c, j, settle("review", time.Second))
	if c.Budget().Remaining.Steps != 3 || c.Budget().Consumed.ExecutionNanoseconds != "1000000000" {
		t.Fatal("settlement did not release unused duration", c.Budget())
	}
	if _, err := c.Propose(reserve("too-long", run.AgentStep, run.Assessment, 3*time.Minute)); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("step timeout ceiling ignored", err)
	}
}

func TestUnknownEffectCannotBeFreedByFailureStopOrWaiting(t *testing.T) {
	c, j := running(t)
	commit(t, c, j, reserve("step", run.AgentStep, run.Writing, time.Minute))
	u := command("unknown", run.ObserveUncertain)
	u.Invocation = "step"
	u.Reason = "Host cannot establish termination"
	commit(t, c, j, u)
	before := c.Budget()
	if c.Current().State != run.Reconciling || before.Held.Steps != 1 || before.Held.ExecutionTime != time.Minute {
		t.Fatal("unknown effect lost hold")
	}
	for _, outcome := range []run.Outcome{run.Failed, run.Stopped, run.Incomplete, run.Stale} {
		finish := command("finish-"+string(outcome), run.FinishRun)
		finish.Outcome = outcome
		finish.Reason = "cannot bypass reconciliation"
		if _, err := c.Propose(finish); !errors.Is(err, run.ErrState) {
			t.Fatal("unknown effect finished", outcome, err)
		}
	}
	if _, err := c.Propose(reserve("other", run.AgentStep, run.Assessment, time.Second)); !errors.Is(err, run.ErrState) {
		t.Fatal("new work while reconciling", err)
	}
	stop := command("stop", run.StopRun)
	stop.Reason = "Owner requested stop"
	commit(t, c, j, stop)
	if !reflect.DeepEqual(before, c.Budget()) {
		t.Fatal("stop released hold")
	}
	commit(t, c, j, settle("step", 20*time.Second))
	finish := command("finished", run.FinishRun)
	finish.Outcome = run.Stopped
	finish.Reason = "Host observed all effects quiescent"
	commit(t, c, j, finish)
	if c.Current().State != run.Finished || c.Current().Outcome != run.Stopped {
		t.Fatal("known quiescence did not finish")
	}
	if _, err := c.Propose(command("restart", run.StartRun)); !errors.Is(err, run.ErrState) {
		t.Fatal("Finished was mutable", err)
	}
}

func TestActualOverrunIsPreservedAndBlocksAllAdmission(t *testing.T) {
	c, j := running(t)
	commit(t, c, j, reserve("step", run.AgentStep, run.Assessment, time.Second))
	commit(t, c, j, settle("step", 11*time.Minute))
	b := c.Budget()
	if !b.Overrun || b.Remaining != (run.Capacity{}) || b.Consumed.Steps != 1 || b.Consumed.ExecutionNanoseconds != "660000000000" {
		t.Fatal("actual overrun discarded or rounded", b)
	}
	if _, err := c.Propose(reserve("new", run.SessionStart, run.Assessment, time.Second)); !errors.Is(err, run.ErrBudget) {
		t.Fatal("admission after overrun", err)
	}
}

func TestProvenNonDispatchAndWaitingDoNotCharge(t *testing.T) {
	c, j := running(t)
	before := c.Budget()
	commit(t, c, j, reserve("start", run.SessionStart, run.Writing, time.Minute))
	u := command("unknown", run.ObserveUncertain)
	u.Invocation = "start"
	u.Reason = "Dispatch uncertain"
	commit(t, c, j, u)
	release := command("release", run.ReleaseUndispatched)
	release.Invocation = "start"
	release.Reason = "Host inspection established no dispatch"
	commit(t, c, j, release)
	commit(t, c, j, command("resume", run.ResumeRun))
	wait := command("wait", run.WaitRun)
	wait.Reason = "Awaiting owner decision"
	wait.NextActor = "owner"
	commit(t, c, j, wait)
	if !reflect.DeepEqual(before, c.Budget()) || c.Current().State != run.Waiting {
		t.Fatal("non-dispatch or waiting charged", c.Budget())
	}
	resume := command("later", run.ResumeRun)
	resume.ObservedAt = observed.Add(100 * 24 * time.Hour)
	commit(t, c, j, resume)
	if !reflect.DeepEqual(before, c.Budget()) {
		t.Fatal("report timestamp charged elapsed waiting")
	}
}

func TestContinuationSharesConsumedBudgetAndCannotRewriteFinishedRun(t *testing.T) {
	c, j := running(t)
	commit(t, c, j, reserve("step", run.AgentStep, run.Assessment, time.Minute))
	commit(t, c, j, settle("step", time.Second))
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
