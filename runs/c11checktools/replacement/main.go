package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

type sourceResource struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   struct { Namespace string `yaml:"namespace"`; Name string `yaml:"name"` } `yaml:"metadata"`
	Purpose    string         `yaml:"purpose"`
	Spec       map[string]any `yaml:"spec"`
}
type emittedResource struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   struct { Namespace string `json:"namespace"`; Name string `json:"name"` } `json:"metadata"`
	Purpose    string         `json:"purpose"`
	Spec       map[string]any `json:"spec"`
}
type schema struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion"`
	Purpose    string         `yaml:"purpose" json:"purpose"`
	Kinds      map[string]any `yaml:"kinds" json:"kinds"`
}

func main() {
	fail := func(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	output, err := os.ReadFile("docs/represented/workflow.md/canonical-projection.md"); if err != nil { fail(err) }
	text := string(output)
	if !strings.HasPrefix(text, "# Canonical Reference Bundle\n") || !strings.Contains(text, "## Schema contracts") || !strings.Contains(text, "## Selected canonical Definitions") || !strings.Contains(text, "## Selected ProjectionPolicies") { fail(fmt.Errorf("reference bundle sections or distinct layout are absent")) }
	blocks := regexp.MustCompile("(?s)~~~json\\n(.*?)\\n~~~").FindAllSubmatch(output, -1)
	var gotSchema *schema
	gotDefinitions := map[string]emittedResource{}
	gotPolicies := map[string]emittedResource{}
	for _, block := range blocks {
		var tag struct { Kind string `json:"kind"` }
		if err := json.Unmarshal(block[1], &tag); err != nil { fail(fmt.Errorf("invalid JSON block: %w", err)) }
		if tag.Kind == "" { var value schema; if err := json.Unmarshal(block[1], &value); err != nil { fail(err) }; if gotSchema != nil { fail(fmt.Errorf("duplicate Schema payload")) }; gotSchema = &value; continue }
		var value emittedResource
		if err := json.Unmarshal(block[1], &value); err != nil { fail(err) }
		key := value.APIVersion + "/" + value.Kind + "/" + value.Metadata.Namespace + "/" + value.Metadata.Name
		if value.Kind == "ProjectionPolicy" { gotPolicies[key] = value } else { gotDefinitions[key] = value }
	}
	if gotSchema == nil { fail(fmt.Errorf("Schema contract payload is absent")) }
	schemaBytes, err := os.ReadFile("examples/canonical-workflow/modules/workflow/schema.yaml"); if err != nil { fail(err) }
	var expectedSchema schema
	if err := yaml.Unmarshal(schemaBytes, &expectedSchema); err != nil { fail(err) }
	if gotSchema.APIVersion != expectedSchema.APIVersion || gotSchema.Purpose != expectedSchema.Purpose || !sameJSON(gotSchema.Kinds, expectedSchema.Kinds) { fail(fmt.Errorf("Schema contract payload differs from the selected public Schema")) }
	definitionPaths := []string{"examples/module-replacement/definitions/workflow.rule.yaml", "examples/module-replacement/definitions/workflow.process.yaml", "examples/module-replacement/definitions/workflow.responsibility.yaml", "examples/module-replacement/definitions/workflow.gate.yaml"}
	for _, name := range definitionPaths { checkResource(name, gotDefinitions, fail) }
	policyPaths := []string{"examples/module-replacement/definitions/policy.rule-to-markdown.yaml", "examples/module-replacement/definitions/policy.process-to-markdown.yaml", "examples/module-replacement/definitions/policy.responsibility-to-markdown.yaml", "examples/module-replacement/definitions/policy.gate-to-markdown.yaml"}
	for _, name := range policyPaths { checkResource(name, gotPolicies, fail) }
	if len(gotDefinitions) != 4 || len(gotPolicies) != 4 { fail(fmt.Errorf("reference bundle contains extra or missing selected resources: definitions=%d policies=%d", len(gotDefinitions), len(gotPolicies))) }
	fmt.Println("replacement JSON reference bundle preserves the fixed workflow Schema, Definition identities, purposes, and complete specs")
}

func checkResource(name string, values map[string]emittedResource, fail func(error)) {
	data, err := os.ReadFile(name); if err != nil { fail(err) }
	var expected sourceResource
	if err := yaml.Unmarshal(data, &expected); err != nil { fail(fmt.Errorf("decode %s: %w", name, err)) }
	key := expected.APIVersion + "/" + expected.Kind + "/" + expected.Metadata.Namespace + "/" + expected.Metadata.Name
	actual, ok := values[key]; if !ok { fail(fmt.Errorf("reference bundle omitted canonical identity %s", key)) }
	if actual.Purpose != expected.Purpose || !sameJSON(actual.Spec, expected.Spec) { fail(fmt.Errorf("reference bundle changed purpose or spec for %s", key)) }
}
func sameJSON(left, right any) bool { a, err := json.Marshal(left); if err != nil { return false }; var x any; if json.Unmarshal(a, &x) != nil { return false }; b, err := json.Marshal(right); if err != nil { return false }; var y any; if json.Unmarshal(b, &y) != nil { return false }; cx, _ := json.Marshal(x); cy, _ := json.Marshal(y); return string(cx) == string(cy) }