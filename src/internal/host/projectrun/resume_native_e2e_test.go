package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

type nativeResumeFixtureInvoker struct {
	result       agentexec.RunResult
	runCalls     int
	recoverCalls int
	retry        bool
	workspaceIDs []string
}

func (i *nativeResumeFixtureInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.runCalls++
	if !i.retry || options.Workspace == nil {
		return agentexec.RunResult{}, errors.New("Resume must not start a new native turn")
	}
	i.workspaceIDs = append(i.workspaceIDs, options.Workspace.TaskID)
	const retryBytes = "bounded native repair output\n"
	if err := os.WriteFile(filepath.Join(options.Workspace.CWD, "src", "project-owner.txt"), []byte(retryBytes), 0o644); err != nil {
		return agentexec.RunResult{}, err
	}
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	tokensIn, tokensOut := int64(4), int64(2)
	reportJSON, err := json.Marshal(TaskResponse{Status: "complete", Summary: "Repaired after the recovered response failed validation.",
		Delegations: []Delegation{}, ReworkRequests: []ReworkRequest{}, Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}})
	if err != nil {
		return agentexec.RunResult{}, err
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{{Path: "src/project-owner.txt", Mode: "0644", Content: retryBytes}},
		EvidenceRefs:   []string{}, VerifierObservations: []agentexec.Observation{}, ReportJSON: reportJSON, Uncertainty: []string{},
		Usage: &agentexec.Usage{Source: "provider-reported", InputTokens: &tokensIn, OutputTokens: &tokensOut}}
	fingerprint, err := NewTransportInvoker(codexappserver.Options{}).Fingerprint(config)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	lifecycle := &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "repair-session", TurnID: "repair-turn", State: "completed", Accounting: "partial",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: invocation.RunID, Role: agentexec.RoleExecutor, State: "completed"}}}
	receipt := agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest,
		ConfigDigest: fingerprint, ProviderVersion: config.ProviderVersion, Outcome: agentexec.OutcomeProposed, Usage: response.Usage, Lifecycle: lifecycle}
	return agentexec.RunResult{Response: response, Receipt: receipt}, nil
}

func (*nativeResumeFixtureInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return NewTransportInvoker(codexappserver.Options{}).Fingerprint(config)
}

func (i *nativeResumeFixtureInvoker) Recover(_ context.Context, config agentexec.Config, handle codexappserver.RecoveryHandle, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.recoverCalls++
	if config.Transport != TransportCodexAppServer || options.Workspace == nil || options.Workspace.ID != handle.Workspace.ID {
		return agentexec.RunResult{}, errors.New("recovery did not retain the original native workspace")
	}
	i.workspaceIDs = append(i.workspaceIDs, options.Workspace.TaskID)
	return i.result, nil
}

type nativeResumeRunFixture struct {
	root      string
	host      Host
	plan      PlanRecord
	store     *runStore
	dir       string
	managerID string
	phase     string
	handle    projectworkspace.Handle
	invoker   *nativeResumeFixtureInvoker
}

func TestResumeConsumesExactNativeWorkAttemptWithoutReplay(t *testing.T) {
	fixture := seedNativeResumeRun(t)
	resumed, resumeErr := Resume(context.Background(), fixture.host, fixture.invoker, fixture.root, fixture.plan.ID)
	assertNativeResumeResult(t, fixture, resumed, resumeErr, false, 1)
}

func TestResumeRepairsRecoveredInvalidResponseWithOneNewAttempt(t *testing.T) {
	fixture := seedNativeResumeRun(t)
	fixture.invoker.result.Response.ReportJSON = json.RawMessage(`{}`)
	fixture.invoker.retry = true
	resumed, resumeErr := Resume(context.Background(), fixture.host, fixture.invoker, fixture.root, fixture.plan.ID)
	assertNativeResumeResult(t, fixture, resumed, resumeErr, true, 1)
}

