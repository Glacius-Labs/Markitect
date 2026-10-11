package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

const MaxTotalInputBytes = 128 << 20
const MaxPreflightNodes = 100_000

type preflightBudget struct {
	nodes, bytes int
	exhausted    bool
}

func (b *preflightBudget) charge(bytes int) bool {
	if b.nodes <= 0 || bytes < 0 || bytes > b.bytes {
		b.exhausted = true
		return false
	}
	b.nodes--
	b.bytes -= bytes
	return true
}

func preflightStringCost(value string) int { return len(value)*6 + 2 }

// Compile validates and normalizes explicitly supplied Schemas and Definitions.
// It performs no source loading, policy evaluation, or I/O. A Model is returned
// only when the entire input compiles successfully; all diagnostics are sorted.
func Compile(schemas []Schema, definitions []Definition, revision string) (Model, []Diagnostic) {
	var diagnostics []Diagnostic
	add := func(d Diagnostic) { diagnostics = append(diagnostics, d) }
	if len(schemas) > MaxSchemas {
		add(Diagnostic{Code: "schema.limit", Message: fmt.Sprintf("schema count %d exceeds limit %d", len(schemas), MaxSchemas)})
	}
	if len(definitions) > MaxDefinitions {
		add(Diagnostic{Code: "definition.limit", Message: fmt.Sprintf("definition count %d exceeds limit %d", len(definitions), MaxDefinitions)})
	}

	if len(schemas) > MaxSchemas || len(definitions) > MaxDefinitions {
		sortDiagnostics(diagnostics)
		return Model{}, diagnostics
	}
	schemaByVersion := make(map[string]Schema, len(schemas))
	kindByIdentity := make(map[string]Kind, len(schemas))
	normalizedSchemas := make([]Schema, 0, len(schemas))
	totalBytes := 0
	preflight := &preflightBudget{nodes: MaxPreflightNodes, bytes: MaxTotalInputBytes}
	apiVersionCounts := make(map[string]int, len(schemas))
	for _, schema := range schemas {
		apiVersionCounts[schema.APIVersion]++
	}
	orderedSchemas := append([]Schema(nil), schemas...)
	sort.Slice(orderedSchemas, func(i, j int) bool {
		if orderedSchemas[i].APIVersion != orderedSchemas[j].APIVersion {
			return orderedSchemas[i].APIVersion < orderedSchemas[j].APIVersion
		}
		if orderedSchemas[i].Source.Path != orderedSchemas[j].Source.Path {
			return orderedSchemas[i].Source.Path < orderedSchemas[j].Source.Path
		}
		return orderedSchemas[i].Source.Digest < orderedSchemas[j].Source.Digest
	})
	for _, schema := range orderedSchemas {
		if !safeSchema(schema, preflight) {
			if preflight.exhausted {
				return Model{}, []Diagnostic{{Code: "model.input-limit", Message: "expanded normalized input exceeds the compile-wide preflight budget"}}
			}
			add(Diagnostic{Code: "schema.limit", Identity: schema.APIVersion, Source: schema.Source, Message: "Schema structure exceeds finite input limits or contains recursive Go values"})
			continue
		}
		if !validAPIVersion(schema.APIVersion) {
			add(Diagnostic{Code: "schema.identity", Identity: schema.APIVersion, Source: schema.Source, Message: "apiVersion must be a valid group/version identity"})
			continue
		}
		ambiguousAPIVersion := apiVersionCounts[schema.APIVersion] > 1
		if ambiguousAPIVersion {
			add(Diagnostic{Code: "schema.duplicate", Identity: schema.APIVersion, Source: schema.Source, Message: "duplicate Schema apiVersion"})
		}
		if !nonemptyText(schema.Purpose) {
			add(Diagnostic{Code: "schema.purpose", Identity: schema.APIVersion, Source: schema.Source, Message: "Schema purpose must be nonempty"})
		}
		copySchema := cloneSchema(schema)
		if copySchema.Kinds == nil {
			copySchema.Kinds = map[string]Kind{}
		}
		for _, kindName := range sortedKindNames(copySchema.Kinds) {
			kind := copySchema.Kinds[kindName]
			kindID := KindIdentity{APIVersion: schema.APIVersion, Kind: kindName}
			if !validIdentifier(kindName) {
				add(Diagnostic{Code: "kind.identity", Identity: kindID.Key(), Source: schema.Source, Message: fmt.Sprintf("Kind name %q is invalid", kindName)})
			}
			if !nonemptyText(kind.Purpose) {
				add(Diagnostic{Code: "kind.purpose", Identity: kindID.Key(), Source: schema.Source, Message: "Kind purpose must be nonempty"})
			}
			if len(kind.Properties) > MaxPropertiesPerObject {
				add(Diagnostic{Code: "property.limit", Identity: kindID.Key(), Source: schema.Source, Message: fmt.Sprintf("property count exceeds limit %d", MaxPropertiesPerObject)})
			}
			for _, propertyName := range sortedPropertyNames(kind.Properties) {
				if !validPropertyName(propertyName) {
					add(Diagnostic{Code: "property.identity", Identity: kindID.Key(), Property: propertyName, Source: schema.Source, Message: fmt.Sprintf("Property name %q is invalid", propertyName)})
				}
				property := kind.Properties[propertyName]
				validatePropertyContract(property, kindID.Key(), propertyName, 0, schema.Source, add)
			}
			if !ambiguousAPIVersion {
				kindByIdentity[kindID.Key()] = kind
			}
		}
		encoded, err := json.Marshal(schemaDigestValue(copySchema))
		if err != nil {
			add(Diagnostic{Code: "schema.encoding", Identity: schema.APIVersion, Source: schema.Source, Message: "Schema contains a value that cannot be normalized"})
		} else if len(encoded) > MaxSchemaBytes {
			add(Diagnostic{Code: "schema.size", Identity: schema.APIVersion, Source: schema.Source, Message: fmt.Sprintf("Schema size exceeds limit %d bytes", MaxSchemaBytes)})
		} else {
			totalBytes += len(encoded) + sourceBytes(schema.Source)
		}
		if !ambiguousAPIVersion {
			schemaByVersion[schema.APIVersion] = copySchema
			normalizedSchemas = append(normalizedSchemas, copySchema)
		}
		if totalBytes > MaxTotalInputBytes {
			add(Diagnostic{Code: "model.size", Message: fmt.Sprintf("combined normalized input size exceeds limit %d bytes", MaxTotalInputBytes)})
			sortDiagnostics(diagnostics)
			return Model{}, diagnostics
		}
	}

	for _, schema := range normalizedSchemas {
		for _, kindName := range sortedKindNames(schema.Kinds) {
			kind := schema.Kinds[kindName]
			for _, propertyName := range sortedPropertyNames(kind.Properties) {
				validateReferenceTargets(kind.Properties[propertyName], KindIdentity{APIVersion: schema.APIVersion, Kind: kindName}.Key(), propertyName, schema.Source, kindByIdentity, add)
			}
		}
	}

	definitionsByIdentity := make(map[string]Definition, len(definitions))
	normalizedDefinitions := make([]Definition, 0, len(definitions))
	orderedDefinitions := append([]Definition(nil), definitions...)
	sort.Slice(orderedDefinitions, func(i, j int) bool {
		left, right := orderedDefinitions[i], orderedDefinitions[j]
		if left.APIVersion != right.APIVersion {
			return left.APIVersion < right.APIVersion
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Metadata.Namespace != right.Metadata.Namespace {
			return left.Metadata.Namespace < right.Metadata.Namespace
		}
		if left.Metadata.Name != right.Metadata.Name {
			return left.Metadata.Name < right.Metadata.Name
		}
		if left.Source.Path != right.Source.Path {
			return left.Source.Path < right.Source.Path
		}
		if left.Source.Line != right.Source.Line {
			return left.Source.Line < right.Source.Line
		}
		if left.Source.Digest != right.Source.Digest {
			return left.Source.Digest < right.Source.Digest
		}
		return false
	})
	definitionIdentityCounts := make(map[string]int, len(orderedDefinitions))
	safeDefinitions := make([]bool, len(orderedDefinitions))
	for i, definition := range orderedDefinitions {
		safeDefinitions[i] = safeDefinition(definition, preflight)
		if preflight.exhausted {
			return Model{}, []Diagnostic{{Code: "model.input-limit", Message: "expanded normalized input exceeds the compile-wide preflight budget"}}
		}
		if safeDefinitions[i] && validateDefinitionIdentity(definition.Identity()) {
			definitionIdentityCounts[definition.Identity().Key()]++
		}
	}
	for definitionIndex, definition := range orderedDefinitions {
		if !safeDefinitions[definitionIndex] {
			add(Diagnostic{Code: "definition.limit", Identity: safeDefinitionIdentityKey(definition), Source: definition.Source, Message: "Definition structure exceeds finite input limits, expanded input budget, or contains recursive Go values"})
			continue
		}
		identity := definition.Identity()
		key := identity.Key()
		identityValid := validateDefinitionIdentity(identity)
		if !identityValid {
			add(Diagnostic{Code: "definition.identity", Identity: key, Source: definition.Source, Message: "Definition identity has an invalid apiVersion, Kind, namespace, or name"})
		}
		if !nonemptyText(definition.Purpose) {
			add(Diagnostic{Code: "definition.purpose", Identity: key, Source: definition.Source, Message: "Definition purpose must be nonempty"})
		}
		schema, schemaExists := schemaByVersion[definition.APIVersion]
		if !schemaExists {
			add(Diagnostic{Code: "definition.schema", Identity: key, Source: definition.Source, Message: fmt.Sprintf("Schema %q is not explicitly available", definition.APIVersion)})
		} else if _, exists := schema.Kinds[definition.Kind]; !exists {
			add(Diagnostic{Code: "definition.kind", Identity: key, Source: definition.Source, Message: fmt.Sprintf("Kind %q is not declared by Schema %q", definition.Kind, definition.APIVersion)})
		}
		if definitionIdentityCounts[key] > 1 {
			add(Diagnostic{Code: "definition.duplicate", Identity: key, Source: definition.Source, Message: "duplicate Definition identity"})
		} else if identityValid {
			definitionsByIdentity[key] = cloneDefinition(definition)
		}
		encoded, err := json.Marshal(definitionDigestValue(definition))
		if err != nil {
			add(Diagnostic{Code: "definition.encoding", Identity: key, Source: definition.Source, Message: "Definition contains a value that cannot be normalized"})
		} else if len(encoded) > MaxDefinitionBytes {
			add(Diagnostic{Code: "definition.size", Identity: key, Source: definition.Source, Message: fmt.Sprintf("Definition size exceeds limit %d bytes", MaxDefinitionBytes)})
		} else {
			totalBytes += len(encoded) + sourceBytes(definition.Source)
		}
		normalizedDefinitions = append(normalizedDefinitions, cloneDefinition(definition))
		if totalBytes > MaxTotalInputBytes {
			add(Diagnostic{Code: "model.size", Message: fmt.Sprintf("combined normalized input size exceeds limit %d bytes", MaxTotalInputBytes)})
			sortDiagnostics(diagnostics)
			return Model{}, diagnostics
		}
	}
	if totalBytes > MaxTotalInputBytes {
		add(Diagnostic{Code: "model.size", Message: fmt.Sprintf("combined normalized input size exceeds limit %d bytes", MaxTotalInputBytes)})
	}

	var edges []Edge
	for index := range normalizedDefinitions {
		definition := &normalizedDefinitions[index]
		identity := definition.Identity()
		schema, schemaExists := schemaByVersion[definition.APIVersion]
		if !schemaExists {
			continue
		}
		kind, kindExists := schema.Kinds[definition.Kind]
		if !kindExists {
			continue
		}
		if definition.Spec == nil {
			definition.Spec = map[string]any{}
		}
		if len(definition.Spec) > MaxPropertiesPerObject {
			add(Diagnostic{Code: "spec.limit", Identity: identity.Key(), Source: definition.Source, Message: fmt.Sprintf("spec property count exceeds limit %d", MaxPropertiesPerObject)})
		}
		for _, field := range sortedAnyMapKeys(definition.Spec) {
			if _, ok := kind.Properties[field]; !ok {
				add(Diagnostic{Code: "spec.unknown-property", Identity: identity.Key(), Property: field, Source: definition.Source, Message: fmt.Sprintf("Kind %q does not declare Property %q", definition.Kind, field)})
			}
		}
		for _, propertyName := range sortedPropertyNames(kind.Properties) {
			property := kind.Properties[propertyName]
			value, present := definition.Spec[propertyName]
			if !present {
				if property.MinCount > 0 {
					add(Diagnostic{Code: "property.required", Identity: identity.Key(), Property: propertyName, Source: definition.Source, Message: fmt.Sprintf("Property %q requires at least %d value(s)", propertyName, property.MinCount)})
				}
				continue
			}
			found, normalized := validatePropertyValue(value, property, identity.Key(), propertyName, definition.Source, definitionsByIdentity, kindByIdentity, add)
			if found {
				definition.Spec[propertyName] = normalized
			}
			if found {
				edges = append(edges, collectEdges(normalized, property, identity, propertyName, definition.Source, definitionsByIdentity)...)
			}
		}
	}

	if len(diagnostics) != 0 {
		sortDiagnostics(diagnostics)
		return Model{}, diagnostics
	}
	sort.Slice(normalizedSchemas, func(i, j int) bool { return normalizedSchemas[i].APIVersion < normalizedSchemas[j].APIVersion })
	sort.Slice(normalizedDefinitions, func(i, j int) bool {
		return normalizedDefinitions[i].Identity().Key() < normalizedDefinitions[j].Identity().Key()
	})
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].Property != edges[j].Property {
			return edges[i].Property < edges[j].Property
		}
		return edges[i].To < edges[j].To
	})
	model := Model{Revision: revision, Schemas: normalizedSchemas, Definitions: normalizedDefinitions, Edges: edges}
	digest, err := modelDigest(model)
	if err != nil {
		return Model{}, []Diagnostic{{Code: "model.digest", Message: "normalized model could not be deterministically encoded"}}
	}
	model.Digest = digest
	return model, nil
}

