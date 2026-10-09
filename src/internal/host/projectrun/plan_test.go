package projectrun

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
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

func TestSelectedBasisAcceptsOperationalStateAndGitNormalizedCheckout(t *testing.T) {
	root := makeFullVerifyFixture(t)
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "name: Process fixture\n", "name: Process fixture\nworkflowMode: guided\n", 1)
	if updated == string(manifest) {
		t.Fatal("could not enable guided workflow in fixture")
	}
	writeE2E(t, root, projectwork.ManifestPath, updated)
	gitE2E(t, root, "add", projectwork.ManifestPath)
	gitE2E(t, root, "commit", "-m", "enable guided workflow")
	writeE2E(t, root, ".gitattributes", "binary.dat -text\n")
	if err := os.WriteFile(filepath.Join(root, "binary.dat"), []byte("binary\ncontent\x00"), 0644); err != nil {
		t.Fatal(err)
	}
	gitE2E(t, root, "add", ".gitattributes", "binary.dat")
	gitE2E(t, root, "commit", "-m", "record a path that disables text normalization")
	gitE2E(t, root, "config", "core.autocrlf", "true")
	readmePath := filepath.Join(root, "README.md")
	readmeBytes, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmePath, bytes.ReplaceAll(readmeBytes, []byte("\n"), []byte("\r\n")), 0644); err != nil {
		t.Fatal(err)
	}
	revision := identityHead(t, root)
	fixed, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatalf("load fixed project: %v", err)
	}
	if _, err := projectbriefing.EnsureAcceptedHistory(root, revision); err != nil {
		t.Fatalf("update accepted-history cursor: %v", err)
	}
	explorationPath := filepath.Join(root, ".markitect", "state", "explorations", "fixture.json")
	if err := os.MkdirAll(filepath.Dir(explorationPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(explorationPath, []byte("operational exploration state\n"), 0644); err != nil {
		t.Fatal(err)
	}
	working, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatalf("load working project after operational updates: %v", err)
	}
	if fixed.Snapshot.Digest() == working.Snapshot.Digest() {
		t.Fatal("fixture did not produce a normal Git-clean CRLF working checkout")
	}
	if err := requireCleanSelectedBasisAtRevision(root, revision, fixed.Snapshot, working.Snapshot); err != nil {
		t.Fatalf("normal Git-clean checkout and operational state should not stale selected inputs: %v", err)
	}
	for _, path := range []string{".markitect/state/briefings/history.json", ".markitect/state/explorations/fixture.json"} {
		if _, included := working.Snapshot.Files[path]; included {
			t.Fatalf("registered operational state %q leaked into selected project snapshot", path)
		}
	}

	binaryPath := filepath.Join(root, "binary.dat")
	binaryBytes, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, bytes.ReplaceAll(binaryBytes, []byte("\n"), []byte("\r\n")), 0644); err != nil {
		t.Fatal(err)
	}
	changedBinary, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatalf("load project after -text source edit: %v", err)
	}
	if err := requireCleanSelectedBasisAtRevision(root, revision, fixed.Snapshot, changedBinary.Snapshot); err == nil || !strings.Contains(err.Error(), "commit accepted selected changes") {
		t.Fatalf("Git -text content change must remain blocked, got %v", err)
	}

	readme := filepath.Join(root, "README.md")
	content, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	gitE2E(t, root, "update-index", "--assume-unchanged", "README.md")
	if err := os.WriteFile(readme, append(content, []byte("manual source change\r\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	dirty, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatalf("load project after substantive source edit: %v", err)
	}
	if err := requireCleanSelectedBasisAtRevision(root, revision, fixed.Snapshot, dirty.Snapshot); err == nil || !strings.Contains(err.Error(), "assume-unchanged or skip-worktree") {
		t.Fatalf("substantive selected source change hidden by Git's assume-unchanged flag should remain blocked, got %v", err)
	}
	gitE2E(t, root, "update-index", "--no-assume-unchanged", "README.md")
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

func TestGuidedPlanCapturesAcceptedHistoryWithoutStalingSelectedBasis(t *testing.T) {
	root := makeProjectRunFixture(t)
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "name: Process fixture\n", "name: Process fixture\nworkflowMode: guided\n", 1)
	if updated == string(manifest) {
		t.Fatal("could not enable guided workflow in fixture")
	}
	writeE2E(t, root, projectwork.ManifestPath, updated)
	gitE2E(t, root, "add", projectwork.ManifestPath)
	gitE2E(t, root, "commit", "-m", "enable guided workflow")
	revision := identityHead(t, root)
	plan, err := Plan(projectworkHost(), root, revision, PlanRequest{Goal: "Improve the bounded project behavior."})
	if err != nil {
		t.Fatalf("guided plan after accepted-history reconciliation: %v", err)
	}
	if plan.Status != StatusPlanned || plan.BaseRevision != revision || plan.WorkingSnapshot == "" {
		t.Fatalf("guided plan did not retain its fixed and raw working bindings: %+v", plan)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "state", "briefings", "history.json")); err != nil {
		t.Fatalf("guided planning did not persist accepted-history cursor: %v", err)
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
	for _, required := range []string{"globalGoal is context", "implement only ownTask", "allowedWritePaths as the complete set", "ArtifactRelations and ForeignOwnership describe read context", "do not implement that Manager's files, tests, or docs", "activeResponsibilities", "outer outcome must be proposed when escalateTo is empty", "escalated when escalateTo names escalationTarget", "Status describes this Manager's local work only", "Child implementation files are intentionally not supplied", "resolvedQuestions and resolvedRisks empty"} {
		if !strings.Contains(work, required) {
			t.Errorf("work guidance omitted %q", required)
		}
	}
	integrate := phaseGuidance("integrate")
	for _, required := range []string{"globalGoal is context", "activeResponsibilities", "Inspect every direct child report", "actual child candidate bytes", "Set integrated=true", "Do not create delegations", "allowedWritePaths", "outer outcome must be proposed when escalateTo is empty", "escalated when escalateTo names escalationTarget"} {
		if !strings.Contains(integrate, required) {
			t.Errorf("integration guidance omitted %q", required)
		}
	}
	if err := validateTaskOutcome(agentexec.OutcomeProposed, TaskResponse{}); err != nil {
		t.Fatalf("guidance's un-escalated outcome is rejected: %v", err)
	}
	escalated := TaskResponse{EscalateTo: "parent-manager"}
	if err := validateTaskOutcome(agentexec.OutcomeEscalated, escalated); err != nil {
		t.Fatalf("guidance's escalated outcome is rejected: %v", err)
	}
	if err := validateTaskOutcome(agentexec.OutcomeProposed, escalated); err == nil {
		t.Fatal("proposed outcome unexpectedly accepted when escalation target is set")
	}
}

func TestActiveResponsibilitiesExposeOnlyInPlanPublicRoutingFields(t *testing.T) {
	orders := e2eManagerID("sales.orders", "orders")
	engineering := e2eManagerID("engineering", "engineering")
	report := projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: orders, Purpose: "Own order lifecycle behavior.", Owns: []string{"src/shop/orders/"}, Instructions: "private orders policy"},
		{ID: engineering, Purpose: "Own project test files.", Owns: []string{"tests/"}, Instructions: "private engineering policy"},
		{ID: e2eManagerID("unused", "unused"), Purpose: "Do not include this inactive Manager.", Owns: []string{"internal/"}},
	}}
	active := activeResponsibilities(report, []ManagerTask{{ManagerID: engineering}, {ManagerID: orders}})
	if len(active) != 2 || active[0].ManagerID != engineering || active[1].ManagerID != orders {
		t.Fatalf("active scope missing or unordered: %+v", active)
	}
	if active[0].Purpose != "Own project test files." || !containsString(active[0].Owns, "tests/") {
		t.Fatalf("active Engineering routing data missing: %+v", active[0])
	}
	encoded, err := json.Marshal(active)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private orders policy", "private engineering policy", "Do not include this inactive Manager", "Instructions", "Statement", "source bytes"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("routing data leaked private or inactive context %q: %s", secret, encoded)
		}
	}
}

