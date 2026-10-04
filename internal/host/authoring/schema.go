package authoring

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

const schemaDialect = "https://json-schema.org/draft/2020-12/schema"

// Schemas returns deterministic YAML-encoded JSON Schemas for each Markitect
// resource kind. The schemas describe structural editing aids; Parse and the
// graph validator remain responsible for semantic validation.
func Schemas() (map[string][]byte, error) {
	return SchemasWithRegistry(NewRegistry())
}

// SchemasWithRegistry includes editor schemas for the active Project's
// registered domain kinds and for the Domain descriptor envelope.
func SchemasWithRegistry(registry *core.Registry) (map[string][]byte, error) {
	if registry == nil {
		registry = NewRegistry()
	}
	kinds := []string{"Text", "Rule", "Workflow", "Skill", "Agent", "Contract", "Project", "Package"}
	out := make(map[string][]byte, len(kinds))
	for _, kind := range kinds {
		fields := AllowedSpecFields(kind)
		if fields == nil {
			return nil, fmt.Errorf("no allowed spec fields for resource kind %q", kind)
		}
		doc := resourceSchema(kind, fields)
		data, err := yaml.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("marshal %s schema: %w", kind, err)
		}
		out["schema/"+kind+".yaml"] = data
	}
	domainDoc := domainResourceSchema()
	domainBytes, err := yaml.Marshal(domainDoc)
	if err != nil {
		return nil, fmt.Errorf("marshal Domain schema: %w", err)
	}
	out["schema/Domain.yaml"] = domainBytes
	for _, domain := range registry.Domains() {
		if domain.APIVersion == core.APIVersion {
			continue
		}
		for _, kind := range sortedKeys(domain.Kinds) {
			doc := genericResourceSchema(domain.APIVersion, kind, domain.Kinds[kind])
			data, err := yaml.Marshal(doc)
			if err != nil {
				return nil, fmt.Errorf("marshal %s/%s schema: %w", domain.APIVersion, kind, err)
			}
			out["schema/domains/"+strings.ReplaceAll(domain.APIVersion, "/", "-")+"-"+kind+".yaml"] = data
		}
	}
	return out, nil
}

func domainResourceSchema() map[string]any {
	propertyRef := map[string]any{"$ref": "#/$defs/Property"}
	propertyDef := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"type":  map[string]any{"type": "string", "enum": []string{"string", "boolean", "integer", "number", "array", "object", "ref"}},
		"items": propertyRef, "properties": map[string]any{"type": "object", "additionalProperties": propertyRef},
		"required": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "uniqueItems": true},
		"enum":     map[string]any{"type": "array", "items": map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "boolean"}, map[string]any{"type": "number"}}}},
		"refKind":  map[string]any{"type": "string"},
	}}
	kindDef := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"required": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "uniqueItems": true}, "inputsField": map[string]any{"type": "string"}, "properties": map[string]any{"type": "object", "additionalProperties": propertyRef}}, "required": []string{"properties"}}
	relationDef := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"description": map[string]any{"type": "string"}, "field": map[string]any{"type": "string"}, "sourceKinds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1, "uniqueItems": true}, "targetKinds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1, "uniqueItems": true}, "minTargets": map[string]any{"type": "integer", "minimum": 0}, "maxTargets": map[string]any{"type": "integer", "minimum": 0}, "context": map[string]any{"type": "boolean"}, "invalidate": map[string]any{"type": "boolean"}, "acyclic": map[string]any{"type": "boolean"}}, "required": []string{"field", "sourceKinds", "targetKinds"}}
	selector := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"kind": map[string]any{"type": "string"}, "labels": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}}}
	assertion := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"op": map[string]any{"type": "string", "enum": []string{"present", "equal", "allowed", "allowed-targets", "count", "unique", "same-target"}}, "scope": map[string]any{"type": "string", "enum": []string{"resource", "selection"}}, "field": map[string]any{"type": "string"}, "relation": map[string]any{"type": "string"}, "value": map[string]any{}, "values": map[string]any{"type": "array", "items": map[string]any{}}, "min": map[string]any{"type": "integer", "minimum": 0}, "max": map[string]any{"type": "integer", "minimum": 0}, "left": map[string]any{"type": "array", "minItems": 1, "maxItems": 2, "items": map[string]any{"type": "string", "minLength": 1}}, "right": map[string]any{"type": "array", "minItems": 1, "maxItems": 2, "items": map[string]any{"type": "string", "minLength": 1}}}, "required": []string{"op"}}
	constraint := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"name": map[string]any{"type": "string"}, "description": map[string]any{"type": "string"}, "select": selector, "assert": assertion}, "required": []string{"name", "select", "assert"}}
	spec := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"apiVersion": map[string]any{"type": "string"}, "kinds": map[string]any{"type": "object", "additionalProperties": kindDef}, "relations": map[string]any{"type": "object", "additionalProperties": relationDef}, "constraints": map[string]any{"type": "array", "items": constraint}}, "required": []string{"apiVersion", "kinds"}}
	properties := map[string]any{"apiVersion": map[string]any{"const": core.APIVersion, "type": "string"}, "kind": map[string]any{"const": "Domain", "type": "string"}, "metadata": metadataSchema("Domain"), "spec": spec}
	return map[string]any{"$schema": schemaDialect, "$defs": map[string]any{"Property": propertyDef}, "title": "Markitect Domain resource", "type": "object", "additionalProperties": false, "required": []string{"apiVersion", "kind", "metadata", "spec"}, "properties": properties}
}

