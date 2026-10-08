package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestReviewFindingRequiresExactCandidatePathAndAcceptedGrounding(t *testing.T) {
	model := projectmodel.ManagerContext{Statements: []projectmodel.Statement{{ID: "accepted-statement"}}, Artifacts: []projectmodel.Artifact{{Paths: []string{"src/orders/result.txt"}}}}
	files := []agentexec.Artifact{{Path: "src/orders/result.txt"}}
	valid := []reviewFindingResponse{{Path: "src/orders/result.txt", Expectation: "write the declared output", Grounding: "statement:accepted-statement"}}
	if _, err := validateReviewFindings(valid, model, files); err != nil {
		t.Fatalf("accepted grounded finding rejected: %v", err)
	}
	for name, finding := range map[string]reviewFindingResponse{
		"foreign path":          {Path: "src/inventory/result.txt", Expectation: "change the sibling", Grounding: "statement:accepted-statement"},
		"unknown statement":     {Path: "src/orders/result.txt", Expectation: "invent a requirement", Grounding: "statement:unknown"},
		"unknown artifact path": {Path: "src/orders/result.txt", Expectation: "invent an artifact", Grounding: "artifact-path:src/other.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateReviewFindings([]reviewFindingResponse{finding}, model, files); err == nil {
				t.Fatal("ungrounded or out-of-scope review finding was accepted")
			}
		})
	}
}

func TestReviewScopeDigestSurvivesSiblingMergeButChangesWithOwnedBytes(t *testing.T) {
	root := makeProjectRunFixture(t)
	base, err := projectworkHost().Load(root, identityHead(t, root))
	if err != nil {
		t.Fatal(err)
	}
	plan := PlanRecord{Goal: "Implement both fixture artifacts."}
	task := ManagerTask{ManagerID: e2eManagerID("orders", "orders"), Goal: "Implement the orders artifact.", Owns: []string{"src/orders/"}}
	initial, err := reviewScopeDigest(plan, base, task, "work")
	if err != nil {
		t.Fatal(err)
	}
	siblingChange := candidateData{Files: map[string]File{"src/inventory/implementation.txt": {Path: "src/inventory/implementation.txt", Mode: "0644", Content: []byte("inventory changed")}}}
	mergedSibling, err := projectForCandidate(projectworkHost(), root, base.Snapshot, siblingChange)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := reviewScopeDigest(plan, mergedSibling, task, "work")
	if err != nil || unchanged != initial {
		t.Fatalf("unaffected review scope changed after sibling merge: got=%s want=%s err=%v", unchanged, initial, err)
	}
	ownedChange := candidateData{Files: map[string]File{"src/orders/implementation.txt": {Path: "src/orders/implementation.txt", Mode: "0644", Content: []byte("orders changed")}}}
	mergedOwned, err := projectForCandidate(projectworkHost(), root, base.Snapshot, ownedChange)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := reviewScopeDigest(plan, mergedOwned, task, "work")
	if err != nil || changed == initial {
		t.Fatalf("review scope did not invalidate after owned bytes changed: got=%s initial=%s err=%v", changed, initial, err)
	}

	parent := ManagerTask{ManagerID: e2eManagerID("", "project-owner"), Goal: "Reconcile integration.", Owns: []string{"."}}
	if reviewRequired(base, parent) {
		t.Fatal("empty routing manager scope unexpectedly requires a reviewer invocation")
	}
	parent.WrittenPaths = []string{"src/deleted-output.txt"}
	if !reviewRequired(base, parent) {
		t.Fatal("recorded deletion path incorrectly became a pure-routing skip")
	}
	parent.WrittenPaths = nil
	parent.Artifacts = []string{"declared-but-missing-artifact"}
	if !reviewRequired(base, parent) {
		t.Fatal("missing declared artifact incorrectly became a pure-routing skip")
	}
	parent.Artifacts = nil
	parentFiles := scopedCandidateFiles(base, parent)
	for _, file := range parentFiles {
		if strings.HasPrefix(file.Path, "src/orders/") || strings.HasPrefix(file.Path, "src/inventory/") {
			t.Fatalf("broad parent owns rule leaked child bytes into reviewer input: %s", file.Path)
		}
	}
	parentInitial, err := reviewScopeDigest(plan, base, parent, "integrate")
	if err != nil {
		t.Fatal(err)
	}
	parentAfterSibling, err := reviewScopeDigest(plan, mergedSibling, parent, "integrate")
	if err != nil || parentAfterSibling != parentInitial {
		t.Fatalf("parent review scope changed with independent child bytes: got=%s want=%s err=%v", parentAfterSibling, parentInitial, err)
	}
}

func TestCheckRepairPreservesReviewScopeHistoryForDeletedPaths(t *testing.T) {
	root := makeProjectRunFixture(t)
	base, err := projectworkHost().Load(root, identityHead(t, root))
	if err != nil {
		t.Fatal(err)
	}
	task := ManagerTask{
		ManagerID:       e2eManagerID("", "project-owner"),
		Goal:            "Reconcile integration.",
		Owns:            []string{"."},
		WrittenPaths:    []string{"src/deleted-output.txt"},
		IntegratedPaths: []string{"src/removed-by-parent.txt"},
	}
	resetTaskForRepair(&task)
	if len(task.WrittenPaths) != 1 || task.WrittenPaths[0] != "src/deleted-output.txt" ||
		len(task.IntegratedPaths) != 1 || task.IntegratedPaths[0] != "src/removed-by-parent.txt" {
		t.Fatalf("check repair discarded cumulative changed-path evidence: written=%v integrated=%v", task.WrittenPaths, task.IntegratedPaths)
	}
	if !reviewRequired(base, task) {
		t.Fatal("deleted paths became a pure-routing skip after check repair")
	}
}

