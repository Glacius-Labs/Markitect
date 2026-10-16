package organization_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func TestContextStructuredOccurrencesOptionalNestedContractsAndUnusedForeign(t *testing.T) {
	compiled := compileFixture(t, richContextInput(t))
	selection := requireContext(t, compiled, "a", scopeSelector())
	neighbor, _ := selection.Definition(contextID("b", "near"))
	var occurrences []string
	for _, reason := range neighbor.Reasons {
		if reason.Kind != organization.OutgoingReference || reason.From != contextID("a", "own") || reason.To != neighbor.Definition.Identity {
			t.Fatalf("wrong directed edge: %+v", reason)
		}
		occurrences = append(occurrences, reason.At)
	}
	if want := []string{"/bodyLinks/0", "/bodyLinks/1", "/spec/links/0", "/spec/links/1", "/spec/payload/links/0"}; !reflect.DeepEqual(occurrences, want) {
		t.Fatalf("structured/body occurrences: %q, want %q", occurrences, want)
	}
	for _, name := range []string{"Record", "Rule", "edge", "payload", "note", "unused", "data", "applies"} {
		if _, found := selection.Definition(contextID("types", name)); !found {
			t.Errorf("missing bound optional/nested contract %s", name)
		}
	}
	for _, excluded := range []model.Identity{contextID("c", "far"), contextID("a/child", "child"), contextID("aother", "other"), contextID("software", "Concept"), contextID("unused", "foreign")} {
		if _, found := selection.Definition(excluded); found {
			t.Errorf("second hop, hierarchy, unused foreign scope/incoming leaked: %+v", excluded)
		}
	}
	incoming, _ := selection.Definition(contextID("incoming", "sender"))
	if len(incoming.Reasons) != 2 || incoming.Reasons[0].Kind != organization.IncomingReference || incoming.Reasons[1].At != "/spec/payload/links/0" {
		t.Fatalf("nested direct local incoming lost: %+v", incoming)
	}
	for _, selected := range selection.Definitions() {
		original := findCompiledDefinition(t, &compiled, selected.Definition.Identity)
		if !reflect.DeepEqual(selected.Definition, *original) {
			t.Errorf("selected content/coordinates changed: %+v", original.Identity)
		}
	}
}

func TestContextModeledScopeIsExplicitLocalAndExact(t *testing.T) {
	compiled := compileFixture(t, richContextInput(t))
	for _, test := range []struct {
		pkg        string
		ancestor   bool
		sibling    bool
		ownPresent bool
	}{
		{".", false, false, false}, {"a", true, false, true}, {"a/child", true, false, true},
		{"aother", false, true, true}, {"empty", false, false, false}, {"software", false, false, false}, {"é/子", false, false, true},
	} {
		t.Run(test.pkg, func(t *testing.T) {
			selection := requireContext(t, compiled, test.pkg, scopeSelector())
			global, found := selection.Definition(contextID("rules", "global"))
			if !found || len(global.Reasons) != 1 || global.Reasons[0].Kind != organization.ModeledScope || global.Reasons[0].ScopePackage != "." || global.Reasons[0].At != "/spec/covers/0" || *global.Reasons[0].Policy != scopeSelector().Source {
				t.Fatalf("global scope or field/policy source lost: %+v", global)
			}
			_, ancestor := selection.Definition(contextID("rules", "ancestor"))
			_, sibling := selection.Definition(contextID("rules", "sibling"))
			if ancestor != test.ancestor || sibling != test.sibling {
				t.Fatal("scope matched a sibling prefix or inherited the wrong Package")
			}
			if repeated, found := selection.Definition(contextID("rules", "repeat")); found && (len(repeated.Reasons) != 2 || repeated.Reasons[0].At != "/spec/covers/0" || repeated.Reasons[1].At != "/spec/covers/1") {
				t.Fatal("repeated modeled scope lost occurrence reasons")
			}
			for _, excluded := range []model.Identity{contextID("rules", "absent"), contextID("rules", "empty"), contextID("unused", "foreign")} {
				if _, found := selection.Definition(excluded); found {
					t.Fatalf("missing/empty/local-only scope widened: %+v", excluded)
				}
			}
			owns := false
			for _, entry := range selection.Definitions() {
				for _, reason := range entry.Reasons {
					owns = owns || reason.Kind == organization.OwnDefinition
				}
			}
			if owns != test.ownPresent {
				t.Fatal("root/empty acquired canonical own Definitions")
			}
		})
	}
	if _, err := organization.SelectContext(compiled, "e\u0301/子", nil); err == nil {
		t.Fatal("Manager lookup normalized a nonexact Unicode spelling")
	}
}

