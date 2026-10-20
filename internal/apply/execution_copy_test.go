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
	content = strings.Replace(content, "profiles:\n  execute", "instructions: {execute: [one.md], verify: [two.md]}\nprofiles:\n  execute", 1)
	content = strings.Replace(content, "    profiles: {verify: library-agent}", "    profiles: {execute: a, verify: b}\n    instructions: {execute: [three.md], verify: [four.md]}", 1)
	content = strings.Replace(content, "checks:\n", libraryOwnershipYAML+libraryTeamYAML+"scope: [{kind: software/Rule, property: lending/maxOpenLoans}]\nchecks:\n", 1)
	result := executionResult(t, content)
	original, ok := result.Document()
	if !ok {
		t.Fatalf("copy fixture invalid: %+v", result.Diagnostics())
	}
	// This stage preserves reference selections. Effective scope semantics belong
	// to organization context selection, not to this decoding observation.
	changed, _ := result.Document()
	changed.Defaults.Execute.Profile.Value = "changed"
	changed.Defaults.Verify.Profile.Value = "changed"
	changed.Defaults.Execute.Instructions.Value[0] = "changed"
	changed.Defaults.Verify.Instructions.Value[0] = "changed"
	p := &changed.Packages.Value[0]
	p.Execute.Profile.Value, p.Verify.Profile.Value = "changed", "changed"
	p.Execute.Instructions.Value[0], p.Verify.Instructions.Value[0] = "changed", "changed"
	changed.Ownership.Value[0].Path.Value = "changed/"
	changed.Ownership.Value[4].Owner.Value = "changed"
	changed.Team.Helpers[0].Instructions.Value[0] = "changed"
	changed.Team.Helpers[0].Profile.Value = "changed"
	changed.Team.Helpers[1].Writes.Value = false
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
	content := strings.Replace(documentedExecution(t), "agentTurns: 120", "agentTurns: wrong", 1)
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
	if result.Partial().Checks.Value[0].Args.Value[0] != "test" || result.Partial().Budget.Passes.Value != 12 {
		t.Fatal("partial known facts mutated or discarded")
	}
	var empty apply.CapturedExecutionDecoding
	if _, ok := empty.Document(); ok || empty.Status() != model.NotCompleted {
		t.Fatal("zero result is usable")
	}
}
