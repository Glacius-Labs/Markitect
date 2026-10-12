//go:build !runtime_unit

package projectrun

import (
	"context"
	"errors"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProjectRunRepairFailedRequiredCheckThenFreshVerifyAndApply(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		runtime.Limits.MaxRetries = 1
		runtime.Limits.MaxStarts = 11 // 4 initial calls + failed check + 4 repair calls + 2 fresh checks.
		runtime.Limits.MaxCostMicros = 450
		for id, agent := range runtime.Agents {
			agent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 1_000_000, OutputMicrosPerMillion: 1_000_000}
			runtime.Agents[id] = agent
		}
	})
	setupE2EProcess(t, "repair-check-fail")
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	initial, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || initial.Status != StatusIntegrated {
		t.Fatalf("initial manager run did not integrate: status=%s err=%v", initial.Status, err)
	}
	failedCandidate := initial.Candidate.ID
	failedStart := initial.StartedAt
	initialCalls := len(initial.Invocations)
	failedVerify, verifyErr := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if verifyErr == nil || failedVerify.Status != "failed" || failedVerify.CandidateID != failedCandidate || len(failedVerify.Checks) != 1 || failedVerify.Checks[0].ExitCode == 0 {
		t.Fatalf("fixture did not produce a known required-check failure: report=%+v err=%v", failedVerify, verifyErr)
	}
	failedState, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	failedRun, err := failedState.readLatestState(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failedRun.Status != StatusFailed || len(failedRun.Checks) != 1 {
		t.Fatalf("failed check attempt was not durably retained: status=%s checks=%+v", failedRun.Status, failedRun.Checks)
	}

	repaired, err := Repair(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || repaired.Status != StatusIntegrated {
		t.Fatalf("Repair did not complete the bounded manager loop: status=%s err=%v", repaired.Status, err)
	}
	if repaired.ID != initial.ID || !repaired.StartedAt.Equal(failedStart) || repaired.Candidate.ID == failedCandidate || len(repaired.Invocations) != initialCalls+4 {
		t.Fatalf("repair reset run identity, original budget, or candidate: old=%+v repaired=%+v", initial, repaired)
	}
	if len(repaired.RepairRounds) != 1 {
		t.Fatalf("repair round evidence missing: %+v", repaired.RepairRounds)
	}
	round := repaired.RepairRounds[0]
	if round.Status != "integrated" || round.PriorCandidateID != failedCandidate || round.CandidateID != repaired.Candidate.ID || round.VerificationDigest != failedVerify.Digest || len(round.PriorTasks) != len(initial.Tasks) || len(round.CheckFeedback) != 1 {
		t.Fatalf("repair history did not bind the failed candidate and check: %+v", round)
	}
	if totalCost(repaired.Invocations) <= totalCost(initial.Invocations) || totalCost(repaired.Invocations) > 450 || !reflect.DeepEqual(repaired.Invocations[:initialCalls], initial.Invocations) {
		t.Fatalf("repair did not retain prior receipts and cumulative cost: before=%+v after=%+v", initial.Invocations, repaired.Invocations)
	}
	if len(repaired.Invocations)+len(failedRun.Checks)+2 != 11 {
		t.Fatalf("fixture did not consume the shared maxStarts ledger exactly: invocations=%d previousChecks=%d freshChecks=2", len(repaired.Invocations), len(failedRun.Checks))
	}
	if err := assertCheckRepairContext(t, runtimeFixtureEnv(t, e2eLogEnv), round.CheckFeedback[0]); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := store.readCandidate(dir, round.SeedCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	if len(seed.Parents) != 1 || seed.Parents[0] != failedCandidate {
		t.Fatalf("repair candidate lineage lost the failed candidate: %+v", seed)
	}
	oldVerification, err := latestVerification(dir, failedCandidate)
	if err != nil || oldVerification.Digest != failedVerify.Digest || oldVerification.Status != "failed" {
		t.Fatalf("failed verification history was not retained: report=%+v err=%v", oldVerification, err)
	}

	newVerify, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || newVerify.Status != "verified" || newVerify.CandidateID != repaired.Candidate.ID {
		t.Fatalf("repaired candidate did not require and pass a fresh verification: report=%+v err=%v", newVerify, err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, repaired.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	oldCandidateRequest := applyRequestFromPreflight(preflight)
	oldCandidateRequest.CandidateID = failedCandidate
	if _, err := Apply(host, projectexecutor.ProcessInvoker{}, root, oldCandidateRequest); err == nil || !strings.Contains(err.Error(), "not the integrated candidate") {
		t.Fatalf("old failed candidate remained applicable: %v", err)
	}
	apply, err := Apply(host, projectexecutor.ProcessInvoker{}, root, applyRequestFromPreflight(preflight))
	if err != nil || apply.Status != "applied" {
		t.Fatalf("freshly verified repaired candidate did not apply: report=%+v err=%v", apply, err)
	}
	orders, err := os.ReadFile(filepath.Join(root, "src", "orders", "implementation.txt"))
	if err != nil || string(orders) != "orders implementation v2\n" {
		t.Fatalf("verified repair content was not applied: content=%q err=%v", orders, err)
	}
}

func TestProjectRunRepairRejectsNonCheckFailuresAndSpentRounds(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name          string
		checkExit     int
		verifier      bool
		maxRetries    int
		runStatus     string
		priorCost     int64
		priorStarts   int
		wantError     string
		wantRunStatus string
	}{
		{name: "unknown nonzero not present", checkExit: 0, maxRetries: 1, wantError: "no failed required declared check", wantRunStatus: StatusFailed},
		{name: "AI verifier failure", checkExit: 1, verifier: true, maxRetries: 1, wantError: "verifier failures are not repairable", wantRunStatus: StatusFailed},
		{name: "repair round cap", checkExit: 1, maxRetries: 0, wantError: "repair round limit 0 reached", wantRunStatus: StatusFailed},
		{name: "spent cost", checkExit: 1, maxRetries: 1, priorCost: 100000, wantError: "estimated cost limit", wantRunStatus: StatusFailed},
		{name: "spent starts", checkExit: 1, maxRetries: 1, priorStarts: 15, wantError: "runtime start limit", wantRunStatus: StatusFailed},
		{name: "uncertain process", checkExit: 1, maxRetries: 1, runStatus: StatusBlocked, wantError: "run status blocked cannot be repaired", wantRunStatus: StatusBlocked},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, "")
			if scenario.maxRetries != 0 {
				updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) { runtime.Limits.MaxRetries = scenario.maxRetries })
			}
			plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Test bounded repair eligibility.",
				Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatal(err)
			}
			makeSyntheticFailedCheckRun(t, root, plan, scenario.checkExit, scenario.verifier, scenario.runStatus, scenario.priorCost, scenario.priorStarts)
			_, err = Repair(context.Background(), projectworkHost(), projectexecutor.ProcessInvoker{}, root, plan.ID)
			if err == nil || !strings.Contains(err.Error(), scenario.wantError) {
				t.Fatalf("Repair accepted an ineligible failure: %v", err)
			}
			store, _ := newRunStore(root)
			state, readErr := store.readLatestState(plan.ID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if state.Status != scenario.wantRunStatus || len(state.RepairRounds) != 0 {
				t.Fatalf("ineligible repair mutated the failed run: %+v", state)
			}
		})
	}
}

