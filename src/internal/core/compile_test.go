package core

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

const (
	domainAPI = "domain.example.org/v1"
	appAPI    = "app.example.org/v1"
)

type hostileJSONValue struct{ calls *int }

func (value hostileJSONValue) MarshalJSON() ([]byte, error) {
	*value.calls++
	return []byte(`"hostile"`), nil
}

func fixtureProperty(kind, targetAPI, targetKind string, min, max int) Property {
	p := Property{Purpose: "Supports the fixture behavior.", Type: kind, MinCount: min, MaxCount: max}
	if kind == TypeReference {
		p.Target = &KindIdentity{APIVersion: targetAPI, Kind: targetKind}
	}
	return p
}

func fixtureSchemas() []Schema {
	return []Schema{
		{
			APIVersion: domainAPI, Purpose: "Declares reusable domain facts.", Source: Source{Path: "schemas/domain.yaml", Digest: "schema-domain", Line: 1},
			Kinds: map[string]Kind{
				"Aggregate": {Purpose: "Owns one consistency boundary.", Properties: map[string]Property{
					"amount": fixtureProperty(TypeNumber, "", "", 1, 1),
					"next":   fixtureProperty(TypeReference, domainAPI, "Aggregate", 0, 1),
				}},
			},
		},
		{
			APIVersion: appAPI, Purpose: "Declares application behavior.", Source: Source{Path: "schemas/app.yaml", Digest: "schema-app", Line: 1},
			Kinds: map[string]Kind{
				"UseCase": {Purpose: "Represents one application behavior.", Properties: map[string]Property{
					"title":     fixtureProperty(TypeString, "", "", 1, 1),
					"parts":     fixtureProperty(TypeReference, domainAPI, "Aggregate", 0, Unbounded),
					"appliesTo": fixtureProperty(TypeKindReference, "", "", 0, 1),
					"children": {Purpose: "Groups child references.", Type: TypeObject, MinCount: 0, MaxCount: Unbounded, Properties: map[string]Property{
						"link": fixtureProperty(TypeReference, domainAPI, "Aggregate", 1, 1),
					}},
					"settings": {Purpose: "Provides local options.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: map[string]Property{
						"mode":    {Purpose: "Selects the processing mode.", Type: TypeEnum, MinCount: 1, MaxCount: 1, Values: []string{"safe", "fast"}},
						"enabled": fixtureProperty(TypeBoolean, "", "", 0, 1),
					}},
				}},
			},
		},
	}
}

func fixtureDefinitions() []Definition {
	return []Definition{
		{
			APIVersion: domainAPI, Kind: "Aggregate", Metadata: Metadata{Namespace: "orders", Name: "order"}, Purpose: "Tracks one order amount.",
			Spec:   map[string]any{"amount": 12.5, "next": map[string]any{"namespace": "orders", "name": "order"}},
			Source: Source{Path: "model/order.yaml", Digest: "order-bytes", Line: 4},
		},
		{
			APIVersion: appAPI, Kind: "UseCase", Metadata: Metadata{Namespace: "orders", Name: "create"}, Purpose: "Creates an order.",
			Spec: map[string]any{
				"title":     "Create order",
				"parts":     []any{map[string]any{"namespace": "orders", "name": "order"}},
				"appliesTo": map[string]any{"apiVersion": domainAPI, "kind": "Aggregate"},
				"settings":  map[string]any{"mode": "safe", "enabled": true},
			},
			Source: Source{Path: "model/create.yaml", Digest: "create-bytes", Line: 8},
		},
	}
}