func safeSchema(schema Schema, budget *preflightBudget) bool {
	if len(schema.Kinds) > MaxPropertiesPerObject || len(schema.Purpose) > MaxSchemaBytes || sourceBytes(schema.Source) > MaxSchemaBytes {
		return false
	}
	if !budget.charge(preflightStringCost(schema.APIVersion) + preflightStringCost(schema.Purpose) + preflightStringCost(schema.Source.Path) + preflightStringCost(schema.Source.Digest)) {
		return false
	}
	// Sorted order makes the first failure, and so the diagnostic, independent of map order.
	for _, name := range sortedKindNames(schema.Kinds) {
		kind := schema.Kinds[name]
		if len(name) > MaxSchemaBytes || len(kind.Purpose) > MaxSchemaBytes || !budget.charge(preflightStringCost(name)+preflightStringCost(kind.Purpose)) || !safePropertyContracts(kind.Properties, 0, budget) {
			return false
		}
	}
	return true
}

func safePropertyContracts(properties map[string]Property, depth int, budget *preflightBudget) bool {
	if depth > MaxObjectDepth || len(properties) > MaxPropertiesPerObject {
		return false
	}
	for _, name := range sortedPropertyNames(properties) {
		property := properties[name]
		if len(name) > MaxSchemaBytes || len(property.Purpose) > MaxSchemaBytes || len(property.Type) > 32 || len(property.Values) > MaxValuesPerProperty {
			return false
		}
		cost := preflightStringCost(name) + preflightStringCost(property.Purpose) + preflightStringCost(property.Type)
		for _, value := range property.Values {
			cost += preflightStringCost(value)
		}
		if property.Target != nil {
			cost += preflightStringCost(property.Target.APIVersion) + preflightStringCost(property.Target.Kind)
		}
		if !budget.charge(cost) {
			return false
		}
		for _, value := range property.Values {
			if len(value) > MaxSchemaBytes {
				return false
			}
		}
		if property.Target != nil && (len(property.Target.APIVersion) > MaxSchemaBytes || len(property.Target.Kind) > MaxSchemaBytes) {
			return false
		}
		if !safePropertyContracts(property.Properties, depth+1, budget) {
			return false
		}
	}
	return true
}

