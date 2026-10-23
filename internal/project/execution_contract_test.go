package project_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func executionResult(t *testing.T, content string) project.CapturedExecutionDecoding {
	t.Helper()
	selection, raw := executionInputs(t, content)
	return registryModel(t, selection, raw).DecodeExecution()
}

// These mutations exercise observable rejection, not a mirror of parser branches.
func TestExecutionRejectsClosedContractViolations(t *testing.T) {
	base := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	for _, tc := range []struct{ name, from, to, code string }{
		{"version", "markitect.execution/v1", "markitect.execution/v2", "execution.version"},
		{"strategy", "strategy: direct", "strategy: unreviewed", "execution.strategy"},
		{"replaced strategy", "strategy: direct", "strategy: recursive-passes-v1", "execution.strategy"},
		{"old strategy", "strategy: direct", "strategy: shared-workspace-v1", "execution.strategy"},
		{"strategy case", "strategy: direct", "strategy: Direct", "execution.strategy"},
		{"root option", "budget:", "disableReview: true\nbudget:", "execution.unknown_key"},
		{"nested option", "agentTurns: 120", "agentTurns: 120\n  unlimited: true", "execution.unknown_key"},
		{"old limits section", "budget:", "limits: {maxRuns: 5}\nbudget:", "execution.unknown_key"},
		{"old assignment section", "budget:", "assignment: {runs: 3}\nbudget:", "execution.unknown_key"},
		{"old defaults section", "budget:", "defaults: {rejectionLimit: 3}\nbudget:", "execution.unknown_key"},
		{"old acceptance section", "checks:", "acceptance: {ownerApproval: []}\nchecks:", "execution.unknown_key"},
		{"old step timeout key", "turnTimeout: 10m", "turnTimeout: 10m\n  stepTimeout: 10m", "execution.unknown_key"},
		{"old role", "execute: library-agent", "manager: library-agent", "execution.unknown_key"},
		{"duplicate", "agentTurns: 120", "agentTurns: 120\n  agentTurns: 6", "execution.duplicate_key"},
		{"duplicate package", "checks:", "  lending: {}\nchecks:", "execution.duplicate_key"},
		{"null", "agentTurns: 120", "agentTurns: null", "execution.yaml_feature"},
		{"implicit null", "agentTurns: 120", "agentTurns:", "execution.yaml_feature"},
		{"anchor", "agentTurns: 120", "agentTurns: &cap 120", "execution.yaml_feature"},
		{"alias", "agentTurns: 120\n  passes: 12", "agentTurns: &cap 120\n  passes: *cap", "execution.yaml_feature"},
		{"custom tag", "agentTurns: 120", "agentTurns: !cap 120", "execution.yaml_feature"},
		{"URI tag", "apiVersion:", "!<tag:example.org,2026:execution>\napiVersion:", "execution.yaml_feature"},
		{"multiple documents", "apiVersion:", "---\n{}\n---\napiVersion:", "execution.documents"},
		{"numeric key", "agentTurns: 120", "99: 120", "execution.key"},
		{"missing budget", "budget:\n  executionTime: 25m\n  turnTimeout: 10m\n  agentTurns: 120\n  passes: 12\n", "", "execution.field_missing"},
		{"missing execution time", "  executionTime: 25m\n", "", "execution.field_missing"},
		{"missing turn timeout", "  turnTimeout: 10m\n", "", "execution.field_missing"},
		{"missing agent turns", "  agentTurns: 120\n", "", "execution.field_missing"},
		{"missing passes", "  passes: 12\n", "", "execution.field_missing"},
		{"old rounds budget", "passes: 12", "rounds: 4", "execution.unknown_key"},
		{"missing profiles", "profiles:\n  execute: library-agent\n  verify: library-agent\n", "", "execution.field_missing"},
		{"missing verify profile", "  verify: library-agent\npackages:", "packages:", "execution.field_missing"},
		{"unknown package", "  lending:", "  missing:", "execution.package"},
		{"imported package", "  lending:", "  software:", "execution.package"},
		{"wildcard package", "  lending:", `  "lending/*":`, "execution.package"},
		{"package option", "profiles: {verify: library-agent}", "profiles: {verify: library-agent}\n    stepTimeout: 8m", "execution.unknown_key"},
		{"package budget", "profiles: {verify: library-agent}", "profiles: {verify: library-agent}\n    budget: {passes: 2}", "execution.unknown_key"},
		{"unknown definition", "library/callable-interface", "library/no-such-definition", "execution.definition_missing"},
		{"short definition", "library/callable-interface", "callable-interface", "execution.definition_address"},
		{"directory escape", "directory: realization", "directory: ../escape", "execution.directory"},
		{"absolute directory", "directory: realization", "directory: /tmp", "execution.directory"},
		{"argument type", "args: [test, ./...]", "args: [test, 5]", "execution.string"},
		{"empty instruction", "profiles:\n  execute", "instructions: {execute: [\"\"]}\nprofiles:\n  execute", "execution.instruction_reference"},
		{"role instruction key", "profiles:\n  execute", "instructions: {manager: []}\nprofiles:\n  execute", "execution.unknown_key"},
		{"scope address", "checks:", "scope: [{kind: Rule, property: software/appliesTo}]\nchecks:", "execution.definition_address"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.from) {
				t.Fatal("test mutation does not match documented source")
			}
			content := strings.Replace(base, tc.from, tc.to, 1)
			result := executionResult(t, content)
			if _, ok := result.Document(); ok || result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), tc.code) {
				t.Fatalf("invalid document accepted or wrong diagnostic: %s %+v", result.Status(), result.Diagnostics())
			}
			if string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
				t.Fatal("rejected source was rewritten or discarded")
			}
		})
	}
	check := "  - id: project-tests\n    profile: library-tests\n    directory: realization\n    args: [test, ./...]\n    requiredBy: library/callable-interface\n    timeout: 2m\n"
	result := executionResult(t, strings.Replace(base, check, check+check, 1))
	if result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), "execution.duplicate_check") {
		t.Fatal("duplicate check identity accepted")
	}
}

