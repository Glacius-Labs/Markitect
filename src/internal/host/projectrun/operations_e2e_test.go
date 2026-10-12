//go:build !runtime_unit

package projectrun

import (
	"context"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

func TestTargetedPlanStillFullVerifiesUnimpactedManagerDrift(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	writeE2E(t, root, "src/inventory/implementation.txt", "UNIMPACTED_DRIFT\n")
	gitE2E(t, root, "add", "src/inventory/implementation.txt")
	gitE2E(t, root, "commit", "-m", "introduce unimpacted fixture drift")
	logPath := configureOperationsFullVerify(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot, PlanEdit: projectwork.PlanEdit}
	plan, err := Plan(host, root, head, PlanRequest{Goal: "Implement the requested Orders change", Managers: []string{e2eManagerID("orders", "orders")}})
	if err != nil {
		t.Fatalf("targeted Plan: %v", err)
	}
	if len(plan.Managers) != 2 || plan.Managers[0].ManagerID != e2eManagerID("", "project-owner") || plan.Managers[1].ManagerID != e2eManagerID("orders", "orders") {
		t.Fatalf("Plan should stay scoped to root and Orders: %+v", plan.Managers)
	}
	report, err := FullVerify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "failed" {
		t.Fatalf("full Verify did not reject the unimpacted drift: status=%s err=%v", report.Status, err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "inventory") {
		t.Fatalf("unimpacted Inventory was not independently audited: %s", data)
	}
}

func TestFullCoverageBlocksUnclassifiedOrdinaryFile(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	writeE2E(t, root, "unclassified.txt", "ordinary repository content\n")
	gitE2E(t, root, "add", "unclassified.txt")
	gitE2E(t, root, "commit", "-m", "add unclassified ordinary file")
	project, err := projectwork.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	if project.Coverage == nil || project.Coverage.Accounted {
		t.Fatalf("full coverage accepted an unclassified ordinary file: %+v", project.Coverage)
	}
	configureOperationsFullVerify(t, root)
	report, err := FullVerify(context.Background(), Host{Load: projectwork.Load}, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: project.Revision})
	if err == nil || report.Status == "passed" {
		t.Fatalf("full Verify closed despite an unclassified ordinary file: report=%+v err=%v", report, err)
	}
}

func TestCleanupAndReconcileNoOpVerifyFullTreeAndGuardedApply(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{OperationCleanup, OperationReconcile} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			root := makeFullVerifyFixture(t)
			configureOperationsNoOpRuntime(t, root)
			head := gitE2E(t, root, "rev-parse", "HEAD")
			host := projectworkHost()
			plan, err := Plan(host, root, head, PlanRequest{Operation: operation, Goal: "Review the complete project and improve only if warranted", ExecuteAuthorized: true})
			if err != nil {
				t.Fatalf("plan %s: %v", operation, err)
			}
			if len(plan.Managers) != 3 || len(plan.Checks) != 2 {
				t.Fatalf("%s plan did not include every Manager and check: managers=%d checks=%d", operation, len(plan.Managers), len(plan.Checks))
			}
			ordersStrictness := plan.Strictness[e2eManagerID("orders", "orders")]
			if ordersStrictness.Counterexamples != 2 || !containsString(ordersStrictness.Evidence, "project checks reviewed") || !containsString(ordersStrictness.Evidence, "orders contract review") {
				t.Fatalf("%s bypassed or replaced additive Orders strictness: %+v", operation, ordersStrictness)
			}
			run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
			if err != nil || run.Status != StatusIntegrated {
				debug, _ := os.ReadFile(runtimeFixtureEnv(t, operationsVerifyLogEnv))
				runtimeConfig, _ := projectconfig.LoadRuntime(root)
				reviewer := runtimeConfig.Review.Agents[e2eManagerID("inventory", "inventory")]
				t.Fatalf("%s no-op run did not integrate: status=%s err=%v reviewer trace=%s reviewer=%+v", operation, run.Status, err, debug, reviewer)
			}
			for _, task := range run.Tasks {
				if task.ManagerID != e2eManagerID("", "project-owner") && task.ReportStatus != "no-op" {
					t.Errorf("Manager %s did not report a justified no-op: %s", task.ManagerID, task.ReportStatus)
				}
			}
			verified, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, run.ID)
			if err != nil || verified.Status != "verified" {
				t.Fatalf("%s no-op did not pass checks and full Verify: status=%s err=%v scope=%s", operation, verified.Status, err, verified.VerificationScope)
			}
			preflight, err := PreflightApply(host, root, run.ID, run.Candidate.ID)
			if err != nil || len(preflight.Paths) != 0 {
				t.Fatalf("%s no-op preflight = %+v, %v; want guarded empty-path apply", operation, preflight, err)
			}
			untracked := filepath.Join(root, "untracked-ordinary.txt")
			if err := os.WriteFile(untracked, []byte("new ordinary file\n"), 0600); err != nil {
				t.Fatal(err)
			}
			_, staleErr := Apply(host, projectexecutor.ProcessInvoker{}, root, ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
				ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree})
			if staleErr == nil {
				t.Fatal("Apply accepted a changed repository after no-op preflight")
			}
			if err := os.Remove(untracked); err != nil {
				t.Fatal(err)
			}
			preflight, err = PreflightApply(host, root, run.ID, run.Candidate.ID)
			if err != nil {
				t.Fatalf("fresh no-op preflight: %v", err)
			}
			applied, err := Apply(host, projectexecutor.ProcessInvoker{}, root, ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
				ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree})
			if err != nil || applied.Status != "applied" || len(applied.Written) != 0 {
				t.Fatalf("%s verified no-op Apply = %+v, %v", operation, applied, err)
			}
		})
	}
}

