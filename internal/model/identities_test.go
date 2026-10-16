package model

import (
	"reflect"
	"slices"
	"testing"
)

func TestExactDefinitionIdentityDoesNotIncludeFilename(t *testing.T) {
	input := sourceInput()
	input.Root.Entries = append(input.Root.Entries, Entry{
		Path: "model/feature/renamed.yaml", Kind: Regular,
		Bytes: []byte("id: item\nkind: Example\nbody: Another declaration.\nspec: {}"),
	})
	got := Compile(input)
	if got.Status != Failed || !slices.ContainsFunc(got.Diagnostics, func(d Diagnostic) bool {
		return d.Code == "name.definition_collision" && d.Source.Path == "model/feature/renamed.yaml"
	}) {
		t.Fatalf("same address in different files was not refused: %+v", got)
	}
	if got.Models[0].Phases[2].Status != Failed || got.Models[0].Phases[1].Status != Passed {
		t.Fatalf("identity failure was misattributed: %+v", got.Models[0].Phases)
	}
	if len(got.Models[0].Definitions) != 0 || len(got.Models[0].Canonical) != 0 || got.Models[0].Digest != "" {
		t.Fatal("partial identity check exposed validated IR or canonical evidence")
	}
	slices.Reverse(input.Root.Entries)
	if reversed := Compile(input); !reflect.DeepEqual(got, reversed) {
		t.Fatalf("declaration enumeration changed the identity report: %+v / %+v", got, reversed)
	}
}

func TestLocalIdentityBelongsOnlyToImmediatePackage(t *testing.T) {
	input := sourceInput()
	input.Root.Entries = append(input.Root.Entries,
		Entry{Path: "model/feature/child/item.yaml", Kind: Regular, Bytes: []byte("id: item\nkind: item\nbody: Child package.\nspec: {}")},
		Entry{Path: "model/other/item.yaml", Kind: Regular, Bytes: []byte("id: item\nkind: item\nbody: Other package.\nspec: {}")},
	)
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("separate Package identities were conflated: %+v", got)
	}
	if got.Models[0].Phases[2].Status != NotCompleted {
		t.Fatal("exact identity checks claimed complete Unicode naming support")
	}
}

func TestRootDefinitionsCannotAcquireAnEmptyPackageIdentity(t *testing.T) {
	input := sourceInput()
	input.Root.Entries[1].Path = "model/item.yaml"
	input.Root.Entries = append(input.Root.Entries, Entry{
		Path: "model/another.yaml", Kind: Regular,
		Bytes: []byte("id: item\nkind: Example\nbody: Also root level.\nspec: {}"),
	})
	got := Compile(input)
	if got.Status != Failed || slices.ContainsFunc(got.Diagnostics, func(d Diagnostic) bool {
		return d.Code == "name.definition_collision"
	}) {
		t.Fatalf("root-level files invented an empty Package: %+v", got)
	}
}

func dependencySource(id, origin, pkg, definitionID string) SourceModel {
	return SourceModel{Origin: origin, ManifestPath: "model.yaml", Entries: []Entry{
		{Path: "model.yaml", Kind: Regular, Bytes: []byte("formatVersion: 1\nid: " + id + "\nversion: '1.0'\nroot: model\ndependencies: {}")},
		{Path: "model/" + pkg + "/item.yaml", Kind: Regular, Bytes: []byte("id: " + definitionID + "\nkind: " + definitionID + "\nbody: text\nspec: {}")},
	}}
}

func TestPopulatedPackageHasOneModelOwnerEvenWithDistinctLocalIDs(t *testing.T) {
	input := sourceInput()
	input.Dependencies = []SourceModel{dependencySource("vocabulary", "dependency", "feature", "another")}
	got := Compile(input)
	if got.Status != Failed || len(got.Models) != 2 {
		t.Fatalf("shared Package ownership accepted: %+v", got)
	}
	count := 0
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Code == "name.package_ownership" {
			count++
		}
	}
	if count != 2 || got.Models[0].Status != Failed || got.Models[1].Status != Failed {
		t.Fatalf("all colliding owners must be reported without choosing one: %+v", got)
	}
}

func TestSharedIntermediateAndEmptyPackagesDoNotClaimGlobalOwnership(t *testing.T) {
	input := sourceInput()
	input.Dependencies = []SourceModel{
		dependencySource("first", "dependency-a", "feature/child", "item"),
		dependencySource("second", "dependency-b", "feature/sibling", "item"),
	}
	input.Dependencies[1].Entries = append(input.Dependencies[1].Entries, Entry{Path: "model/feature/child", Kind: Directory})
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("empty/intermediate Package claimed foreign Definitions: %+v", got)
	}
}

func TestSuppliedModelIdentityIsUniqueAndOrderIndependent(t *testing.T) {
	input := sourceInput()
	input.Dependencies = []SourceModel{
		dependencySource("same-model", "dependency-a", "a", "item"),
		dependencySource("same-model", "dependency-b", "b", "other"),
	}
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "name.model_collision") {
		t.Fatalf("duplicate model ID disappeared: %+v", got)
	}
	slices.Reverse(input.Dependencies)
	if reversed := Compile(input); !reflect.DeepEqual(got, reversed) {
		t.Fatalf("dependency enumeration selected a winner: %+v / %+v", got, reversed)
	}
}
