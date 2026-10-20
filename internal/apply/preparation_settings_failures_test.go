package apply_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func completeSettingsServiceSelection(t *testing.T, selection apply.InputSelection, runtimeLocation apply.SourceLocation) apply.InputSelection {
	t.Helper()
	selection.Ignore = []apply.SourceLocation{{Boundary: selection.Repository, Path: path(t, ".markitect/ignore.yaml")}}
	selection.Instructions = nil
	for _, name := range []string{"ops/one.md", "ops/two.md", "ops/three.md", "ops/four.md", "ops/five.md"} {
		selection.Instructions = append(selection.Instructions, apply.SourceLocation{Boundary: selection.Repository, Path: path(t, name)})
	}
	selection.Instructions = append(selection.Instructions, runtimeLocation)
	return selection
}

func TestPreparationSettingsSecondNeedsRetainCurrentFailureAndStop(t *testing.T) {
	for _, changed := range []bool{false, true} {
		for _, currentFailure := range []bool{false, true} {
			for _, preview := range []bool{false, true} {
				t.Run(fmt.Sprintf("changed=%v/currentFailure=%v/preview=%v", changed, currentFailure, preview), func(t *testing.T) {
					selection, actual, runtimeLocation := settingsServiceInputs(t)
					calls, metadata := 0, 0
					partialError := errors.New("second actual metadata error beside instruction need")
					observe := func(selected apply.InputSelection) apply.RawInputs {
						calls++
						if calls > 2 {
							t.Fatal("third source call")
						}
						return preparationObservation(t, selected, actual)
					}
					source := controlledSource{
						capture: func(_ context.Context, r apply.CaptureRequest) (apply.CapturedInputs, error) {
							return captured(observe(r.Selection)), nil
						},
						preview: func(_ context.Context, r apply.PreviewRequest) (apply.PreviewedInputs, error) {
							return apply.PreviewedInputs{Raw: observe(r.Selection), Head: fixtureHead, Revision: r.Revision}, nil
						},
					}
					agent := runtimeAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
						metadata++
						o := testObservation(t, true)
						o.RequiredInstructions = []apply.SourceLocation{runtimeLocation}
						if metadata == 1 || currentFailure {
							o.Status = model.Failed
						}
						if metadata == 2 && changed {
							o.RequiredInstructions = []apply.SourceLocation{{Boundary: "new-explicit-runtime", Path: path(t, "new.md")}}
						}
						if metadata == 2 {
							return o, partialError
						}
						return o, nil
					})
					_, check := runtimePorts(t)
					service := apply.NewPreparationService(apply.NewModelService(source, source))
					var status, firstStatus, lastStatus model.Status
					var required []apply.SourceLocation
					var err error
					if preview {
						result, failure := service.PreviewSettings(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{}, agent, check)
						err, status = failure, result.Status()
						attempts := result.Attempts()
						if len(attempts) != 2 {
							t.Fatal("attempts lost")
						}
						firstStatus, lastStatus = attempts[0].Settings().Status(), attempts[1].Settings().Status()
						required = attempts[1].Settings().RequiredInstructions()
						if _, ok := result.Prepared(); ok {
							t.Fatal("incomplete preview prepared")
						}
					} else {
						result, failure := service.CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
						err, status = failure, result.Status()
						attempts := result.Attempts()
						if len(attempts) != 2 {
							t.Fatal("attempts lost")
						}
						firstStatus, lastStatus = attempts[0].Settings().Status(), attempts[1].Settings().Status()
						required = attempts[1].Settings().RequiredInstructions()
						if _, ok := result.Prepared(); ok {
							t.Fatal("incomplete capture prepared")
						}
					}
					want := model.NotCompleted
					if currentFailure {
						want = model.Failed
					}
					if !errors.Is(err, apply.ErrPreparationClosure) || !errors.Is(err, partialError) || calls != 2 || metadata != 2 || status != want || lastStatus != want || firstStatus != model.Failed || len(required) != 1 {
						t.Fatalf("second actual status lost: %s %s first=%s calls=%d metadata=%d error=%v", status, lastStatus, firstStatus, calls, metadata, err)
					}
				})
			}
		}
	}
}

