package projectapp

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projecthistory"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

func TestModelOperationsDelegateProjectServices(t *testing.T) {
	root := makePlanFixture(t)
	operations := Operations{}
	selection := Selection{Root: root}

	check, err := operations.Check(selection)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if check.Source.ProjectDigest == "" || check.Report.Digest == "" || len(check.Findings) != len(check.Report.Findings) {
		t.Fatalf("incomplete Check result: %#v", check)
	}
	index, err := operations.Index(selection)
	if err != nil || index.Digest != check.Report.Digest {
		t.Fatalf("Index digest = %q, err=%v; Check report digest = %q", index.Digest, err, check.Report.Digest)
	}
	if len(index.Managers) == 0 {
		t.Fatal("Index returned no fixture Manager")
	}
	managerContext, err := operations.Context(ContextOperation{Selection: selection, ManagerID: index.Managers[0].ID})
	if err != nil || managerContext.Manager.ID != index.Managers[0].ID {
		t.Fatalf("Context = %#v, err=%v", managerContext, err)
	}
	document, err := operations.Document(DocumentOperation{Selection: selection})
	if err != nil || !strings.Contains(document.Content, "# Facade fixture") || document.Path == "" || !strings.HasPrefix(document.Digest, "sha256:") || document.Written {
		t.Fatalf("Document = %+v, err=%v", document, err)
	}
	coverage, err := operations.Coverage(selection)
	if err != nil || coverage.Digest == "" {
		t.Fatalf("Coverage digest=%q, err=%v", coverage.Digest, err)
	}
	impact, err := operations.Impact(ImpactOperation{Root: root})
	if err != nil || impact.Digest == "" || impact.BaseDigest != impact.CandidateDigest {
		t.Fatalf("Impact = %#v, err=%v", impact, err)
	}
}

func TestContextReadsAcceptedHistoryWithoutPersistingIt(t *testing.T) {
	root := makePlanFixture(t)
	modelPath := filepath.Join(root, ".markitect", "model", "manager.yaml")
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "repository-wide engineering mandate", "repository-wide project mandate", 1)
	if updated == string(data) {
		t.Fatal("fixture Manager purpose was not found")
	}
	if err := os.WriteFile(modelPath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".markitect/model/manager.yaml")
	runGit(t, root, "commit", "-m", "change accepted Manager purpose")
	revision := gitOutput(t, root, "rev-parse", "HEAD")
	accepted, err := projecthistory.ReadAcceptedHistory(root, revision)
	if err != nil || len(accepted.Receipt.Bundles) != 1 {
		t.Fatalf("accepted history preview = %+v, err=%v", accepted.Receipt, err)
	}
	storePath := filepath.Join(root, filepath.FromSlash(".markitect/records/briefing/accepted/record.json"))
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Fatalf("accepted history preview unexpectedly persisted the store: %v", err)
	}
	report, err := (Operations{}).Index(Selection{Root: root, Revision: revision})
	if err != nil || len(report.Managers) == 0 {
		t.Fatalf("Index selected accepted revision: managers=%d err=%v", len(report.Managers), err)
	}
	managerID := report.Managers[0].ID
	if _, err := (Operations{}).Context(ContextOperation{Selection: Selection{Root: root}, ManagerID: managerID}); err != nil {
		t.Fatalf("Context with unpersisted accepted history: %v", err)
	}
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Fatalf("Context persisted accepted history: %v", err)
	}
	if _, err := projecthistory.EnsureAcceptedHistory(root, revision); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (Operations{}).Context(ContextOperation{Selection: Selection{Root: root}, ManagerID: managerID}); err != nil {
		t.Fatalf("Context with unchanged persisted history: %v", err)
	}
	after, err := os.ReadFile(storePath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("Context changed an unchanged history store: err=%v", err)
	}
}

