package projectapp

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/knowledgeevidence"
	"github.com/Glacius-Labs/Markitect/internal/host/projectgraph"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestKnowledgeRequiresRootAndLoader(t *testing.T) {
	if _, err := (Operations{}).Knowledge(KnowledgeOperation{}); err == nil {
		t.Fatal("missing root accepted")
	}
	if _, err := (Operations{}).Knowledge(KnowledgeOperation{Selection: Selection{Root: t.TempDir()}}); err == nil {
		t.Fatal("missing loader accepted")
	}
}

func TestKnowledgeFacadeUsesFixedSourceAndNoInvoker(t *testing.T) {
	root := makePlanFixture(t)
	revision := gitOutput(t, root, "rev-parse", "HEAD")
	manager := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Name: "project-owner"}.Key()
	invoker := &countingInvoker{}
	loads := 0
	operations := Operations{Host: projectrun.Host{Load: func(selectedRoot, selectedRevision string) (*projectwork.Project, error) {
		loads++
		if selectedRoot != root || selectedRevision != revision {
			t.Fatalf("source selection lost: %s %s", selectedRoot, selectedRevision)
		}
		return projectwork.Load(selectedRoot, selectedRevision)
	}}, Invoker: invoker}
	request := KnowledgeOperation{Selection: Selection{Root: root, Revision: revision}, Scope: projectgraph.Selection{ManagerID: manager}, Query: projectgraph.Request{Action: projectgraph.ActionGraph}}
	first, err := operations.Knowledge(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := operations.Knowledge(request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || first.Graph.Binding.GraphDigest == "" || first.Evidence.CaptureDigest == "" {
		t.Fatal("repeated fixed selection was not reproducible and bound")
	}
	if loads != 2 || invoker.calls != 0 {
		t.Fatalf("read-only facade load/invocations: %d/%d", loads, invoker.calls)
	}
	request.Records = knowledgeevidence.Selection{RunIDs: []string{"missing-run"}}
	missing, err := operations.Knowledge(request)
	if err != nil {
		t.Fatal(err)
	}
	if missing.Evidence.Completeness != "partial" || len(missing.Evidence.Unknown) == 0 {
		t.Fatal("explicit missing record lost uncertainty")
	}
	request.Scope = projectgraph.Selection{}
	if _, err := operations.Knowledge(request); err == nil {
		t.Fatal("implicit whole-project scope accepted")
	}
}
