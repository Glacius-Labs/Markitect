package format

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

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
	if roundTrip.Kind != r.Kind || roundTrip.Metadata != r.Metadata || roundTrip.Spec.Text != r.Spec.Text {
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

func TestParseProjectChecksAndRejectsInvalidCheckShapes(t *testing.T) {
	valid := `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: workspace}
spec:
  areas: [{name: general, path: docs/general}]
  checks:
    - name: unit-test
      run: [go, test, ./...]
    - name: format
      run: [gofmt, -w, .]
`
	project, err := Parse("markitect.yaml", []byte(valid))
	if err != nil {
		t.Fatalf("valid project checks failed to parse: %v", err)
	}
	if got := project.Spec.Checks; len(got) != 2 || got[0].Name != "unit-test" || strings.Join(got[0].Run, "|") != "go|test|./..." || got[1].Name != "format" {
		t.Fatalf("check order/argv was not preserved: %#v", got)
	}

	invalidChecks := []struct {
		name string
		body string
		want string
	}{
		{name: "duplicate names", body: `- {name: test, run: [go, test]}
    - {name: test, run: [go, vet]}`, want: "duplicated"},
		{name: "non-simple name", body: `- {name: ../test, run: [go, test]}`, want: "must match"},
		{name: "missing name", body: `- {run: [go, test]}`, want: `required field "name" is missing`},
		{name: "missing run", body: `- {name: test}`, want: `required field "run" is missing`},
		{name: "unknown field", body: `- {name: test, run: [go, test], shell: true}`, want: "unknown field"},
		{name: "run is not argv sequence", body: `- {name: test, run: "go test"}`, want: "must be a sequence"},
		{name: "empty argv", body: `- {name: test, run: []}`, want: "at least one argument"},
		{name: "blank executable", body: "- {name: test, run: [\" \\t\", arg]}", want: "bare PATH command name"},
		{name: "relative executable path", body: `- {name: test, run: [./scripts/check]}`, want: "bare PATH command name"},
		{name: "NUL argument", body: `- {name: test, run: [go, "bad\0arg"]}`, want: "NUL"},
		{name: "nonstring argument", body: `- {name: test, run: [go, true]}`, want: "expected string"},
	}
	for _, tt := range invalidChecks {
		t.Run(tt.name, func(t *testing.T) {
			input := "apiVersion: markitect.example.org/v1alpha1\nkind: Project\nmetadata: {name: workspace}\nspec:\n  checks:\n    " + tt.body + "\n"
			if _, err := Parse("project.yaml", []byte(input)); err == nil || !strings.Contains(err.Error(), "project.yaml:") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected path-qualified diagnostic containing %q, got %v", tt.want, err)
			}
		})
	}

	nonProject := `apiVersion: markitect.example.org/v1alpha1
kind: Skill
metadata: {name: skill}
spec:
  text: content
  checks: [{name: test, run: [go, test]}]
`
	if _, err := Parse("skill.yaml", []byte(nonProject)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("checks should only be valid on Project: %v", err)
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
	projectResource, err := Parse("markitect.yaml", []byte(projectYAML))
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
	g := core.Build([]*core.Resource{projectResource, contract, workflow})
	if len(g.Diagnostics) != 0 {
		t.Fatalf("compatible Workflow contract binding produced diagnostics: %#v", g.Diagnostics)
	}
}

func TestParseRejectsOversizedInput(t *testing.T) {
	data := []byte(validText + strings.Repeat("#", maxResourceSize))
	if _, err := Parse("large.yaml", data); err == nil || !strings.Contains(err.Error(), "2 MiB") {
		t.Fatalf("expected size diagnostic, got %v", err)
	}
}

func TestParseProjectAndContractConstraints(t *testing.T) {
	project := `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: workspace}
spec:
  targets: [codex, claude]
  areas:
    - name: general
      path: docs/general
      imports: [other]
      rules: [{name: policy}]
  bindings:
    - contract: {name: review}
      implementation: {kind: Agent, name: reviewer}
  ruleAdapters:
    review: [{name: review-rule}]
`
	if _, err := Parse("markitect.yaml", []byte(project)); err != nil {
		t.Fatal(err)
	}
	contract := `apiVersion: markitect.example.org/v1alpha1
kind: Contract
metadata: {name: review}
spec:
  kind: Agent
  input: [change, change]
  text: Review the change.
`
	if _, err := Parse("contract.yaml", []byte(contract)); err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected duplicate signature diagnostic, got %v", err)
	}
}

func TestParseFilesOnResourcesAndRejectsFilesOnProject(t *testing.T) {
	text := `apiVersion: markitect.example.org/v1alpha1
kind: Contract
metadata: {name: api}
spec:
  kind: Agent
  text: Contract description.
  files: [src/contracts/review.go]
`
	resource, err := Parse("contract.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if len(resource.Spec.Files) != 1 || resource.Spec.Files[0] != "src/contracts/review.go" {
		t.Fatalf("files were not decoded: %#v", resource.Spec.Files)
	}
	project := `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: workspace}
spec:
  files: [docs/general/notes.md]
`
	if _, err := Parse("markitect.yaml", []byte(project)); err == nil {
		t.Fatal("Project files must be rejected")
	}
}

func TestAllowedSpecFieldsReturnsIndependentSlice(t *testing.T) {
	fields := AllowedSpecFields("Contract")
	if len(fields) == 0 || fields[len(fields)-1] != "files" {
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
	b, err := Encode(core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: "text"}, Spec: core.Spec{Text: "body"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse("text.yaml", b); err != nil {
		t.Fatal(err)
	}
}