func TestDismissedAcceptedEventRemainsInApplicationManagerBriefing(t *testing.T) {
	root := deliveryFixture(t)
	operations := NewOperations()
	statementPath := ".markitect/model/orders/statement.yaml"
	statement, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(statementPath)))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(statement), "Implement the orders artifact.", "Implement the orders artifact with audited invariants.", 1)
	if updated == string(statement) {
		t.Fatal("Orders statement fixture text was not found")
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(statementPath)), []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatalf("render changed accepted model document: %v", err)
	}
	deliveryWrite(t, root, document.Path, document.Content)
	deliveryGit(t, root, "add", statementPath, document.Path)
	deliveryGit(t, root, "commit", "-m", "accept Orders model clarification")

	ordersID := deliveryManagerID("orders", "orders")
	before, err := operations.BriefList(root, ordersID)
	if err != nil {
		t.Fatalf("list accepted Orders briefing: %v", err)
	}
	var beforeList struct {
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
		ContextDigest string `json:"contextDigest"`
		StateDigest   string `json:"stateDigest"`
	}
	if err := decodeBriefList(before, &beforeList); err != nil {
		t.Fatal(err)
	}
	if len(beforeList.Events) != 1 || beforeList.ContextDigest == "" || beforeList.StateDigest == "" {
		t.Fatalf("fixture did not create one accepted Orders event: %+v", beforeList)
	}
	if _, err := operations.BriefDismiss(root, BriefInput{Event: beforeList.Events[0].ID, Manager: ordersID, Expect: beforeList.StateDigest, Write: true}); err != nil {
		t.Fatalf("dismiss accepted event through Application: %v", err)
	}
	after, err := operations.BriefList(root, ordersID)
	if err != nil {
		t.Fatalf("list later Orders briefing: %v", err)
	}
	var afterList struct {
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
		ContextDigest string `json:"contextDigest"`
	}
	if err := decodeBriefList(after, &afterList); err != nil {
		t.Fatal(err)
	}
	if len(afterList.Events) != 1 || afterList.Events[0].ID != beforeList.Events[0].ID || afterList.ContextDigest != beforeList.ContextDigest {
		t.Fatalf("dismissal removed accepted context from later Manager briefing: before=%+v after=%+v", beforeList, afterList)
	}
}

func TestDraftModelEditDoesNotInheritHistoryForUnbriefedTargetRevision(t *testing.T) {
	root := deliveryFixture(t)
	statementPath := ".markitect/model/orders/statement.yaml"
	path := filepath.Join(root, filepath.FromSlash(statementPath))
	statement, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	acceptedText := strings.Replace(string(statement), "Implement the orders artifact.", "Implement the orders artifact with audited invariants.", 1)
	if acceptedText == string(statement) {
		t.Fatal("Orders statement fixture text was not found")
	}
	if err := os.WriteFile(path, []byte(acceptedText), 0o600); err != nil {
		t.Fatal(err)
	}
	operations := NewOperations()
	document, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, document.Path, document.Content)
	deliveryGit(t, root, "add", statementPath, document.Path)
	deliveryGit(t, root, "commit", "-m", "accept Orders model clarification")
	acceptedRevision := deliveryGit(t, root, "rev-parse", "HEAD")
	receipt, err := projecthistory.EnsureAcceptedHistory(root, acceptedRevision)
	if err != nil || len(receipt.Bundles) != 1 || len(receipt.Bundles[0].Events) == 0 {
		t.Fatalf("create accepted history for the first model change: bundles=%d err=%v", len(receipt.Bundles), err)
	}

	updated := strings.Replace(acceptedText, "audited invariants", "reconciled invariants", 1)
	if updated == acceptedText {
		t.Fatal("could not create an unbriefed follow-up model revision")
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err = operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, document.Path, document.Content)
	deliveryGit(t, root, "add", statementPath, document.Path)
	deliveryGit(t, root, "commit", "-m", "leave follow-up model change unbriefed")
	targetRevision := deliveryGit(t, root, "rev-parse", "HEAD")
	if targetRevision == acceptedRevision {
		t.Fatal("fixture did not create a distinct follow-up source revision")
	}
	report, err := operations.Index(Selection{Root: root, Revision: targetRevision})
	if err != nil || len(report.Managers) < 2 {
		t.Fatalf("load unbriefed target model: managers=%d err=%v", len(report.Managers), err)
	}
	managerID := deliveryManagerID("orders", "orders")
	if _, err := operations.Host.BriefingForManager(root, targetRevision, report.ModelDigest, managerID, nil); err == nil {
		t.Fatal("fixture unexpectedly has accepted briefing history for the unaccepted target model")
	}
}

