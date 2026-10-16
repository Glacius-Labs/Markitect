package model

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func bindingSource(id, version, dependencies string, definitions ...Entry) SourceModel {
	manifest := "formatVersion: 1\nid: " + id + "\nroot: model\ndependencies: " + dependencies + "\n"
	if version != "" {
		manifest += "version: '" + version + "'\n"
	}
	entries := append([]Entry{{Path: "model.yaml", Kind: Regular, Bytes: []byte(manifest)}}, definitions...)
	entries = append(entries, emptyLockEntry())
	return SourceModel{Origin: id, ManifestPath: "model.yaml", Entries: entries}
}

func bindingDefinition(path, id, kind, imports string) Entry {
	text := "id: " + id + "\nkind: " + kind + "\nbody: Binding fixture, not a complete Core conformance fixture.\nspec: {}\n"
	if imports != "" {
		text += "imports: " + imports + "\n"
	}
	return Entry{Path: "model/" + path, Kind: Regular, Bytes: []byte(text)}
}

func bindingInput() Input {
	return Input{
		Root: bindingSource("project", "", "{types: '1.x'}",
			bindingDefinition("app/item.yaml", "item", "Vocabulary.Thing", "[{package: vocabulary, as: Vocabulary}]")),
		Dependencies: []SourceModel{bindingSource("types", "1.x", "{}",
			bindingDefinition("vocabulary/thing.yaml", "Thing", "Thing", ""))},
	}
}

func phaseStatus(result ModelResult, name string) Status {
	for _, phase := range result.Phases {
		if phase.Name == name {
			return phase.Status
		}
	}
	return ""
}

func privateBindings(input Input) ([]sourceScope, dependencyGraph, authorBindings) {
	var scopes []sourceScope
	for _, model := range append([]SourceModel{input.Root}, input.Dependencies...) {
		_, _, scope := compileSource(model, effectiveLimits(input.Limits))
		scopes = append(scopes, scope)
	}
	graph := bindDependencies(scopes)
	bindings := bindAuthorReferences(scopes, graph)
	return scopes, graph, bindings
}

func TestExplicitModelGraphAndFileLocalKindBinding(t *testing.T) {
	input := bindingInput()
	got := Compile(input)
	if !onlyLaterStageFailures(got) {
		t.Fatalf("valid model/author binding rejected: %+v", got)
	}
	for _, model := range got.Models {
		if phaseStatus(model, "dependencies") != Passed || phaseStatus(model, "imports_and_kinds") != Passed {
			t.Fatalf("completed address/graph phases not reported: %+v", model.Phases)
		}
		if model.Status != Failed || len(model.Definitions) != 0 || len(model.Canonical) != 0 || model.Digest != "" || phaseStatus(model, "core_and_contracts") != Failed {
			t.Fatalf("identity binding was presented as validated Core/IR/content: %+v", model)
		}
	}
	scopes, _, bindings := privateBindings(input)
	target := scopes[0].definitions[0].kindTarget
	if bindings.status[0] != Passed || target == nil || *target != (Identity{Package: "vocabulary", ID: "Thing"}) {
		t.Fatalf("wrong resolved acquisition address: target=%+v binding=%+v", target, bindings)
	}
}

func TestParsedDependencyManifestBindings(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Input)
		code string
	}{
		{"version labels have no wildcard semantics", func(i *Input) {
			i.Dependencies[0] = bindingSource("types", "1.0", "{}", i.Dependencies[0].Entries[1])
		}, "dependency.version_mismatch"},
		{"missing supplied dependency", func(i *Input) { i.Dependencies = nil }, "dependency.missing"},
		{"supplied dependency requires version", func(i *Input) {
			i.Dependencies[0] = bindingSource("types", "", "{}", i.Dependencies[0].Entries[1])
		}, "dependency.provided_version"},
		{"model ID spelling is exact", func(i *Input) {
			i.Root.Entries[0] = bindingSource("project", "", "{strasse: '1.x'}").Entries[0]
			i.Dependencies[0] = bindingSource("straße", "1.x", "{}", i.Dependencies[0].Entries[1])
		}, "dependency.spelling"},
		{"exact model ID cannot hide colliding candidate", func(i *Input) {
			i.Root.Entries[0] = bindingSource("project", "", "{strasse: '1.x'}").Entries[0]
			i.Dependencies[0] = bindingSource("strasse", "1.x", "{}", i.Dependencies[0].Entries[1])
			i.Dependencies = append(i.Dependencies, bindingSource("straße", "1.x", "{}",
				bindingDefinition("extra/item.yaml", "item", "item", "")))
		}, "dependency.ambiguous"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := bindingInput()
			tt.edit(&input)
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("expected %s from parsed manifests: %+v", tt.code, got)
			}
		})
	}
}

