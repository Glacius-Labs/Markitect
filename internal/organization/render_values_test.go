package organization_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestRenderLosslessCompiledValuesAndReferenceOccurrences(t *testing.T) {
	input := richContextInput(t)
	const huge = "12345678901234567890123456789012345678901234567890"
	for i := range input.Root.Entries {
		entry := &input.Root.Entries[i]
		switch entry.Path {
		case "model/types/record.yaml":
			entry.Bytes = []byte(strings.Replace(string(entry.Bytes), "[edge, payload, unused, data]", "[edge, payload, unused, data, huge, precise, flag, flavor]", 1))
		case "model/a/own.yaml":
			entry.Bytes = []byte(strings.Replace(string(entry.Bytes), "properties: literal}", "properties: literal, huge: "+huge+", precise: 1.2300e900000000000000000000000, flag: true, flavor: Exact}", 1))
		}
	}
	for _, property := range []struct{ id, kind, extra string }{{"huge", "integer", ""}, {"precise", "number", ""}, {"flag", "boolean", ""}, {"flavor", "enum", ", values: [exact, Exact]"}} {
		input.Root.Entries = append(input.Root.Entries, model.Entry{Path: "model/types/" + property.id + ".yaml", Kind: model.Regular, Bytes: []byte(fmt.Sprintf("imports: [{package: markitect/core, as: Core}]\nid: %s\nkind: Core.Property\nbody: Exact value fixture.\nspec: {type: %s, minCount: 0, maxCount: 1%s}\n", property.id, property.kind, property.extra))})
	}
	compiled := compileFixture(t, input)
	selection := requireContext(t, compiled, "a", scopeSelector())
	settings := requireSettings(t, requireTopology(t, compiled), settingsFixture(t))
	_, _, renderInput := renderFixture(t, "a", organization.ManagerView)
	result := requireRendered(t, selection, settings, renderInput)
	var spec map[string]any
	if err := json.Unmarshal(sectionTexts(t, result, "lossless-typed-spec-not-original-yaml")[0], &spec); err != nil {
		t.Fatal(err)
	}
	for name, expected := range map[string][]any{
		"huge": {"integer", huge}, "precise": {"number", "123", "899999999999999999999998"}, "flag": {"boolean", true}, "flavor": {"enum", "Exact"}, "properties": {"string", "literal"},
	} {
		if !reflect.DeepEqual(spec[name], expected) {
			t.Fatalf("%s value rounded/coerced: %#v", name, spec[name])
		}
	}
	ref := []any{"reference", "b", "near"}
	if !reflect.DeepEqual(spec["links"], []any{"list", []any{ref, ref}}) {
		t.Fatalf("repeated typed references changed: %#v", spec["links"])
	}
	nested := spec["payload"].([]any)
	if nested[0] != "object" || !reflect.DeepEqual(nested[1].(map[string]any)["links"], []any{"list", []any{ref}}) {
		t.Fatal("nested object/reference type lost")
	}
	var links []model.BodyLink
	if err := json.Unmarshal(sectionTexts(t, result, "original-body-link-occurrences")[0], &links); err != nil {
		t.Fatal(err)
	}
	own, _ := selection.Definition(contextID("a", "own"))
	if !reflect.DeepEqual(links, own.Definition.BodyLinks) {
		t.Fatal("display, repetitions or optional coordinates changed")
	}
	var identity struct {
		ModelID, ModelDigest string
		Identity, Kind       model.Identity
		Reasons              []organization.ContextReason
	}
	if err := json.Unmarshal(sectionTexts(t, result, "definition-identity-contract-and-inclusion")[0], &identity); err != nil {
		t.Fatal(err)
	}
	if identity.ModelID != own.Definition.ModelID || identity.ModelDigest != own.ModelDigest || identity.Identity != own.Definition.Identity || identity.Kind != own.Definition.Kind || !reflect.DeepEqual(identity.Reasons, own.Reasons) {
		t.Fatal("source identity/type/inclusion provenance lost")
	}
}

func TestRenderDeterministicSetProjectionsAndDeclaredOrder(t *testing.T) {
	selection, settings, input := renderFixture(t, organization.Root, organization.ManagerView)
	purposeA, purposeB := hostFact("purpose-a", "A responsibility."), hostFact("purpose-b", "B responsibility.")
	input.Packages = []organization.PackageFact{{Package: "a", Purpose: &purposeA}, {Package: "b", Purpose: &purposeB}}
	input.Children = []organization.ChildFact{{Package: "a", Reports: []organization.HostText{hostFact("a-report", "Summary A.")}}, {Package: "b", Reports: []organization.HostText{hostFact("b-report", "Summary B.")}}}
	input.Task.Findings.IDs = append(input.Task.Findings.IDs, "finding-2")
	input.Task.Findings.Items = append(input.Task.Findings.Items, organization.FindingFact{ID: "finding-2", Grounds: hostFact("finding-2", "Second obligation."), Evidence: []organization.HostSource{hostRef("evidence-2", "")}})
	a, b := renderArtifact(t, organization.Root), renderArtifact(t, organization.Root)
	b.Path, _ = snapshot.NewPath("real/z.go")
	input.OwnArtifacts = []organization.ArtifactFact{b, a}
	input.Task.Constraints = []organization.HostText{hostFact("constraint-1", "First."), hostFact("constraint-2", "Second.")}
	first := requireRendered(t, selection, settings, input)
	slices.Reverse(input.Packages)
	slices.Reverse(input.Children)
	slices.Reverse(input.OwnArtifacts)
	slices.Reverse(input.Task.Findings.IDs)
	slices.Reverse(input.Task.Findings.Items)
	again := requireRendered(t, selection, settings, input)
	if !bytes.Equal(first.Bytes(), again.Bytes()) || !reflect.DeepEqual(first.Sections(), again.Sections()) {
		t.Fatal("set-shaped input order affects fixed rendering")
	}
	slices.Reverse(input.Task.Constraints)
	ordered := requireRendered(t, selection, settings, input)
	if bytes.Equal(first.Bytes(), ordered.Bytes()) || !bytes.Equal(sectionTexts(t, ordered, "task-constraint")[0], []byte("Second.")) {
		t.Fatal("declared constraint order discarded")
	}
}
