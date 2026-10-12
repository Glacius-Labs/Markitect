//go:build !runtime_unit

package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

// A candidate may model a transitional file in place: it drops the exclusion
// and leaves the bytes unchanged. Closure reads those bytes from the base, so
// the declared checks Verify runs must see them too, and so must full
// verification when it runs checks itself.
func TestVerifyChecksSeeFilesModelledInPlace(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	configureOperationsFullVerify(t, root)
	setRuntimeFixtureEnv(t, modelledFileCheckEnv, "1")
	updateE2ERuntime(t, root, func(config *Runtime) {
		for id, agent := range config.Agents {
			agent.Environment = append(agent.Environment, modelledFileCheckEnv)
			config.Agents[id] = agent
		}
	})
	manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	transitional := strings.Replace(string(manifest), "exclusions: []\n", "exclusions: []\ntransitionalExclusions:\n  - path: "+fixtureLegacyOrdersFile+"\n    reason: Existing file awaits explicit modeling\n", 1)
	if transitional == string(manifest) {
		t.Fatal("could not add a transitional exclusion")
	}
	writeE2E(t, root, projectwork.ManifestPath, transitional)
	writeE2E(t, root, fixtureLegacyOrdersFile, legacyOrdersBytes)
	writeE2E(t, root, fixtureOrdersArtifact, e2eArtifact("orders", "orders-code", "orders-work", "orders-check", fixtureOrdersFile+", "+fixtureLegacyOrdersFile))
	for _, check := range []string{".markitect/model/orders/check.yaml", ".markitect/model/inventory/check.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(check)))
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(data), "TestProjectRunCheckProcess", "TestModelledFileCheckProcess", 1)
		if updated == string(data) {
			t.Fatalf("could not point %s at the modelled-file check", check)
		}
		writeE2E(t, root, check, updated)
	}
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "-m", "model the transitional legacy orders file")
	host := projectworkHost()
	plan, err := Plan(host, root, gitE2E(t, root, "rev-parse", "HEAD"), PlanRequest{Goal: "Model the legacy orders file in place.", ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
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
	// The integrated candidate only drops the exclusion and carries the
	// document the run would generate for it.
	final := candidateData{Parents: []string{plan.InitialCandidateID}, Files: map[string]File{projectwork.ManifestPath: candidateWrite(projectwork.ManifestPath, string(manifest))}}
	compiled, err := finalProjectForCandidate(host, root, base, final)
	if err != nil {
		t.Fatal(err)
	}
	document, err := projectwork.Document(compiled, false)
	if err != nil {
		t.Fatal(err)
	}
	documentPath := projectwork.DocumentPath(compiled.Config)
	final.Files[documentPath] = candidateWrite(documentPath, document)
	if final.ID, err = newID(); err != nil {
		t.Fatal(err)
	}
	if err := store.writeCandidate(dir, final); err != nil {
		t.Fatal(err)
	}
	if final, err = store.readCandidate(dir, final.ID); err != nil {
		t.Fatal(err)
	}
	if compiled, err = finalProjectForCandidate(host, root, base, final); err != nil {
		t.Fatal(err)
	}
	if string(compiled.Snapshot.Files[fixtureLegacyOrdersFile]) != legacyOrdersBytes {
		t.Fatal("precondition: closure did not bind the modelled file's base bytes")
	}
	report := RunReport{APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Operation: plan.Operation, Status: StatusIntegrated, Mode: ModeControlledLocal,
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot,
		ModelDigest: plan.ModelDigest, RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers),
		Candidate: CandidateRef{ID: final.ID, Snapshot: final.Digest, Files: map[string]string{}, Integrated: true}, Revision: 1}
	for i := range report.Tasks {
		report.Tasks[i].ReportStatus = "complete"
	}
	var reviews []ReviewRecord
	for _, task := range report.Tasks {
		phase := "work"
		if len(activeChildren(report.Tasks, task.ManagerID)) > 0 {
			phase = "integrate"
		}
		if !phaseReviewRequired(compiled, task, report.Tasks, phase) {
			continue
		}
		scope, err := reviewScopeDigest(plan, compiled, task, phase, report)
		if err != nil {
			t.Fatal(err)
		}
		reviews = append(reviews, ReviewRecord{TaskID: task.ID, ManagerID: task.ManagerID, Phase: phase, CandidateID: final.ID, CandidateDigest: final.Digest,
			ScopeDigest: scope, InputDigest: "sha256:review-input", Outcome: "pass", Findings: []ReviewFinding{}, Receipt: agentexec.Receipt{RunID: "review-" + task.ID}})
	}
	report.Reviews = reviews
	if err := persistState(store, &report); err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified {
		t.Fatalf("Verify: report=%+v err=%v", verified, err)
	}
	assertChecksObservedLegacyFile(t, verified.Checks)
	if verified.ManagerVerification == nil || verified.ManagerVerification.SnapshotDigest != compiled.Snapshot.Digest() {
		t.Fatal("verification did not bind the closure snapshot that holds the modelled file's bytes")
	}
	// Full verification materializes its own check tree when no planned check
	// result covers a declared check.
	runtimeConfig, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	full, err := FullVerifyProject(context.Background(), host, ProcessInvoker{}, root, compiled, runtimeConfig, FullVerifyBinding{ExpectedSnapshot: compiled.Snapshot.Digest()})
	if err != nil || full.Status != "passed" {
		t.Fatalf("FullVerifyProject: report=%+v err=%v", full, err)
	}
	assertChecksObservedLegacyFile(t, full.Checks)
}

// Declared checks once ran in a materialized candidate under the repository's
// run store. Below a deep repository (or GOTMPDIR) that directory exceeded the
// Windows working-directory limit and every check failed with "The directory
// name is invalid". Checks now run in a fresh short temporary directory.
func TestVerifyRunsDeclaredChecksOutsideTheRunStore(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	plan := planBothManagers(t, root)
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified {
		t.Fatalf("Verify: report=%+v err=%v", verified, err)
	}
	cleanRoot := strings.ToLower(filepath.Clean(root))
	for _, check := range verified.Checks {
		_, cwd, found := strings.Cut(strings.TrimSpace(check.Stdout), "check cwd: ")
		if !found {
			t.Fatalf("check did not report its working directory: %+v", check)
		}
		if strings.HasPrefix(strings.ToLower(filepath.Clean(cwd)), cleanRoot) {
			t.Fatalf("check ran inside the repository run store: %s", cwd)
		}
	}
}