func TestAliasImportAlsoOpensShortIDs(t *testing.T) {
	input := bindingInput()
	input.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Thing", "[{package: vocabulary, as: Vocabulary}]")
	if got := Compile(input); !onlyLaterStageFailures(got) {
		t.Fatalf("alias unexpectedly hid the imported short ID: %+v", got)
	}
}

func TestImportsAndAuthorReferenceBoundaries(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Input)
		code string
	}{
		{"model availability grants no import", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "vocabulary/Thing", "")
		}, "binding.package_not_visible"},
		{"alias does not cross files", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, bindingDefinition("app/another.yaml", "another", "Vocabulary.Thing", ""))
		}, "binding.alias_missing"},
		{"foreign model needs direct requirement", func(i *Input) {
			i.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: project\nroot: model\ndependencies: {}")
		}, "binding.model_visibility"},
		{"unused import is checked", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "item", "[{package: absent}]")
		}, "binding.package_missing"},
		{"empty Package is not importable", func(i *Input) {
			i.Dependencies[0].Entries = append(i.Dependencies[0].Entries, Entry{Path: "model/empty", Kind: Directory})
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "item", "[{package: empty}]")
		}, "binding.package_missing"},
		{"intermediate Package is not importable", func(i *Input) {
			i.Dependencies[0].Entries[1].Path = "model/vocabulary/child/thing.yaml"
		}, "binding.package_missing"},
		{"parent Package is not inherited", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, bindingDefinition("app/child/child.yaml", "child", "item", ""))
		}, "binding.reference_missing"},
		{"missing local ID", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Missing", "")
		}, "binding.reference_missing"},
		{"alias spelling is exact", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "vocabulary.Thing", "[{package: vocabulary, as: Vocabulary}]")
		}, "binding.reference_spelling"},
		{"reference ID spelling is exact", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Vocabulary.thing", "[{package: vocabulary, as: Vocabulary}]")
		}, "binding.reference_spelling"},
		{"full Package spelling is exact", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Vocabulary/Thing", "[{package: vocabulary}]")
		}, "binding.reference_spelling"},
		{"imported Package spelling is exact", func(i *Input) {
			i.Dependencies[0].Entries[1].Path = "model/straße/thing.yaml"
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "item", "[{package: strasse}]")
		}, "binding.package_spelling"},
		{"alias collision cannot choose exact candidate", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Straße.Thing", "[{package: vocabulary, as: Straße}, {package: vocabulary, as: Strasse}]")
		}, "binding.alias_ambiguous"},
		{"no field dereference", func(i *Input) {
			i.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Vocabulary.Thing.field", "[{package: vocabulary, as: Vocabulary}]")
		}, "binding.reference_syntax"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := bindingInput()
			tt.edit(&input)
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("expected %s: %+v", tt.code, got)
			}
			for _, diagnostic := range got.Diagnostics {
				if diagnostic.Code == tt.code && diagnostic.Source.Digest == "" {
					t.Fatalf("binding diagnostic is not source-byte bound: %+v", diagnostic)
				}
			}
		})
	}
}

func TestSameModelPackagesNeedImportsAndOwnFullAddressNeedsNone(t *testing.T) {
	input := Input{Root: bindingSource("local", "", "{}",
		bindingDefinition("app/item.yaml", "item", "other/Thing", "[{package: other}]"),
		bindingDefinition("other/thing.yaml", "Thing", "other/Thing", ""))}
	if got := Compile(input); !onlyLaterStageFailures(got) {
		t.Fatalf("same-model import or own full address rejected: %+v", got)
	}
	input.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "other/Thing", "")
	if got := Compile(input); !hasDiagnostic(got, "binding.package_not_visible") {
		t.Fatalf("same-model foreign Package became implicitly visible: %+v", got)
	}
}

