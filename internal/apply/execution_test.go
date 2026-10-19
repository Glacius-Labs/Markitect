package apply_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func executionInputs(t *testing.T, content string) (apply.InputSelection, apply.RawInputs) {
	t.Helper()
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, ".markitect/execution.yaml", snapshot.Executable, []byte(content))
	location := apply.SourceLocation{Boundary: "explicit-execution-source", Path: path(t, "acquired/project-settings.data")}
	file, err := snapshot.NewEntry(location.Path, snapshot.Executable, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	selection.Execution = []apply.SourceLocation{location}
	raw.Execution = []apply.SourceFile{{Location: location, File: file, RepositoryPath: path(t, ".markitect/execution.yaml")}}
	return selection, raw
}

func executionDiagnostic(diagnostics []apply.ExecutionDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestDocumentedExecutionDecodesWithoutInventingResolvedSettings(t *testing.T) {
	content := documentedExecution(t)
	selection, raw := executionInputs(t, content)
	stage := registryModel(t, selection, raw)
	result := stage.DecodeExecution()
	document, ok := result.Document()
	if result.Status() != model.Passed || !ok || len(result.Diagnostics()) != 0 {
		t.Fatalf("documented execution failed: %s %+v", result.Status(), result.Diagnostics())
	}
	budget := document.Budget
	if document.APIVersion.Value != "markitect.execution/v1" || document.Strategy.Value != "shared-workspace-v1" || budget.ExecutionTime.Value != 25*time.Minute || budget.TurnTimeout.Value != 10*time.Minute || budget.AgentTurns.Value != 120 || budget.Rounds.Value != 4 {
		t.Fatal("declared budget changed")
	}
	if document.Defaults.Execute.Profile == nil || document.Defaults.Execute.Profile.Value != "library-agent" || document.Defaults.Execute.Profile.Source.Field != "/profiles/execute" || document.Defaults.Verify.Profile.Source.Field != "/profiles/verify" || document.Defaults.Execute.Instructions != nil || document.Defaults.Verify.Instructions != nil {
		t.Fatal("profile selection or instruction absence was invented")
	}
	if document.Packages == nil || len(document.Packages.Value) != 1 || document.Packages.Value[0].Package != "lending" || document.Packages.Value[0].Verify.Profile.Value != "library-agent" || document.Packages.Value[0].Execute.Profile != nil || document.Layout != nil || document.Scope != nil || document.Checks == nil || len(document.Checks.Value) != 1 || document.Checks.Value[0].RequiredBy.Value != (model.Identity{Package: "library", ID: "callable-interface"}) {
		t.Fatal("authored selections changed")
	}
	if budget.AgentTurns.Source.ContentID != raw.Execution[0].File.ContentID() || budget.AgentTurns.Source.File != ".markitect/execution.yaml" || budget.AgentTurns.Source.Field != "/budget/agentTurns" || !reflect.DeepEqual(result.Model().Inputs(), stage.Inputs()) || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
		t.Fatal("original source/binding changed")
	}
	source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
		return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: fixtureHead}, nil
	}}
	preview, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	prepared := preview.DecodeExecution()
	if prepared.Status() != model.Passed || prepared.Model().Inputs().Revision != fixtureHead {
		t.Fatalf("fixed preview decoding failed: %+v", prepared.Diagnostics())
	}
}

func TestExecutionFieldTypeFailureRetainsKnownAuthoredValues(t *testing.T) {
	content := strings.Replace(documentedExecution(t), "agentTurns: 120", "agentTurns: many", 1)
	selection, raw := executionInputs(t, content)
	result := registryModel(t, selection, raw).DecodeExecution()
	if _, ok := result.Document(); ok || result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), "execution.integer") {
		t.Fatalf("bad budget passed: %+v", result.Diagnostics())
	}
	partial := result.Partial()
	if partial.Budget.AgentTurns.Source.File != "" || partial.Budget.Rounds.Value != 4 || partial.Budget.TurnTimeout.Value != 10*time.Minute || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
		t.Fatal("known values or malformed original bytes discarded")
	}
	for _, diagnostic := range result.Diagnostics() {
		if diagnostic.Code == "execution.integer" && (diagnostic.Source.Field != "/budget/agentTurns" || diagnostic.Source.ContentID != raw.Execution[0].File.ContentID() || diagnostic.Position == nil) {
			t.Fatal("type error lost exact source coordinates")
		}
	}
}

