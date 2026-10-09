package projectrun

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestReviewerContextCarriesOperationStrictnessAndAcceptedBriefing(t *testing.T) {
	const (
		managerID   = "shop/sales"
		modelDigest = "model-accepted"
	)
	root := t.TempDir()
	bundle := projectbriefing.Bundle{
		APIVersion: projectbriefing.APIVersion, Revision: "rev-new", SinceRevision: "rev-old",
		SinceModelDigest: "model-old", ModelDigest: modelDigest,
		Events: []projectbriefing.Event{{ID: "event-order-rule", AffectedManagers: []string{managerID}}},
		Managers: []projectbriefing.Briefing{{ID: "briefing-sales", ManagerID: managerID, Revision: "rev-new", ModelDigest: modelDigest,
			EventIDs: []string{"event-order-rule"}, Summary: "Accepted sales contract changed."}},
	}
	state, storeDigest, err := projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	_ = state
	storeDigest, err = projectbriefing.Write(root, bundle, storeDigest)
	if err != nil {
		t.Fatal(err)
	}
	beforeDismissal, err := managerBriefing(root, modelDigest, managerID)
	if err != nil {
		t.Fatal(err)
	}
	storeDigest, err = projectbriefing.Dismiss(root, "event-order-rule", managerID, storeDigest)
	if err != nil {
		t.Fatal(err)
	}
	_ = storeDigest
	plan := PlanRecord{
		Operation: OperationCleanup, Goal: "Improve order cancellation safely.",
		Strictness:      map[string]StrictnessProfile{managerID: {Evidence: []string{"tests", "public interface"}, Counterexamples: 2}},
		BriefingDigests: map[string]string{managerID: beforeDismissal.Digest},
	}
	project := &Project{Report: projectmodel.Report{ModelDigest: modelDigest}}
	briefing, err := reviewerBriefing(root, plan, project, managerID)
	if err != nil {
		t.Fatal(err)
	}
	if briefing.Digest != beforeDismissal.Digest || len(briefing.Briefings) != 1 || len(briefing.Events) != 1 || briefing.Events[0].ID != "event-order-rule" {
		t.Fatalf("dismissal hid or changed accepted reviewer context: before=%+v after=%+v", beforeDismissal, briefing)
	}

	context := reviewerContext{
		Kind: "projectrun-review/v1", Operation: plan.Operation, OperationGuidance: OperationGuidance(plan.Operation),
		Strictness: plan.Strictness[managerID], Briefing: briefing, ManagerID: managerID,
	}
	encoded, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got["operation"] != OperationCleanup || got["operationGuidance"] == "" || got["strictness"] == nil || got["briefing"] == nil {
		t.Fatalf("reviewer request omitted operation, strictness, or accepted briefing: %s", encoded)
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
	const managerID = "shop/sales"
	root := t.TempDir()
	state, storeDigest, err := projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	_ = state
	_, err = projectbriefing.Write(root, projectbriefing.Bundle{
		APIVersion: projectbriefing.APIVersion, Revision: "rev-accepted", SinceRevision: "rev-prior",
		SinceModelDigest: "model-prior", ModelDigest: "model-accepted",
		Global: projectbriefing.Briefing{ID: "global-accepted", Revision: "rev-accepted", ModelDigest: "model-accepted"},
	}, storeDigest)
	if err != nil {
		t.Fatal(err)
	}
	plan := PlanRecord{ModelEdit: &EditPlan{}, BriefingDigests: map[string]string{managerID: ""}}
	project := &Project{Report: projectmodel.Report{ModelDigest: "model-draft-target"}}
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
	base, err := reviewScopeDigest(plan, project, task, "work")
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
			got, err := reviewScopeDigest(variant, project, task, "work")
			if err != nil || got == base {
				t.Fatalf("review scope digest did not bind %s: got=%s base=%s err=%v", name, got, base, err)
			}
		})
	}
}