func TestCleanupSourceChangeGeneratesAndAppliesExactVerifiedDocument(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	configureOperationsNoOpRuntime(t, root)
	setRuntimeFixtureEnv(t, operationsSourceChangeEnv, "1")
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := projectworkHost()
	plan, err := Plan(host, root, head, PlanRequest{Operation: OperationCleanup,
		Goal: "Review the complete project and improve Orders where the accepted model permits it", ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("plan cleanup source change: %v", err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("integrate cleanup source change: status=%s err=%v", run.Status, err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	composed, err := snapshotWithCandidate(base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	project, err := host.FromSnapshot(root, composed)
	if err != nil {
		t.Fatal(err)
	}
	expectedDocument, err := projectwork.Document(project, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(project.Snapshot.Files["README.md"]) != expectedDocument {
		t.Fatal("integrated candidate README is not the exact Host-generated document for its source snapshot")
	}
	paths := candidateDeltaPaths(base.Snapshot, candidate)
	if !containsString(paths, "src/orders/implementation.txt") {
		t.Fatalf("candidate did not bind its implementation change: %v", paths)
	}
	verified, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, run.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("verify source and generated document together: status=%s err=%v", verified.Status, err)
	}
	preflight, err := PreflightApply(host, root, run.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("source-change preflight: %v", err)
	}
	if !containsString(preflight.Paths, "src/orders/implementation.txt") {
		t.Fatalf("guarded Apply omitted the exact verified implementation path: %+v", preflight.Paths)
	}
	applied, err := Apply(host, projectexecutor.ProcessInvoker{}, root, ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree})
	if err != nil || applied.Status != "applied" {
		t.Fatalf("apply verified source and generated document: report=%+v err=%v", applied, err)
	}
	contents, err := os.ReadFile(filepath.Join(root, "src/orders/implementation.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "orders implementation v2\n" {
		t.Fatalf("applied Orders source = %q", contents)
	}
	final, err := host.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	finalDocument, err := projectwork.Document(final, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(final.Snapshot.Files["README.md"]) != finalDocument || finalDocument != expectedDocument {
		t.Fatal("applied README differs from the same Host-generated document verified in the candidate")
	}
}