func TestCompileBuildsCrossSchemaTypedGraphAndKindReferences(t *testing.T) {
	model, diagnostics := Compile(fixtureSchemas(), fixtureDefinitions(), "revision-a")
	if len(diagnostics) != 0 {
		t.Fatalf("Compile diagnostics = %#v", diagnostics)
	}
	if model.Revision != "revision-a" || !strings.HasPrefix(model.Digest, "sha256:") {
		t.Fatalf("model identity/digest = %#v", model)
	}
	if len(model.Edges) != 2 {
		t.Fatalf("edge count = %d, want 2: %#v", len(model.Edges), model.Edges)
	}
	wantPaths := []string{"parts[0]", "next"}
	for i, edge := range model.Edges {
		if edge.Property != wantPaths[i] {
			t.Errorf("edge[%d].Property = %q, want %q", i, edge.Property, wantPaths[i])
		}
		if edge.Source.Path == "" || edge.From == "" || edge.To == "" {
			t.Errorf("edge[%d] lost identity/provenance: %#v", i, edge)
		}
	}
	if _, ok := model.Definition(DefinitionIdentity{APIVersion: appAPI, Kind: "UseCase", Namespace: "orders", Name: "create"}); !ok {
		t.Fatal("compiled Definition lookup failed")
	}
	kind, ok := model.Kind(KindIdentity{APIVersion: domainAPI, Kind: "Aggregate"})
	if !ok || kind.Purpose == "" {
		t.Fatalf("compiled Kind lookup = %#v, %v", kind, ok)
	}
}

func TestCompileIsOrderAndProvenanceIndependentForSemanticDigest(t *testing.T) {
	schemas, definitions := fixtureSchemas(), fixtureDefinitions()
	first, firstDiags := Compile(schemas, definitions, "rev-a")
	if len(firstDiags) != 0 {
		t.Fatal(firstDiags)
	}
	for i, j := 0, len(schemas)-1; i < j; i, j = i+1, j-1 {
		schemas[i], schemas[j] = schemas[j], schemas[i]
	}
	for i, j := 0, len(definitions)-1; i < j; i, j = i+1, j-1 {
		definitions[i], definitions[j] = definitions[j], definitions[i]
	}
	definitions[0].Source = Source{Path: "elsewhere.yaml", Digest: "different", Line: 100}
	second, secondDiags := Compile(schemas, definitions, "rev-b")
	if len(secondDiags) != 0 {
		t.Fatal(secondDiags)
	}
	if first.Digest != second.Digest {
		t.Fatalf("semantic digest changed with order, revision, or provenance: %s != %s", first.Digest, second.Digest)
	}
	definitions[0].Spec["title"] = "Changed title"
	changed, changedDiags := Compile(schemas, definitions, "rev-b")
	if len(changedDiags) != 0 || changed.Digest == first.Digest {
		t.Fatalf("semantic change should produce a different digest: model=%#v diagnostics=%#v", changed, changedDiags)
	}
}

func TestCompileRequiresClosedTypedValuesAndNestedRequiredProperties(t *testing.T) {
	definitions := fixtureDefinitions()
	delete(definitions[1].Spec, "title")
	definitions[1].Spec["unknown"] = "value"
	definitions[1].Spec["settings"] = map[string]any{"other": true}
	_, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	for _, code := range []string{"property.required", "spec.unknown-property"} {
		if !hasDiagnostic(diagnostics, code) {
			t.Errorf("missing %s: %#v", code, diagnostics)
		}
	}
	if !hasPropertyDiagnostic(diagnostics, "settings.mode", "property.required") || !hasPropertyDiagnostic(diagnostics, "settings.other", "spec.unknown-property") {
		t.Fatalf("nested object was not checked closed: %#v", diagnostics)
	}
}

func TestCompileEnforcesCardinalityNullEnumAndDuplicateReferences(t *testing.T) {
	definitions := fixtureDefinitions()
	definitions[1].Spec["title"] = []any{"scalar must remain scalar"}
	definitions[1].Spec["parts"] = []any{
		map[string]any{"namespace": "orders", "name": "order"},
		map[string]any{"namespace": "orders", "name": "order"},
	}
	definitions[1].Spec["settings"] = map[string]any{"mode": "unknown"}
	_, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	for _, code := range []string{"property.type", "reference.duplicate", "property.enum-value"} {
		if !hasDiagnostic(diagnostics, code) {
			t.Errorf("missing %s: %#v", code, diagnostics)
		}
	}
	definitions = fixtureDefinitions()
	definitions[1].Spec["title"] = nil
	_, diagnostics = Compile(fixtureSchemas(), definitions, "rev")
	if !hasDiagnostic(diagnostics, "property.null") {
		t.Fatalf("null should fail: %#v", diagnostics)
	}
}

