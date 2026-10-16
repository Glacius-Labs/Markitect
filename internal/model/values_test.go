package model

import (
	"bytes"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func singleValueInput(valueType, min, max, extra, spec string) Input {
	return coreModelInput(
		contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{properties: [field]}"),
		contractDefinition("feature/field.yaml", "field", "Core.Property", "{type: "+valueType+", minCount: "+min+", maxCount: "+max+extra+"}"),
		contractDefinition("feature/item.yaml", "item", "Record", spec),
	)
}

func privateValues(input Input) valueResult {
	scopes, graph, bindings := privateBindings(input)
	return checkValues(scopes, graph, checkContracts(scopes, graph, bindings))
}

func requireValuesPassed(t *testing.T, input Input) Value {
	t.Helper()
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("valid concrete values rejected: %+v", got)
	}
	for _, model := range got.Models {
		if phaseStatus(model, "values_and_references") != Passed || phaseStatus(model, "body_links") != Passed || phaseStatus(model, "lock") != NotCompleted || phaseStatus(model, "canonical") != NotCompleted || len(model.Definitions) != 0 || len(model.Canonical) != 0 || model.Digest != "" {
			t.Fatalf("typed private values claimed later/public content: %+v", model)
		}
	}
	return privateValues(input).specs[Identity{"feature", "item"}]
}

func TestConcreteValueTypesWithoutCoercion(t *testing.T) {
	for _, tt := range []struct {
		name, valueType, extra, literal string
		kind                            ValueKind
	}{
		{"string", "string", "", "'42'", StringValue},
		{"empty string", "string", "", "''", StringValue},
		{"boolean", "boolean", "", "true", BooleanValue},
		{"integer arbitrary precision", "integer", "", "12345678901234567890123456789012345678901234567890", IntegerValue},
		{"number exact decimal", "number", "", "1.2300e900000000000000000000000", NumberValue},
		{"number includes integer", "number", "", "1000", NumberValue},
		{"number negative zero", "number", "", "-0.00", NumberValue},
		{"enum", "enum", ", values: ['', exact, Exact]", "Exact", EnumValue},
		{"empty enum string", "enum", ", values: ['', exact]", "''", EnumValue},
		{"closed empty object", "object", "", "{}", ObjectValue},
		{"reference to actual Kind", "reference", ", target: Record", "item", ReferenceValue},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := singleValueInput(tt.valueType, "1", "1", tt.extra, "{field: "+tt.literal+"}")
			spec := requireValuesPassed(t, input)
			value, found := lookup(spec, "field")
			if !found || value.kind != tt.kind {
				t.Fatalf("wrong typed value: %+v", value)
			}
			switch tt.name {
			case "number exact decimal":
				if value.decimal != (Decimal{"123", "899999999999999999999998"}) {
					t.Fatalf("decimal rounded or exponent bounded: %+v", value.decimal)
				}
			case "number includes integer":
				if value.decimal != (Decimal{"1", "3"}) {
					t.Fatalf("integer numeric value lost: %+v", value)
				}
			case "number negative zero":
				if value.decimal != (Decimal{"0", "0"}) {
					t.Fatalf("zero normalization changed numeric value: %+v", value)
				}
			case "reference to actual Kind":
				if value.reference != (Identity{"feature", "item"}) {
					t.Fatalf("reference was not bound: %+v", value)
				}
			}
		})
	}
	for _, tt := range []struct{ name, valueType, extra, literal string }{
		{"number is not string", "string", "", "42"},
		{"quoted boolean", "boolean", "", "'true'"},
		{"capital boolean is text", "boolean", "", "True"},
		{"quoted integer", "integer", "", "'42'"},
		{"decimal is not integer", "integer", "", "42.0"},
		{"hex text is not number", "number", "", "0x10"},
		{"quoted number", "number", "", "'0.1'"},
		{"enum is exact", "enum", ", values: [exact]", "Exact"},
		{"enum boolean is not text", "enum", ", values: ['true']", "true"},
		{"object cannot be text", "object", "", "text"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Compile(singleValueInput(tt.valueType, "1", "1", tt.extra, "{field: "+tt.literal+"}"))
			if got.Status != Failed || !hasDiagnostic(got, "value.type") {
				t.Fatalf("coercion accepted: %+v", got)
			}
		})
	}
}

