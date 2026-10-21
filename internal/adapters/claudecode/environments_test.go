package claudecode

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// envHarness is a harness whose every session is a fresh fake provider.
func envHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t, newFakeRuntime(), validProfile())
	h.next = newFakeRuntime
	return h
}

func envTurn(id string, writable bool) run.Turn {
	turn := testTurn()
	turn.ID, turn.Environment = id, "run-1-lending-m"
	if writable {
		turn.ReadOnly, turn.Mounts = true, []run.PathMount{{Path: "realization/lending", Writable: true}}
	} else {
		turn.ReadOnly = true
	}
	return turn
}

func mustRun(t *testing.T, h *harness, turn run.Turn) run.TurnResult {
	t.Helper()
	res, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string)))
	if err != nil || !res.Stopped {
		t.Fatalf("turn %s: %v %+v", turn.ID, err, res)
	}
	return res
}

// Consecutive turns of one environment with the same capabilities run in one
// instance, each with its own agentd session; a quiet instance stays and is
// never removed by a turn.
func TestTurnsOfOneEnvironmentReuseItsInstance(t *testing.T) {
	h := envHarness(t)
	first := mustRun(t, h, envTurn("t1", true))
	second := mustRun(t, h, envTurn("t2", true))
	if h.starts != 1 || len(h.instances) != 1 || h.instances[0].execs != 2 || h.instances[0].reaps != 2 || h.instances[0].removed != 0 {
		t.Fatalf("starts %d, instances %+v", h.starts, h.instances)
	}
	if first.Instance.Reused || first.Instance.Started != "first" || first.Instance.Generation != 1 || !second.Instance.Reused || second.Instance.Name != first.Instance.Name || second.Instance.Generation != 1 {
		t.Fatalf("instances %+v then %+v", first.Instance, second.Instance)
	}
	if first.Instance.Name != "markitect-run-1-lending-m-g1" || h.instances[0].spec.Name != first.Instance.Name {
		t.Fatalf("instance name %q", first.Instance.Name)
	}
}

// A turn with other capabilities gets its own instance: a prompt cannot revoke
// a writable mount. The environment keeps its read-only and its write view and
// switches between them; a third view replaces the least recently used.
func TestCapabilityViewsGetTheirOwnInstances(t *testing.T) {
	h := envHarness(t)
	write := mustRun(t, h, envTurn("t1", true))
	read := mustRun(t, h, envTurn("t2", false))
	again := mustRun(t, h, envTurn("t3", true))
	if h.starts != 2 || read.Instance.Started != "capabilities" || read.Instance.Generation != 2 || !again.Instance.Reused || again.Instance.Name != write.Instance.Name {
		t.Fatalf("views: %+v %+v %+v", write.Instance, read.Instance, again.Instance)
	}
	if h.instances[1].spec.Mounts[0].ReadOnly != true || len(h.instances[1].spec.Mounts) != 2 {
		t.Fatalf("read-only view mounts %+v", h.instances[1].spec.Mounts)
	}
	// While a helper holds part of the area, the write view differs again.
	helper := envTurn("t4", true)
	helper.Mounts = append(helper.Mounts, run.PathMount{Path: "realization/lending/report"})
	third := mustRun(t, h, helper)
	if h.starts != 3 || third.Instance.Generation != 3 || h.instances[1].removed != 1 || h.instances[0].removed != 0 {
		t.Fatalf("third view %+v; removed read %d write %d", third.Instance, h.instances[1].removed, h.instances[0].removed)
	}
}

// A process that survives the turn's session in its instance is no reason to
// trust the instance: the instance is removed, its removal is the observed
// cessation, and the environment's next turn starts a replacement that says
// why. Without an observed removal the turn is not stopped.
func TestWritersThatDoNotStopInPlaceReplaceTheInstance(t *testing.T) {
	h := envHarness(t)
	h.remaining = []frame.Process{{PID: 42, PPID: 1, State: "S", Command: "nohup"}}
	res, err := h.agents.Run(context.Background(), envTurn("t1", true), echoHandler(new([]string)))
	if err != nil || !res.Stopped || h.instances[0].removed != 1 || !strings.Contains(res.Reason, "1 left") {
		t.Fatalf("%v %+v", err, res)
	}
	h.remaining = nil
	next := mustRun(t, h, envTurn("t2", true))
	if h.starts != 2 || next.Instance.Reused || next.Instance.Generation != 2 || next.Instance.Started != "the writers of its predecessor did not stop in place" {
		t.Fatalf("replacement %+v", next.Instance)
	}
	h.remaining, h.removeErr = []frame.Process{{PID: 43, State: "R"}}, errors.New("engine gone")
	res, _ = h.agents.Run(context.Background(), envTurn("t3", false), echoHandler(new([]string)))
	if res.Stopped || !strings.Contains(res.Reason, "removal was not observed") {
		t.Fatalf("an unremoved instance with a live process stopped: %+v", res)
	}
}

