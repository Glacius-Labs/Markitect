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
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
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

const reviewedOrdersBytes = "reviewed orders v2\n"

// reviewReworkFixture holds the state the work loop and pending-review
// recovery persist after a failed orders work review. The report is not yet
// stored, so a test can change it first.
type reviewReworkFixture struct {
	root     string
	host     Host
	plan     PlanRecord
	store    *runStore
	dir      string
	initial  candidateData
	reviewed candidateData
	report   RunReport
}

func seedReviewRework(t *testing.T) reviewReworkFixture {
	t.Helper()
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
	reviewed := storeReviewFixtureCandidate(t, store, dir, []string{initial.ID}, map[string]string{"src/orders/implementation.txt": reviewedOrdersBytes})
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
	return reviewReworkFixture{root: root, host: host, plan: plan, store: store, dir: dir, initial: initial, reviewed: reviewed, report: report}
}

// A failed work review persists review-rework-ready before the rework turn
// starts. Resume must send that turn the reviewed candidate, as the
// uninterrupted review loop does, not the parent's bytes.
func TestResumedReviewReworkStartsFromReviewedCandidate(t *testing.T) {
	fixture := seedReviewRework(t)
	if err := fixture.store.appendState(fixture.report); err != nil {
		t.Fatal(err)
	}

	invoker := &resumeReviewInvoker{}
	resumed, resumeErr := Resume(context.Background(), fixture.host, invoker, fixture.root, fixture.plan.ID)
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
	if got != reviewedOrdersBytes {
		t.Fatalf("resumed review rework input for orders = %q, want the reviewed candidate bytes %q", got, reviewedOrdersBytes)
	}
}

// Review rework continues only from the exact candidate its failed review
// assessed. A stale review identity or changed reviewed bytes leave no
// trustworthy rework base, so the fresh and the recovered turn both refuse.
func TestReviewReworkRefusesCandidateOutsideItsFailedReview(t *testing.T) {
	for _, test := range []struct {
		name   string
		tamper func(*reviewReworkFixture, *ManagerTask)
		want   string
	}{
		{"stale review identity", func(fixture *reviewReworkFixture, task *ManagerTask) { task.ReviewCandidateID = fixture.initial.ID }, "is not the reviewed candidate"},
		{"changed reviewed bytes", func(fixture *reviewReworkFixture, _ *ManagerTask) {
			fixture.report.Reviews[0].CandidateDigest = fixture.initial.Digest
		}, "differs from the bytes its review assessed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := seedReviewRework(t)
			task := findTask(fixture.report.Tasks, e2eManagerID("orders", "orders"))
			test.tamper(&fixture, task)
			// Recovery refuses before it compiles any input, so no model is needed.
			_, _, _, _, recoveryErr := recoveryManagerInput(fixture.store, fixture.dir, fixture.host, fixture.root, nil, projectmodel.Report{}, fixture.report, fixture.plan, *task, "work", nil)
			if recoveryErr == nil || !strings.Contains(recoveryErr.Error(), test.want) {
				t.Fatalf("recovered rework input error = %v, want %q", recoveryErr, test.want)
			}
			if err := fixture.store.appendState(fixture.report); err != nil {
				t.Fatal(err)
			}
			invoker := &resumeReviewInvoker{}
			resumed, resumeErr := Resume(context.Background(), fixture.host, invoker, fixture.root, fixture.plan.ID)
			if invoker.runCalls != 0 || resumeErr == nil || !strings.Contains(resumeErr.Error(), test.want) {
				t.Fatalf("fresh rework turn was not refused: runs=%d status=%s err=%v", invoker.runCalls, resumed.Status, resumeErr)
			}
		})
	}
}

func storeReviewFixtureCandidate(t *testing.T, store *runStore, dir string, parents []string, files map[string]string) candidateData {
	t.Helper()
	candidate := candidateData{Parents: parents, Files: map[string]File{}}
	for path, content := range files {
		candidate.Files[path] = File{Path: path, Mode: snapshot.RegularMode, Content: []byte(content)}
	}
	var err error
	if candidate.ID, err = newID(); err != nil {
		t.Fatal(err)
	}
	if err := store.writeCandidate(dir, candidate); err != nil {
		t.Fatal(err)
	}
	if candidate, err = store.readCandidate(dir, candidate.ID); err != nil {
		t.Fatal(err)
	}
	return candidate
}