func TestNoLocalPriorityAndNoErrorForUnusedCompetingShortNames(t *testing.T) {
	input := bindingInput()
	input.Root.Entries = append(input.Root.Entries, bindingDefinition("app/thing.yaml", "Thing", "Thing", ""))
	input.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Thing", "[{package: vocabulary, as: Vocabulary}]")
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "binding.reference_ambiguous") {
		t.Fatalf("local candidate took priority over imported short ID: %+v", got)
	}
	input.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Vocabulary.Thing", "[{package: vocabulary, as: Vocabulary}]")
	if got := Compile(input); !onlyLaterStageFailures(got) {
		t.Fatalf("unused competing short IDs invalidated imports: %+v", got)
	}
	input.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "vocabulary/Thing", "[{package: vocabulary, as: Vocabulary}]")
	if got := Compile(input); !onlyLaterStageFailures(got) {
		t.Fatalf("alias disabled full Package qualification: %+v", got)
	}
}

func TestDirectRequirementCannotBeReplacedByTransitiveVisibility(t *testing.T) {
	input := bindingInput()
	input.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: project\nroot: model\ndependencies: {middle: '1.0'}")
	input.Dependencies = append(input.Dependencies, bindingSource("middle", "1.0", "{types: '1.x'}",
		bindingDefinition("middle/item.yaml", "item", "item", "")))
	if got := Compile(input); !hasDiagnostic(got, "binding.model_visibility") {
		t.Fatalf("transitive dependency opened a Package: %+v", got)
	}
	input.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: project\nroot: model\ndependencies: {middle: '1.0', types: '1.x'}")
	got := Compile(input)
	if !onlyLaterStageFailures(got) {
		t.Fatalf("explicit direct requirement was not honored: %+v", got)
	}
	slices.Reverse(input.Dependencies)
	slices.Reverse(input.Root.Entries)
	if reordered := Compile(input); !reflect.DeepEqual(got, reordered) {
		t.Fatalf("model/entry enumeration changed binding evidence: %+v / %+v", got, reordered)
	}
}

func TestPackageReferenceCyclesDoNotBecomeModelDependencyCycles(t *testing.T) {
	input := Input{Root: bindingSource("local", "", "{}",
		bindingDefinition("a/thing.yaml", "Thing", "b/Other", "[{package: b}]"),
		bindingDefinition("b/other.yaml", "Other", "a/Thing", "[{package: a}]"))}
	got := Compile(input)
	if !onlyLaterStageFailures(got) {
		t.Fatalf("ordinary cross-Package cycle rejected: %+v", got)
	}
}

func TestUnicodeBindingNeverChoosesAnExactSubset(t *testing.T) {
	input := Input{Root: bindingSource("local", "", "{}",
		bindingDefinition("feature/one.yaml", "Straße", "Straße", ""),
		bindingDefinition("feature/two.yaml", "Strasse", "Strasse", ""))}
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "binding.reference_ambiguous") {
		t.Fatalf("exact spelling selected one of colliding IDs: %+v", got)
	}
	scopes, _, _ := privateBindings(input)
	for _, definition := range scopes[0].definitions {
		if definition.kindTarget != nil {
			t.Fatal("ambiguous declaration became a resolved Kind address")
		}
	}
}

func TestResourceIncompleteMetadataIsNotMissingModelEvidence(t *testing.T) {
	input := bindingInput()
	input.Root = withLock(input.Root, "formatVersion: 1\nmodels: {types: {version: '1.x', digest: 'sha256:"+strings.Repeat("1", 64)+"'}}", true)
	input.Limits.YAMLDepth = 3
	input.Dependencies[0].Entries[0].Bytes = []byte("formatVersion: 1\nid: types\nroot: model\nversion: '1.x'\ndependencies: {}\nextra: {nested: {deep: value}}")
	got := Compile(input)
	if got.Status != NotCompleted || !hasDiagnostic(got, "limit.yaml_depth") || hasDiagnostic(got, "dependency.missing") || hasDiagnostic(got, "dependency.provided_version") {
		t.Fatalf("unknown metadata became a language violation: %+v", got)
	}
}