func genericResourceSchema(apiVersion, kind string, definition core.KindDefinition) map[string]any {
	properties := map[string]any{"apiVersion": map[string]any{"const": apiVersion, "type": "string"}, "kind": map[string]any{"const": kind, "type": "string"}, "metadata": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"name": map[string]any{"type": "string"}, "namespace": map[string]any{"type": "string"}, "labels": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}}, "required": []string{"name"}}, "spec": genericObjectSchema(definition.Properties, definition.Required)}
	return map[string]any{"$schema": schemaDialect, "title": "Markitect " + apiVersion + " " + kind + " resource", "type": "object", "additionalProperties": false, "required": []string{"apiVersion", "kind", "metadata", "spec"}, "properties": properties}
}

func genericObjectSchema(properties map[string]core.PropertyDefinition, required []string) map[string]any {
	p := map[string]any{}
	for _, name := range sortedKeys(properties) {
		p[name] = genericPropertySchema(properties[name])
	}
	o := map[string]any{"type": "object", "additionalProperties": false, "properties": p}
	if len(required) > 0 {
		o["required"] = append([]string(nil), required...)
	}
	return o
}
func genericPropertySchema(p core.PropertyDefinition) any {
	switch p.Type {
	case "array":
		return map[string]any{"type": "array", "items": genericPropertySchema(*p.Items)}
	case "object":
		return genericObjectSchema(p.Properties, p.Required)
	case "ref":
		return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"apiVersion": map[string]any{"type": "string"}, "kind": map[string]any{"type": "string"}, "name": map[string]any{"type": "string"}, "namespace": map[string]any{"type": "string"}, "package": map[string]any{"type": "string"}}, "required": []string{"name"}, "x-markitect-refKind": p.RefKind}
	default:
		shape := map[string]any{"type": p.Type}
		if len(p.Enum) > 0 {
			shape["enum"] = append([]any(nil), p.Enum...)
		}
		return shape
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func adapterArgvSchema() map[string]any {
	return map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"type": "string"}, "prefixItems": []any{map[string]any{"type": "string", "minLength": 1, "pattern": CheckExecutablePattern}}}
}

