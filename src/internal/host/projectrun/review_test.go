package projectrun

import (
	"context"
	"encoding/json"
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

func TestReviewerNormalizesEmptyArtifactForReceiptBoundReport(t *testing.T) {
	root := makeProjectRunFixture(t)
	host := projectworkHost()
	managerID := e2eManagerID("orders", "orders")
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		agents := map[string]Agent{}
		for id, agent := range runtime.Agents {
			agents[id] = agent
		}
		runtime.Review = &ReviewConfig{Agents: agents, MaxRounds: 2, MaxManagerRounds: 2}
	})
	revision := identityHead(t, root)
	plan, err := Plan(host, root, revision, PlanRequest{Goal: "Review the empty artifact.", Managers: []string{managerID}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("plan review fixture: %v", err)
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	const path = "src/orders/implementation.txt"
	candidate := candidateData{APIVersion: APIVersion, ID: "empty-artifact-candidate", Files: map[string]File{
		path: {Path: path, Mode: "0644", Content: nil},
	}}
	candidate.Digest = candidateSnapshotHash(base.Snapshot, candidate)
	project, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	task := findTask(plan.Managers, managerID)
	if task == nil {
		t.Fatal("review fixture did not plan the selected Manager")
	}
	invoker := &normalizedReviewInvoker{}
	review, log, err := invokeReviewer(context.Background(), host, invoker, root, plan, runtime, project, *task, "work", 1, candidate, nil, RunReport{})
	if err != nil {
		t.Fatalf("valid response for empty candidate artifact was rejected: %v", err)
	}
	if len(invoker.request.Artifacts) != 1 || len(invoker.request.Artifacts[0].Content) != 0 {
		t.Fatalf("review request did not contain the empty candidate artifact: %+v", invoker.request.Artifacts)
	}
	rawDigest, err := digest(invoker.request)
	if err != nil {
		t.Fatal(err)
	}
	if log.InputDigest != rawDigest || log.InputDigest == review.Receipt.InputDigest {
		t.Fatalf("ledger did not preserve its raw digest separately from the protocol receipt: log=%s raw=%s receipt=%s", log.InputDigest, rawDigest, review.Receipt.InputDigest)
	}
	if review.InputDigest != review.Receipt.InputDigest {
		t.Fatalf("review record is not bound to its normalized receipt: record=%s receipt=%s", review.InputDigest, review.Receipt.InputDigest)
	}
	response := agentexec.Response{Role: agentexec.RoleExecutor, InputDigest: review.InputDigest, Outcome: agentexec.OutcomeProposed,
		ReportJSON: json.RawMessage(`{"status":"pass","summary":"The empty artifact satisfies the scoped contract.","findings":[]}`)}
	badReceipt := review.Receipt
	badReceipt.InputDigest = log.InputDigest
	if _, err := validateReviewerResponse(response, badReceipt, review.InputDigest); err == nil {
		t.Fatal("mismatched raw receipt digest was accepted for the normalized review response")
	}
}

type normalizedReviewInvoker struct{ request agentexec.Request }

func (i *normalizedReviewInvoker) Run(_ context.Context, _ agentexec.Config, request agentexec.Request, _ agentexec.RunOptions) (agentexec.RunResult, error) {
	i.request = request
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	zero := int64(0)
	return agentexec.RunResult{
		Response: agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce, Role: agentexec.RoleExecutor,
			InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{},
			VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}, ReportJSON: json.RawMessage(`{"status":"pass","summary":"The empty artifact satisfies the scoped contract.","findings":[]}`)},
		Receipt: agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest,
			Outcome: agentexec.OutcomeProposed, Usage: &agentexec.Usage{InputTokens: &zero, OutputTokens: &zero}},
	}, nil
}

