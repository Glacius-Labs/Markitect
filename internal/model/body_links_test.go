package model

import (
	"bytes"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func withBody(entry Entry, yaml string) Entry {
	entry.Bytes = bytes.Replace(entry.Bytes, []byte("body: Contract fixture."), []byte("body: "+yaml), 1)
	return entry
}

func privateBodies(input Input) bodyResult {
	scopes, graph, bindings := privateBindings(input)
	values := checkValues(scopes, graph, checkContracts(scopes, graph, bindings))
	return checkBodyLinks(scopes, graph, values, effectiveLimits(input.Limits))
}

func TestBodyLinksBindEveryCategoryInOccurrenceOrder(t *testing.T) {
	input := referenceValueInput("{}")
	input.Root.Entries[6] = withBody(input.Root.Entries[6], `"📄\\0 [[Record]] [[link]] [[peer|A\\0B]] [[Core.Kind]] [[Core.Property]] [[peer]]"`)
	// Use YAML NUL escapes, retaining the original bytes separately from the
	// CommonMark view and from the decoded literal display string.
	input.Root.Entries[6].Bytes = bytes.ReplaceAll(input.Root.Entries[6].Bytes, []byte(`\\0`), []byte(`\0`))
	original := bytes.Clone(input.Root.Entries[6].Bytes)
	got := Check(input)
	if got.Status != Passed || got.Coordinates != Passed || len(got.Diagnostics) != 0 || phaseStatus(got.Models[0], "body_links") != Passed {
		t.Fatalf("valid body links rejected: %+v", got)
	}
	links := privateBodies(input).links[Identity{"feature", "item"}]
	wanted := []Identity{{"feature", "Record"}, {"feature", "link"}, {"feature", "peer"}, {"markitect/core", "Kind"}, {"markitect/core", "Property"}, {"feature", "peer"}}
	var targets []Identity
	for index, link := range links {
		targets = append(targets, link.Target)
		if link.Coordinates == nil || link.Coordinates.Body == nil || link.Coordinates.Source == nil || (index > 0 && link.Coordinates.Body.StartByte <= links[index-1].Coordinates.Body.StartByte) {
			t.Fatalf("occurrence lost exact independent coordinates: %+v", links)
		}
	}
	if !reflect.DeepEqual(targets, wanted) || links[0].Display != nil || links[2].Display == nil || *links[2].Display != "A\x00B" || !bytes.Equal(original, input.Root.Entries[6].Bytes) {
		t.Fatalf("identity/order/repetition/literal display/source changed: %+v", links)
	}
	for _, model := range got.Models {
		if len(model.Definitions) == 0 || len(model.Canonical) == 0 || model.Digest == "" || phaseStatus(model, "lock") != Passed || phaseStatus(model, "canonical") != Passed {
			t.Fatal("complete body/lock/canonical stages withheld checked public content")
		}
	}
	slices.Reverse(input.Root.Entries)
	slices.Reverse(input.Dependencies)
	if reordered := Check(input); !reflect.DeepEqual(got, reordered) {
		t.Fatal("body report depends on source enumeration")
	}
}

func TestBodyLinksUseOnlyTheirFileImports(t *testing.T) {
	input := referenceValueInput("{}")
	foreign := addFileImport(contractDefinition("foreign/peer.yaml", "peer", "feature/Record", "{}"), "feature", "Feature")
	input.Root.Entries = append(input.Root.Entries, foreign)
	input.Root.Entries[6] = withBody(addFileImport(input.Root.Entries[6], "foreign", "Refs"), "'[[Refs.peer]] [[foreign/peer]]'")
	if got := Check(input); got.Status != Passed || len(got.Diagnostics) != 0 {
		t.Fatalf("explicit direct import not bound: %+v", got)
	}
	input.Root.Entries[7] = withBody(input.Root.Entries[7], "'[[Refs.peer]]'")
	if got := Check(input); got.Status != Failed || !hasDiagnostic(got, "binding.alias_missing") {
		t.Fatalf("body borrowed another file's alias: %+v", got)
	}
	input.Root.Entries[7] = withBody(contractDefinition("feature/peer.yaml", "peer", "Record", "{}"), "'[[foreign/peer]]'")
	if got := Check(input); got.Status != Failed || !hasDiagnostic(got, "binding.package_not_visible") {
		t.Fatalf("full address bypassed file imports: %+v", got)
	}
	input.Root.Entries[7] = contractDefinition("feature/peer.yaml", "peer", "Record", "{}")
	input.Root.Entries[6] = withBody(addFileImport(contractDefinition("feature/item.yaml", "item", "Record", "{}"), "foreign", "Refs"), "'[[peer]]'")
	if got := Check(input); got.Status != Failed || !hasDiagnostic(got, "binding.reference_ambiguous") {
		t.Fatalf("body selected a winner among visible targets: %+v", got)
	}
}

func TestBodyFailureAndResourceRefusalRemainDistinct(t *testing.T) {
	for _, tt := range []struct{ body, code string }{
		{"'[[Peer]]'", "binding.reference_spelling"},
		{"'[[absent]]'", "binding.reference_missing"},
		{"'[[peer| ]]'", "body.syntax"},
	} {
		input := referenceValueInput("{}")
		input.Root.Entries[6] = withBody(input.Root.Entries[6], tt.body)
		addIncompleteModel(&input)
		got := Check(input)
		if got.Status != Failed || !hasDiagnostic(got, tt.code) || !hasDiagnostic(got, "limit.yaml_depth") {
			t.Fatalf("known body error lost beside unrelated unknown inventory: %+v", got)
		}
	}
	input := referenceValueInput("{}")
	addIncompleteModel(&input)
	input.Root.Entries[6] = withBody(addFileImport(input.Root.Entries[6], "unavailable", "Unavailable"), "'[[Unavailable.absent]]'")
	if got := Check(input); got.Status != NotCompleted || hasDiagnostic(got, "binding.reference_missing") || got.Coordinates != NotCompleted {
		t.Fatalf("incomplete imported inventory became false missing evidence: %+v", got)
	}
	input = referenceValueInput("{link: 42}")
	input.Limits.BodyBytes = 4
	got := Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "value.reference") || !hasDiagnostic(got, "limit.body_bytes") {
		t.Fatalf("Markdown refusal erased independent value failure: %+v", got)
	}
	input = referenceValueInput("{}")
	input.Limits.MarkdownNesting = 4
	input.Root.Entries[6] = withBody(input.Root.Entries[6], "'"+strings.Repeat("> ", 8)+"[[peer]]'")
	if got := Check(input); got.Status != NotCompleted || !hasDiagnostic(got, "limit.markdown_nesting") || phaseStatus(got.Models[0], "body_links") != NotCompleted {
		t.Fatalf("tool budget became a language failure: %+v", got)
	}
	for _, limits := range []Limits{{BodyBytes: -1}, {MarkdownNesting: -1}} {
		input.Limits = limits
		if got := Check(input); got.Status != NotCompleted || !hasDiagnostic(got, "config.invalid_limit") {
			t.Fatal("negative body budget was accepted")
		}
	}
	for _, first := range []string{"[[bad target]]", "[[absent]]"} {
		input = referenceValueInput("{}")
		input.Limits.MarkdownNesting = 8
		input.Root.Entries[6] = withBody(input.Root.Entries[6], "|\n  "+first+"\n\n  "+strings.Repeat("[label ", 9)+"text")
		got := Check(input)
		code := "binding.reference_missing"
		if first == "[[bad target]]" {
			code = "body.syntax"
		}
		if got.Status != Failed || !hasDiagnostic(got, code) || !hasDiagnostic(got, "limit.markdown_nesting") {
			t.Fatalf("proven earlier block error erased by later Markdown abort: %+v", got)
		}
	}
}