func TestCompileNormalizesUnsignedNumbersWithoutWeakeningIntegerRange(t *testing.T) {
	definitions := fixtureDefinitions()
	definitions[0].Spec["amount"] = uint64(math.MaxUint64)
	model, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	if len(diagnostics) != 0 {
		t.Fatalf("finite uint64 should be accepted as a number: %#v", diagnostics)
	}
	definition, ok := model.Definition(DefinitionIdentity{APIVersion: domainAPI, Kind: "Aggregate", Namespace: "orders", Name: "order"})
	if !ok {
		t.Fatal("compiled Aggregate lookup failed")
	}
	if got := definition.Spec["amount"]; got != float64(uint64(math.MaxUint64)) {
		t.Fatalf("normalized uint64 number = %#v, want float64 conversion", got)
	}

	schemas := fixtureSchemas()
	kind := schemas[0].Kinds["Aggregate"]
	amount := kind.Properties["amount"]
	amount.Type = TypeInteger
	kind.Properties["amount"] = amount
	schemas[0].Kinds["Aggregate"] = kind
	_, diagnostics = Compile(schemas, definitions, "rev")
	if !hasPropertyDiagnostic(diagnostics, "amount", "property.type") {
		t.Fatalf("integer accepted uint64 outside int64 range: %#v", diagnostics)
	}
}

func TestCompileResolvesExactCrossSchemaTargetsAndKindReferences(t *testing.T) {
	definitions := fixtureDefinitions()
	definitions[1].Spec["parts"] = []any{map[string]any{"namespace": "orders", "name": "missing"}}
	definitions[1].Spec["appliesTo"] = map[string]any{"apiVersion": domainAPI, "kind": "Missing"}
	_, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	for _, code := range []string{"reference.unresolved", "kind-reference.unresolved"} {
		if !hasDiagnostic(diagnostics, code) {
			t.Errorf("missing %s: %#v", code, diagnostics)
		}
	}
	definitions = fixtureDefinitions()
	definitions[1].Spec["parts"] = []any{map[string]any{"apiVersion": appAPI, "namespace": "orders", "name": "order"}}
	_, diagnostics = Compile(fixtureSchemas(), definitions, "rev")
	if !hasDiagnostic(diagnostics, "reference.target-kind") {
		t.Fatalf("mismatched explicit API version should fail: %#v", diagnostics)
	}
	broken := fixtureSchemas()
	property := broken[1].Kinds["UseCase"].Properties["parts"]
	property.Target = &KindIdentity{APIVersion: "missing.example.org/v1", Kind: "Nope"}
	broken[1].Kinds["UseCase"].Properties["parts"] = property
	_, diagnostics = Compile(broken, fixtureDefinitions(), "rev")
	if !hasPropertyDiagnostic(diagnostics, "parts", "property.target-kind") {
		t.Fatalf("schema target must resolve even without a value: %#v", diagnostics)
	}
}

func TestCompileCollectsReferencesInsideRepeatedObjectsWithIndexedPaths(t *testing.T) {
	definitions := fixtureDefinitions()
	definitions[1].Spec["children"] = []any{
		map[string]any{"link": map[string]any{"namespace": "orders", "name": "order"}},
	}
	model, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	if len(diagnostics) != 0 {
		t.Fatalf("Compile diagnostics = %#v", diagnostics)
	}
	var found bool
	for _, edge := range model.Edges {
		if edge.Property == "children[0].link" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing edge with repeated-object index in path: %#v", model.Edges)
	}

	definitions[1].Spec["children"] = []any{
		map[string]any{"link": map[string]any{"namespace": "orders", "name": "missing"}},
	}
	_, diagnostics = Compile(fixtureSchemas(), definitions, "rev")
	if !hasPropertyDiagnostic(diagnostics, "children[0].link", "reference.unresolved") {
		t.Fatalf("unresolved nested reference diagnostic lost repeated-object path: %#v", diagnostics)
	}
}