func TestExecutionOptionalAbsenceAndExplicitEmptySelections(t *testing.T) {
	base := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	omitted := base[:strings.Index(base, "packages:\n")]
	result := executionResult(t, omitted)
	document, ok := result.Document()
	if !ok || document.Packages != nil || document.Ownership != nil || document.Team != nil || document.Checks != nil || document.Scope != nil || document.Defaults.Execute.Instructions != nil {
		t.Fatalf("absence lost or invented: %+v %+v", document, result.Diagnostics())
	}
	explicit := executionResult(t, omitted+"packages: {}\nscope: []\nchecks: []\nownership: []\nteam: {maxHelpers: 1, helpers: []}\n")
	document, ok = explicit.Document()
	if !ok || document.Packages == nil || document.Scope == nil || document.Checks == nil || document.Ownership == nil || document.Team == nil {
		t.Fatalf("explicit empty selections lost: %+v", explicit.Diagnostics())
	}
}

func TestExecutionPreservesExactOverridesAndLiteralListOrder(t *testing.T) {
	base := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	base = strings.Replace(base, "profiles:\n  execute", "instructions:\n  execute: [one.md, two.md, one.md]\n  verify: []\nprofiles:\n  execute", 1)
	base = strings.Replace(base, "packages:\n  lending:\n    profiles: {verify: library-agent}", "packages:\n  lending:\n    profiles: {execute: unobserved-profile}\n    instructions: {execute: [three.md, one.md], verify: [review.md]}\n  '.':\n    profiles: {verify: other}\n  lending/reminders: {}", 1)
	base = strings.Replace(base, "args: [test, ./...]", `args: [test, '$(literal)', ';', '', test]`, 1)
	base = strings.Replace(base, "directory: realization", "directory: .", 1)
	result := executionResult(t, base)
	doc, ok := result.Document()
	if !ok {
		t.Fatalf("authored overrides failed: %+v", result.Diagnostics())
	}
	if !reflect.DeepEqual(doc.Defaults.Execute.Instructions.Value, []string{"one.md", "two.md", "one.md"}) || doc.Defaults.Verify.Instructions == nil || len(doc.Defaults.Verify.Instructions.Value) != 0 {
		t.Fatal("default instruction order/absence changed")
	}
	packages := doc.Packages.Value
	if len(packages) != 3 || packages[0].Package != "lending" || packages[1].Package != "." || packages[2].Package != "lending/reminders" || packages[2].Execute.Profile != nil || packages[2].Verify.Profile != nil || packages[1].Verify.Profile.Value != "other" || packages[1].Execute.Profile != nil {
		t.Fatal("exact authored overrides changed or inherited")
	}
	if packages[0].Execute.Profile.Value != "unobserved-profile" || packages[0].Execute.Profile.Source.Field != "/packages/lending/profiles/execute" || !reflect.DeepEqual(packages[0].Execute.Instructions.Value, []string{"three.md", "one.md"}) || !reflect.DeepEqual(packages[0].Verify.Instructions.Value, []string{"review.md"}) {
		t.Fatal("profile declarations or override order/source lost")
	}
	if !reflect.DeepEqual(doc.Checks.Value[0].Args.Value, []string{"test", "$(literal)", ";", "", "test"}) || doc.Checks.Value[0].Directory.Value != snapshot.RootDirectory() {
		t.Fatal("literal check order changed")
	}
}