func safeDefinition(definition Definition, budget *preflightBudget) bool {
	if len(definition.APIVersion) > MaxDefinitionBytes || len(definition.Kind) > MaxDefinitionBytes || len(definition.Metadata.Name) > MaxDefinitionBytes || len(definition.Metadata.Namespace) > MaxDefinitionBytes || len(definition.Purpose) > MaxDefinitionBytes || sourceBytes(definition.Source) > MaxDefinitionBytes {
		return false
	}
	cost := preflightStringCost(definition.APIVersion) + preflightStringCost(definition.Kind) + preflightStringCost(definition.Metadata.Name) + preflightStringCost(definition.Metadata.Namespace) + preflightStringCost(definition.Purpose) + preflightStringCost(definition.Source.Path) + preflightStringCost(definition.Source.Digest)
	return budget.charge(cost) && safeValue(reflect.ValueOf(definition.Spec), 0, budget)
}

func safeDefinitionIdentityKey(definition Definition) string {
	identity := definition.Identity()
	if len(identity.APIVersion) > MaxDefinitionBytes || len(identity.Kind) > MaxDefinitionBytes || len(identity.Namespace) > MaxDefinitionBytes || len(identity.Name) > MaxDefinitionBytes {
		return "<oversized-definition-identity>"
	}
	return identity.Key()
}

