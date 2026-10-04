package authoring

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestBuiltinDomainDescriptorPreservesPublishedShape(t *testing.T) {
	domain, ok := NewRegistry().Domain(core.APIVersion)
	if !ok {
		t.Fatal("built-in authoring Domain descriptor is missing")
	}
	if len(domain.Kinds) != 6 || len(domain.Relations) != 4 {
		t.Fatalf("built-in Domain shape changed: kinds=%d relations=%d", len(domain.Kinds), len(domain.Relations))
	}
	for kind, definition := range domain.Kinds {
		if len(definition.Properties) != 4 {
			t.Fatalf("built-in kind %s descriptor changed: %#v", kind, definition.Properties)
		}
		for _, field := range []string{"rules", "uses", "needs", "implements"} {
			if _, ok := definition.Properties[field]; !ok {
				t.Fatalf("built-in kind %s lost reference field %s", kind, field)
			}
		}
	}
}

const validText = `apiVersion: markitect.example.org/v1alpha1
kind: Text
metadata:
  name: welcome
spec:
  text: |-
    Hello.
`

func TestParseAndEncodeRoundTrip(t *testing.T) {
	r, err := Parse("welcome.yaml", []byte(validText))
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != "Text" || r.Metadata.Name != "welcome" || r.Spec.Text != "Hello." || r.Path != "welcome.yaml" || r.Line != 1 {
		t.Fatalf("unexpected resource: %#v", r)
	}
	b, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse("encoded.yaml", b)
	if err != nil {
		t.Fatalf("encoded YAML did not parse: %v\n%s", err, b)
	}
	if roundTrip.Kind != r.Kind || !reflect.DeepEqual(roundTrip.Metadata, r.Metadata) || roundTrip.Spec.Text != r.Spec.Text {
		t.Fatalf("round trip mismatch: %#v", roundTrip)
	}
}

func TestParseRejectsInvalidRepresentations(t *testing.T) {
	tests := []struct{ name, input string }{
		{"duplicate root key", strings.Replace(validText, "kind: Text", "kind: Text\nkind: Rule", 1)},
		{"duplicate nested key", strings.Replace(validText, "  name: welcome", "  name: welcome\n  name: second", 1)},
		{"unknown nested field", strings.Replace(validText, "  name: welcome", "  name: welcome\n  surprise: true", 1)},
		{"unknown spec field", strings.Replace(validText, "  text: |", "  text: |", 1) + "  extra: true\n"},
		{"boolean coerced to string", strings.Replace(validText, "name: welcome", "name: true", 1)},
		{"numeric coerced to string", strings.Replace(validText, "name: welcome", "name: 12", 1)},
		{"empty text", strings.Replace(validText, "    Hello.", "    \n", 1)},
		{"wrong api version", strings.Replace(validText, "markitect.example.org/v1alpha1", "markitect.example.org/v2", 1)},
		{"alias", strings.Replace(validText, "  text: |", "  text: &body |", 1) + "  another: *body\n"},
		{"anchor", strings.Replace(validText, "  text: |", "  text: &body |", 1)},
		{"merge key", strings.Replace(validText, "  name: welcome", "  <<: {name: other}\n  name: welcome", 1)},
		{"custom tag", strings.Replace(validText, "name: welcome", "name: !custom welcome", 1)},
		{"two documents", validText + "---\n"},
		{"nonempty second document", validText + "---\nkind: Text\n"},
		{"uses needs kind", `apiVersion: markitect.example.org/v1alpha1
kind: Skill
metadata: {name: skill}
spec:
  text: content
  uses: [{name: wf}]
`},
		{"path traversal", `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: project}
spec:
  areas: [{name: scope, path: docs/../secret}]
`},
		{"removed project profile", `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: project}
spec:
  profile: generic
`},
		{"duplicate area path", `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: project}
spec:
  areas: [{name: one, path: docs/a}, {name: two, path: docs/a}]
`},
		{"wrong provider field", `apiVersion: markitect.example.org/v1alpha1
kind: Agent
metadata: {name: agent}
spec:
  text: content
  providers:
    codex: {model: x, permissionMode: ask}
`},
		{"providers unsupported on Skill", `apiVersion: markitect.example.org/v1alpha1
kind: Skill
metadata: {name: skill}
spec:
  text: content
  providers:
    codex: {model: x}
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse("fixture.yaml", []byte(tt.input)); err == nil {
				t.Fatal("expected parse error")
			} else if !strings.Contains(err.Error(), "fixture.yaml:") {
				t.Fatalf("diagnostic has no path and line: %v", err)
			}
		})
	}
}

func TestWorkflowCanImplementWorkflowContract(t *testing.T) {
	projectYAML := `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: workspace}
spec:
  areas: [{name: general, path: docs/general}]
  bindings:
    - contract: {kind: Contract, name: review, namespace: general}
      implementation: {kind: Workflow, name: reviewer, namespace: general}
`
	contractYAML := `apiVersion: markitect.example.org/v1alpha1
kind: Contract
metadata: {name: review, namespace: general}
spec:
  text: Review workflow contract.
  kind: Workflow
  input: [change]
  output: [findings]
`
	workflowYAML := `apiVersion: markitect.example.org/v1alpha1
kind: Workflow
metadata: {name: reviewer, namespace: general}
spec:
  text: Review the change.
  input: [change]
  output: [findings]
  implements: [{name: review}]
`
	_, err := Parse("markitect.yaml", []byte(projectYAML))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Parse("docs/general/review.yaml", []byte(contractYAML))
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := Parse("docs/general/reviewer.yaml", []byte(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	resources := []*core.Resource{&contract.Core, &workflow.Core}
	relationships, diagnostics := core.ResolveTypedRelationships(resources, NewRegistry())
	if len(diagnostics) != 0 {
		t.Fatalf("typed relationship resolution produced diagnostics: %#v", diagnostics)
	}
	g := core.BuildNormalized(resources, NewRegistry(), relationships, nil, "", nil)
	if len(g.Diagnostics) != 0 {
		t.Fatalf("normalized resources produced diagnostics: %#v", g.Diagnostics)
	}
	if len(g.Edges[workflow.Core.GraphKey()]) != 1 {
		t.Fatalf("generic typed relationship was not compiled: %#v", g.Relationships)
	}
}

func TestParseRejectsOversizedInput(t *testing.T) {
	data := []byte(validText + strings.Repeat("#", maxResourceSize))
	if _, err := Parse("large.yaml", data); err == nil || !strings.Contains(err.Error(), "2 MiB") {
		t.Fatalf("expected size diagnostic, got %v", err)
	}
}

func TestAllowedSpecFieldsReturnsIndependentSlice(t *testing.T) {
	fields := AllowedSpecFields("Contract")
	if len(fields) == 0 || !slices.Contains(fields, "files") {
		t.Fatalf("unexpected contract fields: %#v", fields)
	}
	fields[0] = "changed"
	if AllowedSpecFields("Contract")[0] == "changed" {
		t.Fatal("caller mutated centralized allowed fields")
	}
	if AllowedSpecFields("Unknown") != nil {
		t.Fatalf("unknown kind returned fields")
	}
}

func TestEncodeAcceptsCoreValue(t *testing.T) {
	b, err := Encode(Resource{Core: core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: "text"}}, Spec: Spec{Text: "body"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse("text.yaml", b); err != nil {
		t.Fatal(err)
	}
}
