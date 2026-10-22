package bootstrap

import (
	"context"
	"fmt"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

// agentsRouter dispatches each turn to the provider adapter that owns its
// profile. Routes are fixed by the captured settings: a profile without a route
// is refused, never sent to another provider.
type agentsRouter struct {
	routes map[string]runport.Agents
}

// assignmentAgentsRouter uses the same captured profile routing for persistent
// Manager and independent reviewer conversations.
type assignmentAgentsRouter struct {
	routes map[string]runport.AssignmentAgents
}

var _ runport.AssignmentAgents = assignmentAgentsRouter{}

func (r assignmentAgentsRouter) OpenAssignment(ctx context.Context, spec runport.AssignmentSpec, handler runport.ToolHandler) (runport.AssignmentSession, error) {
	agents, found := r.routes[spec.Profile]
	if !found || agents == nil {
		return nil, fmt.Errorf("%w: assignment %s names profile %q without an Agent adapter", runport.ErrInvalid, spec.ID, spec.Profile)
	}
	return agents.OpenAssignment(ctx, spec, handler)
}

var _ runport.Agents = agentsRouter{}

func newAgentsRouter(routes map[string]runport.Agents) runport.Agents {
	copied := make(map[string]runport.Agents, len(routes))
	for profile, agents := range routes {
		copied[profile] = agents
	}
	return agentsRouter{routes: copied}
}

func (r agentsRouter) Run(ctx context.Context, turn runport.Turn, tools runport.ToolHandler) (runport.TurnResult, error) {
	agents, found := r.routes[turn.Profile]
	if !found || agents == nil {
		// No container was started, so there is nothing that could still run.
		return runport.TurnResult{Stopped: true, Reason: "no Agent adapter for the turn profile"}, fmt.Errorf("%w: turn %s names profile %q, which has no Agent adapter", runport.ErrInvalid, turn.ID, turn.Profile)
	}
	return agents.Run(ctx, turn, tools)
}