const libraryLayoutYAML = `layout:
  roots: [realization]
  exceptions:
    - path: realization/cmd/library
      owner: library
      reason: The callable interface fixes the executable location.
`

func withLayout(t *testing.T, layout string) string {
	t.Helper()
	return strings.Replace(documentedExecution(t), "checks:\n", layout+"checks:\n", 1)
}

func TestExecutionLayoutDecodesWithSourcesAndIsValidatedAgainstTheModel(t *testing.T) {
	result := executionResult(t, withLayout(t, libraryLayoutYAML))
	document, ok := result.Document()
	if !ok || document.Layout == nil {
		t.Fatalf("layout failed: %s %+v", result.Status(), result.Diagnostics())
	}
	layout := document.Layout
	if len(layout.Roots.Value) != 1 || layout.Roots.Value[0].String() != "realization" || layout.Roots.Source.Field != "/layout/roots" || len(layout.Exceptions) != 1 {
		t.Fatalf("roots changed: %+v", layout)
	}
	exception := layout.Exceptions[0]
	if exception.Path.Value.String() != "realization/cmd/library" || exception.Path.Source.Field != "/layout/exceptions/0/path" || exception.Owner.Value != "library" || exception.Owner.Source.Field != "/layout/exceptions/0/owner" || exception.Reason.Source.Field != "/layout/exceptions/0/reason" {
		t.Fatalf("exception or its sources changed: %+v", exception)
	}
	empty := executionResult(t, withLayout(t, "layout:\n  roots: []\n"))
	if document, ok := empty.Document(); !ok || document.Layout == nil || len(document.Layout.Roots.Value) != 0 || document.Layout.Exceptions != nil {
		t.Fatalf("explicit empty roots lost: %+v", empty.Diagnostics())
	}
}

func TestExecutionLayoutRefusals(t *testing.T) {
	for _, tc := range []struct{ name, layout, code string }{
		{"unknown key", "layout:\n  roots: [realization]\n  area: x\n", "execution.unknown_key"},
		{"missing roots", "layout:\n  exceptions: []\n", "execution.field_missing"},
		{"roots not a list", "layout:\n  roots: realization\n", "execution.list"},
		{"exceptions not a list", "layout:\n  roots: []\n  exceptions: {}\n", "execution.list"},
		{"layout not a mapping", "layout: [realization]\n", "execution.mapping"},
		{"missing reason", "layout:\n  roots: []\n  exceptions:\n    - {path: realization/a, owner: library}\n", "execution.field_missing"},
		{"empty reason", "layout:\n  roots: []\n  exceptions:\n    - {path: realization/a, owner: library, reason: ''}\n", "execution.string"},
		{"escaping path", "layout:\n  roots: []\n  exceptions:\n    - {path: ../a, owner: library, reason: Why.}\n", "execution.layout_path"},
		{"unknown owner", "layout:\n  roots: []\n  exceptions:\n    - {path: realization/a, owner: missing, reason: Why.}\n", "execution.package"},
		{"imported owner", "layout:\n  roots: []\n  exceptions:\n    - {path: realization/a, owner: software, reason: Why.}\n", "execution.package"},
		{"root directory as root", "layout:\n  roots: [.]\n", "execution.layout"},
		{"nested roots", "layout:\n  roots: [realization, realization/lending]\n", "execution.layout"},
		{"root in markitect", "layout:\n  roots: [.markitect/x]\n", "execution.layout"},
		{"repeated exception", "layout:\n  roots: []\n  exceptions:\n    - {path: a/b, owner: library, reason: Why.}\n    - {path: a/b, owner: lending, reason: Why.}\n", "execution.layout"},
		{"roots differ by case", "layout:\n  roots: [realization, Realization]\n", "execution.layout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := executionResult(t, withLayout(t, tc.layout))
			if _, ok := result.Document(); ok || result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), tc.code) {
				t.Fatalf("invalid layout accepted or wrong diagnostic: %s %+v", result.Status(), result.Diagnostics())
			}
		})
	}
}
