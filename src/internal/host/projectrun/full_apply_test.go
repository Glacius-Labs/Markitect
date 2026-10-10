package projectrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
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
	if err := validateFinalCandidate(host, root, base, candidate, PlanRecord{ModelDigest: base.Report.ModelDigest}); err != nil {
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
	if err := validateFinalCandidate(Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}, root, base, candidate, PlanRecord{ModelDigest: compiled.Report.ModelDigest}); err == nil {
		t.Fatal("unclassified candidate path passed final candidate validation")
	}
}

// The closure gate classifies the candidate against the base repository
// census. The candidate snapshot omits ignored and transitional files, yet an
// exact ignore entry must stay satisfied and a transitional path must still
// block closure (DEC-006).
func TestFinalCandidateCoverageUsesRepositoryCensus(t *testing.T) {
	host := Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}
	loadHead := func(t *testing.T, root string) *Project {
		t.Helper()
		base, err := projectwork.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
		if err != nil {
			t.Fatal(err)
		}
		return base
	}
	t.Run("exact ignore", func(t *testing.T) {
		root := makeFullVerifyFixture(t)
		writeE2E(t, root, projectcoverage.IgnorePath, "apiVersion: "+projectcoverage.IgnoreAPIVersion+"\nkind: RepositoryIgnore\nentries:\n  - path: scratch.txt\n    reason: Local notes outside the model\n")
		writeE2E(t, root, "scratch.txt", "ignored notes\n")
		gitE2E(t, root, "add", projectcoverage.IgnorePath, "scratch.txt")
		gitE2E(t, root, "commit", "-m", "ignore one exact path")
		base := loadHead(t, root)
		if base.Coverage == nil || !base.Coverage.Conforming {
			t.Fatalf("precondition: census not conforming: %+v", base.Coverage)
		}
		candidate := candidateData{ID: "candidate-exact-ignore", Files: map[string]File{}}
		if err := validateFinalCandidate(host, root, base, candidate, PlanRecord{ModelDigest: base.Report.ModelDigest}); err != nil {
			t.Fatalf("census-satisfied exact ignore entry failed final candidate validation: %v", err)
		}
	})
	t.Run("transitional", func(t *testing.T) {
		root := makeFullVerifyFixture(t)
		manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath)))
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(manifest), "exclusions: []\n", "exclusions: []\ntransitionalExclusions:\n  - path: legacy/\n    reason: Existing file awaits explicit modeling\n", 1)
		if updated == string(manifest) {
			t.Fatal("could not add a transitional exclusion")
		}
		writeE2E(t, root, projectwork.ManifestPath, updated)
		writeE2E(t, root, "legacy/old.txt", "legacy bytes\n")
		gitE2E(t, root, "add", projectwork.ManifestPath, "legacy/old.txt")
		gitE2E(t, root, "commit", "-m", "mark legacy path transitional")
		base := loadHead(t, root)
		if base.Coverage == nil || !base.Coverage.Accounted || base.Coverage.Conforming {
			t.Fatalf("precondition: census must be accounted but nonconforming: %+v", base.Coverage)
		}
		candidate := candidateData{ID: "candidate-transitional", Files: map[string]File{}}
		if err := validateFinalCandidate(host, root, base, candidate, PlanRecord{ModelDigest: base.Report.ModelDigest}); err == nil || !strings.Contains(err.Error(), "coverage is not conforming") {
			t.Fatalf("transitional path did not block final candidate validation: %v", err)
		}
	})
	// The remaining candidates change the model, so they carry the regenerated
	// document the run adds before closure.
	validateModelChange := func(t *testing.T, root string, base *Project, files map[string]File) error {
		t.Helper()
		candidate := candidateData{ID: "candidate-model-change", Files: files}
		compiled, err := finalProjectForCandidate(host, root, base, candidate)
		if err != nil {
			return err
		}
		document, err := projectwork.Document(compiled, false)
		if err != nil {
			t.Fatal(err)
		}
		documentPath := projectwork.DocumentPath(compiled.Config)
		candidate.Files[documentPath] = File{Path: documentPath, Mode: "100644", Content: []byte(document)}
		return validateFinalCandidate(host, root, base, candidate, PlanRecord{ModelDigest: compiled.Report.ModelDigest})
	}
	const ordersArtifact, ordersFile = ".markitect/model/orders/artifact.yaml", "src/orders/implementation.txt"
	writeFile := func(path, content string) File { return File{Path: path, Mode: "100644", Content: []byte(content)} }
	t.Run("rename modelled file", func(t *testing.T) {
		root := makeFullVerifyFixture(t)
		base := loadHead(t, root)
		const renamed = "src/orders/renamed.txt"
		files := map[string]File{
			ordersFile:     {Path: ordersFile, Delete: true},
			renamed:        writeFile(renamed, string(base.Snapshot.Files[ordersFile])),
			ordersArtifact: writeFile(ordersArtifact, e2eArtifact("orders", "orders-code", "orders-work", "orders-check", renamed)),
		}
		if err := validateModelChange(t, root, base, files); err != nil {
			t.Fatalf("rename with its Artifact path moved failed final candidate validation: %v", err)
		}
	})
	t.Run("delete modelled file", func(t *testing.T) {
		root := makeFullVerifyFixture(t)
		const notes = "src/orders/notes.txt"
		writeE2E(t, root, ordersArtifact, e2eArtifact("orders", "orders-code", "orders-work", "orders-check", ordersFile+", "+notes))
		writeE2E(t, root, notes, "orders notes\n")
		gitE2E(t, root, "add", ordersArtifact, notes)
		gitE2E(t, root, "commit", "-m", "realize orders notes")
		base := loadHead(t, root)
		files := map[string]File{
			notes:          {Path: notes, Delete: true},
			ordersArtifact: writeFile(ordersArtifact, e2eArtifact("orders", "orders-code", "orders-work", "orders-check", ordersFile)),
		}
		if err := validateModelChange(t, root, base, files); err != nil {
			t.Fatalf("delete with its Artifact path dropped failed final candidate validation: %v", err)
		}
	})
	t.Run("transitional file modelled in place", func(t *testing.T) {
		root := makeFullVerifyFixture(t)
		manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath)))
		if err != nil {
			t.Fatal(err)
		}
		const legacy = "src/orders/legacy.txt"
		transitional := strings.Replace(string(manifest), "exclusions: []\n", "exclusions: []\ntransitionalExclusions:\n  - path: "+legacy+"\n    reason: Existing file awaits explicit modeling\n", 1)
		if transitional == string(manifest) {
			t.Fatal("could not add a transitional exclusion")
		}
		writeE2E(t, root, projectwork.ManifestPath, transitional)
		writeE2E(t, root, legacy, "legacy orders bytes\n")
		gitE2E(t, root, "add", projectwork.ManifestPath, legacy)
		gitE2E(t, root, "commit", "-m", "mark legacy orders file transitional")
		base := loadHead(t, root)
		if base.Coverage == nil || !base.Coverage.Accounted || base.Coverage.Conforming {
			t.Fatalf("precondition: census must be accounted but nonconforming: %+v", base.Coverage)
		}
		files := func() map[string]File {
			return map[string]File{
				projectwork.ManifestPath: writeFile(projectwork.ManifestPath, string(manifest)),
				ordersArtifact:           writeFile(ordersArtifact, e2eArtifact("orders", "orders-code", "orders-work", "orders-check", ordersFile+", "+legacy)),
			}
		}
		if err := validateModelChange(t, root, base, files()); err != nil {
			t.Fatalf("transitional file modelled in place failed final candidate validation: %v", err)
		}
		// The run renders its document from the same closure compile.
		store, err := newRunStore(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(RunsPath)), 0o700); err != nil {
			t.Fatal(err)
		}
		dir, err := store.createRun("00000000000000000000000000000001")
		if err != nil {
			t.Fatal(err)
		}
		completed, err := completeCandidateDocument(host, root, store, dir, base, candidateData{ID: "00000000000000000000000000000002", Files: files()})
		if err != nil {
			t.Fatal(err)
		}
		compiled, err := finalProjectForCandidate(host, root, base, completed)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateFinalCandidate(host, root, base, completed, PlanRecord{ModelDigest: compiled.Report.ModelDigest}); err != nil {
			t.Fatalf("run-generated document did not match the closure compile: %v", err)
		}
	})
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
