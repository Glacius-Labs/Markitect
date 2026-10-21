package cockpit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ProjectInfo is the identity of the repository the cockpit was started in,
// as structural preparation observed it: the bound model and source.
type ProjectInfo struct {
	Repository  string    `json:"repository"`
	ModelID     string    `json:"modelID,omitempty"`
	ModelDigest string    `json:"modelDigest,omitempty"`
	Head        string    `json:"head,omitempty"`
	Packages    int       `json:"packages"`
	Definitions int       `json:"definitions"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	ObservedAt  time.Time `json:"observedAt"`
	// Model is served on its own route; the project summary stays small.
	Model *ModelView `json:"-"`
}

// ProjectObserver prepares the project's identity. Bootstrap supplies the
// product's structural preparation; it reads and writes nothing else.
type ProjectObserver func(context.Context) (ProjectInfo, error)

// projectCache keeps the last observation. Structural preparation of a
// repository takes seconds, so it runs in the background: a request returns
// the last observation and whether a new one is under way.
type projectCache struct {
	mu        sync.Mutex
	value     *ProjectInfo
	observing bool
	observe   ProjectObserver
}

// get returns the last observation and starts one when there is none yet or
// a refresh is asked for.
func (c *projectCache) get(refresh bool) (*ProjectInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.observe == nil {
		return nil, false
	}
	if (c.value == nil || refresh) && !c.observing {
		c.observing = true
		go c.run()
	}
	return c.value, c.observing
}

// observeTimeout bounds one structural preparation of the repository.
const observeTimeout = 2 * time.Minute

func (c *projectCache) run() {
	p := ProjectInfo{}
	defer func() {
		if r := recover(); r != nil {
			p.Status, p.Error = "not_completed", fmt.Sprint("the observation failed: ", r)
		}
		p.ObservedAt = time.Now().UTC()
		c.mu.Lock()
		c.value, c.observing = &p, false
		c.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), observeTimeout)
	defer cancel()
	var err error
	p, err = c.observe(ctx)
	if err != nil {
		p.Error = err.Error()
	}
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	s.writeProject(w, false)
}

// handleObserve runs the repository's structural preparation again. It is a
// POST with the cockpit's own header, so another page cannot start one.
func (s *Server) handleObserve(w http.ResponseWriter, r *http.Request) {
	if !s.ownRequest(w, r, "observe") {
		return
	}
	s.writeProject(w, true)
}

func (s *Server) writeProject(w http.ResponseWriter, refresh bool) {
	p, observing := s.project.get(refresh)
	writeJSON(w, struct {
		Project    *ProjectInfo `json:"project"`
		Observing  bool         `json:"observing"`
		Configured bool         `json:"configured"`
		States     []string     `json:"states"`
		Proposals  bool         `json:"proposals"`
		Sessions   bool         `json:"sessions"`
	}{p, observing, s.project.observe != nil, s.stateDirs(), s.authoring != nil, s.authoring != nil && s.authoring.SessionsAvailable()})
}

func (s *Server) stateDirs() []string {
	out := make([]string, len(s.states))
	for i, st := range s.states {
		out[i] = st.dir
	}
	return out
}

// requestHeader must accompany every write. A cross-site form cannot send it,
// and a cross-site script cannot send it without a preflight the cockpit
// refuses.
const requestHeader = "X-Markitect-Cockpit"

// ownRequest admits a write only from the cockpit's own page: its header with
// the expected purpose and, when the browser names one, its own origin.
func (s *Server) ownRequest(w http.ResponseWriter, r *http.Request, purpose string) bool {
	return checkOwnRequest(w, r, purpose, s.admitHost)
}

// checkOwnRequest admits a write only when it carries the cockpit's own
// header for its purpose, which a cross-site form cannot send, and no foreign
// origin.
func checkOwnRequest(w http.ResponseWriter, r *http.Request, purpose string, admit func(host string) bool) bool {
	if r.Header.Get(requestHeader) != purpose {
		http.Error(w, "a write needs the cockpit's own request", http.StatusForbidden)
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" && !admit(strings.TrimPrefix(origin, "http://")) {
		http.Error(w, "foreign origin", http.StatusForbidden)
		return false
	}
	return true
}
