package apply_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestExecutionRequiresExactlyOneActualProjectSource(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		status     model.Status
	}{
		{"absent", "execution.source_uncaptured", model.NotCompleted},
		{"duplicate", "execution.source_count", model.Failed},
		{"wrong path", "execution.source_path", model.Failed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := executionInputs(t, documentedExecution(t))
			switch tc.name {
			case "absent":
				selection.Execution, raw.Execution = nil, nil
			case "duplicate":
				selection.Execution = append(selection.Execution, selection.Execution[0])
				raw.Execution = append(raw.Execution, raw.Execution[0])
			case "wrong path":
				raw.Execution[0].RepositoryPath = path(t, ".markitect/other.yaml")
				censusFile(t, &raw, ".markitect/other.yaml", snapshot.Executable, raw.Execution[0].File.Bytes())
			}
			stage := registryModel(t, selection, raw)
			result := stage.DecodeExecution()
			if _, ok := result.Document(); ok || result.Status() != tc.status || !executionDiagnostic(result.Diagnostics(), tc.code) || !reflect.DeepEqual(result.Model().Inputs(), stage.Inputs()) {
				t.Fatalf("source boundary lost: %s %+v", result.Status(), result.Diagnostics())
			}
		})
	}
}

func TestExecutionCannotBypassParentSourceDriftOrUnfinishedCheck(t *testing.T) {
	for _, name := range []string{"bytes", "mode", "Model check limit", "unavailable dependency"} {
		t.Run(name, func(t *testing.T) {
			selection, raw := executionInputs(t, documentedExecution(t))
			limits := model.Limits{}
			switch name {
			case "bytes", "mode":
				content, mode := raw.Execution[0].File.Bytes(), snapshot.Executable
				if name == "bytes" {
					content = append(content, '\n')
				} else {
					mode = snapshot.Regular
				}
				entry, err := snapshot.NewEntry(raw.Execution[0].Location.Path, mode, content)
				if err != nil {
					t.Fatal(err)
				}
				raw.Execution[0].File = entry
			case "Model check limit":
				limits.Entries = 1
			case "unavailable dependency":
				selection.Dependencies, raw.Dependencies = nil, nil
			}
			stage, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, limits)
			if (name == "bytes" || name == "mode") && !errors.Is(err, apply.ErrIncompleteInputs) {
				t.Fatalf("source drift escaped acquisition: %v", err)
			}
			if stage.Checked().Status == model.Passed {
				t.Fatal("incomplete parent unexpectedly passed")
			}
			result := stage.DecodeExecution()
			if _, ok := result.Document(); ok || result.Status() != model.NotCompleted || !executionDiagnostic(result.Diagnostics(), "execution.model_not_passed") || !reflect.DeepEqual(result.Model().Inputs().Raw, raw) {
				t.Fatalf("nonpassing parent promoted or lost: %s %+v", result.Status(), result.Diagnostics())
			}
		})
	}
	var zero apply.CapturedModel
	if zero.DecodeExecution().Status() != model.NotCompleted {
		t.Fatal("zero model decoded as usable execution")
	}
	var preview apply.PreviewedModel
	if preview.DecodeExecution().Status() != model.NotCompleted {
		t.Fatal("zero preview decoded as usable execution")
	}
}
