package model

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func contractDefinition(path, id, kind, spec string) Entry {
	return Entry{Path: "model/" + path, Kind: Regular, Bytes: []byte("id: " + id + "\nkind: " + kind + "\nbody: Contract fixture.\nimports: [{package: markitect/core, as: Core}]\nspec: " + spec)}
}

func TestFullPrescribedCoreCompletesOnlyImplementedPhases(t *testing.T) {
	input := Input{Root: completeCoreSource()}
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 || len(got.Models) != 1 {
		t.Fatalf("prescribed nine-file Core rejected: %+v", got)
	}
	model := got.Models[0]
	for _, phase := range model.Phases {
		switch phase.Name {
		case "source_structure", "yaml_and_envelopes", "names_and_unicode", "dependencies", "imports_and_kinds", "core_and_contracts", "values_and_references", "body_links":
			if phase.Status != Passed {
				t.Fatalf("implemented phase incomplete: %+v", phase)
			}
		default:
			if phase.Status != NotCompleted {
				t.Fatalf("later phase claimed completion: %+v", phase)
			}
		}
	}
	if model.Status != NotCompleted || len(model.Definitions) != 0 || len(model.Canonical) != 0 || model.Digest != "" {
		t.Fatal("contract check produced full IR or canonical evidence")
	}
	slices.Reverse(input.Root.Entries)
	if reversed := Compile(input); !reflect.DeepEqual(got, reversed) {
		t.Fatalf("Core report depends on entry order: %+v / %+v", got, reversed)
	}
}

func TestCoreComparesEffectiveContent(t *testing.T) {
	input := Input{Root: completeCoreSource()}
	for index := range input.Root.Entries {
		entry := &input.Root.Entries[index]
		text := string(entry.Bytes)
		if entry.Path == "model/markitect/core/property.yaml" {
			updated := strings.Replace(text, "    - name\n    - type\n    - minCount\n    - maxCount\n    - values\n    - target\n    - properties", "    - Core.properties\n    - Core.target\n    - Core.values\n    - Core.maxCount\n    - Core.minCount\n    - Core.type\n    - Core.name", 1)
			if updated == text {
				t.Fatal("equivalence edit did not reorder and qualify Core fields")
			}
			text = updated
		}
		if entry.Path == "model/markitect/core/type.yaml" {
			text = strings.Replace(text, "    - string\n    - boolean", "    - boolean\n    - string", 1)
		}
		if strings.HasPrefix(entry.Path, "model/markitect/core/") {
			text += "\nimports: [{package: markitect/core, as: Core}]\n"
			text = strings.Replace(text, "kind: Kind", "kind: markitect/core/Kind", 1)
			text = strings.Replace(text, "kind: Property", "kind: Core.Property", 1)
			text = strings.Replace(text, "target: Kind", "target: Core.Kind", 1)
			text = strings.Replace(text, "target: Property", "target: markitect/core/Property", 1)
			if !strings.HasSuffix(entry.Path, "/kind.yaml") && !strings.HasSuffix(entry.Path, "/property.yaml") {
				idLine := strings.Split(text, "\n")[0]
				text = strings.Replace(text, "spec:", "spec:\n  name: "+strings.TrimPrefix(idLine, "id: "), 1)
			}
		}
		entry.Bytes = []byte(text)
	}
	if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("equivalent effective Core rejected: %+v", got)
	}
}

func TestFixedCoreRejectsContentReplacements(t *testing.T) {
	for _, tt := range []struct{ name, path, before, after, code string }{
		{"body", "type.yaml", "Der vom Core verstandene Werttyp dieser Property.", "Another body.", "core.content"},
		{"title", "kind.yaml", "id: Kind", "title: Extra\nid: Kind", "core.content"},
		{"cardinality", "name.yaml", "maxCount: 1", "maxCount: -1", "core.content"},
		{"field default", "name.yaml", "spec:", "spec:\n  name: other", "core.content"},
		{"missing field", "property.yaml", "    - target\n", "", "core.content"},
		{"extra field", "kind.yaml", "spec:", "spec:\n  extra: []", "contract.unknown_field"},
		{"duplicate field", "kind.yaml", "    - properties", "    - properties\n    - markitect/core/properties", "contract.property_duplicate"},
		{"bad enum", "type.yaml", "    - boolean", "    - other", "core.content"},
		{"case exact enum", "type.yaml", "    - boolean", "    - Boolean", "core.content"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := Input{Root: completeCoreSource()}
			for index := range input.Root.Entries {
				entry := &input.Root.Entries[index]
				if entry.Path == "model/markitect/core/"+tt.path {
					text := strings.Replace(string(entry.Bytes), tt.before, tt.after, 1)
					if text == string(entry.Bytes) {
						t.Fatal("negative edit did not change fixture")
					}
					entry.Bytes = []byte(text)
				}
			}
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("Core replacement accepted; want %s: %+v", tt.code, got)
			}
		})
	}
}

