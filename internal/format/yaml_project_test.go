package format

import (
	"strings"
	"testing"
)

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

func TestParseExplicitDomainDefinitionSelections(t *testing.T) {
	project := `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: workspace}
spec:
  domains: [domains/software.yaml, package:standards/domains/software.yaml]
`
	if _, err := Parse("markitect.yaml", []byte(project)); err != nil {
		t.Fatalf("valid local and pinned package domain selections were rejected: %v", err)
	}
	invalid := []string{
		"../domains/software.yaml",
		"package:/domains/software.yaml",
		"package:standards/../domains/software.yaml",
		"domains/software.yaml, DOMAINS/software.yaml",
	}
	for _, domains := range invalid {
		input := "apiVersion: markitect.example.org/v1alpha1\nkind: Project\nmetadata: {name: workspace}\nspec:\n  domains: [" + domains + "]\n"
		if _, err := Parse("markitect.yaml", []byte(input)); err == nil {
			t.Errorf("accepted invalid domain selection %q", domains)
		}
	}
	packageManifest := `apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: standards}
spec:
  version: 1.0.0
  areas: [{name: content, path: content}]
  domains: [domains/software.yaml]
  exports: [{kind: Rule, namespace: base, name: basics}]
`
	if _, err := Parse("markitect-package.yaml", []byte(packageManifest)); err != nil {
		t.Fatalf("valid package-relative domain path was rejected: %v", err)
	}
	packageManifest = strings.Replace(packageManifest, "domains/software.yaml", "package:other/domains/software.yaml", 1)
	if _, err := Parse("markitect-package.yaml", []byte(packageManifest)); err == nil {
		t.Fatal("Package domain declarations must not refer to another package")
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
