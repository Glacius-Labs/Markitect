package projectrun

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

func TestFullApplyVerificationBindsAllManagersAndReceipts(t *testing.T) {
	compiled := fullApplyProjectFacts()
	candidate := candidateData{ID: "candidate-full-apply", Files: map[string]File{}}
	briefings := map[string]string{}
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
	digest, err := FullVerifyReportDigest(*verification.ManagerVerification)
	if err != nil {
		t.Fatal(err)
	}
	verification.ManagerVerification.Digest = digest
	if err := validateFullApplyVerification(plan, candidate, compiled, verification, briefings); err == nil {
		t.Fatal("full verification with an omitted Manager passed apply validation")
	}
}

func TestFullApplyAcceptsVerifiedImplementationDigestDifferentFromPlanReport(t *testing.T) {
	base := fullApplyProjectFacts()
	compiled := fullApplyProjectFacts()
	compiled.Report.Digest = "sha256:candidate-report"
	compiled.Digest = "sha256:candidate-project"
	content := []byte("verified candidate implementation update\n")
	compiled.Snapshot.Files["src/orders/implementation.txt"] = content
	candidate := candidateData{ID: "candidate-updated-implementation", Files: map[string]File{
		"src/orders/implementation.txt": {Path: "src/orders/implementation.txt", Mode: snapshot.RegularMode, Content: content},
	}}
	host := Host{
		FromSnapshot: func(root string, selected *snapshot.Snapshot) (*projectvalue.Project, error) {
			projected := *base
			projected.Root, projected.Snapshot = root, selected
			return &projected, nil
		},
		ClassifyCandidate: func(_, candidate *projectvalue.Project, _ ...string) (*projectvalue.Project, error) {
			classified := *candidate
			classified.Report, classified.Digest, classified.Coverage = compiled.Report, compiled.Digest, compiled.Coverage
			return &classified, nil
		},
		RenderDocument: func(*projectvalue.Project) (string, error) { return "fixed generated view\n", nil },
	}
	if compiled.Report.ModelDigest != base.Report.ModelDigest || compiled.Report.Digest == base.Report.Digest {
		t.Fatalf("finite implementation facts did not isolate model-stable project digest change: model=%t report=%t", compiled.Report.ModelDigest == base.Report.ModelDigest, compiled.Report.Digest != base.Report.Digest)
	}
	if err := validateFinalCandidate(host, t.TempDir(), base, candidate, PlanRecord{ModelDigest: base.Report.ModelDigest}); err != nil {
		t.Fatalf("valid implementation candidate facts rejected: %v", err)
	}
	briefings := map[string]string{}
	plan, verification := passingFullApplyEvidence(t, candidate, compiled, briefings)
	plan.ReportDigest = base.Report.Digest
	if err := validateFullApplyVerification(plan, candidate, compiled, verification, briefings); err != nil {
		t.Fatalf("valid candidate-bound report digest differing from the pre-execution plan was rejected: %v", err)
	}
}

func TestFullApplyRejectsUnclassifiedCandidatePath(t *testing.T) {
	base := fullApplyProjectFacts()
	compiled := fullApplyProjectFacts()
	compiled.Coverage.Conforming = false
	candidate := candidateData{ID: "candidate-unclassified", Files: map[string]File{
		"unexpected/unowned.txt": {Path: "unexpected/unowned.txt", Mode: snapshot.RegularMode, Content: []byte("unclassified candidate content\n")},
	}}
	host := Host{
		FromSnapshot: func(root string, selected *snapshot.Snapshot) (*projectvalue.Project, error) {
			projected := *base
			projected.Root, projected.Snapshot = root, selected
			return &projected, nil
		},
		ClassifyCandidate: func(_, candidate *projectvalue.Project, _ ...string) (*projectvalue.Project, error) {
			classified := *candidate
			classified.Coverage = compiled.Coverage
			return &classified, nil
		},
	}
	if err := validateFinalCandidate(host, t.TempDir(), base, candidate, PlanRecord{ModelDigest: compiled.Report.ModelDigest}); err == nil {
		t.Fatal("unclassified candidate path passed final candidate validation")
	}
	if err := requireFullCoverage(compiled); err == nil {
		t.Fatal("nonconforming explicit candidate facts passed the full-coverage closure gate")
	}
}

