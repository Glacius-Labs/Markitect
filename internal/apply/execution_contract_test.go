package apply_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func executionResult(t *testing.T, content string) apply.CapturedExecutionDecoding {
	t.Helper()
	selection, raw := executionInputs(t, content)
	return registryModel(t, selection, raw).DecodeExecution()
}

// These mutations exercise observable rejection, not a mirror of parser branches.
func TestExecutionRejectsClosedContractViolations(t *testing.T) {
	base := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	for _, tc := range []struct{ name, from, to, code string }{
		{"version", "markitect.execution/v1", "markitect.execution/v2", "execution.version"},
		{"strategy", "recursive-apply-v1", "unreviewed", "execution.strategy"},
		{"root option", "limits:", "disableReview: true\nlimits:", "execution.unknown_key"},
		{"nested option", "maxRuns: 5", "maxRuns: 5\n  unlimited: true", "execution.unknown_key"},
		{"old rejection ceiling key", "maxRejectionLimit: 3", "maxReviewRounds: 3", "execution.unknown_key"},
		{"old rejection setting key", "rejectionLimit: 3", "reviewRounds: 3", "execution.unknown_key"},
		{"role", "manager: library-agent", "owner: library-agent", "execution.unknown_key"},
		{"duplicate", "maxRuns: 5", "maxRuns: 5\n  maxRuns: 6", "execution.duplicate_key"},
		{"duplicate override", "stepTimeout: 8m", "stepTimeout: 8m\n    stepTimeout: 7m", "execution.duplicate_key"},
		{"duplicate package", "checks:", "  lending: {}\nchecks:", "execution.duplicate_key"},
		{"null", "maxRuns: 5", "maxRuns: null", "execution.yaml_feature"},
		{"implicit null", "maxRuns: 5", "maxRuns:", "execution.yaml_feature"},
		{"anchor", "maxRuns: 5", "maxRuns: &cap 5", "execution.yaml_feature"},
		{"alias", "maxRuns: 5\n  maxSessionStarts: 160", "maxRuns: &cap 5\n  maxSessionStarts: *cap", "execution.yaml_feature"},
		{"custom tag", "maxRuns: 5", "maxRuns: !cap 5", "execution.yaml_feature"},
		{"URI tag", "apiVersion:", "!<tag:example.org,2026:execution>\napiVersion:", "execution.yaml_feature"},
		{"multiple documents", "apiVersion:", "---\n{}\n---\napiVersion:", "execution.documents"},
		{"numeric key", "maxRuns: 5", "99: 5", "execution.key"},
		{"missing cap", "  maxRuns: 5\n", "", "execution.field_missing"},
		{"missing hard rejection ceiling", "  maxRejectionLimit: 3\n", "", "execution.field_missing"},
		{"missing assignment", "  runs: 3\n", "", "execution.field_missing"},
		{"missing reserve", "  sessionStarts: 24\n", "", "execution.field_missing"},
		{"missing profile", "    manager: library-agent\n", "", "execution.field_missing"},
		{"string boolean", "batchNoChangeReviews: false", `batchNoChangeReviews: "false"`, "execution.boolean"},
		{"unknown package", "  lending:", "  missing:", "execution.package"},
		{"imported package", "  lending:", "  software:", "execution.package"},
		{"wildcard package", "  lending:", `  "lending/*":`, "execution.package"},
		{"unknown approval", "ownerApproval: []", "ownerApproval: [missing]", "execution.package"},
		{"unknown definition", "library/callable-interface", "library/no-such-definition", "execution.definition_missing"},
		{"short definition", "library/callable-interface", "callable-interface", "execution.definition_address"},
		{"directory escape", "directory: realization", "directory: ../escape", "execution.directory"},
		{"absolute directory", "directory: realization", "directory: /tmp", "execution.directory"},
		{"argument type", "args: [test, ./...]", "args: [test, 5]", "execution.string"},
		{"empty instruction", "  profiles:", "  instructions: {manager: [\"\"]}\n  profiles:", "execution.instruction_reference"},
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
	omitted := strings.Replace(base, "  rejectionLimit: 3\n", "", 1)
	omitted = strings.Replace(omitted, "  batchNoChangeReviews: false\n", "", 1)
	omitted = omitted[:strings.Index(omitted, "packages:\n")]
	result := executionResult(t, omitted)
	document, ok := result.Document()
	if !ok || document.Defaults.RejectionLimit != nil || document.Defaults.BatchNoChangeReviews != nil || document.Packages != nil || document.Checks != nil || document.Scope != nil || document.OwnerApproval != nil || document.Defaults.Manager.Instructions != nil {
		t.Fatalf("absence lost or invented: %+v %+v", document, result.Diagnostics())
	}
	if document.Limits.MaxRejectionLimit.Value != 3 {
		t.Fatal("hard ceiling lost with optional authored default")
	}
	explicit := executionResult(t, omitted+"packages: {}\nscope: []\nchecks: []\nacceptance: {ownerApproval: []}\n")
	document, ok = explicit.Document()
	if !ok || document.Packages == nil || document.Scope == nil || document.Checks == nil || document.OwnerApproval == nil {
		t.Fatal("explicit empty selections lost")
	}
	declared := executionResult(t, base).Partial().Defaults.RejectionLimit
	if declared == nil || declared.Value != 3 || declared.Source.Field != "/defaults/rejectionLimit" {
		t.Fatal("explicit rejection limit lost its source")
	}
	emptyRoots := executionResult(t, strings.ReplaceAll(base, "creationRoots: [realization]", "creationRoots: []"))
	if emptyRoots.Status() != model.Passed || emptyRoots.Partial().Limits.CreationRoots.Source.File == "" || emptyRoots.Partial().Defaults.CreationRoots.Source.File == "" {
		t.Fatal("empty creation selection treated as a nonpositive cap")
	}
}

func TestExecutionPreservesExactOverridesAndLiteralListOrder(t *testing.T) {
	base := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	base = strings.Replace(base, "  profiles:\n", "  instructions:\n    manager: [one.md, two.md, one.md]\n    reviewer: []\n  profiles:\n", 1)
	base = strings.Replace(base, "packages:\n  lending:\n    stepTimeout: 8m", "packages:\n  lending:\n    profiles: {manager: unobserved-profile}\n    instructions: {manager: [three.md, one.md], reviewer: [review.md]}\n    stepTimeout: 8m\n    rejectionLimit: 2\n    creationRoots: [realization/lending]\n  '.':\n    stepTimeout: 9m\n  lending/reminders: {}", 1)
	base = strings.Replace(base, "args: [test, ./...]", `args: [test, '$(literal)', ';', '', test]`, 1)
	base = strings.Replace(base, "directory: realization", "directory: .", 1)
	base = strings.Replace(base, "ownerApproval: []", "ownerApproval: [lending/reminders, ., lending]", 1)
	result := executionResult(t, base)
	doc, ok := result.Document()
	if !ok {
		t.Fatalf("authored overrides failed: %+v", result.Diagnostics())
	}
	if !reflect.DeepEqual(doc.Defaults.Manager.Instructions.Value, []string{"one.md", "two.md", "one.md"}) || doc.Defaults.Reviewer.Instructions == nil || len(doc.Defaults.Reviewer.Instructions.Value) != 0 {
		t.Fatal("default instruction order/absence changed")
	}
	packages := doc.Packages.Value
	if len(packages) != 3 || packages[0].Package != "lending" || packages[1].Package != "." || packages[2].Package != "lending/reminders" || packages[2].StepTimeout != nil || packages[2].Manager.Profile != nil || packages[0].StepTimeout.Value != 8*time.Minute || packages[1].StepTimeout.Value != 9*time.Minute {
		t.Fatal("exact authored overrides changed or inherited")
	}
	if packages[0].Manager.Profile.Value != "unobserved-profile" || packages[0].RejectionLimit.Value != 2 || packages[0].StepTimeout.Source.Field != "/packages/lending/stepTimeout" || !reflect.DeepEqual(packages[0].Manager.Instructions.Value, []string{"three.md", "one.md"}) {
		t.Fatal("profile declarations or override order/source lost")
	}
	if !reflect.DeepEqual(doc.Checks.Value[0].Args.Value, []string{"test", "$(literal)", ";", "", "test"}) || doc.Checks.Value[0].Directory.Value != snapshot.RootDirectory() || !reflect.DeepEqual(doc.OwnerApproval.Value, []string{"lending/reminders", ".", "lending"}) {
		t.Fatal("literal check/approval order changed")
	}
}
