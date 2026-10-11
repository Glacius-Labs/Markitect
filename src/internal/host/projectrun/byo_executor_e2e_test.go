package projectrun

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// These end-to-end tests drive the delegated method through bring-your-own
// process executors with a separate profile per role: metered Managers that
// report usage, and unmetered reviewers and verifier that report none. The
// scripted executor injects faults; scope, review, Verify and Apply guards
// must hold exactly as for the native runtime.

const (
	e2eManagerModel  = "fixture-manager-model"
	e2eReviewerModel = "fixture-review-model"
	e2eVerifierModel = "fixture-verifier-model"
)

func configureMixedRoleRuntime(t *testing.T, root string) {
	t.Helper()
	updateE2ERuntime(t, root, func(config *Runtime) {
		reviewers := make(map[string]Agent, len(config.Agents))
		for id, agent := range config.Agents {
			agent.Model, agent.ModelOptions = e2eManagerModel, map[string]any{"reasoningEffort": "high"}
			agent.Pricing = Pricing{InputMicrosPerMillion: 1_000_000, OutputMicrosPerMillion: 1_000_000}
			config.Agents[id] = agent
			reviewer := agent
			reviewer.Model, reviewer.ModelOptions = e2eReviewerModel, map[string]any{"reasoningEffort": "medium"}
			reviewer.CostMode, reviewer.Pricing = CostModeUnmetered, Pricing{}
			reviewers[id] = reviewer
		}
		verifier := config.Agents[e2eManagerID("", "project-owner")]
		verifier.Model, verifier.ModelOptions = e2eVerifierModel, nil
		verifier.CostMode, verifier.Pricing = CostModeUnmetered, Pricing{}
		config.Verifier = &verifier
		config.Review = &ReviewConfig{Agents: reviewers, MaxRounds: 3, MaxManagerRounds: 2}
		config.Limits.MaxStarts = 64
		// One bounded repair round for check failures found by Verify.
		config.Limits.MaxRetries = 1
		// A loaded Windows host needs minutes per scenario for its Git calls;
		// no scenario here tests the duration budget.
		config.Limits.MaxDuration = Duration(30 * time.Minute)
	})
	t.Setenv(e2eOmitUsageEnv, "review,verify")
}

func planBothManagers(t *testing.T, root string) PlanRecord {
	t.Helper()
	plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	return plan
}

func verifyAndApply(t *testing.T, root string, plan PlanRecord, candidateID string) {
	t.Helper()
	host := projectworkHost()
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified || verified.Verifier == nil {
		t.Fatalf("Verify did not pass with the independent verifier: report=%+v err=%v", verified, err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, candidateID)
	if err != nil {
		t.Fatalf("PreflightApply: %v", err)
	}
	applied, err := Apply(host, ProcessInvoker{}, root, applyRequestFromPreflight(preflight))
	if err != nil || applied.Status != StatusApplied {
		t.Fatalf("guarded Apply failed: report=%+v err=%v", applied, err)
	}
}

func TestBringYourOwnExecutorsWithMixedRoleProfilesDeliverWithHonestCost(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	configureMixedRoleRuntime(t, root)
	plan := planBothManagers(t, root)
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
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
	records, err := readE2ERecords(os.Getenv(e2eLogEnv))
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
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	configureMixedRoleRuntime(t, root)
	updateE2ERuntime(t, root, func(config *Runtime) {
		for id, reviewer := range config.Review.Agents {
			reviewer.CostMode, reviewer.Pricing = "", Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}
			config.Review.Agents[id] = reviewer
		}
	})
	plan := planBothManagers(t, root)
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err == nil || !strings.Contains(err.Error(), "reviewer usage is missing") || run.Status == StatusIntegrated {
		t.Fatalf("metered reviewer without usage was accepted: status=%s err=%v", run.Status, err)
	}
}