func TestCompileRejectsUnboundedCardinalityBeyondFiniteValueLimit(t *testing.T) {
	schemas := fixtureSchemas()
	kind := schemas[1].Kinds["UseCase"]
	property := kind.Properties["parts"]
	property.MinCount = MaxValuesPerProperty + 1
	kind.Properties["parts"] = property
	schemas[1].Kinds["UseCase"] = kind
	_, diagnostics := Compile(schemas, fixtureDefinitions(), "rev")
	if !hasPropertyDiagnostic(diagnostics, "parts", "property.count") {
		t.Fatalf("unbounded cardinality with impossible finite minimum was accepted: %#v", diagnostics)
	}
}

func TestCompileIdentityIncludesVersionAndNamespaceAndRejectsDuplicates(t *testing.T) {
	schemas := fixtureSchemas()
	schemas = append(schemas, Schema{APIVersion: "domain.example.org/v2", Purpose: "Second exact version.", Kinds: schemas[0].Kinds})
	definitions := fixtureDefinitions()
	definitions = append(definitions, Definition{APIVersion: "domain.example.org/v2", Kind: "Aggregate", Metadata: Metadata{Namespace: "orders", Name: "order"}, Purpose: "A different version identity.", Spec: map[string]any{"amount": 1}})
	if _, diagnostics := Compile(schemas, definitions, "rev"); len(diagnostics) != 0 {
		t.Fatalf("different API versions must have distinct identities: %#v", diagnostics)
	}
	definitions = append(definitions, definitions[0])
	definitions[len(definitions)-1].Source = Source{Path: "another-module/order.yaml"}
	_, diagnostics := Compile(schemas, definitions, "rev")
	if !hasDiagnostic(diagnostics, "definition.duplicate") {
		t.Fatalf("duplicate cross-module identity must fail: %#v", diagnostics)
	}
	definitions = fixtureDefinitions()
	definitions = append(definitions, Definition{APIVersion: domainAPI, Kind: "Aggregate", Metadata: Metadata{Namespace: "other", Name: "order"}, Purpose: "Separate namespace.", Spec: map[string]any{"amount": 1}})
	if _, diagnostics := Compile(fixtureSchemas(), definitions, "rev"); len(diagnostics) != 0 {
		t.Fatalf("namespace is part of identity: %#v", diagnostics)
	}
}

func TestCompileRejectsAmbiguousSchemaVersionsWithoutSelectingWinner(t *testing.T) {
	schemas := fixtureSchemas()
	duplicate := cloneSchema(schemas[0])
	duplicate.Source = Source{Path: "schemas/alternate.yaml", Digest: "alternate"}
	kind := duplicate.Kinds["Aggregate"]
	amount := kind.Properties["amount"]
	amount.Type = TypeString
	kind.Properties["amount"] = amount
	duplicate.Kinds["Aggregate"] = kind
	schemas = append(schemas, duplicate)
	_, first := Compile(schemas, fixtureDefinitions(), "rev")
	for i, j := 0, len(schemas)-1; i < j; i, j = i+1, j-1 {
		schemas[i], schemas[j] = schemas[j], schemas[i]
	}
	_, second := Compile(schemas, fixtureDefinitions(), "rev")
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("ambiguous schema diagnostics depend on input order:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if !hasDiagnostic(first, "schema.duplicate") || !hasDiagnostic(first, "definition.schema") || hasPropertyDiagnostic(first, "amount", "property.type") {
		t.Fatalf("duplicate apiVersion selected a schema contract despite ambiguity: %#v", first)
	}
}

func TestCompileDoesNotResolveReferencesToDuplicateDefinitionIdentities(t *testing.T) {
	definitions := fixtureDefinitions()
	duplicate := cloneDefinition(definitions[0])
	duplicate.Spec["amount"] = 99.0
	definitions = append(definitions, duplicate)
	_, first := Compile(fixtureSchemas(), definitions, "rev")
	definitions[0], definitions[2] = definitions[2], definitions[0]
	_, second := Compile(fixtureSchemas(), definitions, "rev")
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("duplicate-definition diagnostics depend on input order:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if !hasDiagnostic(first, "definition.duplicate") || !hasPropertyDiagnostic(first, "parts[0]", "reference.unresolved") {
		t.Fatalf("reference bound to one of several duplicate target definitions: %#v", first)
	}
}

