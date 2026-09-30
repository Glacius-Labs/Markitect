package format

import (
	"fmt"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
	"markitect/internal/core"
)

const schemaDialect = "https://json-schema.org/draft/2020-12/schema"

// Schemas returns deterministic YAML-encoded JSON Schemas for each Markitect
// resource kind. The schemas describe structural editing aids; Parse and the
// graph validator remain responsible for semantic validation.
func Schemas() (map[string][]byte, error) {
	kinds := []string{"Text", "Rule", "Workflow", "Skill", "Agent", "Contract", "Project"}
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
	return out, nil
}

func resourceSchema(kind string, specFields []string) map[string]any {
	properties := map[string]any{
		"apiVersion": map[string]any{"const": core.APIVersion, "type": "string"},
		"kind":       map[string]any{"const": kind, "type": "string"},
		"metadata":   metadataSchema(kind),
		"spec":       filteredSpecSchema(specFields),
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
	if kind == "Project" {
		delete(properties, "namespace")
	}
	return map[string]any{
		"additionalProperties": false,
		"properties":           properties,
		"required":             []string{"name"},
		"type":                 "object",
	}
}

func filteredSpecSchema(fields []string) map[string]any {
	all := reflectSchema(reflect.TypeOf(core.Spec{})).(map[string]any)
	allProperties := all["properties"].(map[string]any)
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		if shape, ok := allProperties[field]; ok {
			properties[field] = shape
		}
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
	required := make([]string, 0, 1)
	for _, field := range fields {
		if field == "text" || field == "profile" {
			required = append(required, field)
		}
	}
	return map[string]any{
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
		"type":                 "object",
	}
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
