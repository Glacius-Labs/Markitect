package projectrun

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const knowledgeTestRunID = "0123456789abcdef0123456789abcdef"
const knowledgeTestCandidateID = "abcdef0123456789abcdef0123456789"

func knowledgeFixture(t *testing.T, status string) (string, *runStore, PlanRecord, RunReport, candidateData) {
	t.Helper()
	root := knowledgeTestRoot(t)
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.createRun(knowledgeTestRunID)
	if err != nil {
		t.Fatal(err)
	}
	candidate := candidateData{ID: knowledgeTestCandidateID, Files: map[string]File{}}
	if err := store.writeCandidate(dir, candidate); err != nil {
		t.Fatal(err)
	}
	candidate, err = store.readCandidate(dir, candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan := PlanRecord{
		APIVersion: APIVersion, ID: knowledgeTestRunID, Status: StatusPlanned, Operation: OperationApply,
		Goal: "fixture goal", BaseRevision: "0123456789012345678901234567890123456789",
		TargetBranch: "main", TargetHead: "0123456789012345678901234567890123456789",
		RepositoryDigest: "repo", BaseSnapshot: "base-snapshot", WorkingSnapshot: "working-snapshot",
		BaseProjectDigest: "base-project", WorkingProjectDigest: "working-project", BaseModelDigest: "base-model",
		ModelDigest: "model", ReportDigest: "report", RuntimeDigest: "planned-runtime",
		InitialCandidateID: candidate.ID, RuntimeAgents: map[string]string{"root": "fixture"},
	}
	plan.Digest, err = planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.writePlan(plan); err != nil {
		t.Fatal(err)
	}
	run := RunReport{
		APIVersion: APIVersion, ID: knowledgeTestRunID, PlanID: plan.ID, Status: status,
		Operation: plan.Operation, Mode: ModeControlledLocal,
		BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot, ModelDigest: plan.ModelDigest,
		RuntimeDigest: plan.RuntimeDigest, Candidate: candidateRef(candidate, true), Revision: 1,
	}
	if err := store.appendState(run); err != nil {
		t.Fatal(err)
	}
	run, err = store.readLatestState(knowledgeTestRunID)
	if err != nil {
		t.Fatal(err)
	}
	return root, store, plan, run, candidate
}

func knowledgeTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".markitect"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, RuntimePath), []byte("fixture runtime\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(root, RunsPath)); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestKnowledgeRecordsLoadsOnlyValidatedSelectedRunEvidence(t *testing.T) {
	root, store, _, run, candidate := knowledgeFixture(t, StatusApplied)
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	verification := VerifyReport{
		APIVersion: APIVersion, RunID: run.ID, CandidateID: candidate.ID,
		CandidateHash: candidate.Digest, Status: "verified", VerifiedAt: time.Now().UTC(),
		Checks: []CheckResult{},
	}
	verification.Digest, err = verificationDigest(verification)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistVerify(nil, dir, verification); err != nil {
		t.Fatal(err)
	}
	apply := ApplyReport{
		APIVersion: APIVersion, RunID: run.ID, CandidateID: candidate.ID, Status: StatusApplied,
		Written: []string{}, Journal: []string{}, AppliedAt: time.Now().UTC(),
	}
	if err := persistApply(nil, dir, apply); err != nil {
		t.Fatal(err)
	}

	got, err := KnowledgeRecords(root, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CaptureState != KnowledgeCaptureConsistent || got.PlanState != KnowledgeRecordPresent || got.RunState != KnowledgeRecordPresent || got.CandidateState != KnowledgeRecordPresent {
		t.Fatalf("selected records were not captured consistently: %+v", got)
	}
	if got.Status.Run.Status != StatusApplied || got.PlanDigest == "" || got.RunDigest == "" || got.CandidateHash != candidate.Digest || got.RuntimeBindingState != KnowledgeRuntimeBindingNotCompared {
		t.Fatalf("plan, run, candidate, or runtime metadata is incomplete: %+v", got)
	}
	if got.VerificationState != KnowledgeRecordPresent || got.Verification == nil || got.VerificationDigest != verification.Digest {
		t.Fatalf("validated verification was not projected: %+v", got)
	}
	if got.ApplyState != KnowledgeRecordPresent || got.Apply == nil || got.Apply.Status != StatusApplied || got.ApplyContentDigest == "" {
		t.Fatalf("Apply report was not projected with its derived content digest: %+v", got)
	}
}

func TestKnowledgeRecordsKeepsMissingEvidenceExplicit(t *testing.T) {
	root := knowledgeTestRoot(t)
	got, err := KnowledgeRecords(root, knowledgeTestRunID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlanState != KnowledgeRecordMissing || got.RunState != KnowledgeRecordMissing || got.VerificationState != KnowledgeRecordMissing || got.ApplyState != KnowledgeRecordMissing {
		t.Fatalf("absent records were not represented as missing: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(root, RunsPath, knowledgeTestRunID)); !os.IsNotExist(err) {
		t.Fatalf("read-only lookup created runtime state: stat err=%v", err)
	}
}

func TestKnowledgeRecordsKeepsPlannedCandidateAndMissingRunExplicit(t *testing.T) {
	root := knowledgeTestRoot(t)
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.createRun(knowledgeTestRunID)
	if err != nil {
		t.Fatal(err)
	}
	candidate := candidateData{ID: knowledgeTestCandidateID, Files: map[string]File{}}
	if err := store.writeCandidate(dir, candidate); err != nil {
		t.Fatal(err)
	}
	candidate, err = store.readCandidate(dir, candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan := PlanRecord{APIVersion: APIVersion, ID: knowledgeTestRunID, Status: StatusPlanned,
		InitialCandidateID: candidate.ID, RuntimeAgents: map[string]string{}}
	plan.Digest, err = planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.writePlan(plan); err != nil {
		t.Fatal(err)
	}

	got, err := KnowledgeRecords(root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlanState != KnowledgeRecordPresent || got.RunState != KnowledgeRecordMissing || got.CandidateState != KnowledgeRecordPresent || got.CandidateID != candidate.ID || got.CandidateHash != candidate.Digest {
		t.Fatalf("planned candidate and missing state were conflated: %+v", got)
	}
	if got.VerificationState != KnowledgeRecordMissing || got.ApplyState != KnowledgeRecordMissing || got.CaptureState != KnowledgeCaptureConsistent {
		t.Fatalf("missing verification or Apply evidence was not explicit: %+v", got)
	}
}

func TestKnowledgeRecordsRejectsRunCandidateAndVerificationLinkMismatch(t *testing.T) {
	t.Run("candidate reference", func(t *testing.T) {
		root, store, _, run, _ := knowledgeFixture(t, StatusVerified)
		run.Candidate.Snapshot = "different-candidate-hash"
		run.Revision++
		if err := store.appendState(run); err != nil {
			t.Fatal(err)
		}
		if _, err := KnowledgeRecords(root, run.ID); err == nil {
			t.Fatal("candidate reference mismatch was accepted")
		}
	})

	t.Run("verification run identity", func(t *testing.T) {
		root, store, _, run, candidate := knowledgeFixture(t, StatusVerified)
		dir, err := store.runDir(run.ID)
		if err != nil {
			t.Fatal(err)
		}
		verification := VerifyReport{APIVersion: APIVersion, RunID: "ffffffffffffffffffffffffffffffff",
			CandidateID: candidate.ID, CandidateHash: candidate.Digest, Status: "verified", VerifiedAt: time.Now().UTC(), Checks: []CheckResult{}}
		verification.Digest, err = verificationDigest(verification)
		if err != nil {
			t.Fatal(err)
		}
		if err := persistVerify(nil, dir, verification); err != nil {
			t.Fatal(err)
		}
		if _, err := KnowledgeRecords(root, run.ID); err == nil {
			t.Fatal("verification linked to another run was accepted")
		}
	})

	t.Run("verification candidate hash", func(t *testing.T) {
		root, store, _, run, candidate := knowledgeFixture(t, StatusVerified)
		dir, err := store.runDir(run.ID)
		if err != nil {
			t.Fatal(err)
		}
		verification := VerifyReport{APIVersion: APIVersion, RunID: run.ID, CandidateID: candidate.ID,
			CandidateHash: "sha256:stale", Status: "verified", VerifiedAt: time.Now().UTC(), Checks: []CheckResult{}}
		verification.Digest, err = verificationDigest(verification)
		if err != nil {
			t.Fatal(err)
		}
		if err := persistVerify(nil, dir, verification); err != nil {
			t.Fatal(err)
		}
		if _, err := KnowledgeRecords(root, run.ID); err == nil {
			t.Fatal("verification linked to a different candidate hash was accepted")
		}
	})

	t.Run("run basis differs from plan", func(t *testing.T) {
		root, store, _, run, _ := knowledgeFixture(t, StatusVerified)
		run.ModelDigest = "different-model"
		run.Revision++
		if err := store.appendState(run); err != nil {
			t.Fatal(err)
		}
		if _, err := KnowledgeRecords(root, run.ID); err == nil {
			t.Fatal("run state with a different plan basis was accepted")
		}
	})

	t.Run("candidate file digest", func(t *testing.T) {
		root, store, _, run, candidate := knowledgeFixture(t, StatusVerified)
		dir, err := store.runDir(run.ID)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "candidates", run.Candidate.ID+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		tampered := bytes.Replace(data, []byte(candidate.Digest), []byte("sha256:tampered"), 1)
		if bytes.Equal(tampered, data) {
			t.Fatal("fixture candidate digest was not found")
		}
		if err := os.WriteFile(path, tampered, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := KnowledgeRecords(root, run.ID); err == nil {
			t.Fatal("candidate with modified bytes was accepted")
		}
	})
}

func TestKnowledgeRecordsDoesNotReuseVerificationForAnotherCandidate(t *testing.T) {
	root, store, _, run, candidate := knowledgeFixture(t, StatusVerified)
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	verification := VerifyReport{APIVersion: APIVersion, RunID: run.ID, CandidateID: "11111111111111111111111111111111",
		CandidateHash: "sha256:other", Status: "verified", VerifiedAt: time.Now().UTC(), Checks: []CheckResult{}}
	verification.Digest, err = verificationDigest(verification)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistVerify(nil, dir, verification); err != nil {
		t.Fatal(err)
	}
	got, err := KnowledgeRecords(root, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CandidateID != candidate.ID || got.VerificationState != KnowledgeRecordMissing || got.Verification != nil {
		t.Fatalf("verification for another candidate was reused: %+v", got)
	}
}

func TestKnowledgeRecordsDoesNotTreatAppliedStatusAsApplyEvidence(t *testing.T) {
	root, _, _, run, _ := knowledgeFixture(t, StatusApplied)
	got, err := KnowledgeRecords(root, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplyState != KnowledgeRecordMissing || got.Apply != nil || got.CaptureState != KnowledgeCaptureUnknown {
		t.Fatalf("applied status without a receipt was promoted: %+v", got)
	}
}