// safeValue imposes depth and container bounds before cloning or encoding. A
// recursive map/slice from a direct Go caller therefore fails at a fixed depth.
func safeValue(value reflect.Value, depth int, budget *preflightBudget) bool {
	if !value.IsValid() {
		return true
	}
	if depth > MaxObjectDepth*2+8 || !budget.charge(2) {
		return false
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return true
		}
		// An interface only wraps a value; only maps and slices add a level.
		return safeValue(value.Elem(), depth, budget)
	case reflect.Map:
		if value.Type() != reflect.TypeOf(map[string]any{}) {
			return false
		}
		if value.IsNil() {
			return true
		}
		if value.Len() > MaxPropertiesPerObject {
			return false
		}
		if !budget.charge(value.Len()*2 + 2) {
			return false
		}
		// Sorted keys make the first failure, and so the diagnostic, independent of map order.
		object := value.Interface().(map[string]any)
		for _, key := range sortedAnyMapKeys(object) {
			if len(key) > MaxDefinitionBytes || !budget.charge(preflightStringCost(key)) {
				return false
			}
			if !safeValue(value.MapIndex(reflect.ValueOf(key)), depth+1, budget) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		if value.Kind() != reflect.Slice || value.Type() != reflect.TypeOf([]any{}) {
			return false
		}
		if value.Len() > MaxValuesPerProperty {
			return false
		}
		if !budget.charge(value.Len()*2 + 2) {
			return false
		}
		for i := 0; i < value.Len(); i++ {
			if !safeValue(value.Index(i), depth+1, budget) {
				return false
			}
		}
	case reflect.String:
		if value.Type() != reflect.TypeOf("") && value.Type() != reflect.TypeOf(json.Number("")) {
			return false
		}
		if value.Len() > MaxDefinitionBytes || !budget.charge(preflightStringCost(value.String())) {
			return false
		}
	case reflect.Bool:
		return value.Type() == reflect.TypeOf(false) && budget.charge(0)
	case reflect.Int:
		return value.Type() == reflect.TypeOf(int(0)) && budget.charge(0)
	case reflect.Int8:
		return value.Type() == reflect.TypeOf(int8(0)) && budget.charge(0)
	case reflect.Int16:
		return value.Type() == reflect.TypeOf(int16(0)) && budget.charge(0)
	case reflect.Int32:
		return value.Type() == reflect.TypeOf(int32(0)) && budget.charge(0)
	case reflect.Int64:
		return value.Type() == reflect.TypeOf(int64(0)) && budget.charge(0)
	case reflect.Uint:
		return value.Type() == reflect.TypeOf(uint(0)) && budget.charge(0)
	case reflect.Uint8:
		return value.Type() == reflect.TypeOf(uint8(0)) && budget.charge(0)
	case reflect.Uint16:
		return value.Type() == reflect.TypeOf(uint16(0)) && budget.charge(0)
	case reflect.Uint32:
		return value.Type() == reflect.TypeOf(uint32(0)) && budget.charge(0)
	case reflect.Uint64:
		return value.Type() == reflect.TypeOf(uint64(0)) && budget.charge(0)
	case reflect.Float32:
		return value.Type() == reflect.TypeOf(float32(0)) && budget.charge(0)
	case reflect.Float64:
		return value.Type() == reflect.TypeOf(float64(0)) && budget.charge(0)
	default:
		// Reject structs, pointers, named custom types, and all values with
		// behavior that encoding/json could otherwise invoke.
		return false
	}
	return true
}