func TestResumeRechecksSameNativeTurnAfterUnknownLifecycle(t *testing.T) {
	fixture := seedNativeResumeRun(t)
	lifecycle := *fixture.invoker.result.Receipt.Lifecycle
	lifecycle.State = "inProgress"
	lifecycle.StartRequests = append([]agentexec.RoleStartRequest(nil), lifecycle.StartRequests...)
	lifecycle.StartRequests[0].State = "inProgress"
	fixture.invoker.result.Receipt.Lifecycle = &lifecycle

	blocked, firstErr := Resume(context.Background(), fixture.host, fixture.invoker, fixture.root, fixture.plan.ID)
	if firstErr == nil || blocked.Status != StatusBlocked || fixture.invoker.runCalls != 0 || fixture.invoker.recoverCalls != 1 {
		t.Fatalf("unknown native lifecycle should block without replay: status=%s err=%v run=%d recover=%d", blocked.Status, firstErr, fixture.invoker.runCalls, fixture.invoker.recoverCalls)
	}
	blockedTask := findTask(blocked.Tasks, fixture.managerID)
	if blockedTask == nil || blockedTask.State != "uncertain" || blockedTask.WorkAttempts != 1 || blockedTask.Attempts != 1 {
		t.Fatalf("unknown turn lost its reserved attempt: %+v", blockedTask)
	}

	lifecycle.State = "completed"
	lifecycle.StartRequests[0].State = "completed"
	fixture.invoker.result.Receipt.Lifecycle = &lifecycle
	resumed, secondErr := Resume(context.Background(), fixture.host, fixture.invoker, fixture.root, fixture.plan.ID)
	assertNativeResumeResult(t, fixture, resumed, secondErr, false, 2)
}