func TestCompileRejectsRecursiveDirectGoInputsAndReturnsDefensiveCopies(t *testing.T) {
	definitions := fixtureDefinitions()
	cycle := map[string]any{}
	cycle["self"] = cycle
	definitions[1].Spec["settings"] = cycle
	_, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	if !hasDiagnostic(diagnostics, "definition.limit") {
		t.Fatalf("recursive Go map should fail within a bound: %#v", diagnostics)
	}
	props := map[string]Property{}
	props["self"] = Property{Purpose: "Recursive schema fixture.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: props}
	schemas := []Schema{{APIVersion: "cycle.example.org/v1", Purpose: "Recursive schema.", Kinds: map[string]Kind{"Thing": {Purpose: "Thing.", Properties: props}}}}
	_, diagnostics = Compile(schemas, nil, "rev")
	if !hasDiagnostic(diagnostics, "schema.limit") {
		t.Fatalf("recursive Schema map should fail within a bound: %#v", diagnostics)
	}
	model, diagnostics := Compile(fixtureSchemas(), fixtureDefinitions(), "rev")
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	identity := DefinitionIdentity{APIVersion: appAPI, Kind: "UseCase", Namespace: "orders", Name: "create"}
	found, ok := model.Definition(identity)
	if !ok {
		t.Fatal("Definition not found")
	}
	found.Spec["title"] = "mutated"
	again, _ := model.Definition(identity)
	if again.Spec["title"] != "Create order" {
		t.Fatal("Definition lookup leaked mutable map state")
	}
}

func TestCompileBoundsExpandedSharedGoGraphsBeforeTraversalOrEncoding(t *testing.T) {
	definitions := fixtureDefinitions()
	sharedValue := any("leaf")
	for i := 0; i < 30; i++ {
		sharedValue = map[string]any{"left": sharedValue, "right": sharedValue}
	}
	definitions[1].Spec["unexpected"] = sharedValue
	_, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	if !hasDiagnostic(diagnostics, "model.input-limit") {
		t.Fatalf("expanded shared Definition graph should exceed the bounded preflight: %#v", diagnostics)
	}

	schemas := fixtureSchemas()
	leaf := map[string]Property{"leaf": fixtureProperty(TypeString, "", "", 0, 1)}
	for i := 0; i < 25; i++ {
		leaf = map[string]Property{
			"left":  {Purpose: "Selects the left branch.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: leaf},
			"right": {Purpose: "Selects the right branch.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: leaf},
		}
	}
	kind := schemas[1].Kinds["UseCase"]
	kind.Properties["tree"] = Property{Purpose: "Contains a shared contract graph.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: leaf}
	schemas[1].Kinds["UseCase"] = kind
	_, diagnostics = Compile(schemas, fixtureDefinitions(), "rev")
	if !hasDiagnostic(diagnostics, "model.input-limit") {
		t.Fatalf("expanded shared Schema graph should exceed the bounded preflight: %#v", diagnostics)
	}
}

func TestCompileRejectsUnsupportedValuesWithoutCallingMarshalers(t *testing.T) {
	calls := 0
	definitions := fixtureDefinitions()
	definitions[1].Spec["title"] = hostileJSONValue{calls: &calls}
	_, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
	if calls != 0 || !hasDiagnostic(diagnostics, "definition.limit") {
		t.Fatalf("custom MarshalJSON value was invoked or not rejected: calls=%d diagnostics=%#v", calls, diagnostics)
	}

	type oversizedStruct struct{ Data string }
	definitions = fixtureDefinitions()
	definitions[1].Spec["title"] = oversizedStruct{Data: strings.Repeat("x", MaxDefinitionBytes+1)}
	_, diagnostics = Compile(fixtureSchemas(), definitions, "rev")
	if !hasDiagnostic(diagnostics, "definition.limit") {
		t.Fatalf("unsupported struct was not rejected during preflight: %#v", diagnostics)
	}
}

func TestCompileDiagnosticsAreStableAndOutputIsAtomic(t *testing.T) {
	schemas := fixtureSchemas()
	definitions := fixtureDefinitions()
	definitions[0].Spec["unknown"] = true
	definitions[1].Spec["unknown"] = true
	_, first := Compile(schemas, definitions, "rev")
	for i, j := 0, len(definitions)-1; i < j; i, j = i+1, j-1 {
		definitions[i], definitions[j] = definitions[j], definitions[i]
	}
	_, second := Compile(schemas, definitions, "rev")
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("diagnostics depend on input order:\n%#v\n%#v", first, second)
	}
	model, diagnostics := Compile(schemas, definitions, "rev")
	if len(diagnostics) == 0 || model.Digest != "" {
		t.Fatalf("failed compile must return no partial Model: %#v %#v", model, diagnostics)
	}
}

// BUG-01: a reference Property without a target Kind is a Schema error; a
// Definition that uses it must yield diagnostics, not a nil dereference.
func TestCompileReportsTargetlessReferenceWithoutPanicking(t *testing.T) {
	schemas := []Schema{{APIVersion: domainAPI, Purpose: "Declares a broken reference.", Kinds: map[string]Kind{
		"Thing": {Purpose: "Has a targetless link.", Properties: map[string]Property{
			"link": {Purpose: "Link without target.", Type: TypeReference, MinCount: 0, MaxCount: 1},
		}},
	}}}
	definitions := []Definition{{APIVersion: domainAPI, Kind: "Thing", Metadata: Metadata{Namespace: "x", Name: "a"}, Purpose: "First thing.", Spec: map[string]any{"link": map[string]any{"namespace": "x", "name": "a"}}}}
	var model Model
	var diagnostics []Diagnostic
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("Compile panicked: %v", recovered)
			}
		}()
		model, diagnostics = Compile(schemas, definitions, "targetless")
	}()
	if !hasPropertyDiagnostic(diagnostics, "link", "property.target") || model.Digest != "" {
		t.Fatalf("want property.target and no Model: digest=%q diagnostics=%#v", model.Digest, diagnostics)
	}
}

