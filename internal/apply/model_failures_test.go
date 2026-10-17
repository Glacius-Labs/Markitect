package apply_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func captured(raw apply.RawInputs) apply.CapturedInputs {
	return apply.CapturedInputs{Raw: raw, Head: fixtureHead, Guard: "controlled-comparison-token"}
}

func captureService(observed apply.CapturedInputs, failure error) *apply.ModelService {
	source := controlledSource{capture: func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) { return observed, failure }}
	return apply.NewModelService(source, nil)
}

// Synchronize deliberate compiler fixture edits with their raw file views.
// Keep every source Entry, including duplicates, for the real compiler.
func refreshLocalFiles(t *testing.T, raw *apply.RawInputs) {
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
		if !raw.LocalModelArea.IsRoot() {
			p = raw.LocalModelArea.String() + "/" + p
		}
		file, err = snapshot.NewEntry(path(t, p), mode, entry.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		census = append(census, file)
	}
	for _, entry := range raw.Census.Entries() {
		if !raw.LocalModelArea.Contains(entry.Path()) {
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

func TestLanguageFailureAndCompilerNotCompletedAreNotAcquisitionErrors(t *testing.T) {
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
			result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, limits)
			compiled := result.Compilation()
			if err != nil || compiled.Status != want || !hasDiagnostic(compiled, code) {
				t.Fatalf("%v %+v", err, compiled)
			}
			if !reflect.DeepEqual(raw, result.Inputs().Raw) {
				t.Fatal("original compiler input lost")
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
				observation = apply.CapturedInputs{}
			case "head":
				observation.Head = ""
			case "guard":
				observation.Guard = ""
			case "local area":
				observation.Raw.LocalModelArea = snapshot.DirectoryBoundary{}
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
			result, err := captureService(observation, nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
			if !errors.Is(err, apply.ErrIncompleteInputs) || result.Compilation().Status != model.NotCompleted || !reflect.DeepEqual(result.Inputs(), observation) {
				t.Fatalf("%v %+v", err, result.Compilation())
			}
		})
	}
}

func TestPreviewErrorAndRevisionBindingRetainPartialObservations(t *testing.T) {
	for _, kind := range []string{"error", "fixed unresolved", "fixed drift", "draft carries revision", "missing head"} {
		t.Run(kind, func(t *testing.T) {
			selection, raw := libraryInputs(t)
			request := apply.PreviewRequest{Selection: selection}
			observation := apply.PreviewedInputs{Raw: raw, Head: fixtureHead}
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
			source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
				return observation, failure
			}}
			result, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), request, model.Limits{})
			if err == nil || result.Compilation().Status != model.NotCompleted || !reflect.DeepEqual(result.Inputs(), observation) {
				t.Fatalf("%v %+v", err, result.Compilation())
			}
			if failure != nil && !errors.Is(err, failure) {
				t.Fatal("port error replaced")
			}
		})
	}
}

func TestMissingPortsAndCancellationNeverCompile(t *testing.T) {
	selection, raw := libraryInputs(t)
	request := apply.CaptureRequest{Selection: selection}
	for _, service := range []*apply.ModelService{nil, apply.NewModelService(nil, nil)} {
		result, err := service.CaptureModel(context.Background(), request, model.Limits{})
		if !errors.Is(err, apply.ErrSourceUnavailable) || result.Compilation().Status != model.NotCompleted {
			t.Fatal(err)
		}
		preview, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
		if !errors.Is(err, apply.ErrSourceUnavailable) || preview.Compilation().Status != model.NotCompleted {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	source := controlledSource{capture: func(got context.Context, _ apply.CaptureRequest) (apply.CapturedInputs, error) {
		if got != ctx {
			t.Fatal("context changed")
		}
		calls++
		cancel()
		return captured(raw), nil
	}}
	service := apply.NewModelService(source, nil)
	result, err := service.CaptureModel(ctx, request, model.Limits{})
	if !errors.Is(err, context.Canceled) || result.Inputs().Head != fixtureHead || result.Compilation().Status != model.NotCompleted {
		t.Fatal("cancellation lost observations")
	}
	_, err = service.CaptureModel(ctx, request, model.Limits{})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("already canceled request dispatched")
	}
}
