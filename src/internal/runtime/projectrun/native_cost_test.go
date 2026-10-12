package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectworkspace"
)

type fullVerifyUnknownUsageInvoker struct {
	*fullVerifyNativeWorkspaceInvoker
}

func (i *fullVerifyUnknownUsageInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	result, err := i.fullVerifyNativeWorkspaceInvoker.Run(ctx, config, request, options)
	result.Receipt.Usage = nil
	result.Response.Usage = nil
	return result, err
}

type fullVerifyPartialLifecycleInvoker struct {
	*fullVerifyNativeWorkspaceInvoker
}

type fullVerifyOverflowUsageInvoker struct {
	*fullVerifyNativeWorkspaceInvoker
}

func (i *fullVerifyOverflowUsageInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	result, err := i.fullVerifyNativeWorkspaceInvoker.Run(ctx, config, request, options)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(2_000_000), OutputTokens: int64Ptr(0)}
	result.Receipt.Usage, result.Response.Usage = usage, usage
	return result, err
}

func (i *fullVerifyPartialLifecycleInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	result, err := i.fullVerifyNativeWorkspaceInvoker.Run(ctx, config, request, options)
	if result.Receipt.Usage != nil {
		result.Receipt.Usage.InputTokens = int64Ptr(10)
		result.Receipt.Usage.OutputTokens = int64Ptr(5)
		result.Response.Usage = result.Receipt.Usage
	}
	result.Receipt.Lifecycle.Accounting = "partial"
	result.Receipt.Lifecycle.StartRequests = append(result.Receipt.Lifecycle.StartRequests,
		agentexec.RoleStartRequest{RequestID: result.Receipt.RunID + "-helper", Role: "helper", State: "completed"})
	return result, err
}

