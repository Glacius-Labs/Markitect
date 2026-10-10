package projectrun

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
)

func TestReviewerContextCarriesOperationStrictnessAndAcceptedBriefing(t *testing.T) {
	root, revision, project, managerID, eventID := acceptedBriefingFixture(t)
	beforeDismissal, err := managerBriefing(root, project.Report.ModelDigest, managerID, revision)
	if err != nil {
		t.Fatal(err)
	}
	_, storeDigest, err := projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = projectbriefing.Dismiss(root, eventID, managerID, storeDigest)
	if err != nil {
		t.Fatal(err)
	}
	plan := PlanRecord{
		Operation: OperationCleanup, Goal: "Improve order cancellation safely.",
		Strictness:      map[string]StrictnessProfile{managerID: {Evidence: []string{"tests", "public interface"}, Counterexamples: 2}},
		BriefingDigests: map[string]string{managerID: beforeDismissal.Digest},
	}
	briefing, err := reviewerBriefing(root, plan, project, managerID)
	if err != nil {
		t.Fatal(err)
	}
	if briefing.Digest != beforeDismissal.Digest || len(briefing.Briefings) != 1 || len(briefing.Events) != 1 || briefing.Events[0].ID != eventID {
		t.Fatalf("dismissal hid or changed accepted reviewer context: before=%+v after=%+v", beforeDismissal, briefing)
	}

	context := reviewerContext{
		Kind: "projectrun-review/v1", Operation: plan.Operation, ReviewerGuidance: reviewerAssessmentGuidance,
		Strictness: plan.Strictness[managerID], Briefing: briefing, ManagerID: managerID,
		Delegations: []Delegation{{ManagerID: "child-tests", Goal: "Add focused behavior tests."}},
	}
	encoded, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	guidance, _ := got["reviewerGuidance"].(string)
	if got["operation"] != OperationCleanup || !strings.Contains(guidance, "Assessment only") || strings.Contains(guidance, "Apply mandate") || got["operationGuidance"] != nil || got["strictness"] == nil || got["briefing"] == nil {
		t.Fatalf("reviewer request omitted assessment guidance, operation, strictness, or accepted briefing: %s", encoded)
	}
	if delegations, ok := got["delegations"].([]any); !ok || len(delegations) != 1 || delegations[0].(map[string]any)["managerId"] != "child-tests" {
		t.Fatalf("reviewer context omitted the Manager's delegation: %s", encoded)
	}
	if got["kind"] != "projectrun-review/v1" {
		t.Fatalf("reviewer discriminator changed: %v", got["kind"])
	}

	plan.BriefingDigests[managerID] = "sha256:wrong"
	if _, err := reviewerBriefing(root, plan, project, managerID); !errors.Is(err, ErrStale) {
		t.Fatalf("review accepted a briefing digest that differs from its plan: %v", err)
	}
}

func TestDraftModelEditReviewerDoesNotLoadUnacceptedBriefingHistory(t *testing.T) {
	root, acceptedRevision, _, managerID, _ := acceptedBriefingFixture(t)
	const statementPath = ".markitect/model/orders/statement.yaml"
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(statementPath)))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(content), "audited invariants", "reconciled invariants", 1)
	if updated == string(content) {
		t.Fatal("could not create a second, unbriefed model revision")
	}
	writeE2E(t, root, statementPath, updated)
	gitE2E(t, root, "add", statementPath)
	gitE2E(t, root, "commit", "-m", "unbriefed follow-up model change")
	targetRevision := gitE2E(t, root, "rev-parse", "HEAD")
	project, err := projectworkHost().Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := managerBriefing(root, project.Report.ModelDigest, managerID, targetRevision); err == nil {
		t.Fatal("fixture unexpectedly has accepted briefing history for the unaccepted target model")
	}
	if acceptedRevision == targetRevision {
		t.Fatal("draft fixture did not create a distinct source revision")
	}
	plan := PlanRecord{ModelEdit: &EditPlan{}, BriefingDigests: map[string]string{managerID: ""}}
	briefing, err := reviewerBriefing(root, plan, project, managerID)
	if err != nil {
		t.Fatalf("draft reviewer tried to load unaccepted target-model history: %v", err)
	}
	if briefing.Digest != "" || len(briefing.Briefings) != 0 || len(briefing.Events) != 0 {
		t.Fatalf("draft reviewer inherited accepted history: %+v", briefing)
	}
	plan.BriefingDigests[managerID] = "unexpected-accepted-binding"
	if _, err := reviewerBriefing(root, plan, project, managerID); !errors.Is(err, ErrStale) {
		t.Fatalf("draft reviewer accepted an accepted-history binding: %v", err)
	}
}