func sourceBytes(source Source) int { return len(source.Path) + len(source.Digest) + 24 }

func validateReferenceTargets(property Property, identity, path string, source Source, kinds map[string]Kind, add func(Diagnostic)) {
	if property.Type == TypeReference && property.Target != nil {
		if _, exists := kinds[property.Target.Key()]; !exists {
			add(Diagnostic{Code: "property.target-kind", Identity: identity, Property: path, Source: source, Message: fmt.Sprintf("reference target Kind %s is not explicitly available", property.Target.Key())})
		}
	}
	if property.Type == TypeObject {
		for _, name := range sortedPropertyNames(property.Properties) {
			validateReferenceTargets(property.Properties[name], identity, path+"."+name, source, kinds, add)
		}
	}
}

func validatePropertyContract(p Property, identity, path string, depth int, source Source, add func(Diagnostic)) {
	if !nonemptyText(p.Purpose) {
		add(Diagnostic{Code: "property.purpose", Identity: identity, Property: path, Source: source, Message: "Property purpose must be nonempty"})
	}
	if depth > MaxObjectDepth {
		add(Diagnostic{Code: "property.depth", Identity: identity, Property: path, Source: source, Message: fmt.Sprintf("object contract depth exceeds limit %d", MaxObjectDepth)})
		return
	}
	if p.MinCount < 0 || p.MinCount > MaxValuesPerProperty || p.MaxCount < Unbounded || p.MaxCount > MaxValuesPerProperty || p.MaxCount != Unbounded && p.MinCount > p.MaxCount {
		add(Diagnostic{Code: "property.count", Identity: identity, Property: path, Source: source, Message: "minCount and maxCount must be nonnegative and ordered, or maxCount must be unbounded"})
	}
	switch p.Type {
	case TypeString, TypeBoolean, TypeInteger, TypeNumber:
		if p.Target != nil || len(p.Values) > 0 || p.Properties != nil {
			add(Diagnostic{Code: "property.contract", Identity: identity, Property: path, Source: source, Message: "scalar Property cannot declare target, values, or nested properties"})
		}
	case TypeEnum:
		if len(p.Values) == 0 || p.Target != nil || p.Properties != nil {
			add(Diagnostic{Code: "property.enum", Identity: identity, Property: path, Source: source, Message: "enum Property requires values and cannot declare target or nested properties"})
		}
		seen := map[string]bool{}
		for _, value := range p.Values {
			if !nonemptyText(value) || seen[value] {
				add(Diagnostic{Code: "property.enum", Identity: identity, Property: path, Source: source, Message: "enum values must be nonempty and unique"})
				break
			}
			seen[value] = true
		}
	case TypeObject:
		if p.Target != nil || len(p.Values) > 0 {
			add(Diagnostic{Code: "property.contract", Identity: identity, Property: path, Source: source, Message: "object Property cannot declare target or enum values"})
		}
		if len(p.Properties) > MaxPropertiesPerObject {
			add(Diagnostic{Code: "property.limit", Identity: identity, Property: path, Source: source, Message: fmt.Sprintf("nested property count exceeds limit %d", MaxPropertiesPerObject)})
		}
		for _, name := range sortedPropertyNames(p.Properties) {
			if !validPropertyName(name) {
				add(Diagnostic{Code: "property.identity", Identity: identity, Property: path + "." + name, Source: source, Message: fmt.Sprintf("Property name %q is invalid", name)})
			}
			validatePropertyContract(p.Properties[name], identity, path+"."+name, depth+1, source, add)
		}
	case TypeReference:
		if p.Target == nil || !validAPIVersion(p.Target.APIVersion) || !validIdentifier(p.Target.Kind) || len(p.Values) > 0 || p.Properties != nil {
			add(Diagnostic{Code: "property.target", Identity: identity, Property: path, Source: source, Message: "reference Property requires one exact target Kind and no enum or nested properties"})
		}
	case TypeKindReference:
		if p.Target != nil || len(p.Values) > 0 || p.Properties != nil {
			add(Diagnostic{Code: "property.contract", Identity: identity, Property: path, Source: source, Message: "kindReference Property cannot declare a target, enum values, or nested properties"})
		}
	default:
		add(Diagnostic{Code: "property.type", Identity: identity, Property: path, Source: source, Message: fmt.Sprintf("unknown Property type %q", p.Type)})
	}
}

