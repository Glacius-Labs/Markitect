package projectgraph

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/knowledgeevidence"
	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectknowledge"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestManagerGraphPreservesContextAndFiltersPrivateEdgesAndSources(t *testing.T) {
	p := graphFixture(t, true, false)
	orders := id("Manager", "orders", "orders")
	view, err := Build(p, Selection{ManagerID: orders}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Query(view, Request{Action: ActionGraph})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(result)
	text := string(data)
	for _, forbidden := range []string{"inventory-private-secret", "src/inventory/secret.go", ".markitect/model/inventory/release.yaml", "inventory-secret-code", "CHILD-INSTRUCTION-SECRET", "orders-secret-manager"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("manager graph leaked %q: %s", forbidden, text)
		}
	}
	for _, expected := range []string{"cancel-order", "Release reservation.", "src/orders/cancel.go", "cancel-order-tests", `"command":["go","test","./orders"]`, "owns-file", "checks-file"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("manager graph omitted %q: %s", expected, text)
		}
	}
	// Definition ownership is the nearest Manager declaration from Report;
	// the exact file owner is a separate Report fact and must remain distinct.
	statementOwner := false
	fileOwner := false
	for _, edge := range result.Edges {
		if edge.From == id("Statement", "orders", "cancel-order") && edge.To == orders && edge.Property == "owned-by" && strings.Contains(edge.Basis, "nearest Manager ownership") {
			statementOwner = true
		}
		if edge.From == orders && edge.To == fileID("src/orders/cancel.go") && edge.Property == "owns-file" {
			fileOwner = true
		}
	}
	if !statementOwner || !fileOwner {
		t.Fatalf("definition owner and exact file owner were not represented separately: statement=%v file=%v", statementOwner, fileOwner)
	}
	foreignOwner := id("Manager", "inventory", "inventory")
	foreignOwnerNode := false
	for _, node := range result.Nodes {
		if node.ID == foreignOwner {
			foreignOwnerNode = true
			if len(node.Properties) != 0 || node.Purpose != "" || node.Source != nil {
				t.Fatalf("public contract owner identity exposed private manager details: %+v", node)
			}
		}
	}
	if !foreignOwnerNode {
		t.Fatal("public contract owner identity was not admitted to explain definition ownership")
	}
	for _, edge := range result.Edges {
		if edge.From == id("Statement", "inventory", "release-reservation") && (edge.Property == "uses" || edge.Property == "requires") {
			t.Fatalf("manager query followed a public contract's outgoing relation: %+v", edge)
		}
	}
	if result.Binding.Revision != p.Revision || result.Provisional != p.Provisional || result.Binding.ModelDigest != p.Model.Digest {
		t.Fatalf("source bindings were not preserved: %+v", result)
	}
	if result.QueryDigest == "" || result.QueryDigest == result.Binding.GraphDigest {
		t.Fatalf("query digest is absent or conflated with graph digest: %+v", result)
	}
	trace, err := Query(view, Request{Action: ActionTrace, TargetID: id("Statement", "orders", "cancel-order"), Bidirectional: true})
	if err != nil {
		t.Fatal(err)
	}
	foundCheck := false
	for _, w := range trace.Trace.Results {
		if w.Node == id("Check", "orders", "cancel-order-tests") && len(w.Path) == 2 {
			foundCheck = true
		}
	}
	if !foundCheck {
		t.Fatalf("bidirectional realization/check witness was not returned: %+v", trace.Trace)
	}
}

