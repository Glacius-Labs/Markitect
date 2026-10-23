package project_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func captured(raw project.RawInputs) project.CapturedInputs {
	return project.CapturedInputs{Raw: raw, Head: fixtureHead, Guard: "controlled-comparison-token"}
}

func captureService(observed project.CapturedInputs, failure error) *project.ModelService {
	source := controlledSource{capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
		return observed, failure
	}}
	return project.NewModelService(source, nil)
}

// Synchronize deliberate Model check fixture edits with their raw file views.
// Keep every source Entry, including duplicates, for the real Model check.
func refreshLocalFiles(t *testing.T, raw *project.RawInputs) {
	t.Helper()
	var files, census []snapshot.Entry
	seen := make(map[string]bool)
	for _, entry := range raw.Local.Source.Entries {
		if entry.Kind != model.Regular || seen[entry.Path] {
			continue
		}
		seen[entry.Path] = true
		mode := snapshot.Regular
		if old, ok := raw.Local.Files.Lookup(path(t, entry.Path)); ok {
			mode = old.Mode()
		}
		file, err := snapshot.NewEntry(path(t, entry.Path), mode, entry.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
		p := entry.Path
		if !raw.Local.RepositoryArea.IsRoot() {
			p = raw.Local.RepositoryArea.String() + "/" + p
		}
		file, err = snapshot.NewEntry(path(t, p), mode, entry.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		census = append(census, file)
	}
	for _, entry := range raw.Census.Entries() {
		if !raw.Local.RepositoryArea.Contains(entry.Path()) {
			census = append(census, entry)
		}
	}
	var err error
	raw.Local.Files, err = snapshot.NewTree(files)
	if err != nil {
		t.Fatal(err)
	}
	raw.Census, err = snapshot.NewTree(census)
	if err != nil {
		t.Fatal(err)
	}
}

func hasDiagnostic(result model.Result, prefix string) bool {
	for _, diagnostic := range result.Diagnostics {
		if strings.HasPrefix(diagnostic.Code, prefix) {
			return true
		}
	}
	return false
}

func TestLanguageFailureAndModelCheckNotCompletedAreNotAcquisitionErrors(t *testing.T) {
	for _, kind := range []string{"syntax", "duplicate", "budget"} {
		t.Run(kind, func(t *testing.T) {
			selection, raw := libraryInputs(t)
			want, code, limits := model.Failed, "yaml.", model.Limits{}
			switch kind {
			case "syntax":
				for i := range raw.Local.Source.Entries {
					if strings.HasSuffix(raw.Local.Source.Entries[i].Path, "/borrow.yaml") {
						raw.Local.Source.Entries[i].Bytes = []byte("id: [unterminated\n")
					}
				}
				refreshLocalFiles(t, &raw)
			case "duplicate":
				raw.Local.Source.Entries = append(raw.Local.Source.Entries, raw.Local.Source.Entries[0])
				code = "input.duplicate_path"
			case "budget":
				limits = model.Limits{Entries: 1}
				want, code = model.NotCompleted, "limit.entries"
			}
			result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, limits)
			checked := result.Checked()
			if err != nil || checked.Status != want || !hasDiagnostic(checked, code) {
				t.Fatalf("%v %+v", err, checked)
			}
			if !reflect.DeepEqual(raw, result.Inputs().Raw) {
				t.Fatal("original Model check input lost")
			}
		})
	}
}

func TestIncompleteOrMismatchedCaptureCannotPassModel(t *testing.T) {
	for _, kind := range []string{"zero", "head", "guard", "local area", "local selection", "root", "dependency missing", "dependency order", "manifest", "model files", "census", "mode"} {
		t.Run(kind, func(t *testing.T) {
			selection, raw := libraryInputs(t)
			observation := captured(raw)
			switch kind {
			case "zero":
				observation = project.CapturedInputs{}
			case "head":
				observation.Head = ""
			case "guard":
				observation.Guard = ""
			case "local area":
				observation.Raw.Local.RepositoryArea = snapshot.DirectoryBoundary{}
			case "local selection":
				observation.Raw.Local.Selection.Boundary = "different"
			case "root":
				observation.Raw.Local.Root = snapshot.DirectoryBoundary{}
			case "dependency missing":
				observation.Raw.Dependencies = nil
			case "dependency order":
				observation.Raw.Dependencies[0], observation.Raw.Dependencies[1] = observation.Raw.Dependencies[1], observation.Raw.Dependencies[0]
			case "manifest":
				observation.Raw.Local.Source.ManifestPath = "other/model.yaml"
			case "model files":
				observation.Raw.Local.Files = snapshot.Tree{}
			case "census":
				observation.Raw.Census = snapshot.Tree{}
			case "mode":
				entries := observation.Raw.Census.Entries()
				first := entries[0]
				entries[0], _ = snapshot.NewEntry(first.Path(), snapshot.Executable, first.Bytes())
				observation.Raw.Census, _ = snapshot.NewTree(entries)
			}
			result, err := captureService(observation, nil).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
			if !errors.Is(err, project.ErrIncompleteInputs) || result.Checked().Status != model.NotCompleted || !reflect.DeepEqual(result.Inputs(), observation) {
				t.Fatalf("%v %+v", err, result.Checked())
			}
		})
	}
}

func TestPreviewErrorAndRevisionBindingRetainPartialObservations(t *testing.T) {
	for _, kind := range []string{"error", "fixed unresolved", "fixed drift", "draft carries revision", "missing head"} {
		t.Run(kind, func(t *testing.T) {
			selection, raw := libraryInputs(t)
			request := project.PreviewRequest{Selection: selection}
			observation := project.PreviewedInputs{Raw: raw, Head: fixtureHead}
			var failure error
			switch kind {
			case "error":
				failure = errors.New("observed partial preview")
			case "fixed unresolved":
				request.Revision = fixtureHead
			case "fixed drift":
				request.Revision = fixtureHead
				observation.Revision = "712c4b884f3464d5569572c34207a827c2d7f52e"
			case "draft carries revision":
				observation.Revision = fixtureHead
			case "missing head":
				observation.Head = ""
			}
			source := controlledSource{preview: func(context.Context, project.PreviewRequest) (project.PreviewedInputs, error) {
				return observation, failure
			}}
			result, err := project.NewModelService(nil, source).PreviewModel(context.Background(), request, model.Limits{})
			if err == nil || result.Checked().Status != model.NotCompleted || !reflect.DeepEqual(result.Inputs(), observation) {
				t.Fatalf("%v %+v", err, result.Checked())
			}
			if failure != nil && !errors.Is(err, failure) {
				t.Fatal("port error replaced")
			}
		})
	}
}

func TestMissingPortsAndCancellationNeverCheck(t *testing.T) {
	selection, raw := libraryInputs(t)
	request := project.CaptureRequest{Selection: selection}
	for _, service := range []*project.ModelService{nil, project.NewModelService(nil, nil)} {
		result, err := service.CaptureModel(context.Background(), request, model.Limits{})
		if !errors.Is(err, project.ErrSourceUnavailable) || result.Checked().Status != model.NotCompleted {
			t.Fatal(err)
		}
		preview, err := service.PreviewModel(context.Background(), project.PreviewRequest{Selection: selection}, model.Limits{})
		if !errors.Is(err, project.ErrSourceUnavailable) || preview.Checked().Status != model.NotCompleted {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	source := controlledSource{capture: func(got context.Context, _ project.CaptureRequest) (project.CapturedInputs, error) {
		if got != ctx {
			t.Fatal("context changed")
		}
		calls++
		cancel()
		return captured(raw), nil
	}}
	service := project.NewModelService(source, nil)
	result, err := service.CaptureModel(ctx, request, model.Limits{})
	if !errors.Is(err, context.Canceled) || result.Inputs().Head != fixtureHead || result.Checked().Status != model.NotCompleted {
		t.Fatal("cancellation lost observations")
	}
	_, err = service.CaptureModel(ctx, request, model.Limits{})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("already canceled request dispatched")
	}
}
