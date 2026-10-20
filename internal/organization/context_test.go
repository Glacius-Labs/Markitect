package organization_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func contextID(pkg, id string) model.Identity { return model.Identity{Package: pkg, ID: id} }

func contextInput(t *testing.T) model.Input {
	t.Helper()
	input := localFixture(t, nil, "a/child", "empty", "software")
	for _, d := range []struct{ pkg, id, imports, body string }{
		{"a", "own", "[{package: software}, {package: b}]", "Own [[b/near]] and again [[b/near]]."},
		{"a/child", "child", "[{package: software}]", "Child stays separate."},
		{"b", "near", "[{package: software}, {package: c}]", "Neighbor [[c/far]]."},
		{"c", "far", "[{package: software}]", "Second hop stays inspectable."},
		{"incoming", "sender", "[{package: software}, {package: a}]", "Incoming [[a/own]]."},
	} {
		input.Root.Entries = append(input.Root.Entries, model.Entry{
			Path: "model/" + d.pkg + "/" + d.id + ".yaml", Kind: model.Regular,
			Bytes: []byte(fmt.Sprintf("imports: %s\nid: %s\nkind: software/Concept\nbody: '%s'\nspec: {}\n", d.imports, d.id, d.body)),
		})
	}
	return input
}

func requireContext(t *testing.T, checked model.Result, pkg string, scope ...organization.Setting[organization.ScopeRule]) organization.ContextSelection {
	t.Helper()
	selection, err := organization.SelectContext(checked, pkg, scope)
	if err != nil {
		t.Fatal(err)
	}
	return selection
}

func TestContextDirectLinksAndContractsWithoutSecondBusinessHop(t *testing.T) {
	checked := checkFixture(t, contextInput(t))
	selection := requireContext(t, checked, "a")
	if selection.Package() != "a" || selection.ModelID() != checked.Models[0].ID || selection.ModelDigest() != checked.Models[0].Digest {
		t.Fatal("context lost its exact local model/Manager binding")
	}
	if first := selection.Definitions()[0]; first.Definition.Identity != contextID("a", "own") || first.Reasons[0].Kind != organization.OwnDefinition {
		t.Fatalf("own Definition not first: %+v", first)
	}
	neighbor, found := selection.Definition(contextID("b", "near"))
	if !found || len(neighbor.Reasons) != 2 || neighbor.Reasons[0].Kind != organization.OutgoingReference || neighbor.Reasons[0].At != "/bodyLinks/0" || neighbor.Reasons[1].At != "/bodyLinks/1" {
		t.Fatalf("direct repeated outgoing reasons lost: %+v", neighbor)
	}
	incoming, found := selection.Definition(contextID("incoming", "sender"))
	if !found || len(incoming.Reasons) != 1 || incoming.Reasons[0].Kind != organization.IncomingReference || incoming.Reasons[0].From != incoming.Definition.Identity || incoming.Reasons[0].To != contextID("a", "own") {
		t.Fatalf("direct incoming direction lost: %+v", incoming)
	}
	for _, excluded := range []model.Identity{contextID("a/child", "child"), contextID("c", "far")} {
		if _, found := selection.Definition(excluded); found {
			t.Errorf("child or second business hop included: %+v", excluded)
		}
	}
	contract, found := selection.Definition(contextID("software", "Concept"))
	if !found || contract.Definition.ModelID != "software-basics" || contract.ModelDigest == "" || len(contract.Reasons) != 3 {
		t.Fatalf("bound imported Kind contract or reasons lost: %+v", contract)
	}
	for _, path := range []string{".", "empty", "software"} {
		if got := requireContext(t, checked, path); len(got.Definitions()) != 0 || got.Package() != path {
			t.Fatalf("root/empty local Package gained foreign or descendant content: %s %+v", path, got.Definitions())
		}
	}
	// The exact checked content, including original body and optional spans,
	// is carried through without a rendering/prose interpretation phase.
	own, _ := selection.Definition(contextID("a", "own"))
	if !reflect.DeepEqual(own.Definition, checked.Models[0].Definitions[0]) {
		t.Fatal("original own Definition changed")
	}
}