func TestResourceIncompleteInventoryCannotProveMissingReference(t *testing.T) {
	input := bindingInput()
	input.Root = withLock(input.Root, "formatVersion: 1\nmodels: {types: {version: '1.x', digest: 'sha256:"+strings.Repeat("1", 64)+"'}}", true)
	input.Limits.YAMLDepth = 4
	input.Dependencies[0].Entries = append(input.Dependencies[0].Entries, Entry{
		Path: "model/vocabulary/partial.yaml", Kind: Regular,
		Bytes: []byte("id: Other\nkind: Other\nbody: Partial inventory\nspec: {a: {b: {c: value}}}"),
	})
	input.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", "Vocabulary.Other", "[{package: vocabulary, as: Vocabulary}]")
	got := Compile(input)
	if got.Status != NotCompleted || !hasDiagnostic(got, "limit.yaml_depth") || hasDiagnostic(got, "binding.reference_missing") {
		t.Fatalf("an unacquired Definition was treated as proven absent: %+v", got)
	}
	scopes, _, _ := privateBindings(input)
	if len(scopes[0].definitions) != 1 || scopes[0].definitions[0].kindTarget != nil {
		t.Fatal("partial inventory produced a bound Kind address")
	}
}

func TestKnownVisibilityErrorsSurviveUnrelatedIncompleteInventory(t *testing.T) {
	for _, tt := range []struct{ kind, imports, code string }{
		{"vocabulary/Thing", "", "binding.package_not_visible"},
		{"Vocabulary.Thing", "", "binding.alias_missing"},
		{"item", "[{package: vocabulary}]", "binding.model_visibility"},
	} {
		t.Run(tt.code, func(t *testing.T) {
			input := bindingInput()
			input.Limits.YAMLDepth = 4
			input.Root.Entries[0] = bindingSource("project", "", "{middle: '1.0'}").Entries[0]
			input.Root.Entries[1] = bindingDefinition("app/item.yaml", "item", tt.kind, tt.imports)
			input.Dependencies = append(input.Dependencies, bindingSource("middle", "1.0", "{}", Entry{
				Path: "model/middle/partial.yaml", Kind: Regular,
				Bytes: []byte("id: Other\nkind: Other\nbody: Partial\nspec: {a: {b: {c: value}}}"),
			}))
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, "limit.yaml_depth") || !hasDiagnostic(got, tt.code) {
				t.Fatalf("known file-local violation was suppressed by unrelated refusal: %+v", got)
			}
		})
	}
}

func TestKnownDependencyCycleSurvivesAnIndependentResourceRefusal(t *testing.T) {
	input := Input{Root: bindingSource("project", "1.0", "{middle: '1.0'}", bindingDefinition("app/item.yaml", "item", "item", "")),
		Dependencies: []SourceModel{bindingSource("middle", "1.0", "{project: '1.0'}", bindingDefinition("middle/item.yaml", "item", "item", ""))},
		Limits:       Limits{YAMLDepth: 3}}
	input.Dependencies = append(input.Dependencies, SourceModel{Origin: "incomplete", ManifestPath: "model.yaml", Entries: []Entry{
		{Path: "model.yaml", Kind: Regular, Bytes: []byte("deep: {nested: {too: deep}}")},
	}})
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "dependency.cycle") || !hasDiagnostic(got, "limit.yaml_depth") {
		t.Fatalf("known cycle was lost beside a resource refusal: %+v", got)
	}
}

func TestUnclassifiedSpecStringsDoNotInstallOperators(t *testing.T) {
	input := sourceInput()
	input.Root.Entries[1].Bytes = []byte("id: item\nkind: item\nbody: '[[Missing]]'\nspec: {target: Missing, properties: [Missing]}")
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "binding.reference_missing") || phaseStatus(got.Models[0], "values_and_references") == Passed || phaseStatus(got.Models[0], "body_links") != Failed {
		t.Fatalf("ordinary strings were treated as Core operators or body links prematurely: %+v", got)
	}
}
