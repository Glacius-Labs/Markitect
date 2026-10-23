package gitrepo

import (
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

// Record calls and original results only; all acquisition is performed by the
// same actual Git adapter. No source, classification or accounting is invented.
type recordingPreparationGit struct {
	repository      *Repository
	captureRequests []apply.CaptureRequest
	captures        []apply.CapturedInputs
	previewRequests []apply.PreviewRequest
	previews        []apply.PreviewedInputs
}

func (r *recordingPreparationGit) Capture(ctx context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
	inputs, err := r.repository.Capture(ctx, request)
	r.captureRequests = append(r.captureRequests, request)
	r.captures = append(r.captures, inputs)
	return inputs, err
}

func (r *recordingPreparationGit) CapturePreview(ctx context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
	inputs, err := r.repository.CapturePreview(ctx, request)
	r.previewRequests = append(r.previewRequests, request)
	r.previews = append(r.previews, inputs)
	return inputs, err
}

func TestM1GitPreparationAcquiresCompleteClosureWithoutExtendingEarlierInputs(t *testing.T) {
	_, completeSelection, head := fileCheckSeamFixture(t)
	for _, complete := range []bool{false, true} {
		name := "derived"
		if complete {
			name = "complete"
		}
		for _, mode := range []string{"capture", "working", "fixed"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				selection := completeSelection
				wantCalls := 1
				if !complete {
					selection.Execution, selection.Registry, selection.Ignore, selection.Instructions = nil, nil, nil, nil
					wantCalls = 2
				}
				port := &recordingPreparationGit{repository: adapter(t)}
				service := apply.NewPreparationService(apply.NewModelService(port, port))
				var raw apply.RawInputs
				var checked model.Result
				var execution apply.AuthoredExecution
				var accounting realization.Accounting
				var contexts []organization.ContextSelection
				if mode == "capture" {
					request := apply.CaptureRequest{Selection: selection}
					result, err := service.Capture(context.Background(), request, model.Limits{})
					prepared, available := result.Prepared()
					if err != nil || result.Status() != model.Passed || !available || !reflect.DeepEqual(result.Request(), request) || len(port.captures) != wantCalls || len(port.previews) != 0 || len(result.Attempts()) != wantCalls {
						t.Fatalf("real capture preparation failed: %s %v", result.Status(), err)
					}
					if complete && !reflect.DeepEqual(port.captureRequests[0], request) {
						t.Fatal("complete explicit capture selection changed before acquisition")
					}
					for i, attempt := range result.Attempts() {
						if !reflect.DeepEqual(attempt.Model().Request(), port.captureRequests[i]) || !reflect.DeepEqual(attempt.Model().Inputs(), port.captures[i]) || attempt.Error() != nil {
							t.Fatal("attempt changed its actual full port request or observation")
						}
					}
					last := port.captures[wantCalls-1]
					// Guard values are opaque and may be equal on the same basis.
					if last.Head != head || last.Guard == "" || !reflect.DeepEqual(prepared.Model().Inputs(), last) || !reflect.DeepEqual(prepared.Model().Request(), port.captureRequests[wantCalls-1]) {
						t.Fatal("Prepared does not retain the last complete actual capture")
					}
					if !reflect.DeepEqual(prepared.Files().RegistryPreparation().Model().Inputs(), last) || !reflect.DeepEqual(prepared.Contexts().Execution().Model().Inputs(), last) {
						t.Fatal("bound file/context stages changed the final capture")
					}
					raw, checked = last.Raw, prepared.Model().Checked()
					execution, _ = prepared.Execution().Document()
					accounting, _ = prepared.Files().Accounting()
					contexts, _ = prepared.Contexts().Selections()
					if !complete {
						assertOriginalPreparationGap(t, port.captures[0].Raw, raw)
					}
				} else {
					request := apply.PreviewRequest{Selection: selection}
					if mode == "fixed" {
						request.Revision = head
					}
					result, err := service.Preview(context.Background(), request, model.Limits{})
					prepared, available := result.Prepared()
					if err != nil || result.Status() != model.Passed || !available || !reflect.DeepEqual(result.Request(), request) || len(port.previews) != wantCalls || len(port.captures) != 0 || len(result.Attempts()) != wantCalls {
						t.Fatalf("real preview preparation failed: %s %v", result.Status(), err)
					}
					if complete && !reflect.DeepEqual(port.previewRequests[0], request) {
						t.Fatal("complete explicit preview selection changed before acquisition")
					}
					for i, attempt := range result.Attempts() {
						if !reflect.DeepEqual(attempt.Model().Request(), port.previewRequests[i]) || !reflect.DeepEqual(attempt.Model().Inputs(), port.previews[i]) || attempt.Error() != nil || port.previewRequests[i].Revision != request.Revision || port.previews[i].Revision != request.Revision {
							t.Fatal("preview attempt changed its own request, observation or fixed revision")
						}
					}
					last := port.previews[wantCalls-1]
					if last.Head != head || !reflect.DeepEqual(prepared.Model().Inputs(), last) || !reflect.DeepEqual(prepared.Model().Request(), port.previewRequests[wantCalls-1]) {
						t.Fatal("Prepared does not retain the last complete actual preview")
					}
					if !reflect.DeepEqual(prepared.Files().RegistryPreparation().Model().Inputs(), last) || !reflect.DeepEqual(prepared.Contexts().Execution().Model().Inputs(), last) {
						t.Fatal("bound file/context stages changed the final preview")
					}
					raw, checked = last.Raw, prepared.Model().Checked()
					execution, _ = prepared.Execution().Document()
					accounting, _ = prepared.Files().Accounting()
					contexts, _ = prepared.Contexts().Selections()
					if !complete {
						assertOriginalPreparationGap(t, port.previews[0].Raw, raw)
					}
				}
				if checked.Status != model.Passed || len(checked.Models) != 3 || !checked.Models[0].Local || len(checked.Models[0].Definitions) != 22 || accounting.Status() != realization.Accounted || accounting.TreeID() != raw.Census.Digest() || len(accounting.Files()) != len(raw.Census.Entries()) || len(accounting.Findings()) != 0 {
					t.Fatal("actual M1/Vocabulary/Core Model check or complete Census accounting missing")
				}
				if raw.Local.Selection != selection.Local || !reflect.DeepEqual([]apply.ModelSelection{raw.Dependencies[0].Selection, raw.Dependencies[1].Selection}, selection.Dependencies) {
					t.Fatal("explicit model/dependency boundaries changed")
				}
				assertPreparationFileClosure(t, raw)
				if !reflect.DeepEqual(execution.Defaults.Execute.Instructions.Value, []string{"ops/manager.md", "ops/reviewer.md", "ops/manager.md"}) || !reflect.DeepEqual(execution.Defaults.Verify.Instructions.Value, []string{"ops/reviewer.md"}) || !reflect.DeepEqual(execution.Packages.Value[0].Execute.Instructions.Value, []string{"ops/lending.md"}) || !reflect.DeepEqual(execution.Packages.Value[0].Verify.Instructions.Value, []string{"ops/manager.md", "ops/reviewer.md"}) {
					t.Fatal("authored instruction lists lost their order or repeats")
				}
				var packages []string
				for _, selected := range contexts {
					packages = append(packages, selected.Package())
					if selected.ModelDigest() != checked.Models[0].Digest || selected.ModelID() != checked.Models[0].ID {
						t.Fatal("static context lost its actual model binding")
					}
					if selected.Package() == "lending" {
						if _, found := selected.Definition(model.Identity{Package: "lending", ID: "borrow"}); !found {
							t.Fatal("lending context omitted its actual borrowed Definition")
						}
					}
				}
				if !reflect.DeepEqual(packages, []string{".", "catalog", "lending", "lending/reminders", "library", "members"}) {
					t.Fatalf("wrong static local Manager set: %q", packages)
				}
			})
		}
	}
}

