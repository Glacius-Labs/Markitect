package model

import "testing"

// interfaceInput declares the interface Operation with the Property label,
// an implementing Kind Ledger, an unrelated Other, and a Holder whose ref
// targets Operation.
func interfaceInput(ledgerSpec, itemSpec, holderSpec string) Input {
	return coreModelInput(
		contractDefinition("feature/operation.yaml", "Operation", "Core.Kind", "{properties: [label]}"),
		contractDefinition("feature/ledger.yaml", "Ledger", "Core.Kind", ledgerSpec),
		contractDefinition("feature/other.yaml", "Other", "Core.Kind", "{}"),
		contractDefinition("feature/holder-kind.yaml", "Holder", "Core.Kind", "{properties: [ref]}"),
		contractDefinition("feature/label.yaml", "label", "Core.Property", "{valueType: string, minCount: 1, maxCount: 1}"),
		contractDefinition("feature/account.yaml", "account", "Core.Property", "{valueType: string, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/ref-field.yaml", "ref", "Core.Property", "{valueType: reference, target: Operation, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/item.yaml", "item", "Ledger", itemSpec),
		contractDefinition("feature/stranger.yaml", "stranger", "Other", "{}"),
		contractDefinition("feature/holding.yaml", "holding", "Holder", holderSpec),
	)
}

const conforming = "{implements: [Operation], properties: [label, account]}"

func TestKindInterfacesAcceptImplementersWithoutInheritingFields(t *testing.T) {
	spec := requireValuesPassed(t, interfaceInput(conforming, "{label: posted, account: cash}", "{ref: item}"))
	if value, present := lookup(spec, "label"); !present || value.text != "posted" {
		t.Fatalf("declared interface field not typed on the implementing item: %+v", spec)
	}
	for _, tt := range []struct{ item, holder, code string }{
		{"{account: cash}", "{}", "value.cardinality"},            // the implementer's own required field
		{"{label: x, extra: y}", "{}", "value.unknown_field"},     // its own contract stays closed
		{"{label: x}", "{ref: stranger}", "value.reference_kind"}, // an unrelated Kind is refused
		{"{label: x}", "{ref: Ledger}", "value.reference_kind"},   // a Kind itself is not an item
	} {
		got := Check(interfaceInput(conforming, tt.item, tt.holder))
		if got.Status != Failed || !hasDiagnostic(got, tt.code) {
			t.Fatalf("want %s for item %s holder %s: %+v", tt.code, tt.item, tt.holder, got)
		}
	}
}

func TestKindInterfaceConformanceAndContractRules(t *testing.T) {
	for _, tt := range []struct{ ledger, code string }{
		{"{implements: [Operation], properties: [account]}", "contract.implements_missing"},
		{"{implements: [Operation, Operation], properties: [label]}", "contract.implements_duplicate"},
		{"{implements: [Core.Kind]}", "contract.implements_meta"},
		{"{implements: [Core.Property]}", "contract.implements_meta"},
		{"{implements: [label]}", "contract.reference_target"},
		{"{implements: Operation, properties: [label]}", "contract.implements_list"},
	} {
		got := Check(interfaceInput(tt.ledger, "{label: x}", "{}"))
		if got.Status != Failed || !hasDiagnostic(got, tt.code) {
			t.Fatalf("want %s for %s: %+v", tt.code, tt.ledger, got)
		}
	}
	// Without implements, the same fields give no acceptance at an Operation reference.
	got := Check(interfaceInput("{properties: [label, account]}", "{label: x}", "{ref: item}"))
	if got.Status != Failed || !hasDiagnostic(got, "value.reference_kind") {
		t.Fatalf("structural match accepted without implements: %+v", got)
	}
}