func resourceSchema(kind string, specFields []string) map[string]any {
	properties := map[string]any{
		"apiVersion": map[string]any{"const": core.APIVersion, "type": "string"},
		"kind":       map[string]any{"const": kind, "type": "string"},
		"metadata":   metadataSchema(kind),
		"spec":       filteredSpecSchema(kind, specFields),
	}
	return map[string]any{
		"$schema":              schemaDialect,
		"additionalProperties": false,
		"properties":           properties,
		"required":             []string{"apiVersion", "kind", "metadata", "spec"},
		"title":                "Markitect " + kind + " resource",
		"type":                 "object",
	}
}

func metadataSchema(kind string) map[string]any {
	properties := map[string]any{
		"name":      map[string]any{"type": "string"},
		"namespace": map[string]any{"type": "string"},
	}
	if kind == "Project" || kind == "Package" {
		delete(properties, "namespace")
	}
	return map[string]any{
		"additionalProperties": false,
		"properties":           properties,
		"required":             []string{"name"},
		"type":                 "object",
	}
}

func filteredSpecSchema(kind string, fields []string) map[string]any {
	all := reflectSchema(reflect.TypeOf(Spec{})).(map[string]any)
	allProperties := all["properties"].(map[string]any)
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		if shape, ok := allProperties[field]; ok {
			properties[field] = shape
		}
	}
	if documentationShape, ok := properties["documentation"].(map[string]any); ok {
		documentationShape["properties"].(map[string]any)["roots"].(map[string]any)["minItems"] = 1
	}
	if targetsShape, ok := properties["targets"].(map[string]any); ok {
		targetsShape["items"].(map[string]any)["enum"] = []string{"codex", "claude", "markdown"}
	}
	if checksShape, ok := properties["checks"].(map[string]any); ok {
		checkShape := checksShape["items"].(map[string]any)
		checkProperties := checkShape["properties"].(map[string]any)
		checkProperties["name"].(map[string]any)["pattern"] = CheckNamePattern
		runShape := checkProperties["run"].(map[string]any)
		runShape["minItems"] = 1
		noNUL := map[string]any{"pattern": `\u0000`}
		runShape["items"] = map[string]any{
			"not":  noNUL,
			"type": "string",
		}
		runShape["prefixItems"] = []any{map[string]any{
			"minLength": 1,
			"not":       noNUL,
			"pattern":   CheckExecutablePattern,
			"type":      "string",
		}}
	}
	if policyDate, ok := properties["policyDate"].(map[string]any); ok {
		policyDate["format"] = "date"
	}
	if exceptionsShape, ok := properties["policyExceptions"].(map[string]any); ok {
		exceptionsShape["maxItems"] = 64
		item := exceptionsShape["items"].(map[string]any)
		itemProperties := item["properties"].(map[string]any)
		for _, field := range []string{"constraintDigest", "subjectDigest"} {
			itemProperties[field].(map[string]any)["pattern"] = `^sha256:[0-9a-f]{64}$`
		}
		itemProperties["expiresOn"].(map[string]any)["format"] = "date"
		item["required"] = []string{"name", "apiVersion", "constraint", "subject", "constraintDigest", "subjectDigest", "rationale", "owner", "decision"}
	}
	if providerShape, ok := properties["providers"].(map[string]any); ok {
		providerProperties := providerShape["properties"].(map[string]any)
		for provider, value := range providerProperties {
			shape := value.(map[string]any)
			all := shape["properties"].(map[string]any)
			allowed := set(allowedProviderFields(provider)...)
			for field := range all {
				if _, ok := allowed[field]; !ok {
					delete(all, field)
				}
			}
		}
	}
	if packageShape, ok := properties["packages"].(map[string]any); ok {
		pin := packageShape["items"].(map[string]any)
		pinProperties := pin["properties"].(map[string]any)
		pinProperties["name"].(map[string]any)["pattern"] = dnsLabel.String()
		pinProperties["version"].(map[string]any)["pattern"] = `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`
		pinProperties["source"].(map[string]any)["minLength"] = 1
		pinProperties["archive"].(map[string]any)["pattern"] = `^(?!/)(?!.*(?:^|/)\.\.?/)[^\\:*?\[\]{}\x00]+\.zip$`
		pinProperties["sha256"].(map[string]any)["pattern"] = `^[0-9a-f]{64}$`
	}
	if domainsShape, ok := properties["domains"].(map[string]any); ok {
		domainsShape["uniqueItems"] = true
		domainsShape["minItems"] = 1
		item := domainsShape["items"].(map[string]any)
		item["minLength"] = 1
		if kind == "Project" {
			item["pattern"] = `^(?:package:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?/)?(?!/)(?!.*(?:^|/)\.\.?/)[^\\:\x00]+\.ya?ml$`
		} else {
			item["pattern"] = `^(?!/)(?!.*(?:^|/)\.\.?/)[^\\:\x00]+\.ya?ml$`
		}
	}
	if adaptersShape, ok := properties["adapters"].(map[string]any); ok {
		configProperties := map[string]any{
			"inputs":     map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"type": "string", "minLength": 1, "pattern": `^(?!/)(?!.*(?:^|/)\.\.?/)[^\\:\x00]+$`}},
			"parameters": map[string]any{"type": "object", "additionalProperties": true}, "target": map[string]any{"type": "string", "minLength": 1},
			"observe": adapterArgvSchema(), "plan": adapterArgvSchema(), "verify": adapterArgvSchema(), "apply": adapterArgvSchema(),
			"allowApply": map[string]any{"type": "boolean"}, "timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "maximum": 600}, "outputLimitBytes": map[string]any{"type": "integer", "minimum": 1024, "maximum": 10485760},
		}
		adapterProperties := map[string]any{
			"name": map[string]any{"type": "string", "pattern": dnsLabel.String()}, "type": map[string]any{"type": "string", "enum": []string{"command"}}, "version": map[string]any{"type": "string", "minLength": 1},
			"config": map[string]any{"type": "object", "additionalProperties": false, "properties": configProperties},
		}
		item := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"name", "type", "version"}, "properties": adapterProperties}
		adaptersShape["items"] = item
	}
	if exportsShape, ok := properties["exports"].(map[string]any); ok {
		item := exportsShape["items"].(map[string]any)
		item["required"] = []string{"kind", "namespace", "name"}
		itemProperties := item["properties"].(map[string]any)
		delete(itemProperties, "package")
	}
	required := make([]string, 0, 1)
	for _, field := range fields {
		if field == "text" {
			required = append(required, field)
		}
		if kind == "Package" && (field == "version" || field == "areas" || field == "exports") {
			required = append(required, field)
		}
	}
	result := map[string]any{
		"additionalProperties": false,
		"properties":           properties,
		"type":                 "object",
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

// reflectSchema translates the YAML-tagged model types into JSON Schema shape.
// Required keys follow yaml omitempty tags; kind-specific requirements are
// added by filteredSpecSchema.
func reflectSchema(t reflect.Type) any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		properties := make(map[string]any)
		required := make([]string, 0)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name, options, skip := yamlField(field.Tag.Get("yaml"))
			if skip || field.PkgPath != "" {
				continue
			}
			properties[name] = reflectSchema(field.Type)
			if !options["omitempty"] {
				required = append(required, name)
			}
		}
		result := map[string]any{"additionalProperties": false, "properties": properties, "type": "object"}
		if len(required) > 0 {
			result["required"] = required
		}
		return result
	case reflect.Slice, reflect.Array:
		return map[string]any{"items": reflectSchema(t.Elem()), "type": "array"}
	case reflect.Map:
		return map[string]any{"additionalProperties": reflectSchema(t.Elem()), "type": "object"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	default:
		return map[string]any{}
	}
}

func yamlField(tag string) (string, map[string]bool, bool) {
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "-" {
		return "", nil, true
	}
	options := make(map[string]bool, len(parts)-1)
	for _, option := range parts[1:] {
		options[option] = true
	}
	return name, options, false
}
