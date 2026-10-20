package organization_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func assertZeroContext(t *testing.T, selection organization.ContextSelection) {
	t.Helper()
	if selection.Package() != "" || selection.ModelID() != "" || selection.ModelDigest() != "" || len(selection.Definitions()) != 0 {
		t.Fatal("refusal exposed partial context or model binding")
	}
	if _, found := selection.Definition(contextID("a", "own")); found {
		t.Fatal("refusal exposed a Definition lookup")
	}
}

func cloneContextResult(checked model.Result) model.Result {
	result := copyResult(checked)
	for i := range result.Models {
		for j := range result.Models[i].Definitions {
			definition := &result.Models[i].Definitions[j]
			definition.Spec = slices.Clone(definition.Spec)
			definition.BodyLinks = slices.Clone(definition.BodyLinks)
		}
	}
	return result
}

func TestContextRefusesInvalidOrInconsistentRequiredFacts(t *testing.T) {
	baseline := checkFixture(t, richContextInput(t))
	for _, status := range []model.Status{"", model.Failed, model.NotCompleted} {
		result := cloneContextResult(baseline)
		result.Status = status // The local model itself remains Passed.
		selection, err := organization.SelectContext(result, "a", nil)
		if err == nil {
			t.Fatal("isolated passed local model bypassed whole-supplied-set failure")
		}
		assertZeroContext(t, selection)
	}
	for _, test := range []struct {
		name   string
		mutate func(*model.Result)
	}{
		{"zero", func(r *model.Result) { *r = model.Result{} }},
		{"failed owner beside claimed global success", func(r *model.Result) { r.Models[1].Status = model.Failed }},
		{"missing owner digest", func(r *model.Result) { r.Models[1].Digest = "" }},
		{"duplicate model owner", func(r *model.Result) { r.Models = append(r.Models, r.Models[1]) }},
		{"duplicate unused global identity", func(r *model.Result) {
			r.Models[0].Definitions = append(r.Models[0].Definitions, *findCheckedDefinition(t, r, contextID("types", "note")))
		}},
		{"mismatched Definition owner", func(r *model.Result) { findCheckedDefinition(t, r, contextID("b", "near")).ModelID = "other" }},
		{"empty Definition identity", func(r *model.Result) { findCheckedDefinition(t, r, contextID("b", "near")).Identity.ID = "" }},
		{"missing selected source path", func(r *model.Result) { findCheckedDefinition(t, r, contextID("b", "near")).Source.Path = "" }},
		{"missing selected source digest", func(r *model.Result) { findCheckedDefinition(t, r, contextID("b", "near")).Source.Digest = "" }},
		{"missing body target", func(r *model.Result) {
			findCheckedDefinition(t, r, contextID("a", "own")).BodyLinks[0].Target = contextID("missing", "target")
		}},
		{"missing kind target", func(r *model.Result) {
			findCheckedDefinition(t, r, contextID("a", "own")).Kind = contextID("missing", "Kind")
		}},
		{"kind target is not Kind", func(r *model.Result) {
			findCheckedDefinition(t, r, contextID("a", "own")).Kind = contextID("types", "note")
		}},
		{"missing optional Property target", func(r *model.Result) {
			findCheckedDefinition(t, r, contextID("types", "unused")).Identity.ID = "changed"
		}},
		{"Property target is not Property", func(r *model.Result) {
			findCheckedDefinition(t, r, contextID("types", "unused")).Kind = contextID("types", "Record")
		}},
		{"zero checked spec value", func(r *model.Result) {
			findCheckedDefinition(t, r, contextID("a", "own")).Spec[0].Value = model.Value{}
		}},
		{"duplicate used metadata field", func(r *model.Result) {
			d := findCheckedDefinition(t, r, contextID("types", "Record"))
			d.Spec = append(d.Spec, d.Spec[0])
		}},
		{"scope cardinality contradiction", func(r *model.Result) {
			global := findCheckedDefinition(t, r, contextID("rules", "global"))
			items, _ := global.Spec[0].Value.Items()
			global.Spec[0].Value = items[0]
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := cloneContextResult(baseline)
			test.mutate(&result)
			selection, err := organization.SelectContext(result, "a", []organization.Setting[organization.ScopeRule]{scopeSelector()})
			if err == nil {
				t.Fatal("inconsistent claimed-Passed basis unexpectedly accepted")
			}
			assertZeroContext(t, selection)
		})
	}
	for _, key := range []string{"", "missing", "markitect/core", "a/", "a/*", "A"} {
		selection, err := organization.SelectContext(baseline, key, nil)
		if err == nil {
			t.Fatalf("nonexact/nonlocal Manager accepted: %q", key)
		}
		assertZeroContext(t, selection)
	}
}

