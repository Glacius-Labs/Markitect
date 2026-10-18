package run

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var (
	ErrBudget = errors.New("assignment capacity or completion reserve would be exceeded")
	ErrState  = errors.New("transition unavailable in this run state")
)

type RunView struct {
	ID, Assignment    string
	Basis             BlobRef
	State             State
	Outcome           Outcome
	Reason, NextActor string
	StopRequested     bool
}
type operation struct {
	digest  snapshot.ContentID
	receipt AppendReceipt
}
type stateValue struct {
	repository   string
	revision     uint64
	assignment   string
	ownerRequest BlobRef
	// bound is the binding that carries the owner's earlier decisions (DEC-053).
	bound          BlobRef
	boundDecisions []string
	policy         BudgetPolicy
	initialPolicy  BudgetPolicy
	extensions     []AssignmentExtension
	closure        AssignmentClosure
	history        []AssignmentView
	runCapRevision uint64
	current        RunView
	runs           map[string]RunView
	invocations    map[string]Invocation
	operations     map[string]operation
	sessions       map[SessionID]SessionView
	steps          map[StepID]StepView
	transports     map[messageKey]TransportView
	tools          map[toolKey]ToolView
	basisVersion   uint32
	managerBasis   bindingWire
	tasks          map[string]taskValue
	issues         map[string]Issue
	issueOrder     []string
	messages       []Message
	// aggregateVersion counts the deliveries composed into the run's
	// aggregate (core rule D2); aggregateTree is its tree identity.
	aggregateVersion   uint64
	aggregateTree      snapshot.TreeID
	sourceTree         snapshot.TreeID
	checks             map[CheckID]checkValue
	checkOrder         []CheckID
	checkInvocations   map[CheckID]string
	verification       VerificationRecord
	publication        PublicationRecord
	publicationChanges int
}

// Coordinator owns one repository's confirmed projection and at most one
// uncertain append. Its methods are serialized by the Host's single controller;
// it is not a lock, runtime, scheduler or concurrent service. A new instance has
// revision zero and may only be used with an actually new journal stream. Use
// Restore when opening a store whose history has not already been established.
type Coordinator struct {
	state      stateValue
	pending    *Proposal
	recovering bool
}

func NewCoordinator(repository string) (*Coordinator, error) {
	if !text(repository) {
		return nil, ErrInvalid
	}
	return &Coordinator{state: stateValue{repository: repository, runs: map[string]RunView{}, invocations: map[string]Invocation{}, operations: map[string]operation{}, sessions: map[SessionID]SessionView{}, steps: map[StepID]StepView{}, transports: map[messageKey]TransportView{}, tools: map[toolKey]ToolView{}}}, nil
}
func (c *Coordinator) Current() RunView { return c.state.current }
func (c *Coordinator) Revision() uint64 { return c.state.revision }

// RecoveryRequired is a Host observation barrier, separate from committed run
// state. Restore keeps it while any recorded invocation still needs explicit
// reconciliation. Reading alone never settles or releases a reservation.
func (c *Coordinator) RecoveryRequired() bool { return c.recovering }
func (c *Coordinator) Budget() BudgetView {
	return budget(c.state.policy, int64(len(c.state.runs)), c.state.invocations)
}
func (c *Coordinator) Invocations() []Invocation {
	var out []Invocation
	for _, v := range c.state.invocations {
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b Invocation) int {
		if a.Reservation.ID < b.Reservation.ID {
			return -1
		}
		if a.Reservation.ID > b.Reservation.ID {
			return 1
		}
		return 0
	})
	return out
}

// Pending reports an observation limit, not an invented committed run state.
// No dependent proposal is permitted while an append outcome is unconfirmed.
func (c *Coordinator) Pending() (AppendBatch, bool) {
	if c.pending == nil {
		return AppendBatch{}, false
	}
	return c.pending.batch, true
}

type Proposal struct {
	batch    AppendBatch
	next     stateValue
	command  Command
	payloads []Payload
	replay   *AppendReceipt
}

func (p Proposal) Batch() AppendBatch { return p.batch }