func nativeFullVerifyCostFixture(t *testing.T) (string, *projectvalue.Project, projectconfig.Runtime, Host, *fullVerifyNativeWorkspaceInvoker) {
	t.Helper()
	// These Runtime accounting cases supply the finite audit inputs directly;
	// repository census and document-rendering behavior is owned elsewhere.
	root := makeProjectRunFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	revision := identityHead(t, root)
	runtimeBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projectconfig.RuntimePath)))
	if err != nil {
		t.Fatal(err)
	}
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	const instruction = "project-run process fixture\n"
	instructionPath := filepath.Join(root, "README.md")
	baseAgent := projectconfig.Agent{Transport: projectconfig.TransportCodexAppServer, WorkspaceMode: "git", InstructionPaths: []string{"README.md"},
		RuntimeFiles: []agentexec.RuntimeFile{{Path: instructionPath, Mode: "0644", Digest: "sha256:" + digestBytes([]byte(instruction))}}}
	baseAgent.Command = filepath.Join(t.TempDir(), "codex-app-server")
	baseAgent.Model = "gpt-6-luna"
	baseAgent.ProviderVersion = "codex-cli 0.162.0"
	baseAgent.AppServer = &projectconfig.AppServerSettings{ReasoningEffort: "high", MaxEventBytes: 1 << 20}
	baseAgent.Timeout = projectconfig.Duration(time.Hour)
	baseAgent.MaxStdoutBytes, baseAgent.MaxStderrBytes = 1<<20, 1<<20
	baseAgent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 1_000_000, OutputMicrosPerMillion: 1_000_000}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	rootManagerID := e2eManagerID("", "project-owner")
	ordersManagerID := e2eManagerID("orders", "orders")
	inventoryManagerID := e2eManagerID("inventory", "inventory")
	runtime.Review.Agents = map[string]projectconfig.Agent{
		rootManagerID: baseAgent, ordersManagerID: baseAgent, inventoryManagerID: baseAgent,
	}
	checkExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	checkArgs := append([]string{filepath.Base(checkExecutable)}, runtimeFixtureArgs(t, []string{"-test.run=^TestFullVerifyCheckProcess$"})...)
	files := map[string][]byte{
		"README.md":                        []byte(instruction),
		projectconfig.RuntimePath:          runtimeBytes,
		"src/orders/implementation.txt":    []byte("orders implementation v1\n"),
		"src/inventory/implementation.txt": []byte("inventory implementation v1\n"),
	}
	modes := map[string]string{}
	for path := range files {
		modes[path] = "100644"
	}
	ordersStatement := nativeCostIdentity("Statement", "orders", "orders-work")
	inventoryStatement := nativeCostIdentity("Statement", "inventory", "inventory-work")
	ordersArtifact := nativeCostIdentity("Artifact", "orders", "orders-code")
	inventoryArtifact := nativeCostIdentity("Artifact", "inventory", "inventory-code")
	ordersCheck := nativeCostIdentity("Check", "orders", "orders-check")
	inventoryCheck := nativeCostIdentity("Check", "inventory", "inventory-check")
	report := projectmodel.Report{
		APIVersion: projectmodel.APIVersion, ModelDigest: "runtime-cost-fixture-model", Digest: "runtime-cost-fixture-report",
		Status: "succeeded",
		Managers: []projectmodel.Manager{
			{ID: rootManagerID, Name: "project-owner", Purpose: "Owns integration and project-wide delivery.", Owns: []string{"."}},
			{ID: ordersManagerID, Name: "orders", Namespace: "orders", Purpose: "Owns the orders implementation.", Parent: rootManagerID, Owns: []string{"src/orders/"}},
			{ID: inventoryManagerID, Name: "inventory", Namespace: "inventory", Purpose: "Owns the inventory implementation.", Parent: rootManagerID, Owns: []string{"src/inventory/"}},
		},
		Statements: []projectmodel.Statement{
			{ID: ordersStatement, Name: "orders-work", Namespace: "orders", Owner: ordersManagerID, Category: "concept", Description: "Implement the orders artifact."},
			{ID: inventoryStatement, Name: "inventory-work", Namespace: "inventory", Owner: inventoryManagerID, Category: "concept", Description: "Implement the inventory artifact."},
		},
		Artifacts: []projectmodel.Artifact{
			{ID: ordersArtifact, Name: "orders-code", Owner: ordersManagerID, Role: "implementation", Realizes: []string{ordersStatement}, Paths: []string{"src/orders/implementation.txt"}, Checks: []string{ordersCheck}, Required: true},
			{ID: inventoryArtifact, Name: "inventory-code", Owner: inventoryManagerID, Role: "implementation", Realizes: []string{inventoryStatement}, Paths: []string{"src/inventory/implementation.txt"}, Checks: []string{inventoryCheck}, Required: true},
		},
		Checks: []projectmodel.Check{
			{ID: ordersCheck, Name: "orders-check", Owner: ordersManagerID, Command: checkArgs, Uses: []string{ordersStatement}, Limitation: "Validates the explicit Runtime fixture bytes."},
			{ID: inventoryCheck, Name: "inventory-check", Owner: inventoryManagerID, Command: checkArgs, Uses: []string{inventoryStatement}, Limitation: "Validates the explicit Runtime fixture bytes."},
		},
		Files: []projectmodel.FileEntry{
			{Path: "src/orders/implementation.txt", Owner: ordersManagerID, Class: "source", Statements: []string{ordersStatement}, Artifacts: []string{ordersArtifact}, Checks: []string{ordersCheck}, Exists: true},
			{Path: "src/inventory/implementation.txt", Owner: inventoryManagerID, Class: "source", Statements: []string{inventoryStatement}, Artifacts: []string{inventoryArtifact}, Checks: []string{inventoryCheck}, Exists: true},
		},
	}
	project := runtimeProjectFixture(revision, files, modes, core.Model{}, report)
	project.Root = root
	project.Digest = "runtime-cost-fixture-project"
	project.Config = projectvalue.Config{DocumentPath: "README.md", CoverageMode: "full", WorkflowMode: projectvalue.WorkflowModeEmpty}
	project.Coverage = &projectcoverage.Report{Accounted: true, Conforming: true, Digest: "runtime-cost-fixture-coverage"}
	host := hostForProjectWithWorkspace(root, project, service)
	host.RenderDocument = func(project *projectvalue.Project) (string, error) {
		if project == nil || project.Root != root || project.Revision != revision || project.Snapshot == nil {
			return "", errors.New("native cost fixture received an unbound document request")
		}
		return string(project.Snapshot.Files["README.md"]), nil
	}
	host.BriefingForManager = func(loadRoot, loadRevision, modelDigest, requestedManager string, _ *projectbriefing.Store) (BriefingContext, error) {
		if loadRoot != root || loadRevision != revision || modelDigest != report.ModelDigest || requestedManager != rootManagerID && requestedManager != ordersManagerID && requestedManager != inventoryManagerID {
			return BriefingContext{}, errors.New("native cost fixture received an unbound briefing request")
		}
		return BriefingContext{Digest: "runtime-cost-fixture-empty-briefing"}, nil
	}
	return root, project, runtime, host, &fullVerifyNativeWorkspaceInvoker{root: root}
}

