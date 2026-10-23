package model

import (
	"bytes"
	"testing"
)

func referenceValueInput(spec string) Input {
	return coreModelInput(
		contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{properties: [link, links, kindLink]}"),
		contractDefinition("feature/alternate.yaml", "Alternate", "Core.Kind", "{}"),
		contractDefinition("feature/link.yaml", "link", "Core.Property", "{type: reference, target: Record, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/links.yaml", "links", "Core.Property", "{type: reference, target: Record, minCount: 0, maxCount: -1}"),
		contractDefinition("feature/kind-link.yaml", "kindLink", "Core.Property", "{type: reference, target: Core.Kind, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/item.yaml", "item", "Record", spec),
		contractDefinition("feature/peer.yaml", "peer", "Record", "{}"),
		contractDefinition("feature/wrong.yaml", "wrong", "Alternate", "{}"),
	)
}

func addFileImport(entry Entry, pkg, alias string) Entry {
	entry.Bytes = bytes.Replace(entry.Bytes, []byte("package: markitect/core, as: Core}]"), []byte("package: markitect/core, as: Core}, {package: "+pkg+", as: "+alias+"}]"), 1)
	return entry
}

func TestDataReferencesResolveBeforeCheckingActualKind(t *testing.T) {
	input := referenceValueInput("{link: peer, links: [peer, item, peer], kindLink: Record}")
	spec := requireValuesPassed(t, input)
	links, _ := lookup(spec, "links")
	if len(links.items) != 3 || links.items[0].reference.ID != "peer" || links.items[1].reference.ID != "item" || links.items[2].reference.ID != "peer" {
		t.Fatal("ordinary reference list order/repetition changed")
	}
	// An ordinary reference cycle is an identity relation, not object expansion.
	input.Root.Entries[7] = contractDefinition("feature/peer.yaml", "peer", "Record", "{link: item}")
	requireValuesPassed(t, input)
	for _, tt := range []struct{ spec, code string }{
		{"{link: wrong}", "value.reference_kind"},
		{"{link: Record}", "value.reference_kind"},
		{"{link: Core.Kind}", "value.reference_kind"},
		{"{link: Peer}", "binding.reference_spelling"},
		{"{link: absent}", "binding.reference_missing"},
		{"{link: Unknown.peer}", "binding.alias_missing"},
		{"{link: ' peer '}", "value.reference"},
		{"{link: ''}", "value.reference"},
		{"{link: 42}", "value.reference"},
		{"{link: 'Core.Kind.field'}", "value.reference"},
		{"{kindLink: peer}", "value.reference_kind"},
	} {
		input := referenceValueInput(tt.spec)
		got := Check(input)
		if got.Status != Failed || !hasDiagnostic(got, tt.code) {
			t.Fatalf("want %s: %+v", tt.code, got)
		}
	}
	requireValuesPassed(t, referenceValueInput("{link: feature/peer}"))
}

func TestDataReferenceFileContextAndNoExpectedTypeFiltering(t *testing.T) {
	input := referenceValueInput("{link: Refs.peer}")
	input.Root.Entries[6] = addFileImport(input.Root.Entries[6], "foreign", "Refs")
	foreign := addFileImport(contractDefinition("foreign/peer.yaml", "peer", "feature/Record", "{}"), "feature", "Feature")
	input.Root.Entries = append(input.Root.Entries, foreign)
	requireValuesPassed(t, input)
	// The import and alias in item do not leak into peer.
	input.Root.Entries[7] = contractDefinition("feature/peer.yaml", "peer", "Record", "{link: Refs.peer}")
	if got := Check(input); got.Status != Failed || !hasDiagnostic(got, "binding.alias_missing") {
		t.Fatalf("data reference borrowed another file's imports: %+v", got)
	}
	input.Root.Entries[7] = contractDefinition("feature/peer.yaml", "peer", "Alternate", "{}")
	input.Root.Entries[6] = addFileImport(contractDefinition("feature/item.yaml", "item", "Record", "{link: peer}"), "foreign", "Refs")
	if got := Check(input); got.Status != Failed || !hasDiagnostic(got, "binding.reference_ambiguous") {
		t.Fatalf("expected Kind filtered out the wrong local candidate: %+v", got)
	}
	input.Root.Entries[6] = addFileImport(contractDefinition("feature/item.yaml", "item", "Record", "{link: Refs.peer}"), "foreign", "Refs")
	requireValuesPassed(t, input)
	input.Root.Entries[6] = contractDefinition("feature/item.yaml", "item", "Record", "{link: foreign/peer}")
	if got := Check(input); got.Status != Failed || !hasDiagnostic(got, "binding.package_not_visible") {
		t.Fatalf("full foreign address bypassed file import: %+v", got)
	}
}