func assertOriginalPreparationGap(t *testing.T, first, last apply.RawInputs) {
	t.Helper()
	if len(first.Registry) != 0 || len(first.Ignore) != 0 || len(first.Instructions) != 0 || len(first.Execution) != 1 {
		t.Fatal("earlier actual Raw gained later selected files")
	}
	if !first.Census.Equal(last.Census) || !reflect.DeepEqual(first.Local, last.Local) || !reflect.DeepEqual(first.Dependencies, last.Dependencies) {
		t.Fatal("complete recapture changed the unchanged actual model/Census basis")
	}
	for _, name := range []string{seamRootShard, seamChildShard, ".markitect/ignore.yaml", "ops/manager.md", "ops/reviewer.md", "ops/lending.md"} {
		if _, observed := first.Census.Lookup(fixturePath(t, name)); !observed {
			t.Fatalf("first Census did not actually observe %s", name)
		}
	}
}

func assertPreparationFileClosure(t *testing.T, raw apply.RawInputs) {
	t.Helper()
	for _, group := range []struct {
		files []apply.SourceFile
		paths []string
	}{
		{raw.Execution, []string{".markitect/execution.yaml"}},
		{raw.Registry, []string{seamRootShard, seamChildShard}},
		{raw.Ignore, []string{".markitect/ignore.yaml"}},
		{raw.Instructions, []string{"ops/manager.md", "ops/reviewer.md", "ops/lending.md"}},
	} {
		if len(group.files) != len(group.paths) {
			t.Fatal("closure selected missing or repeated files")
		}
		seen := make(map[string]bool)
		for _, file := range group.files {
			name := file.RepositoryPath.String()
			entry, present := raw.Census.Lookup(file.RepositoryPath)
			if !present || seen[name] || entry.Mode() != file.File.Mode() || !reflect.DeepEqual(entry.Bytes(), file.File.Bytes()) {
				t.Fatal("closure source repeated or changed actual Census bytes/mode")
			}
			seen[name] = true
		}
		for _, name := range group.paths {
			if !seen[name] {
				t.Fatalf("closure omitted actual %s", name)
			}
		}
	}
}
