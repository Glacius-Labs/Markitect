package apply_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

// This controlled port makes a fresh observation for exactly each port request.
// Census bytes alone never stand in for an unselected SourceFile observation.
func preparationObservation(t *testing.T, selection apply.InputSelection, actual apply.RawInputs) apply.RawInputs {
	t.Helper()
	selectFiles := func(locations []apply.SourceLocation, existing []apply.SourceFile) []apply.SourceFile {
		var files []apply.SourceFile
		for _, location := range locations {
			var source apply.SourceFile
			for _, candidate := range existing {
				if candidate.Location == location {
					source = candidate
					break
				}
			}
			if source.Location.Boundary == "" {
				if location.Boundary != selection.Repository {
					t.Fatalf("invented boundary: %+v", location)
				}
				entry, found := actual.Census.Lookup(location.Path)
				if !found {
					t.Fatalf("requested unobserved file: %+v", location)
				}
				source = apply.SourceFile{Location: location, File: entry, RepositoryPath: location.Path}
			}
			files = append(files, source)
		}
		return files
	}
	raw := actual
	raw.Execution = selectFiles(selection.Execution, actual.Execution)
	raw.Registry = selectFiles(selection.Registry, actual.Registry)
	raw.Ignore = selectFiles(selection.Ignore, actual.Ignore)
	raw.Instructions = selectFiles(selection.Instructions, actual.Instructions)
	raw.HostProfiles = selectFiles(selection.HostProfiles, actual.HostProfiles)
	return raw
}

func TestPreparationM1UsesOneCompleteAcquisitionOrOneNewFullCapture(t *testing.T) {
	for _, complete := range []bool{true, false} {
		t.Run(fmt.Sprint(complete), func(t *testing.T) {
			selection, actual := fileCheckInputs(t)
			if !complete {
				selection.Execution = nil
				selection.Registry = nil
			}
			request := apply.CaptureRequest{Selection: selection}
			var requests []apply.CaptureRequest
			source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
				requests = append(requests, request)
				return apply.CapturedInputs{Raw: preparationObservation(t, request.Selection, actual), Head: fixtureHead, Guard: apply.PublicationGuard(fmt.Sprintf("new-capture-%d", len(requests)))}, nil
			}}
			result, err := apply.NewPreparationService(apply.NewModelService(source, nil)).Capture(context.Background(), request, model.Limits{})
			prepared, available := result.Prepared()
			if err != nil || result.Status() != model.Passed || !available {
				t.Fatalf("M1 preparation: %v %s %+v", err, result.Status(), result.Attempts())
			}
			wantCalls := 1
			if !complete {
				wantCalls = 2
			}
			attempts := result.Attempts()
			if len(requests) != wantCalls || len(attempts) != wantCalls || !reflect.DeepEqual(result.Request(), request) {
				t.Fatal("original request or finite acquisition count changed")
			}
			if prepared.Model().Inputs().Guard != apply.PublicationGuard(fmt.Sprintf("new-capture-%d", wantCalls)) {
				t.Fatal("final preparation extended an old guard")
			}
			accounting, ran := prepared.Files().Accounting()
			contexts, selected := prepared.Contexts().Selections()
			if !ran || accounting.Status() != realization.Accounted || accounting.TreeID() != actual.Census.Digest() || !selected || len(contexts) != 6 {
				t.Fatal("real M1 accounting/static selections were not prepared")
			}
			if !reflect.DeepEqual(prepared.Files().RegistryPreparation().Model().Inputs(), prepared.Model().Inputs()) || !reflect.DeepEqual(prepared.Contexts().Execution().Model().Inputs(), prepared.Model().Inputs()) {
				t.Fatal("stage acquisition bindings diverged")
			}
			if !complete {
				if len(attempts[0].Model().Inputs().Raw.Registry) != 0 || attempts[0].Registry().Status() != model.NotCompleted || attempts[0].Model().Inputs().Guard != "new-capture-1" {
					t.Fatal("first observation was augmented")
				}
				if len(requests[1].Selection.Registry) != 2 || requests[1].Selection.Execution[0].Boundary != selection.Repository {
					t.Fatal("recapture did not select exact repository files")
				}
			}
		})
	}
}

func TestPreparationZeroValuesExposeNoPreparedStage(t *testing.T) {
	var capture apply.CapturedPreparation
	var preview apply.PreviewPreparation
	if _, ok := capture.Prepared(); ok || capture.Status() != model.NotCompleted {
		t.Fatal("zero capture prepared")
	}
	if _, ok := preview.Prepared(); ok || preview.Status() != model.NotCompleted {
		t.Fatal("zero preview prepared")
	}
}