func addIncompleteModel(input *Input) {
	input.Limits.YAMLDepth = 4
	input.Dependencies = append(input.Dependencies, bindingSource("incomplete", "1.0", "{}", Entry{
		Path: "model/unavailable/partial.yaml", Kind: Regular,
		Bytes: []byte("spec: {a: {b: {c: {d: text}}}}"),
	}))
}

func TestValueErrorsVersusUnrelatedOrRelevantUnknownInventory(t *testing.T) {
	for _, tt := range []struct{ spec, code string }{
		{"{link: wrong}", "value.reference_kind"},
		{"{link: 42}", "value.reference"},
		{"{unknown: a}", "value.unknown_field"},
	} {
		input := referenceValueInput(tt.spec)
		addIncompleteModel(&input)
		got := Check(input)
		if got.Status != Failed || !hasDiagnostic(got, tt.code) || !hasDiagnostic(got, "limit.yaml_depth") {
			t.Fatalf("known value error lost beside a separate limit: %+v", got)
		}
	}
	input := referenceValueInput("{link: peer}")
	addIncompleteModel(&input)
	got := Check(input)
	if got.Status != NotCompleted || !hasDiagnostic(got, "limit.yaml_depth") || hasDiagnostic(got, "binding.reference_missing") || hasDiagnostic(got, "value.type") {
		t.Fatalf("unknown inventory became false missing/type evidence: %+v", got)
	}
	if len(privateValues(input).specs) != 0 {
		t.Fatal("incomplete scope retained typed specs as checked candidates")
	}
	// An unresolved Property declaration may supply the apparent extra field;
	// the partial Kind's known subset is not a complete closed schema.
	input.Root.Entries[1] = addFileImport(contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{properties: [Unavailable.field]}"), "unavailable", "Unavailable")
	input.Root.Entries[6] = contractDefinition("feature/item.yaml", "item", "feature/Record", "{extra: a}")
	got = Check(input)
	if got.Status != NotCompleted || !hasDiagnostic(got, "limit.yaml_depth") || hasDiagnostic(got, "value.unknown_field") || hasDiagnostic(got, "binding.reference_missing") {
		t.Fatalf("partial contract was treated as complete: %+v", got)
	}
}

func TestNestedSchemaWithIncompleteFieldContractRemainsUnknown(t *testing.T) {
	input := coreModelInput(
		contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{properties: [box]}"),
		contractDefinition("feature/box.yaml", "box", "Core.Property", "{type: object, minCount: 0, maxCount: 1, properties: [link]}"),
		addFileImport(contractDefinition("feature/link.yaml", "link", "Core.Property", "{type: reference, target: Unavailable.Kind, minCount: 0, maxCount: -1}"), "unavailable", "Unavailable"),
		contractDefinition("feature/item.yaml", "item", "Record", "{box: {extra: a}}"),
	)
	addIncompleteModel(&input)
	input.Limits.YAMLDepth = 5
	got := Check(input)
	if got.Status != NotCompleted || !hasDiagnostic(got, "limit.yaml_depth") || hasDiagnostic(got, "value.unknown_field") || len(privateValues(input).specs) != 0 {
		t.Fatalf("partially checked Property closed the object schema: %+v", got)
	}
	input.Root.Entries[4] = contractDefinition("feature/item.yaml", "item", "Record", "{box: {link: [42]}}")
	got = Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "value.reference") || !hasDiagnostic(got, "limit.yaml_depth") {
		t.Fatalf("independent known reference type error was suppressed: %+v", got)
	}
}
