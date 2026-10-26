package organization_test

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// interfaceContextInput adds Policy, a Kind that implements Rule and declares
// its scope Property itself, and one global Policy item to the rich fixture.
func interfaceContextInput(t *testing.T) model.Input {
	t.Helper()
	input := richContextInput(t)
	input.Root.Entries = append(input.Root.Entries,
		model.Entry{Path: "model/types/policy.yaml", Kind: model.Regular,
			Bytes: []byte("imports: [{package: markitect/core, as: Core}]\nid: Policy\nkind: Core.Kind\nbody: A Rule that every Manager in its scope follows.\nspec: {implements: [Rule], properties: [applies]}\n")},
		model.Entry{Path: "model/rules/derived.yaml", Kind: model.Regular,
			Bytes: []byte("imports: [{package: types}]\nid: derived\nkind: types/Policy\nbody: 'Derived global policy.'\nspec: {covers: ['.']}\n")},
	)
	return input
}

func TestContextModeledScopeSelectsImplementingKinds(t *testing.T) {
	checked := checkFixture(t, interfaceContextInput(t))
	// A selector on the interface Kind also selects Definitions of a Kind that
	// implements it, and their type contract includes the interface.
	selection := requireContext(t, checked, "a", scopeSelector())
	derived, found := selection.Definition(contextID("rules", "derived"))
	if !found || derived.Reasons[0].Kind != organization.ModeledScope || derived.Reasons[0].At != "/spec/covers/0" {
		t.Fatalf("interface selector missed the implementing Definition: %+v", derived)
	}
	for _, contract := range []string{"Policy", "Rule", "applies"} {
		if _, found := selection.Definition(contextID("types", contract)); !found {
			t.Errorf("type contract %s of the implementing Definition not selected", contract)
		}
	}
	// A selector on the implementing Kind selects only its own Definitions.
	policySelector := setting(organization.ScopeRule{Kind: contextID("types", "Policy"), Property: contextID("types", "applies")}, "/scope/0")
	selection = requireContext(t, checked, "a", policySelector)
	if _, found := selection.Definition(contextID("rules", "derived")); !found {
		t.Fatal("scope field refused on the implementing Kind")
	}
	if _, found := selection.Definition(contextID("rules", "global")); found {
		t.Fatal("an implementing-Kind selector widened to Definitions of the interface")
	}
}
