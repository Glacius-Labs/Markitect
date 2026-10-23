package project_test

import (
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"strings"
	"testing"
)

func executionResult(t *testing.T, content string) project.CapturedExecutionDecoding {
	t.Helper()
	selection, raw := executionInputs(t, content)
	return registryModel(t, selection, raw).DecodeExecution()
}

func TestExecutionRejectsClosedContractViolations(t *testing.T) {
	base := documentedExecution(t)
	cases := []struct{ name, from, to, code string }{
		{"version", "markitect.execution/v2", "markitect.execution/v1", "execution.version"},
		{"unknown", "checks:", "disableReview: true\nchecks:", "execution.unknown_key"},
		{"duplicate", "apiVersion: markitect.execution/v2", "apiVersion: markitect.execution/v2\napiVersion: markitect.execution/v2", "execution.duplicate_key"},
		{"null", "checks:", "instructions: null\nchecks:", "execution.yaml_feature"},
		{"anchor", "checks:", "instructions: &refs {}\nchecks:", "execution.yaml_feature"},
		{"documents", "apiVersion:", "---\n{}\n---\napiVersion:", "execution.documents"},
		{"numeric key", "checks:", "99: invalid\nchecks:", "execution.key"},
		{"unknown Package", "checks:", "instructions: {missing: [ops/one.md]}\nchecks:", "execution.package"},
		{"imported Package", "checks:", "instructions: {software: []}\nchecks:", "execution.package"},
		{"empty ref", "checks:", "instructions: {'.': ['']}\nchecks:", "execution.instruction_reference"},
		{"unknown Definition", "library/callable-interface", "library/missing", "execution.definition_missing"},
		{"short Definition", "library/callable-interface", "callable-interface", "execution.definition_address"},
		{"directory escape", "directory: realization", "directory: ../escape", "execution.directory"},
		{"absolute directory", "directory: realization", "directory: /tmp", "execution.directory"},
		{"argument type", "args: [test, ./...]", "args: [test, 5]", "execution.string"},
		{"unknown check key", "timeout: 2m", "timeout: 2m\n    skip: true", "execution.unknown_key"},
	}
	for _, section := range []string{"strategy: direct", "budget: {}", "profiles: {}", "packages: {}", "team: {}"} {
		cases = append(cases, struct{ name, from, to, code string }{section, "checks:", section + "\nchecks:", "execution.unknown_key"})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.from) {
				t.Fatal("ineffective test mutation")
			}
			content := strings.Replace(base, tc.from, tc.to, 1)
			result := executionResult(t, content)
			if result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), tc.code) {
				t.Fatalf("%s %+v", result.Status(), result.Diagnostics())
			}
			if _, ok := result.Document(); ok || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
				t.Fatal("invalid input became usable or original bytes lost")
			}
		})
	}
}
