package projectrun

import (
	"encoding/json"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
)

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
	run := RunReport{APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Status: runStatus, Mode: projectconfig.ModeControlledLocal,
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