func TestChildRevertKeepsRevertAndPreservesUnrelatedParentIntegrationEdit(t *testing.T) {
	root := makeProjectRunFixture(t)
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(RunsPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := store.createRun("00000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	const (
		workID       = "00000000000000000000000000000002"
		integratedID = "00000000000000000000000000000003"
		reworkedID   = "00000000000000000000000000000004"
	)
	write := func(id string, files map[string]File) {
		t.Helper()
		if err := store.writeCandidate(dir, candidateData{ID: id, Files: files}); err != nil {
			t.Fatal(err)
		}
	}
	baseFile := File{Path: "src/orders/implementation.txt", Mode: "0644", Content: []byte("orders implementation v1\n")}
	write(workID, map[string]File{baseFile.Path: baseFile})
	write(integratedID, map[string]File{
		baseFile.Path:                 {Path: baseFile.Path, Mode: "0644", Content: []byte("orders implementation v2\n")},
		"src/project-integration.txt": {Path: "src/project-integration.txt", Mode: "0644", Content: []byte("parent edit\n")},
	})
	// The reworked child explicitly restores the original work bytes. Comparing
	// its result only against the original work candidate would otherwise skip
	// this path and accidentally retain the stale integrated v2 bytes.
	write(reworkedID, map[string]File{baseFile.Path: baseFile})
	parentID := e2eManagerID("", "project-owner")
	childID := e2eManagerID("orders", "orders")
	parent := ManagerTask{ManagerID: parentID, CandidateID: workID, IntegrationCandidateID: integratedID, IntegratedPaths: []string{"src/project-integration.txt"}}
	child := ManagerTask{ManagerID: childID, CandidateID: reworkedID}
	merged, conflicts, err := mergeChildCandidates(store, dir, []ManagerTask{parent, child}, parent, []string{childID})
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unrelated parent integration edit caused conflict: %v", conflicts)
	}
	if got := string(merged.Files[baseFile.Path].Content); got != string(baseFile.Content) {
		t.Fatalf("child revert was lost during remerge: got %q want %q", got, baseFile.Content)
	}
	if got := string(merged.Files["src/project-integration.txt"].Content); got != "parent edit\n" {
		t.Fatalf("unrelated parent integration edit was lost: %q", got)
	}
	// A second no-op parent integration must retain its cumulative ownership
	// paths so a later child merge can still carry the same parent edit.
	secondIntegration := merged
	secondIntegration.ID = "00000000000000000000000000000005"
	if err := store.writeCandidate(dir, secondIntegration); err != nil {
		t.Fatal(err)
	}
	parent.IntegrationCandidateID = secondIntegration.ID
	parent.IntegratedPaths = unionPaths(parent.IntegratedPaths, changedCandidatePaths(merged, secondIntegration))
	mergedAgain, conflicts, err := mergeChildCandidates(store, dir, []ManagerTask{parent, child}, parent, []string{childID})
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 || string(mergedAgain.Files[baseFile.Path].Content) != string(baseFile.Content) || string(mergedAgain.Files["src/project-integration.txt"].Content) != "parent edit\n" {
		t.Fatalf("second no-op reintegration lost candidate state: conflicts=%v files=%+v", conflicts, mergedAgain.Files)
	}
}

func TestNestedManagerReworkRequestsAreDrainedForNextBoundedRound(t *testing.T) {
	request := ReworkRequest{ManagerID: "leaf", Goal: "Correct the leaf output.", Reason: "Nested integration found a specific defect."}
	report := RunReport{Tasks: []ManagerTask{{ManagerID: "root"}, {ManagerID: "middle", ReworkRequests: []ReworkRequest{request}}, {ManagerID: "leaf"}}}
	pending, err := takePendingReworkRequests(&report)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].requester != "middle" || pending[0].target != "leaf" || pending[0].request != request {
		t.Fatalf("nested request was not propagated through its direct-child edge: %+v", pending)
	}
	if len(report.Tasks[1].ReworkRequests) != 0 {
		t.Fatalf("consumed request remained queued: %+v", report.Tasks[1].ReworkRequests)
	}
}

func TestResumeFailsClosedAtPersistedManagerReworkMarker(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
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
	rootID := e2eManagerID("", "project-owner")
	ordersID := e2eManagerID("orders", "orders")
	report := RunReport{APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Status: StatusInterrupted, Mode: ModeControlledLocal,
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot,
		ModelDigest: plan.ModelDigest, RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers),
		Candidate: CandidateRef{ID: initial.ID, Snapshot: initial.Digest, Files: map[string]string{}}, Revision: 1,
		ManagerReworkRounds: []ManagerReworkRound{{Number: 1, Status: "running", Requests: []ManagerReworkRequestRecord{{
			Requester: rootID, Request: ReworkRequest{ManagerID: ordersID, Goal: "Correct the orders artifact.", Reason: "Persisted targeted rework remains open."}, Status: "invoking",
		}}}}}
	if err := store.appendState(report); err != nil {
		t.Fatal(err)
	}
	resumed, err := Resume(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err == nil || resumed.Status != StatusBlocked || len(resumed.Invocations) != 0 {
		t.Fatalf("Resume crossed an unfinished persisted rework boundary: status=%s invocations=%d err=%v", resumed.Status, len(resumed.Invocations), err)
	}
	if len(resumed.Findings) == 0 || !strings.Contains(resumed.Findings[len(resumed.Findings)-1], "manager-directed rework round") {
		t.Fatalf("blocked rework boundary was not explained: %v", resumed.Findings)
	}
}
