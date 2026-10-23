package apply_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func settingsServiceInputs(t *testing.T) (apply.InputSelection, apply.RawInputs, apply.SourceLocation) {
	t.Helper()
	selection, raw := fileCheckInputs(t)
	preparationInstructions(t, &raw)
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, emptyFilePolicy, false)
	addRuntimeDocument(t, &selection, &raw, testRuntimeDocument)
	// The same spelling as a project instruction deliberately names other bytes.
	location := apply.SourceLocation{Boundary: "explicit-external-runtime", Path: path(t, "ops/one.md")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Executable, []byte("Original external Runtime instructions.\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	raw.Instructions = []apply.SourceFile{{Location: location, File: entry}}
	return selection, raw, location
}

// Metadata remains controlled static evidence. This port never resolves a
// missing instruction itself; it requests a whole new acquisition instead.
func settingsServiceAgent(t *testing.T, location apply.SourceLocation) runtimeAgentPort {
	t.Helper()
	return func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		o := testObservation(t, true)
		found := false
		for _, file := range input.Instructions() {
			found = found || file.Location == location
		}
		if found {
			o.Instructions = []apply.SourceLocation{location, location}
		} else {
			o.Status = model.NotCompleted
			o.RequiredInstructions = []apply.SourceLocation{location, location}
		}
		return o, nil
	}
}

func assertSettingsServiceAttempt(t *testing.T, structural apply.CapturedPreparationAttempt, settings apply.CapturedSettingsPreparation, defaults apply.CapturedDefaultsPreparation) {
	t.Helper()
	if !reflect.DeepEqual(settings.FileCheck(), structural.Files()) || !reflect.DeepEqual(settings.Defaults(), defaults) || !reflect.DeepEqual(defaults.Execution().Model().Inputs(), structural.Model().Inputs()) {
		t.Fatal("settings/defaults mixed acquisition parents")
	}
}

