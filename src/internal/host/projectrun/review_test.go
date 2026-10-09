package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestReviewFindingRequiresExactCandidatePathAndAcceptedGrounding(t *testing.T) {
	model := projectmodel.ManagerContext{Statements: []projectmodel.Statement{{ID: "accepted-statement"}}, Artifacts: []projectmodel.Artifact{{Paths: []string{"src/orders/result.txt"}}}}
	files := []agentexec.Artifact{{Path: "src/orders/result.txt"}}
	refs := reviewFileReferences(projectmodel.Report{}, model, files)
	valid := []reviewFindingResponse{{Path: "src/orders/result.txt", Expectation: "write the declared output", Grounding: "statement:accepted-statement"}}
	if _, err := validateReviewFindings(valid, refs); err != nil {
		t.Fatalf("accepted grounded finding rejected: %v", err)
	}
	for name, finding := range map[string]reviewFindingResponse{
		"foreign path":          {Path: "src/inventory/result.txt", Expectation: "change the sibling", Grounding: "statement:accepted-statement"},
		"unknown statement":     {Path: "src/orders/result.txt", Expectation: "invent a requirement", Grounding: "statement:unknown"},
		"unknown artifact path": {Path: "src/orders/result.txt", Expectation: "invent an artifact", Grounding: "artifact-path:src/other.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateReviewFindings([]reviewFindingResponse{finding}, refs); err == nil {
				t.Fatal("ungrounded or out-of-scope review finding was accepted")
			}
		})
	}
	model.Artifacts[0].Paths = append(model.Artifacts[0].Paths, "src/orders/other.txt")
	refs = reviewFileReferences(projectmodel.Report{}, model, files)
	if _, err := validateReviewFindings([]reviewFindingResponse{{Path: "src/orders/result.txt", Expectation: "cite the matching artifact", Grounding: "artifact-path:src/orders/other.txt"}}, refs); err == nil {
		t.Fatal("unrelated accepted artifact path grounded a finding for another candidate path")
	}
}

