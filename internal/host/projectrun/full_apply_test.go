package projectrun

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

func TestFullApplyVerificationBindsAllManagersAndReceipts(t *testing.T) {
	root := makeFullVerifyFixture(t)
	revision := gitE2E(t, root, "rev-parse", "HEAD")
	base, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	candidate := candidateData{ID: "candidate-full-apply", Files: map[string]File{}}
	compiled, err := projectForCandidate(Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}, root, base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	briefings, err := briefingBindings(root, compiled)
	if err != nil {
		t.Fatal(err)
	}
	plan, verification := passingFullApplyEvidence(t, candidate, compiled, briefings)
	if err := validateFullApplyVerification(plan, candidate, compiled, verification, briefings); err != nil {
		t.Fatalf("valid full verification rejected: %v", err)
	}

	// Altering the receipt while retaining the old nested digest must fail.
	verification.ManagerVerification.Managers[0].Receipt.RunID = "substituted-run"
	if err := validateFullApplyVerification(plan, candidate, compiled, verification, briefings); err == nil {
		t.Fatal("tampered Manager receipt passed full apply validation")
	}

	// Even a newly digested report cannot omit a declared Manager.
	_, verification = passingFullApplyEvidence(t, candidate, compiled, briefings)
	verification.ManagerVerification.Managers = verification.ManagerVerification.Managers[:len(verification.ManagerVerification.Managers)-1]
	verification.ManagerVerification.Digest, err = FullVerifyReportDigest(*verification.ManagerVerification)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFullApplyVerification(plan, candidate, compiled, verification, briefings); err == nil {
		t.Fatal("full verification with an omitted Manager passed apply validation")
	}
}

func TestFullApplyAcceptsVerifiedImplementationDigestDifferentFromPlanReport(t *testing.T) {
	root := makeFullVerifyFixture(t)
	revision := gitE2E(t, root, "rev-parse", "HEAD")
	base, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	var changedPath string
	for _, file := range base.Report.Files {
		if file.Exists && file.Owner != "" {
			changedPath = file.Path
			break
		}
	}
	if changedPath == "" {
		t.Fatal("fixture has no Manager-owned realization file")
	}
	content := append([]byte(nil), base.Snapshot.Files[changedPath]...)
	content = append(content, []byte("\nverified candidate implementation update\n")...)
	candidate := candidateData{ID: "candidate-updated-implementation", Files: map[string]File{
		changedPath: {Path: changedPath, Mode: base.Snapshot.Modes[changedPath], Content: content},
	}}
	host := Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}
	compiled, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Report.ModelDigest != base.Report.ModelDigest || compiled.Report.Digest == base.Report.Digest {
		t.Fatalf("fixture edit did not isolate model-stable implementation digest change: model=%t report=%t", compiled.Report.ModelDigest == base.Report.ModelDigest, compiled.Report.Digest != base.Report.Digest)
	}
	if err := validateFinalCandidate(host, root, base.Snapshot, candidate, PlanRecord{ModelDigest: base.Report.ModelDigest}); err != nil {
		t.Fatalf("valid full coverage implementation update rejected: %v", err)
	}
	briefings, err := briefingBindings(root, compiled)
	if err != nil {
		t.Fatal(err)
	}
	plan, verification := passingFullApplyEvidence(t, candidate, compiled, briefings)
	plan.ReportDigest = base.Report.Digest
	if err := validateFullApplyVerification(plan, candidate, compiled, verification, briefings); err != nil {
		t.Fatalf("valid candidate-bound report digest differing from the pre-execution plan was rejected: %v", err)
	}
}

func TestFullApplyRejectsUnclassifiedCandidatePath(t *testing.T) {
	root := makeFullVerifyFixture(t)
	revision := gitE2E(t, root, "rev-parse", "HEAD")
	base, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	candidate := candidateData{ID: "candidate-unclassified", Files: map[string]File{
		"unexpected/unowned.txt": {Path: "unexpected/unowned.txt", Mode: "100644", Content: []byte("unclassified candidate content\n")},
	}}
	compiled, err := projectForCandidate(Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}, root, base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Coverage == nil || compiled.Coverage.Conforming {
		t.Fatalf("unclassified candidate path did not break full coverage: coverage=%+v", compiled.Coverage)
	}
	if err := validateFinalCandidate(Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}, root, base.Snapshot, candidate, PlanRecord{ModelDigest: compiled.Report.ModelDigest}); err == nil {
		t.Fatal("unclassified candidate path passed final candidate validation")
	}
}

func TestFullApplyRejectsUnclassifiedWorktreeAddedAfterPlan(t *testing.T) {
	root := makeFullVerifyFixture(t)
	planned, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if planned.Coverage == nil || !planned.Coverage.Conforming {
		t.Fatalf("fixture worktree is not conforming: %+v", planned.Coverage)
	}
	writeE2E(t, root, "unexpected/unowned.txt", "appeared after planning\n")
	current, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if current.Snapshot.Digest() == planned.Snapshot.Digest() || current.Digest == planned.Digest {
		t.Fatal("worktree change did not invalidate the planned project and snapshot digests")
	}
	if current.Coverage == nil || current.Coverage.Conforming {
		t.Fatalf("unclassified post-plan worktree path remained conforming: %+v", current.Coverage)
	}
	if err := requireFullCoverage(current); err == nil {
		t.Fatal("unclassified post-plan worktree passed the full-coverage apply gate")
	}
}

func passingFullApplyEvidence(t *testing.T, candidate candidateData, compiled *Project, briefings map[string]string) (PlanRecord, VerifyReport) {
	t.Helper()
	plan := PlanRecord{ModelDigest: compiled.Report.ModelDigest, ReportDigest: compiled.Report.Digest, RuntimeDigest: "runtime-pin", BriefingDigests: briefings}
	checks := make([]CheckResult, 0, len(compiled.Report.Checks))
	for _, check := range compiled.Report.Checks {
		plan.Checks = append(plan.Checks, CheckPlan{ID: check.ID, Command: check.Command, ExecutablePath: "check-bin", ExecutableDigest: "check-digest"})
		checks = append(checks, CheckResult{ID: check.ID, Command: append([]string(nil), check.Command...), ExecutablePath: "check-bin", ExecutableDigest: "check-digest", CandidateID: candidate.ID, ExitCode: 0, Outcome: "passed"})
	}
	managers := make([]FullManagerAssessment, 0, len(compiled.Report.Managers))
	for _, manager := range compiled.Report.Managers {
		inputDigest := "input-" + manager.ID
		managers = append(managers, FullManagerAssessment{ManagerID: manager.ID, Status: "passed", ScopeDigest: "scope-" + manager.ID, InputDigest: inputDigest, Receipt: &agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: "receipt-" + manager.ID, InputDigest: inputDigest, Outcome: agentexec.OutcomeProposed}})
	}
	full := &FullVerifyReport{APIVersion: APIVersion, Status: "passed", CandidateID: candidate.ID, SnapshotDigest: compiled.Snapshot.Digest(), ProjectDigest: compiled.Digest,
		CoverageDigest: compiled.Coverage.Digest, ModelDigest: compiled.Report.ModelDigest, ReportDigest: compiled.Report.Digest,
		RuntimeDigest: plan.RuntimeDigest, BriefingDigests: briefings, Managers: managers, Checks: checks}
	var err error
	full.Digest, err = FullVerifyReportDigest(*full)
	if err != nil {
		t.Fatal(err)
	}
	return plan, VerifyReport{VerificationScope: "full", ManagerVerification: full, Checks: checks}
}
