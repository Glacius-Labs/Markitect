package run

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// lineGit is a target whose history is one line of landings l1 (oldest) to
// lN, each with its message, and a set of Run branches.
type lineGit struct {
	Git
	n        int
	messages map[string]string
	branches map[string]string
}

func (g *lineGit) index(id string) int {
	var i int
	fmt.Sscanf(id, "l%d", &i)
	return i
}

func (g *lineGit) Landings(_ context.Context, tip, since string) ([]string, error) {
	var out []string
	for i := g.index(tip); i > g.index(since); i-- {
		out = append(out, fmt.Sprintf("l%d", i))
	}
	return out, nil
}

func (g *lineGit) Commit(_ context.Context, id string) (Commit, error) {
	return Commit{ID: id, Message: g.messages[id]}, nil
}

func (g *lineGit) Contains(_ context.Context, tip, commit string) (bool, error) {
	return strings.HasPrefix(commit, "l") && g.index(commit) <= g.index(tip), nil
}

func (g *lineGit) Branches(context.Context, string) (map[string]string, error) {
	return g.branches, nil
}

func landingLine(n int) *lineGit {
	g := &lineGit{n: n, messages: map[string]string{}, branches: map[string]string{}}
	for i := 1; i <= n; i++ {
		g.messages[fmt.Sprintf("l%d", i)] = fmt.Sprintf("Markitect Run r%d: landed\n\nRun-Name: r%d\nRun-Work: w%d\n", i, i, i)
	}
	return g
}

func TestTheFullCheckCount(t *testing.T) {
	ctx := context.Background()
	count := func(g *lineGit, tip string) FullCheck {
		t.Helper()
		e := &Engine{Git: g}
		fc := e.fullCheck(ctx, "main", tip)
		if fc.Note != "" {
			t.Fatal(fc.Note)
		}
		return fc
	}

	// Without a full check every landing counts, and from FullCheckEvery
	// on a full check is due.
	g := landingLine(FullCheckEvery)
	if fc := count(g, fmt.Sprintf("l%d", FullCheckEvery-1)); fc.Landed != FullCheckEvery-1 || fc.Due || fc.Since != "" {
		t.Fatalf("before: %+v", fc)
	}
	if fc := count(g, fmt.Sprintf("l%d", FullCheckEvery)); fc.Landed != FullCheckEvery || !fc.Due {
		t.Fatalf("at the limit: %+v", fc)
	}

	// A landed apply --full starts the count again at its own landing.
	g = landingLine(12)
	g.messages["l4"] += "Run-Full: true\n"
	if fc := count(g, "l12"); fc.Landed != 8 || fc.Since != "l4" || fc.Due {
		t.Fatalf("after apply --full: %+v", fc)
	}

	// A verify --full that ended verified or with findings counts from the
	// commit it examined; an incomplete one, or one of another target, does
	// not.
	verify := func(state, target, start string) string {
		return fmt.Sprintf("Markitect Run v: %s\n\nRun-Name: v\nRun-Target: %s\nRun-Start: %s\nRun-Command: verify\nRun-State: %s\nRun-Full: true\n", state, target, start, state)
	}
	g = landingLine(12)
	g.branches = map[string]string{"markitect/runs/v1": "b1", "markitect/runs/v2": "b2", "markitect/runs/v3": "b3"}
	g.messages["b1"] = verify(StateIncomplete, "main", "l11")
	g.messages["b2"] = verify(StateFindings, "release", "l10")
	g.messages["b3"] = verify(StateFindings, "main", "l6")
	if fc := count(g, "l12"); fc.Landed != 6 || fc.Since != "l6" {
		t.Fatalf("after verify --full: %+v", fc)
	}

	// The newest landing that names a full check carries it forward, also
	// after that verify's branch is gone.
	g.branches = nil
	g.messages["l9"] += "Run-Full-Check: l6\n"
	if fc := count(g, "l12"); fc.Landed != 6 || fc.Since != "l6" {
		t.Fatalf("carried forward: %+v", fc)
	}
}

// A landing names the full check it carries forward, or that it is one.
func TestLandingCommitsCarryTheFullCheck(t *testing.T) {
	r := &Result{Run: "r1", Command: "apply", State: StateLanded, Target: "main", FullCheck: FullCheck{Since: "abc"}}
	if got := trailers(resultMessage(r, "w"))[trailerFullCheck]; len(got) != 1 || got[0] != "abc" {
		t.Fatalf("carried %v", got)
	}
	r.Full = true
	if t2 := trailers(resultMessage(r, "w")); len(t2[trailerFull]) != 1 || len(t2[trailerFullCheck]) != 0 {
		t.Fatalf("a full landing: %v", t2)
	}
	kept := &Result{Run: "v", Command: "verify", State: StateVerified, Target: "main", Start: "s", Full: true}
	if start, ok := verifiedWhole(trailers(resultMessage(kept, "")), "main"); !ok || start != "s" {
		t.Fatalf("a verify --full's Run branch reads as %q %v", start, ok)
	}
}
