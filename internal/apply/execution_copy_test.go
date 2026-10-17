package apply_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestExecutionAccessorsCopyNestedAuthoredValues(t *testing.T) {
	content := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	content = strings.Replace(content, "  profiles:\n", "  instructions: {manager: [one.md], reviewer: [two.md]}\n  profiles:\n", 1)
	content = strings.Replace(content, "    stepTimeout: 8m", "    profiles: {manager: a, reviewer: b}\n    instructions: {manager: [three.md], reviewer: [four.md]}\n    stepTimeout: 8m\n    reviewRounds: 2\n    creationRoots: [realization/lending]", 1)
	content = strings.Replace(content, "checks:\n", "scope: [{kind: software/Rule, property: lending/maxOpenLoans}]\nchecks:\n", 1)
	content = strings.Replace(content, "ownerApproval: []", "ownerApproval: [lending]", 1)
	result := executionResult(t, content)
	original, ok := result.Document()
	if !ok {
		t.Fatalf("copy fixture invalid: %+v", result.Diagnostics())
	}
	// This stage preserves reference selections. Effective scope semantics belong
	// to organization context selection, not to this decoding observation.
	changed, _ := result.Document()
	changed.Limits.CreationRoots.Value[0] = area(t, "other")
	changed.Defaults.CreationRoots.Value[0] = area(t, "other")
	changed.Defaults.Manager.Profile.Value = "changed"
	changed.Defaults.Reviewer.Profile.Value = "changed"
	changed.Defaults.Manager.Instructions.Value[0] = "changed"
	changed.Defaults.Reviewer.Instructions.Value[0] = "changed"
	changed.Defaults.ReviewRounds.Value = 99
	changed.Defaults.BatchNoChangeReviews.Value = true
	p := &changed.Packages.Value[0]
	p.Manager.Profile.Value, p.Reviewer.Profile.Value = "changed", "changed"
	p.Manager.Instructions.Value[0], p.Reviewer.Instructions.Value[0] = "changed", "changed"
	p.StepTimeout.Value, p.ReviewRounds.Value = 99, 99
	p.CreationRoots.Value[0] = area(t, "other")
	changed.Scope.Value[0].Value.Kind = model.Identity{Package: "other", ID: "other"}
	changed.Checks.Value[0].Args.Value[0] = "changed"
	changed.Checks.Value[0].RequiredBy.Value = model.Identity{Package: "other", ID: "other"}
	changed.OwnerApproval.Value[0] = "changed"
	if fresh, _ := result.Document(); !reflect.DeepEqual(fresh, original) {
		t.Fatal("document accessor leaked nested mutable state")
	}
	partial := result.Partial()
	partial.Checks.Value[0].Args.Value[0] = "changed"
	if !reflect.DeepEqual(result.Partial(), original) {
		t.Fatal("partial accessor leaked nested mutable state")
	}
	inputs := result.Model().Inputs()
	inputs.Raw.Execution[0].RepositoryPath = path(t, "other.yaml")
	inputs.Raw.Execution[0].File.Bytes()[0] = '!'
	if string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content || result.Model().Inputs().Raw.Execution[0].RepositoryPath.String() != ".markitect/execution.yaml" {
		t.Fatal("parent source accessor leaked mutable state")
	}
}

func TestExecutionPartialDiagnosticsRemainIndependent(t *testing.T) {
	content := strings.Replace(documentedExecution(t), "maxRuns: 5", "maxRuns: wrong", 1)
	result := executionResult(t, content)
	before := result.Diagnostics()
	changed := result.Diagnostics()
	if len(changed) == 0 || changed[0].Position == nil {
		t.Fatal("source coordinates missing")
	}
	changed[0].Code, changed[0].Source.Field, changed[0].Position.Line = "changed", "changed", 99
	if !reflect.DeepEqual(result.Diagnostics(), before) {
		t.Fatal("diagnostic accessor leaked nested mutable state")
	}
	partial := result.Partial()
	partial.Checks.Value[0].Args.Value[0] = "changed"
	if result.Partial().Checks.Value[0].Args.Value[0] != "test" || result.Partial().Limits.MaxSteps.Value != 320 {
		t.Fatal("partial known facts mutated or discarded")
	}
	var empty apply.CapturedExecutionDecoding
	if _, ok := empty.Document(); ok || empty.Status() != model.NotCompleted {
		t.Fatal("zero result is usable")
	}
}