func TestPreparationSettingsMissingMetadataDoesNotRecaptureClosedInputs(t *testing.T) {
	selection, actual, runtimeLocation := settingsServiceInputs(t)
	selection = completeSettingsServiceSelection(t, selection, runtimeLocation)
	calls := 0
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		calls++
		return captured(preparationObservation(t, request.Selection, actual)), nil
	}}
	agent := runtimeAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
		return apply.ProfileObservation{Status: model.NotCompleted, Diagnostics: []apply.RuntimeProfileDiagnostic{{Code: "controlled.metadata_unavailable", Message: "Executable/schema metadata unavailable."}}}, nil
	})
	_, check := runtimePorts(t)
	result, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
	if err != nil || result.Status() != model.NotCompleted || calls != 1 || len(result.Attempts()) != 1 || len(result.Attempts()[0].Settings().RequiredInstructions()) != 0 {
		t.Fatal("metadata uncertainty caused an unrelated source retry")
	}
	profile := result.Attempts()[0].Settings().ProfileObservations()[0]
	if _, ok := profile.Identity(); ok {
		t.Fatal("missing metadata supplied static profile identity")
	}
	if _, ok := result.Prepared(); ok {
		t.Fatal("missing metadata prepared")
	}
	if len(profile.Observation().Diagnostics) == 0 {
		t.Fatal("partial metadata diagnostic lost")
	}
}

func TestPreparationSettingsPreservesPartialSourceAndMetadataErrors(t *testing.T) {
	for _, kind := range []string{"source", "metadata", "known-invalid-metadata", "cancel"} {
		for _, preview := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/preview=%v", kind, preview), func(t *testing.T) {
				selection, actual, runtimeLocation := settingsServiceInputs(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				failure := errors.New("actual partial observation failure")
				calls, metadata := 0, 0
				var observed apply.RawInputs
				observe := func(selected apply.InputSelection) (apply.RawInputs, error) {
					calls++
					observed = preparationObservation(t, selected, actual)
					if kind == "source" && calls == 2 {
						observed.Instructions = nil
						return observed, failure
					}
					return observed, nil
				}
				source := controlledSource{
					capture: func(_ context.Context, r apply.CaptureRequest) (apply.CapturedInputs, error) {
						raw, err := observe(r.Selection)
						return captured(raw), err
					},
					preview: func(_ context.Context, r apply.PreviewRequest) (apply.PreviewedInputs, error) {
						raw, err := observe(r.Selection)
						return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: r.Revision}, err
					},
				}
				agent := runtimeAgentPort(func(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
					metadata++
					o, _ := settingsServiceAgent(t, runtimeLocation)(ctx, input)
					if kind == "cancel" {
						cancel()
						return o, context.Canceled
					}
					if metadata == 2 && kind != "source" {
						o.Status = model.NotCompleted
						if kind == "known-invalid-metadata" {
							o.Status = model.Failed
						}
						return o, failure
					}
					return o, nil
				})
				_, check := runtimePorts(t)
				service := apply.NewPreparationService(apply.NewModelService(source, source))
				var status model.Status
				var err, retained error
				var raw apply.RawInputs
				var required []apply.SourceLocation
				var profiles []apply.PreparedProfile
				if preview {
					result, failure := service.PreviewSettings(ctx, apply.PreviewRequest{Selection: selection}, model.Limits{}, agent, check)
					err, status = failure, result.Status()
					attempts := result.Attempts()
					last := attempts[len(attempts)-1]
					retained, raw = last.Error(), last.Structural().Model().Inputs().Raw
					required, profiles = attempts[0].Settings().RequiredInstructions(), last.Settings().ProfileObservations()
					if _, ok := result.Prepared(); ok {
						t.Fatal("errored preview prepared")
					}
				} else {
					result, failure := service.CaptureSettings(ctx, apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
					err, status = failure, result.Status()
					attempts := result.Attempts()
					last := attempts[len(attempts)-1]
					retained, raw = last.Error(), last.Structural().Model().Inputs().Raw
					required, profiles = attempts[0].Settings().RequiredInstructions(), last.Settings().ProfileObservations()
					if _, ok := result.Prepared(); ok {
						t.Fatal("errored capture prepared")
					}
				}
				want, wantCalls, wantMetadata := model.NotCompleted, 2, 2
				if kind == "source" {
					wantMetadata = 1
				}
				if kind == "known-invalid-metadata" {
					want = model.Failed
				}
				if kind == "cancel" {
					failure, wantCalls, wantMetadata = context.Canceled, 1, 1
				}
				if !errors.Is(err, failure) || retained != err || status != want || calls != wantCalls || metadata != wantMetadata || !reflect.DeepEqual(raw, observed) || len(required) != 1 || required[0] != runtimeLocation {
					t.Fatalf("partial acquisition/error lost: %s %v calls=%d metadata=%d", status, err, calls, metadata)
				}
				if kind != "source" && len(profiles) == 0 {
					t.Fatal("metadata error discarded partial profile observations")
				}
			})
		}
	}
}

