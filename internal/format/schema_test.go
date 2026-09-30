package format

import (
	"bytes"
	"reflect"
	"sort"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

func TestSchemasAreDeterministicAndCoverEveryKind(t *testing.T) {
	first, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"schema/Agent.yaml", "schema/Contract.yaml", "schema/Project.yaml",
		"schema/Rule.yaml", "schema/Skill.yaml", "schema/Text.yaml", "schema/Workflow.yaml",
	}
	got := make([]string, 0, len(first))
	for path := range first {
		got = append(got, path)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema paths = %v, want %v", got, want)
	}
	for path, data := range first {
		if !bytes.Equal(data, second[path]) {
			t.Errorf("%s output changed between calls", path)
		}
	}
}

func TestSchemasMatchAllowedSpecFieldsAndKeepObjectsStrict(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"Text", "Rule", "Workflow", "Skill", "Agent", "Contract", "Project"} {
		t.Run(kind, func(t *testing.T) {
			var document map[string]any
			if err := yaml.Unmarshal(schemas["schema/"+kind+".yaml"], &document); err != nil {
				t.Fatal(err)
			}
			if document["$schema"] != schemaDialect {
				t.Errorf("$schema = %v, want %s", document["$schema"], schemaDialect)
			}
			if document["additionalProperties"] != false {
				t.Errorf("root additionalProperties = %v, want false", document["additionalProperties"])
			}
			rootProperties := mapping(t, document["properties"])
			if mapping(t, rootProperties["kind"])["const"] != kind {
				t.Errorf("kind const = %v, want %s", mapping(t, rootProperties["kind"])["const"], kind)
			}
			metadata := mapping(t, rootProperties["metadata"])
			if metadata["additionalProperties"] != false {
				t.Errorf("metadata additionalProperties = %v, want false", metadata["additionalProperties"])
			}
			if !containsString(sequence(t, metadata["required"]), "name") {
				t.Error("metadata.name is not required")
			}
			metadataProperties := mapping(t, metadata["properties"])
			if kind == "Project" {
				if _, ok := metadataProperties["namespace"]; ok {
					t.Error("Project metadata unexpectedly allows namespace")
				}
			} else if _, ok := metadataProperties["namespace"]; !ok {
				t.Error("metadata.namespace is missing")
			}

			spec := mapping(t, rootProperties["spec"])
			if spec["additionalProperties"] != false {
				t.Errorf("spec additionalProperties = %v, want false", spec["additionalProperties"])
			}
			properties := mapping(t, spec["properties"])
			gotFields := make([]string, 0, len(properties))
			for field := range properties {
				gotFields = append(gotFields, field)
			}
			sort.Strings(gotFields)
			wantFields := append([]string(nil), AllowedSpecFields(kind)...)
			sort.Strings(wantFields)
			if !reflect.DeepEqual(gotFields, wantFields) {
				t.Errorf("spec properties = %v, want parser fields %v", gotFields, wantFields)
			}
			if kind == "Project" {
				if required, ok := spec["required"]; ok && containsString(sequence(t, required), "checks") {
					t.Error("Project spec.checks should be optional")
				}
				if _, ok := properties["profile"]; ok {
					t.Error("removed Project spec.profile is still in the schema")
				}
			} else if !containsString(sequence(t, spec["required"]), "text") {
				t.Errorf("%s spec.text is not required", kind)
			}
		})
	}
}

func TestProjectCheckSchemaConstraints(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(schemas["schema/Project.yaml"], &document); err != nil {
		t.Fatal(err)
	}
	spec := mapping(t, mapping(t, document["properties"])["spec"])
	properties := mapping(t, spec["properties"])
	checks := mapping(t, properties["checks"])
	if checks["type"] != "array" {
		t.Fatalf("checks type = %v, want array", checks["type"])
	}
	if _, ok := spec["required"]; ok {
		t.Errorf("Project fields should remain optional, required = %v", spec["required"])
	}
	item := mapping(t, checks["items"])
	if item["type"] != "object" || item["additionalProperties"] != false {
		t.Fatalf("check items must be strict objects: %#v", item)
	}
	if !containsString(sequence(t, item["required"]), "name") || !containsString(sequence(t, item["required"]), "run") {
		t.Fatalf("check name and run must be required: %#v", item["required"])
	}
	checkProperties := mapping(t, item["properties"])
	if mapping(t, checkProperties["name"])["pattern"] != core.CheckNamePattern {
		t.Errorf("check name pattern = %v, want %s", mapping(t, checkProperties["name"])["pattern"], core.CheckNamePattern)
	}
	run := mapping(t, checkProperties["run"])
	if run["minItems"] != 1 {
		t.Errorf("run minItems = %v, want 1", run["minItems"])
	}
	firstArg := sequence(t, run["prefixItems"])[0]
	if mapping(t, firstArg)["pattern"] != core.CheckExecutablePattern {
		t.Errorf("first argv pattern = %v, want %s", mapping(t, firstArg)["pattern"], core.CheckExecutablePattern)
	}
	if mapping(t, run["items"])["type"] != "string" || mapping(t, run["items"])["not"] == nil {
		t.Errorf("additional argv items must be strings without NUL: %#v", run["items"])
	}
	if _, ok := properties["profile"]; ok {
		t.Error("Project schema exposes removed spec.profile")
	}
}

func TestSchemaShapesComeFromYAMLTaggedModel(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(schemas["schema/Agent.yaml"], &document); err != nil {
		t.Fatal(err)
	}
	spec := mapping(t, mapping(t, document["properties"])["spec"])
	properties := mapping(t, spec["properties"])
	if got := mapping(t, properties["files"])["type"]; got != "array" {
		t.Errorf("files type = %v, want array", got)
	}
	files := mapping(t, properties["files"])
	if got := mapping(t, files["items"])["type"]; got != "string" {
		t.Errorf("files item type = %v, want string", got)
	}
	if got := mapping(t, properties["providers"])["type"]; got != "object" {
		t.Errorf("providers type = %v, want object", got)
	}
	if mapping(t, properties["providers"])["additionalProperties"] != false {
		t.Error("providers schema is not strict")
	}
	if mapping(t, mapping(t, properties["providers"])["properties"])["claude"] == nil {
		t.Error("providers.claude shape is missing")
	}
}

func TestProviderSchemaPropertiesMatchProviderAllowLists(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(schemas["schema/Agent.yaml"], &document); err != nil {
		t.Fatal(err)
	}
	spec := mapping(t, mapping(t, document["properties"])["spec"])
	providers := mapping(t, mapping(t, spec["properties"])["providers"])
	providerProperties := mapping(t, providers["properties"])
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			shape := mapping(t, providerProperties[provider])
			got := make([]string, 0, len(mapping(t, shape["properties"])))
			for name := range mapping(t, shape["properties"]) {
				got = append(got, name)
			}
			sort.Strings(got)
			want := append([]string(nil), allowedProviderFields(provider)...)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s schema properties = %v, want allowed fields %v", provider, got, want)
			}
			if shape["additionalProperties"] != false {
				t.Errorf("%s additionalProperties = %v, want false", provider, shape["additionalProperties"])
			}
		})
	}
}

func mapping(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value %T = %v, want mapping", value, value)
	}
	return result
}

func sequence(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("value %T = %v, want sequence", value, value)
	}
	return result
}

func containsString(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
