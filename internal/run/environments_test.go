package run

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
)

// keepingAgents is fakeAgents that keeps environments: it records every
// release and reports one kept instance per named environment it saw.
type keepingAgents struct {
	*fakeAgents
	mu       sync.Mutex
	seen     map[string]bool
	releases [][]string
}

func (a *keepingAgents) Run(ctx context.Context, turn Turn, h ToolHandler) (TurnResult, error) {
	a.mu.Lock()
	if turn.Environment != "" {
		a.seen[turn.Environment] = true
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
		}
	}
	return out
}

// Every Manager's work turns share one environment across the work Passes of
// the run, every reviewer has its own for its Verify Pass, and helpers have
// none. A Verify Pass releases its reviewers' environments before its candidate
// volume closes; the run's end releases the rest. Each turn records its
// instance.
func TestEnvironmentsFollowTheirConversationsAndEndWithThem(t *testing.T) {
	f := newFixture(t)
	f.addCheck("unit", ".")
	agents := &keepingAgents{fakeAgents: f.agents, seen: map[string]bool{}}
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
	// The first Verify Pass released exactly its reviewers before its
	// candidate closed; the end of the run released everything else.
	if len(agents.releases) != 3 || agents.releases[2] != nil || len(agents.releases[0]) != len(teamPaths) {
		t.Fatalf("releases %v", agents.releases)
	}
	for i, path := range teamPaths {
		if agents.releases[0][i] != byKind[VerifyTurn][path][0] && !slices.Contains(agents.releases[0], byKind[VerifyTurn][path][0]) {
			t.Errorf("first review release %v lacks %s's reviewer", agents.releases[0], path)
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
	if kinds[len(kinds)-1] != "outcome" || kinds[len(kinds)-2] != "environment.release" {
		t.Errorf("the work environments did not end before the outcome: %v", kinds)
	}
	for _, tr := range out.Turns {
		if tr.Instance.Name == "" || tr.Instance.Generation != 1 {
			t.Errorf("turn %s records no instance: %+v", tr.ID, tr.Instance)
		}
	}
}
