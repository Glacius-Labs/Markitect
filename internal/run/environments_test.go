package run

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
)

// keepingAgents is fakeAgents that keeps environments: it records every
// release and reports one kept instance per named environment it saw, which
// mounts its turn's volume until its release, as a held container does.
type keepingAgents struct {
	*fakeAgents
	mu       sync.Mutex
	seen     map[string]bool
	volumes  map[string]string
	releases [][]string
}

func (a *keepingAgents) Run(ctx context.Context, turn Turn, h ToolHandler) (TurnResult, error) {
	a.mu.Lock()
	if turn.Environment != "" && !a.seen[turn.Environment] {
		a.seen[turn.Environment] = true
		a.volumes[turn.Environment] = turn.Workspace
		a.ws.mu.Lock()
		a.ws.mounted[turn.Workspace]++
		a.ws.mu.Unlock()
	}
	a.mu.Unlock()
	res, err := a.fakeAgents.Run(ctx, turn, h)
	res.Instance = InstanceObservation{Name: "inst-" + turn.Environment, Generation: 1, Started: "first"}
	return res, err
}

func (a *keepingAgents) Release(_ context.Context, names []string) []EnvironmentRelease {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.releases = append(a.releases, slices.Clone(names))
	var out []EnvironmentRelease
	for name := range a.seen {
		if names == nil || slices.Contains(names, name) {
			out = append(out, EnvironmentRelease{Environment: name, Instance: "inst-" + name, Removed: true})
			delete(a.seen, name)
			a.ws.mu.Lock()
			a.ws.mounted[a.volumes[name]]--
			a.ws.mu.Unlock()
		}
	}
	return out
}

// Every Manager's work turns share one named environment, every reviewer has
// its own for its Verify Pass, and helpers have none. A work Pass releases the
// work environments before it captures the shared tree, whose capture admits
// no mounting container; a Verify Pass releases its reviewers' environments
// before its candidate volume closes; the run's end releases the rest. Each
// turn records its instance.
func TestEnvironmentsFollowTheirConversationsAndEndWithThem(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", ".")
	agents := &keepingAgents{fakeAgents: f.agents, seen: map[string]bool{}, volumes: map[string]string{}}
	f.cfg.Agents = agents
	r := f.open()
	for _, spec := range []PassSpec{implementSpec(), verifySpec(), implementSpec(), verifySpec()} {
		if res := r.Pass(spec); res.Stopped {
			t.Fatalf("pass %s stopped", spec.Purpose)
		}
	}
	out := f.finish(r, "")
	byKind := map[TurnKind]map[string][]string{ExecuteTurn: {}, VerifyTurn: {}}
	for _, turn := range f.agents.turns {
		if turn.Environment != turn.Home || turn.Environment == "" {
			t.Errorf("turn %s of %s: environment %q, home %q", turn.ID, turn.Manager, turn.Environment, turn.Home)
		}
		envs := byKind[turn.Kind]
		if !slices.Contains(envs[turn.Manager], turn.Environment) {
			envs[turn.Manager] = append(envs[turn.Manager], turn.Environment)
		}
	}
	for _, path := range teamPaths {
		if work := byKind[ExecuteTurn][path]; len(work) != 1 || !strings.HasSuffix(work[0], "-m") {
			t.Errorf("%s works in %v, not one environment for the run", path, work)
		}
		review := byKind[VerifyTurn][path]
		if len(review) != 2 || review[0] == review[1] || slices.Contains(byKind[ExecuteTurn][path], review[0]) {
			t.Errorf("%s reviews in %v, not one fresh environment per Verify Pass", path, review)
		}
	}
	// Each work Pass released the work environments before its capture, each
	// Verify Pass exactly its reviewers before its candidate closed; the end
	// of the run released everything else.
	if len(agents.releases) != 5 || agents.releases[4] != nil {
		t.Fatalf("releases %v", agents.releases)
	}
	for _, i := range []int{0, 2} {
		for _, path := range teamPaths {
			if !slices.Contains(agents.releases[i], byKind[ExecuteTurn][path][0]) {
				t.Errorf("work release %d %v lacks %s's work environment", i, agents.releases[i], path)
			}
		}
	}
	for _, path := range teamPaths {
		if !slices.Contains(agents.releases[1], byKind[VerifyTurn][path][0]) {
			t.Errorf("first review release %v lacks %s's reviewer", agents.releases[1], path)
		}
	}
	var kinds []string
	for _, e := range f.ev.events {
		if e.Kind == "environment.release" || e.Kind == "workspace.close" || e.Kind == "outcome" {
			kinds = append(kinds, e.Kind)
		}
	}
	first := slices.Index(kinds, "workspace.close")
	if first < len(teamPaths) || slices.ContainsFunc(kinds[:first], func(k string) bool { return k != "environment.release" }) {
		t.Errorf("the reviewers' environments did not end before their candidate: %v", kinds)
	}
	if kinds[len(kinds)-1] != "outcome" || len(agents.seen) != 0 {
		t.Errorf("environments outlived the run: %v, left %v", kinds, agents.seen)
	}
	for _, tr := range out.Turns {
		if tr.Instance.Name == "" || tr.Instance.Generation != 1 {
			t.Errorf("turn %s records no instance: %+v", tr.ID, tr.Instance)
		}
	}
}