func TestScopedReviewModelIncludesOnlyRelatedPublicForeignInterfaces(t *testing.T) {
	const (
		salesID     = "shop/sales"
		ordersID    = "shop/orders"
		financeID   = "shop/finance"
		path        = "src/shop/commerce/cancellation.py"
		financePath = "src/shop/commerce/finance_test.py"
	)
	report := projectmodel.Report{
		Managers: []projectmodel.Manager{
			{ID: salesID}, {ID: ordersID, Parent: salesID}, {ID: financeID, Parent: salesID},
		},
		Statements: []projectmodel.Statement{
			{ID: "order-cancellation-contract", Owner: ordersID, Public: true, Description: "Cancellation interface"},
			{ID: "orders-private-detail", Owner: ordersID, Public: false, Description: "Private implementation"},
			{ID: "finance-ledger-contract", Owner: financeID, Public: true, Description: "Finance test contract"},
			{ID: "finance-unrelated-contract", Owner: financeID, Public: true, Description: "Unrelated finance contract"},
		},
		Artifacts: []projectmodel.Artifact{
			{ID: "order-lifecycle", Name: "Order lifecycle", Owner: ordersID, Role: "implementation", Required: true,
				Paths: []string{path}, Realizes: []string{"order-cancellation-contract", "orders-private-detail"}},
			{ID: "finance-ledger", Name: "Finance ledger", Owner: financeID, Required: true,
				Paths: []string{financePath}, Realizes: []string{"finance-ledger-contract"}},
			{ID: "finance-private-notes", Name: "Finance private notes", Owner: financeID,
				Paths: []string{"src/shop/finance/private-notes.md"}},
		},
		Files: []projectmodel.FileEntry{
			{Path: path, Owner: salesID, Class: "source", Statements: []string{"order-cancellation-contract", "orders-private-detail"}, Artifacts: []string{"order-lifecycle"}},
			{Path: financePath, Owner: salesID, Class: "test", Statements: []string{"finance-ledger-contract"}, Artifacts: []string{"finance-ledger"}},
		},
	}
	tasks := []ManagerTask{{ManagerID: salesID}, {ManagerID: ordersID, ParentTask: salesID}}
	files := []agentexec.Artifact{{Path: path, Mode: "0644", Content: []byte("cancel order")}, {Path: financePath, Mode: "0644", Content: []byte("finance test")}}
	accepted, err := scopedReviewModel(report, salesID, tasks, files)
	if err != nil {
		t.Fatal(err)
	}
	contracts := map[string]projectmodel.Statement{}
	for _, statement := range accepted.Contracts {
		contracts[statement.ID] = statement
	}
	if _, ok := contracts["order-cancellation-contract"]; !ok {
		t.Fatalf("public contract related to the admitted file is missing: %+v", accepted.Contracts)
	}
	if _, ok := contracts["finance-ledger-contract"]; !ok {
		t.Fatalf("public contract explicitly related to the second admitted file is missing: %+v", accepted.Contracts)
	}
	for _, excluded := range []string{"orders-private-detail", "finance-unrelated-contract"} {
		if _, ok := contracts[excluded]; ok {
			t.Fatalf("private or unrelated contract %q leaked into review model", excluded)
		}
	}
	artifacts := map[string]projectmodel.Artifact{}
	for _, artifact := range accepted.Artifacts {
		artifacts[artifact.ID] = artifact
	}
	if _, ok := artifacts["order-lifecycle"]; !ok {
		t.Fatalf("required related artifact interface is missing: %+v", accepted.Artifacts)
	}
	if _, ok := artifacts["finance-ledger"]; !ok {
		t.Fatal("foreign artifact explicitly related to the second candidate file is missing")
	}
	if _, ok := artifacts["finance-private-notes"]; ok {
		t.Fatal("unrelated foreign artifact leaked into review model")
	}
	refs := reviewFileReferences(report, accepted, files)
	if !containsString(refs[0].Grounding, "statement:order-cancellation-contract") {
		t.Fatal("explicitly supplied contract cannot ground a finding")
	}
	if _, err := validateReviewFindings([]reviewFindingResponse{{Path: path, Expectation: "Preserve cancellation behavior.", Grounding: "statement:order-cancellation-contract"}}, refs); err != nil {
		t.Fatalf("finding grounded in supplied contract rejected: %v", err)
	}
	if _, err := validateReviewFindings([]reviewFindingResponse{{Path: path, Expectation: "Preserve cancellation behavior.", Grounding: "statement:finance-ledger-contract"}}, refs); err == nil {
		t.Fatal("contract related only to the other candidate file grounded this finding")
	}
	if _, err := validateReviewFindings([]reviewFindingResponse{{Path: financePath, Expectation: "Preserve the test obligation.", Grounding: "statement:finance-ledger-contract"}}, refs); err != nil {
		t.Fatalf("file-specific public contract rejected for its related candidate: %v", err)
	}
	ids := reviewScopeIDs(salesID, accepted)
	if !containsString(ids, "statement:order-cancellation-contract") || !containsString(ids, "statement:finance-ledger-contract") || !containsString(ids, "artifact:order-lifecycle") || containsString(ids, "statement:finance-unrelated-contract") {
		t.Fatalf("review request scope IDs do not match accepted interface context: %v", ids)
	}
}

