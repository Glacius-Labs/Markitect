package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func TestProjectRunRepairFailedRequiredCheckThenFreshVerifyAndApply(t *testing.T) {
	root := makeProjectRunFixture(t)
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		runtime.Limits.MaxRetries = 1
		runtime.Limits.MaxStarts = 11 // 4 initial calls + failed check + 4 repair calls + 2 fresh checks.
		runtime.Limits.MaxCostMicros = 450
		for id, agent := range runtime.Agents {
			agent.Pricing = Pricing{InputMicrosPerMillion: 1_000_000, OutputMicrosPerMillion: 1_000_000}
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
	initial, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || initial.Status != StatusIntegrated {
		t.Fatalf("initial manager run did not integrate: status=%s err=%v", initial.Status, err)
	}
	failedCandidate := initial.Candidate.ID
	failedStart := initial.StartedAt
	initialCalls := len(initial.Invocations)
	failedVerify, verifyErr := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
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

	repaired, err := Repair(context.Background(), host, ProcessInvoker{}, root, plan.ID)
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
	if err := assertCheckRepairContext(t, os.Getenv(e2eLogEnv), round.CheckFeedback[0]); err != nil {
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

	newVerify, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || newVerify.Status != "verified" || newVerify.CandidateID != repaired.Candidate.ID {
		t.Fatalf("repaired candidate did not require and pass a fresh verification: report=%+v err=%v", newVerify, err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, repaired.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	oldCandidateRequest := applyRequestFromPreflight(preflight)
	oldCandidateRequest.CandidateID = failedCandidate
	if _, err := Apply(host, ProcessInvoker{}, root, oldCandidateRequest); err == nil || !strings.Contains(err.Error(), "not the integrated candidate") {
		t.Fatalf("old failed candidate remained applicable: %v", err)
	}
	apply, err := Apply(host, ProcessInvoker{}, root, applyRequestFromPreflight(preflight))
	if err != nil || apply.Status != "applied" {
		t.Fatalf("freshly verified repaired candidate did not apply: report=%+v err=%v", apply, err)
	}
	orders, err := os.ReadFile(filepath.Join(root, "src", "orders", "implementation.txt"))
	if err != nil || string(orders) != "orders implementation v2\n" {
		t.Fatalf("verified repair content was not applied: content=%q err=%v", orders, err)
	}
}

func assertCheckRepairContext(t *testing.T, logPath string, expected RepairCheckFeedback) error {
	t.Helper()
	records, err := readE2ERecords(logPath)
	if err != nil {
		return err
	}
	found := false
	for _, record := range records {
		if record["repairRound"] != float64(1) {
			continue
		}
		managerID, _ := record["managerId"].(string)
		raw, marshalErr := json.Marshal(record["repairChecks"])
		if marshalErr != nil {
			return marshalErr
		}
		var checks []RepairCheckFeedback
		if unmarshalErr := json.Unmarshal(raw, &checks); unmarshalErr != nil {
			return unmarshalErr
		}
		for _, check := range checks {
			if check.ID != expected.ID {
				continue
			}
			found = true
			if check.Owner != expected.Owner || check.ExitCode != expected.ExitCode {
				return fmt.Errorf("repair feedback metadata changed: got %+v want %+v", check, expected)
			}
			detailed := managerID == expected.Owner || managerID == e2eManagerID("", "project-owner")
			if detailed && !strings.Contains(check.Stdout+check.Stderr, "orders check failed") {
				return fmt.Errorf("check owner did not receive bounded failure output: %+v", check)
			}
			if !detailed && (check.Stdout != "" || check.Stderr != "" || check.Error != "") {
				return fmt.Errorf("unrelated manager received check output: manager=%s feedback=%+v", managerID, check)
			}
		}
	}
	if found {
		return nil
	}
	return fmt.Errorf("repair requests omitted failed check feedback for %s", expected.ID)
}

func TestProjectRunRepairRejectsNonCheckFailuresAndSpentRounds(t *testing.T) {
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
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, "")
			if scenario.maxRetries != 0 {
				updateE2ERuntime(t, root, func(runtime *Runtime) { runtime.Limits.MaxRetries = scenario.maxRetries })
			}
			plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Test bounded repair eligibility.",
				Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatal(err)
			}
			makeSyntheticFailedCheckRun(t, root, plan, scenario.checkExit, scenario.verifier, scenario.runStatus, scenario.priorCost, scenario.priorStarts)
			_, err = Repair(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
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

func makeSyntheticFailedCheckRun(t *testing.T, root string, plan PlanRecord, exitCode int, withVerifier bool, runStatus string, priorCost int64, priorStarts int) {
	t.Helper()
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := store.readCandidate(dir, plan.InitialCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if runStatus == "" {
		runStatus = StatusFailed
	}
	check := plan.Checks[0]
	result := CheckResult{ID: check.ID, Command: append([]string(nil), check.Command...), ExecutablePath: check.ExecutablePath,
		ExecutableDigest: check.ExecutableDigest, CandidateID: candidate.ID, StartedAt: now, Duration: "1s", ExitCode: exitCode,
		Outcome: "failed", Error: "exit status 1", Stdout: "found 0, want 1"}
	run := RunReport{APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Status: runStatus, Mode: ModeControlledLocal,
		StartedAt: now, UpdatedAt: now, BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot, ModelDigest: plan.ModelDigest,
		RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers), Candidate: candidateRef(candidate, true), Checks: []CheckResult{result}, Revision: 1}
	if runStatus == StatusBlocked && len(run.Tasks) > 0 {
		run.Tasks[0].State = "uncertain"
	}
	for i := 0; i < priorStarts; i++ {
		run.Invocations = append(run.Invocations, InvocationLog{TaskID: "prior", Role: agentexec.RoleExecutor, Phase: "work", Outcome: agentexec.OutcomeProposed})
	}
	if priorCost > 0 {
		run.Invocations = append(run.Invocations, InvocationLog{TaskID: "prior", Role: agentexec.RoleExecutor, Phase: "work", Outcome: agentexec.OutcomeProposed, CostMicros: priorCost})
	}
	if err := store.appendState(run); err != nil {
		t.Fatal(err)
	}
	verification := VerifyReport{APIVersion: APIVersion, RunID: plan.ID, CandidateID: candidate.ID, CandidateHash: candidate.Digest,
		Status: "failed", Checks: []CheckResult{result}, VerifiedAt: now, Digest: ""}
	if withVerifier {
		verification.Verifier = &VerifierReport{Role: "verifier", Outcome: "failed", Observations: []agentexec.Observation{}}
	}
	verification.Digest, err = verificationDigest(verification)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistVerify(store, dir, verification); err != nil {
		t.Fatal(err)
	}
}

func applyRequestFromPreflight(preflight ApplyPreflight) ApplyRequest {
	return ApplyRequest{RunID: preflight.RunID, PlanID: preflight.PlanID, CandidateID: preflight.CandidateID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch,
		ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree}
}

func TestProjectRunRepairRejectsStaleRunBeforeManagerInvocation(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "")
	updateE2ERuntime(t, root, func(runtime *Runtime) { runtime.Limits.MaxRetries = 1 })
	plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Test stale repair rejection.",
		Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	makeSyntheticFailedCheckRun(t, root, plan, 1, false, "", 0, 0)
	writeE2E(t, root, "src/orders/implementation.txt", "user changed selected source\n")
	_, err = Repair(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
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
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "repair-check-no-change")
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		runtime.Limits.MaxRetries = 1
		runtime.Limits.MaxStarts = 11
	})
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Test unchanged failed snapshot repair rejection.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || initial.Status != StatusIntegrated {
		t.Fatalf("initial run did not integrate: status=%s err=%v", initial.Status, err)
	}
	failedVerification, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
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
	repaired, repairErr := Repair(context.Background(), host, ProcessInvoker{}, root, plan.ID)
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
	if _, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID); err == nil || !strings.Contains(err.Error(), "no exact pending native verifier to recover") {
		t.Fatalf("new candidate ID bypassed failed verification state: %v", err)
	}
	if _, err := latestVerification(dir, repaired.Candidate.ID); err == nil {
		t.Fatal("no verification report should exist for unchanged repair candidate")
	}
}