func TestFullApplyRejectsDeletionOutsideReviewedSnapshot(t *testing.T) {
	root := t.TempDir()
	base := closureReviewProjectFacts(root)
	deletion := candidateData{ID: "candidate-delete-census-only-file", Files: map[string]File{
		fixtureLegacyOrdersFile: {Path: fixtureLegacyOrdersFile, Delete: true},
	}}
	if _, included := base.Snapshot.Files[fixtureLegacyOrdersFile]; included {
		t.Fatal("precondition: census-only legacy file unexpectedly appears in the reviewed snapshot")
	}
	if changed := candidateDeltaPaths(base.Snapshot, deletion); len(changed) != 0 {
		t.Fatalf("deleting a census-only path changed the reviewed snapshot delta: %v", changed)
	}
	classifiedDeletion := false
	host := Host{
		FromSnapshot: func(gotRoot string, selected *snapshot.Snapshot) (*projectvalue.Project, error) {
			candidate := *base
			candidate.Root, candidate.Snapshot = gotRoot, selected
			return &candidate, nil
		},
		ClassifyCandidate: func(gotBase, candidate *projectvalue.Project, deleted ...string) (*projectvalue.Project, error) {
			if gotBase != base || candidate.Snapshot == nil {
				return nil, fmt.Errorf("candidate classifier did not receive the reviewed base and projected snapshot")
			}
			if _, stillPresent := candidate.Snapshot.Files[fixtureLegacyOrdersFile]; stillPresent {
				return nil, fmt.Errorf("deleted census-only path remained in the projected candidate")
			}
			if len(deleted) != 1 || deleted[0] != fixtureLegacyOrdersFile {
				return nil, fmt.Errorf("candidate deletion set = %v", deleted)
			}
			classifiedDeletion = true
			return nil, fmt.Errorf("candidate deletes %s outside the reviewed snapshot", deleted[0])
		},
	}
	err := validateFinalCandidate(host, root, base, deletion, PlanRecord{ModelDigest: base.Report.ModelDigest})
	if !classifiedDeletion || err == nil || !strings.Contains(err.Error(), "outside the reviewed snapshot") {
		t.Fatalf("census-only deletion did not reach and fail candidate classification: classified=%t err=%v", classifiedDeletion, err)
	}
}