func (*normalizedReviewInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return agentexec.Fingerprint(config)
}

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
	accepted, err := scopedReviewModel(report, salesID, tasks, files, "work")
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
	accepted, err := scopedReviewModel(project.Report, ordersID, []ManagerTask{ordersTask, engineeringTask}, ordersFiles, "work")
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
	initialDigest, err := reviewScopeDigest(plan, project, engineeringTask, "work", RunReport{})
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
	changedDigest, err := reviewScopeDigest(plan, &changed, engineeringTask, "work", RunReport{})
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
	initial, err := reviewScopeDigest(plan, base, task, "work", RunReport{})
	if err != nil {
		t.Fatal(err)
	}
	siblingChange := candidateData{Files: map[string]File{"src/inventory/implementation.txt": {Path: "src/inventory/implementation.txt", Mode: "0644", Content: []byte("inventory changed")}}}
	mergedSibling, err := projectForCandidate(projectworkHost(), root, base.Snapshot, siblingChange)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := reviewScopeDigest(plan, mergedSibling, task, "work", RunReport{})
	if err != nil || unchanged != initial {
		t.Fatalf("unaffected review scope changed after sibling merge: got=%s want=%s err=%v", unchanged, initial, err)
	}
	ownedChange := candidateData{Files: map[string]File{"src/orders/implementation.txt": {Path: "src/orders/implementation.txt", Mode: "0644", Content: []byte("orders changed")}}}
	mergedOwned, err := projectForCandidate(projectworkHost(), root, base.Snapshot, ownedChange)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := reviewScopeDigest(plan, mergedOwned, task, "work", RunReport{})
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
	parentInitial, err := reviewScopeDigest(plan, base, parent, "integrate", RunReport{})
	if err != nil {
		t.Fatal(err)
	}
	parentAfterSibling, err := reviewScopeDigest(plan, mergedSibling, parent, "integrate", RunReport{})
	if err != nil || parentAfterSibling != parentInitial {
		t.Fatalf("parent review scope changed with independent child bytes: got=%s want=%s err=%v", parentAfterSibling, parentInitial, err)
	}
}