func TestValueCardinalityAndListShapes(t *testing.T) {
	for _, tt := range []struct{ name, min, max, spec, code string }{
		{"required absent", "1", "1", "{}", "value.cardinality"},
		{"single list of one", "0", "1", "{field: [a]}", "value.shape"},
		{"single empty list", "0", "1", "{field: []}", "value.shape"},
		{"list scalar", "0", "-1", "{field: a}", "value.shape"},
		{"finite list lower bound", "2", "3", "{field: [a]}", "value.cardinality"},
		{"finite list upper bound", "0", "2", "{field: [a, b, c]}", "value.cardinality"},
		{"zero list nonempty", "0", "0", "{field: [a]}", "value.cardinality"},
		{"huge minimum is not truncated", "99999999999999999999999999", "-1", "{field: []}", "value.cardinality"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Compile(singleValueInput("string", tt.min, tt.max, "", tt.spec))
			if got.Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("want %s: %+v", tt.code, got)
			}
		})
	}
	for _, tt := range []struct{ min, max, spec string }{
		{"0", "1", "{}"}, {"0", "0", "{}"}, {"0", "0", "{field: []}"},
		{"2", "2", "{field: [b, a]}"}, {"0", "-1", "{field: [b, a, b]}"},
	} {
		requireValuesPassed(t, singleValueInput("string", tt.min, tt.max, "", tt.spec))
	}
	input := singleValueInput("string", "0", "-1", "", "{field: [b, a, b]}")
	field, _ := lookup(requireValuesPassed(t, input), "field")
	if len(field.items) != 3 || field.items[0].text != "b" || field.items[1].text != "a" || field.items[2].text != "b" {
		t.Fatalf("ordinary list sorted or deduplicated: %+v", field)
	}
}

func nestedValueInput(spec string) Input {
	return coreModelInput(
		contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{properties: [node, rows]}"),
		contractDefinition("feature/node.yaml", "node", "Core.Property", "{type: object, minCount: 0, maxCount: 1, properties: [node, words, required]}"),
		contractDefinition("feature/rows.yaml", "rows", "Core.Property", "{type: object, minCount: 0, maxCount: -1, properties: [words]}"),
		contractDefinition("feature/words.yaml", "words", "Core.Property", "{type: string, minCount: 0, maxCount: -1}"),
		contractDefinition("feature/required.yaml", "required", "Core.Property", "{type: boolean, minCount: 1, maxCount: 1}"),
		contractDefinition("feature/item.yaml", "item", "Record", spec),
	)
}

func TestNestedDefaultsVisitOnlyExistingFiniteObjects(t *testing.T) {
	spec := requireValuesPassed(t, nestedValueInput("{node: {required: true, node: {required: false}}, rows: [{}, {words: [b, a, b]}]}"))
	node, _ := lookup(spec, "node")
	words, present := lookup(node, "words")
	if !present || words.kind != ListValue || len(words.items) != 0 {
		t.Fatalf("nested optional list was not defaulted: %+v", node)
	}
	child, _ := lookup(node, "node")
	if _, exists := lookup(child, "node"); exists {
		t.Fatal("absent optional object scalar was instantiated")
	}
	rows, _ := lookup(spec, "rows")
	if len(rows.items) != 2 {
		t.Fatal("object list changed")
	}
	if nested, ok := lookup(rows.items[0], "words"); !ok || nested.kind != ListValue || len(nested.items) != 0 {
		t.Fatal("default inside an existing list object is missing")
	}
	missing := requireValuesPassed(t, nestedValueInput("{}"))
	if _, exists := lookup(missing, "node"); exists {
		t.Fatal("optional absent object is different from present {}")
	}
	if rows, exists := lookup(missing, "rows"); !exists || rows.kind != ListValue || len(rows.items) != 0 {
		t.Fatal("optional object list was not defaulted to empty")
	}
	for _, tt := range []struct{ spec, code string }{
		{"{node: {}}", "value.cardinality"},
		{"{node: {required: true, unknown: text}}", "value.unknown_field"},
		{"{node: {required: true, node: {required: 'false'}}}", "value.type"},
		{"{rows: [text]}", "value.type"},
	} {
		got := Compile(nestedValueInput(tt.spec))
		if got.Status != Failed || !hasDiagnostic(got, tt.code) {
			t.Fatalf("nested %s was not checked: %+v", tt.code, got)
		}
	}
}

func TestClosedFieldsAndExactDeclaredSpelling(t *testing.T) {
	input := singleValueInput("string", "0", "1", ", name: displayName", "{displayname: a}")
	if got := Compile(input); got.Status != Failed || !hasDiagnostic(got, "value.field_spelling") {
		t.Fatalf("casefold matched spelling was accepted: %+v", got)
	}
	input.Root.Entries[3] = contractDefinition("feature/item.yaml", "item", "Record", "{field: a}")
	if got := Compile(input); got.Status != Failed || !hasDiagnostic(got, "value.unknown_field") {
		t.Fatalf("Property ID incorrectly replaced its explicit field name: %+v", got)
	}
}

