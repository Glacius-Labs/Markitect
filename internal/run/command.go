package run

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
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
	ExtendAssignment    = "assignment.extended"
	CloseAssignment     = "assignment.closed"
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
	Caps                      Capacity           `json:",omitzero"`
	OwnerInput                string             `json:",omitempty"`
	Quiescence                string             `json:",omitempty"`
	Session                   SessionRecord      `json:",omitzero"`
	Manager                   ManagerRecord      `json:",omitzero"`
	Conclusion                ConclusionRecord   `json:",omitzero"`
	Check                     CheckRecord        `json:",omitzero"`
	Verification              VerificationRecord `json:",omitzero"`
	Publication               PublicationRecord  `json:",omitzero"`
	Answer                    AnswerRecord       `json:",omitzero"`
	managerProof              [32]byte
	replaying                 bool
	binding                   Binding
	sessionPayloads           []Payload
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
	case ExtendAssignment:
		if !capacityPositive(c.Caps) || !boundedInput(c.OwnerInput) {
			return ErrInvalid
		}
		allowed.Caps = c.Caps
		allowed.OwnerInput = c.OwnerInput
	case CloseAssignment:
		if !boundedInput(c.OwnerInput) || !boundedInput(c.Quiescence) {
			return ErrInvalid
		}
		allowed.OwnerInput = c.OwnerInput
		allowed.Quiescence = c.Quiescence
	case PrepareSession, ReadySession, BindSession, PrepareStep, ObserveStep, PrepareTransport, ObserveTransport, RequestTool, ReplyTool, ObserveSessionStop:
		if err := c.Session.validate(c.Kind); err != nil {
			return err
		}
		allowed.Session = c.Session
		if c.Kind == PrepareSession || c.Kind == PrepareStep {
			if err := c.Manager.validate(); err != nil {
				return err
			}
			allowed.Manager = c.Manager
		}
		if c.Kind == PrepareSession || c.Kind == PrepareStep {
			if !c.Reservation.valid() || c.Kind == PrepareSession && c.Reservation.Kind != SessionStart || c.Kind == PrepareStep && c.Reservation.Kind != AgentStep {
				return ErrInvalid
			}
			allowed.Reservation = c.Reservation
		}
	case TaskConclude:
		if err := c.Conclusion.validate(); err != nil {
			return err
		}
		allowed.Conclusion = c.Conclusion
	case PrepareCheck:
		k := c.Check
		if !identifier(string(k.ID)) || !identifier(k.Name) || k.Task != "" && !text(k.Task) || !k.Request.Valid() || k.Tree == (snapshot.TreeID{}) || k.Receipt != (BlobRef{}) || !c.Reservation.valid() || c.Reservation.Kind != ProjectCheck {
			return ErrInvalid
		}
		allowed.Check = k
		allowed.Reservation = c.Reservation
	case ObserveCheck:
		k := c.Check
		if !identifier(string(k.ID)) || !k.Receipt.Valid() || k.Name != "" || k.Task != "" || k.Request != (BlobRef{}) || k.Tree != (snapshot.TreeID{}) || c.Elapsed < 0 || !text(c.Reason) {
			return ErrInvalid
		}
		allowed.Check = k
		allowed.Elapsed = c.Elapsed
		allowed.Reason = c.Reason
	case VerifyRun:
		if c.Verification.Tree == (snapshot.TreeID{}) || slices.ContainsFunc(c.Verification.Checks, func(id CheckID) bool { return !identifier(string(id)) }) {
			return ErrInvalid
		}
		allowed.Verification = c.Verification
	case PreparePublication:
		p := c.Publication
		if !identifier(p.ID) || !p.Changes.Valid() || p.Receipt != (BlobRef{}) || p.Source == (snapshot.TreeID{}) || p.Target == (snapshot.TreeID{}) {
			return ErrInvalid
		}
		allowed.Publication = p
	case AnswerQuestion:
		if !identifier(c.Answer.Question) || !c.Answer.Text.Valid() {
			return ErrInvalid
		}
		allowed.Answer = c.Answer
	case PublishRun:
		p := c.Publication
		if !identifier(p.ID) || !p.Receipt.Valid() || p.Changes != (BlobRef{}) || p.Aggregate != 0 || p.Source != (snapshot.TreeID{}) || p.Target != (snapshot.TreeID{}) || !boundedInput(c.Reason) {
			return ErrInvalid
		}
		allowed.Publication = p
		allowed.Reason = c.Reason
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

// payloadRefs lists every payload a command references, which the journal
// retains before acknowledging its batch.
func (c Command) payloadRefs() []BlobRef {
	refs := append(c.Session.payloadRefs(), c.Conclusion.payloadRefs()...)
	refs = append(refs, c.Check.payloadRefs()...)
	refs = append(refs, c.Answer.payloadRefs()...)
	return append(refs, c.Publication.payloadRefs()...)
}