func TestDerivedContractsAndOrdinaryPropertiesField(t *testing.T) {
	input := coreModelInput(
		contractDefinition("vocabulary/catalog.yaml", "Catalog", "Core.Kind", "{properties: [Core.properties]}"),
		contractDefinition("vocabulary/display-name.yaml", "displayName", "Core.Property", "{type: string, minCount: 0, maxCount: 1}"),
		contractDefinition("vocabulary/nested.yaml", "nested", "Core.Property", "{type: object, minCount: 0, maxCount: -1, properties: [displayName, nested]}"),
		contractDefinition("vocabulary/item.yaml", "item", "Catalog", "{properties: [Core.name, Core.name]}"),
	)
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 || phaseStatus(got.Models[0], "core_and_contracts") != Passed || phaseStatus(got.Models[0], "values_and_references") != Passed {
		t.Fatalf("contracts or ordinary data field misinterpreted: %+v", got)
	}
}

func TestContractMetadataAndTargetTypes(t *testing.T) {
	for _, tt := range []struct{ name, id, kind, spec, code string }{
		{"unknown value type", "field", "Core.Property", "{type: kindReference, minCount: 0, maxCount: 1}", "contract.value_type"},
		{"missing minimum", "field", "Core.Property", "{type: string, maxCount: 1}", "contract.cardinality"},
		{"decimal minimum", "field", "Core.Property", "{type: string, minCount: 0.0, maxCount: 1}", "contract.cardinality"},
		{"negative minimum", "field", "Core.Property", "{type: string, minCount: -1, maxCount: 1}", "contract.cardinality"},
		{"max below min", "field", "Core.Property", "{type: string, minCount: 2, maxCount: 1}", "contract.cardinality"},
		{"invalid negative max", "field", "Core.Property", "{type: string, minCount: 0, maxCount: -2}", "contract.cardinality"},
		{"empty explicit name", "field", "Core.Property", "{name: '', type: string, minCount: 0, maxCount: 1}", "contract.field_name"},
		{"dotted name", "field", "Core.Property", "{name: a.b, type: string, minCount: 0, maxCount: 1}", "contract.field_name"},
		{"bad kind ID", "wrongKind", "Core.Kind", "{}", "name.category"},
		{"bad property ID despite name", "Wrong", "Core.Property", "{name: field, type: string, minCount: 0, maxCount: 1}", "name.category"},
		{"bad concrete ID", "Wrong", "Example", "{}", "name.category"},
		{"unknown kind spec", "Wrong", "Core.Kind", "{other: []}", "contract.unknown_field"},
		{"forbidden object fields", "field", "Core.Property", "{type: string, minCount: 0, maxCount: 1, properties: []}", "contract.forbidden_field"},
		{"forbidden values", "field", "Core.Property", "{type: object, minCount: 0, maxCount: 1, values: []}", "contract.forbidden_field"},
		{"forbidden target", "field", "Core.Property", "{type: string, minCount: 0, maxCount: 1, target: Example}", "contract.forbidden_field"},
		{"empty enum", "field", "Core.Property", "{type: enum, minCount: 0, maxCount: 1, values: []}", "contract.enum_values"},
		{"duplicate exact enum", "field", "Core.Property", "{type: enum, minCount: 0, maxCount: 1, values: [a, a]}", "contract.enum_values"},
		{"nonstring enum", "field", "Core.Property", "{type: enum, minCount: 0, maxCount: 1, values: [1]}", "contract.enum_values"},
		{"reference needs target", "field", "Core.Property", "{type: reference, minCount: 0, maxCount: 1}", "contract.reference"},
		{"target must be Kind", "field", "Core.Property", "{type: reference, minCount: 0, maxCount: 1, target: Core.name}", "contract.reference_target"},
		{"field must be Property", "Wrong", "Core.Kind", "{properties: [Example]}", "contract.reference_target"},
		{"properties must be list", "Wrong", "Core.Kind", "{properties: Core.name}", "contract.properties_list"},
		{"same target twice", "Wrong", "Core.Kind", "{properties: [Core.name, markitect/core/name]}", "contract.property_duplicate"},
		{"unknown property spec", "field", "Core.Property", "{type: string, minCount: 0, maxCount: 1, extra: 1}", "contract.unknown_field"},
		{"kind cannot target Property", "item", "Core.name", "{}", "contract.kind_target"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := coreModelInput(contractDefinition("feature/example.yaml", "Example", "Core.Kind", "{}"), contractDefinition("feature/test.yaml", tt.id, tt.kind, tt.spec))
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("want %s: %+v", tt.code, got)
			}
		})
	}
}

