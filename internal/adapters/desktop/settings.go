package desktop

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Settings shows how this Markitect runs its agents and the project's
// required checks. Until the Runs phase, the app reads the agents' settings
// from the environment it was started in, as markitect apply does, and
// shows them; it changes none.

// AppSettings is what Settings shows of this Markitect, as the composition
// reads it, without asking the container engine.
type AppSettings struct {
	Agents AgentSettings `json:"agents"`
	Logins []LoginState  `json:"logins"`
}

// AgentSettings are the settings every chat and Run agent starts with.
type AgentSettings struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Effort   string `json:"effort"`
	// Image is the Manager image as set, empty when none is.
	Image   string `json:"image"`
	Network string `json:"network"`
	// Engine is the Docker CLI the agents use, empty when none is found.
	Engine string `json:"engine"`
	// Problems say why no agent can start yet.
	Problems []string `json:"problems"`
}

// LoginState is one provider's account login on this host.
type LoginState struct {
	Provider string `json:"provider"`
	File     string `json:"file"`
	// State is ok, missing or refused; Detail says why it was refused.
	State   string     `json:"state"`
	Detail  string     `json:"detail,omitempty"`
	Expires *time.Time `json:"expires,omitempty"`
	// Command logs in with the provider's own CLI.
	Command string `json:"command"`
	// Used: the agents use this provider.
	Used bool `json:"used"`
}

// projectSettings is what Settings shows of one project.
type projectSettings struct {
	Initialized bool            `json:"initialized"`
	Checks      []project.Check `json:"checks"`
	// Problem says why the required checks cannot be read.
	Problem string `json:"problem,omitempty"`
}

func (s *server) appSettings(w http.ResponseWriter, r *http.Request) {
	if s.settings == nil {
		fail(w, http.StatusNotImplemented, errors.New("this Markitect shows no settings"))
		return
	}
	reply(w, s.settings(r.Context()))
}

func (s *server) projectSettings(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if !ok {
		return
	}
	out := projectSettings{Checks: []project.Check{}}
	ws, err := s.repos.Open(r.Context(), p.Path)
	if err != nil {
		out.Problem = err.Error()
		reply(w, out)
		return
	}
	pr, err := project.Load(ws)
	switch {
	case err != nil:
		out.Problem = err.Error()
	case slices.Contains(pr.Files, project.ManifestFile):
		out.Initialized = true
		if pr.Checks != nil {
			out.Checks = pr.Checks
		}
		for _, d := range pr.Errors {
			if d.File == project.ChecksFile {
				out.Problem = d.String()
				break
			}
		}
	}
	reply(w, out)
}

// settingsFunc reads this Markitect's settings.
type settingsFunc func(ctx context.Context) AppSettings