func TestAllowedWritePathsSeparateArtifactReadRelationsFromOwnership(t *testing.T) {
	orders := e2eManagerID("orders", "orders")
	engineering := e2eManagerID("engineering", "engineering")
	report := projectmodel.Report{
		Files: []projectmodel.FileEntry{
			{Path: "src/shop/orders/order.py", Owner: orders, Class: "source"},
			{Path: "tests/test_cancellation.py", Owner: engineering, Class: "test", Artifacts: []string{"orders-lifecycle"}, Checks: []string{"cancellation-tests"}},
		},
		Artifacts: []projectmodel.Artifact{{ID: "orders-lifecycle", Owner: orders, Paths: []string{"src/shop/orders/", "tests/test_cancellation.py"}}},
	}
	task := ManagerTask{ManagerID: orders, Artifacts: []string{"orders-lifecycle"}}
	config := projectwork.Config{InventoryRoots: []string{"src", "tests"}}
	paths := allowedWritePaths(config, report, task, "work", nil)
	if !containsString(paths, "src/shop/orders/") || !containsString(paths, "src/shop/orders/order.py") {
		t.Fatalf("owned artifact scope or file missing from write paths: %v", paths)
	}
	if containsString(paths, "tests/test_cancellation.py") {
		t.Fatalf("artifact relation incorrectly granted ownership of another Manager's test: %v", paths)
	}

	inputs := []agentexec.Artifact{{Path: "tests/test_cancellation.py", Mode: "0644", Content: []byte("read-only supplied content")}}
	relations, foreign := suppliedArtifactOwnership(report, inputs, orders, paths)
	if len(relations) != 1 || relations[0].ArtifactOwner != orders || relations[0].PathOwner != engineering || !relations[0].Readable || relations[0].Writable {
		t.Fatalf("artifact relation did not distinguish readable from writable ownership: %+v", relations)
	}
	if len(foreign) != 1 || foreign[0].Path != "tests/test_cancellation.py" || foreign[0].Owner != engineering || !containsString(foreign[0].Checks, "cancellation-tests") {
		t.Fatalf("foreign ownership metadata missing: %+v", foreign)
	}
	encoded, err := json.Marshal(struct {
		Relations []artifactPathRelation     `json:"artifactRelations"`
		Foreign   []foreignOwnershipMetadata `json:"foreignOwnership"`
	}{relations, foreign})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "read-only supplied content") || strings.Contains(string(encoded), "sha256:") {
		t.Fatalf("ownership context included source content or file digest: %s", encoded)
	}

	integrated := allowedWritePaths(config, report, task, "integrate", []string{"tests/test_cancellation.py"})
	if !containsString(integrated, "tests/test_cancellation.py") {
		t.Fatalf("exact authorized integration conflict path was omitted: %v", integrated)
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
