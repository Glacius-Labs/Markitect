package run_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// A step whose workspace cannot be observed after a clean stop concludes as
// failed; its task is admitted again and the run continues (core rule T6).
func TestHostFailureAfterTheStepConcludesItAsFailed(t *testing.T) {
	h := newHarness(t, true, happyAgent)
	h.ws.failHarvest = 1
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not recover from a failed harvest", h.c.Current())
	}
	order := admitted(h)
	if order[0] != "lending explore" || order[1] != "lending explore" {
		t.Fatal("the failed Explore was not admitted again", order)
	}
	if b := h.c.Budget(); b.Held != (run.Capacity{}) {
		t.Fatal("a hold survived the failed step", b.Held)
	}
	// The retry sees a neutral note, not the Host's error text.
	retry := h.sessions.packets[1].Context.Bytes
	if !bytes.Contains(retry, []byte("because the Host could not finish it")) || bytes.Contains(retry, []byte("injected")) {
		t.Fatal("the retry's context misreports the Host's failure")
	}
}

// A container that fails to start is still stopped and charged, and its step
// concludes as failed (core rules S1, S2).
func TestFailedStartIsStoppedChargedAndConcluded(t *testing.T) {
	h := newHarness(t, true, happyAgent)
	h.sessions.failOpen = 1
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not recover from a failed start", h.c.Current())
	}
	if h.sessions.stops != len(admitted(h)) {
		t.Fatal("not every session was stopped", h.sessions.stops, len(admitted(h)))
	}
	if v := h.task("lending"); v.Failure != "" {
		t.Fatal("a later successful step kept the failure", v.Failure)
	}
}

// failingInput refuses to read step inputs, so the step is prepared but never
// dispatched.
type failingInput struct {
	*memoryJournal
	media    string
	failures int
}

func (j *failingInput) ReadBlob(ctx context.Context, ref run.BlobRef) (run.Payload, error) {
	if j.failures > 0 && ref.MediaType == j.media {
		j.failures--
		return run.Payload{}, errors.New("injected read failure")
	}
	return j.memoryJournal.ReadBlob(ctx, ref)
}

// A step that was prepared but never dispatched is released, not charged, and
// concludes as failed.
func TestUndispatchedStepIsReleased(t *testing.T) {
	h := newHarness(t, true, happyAgent)
	h.journal = &failingInput{memoryJournal: h.j, media: "application/vnd.markitect.step-input+json", failures: 1}
	h.engine = h.newEngine(h.c)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not recover from an undispatched step", h.c.Current())
	}
	released := 0
	for _, i := range h.c.Invocations() {
		if i.State == run.Released {
			released++
		}
	}
	if released != 1 {
		t.Fatal("the undispatched step was not released", released)
	}
}

// Without a workspace no step starts; the workspace is still closed and the
// run is not charged.
func TestMaterializationFailureClosesTheWorkspace(t *testing.T) {
	h := newHarness(t, true, happyAgent)
	h.ws.failMaterialize = true
	if err := h.engine.Run(context.Background()); err == nil {
		t.Fatal("a run without workspaces reported success")
	}
	if h.ws.closes != 1 || h.c.Budget().Consumed.SessionStarts != 0 {
		t.Fatal("workspace not closed or start charged", h.ws.closes, h.c.Budget().Consumed)
	}
}

// A registry proposal the capture would refuse is refused when it is made,
// and the proposals of a failed step are undone (core rule T6). The next step
// sees why its last step failed.
func TestRegistryProposalsAreCheckedAndUndoneWithTheirFailedStep(t *testing.T) {
	agent := func(s agentStep) []action {
		if s.Manager != "lending" || s.Phase != run.PhaseWork {
			return happyAgent(s)
		}
		switch s.Attempt {
		case 0:
			return []action{
				{Tool: run.ToolRegister, Args: `{"path":"out/library.go","form":"code","realizes":["lending/Missing"],"why":"typo"}`, Want: run.ToolRefused},
				{Tool: run.ToolRegister, Args: `{"path":"out/other.go","form":"code","realizes":["lending/Item"],"why":"unreserved"}`, Want: run.ToolRefused},
				{Tool: run.ToolRegister, Args: `{"path":"out/library.go","form":"code","realizes":["lending/Item"],"why":"Realizes Item again."}`, Want: run.ToolOK},
				// No outcome: the step fails and its proposal is undone.
			}
		case 1:
			if !strings.Contains(string(s.Packet.Context.Bytes), "Your last step failed the Host's check") {
				t.Error("the next step did not see why its last step failed")
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current(), h.sessions.answers)
	}
	registered, ok := h.engine.Aggregate().Registry().Lookup(mustPath(t, "out/library.go"))
	if !ok || registered.Artifact().Why() != "Existing realization" {
		t.Fatal("the failed step's registry proposal reached the aggregate", registered.Artifact().Why())
	}
}

// A no-change Explore result that its review rejects still reaches the
// parent's Explore as the child's report (core rule T1).
func TestRejectedNoChangeExploreStillReachesTheParent(t *testing.T) {
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "empty" && s.Phase == run.PhaseExplore:
			return []action{{Tool: run.ToolSubmit, Args: `{"purpose":"local","scope":"complete","summary":"Nothing to do in empty.","assessments":[],"dependencies":[]}`, Want: run.ToolOK}}
		case s.Manager == "empty" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: rejectArgs, Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, false, agent)
	h.advanceUntil(func(a run.Admission) bool { return a.Phase == run.PhaseWork })
	if !bytes.Contains(h.packet(".", run.PhaseExplore).Context.Bytes, []byte("Nothing to do in empty.")) {
		t.Fatal("the root's Explore lost the rejected child's report")
	}
}

func mustPath(t *testing.T, s string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A failure before the container starts costs nothing, so the run stops
// instead of retrying it without bound.
func TestUnchargedFailureStopsTheRun(t *testing.T) {
	h := newHarness(t, true, happyAgent)
	h.journal = &failingInput{memoryJournal: h.j, media: run.ToolCatalogMedia, failures: 1000}
	h.engine = h.newEngine(h.c)
	if err := h.engine.Run(context.Background()); err == nil {
		t.Fatal("a run whose sessions cannot be admitted reported success")
	}
	if n := len(admitted(h)); n != 1 || h.c.Budget().Consumed.SessionStarts != 0 {
		t.Fatal("an uncharged failure was retried", n, h.c.Budget().Consumed)
	}
	if v := h.task("lending"); v.Session != "" || v.State != run.TaskQueued {
		t.Fatal("the abandoned session stayed bound", v)
	}
}

// A failed start that overran its reservation ends the run Incomplete once
// the container is stopped, instead of leaving it in Reconciling.
func TestOverrunningFailedStartFinishesTheRun(t *testing.T) {
	h := newHarness(t, true, happyAgent)
	h.sessions.failOpen = 1
	h.sessions.onOpen = func() { h.clock = h.clock.Add(2 * time.Second) }
	runToEnd(t, h)
	if r := h.c.Current(); r.State != run.Finished || r.Outcome != run.Incomplete || !h.c.Budget().Overrun {
		t.Fatal("an overrunning failed start did not finish the run", r, h.c.Budget())
	}
}