func TestBodyDependencyFailureReachesDependentPhase(t *testing.T) {
	input := referenceValueInput("{}")
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{markitect-core: '1.0'}"), []byte("{markitect-core: '1.0', support: '1.0'}"), 1)
	dependency := bindingSource("support", "1.0", "{markitect-core: '1.0'}",
		withBody(contractDefinition("support/record.yaml", "Record", "Core.Kind", "{}"), "'[[broken'"))
	input.Dependencies = append(input.Dependencies, dependency)
	got := Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "body.syntax") || phaseStatus(got.Models[0], "body_links") != Failed || phaseStatus(got.Models[0], "values_and_references") != Passed {
		t.Fatalf("dependency body failure failed to reach dependent: %+v", got)
	}
}

func TestBodyCoordinatesAreIndependentOptionalMetadata(t *testing.T) {
	input := referenceValueInput("{}")
	input.Root.Entries[6] = withBody(input.Root.Entries[6], "'[[peer]]'")
	scopes, graph, bindings := privateBindings(input)
	values := checkValues(scopes, graph, checkContracts(scopes, graph, bindings))
	// Refuse deliberately stale optional metadata without changing the
	// independently acquired content or its file-local reference context.
	for index := range scopes[0].definitions {
		definition := &scopes[0].definitions[index]
		if definition.identity == (Identity{"feature", "item"}) {
			for field := range definition.envelope.fields {
				if definition.envelope.fields[field].Name == "body" {
					definition.envelope.fields[field].Value.position = Position{}
				}
			}
		}
	}
	bodies := checkBodyLinks(scopes, graph, values, DefaultLimits())
	links := bodies.links[Identity{"feature", "item"}]
	if bodies.status[0] != Passed || bodies.coordinates != NotCompleted || len(bodies.findings) != 1 || bodies.findings[0].diagnostic.Code != "coordinate.yaml_mapping" || len(links) != 1 || links[0].Coordinates.Body == nil || links[0].Coordinates.Source != nil {
		t.Fatalf("missing YAML map changed content or erased occurrence: %+v / %+v", bodies, links)
	}
	input.Root.Entries[6] = withBody(contractDefinition("feature/item.yaml", "item", "Record", "{}"), "'[[absent]]'")
	got := Check(input)
	if got.Status != Failed || got.Coordinates != NotCompleted {
		t.Fatal("unresolved content claimed complete coordinate profile")
	}
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Code == "binding.reference_missing" && (diagnostic.Position == nil || diagnostic.Source.Digest == "" || !strings.Contains(diagnostic.Message, "body byte 0")) {
			t.Fatal("failed binding lost its independently proven source location")
		}
	}
}