func TestReviewerContextSeparatesWorkDelegationFromIntegrationDelivery(t *testing.T) {
	const (
		rootID  = "project/root"
		childID = "project/greeting"
		readme  = "README.md"
		source  = "src/greeting.py"
		tests   = "tests/test_greeting.py"
		docs    = "docs/greeting.md"
	)
	project := &Project{
		Config: projectwork.Config{CoverageMode: "full", InventoryRoots: []string{"src", "tests", "docs"}},
		Snapshot: &snapshot.Snapshot{
			Files: map[string][]byte{readme: []byte("Project README"), source: []byte("def greet(name): return name"), tests: []byte("assert greet('Ada')"), docs: []byte("Use greet(name).")},
			Modes: map[string]string{readme: snapshot.RegularMode, source: snapshot.RegularMode, tests: snapshot.RegularMode, docs: snapshot.RegularMode},
		},
		Report: projectmodel.Report{
			Managers: []projectmodel.Manager{{ID: rootID}, {ID: childID, Parent: rootID}},
			Artifacts: []projectmodel.Artifact{
				{ID: "readme-artifact", Owner: rootID, Required: true, Paths: []string{readme}},
				{ID: "greeting-source", Owner: childID, Required: true, Paths: []string{source}},
				{ID: "greeting-tests", Owner: childID, Required: true, Paths: []string{tests}},
				{ID: "greeting-guide", Owner: childID, Required: true, Paths: []string{docs}},
			},
			Files: []projectmodel.FileEntry{
				{Path: readme, Owner: rootID, Class: "documentation", Artifacts: []string{"readme-artifact"}},
				{Path: source, Owner: childID, Class: "source", Artifacts: []string{"greeting-source"}},
				{Path: tests, Owner: childID, Class: "test", Artifacts: []string{"greeting-tests"}},
				{Path: docs, Owner: childID, Class: "documentation", Artifacts: []string{"greeting-guide"}},
			},
		},
	}
	plannedRoot := ManagerTask{ID: "root-task", ManagerID: rootID, Goal: "Coordinate the greeting feature."}
	plannedChild := ManagerTask{ID: "child-task", ManagerID: childID, ParentTask: rootID, Goal: "Implement greeting behavior."}
	plan := PlanRecord{Goal: "Implement greeting and provide its usage guide.", Managers: []ManagerTask{plannedRoot, plannedChild}}
	// Delivery facts exist only on the run report's tasks, never on the plan.
	rootTask := plannedRoot
	rootTask.WrittenPaths, rootTask.ReportID, rootTask.IntegrationReportID = []string{readme}, "root-work-run", "root-integrate-run"
	rootTask.Delegations = []Delegation{{ManagerID: childID, Goal: "Implement greeting behavior and document it."}}
	childTask := plannedChild
	childTask.WrittenPaths, childTask.ReportID, childTask.State = []string{source, tests, docs}, "child-work-run", "integrated"
	run := RunReport{Tasks: []ManagerTask{rootTask, childTask}}
	candidate := candidateData{ID: "candidate-1", Digest: "digest-1"}

	work, workFiles, _, err := buildReviewerContext(plan, project, rootTask, "work", 1, candidate, BriefingContext{}, RunReport{})
	if err != nil {
		t.Fatal(err)
	}
	if got := reviewScopePaths(workFiles); len(got) != 1 || got[0] != readme {
		t.Fatalf("work review should receive only current root delivery: %v", got)
	}
	workReport := RunReport{RoleStartReservations: []RoleStartReservation{{Key: "helper-work", Kind: "helper", ManagerID: rootID, Phase: "work", ParentRunID: rootTask.ReportID,
		Request:        agentexec.RoleStartRequest{RequestID: "helper-work-request", Role: "helper", ParentSessionID: "parent-session", SessionID: "child-session", State: "completed"},
		HelperDelivery: &HelperDelivery{State: "applied-and-closed", RequestID: "helper-work-request", Task: "write scoped docs", RequestedPaths: []string{readme}, DeltaDigest: "sha256:" + strings.Repeat("d", 64), Changes: []HelperDeliveryChange{{Kind: "modify", Path: readme, Mode: workFiles[0].Mode, ContentDigest: workFiles[0].Digest}}}}}}
	workWithHelper, _, _, err := buildReviewerContext(plan, project, rootTask, "work", 1, candidate, BriefingContext{}, workReport)
	if err != nil || len(workWithHelper.HostHelperResults) != 1 || !workWithHelper.HostHelperResults[0].Changes[0].PresentInCandidate {
		t.Fatalf("current work helper delivery was not bound to candidate: evidence=%+v err=%v", workWithHelper.HostHelperResults, err)
	}
	withoutHelperDigest, err := reviewScopeDigest(plan, project, rootTask, "work", RunReport{})
	if err != nil {
		t.Fatal(err)
	}
	withHelperDigest, err := reviewScopeDigest(plan, project, rootTask, "work", workReport)
	if err != nil {
		t.Fatal(err)
	}
	if withoutHelperDigest == withHelperDigest {
		t.Fatal("helper delivery facts did not invalidate the review scope digest")
	}
	wrongParent := workReport
	wrongParent.RoleStartReservations = append([]RoleStartReservation(nil), workReport.RoleStartReservations...)
	wrongParent.RoleStartReservations[0].ParentRunID = "repair-history-only"
	ignored, _, _, err := buildReviewerContext(plan, project, rootTask, "work", 1, candidate, BriefingContext{}, wrongParent)
	if err != nil || len(ignored.HostHelperResults) != 0 {
		t.Fatalf("repair-history helper leaked into current review: %+v err=%v", ignored.HostHelperResults, err)
	}
	unresolved := workReport
	unresolved.RoleStartReservations = append([]RoleStartReservation(nil), workReport.RoleStartReservations...)
	unresolved.RoleStartReservations[0].Request.State = "unknown"
	if _, _, _, err := buildReviewerContext(plan, project, rootTask, "work", 1, candidate, BriefingContext{}, unresolved); err == nil {
		t.Fatal("review context hid an unresolved current helper request")
	}
	malformedRename := workReport
	malformedRename.RoleStartReservations = append([]RoleStartReservation(nil), workReport.RoleStartReservations...)
	malformedRename.RoleStartReservations[0].HelperDelivery = cloneHelperDeliveryPtr(workReport.RoleStartReservations[0].HelperDelivery)
	malformedRename.RoleStartReservations[0].HelperDelivery.Changes[0].Kind = "rename"
	malformedRename.RoleStartReservations[0].HelperDelivery.Changes[0].OldPath = ""
	if _, _, _, err := buildReviewerContext(plan, project, rootTask, "work", 1, candidate, BriefingContext{}, malformedRename); err == nil {
		t.Fatal("review context accepted rename evidence without its source path")
	}
	if !hasProjectArtifact(work.AcceptedModel.Artifacts, "readme-artifact") || hasProjectArtifact(work.AcceptedModel.Artifacts, "greeting-source") || hasProjectArtifact(work.AcceptedModel.Artifacts, "greeting-tests") || hasProjectArtifact(work.AcceptedModel.Artifacts, "greeting-guide") {
		t.Fatalf("work review accepted model mixes in future child obligations: %+v", work.AcceptedModel.Artifacts)
	}
	for _, id := range []string{"greeting-source", "greeting-tests", "greeting-guide"} {
		if !hasProjectArtifact(work.DelegatedArtifacts, id) {
			t.Fatalf("work review lost child artifact %q from delegation context: %+v", id, work.DelegatedArtifacts)
		}
	}

	integration, integrationFiles, _, err := buildReviewerContext(plan, project, rootTask, "integrate", 1, candidate, BriefingContext{}, run)
	if err != nil {
		t.Fatal(err)
	}
	if len(integration.DelegatedArtifacts) != 0 {
		t.Fatalf("integration review should treat child outputs as aggregate obligations, not future delegation artifacts: %+v", integration.DelegatedArtifacts)
	}
	childHelper := RunReport{Tasks: run.Tasks, RoleStartReservations: []RoleStartReservation{{Key: "child-helper", Kind: "helper", ManagerID: childID, Phase: "work", ParentRunID: childTask.ReportID,
		Request:        agentexec.RoleStartRequest{RequestID: "child-helper-request", Role: "helper", ParentSessionID: "child-parent-session", SessionID: "child-helper-session", State: "completed"},
		HelperDelivery: &HelperDelivery{State: "applied-and-closed", RequestID: "child-helper-request", Task: "write greeting tests", RequestedPaths: []string{tests + "/"}, DeltaDigest: "sha256:" + strings.Repeat("e", 64), Changes: []HelperDeliveryChange{{Kind: "modify", Path: tests + "/greeting_test.go", Mode: "0644", ContentDigest: "sha256:" + strings.Repeat("f", 64)}}}}}}
	childHelper.RoleStartReservations = append(childHelper.RoleStartReservations, workReport.RoleStartReservations...)
	integrationWithHelper, _, _, err := buildReviewerContext(plan, project, rootTask, "integrate", 1, candidate, BriefingContext{}, childHelper)
	owners := map[string]bool{}
	for _, evidence := range integrationWithHelper.HostHelperResults {
		owners[evidence.ManagerID] = true
	}
	if err != nil || len(integrationWithHelper.HostHelperResults) != 2 || !owners[childID] || !owners[rootID] {
		t.Fatalf("aggregate review omitted active child helper delivery: evidence=%+v err=%v", integrationWithHelper.HostHelperResults, err)
	}
	gotPaths := reviewScopePaths(integrationFiles)
	for _, path := range []string{readme, source, tests, docs} {
		if !containsString(gotPaths, path) {
			t.Fatalf("integration review omitted delivered aggregate path %q: %v", path, gotPaths)
		}
	}
	for _, id := range []string{"greeting-source", "greeting-tests", "greeting-guide"} {
		if !hasProjectArtifact(integration.AcceptedModel.Artifacts, id) {
			t.Fatalf("integration review omitted required child artifact %q: %+v", id, integration.AcceptedModel.Artifacts)
		}
	}
	if got := integration.ScopedModel.OwnedPaths; len(got) != len(gotPaths) {
		t.Fatalf("integration scoped model paths do not bind aggregate candidate: scoped=%v files=%v", got, gotPaths)
	}

	workDigest, err := reviewScopeDigest(plan, project, rootTask, "work", run)
	if err != nil {
		t.Fatal(err)
	}
	integrationDigest, err := reviewScopeDigest(plan, project, rootTask, "integrate", run)
	if err != nil {
		t.Fatal(err)
	}
	if workDigest == integrationDigest {
		t.Fatal("phase-specific review obligations and candidate scope did not change the scope digest")
	}
}