func TestKindInterfaceCyclesAndTransitiveAcceptance(t *testing.T) {
	cycle := coreModelInput(
		contractDefinition("feature/a.yaml", "A", "Core.Kind", "{implements: [B]}"),
		contractDefinition("feature/b.yaml", "B", "Core.Kind", "{implements: [A]}"),
	)
	if got := Check(cycle); got.Status != Failed || !hasDiagnostic(got, "contract.implements_cycle") {
		t.Fatalf("implements cycle accepted: %+v", got)
	}
	self := coreModelInput(contractDefinition("feature/a.yaml", "A", "Core.Kind", "{implements: [A]}"))
	if got := Check(self); got.Status != Failed || !hasDiagnostic(got, "contract.implements_cycle") {
		t.Fatalf("self-implementation accepted: %+v", got)
	}
	// Leaf implements Middle, which implements Top: Leaf declares every
	// Property of both and is accepted where Top is required.
	transitive := coreModelInput(
		contractDefinition("feature/top.yaml", "Top", "Core.Kind", "{properties: [label]}"),
		contractDefinition("feature/middle.yaml", "Middle", "Core.Kind", "{implements: [Top], properties: [label, note]}"),
		contractDefinition("feature/leaf.yaml", "Leaf", "Core.Kind", "{implements: [Middle], properties: [label, note]}"),
		contractDefinition("feature/holder-kind.yaml", "Holder", "Core.Kind", "{properties: [ref]}"),
		contractDefinition("feature/label.yaml", "label", "Core.Property", "{valueType: string, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/note.yaml", "note", "Core.Property", "{valueType: string, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/ref-field.yaml", "ref", "Core.Property", "{valueType: reference, target: Top, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/item.yaml", "item", "Leaf", "{label: x}"),
		contractDefinition("feature/holding.yaml", "holding", "Holder", "{ref: item}"),
	)
	if got := Check(transitive); got.Status != Passed || len(got.Diagnostics) != 0 {
		t.Fatalf("transitive interface acceptance rejected: %+v", got)
	}
}

func TestImplementsIsOrdinaryDataOutsideTheMetaKindAndForbiddenOnProperties(t *testing.T) {
	data := coreModelInput(
		contractDefinition("feature/catalog.yaml", "Catalog", "Core.Kind", "{properties: [implementations]}"),
		contractDefinition("feature/implementations.yaml", "implementations", "Core.Property", "{name: implements, valueType: string, minCount: 0, maxCount: -1}"),
		contractDefinition("feature/entry.yaml", "entry", "Catalog", "{implements: [anything, at, all]}"),
	)
	if got := Check(data); got.Status != Passed || len(got.Diagnostics) != 0 {
		t.Fatalf("an ordinary implements data field was given meta meaning: %+v", got)
	}
	property := coreModelInput(
		contractDefinition("feature/base.yaml", "Base", "Core.Kind", "{}"),
		contractDefinition("feature/field.yaml", "field", "Core.Property", "{valueType: string, minCount: 0, maxCount: 1, implements: [Base]}"),
	)
	if got := Check(property); got.Status != Failed || !hasDiagnostic(got, "contract.unknown_field") {
		t.Fatalf("a Property accepted implements: %+v", got)
	}
}

func TestKindInterfaceEdgeCases(t *testing.T) {
	base := func(order string) Input {
		return coreModelInput(
			contractDefinition("feature/top.yaml", "Top", "Core.Kind", "{properties: [label]}"),
			contractDefinition("feature/middle.yaml", "Middle", "Core.Kind", "{implements: [Top], properties: [label]}"),
			contractDefinition("feature/leaf.yaml", "Leaf", "Core.Kind", "{implements: "+order+", properties: [label]}"),
			contractDefinition("feature/label.yaml", "label", "Core.Property", "{valueType: string, minCount: 0, maxCount: 1}"),
		)
	}
	// Naming an interface that is also reached through another one is permitted,
	// and the order of the declaration does not change the canonical content.
	forward := requireChecked(t, base("[Middle, Top]")).Models[0]
	backward := requireChecked(t, base("[Top, Middle]")).Models[0]
	if forward.Digest != backward.Digest {
		t.Fatalf("implements order changed the digest: %s / %s", forward.Digest, backward.Digest)
	}
	// An implementer whose own properties did not resolve is not also blamed
	// for missing interface Properties.
	got := Check(interfaceInput("{implements: [Operation], properties: [absent]}", "{}", "{}"))
	if got.Status != Failed || !hasDiagnostic(got, "binding.reference_missing") || hasDiagnostic(got, "contract.implements_missing") {
		t.Fatalf("unresolved own properties reported as missing interface Properties: %+v", got)
	}
}
