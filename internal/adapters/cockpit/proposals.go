package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/authoring"
)

// Authoring makes and decides the owner's proposals for the project's
// .markitect inputs. Bootstrap supplies the service over the repository's
// committed capture and guarded publisher.
type Authoring interface {
	Settings(context.Context) (authoring.Settings, error)
	ProposeSettings(context.Context, authoring.Settings) (authoring.Proposal, error)
	Proposals() []authoring.Proposal
	Proposal(string) (authoring.Proposal, bool)
	Accept(context.Context, string) (authoring.Proposal, error)
	Discard(string) (authoring.Proposal, error)
}

func (s *Server) authoringOrRefuse(w http.ResponseWriter) bool {
	if s.authoring == nil {
		http.Error(w, "proposals need the cockpit to be started in a repository (--repo)", http.StatusNotImplemented)
		return false
	}
	return true
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if !s.authoringOrRefuse(w) {
		return
	}
	settings, err := s.authoring.Settings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), proposalStatus(err))
		return
	}
	writeJSON(w, settings)
}

func (s *Server) handleProposeSettings(w http.ResponseWriter, r *http.Request) {
	if !s.authoringOrRefuse(w) || !s.ownRequest(w, r, "propose") {
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "a proposal is a JSON request", http.StatusForbidden)
		return
	}
	var edit authoring.Settings
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&edit); err != nil || dec.More() {
		http.Error(w, "the settings are not a JSON object", http.StatusBadRequest)
		return
	}
	proposal, err := s.authoring.ProposeSettings(r.Context(), edit)
	if err != nil {
		http.Error(w, err.Error(), proposalStatus(err))
		return
	}
	writeJSON(w, proposal)
}

func (s *Server) handleProposals(w http.ResponseWriter, r *http.Request) {
	if !s.authoringOrRefuse(w) {
		return
	}
	writeJSON(w, s.authoring.Proposals())
}

func (s *Server) handleProposal(w http.ResponseWriter, r *http.Request) {
	if !s.authoringOrRefuse(w) {
		return
	}
	proposal, ok := s.authoring.Proposal(r.PathValue("id"))
	if !ok {
		http.Error(w, "no proposal "+r.PathValue("id"), http.StatusNotFound)
		return
	}
	writeJSON(w, proposal)
}

// handleDecide accepts or discards a proposal. An accepted proposal that the
// publication could not apply comes back with its failed state and reason.
func (s *Server) handleDecide(w http.ResponseWriter, r *http.Request) {
	decision := r.PathValue("decision")
	if decision != "accept" && decision != "discard" {
		http.Error(w, "a proposal is accepted or discarded", http.StatusNotFound)
		return
	}
	if !s.authoringOrRefuse(w) || !s.ownRequest(w, r, decision) {
		return
	}
	id := r.PathValue("id")
	if _, ok := s.authoring.Proposal(id); !ok {
		http.Error(w, "no proposal "+id, http.StatusNotFound)
		return
	}
	var proposal authoring.Proposal
	var err error
	if decision == "accept" {
		// A started publication runs to its receipt even when the page that
		// asked for it goes away; interrupting it could leave a partial write.
		proposal, err = s.authoring.Accept(context.WithoutCancel(r.Context()), id)
	} else {
		proposal, err = s.authoring.Discard(id)
	}
	if err != nil && proposal.State != authoring.Failed {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, proposal)
}

func proposalStatus(err error) int {
	switch {
	case errors.Is(err, authoring.ErrNotClean), errors.Is(err, authoring.ErrNoExecution):
		return http.StatusConflict
	case strings.Contains(err.Error(), "changes nothing"), strings.Contains(err.Error(), "may not change"):
		return http.StatusBadRequest
	}
	return http.StatusUnprocessableEntity
}
