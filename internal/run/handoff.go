package run

import (
	"fmt"
	"slices"
)

// requiredDecision propagates required adverse findings without treating a
// previous ordinary submission as fresh acceptance during correction.
func requiredDecision(current, next Decision) Decision {
	if current == ModelBlocker || next == ModelBlocker {
		return ModelBlocker
	}
	if current == Reject || next == Reject {
		return Reject
	}
	return current
}

// finish serializes candidate completion with late findings. Child summaries
// are read before this owner lock, preserving the child-to-parent lock order.
func (s *managerState) finish(commit Commit, response Response, child Decision) Handoff {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.judgment = response.Decision
	h := s.handoff
	h.Manager = s.manager.Package
	h.Base = s.base
	h.Result = commit
	h.Findings = append(h.Findings, response.Findings...)
	h.Decision = requiredDecision(response.Decision, requiredDecision(s.lateDecision, child))
	s.handoff = h
	return h
}

// Recompute current outstanding judgments after the full hierarchy drains.
// Corrected and newly approved work replaces Reject; ModelBlocker stays terminal.
func (x *execution) aggregate(s *managerState) (Decision, error) {
	s.mu.Lock()
	decision := s.judgment
	late := s.lateDecision
	s.mu.Unlock()
	adverse := late
	for _, name := range s.manager.Children {
		child := x.states[name]
		value, err := x.aggregate(child)
		if err != nil {
			return "", err
		}
		child.mu.Lock()
		review := child.handoff.Review
		working := child.working
		if review == nil || working {
			child.mu.Unlock()
			return "", fmt.Errorf("%w: child %s requires renewed parent review", ErrObservationIncomplete, name)
		}
		adverse = requiredDecision(adverse, value)
		adverse = requiredDecision(adverse, review.Decision)
		child.mu.Unlock()
	}
	decision = requiredDecision(decision, adverse)
	s.mu.Lock()
	s.handoff.Decision = decision
	s.mu.Unlock()
	return decision, nil
}

func copyChecks(observed []CheckObservation) []CheckObservation {
	copied := slices.Clone(observed)
	for i := range copied {
		copied[i].Receipt.Output = slices.Clone(copied[i].Receipt.Output)
		copied[i].Receipt.Directories = slices.Clone(copied[i].Receipt.Directories)
	}
	return copied
}

// The terminal result owns every mutable value passed to persistence and the
// caller. Bounded cleanup may leave a worker alive; it owns no part of this value.
func copyResult(r Result) Result {
	r.Sessions = slices.Clone(r.Sessions)
	r.Findings = slices.Clone(r.Findings)
	r.Handoffs = slices.Clone(r.Handoffs)
	for i := range r.Handoffs {
		h := &r.Handoffs[i]
		h.Children = slices.Clone(h.Children)
		h.Findings = slices.Clone(h.Findings)
		if h.Review != nil {
			review := *h.Review
			review.Findings = slices.Clone(review.Findings)
			h.Review = &review
		}
	}
	r.Checks = copyChecks(r.Checks)
	r.Cleanup = slices.Clone(r.Cleanup)
	r.Errors = slices.Clone(r.Errors)
	if r.Candidate != nil {
		ref := *r.Candidate
		r.Candidate = &ref
	}
	return r
}
