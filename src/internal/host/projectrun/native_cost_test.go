package projectrun

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
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

func nativeFullVerifyCostFixture(t *testing.T) (string, *projectwork.Project, Runtime, Host, *fullVerifyNativeWorkspaceInvoker) {
	t.Helper()
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	revision := identityHead(t, root)
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	instructionPath := filepath.Join(root, "README.md")
	instructionBytes, err := os.ReadFile(instructionPath)
	if err != nil {
		t.Fatal(err)
	}
	baseAgent := Agent{Transport: TransportCodexAppServer, WorkspaceMode: "git", InstructionPaths: []string{"README.md"},
		RuntimeFiles: []agentexec.RuntimeFile{{Path: instructionPath, Mode: "0644", Digest: "sha256:" + digestBytes(instructionBytes)}}}
	baseAgent.Command = filepath.Join(t.TempDir(), "codex-app-server")
	baseAgent.Model = "gpt-6-luna"
	baseAgent.ProviderVersion = "codex-cli 0.162.0"
	baseAgent.AppServer = &AppServerSettings{ReasoningEffort: "high", MaxEventBytes: 1 << 20}
	baseAgent.Timeout = Duration(time.Hour)
	baseAgent.MaxStdoutBytes, baseAgent.MaxStderrBytes = 1<<20, 1<<20
	baseAgent.Pricing = Pricing{InputMicrosPerMillion: 1_000_000, OutputMicrosPerMillion: 1_000_000}
	runtime, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	reviewers := map[string]Agent{}
	for _, manager := range project.Report.Managers {
		reviewers[manager.ID] = baseAgent
	}
	runtime.Review.Agents = reviewers
	return root, project, runtime, Host{Workspaces: service, Load: projectwork.Load}, &fullVerifyNativeWorkspaceInvoker{root: root}
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
		Provider: TransportCodexAppServer, Accounting: "complete", StartRequests: []agentexec.RoleStartRequest{
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
	cost, known, overflow := estimateCostDetailed(usage, Pricing{InputMicrosPerMillion: math.MaxInt64, OutputMicrosPerMillion: 1})
	if !known || !overflow || cost != math.MaxInt64 {
		t.Fatalf("complete usage overflow = (%d, %t, %t), want (MaxInt64, true, true)", cost, known, overflow)
	}
	exactMaxCost, exactKnown, exactOverflow := estimateCostDetailed(
		&agentexec.Usage{InputTokens: int64Ptr(1_000_000), OutputTokens: int64Ptr(0)},
		Pricing{InputMicrosPerMillion: math.MaxInt64, OutputMicrosPerMillion: 0})
	if !exactKnown || exactOverflow || exactMaxCost != math.MaxInt64 {
		t.Fatalf("representable MaxInt64 estimate = (%d, %t, %t), want (MaxInt64, true, false)", exactMaxCost, exactKnown, exactOverflow)
	}
}

func TestNativeFullVerificationStopsOnKnownEstimateOverflow(t *testing.T) {
	root, project, runtime, host, baseInvoker := nativeFullVerifyCostFixture(t)
	invoker := &fullVerifyOverflowUsageInvoker{fullVerifyNativeWorkspaceInvoker: baseInvoker}
	runtime.Limits.MaxCostMicros = math.MaxInt64
	for id, agent := range runtime.Review.Agents {
		agent.Pricing = Pricing{InputMicrosPerMillion: math.MaxInt64, OutputMicrosPerMillion: 1}
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