func fullApplyProjectFacts() *projectvalue.Project {
	managerID := "manager:orders"
	return &projectvalue.Project{
		Revision: strings.Repeat("a", 40), Digest: "sha256:fixed-project",
		Config: projectvalue.Config{CoverageMode: "full", DocumentPath: "README.md"},
		Snapshot: &snapshot.Snapshot{Files: map[string][]byte{
			".markitect/repository-ignore.yaml": []byte("apiVersion: " + projectcoverage.IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries: []\n"),
			"README.md":                         []byte("fixed generated view\n"), "src/orders/implementation.txt": []byte("orders implementation v1\n"),
		}, Modes: map[string]string{".markitect/repository-ignore.yaml": snapshot.RegularMode, "README.md": snapshot.RegularMode, "src/orders/implementation.txt": snapshot.RegularMode}},
		Coverage: &projectcoverage.Report{Digest: "sha256:coverage", Accounted: true, Conforming: true},
		Report: projectmodel.Report{Digest: "sha256:report", ModelDigest: "sha256:model",
			Managers: []projectmodel.Manager{{ID: managerID, Name: "orders"}},
			Checks:   []projectmodel.Check{{ID: "check:orders", Owner: managerID, Command: []string{"true"}}},
		},
	}
}

// Run records reviews and Verify checks them, so both must see a candidate the
// way closure does. Modelling a transitional file in place adds base bytes to
// that view; a snapshot-only review view scopes different files, and Verify
// then finds no fresh passed review.
func TestVerifyReviewChecksAcceptReviewsOfTheClosureCandidateView(t *testing.T) {
	root := t.TempDir()
	base := closureReviewProjectFacts(root)
	closureManifest := "apiVersion: " + projectvalue.APIVersion + "\nname: Review fixture\ncoverageMode: full\ndocumentPath: docs/project.md\n"
	closureArtifact := "artifact: orders owns src/orders/implementation.txt and src/orders/legacy.txt\n"
	closureReport := base.Report
	closureReport.Digest = "sha256:closure-report"
	closureReport.ModelDigest = "sha256:closure-model"
	closureReport.Artifacts = []projectmodel.Artifact{{ID: "artifact:orders", Owner: "manager:orders", Required: true,
		Realizes: []string{"statement:orders"}, Paths: []string{"src/orders/implementation.txt", fixtureLegacyOrdersFile, "docs/project.md"}}}
	closureReport.Files = []projectmodel.FileEntry{
		{Path: "src/orders/implementation.txt", Mode: snapshot.RegularMode, Owner: "manager:orders", Class: "implementation", Artifacts: []string{"artifact:orders"}, Exists: true},
		{Path: fixtureLegacyOrdersFile, Mode: snapshot.RegularMode, Owner: "manager:orders", Class: "implementation", Artifacts: []string{"artifact:orders"}, Exists: true},
		{Path: "docs/project.md", Mode: snapshot.RegularMode, Owner: "manager:orders", Class: "documentation", Artifacts: []string{"artifact:orders"}, Exists: true},
	}
	host := Host{
		Load: func(_ string, revision string) (*projectvalue.Project, error) {
			if revision != base.Revision {
				return nil, fmt.Errorf("unexpected fixed review revision %q", revision)
			}
			return base, nil
		},
		FromSnapshot: func(root string, source *snapshot.Snapshot) (*projectvalue.Project, error) {
			candidate := *base
			candidate.Root, candidate.Snapshot = root, source
			return &candidate, nil
		},
		ClassifyCandidate: func(_ *projectvalue.Project, candidate *projectvalue.Project, _ ...string) (*projectvalue.Project, error) {
			classified := *candidate
			classified.Snapshot = cloneReviewSnapshot(candidate.Snapshot)
			classified.Snapshot.Files[fixtureLegacyOrdersFile] = []byte("fixed transitional source bytes\n")
			classified.Snapshot.Modes[fixtureLegacyOrdersFile] = snapshot.RegularMode
			classified.Report = closureReport
			classified.Report.Digest, classified.Report.ModelDigest = "sha256:closure-report", "sha256:closure-model"
			classified.Digest = "sha256:closure-project"
			classified.Coverage = &projectcoverage.Report{Digest: "sha256:closure-coverage", Accounted: true, Conforming: true}
			return &classified, nil
		},
		RenderDocument: func(*projectvalue.Project) (string, error) {
			return projectvalue.GeneratedViewMarker + "\n" + fixtureLegacyOrdersFile + "\n", nil
		},
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(RunsPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	const runID = "00000000000000000000000000000001"
	dir, err := store.createRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	initial := candidateData{ID: "00000000000000000000000000000002", Files: map[string]File{
		projectvalue.ManifestPath: candidateWrite(projectvalue.ManifestPath, closureManifest),
		fixtureOrdersArtifact:     candidateWrite(fixtureOrdersArtifact, closureArtifact),
	}}
	if err := store.writeCandidate(dir, initial); err != nil {
		t.Fatal(err)
	}
	candidate, err := completeCandidateDocument(host, root, store, dir, base, initial)
	if err != nil {
		t.Fatal(err)
	}
	reviewed, err := finalProjectForCandidate(host, root, base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	managerID := e2eManagerID("orders", "orders")
	var artifactID string
	for _, artifact := range reviewed.Report.Artifacts {
		if artifact.Owner == managerID {
			artifactID = artifact.ID
		}
	}
	task := ManagerTask{ID: "task-orders", ManagerID: managerID, Goal: "Model the legacy orders file.", Artifacts: []string{artifactID}}
	plan := PlanRecord{ID: runID, Goal: "Model the legacy orders file.", Operation: OperationApply, BaseRevision: base.Revision,
		InitialCandidateID: initial.ID, Managers: []ManagerTask{task},
		BriefingDigests: map[string]string{}, Strictness: map[string]projectconfig.StrictnessProfile{}}
	run := RunReport{ID: runID, Tasks: []ManagerTask{task}}
	reviewState, err := reviewCandidateStateFor(host, root, plan, run, candidate)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := reviewScopeDigest(plan, reviewed, task, "work", run, reviewState)
	if err != nil {
		t.Fatal(err)
	}
	run.Reviews = []ReviewRecord{{TaskID: task.ID, ManagerID: managerID, Phase: "work", CandidateID: candidate.ID, CandidateDigest: candidate.Digest,
		ScopeDigest: scope, InputDigest: "sha256:review-input", Outcome: "pass", Findings: []ReviewFinding{}, Receipt: agentexec.Receipt{RunID: "review-run"}}}
	runtime := projectconfig.Runtime{Review: &projectconfig.ReviewConfig{}}
	if err := requireFreshReviews(host, root, store, dir, base, candidate, plan, runtime, run); err != nil {
		t.Fatalf("Verify's review freshness check rejected a review of the closure view: %v", err)
	}
	evidence, err := freshReviewEvidenceForVerify(store, dir, reviewed, candidate, plan, runtime, run, host)
	if err != nil || len(evidence) != 1 {
		t.Fatalf("Verify's review evidence = %+v, err=%v", evidence, err)
	}
	files := reviewCandidateFiles(reviewed, task, run.Tasks, "work")
	if !containsString(func() []string {
		paths := make([]string, 0, len(files))
		for _, file := range files {
			paths = append(paths, file.Path)
		}
		return paths
	}(), fixtureLegacyOrdersFile) {
		t.Fatalf("closure review scope omitted the unchanged census-sourced file: %+v", files)
	}
}

func closureReviewProjectFacts(root string) *projectvalue.Project {
	const document = "docs/project.md"
	managerID := "manager:orders"
	return &projectvalue.Project{
		Root: root, Revision: strings.Repeat("b", 40), Digest: "sha256:base-project",
		Config: projectvalue.Config{CoverageMode: "full", DocumentPath: document, InventoryRoots: []string{"src"}},
		Snapshot: &snapshot.Snapshot{Files: map[string][]byte{
			projectvalue.ManifestPath:       []byte("apiVersion: " + projectvalue.APIVersion + "\nname: Review fixture\ncoverageMode: full\ndocumentPath: docs/project.md\ntransitionalExclusions:\n  - path: src/orders/legacy.txt\n    reason: Awaiting explicit modeling\n"),
			fixtureOrdersArtifact:           []byte("artifact: orders owns src/orders/implementation.txt\n"),
			projectcoverage.IgnorePath:      []byte("apiVersion: " + projectcoverage.IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries: []\n"),
			document:                        []byte(projectvalue.GeneratedViewMarker + "\ntransitional legacy path\n"),
			"src/orders/implementation.txt": []byte("fixed orders implementation\n"),
		}, Modes: map[string]string{
			projectvalue.ManifestPath: snapshot.RegularMode, fixtureOrdersArtifact: snapshot.RegularMode,
			projectcoverage.IgnorePath: snapshot.RegularMode, document: snapshot.RegularMode,
			"src/orders/implementation.txt": snapshot.RegularMode,
		}},
		Coverage: &projectcoverage.Report{Digest: "sha256:base-coverage", Accounted: true, Conforming: false},
		Report: projectmodel.Report{Digest: "sha256:base-report", ModelDigest: "sha256:base-model",
			Managers:   []projectmodel.Manager{{ID: managerID, Name: "orders"}},
			Statements: []projectmodel.Statement{{ID: "statement:orders", Owner: managerID, Public: true, Description: "Own orders implementation."}},
			Artifacts:  []projectmodel.Artifact{{ID: "artifact:orders", Owner: managerID, Required: true, Realizes: []string{"statement:orders"}, Paths: []string{"src/orders/implementation.txt", document}}},
			Files: []projectmodel.FileEntry{
				{Path: "src/orders/implementation.txt", Mode: snapshot.RegularMode, Owner: managerID, Class: "implementation", Artifacts: []string{"artifact:orders"}, Exists: true},
				{Path: document, Mode: snapshot.RegularMode, Owner: managerID, Class: "documentation", Artifacts: []string{"artifact:orders"}, Exists: true},
			},
		},
	}
}

func cloneReviewSnapshot(source *snapshot.Snapshot) *snapshot.Snapshot {
	clone := &snapshot.Snapshot{ID: source.ID, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, content := range source.Files {
		clone.Files[path] = append([]byte(nil), content...)
		clone.Modes[path] = source.Modes[path]
	}
	return clone
}

const (
	fixtureOrdersArtifact   = ".markitect/model/orders/artifact.yaml"
	fixtureLegacyOrdersFile = "src/orders/legacy.txt"
)

func candidateWrite(path, content string) File {
	return File{Path: path, Mode: "100644", Content: []byte(content)}
}

func passingFullApplyEvidence(t *testing.T, candidate candidateData, compiled *projectvalue.Project, briefings map[string]string) (PlanRecord, VerifyReport) {
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
