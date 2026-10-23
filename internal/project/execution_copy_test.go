package project_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

func TestExecutionAccessorsCopyNestedAuthoredValues(t *testing.T) {
	content := strings.Replace(documentedExecution(t), "checks:", "instructions: {'.': [one.md,two.md,one.md], lending: [three.md]}\n"+libraryOwnershipYAML+"scope: [{kind: software/Rule, property: lending/maxOpenLoans}]\nchecks:", 1)
	result := executionResult(t, content)
	original, ok := result.Document()
	if !ok {
		t.Fatalf("copy fixture invalid: %+v", result.Diagnostics())
	}
	// This stage preserves reference selections. Effective scope semantics belong
	// to organization context selection, not to this decoding observation.
	changed, _ := result.Document()
	changed.Instructions.Value[0].Files.Value[0] = "changed"
	changed.Instructions.Value[1].Package.Value = "changed"
	changed.Ownership.Value[0].Path.Value = "changed/"
	changed.Scope.Value[0].Value.Kind = model.Identity{Package: "other", ID: "other"}
	changed.Checks.Value[0].Args.Value[0] = "changed"
	changed.Checks.Value[0].RequiredBy.Value = model.Identity{Package: "other", ID: "other"}
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
	content := strings.Replace(documentedExecution(t), "timeout: 2m", "timeout: wrong", 1)
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
	if result.Partial().Checks.Value[0].Args.Value[0] != "test" {
		t.Fatal("partial known facts mutated or discarded")
	}
	var empty project.CapturedExecutionDecoding
	if _, ok := empty.Document(); ok || empty.Status() != model.NotCompleted {
		t.Fatal("zero result is usable")
	}
}
