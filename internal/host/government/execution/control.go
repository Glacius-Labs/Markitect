package execution

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

type Control interface {
	StartRun(runID, reportPath string) error
	ReserveActor(runID string, sequence int, phase string, spec RunnerSpec) error
	CompleteActor(runID string, record ActorRecord) error
	ReserveRepair(runID, area string, attempt int) error
	Checkpoint(report Report) error
	Fence() error
}

type queueControl struct {
	dir   string
	state *queueState
	lease *government.Lease
	mu    sync.Mutex
	fatal error
}

func (q *queueControl) Fence() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.lease.Verify(); err != nil {
		return err
	}
	return q.fatal
}
func (q *queueControl) StartRun(runID, reportPath string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.lease.Verify(); err != nil {
		return err
	}
	if q.fatal != nil {
		return q.fatal
	}
	for i := range q.state.Report.Jobs {
		if q.state.Report.Jobs[i].ID == q.state.activeJob {
			q.state.Report.Jobs[i].RunID = runID
			q.state.Report.Jobs[i].ReportPath = reportPath
		}
	}
	err := appendQueueEventLocked(q.dir, q.state, "run-started", map[string]string{"jobId": q.state.activeJob, "runId": runID, "reportPath": reportPath})
	if err != nil {
		q.fatal = err
	}
	return err
}
func (q *queueControl) ReserveActor(runID string, sequence int, phase string, spec RunnerSpec) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.lease.Verify(); err != nil {
		return err
	}
	if q.fatal != nil {
		return q.fatal
	}
	if sequence < 1 || runID == "" {
		return errors.New("actor reservation requires a run ID and positive sequence")
	}
	key := fmt.Sprintf("%s/%d", runID, sequence)
	if q.state.actorReservations[key] {
		return errors.New("actor reservation identity was already consumed")
	}
	if q.state.Report.ActorStarts >= q.state.Report.Limits.ActorStarts {
		return errors.New("cumulative queue actor-start limit exhausted")
	}
	if q.state.Report.InFlightActors >= q.state.Report.Limits.MaxParallelism {
		return errors.New("cumulative queue parallelism limit is currently occupied")
	}
	q.state.Report.ActorStarts++
	q.state.Report.InFlightActors++
	if q.state.Report.InFlightActors > q.state.Report.PeakParallelism {
		q.state.Report.PeakParallelism = q.state.Report.InFlightActors
	}
	if q.state.actorReservations == nil {
		q.state.actorReservations = map[string]bool{}
	}
	if q.state.pendingActors == nil {
		q.state.pendingActors = map[string]bool{}
	}
	q.state.actorReservations[key] = true
	q.state.pendingActors[key] = true
	if err := appendQueueEventLocked(q.dir, q.state, "actor-reserved", map[string]any{"runId": runID, "sequence": sequence, "phase": phase, "slotId": spec.SlotID, "count": q.state.Report.ActorStarts, "inFlight": q.state.Report.InFlightActors, "peak": q.state.Report.PeakParallelism}); err != nil {
		q.fatal = err
		return err
	}
	return nil
}
func (q *queueControl) CompleteActor(runID string, record ActorRecord) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.lease.Verify(); err != nil {
		return err
	}
	if q.fatal != nil {
		return q.fatal
	}
	key := fmt.Sprintf("%s/%d", runID, record.Sequence)
	if !q.state.pendingActors[key] || q.state.Report.InFlightActors < 1 {
		return errors.New("actor completion has no durable start reservation")
	}
	applyUsage(&q.state.Report.Usage, record.Result.Receipt.Usage)
	if record.Result.Receipt.RunID != "" && record.Result.Receipt.WallTimeMilliseconds >= 0 && !record.StartedAt.IsZero() && !record.FinishedAt.IsZero() {
		q.state.Report.Usage.ObservedActorWallTimeMillis.Known = true
		q.state.Report.Usage.ObservedActorWallTimeMillis.Total += record.Result.Receipt.WallTimeMilliseconds
	} else {
		q.state.Report.Usage.ObservedActorWallTimeMillis.Unknown = true
	}
	delete(q.state.pendingActors, key)
	q.state.Report.InFlightActors--
	if err := appendQueueEventLocked(q.dir, q.state, "actor-completed", map[string]any{"runId": runID, "record": record, "usage": q.state.Report.Usage, "inFlight": q.state.Report.InFlightActors, "peak": q.state.Report.PeakParallelism}); err != nil {
		q.fatal = err
		return err
	}
	return nil
}
func (q *queueControl) ReserveRepair(runID, area string, attempt int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.lease.Verify(); err != nil {
		return err
	}
	if q.fatal != nil {
		return q.fatal
	}
	if q.state.Report.Repairs >= q.state.Report.Limits.MaxRepairs {
		return errors.New("cumulative queue repair limit exhausted")
	}
	q.state.Report.Repairs++
	if err := appendQueueEventLocked(q.dir, q.state, "repair-reserved", map[string]any{"runId": runID, "area": area, "attempt": attempt, "count": q.state.Report.Repairs}); err != nil {
		q.fatal = err
		return err
	}
	return nil
}
func (q *queueControl) Checkpoint(report Report) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.lease.Verify(); err != nil {
		return err
	}
	if q.fatal != nil {
		return q.fatal
	}
	q.state.Checkpoints[report.RunID] = report
	err := appendQueueEventLocked(q.dir, q.state, "checkpoint", report)
	if err != nil {
		q.fatal = err
	}
	return err
}
func (q *queueControl) event(typ string, payload any) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.lease.Verify(); err != nil {
		return err
	}
	if q.fatal != nil {
		return q.fatal
	}
	err := appendQueueEventLocked(q.dir, q.state, typ, payload)
	if err != nil {
		q.fatal = err
	}
	return err
}
func applyUsage(dst *QueueUsage, u *agentexec.Usage) {
	if u == nil {
		dst.InputTokens.Unknown = true
		dst.OutputTokens.Unknown = true
		dst.CachedTokens.Unknown = true
		dst.ToolCalls.Unknown = true
		return
	}
	add := func(c *UsageCounter, v *int64) {
		if v == nil {
			c.Unknown = true
			return
		}
		c.Known = true
		c.Total += *v
	}
	add(&dst.InputTokens, u.InputTokens)
	add(&dst.OutputTokens, u.OutputTokens)
	add(&dst.CachedTokens, u.CachedTokens)
	add(&dst.ToolCalls, u.ToolCalls)
}
