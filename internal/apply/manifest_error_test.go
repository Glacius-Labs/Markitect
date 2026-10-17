package apply_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Actual lexical inspection, supplied through a controlled port. These partial
// bytes are not a complete model acquisition or physical preflight proof.
func partialManifest(t *testing.T, incomplete bool) (apply.InputSelection, apply.RawInputs, model.ManifestResult) {
	t.Helper()
	selection := apply.InputSelection{Repository: "controlled-repository", Local: apply.ModelSelection{
		Boundary: "selected/acquisition", ManifestPath: path(t, "nested/model.yaml"),
	}}
	content := []byte("formatVersion: 1\nid: example\nID: collision\nroot: model\ndependencies: {}\n")
	limits := model.Limits{}
	if incomplete {
		content = []byte("formatVersion: 1\nid: example\nroot: model\ndependencies: {}\n")
		limits.SourceBytes = 1
	}
	inspection := model.InspectManifest(model.ManifestInput{
		Origin: "opaque:inspection", ManifestPath: selection.Local.ManifestPath.String(), Bytes: content, Limits: limits,
	})
	file, err := snapshot.NewEntry(selection.Local.ManifestPath, snapshot.Regular, content)
	if err != nil {
		t.Fatal(err)
	}
	files, err := snapshot.NewTree([]snapshot.Entry{file})
	if err != nil {
		t.Fatal(err)
	}
	observed, err := snapshot.NewEntry(path(t, ".markitect/"+selection.Local.ManifestPath.String()), snapshot.Regular, content)
	if err != nil {
		t.Fatal(err)
	}
	census, err := snapshot.NewTree([]snapshot.Entry{observed})
	if err != nil {
		t.Fatal(err)
	}
	raw := apply.RawInputs{Census: census, Local: apply.ModelSource{
		Selection: selection.Local, RepositoryArea: area(t, ".markitect"), Files: files,
		Source: model.SourceModel{Origin: "opaque:inspection", ManifestPath: selection.Local.ManifestPath.String(), Entries: []model.Entry{
			{Path: selection.Local.ManifestPath.String(), Kind: model.Regular, Bytes: content},
		}},
	}}
	return selection, raw, inspection
}

func TestManifestPreflightFailureRetainsActualInspectionWithoutFullCompilation(t *testing.T) {
	for _, incomplete := range []bool{false, true} {
		for _, preview := range []bool{false, true} {
			t.Run(fmt.Sprintf("incomplete=%v/preview=%v", incomplete, preview), func(t *testing.T) {
				selection, raw, inspection := partialManifest(t, incomplete)
				want := model.Failed
				if incomplete {
					want = model.NotCompleted
				}
				if inspection.Status != want || inspection.Root != "" || len(inspection.Diagnostics) == 0 {
					t.Fatalf("actual inspection fixture: %+v", inspection)
				}
				failure := fmt.Errorf("source manifest preflight: %w", apply.NewManifestPreflightError(selection.Local, inspection))
				source := controlledSource{
					capture: func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) {
						return apply.CapturedInputs{Raw: raw, Head: fixtureHead}, failure
					},
					preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
						return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: fixtureHead}, failure
					},
				}
				service := apply.NewModelService(source, source)
				var actual apply.RawInputs
				var compiled model.Result
				var err error
				if preview {
					var result apply.PreviewedModel
					result, err = service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
					actual, compiled = result.Inputs().Raw, result.Compilation()
				} else {
					var result apply.CapturedModel
					result, err = service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
					actual, compiled = result.Inputs().Raw, result.Compilation()
				}
				var observed *apply.ManifestPreflightError
				if !errors.As(err, &observed) || observed.Selection() != selection.Local || !reflect.DeepEqual(observed.Inspection(), inspection) {
					t.Fatalf("inspection/coordinates lost: %v", err)
				}
				if !reflect.DeepEqual(actual, raw) || actual.Local.Root.String() != "" || len(actual.Local.Source.Entries) != 1 {
					t.Fatal("partial manifest observations replaced or root invented")
				}
				if compiled.Status != model.NotCompleted || len(compiled.Models) != 0 || len(compiled.Diagnostics) != 0 {
					t.Fatalf("partial acquisition was fully compiled: %+v", compiled)
				}
			})
		}
	}
}

func TestManifestPreflightErrorCopiesActualDiagnosticsAndPositions(t *testing.T) {
	selection, _, inspection := partialManifest(t, false)
	failure := apply.NewManifestPreflightError(selection.Local, inspection)
	before := failure.Inspection()
	inspection.Diagnostics[0].Message = "caller mutation"
	positions := 0
	for i := range inspection.Diagnostics {
		if inspection.Diagnostics[i].Position != nil {
			inspection.Diagnostics[i].Position.Line = -1
			positions++
		}
	}
	returned := failure.Inspection()
	returned.Diagnostics[0].Source.Path = "getter mutation"
	for i := range returned.Diagnostics {
		if returned.Diagnostics[i].Position != nil {
			returned.Diagnostics[i].Position.Line = -2
		}
	}
	if positions == 0 || !reflect.DeepEqual(failure.Inspection(), before) {
		t.Fatal("inspection diagnostics alias source or getter output")
	}
	if failure.Selection() != selection.Local || !strings.Contains(failure.Error(), "failed") || !strings.Contains(failure.Error(), selection.Local.ManifestPath.String()) {
		t.Fatal("selected inspection context missing")
	}
}

func TestPhysicalAcquisitionErrorIsDistinctFromManifestPreflight(t *testing.T) {
	selection, raw, _ := partialManifest(t, false)
	failure := &os.PathError{Op: "open", Path: "selected/acquisition/nested/model.yaml", Err: fs.ErrPermission}
	result, err := captureService(apply.CapturedInputs{Raw: raw, Head: fixtureHead}, failure).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	var manifestError *apply.ManifestPreflightError
	if !errors.Is(err, fs.ErrPermission) || errors.As(err, &manifestError) || result.Compilation().Status != model.NotCompleted || !reflect.DeepEqual(result.Inputs().Raw, raw) {
		t.Fatal("physical error was converted to language diagnosis or lost observations")
	}
}