func acceptedBriefingFixture(t *testing.T) (root, revision string, project *Project, managerID, eventID string) {
	t.Helper()
	root = makeFullVerifyFixture(t)
	const statementPath = ".markitect/model/orders/statement.yaml"
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(statementPath)))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(content), "Implement the orders artifact.", "Implement the orders artifact with audited invariants.", 1)
	if updated == string(content) {
		t.Fatal("Orders statement fixture text was not found")
	}
	writeE2E(t, root, statementPath, updated)
	gitE2E(t, root, "add", statementPath)
	gitE2E(t, root, "commit", "-m", "accept Orders model clarification")
	revision = gitE2E(t, root, "rev-parse", "HEAD")
	receipt, err := projectbriefing.EnsureAcceptedHistory(root, revision)
	if err != nil {
		t.Fatalf("ensure accepted-model history: %v", err)
	}
	if len(receipt.Bundles) != 1 {
		t.Fatalf("fixture did not create one accepted briefing bundle: %d", len(receipt.Bundles))
	}
	bundle := receipt.Bundles[0]
	managerID = e2eManagerID("orders", "orders")
	for _, event := range bundle.Events {
		if containsString(event.AffectedManagers, managerID) {
			eventID = event.ID
			break
		}
	}
	if eventID == "" {
		t.Fatalf("accepted change did not affect Manager %s", managerID)
	}
	project, err = projectworkHost().Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	return root, revision, project, managerID, eventID
}

func TestReviewScopeDigestBindsOperationStrictnessAndBriefing(t *testing.T) {
	root := makeProjectRunFixture(t)
	project, err := projectworkHost().Load(root, identityHead(t, root))
	if err != nil {
		t.Fatal(err)
	}
	task := ManagerTask{ManagerID: e2eManagerID("orders", "orders"), Goal: "Implement the orders artifact.", Owns: []string{"src/orders/"}}
	plan := PlanRecord{Goal: "Implement both fixture artifacts.", Operation: OperationApply,
		Strictness:      map[string]StrictnessProfile{task.ManagerID: {Evidence: []string{"tests"}, Counterexamples: 1}},
		BriefingDigests: map[string]string{task.ManagerID: "briefing-digest-a"}}
	base, err := reviewScopeDigest(plan, project, task, "work", RunReport{})
	if err != nil {
		t.Fatal(err)
	}
	variants := map[string]PlanRecord{}
	operation := plan
	operation.Operation = OperationCleanup
	variants["operation"] = operation
	strictness := plan
	strictness.Strictness = map[string]StrictnessProfile{task.ManagerID: {Evidence: []string{"tests", "interfaces"}, Counterexamples: 1}}
	variants["strictness"] = strictness
	briefing := plan
	briefing.BriefingDigests = map[string]string{task.ManagerID: "briefing-digest-b"}
	variants["briefing"] = briefing
	for name, variant := range variants {
		t.Run(name, func(t *testing.T) {
			got, err := reviewScopeDigest(variant, project, task, "work", RunReport{})
			if err != nil || got == base {
				t.Fatalf("review scope digest did not bind %s: got=%s base=%s err=%v", name, got, base, err)
			}
		})
	}
}
