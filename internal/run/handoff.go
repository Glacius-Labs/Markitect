package run

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
	h := s.handoff
	h.Manager = s.manager.Package
	h.Base = s.base
	h.Result = commit
	h.Findings = append(h.Findings, response.Findings...)
	h.Decision = response.Decision
	if adverse := requiredDecision(s.lateDecision, child); adverse != "" {
		h.Decision = adverse
	}
	s.handoff = h
	return h
}
