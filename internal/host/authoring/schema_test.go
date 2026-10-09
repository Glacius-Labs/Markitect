package authoring

import (
	"bytes"
	"reflect"
	"sort"
	"testing"

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
		"schema/Agent.yaml", "schema/Contract.yaml", "schema/Domain.yaml", "schema/Package.yaml", "schema/Project.yaml",
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

func TestDomainSchemaDescribesSameTargetPaths(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(schemas["schema/Domain.yaml"], &document); err != nil {
		t.Fatal(err)
	}
	root := mapping(t, document["properties"])
	spec := mapping(t, root["spec"])
	constraints := mapping(t, spec["properties"])["constraints"]
	constraint := mapping(t, mapping(t, constraints)["items"])
	assertion := mapping(t, mapping(t, constraint["properties"])["assert"])
	assertionProperties := mapping(t, assertion["properties"])
	if !containsString(sequence(t, mapping(t, assertionProperties["op"])["enum"]), "same-target") {
		t.Fatal("Domain schema does not list same-target")
	}
	for _, side := range []string{"left", "right"} {
		path := mapping(t, assertionProperties[side])
		if path["minItems"] != 1 || path["maxItems"] != 2 {
			t.Errorf("%s path bounds = %v..%v, want one or two relations", side, path["minItems"], path["maxItems"])
		}
	}
}

func TestSchemasMatchAllowedSpecFieldsAndKeepObjectsStrict(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"Text", "Rule", "Workflow", "Skill", "Agent", "Contract", "Project", "Package"} {
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
			if kind == "Project" || kind == "Package" {
				if _, ok := metadataProperties["namespace"]; ok {
					t.Errorf("%s metadata unexpectedly allows namespace", kind)
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
			} else if kind == "Package" {
				for _, field := range []string{"version", "areas", "exports"} {
					if !containsString(sequence(t, spec["required"]), field) {
						t.Errorf("Package spec.%s is not required", field)
					}
				}
			} else if !containsString(sequence(t, spec["required"]), "text") {
				t.Errorf("%s spec.text is not required", kind)
			}
		})
	}
}

func TestPackageSchemaPinsAndExportsStayStructurallyQualified(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	var projectDocument, packageDocument map[string]any
	if err := yaml.Unmarshal(schemas["schema/Project.yaml"], &projectDocument); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(schemas["schema/Package.yaml"], &packageDocument); err != nil {
		t.Fatal(err)
	}
	projectSpec := mapping(t, mapping(t, projectDocument["properties"])["spec"])
	pins := mapping(t, mapping(t, mapping(t, projectSpec["properties"])["packages"])["items"])
	for _, field := range []string{"name", "version", "source", "archive", "sha256"} {
		if !containsString(sequence(t, pins["required"]), field) {
			t.Errorf("package pin %s is not required", field)
		}
	}
	if pins["additionalProperties"] != false {
		t.Errorf("package pin must reject unknown fields: %#v", pins)
	}
	packageSpec := mapping(t, mapping(t, packageDocument["properties"])["spec"])
	exports := mapping(t, mapping(t, mapping(t, packageSpec["properties"])["exports"])["items"])
	if _, ok := mapping(t, exports["properties"])["package"]; ok {
		t.Error("package exports must not reference another package")
	}
	if !reflect.DeepEqual(sequence(t, exports["required"]), []any{"kind", "namespace", "name"}) {
		t.Errorf("package exports are not fully qualified: %#v", exports["required"])
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
	timeout := mapping(t, checkProperties["timeoutSeconds"])
	if timeout["type"] != "integer" || timeout["minimum"] != 1 || timeout["maximum"] != 3600 || timeout["default"] != 600 || containsString(sequence(t, item["required"]), "timeoutSeconds") {
		t.Fatalf("optional timeout bounds/default are incorrect: %#v", timeout)
	}
	if mapping(t, checkProperties["name"])["pattern"] != CheckNamePattern {
		t.Errorf("check name pattern = %v, want %s", mapping(t, checkProperties["name"])["pattern"], CheckNamePattern)
	}
	run := mapping(t, checkProperties["run"])
	if run["minItems"] != 1 {
		t.Errorf("run minItems = %v, want 1", run["minItems"])
	}
	firstArg := sequence(t, run["prefixItems"])[0]
	if mapping(t, firstArg)["pattern"] != CheckExecutablePattern {
		t.Errorf("first argv pattern = %v, want %s", mapping(t, firstArg)["pattern"], CheckExecutablePattern)
	}
	if mapping(t, run["items"])["type"] != "string" || mapping(t, run["items"])["not"] == nil {
		t.Errorf("additional argv items must be strings without NUL: %#v", run["items"])
	}
	if _, ok := properties["profile"]; ok {
		t.Error("Project schema exposes removed spec.profile")
	}
}

func TestProjectLocalProjectionSchemaIsVersionedAndClosed(t *testing.T) {
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(schemas["schema/Project.yaml"], &document); err != nil {
		t.Fatal(err)
	}
	spec := mapping(t, mapping(t, document["properties"])["spec"])
	adapters := mapping(t, mapping(t, spec["properties"])["adapters"])
	item := mapping(t, adapters["items"])
	if item["additionalProperties"] != false {
		t.Fatal("adapter item schema must reject unknown fields")
	}
	branches := sequence(t, item["oneOf"])
	if len(branches) != 2 {
		t.Fatalf("adapter oneOf branches = %d, want command and local-projection", len(branches))
	}
	var projectionBranch map[string]any
	for _, raw := range branches {
		branch := mapping(t, raw)
		branchProperties := mapping(t, branch["properties"])
		if mapping(t, branchProperties["type"])["const"] == "local-projection" {
			projectionBranch = branch
		}
	}
	if projectionBranch == nil {
		t.Fatal("schema has no local-projection branch")
	}
	branchProperties := mapping(t, projectionBranch["properties"])
	if mapping(t, branchProperties["version"])["const"] != "v1alpha1" {
		t.Fatal("local-projection schema does not pin version")
	}
	config := mapping(t, branchProperties["config"])
	if config["additionalProperties"] != false || !reflect.DeepEqual(sequence(t, config["required"]), []any{"contracts", "coverage"}) {
		t.Fatalf("local-projection config must be closed and require both paths: %#v", config)
	}
	if got := mapping(t, config["properties"]); len(got) != 2 || got["contracts"] == nil || got["coverage"] == nil {
		t.Fatalf("local-projection config keys = %#v", got)
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
