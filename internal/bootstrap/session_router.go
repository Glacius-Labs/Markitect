package bootstrap

import (
	"context"
	"fmt"
	"sync"
	"unicode/utf8"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

type sessionRoute struct {
	plan    runport.SessionPlan
	adapter runport.Sessions
}

type sessionsRouter struct {
	mu            sync.Mutex
	codex, claude runport.Sessions
	routes        map[runport.SessionID]sessionRoute
}

// NewSessionsRouter routes admitted profiles to the explicitly supplied provider
// adapters. A nil adapter refuses that provider. The router retains every route
// for its lifetime, including failed Opens and stopped sessions; it supplies no
// restart or recovery authority. RUN and the adapters own admission and readiness.
func NewSessionsRouter(codex, claude runport.Sessions) runport.Sessions {
	return &sessionsRouter{codex: codex, claude: claude, routes: make(map[runport.SessionID]sessionRoute)}
}

func (r *sessionsRouter) Open(ctx context.Context, a runport.SessionAdmission, io runport.SessionIO) (runport.SessionObservation, error) {
	r.mu.Lock()
	if _, exists := r.routes[a.Plan.ID]; exists {
		r.mu.Unlock()
		return runport.SessionObservation{}, fmt.Errorf("%w: session already routed", runport.ErrConflict)
	}
	if a.Plan.ID == "" || !a.Profile.Valid() || a.Profile.Ref() != a.Plan.Profile || a.Plan.Profile.Encoding != "utf-8" || !utf8.Valid(a.Profile.Bytes()) {
		r.mu.Unlock()
		return runport.SessionObservation{}, fmt.Errorf("%w: invalid session routing profile", runport.ErrInvalid)
	}
	var adapter runport.Sessions
	switch a.Plan.Profile.MediaType {
	case "application/vnd.markitect.codex-session-profile+json":
		adapter = r.codex
	case "application/vnd.markitect.claude-session-profile+json":
		adapter = r.claude
	}
	if adapter == nil {
		r.mu.Unlock()
		return runport.SessionObservation{}, fmt.Errorf("%w: session provider unavailable", runport.ErrInvalid)
	}
	// Open may create effects before returning an error. Bind before dispatch so
	// Stop and Inspect still reach that adapter, and duplicate Open cannot retry.
	r.routes[a.Plan.ID] = sessionRoute{plan: a.Plan, adapter: adapter}
	r.mu.Unlock()
	return adapter.Open(ctx, a, io)
}

func (r *sessionsRouter) route(id runport.SessionID) (sessionRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	route, exists := r.routes[id]
	if !exists {
		return sessionRoute{}, fmt.Errorf("%w: unknown routed session", runport.ErrInvalid)
	}
	return route, nil
}

func (r *sessionsRouter) Step(ctx context.Context, a runport.StepAdmission, io runport.SessionIO) (runport.StepObservation, error) {
	route, err := r.route(a.Plan.Session)
	if err != nil {
		return runport.StepObservation{}, err
	}
	if a.Session != route.plan || a.Plan.SessionBinding != route.plan.Binding() {
		return runport.StepObservation{}, fmt.Errorf("%w: routed session plan changed", runport.ErrConflict)
	}
	return route.adapter.Step(ctx, a, io)
}

func (r *sessionsRouter) Inspect(ctx context.Context, request runport.InspectionRequest) (runport.StepObservation, error) {
	route, err := r.route(request.Session)
	if err != nil {
		return runport.StepObservation{}, err
	}
	return route.adapter.Inspect(ctx, request)
}

func (r *sessionsRouter) Stop(ctx context.Context, request runport.StopRequest, io runport.SessionIO) (runport.StopObservation, error) {
	route, err := r.route(request.Session)
	if err != nil {
		return runport.StopObservation{}, err
	}
	return route.adapter.Stop(ctx, request, io)
}
