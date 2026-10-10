package projectrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

// This test exercises the reviewer recovery seam without a provider. A dry
// scripted invocation first captures the exact request digest to seed the
// durable start ledger; the recovery call must then require the private native
// journal, without calling Run or reserving a second start.
type resumeReviewInvoker struct {
	request  agentexec.Request
	runCalls int
}

func (i *resumeReviewInvoker) Run(_ context.Context, _ agentexec.Config, request agentexec.Request, _ agentexec.RunOptions) (agentexec.RunResult, error) {
	i.runCalls++
	i.request = request
	return agentexec.RunResult{}, errors.New("fixture has no provider")
}
func (i *resumeReviewInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return NewTransportInvoker(codexappserver.Options{}).Fingerprint(config)
}

func TestPendingNativeReviewRequiresExactJournalWithoutReservingOrRunning(t *testing.T) {
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		base := runtime.Agents[e2eManagerID("", "project-owner")]
		native := workspaceBridgeAgent(t, root)
		native.Command = base.Command
		native.Model = base.Model
		native.ProviderVersion = codexappserver.SupportedProviderVersion
		native.Timeout = base.Timeout
		native.MaxStdoutBytes = base.MaxStdoutBytes
		native.MaxStderrBytes = base.MaxStderrBytes
		native.Pricing = base.Pricing
		native.AppServer = &AppServerSettings{ReasoningEffort: "medium", MaxEventBytes: 1 << 20}
		reviewers := map[string]Agent{}
		for _, id := range []string{e2eManagerID("", "project-owner"), e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")} {
			reviewers[id] = native
		}
		runtime.Review = &ReviewConfig{Agents: reviewers, MaxRounds: 2, MaxManagerRounds: 2}
	})
	host := projectworkHost()
	workspaceService, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	host.Workspaces = workspaceService
	revision := identityHead(t, root)
	plan, err := Plan(host, root, revision, PlanRequest{Goal: "Review the parent-owned candidate.",
		Managers: []string{e2eManagerID("", "project-owner"), e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
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
	candidate, err := store.readCandidate(dir, plan.InitialCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	task := *findTask(plan.Managers, e2eManagerID("", "project-owner"))
	if len(activeChildren(plan.Managers, task.ManagerID)) == 0 {
		t.Fatal("fixture should exercise a parent task with child responsibilities")
	}
	task.State = "reviewing"
	task.ReviewStatus = "uncertain"
	task.ReviewCandidateID = candidate.ID
	task.CandidateID = candidate.ID // Parent review is work-phase even while children remain active.
	task.ReviewRound = 1
	project, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	invoker := &resumeReviewInvoker{}
	_, originalLog, err := invokeReviewer(context.Background(), host, invoker, root, plan, runtime, project, task,
		"work", 1, candidate, nil, RunReport{})
	if err == nil || invoker.runCalls != 1 || invoker.request.Role != agentexec.RoleExecutor {
		t.Fatalf("dry request capture did not reach the scripted invoker: err=%v calls=%d", err, invoker.runCalls)
	}
	originalLog.Receipt = agentexec.Receipt{RunID: "original-review-run", InputDigest: originalLog.InputDigest}
	originalLog.ReportID = originalLog.Receipt.RunID
	// The dry setup invocation deliberately leaves its private workspace journal;
	// remove that fixture-owned journal to represent the crash-loss case.
	privateWorkspaces := filepath.Join(root, ".markitect", "runs", "private", "workspaces")
	if err := os.RemoveAll(privateWorkspaces); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(invoker.request.Context), `"phase":"integrate"`) {
		t.Fatal("parent work candidate was incorrectly bound to integration review")
	}

	// Model the crash state: the original reviewer start is durable, but its
	// private workspace journal is absent. Recovery must stop without a second
	// OnStart reservation or a new role Run.
	invoker.runCalls = 0
	report := RunReport{ID: plan.ID, PlanID: plan.ID, StartedAt: time.Now().UTC(), Tasks: cloneTasks(plan.Managers),
		Candidate: CandidateRef{ID: candidate.ID}, Invocations: []InvocationLog{originalLog}}
	current := findTask(report.Tasks, task.ManagerID)
	*current = task
	err = recoverPendingNativeReview(context.Background(), host, invoker, root, store, dir, plan, runtime, base, &report, task.ManagerID)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "journal is missing") {
		t.Fatalf("missing native review journal was not preserved as blocking: %v", err)
	}
	if invoker.runCalls != 0 || len(report.Invocations) != 1 || report.Invocations[0].InputDigest != originalLog.InputDigest {
		t.Fatalf("recovery replayed or duplicated the reviewer start: runs=%d invocations=%+v", invoker.runCalls, report.Invocations)
	}
	if current.ReviewStatus != "uncertain" || current.ReviewCandidateID != candidate.ID || current.ReviewRound != 1 {
		t.Fatalf("pending exact review identity was not preserved: %+v", current)
	}
	if _, statErr := os.Stat(privateWorkspaces); !os.IsNotExist(statErr) {
		t.Fatalf("test unexpectedly created a native workspace journal: %v", statErr)
	}
}

// A failed work review persists review-rework-ready before the rework turn
// starts. Resume must send that turn the reviewed candidate, as the
// uninterrupted review loop does, not the parent's bytes.
func TestResumedReviewReworkStartsFromReviewedCandidate(t *testing.T) {
	root := makeProjectRunFixture(t)
	enableE2EReviews(t, root, 100000)
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement and review owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
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
	initial, err := store.readCandidate(dir, plan.InitialCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	const reviewedBytes = "reviewed orders v2\n"
	reviewed := candidateData{Parents: []string{initial.ID}, Files: map[string]File{
		"src/orders/implementation.txt": {Path: "src/orders/implementation.txt", Mode: snapshot.RegularMode, Content: []byte(reviewedBytes)},
	}}
	if reviewed.ID, err = newID(); err != nil {
		t.Fatal(err)
	}
	if err := store.writeCandidate(dir, reviewed); err != nil {
		t.Fatal(err)
	}
	if reviewed, err = store.readCandidate(dir, reviewed.ID); err != nil {
		t.Fatal(err)
	}
	rootID, ordersID, inventoryID := e2eManagerID("", "project-owner"), e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")
	report := RunReport{APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Operation: plan.Operation, Status: StatusInterrupted, Mode: ModeControlledLocal,
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot,
		ModelDigest: plan.ModelDigest, RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers),
		Candidate: CandidateRef{ID: reviewed.ID, Snapshot: reviewed.Digest, Files: map[string]string{}}, Revision: 1}
	rootTask, ordersTask, inventoryTask := findTask(report.Tasks, rootID), findTask(report.Tasks, ordersID), findTask(report.Tasks, inventoryID)
	if rootTask == nil || ordersTask == nil || inventoryTask == nil {
		t.Fatalf("plan omitted expected tasks: %+v", report.Tasks)
	}
	rootTask.State, rootTask.CandidateID, rootTask.WorkAttempts, rootTask.Attempts, rootTask.ReportStatus, rootTask.ReviewStatus = "worked", initial.ID, 1, 1, "complete", "not-required"
	inventoryTask.State, inventoryTask.CandidateID, inventoryTask.WorkAttempts, inventoryTask.Attempts, inventoryTask.ReportStatus, inventoryTask.ReviewStatus = "worked", initial.ID, 1, 1, "complete", "pass"
	// The state the work loop and pending-review recovery persist after a failed review.
	ordersTask.State, ordersTask.ReviewStatus, ordersTask.CandidateID = "review-rework-ready", "rework-requested", reviewed.ID
	ordersTask.ReviewCandidateID, ordersTask.ReviewRound, ordersTask.WorkAttempts, ordersTask.Attempts, ordersTask.ReportStatus = reviewed.ID, 1, 1, 1, "complete"
	ordersTask.WrittenPaths = []string{"src/orders/implementation.txt"}
	ordersTask.RepairDiagnostic = "independent review findings: src/orders/implementation.txt: tighten the reviewed implementation (statement:orders-work)"
	report.Reviews = []ReviewRecord{{TaskID: ordersTask.ID, ManagerID: ordersID, Round: 1, Phase: "work", CandidateID: reviewed.ID, CandidateDigest: reviewed.Digest,
		Outcome: "fail", Findings: []ReviewFinding{{Path: "src/orders/implementation.txt", Expectation: "tighten the reviewed implementation", Grounding: "statement:orders-work"}}, At: time.Now().UTC()}}
	if err := store.appendState(report); err != nil {
		t.Fatal(err)
	}

	invoker := &resumeReviewInvoker{}
	resumed, resumeErr := Resume(context.Background(), host, invoker, root, plan.ID)
	if invoker.runCalls != 1 {
		t.Fatalf("Resume dispatched %d Manager requests, want the single orders rework request: status=%s err=%v", invoker.runCalls, resumed.Status, resumeErr)
	}
	got, found := "", false
	for _, artifact := range invoker.request.Artifacts {
		if artifact.Path == "src/orders/implementation.txt" {
			got, found = string(artifact.Content), true
		}
	}
	if !found {
		t.Fatalf("orders rework request omitted its owned artifact: %+v", invoker.request.Artifacts)
	}
	if got != reviewedBytes {
		t.Fatalf("resumed review rework input for orders = %q, want the reviewed candidate bytes %q", got, reviewedBytes)
	}
}
