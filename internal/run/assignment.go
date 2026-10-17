package run

import (
	"slices"
	"time"
)

// OwnerInput retains the exact raw UTF-8 extension or closure input, up to 64 KiB.
// It is a Host-supplied observation, not authenticated owner authority. Quiescence
// is a separate explicit Host observation; RUN cannot establish physical runtime
// termination. Neither cancellation nor released budget alone supplies it.
type AssignmentExtension struct {
	CommandID   string
	ObservedAt  time.Time
	CapRevision uint64
	Caps        Capacity
	OwnerInput  string
}

type AssignmentClosure struct {
	CommandID              string
	ObservedAt             time.Time
	LastRun                string
	Outcome                Outcome
	OwnerInput, Quiescence string
}

// AssignmentView is a copied projection. Historical ledgers never contribute
// capacity or authority to the next ordinary assignment on the same stream.
type AssignmentView struct {
	ID                    string
	OwnerRequest          BlobRef
	InitialPolicy, Policy BudgetPolicy
	CapRevision           uint64
	Budget                BudgetView
	Runs                  []RunView
	Invocations           []Invocation
	Extensions            []AssignmentExtension
	Closure               AssignmentClosure
}

// Assignments returns creation-ordered closed history followed by the current
// assignment (which can itself be closed). All slice storage is copied.
func (c *Coordinator) Assignments() []AssignmentView {
	out := make([]AssignmentView, 0, len(c.state.history)+1)
	for _, prior := range c.state.history {
		out = append(out, copyAssignment(prior))
	}
	if c.state.assignment != "" {
		out = append(out, c.state.assignmentView())
	}
	return out
}

func copyAssignment(v AssignmentView) AssignmentView {
	v.Runs = slices.Clone(v.Runs)
	v.Invocations = slices.Clone(v.Invocations)
	v.Extensions = slices.Clone(v.Extensions)
	return v
}

func (s stateValue) capRevision() uint64 {
	if s.assignment == "" {
		return 0
	}
	return 1 + uint64(len(s.extensions))
}

func (s stateValue) assignmentView() AssignmentView {
	v := AssignmentView{ID: s.assignment, OwnerRequest: s.ownerRequest,
		InitialPolicy: s.initialPolicy, Policy: s.policy, CapRevision: s.capRevision(),
		Budget:     budget(s.policy, int64(len(s.runs)), s.invocations),
		Extensions: slices.Clone(s.extensions), Closure: s.closure}
	for _, r := range s.runs {
		v.Runs = append(v.Runs, r)
	}
	slices.SortFunc(v.Runs, func(a, b RunView) int { return compareID(a.ID, b.ID) })
	for _, i := range s.invocations {
		v.Invocations = append(v.Invocations, i)
	}
	slices.SortFunc(v.Invocations, func(a, b Invocation) int { return compareID(a.Reservation.ID, b.Reservation.ID) })
	return v
}

func compareID(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func (s *stateValue) amendAssignment(c Command) error {
	switch c.Kind {
	case ExtendAssignment:
		// The target is absolute. No arithmetic can overflow or replenish spent
		// entries, holds, completion reserve or an overrun observation.
		if !fits(s.policy.Assignment, c.Caps) || !fits(c.Caps, s.initialPolicy.Ceilings) || c.Caps == s.policy.Assignment {
			return ErrBudget
		}
		s.policy.Assignment = c.Caps
		s.extensions = append(s.extensions, AssignmentExtension{c.ID, c.ObservedAt, s.capRevision() + 1, c.Caps, c.OwnerInput})
	case CloseAssignment:
		if s.current.State != Finished || s.hasOpen() {
			return ErrState
		}
		s.closure = AssignmentClosure{c.ID, c.ObservedAt, s.current.ID, s.current.Outcome, c.OwnerInput, c.Quiescence}
	default:
		return ErrInvalid
	}
	return nil
}

func capacityPositive(c Capacity) bool {
	return c.Runs > 0 && c.SessionStarts > 0 && c.Steps > 0 && c.ExecutionTime > 0
}
func boundedInput(s string) bool { return len(s) <= 64<<10 && text(s) }
