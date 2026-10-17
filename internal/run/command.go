package run

import (
	"encoding/json"
	"time"
)

type State string

const (
	Prepared    State = "prepared"
	Running     State = "running"
	Waiting     State = "waiting"
	Verifying   State = "verifying"
	Publishing  State = "publishing"
	Reconciling State = "reconciling"
	Finished    State = "finished"
)

type Outcome string

const (
	Applied    Outcome = "applied"
	NoChange   Outcome = "no_change"
	Stale      Outcome = "stale"
	Stopped    Outcome = "stopped"
	Incomplete Outcome = "incomplete"
	Failed     Outcome = "failed"
)

const (
	PrepareRun          = "run.prepared"
	StartRun            = "run.started"
	ReserveInvocation   = "budget.reserved"
	SettleInvocation    = "budget.settled"
	ObserveUncertain    = "invocation.uncertain"
	ReleaseUndispatched = "budget.released"
	WaitRun             = "run.waiting"
	ResumeRun           = "run.resumed"
	StopRun             = "run.stop-requested"
	FinishRun           = "run.finished"
)

// Command is a closed RUN-1 command record. ObservedAt labels facts in UTC;
// only explicit Elapsed observations charge active time. An identical retry
// retains the original complete command, including its timestamp. A new external
// invocation is a new command/effect ID, not a retry of an old observation.
type Command struct {
	ID, Kind, Assignment, Run string
	ObservedAt                time.Time
	Basis                     BlobRef
	Predecessor               string
	Reservation               Reservation
	Invocation                string
	Elapsed                   time.Duration
	Reason, NextActor         string
	Outcome                   Outcome
	binding                   Binding
}

func PrepareCommand(id, assignment, run, predecessor string, at time.Time, basis Binding) Command {
	return Command{ID: id, Kind: PrepareRun, Assignment: assignment, Run: run, ObservedAt: at, Basis: basis.Ref(), Predecessor: predecessor, binding: basis}
}

func (c Command) validate() error {
	if !identifier(c.ID) || !identifier(c.Assignment) || !identifier(c.Run) || c.ObservedAt.IsZero() || c.ObservedAt.Location() != time.UTC {
		return ErrInvalid
	}
	allowed := Command{ID: c.ID, Kind: c.Kind, Assignment: c.Assignment, Run: c.Run, ObservedAt: c.ObservedAt}
	switch c.Kind {
	case PrepareRun:
		if !c.Basis.Valid() || c.Basis.Size <= 0 {
			return ErrInvalid
		}
		allowed.Basis = c.Basis
		allowed.Predecessor = c.Predecessor
	case StartRun, ResumeRun:
	case ReserveInvocation:
		if !c.Reservation.valid() {
			return ErrInvalid
		}
		allowed.Reservation = c.Reservation
	case SettleInvocation:
		if !identifier(c.Invocation) || c.Elapsed < 0 || !text(c.Reason) {
			return ErrInvalid
		}
		allowed.Invocation = c.Invocation
		allowed.Elapsed = c.Elapsed
		allowed.Reason = c.Reason
	case ObserveUncertain, ReleaseUndispatched:
		if !identifier(c.Invocation) || !text(c.Reason) {
			return ErrInvalid
		}
		allowed.Invocation = c.Invocation
		allowed.Reason = c.Reason
	case WaitRun:
		if !text(c.Reason) || !identifier(c.NextActor) {
			return ErrInvalid
		}
		allowed.Reason = c.Reason
		allowed.NextActor = c.NextActor
	case StopRun:
		if !text(c.Reason) {
			return ErrInvalid
		}
		allowed.Reason = c.Reason
	case FinishRun:
		if !text(c.Reason) || c.Outcome != Stale && c.Outcome != Stopped && c.Outcome != Incomplete && c.Outcome != Failed {
			return ErrInvalid
		}
		allowed.Reason = c.Reason
		allowed.Outcome = c.Outcome
	default:
		return ErrInvalid
	}
	a, _ := json.Marshal(allowed)
	b, _ := json.Marshal(c)
	if string(a) != string(b) {
		return ErrInvalid
	}
	return nil
}
