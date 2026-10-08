package projectrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestProjectModelEditCannotChangeRuntimeConfiguration(t *testing.T) {
	for _, path := range []string{RuntimePath, ".markitect/runtime.yaml/child", ".markitect/project.yaml"} {
		if err := validateModelEditPaths(Mutation{Files: []FileChange{{Path: path}}}); err == nil {
			t.Fatalf("ModelEdit path %q unexpectedly passed the model-only contract", path)
		}
	}
	if err := validateModelEditPaths(Mutation{Files: []FileChange{{Path: ".markitect/model/managers/root.yaml"}}}); err != nil {
		t.Fatalf("valid model edit path rejected: %v", err)
	}
}

func TestPlanChecksBlockRequiredArtifactsWithMissingChecks(t *testing.T) {
	owner := "orders"
	report := projectmodel.Report{Artifacts: []projectmodel.Artifact{{ID: "orders-code", Owner: owner, Required: true, Paths: []string{"src/orders/"}, Checks: []string{"orders-test"}}}}
	checks, findings := planChecks(report, map[string]bool{owner: true})
	if len(checks) != 0 {
		t.Fatalf("unexpected checks: %+v", checks)
	}
	if len(findings) == 0 {
		t.Fatal("required artifact with a missing check did not block planning")
	}
}

func TestPlanRequiresWorkingSelectedInputsEqualFixedRevision(t *testing.T) {
	fixed := &snapshot.Snapshot{Files: map[string][]byte{"src/main.go": []byte("committed")}, Modes: map[string]string{"src/main.go": snapshot.RegularMode}}
	working := &snapshot.Snapshot{Files: map[string][]byte{"src/main.go": []byte("uncommitted")}, Modes: map[string]string{"src/main.go": snapshot.RegularMode}}
	err := requireCleanSelectedBasis(fixed, working)
	if err == nil || !strings.Contains(err.Error(), "commit accepted selected changes") {
		t.Fatalf("expected dirty selected input to block planning, got %v", err)
	}
	working.Files["src/main.go"] = []byte("committed")
	if err := requireCleanSelectedBasis(fixed, working); err != nil {
		t.Fatalf("matching selected snapshots rejected: %v", err)
	}
}

func TestPlanSinceUsesOldComparisonAndCurrentHeadExecutionBase(t *testing.T) {
	root := makeProjectRunFixture(t)
	old := gitE2E(t, root, "rev-parse", "HEAD")
	statement := filepath.Join(root, ".markitect/model/orders/statement.yaml")
	data, err := os.ReadFile(statement)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "Implement the orders artifact.", "Prevent cancelled orders from shipping.", 1)
	if updated == string(data) {
		t.Fatal("fixture statement description was not found")
	}
	writeE2E(t, root, ".markitect/model/orders/statement.yaml", updated)
	gitE2E(t, root, "add", ".markitect/model/orders/statement.yaml")
	gitE2E(t, root, "commit", "-m", "change orders requirement")
	current := gitE2E(t, root, "rev-parse", "HEAD")

	plan, err := Plan(projectworkHost(), root, "", PlanRequest{Goal: "Prevent cancelled orders from shipping.",
		Managers: []string{e2eManagerID("", "project-owner")}, SinceRevision: old})
	if err != nil {
		t.Fatalf("plan since revision: %v", err)
	}
	if plan.BaseRevision != current || plan.TargetHead != current {
		t.Fatalf("execution basis did not remain current HEAD: base=%s target=%s current=%s", plan.BaseRevision, plan.TargetHead, current)
	}
	if plan.ChangeBaseRevision != old || plan.ChangeBaseSnapshot == "" || plan.ChangeBaseProjectDigest == "" ||
		plan.ChangeBaseModelDigest == "" || plan.ChangeBaseReportDigest == "" || plan.ChangeImpact == nil || plan.ChangeImpactDigest == "" {
		t.Fatalf("since comparison was not fully bound: %+v", plan)
	}
	if plan.ChangeImpact.BaseDigest != plan.ChangeBaseReportDigest || plan.ChangeImpact.CandidateDigest != plan.ReportDigest || plan.ChangeImpact.Digest != plan.ChangeImpactDigest {
		t.Fatalf("impact does not bind old and current reports: %+v", plan.ChangeImpact)
	}
	seen := map[string]bool{}
	for _, task := range plan.Managers {
		seen[task.ManagerID] = true
	}
	for _, id := range []string{e2eManagerID("", "project-owner"), e2eManagerID("orders", "orders")} {
		if !seen[id] {
			t.Fatalf("explicit inventory manager filtered required since-impact manager %s: %+v", id, plan.Managers)
		}
	}
	if len(plan.ChangeImpact.Checks) == 0 {
		t.Fatal("statement impact did not derive its owning checks")
	}
	checkSelected := false
	for _, check := range plan.Checks {
		if check.ID == "[\""+projectmodel.APIVersion+"\",\"Check\",\"orders\",\"orders-check\"]" && check.Required {
			checkSelected = true
		}
	}
	if !checkSelected {
		t.Fatalf("since impact did not activate the owning check: %+v", plan.Checks)
	}
}