func TestResumeConsumesAlreadyReservedIntegrationAtExactStartLimit(t *testing.T) {
	fixture := seedNativeIntegrationRun(t)
	resumed, resumeErr := Resume(context.Background(), fixture.host, fixture.invoker, fixture.root, fixture.plan.ID)
	if resumeErr != nil {
		t.Fatalf("Resume reserved native integration: status=%s err=%v", resumed.Status, resumeErr)
	}
	if fixture.invoker.runCalls != 0 || fixture.invoker.recoverCalls != 1 || resumed.Status != StatusIntegrated || len(resumed.Invocations) != 1 {
		t.Fatalf("integration recovery dispatched/recovered unexpected calls: status=%s calls=%d/%d logs=%d", resumed.Status, fixture.invoker.runCalls, fixture.invoker.recoverCalls, len(resumed.Invocations))
	}
	rootTask := findTask(resumed.Tasks, fixture.managerID)
	if rootTask == nil || rootTask.State != "integrated" || rootTask.WorkAttempts != 1 || rootTask.IntegrationAttempts != 1 || rootTask.Attempts != 2 {
		t.Fatalf("integration recovery changed its reserved attempt: %+v", rootTask)
	}
	child := findTask(resumed.Tasks, e2eManagerID("orders", "orders"))
	if child == nil || child.State != "worked" || child.WorkAttempts != 1 {
		t.Fatalf("integration prerequisite work was not preserved: %+v", child)
	}
	accounting, _ := reportStartAccounting(resumed)
	if accounting.ObservedTotal != 3 {
		t.Fatalf("recovery at the exact MaxStarts boundary changed root accounting: %+v", accounting)
	}
	candidate, err := fixture.store.readCandidate(fixture.dir, rootTask.IntegrationCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(candidate.Files["src/orders/implementation.txt"].Content); got != "orders implementation v2\n" {
		t.Fatalf("integrated candidate did not preserve child bytes: %q", got)
	}
	if len(fixture.invoker.workspaceIDs) != 1 || fixture.invoker.workspaceIDs[0] != nativeTaskAttemptID(*rootTask, "integrate") {
		t.Fatalf("integration recovery used an unexpected attempt identity: %v", fixture.invoker.workspaceIDs)
	}
}

func TestResumeReconstructsReviewReworkFromReviewedCandidate(t *testing.T) {
	fixture := seedNativeReworkResumeRun(t)
	resumed, resumeErr := Resume(context.Background(), fixture.host, fixture.invoker, fixture.root, fixture.plan.ID)
	if resumeErr != nil {
		t.Fatalf("Resume reviewed candidate rework: status=%s err=%v", resumed.Status, resumeErr)
	}
	if fixture.invoker.runCalls != 0 || fixture.invoker.recoverCalls != 1 || resumed.Status != StatusIntegrated {
		t.Fatalf("rework Resume dispatched/recovered unexpected calls: status=%s calls=%d/%d", resumed.Status, fixture.invoker.runCalls, fixture.invoker.recoverCalls)
	}
	task := findTask(resumed.Tasks, fixture.managerID)
	if task == nil || task.WorkAttempts != 2 || task.Attempts != 2 || task.State != "worked" {
		t.Fatalf("recovered rework changed its reserved attempt: %+v", task)
	}
	candidate, err := fixture.store.readCandidate(fixture.dir, task.CandidateID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(candidate.Files["src/project-owner.txt"].Content); got != "recovered native proposal\n" {
		t.Fatalf("recovered rework candidate bytes = %q", got)
	}
}

// nativeDeltaOnlyInvoker follows the documented App Server contract: the
// report carries empty candidateFiles and the Host owns the bytes through the
// harvested workspace delta. A set report replaces the fixture's report, and
// writes land in the native workspace beside the fixture's own edit. Review
// requests reach the configured process reviewer, which records each request
// and passes.
type nativeDeltaOnlyInvoker struct {
	inner   *nativeResumeFixtureInvoker
	report  *TaskResponse
	writes  map[string]string
	reviews []agentexec.Request
}

func (i *nativeDeltaOnlyInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	var discriminator struct {
		Kind string `json:"kind"`
	}
	if json.Unmarshal(request.Context, &discriminator) == nil && discriminator.Kind == "projectrun-review/v1" {
		i.reviews = append(i.reviews, request)
		return (&normalizedReviewInvoker{}).Run(ctx, config, request, options)
	}
	result, err := i.inner.Run(ctx, config, request, options)
	if err != nil {
		return result, err
	}
	for path, content := range i.writes {
		if err := os.WriteFile(filepath.Join(options.Workspace.CWD, filepath.FromSlash(path)), []byte(content), 0o644); err != nil {
			return agentexec.RunResult{}, err
		}
	}
	if i.report != nil {
		reportJSON, err := json.Marshal(*i.report)
		if err != nil {
			return agentexec.RunResult{}, err
		}
		result.Response.ReportJSON = reportJSON
	}
	result.Response.CandidateFiles = []agentexec.CandidateFile{}
	return result, nil
}

func (i *nativeDeltaOnlyInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return i.inner.Fingerprint(config)
}

func (i *nativeDeltaOnlyInvoker) Recover(ctx context.Context, config agentexec.Config, handle codexappserver.RecoveryHandle, options agentexec.RunOptions) (agentexec.RunResult, error) {
	result, err := i.inner.Recover(ctx, config, handle, options)
	result.Response.CandidateFiles = []agentexec.CandidateFile{}
	return result, err
}

func TestMainLoopAppliesNativeDeltaWithEmptyCandidateFiles(t *testing.T) {
	fixture := seedNativeResumeRun(t)
	fixture.invoker.result.Response.ReportJSON = json.RawMessage(`{}`)
	fixture.invoker.retry = true
	resumed, resumeErr := Resume(context.Background(), fixture.host, &nativeDeltaOnlyInvoker{inner: fixture.invoker}, fixture.root, fixture.plan.ID)
	assertNativeResumeResult(t, fixture, resumed, resumeErr, true, 1)
	if task := findTask(resumed.Tasks, fixture.managerID); !containsString(task.WrittenPaths, "src/project-owner.txt") {
		t.Fatalf("native work did not record its harvested path as written: %v", task.WrittenPaths)
	}
}

func TestTargetedReworkAppliesNativeWorkspaceDelta(t *testing.T) {
	fixture := seedNativeReviewedRun(t, "work")
	fixture.invoker.retry = true
	report, base, runtime := loadNativeReworkState(t, fixture)
	starts, spent := 1, int64(0)
	invoker := &nativeDeltaOnlyInvoker{inner: fixture.invoker}
	reworkErr := executeReworkSubtree(context.Background(), fixture.host, invoker, fixture.root, fixture.store, fixture.dir, fixture.plan, runtime, base, &report,
		fixture.managerID, "Correct the project owner artifact.", "Independent review requested a correction.", &starts, &spent)
	if reworkErr != nil {
		t.Fatalf("targeted native rework and its review: %v", reworkErr)
	}
	if fixture.invoker.runCalls != 1 {
		t.Fatalf("targeted rework did not dispatch exactly one native turn: runCalls=%d err=%v", fixture.invoker.runCalls, reworkErr)
	}
	task := findTask(report.Tasks, fixture.managerID)
	if task == nil || task.CandidateID == "" {
		t.Fatalf("targeted rework did not bind a new candidate: task=%+v err=%v", task, reworkErr)
	}
	candidate, err := fixture.store.readCandidate(fixture.dir, task.CandidateID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(candidate.Files["src/project-owner.txt"].Content); got != "bounded native repair output\n" {
		t.Fatalf("native targeted rework lost the harvested workspace bytes: %q (err=%v)", got, reworkErr)
	}
	if !containsString(task.WrittenPaths, "src/project-owner.txt") {
		t.Fatalf("native targeted rework did not record its harvested path as written: %v", task.WrittenPaths)
	}
	assertNativeReviewPassed(t, invoker, report, *task, "work", task.CandidateID, map[string]string{"src/project-owner.txt": "bounded native repair output\n"})
}

func TestTargetedReworkRejectsNativeNoOpWithWorkspaceChanges(t *testing.T) {
	fixture := seedNativeReviewedRun(t, "work")
	fixture.invoker.retry = true
	report, base, runtime := loadNativeReworkState(t, fixture)
	noOp := TaskResponse{Status: "no-op", Summary: "Nothing to change.", Delegations: []Delegation{}, ReworkRequests: []ReworkRequest{},
		Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	starts, spent := 1, int64(0)
	reworkErr := executeReworkSubtree(context.Background(), fixture.host, &nativeDeltaOnlyInvoker{inner: fixture.invoker, report: &noOp}, fixture.root, fixture.store, fixture.dir, fixture.plan, runtime, base, &report,
		fixture.managerID, "Correct the project owner artifact.", "Independent review requested a correction.", &starts, &spent)
	if reworkErr == nil || !strings.Contains(reworkErr.Error(), "claimed no-op while proposing files") {
		t.Fatalf("native no-op with harvested workspace changes was accepted: %v", reworkErr)
	}
}

func TestReintegrationAfterReworkAppliesNativeWorkspaceDelta(t *testing.T) {
	const childPath, resolvedBytes = "src/orders/implementation.txt", "orders implementation resolved\n"
	for _, test := range []struct {
		name      string
		conflict  bool
		writes    map[string]string
		wantBytes string
		wantErr   string
	}{
		{name: "parent edit", wantBytes: "orders implementation v2\n"},
		{name: "resolved conflict", conflict: true, writes: map[string]string{childPath: resolvedBytes}, wantBytes: resolvedBytes},
		{name: "unresolved conflict", conflict: true, wantErr: "did not resolve integration conflict paths"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := seedNativeReviewedRun(t, "integrate")
			fixture.invoker.retry = true
			report, base, runtime := loadNativeReworkState(t, fixture)
			task := findTask(report.Tasks, fixture.managerID)
			if task == nil {
				t.Fatal("fixture omitted root task")
			}
			if test.conflict {
				// A prior parent integration of the child's path conflicts with the
				// child's new output, so the parent must resolve that path itself.
				task.IntegrationCandidateID, task.IntegratedPaths = task.CandidateID, []string{childPath}
			}
			prior := task.IntegrationCandidateID
			integrated := TaskResponse{Status: "complete", Summary: "Reintegrated with a native workspace edit.", Delegations: []Delegation{},
				ReworkRequests: []ReworkRequest{}, Integrated: true, Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
			invoker := &nativeDeltaOnlyInvoker{inner: fixture.invoker, report: &integrated, writes: test.writes}
			starts, spent := 2, int64(0)
			_, reintegrateErr := reintegrateAfterRework(context.Background(), fixture.host, invoker, fixture.root, fixture.store, fixture.dir, fixture.plan, runtime, base, &report, task, &starts, &spent)
			if test.wantErr != "" {
				if reintegrateErr == nil || !strings.Contains(reintegrateErr.Error(), test.wantErr) {
					t.Fatalf("reintegration error = %v, want %q", reintegrateErr, test.wantErr)
				}
				return
			}
			if reintegrateErr != nil {
				t.Fatalf("native reintegration and its integration review: %v", reintegrateErr)
			}
			if fixture.invoker.runCalls != 1 || task.IntegrationCandidateID == "" || task.IntegrationCandidateID == prior {
				t.Fatalf("reintegration did not dispatch one native turn and bind a new candidate: runCalls=%d task=%+v err=%v", fixture.invoker.runCalls, task, reintegrateErr)
			}
			candidate, err := fixture.store.readCandidate(fixture.dir, task.IntegrationCandidateID)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(candidate.Files[childPath].Content); got != test.wantBytes {
				t.Fatalf("reintegrated %s = %q, want %q (err=%v)", childPath, got, test.wantBytes, reintegrateErr)
			}
			if got := string(candidate.Files["src/project-owner.txt"].Content); got != "bounded native repair output\n" {
				t.Fatalf("native reintegration lost the parent's harvested workspace edit: %q (err=%v)", got, reintegrateErr)
			}
			assertNativeReviewPassed(t, invoker, report, *task, "integrate", task.IntegrationCandidateID,
				map[string]string{childPath: test.wantBytes, "src/project-owner.txt": "bounded native repair output\n"})
		})
	}
}

// assertNativeReviewPassed checks that the one independent review of a native
// candidate ran end to end: it passed, bound the exact candidate, and its
// input carried the candidate's bytes.
func assertNativeReviewPassed(t *testing.T, invoker *nativeDeltaOnlyInvoker, report RunReport, task ManagerTask, phase, candidateID string, wantFiles map[string]string) {
	t.Helper()
	if len(invoker.reviews) != 1 || len(report.Reviews) != 1 {
		t.Fatalf("native candidate reviews: requests=%d records=%d, want one each", len(invoker.reviews), len(report.Reviews))
	}
	review := report.Reviews[0]
	if review.ManagerID != task.ManagerID || review.Phase != phase || review.Outcome != "pass" || review.CandidateID != candidateID || task.ReviewStatus != "pass" {
		t.Fatalf("native candidate review = %+v (task review status %q), want a %s pass bound to candidate %s", review, task.ReviewStatus, phase, candidateID)
	}
	reviewed := map[string]string{}
	for _, artifact := range invoker.reviews[0].Artifacts {
		reviewed[artifact.Path] = string(artifact.Content)
	}
	for path, want := range wantFiles {
		if got, ok := reviewed[path]; !ok || got != want {
			t.Fatalf("native candidate review input %s = %q (present=%t), want %q", path, got, ok, want)
		}
	}
}

// loadNativeReworkState prepares the persisted run state for a direct targeted
// rework call. The planned runtime already binds the reviewers, so the review
// of the reworked candidate runs end to end.
func loadNativeReworkState(t *testing.T, fixture nativeResumeRunFixture) (RunReport, *Project, Runtime) {
	t.Helper()
	runtime := mustLoadRuntime(t, fixture.root)
	report, err := fixture.store.readLatestState(fixture.plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	base, err := fixture.host.Load(fixture.root, fixture.plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	return report, base, runtime
}

func seedNativeResumeRun(t *testing.T) nativeResumeRunFixture {
	return seedNativeRun(t, "work", false, false)
}

func seedNativeIntegrationRun(t *testing.T) nativeResumeRunFixture {
	return seedNativeRun(t, "integrate", false, false)
}

func seedNativeReworkResumeRun(t *testing.T) nativeResumeRunFixture {
	return seedNativeRun(t, "work", true, false)
}

// seedNativeReviewedRun plans with process reviewers for every Manager, so a
// review of a native candidate is bound to the planned runtime.
func seedNativeReviewedRun(t *testing.T, phase string) nativeResumeRunFixture {
	return seedNativeRun(t, phase, false, true)
}

func seedNativeRun(t *testing.T, phase string, reviewRework, reviewed bool) nativeResumeRunFixture {
	t.Helper()
	root := makeProjectRunFixture(t)
	writeE2E(t, root, "AGENTS.md", "native manager instructions\n")
	writeE2E(t, root, "src/project-owner.txt", "project owner baseline\n")
	selectorID := e2eManagerID("", "project-owner")
	managerID := selectorID
	if phase == "integrate" {
		selectorID = e2eManagerID("orders", "orders")
		checkBinding := regexp.MustCompile(`(?s)  checks:\n    - apiVersion: .*?\n  required:`)
		for _, artifactPath := range []string{".markitect/model/orders/artifact.yaml", ".markitect/model/inventory/artifact.yaml"} {
			artifact, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifactPath)))
			if err != nil {
				t.Fatal(err)
			}
			updated := checkBinding.ReplaceAllString(string(artifact), "  checks: []\n  required:")
			if updated == string(artifact) {
				t.Fatalf("could not remove plan check binding from %s", artifactPath)
			}
			writeE2E(t, root, artifactPath, updated)
		}
		projectPath := filepath.Join(root, ".markitect/project.yaml")
		projectConfig, err := os.ReadFile(projectPath)
		if err != nil {
			t.Fatal(err)
		}
		projectText := strings.Replace(string(projectConfig), "  - .markitect/model/orders/check.yaml\n", "", 1)
		if projectText == string(projectConfig) {
			t.Fatal("could not remove the orders check from the model file set")
		}
		writeE2E(t, root, ".markitect/project.yaml", projectText)
		writeE2E(t, root, "src/inventory/implementation.txt", "inventory implementation v2\n")
		gitE2E(t, root, "add", "AGENTS.md", "src/project-owner.txt", ".markitect/project.yaml", ".markitect/model/orders/artifact.yaml", ".markitect/model/inventory/artifact.yaml", "src/inventory/implementation.txt")
	} else {
		gitE2E(t, root, "add", "AGENTS.md", "src/project-owner.txt")
	}
	gitE2E(t, root, "commit", "-m", "add native instruction and owner fixtures")
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		runtime.Limits.MaxRetries = 1
		if phase == "integrate" {
			runtime.Limits.MaxStarts = 3
		}
		if reviewed {
			runtime.Review = &ReviewConfig{Agents: map[string]Agent{}, MaxRounds: 1, MaxManagerRounds: 1}
			for id, agent := range runtime.Agents {
				runtime.Review.Agents[id] = agent
			}
			runtime.Limits.MaxStarts = 32
		}
		agent := runtime.Agents[managerID]
		native := appServerAgent(t)
		native.ProviderVersion = codexappserver.SupportedProviderVersion
		native.Pricing = agent.Pricing
		native.Environment = append([]string(nil), agent.Environment...)
		instructionPath, err := filepath.Abs(filepath.Join(root, "AGENTS.md"))
		if err != nil {
			t.Fatal(err)
		}
		instructionBytes, err := os.ReadFile(instructionPath)
		if err != nil {
			t.Fatal(err)
		}
		native.RuntimeFiles = []agentexec.RuntimeFile{{Path: instructionPath, Mode: "0644", Digest: "sha256:" + digestBytes(instructionBytes)}}
		runtime.Agents[managerID] = native
	})
	storage := filepath.Join(t.TempDir(), "workspace-cache")
	workspaceLimits := projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20}
	preparationService, err := projectworkspace.NewGitService(storage, workspaceLimits)
	if err != nil {
		t.Fatal(err)
	}
	host := projectworkHost()
	host.Workspaces = preparationService
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{
		Goal: "Implement the project owner's isolated fixture artifact.", Managers: []string{selectorID}, ExecuteAuthorized: true,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if phase == "work" && (len(plan.Managers) != 1 || plan.Managers[0].ManagerID != managerID) {
		t.Fatalf("selected plan tasks = %+v, want the project owner only", plan.Managers)
	}
	if phase == "integrate" && (len(plan.Managers) != 2 || findTask(plan.Managers, managerID) == nil || findTask(plan.Managers, e2eManagerID("orders", "orders")) == nil || len(plan.Checks) != 0) {
		t.Fatalf("integration fixture plan = tasks:%+v checks:%+v, want root+orders with no check starts", plan.Managers, plan.Checks)
	}

	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := store.readCandidate(dir, plan.InitialCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	report := RunReport{APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Operation: plan.Operation,
		Status: StatusInterrupted, Mode: ModeControlledLocal, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot, ModelDigest: plan.ModelDigest,
		RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers),
		Candidate: CandidateRef{ID: initial.ID, Snapshot: initial.Digest, Files: map[string]string{}}, Revision: 1}
	task := findTask(report.Tasks, managerID)
	if task == nil {
		t.Fatal("plan omitted native task")
	}
	if phase == "work" {
		task.State, task.RepairPhase, task.WorkAttempts, task.Attempts = "invoking", "work", 1, 1
		if reviewRework {
			reviewed := candidateData{Parents: []string{initial.ID}, Files: map[string]File{
				"src/project-owner.txt": {Path: "src/project-owner.txt", Mode: "100644", Content: []byte("candidate accepted for review\n")},
			}}
			reviewed.ID, err = newID()
			if err != nil {
				t.Fatal(err)
			}
			if err := store.writeCandidate(dir, reviewed); err != nil {
				t.Fatal(err)
			}
			reviewed, err = store.readCandidate(dir, reviewed.ID)
			if err != nil {
				t.Fatal(err)
			}
			task.CandidateID, task.ReviewCandidateID, task.ReviewStatus, task.WorkAttempts, task.Attempts = reviewed.ID, reviewed.ID, "rework-requested", 2, 2
			report.Reviews = []ReviewRecord{{TaskID: task.ID, ManagerID: managerID, Round: 1, Phase: "work", CandidateID: reviewed.ID, CandidateDigest: reviewed.Digest, Outcome: "fail",
				Findings: []ReviewFinding{{Path: "src/project-owner.txt", Expectation: "tighten the reviewed output", Grounding: "statement:project-owner"}}, At: time.Now().UTC()}}
		}
	} else {
		task.State, task.RepairPhase, task.WorkAttempts, task.IntegrationAttempts, task.Attempts = "integrating", "integrate", 1, 1, 2
		task.CandidateID = initial.ID
		child := findTask(report.Tasks, e2eManagerID("orders", "orders"))
		if child == nil {
			t.Fatal("integration plan omitted orders child")
		}
		childWork := candidateData{Parents: []string{initial.ID}, Files: map[string]File{
			"src/orders/implementation.txt": {Path: "src/orders/implementation.txt", Mode: "100644", Content: []byte("orders implementation v2\n")},
		}}
		childWork.ID, err = newID()
		if err != nil {
			t.Fatal(err)
		}
		childWork.Parents = []string{initial.ID}
		if err := store.writeCandidate(dir, childWork); err != nil {
			t.Fatal(err)
		}
		childWork, err = store.readCandidate(dir, childWork.ID)
		if err != nil {
			t.Fatal(err)
		}
		child.State, child.WorkAttempts, child.Attempts, child.CandidateID, child.ReportStatus = "worked", 1, 1, childWork.ID, "complete"
	}
	if err := store.appendState(report); err != nil {
		t.Fatal(err)
	}
	baseProject, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	boundSnapshot, err := snapshotWithCandidate(baseProject.Snapshot, initial)
	if err != nil {
		t.Fatal(err)
	}
	boundProject, err := host.FromSnapshot(root, boundSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	dependencies, err := managerDependencies(boundProject.Report, report.Tasks)
	if err != nil {
		t.Fatal(err)
	}
	input, children, childReports, conflicts, err := recoveryManagerInput(store, dir, host, root, baseProject.Snapshot, boundProject.Report, report, plan, *task, phase, dependencies[managerID])
	if err != nil {
		t.Fatal(err)
	}
	repairRound, repairChecks := repairContext(report, *task)
	request, err := managerInvocationRequest(root, mustLoadRuntime(t, root), plan, input, *task, phase, children, conflicts, childReports, task.RepairDiagnostic, repairRound, repairChecks)
	if err != nil {
		t.Fatal(err)
	}
	configAgent := mustLoadRuntime(t, root).Agents[managerID]
	config, err := configAgent.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := host.Load(root, input.Revision)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := projectworkspace.InspectRepository(context.Background(), root, input.Revision)
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := workspaceOverlay(fixed.Snapshot, input.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(overlay)
	if err != nil {
		t.Fatal(err)
	}
	excluded, err := ignoredWritePaths(input.Config, input.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	allowed := allowedWritePaths(input.Config, input.Report, *task, phase, conflicts, input.Snapshot)
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	workspaceRequest := projectworkspace.Request{RepositoryRoot: filepath.Clean(absoluteRoot), RepositoryIdentity: identity.Digest,
		BaseSHA: input.Revision, OverlayDigest: binding.OverlayDigest, TaskID: nativeTaskAttemptID(*task, phase),
		AllowedPaths: allowed, ExcludedPaths: excluded}
	handle, err := preparationService.PrepareCandidate(context.Background(), workspaceRequest, overlay, overlayDigest)
	if err != nil {
		t.Fatal(err)
	}
	const recoveredBytes = "recovered native proposal\n"
	if phase == "work" {
		if err := os.WriteFile(filepath.Join(handle.CWD, "src", "project-owner.txt"), []byte(recoveredBytes), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	fingerprint, err := NewTransportInvoker(codexappserver.Options{}).Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	lifecycle := &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "resume-session", TurnID: "resume-turn", State: "completed", Accounting: "partial",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: invocation.RunID, Role: agentexec.RoleExecutor, State: "completed"}}}
	tokensIn, tokensOut := int64(3), int64(2)
	taskResponse := TaskResponse{Status: "complete", Summary: "Recovered exact native work.", Delegations: []Delegation{},
		ReworkRequests: []ReworkRequest{}, Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	if phase == "integrate" {
		taskResponse.Summary = "Recovered exact native integration."
		taskResponse.Integrated = true
	}
	reportJSON, err := json.Marshal(taskResponse)
	if err != nil {
		t.Fatal(err)
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{},
		EvidenceRefs:   []string{}, VerifierObservations: []agentexec.Observation{}, ReportJSON: reportJSON, Uncertainty: []string{},
		Usage: &agentexec.Usage{Source: "provider-reported", InputTokens: &tokensIn, OutputTokens: &tokensOut}}
	if phase == "work" {
		response.CandidateFiles = []agentexec.CandidateFile{{Path: "src/project-owner.txt", Mode: "0644", Content: recoveredBytes}}
	}
	recovered := agentexec.RunResult{Response: response, Receipt: agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID,
		InputDigest: invocation.InputDigest, ConfigDigest: fingerprint, ProviderVersion: configAgent.ProviderVersion,
		Outcome: agentexec.OutcomeProposed, Usage: response.Usage, Lifecycle: lifecycle}}
	privateDir := filepath.Join(root, ".markitect", "runs", "private")
	nativeJournal, err := newNativeJournal(privateDir, handle.CWD, handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	journalOptions := nativeJournal.wrapOptions(codexappserver.Options{})
	if err := journalOptions.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: invocation.RunID, Role: agentexec.RoleExecutor}); err != nil {
		t.Fatal(err)
	}
	recoveryHandle := codexappserver.RecoveryHandle{Protocol: "codex-app-server/fixture", Fingerprint: fingerprint,
		Invocation: invocation, Workspace: handle, ThreadID: "resume-thread", SessionID: lifecycle.SessionID,
		TurnID: lifecycle.TurnID, TurnDispatched: true}
	if err := journalOptions.OnHandle(context.Background(), recoveryHandle); err != nil {
		t.Fatal(err)
	}
	journal := workspaceJournal{OwnerRunID: plan.ID, Request: workspaceRequest, Handle: handle, Overlay: overlay, OverlayDigest: overlayDigest,
		State: "preserved"}
	if err := persistWorkspaceJournal(filepath.Join(privateDir, "workspaces", handle.ID+".json"), journal); err != nil {
		t.Fatal(err)
	}

	service, err := projectworkspace.NewGitService(storage, workspaceLimits)
	if err != nil {
		t.Fatal(err)
	}
	host.Workspaces = service
	invoker := &nativeResumeFixtureInvoker{result: recovered}
	return nativeResumeRunFixture{root: root, host: host, plan: plan, store: store, dir: dir, managerID: managerID, phase: phase, handle: handle, invoker: invoker}
}

func assertNativeResumeResult(t *testing.T, fixture nativeResumeRunFixture, resumed RunReport, resumeErr error, repaired bool, wantRecoverCalls int) {
	t.Helper()
	if resumeErr != nil {
		t.Fatalf("Resume exact native attempt: status=%s err=%v", resumed.Status, resumeErr)
	}
	wantRunCalls := 0
	wantAttempts := 1
	wantInvocations := 1
	wantBytes := "recovered native proposal\n"
	if repaired {
		wantRunCalls, wantAttempts, wantInvocations, wantBytes = 1, 2, 2, "bounded native repair output\n"
	}
	if fixture.invoker.runCalls != wantRunCalls || fixture.invoker.recoverCalls != wantRecoverCalls {
		t.Fatalf("Resume dispatched/recovered calls = %d/%d, want %d/%d", fixture.invoker.runCalls, fixture.invoker.recoverCalls, wantRunCalls, wantRecoverCalls)
	}
	if len(fixture.invoker.workspaceIDs) != wantRunCalls+wantRecoverCalls {
		t.Fatalf("workspace attempt IDs = %v, want %d entries", fixture.invoker.workspaceIDs, wantRunCalls+wantRecoverCalls)
	}
	taskTemplate := findTask(fixture.plan.Managers, fixture.managerID)
	if taskTemplate == nil {
		t.Fatal("plan omitted task used to bind attempt ID")
	}
	expectedFirstID := nativeTaskAttemptID(ManagerTask{ID: taskTemplate.ID, WorkAttempts: 1}, "work")
	for index := 0; index < wantRecoverCalls; index++ {
		if fixture.invoker.workspaceIDs[index] != expectedFirstID {
			t.Fatalf("recovered workspace attempt ID[%d] = %q, want %q", index, fixture.invoker.workspaceIDs[index], expectedFirstID)
		}
	}
	if repaired {
		expectedRetryID := nativeTaskAttemptID(ManagerTask{ID: taskTemplate.ID, WorkAttempts: 2}, "work")
		if fixture.invoker.workspaceIDs[len(fixture.invoker.workspaceIDs)-1] != expectedRetryID {
			t.Fatalf("repair reused the wrong native attempt ID: got %v want last %q", fixture.invoker.workspaceIDs, expectedRetryID)
		}
	}
	if resumed.Status != StatusIntegrated || len(resumed.Invocations) != wantInvocations {
		t.Fatalf("recovered run status or receipt count = %s/%d, want integrated/%d: %+v", resumed.Status, len(resumed.Invocations), wantInvocations, resumed)
	}
	recoveredTask := findTask(resumed.Tasks, fixture.managerID)
	if recoveredTask == nil || recoveredTask.WorkAttempts != wantAttempts || recoveredTask.Attempts != wantAttempts || recoveredTask.State != "worked" {
		t.Fatalf("Resume changed the reserved attempt instead of consuming it: %+v", recoveredTask)
	}
	candidate, err := fixture.store.readCandidate(fixture.dir, recoveredTask.CandidateID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(candidate.Files["src/project-owner.txt"].Content); got != wantBytes {
		t.Fatalf("final candidate bytes = %q, want %q", got, wantBytes)
	}
	if _, err := os.Stat(fixture.handle.CWD); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("terminal recovered workspace was not closed: stat err=%v", err)
	}
}

func mustLoadRuntime(t *testing.T, root string) Runtime {
	t.Helper()
	runtime, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}
