package apply_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Use the original documented YAML bytes, including their original line endings.
func documentedExecution(t *testing.T) string {
	t.Helper()
	text, err := os.ReadFile("../../docs/design/canonical-model/manager-apply-configuration.md")
	if err != nil {
		t.Fatal(err)
	}
	start := bytes.Index(text, []byte("```yaml"))
	if start < 0 {
		t.Fatal("execution example missing")
	}
	start += bytes.IndexByte(text[start:], '\n') + 1
	end := bytes.Index(text[start:], []byte("```"))
	if end < 0 {
		t.Fatal("execution example is not closed")
	}
	return string(text[start : start+end])
}

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
	if document.APIVersion.Value != "markitect.execution/v1" || document.Strategy.Value != "recursive-apply-v1" || document.Limits.MaxRuns.Value != 5 || document.Limits.MaxExecutionTime.Value != 16*time.Hour || document.Assignment.ExecutionTime.Value != 8*time.Hour || document.CompletionReserve.Steps.Value != 40 {
		t.Fatal("declared caps changed")
	}
	if document.Defaults.Manager.Profile == nil || document.Defaults.Manager.Profile.Value != "library-agent" || document.Defaults.Manager.Profile.Source.Field != "/defaults/profiles/manager" || document.Defaults.Manager.Instructions != nil {
		t.Fatal("profile selection or instruction absence was invented")
	}
	if document.Packages == nil || len(document.Packages.Value) != 1 || document.Packages.Value[0].Package != "lending" || document.Packages.Value[0].StepTimeout.Value != 8*time.Minute || document.Scope != nil || document.Checks == nil || len(document.Checks.Value) != 1 || document.Checks.Value[0].RequiredBy.Value != (model.Identity{Package: "library", ID: "callable-interface"}) {
		t.Fatal("authored selections changed")
	}
	if document.OwnerApproval == nil || len(document.OwnerApproval.Value) != 0 || document.Defaults.BatchNoChangeReviews == nil || document.Defaults.BatchNoChangeReviews.Value {
		t.Fatal("explicit false/empty selections lost")
	}
	if document.Limits.MaxRuns.Source.ContentID != raw.Execution[0].File.ContentID() || document.Limits.MaxRuns.Source.File != ".markitect/execution.yaml" || !reflect.DeepEqual(result.Model().Inputs(), stage.Inputs()) || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
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
	content := strings.Replace(documentedExecution(t), "maxRuns: 5", "maxRuns: five", 1)
	selection, raw := executionInputs(t, content)
	result := registryModel(t, selection, raw).DecodeExecution()
	if _, ok := result.Document(); ok || result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), "execution.integer") {
		t.Fatalf("bad cap passed: %+v", result.Diagnostics())
	}
	partial := result.Partial()
	if partial.Limits.MaxRuns.Source.File != "" || partial.Limits.MaxSteps.Value != 320 || partial.Defaults.StepTimeout.Value != 10*time.Minute || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
		t.Fatal("known values or malformed original bytes discarded")
	}
	for _, diagnostic := range result.Diagnostics() {
		if diagnostic.Code == "execution.integer" && (diagnostic.Source.Field != "/limits/maxRuns" || diagnostic.Source.ContentID != raw.Execution[0].File.ContentID() || diagnostic.Position == nil) {
			t.Fatal("type error lost exact source coordinates")
		}
	}
}