func nativeCostIdentity(kind, namespace, name string) string {
	data, _ := json.Marshal([]string{projectmodel.APIVersion, kind, namespace, name})
	return string(data)
}

func TestNativeFullVerificationAllowsUnknownUsageWithHonestAccounting(t *testing.T) {
	root, project, runtime, host, baseInvoker := nativeFullVerifyCostFixture(t)
	invoker := &fullVerifyUnknownUsageInvoker{fullVerifyNativeWorkspaceInvoker: baseInvoker}
	report, err := FullVerifyProject(context.Background(), host, invoker, root, project, runtime,
		FullVerifyBinding{ExpectedSnapshot: project.Snapshot.Digest()})
	if err != nil {
		t.Fatalf("native assessment with unavailable token telemetry failed functional verification: report=%+v err=%v", report, err)
	}
	if report.Status != "passed" || report.CostAccounting != CostAccountingUnknown || report.CostMicros != 0 {
		t.Fatalf("unknown telemetry was not retained separately from functional status: status=%q accounting=%q cost=%d err=%q", report.Status, report.CostAccounting, report.CostMicros, report.Error)
	}
	if len(report.Managers) == 0 {
		t.Fatal("full verification returned no Manager assessments")
	}
	for _, manager := range report.Managers {
		if manager.Status != "passed" || manager.CostKnown || manager.CostMicros != 0 || manager.Receipt == nil || manager.Receipt.Usage != nil {
			t.Fatalf("functional assessment or honest unknown-cost state was lost: %+v", manager)
		}
	}
	if !invoker.fullVerifyNativeWorkspaceInvoker.called {
		t.Fatal("native adapter fixture was not invoked")
	}
}

func TestNativeFullVerificationMarksPartialLifecycleCostWithoutBlockingAssessment(t *testing.T) {
	root, project, runtime, host, baseInvoker := nativeFullVerifyCostFixture(t)
	invoker := &fullVerifyPartialLifecycleInvoker{fullVerifyNativeWorkspaceInvoker: baseInvoker}
	report, err := FullVerifyProject(context.Background(), host, invoker, root, project, runtime,
		FullVerifyBinding{ExpectedSnapshot: project.Snapshot.Digest()})
	if err != nil {
		t.Fatalf("native assessment with partial child accounting failed functionally: report=%+v err=%v", report, err)
	}
	wantKnownEstimate := int64(len(report.Managers) * 15)
	if report.Status != "passed" || report.CostAccounting != CostAccountingPartial || report.CostMicros != wantKnownEstimate {
		t.Fatalf("partial lifecycle accounting was not exposed separately from functional status: status=%q accounting=%q cost=%d err=%q", report.Status, report.CostAccounting, report.CostMicros, report.Error)
	}
	for _, manager := range report.Managers {
		if manager.Status != "passed" || !manager.CostKnown || manager.Receipt == nil || manager.Receipt.Lifecycle.Accounting != "partial" {
			t.Fatalf("functional assessment or receipt accounting was lost: %+v", manager)
		}
	}
}