func TestBringYourOwnExecutorFaultInjection(t *testing.T) {
	orders, inventory, owner := e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory"), e2eManagerID("", "project-owner")

	t.Run("write outside scope is rejected and never applied", func(t *testing.T) {
		root := makeProjectRunFixture(t)
		setupE2EProcess(t, "out-of-scope")
		configureMixedRoleRuntime(t, root)
		plan := planBothManagers(t, root)
		run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
		if err == nil || run.Status == StatusIntegrated {
			t.Fatalf("out-of-scope write closed the run: status=%s err=%v", run.Status, err)
		}
		if _, ok := run.Candidate.Files["src/inventory/foreign.txt"]; ok {
			t.Fatalf("foreign path entered the candidate: %+v", run.Candidate.Files)
		}
		if _, err := PreflightApply(projectworkHost(), root, plan.ID, run.Candidate.ID); err == nil {
			t.Fatal("an unverified candidate passed Apply preflight")
		}
		if _, err := os.Stat(filepath.Join(root, "src", "inventory", "foreign.txt")); !os.IsNotExist(err) {
			t.Fatalf("foreign file reached the checkout: %v", err)
		}
	})

	t.Run("forgotten file is caught by the declared check and repaired", func(t *testing.T) {
		root := makeProjectRunFixture(t)
		setupE2EProcess(t, "forgotten-file")
		configureMixedRoleRuntime(t, root)
		plan := planBothManagers(t, root)
		host := projectworkHost()
		run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
		if err != nil || run.Status != StatusIntegrated {
			t.Fatalf("forgotten-file run did not integrate for verification: status=%s err=%v", run.Status, err)
		}
		failed, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
		if err == nil || failed.Status != StatusFailed || !failedCheckWith(failed.Checks, "orders check failed") {
			t.Fatalf("declared check did not catch the forgotten file: report=%+v err=%v", failed, err)
		}
		repaired, err := Repair(context.Background(), host, ProcessInvoker{}, root, plan.ID)
		if err != nil || repaired.Status != StatusIntegrated || len(repaired.RepairRounds) != 1 {
			t.Fatalf("Repair did not close the check failure: status=%s rounds=%d err=%v", repaired.Status, len(repaired.RepairRounds), err)
		}
		verifyAndApply(t, root, plan, repaired.Candidate.ID)
		assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v2\n")
	})

	t.Run("review finding drives a targeted repair", func(t *testing.T) {
		root := makeProjectRunFixture(t)
		setupE2EProcess(t, "review-defect-fix")
		configureMixedRoleRuntime(t, root)
		plan := planBothManagers(t, root)
		run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
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
		if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), orders, "work"); calls != 2 {
			t.Fatalf("orders implementer ran %d times, want the original and one repair", calls)
		}
		if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), inventory, "work"); calls != 1 {
			t.Fatalf("unaffected inventory ran %d times, want 1", calls)
		}
		verifyAndApply(t, root, plan, run.Candidate.ID)
		assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v2\n")
	})

	t.Run("integration failure despite green children is caught and repaired", func(t *testing.T) {
		root := makeProjectRunFixture(t)
		addRootIntegrationCheck(t, root)
		setupE2EProcess(t, "integration-stale-summary")
		t.Setenv(e2eIntegrationCheckEnv, "1")
		configureMixedRoleRuntime(t, root)
		plan := planBothManagers(t, root)
		if !planHasCheck(plan, "integration-check") {
			t.Fatalf("plan omitted the root integration check: %+v", plan.Checks)
		}
		host := projectworkHost()
		run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
		if err != nil || run.Status != StatusIntegrated {
			t.Fatalf("integration run did not integrate: status=%s err=%v", run.Status, err)
		}
		for _, child := range []string{orders, inventory} {
			if task := findTask(run.Tasks, child); task == nil || task.ReviewStatus != "pass" {
				t.Fatalf("child %s is not green before integration verification: %+v", child, task)
			}
		}
		failed, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
		if err == nil || failed.Status != StatusFailed || !failedCheckWith(failed.Checks, "integration summary") {
			t.Fatalf("integration check did not catch the inconsistent aggregate: report=%+v err=%v", failed, err)
		}
		repaired, err := Repair(context.Background(), host, ProcessInvoker{}, root, plan.ID)
		if err != nil || repaired.Status != StatusIntegrated {
			t.Fatalf("Repair did not close the integration failure: status=%s err=%v", repaired.Status, err)
		}
		if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), owner, "integrate"); calls < 2 {
			t.Fatalf("integration owner ran %d times, want a repair integration", calls)
		}
		verifyAndApply(t, root, plan, repaired.Candidate.ID)
		assertFileContents(t, root, "src/project-integration.txt", "orders implementation v2\ninventory implementation v2\n")
	})
}

// failedCheckWith finds a failed declared check by its diagnostic. The fixture
// checks inspect the whole candidate, so the diagnostic names the defect.
func failedCheckWith(checks []CheckResult, diagnostic string) bool {
	for _, check := range checks {
		if check.Outcome != "passed" && strings.Contains(check.Stderr, diagnostic) {
			return true
		}
	}
	return false
}

func planHasCheck(plan PlanRecord, name string) bool {
	for _, check := range plan.Checks {
		if strings.Contains(check.ID, name) {
			return true
		}
	}
	return false
}

