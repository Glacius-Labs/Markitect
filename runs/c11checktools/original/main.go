package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
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

func main() {
	fail := func(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	output, err := os.ReadFile("docs/represented/index.md"); if err != nil { fail(err) }
	text := string(output)
	if !strings.HasPrefix(text, "# Canonical projection\n") { fail(fmt.Errorf("original Markdown layout marker is absent")) }
	// Schema and rendered contract facts are fixed to the declared public fixture.
	for _, fragment := range []string{
		"A project-owned vocabulary for describing an operational workflow and its guidance.",
		"States a project-owned constraint relevant to workflow behavior.", "Describes a workflow process and its explicit related guidance.",
		"Identifies an accountable role or party for workflow work.", "States a review condition that controls workflow progression.",
		"`statement` (string, 1..1): The canonical rule statement.",
		"`rule` (reference, 1..1): The rule associated with this process.",
		"`responsibility` (reference, 1..1): The responsibility associated with this process.",
		"`gate` (reference, 1..1): The gate associated with this process.",
		"`sequence` (string, 1..unbounded): Ordered human-readable process steps.",
		"`accountableParty` (string, 1..1): The project-defined role accountable for the process.",
		"`condition` (string, 1..1): The project-defined condition to satisfy before progression.",
		"## retain-source-authority", "Kind: Rule (workflow.example.org/v1)", "Namespace: example",
		"## review-workflow-change", "Kind: Process (workflow.example.org/v1)",
		"## workflow-owner", "Kind: Responsibility (workflow.example.org/v1)",
		"## source-review", "Kind: Gate (workflow.example.org/v1)",
		"Keep project-defined workflow meaning authoritative in the canonical source.",
		"Describe the project-owned review path for a workflow change.", "Assign accountability for project workflow meaning.",
		"Require source review before a workflow change is treated as ready.",
		"## Projection policies", "### rule-to-markdown", "### process-to-markdown", "### responsibility-to-markdown", "### gate-to-markdown",
		"Provides bounded markdown rendering guidance for the selected Rule Kind.",
		"Provides bounded markdown rendering guidance for the selected Process Kind.",
		"Provides bounded markdown rendering guidance for the selected Responsibility Kind.",
		"Provides bounded markdown rendering guidance for the selected Gate Kind.",
	} { if !strings.Contains(text, fragment) { fail(fmt.Errorf("original projection is missing required semantic fragment %q", fragment)) } }
	paths := []string{
		"examples/module-replacement/definitions/workflow.rule.yaml", "examples/module-replacement/definitions/workflow.process.yaml",
		"examples/module-replacement/definitions/workflow.responsibility.yaml", "examples/module-replacement/definitions/workflow.gate.yaml",
		"examples/module-replacement/definitions/policy.rule-to-markdown.yaml", "examples/module-replacement/definitions/policy.process-to-markdown.yaml",
		"examples/module-replacement/definitions/policy.responsibility-to-markdown.yaml", "examples/module-replacement/definitions/policy.gate-to-markdown.yaml",
	}
	expected := make([]string, 0, len(paths))
	for _, name := range paths {
		data, err := os.ReadFile(name); if err != nil { fail(err) }
		var resource sourceResource
		if err := yaml.Unmarshal(data, &resource); err != nil { fail(fmt.Errorf("decode %s: %w", name, err)) }
		encoded, err := json.Marshal(resource.Spec); if err != nil { fail(err) }
		var canonical any
		if err := json.Unmarshal(encoded, &canonical); err != nil { fail(err) }
		encoded, err = json.Marshal(canonical); if err != nil { fail(err) }
		expected = append(expected, string(encoded))
	}
	blocks := regexp.MustCompile("(?s)~~~json\\n(.*?)\\n~~~").FindAllSubmatch(output, -1)
	actual := make([]string, 0, len(blocks))
	for _, block := range blocks {
		var value any
		if err := json.Unmarshal(block[1], &value); err != nil { fail(fmt.Errorf("invalid emitted JSON values: %w", err)) }
		encoded, err := json.Marshal(value); if err != nil { fail(err) }
		actual = append(actual, string(encoded))
	}
	sort.Strings(expected); sort.Strings(actual)
	if !equal(expected, actual) { fail(fmt.Errorf("the rendered Definition and ProjectionPolicy JSON values do not exactly cover the selected canonical specs")) }
	fmt.Println("original Markdown projection preserves the fixed workflow schema, identities, purposes, policy guidance, and canonical specs")
}

func equal(a, b []string) bool { return bytes.Equal([]byte(strings.Join(a, "\n")), []byte(strings.Join(b, "\n"))) }