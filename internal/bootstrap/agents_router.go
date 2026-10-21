package bootstrap

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

// agentsRouter dispatches each turn to the provider adapter that owns its
// profile. Routes are fixed by the captured settings: a profile without a route
// is refused, never sent to another provider.
type agentsRouter struct {
	routes map[string]runport.Agents
}

var (
	_ runport.Agents       = agentsRouter{}
	_ runport.Environments = agentsRouter{}
)

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

// Release asks every adapter that keeps working environments to end the named
// ones, once per adapter.
func (r agentsRouter) Release(ctx context.Context, names []string) []runport.EnvironmentRelease {
	var done []runport.Agents
	var out []runport.EnvironmentRelease
	for _, agents := range r.routes {
		envs, ok := agents.(runport.Environments)
		if !ok || slices.ContainsFunc(done, func(d runport.Agents) bool { return sameAdapter(d, agents) }) {
			continue
		}
		done = append(done, agents)
		out = append(out, envs.Release(ctx, names)...)
	}
	return out
}

// sameAdapter reports whether two routes hold the same adapter value; values
// of a type that cannot be compared count as different, and a second Release
// of an adapter finds nothing left to end.
func sameAdapter(a, b runport.Agents) bool {
	ta := reflect.TypeOf(a)
	return ta == reflect.TypeOf(b) && ta != nil && ta.Comparable() && a == b
}