func decodeBriefList(value any, target any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func TestInitAndEditOperationsPreservePreviewAndExpectedDigestGuards(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-b", "facade-init")
	runGit(t, root, "config", "user.email", "projectapp@example.test")
	runGit(t, root, "config", "user.name", "Project App Test")
	operations := Operations{}
	initPlan, err := operations.Init(InitOperation{Root: root, Name: "Facade Init"})
	if err != nil || initPlan.Digest == "" || initPlan.Written != nil {
		t.Fatalf("Init preview = %#v, err=%v", initPlan, err)
	}
	if _, err := operations.Init(InitOperation{Root: root, Name: "Facade Init", Write: true, ExpectedDigest: "stale"}); !errors.Is(err, projectrun.ErrStale) {
		t.Fatalf("Init write with a stale digest = %v, want ErrStale", err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 {
		t.Fatalf("stale Init write changed the repository: %d entries", len(entries))
	}
	written, err := operations.Init(InitOperation{Root: root, Name: "Facade Init", Write: true, ExpectedDigest: initPlan.Digest})
	if err != nil || written.Digest != initPlan.Digest {
		t.Fatalf("Init write = %#v, err=%v", written, err)
	}
	runGit(t, root, "add", "--all")
	runGit(t, root, "commit", "-m", "init")
	document, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}, Write: true, ExpectedDigest: "sha256:stale"}); !errors.Is(err, projectrun.ErrStale) {
		t.Fatalf("Document write with a stale digest = %v, want ErrStale", err)
	}
	if rewritten, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}, Write: true, ExpectedDigest: document.Digest}); err != nil || !rewritten.Written || rewritten.Digest != document.Digest {
		t.Fatalf("Document write = %+v, err=%v", rewritten, err)
	}

	fixture := makePlanFixture(t)
	project, err := projectwork.Load(fixture, "")
	if err != nil {
		t.Fatal(err)
	}
	mutation := projectvalue.Mutation{
		APIVersion: projectvalue.APIVersion, BaseDigest: project.Digest, Actor: projectvalue.HumanActor,
		Goal: "Configure a fixture runtime", Files: []projectvalue.FileChange{{Path: projectvalue.RuntimePath, Content: "mode: deliberately-invalid\n"}},
	}
	selection := Selection{Root: fixture}
	preview, err := operations.Edit(EditOperation{Selection: selection, Mutation: mutation})
	if err != nil || preview.Digest == "" {
		t.Fatalf("Edit preview = %#v, err=%v", preview, err)
	}
	if _, err := operations.Edit(EditOperation{Selection: selection, Mutation: mutation, Write: true, ExpectedDigest: "sha256:stale"}); err == nil || !strings.Contains(err.Error(), "exact edit plan digest") {
		t.Fatalf("stale Edit write error = %v", err)
	}
	edited, err := operations.Edit(EditOperation{Selection: selection, Mutation: mutation, Write: true, ExpectedDigest: preview.Digest})
	if err != nil || edited.Digest != preview.Digest {
		t.Fatalf("Edit write = %#v, err=%v", edited, err)
	}
}

func TestModelOperationsRequireExplicitRoots(t *testing.T) {
	operations := Operations{}
	if _, err := operations.Init(InitOperation{Name: "missing root"}); err == nil {
		t.Fatal("Init accepted an empty root")
	}
	if _, err := operations.Check(Selection{}); err == nil {
		t.Fatal("Check accepted an empty root")
	}
	if _, err := operations.Coverage(Selection{}); err == nil {
		t.Fatal("Coverage accepted an empty root")
	}
	if _, err := operations.Impact(ImpactOperation{}); err == nil {
		t.Fatal("Impact accepted an empty root")
	}
}