func TestPreparationSettingsM1ClosesRepoAndRuntimeInputsWithinOneSharedBound(t *testing.T) {
	for _, complete := range []bool{false, true} {
		for _, mode := range []string{"capture", "working", "fixed"} {
			t.Run(fmt.Sprintf("complete=%v/%s", complete, mode), func(t *testing.T) {
				selection, actual, runtimeLocation := settingsServiceInputs(t)
				if complete {
					selection = completeSettingsServiceSelection(t, selection, runtimeLocation)
				} else {
					selection.Registry = nil
				}
				var captures []apply.CapturedInputs
				var captureRequests []apply.CaptureRequest
				var previews []apply.PreviewedInputs
				var previewRequests []apply.PreviewRequest
				source := controlledSource{
					capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
						input := captured(preparationObservation(t, request.Selection, actual))
						input.Guard = apply.PublicationGuard(fmt.Sprint("actual-guard-", len(captures)+1))
						captures = append(captures, input)
						captureRequests = append(captureRequests, request)
						return input, nil
					},
					preview: func(_ context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
						input := apply.PreviewedInputs{Raw: preparationObservation(t, request.Selection, actual), Head: fixtureHead, Revision: request.Revision}
						previews = append(previews, input)
						previewRequests = append(previewRequests, request)
						return input, nil
					},
				}
				_, check := runtimePorts(t)
				service := apply.NewPreparationService(apply.NewModelService(source, source))
				wantCalls := 1
				if !complete {
					wantCalls = 2
				}
				var raw apply.RawInputs
				var prepared apply.CapturedSettingsPreparation
				var settingsStatus model.Status
				var firstRequired []apply.SourceLocation
				var accounting realization.Accounting
				if mode == "capture" {
					request := apply.CaptureRequest{Selection: selection}
					result, err := service.CaptureSettings(context.Background(), request, model.Limits{}, settingsServiceAgent(t, runtimeLocation), check)
					last, ok := result.Prepared()
					if err != nil || !ok || result.Status() != model.Passed || len(result.Attempts()) != wantCalls || len(captures) != wantCalls || !reflect.DeepEqual(result.Request(), request) {
						t.Fatalf("combined M1 preparation: %s %v %+v", result.Status(), err, result.Attempts())
					}
					for i, attempt := range result.Attempts() {
						if !reflect.DeepEqual(attempt.Structural().Model().Inputs(), captures[i]) || !reflect.DeepEqual(attempt.Structural().Model().Request(), captureRequests[i]) {
							t.Fatal("original capture request/bytes/guard changed")
						}
						assertSettingsServiceAttempt(t, attempt.Structural(), attempt.Settings(), attempt.Defaults())
					}
					raw, prepared = last.Structural().Model().Inputs().Raw, last.Settings()
					accounting, _ = last.Structural().Files().Accounting()
					settingsStatus = prepared.Status()
					firstRequired = result.Attempts()[0].Settings().RequiredInstructions()
					if !reflect.DeepEqual(last.Structural().Model().Inputs(), captures[wantCalls-1]) {
						t.Fatal("Prepared does not use the last full acquisition")
					}
				} else {
					request := apply.PreviewRequest{Selection: selection}
					if mode == "fixed" {
						request.Revision = fixtureHead
					}
					result, err := service.PreviewSettings(context.Background(), request, model.Limits{}, settingsServiceAgent(t, runtimeLocation), check)
					last, ok := result.Prepared()
					if err != nil || !ok || result.Status() != model.Passed || len(result.Attempts()) != wantCalls || len(previews) != wantCalls || !reflect.DeepEqual(result.Request(), request) {
						t.Fatalf("combined M1 preview: %s %v", result.Status(), err)
					}
					for i, attempt := range result.Attempts() {
						if !reflect.DeepEqual(attempt.Structural().Model().Inputs(), previews[i]) || !reflect.DeepEqual(attempt.Structural().Model().Request(), previewRequests[i]) || previews[i].Revision != request.Revision || !reflect.DeepEqual(attempt.Settings().FileCheck(), attempt.Structural().Files()) || !reflect.DeepEqual(attempt.Settings().Defaults(), attempt.Defaults()) {
							t.Fatal("original preview parents/bytes/revision changed")
						}
					}
					raw = last.Structural().Model().Inputs().Raw
					accounting, _ = last.Structural().Files().Accounting()
					settingsStatus = last.Settings().Status()
					firstRequired = result.Attempts()[0].Settings().RequiredInstructions()
					if _, ok := last.Settings().Settings(); !ok {
						t.Fatal("preview settings absent")
					}
					if checks, ok := last.Settings().Checks(); !ok || len(checks) != 1 {
						t.Fatal("preview check selection absent")
					}
				}
				if settingsStatus != model.Passed || accounting.Status() != realization.Accounted || len(raw.Registry) != 2 || len(raw.Ignore) != 1 || len(raw.Instructions) != 6 {
					t.Fatal("joint closure or real accounting incomplete")
				}
				if !complete && (len(firstRequired) != 1 || firstRequired[0] != runtimeLocation) {
					t.Fatal("first attempt lost new Runtime acquisition need")
				}
				if complete && len(firstRequired) != 0 {
					t.Fatal("complete initial selection invented a recapture need")
				}
				if mode == "capture" {
					settings, ok := prepared.Settings()
					root, exists := settings.Manager(".")
					checks, checkOK := prepared.Checks()
					if !ok || !exists || !checkOK || len(checks) != 1 || root.Execute.Profile.Value.Identity == "" || len(root.Execute.Instructions) != 3 || root.Execute.Instructions[0].Value.ID != "ops/one.md" || !reflect.DeepEqual(root.Execute.Instructions[0].Value.Bytes, root.Execute.Instructions[2].Value.Bytes) {
						t.Fatal("real static settings/profile selection or project repeats missing")
					}
				}
				seen := make(map[apply.SourceLocation]bool)
				for _, source := range raw.Instructions {
					if seen[source.Location] {
						t.Fatal("derived input selected more than once")
					}
					seen[source.Location] = true
					if source.Location == runtimeLocation && source.RepositoryPath.String() != "" {
						t.Fatal("Runtime instruction acquired project coordinates")
					}
				}
			})
		}
	}
}