func TestCostAccountingDistinguishesPartialFromUnknown(t *testing.T) {
	if got := costAccounting(nil); got != CostAccountingComplete {
		t.Fatalf("empty accounting = %q, want complete", got)
	}
	if got := costAccounting([]InvocationLog{{}, {CostKnown: true, CostMicros: 0}}); got != CostAccountingPartial {
		t.Fatalf("mixed accounting = %q, want partial", got)
	}
	if got := costAccounting([]InvocationLog{{}, {}}); got != CostAccountingUnknown {
		t.Fatalf("unavailable accounting = %q, want unknown", got)
	}
	report := RunReport{Invocations: []InvocationLog{{CostKnown: true, Receipt: agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{
		Provider: projectconfig.TransportCodexAppServer, Accounting: "complete", StartRequests: []agentexec.RoleStartRequest{
			{RequestID: "root", Role: "executor", State: "completed"},
			{RequestID: "helper-1", Role: "helper", State: "completed"},
		},
	}}}}}
	known, unknown := runCostCounts(report)
	if got := costAccountingFromCounts(known, unknown); got != CostAccountingPartial {
		t.Fatalf("known root plus unpriced native helper accounting = %q, want partial", got)
	}
}

func TestEstimateCostSaturatesKnownArithmeticOverflow(t *testing.T) {
	usage := &agentexec.Usage{InputTokens: int64Ptr(2_000_000), OutputTokens: int64Ptr(0)}
	cost, known, overflow := estimateCostDetailed(usage, projectconfig.Pricing{InputMicrosPerMillion: math.MaxInt64, OutputMicrosPerMillion: 1})
	if !known || !overflow || cost != math.MaxInt64 {
		t.Fatalf("complete usage overflow = (%d, %t, %t), want (MaxInt64, true, true)", cost, known, overflow)
	}
	exactMaxCost, exactKnown, exactOverflow := estimateCostDetailed(
		&agentexec.Usage{InputTokens: int64Ptr(1_000_000), OutputTokens: int64Ptr(0)},
		projectconfig.Pricing{InputMicrosPerMillion: math.MaxInt64, OutputMicrosPerMillion: 0})
	if !exactKnown || exactOverflow || exactMaxCost != math.MaxInt64 {
		t.Fatalf("representable MaxInt64 estimate = (%d, %t, %t), want (MaxInt64, true, false)", exactMaxCost, exactKnown, exactOverflow)
	}
}

func TestNativeFullVerificationStopsOnKnownEstimateOverflow(t *testing.T) {
	root, project, runtime, host, baseInvoker := nativeFullVerifyCostFixture(t)
	invoker := &fullVerifyOverflowUsageInvoker{fullVerifyNativeWorkspaceInvoker: baseInvoker}
	runtime.Limits.MaxCostMicros = math.MaxInt64
	for id, agent := range runtime.Review.Agents {
		agent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: math.MaxInt64, OutputMicrosPerMillion: 1}
		runtime.Review.Agents[id] = agent
	}
	report, err := FullVerifyProject(context.Background(), host, invoker, root, project, runtime,
		FullVerifyBinding{ExpectedSnapshot: project.Snapshot.Digest()})
	if err == nil || report.Status != "incomplete" || report.CostMicros != math.MaxInt64 {
		t.Fatalf("unrepresentable known estimate did not trip cost limit: status=%q cost=%d err=%v", report.Status, report.CostMicros, err)
	}
	if len(report.Managers) < 2 || !report.Managers[0].CostKnown || report.Managers[0].CostMicros != math.MaxInt64 || report.Managers[1].Receipt != nil {
		t.Fatalf("overflow was treated as missing telemetry or later Manager ran: %+v", report.Managers)
	}
}

func TestKnownCostAggregateOverflowExceedsMaxInt64Limit(t *testing.T) {
	logs := []InvocationLog{{CostKnown: true, CostMicros: math.MaxInt64}, {CostKnown: true, CostMicros: 1}}
	if got := totalCost(logs); got != math.MaxInt64 {
		t.Fatalf("saturated aggregate = %d, want MaxInt64", got)
	}
	if !knownCostOverflow(logs) || !costExceedsLimit(logs, math.MaxInt64) {
		t.Fatal("aggregate overflow was allowed through an exact MaxInt64 cost cap")
	}
	if costExceedsLimit([]InvocationLog{{CostKnown: true, CostMicros: math.MaxInt64}}, math.MaxInt64) {
		t.Fatal("representable exact MaxInt64 estimate incorrectly overflowed its cap")
	}
}