func TestContextRefusesInvalidScopeSelectorsAndPaths(t *testing.T) {
	checked := checkFixture(t, richContextInput(t))
	for _, test := range []struct {
		name   string
		mutate func(*organization.Setting[organization.ScopeRule])
	}{
		{"unsourced", func(s *organization.Setting[organization.ScopeRule]) { s.Source = organization.SettingSource{} }},
		{"unknown Kind", func(s *organization.Setting[organization.ScopeRule]) { s.Value.Kind.ID = "Missing" }},
		{"Kind is a Property", func(s *organization.Setting[organization.ScopeRule]) { s.Value.Kind = contextID("types", "applies") }},
		{"unknown Property", func(s *organization.Setting[organization.ScopeRule]) { s.Value.Property.ID = "missing" }},
		{"Property not in Kind", func(s *organization.Setting[organization.ScopeRule]) { s.Value.Property = contextID("types", "note") }},
		{"nontextual Property", func(s *organization.Setting[organization.ScopeRule]) {
			s.Value.Kind = contextID("types", "Record")
			s.Value.Property = contextID("types", "edge")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			selector := scopeSelector()
			test.mutate(&selector)
			selection, err := organization.SelectContext(checked, "a", []organization.Setting[organization.ScopeRule]{selector})
			if err == nil {
				t.Fatal("invalid selector unexpectedly accepted")
			}
			assertZeroContext(t, selection)
		})
	}
	selection, err := organization.SelectContext(checked, "a", []organization.Setting[organization.ScopeRule]{scopeSelector(), scopeSelector()})
	if err == nil {
		t.Fatal("duplicate selector accepted")
	}
	assertZeroContext(t, selection)
	for _, target := range []string{"missing", "markitect/core", "a/", "A", "e\u0301/子", "a/../b", ""} {
		t.Run("scope path "+target, func(t *testing.T) {
			input := richContextInput(t)
			for i := range input.Root.Entries {
				entry := &input.Root.Entries[i]
				if entry.Path == "model/rules/ancestor.yaml" {
					entry.Bytes = []byte(strings.Replace(string(entry.Bytes), "covers: [a]", "covers: ['"+target+"']", 1))
				}
			}
			// Canonical string values do not acquire Package semantics in the
			// Model check. Only the explicit organization scope selector applies it.
			checked := checkFixture(t, input)
			selection, err := organization.SelectContext(checked, "empty", []organization.Setting[organization.ScopeRule]{scopeSelector()})
			if err == nil {
				t.Fatal("invalid scope path hidden by an unrelated selected Manager")
			}
			assertZeroContext(t, selection)
		})
	}
}

func TestContextCoordinateCoverageDoesNotBecomeContentGate(t *testing.T) {
	checked := checkFixture(t, richContextInput(t))
	checked.Coordinates = model.NotCompleted
	definition := findCheckedDefinition(t, &checked, contextID("a", "own"))
	for i := range definition.BodyLinks {
		definition.BodyLinks[i].Coordinates = nil
	}
	selection := requireContext(t, checked, "a")
	selected, _ := selection.Definition(definition.Identity)
	if !reflect.DeepEqual(selected.Definition, *definition) {
		t.Fatal("unavailable spans were invented or original content changed")
	}
}
