package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

const (
	fullVerifyHelperEnv  = "MARKITECT_FULL_VERIFY_HELPER"
	fullVerifyCheckEnv   = "MARKITECT_FULL_VERIFY_CHECK"
	fullVerifyBadEnv     = "MARKITECT_FULL_VERIFY_BAD_MANAGER"
	fullVerifyNoUsageEnv = "MARKITECT_FULL_VERIFY_NO_USAGE"
	fullVerifyLogEnv     = "MARKITECT_FULL_VERIFY_LOG"
)

// Process fixture for the same stdin/stdout agentexec protocol used by real
// adapters; it returns typed evidence reports without calling a model.
func TestFullVerifyHelperProcess(t *testing.T) {
	if os.Getenv(fullVerifyHelperEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		t.Fatal(err)
	}
	var contextPayload struct {
		Kind    string `json:"kind"`
		Manager struct {
			Manager struct {
				ID string `json:"id"`
			} `json:"manager"`
		} `json:"manager"`
		RequiredSubjects []string              `json:"requiredSubjects"`
		ChildAssessments []fullChildAssessment `json:"childAssessments"`
		CheckResults     []CheckResult         `json:"checkResults"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextPayload); err != nil {
		t.Fatal(err)
	}
	if contextPayload.Kind != "projectrun-full-verify/v1" || contextPayload.Manager.Manager.ID == "" {
		t.Fatalf("unexpected context: %s", invocation.Request.Context)
	}
	if path := os.Getenv(fullVerifyLogEnv); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(file).Encode(map[string]any{"managerId": contextPayload.Manager.Manager.ID, "context": contextPayload})
		_ = file.Close()
	}
	assessments := make([]FullAssessment, 0, len(contextPayload.RequiredSubjects))
	for _, subject := range contextPayload.RequiredSubjects {
		assessments = append(assessments, FullAssessment{Subject: subject, Outcome: "pass", Detail: "bound project snapshot inspected"})
	}
	status := "pass"
	if bad := os.Getenv(fullVerifyBadEnv); bad != "" && strings.Contains(contextPayload.Manager.Manager.ID, bad) {
		status = "incomplete"
		assessments = nil
	}
	report, err := json.Marshal(fullAuditResponse{Status: status, Summary: "completed bounded Manager audit", Assessments: assessments, Findings: []string{}, Counterexamples: []FullCounterexample{}})
	if err != nil {
		t.Fatal(err)
	}
	var usage *agentexec.Usage
	if noUsage := os.Getenv(fullVerifyNoUsageEnv); noUsage == "" || !strings.Contains(contextPayload.Manager.Manager.ID, noUsage) {
		usage = &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	}
	response, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: report, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(response, '\n'))
	os.Exit(0)
}

func TestFullVerifyCheckProcess(t *testing.T) {
	if os.Getenv(fullVerifyCheckEnv) != "1" {
		return
	}
	for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("snapshot check input %s missing: %v", path, err)
		}
	}
}

func TestFullVerifyAuditsAllManagersAndParentsReceiveIntegrationScope(t *testing.T) {
	root := makeFullVerifyFixture(t)
	loaded, loadErr := projectwork.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if loaded.Coverage != nil && !loaded.Coverage.Conforming {
		t.Fatalf("fixture coverage not conforming: %+v", loaded.Coverage.Findings)
	}
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	before := gitE2E(t, root, "status", "--porcelain")
	host := Host{Load: projectwork.Load}
	got, err := FullVerify(context.Background(), host, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err != nil {
		t.Fatalf("full verify failed: %v report=%+v", err, got)
	}
	if got.Status != "passed" || len(got.Managers) != 3 || len(got.Checks) != 2 {
		t.Fatalf("unexpected full verification result: %+v", got)
	}
	if got.Starts != 5 {
		t.Fatalf("starts = %d, want three Manager audits and two checks", got.Starts)
	}
	if after := gitE2E(t, root, "status", "--porcelain"); after != before {
		t.Fatalf("read-only verification changed working tree: before %q after %q", before, after)
	}
	data, err := os.ReadFile(os.Getenv(fullVerifyLogEnv))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), `"context":`) != 3 {
		t.Fatalf("logged Manager calls = %d, want 3", strings.Count(string(data), `"context":`))
	}
	if !strings.Contains(string(data), "integration:manager:") || !strings.Contains(string(data), "integration:artifact:") {
		t.Fatalf("root Manager did not receive direct-child integration obligations: %s", data)
	}
	if !strings.Contains(string(data), `"childAssessments":[{"managerId"`) || !strings.Contains(string(data), `"checkResults":[{"id"`) {
		t.Fatalf("root Manager did not receive direct-child assessment and check evidence: %s", data)
	}
	written, err := FullVerify(context.Background(), host, ProcessInvoker{}, root, FullVerifyRequest{Revision: head, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := os.ReadFile(filepath.FromSlash(written.PersistedPath))
	if err != nil {
		t.Fatalf("persisted report missing: %v", err)
	}
	var stored FullVerifyReport
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Status != "passed" || len(stored.Managers) != 3 || stored.Managers[0].Receipt == nil || len(stored.Checks) != 2 {
		t.Fatalf("persisted report omitted receipt or check evidence: %+v", stored)
	}
}

func TestFullVerifyMissingManagerAssessmentCannotPassAndStaleSnapshotRejected(t *testing.T) {
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	before := gitE2E(t, root, "status", "--porcelain")
	host := Host{Load: projectwork.Load}
	t.Setenv(fullVerifyBadEnv, "orders")
	report, err := FullVerify(context.Background(), host, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status == "passed" {
		t.Fatalf("missing Manager evidence passed: report=%+v err=%v", report, err)
	}
	if after := gitE2E(t, root, "status", "--porcelain"); after != before {
		t.Fatalf("failed read-only audit changed project tree: before=%q after=%q", before, after)
	}
	t.Setenv(fullVerifyBadEnv, "")
	project, err := projectwork.Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FullVerifyProject(context.Background(), host, ProcessInvoker{}, root, project, runtime, FullVerifyBinding{ExpectedSnapshot: "stale"}); err != ErrStale {
		t.Fatalf("stale snapshot error = %v, want ErrStale", err)
	}
}

func TestFullVerifyProjectAuditsComposedCandidateBytesWithoutReopeningBaseRevision(t *testing.T) {
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}
	base, err := host.Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	baseDigest := base.Snapshot.Digest()
	base.Snapshot.Files["src/orders/implementation.txt"] = []byte("uncommitted composed candidate bytes\n")
	candidate, err := host.FromSnapshot(root, base.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Snapshot.Digest() == baseDigest {
		t.Fatal("candidate composition did not change snapshot binding")
	}
	report, err := FullVerifyProject(context.Background(), host, ProcessInvoker{}, root, candidate, runtime, FullVerifyBinding{ExpectedSnapshot: candidate.Snapshot.Digest()})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.SnapshotDigest != candidate.Snapshot.Digest() {
		t.Fatalf("candidate verification did not bind to composed bytes: %+v", report)
	}
	assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
}

type fullVerifyNativeWorkspaceInvoker struct {
	root          string
	called        bool
	workspace     projectworkspace.Handle
	roleTimeout   time.Duration
	requestDigest string
}

func (i *fullVerifyNativeWorkspaceInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.called = true
	i.roleTimeout = config.Timeout
	if options.Workspace == nil {
		return agentexec.RunResult{}, errors.New("native assessment received no owned workspace")
	}
	i.workspace = *options.Workspace
	if sameWorkspacePath(i.workspace.CWD, i.root) {
		return agentexec.RunResult{}, errors.New("native assessment received the adopting checkout as CWD")
	}
	journalBytes, err := os.ReadFile(filepath.Join(i.root, ".markitect", "runs", "private", "workspaces", i.workspace.ID+".json"))
	if err != nil {
		return agentexec.RunResult{}, err
	}
	var journal workspaceJournal
	if err := json.Unmarshal(journalBytes, &journal); err != nil {
		return agentexec.RunResult{}, err
	}
	if len(journal.Request.AllowedPaths) != 0 {
		return agentexec.RunResult{}, errors.New("read-only assessment workspace has writable paths")
	}
	var payload struct {
		RequiredSubjects []string `json:"requiredSubjects"`
	}
	if err := json.Unmarshal(request.Context, &payload); err != nil {
		return agentexec.RunResult{}, err
	}
	assessments := make([]FullAssessment, 0, len(payload.RequiredSubjects))
	for _, subject := range payload.RequiredSubjects {
		assessments = append(assessments, FullAssessment{Subject: subject, Outcome: "pass", Detail: "fixed audit subject inspected"})
	}
	report, err := json.Marshal(fullAuditResponse{Status: "pass", Summary: "native read-only workspace inspected", Assessments: assessments, Findings: []string{}, Counterexamples: []FullCounterexample{}})
	if err != nil {
		return agentexec.RunResult{}, err
	}
	inputDigest, err := digest(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	i.requestDigest = inputDigest
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: "full-verify-native-run", Nonce: "full-verify-native-nonce",
		Role: request.Role, InputDigest: inputDigest, Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{},
		EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, ReportJSON: report, Uncertainty: []string{}, Usage: usage}
	receipt := terminalWorkspaceReceipt("full-verify-native-run")
	receipt.InputDigest = inputDigest
	receipt.Outcome = agentexec.OutcomeProposed
	receipt.Usage = usage
	return agentexec.RunResult{Response: response, Receipt: receipt}, nil
}

func (*fullVerifyNativeWorkspaceInvoker) Fingerprint(agentexec.Config) (string, error) {
	return "native-fixture-fingerprint", nil
}

func TestFullVerifyNativeAssessmentUsesReadOnlyOwnedWorkspaceWithoutProvider(t *testing.T) {
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	revision := identityHead(t, root)
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	managerID := project.Report.Managers[0].ID
	assessmentAgent := workspaceBridgeAgent(t, root)
	assessmentAgent.Command = filepath.Join(t.TempDir(), "codex-app-server")
	assessmentAgent.Model = "gpt-6-luna"
	assessmentAgent.ProviderVersion = "codex-cli 0.162.0"
	assessmentAgent.AppServer = &AppServerSettings{ReasoningEffort: "high", MaxEventBytes: 1 << 20}
	assessmentAgent.Timeout = Duration(time.Hour)
	assessmentAgent.MaxStdoutBytes = 1 << 20
	assessmentAgent.MaxStderrBytes = 1 << 20
	assessmentAgent.Pricing = Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}
	runtime := Runtime{Agents: map[string]Agent{managerID: assessmentAgent}, Review: &ReviewConfig{Agents: map[string]Agent{managerID: assessmentAgent}},
		Limits: Limits{MaxDepth: 1, MaxStarts: 256, MaxParallel: 2, MaxDuration: Duration(4 * time.Hour), MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}}
	invoker := &fullVerifyNativeWorkspaceInvoker{root: root}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	row, err := fullAuditManager(ctx, Host{Workspaces: service, Load: projectwork.Load}, invoker, root, project, runtime,
		managerID, StrictnessProfile{}, BriefingContext{}, nil, nil, false)
	if err != nil {
		t.Fatalf("native Manager audit failed: row=%+v err=%v", row, err)
	}
	if !invoker.called || invoker.roleTimeout != time.Hour || invoker.requestDigest != row.InputDigest {
		t.Fatalf("native audit did not preserve the role configuration and request binding: called=%t timeout=%s digest=%q row=%+v", invoker.called, invoker.roleTimeout, invoker.requestDigest, row)
	}
	if row.Status != "passed" || row.Receipt == nil || row.Receipt.RunID != "full-verify-native-run" || row.Receipt.InputDigest != row.InputDigest {
		t.Fatalf("native audit report did not preserve terminal receipt and validated status: %+v", row)
	}
	if _, err := os.Stat(invoker.workspace.CWD); !os.IsNotExist(err) {
		t.Fatalf("terminal read-only workspace was not closed: %v", err)
	}
	journal := readWorkspaceJournal(t, root, invoker.workspace.ID)
	if journal.State != "closed" || journal.Delta == nil || len(journal.Delta.Changes) != 0 || journal.Receipt.RunID != "full-verify-native-run" {
		t.Fatalf("read-only workspace journal did not retain empty harvested evidence and receipt: %+v", journal)
	}
}

func TestFullVerifyStopsWhenUsageCannotBoundCumulativeCost(t *testing.T) {
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load}
	t.Setenv(fullVerifyNoUsageEnv, "inventory")
	report, err := FullVerify(context.Background(), host, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "incomplete" {
		t.Fatalf("missing usage did not leave full verification incomplete: report=%+v err=%v", report, err)
	}
	if report.Starts != 3 {
		t.Fatalf("starts=%d; want two checks and exactly one Manager before stopping", report.Starts)
	}
	if report.Managers[1].Status != "incomplete" || report.Managers[2].Status != "incomplete" {
		t.Fatalf("later Managers ran without a bounded cost basis: %+v", report.Managers)
	}
}

func TestFullVerifyRecordsKnownCostBeyondLimitForSuccessfulAndFailedCalls(t *testing.T) {
	for _, failReview := range []bool{false, true} {
		name := "successful review"
		if failReview {
			name = "failed review"
		}
		t.Run(name, func(t *testing.T) {
			root := makeFullVerifyFixture(t)
			setupFullVerifyProcesses(t)
			configureFullVerifyRuntime(t, root)
			updateE2ERuntime(t, root, func(runtime *Runtime) {
				runtime.Limits.MaxCostMicros = 1
				for id, reviewer := range runtime.Review.Agents {
					reviewer.Pricing.InputMicrosPerMillion = 1_000_000
					reviewer.Pricing.OutputMicrosPerMillion = 1_000_000
					runtime.Review.Agents[id] = reviewer
				}
			})
			if failReview {
				t.Setenv(fullVerifyBadEnv, "inventory")
			}
			head := gitE2E(t, root, "rev-parse", "HEAD")
			report, err := FullVerify(context.Background(), Host{Load: projectwork.Load}, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
			if err == nil || report.Status == "passed" {
				t.Fatalf("over-budget Manager call passed: report=%+v err=%v", report, err)
			}
			if report.CostMicros != 15 || report.Starts != 3 || report.Managers[0].CostMicros != 15 {
				t.Fatalf("known over-budget cost was not retained: total=%d starts=%d first=%+v", report.CostMicros, report.Starts, report.Managers[0])
			}
		})
	}
}

func TestFullVerifyKnownCostOverflowSaturatesExplicitly(t *testing.T) {
	total := int64(math.MaxInt64 - 2)
	if !addFullKnownCost(&total, 3) || total != math.MaxInt64 {
		t.Fatalf("known cost overflow = (%d, false), want saturated MaxInt64 and overflow=true", total)
	}
}

func TestFullVerifyRejectsStrictnessForUnknownManagerBeforeChecks(t *testing.T) {
	root := makeFullVerifyFixture(t)
	configureFullVerifyRuntime(t, root)
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		runtime.Strictness = &StrictnessConfig{Managers: map[string]StrictnessProfile{"not-in-model": {Evidence: []string{"typed evidence"}}}}
	})
	head := gitE2E(t, root, "rev-parse", "HEAD")
	report, err := FullVerify(context.Background(), Host{Load: projectwork.Load}, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "failed" || !strings.Contains(report.Error, "Manager absent from the accepted model") {
		t.Fatalf("unknown strictness Manager was not rejected: report=%+v err=%v", report, err)
	}
	if report.Starts != 0 {
		t.Fatalf("strictness validation happened after checks or provider calls: starts=%d", report.Starts)
	}
}

func TestFullVerifyCumulativeStartLimitLeavesUnstartedManagersIncomplete(t *testing.T) {
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	updateE2ERuntime(t, root, func(runtime *Runtime) { runtime.Limits.MaxStarts = 4 })
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load}
	report, err := FullVerify(context.Background(), host, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "incomplete" {
		t.Fatalf("start-budget exhaustion passed: report=%+v err=%v", report, err)
	}
	if report.Starts != 4 || report.Managers[0].Status != "passed" || report.Managers[1].Status != "passed" || report.Managers[2].Status != "incomplete" {
		t.Fatalf("cumulative start budget was not shared across checks and Managers: %+v", report)
	}
}

func TestFullVerifyStrictnessRequiresTypedEvidenceAndGroundedCounterexamples(t *testing.T) {
	required := []string{"statement:orders", "evidence:negative stock case"}
	base := fullAuditResponse{Status: "pass", Summary: "review complete", Findings: []string{},
		Assessments:     []FullAssessment{{Subject: "statement:orders", Outcome: "pass", Detail: "checked"}, {Subject: "evidence:negative stock case", Outcome: "pass", Detail: "checked"}},
		Counterexamples: []FullCounterexample{{Expected: "reject negative stock", Observed: "negative stock can be submitted", EvidenceRefs: []string{"statement:orders"}}}}
	if err := validateFullAssessments(base, required, 1, required, []string{"src/orders.go"}); err != nil {
		t.Fatalf("valid strict response rejected: %v", err)
	}
	missingEvidence := base
	missingEvidence.Assessments = base.Assessments[:1]
	if err := validateFullAssessments(missingEvidence, required, 1, required, []string{"src/orders.go"}); err == nil {
		t.Fatal("missing typed evidence subject was accepted")
	}
	missingCounterexample := base
	missingCounterexample.Counterexamples = nil
	if err := validateFullAssessments(missingCounterexample, required, 1, required, []string{"src/orders.go"}); err == nil {
		t.Fatal("missing required counterexample was accepted")
	}
	unbound := base
	unbound.Counterexamples = []FullCounterexample{{Expected: "reject negative stock", Observed: "it passed", EvidenceRefs: []string{"unrelated:file"}}}
	if err := validateFullAssessments(unbound, required, 1, required, []string{"src/orders.go"}); err == nil {
		t.Fatal("counterexample with out-of-scope evidence was accepted")
	}
}

func TestFullVerifyRejectsUnknownRepositoryCoverageBeforeManagerCalls(t *testing.T) {
	root := makeFullVerifyFixture(t)
	writeE2E(t, root, "outside/unclassified.txt", "not represented by the project model\n")
	gitE2E(t, root, "add", "outside/unclassified.txt")
	gitE2E(t, root, "commit", "-m", "add unclassified repository path")
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load}
	report, err := FullVerify(context.Background(), host, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "failed" || !strings.Contains(report.Error, "coverage census") {
		t.Fatalf("unknown repository path did not fail the full census gate: report=%+v err=%v", report, err)
	}
	if report.Starts != 0 || len(report.Managers) != 0 {
		t.Fatalf("provider or check calls started before coverage was conforming: %+v", report)
	}
	if _, statErr := os.Stat(os.Getenv(fullVerifyLogEnv)); !os.IsNotExist(statErr) {
		if data, readErr := os.ReadFile(os.Getenv(fullVerifyLogEnv)); readErr == nil && len(data) != 0 {
			t.Fatalf("Manager called before coverage census passed: %s", data)
		}
	}
}

func TestFullVerifyRejectsStaleGeneratedDocumentBeforeManagerCalls(t *testing.T) {
	root := makeFullVerifyFixture(t)
	writeE2E(t, root, "README.md", "stale hand-edited document\n")
	gitE2E(t, root, "add", "README.md")
	gitE2E(t, root, "commit", "-m", "make generated document stale")
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	report, err := FullVerify(context.Background(), Host{Load: projectwork.Load}, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "failed" || !strings.Contains(report.Error, "generated documentation is missing or stale") {
		t.Fatalf("stale generated document did not fail the pre-provider gate: report=%+v err=%v", report, err)
	}
	if report.Starts != 0 || len(report.Managers) != 0 {
		t.Fatalf("provider or check calls started before document validation: %+v", report)
	}
}

func makeFullVerifyFixture(t *testing.T) string {
	t.Helper()
	root := makeProjectRunFixture(t)
	manifest := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "name: Process fixture\n", "name: Process fixture\ndocumentPath: README.md\ncoverageMode: full\n", 1)
	if updated == string(data) {
		t.Fatal("could not set full coverage mode in fixture manifest")
	}
	writeE2E(t, root, projectwork.ManifestPath, updated)
	writeE2E(t, root, projectcoverage.IgnorePath, "apiVersion: "+projectcoverage.IgnoreAPIVersion+"\nkind: RepositoryIgnore\nentries: []\n")
	gitE2E(t, root, "add", projectwork.ManifestPath)
	gitE2E(t, root, "add", projectcoverage.IgnorePath)
	gitE2E(t, root, "commit", "-m", "enable full project coverage")
	project, err := projectwork.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := projectwork.Document(project, false)
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, projectwork.DocumentPath(project.Config), document)
	gitE2E(t, root, "add", projectwork.DocumentPath(project.Config))
	gitE2E(t, root, "commit", "-m", "generate full project document")
	return root
}

func setupFullVerifyProcesses(t *testing.T) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(fullVerifyHelperEnv, "1")
	t.Setenv(fullVerifyLogEnv, filepath.Join(t.TempDir(), "full-verify.jsonl"))
	// Test subprocesses receive this path through the deliberately narrow allowlist.
	t.Setenv(fullVerifyCheckEnv, "1")
}

func configureFullVerifyRuntime(t *testing.T, root string) {
	t.Helper()
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"-test.run=^TestFullVerifyHelperProcess$"}
		reviewers := make(map[string]Agent, len(runtime.Agents))
		for id, agent := range runtime.Agents {
			agent.Command, agent.Args = executable, args
			agent.Environment = append(agent.Environment, fullVerifyHelperEnv, fullVerifyBadEnv, fullVerifyLogEnv, fullVerifyNoUsageEnv)
			reviewers[id] = agent
		}
		runtime.Review = &ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
		for id, agent := range runtime.Agents {
			agent.Args = []string{"-test.run=^TestFullVerifyCheckProcess$"}
			agent.Environment = append(agent.Environment, fullVerifyCheckEnv)
			runtime.Agents[id] = agent
		}
	})
	// Ensure the helper executable remains available to the pinned check resolver.
	t.Setenv("PATHEXT", ".EXE;.COM;.BAT;.CMD")
}

func int64Ptr(value int64) *int64 { return &value }