func TestOrdinaryPropertiesDataAndConditionalMetaDefaults(t *testing.T) {
	input := coreModelInput(
		contractDefinition("feature/catalog.yaml", "Catalog", "Core.Kind", "{properties: [Core.properties, words, box]}"),
		contractDefinition("feature/words.yaml", "words", "Core.Property", "{type: string, minCount: 0, maxCount: -1}"),
		contractDefinition("feature/box.yaml", "box", "Core.Property", "{type: object, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/item.yaml", "item", "Catalog", "{properties: [words, box, words]}"),
	)
	spec := requireValuesPassed(t, input)
	properties, _ := lookup(spec, "properties")
	if len(properties.items) != 3 || properties.items[0].reference.ID != "words" || properties.items[1].reference.ID != "box" || properties.items[2].reference.ID != "words" {
		t.Fatal("ordinary properties field was treated as an unordered declaration")
	}
	typed := privateValues(input)
	words := typed.specs[Identity{"feature", "words"}]
	for _, forbidden := range []string{"properties", "values", "target"} {
		if _, exists := lookup(words, forbidden); exists {
			t.Fatalf("forbidden meta-field %s was invented by defaults", forbidden)
		}
	}
	box := typed.specs[Identity{"feature", "box"}]
	if fields, exists := lookup(box, "properties"); !exists || fields.kind != ListValue || len(fields.items) != 0 {
		t.Fatal("allowed optional object declaration list was not defaulted")
	}
}

func TestTypedValuesPreserveTextAndSourceBytes(t *testing.T) {
	input := singleValueInput("string", "1", "1", "", "{field: \" u\\u0308 A\\0B [[Missing]] \"}")
	entry := &input.Root.Entries[3]
	entry.Bytes = bytes.ReplaceAll(entry.Bytes, []byte("body: Contract fixture."), []byte("body: \" u\\u0308 A\\0B `[[Missing]]` \"\ntitle: \"u\\u0308\""))
	original := bytes.Clone(entry.Bytes)
	field, _ := lookup(requireValuesPassed(t, input), "field")
	if field.text != " u\u0308 A\x00B [[Missing]] " || !bytes.Equal(original, entry.Bytes) {
		t.Fatal("ordinary text or author bytes were rewritten")
	}
	parsed, problems, _ := parseYAML(entry.Bytes, 100)
	body, _ := lookup(parsed, "body")
	title, _ := lookup(parsed, "title")
	if len(problems) != 0 || body.text != " u\u0308 A\x00B `[[Missing]]` " || title.text != "u\u0308" {
		t.Fatal("body/title text did not remain unchanged")
	}
}

func TestOrdinaryPropertiesStringAndNestedListDefaults(t *testing.T) {
	input := coreModelInput(
		contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{properties: [properties]}"),
		contractDefinition("feature/properties.yaml", "properties", "Core.Property", "{type: string, minCount: 0, maxCount: 1}"),
		contractDefinition("feature/item.yaml", "item", "Record", "{properties: 'ordinary literal'}"),
	)
	// Fully qualify the declaration because both the own and Core Packages
	// contain a Property with this ID. The ordinary value is still a String.
	input.Root.Entries[1] = contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{properties: [feature/properties]}")
	properties, _ := lookup(requireValuesPassed(t, input), "properties")
	if properties.kind != StringValue || properties.text != "ordinary literal" {
		t.Fatal("ordinary field name installed a declaration-list operator")
	}
	missing := nestedValueInput("{rows: [{}]}")
	explicit := nestedValueInput("{rows: [{words: []}]}")
	a := requireValuesPassed(t, missing)
	b := requireValuesPassed(t, explicit)
	// Source positions remain separate provenance; compare the typed content.
	arows, _ := lookup(a, "rows")
	brows, _ := lookup(b, "rows")
	awords, aok := lookup(arows.items[0], "words")
	bwords, bok := lookup(brows.items[0], "words")
	if !aok || !bok || awords.kind != ListValue || bwords.kind != ListValue || len(awords.items) != 0 || len(bwords.items) != 0 {
		t.Fatal("missing/empty permitted nested lists have different typed content")
	}
}

func TestValueReportsAreEnumerationIndependent(t *testing.T) {
	input := nestedValueInput("{node: {required: 'true'}, rows: [text]}")
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "value.type") {
		t.Fatal("negative control did not fail")
	}
	slices.Reverse(input.Dependencies)
	slices.Reverse(input.Root.Entries)
	if reordered := Compile(input); !reflect.DeepEqual(got, reordered) {
		t.Fatalf("enumeration affected diagnostics: %+v / %+v", got, reordered)
	}
	for _, diagnostic := range got.Diagnostics {
		if strings.HasPrefix(diagnostic.Code, "value.") && (diagnostic.Source.Digest == "" || !strings.Contains(diagnostic.Message, "spec.")) {
			t.Fatalf("value failure lacks source binding/path: %+v", diagnostic)
		}
	}
}
