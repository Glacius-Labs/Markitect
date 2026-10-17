//go:build !runtime_unit

package projectrun

import (
	"context"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectworkspace"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBringYourOwnExecutorsWithMixedRoleProfilesDeliverWithHonestCost(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	configureMixedRoleRuntime(t, root)
	plan := planBothManagers(t, root)
	run, err := Run(context.Background(), runtimeLifecycleHost(t, root), projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("mixed-role run did not integrate: status=%s err=%v", run.Status, err)
	}
	var managerCost int64
	reviews := 0
	for _, invocation := range run.Invocations {
		switch invocation.Role {
		case "reviewer":
			reviews++
			if invocation.CostKnown || invocation.CostMicros != 0 || invocation.Receipt.Usage != nil {
				t.Fatalf("unmetered reviewer cost was invented: %+v", invocation)
			}
		default:
			if !invocation.CostKnown || invocation.CostMicros <= 0 {
				t.Fatalf("metered Manager invocation lost its known cost: %+v", invocation)
			}
			managerCost += invocation.CostMicros
		}
	}
	if reviews == 0 || run.CostAccounting != CostAccountingPartial || run.CostMicros != managerCost {
		t.Fatalf("cost report = %s/%d with %d reviews, want partial accounting of the metered Manager cost %d", run.CostAccounting, run.CostMicros, reviews, managerCost)
	}
	verifyAndApply(t, root, plan, run.Candidate.ID)
	records, err := readE2ERecords(runtimeFixtureEnv(t, e2eLogEnv))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, record := range records {
		phase, _ := record["phase"].(string)
		model, _ := record["agentModel"].(string)
		want := e2eManagerModel
		switch phase {
		case "review":
			want = e2eReviewerModel
		case "verify":
			want = e2eVerifierModel
		}
		if model != want {
			t.Fatalf("%s invocation ran model %q, want its role profile %q", phase, model, want)
		}
		seen[phase] = true
	}
	if !seen["work"] || !seen["integrate"] || !seen["review"] || !seen["verify"] {
		t.Fatalf("not every role ran: %v", seen)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	final, err := store.readLatestState(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if verifier := lastInvocation(final.Invocations); verifier.Role != "verifier" || verifier.CostKnown {
		t.Fatalf("unmetered verifier was not recorded with unknown cost: %+v", verifier)
	}
	assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v2\n")
	assertFileContents(t, root, "src/inventory/implementation.txt", "inventory implementation v2\n")
}

func TestMeteredProcessReviewerWithoutUsageStillStops(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	configureMixedRoleRuntime(t, root)
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
		for id, reviewer := range config.Review.Agents {
			reviewer.CostMode, reviewer.Pricing = "", projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}
			config.Review.Agents[id] = reviewer
		}
	})
	plan := planBothManagers(t, root)
	run, err := Run(context.Background(), runtimeLifecycleHost(t, root), projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err == nil || !strings.Contains(err.Error(), "reviewer usage is missing") || run.Status == StatusIntegrated {
		t.Fatalf("metered reviewer without usage was accepted: status=%s err=%v", run.Status, err)
	}
}