func TestProjectRunRepairRejectsStaleRunBeforeManagerInvocation(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "")
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) { runtime.Limits.MaxRetries = 1 })
	plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Test stale repair rejection.",
		Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	makeSyntheticFailedCheckRun(t, root, plan, 1, false, "", 0, 0)
	writeE2E(t, root, "src/orders/implementation.txt", "user changed selected source\n")
	_, err = Repair(context.Background(), projectworkHost(), projectexecutor.ProcessInvoker{}, root, plan.ID)
	if !errors.Is(err, ErrStale) {
		t.Fatalf("stale selected source was accepted for repair: %v", err)
	}
	store, _ := newRunStore(root)
	state, readErr := store.readLatestState(plan.ID)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if state.Status != StatusSuperseded || len(state.RepairRounds) != 0 {
		t.Fatalf("stale repair did not supersede without starting a round: %+v", state)
	}
}

func TestProjectRunRepairCannotVerifyNewIDForUnchangedFailedSnapshot(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "repair-check-no-change")
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		runtime.Limits.MaxRetries = 1
		runtime.Limits.MaxStarts = 11
	})
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Test unchanged failed snapshot repair rejection.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || initial.Status != StatusIntegrated {
		t.Fatalf("initial run did not integrate: status=%s err=%v", initial.Status, err)
	}
	failedVerification, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err == nil || failedVerification.Status != "failed" {
		t.Fatalf("fixture did not fail its first required check: report=%+v err=%v", failedVerification, err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	oldCandidate, err := store.readCandidate(dir, initial.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	oldSnapshot, err := snapshotWithCandidate(base.Snapshot, oldCandidate)
	if err != nil {
		t.Fatal(err)
	}
	repaired, repairErr := Repair(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if repairErr == nil || !strings.Contains(repairErr.Error(), "unchanged failed candidate cannot be verified again") || repaired.Status != StatusFailed {
		t.Fatalf("unchanged candidate was treated as repaired: status=%s err=%v", repaired.Status, repairErr)
	}
	if repaired.Candidate.ID == initial.Candidate.ID {
		t.Fatal("fixture did not allocate a distinct candidate ID for its no-change repair")
	}
	newCandidate, err := store.readCandidate(dir, repaired.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	newSnapshot, err := snapshotWithCandidate(base.Snapshot, newCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if oldSnapshot.Digest() != newSnapshot.Digest() {
		t.Fatalf("no-change fixture changed snapshot bytes unexpectedly: old=%s new=%s", oldSnapshot.Digest(), newSnapshot.Digest())
	}
	if _, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID); err == nil || !strings.Contains(err.Error(), "no exact pending native verifier to recover") {
		t.Fatalf("new candidate ID bypassed failed verification state: %v", err)
	}
	if _, err := latestVerification(dir, repaired.Candidate.ID); err == nil {
		t.Fatal("no verification report should exist for unchanged repair candidate")
	}
}
