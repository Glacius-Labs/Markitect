package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

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

	SessionsAvailable() bool
	Sessions() []authoring.Session
	Session(string) (authoring.Session, bool)
	StartSession(context.Context, string) (authoring.Session, error)
	PromptSession(string, string) (authoring.Session, error)
	CloseSession(context.Context, string) (authoring.Session, error)
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
	case errors.Is(err, authoring.ErrNotClean), errors.Is(err, authoring.ErrNoExecution), errors.Is(err, authoring.ErrMoved):
		return http.StatusConflict
	case errors.Is(err, authoring.ErrNoChange), errors.Is(err, authoring.ErrOutOfScope):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if !s.authoringOrRefuse(w) {
		return
	}
	writeJSON(w, struct {
		Available bool                `json:"available"`
		Sessions  []authoring.Session `json:"sessions"`
	}{s.authoring.SessionsAvailable(), s.authoring.Sessions()})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if !s.authoringOrRefuse(w) {
		return
	}
	session, ok := s.authoring.Session(r.PathValue("id"))
	if !ok {
		http.Error(w, "no session "+r.PathValue("id"), http.StatusNotFound)
		return
	}
	writeJSON(w, session)
}

// promptOf reads the owner's prompt from the cockpit's own JSON request.
func (s *Server) promptOf(w http.ResponseWriter, r *http.Request, purpose string) (string, bool) {
	if !s.authoringOrRefuse(w) || !s.ownRequest(w, r, purpose) {
		return "", false
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "a prompt is a JSON request", http.StatusForbidden)
		return "", false
	}
	var body struct {
		Prompt string `json:"prompt"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || dec.More() {
		http.Error(w, "the prompt is not a JSON object with prompt", http.StatusBadRequest)
		return "", false
	}
	return body.Prompt, true
}

// handleStartSession starts a model session: it captures the source and
// starts the first provider turn with the owner's prompt.
func (s *Server) handleStartSession(w http.ResponseWriter, r *http.Request) {
	prompt, ok := s.promptOf(w, r, "start")
	if !ok {
		return
	}
	// Preparing runs to its end when the page goes away, within a bound of
	// its own; a closed tab does not abort it half-way.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), sessionPreparation)
	defer cancel()
	session, err := s.authoring.StartSession(ctx, prompt)
	if err != nil {
		http.Error(w, err.Error(), sessionStatus(err))
		return
	}
	writeJSON(w, session)
}

func (s *Server) handlePromptSession(w http.ResponseWriter, r *http.Request) {
	prompt, ok := s.promptOf(w, r, "prompt")
	if !ok {
		return
	}
	session, err := s.authoring.PromptSession(r.PathValue("id"), prompt)
	if err != nil {
		http.Error(w, err.Error(), sessionStatus(err))
		return
	}
	writeJSON(w, session)
}

func (s *Server) handleCloseSession(w http.ResponseWriter, r *http.Request) {
	if !s.authoringOrRefuse(w) || !s.ownRequest(w, r, "close") {
		return
	}
	if _, ok := s.authoring.Session(r.PathValue("id")); !ok {
		http.Error(w, "no session "+r.PathValue("id"), http.StatusNotFound)
		return
	}
	session, err := s.authoring.CloseSession(context.WithoutCancel(r.Context()), r.PathValue("id"))
	if err != nil && session.State != authoring.Closed {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, session)
}

// sessionPreparation bounds capturing the source and materializing a session's
// workspace.
const sessionPreparation = 10 * time.Minute

func sessionStatus(err error) int {
	switch {
	case errors.Is(err, authoring.ErrNoSessions):
		return http.StatusNotImplemented
	case errors.Is(err, authoring.ErrSessionOpen), errors.Is(err, authoring.ErrBusy), errors.Is(err, authoring.ErrLimit), errors.Is(err, authoring.ErrHalted), errors.Is(err, authoring.ErrNotClean), errors.Is(err, authoring.ErrSessionClosed):
		return http.StatusConflict
	case errors.Is(err, authoring.ErrPrompt):
		return http.StatusBadRequest
	case errors.Is(err, authoring.ErrNoSession):
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
