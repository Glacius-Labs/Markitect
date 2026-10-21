package cockpit

import "github.com/Glacius-Labs/Markitect/internal/authoring"

// Busy says why this project must not be left now, or is empty when it can
// be: a run this cockpit started is active, a model session prepares or works
// (or closes), or an accepted proposal is being written. Leaving it ends its
// idle model sessions, as a stopping cockpit does, and nothing else.
func (s *Server) Busy() string {
	if s.runs != nil {
		if run := s.runs.Status(); run != nil && run.State != RunEnded {
			return "a run started in this cockpit is still active; stop it or wait for its end"
		}
	}
	if s.authoring != nil {
		for _, session := range s.authoring.Sessions() {
			switch session.State {
			case authoring.Preparing, authoring.Working, authoring.Closing:
				return "a model session is working; wait for its turn to end"
			}
		}
		for _, proposal := range s.authoring.Proposals() {
			if proposal.State == authoring.Publishing {
				return "an accepted proposal is being written; wait for its receipt"
			}
		}
	}
	return ""
}