func TestIntegrationReviewBindsChildDeliveredFileOutsideRequiredArtifacts(t *testing.T) {
	const (
		rootID  = "project/root"
		childID = "project/orders"
		impl    = "src/orders/implementation.txt"
		notes   = "src/orders/notes.txt"
	)
	newProject := func(notesBytes string) *Project {
		return &Project{
			Config: projectwork.Config{InventoryRoots: []string{"src"}},
			Snapshot: &snapshot.Snapshot{
				Files: map[string][]byte{impl: []byte("orders implementation v2\n"), notes: []byte(notesBytes)},
				Modes: map[string]string{impl: snapshot.RegularMode, notes: snapshot.RegularMode},
			},
			Report: projectmodel.Report{
				Managers:  []projectmodel.Manager{{ID: rootID}, {ID: childID, Parent: rootID}},
				Artifacts: []projectmodel.Artifact{{ID: "orders-code", Owner: childID, Required: true, Paths: []string{impl}}},
				Files: []projectmodel.FileEntry{
					{Path: impl, Owner: childID, Class: "source", Artifacts: []string{"orders-code"}},
					{Path: notes, Owner: childID, Class: "documentation"},
				},
			},
		}
	}
	// Planned tasks never carry WrittenPaths/IntegratedPaths; only the run
	// report's tasks do.
	plannedRoot := ManagerTask{ID: "root-task", ManagerID: rootID, Goal: "Integrate orders."}
	plannedChild := ManagerTask{ID: "child-task", ManagerID: childID, ParentTask: rootID, Goal: "Implement orders."}
	plan := PlanRecord{Goal: "Implement orders.", Managers: []ManagerTask{plannedRoot, plannedChild}}
	runRoot := plannedRoot
	runRoot.State, runRoot.IntegratedPaths = "integrated", []string{}
	runChild := plannedChild
	runChild.State, runChild.WrittenPaths = "worked", []string{impl, notes}
	report := RunReport{Tasks: []ManagerTask{runRoot, runChild}}
	candidate := candidateData{ID: "candidate-1", Digest: "digest-1"}

	project := newProject("orders notes v1\n")
	reviewContext, files, _, err := buildReviewerContext(plan, project, runRoot, "integrate", 1, candidate, BriefingContext{}, report)
	if err != nil {
		t.Fatal(err)
	}
	if paths := reviewScopePaths(files); !containsString(paths, notes) {
		t.Errorf("integration review input omitted child-delivered file %s: %v", notes, paths)
	}
	if !containsString(reviewContext.ChangedPaths, notes) {
		t.Errorf("integration review changed paths omitted child-delivered file %s: %v", notes, reviewContext.ChangedPaths)
	}
	before, err := reviewScopeDigest(plan, project, runRoot, "integrate", report)
	if err != nil {
		t.Fatal(err)
	}
	after, err := reviewScopeDigest(plan, newProject("orders notes CHANGED\n"), runRoot, "integrate", report)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Errorf("integration review scope digest did not change when child-delivered %s changed", notes)
	}
}

