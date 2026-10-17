package apply_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type controlledSource struct {
	capture func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error)
	preview func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error)
}

func (s controlledSource) Capture(ctx context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
	return s.capture(ctx, request)
}
func (s controlledSource) CapturePreview(ctx context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
	return s.preview(ctx, request)
}

func path(t *testing.T, value string) snapshot.Path {
	t.Helper()
	result, err := snapshot.NewPath(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func area(t *testing.T, value string) snapshot.DirectoryBoundary {
	t.Helper()
	result, err := snapshot.DirectoryAt(path(t, value))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// Read existing language fixture bytes into a controlled source. This is not a
// repository adapter or evidence of hardened acquisition, clean HEAD or guards.
func fixtureModel(t *testing.T, directory, boundary string) apply.ModelSource {
	t.Helper()
	result := apply.ModelSource{
		Selection: apply.ModelSelection{Boundary: boundary, ManifestPath: path(t, "model.yaml")},
		Root:      area(t, "model"),
		Source:    model.SourceModel{Origin: "opaque:" + boundary, ManifestPath: "model.yaml"},
	}
	var files []snapshot.Entry
	err := fs.WalkDir(os.DirFS(directory), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		entry := model.Entry{Path: p, Kind: model.Directory}
		if !d.IsDir() {
			entry.Kind = model.Regular
			entry.Bytes, err = os.ReadFile(filepath.Join(directory, filepath.FromSlash(p)))
			if err != nil {
				return err
			}
			file, err := snapshot.NewEntry(path(t, p), snapshot.Regular, entry.Bytes)
			if err != nil {
				return err
			}
			files = append(files, file)
		}
		result.Source.Entries = append(result.Source.Entries, entry)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	result.Files, err = snapshot.NewTree(files)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func libraryInputs(t *testing.T) (apply.InputSelection, apply.RawInputs) {
	t.Helper()
	base := "../../docs/design/canonical-model"
	raw := apply.RawInputs{
		LocalModelArea: area(t, ".markitect"),
		Local:          fixtureModel(t, base+"/library-example/m1", ".markitect"),
		Dependencies: []apply.ModelSource{
			fixtureModel(t, base+"/library-example/vocabulary", "selected/vocabulary"),
			fixtureModel(t, base+"/minimal-language/exchange-example/dependencies/core", "selected/core"),
		},
	}
	var files []snapshot.Entry
	for _, entry := range raw.Local.Files.Entries() {
		file, err := snapshot.NewEntry(path(t, ".markitect/"+entry.Path().String()), entry.Mode(), entry.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	var err error
	raw.Census, err = snapshot.NewTree(files)
	if err != nil {
		t.Fatal(err)
	}
	selection := apply.InputSelection{Repository: "controlled-repository", Local: raw.Local.Selection}
	for _, dependency := range raw.Dependencies {
		selection.Dependencies = append(selection.Dependencies, dependency.Selection)
	}
	return selection, raw
}

const fixtureHead = "e96eef6b9751298fd930f7fcd43e00a251a0d86e"

func TestCaptureAndPreviewCompileActualM1ThroughSeparatePorts(t *testing.T) {
	selection, raw := libraryInputs(t)
	captures, previews := 0, 0
	source := controlledSource{
		capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
			captures++
			if !reflect.DeepEqual(request.Selection, selection) {
				t.Fatal("capture selection changed")
			}
			return apply.CapturedInputs{Raw: raw, Head: fixtureHead, Guard: "opaque-comparison"}, nil
		},
		preview: func(_ context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
			previews++
			return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: request.Revision}, nil
		},
	}
	service := apply.NewModelService(source, source)
	captured, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || captured.Compilation().Status != model.Passed {
		t.Fatalf("capture: %v %+v", err, captured.Compilation())
	}
	if captures != 1 || previews != 0 || captured.Inputs().Guard != "opaque-comparison" {
		t.Fatal("capture/preview authority mixed")
	}
	for _, revision := range []string{"", "712c4b884f3464d5569572c34207a827c2d7f52e"} {
		preview, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
		if err != nil || preview.Compilation().Status != model.Passed {
			t.Fatalf("preview: %v %+v", err, preview.Compilation())
		}
		if preview.Inputs().Revision != revision || preview.Inputs().Head != fixtureHead || !preview.Inputs().Raw.Census.Equal(raw.Census) {
			t.Fatal("preview binding changed")
		}
	}
	if captures != 1 || previews != 2 {
		t.Fatal("unexpected port calls")
	}
	local := captured.Compilation().Models[0]
	if local.ID != "small-library" || len(local.Definitions) != 22 || len(local.Packages) != 5 {
		t.Fatalf("actual M1 not compiled: %+v", local)
	}
}

func TestAcquisitionFailurePreservesKnownInputsWithoutPassingModel(t *testing.T) {
	selection, raw := libraryInputs(t)
	failure := errors.New("capture interrupted after observations")
	source := controlledSource{capture: func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) {
		return apply.CapturedInputs{Raw: raw, Head: fixtureHead}, failure
	}}
	result, err := apply.NewModelService(source, nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if !errors.Is(err, failure) || result.Compilation().Status != model.NotCompleted || !reflect.DeepEqual(result.Inputs().Raw, raw) || result.Inputs().Head != fixtureHead {
		t.Fatalf("partial observations lost or compilation asserted: %v %+v", err, result.Compilation())
	}
}
