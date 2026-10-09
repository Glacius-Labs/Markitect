package knowledgeevidence

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/internal/host/projectgraph"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectknowledge"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestReadUsesValidatedExplicitExplorationAndPreservesManagerPrivacy(t *testing.T) {
	root := t.TempDir()
	initTestRepo(t, root)
	record := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "work-one", Status: projectexplore.StatusActive,
		Request: "PRIVATE REQUEST BODY", CreatedAgainst: "sha256:" + strings.Repeat("a", 64),
		Scopes:    []projectexplore.Scope{{ID: "orders", Name: "PRIVATE SCOPE NAME", Goal: "PRIVATE SCOPE GOAL", Operation: "implement", ManagerIDs: []string{"foreign-peer-sentinel", "orders"}}},
		Decisions: []projectexplore.Decision{{ID: "decision-one", ScopeIDs: []string{"orders"}, Question: "PRIVATE DECISION QUESTION", Blocking: true, Status: "open"}},
		Drafts:    []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	encoded, err := projectexplore.EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := projectexplore.RecordPath(record.ID)
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(fullPath)
	project := &projectwork.Project{Root: root, Revision: "revision-one", Digest: "sha256:" + strings.Repeat("b", 64), Model: core.Model{Digest: "sha256:" + strings.Repeat("c", 64)}, Snapshot: &snapshot.Snapshot{ID: "revision-one", Files: map[string][]byte{}}}
	got, err := Read(project, Scope{ManagerID: "orders", VisiblePaths: []string{"src/orders.go"}}, Selection{ExplorationIDs: []string{record.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Completeness != projectknowledge.FactPartial || got.Facts.Digest == "" || got.CaptureDigest == "" || got.CaptureDigest == got.Facts.Digest {
		t.Fatalf("unexpected capture binding: %#v", got)
	}
	var rootFact, scopeFact, decisionFact bool
	for _, fact := range got.Facts.Facts {
		switch fact.Kind {
		case "Exploration":
			rootFact = true
		case "WorkScope":
			scopeFact = true
		case "WorkDecision":
			decisionFact = true
		}
	}
	if !rootFact || !scopeFact || !decisionFact {
		t.Fatalf("missing scoped exploration facts: %#v", got.Facts.Facts)
	}
	for _, fact := range got.Facts.Facts {
		if fact.Kind == "WorkScope" && bytes.Contains(fact.Properties["managerIds"], []byte("foreign-peer-sentinel")) {
			t.Fatal("shared-scope peer Manager ID leaked into Manager-scoped evidence")
		}
	}
	if _, err := projectknowledge.Build(project.Model, got.Facts, projectknowledge.Scope{ID: "manager:orders", Nodes: []projectknowledge.NodeSelection{{ID: "record/exploration/work-one", Fields: []string{"status", "bindingStatus"}}, {ID: "record/exploration/work-one/scope/orders", Fields: []string{"operation"}}}, Edges: []projectknowledge.EdgeKey{{From: "record/exploration/work-one", To: "record/exploration/work-one/scope/orders", Property: "containsScope"}}}); err != nil {
		t.Fatalf("captured facts were not consumable by the pure graph: %v", err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"PRIVATE REQUEST BODY", "PRIVATE SCOPE NAME", "PRIVATE SCOPE GOAL", "PRIVATE DECISION QUESTION"} {
		if bytes.Contains(data, []byte(private)) {
			t.Fatalf("private text leaked: %q", private)
		}
	}
	after, _ := os.ReadFile(fullPath)
	if !bytes.Equal(before, after) {
		t.Fatal("read changed the selected operational record")
	}
}

func TestReadMissingAndForeignExplorationAreSameUnavailableUnknown(t *testing.T) {
	root := t.TempDir()
	initTestRepo(t, root)
	project := &projectwork.Project{Root: root, Revision: "revision-one", Digest: "sha256:" + strings.Repeat("b", 64), Model: core.Model{Digest: "sha256:" + strings.Repeat("c", 64)}, Snapshot: &snapshot.Snapshot{ID: "revision-one", Files: map[string][]byte{}}}
	missing, err := Read(project, Scope{ManagerID: "orders", VisiblePaths: []string{"src/orders.go"}}, Selection{ExplorationIDs: []string{"missing"}})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Completeness != projectknowledge.FactPartial || len(missing.Unknown) != 1 || missing.Unknown[0].Reason != "selected exploration is unavailable" {
		t.Fatalf("missing record was not explicit unknown: %#v", missing)
	}

	foreign := projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: "foreign", Status: projectexplore.StatusActive, Request: "private", CreatedAgainst: "sha256:" + strings.Repeat("a", 64), Scopes: []projectexplore.Scope{{ID: "other", Name: "foreign", Goal: "foreign", Operation: "implement", ManagerIDs: []string{"other-manager"}}}, Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
	encoded, err := projectexplore.EncodeRecord(foreign)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := projectexplore.RecordPath(foreign.ID)
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	hidden, err := Read(project, Scope{ManagerID: "orders", VisiblePaths: []string{"src/orders.go"}}, Selection{ExplorationIDs: []string{foreign.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if hidden.Completeness != missing.Completeness || !bytes.Equal(mustJSON(t, hidden.Unknown), mustJSON(t, missing.Unknown)) || len(hidden.Facts.Facts) != 0 {
		t.Fatalf("foreign record differs from missing behavior: %#v", hidden)
	}
	corruptPath, _ := projectexplore.RecordPath("corrupt")
	corruptFull := filepath.Join(root, filepath.FromSlash(corruptPath))
	if err := os.WriteFile(corruptFull, []byte(`{"private":"CORRUPT FOREIGN BODY"}`), 0600); err != nil {
		t.Fatal(err)
	}
	corrupt, err := Read(project, Scope{ManagerID: "orders", VisiblePaths: []string{"src/orders.go"}}, Selection{ExplorationIDs: []string{"corrupt"}})
	if err != nil {
		t.Fatal(err)
	}
	if corrupt.CaptureDigest != missing.CaptureDigest || !bytes.Equal(mustJSON(t, corrupt.Unknown), mustJSON(t, missing.Unknown)) {
		t.Fatalf("corrupt foreign record differs from absent record: %#v", corrupt)
	}
	if bytes.Contains(mustJSON(t, corrupt), []byte("CORRUPT FOREIGN BODY")) {
		t.Fatal("corrupt foreign record body leaked")
	}
}

func TestReadRejectsAmbiguousScopeAndCorruptSelectedRecord(t *testing.T) {
	project := &projectwork.Project{Root: t.TempDir(), Revision: "revision-one", Digest: "sha256:" + strings.Repeat("b", 64), Model: core.Model{Digest: "sha256:" + strings.Repeat("c", 64)}, Snapshot: &snapshot.Snapshot{ID: "revision-one", Files: map[string][]byte{}}}
	if _, err := Read(project, Scope{}, Selection{}); err == nil {
		t.Fatal("accepted unscoped read")
	}
	if _, err := Read(project, Scope{ManagerID: "orders", WholeProject: true, VisiblePaths: []string{"src/orders.go"}}, Selection{}); err == nil {
		t.Fatal("accepted ambiguous scope")
	}
	path, _ := projectexplore.RecordPath("broken")
	fullPath := filepath.Join(project.Root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(`{"broken":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(project, Scope{WholeProject: true}, Selection{ExplorationIDs: []string{"broken"}}); err == nil || !strings.Contains(err.Error(), "invalid or unavailable") || strings.Contains(err.Error(), "private") {
		t.Fatalf("corrupt selected project record did not produce a safe actionable error: %v", err)
	}
}

func TestReadPlannedRunKeepsExpectedAndObservedTaskEdgesDistinct(t *testing.T) {
	root := t.TempDir()
	initTestRepo(t, root)
	if _, err := projectwork.Init(root, "Knowledge evidence run target", true); err != nil {
		t.Fatal(err)
	}
	managerID, _ := json.Marshal([]string{projectmodel.APIVersion, "Manager", "orders", "orders"})
	rootManagerID, _ := json.Marshal([]string{projectmodel.APIVersion, "Manager", "", "project-owner"})
	manifest := "apiVersion: " + projectwork.APIVersion + "\nname: Knowledge evidence run target\ncoverageMode: selected\nworkflowMode: empty\nacceptancePolicy: committed-model\nmodelFiles:\n  - .markitect/model/manager.yaml\n  - .markitect/model/orders/manager.yaml\n  - .markitect/model/orders/statement.yaml\ninventoryRoots:\n  - src\nexclusions: []\ntransitionalExclusions: []\n"
	if err := os.WriteFile(filepath.Join(root, projectwork.ManifestPath), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".markitect/model/orders"), 0755); err != nil {
		t.Fatal(err)
	}
	childManager := "apiVersion: " + projectmodel.APIVersion + "\nkind: Manager\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Own the orders behavior.\nspec:\n  parent:\n    apiVersion: " + projectmodel.APIVersion + "\n    kind: Manager\n    namespace: \"\"\n    name: project-owner\n  owns: [src/orders/]\n"
	statement := "apiVersion: " + projectmodel.APIVersion + "\nkind: Statement\nmetadata:\n  name: orders-work\n  namespace: orders\npurpose: Implement the owned orders behavior.\nspec:\n  category: concept\n  description: Implement the owned orders behavior.\n"
	for path, body := range map[string]string{".markitect/model/orders/manager.yaml": childManager, ".markitect/model/orders/statement.yaml": statement, "src/orders/example.txt": "orders\n"} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	agent := "    command: go\n    model: fixture\n    providerVersion: fixture-v1\n    timeout: 1m\n    maxStdoutBytes: 1024\n    maxStderrBytes: 1024\n    pricing:\n      inputMicrosPerMillion: 1\n      outputMicrosPerMillion: 1\n"
	runtime := "apiVersion: " + projectrun.APIVersion + "\nmode: controlled-local\nagents:\n  " + strconv.Quote(string(managerID)) + ":\n" + agent + "  " + strconv.Quote(string(rootManagerID)) + ":\n" + agent + "limits:\n  maxDepth: 2\n  maxStarts: 5\n  maxRetries: 0\n  maxParallel: 1\n  maxDuration: 1m\n  maxCostMicros: 1000\n  maxCandidateFileBytes: 1024\n  maxCandidateBytes: 1024\n"
	if err := os.WriteFile(filepath.Join(root, projectrun.RuntimePath), []byte(runtime), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "prepared project model")
	statementID, _ := json.Marshal([]string{projectmodel.APIVersion, "Statement", "orders", "orders-work"})
	host := projectrun.Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot, PlanEdit: projectwork.PlanEdit, ApplyEdit: projectwork.ApplyEdit}
	plan, err := projectrun.Plan(host, root, "", projectrun.PlanRequest{Goal: "Implement the owned orders behavior.", Managers: []string{string(managerID)}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	project, err := projectwork.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(project, Scope{ManagerID: string(managerID), VisibleDefinitionIDs: []string{string(managerID), string(statementID)}, VisiblePaths: []string{"src/orders/example.txt"}}, Selection{RunIDs: []string{plan.ID}})
	if err != nil {
		t.Fatal(err)
	}
	planNode, runNode := "record/plan/"+plan.ID, "record/run/"+plan.ID
	taskNode := ""
	for _, task := range plan.Managers {
		if task.ManagerID == string(managerID) {
			taskNode = "record/task/" + plan.ID + "/" + task.ID
		}
	}
	var hasExpectedRun, hasObservedRun, hasPlannedTask, hasObservedTask bool
	for _, relation := range got.Facts.Relations {
		switch {
		case relation.From == planNode && relation.To == runNode && relation.Property == "expectsRun":
			hasExpectedRun = true
		case relation.From == runNode && relation.To == planNode && relation.Property == "usesPlan":
			hasObservedRun = true
		case relation.From == planNode && relation.To == taskNode && relation.Property == "plansTask":
			hasPlannedTask = true
		case relation.From == runNode && relation.To == taskNode && relation.Property == "hasTask":
			hasObservedTask = true
		}
	}
	if !hasExpectedRun || hasObservedRun || !hasPlannedTask || hasObservedTask {
		t.Fatalf("planned/missing run was projected as observed evidence: expectedRun=%v observedRun=%v plannedTask=%v observedTask=%v facts=%+v", hasExpectedRun, hasObservedRun, hasPlannedTask, hasObservedTask, got.Facts)
	}
	selectedScope := projectgraph.Selection{ManagerID: string(managerID)}
	scope, err := projectgraph.SafeScope(project, selectedScope)
	if err != nil {
		t.Fatal(err)
	}
	selectedIDs := make([]string, 0, len(got.Facts.Facts))
	for _, fact := range got.Facts.Facts {
		selectedIDs = append(selectedIDs, fact.ID)
	}
	bindings := make([]projectgraph.RecordBinding, 0, len(got.SourceBindings))
	for _, source := range got.SourceBindings {
		bindings = append(bindings, projectgraph.RecordBinding{Kind: source.Kind, Digest: source.Digest, Schema: source.Schema, Source: source.Source, ModelDigest: source.ModelDigest, Revision: source.Revision, RuntimeDigest: source.RuntimeDigest, DigestKind: source.DigestKind})
	}
	view, err := projectgraph.Build(project, selectedScope, &projectgraph.ExtraFacts{ScopeID: scope.ID, SelectedIDs: selectedIDs, Facts: got.Facts, CaptureDigest: got.CaptureDigest, Completeness: got.Completeness, EvidenceSelected: true, SourceBindings: bindings})
	if err != nil {
		t.Fatalf("selected record capture did not survive the source-bound project graph: %v", err)
	}
	if _, err := view.Index().Node(taskNode); err != nil {
		t.Fatalf("planned task fact was dropped from the project graph: %v", err)
	}
	encodedCapture, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encodedCapture, []byte(plan.InitialCandidateID)) {
		t.Fatal("Manager-scoped capture exposed the operational candidate ID")
	}
}

func TestFilteredDefinitionsHidesUnselectedCheckIdentity(t *testing.T) {
	got := filteredDefinitions([]string{"owned-check", "foreign-check-sentinel"}, map[string]bool{"owned-check": true}, false)
	if len(got) != 1 || got[0] != "owned-check" || containsString(got, "foreign-check-sentinel") {
		t.Fatalf("unselected check identity escaped Manager scoping: %v", got)
	}
}

func TestHistoryProjectionJoinsOnlyVisibleEventsAndCurrentDefinitions(t *testing.T) {
	current := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "orders", Name: "current"}.Key()
	removed := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "orders", Name: "removed"}.Key()
	b := factBuilder{out: &Capture{SourceBindings: []SourceBinding{}}, scope: Scope{ManagerID: "orders", VisibleDefinitionIDs: []string{current}}, defs: map[string]bool{current: true}, currentDefs: map[string]bool{current: true}, modelDigest: "model", projectRevision: "revision", added: map[string]bool{}}
	briefings := []projectbriefing.Briefing{{ID: "brief", Revision: "revision", ModelDigest: "model", EventIDs: []string{"visible-event", "hidden-event"}, Summary: "summary"}}
	events := []projectbriefing.Event{
		{ID: "visible-event", Digest: "sha256:visible", DefinitionID: core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "orders", Name: "current"}, AffectedManagers: []string{"orders"}, Change: "modified"},
		{ID: "hidden-event", Digest: "sha256:hidden", DefinitionID: core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "orders", Name: "removed"}, AffectedManagers: []string{"inventory"}, Change: "removed"},
	}
	b.addHistory(briefings, events, "sha256:history", projectbriefing.Store{})
	eventIDs := []string{}
	for _, fact := range b.facts {
		if fact.Kind == "ModelHistoryEvent" {
			eventIDs = append(eventIDs, fact.ID)
		}
	}
	if len(eventIDs) != 1 || eventIDs[0] != "record/history/orders/event/visible-event" {
		t.Fatalf("Manager history admitted hidden event: %v", eventIDs)
	}
	var hasBriefEvent, hasCurrentDefinition, hasRemovedDefinition bool
	for _, edge := range b.relations {
		if edge.From == "record/history/orders/briefing/brief" && edge.To == eventIDs[0] && edge.Property == "containsEvent" {
			hasBriefEvent = true
		}
		if edge.From == eventIDs[0] && edge.To == current && edge.Property == "changedDefinition" {
			hasCurrentDefinition = true
		}
		if edge.Property == "changedDefinition" && edge.To == removed {
			hasRemovedDefinition = true
		}
	}
	if !hasBriefEvent || !hasCurrentDefinition || hasRemovedDefinition {
		t.Fatalf("history joins did not preserve exact current-only semantics: %+v", b.relations)
	}
}

func TestReadProjectsValidatedBrownfieldContradictionWithoutSourceBody(t *testing.T) {
	root := t.TempDir()
	initTestRepo(t, root)
	privatePath := filepath.Join(root, "src", "private-source.txt")
	if err := os.MkdirAll(filepath.Dir(privatePath), 0755); err != nil {
		t.Fatal(err)
	}
	const sourceBody = "SOURCE_FILE_BODY_SENTINEL\n"
	if err := os.WriteFile(privatePath, []byte(sourceBody), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "src/private-source.txt")
	gitRun(t, root, "commit", "--quiet", "-m", "selected source")
	sourceCommit := gitRun(t, root, "rev-parse", "HEAD")
	if _, err := projectwork.Init(root, "Knowledge evidence target", true); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "initialize target")
	targetRevision := gitRun(t, root, "rev-parse", "HEAD")
	target, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := projectadoption.Discover(root, projectadoption.DiscoveryRequest{APIVersion: projectadoption.DiscoveryVersion, ID: "kg-discovery", Purpose: "PRIVATE discovery purpose", Review: "PRIVATE discovery review", Commit: sourceCommit, ScopeRoots: []string{"src"}, Selected: []projectadoption.SelectedPath{{ID: "private-evidence", Path: "src/private-source.txt", Reason: "selected", Basis: "code"}}, Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := projectadoption.StartBrownfieldSession(root, target, discovery, []projectadoption.ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	managerID := session.TargetContext.RootManagerID
	iteration, err := projectadoption.BeginReverseIteration(root, target, session, projectadoption.ReverseIterationRequest{ID: "kg-root", ManagerID: managerID, EvidenceIDs: []string{"private-evidence"}, DelegationEvidenceIDs: []string{}, Purpose: "PRIVATE iteration purpose", Review: "PRIVATE review body"})
	if err != nil {
		t.Fatal(err)
	}
	lineRef := func() projectadoption.EvidenceRef {
		return projectadoption.EvidenceRef{EvidenceID: "private-evidence", StartLine: 1, EndLine: 1, Excerpt: "SOURCE_FILE_BODY_SENTINEL"}
	}
	blocking := true
	schemaDigest, _, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	report := projectadoption.Distillation{APIVersion: projectadoption.DistillationVersion, DiscoveryDigest: session.Source.Digest, Method: "human-review", SchemaDigest: schemaDigest,
		Claims: []projectadoption.Claim{
			{ID: "observed", ScopeID: "order-scope", Kind: "observation", Method: "static-source", Statement: "An order operation is present.", Evidence: []projectadoption.EvidenceRef{lineRef()}, Uncertainty: []string{}},
			{ID: "hypothesis", ScopeID: "order-scope", Kind: "hypothesis", Method: "synthesis", Statement: "The order operation is intended to be exclusive.", Evidence: []projectadoption.EvidenceRef{lineRef()}, Uncertainty: []string{}},
		}, Terms: []projectadoption.Term{}, Contradictions: []projectadoption.Contradiction{{ID: "conflict-one", ScopeID: "order-scope", ClaimIDs: []string{"hypothesis", "observed"}, QuestionID: "clarify-one", Description: "The supplied claims differ in kind and meaning."}},
		Questions: []projectadoption.Question{{ID: "clarify-one", ScopeID: "order-scope", Prompt: "PRIVATE question prompt", Alternatives: []string{"exclusive", "shared"}, ClaimIDs: []string{"hypothesis", "observed"}, Blocking: &blocking}},
		Scopes:    []projectadoption.ScopeProposal{{ID: "order-scope", Name: "Order scope", ClaimIDs: []string{"hypothesis", "observed"}}},
		Proposal:  projectadoption.ModelProposal{Goal: "Model the order scope", Files: []projectadoption.ProposedFile{{ScopeID: "order-scope", Path: ".markitect/model/orders/statement.yaml", Content: "apiVersion: example/v1\nkind: Statement\n"}}},
	}
	projectadoption.SealDistillation(&report)
	iteration, err = projectadoption.RecordManagerProposal(iteration, "kg-root", projectadoption.ManagerProposal{ManagerID: managerID, EvidenceIDs: []string{"private-evidence"}, Hierarchy: []projectadoption.ProposedManager{}, PublicContracts: []projectadoption.ManagerPublicContract{}, Report: report})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(root, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(root, iteration, session.Digest); err != nil {
		t.Fatal(err)
	}
	// Read the fixture through the actual loader. The session is fixed and the
	// exact validated record is then supplied to the graph adapter explicitly.
	loaded, err := projectadoption.LoadBrownfieldSession(root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(target, Scope{ManagerID: managerID, VisiblePaths: []string{"src/visible.go"}}, Selection{BrownfieldSessionIDs: []string{loaded.ID}})
	if err != nil {
		t.Fatal(err)
	}
	var claimKinds []string
	var contradiction, question, evidence bool
	for _, fact := range got.Facts.Facts {
		switch fact.Kind {
		case "BrownfieldClaim":
			var kind string
			_ = json.Unmarshal(fact.Properties["kind"], &kind)
			claimKinds = append(claimKinds, kind)
		case "BrownfieldContradiction":
			contradiction = true
		case "BrownfieldQuestion":
			question = true
		case "BrownfieldEvidenceRef":
			evidence = true
		}
	}
	if !contradiction || !question || !evidence || !containsString(claimKinds, "observation") || !containsString(claimKinds, "hypothesis") {
		t.Fatalf("Brownfield contradiction chain was not projected: %#v", got.Facts.Facts)
	}
	encoded, _ := json.Marshal(got)
	for _, private := range []string{sourceBody, "src/private-source.txt", "PRIVATE discovery purpose", "PRIVATE discovery review", "PRIVATE iteration purpose", "PRIVATE review body", "PRIVATE question prompt"} {
		if bytes.Contains(encoded, []byte(private)) {
			t.Fatalf("private Brownfield content leaked: %q", private)
		}
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func initTestRepo(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{{"init", "-b", "codex/knowledge-evidence-test"}, {"config", "user.name", "Knowledge Evidence Test"}, {"config", "user.email", "knowledge-evidence@example.invalid"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func gitRun(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