func validatePropertyValue(value any, p Property, identity, path string, source Source, definitions map[string]Definition, kinds map[string]Kind, add func(Diagnostic)) (bool, any) {
	// Only an untyped nil is null. A typed nil list or map from a direct Go caller is
	// taken as empty on purpose; decoded YAML never produces one.
	if value == nil {
		add(Diagnostic{Code: "property.null", Identity: identity, Property: path, Source: source, Message: "null is not a valid Property value; omit an optional Property instead"})
		return false, value
	}
	if p.MaxCount != 1 {
		values, ok := asValues(value)
		if !ok {
			add(Diagnostic{Code: "property.cardinality", Identity: identity, Property: path, Source: source, Message: "Property with maxCount other than one must be a list"})
			return false, value
		}
		if len(values) > MaxValuesPerProperty || p.MaxCount != Unbounded && len(values) > p.MaxCount || len(values) < p.MinCount {
			add(Diagnostic{Code: "property.cardinality", Identity: identity, Property: path, Source: source, Message: fmt.Sprintf("list count %d is outside declared bounds", len(values))})
			return false, value
		}
		normalized := make([]any, len(values))
		seenTargets := map[string]bool{}
		for i, item := range values {
			valid, normalizedItem := validateSingleValue(item, p, identity, fmt.Sprintf("%s[%d]", path, i), source, definitions, kinds, seenTargets, add)
			if !valid {
				return false, value
			}
			normalized[i] = normalizedItem
		}
		return true, normalized
	}
	valid, normalized := validateSingleValue(value, p, identity, path, source, definitions, kinds, map[string]bool{}, add)
	if !valid {
		return false, value
	}
	return true, normalized
}

