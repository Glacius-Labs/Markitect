package run_test

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func runToEnd(t *testing.T, h *harness) {
	t.Helper()
	if err := h.engine.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func aggregateFile(t *testing.T, e *run.Engine, path string) string {
	t.Helper()
	p, _ := snapshot.NewPath(path)
	entry, ok := e.Aggregate().Tree().Lookup(p)
	if !ok {
		t.Fatal("aggregate lacks", path)
	}
	return string(entry.Bytes())
}

// The whole core runs one Package end to end: children explore first, the
// barrier opens, the leaf's first candidate is rejected and reworked, the
// accepted delivery is composed into the aggregate, the parent integrates,
// and the root's completion moves the run to Verifying.
func TestEngineRunsOnePackageThroughRejectionReworkAndIntegration(t *testing.T) {
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package library\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: rejectArgs, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate:
			findings := strings.Contains(string(s.Packet.Context.Bytes), "No type realizes Item.")
			if !findings {
				t.Error("the second review did not see the open finding")
			}
			return []action{
				{Tool: run.ToolSubmitReview, Args: passArgs(), Want: run.ToolRefused},
				{Tool: run.ToolSubmitReview, Args: passArgs(openFinding(t, s)), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			if !strings.Contains(string(s.Packet.Context.Bytes), "No type realizes Item.") {
				t.Error("rework did not see the finding")
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if got := h.c.Current().State; got != run.Verifying {
		t.Fatal("run did not reach Verifying", got, h.sessions.answers)
	}
	root, lending := h.task("."), h.task("lending")
	if root.State != run.TaskDone || lending.State != run.TaskDone || !root.LocalDone || len(root.Deliveries) != 2 || len(lending.Deliveries) != 1 {
		t.Fatalf("tasks not done as T13 requires: %+v %+v", root, lending)
	}
	if lending.Rejections["local"] != 1 || lending.Deliveries[0].Aggregate != 1 || h.c.AggregateVersion() != 1 {
		t.Fatal("rejection or composition not recorded", lending.Rejections, lending.Deliveries, h.c.AggregateVersion())
	}
	if got := aggregateFile(t, h.engine, "out/library.go"); got != "package library\n\ntype Item struct{}\n" {
		t.Fatalf("aggregate does not hold the accepted rework: %q", got)
	}
	issues := h.c.Issues("lending")
	if len(issues) != 1 || issues[0].State != "resolved" || issues[0].Severity != "blocking" {
		t.Fatalf("finding not resolved by the independent review: %+v", issues)
	}
	order := admitted(h)
	want := []string{
		"lending explore", ". explore",
		". work", "lending work", ". reviewer-expectations", "lending reviewer-expectations", ". reviewer-candidate", "lending reviewer-candidate",
		"lending work", "lending reviewer-expectations", "lending reviewer-candidate",
		". integrate", ". reviewer-expectations", ". reviewer-candidate",
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("admission order differs from T4:\n got %v\nwant %v", order, want)
	}

	restored, err := run.Restore(context.Background(), &readJournal{store: h.j}, "repo", run.RestoreLimits{PageSize: 16, Batches: 2000, Events: 4000, Bytes: 64 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Tasks(), h.c.Tasks()) || !reflect.DeepEqual(restored.Issues("lending"), issues) || restored.Current() != h.c.Current() {
		t.Fatal("restore does not rebuild the task machine")
	}
	again := h.newEngine(restored)
	if !again.Aggregate().Tree().Equal(h.engine.Aggregate().Tree()) {
		t.Fatal("a new engine does not rebuild the aggregate from the deliveries")
	}
}

func openFinding(t *testing.T, s agentStep) string {
	t.Helper()
	text := string(s.Packet.Context.Bytes)
	i := strings.Index(text, "finding-step-")
	if i < 0 {
		t.Fatal("no finding in context")
	}
	j := i
	for j < len(text) && (text[j] == '-' || text[j] >= '0' && text[j] <= '9' || text[j] >= 'a' && text[j] <= 'z') {
		j++
	}
	return text[i:j]
}

func admitted(h *harness) []string {
	var out []string
	for _, b := range h.j.batches {
		for _, e := range b.Events() {
			if e.Kind == run.PrepareSession && e.Body.Manager.Manager != "" {
				out = append(out, e.Body.Manager.Manager+" "+string(e.Body.Manager.Phase))
			}
		}
	}
	return out
}

// A Manager that finds nothing to change submits a complete no-change result
// from Explore; its review runs before the write barrier because nothing is
// written (core rules T2, T3).
func TestNoChangeExploreIsReviewedBeforeTheBarrier(t *testing.T) {
	agent := func(s agentStep) []action {
		if s.Manager == "empty" && s.Phase == run.PhaseExplore {
			return []action{
				{Tool: run.ToolSubmit, Args: submitArgs("local", "limited"), Want: run.ToolRefused},
				{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, false, agent)
	h.advanceUntil(func(a run.Admission) bool { return a.Phase == run.PhaseWork })
	order := admitted(h)
	review := slices.Index(order, "empty reviewer-candidate")
	if review < 0 || review > slices.Index(order, ". explore") {
		t.Fatal("no-change review did not run before the barrier", order)
	}
	if v := h.task("empty"); v.State != run.TaskDone || len(v.Deliveries) != 1 || v.Deliveries[0].Candidate != (run.BlobRef{}) {
		t.Fatalf("no-change delivery not accepted: %+v", v)
	}
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.sessions.answers)
	}
}

// At the rejection limit the task becomes incomplete and the run finishes
// Incomplete with all work retained (core rules T8, T14).
func TestRejectionLimitFinishesTheRunIncomplete(t *testing.T) {
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate {
			return []action{{Tool: run.ToolSubmitReview, Args: rejectArgs, Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	lending := h.task("lending")
	if lending.State != run.TaskIncomplete || lending.Rejections["local"] != 2 || lending.Own == (run.BlobRef{}) {
		t.Fatalf("limit not applied or work dropped: %+v", lending)
	}
	if r := h.c.Current(); r.State != run.Finished || r.Outcome != run.Incomplete || !strings.Contains(r.Reason, "rejection limit") {
		t.Fatalf("run not finished Incomplete: %+v", r)
	}
}

// A review that needs an owner decision blocks the task on an owner question,
// and the run waits for the owner (core rules T8, W1).
func TestNeedsDecisionWaitsForTheOwner(t *testing.T) {
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate {
			return []action{{Tool: run.ToolSubmitReview, Args: `{"verdict":"needs_decision","summary":"The model leaves the format open.","question":"Which format should Item use?","evidence":[],"findings":[],"resolutions":[]}`, Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	lending := h.task("lending")
	if lending.State != run.TaskBlocked || len(lending.Waits) != 1 || lending.Waits[0].Kind != "owner" || lending.Submission.Proposal == (run.BlobRef{}) {
		t.Fatalf("task not blocked on the owner with its frozen candidate: %+v", lending)
	}
	issues := h.c.Issues("lending")
	if len(issues) != 1 || issues[0].Kind != "question" || issues[0].To != "owner" || issues[0].Request != "Which format should Item use?" {
		t.Fatalf("owner question not recorded: %+v", issues)
	}
	if r := h.c.Current(); r.State != run.Waiting || r.NextActor != "owner" {
		t.Fatalf("run not waiting for the owner: %+v", r)
	}
}

// A step without a valid outcome fails, stays charged and returns its task
// to the state before the step (core rule T6). A write outside the owner's
// claims fails the step whatever its outcome said (core rule S4).
func TestFailedStepsReturnTasksToTheirStateBeforeTheStep(t *testing.T) {
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseExplore && s.Attempt == 0:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"no"}`, Want: run.ToolRefused}}
		case s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package root\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not recover from failed steps", h.c.Current(), h.sessions.answers)
	}
	order := admitted(h)
	if order[0] != "lending explore" || order[1] != "lending explore" {
		t.Fatal("failed Explore was not admitted again", order)
	}
	if got := aggregateFile(t, h.engine, "out/library.go"); got != "package library\n\ntype Item struct{}\n" {
		t.Fatalf("a write outside the root's claims reached the aggregate: %q", got)
	}
	if n := strings.Count(strings.Join(order, ","), ". work"); n != 2 {
		t.Fatal("failed work step was not admitted again", order)
	}
}

// A consumer that waits for a producer's delivery wakes when it is accepted
// (core rules T5, T10).
func TestDeliveryWaitWakesTheConsumer(t *testing.T) {
	agent := func(s agentStep) []action {
		if s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0 {
			return []action{
				{Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"."}],"reason":"self"}`, Want: run.ToolRefused},
				{Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"lending"}],"reason":"Needs the Item type."}`, Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	h.advanceUntil(func(a run.Admission) bool { return a.Manager == "lending" && a.Phase == run.PhaseReviewerCandidate })
	if v := h.task("."); v.State != run.TaskBlocked || v.Waits[0].Ref != "lending" {
		t.Fatalf("root not blocked on lending: %+v", v)
	}
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.sessions.answers)
	}
	order := admitted(h)
	accepted := slices.Index(order, "lending reviewer-candidate")
	if rework := slices.Index(order[accepted:], ". work"); rework < 0 {
		t.Fatal("root did not work again after lending's delivery", order)
	}
}
