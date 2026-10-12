package projectapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

func TestPlanSinceUsesOldComparisonAndCurrentHeadExecutionBase(t *testing.T) {
	root := deliveryFixture(t)
	old := gitOutput(t, root, "rev-parse", "HEAD")
	statementPath := ".markitect/model/orders/statement.yaml"
	statement, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(statementPath)))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(statement), "Implement the orders artifact.", "Prevent cancelled orders from shipping.", 1)
	if updated == string(statement) {
		t.Fatal("fixture statement description was not found")
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(statementPath)), []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	operations := NewOperations()
	document, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatalf("render changed source model through Application: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(document.Path)), []byte(document.Content), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", statementPath, document.Path)
	runGit(t, root, "commit", "-m", "change orders requirement")
	current := gitOutput(t, root, "rev-parse", "HEAD")

	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: projectrun.PlanRequest{
		Goal: "Prevent cancelled orders from shipping.", Managers: []string{deliveryManagerID("", "project-owner")}, SinceRevision: old,
	}})
	if err != nil {
		t.Fatalf("plan since revision through Application: %v", err)
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
	for _, id := range []string{deliveryManagerID("", "project-owner"), deliveryManagerID("orders", "orders")} {
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