func validateSingleValue(value any, p Property, identity, path string, source Source, definitions map[string]Definition, kinds map[string]Kind, seenTargets map[string]bool, add func(Diagnostic)) (bool, any) {
	bad := func(code, message string) (bool, any) {
		add(Diagnostic{Code: code, Identity: identity, Property: path, Source: source, Message: message})
		return false, value
	}
	switch p.Type {
	case TypeString:
		v, ok := value.(string)
		if !ok {
			return bad("property.type", "expected string")
		}
		return true, v
	case TypeBoolean:
		v, ok := value.(bool)
		if !ok {
			return bad("property.type", "expected boolean")
		}
		return true, v
	case TypeInteger:
		v, ok := integerValue(value)
		if !ok {
			return bad("property.type", "expected integer")
		}
		return true, v
	case TypeNumber:
		v, ok := numberValue(value)
		if !ok {
			return bad("property.type", "expected finite number")
		}
		return true, v
	case TypeEnum:
		v, ok := value.(string)
		if !ok || !containsString(p.Values, v) {
			return bad("property.enum-value", "value is not one of the declared enum strings")
		}
		return true, v
	case TypeObject:
		object, ok := value.(map[string]any)
		if !ok {
			return bad("property.type", "expected object")
		}
		if len(object) > MaxPropertiesPerObject {
			return bad("property.limit", fmt.Sprintf("object property count exceeds limit %d", MaxPropertiesPerObject))
		}
		out := make(map[string]any, len(object))
		for _, name := range sortedAnyMapKeys(object) {
			child, exists := p.Properties[name]
			if !exists {
				add(Diagnostic{Code: "spec.unknown-property", Identity: identity, Property: path + "." + name, Source: source, Message: fmt.Sprintf("object does not declare Property %q", name)})
				continue
			}
			ok, childValue := validatePropertyValue(object[name], child, identity, path+"."+name, source, definitions, kinds, add)
			if ok {
				out[name] = childValue
			}
		}
		for _, name := range sortedPropertyNames(p.Properties) {
			if _, exists := object[name]; !exists && p.Properties[name].MinCount > 0 {
				add(Diagnostic{Code: "property.required", Identity: identity, Property: path + "." + name, Source: source, Message: fmt.Sprintf("Property %q requires at least %d value(s)", name, p.Properties[name].MinCount)})
			}
		}
		return true, out
	case TypeReference:
		ref, ok := value.(map[string]any)
		if !ok {
			return bad("reference.value", "reference value must be an object")
		}
		allowed := map[string]bool{"apiVersion": true, "kind": true, "namespace": true, "name": true}
		for _, key := range sortedAnyMapKeys(ref) {
			if !allowed[key] {
				return bad("reference.value", fmt.Sprintf("reference contains unknown field %q", key))
			}
		}
		namespaceValue, hasNamespace := ref["namespace"]
		nameValue, hasName := ref["name"]
		namespace, nsOK := namespaceValue.(string)
		name, nameOK := nameValue.(string)
		if !hasNamespace || !hasName || !nsOK || !nameOK || !validNamespace(namespace) || !validIdentifier(name) {
			return bad("reference.value", "reference requires an explicit namespace (empty is valid) and a valid name")
		}
		if p.Target == nil {
			// The Schema check reports the missing target Kind as property.target.
			return bad("reference.target-kind", "reference Property declares no target Kind")
		}
		target := *p.Target
		if apiValue, exists := ref["apiVersion"]; exists {
			api, ok := apiValue.(string)
			if !ok || api != target.APIVersion {
				return bad("reference.target-kind", "reference apiVersion does not match the declared target Kind")
			}
		}
		if kindValue, exists := ref["kind"]; exists {
			kind, ok := kindValue.(string)
			if !ok || kind != target.Kind {
				return bad("reference.target-kind", "reference kind does not match the declared target Kind")
			}
		}
		targetIdentity := DefinitionIdentity{APIVersion: target.APIVersion, Kind: target.Kind, Namespace: namespace, Name: name}
		targetKey := targetIdentity.Key()
		if _, exists := kinds[target.Key()]; !exists {
			return bad("reference.target-kind", "declared target Kind is not in the explicitly supplied Schemas")
		}
		if _, exists := definitions[targetKey]; !exists {
			return bad("reference.unresolved", fmt.Sprintf("reference target %s is not defined", targetKey))
		}
		if seenTargets[targetKey] {
			return bad("reference.duplicate", "a repeated reference Property cannot contain duplicate targets")
		}
		seenTargets[targetKey] = true
		return true, map[string]any{"apiVersion": target.APIVersion, "kind": target.Kind, "namespace": namespace, "name": name}
	case TypeKindReference:
		ref, ok := value.(map[string]any)
		if !ok {
			return bad("kind-reference.value", "kindReference value must be an object")
		}
		allowed := map[string]bool{"apiVersion": true, "kind": true}
		for _, key := range sortedAnyMapKeys(ref) {
			if !allowed[key] {
				return bad("kind-reference.value", fmt.Sprintf("kindReference contains unknown field %q", key))
			}
		}
		api, apiOK := ref["apiVersion"].(string)
		kind, kindOK := ref["kind"].(string)
		if !apiOK || !kindOK || !validAPIVersion(api) || !validIdentifier(kind) {
			return bad("kind-reference.value", "kindReference requires apiVersion and kind")
		}
		kindIdentity := KindIdentity{APIVersion: api, Kind: kind}
		if _, exists := kinds[kindIdentity.Key()]; !exists {
			return bad("kind-reference.unresolved", fmt.Sprintf("Kind %s is not explicitly available", kindIdentity.Key()))
		}
		return true, map[string]any{"apiVersion": api, "kind": kind}
	default:
		return bad("property.type", fmt.Sprintf("unknown Property type %q", p.Type))
	}
}

func collectEdges(value any, p Property, sourceIdentity DefinitionIdentity, path string, source Source, definitions map[string]Definition) []Edge {
	if p.MaxCount != 1 {
		values, ok := asValues(value)
		if !ok {
			return nil
		}
		var result []Edge
		for i, item := range values {
			result = append(result, collectSingleEdge(item, p, sourceIdentity, fmt.Sprintf("%s[%d]", path, i), source, definitions)...)
		}
		return result
	}
	return collectSingleEdge(value, p, sourceIdentity, path, source, definitions)
}

func collectSingleEdge(value any, p Property, sourceIdentity DefinitionIdentity, path string, source Source, definitions map[string]Definition) []Edge {
	if p.Type == TypeObject {
		object, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		var result []Edge
		for _, name := range sortedPropertyNames(p.Properties) {
			childValue, exists := object[name]
			if exists {
				result = append(result, collectEdges(childValue, p.Properties[name], sourceIdentity, path+"."+name, source, definitions)...)
			}
		}
		return result
	}
	if p.Type != TypeReference {
		return nil
	}
	ref, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	identity := DefinitionIdentity{APIVersion: fmt.Sprint(ref["apiVersion"]), Kind: fmt.Sprint(ref["kind"]), Namespace: fmt.Sprint(ref["namespace"]), Name: fmt.Sprint(ref["name"])}
	if _, exists := definitions[identity.Key()]; !exists {
		return nil
	}
	return []Edge{{From: sourceIdentity.Key(), To: identity.Key(), Property: path, Source: source}}
}