func TestPreparationSettingsClosesKnownSourcesBesideFirstMetadataError(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(fmt.Sprint("preview=", preview), func(t *testing.T) {
			selection, actual, runtimeLocation := settingsServiceInputs(t)
			selection.Registry = nil
			calls, metadata := 0, 0
			failure := errors.New("first actual partial metadata error")
			observe := func(selected apply.InputSelection) apply.RawInputs {
				calls++
				return preparationObservation(t, selected, actual)
			}
			source := controlledSource{
				capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
					return captured(observe(request.Selection)), nil
				},
				preview: func(_ context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
					return apply.PreviewedInputs{Raw: observe(request.Selection), Head: fixtureHead, Revision: request.Revision}, nil
				},
			}
			agent := runtimeAgentPort(func(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
				metadata++
				o, _ := settingsServiceAgent(t, runtimeLocation)(ctx, input)
				if metadata == 1 {
					o.Status = model.Failed
					o.Diagnostics = append(o.Diagnostics, apply.RuntimeProfileDiagnostic{Code: "controlled.first_metadata_error", Message: failure.Error()})
					return o, failure
				}
				return o, nil
			})
			_, check := runtimePorts(t)
			service := apply.NewPreparationService(apply.NewModelService(source, source))
			var status model.Status
			var err, firstError error
			var firstStatus model.Status
			var first, last apply.RawInputs
			if preview {
				result, failure := service.PreviewSettings(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{}, agent, check)
				err, status = failure, result.Status()
				attempts := result.Attempts()
				firstError, firstStatus = attempts[0].Error(), attempts[0].Settings().Status()
				first, last = attempts[0].Structural().Model().Inputs().Raw, attempts[len(attempts)-1].Structural().Model().Inputs().Raw
				if _, complete := result.Prepared(); !complete {
					t.Fatal("complete second actual preview withheld")
				}
			} else {
				result, failure := service.CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
				err, status = failure, result.Status()
				attempts := result.Attempts()
				firstError, firstStatus = attempts[0].Error(), attempts[0].Settings().Status()
				first, last = attempts[0].Structural().Model().Inputs().Raw, attempts[len(attempts)-1].Structural().Model().Inputs().Raw
				if _, complete := result.Prepared(); !complete {
					t.Fatal("complete second actual capture withheld")
				}
			}
			if err != nil || status != model.Passed || firstStatus != model.Failed || !errors.Is(firstError, failure) || calls != 2 || metadata != 2 || len(first.Registry) != 0 || len(first.Instructions) != 0 || len(last.Registry) != 2 || len(last.Ignore) != 1 || len(last.Instructions) != 6 {
				t.Fatalf("metadata error suppressed observed closure or contaminated current status: %s %v calls=%d metadata=%d", status, err, calls, metadata)
			}
		})
	}
}