func TestContextOwnPathOrderAndInputEnumerationDeterminism(t *testing.T) {
	compiled := compileFixture(t, richContextInput(t))
	// Physical filenames are presentation provenance: own ordering follows them,
	// not identity order. This does not change canonical addresses or content.
	findCompiledDefinition(t, &compiled, contextID("a", "own")).Source.Path = "model/a/z.yaml"
	findCompiledDefinition(t, &compiled, contextID("a", "z-other")).Source.Path = "model/a/a.yaml"
	want := requireContext(t, compiled, "a", scopeSelector())
	entries := want.Definitions()
	if entries[0].Definition.Identity.ID != "z-other" || entries[1].Definition.Identity.ID != "own" {
		t.Fatal("own Definitions not in stable exact source-path order")
	}
	slices.Reverse(compiled.Models)
	for i := range compiled.Models {
		slices.Reverse(compiled.Models[i].Definitions)
		slices.Reverse(compiled.Models[i].Packages)
		for j := range compiled.Models[i].Definitions {
			definition := &compiled.Models[i].Definitions[j]
			definition.Spec = slices.Clone(definition.Spec)
			slices.Reverse(definition.Spec)
		}
	}
	got := requireContext(t, compiled, "a", scopeSelector())
	// Original spec ordering is preserved, so compare deterministic identities
	// and inclusion reasons independently of that deliberately changed source IR.
	for i, entry := range got.Definitions() {
		if entry.Definition.Identity != entries[i].Definition.Identity || !reflect.DeepEqual(entry.Reasons, entries[i].Reasons) {
			t.Fatal("enumeration changed selection ordering or reasons")
		}
	}
}

func TestContextCopiesAllMutableDefinitionAndReasonViews(t *testing.T) {
	compiled := compileFixture(t, richContextInput(t))
	selector := scopeSelector()
	selection := requireContext(t, compiled, "a", selector)
	want := selection.Definitions()
	original := findCompiledDefinition(t, &compiled, contextID("a", "own"))
	*original.Title = "changed"
	original.Spec[0].Name = "changed"
	*original.BodyLinks[0].Display = "changed"
	if original.BodyLinks[0].Coordinates != nil {
		if original.BodyLinks[0].Coordinates.Body != nil {
			original.BodyLinks[0].Coordinates.Body.StartByte = 999
		}
		if original.BodyLinks[0].Coordinates.Source != nil {
			original.BodyLinks[0].Coordinates.Source.Start.Line = 999
		}
	}
	original.BodyLinks[0].Target.ID = "changed"
	selector.Source.Field = "changed"
	all := selection.Definitions()
	*all[0].Definition.Title = "changed"
	all[0].Definition.Spec[0].Name = "changed"
	all[0].Reasons[0].At = "changed"
	*all[0].Definition.BodyLinks[0].Display = "changed"
	if coordinates := all[0].Definition.BodyLinks[0].Coordinates; coordinates != nil {
		if coordinates.Body != nil {
			coordinates.Body.EndByte = 999
		}
		if coordinates.Source != nil {
			coordinates.Source.End.Line = 999
		}
	}
	global, _ := selection.Definition(contextID("rules", "global"))
	global.Reasons[0].Policy.Field = "changed"
	lookup, _ := selection.Definition(contextID("a", "own"))
	lookup.Definition.BodyLinks[0].Target.ID = "changed"
	lookup.Definition.Spec[0].Name = "changed"
	if !reflect.DeepEqual(selection.Definitions(), want) {
		t.Fatal("input or output mutation changed fixed selected content/provenance")
	}
	if _, found := selection.Definition(contextID("missing", "missing")); found {
		t.Fatal("unknown Definition lookup succeeded")
	}
}

func TestContextScopeScalarAndDefaultPropertyName(t *testing.T) {
	input := richContextInput(t)
	// A single-valued scope field is still a Package value. The default field
	// name is the exact Property ID when no explicit name was authored.
	for i := range input.Root.Entries {
		entry := &input.Root.Entries[i]
		if entry.Path == "model/types/applies.yaml" {
			entry.Bytes = []byte(strings.ReplaceAll(strings.ReplaceAll(string(entry.Bytes), "name: covers, ", ""), "maxCount: -1", "maxCount: 1"))
		}
		if strings.HasPrefix(entry.Path, "model/rules/") || entry.Path == "model/unused/foreign.yaml" {
			// Local global rule exercises scalar selection. All other local
			// optional scope fields are absent to keep the compiler fixture valid.
			if entry.Path == "model/rules/global.yaml" {
				entry.Bytes = []byte(strings.ReplaceAll(string(entry.Bytes), "covers: ['.']", "applies: '.'"))
			} else {
				start := strings.Index(string(entry.Bytes), "spec:")
				entry.Bytes = append(slices.Clone(entry.Bytes[:start]), []byte("spec: {}\n")...)
			}
		}
	}
	// The independently supplied foreign rule also needs to satisfy its now
	// scalar producer contract, while remaining outside local scope selection.
	foreign := &input.Dependencies[len(input.Dependencies)-1].Entries[1]
	foreign.Bytes = []byte(strings.ReplaceAll(string(foreign.Bytes), "covers: ['.']", "applies: '.'"))
	compiled := compileFixture(t, input)
	selection := requireContext(t, compiled, "empty", scopeSelector())
	rule, found := selection.Definition(contextID("rules", "global"))
	if !found || rule.Reasons[0].At != "/spec/applies" || rule.Reasons[0].ScopePackage != "." {
		t.Fatalf("single/default-name scope selection lost: %+v", rule)
	}
}
