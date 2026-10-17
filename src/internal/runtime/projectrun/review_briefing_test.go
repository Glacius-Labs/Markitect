package projectrun

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

func TestReviewerContextCarriesOperationStrictnessAndAcceptedBriefing(t *testing.T) {
	project, managerID, context := reviewBriefingFacts()
	host := fixedBriefingHost(context)
	plan := PlanRecord{
		Operation: OperationCleanup, Goal: "Improve order cancellation safely.",
		Strictness:      map[string]projectconfig.StrictnessProfile{managerID: {Evidence: []string{"tests", "public interface"}, Counterexamples: 2}},
		BriefingDigests: map[string]string{managerID: context.Digest},
	}
	briefing, err := reviewerBriefing(host, "", plan, project, managerID)
	if err != nil {
		t.Fatal(err)
	}
	if briefing.Digest != context.Digest || len(briefing.Briefings) != 1 || len(briefing.Events) != 1 || briefing.Events[0].ID != context.Events[0].ID {
		t.Fatalf("reviewer context lost its accepted briefing facts: %+v", briefing)
	}

	reviewer := reviewerContext{
		Kind: "projectrun-review/v1", Operation: plan.Operation, ReviewerGuidance: reviewerAssessmentGuidance,
		Strictness: plan.Strictness[managerID], Briefing: briefing, ManagerID: managerID,
		Delegations: []Delegation{{ManagerID: "child-tests", Goal: "Add focused behavior tests."}},
	}
	encoded, err := json.Marshal(reviewer)
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
	if _, err := reviewerBriefing(host, "", plan, project, managerID); !errors.Is(err, ErrStale) {
		t.Fatalf("review accepted a briefing digest that differs from its plan: %v", err)
	}
}

func TestDraftModelEditReviewerDoesNotLoadUnacceptedBriefingHistory(t *testing.T) {
	project, managerID, _ := reviewBriefingFacts()
	calls := 0
	host := Host{BriefingForManager: func(root, revision, modelDigest, manager string, view *projectbriefing.Store) (BriefingContext, error) {
		calls++
		return BriefingContext{}, errors.New("unexpected accepted-history acquisition")
	}}
	plan := PlanRecord{ModelEdit: &projectvalue.EditPlan{}, BriefingDigests: map[string]string{managerID: ""}}
	briefing, err := reviewerBriefing(host, "", plan, project, managerID)
	if err != nil {
		t.Fatalf("draft reviewer tried to load accepted history: %v", err)
	}
	if briefing.Digest != "" || len(briefing.Briefings) != 0 || len(briefing.Events) != 0 || calls != 0 {
		t.Fatalf("draft reviewer inherited accepted history or called the history port: briefing=%+v calls=%d", briefing, calls)
	}
	plan.BriefingDigests[managerID] = "unexpected-accepted-binding"
	if _, err := reviewerBriefing(host, "", plan, project, managerID); !errors.Is(err, ErrStale) {
		t.Fatalf("draft reviewer accepted an accepted-history binding: %v", err)
	}
}

func TestReviewScopeDigestBindsOperationStrictnessAndBriefing(t *testing.T) {
	project, managerID, _ := reviewBriefingFacts()
	task := ManagerTask{ManagerID: managerID, Goal: "Implement the orders artifact.", Artifacts: []string{"artifact-orders"}}
	plan := PlanRecord{Goal: "Implement the orders artifact.", Operation: OperationApply,
		Strictness:      map[string]projectconfig.StrictnessProfile{task.ManagerID: {Evidence: []string{"tests"}, Counterexamples: 1}},
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
	strictness.Strictness = map[string]projectconfig.StrictnessProfile{task.ManagerID: {Evidence: []string{"tests", "interfaces"}, Counterexamples: 1}}
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

func reviewBriefingFacts() (*projectvalue.Project, string, BriefingContext) {
	managerID := "manager-orders"
	statementID := "statement-orders"
	artifactID := "artifact-orders"
	project := &projectvalue.Project{
		Revision: "fixed-revision",
		Config:   projectvalue.Config{InventoryRoots: []string{"src"}},
		Report: projectmodel.Report{
			ModelDigest: "model-digest",
			Managers:    []projectmodel.Manager{{ID: managerID, Name: "orders", Namespace: "orders", Owns: []string{"src/orders/"}}},
			Statements:  []projectmodel.Statement{{ID: statementID, Name: "orders-work", Owner: managerID, Category: "concept", Description: "Orders invariants", Public: true}},
			Artifacts:   []projectmodel.Artifact{{ID: artifactID, Name: "orders-code", Owner: managerID, Role: "implementation", Realizes: []string{statementID}, Paths: []string{"src/orders/implementation.txt"}, Required: true}},
			Files:       []projectmodel.FileEntry{{Path: "src/orders/implementation.txt", Owner: managerID, Class: "managed", Statements: []string{statementID}, Artifacts: []string{artifactID}}},
		},
		Snapshot: &source.Input{ID: "fixed-revision", Files: map[string][]byte{"src/orders/implementation.txt": []byte("orders bytes\n")}, Modes: map[string]string{"src/orders/implementation.txt": "100644"}},
	}
	context := BriefingContext{Digest: "briefing-digest-a", Briefings: []projectbriefing.Briefing{{}}, Events: []projectbriefing.Event{{ID: "accepted-event"}}}
	return project, managerID, context
}

func fixedBriefingHost(context BriefingContext) Host {
	return Host{BriefingForManager: func(root, revision, modelDigest, manager string, view *projectbriefing.Store) (BriefingContext, error) {
		return context, nil
	}}
}