func TestContractFieldNameCollisionsAndFullDefaultIDs(t *testing.T) {
	input := coreModelInput(
		contractDefinition("feature/example.yaml", "Example", "Core.Kind", "{properties: [displayName, other]}"),
		contractDefinition("feature/name.yaml", "displayName", "Core.Property", "{type: string, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/other.yaml", "other", "Core.Property", "{name: displayName, type: string, minCount: 0, maxCount: 1}"),
	)
	if got := Compile(input); !hasDiagnostic(got, "contract.field_collision") {
		t.Fatalf("effective field collision not detected: %+v", got)
	}
	input.Root.Entries[3] = contractDefinition("feature/other.yaml", "other", "Core.Property", "{name: different, type: string, minCount: 0, maxCount: 1}")
	if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("full local default ID was shortened or otherwise changed: %+v", got)
	}
}

func TestMandatoryObjectCyclesAndOptionalTermination(t *testing.T) {
	for _, tt := range []struct {
		name, aMin, bMin, aMax string
		fail                   bool
	}{
		{"mandatory single values", "1", "1", "1", true},
		{"mandatory lists", "1", "1", "-1", true},
		{"optional first edge", "0", "1", "1", false},
		{"optional second edge", "1", "0", "1", false},
		{"empty optional list", "0", "0", "0", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := coreModelInput(
				contractDefinition("feature/a.yaml", "a", "Core.Property", "{type: object, minCount: "+tt.aMin+", maxCount: "+tt.aMax+", properties: [b]}"),
				contractDefinition("feature/b.yaml", "b", "Core.Property", "{type: object, minCount: "+tt.bMin+", maxCount: 1, properties: [a]}"),
			)
			got := Compile(input)
			if tt.fail {
				if got.Status != Failed || !hasDiagnostic(got, "contract.object_cycle") {
					t.Fatalf("unused impossible object contract accepted: %+v", got)
				}
			} else if got.Status != NotCompleted || len(got.Diagnostics) != 0 {
				t.Fatalf("terminating optional contract cycle rejected: %+v", got)
			}
		})
	}
}

func TestCoreDefinitionSetAndUnusedImports(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		input := Input{Root: completeCoreSource()}
		input.Root.Entries = slices.DeleteFunc(input.Root.Entries, func(e Entry) bool { return strings.HasSuffix(e.Path, "/name.yaml") })
		if got := Compile(input); got.Status != Failed || !hasDiagnostic(got, "core.missing") {
			t.Fatalf("missing prescribed Core definition accepted: %+v", got)
		}
	})
	t.Run("extra", func(t *testing.T) {
		input := Input{Root: completeCoreSource()}
		input.Root.Entries = append(input.Root.Entries, bindingDefinition("markitect/core/extra.yaml", "Extra", "Kind", ""))
		if got := Compile(input); got.Status != Failed || !hasDiagnostic(got, "core.definition_set") {
			t.Fatalf("extra Core definition accepted: %+v", got)
		}
	})
	t.Run("two supplied cores", func(t *testing.T) {
		input := Input{Root: completeCoreSource(), Dependencies: []SourceModel{completeCoreSource()}}
		input.Dependencies[0].Origin = "second-core"
		got := Compile(input)
		if got.Status != Failed || !hasDiagnostic(got, "core.definition_set") {
			t.Fatalf("duplicate Core owner accepted: %+v", got)
		}
		slices.Reverse(input.Dependencies[0].Entries)
		if reordered := Compile(input); !reflect.DeepEqual(got, reordered) {
			t.Fatalf("Core candidate order selected an owner: %+v / %+v", got, reordered)
		}
	})
	t.Run("unused valid import", func(t *testing.T) {
		input := Input{Root: completeCoreSource()}
		for index := range input.Root.Entries {
			entry := &input.Root.Entries[index]
			if strings.Contains(entry.Path, "/markitect/core/") {
				entry.Bytes = append(entry.Bytes, []byte("\nimports: [{package: unused}]\n")...)
			}
		}
		input.Root.Entries = append(input.Root.Entries, contractDefinition("unused/aux.yaml", "Aux", "Core.Kind", "{}"))
		if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 {
			t.Fatalf("valid unused import changed effective Core content: %+v", got)
		}
	})
}