func TestIntegrationReviewExpandsRequiredChildDirectoryArtifact(t *testing.T) {
	const (
		rootID  = "project/root"
		childID = "project/orders"
		file    = "src/orders/api.txt"
	)
	project := &Project{
		Config: projectwork.Config{InventoryRoots: []string{"src"}},
		Snapshot: &snapshot.Snapshot{
			Files: map[string][]byte{file: []byte("orders api v2\n")},
			Modes: map[string]string{file: snapshot.RegularMode},
		},
		Report: projectmodel.Report{
			Managers:  []projectmodel.Manager{{ID: rootID}, {ID: childID, Parent: rootID}},
			Artifacts: []projectmodel.Artifact{{ID: "orders-dir", Owner: childID, Required: true, Paths: []string{"src/orders/"}}},
			Files:     []projectmodel.FileEntry{{Path: file, Owner: childID, Class: "source", Artifacts: []string{"orders-dir"}}},
		},
	}
	rootTask := ManagerTask{ID: "root-task", ManagerID: rootID, Goal: "Integrate orders."}
	childTask := ManagerTask{ID: "child-task", ManagerID: childID, ParentTask: rootID, Goal: "Implement orders."}
	plan := PlanRecord{Goal: "Implement orders.", Managers: []ManagerTask{rootTask, childTask}}
	managerInput, err := scopedArtifacts(project, rootTask, Agent{}, Limits{MaxCandidateFileBytes: 4096, MaxCandidateBytes: 16384}, "integrate", []string{childID})
	if err != nil {
		t.Fatal(err)
	}
	managerPaths := reviewScopePaths(managerInput)
	if !containsString(managerPaths, file) {
		t.Fatalf("Manager integration input did not expand required directory artifact src/orders/: %v", managerPaths)
	}
	_, files, _, err := buildReviewerContext(plan, project, rootTask, "integrate", 1, candidateData{ID: "c", Digest: "d"}, BriefingContext{}, RunReport{Tasks: []ManagerTask{rootTask, childTask}})
	if err != nil {
		t.Fatal(err)
	}
	if reviewPaths := reviewScopePaths(files); !containsString(reviewPaths, file) {
		t.Errorf("integration review omitted file %s of required directory artifact src/orders/ (Manager input had %v): %v", file, managerPaths, reviewPaths)
	}
}

func hasProjectArtifact(artifacts []projectmodel.Artifact, id string) bool {
	for _, artifact := range artifacts {
		if artifact.ID == id {
			return true
		}
	}
	return false
}

func cloneHelperDeliveryPtr(delivery *HelperDelivery) *HelperDelivery {
	if delivery == nil {
		return nil
	}
	clone := cloneHelperDelivery(*delivery)
	return &clone
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
