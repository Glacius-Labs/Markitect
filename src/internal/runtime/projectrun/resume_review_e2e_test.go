//go:build !runtime_unit

package projectrun

import (
	"context"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
	"testing"
)

// A failed work review persists review-rework-ready before the rework turn
// starts. Resume must send that turn the reviewed candidate, as the
// uninterrupted review loop does, not the parent's bytes; a crash during the
// turn must let recovery rebuild the same request; and the reworked candidate
// must be reviewed again.
func TestResumedReviewReworkStartsFromReviewedCandidate(t *testing.T) {
	t.Parallel()
	fixture := seedReviewRework(t)
	if err := fixture.store.appendState(fixture.report); err != nil {
		t.Fatal(err)
	}
	ordersID := e2eManagerID("orders", "orders")
	invoker := &reworkDispatchInvoker{inner: projectexecutor.ProcessInvoker{}, store: fixture.store, runID: fixture.plan.ID, managerID: ordersID, phase: "work"}
	resumed, resumeErr := Resume(context.Background(), fixture.host, invoker, fixture.root, fixture.plan.ID)
	if invoker.request == nil || invoker.stateErr != nil {
		t.Fatalf("Resume did not dispatch the orders rework: status=%s err=%v state=%v", resumed.Status, resumeErr, invoker.stateErr)
	}
	got, found := "", false
	for _, artifact := range invoker.request.Artifacts {
		if artifact.Path == "src/orders/implementation.txt" {
			got, found = string(artifact.Content), true
		}
	}
	if !found {
		t.Fatalf("orders rework request omitted its owned artifact: %+v", invoker.request.Artifacts)
	}
	if got != reviewedOrdersBytes {
		t.Fatalf("resumed review rework input for orders = %q, want the reviewed candidate bytes %q", got, reviewedOrdersBytes)
	}

	dispatched := findTask(invoker.state.Tasks, ordersID)
	base, err := fixture.host.Load(fixture.root, fixture.plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := snapshotWithCandidate(base.Snapshot, fixture.initial)
	if err != nil {
		t.Fatal(err)
	}
	boundProject, err := fixture.host.FromSnapshot(fixture.root, bound)
	if err != nil {
		t.Fatal(err)
	}
	dependencies, err := managerDependencies(boundProject.Report, invoker.state.Tasks)
	if err != nil {
		t.Fatal(err)
	}
	input, children, childReports, conflicts, err := recoveryManagerInput(fixture.store, fixture.dir, fixture.host, fixture.root, base.Snapshot, boundProject.Report,
		invoker.state, fixture.plan, *dispatched, "work", dependencies[ordersID])
	if err != nil {
		t.Fatal(err)
	}
	repairRound, repairChecks := repairContext(invoker.state, *dispatched)
	rebuilt, err := managerInvocationRequest(fixture.host, fixture.root, mustLoadRuntime(t, fixture.root), fixture.plan, input, *dispatched, "work", children, conflicts, childReports,
		dispatched.RepairDiagnostic, repairRound, repairChecks)
	if err != nil {
		t.Fatal(err)
	}
	dispatchedDigest, err := nativeRequestInputDigest(*invoker.request)
	if err != nil {
		t.Fatal(err)
	}
	if rebuiltDigest, err := nativeRequestInputDigest(rebuilt); err != nil || rebuiltDigest != dispatchedDigest {
		t.Fatalf("recovery rebuilds rework input digest %s, dispatched %s (err=%v)", rebuiltDigest, dispatchedDigest, err)
	}

	final, err := fixture.store.readLatestState(fixture.plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	rereviewed := false
	for _, review := range final.Reviews {
		if review.ManagerID != ordersID || review.Phase != "work" || review.Round != 2 {
			continue
		}
		candidate, readErr := fixture.store.readCandidate(fixture.dir, review.CandidateID)
		rereviewed = rereviewed || readErr == nil && review.CandidateDigest == candidate.Digest && len(candidate.Parents) == 1 && candidate.Parents[0] == fixture.reviewed.ID
	}
	if !rereviewed {
		t.Fatalf("reworked orders candidate was not reviewed again: reviews=%+v status=%s err=%v", final.Reviews, final.Status, resumeErr)
	}
}