func TestProjectGraphUsesExactReportFileEntriesAndReportsPartialUnknownCoverage(t *testing.T) {
	p := graphFixture(t, false, true)
	view, err := Build(p, Selection{ProjectScope: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Query(view, Request{Action: ActionCoverage})
	if err != nil {
		t.Fatal(err)
	}
	if result.Coverage == nil || result.Coverage.Inventory != projectknowledge.FactPartial || result.Coverage.Checks != projectknowledge.FactUnknown {
		t.Fatalf("coverage does not preserve unknown and partial state: %+v", result.Coverage)
	}
	graph, err := Query(view, Request{Action: ActionGraph})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range graph.Nodes {
		if n.Kind == "ProjectFile" {
			if raw := n.Properties["path"]; raw != nil && string(raw) == `"src/inventory/release.go"` {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("exact report-derived file fact was omitted")
	}
	for _, n := range graph.Nodes {
		if n.Kind == "ProjectFile" && string(n.Properties["path"]) == `"src/unassigned.txt"` && len(n.Properties) == 0 {
			t.Fatal("unexpected malformed file fact")
		}
	}
}

func TestScopeAndNotFoundAreExplicitAndDeterministic(t *testing.T) {
	p := graphFixture(t, true, false)
	orders := id("Manager", "orders", "orders")
	first, err := SafeScope(p, Selection{ManagerID: orders})
	if err != nil {
		t.Fatal(err)
	}
	second, err := SafeScope(p, Selection{ManagerID: orders})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || len(first.Paths) == 0 {
		t.Fatalf("scope is not deterministic or did not expose exact paths: %+v %+v", first, second)
	}
	if _, err := Build(p, Selection{}, nil); err != ErrInvalidSelection {
		t.Fatalf("empty scope accepted: %v", err)
	}
	view, err := Build(p, Selection{ManagerID: orders}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Query(view, Request{Action: ActionExplain, TargetID: "missing"})
	if err != projectknowledge.ErrNotFound {
		t.Fatalf("missing node response differs: %v", err)
	}
}

func TestBuildRejectsUnboundProjectBeforeCreatingEvidenceView(t *testing.T) {
	p := graphFixture(t, true, false)
	selection := Selection{ManagerID: id("Manager", "orders", "orders")}
	withoutSnapshot := *p
	withoutSnapshot.Snapshot = nil
	if _, err := Build(&withoutSnapshot, selection, nil); err == nil {
		t.Fatal("project graph accepted a project without a source snapshot")
	}
	wrongReportBinding := *p
	wrongReportBinding.Report.ModelDigest = "sha256:stale-model"
	if _, err := Build(&wrongReportBinding, selection, nil); err == nil {
		t.Fatal("project graph accepted a report bound to another model")
	}
}

func TestSelectedEvidenceIsSourceBoundAndUnsafeFieldsBecomePartial(t *testing.T) {
	p := graphFixture(t, true, false)
	sel := Selection{ManagerID: id("Manager", "orders", "orders")}
	base, err := Build(p, sel, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkID := id("Check", "orders", "cancel-order-tests")
	capture := knowledgeevidence.Capture{Completeness: projectknowledge.FactKnown, CaptureDigest: "sha256:capture", Facts: projectknowledge.ProjectFacts{SnapshotDigest: p.Snapshot.Digest(), ProjectDigest: p.Digest, Digest: "sha256:captured-records", Facts: []projectknowledge.Fact{
		{ID: "record/run/run-1", Kind: "ExecutionRun", State: projectknowledge.FactPartial, Properties: map[string]json.RawMessage{"runtimeBindingStatus": json.RawMessage(`"not-compared"`), "captureState": json.RawMessage(`"consistent"`), "prompt": json.RawMessage(`"PROMPT-SECRET"`)}},
		{ID: "record/check/run-1/" + checkID, Kind: "CheckExecution", State: projectknowledge.FactKnown, Properties: map[string]json.RawMessage{"id": mustJSON(checkID), "checkPassed": json.RawMessage(`true`), "verificationStatus": json.RawMessage(`"unknown"`)}},
		{ID: "record/review/run-1/task/implement/1", Kind: "CandidateReview", State: projectknowledge.FactKnown, Properties: map[string]json.RawMessage{"managerId": mustJSON(id("Manager", "orders", "orders")), "phase": mustJSON("implement"), "round": json.RawMessage(`1`), "outcome": mustJSON("accepted"), "candidateBindingStatus": mustJSON("validated-digest-binding")}},
		{ID: "record/verification/run-1/selected", Kind: "CandidateVerification", State: projectknowledge.FactPartial, Properties: map[string]json.RawMessage{"verificationStatus": mustJSON("unknown"), "candidateDigest": mustJSON("sha256:candidate")}},
	}}}
	capture.SourceBindings = []knowledgeevidence.SourceBinding{{Kind: "run", RecordID: "run-1", Digest: "sha256:run", Schema: "run/v1", Source: "live-operational-record", ModelDigest: p.Model.Digest, Revision: p.Revision}}
	extra := &ExtraFacts{ScopeID: base.VisibleScope().ID, SelectedIDs: []string{}, Facts: capture.Facts, CaptureDigest: capture.CaptureDigest, Completeness: capture.Completeness, EvidenceSelected: true}
	for _, f := range capture.Facts.Facts {
		extra.SelectedIDs = append(extra.SelectedIDs, f.ID)
	}
	for _, b := range capture.SourceBindings {
		extra.SourceBindings = append(extra.SourceBindings, RecordBinding{Kind: b.Kind, Digest: b.Digest, Schema: b.Schema, Source: b.Source, ModelDigest: b.ModelDigest, Revision: b.Revision, RuntimeDigest: b.RuntimeDigest})
	}
	view, err := Build(p, sel, extra)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Query(view, Request{Action: ActionGraph})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	text := string(encoded)
	if strings.Contains(text, "PROMPT-SECRET") || strings.Contains(text, "inventory-private-secret") {
		t.Fatalf("evidence crossed the declared projection boundary: %s", text)
	}
	if !strings.Contains(text, "runtimeBindingStatus") || !strings.Contains(text, "not-compared") || !strings.Contains(text, "checkPassed") || !strings.Contains(text, "CandidateVerification") || !strings.Contains(text, "CandidateReview") {
		t.Fatalf("safe runtime/check/review/verification evidence was dropped: %s", text)
	}
	coverage, err := Query(view, Request{Action: ActionCoverage})
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Coverage == nil || coverage.Coverage.Records != projectknowledge.FactPartial {
		t.Fatalf("dropped unsafe data did not preserve partial status: %+v", coverage.Coverage)
	}
	stale := *extra
	stale.Facts = capture.Facts
	stale.Facts.SnapshotDigest = "different-source"
	if _, err := Build(p, sel, &stale); err == nil {
		t.Fatal("stale evidence capture was merged into the current graph")
	}
}

func TestActualEvidenceAdapterCaptureMergesThroughFacadeScope(t *testing.T) {
	p := graphFixture(t, true, false)
	p.Root = t.TempDir()
	if output, err := exec.Command("git", "-C", p.Root, "init", "--quiet", "-b", "evidence-test").CombinedOutput(); err != nil {
		t.Fatalf("initialize temporary evidence repository: %v: %s", err, output)
	}
	selection := Selection{ManagerID: id("Manager", "orders", "orders")}
	base, err := Build(p, selection, nil)
	if err != nil {
		t.Fatal(err)
	}
	viewScope := base.VisibleScope()
	record := projectexplore.Record{
		APIVersion:     projectexplore.APIVersion,
		ID:             "orders-evidence",
		Status:         projectexplore.StatusActive,
		Request:        "PRIVATE-REQUEST-BODY",
		CreatedAgainst: "sha256:" + strings.Repeat("a", 64),
		Scopes:         []projectexplore.Scope{{ID: "orders-task", Name: "PRIVATE-SCOPE-NAME", Goal: "PRIVATE-SCOPE-GOAL", Operation: "implement", ManagerIDs: []string{"inventory", "orders"}}},
		Decisions:      []projectexplore.Decision{{ID: "private-decision", ScopeIDs: []string{"orders-task"}, Question: "PRIVATE-QUESTION", Blocking: true, Status: "open"}},
		Drafts:         []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	encoded, err := projectexplore.EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := projectexplore.RecordPath(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(p.Root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := projectexplore.Load(p.Root, record.ID)
	if err != nil {
		t.Fatalf("load encoded evidence fixture: %v", err)
	}
	if len(loaded.Scopes) != 1 || loaded.Scopes[0].ManagerIDs[1] != "orders" {
		t.Fatalf("encoded exploration scope did not round-trip: %+v", loaded.Scopes)
	}
	capture, err := knowledgeevidence.Read(p, knowledgeevidence.Scope{ManagerID: "orders", VisibleDefinitionIDs: viewScope.DefinitionIDs, VisiblePaths: viewScope.Paths}, knowledgeevidence.Selection{ExplorationIDs: []string{record.ID}})
	if err != nil {
		t.Fatal(err)
	}
	extra := &ExtraFacts{ScopeID: viewScope.ID, Facts: capture.Facts, CaptureDigest: capture.CaptureDigest, Completeness: capture.Completeness, EvidenceSelected: true}
	for _, fact := range capture.Facts.Facts {
		extra.SelectedIDs = append(extra.SelectedIDs, fact.ID)
	}
	for _, binding := range capture.SourceBindings {
		extra.SourceBindings = append(extra.SourceBindings, RecordBinding{Kind: binding.Kind, Digest: binding.Digest, Schema: binding.Schema, Source: binding.Source, ModelDigest: binding.ModelDigest, Revision: binding.Revision, RuntimeDigest: binding.RuntimeDigest, DigestKind: binding.DigestKind})
	}
	view, err := Build(p, selection, extra)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Query(view, Request{Action: ActionGraph})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(result)
	text := string(data)
	for _, forbidden := range []string{"PRIVATE-REQUEST-BODY", "PRIVATE-SCOPE-NAME", "PRIVATE-SCOPE-GOAL", "PRIVATE-QUESTION", id("Manager", "inventory", "inventory")} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("actual evidence adapter crossed the selected graph boundary with %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "orders-evidence") || result.CaptureDigest != capture.CaptureDigest {
		t.Fatalf("actual evidence adapter output was not merged with its safe bindings: %+v", result)
	}
}

func TestHistoryFollowsOnlyVisibleDefinitionEventAndBoundedResolution(t *testing.T) {
	p := graphFixture(t, true, false)
	selection := Selection{ManagerID: id("Manager", "orders", "orders")}
	base, err := Build(p, selection, nil)
	if err != nil {
		t.Fatal(err)
	}
	eventID := "record/history/orders/event/evt-1"
	briefingID := "record/history/orders/briefing/brief-1"
	resolutionID := eventID + "/resolution"
	runID := "record/run/run-private"
	extra := &ExtraFacts{ScopeID: base.VisibleScope().ID, SelectedIDs: []string{eventID, briefingID, resolutionID, runID}, EvidenceSelected: true, CaptureDigest: "sha256:history-capture", Completeness: projectknowledge.FactKnown, Facts: projectknowledge.ProjectFacts{SnapshotDigest: p.Snapshot.Digest(), ProjectDigest: p.Digest, Digest: "sha256:history-facts", Facts: []projectknowledge.Fact{
		{ID: eventID, Kind: "ModelHistoryEvent", State: projectknowledge.FactKnown, Properties: map[string]json.RawMessage{"digest": mustJSON("sha256:event"), "change": mustJSON("description changed"), "definitionId": mustJSON(id("Statement", "orders", "cancel-order")), "resolutionStatus": mustJSON("resolved"), "authenticated": json.RawMessage(`false`)}},
		{ID: briefingID, Kind: "Briefing", State: projectknowledge.FactKnown, Properties: map[string]json.RawMessage{"revision": mustJSON("rev-1"), "modelDigest": mustJSON(p.Model.Digest), "summary": mustJSON("Recorded change summary")}},
		{ID: resolutionID, Kind: "BriefingResolution", State: projectknowledge.FactKnown, Properties: map[string]json.RawMessage{"digest": mustJSON("sha256:resolution"), "modelRevision": mustJSON("rev-1"), "modelDigest": mustJSON(p.Model.Digest), "recordedFullVerifyPassed": json.RawMessage(`true`), "authenticated": json.RawMessage(`false`)}},
		{ID: runID, Kind: "ExecutionRun", State: projectknowledge.FactKnown, Properties: map[string]json.RawMessage{"digest": mustJSON("sha256:run"), "status": mustJSON("completed")}},
	}, Relations: []projectknowledge.Relation{
		{From: eventID, To: id("Statement", "orders", "cancel-order"), Property: "changedDefinition"},
		{From: briefingID, To: eventID, Property: "containsEvent"},
		{From: eventID, To: resolutionID, Property: "hasResolution"},
		{From: resolutionID, To: runID, Property: "referencesRun"},
	}}}
	view, err := Build(p, selection, extra)
	if err != nil {
		t.Fatal(err)
	}
	history, err := Query(view, Request{Action: ActionHistory, TargetID: id("Statement", "orders", "cancel-order")})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(history)
	text := string(data)
	for _, expected := range []string{"ModelHistoryEvent", "BriefingResolution", "Recorded change summary", "description changed"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("history view omitted explicitly linked %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "record/run/run-private") || strings.Contains(text, "sha256:run") {
		t.Fatalf("history followed an operational reference beyond its bounded resolution: %s", text)
	}
}

func TestExtraFactsCannotIntroduceHiddenManagerIDs(t *testing.T) {
	p := graphFixture(t, true, false)
	selection := Selection{ManagerID: id("Manager", "orders", "orders")}
	base, err := Build(p, selection, nil)
	if err != nil {
		t.Fatal(err)
	}
	privateManager := id("Manager", "", "root")
	factID := "record/scope/scope-1"
	extra := &ExtraFacts{ScopeID: base.VisibleScope().ID, SelectedIDs: []string{factID}, EvidenceSelected: true, CaptureDigest: "sha256:capture", Completeness: projectknowledge.FactKnown, Facts: projectknowledge.ProjectFacts{SnapshotDigest: p.Snapshot.Digest(), ProjectDigest: p.Digest, Digest: "sha256:extra", Facts: []projectknowledge.Fact{{ID: factID, Kind: "WorkScope", State: projectknowledge.FactKnown, Properties: map[string]json.RawMessage{"id": mustJSON("scope-1"), "operation": mustJSON("implement"), "managerIds": mustJSON([]string{privateManager})}}}}}
	view, err := Build(p, selection, extra)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := Query(view, Request{Action: ActionGraph})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(graph)
	if strings.Contains(string(data), privateManager) {
		t.Fatalf("direct ExtraFacts caller introduced a hidden Manager ID: %s", data)
	}
	coverage, err := Query(view, Request{Action: ActionCoverage})
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Coverage == nil || coverage.Coverage.Records != projectknowledge.FactPartial {
		t.Fatalf("removed hidden ID was not reflected as partial record coverage: %+v", coverage.Coverage)
	}
}

func mustJSON(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

func TestHistoryUsesExplicitHistoricalIdentityWithoutCreatingLiveCoreNode(t *testing.T) {
	p := graphFixture(t, true, false)
	view, err := Build(p, Selection{ManagerID: id("Manager", "orders", "orders")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldID := id("Statement", "orders", "old-cancel-order")
	if _, err := view.Index().Node(oldID); err != projectknowledge.ErrNotFound {
		t.Fatalf("historical identity became a live Core node: %v", err)
	}
	result, err := Query(view, Request{Action: ActionHistory, TargetID: oldID})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(result)
	text := string(b)
	if !strings.Contains(text, "old-cancel-order") || !strings.Contains(text, "renamed") || !strings.Contains(text, "cancel-order") {
		t.Fatalf("explicit historical claim is not queryable: %s", text)
	}
}

func graphFixture(t *testing.T, withCheck, withUnknown bool) *projectwork.Project {
	t.Helper()
	api := projectmodel.APIVersion
	modelFiles := []string{".markitect/model/root.yaml", ".markitect/model/orders/manager.yaml", ".markitect/model/orders/payments/manager.yaml", ".markitect/model/inventory/manager.yaml", ".markitect/model/orders/cancel.yaml", ".markitect/model/orders/rename.yaml", ".markitect/model/inventory/release.yaml", ".markitect/model/inventory/secret.yaml", ".markitect/model/orders/artifact.yaml", ".markitect/model/inventory/artifact.yaml"}
	if withCheck {
		modelFiles = append(modelFiles, ".markitect/model/orders/check.yaml")
	}
	list := ""
	for _, path := range modelFiles {
		list += "  - " + path + "\n"
	}
	files := map[string]string{
		projectwork.ManifestPath:                        "apiVersion: " + api + "\nname: Shop\nmodelFiles:\n" + list + "inventoryRoots:\n  - src\nexclusions: []\n",
		".markitect/model/root.yaml":                    "apiVersion: " + api + "\nkind: Manager\nmetadata:\n  name: root\n  namespace: \"\"\npurpose: Shop root.\nspec:\n  owns: [" + rootOwner(withUnknown) + "]\n",
		".markitect/model/orders/manager.yaml":          "apiVersion: " + api + "\nkind: Manager\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Orders manager.\nspec:\n  parent: {apiVersion: " + api + ", kind: Manager, namespace: \"\", name: root}\n  owns: [src/orders/]\n  instructions: Keep this child instruction private.\n",
		".markitect/model/orders/payments/manager.yaml": "apiVersion: " + api + "\nkind: Manager\nmetadata:\n  name: payments\n  namespace: orders.payments\npurpose: Payments manager.\nspec:\n  parent: {apiVersion: " + api + ", kind: Manager, namespace: orders, name: orders}\n  owns: [src/orders/payments/]\n  instructions: CHILD-INSTRUCTION-SECRET\n",
		".markitect/model/inventory/manager.yaml":       "apiVersion: " + api + "\nkind: Manager\nmetadata:\n  name: inventory\n  namespace: inventory\npurpose: Inventory manager.\nspec:\n  parent: {apiVersion: " + api + ", kind: Manager, namespace: \"\", name: root}\n  owns: [src/inventory/]\n  instructions: orders-secret-manager\n",
		".markitect/model/orders/cancel.yaml":           "apiVersion: " + api + "\nkind: Statement\nmetadata:\n  name: cancel-order\n  namespace: orders\npurpose: Cancel an order.\nspec:\n  category: use-case\n  description: Cancel before shipment.\n  requires: [{apiVersion: " + api + ", kind: Statement, namespace: inventory, name: release-reservation}]\n",
		".markitect/model/orders/rename.yaml":           "apiVersion: " + api + "\nkind: IdentityChange\nmetadata:\n  name: rename-old-cancel\n  namespace: orders\npurpose: Record a declared rename.\nspec:\n  operation: renamed\n  previous: {apiVersion: " + api + ", kind: Statement, namespace: orders, name: old-cancel-order}\n  subject: {apiVersion: " + api + ", kind: Statement, namespace: orders, name: cancel-order}\n  reason: The canonical statement was renamed.\n  actorManager: {apiVersion: " + api + ", kind: Manager, namespace: orders, name: orders}\n  public: false\n",
		".markitect/model/inventory/release.yaml":       "apiVersion: " + api + "\nkind: Statement\nmetadata:\n  name: release-reservation\n  namespace: inventory\npurpose: Release reservation.\nspec:\n  category: rule\n  description: Release reservation once.\n  public: true\n  requires: [{apiVersion: " + api + ", kind: Statement, namespace: inventory, name: inventory-private-secret}]\n",
		".markitect/model/inventory/secret.yaml":        "apiVersion: " + api + "\nkind: Statement\nmetadata:\n  name: inventory-private-secret\n  namespace: inventory\npurpose: Internal guard.\nspec:\n  category: rule\n  description: SECRET inventory-private-secret\n",
		".markitect/model/orders/artifact.yaml":         "apiVersion: " + api + "\nkind: Artifact\nmetadata:\n  name: cancel-order-code\n  namespace: orders\npurpose: Cancellation source.\nspec:\n  role: implementation\n  realizes: [{apiVersion: " + api + ", kind: Statement, namespace: orders, name: cancel-order}]\n  paths: [src/orders/cancel.go]\n  required: true\n" + checkRef(api, withCheck),
		".markitect/model/inventory/artifact.yaml":      "apiVersion: " + api + "\nkind: Artifact\nmetadata:\n  name: inventory-secret-code\n  namespace: inventory\npurpose: Secret source.\nspec:\n  role: implementation\n  realizes: [{apiVersion: " + api + ", kind: Statement, namespace: inventory, name: inventory-private-secret}]\n  paths: [src/inventory/secret.go]\n  required: true\n",
		"src/orders/cancel.go":                          "package orders\n",
		"src/inventory/secret.go":                       "package inventory\n",
		"src/inventory/release.go":                      "package inventory\n",
	}
	if withCheck {
		files[".markitect/model/orders/check.yaml"] = "apiVersion: " + api + "\nkind: Check\nmetadata:\n  name: cancel-order-tests\n  namespace: orders\npurpose: Check cancellation.\nspec:\n  command: [go, test, ./orders]\n  uses: [{apiVersion: " + api + ", kind: Statement, namespace: orders, name: cancel-order}]\n  limitation: Does not prove production behavior.\n"
	}
	if withUnknown {
		files["src/unassigned.txt"] = "unassigned\n"
	}
	s := &snapshot.Snapshot{ID: "fixture-revision", Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, content := range files {
		s.Files[path] = []byte(content)
		s.Modes[path] = snapshot.RegularMode
	}
	p, err := projectwork.FromSnapshot("C:/fixture", s)
	if err != nil {
		t.Fatalf("build typed project fixture: %v", err)
	}
	return p
}

func checkRef(api string, enabled bool) string {
	if !enabled {
		return ""
	}
	return "  checks: [{apiVersion: " + api + ", kind: Check, namespace: orders, name: cancel-order-tests}]\n"
}
func rootOwner(unknown bool) string {
	if unknown {
		return "docs/"
	}
	return "."
}
func id(kind, namespace, name string) string {
	return (core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: kind, Namespace: namespace, Name: name}).Key()
}
