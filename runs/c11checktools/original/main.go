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
	APIVersion string `yaml:"apiVersion"`
	Kind string `yaml:"kind"`
	Metadata struct { Namespace string `yaml:"namespace"`; Name string `yaml:"name"` } `yaml:"metadata"`
	Purpose string `yaml:"purpose"`
	Spec map[string]any `yaml:"spec"`
}

func main() {
	fail := func(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	output, err := os.ReadFile("docs/represented/workflow.md/index.md"); if err != nil { fail(err) }
	text := string(output)
	if !strings.HasPrefix(text, "# Canonical projection\n") { fail(fmt.Errorf("original Markdown layout marker is absent")) }
	for _, fragment := range []string{
		"A project-owned vocabulary for describing an operational workflow and its guidance.",
		"States a project-owned constraint relevant to workflow behavior.", "Describes a workflow process and its explicit related guidance.",
		"Identifies an accountable role or party for workflow work.", "States a review condition that controls workflow progression.",
		"`statement` (string, 1..1): The canonical rule statement.", "`rule` (reference, 1..1): The rule associated with this process.",
		"`responsibility` (reference, 1..1): The responsibility associated with this process.", "`gate` (reference, 1..1): The gate associated with this process.",
		"`sequence` (string, 1..unbounded): Ordered human-readable process steps.", "`accountableParty` (string, 1..1): The project-defined role accountable for the process.",
		"`condition` (string, 1..1): The project-defined condition to satisfy before progression.",
		"## retain-source-authority", "Kind: Rule (workflow.example.org/v1)", "## review-workflow-change", "Kind: Process (workflow.example.org/v1)",
		"## workflow-owner", "Kind: Responsibility (workflow.example.org/v1)", "## source-review", "Kind: Gate (workflow.example.org/v1)",
		"Keep project-defined workflow meaning authoritative in the canonical source.", "Describe the project-owned review path for a workflow change.",
		"Assign accountability for project workflow meaning.", "Require source review before a workflow change is treated as ready.",
		"## Projection policies", "### rule-to-markdown", "### process-to-markdown", "### responsibility-to-markdown", "### gate-to-markdown",
		"Provides bounded markdown rendering guidance for the selected Rule Kind.", "Provides bounded markdown rendering guidance for the selected Process Kind.",
		"Provides bounded markdown rendering guidance for the selected Responsibility Kind.", "Provides bounded markdown rendering guidance for the selected Gate Kind.",
	} { if !strings.Contains(text, fragment) { fail(fmt.Errorf("original projection is missing required semantic fragment %q", fragment)) } }
	resources := []string{
		"examples/module-replacement/definitions/workflow.rule.yaml", "examples/module-replacement/definitions/workflow.process.yaml",
		"examples/module-replacement/definitions/workflow.responsibility.yaml", "examples/module-replacement/definitions/workflow.gate.yaml",
		"examples/module-replacement/definitions/policy.rule-to-markdown.yaml", "examples/module-replacement/definitions/policy.process-to-markdown.yaml",
		"examples/module-replacement/definitions/policy.responsibility-to-markdown.yaml", "examples/module-replacement/definitions/policy.gate-to-markdown.yaml",
	}
	for _, name := range resources {
		data, err := os.ReadFile(name); if err != nil { fail(err) }
		var expected sourceResource
		if err := yaml.Unmarshal(data, &expected); err != nil { fail(fmt.Errorf("decode %s: %w", name, err)) }
		marker := "\n## " + expected.Metadata.Name + "\n"
		if expected.Kind == "ProjectionPolicy" { marker = "\n### " + expected.Metadata.Name + "\n" }
		start := strings.Index(text, marker); if start < 0 { fail(fmt.Errorf("projection omitted exact canonical identity %s/%s/%s/%s", expected.APIVersion, expected.Kind, expected.Metadata.Namespace, expected.Metadata.Name)) }
		section := text[start+len(marker):]
		if expected.Kind == "ProjectionPolicy" { section = beforeNext(section, "\n### ", "\n## ") } else { section = beforeNext(section, "\n## ") }
		blocks := regexp.MustCompile("(?s)~~~json\\n(.*?)\\n~~~").FindAllStringSubmatch(section, -1)
		if len(blocks) != 1 { fail(fmt.Errorf("expected one exact value block for %s, found %d", expected.Metadata.Name, len(blocks))) }
		var got any
		if err := json.Unmarshal([]byte(blocks[0][1]), &got); err != nil { fail(fmt.Errorf("decode emitted %s values: %w", expected.Metadata.Name, err)) }
		want, err := canonicalJSON(expected.Spec); if err != nil { fail(err) }
		actual, err := canonicalJSON(got); if err != nil { fail(err) }
		if actual != want { fail(fmt.Errorf("rendered spec differs from the canonical spec for %s/%s/%s/%s", expected.APIVersion, expected.Kind, expected.Metadata.Namespace, expected.Metadata.Name)) }
	}
	fmt.Println("original Markdown projection preserves the fixed workflow schema, identities, purposes, policy guidance, and each associated canonical spec")
}

func beforeNext(value string, markers ...string) string { end:=len(value); for _,marker:=range markers { if i:=strings.Index(value,marker); i>=0 && i<end { end=i } }; return value[:end] }
func canonicalJSON(value any) (string,error) { encoded,err:=json.Marshal(value); if err!=nil{return "",err}; var normalized any; if err=json.Unmarshal(encoded,&normalized);err!=nil{return "",err}; encoded,err=json.Marshal(normalized);return string(encoded),err }