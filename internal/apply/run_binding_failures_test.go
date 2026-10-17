package apply_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestCapturedRunBindingRequiresExplicitHostFactsAndBoundedAllocation(t *testing.T) {
	result, _, _ := runBindingCapture(t, nil, true)
	a, _ := result.Prepared()
	e, _ := a.Structural().Execution().Document()
	for _, tc := range []struct {
		name   string
		change func(*apply.RunBindingFacts)
	}{
		{"repository absent", func(f *apply.RunBindingFacts) { f.RepositoryIdentity = "" }},
		{"build absent", func(f *apply.RunBindingFacts) { f.ProductBuild = "" }},
		{"owner absent", func(f *apply.RunBindingFacts) { f.OwnerRequest = run.Payload{} }},
		{"allocation absent", func(f *apply.RunBindingFacts) { f.Allocation = run.Capacity{} }},
		{"run ceiling", func(f *apply.RunBindingFacts) { f.Allocation.Runs = e.Limits.MaxRuns.Value + 1 }},
		{"start ceiling", func(f *apply.RunBindingFacts) { f.Allocation.SessionStarts = e.Limits.MaxSessionStarts.Value + 1 }},
		{"step ceiling", func(f *apply.RunBindingFacts) { f.Allocation.Steps = e.Limits.MaxSteps.Value + 1 }},
		{"time ceiling", func(f *apply.RunBindingFacts) {
			f.Allocation.ExecutionTime = e.Limits.MaxExecutionTime.Value + time.Nanosecond
		}},
		{"start reserve", func(f *apply.RunBindingFacts) {
			f.Allocation.SessionStarts = e.CompletionReserve.SessionStarts.Value - 1
		}},
		{"step reserve", func(f *apply.RunBindingFacts) { f.Allocation.Steps = e.CompletionReserve.Steps.Value - 1 }},
		{"time reserve", func(f *apply.RunBindingFacts) {
			f.Allocation.ExecutionTime = e.CompletionReserve.ExecutionTime.Value - time.Nanosecond
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := runBindingHost(t)
			tc.change(&facts)
			binding, err := result.BindRun(facts)
			if !errors.Is(err, run.ErrInvalid) || binding.Valid() {
				t.Fatalf("invalid Host facts produced binding: %v", err)
			}
		})
	}
}

func TestCapturedRunBindingRefusesIncompleteActualPreparation(t *testing.T) {
	for _, kind := range []string{"source-error", "second-source-error", "guardless", "registry", "ignore", "instruction", "host-provenance", "execution-provenance"} {
		t.Run(kind, func(t *testing.T) {
			selection, actual, host := settingsServiceInputs(t)
			if kind != "second-source-error" {
				selection = completeSettingsServiceSelection(t, selection, host)
			}
			calls := 0
			source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
				calls++
				input := captured(preparationObservation(t, request.Selection, actual))
				switch kind {
				case "source-error":
					return input, errors.New("actual partial source failure")
				case "second-source-error":
					if calls == 2 {
						return input, errors.New("final acquisition failed")
					}
				case "guardless":
					input.Guard = ""
				case "registry":
					input.Raw.Registry = nil
				case "ignore":
					input.Raw.Ignore = nil
				case "instruction":
					input.Raw.Instructions = nil
				case "host-provenance":
					input.Raw.HostProfiles[0].Location.Boundary = ""
				case "execution-provenance":
					input.Raw.Execution[0].RepositoryPath = path(t, "wrong-execution.yaml")
				}
				return input, nil
			}}
			agent := settingsServiceAgent(t, host)
			_, check := hostPorts(t)
			result, _ := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
			if _, ok := result.Prepared(); ok {
				t.Fatal("invalid actual source became complete preparation")
			}
			if kind == "second-source-error" && (calls != 2 || len(result.Attempts()) != 2) {
				t.Fatal("fixture did not exercise failure after the first incomplete acquisition")
			}
			before := calls
			binding, err := result.BindRun(runBindingHost(t))
			if !errors.Is(err, run.ErrInvalid) || binding.Valid() || calls != before {
				t.Fatalf("incomplete preparation produced binding or reacquired: %v", err)
			}
		})
	}
	if binding, err := (apply.CapturedSettingsResult{}).BindRun(runBindingHost(t)); !errors.Is(err, run.ErrInvalid) || binding.Valid() {
		t.Fatal("zero preparation became bound")
	}
}

func TestCapturedRunBindingHasNoGuardlessPreviewEntryPoint(t *testing.T) {
	selection, actual, host := settingsServiceInputs(t)
	selection = completeSettingsServiceSelection(t, selection, host)
	source := controlledSource{preview: func(_ context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
		return apply.PreviewedInputs{Raw: preparationObservation(t, request.Selection, actual), Head: fixtureHead, Revision: request.Revision}, nil
	}}
	agent := settingsServiceAgent(t, host)
	_, check := hostPorts(t)
	result, err := apply.NewPreparationService(apply.NewModelService(nil, source)).PreviewSettings(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{}, agent, check)
	if _, ok := result.Prepared(); err != nil || !ok {
		t.Fatalf("actual complete guardless preview: %v", err)
	}
	// The public API accepts the committed result type only. A passing Preview
	// must not acquire this capability through a shared result interface.
	if _, ok := any(result).(interface {
		BindRun(apply.RunBindingFacts) (run.Binding, error)
	}); ok {
		t.Fatal("guardless preview exposes run binding")
	}
}