func TestBringYourOwnExecutorFaultInjection(t *testing.T) {
	t.Parallel()
	orders, inventory, owner := e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory"), e2eManagerID("", "project-owner")

	t.Run("write outside scope is rejected and never applied", func(t *testing.T) {
		t.Parallel()
		root := makeProjectRunFixture(t)
		setupE2EProcess(t, "out-of-scope")
		configureMixedRoleRuntime(t, root)
		plan := planBothManagers(t, root)
		run, err := Run(context.Background(), runtimeLifecycleHost(t, root), projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err == nil || run.Status == StatusIntegrated {
			t.Fatalf("out-of-scope write closed the run: status=%s err=%v", run.Status, err)
		}
		if _, ok := run.Candidate.Files["src/inventory/foreign.txt"]; ok {
			t.Fatalf("foreign path entered the candidate: %+v", run.Candidate.Files)
		}
		if _, err := PreflightApply(runtimeLifecycleHost(t, root), root, plan.ID, run.Candidate.ID); err == nil {
			t.Fatal("an unverified candidate passed Apply preflight")
		}
		if _, err := os.Stat(filepath.Join(root, "src", "inventory", "foreign.txt")); !os.IsNotExist(err) {
			t.Fatalf("foreign file reached the checkout: %v", err)
		}
	})

	t.Run("forgotten file is caught by the declared check and repaired", func(t *testing.T) {
		t.Parallel()
		root := makeProjectRunFixture(t)
		setupE2EProcess(t, "forgotten-file")
		configureMixedRoleRuntime(t, root)
		plan := planBothManagers(t, root)
		host := runtimeLifecycleHost(t, root)
		run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err != nil || run.Status != StatusIntegrated {
			t.Fatalf("forgotten-file run did not integrate for verification: status=%s err=%v", run.Status, err)
		}
		failed, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err == nil || failed.Status != StatusFailed || !failedCheckWith(failed.Checks, "orders check failed") {
			t.Fatalf("declared check did not catch the forgotten file: report=%+v err=%v", failed, err)
		}
		repaired, err := Repair(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err != nil || repaired.Status != StatusIntegrated || len(repaired.RepairRounds) != 1 {
			t.Fatalf("Repair did not close the check failure: status=%s rounds=%d err=%v", repaired.Status, len(repaired.RepairRounds), err)
		}
		verifyAndApply(t, root, plan, repaired.Candidate.ID)
		assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v2\n")
	})

	t.Run("review finding drives a targeted repair", func(t *testing.T) {
		t.Parallel()
		root := makeProjectRunFixture(t)
		setupE2EProcess(t, "review-defect-fix")
		configureMixedRoleRuntime(t, root)
		plan := planBothManagers(t, root)
		run, err := Run(context.Background(), runtimeLifecycleHost(t, root), projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err != nil || run.Status != StatusIntegrated {
			t.Fatalf("review repair did not integrate: status=%s err=%v", run.Status, err)
		}
		var outcomes []string
		for _, review := range run.Reviews {
			if review.ManagerID == orders && review.Phase == "work" {
				outcomes = append(outcomes, review.Outcome)
			}
		}
		if strings.Join(outcomes, ",") != "fail,pass" {
			t.Fatalf("orders reviews = %v, want a grounded failure followed by a fresh pass", outcomes)
		}
		if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), orders, "work"); calls != 2 {
			t.Fatalf("orders implementer ran %d times, want the original and one repair", calls)
		}
		if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), inventory, "work"); calls != 1 {
			t.Fatalf("unaffected inventory ran %d times, want 1", calls)
		}
		verifyAndApply(t, root, plan, run.Candidate.ID)
		assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v2\n")
	})

	t.Run("integration failure despite green children is caught and repaired", func(t *testing.T) {
		t.Parallel()
		root := makeProjectRunFixture(t)
		addRootIntegrationCheck(t, root)
		setupE2EProcess(t, "integration-stale-summary")
		setRuntimeFixtureEnv(t, e2eIntegrationCheckEnv, "1")
		configureMixedRoleRuntime(t, root)
		project := runtimeLifecycleProject(t, root, false)
		addRuntimeIntegrationFacts(t, project)
		host := processRuntimeHost(root, project)
		plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
			Managers: []string{orders, inventory}, ExecuteAuthorized: true})
		if err != nil {
			t.Fatal(err)
		}
		if !planHasCheck(plan, "integration-check") {
			t.Fatalf("plan omitted the root integration check: %+v", plan.Checks)
		}
		run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err != nil || run.Status != StatusIntegrated {
			t.Fatalf("integration run did not integrate: status=%s err=%v", run.Status, err)
		}
		for _, child := range []string{orders, inventory} {
			if task := findTask(run.Tasks, child); task == nil || task.ReviewStatus != "pass" {
				t.Fatalf("child %s is not green before integration verification: %+v", child, task)
			}
		}
		failed, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err == nil || failed.Status != StatusFailed || !failedCheckWith(failed.Checks, "integration summary") {
			t.Fatalf("integration check did not catch the inconsistent aggregate: report=%+v err=%v", failed, err)
		}
		repaired, err := Repair(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
		if err != nil || repaired.Status != StatusIntegrated {
			t.Fatalf("Repair did not close the integration failure: status=%s err=%v", repaired.Status, err)
		}
		if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), owner, "integrate"); calls < 2 {
			t.Fatalf("integration owner ran %d times, want a repair integration", calls)
		}
		verifyAndApply(t, root, plan, repaired.Candidate.ID, host)
		assertFileContents(t, root, "src/project-integration.txt", "orders implementation v2\ninventory implementation v2\n")
	})
}

// A run may mix a native Manager with bring-your-own process Managers. When the
// native turn comes first it creates the run's private log directory, and the
// later process turns must still accept it as their owner-only log directory.
func TestNativeManagerBeforeProcessManagersSharesThePrivateLogDirectory(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	writeE2E(t, root, "AGENTS.md", "Native manager instructions for the pinned mixed-runtime fixture.\n")
	gitE2E(t, root, "add", "AGENTS.md")
	gitE2E(t, root, "commit", "--quiet", "-m", "add pinned native manager instructions")
	setupE2EProcess(t, "normal")
	rootID := e2eManagerID("", "project-owner")
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		base := runtime.Agents[rootID]
		native := workspaceBridgeAgent(t, root)
		native.Command, native.Model, native.ProviderVersion, native.Timeout = base.Command, base.Model, codexappserver.SupportedProviderVersion, base.Timeout
		native.MaxStdoutBytes, native.MaxStderrBytes, native.Pricing = base.MaxStdoutBytes, base.MaxStderrBytes, base.Pricing
		native.AppServer = &projectconfig.AppServerSettings{ReasoningEffort: "medium", MaxEventBytes: 1 << 20}
		runtime.Agents[rootID] = native
	})
	project := runtimeLifecycleProject(t, root, false)
	project.Config.DocumentPath = workspaceInstructionPath
	includeFiniteRuntimeFiles(t, project, root, workspaceInstructionPath, "AGENTS.md")
	host := processRuntimeHost(root, project)
	workspaces, err := projectworkspace.NewGitService(testkit.TempDir(t), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	host.Workspaces = workspaces
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, nativeRootInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("native-first mixed run did not integrate: status=%s err=%v", run.Status, err)
	}
	if len(run.Invocations) != 4 || run.Invocations[0].TaskID != findTask(run.Tasks, rootID).ID {
		t.Fatalf("run did not start natively and continue with both process Managers: %+v", run.Invocations)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(candidate.Files["src/orders/implementation.txt"].Content); got != "orders implementation v2\n" {
		t.Fatalf("process Manager output = %q, want the fixture executor's bytes", got)
	}
}