func TestReviewerScopeExcludesForeignArtifactBytesButRetainsInterfaceAndFreshness(t *testing.T) {
	const (
		ordersID      = "shop/orders"
		engineeringID = "shop/engineering"
		orderPath     = "src/shop/orders/cancellation.py"
		engineerPath  = "tests/test_cancellation.py"
	)
	project := &Project{
		Config: projectwork.Config{InventoryRoots: []string{"src", "tests"}},
		Snapshot: &snapshot.Snapshot{
			Files: map[string][]byte{orderPath: []byte("orders implementation"), engineerPath: []byte("engineering test")},
			Modes: map[string]string{orderPath: snapshot.RegularMode, engineerPath: snapshot.RegularMode},
		},
		Report: projectmodel.Report{
			Managers:   []projectmodel.Manager{{ID: ordersID}, {ID: engineeringID}},
			Statements: []projectmodel.Statement{{ID: "order-lifecycle-contract", Owner: ordersID, Public: true}},
			Artifacts: []projectmodel.Artifact{{ID: "orders-lifecycle", Name: "Orders lifecycle", Owner: ordersID, Role: "implementation", Required: true,
				Paths: []string{orderPath, engineerPath}, Realizes: []string{"order-lifecycle-contract"}}},
			Files: []projectmodel.FileEntry{
				{Path: orderPath, Owner: ordersID, Class: "source", Statements: []string{"order-lifecycle-contract"}, Artifacts: []string{"orders-lifecycle"}},
				{Path: engineerPath, Owner: engineeringID, Class: "test", Artifacts: []string{"orders-lifecycle"}},
			},
		},
	}
	ordersTask := ManagerTask{ManagerID: ordersID, Goal: "Implement order cancellation.", Artifacts: []string{"orders-lifecycle"}}
	engineeringTask := ManagerTask{ManagerID: engineeringID, Goal: "Test order cancellation."}
	ordersFiles := scopedCandidateFiles(project, ordersTask)
	if len(ordersFiles) != 1 || ordersFiles[0].Path != orderPath {
		t.Fatalf("Orders reviewer received foreign Engineering bytes: %+v", ordersFiles)
	}
	accepted, err := scopedReviewModel(project.Report, ordersID, []ManagerTask{ordersTask, engineeringTask}, ordersFiles)
	if err != nil {
		t.Fatal(err)
	}
	if len(accepted.Artifacts) != 1 || accepted.Artifacts[0].ID != "orders-lifecycle" || !containsString(accepted.Artifacts[0].Paths, engineerPath) {
		t.Fatalf("foreign implementation filtering discarded the relevant artifact interface: %+v", accepted.Artifacts)
	}
	if !containsString(reviewFileReferences(project.Report, accepted, ordersFiles)[0].Grounding, "statement:order-lifecycle-contract") {
		t.Fatal("Orders reviewer lost the public contract realized by its owned file")
	}
	engineeringFiles := scopedCandidateFiles(project, engineeringTask)
	if len(engineeringFiles) != 1 || engineeringFiles[0].Path != engineerPath {
		t.Fatalf("Engineering reviewer did not receive its own changed test bytes: %+v", engineeringFiles)
	}
	plan := PlanRecord{Goal: "Implement and test order cancellation.", Managers: []ManagerTask{ordersTask, engineeringTask}}
	initialDigest, err := reviewScopeDigest(plan, project, engineeringTask, "work")
	if err != nil {
		t.Fatal(err)
	}
	changed := *project
	changed.Snapshot = &snapshot.Snapshot{ID: project.Snapshot.ID, Provisional: project.Snapshot.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, content := range project.Snapshot.Files {
		changed.Snapshot.Files[path] = append([]byte(nil), content...)
		changed.Snapshot.Modes[path] = project.Snapshot.Modes[path]
	}
	changed.Snapshot.Files[engineerPath] = []byte("engineering test changed")
	changedDigest, err := reviewScopeDigest(plan, &changed, engineeringTask, "work")
	if err != nil || changedDigest == initialDigest {
		t.Fatalf("final Engineering review scope did not bind changed test bytes: initial=%s changed=%s err=%v", initialDigest, changedDigest, err)
	}
	ordersTask.IntegratedPaths = []string{engineerPath}
	integratedFiles := scopedCandidateFiles(project, ordersTask)
	if len(integratedFiles) != 2 || integratedFiles[1].Path != engineerPath {
		t.Fatal("explicit parent integration edit was dropped from its review scope")
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
