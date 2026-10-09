package projectapp

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
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
	if err != nil || !strings.Contains(document, "# Facade fixture") {
		t.Fatalf("Document length=%d, err=%v", len(document), err)
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
	if _, err := operations.Init(InitOperation{Root: root, Name: "Facade Init", Write: true}); err != nil {
		t.Fatalf("Init write: %v", err)
	}

	fixture := makePlanFixture(t)
	project, err := projectwork.Load(fixture, "")
	if err != nil {
		t.Fatal(err)
	}
	mutation := projectwork.Mutation{
		APIVersion: projectwork.APIVersion, BaseDigest: project.Digest, Actor: projectwork.HumanActor,
		Goal: "Configure a fixture runtime", Files: []projectwork.FileChange{{Path: projectwork.RuntimePath, Content: "mode: deliberately-invalid\n"}},
	}
	selection := Selection{Root: fixture}
	preview, err := operations.Edit(EditOperation{Selection: selection, Mutation: mutation})
	if err != nil || preview.Digest == "" {
		t.Fatalf("Edit preview = %#v, err=%v", preview, err)
	}
	if _, err := operations.Edit(EditOperation{Selection: selection, Mutation: mutation, Write: true, ExpectedDigest: "sha256:stale"}); err == nil || !strings.Contains(err.Error(), "exact edit plan digest") {
		t.Fatalf("stale Edit write error = %v", err)
	}
	written, err := operations.Edit(EditOperation{Selection: selection, Mutation: mutation, Write: true, ExpectedDigest: preview.Digest})
	if err != nil || written.Digest != preview.Digest {
		t.Fatalf("Edit write = %#v, err=%v", written, err)
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