func modelDigest(model Model) (string, error) {
	type schemaValue struct {
		APIVersion, Purpose string
		Kinds               map[string]Kind
	}
	type definitionValue struct {
		APIVersion, Kind string
		Metadata         Metadata
		Purpose          string
		Spec             map[string]any
	}
	payload := struct {
		Schemas     []schemaValue
		Definitions []definitionValue
	}{}
	for _, schema := range model.Schemas {
		payload.Schemas = append(payload.Schemas, schemaValue{schema.APIVersion, schema.Purpose, schema.Kinds})
	}
	for _, definition := range model.Definitions {
		payload.Definitions = append(payload.Definitions, definitionValue{definition.APIVersion, definition.Kind, definition.Metadata, definition.Purpose, definition.Spec})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func schemaDigestValue(schema Schema) any {
	return struct {
		APIVersion, Purpose string
		Kinds               map[string]Kind
	}{schema.APIVersion, schema.Purpose, schema.Kinds}
}
func definitionDigestValue(d Definition) any {
	return struct {
		APIVersion, Kind string
		Metadata         Metadata
		Purpose          string
		Spec             map[string]any
	}{d.APIVersion, d.Kind, d.Metadata, d.Purpose, d.Spec}
}
func cloneSchema(s Schema) Schema {
	out := s
	out.Kinds = make(map[string]Kind, len(s.Kinds))
	for key, value := range s.Kinds {
		out.Kinds[key] = cloneKind(value)
	}
	return out
}
func cloneKind(k Kind) Kind {
	out := k
	out.Properties = make(map[string]Property, len(k.Properties))
	for key, value := range k.Properties {
		out.Properties[key] = cloneProperty(value)
	}
	return out
}
func cloneProperty(p Property) Property {
	out := p
	out.Values = append([]string(nil), p.Values...)
	if p.Target != nil {
		target := *p.Target
		out.Target = &target
	}
	if p.Properties != nil {
		out.Properties = make(map[string]Property, len(p.Properties))
		for key, value := range p.Properties {
			out.Properties[key] = cloneProperty(value)
		}
	}
	return out
}
func cloneDefinition(d Definition) Definition { out := d; out.Spec = cloneObject(d.Spec); return out }
func cloneObject(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for key, value := range m {
		out[key] = cloneValue(value)
	}
	return out
}
func cloneValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneObject(v)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = cloneValue(v[i])
		}
		return out
	default:
		return v
	}
}

func asValues(value any) ([]any, bool) {
	if values, ok := value.([]any); ok {
		return values, true
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array || rv.Type().Elem().Kind() == reflect.Uint8 {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}
func integerValue(value any) (int64, bool) {
	switch v := value.(type) {
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		if uint64(v) <= math.MaxInt64 {
			return int64(v), true
		}
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v <= math.MaxInt64 {
			return int64(v), true
		}
	}
	return 0, false
}
func numberValue(value any) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		n, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return n, !math.IsNaN(n) && !math.IsInf(n, 0)
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
	case float64:
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	}
	return 0, false
}
func validAPIVersion(value string) bool {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || len(parts[0]) > 253 || len(parts[1]) > 63 {
		return false
	}
	for _, r := range parts[0] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-') {
			return false
		}
	}
	if strings.HasPrefix(parts[0], ".") || strings.HasSuffix(parts[0], ".") {
		return false
	}
	for i, r := range parts[1] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') || i == 0 && !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}
func validIdentifier(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	for i, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) || !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.') || i == 0 && !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}
func validPropertyName(value string) bool { return validIdentifier(value) }
func validNamespace(value string) bool    { return value == "" || validIdentifier(value) }
func validateDefinitionIdentity(i DefinitionIdentity) bool {
	return validAPIVersion(i.APIVersion) && validIdentifier(i.Kind) && validNamespace(i.Namespace) && validIdentifier(i.Name)
}
func nonemptyText(value string) bool { return strings.TrimSpace(value) != "" }
func sortedKindNames(values map[string]Kind) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func sortedPropertyNames(values map[string]Property) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func sortedAnyMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func sortDiagnostics(values []Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Identity != b.Identity {
			return a.Identity < b.Identity
		}
		if a.Property != b.Property {
			return a.Property < b.Property
		}
		if a.Source.Path != b.Source.Path {
			return a.Source.Path < b.Source.Path
		}
		if a.Source.Line != b.Source.Line {
			return a.Source.Line < b.Source.Line
		}
		if a.Source.Digest != b.Source.Digest {
			return a.Source.Digest < b.Source.Digest
		}
		return a.Message < b.Message
	})
}