// A turn without an environment, such as a helper's, runs in a fresh instance
// that ends with it.
func TestATurnWithoutEnvironmentGetsAFreshInstance(t *testing.T) {
	h := envHarness(t)
	turn := testTurn()
	res := mustRun(t, h, turn)
	if res.Instance.Started != "fresh" || res.Instance.Name != "markitect-turn-1" || h.instances[0].removed != 1 {
		t.Fatalf("%+v removed %d", res.Instance, h.instances[0].removed)
	}
	if got := h.agents.Release(context.Background(), nil); len(got) != 0 {
		t.Fatalf("a fresh instance was kept: %+v", got)
	}
}

// One environment runs one turn at a time; a second concurrent turn is refused
// before anything starts.
func TestABusyEnvironmentRefusesAConcurrentTurn(t *testing.T) {
	h := envHarness(t)
	mustRun(t, h, envTurn("t1", true))
	h.agents.envs["run-1-lending-m"].busy = true
	res, err := h.agents.Run(context.Background(), envTurn("t2", true), echoHandler(new([]string)))
	if err == nil || !strings.Contains(err.Error(), "in use") || h.instances[0].execs != 1 || res.Stopped {
		t.Fatalf("%v %+v", err, res)
	}
}

// A session that cannot start in a kept instance starts no provider; the
// instance is reaped like after a turn and the error reported.
func TestAFailedSessionStartsNoProvider(t *testing.T) {
	h := envHarness(t)
	h.execErr = errors.New("exec refused")
	res, err := h.agents.Run(context.Background(), envTurn("t1", true), echoHandler(new([]string)))
	if err == nil || !res.Stopped || !strings.Contains(res.Reason, "agentd session did not start") || h.instances[0].reaps != 1 {
		t.Fatalf("%v %+v", err, res)
	}
}

// Release removes the kept instances of the named environments only, or of
// all, and reports every removal.
func TestReleaseRemovesTheNamedEnvironments(t *testing.T) {
	h := envHarness(t)
	mustRun(t, h, envTurn("t1", true))
	mustRun(t, h, envTurn("t2", false))
	review := envTurn("t3", false)
	review.Environment, review.Kind = "run-1-p2-lending-r", run.VerifyTurn
	mustRun(t, h, review)
	got := h.agents.Release(context.Background(), []string{"run-1-p2-lending-r"})
	if len(got) != 1 || got[0].Environment != "run-1-p2-lending-r" || !got[0].Removed || h.instances[2].removed != 1 || h.instances[0].removed != 0 {
		t.Fatalf("review release %+v", got)
	}
	h.instances[0].removeErr = errors.New("engine gone")
	got = h.agents.Release(context.Background(), nil)
	if len(got) != 2 || got[0].Removed || got[0].Error == "" || !got[1].Removed {
		t.Fatalf("final release %+v", got)
	}
	if again := h.agents.Release(context.Background(), nil); len(again) != 0 {
		t.Fatalf("released twice: %+v", again)
	}
}

// An evicted view whose removal was not observed stays the environment's
// responsibility: its release removes it again and reports it, so no kept
// instance is left behind unreported.
func TestAnEvictedViewIsRemovedAgainAtTheRelease(t *testing.T) {
	h := envHarness(t)
	mustRun(t, h, envTurn("t1", true))
	mustRun(t, h, envTurn("t2", false))
	h.instances[0].removeErr = errors.New("engine busy")
	helper := envTurn("t3", true)
	helper.Mounts = append(helper.Mounts, run.PathMount{Path: "realization/lending/report"})
	mustRun(t, h, helper)
	if h.instances[0].removed != 1 {
		t.Fatalf("the evicted view was not removed: %d", h.instances[0].removed)
	}
	h.instances[0].removeErr = nil
	got := h.agents.Release(context.Background(), nil)
	if len(got) != 3 || got[0].Instance != h.instances[0].name || !got[0].Removed || h.instances[0].removed != 2 {
		t.Fatalf("release %+v", got)
	}
}