// Propose has no journal or external effect. In particular a Prepared proposal
// does not expose a prepared run or consume a run unit on this Coordinator.
func (c *Coordinator) Propose(command Command) (Proposal, error) {
	if c.pending != nil {
		return Proposal{}, ErrUncertain
	}
	if err := command.validate(); err != nil {
		return Proposal{}, err
	}
	data, _ := json.Marshal(command)
	digest := snapshot.ContentID(sha256.Sum256(data))
	if previous, ok := c.state.operations[command.ID]; ok {
		if previous.digest != digest {
			return Proposal{}, ErrConflict
		}
		r := previous.receipt
		return Proposal{command: command, replay: &r}, nil
	}
	if c.recovering && command.Kind != SettleInvocation && command.Kind != ReleaseUndispatched && command.Kind != ObserveUncertain && command.Kind != StopRun && !sessionReconciliation(command.Kind) && !(command.Kind == PrepareTransport && command.Session.Control) {
		return Proposal{}, ErrRecovery
	}
	if !command.replaying {
		if err := c.freshTaskAdmission(command, digest); err != nil {
			return Proposal{}, err
		}
	}
	next := c.state
	next.runs = maps.Clone(c.state.runs)
	next.invocations = maps.Clone(c.state.invocations)
	next.operations = maps.Clone(c.state.operations)
	next.extensions = slices.Clone(c.state.extensions)
	next.history = slices.Clone(c.state.history)
	next.sessions = maps.Clone(c.state.sessions)
	next.steps = maps.Clone(c.state.steps)
	next.transports = maps.Clone(c.state.transports)
	next.tools = maps.Clone(c.state.tools)
	next.tasks = maps.Clone(c.state.tasks)
	next.issues = maps.Clone(c.state.issues)
	next.issueOrder = slices.Clone(c.state.issueOrder)
	next.messages = slices.Clone(c.state.messages)
	next.checks = maps.Clone(c.state.checks)
	next.checkOrder = slices.Clone(c.state.checkOrder)
	next.checkInvocations = maps.Clone(c.state.checkInvocations)
	if err := next.apply(command); err != nil {
		return Proposal{}, err
	}
	var events []Event
	if command.Kind == PrepareRun && (c.state.assignment == "" || c.state.closure.CommandID != "") {
		events = append(events, Event{ID: fmt.Sprintf("%x", sha256.Sum256(append(data, byte(0)))), Kind: "assignment.created", Body: command})
	}
	if command.Kind == PrepareSession || command.Kind == PrepareStep || command.Kind == PrepareCheck {
		events = append(events, Event{ID: fmt.Sprintf("%x", sha256.Sum256(append(data, byte(2)))), Kind: ReserveInvocation, Body: command})
	}
	events = append(events, Event{ID: fmt.Sprintf("%x", sha256.Sum256(append(data, byte(1)))), Kind: eventTag(command.Kind), Body: command})
	batch, err := newBatch(command.ID, c.state.revision, events)
	if err != nil {
		return Proposal{}, err
	}
	next.revision = batch.Revision()
	next.operations[command.ID] = operation{digest, ReceiptFor(batch)}
	next.sessionReceipt(command, ReceiptFor(batch))
	p := Proposal{batch: batch, next: next, command: command}
	if command.Kind == PrepareRun {
		p.payloads = []Payload{command.binding.owner, command.binding.payload}
	}
	p.payloads = append(p.payloads, command.sessionPayloads...)
	return p, nil
}