func TestLocalKindAndPropertyNamesDoNotInstallMetaOperators(t *testing.T) {
	input := coreModelInput(
		contractDefinition("feature/kind.yaml", "Kind", "Core.Kind", "{}"),
		contractDefinition("feature/property.yaml", "Property", "Core.Kind", "{}"),
		contractDefinition("feature/item.yaml", "item", "feature/Property", "{}"),
	)
	if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("local Property name granted meta semantics: %+v", got)
	}
	input.Root.Entries[3] = contractDefinition("feature/item.yaml", "camelName", "feature/Kind", "{properties: [Missing]}")
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "name.category") || hasDiagnostic(got, "binding.reference_missing") || hasDiagnostic(got, "contract.reference_target") {
		t.Fatalf("local Kind name changed concrete category or interpreted ordinary properties: %+v", got)
	}
}

func TestLargeCountsAndExactEnumStrings(t *testing.T) {
	input := coreModelInput(
		contractDefinition("feature/large.yaml", "large", "Core.Property", "{type: string, minCount: 999999999999999999999999999999999999999999, maxCount: -1}"),
		contractDefinition("feature/values.yaml", "choices", "Core.Property", "{type: enum, minCount: 0, maxCount: 0, values: ['', a, A, 'ü', ü]}"),
	)
	if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("counts were bounded or enum text was normalized/casefolded: %+v", got)
	}
}

func TestKnownContractErrorsSurviveUnrelatedResourceRefusal(t *testing.T) {
	for _, tt := range []struct {
		name, code  string
		definitions []Entry
	}{
		{"cardinality", "contract.cardinality", []Entry{contractDefinition("feature/a.yaml", "a", "Core.Property", "{type: string, minCount: -1, maxCount: 1}")}},
		{"mandatory cycle", "contract.object_cycle", []Entry{contractDefinition("feature/a.yaml", "a", "Core.Property", "{type: object, minCount: 1, maxCount: 1, properties: [a]}")}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := coreModelInput(tt.definitions...)
			input.Limits.YAMLDepth = 4
			input.Dependencies = append(input.Dependencies, bindingSource("incomplete", "1.0", "{}", Entry{
				Path: "model/other/partial.yaml", Kind: Regular, Bytes: []byte("spec: {a: {b: {c: {d: text}}}}"),
			}))
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, tt.code) || !hasDiagnostic(got, "limit.yaml_depth") {
				t.Fatalf("known error was hidden by independent limit: %+v", got)
			}
		})
	}
}

func TestResolvedUnicodeCategoriesAndEffectiveFieldCollisions(t *testing.T) {
	input := coreModelInput(
		contractDefinition("feature/文档.yaml", "文档", "Core.Kind", "{properties: [größe]}"),
		contractDefinition("feature/größe.yaml", "größe", "Core.Property", "{type: integer, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/größe-٢.yaml", "größe-٢", "文档", "{größe: 2}"),
	)
	if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 || phaseStatus(got.Models[0], "names_and_unicode") != Passed {
		t.Fatalf("resolved Unicode categories were rejected: %+v", got)
	}
	input = coreModelInput(
		contractDefinition("feature/example.yaml", "Example", "Core.Kind", "{properties: [one, two]}"),
		contractDefinition("feature/one.yaml", "one", "Core.Property", "{name: straße, type: string, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/two.yaml", "two", "Core.Property", "{name: strasse, type: string, minCount: 0, maxCount: 1}"),
	)
	if got := Compile(input); got.Status != Failed || !hasDiagnostic(got, "contract.field_collision") {
		t.Fatalf("Unicode effective field collision was not detected: %+v", got)
	}
}
