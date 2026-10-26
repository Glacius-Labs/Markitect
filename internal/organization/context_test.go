package organization_test

import (
	"fmt"
	"reflect"
	"slices"
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
	if children := selection.Children(); len(children) != 1 || children[0].Package != "a/child" || len(children[0].Definitions) != 1 || children[0].Definitions[0].Identity != contextID("a/child", "child") {
		t.Fatalf("parent child area lost: %+v", children)
	}
	// The exact checked content, including original body and optional spans,
	// is carried through without a rendering/prose interpretation phase.
	own, _ := selection.Definition(contextID("a", "own"))
	if !reflect.DeepEqual(own.Definition, checked.Models[0].Definitions[0]) {
		t.Fatal("original own Definition changed")
	}
}

// Child areas index each direct child's whole subtree compactly, without
// entering the selected set: Root, a parent, an intermediate and a leaf.
func TestContextChildAreasIndexWholeSubtrees(t *testing.T) {
	input := contextInput(t)
	input.Root.Entries = append(input.Root.Entries, model.Entry{
		Path: "model/a/child/leaf/deep.yaml", Kind: model.Regular,
		Bytes: []byte("imports: [{package: software}]\nid: deep\nkind: software/Concept\ntitle: Deep title\nbody: Deep descendant.\nspec: {}\n"),
	})
	checked := checkFixture(t, input)
	areas := func(pkg string) map[string][]model.Identity {
		selection := requireContext(t, checked, pkg)
		result := make(map[string][]model.Identity)
		var order []string
		for _, child := range selection.Children() {
			order = append(order, child.Package)
			if child.Definitions == nil {
				t.Fatalf("%s: empty child area %q is not explicit", pkg, child.Package)
			}
			result[child.Package] = []model.Identity{}
			for _, summary := range child.Definitions {
				if _, selected := selection.Definition(summary.Identity); selected {
					t.Errorf("%s: child summary entered the selected set: %+v", pkg, summary.Identity)
				}
				if summary.Kind != contextID("software", "Concept") || summary.Source.Path == "" || summary.Source.Digest == "" {
					t.Errorf("%s: summary lost Kind or source: %+v", pkg, summary)
				}
				result[child.Package] = append(result[child.Package], summary.Identity)
			}
		}
		if !slices.IsSorted(order) {
			t.Errorf("%s: child areas not in topology order: %v", pkg, order)
		}
		return result
	}
	deep, own, child := contextID("a/child/leaf", "deep"), contextID("a", "own"), contextID("a/child", "child")
	for pkg, want := range map[string]map[string][]model.Identity{
		".": {
			"a": {child, deep, own}, "b": {contextID("b", "near")}, "c": {contextID("c", "far")},
			"empty": {}, "incoming": {contextID("incoming", "sender")}, "software": {},
		},
		"a":            {"a/child": {child, deep}},
		"a/child":      {"a/child/leaf": {deep}},
		"a/child/leaf": {},
	} {
		if got := areas(pkg); !reflect.DeepEqual(got, want) {
			t.Errorf("%s child areas = %v, want %v", pkg, got, want)
		}
	}
	if root := requireContext(t, checked, "."); len(root.Definitions()) != 0 {
		t.Fatalf("Root gained selected Definitions: %+v", root.Definitions())
	}
	if leaf := requireContext(t, checked, "a/child/leaf"); leaf.Children() != nil {
		t.Fatal("leaf has child areas")
	}
	if (organization.ContextSelection{}).Children() != nil {
		t.Fatal("zero selection exposes child areas")
	}
	// Returned summaries are copies: changing one leaves the selection intact.
	parent := requireContext(t, checked, "a/child")
	first := parent.Children()
	*first[0].Definitions[0].Title = "changed"
	first[0].Definitions[0].Identity = own
	if again := parent.Children()[0].Definitions[0]; *again.Title != "Deep title" || again.Identity != deep {
		t.Fatalf("child summary aliases the selection: %+v", again)
	}
}