func (s *stateValue) apply(c Command) error {
	if c.Kind == PrepareRun {
		b := c.binding
		if !b.Valid() || b.Ref() != c.Basis || b.source.Repository != s.repository {
			return ErrInvalid
		}
		if _, exists := s.runs[c.Run]; exists {
			return ErrConflict
		}
		for _, prior := range s.history {
			if prior.ID == c.Assignment {
				return ErrState
			}
			for _, priorRun := range prior.Runs {
				if priorRun.ID == c.Run {
					return ErrConflict
				}
			}
		}
		if s.assignment == "" || s.closure.CommandID != "" {
			if c.Predecessor != "" {
				return ErrInvalid
			}
			if s.assignment == c.Assignment {
				return ErrState
			}
			if s.assignment != "" {
				s.history = append(s.history, s.assignmentView())
			}
			s.assignment = c.Assignment
			s.ownerRequest = b.owner.Ref()
			s.policy = b.policy
			s.initialPolicy = b.policy
			s.extensions = nil
			s.closure = AssignmentClosure{}
			s.runs = map[string]RunView{}
			s.invocations = map[string]Invocation{}
		} else {
			if s.assignment != c.Assignment || s.current.State != Finished || c.Predecessor != s.current.ID || c.Basis == s.current.Basis && s.runCapRevision == s.capRevision() || s.initialPolicy != b.policy || s.ownerRequest != b.owner.Ref() {
				return ErrState
			}
		}
		// Each run binds its own owner decisions, so a continuation can carry
		// decisions the owner made after the earlier run (DEC-053).
		s.bound, s.boundDecisions = b.payload.Ref(), slices.Clone(b.decisions)
		view := budget(s.policy, int64(len(s.runs)), s.invocations)
		// Assessment may spend the configured reserve. Each new basis still
		// needs its own structural review floor in the shared remaining ledger.
		if !reviewCapacity(view, b.demand) {
			return ErrBudget
		}
		s.basisVersion = b.version
		if err := s.prepareTasks(b, c.Run); err != nil {
			return err
		}
		s.current = RunView{ID: c.Run, Assignment: c.Assignment, Basis: c.Basis, State: Prepared}
		s.runCapRevision = s.capRevision()
		s.runs[c.Run] = s.current
		return nil
	}
	if s.assignment != c.Assignment || s.current.ID != c.Run || s.closure.CommandID != "" {
		return ErrState
	}
	if c.Kind == ExtendAssignment || c.Kind == CloseAssignment {
		return s.amendAssignment(c)
	}
	if s.current.State == Finished {
		return ErrState
	}
	if c.Session.Session != "" {
		if err := s.beforeTaskSession(c); err != nil {
			return err
		}
		if err := s.applySession(c); err != nil {
			return err
		}
		return s.afterTaskSession(c)
	}
	if c.Kind == TaskConclude {
		if s.current.State != Running {
			return ErrState
		}
		return s.concludeStep(c)
	}
	switch c.Kind {
	case PrepareCheck, ObserveCheck, VerifyRun, PreparePublication, PublishRun:
		return s.applyVerification(c)
	case AnswerQuestion:
		return s.answer(c)
	}
	r := s.current
	switch c.Kind {
	case StartRun:
		if r.State != Prepared {
			return ErrState
		}
		r.State = Running
	case ReserveInvocation:
		if s.basisVersion == 3 {
			return ErrState
		}
		if err := s.reserve(c.Reservation); err != nil {
			return err
		}
	case SettleInvocation, ObserveUncertain, ReleaseUndispatched:
		invocation, ok := s.invocations[c.Invocation]
		if !ok || !open(invocation) {
			return ErrState
		}
		if !s.sessionSettlementAllowed(c.Invocation, c.Kind) {
			return ErrState
		}
		if c.Kind == ObserveUncertain {
			invocation.State = Uncertain
			r.State = Reconciling
			r.Reason = c.Reason
		} else if c.Kind == ReleaseUndispatched {
			invocation.State = Released
			invocation.Observation = c.Reason
			for id, step := range s.steps {
				if step.Invocation == c.Invocation && !step.Terminal {
					step.Terminal = true
					s.steps[id] = step
				}
			}
		} else {
			invocation.State = Settled
			invocation.Elapsed = c.Elapsed
			invocation.Observation = c.Reason
		}
		s.invocations[c.Invocation] = invocation
	case WaitRun:
		if r.State != Running || s.hasOpen() {
			return ErrState
		}
		r.State = Waiting
		r.Reason = c.Reason
		r.NextActor = c.NextActor
	case ResumeRun:
		if (r.State != Waiting && r.State != Reconciling) || s.hasOpen() || r.StopRequested || budget(s.policy, int64(len(s.runs)), s.invocations).Overrun {
			return ErrState
		}
		r.State = Running
		r.Reason = ""
		r.NextActor = ""
	case StopRun:
		r.StopRequested = true
		r.State = Reconciling
		r.Reason = c.Reason
		r.NextActor = ""
	case FinishRun:
		if s.hasOpen() || s.hasLiveSessions() || c.Outcome == Stopped && !r.StopRequested || s.publication.ID != "" && s.publication.Receipt == (BlobRef{}) {
			return ErrState
		}
		r.State = Finished
		r.Outcome = c.Outcome
		r.Reason = c.Reason
		r.NextActor = ""
	default:
		return ErrState
	}
	s.current = r
	s.runs[r.ID] = r
	return nil
}
func open(invocation Invocation) bool {
	return invocation.State == Reserved || invocation.State == Uncertain
}
func (s stateValue) hasOpen() bool {
	if s.hasSessionActivity() {
		return true
	}
	for _, invocation := range s.invocations {
		if open(invocation) {
			return true
		}
	}
	return false
}