// BUG-01: the preflight stops at the first value or property that breaks a
// limit. It visits maps in sorted order, so the same input always gives the
// same diagnostics, whether the local limit or the shared budget trips first.
func TestCompilePreflightDiagnosticsDoNotDependOnMapOrder(t *testing.T) {
	sharedValues := func(depth int) any {
		value := any("leaf")
		for i := 0; i < depth; i++ {
			value = map[string]any{"left": value, "right": value}
		}
		return value
	}
	sharedProperties := func(depth int) map[string]Property {
		properties := map[string]Property{"leaf": {Purpose: "Leaf.", Type: TypeString, MinCount: 0, MaxCount: 1}}
		for i := 0; i < depth; i++ {
			properties = map[string]Property{
				"left":  {Purpose: "Left.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: properties},
				"right": {Purpose: "Right.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: properties},
			}
		}
		return properties
	}
	type unsupported struct{}
	for name, compile := range map[string]func() []Diagnostic{
		"definition values": func() []Diagnostic {
			definitions := fixtureDefinitions()
			definitions[1].Spec["bad"] = unsupported{}
			definitions[1].Spec["big"] = sharedValues(20)
			_, diagnostics := Compile(fixtureSchemas(), definitions, "rev")
			return diagnostics
		},
		"schema properties": func() []Diagnostic {
			schemas := fixtureSchemas()
			kind := schemas[1].Kinds["UseCase"]
			kind.Properties["bad"] = Property{Purpose: "Overlong type token.", Type: strings.Repeat("x", 33), MinCount: 0, MaxCount: 1}
			kind.Properties["big"] = Property{Purpose: "Shared tree.", Type: TypeObject, MinCount: 0, MaxCount: 1, Properties: sharedProperties(20)}
			schemas[1].Kinds["UseCase"] = kind
			_, diagnostics := Compile(schemas, fixtureDefinitions(), "rev")
			return diagnostics
		},
	} {
		first := compile()
		for i := 0; i < 64; i++ {
			if again := compile(); !reflect.DeepEqual(again, first) {
				t.Fatalf("%s: run %d gave %+v, first run %+v", name, i, again, first)
			}
		}
	}
}

func hasDiagnostic(values []Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
func hasPropertyDiagnostic(values []Diagnostic, property, code string) bool {
	for _, value := range values {
		if value.Property == property && value.Code == code {
			return true
		}
	}
	return false
}