// addRootIntegrationCheck gives the root Manager an integration artifact with
// its own declared check, which only the aggregate candidate can satisfy.
func addRootIntegrationCheck(t *testing.T, root string) {
	t.Helper()
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "  - .markitect/model/manager.yaml\n",
		"  - .markitect/model/manager.yaml\n  - .markitect/model/integration-statement.yaml\n  - .markitect/model/integration-artifact.yaml\n  - .markitect/model/integration-check.yaml\n", 1)
	if updated == string(manifest) {
		t.Fatal("could not add the root integration model files")
	}
	writeE2E(t, root, projectwork.ManifestPath, updated)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, ".markitect/model/integration-statement.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Statement\nmetadata:\n  name: integration-summary\n  namespace: \"\"\npurpose: The integration summary lists the integrated child artifacts.\nspec:\n  category: concept\n  description: The integration summary lists the integrated child artifacts.\n")
	writeE2E(t, root, ".markitect/model/integration-artifact.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Artifact\nmetadata:\n  name: integration-summary-file\n  namespace: \"\"\npurpose: Summarizes the integrated child artifacts.\nspec:\n  role: implementation\n  realizes:\n    - apiVersion: "+projectmodel.APIVersion+"\n      kind: Statement\n      namespace: \"\"\n      name: integration-summary\n  paths: [src/project-integration.txt]\n  checks:\n    - apiVersion: "+projectmodel.APIVersion+"\n      kind: Check\n      namespace: \"\"\n      name: integration-check\n  required: true\n")
	writeE2E(t, root, ".markitect/model/integration-check.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Check\nmetadata:\n  name: integration-check\n  namespace: \"\"\npurpose: Check that the integration summary matches the integrated children.\nspec:\n  command: ["+filepath.Base(executable)+", -test.run=^TestProjectRunIntegrationCheckProcess$ ]\n  uses:\n    - apiVersion: "+projectmodel.APIVersion+"\n      kind: Statement\n      namespace: \"\"\n      name: integration-summary\n  limitation: This fixture check compares candidate bytes only.\n")
	writeE2E(t, root, "src/project-integration.txt", "pending integration\n")
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "-m", "add root integration check")
}

// nativeRootInvoker answers native Manager turns in-process, as a completed
// App Server turn without edits that delegates to every direct child, and runs
// process Managers through the fixture executor.
type nativeRootInvoker struct{ process ProcessInvoker }

func (i nativeRootInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	if config.Transport != TransportCodexAppServer {
		return i.process.Run(ctx, config, request, options)
	}
	var task struct {
		Phase          string   `json:"phase"`
		DirectChildren []string `json:"directChildren"`
	}
	if err := json.Unmarshal(request.Context, &task); err != nil {
		return agentexec.RunResult{}, err
	}
	response := TaskResponse{Status: "complete", Summary: "native root " + task.Phase, Integrated: task.Phase == "integrate", Delegations: []Delegation{},
		ReworkRequests: []ReworkRequest{}, Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	if task.Phase == "work" {
		for _, child := range task.DirectChildren {
			response.Delegations = append(response.Delegations, Delegation{ManagerID: child, Goal: "Implement the owned source artifact."})
		}
	}
	reportJSON, err := json.Marshal(response)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	fingerprint, err := i.Fingerprint(config)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	tokens := int64(1)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: &tokens, OutputTokens: &tokens}
	lifecycle := &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "native-root-session", TurnID: "native-root-" + task.Phase, State: "completed", Accounting: "partial",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: invocation.RunID, Role: agentexec.RoleExecutor, State: "completed"}}}
	return agentexec.RunResult{
		Response: agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce, Role: agentexec.RoleExecutor,
			InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{},
			VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}, ReportJSON: reportJSON, Usage: usage},
		Receipt: agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest, ConfigDigest: fingerprint,
			ProviderVersion: config.ProviderVersion, Outcome: agentexec.OutcomeProposed, Usage: usage, Lifecycle: lifecycle},
	}, nil
}

func (nativeRootInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return NewTransportInvoker(codexappserver.Options{}).Fingerprint(config)
}

// A run may mix a native Manager with bring-your-own process Managers. When the
// native turn comes first it creates the run's private log directory, and the
// later process turns must still accept it as their owner-only log directory.
func TestNativeManagerBeforeProcessManagersSharesThePrivateLogDirectory(t *testing.T) {
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	setupE2EProcess(t, "normal")
	rootID := e2eManagerID("", "project-owner")
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		base := runtime.Agents[rootID]
		native := workspaceBridgeAgent(t, root)
		native.Command, native.Model, native.ProviderVersion, native.Timeout = base.Command, base.Model, codexappserver.SupportedProviderVersion, base.Timeout
		native.MaxStdoutBytes, native.MaxStderrBytes, native.Pricing = base.MaxStdoutBytes, base.MaxStderrBytes, base.Pricing
		native.AppServer = &AppServerSettings{ReasoningEffort: "medium", MaxEventBytes: 1 << 20}
		runtime.Agents[rootID] = native
	})
	host := projectworkHost()
	workspaces, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
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