func TestPlanWithoutRoutingSelectsAllManagers(t *testing.T) {
	root := makeProjectRunFixture(t)
	plan, err := Plan(projectworkHost(), root, "", PlanRequest{Goal: "Improve the bounded project behavior."})
	if err != nil {
		t.Fatalf("plan default manager selection: %v", err)
	}
	project, err := projectwork.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Managers) != len(project.Report.Managers) {
		t.Fatalf("goal-only request selected %d of %d declared managers", len(plan.Managers), len(project.Report.Managers))
	}
}

func TestSinceImpactAndExplicitManagerRouteThroughRequiredHierarchy(t *testing.T) {
	root := e2eManagerID("", "shop")
	commerce := e2eManagerID("commerce", "commerce")
	sales := e2eManagerID("commerce.sales", "sales")
	orders := e2eManagerID("commerce.sales.orders", "orders")
	inventory := e2eManagerID("commerce.sales.inventory", "inventory")
	report := projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: root},
		{ID: commerce, Parent: root},
		{ID: sales, Parent: commerce},
		{ID: orders, Parent: sales},
		{ID: inventory, Parent: sales},
	}}
	impact := &projectmodel.ChangeImpact{Managers: []string{orders}}
	tasks, _, _, err := planManagers(report, map[string][]byte{}, PlanRequest{Goal: "Prevent cancelled orders from shipping.", Managers: []string{inventory}}, nil, impact, Limits{MaxDepth: 8})
	if err != nil {
		t.Fatalf("plan hierarchical impact: %v", err)
	}
	want := map[string]bool{root: true, commerce: true, sales: true, orders: true, inventory: true}
	if len(tasks) != len(want) {
		t.Fatalf("selected %d managers, want full root→commerce→sales→orders/inventory path: %+v", len(tasks), tasks)
	}
	for _, task := range tasks {
		if !want[task.ManagerID] {
			t.Errorf("unexpected Manager in selected hierarchy: %s", task.ManagerID)
		}
		delete(want, task.ManagerID)
	}
	if len(want) != 0 {
		t.Errorf("required Managers omitted from selection: %+v", want)
	}
}

func TestPhaseGuidanceDefinesLocalWorkAndIntegrationResponsibilities(t *testing.T) {
	work := phaseGuidance("work")
	for _, required := range []string{"outer agent outcome must be proposed", "Status complete means this Manager completed its own work", "Child implementation files are intentionally not supplied", "resolvedQuestions and resolvedRisks empty"} {
		if !strings.Contains(work, required) {
			t.Errorf("work guidance omitted %q", required)
		}
	}
	integrate := phaseGuidance("integrate")
	for _, required := range []string{"inspect every direct child report", "actual child candidate bytes", "Set integrated=true", "Do not create delegations"} {
		if !strings.Contains(integrate, required) {
			t.Errorf("integration guidance omitted %q", required)
		}
	}
}

func TestExecutorOutcomeErrorUsesOnlyWhitelistedDiagnostics(t *testing.T) {
	known := "Codex authentication was unavailable or rejected. Authenticate the configured account and retry."
	err := executorOutcomeError(agentexec.Response{Outcome: agentexec.OutcomeIncomplete, Uncertainty: []string{known, "private provider body: token=secret"}})
	if err == nil || !strings.Contains(err.Error(), known) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("safe diagnostic handling failed: %v", err)
	}
	unknown := executorOutcomeError(agentexec.Response{Outcome: agentexec.OutcomeFailed, Uncertainty: []string{"private provider body"}})
	if unknown == nil || strings.Contains(unknown.Error(), "private provider body") || !strings.Contains(unknown.Error(), "no safe provider diagnostic") {
		t.Fatalf("untrusted uncertainty escaped or generic fallback missing: %v", unknown)
	}
}
