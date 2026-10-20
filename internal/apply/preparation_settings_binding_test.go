package apply_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestPreparationSettingsDistinctAcquisitionPreservesRuntimeAndProjectPolicy(t *testing.T) {
	selection, actual, runtimeLocation := settingsServiceInputs(t)
	project := apply.SourceLocation{Boundary: selection.Repository, Path: path(t, "ops/one.md")}
	policy := []apply.SourceLocation{project, runtimeLocation, project}
	calls := 0
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		calls++
		return captured(preparationObservation(t, request.Selection, actual)), nil
	}}
	agent := runtimeAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
		o := testObservation(t, true)
		o.Instructions = policy
		return o, nil
	})
	_, check := runtimePorts(t)
	result, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
	prepared, ok := result.Prepared()
	if err != nil || !ok || calls != 2 {
		t.Fatalf("shared project/Runtime location acquisition: %s %v", result.Status(), err)
	}
	raw := prepared.Structural().Model().Inputs().Raw
	if len(raw.Instructions) != 6 {
		t.Fatal("one project/Runtime location was acquired twice")
	}
	var projectFile, runtimeFile apply.SourceFile
	for _, source := range raw.Instructions {
		if source.Location == project {
			projectFile = source
		}
		if source.Location == runtimeLocation {
			runtimeFile = source
		}
	}
	if projectFile.RepositoryPath.String() != "ops/one.md" || runtimeFile.RepositoryPath.String() != "" || reflect.DeepEqual(projectFile.File.Bytes(), runtimeFile.File.Bytes()) {
		t.Fatal("same spelling mixed Runtime/project bytes or coordinates")
	}
	observation := prepared.Settings().ProfileObservations()[0]
	if !reflect.DeepEqual(observation.Observation().Instructions, policy) {
		t.Fatal("Runtime policy order/repeats changed")
	}
	settings, _ := prepared.Settings().Settings()
	root, _ := settings.Manager(".")
	if len(root.Execute.Instructions) != 3 || !reflect.DeepEqual(root.Execute.Instructions[0].Value.Bytes, projectFile.File.Bytes()) {
		t.Fatal("Runtime bytes replaced project instruction facts")
	}
}

func TestPreparationSettingsKeepsFinalParentsAndCopiesAllAttemptViews(t *testing.T) {
	selection, actual, runtimeLocation := settingsServiceInputs(t)
	selection.Registry = nil
	calls := 0
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		calls++
		input := captured(preparationObservation(t, request.Selection, actual))
		if calls == 2 {
			input.Guard = "new-full-capture"
		}
		return input, nil
	}}
	agent := settingsServiceAgent(t, runtimeLocation)
	_, check := runtimePorts(t)
	service := apply.NewPreparationService(apply.NewModelService(source, nil))
	request := apply.CaptureRequest{Selection: selection}
	result, err := service.CaptureSettings(context.Background(), request, model.Limits{}, agent, check)
	prepared, ok := result.Prepared()
	if err != nil || !ok || calls != 2 {
		t.Fatalf("M1 settings: %v", err)
	}
	first, last := result.Attempts()[0], result.Attempts()[1]
	mixed, err := last.Structural().Files().PrepareSettings(context.Background(), first.Defaults(), agent, check)
	if err != nil || mixed.Status() != model.Failed || len(mixed.ProfileObservations()) != 0 {
		t.Fatal("old defaults accepted against final full parent")
	}
	if _, ok := mixed.Settings(); ok {
		t.Fatal("mixed parents supplied settings")
	}
	if !reflect.DeepEqual(prepared.Defaults().ProductInput(), apply.ProductStrategyDefaults()) || !reflect.DeepEqual(prepared.Settings().Defaults(), last.Defaults()) || !reflect.DeepEqual(prepared.Settings().FileCheck().RegistryPreparation().Model().Inputs(), last.Structural().Model().Inputs()) || last.Structural().Model().Inputs().Guard != "new-full-capture" {
		t.Fatal("actual product/default/guard parent lost")
	}
	firstProfile := first.Settings().ProfileObservations()[0]
	// Preserve the genuinely stale first binding rather than echoing the new one.
	staleResult, err := last.Structural().Files().PrepareSettings(context.Background(), last.Defaults(), staleBindingAgent{firstProfile.Observation()}, check)
	if err != nil || staleResult.Status() != model.Failed {
		t.Fatal("first metadata input binding accepted for new instruction view")
	}
	request.Selection.Dependencies[0] = apply.ModelSelection{}
	copyRequest := result.Request()
	copyRequest.Selection.Dependencies[0] = apply.ModelSelection{}
	attempts := result.Attempts()
	attempts[0] = apply.CapturedSettingsAttempt{}
	product := prepared.Defaults().ProductInput()
	product.Bytes[0] ^= 1
	needs := first.Settings().RequiredInstructions()
	needs[0] = apply.SourceLocation{}
	profiles := prepared.Settings().ProfileObservations()
	observation := profiles[0].Observation()
	observation.Files[0].Location.Boundary = "mutated"
	observation.Inventories[0].Files[0].Size++
	policy := observation.Instructions
	policy[0] = apply.SourceLocation{}
	inline := profiles[0].Input().Inline()
	inline.Content[0].Value = "mutated"
	inputFiles := profiles[0].Input().Instructions()
	inputFiles[0] = apply.SourceFile{}
	settings, _ := prepared.Settings().Settings()
	root, _ := settings.Manager(".")
	root.Execute.Instructions[0].Value.Bytes[0] ^= 1
	final, _ := result.Prepared()
	finalSettings, _ := final.Settings().Settings()
	finalRoot, _ := finalSettings.Manager(".")
	if result.Request().Selection.Dependencies[0].Boundary == "" || result.Attempts()[0].Structural().Model().Inputs().Guard == "" || !reflect.DeepEqual(final.Defaults().ProductInput(), apply.ProductStrategyDefaults()) || result.Attempts()[0].Settings().RequiredInstructions()[0] != runtimeLocation || final.Settings().ProfileObservations()[0].Observation().Files[0].Location.Boundary == "mutated" || final.Settings().ProfileObservations()[0].Observation().Inventories[0].Files[0].Size == observation.Inventories[0].Files[0].Size || final.Settings().ProfileObservations()[0].Observation().Instructions[0] != runtimeLocation || final.Settings().ProfileObservations()[0].Input().Inline().Content[0].Value == "mutated" || reflect.DeepEqual(root.Execute.Instructions[0].Value.Bytes, finalRoot.Execute.Instructions[0].Value.Bytes) {
		t.Fatal("caller mutation escaped a copied settings result/attempt view")
	}
}

type staleBindingAgent struct{ observation apply.ProfileObservation }

func (p staleBindingAgent) PrepareAgentProfile(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
	return p.observation, nil
}

func TestPreparationSettingsPreviewCopiesAndSourceRefusalRemainVisible(t *testing.T) {
	selection, actual, runtimeLocation := settingsServiceInputs(t)
	_, check := runtimePorts(t)
	// This controlled refusal proves forwarding only. The real Git adapter's
	// external fixed-revision refusal is tested in gitrepo's own allowed layer.
	refusal := errors.New("source refuses external Runtime at fixed revision")
	for _, refused := range []bool{false, true} {
		calls := 0
		source := controlledSource{preview: func(_ context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
			calls++
			raw := actual
			if !refused {
				raw = preparationObservation(t, request.Selection, actual)
			}
			input := apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: request.Revision}
			if refused {
				return input, refusal
			}
			return input, nil
		}}
		request := apply.PreviewRequest{Selection: selection, Revision: fixtureHead}
		result, err := apply.NewPreparationService(apply.NewModelService(nil, source)).PreviewSettings(context.Background(), request, model.Limits{}, settingsServiceAgent(t, runtimeLocation), check)
		if refused {
			if err != refusal || result.Status() != model.NotCompleted || calls != 1 || len(result.Attempts()) != 1 || result.Attempts()[0].Error() != refusal || !reflect.DeepEqual(result.Attempts()[0].Structural().Model().Inputs().Raw, actual) || result.Attempts()[0].Structural().Model().Request().Revision != fixtureHead {
				t.Fatal("source fixed-revision refusal/partial observations lost")
			}
			if _, ok := result.Prepared(); ok {
				t.Fatal("refused fixed preview prepared")
			}
			continue
		}
		prepared, ok := result.Prepared()
		if err != nil || !ok || calls != 2 {
			t.Fatalf("controlled preview: %s %v", result.Status(), err)
		}
		attempts := result.Attempts()
		attempts[0] = apply.PreviewSettingsAttempt{}
		copyRequest := result.Request()
		copyRequest.Selection.Dependencies[0] = apply.ModelSelection{}
		product := prepared.Defaults().ProductInput()
		product.Bytes[0] ^= 1
		settings, _ := prepared.Settings().Settings()
		root, _ := settings.Manager(".")
		root.Execute.Instructions[0].Value.Bytes[0] ^= 1
		profiles := prepared.Settings().ProfileObservations()
		observation := profiles[0].Observation()
		observation.Instructions[0] = apply.SourceLocation{}
		final, _ := result.Prepared()
		finalSettings, _ := final.Settings().Settings()
		finalRoot, _ := finalSettings.Manager(".")
		if result.Request().Selection.Dependencies[0].Boundary == "" || result.Attempts()[0].Structural().Model().Inputs().Revision != fixtureHead || !reflect.DeepEqual(final.Defaults().ProductInput(), apply.ProductStrategyDefaults()) || final.Settings().ProfileObservations()[0].Observation().Instructions[0] != runtimeLocation || reflect.DeepEqual(root.Execute.Instructions[0].Value.Bytes, finalRoot.Execute.Instructions[0].Value.Bytes) {
			t.Fatal("preview copy/binding escaped")
		}
	}
}

func TestPreparationSettingsZeroValuesAndStructuralEntriesStayDistinct(t *testing.T) {
	var capture apply.CapturedSettingsResult
	var preview apply.PreviewSettingsResult
	if _, ok := capture.Prepared(); ok || capture.Status() != model.NotCompleted {
		t.Fatal("zero capture settings prepared")
	}
	if _, ok := preview.Prepared(); ok || preview.Status() != model.NotCompleted {
		t.Fatal("zero preview settings prepared")
	}
	selection, raw := fileCheckInputs(t) // no Runtime profile document at all
	source := controlledSource{capture: func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) { return captured(raw), nil }}
	service := apply.NewPreparationService(apply.NewModelService(source, nil))
	structural, err := service.Capture(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || structural.Status() != model.Passed {
		t.Fatal("structural entry began requiring Runtime settings")
	}
	configured, err := service.CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, nil, nil)
	if err != nil || configured.Status() != model.NotCompleted || len(configured.Attempts()) != 1 || len(configured.Attempts()[0].Settings().Diagnostics()) == 0 {
		t.Fatal("explicit settings entry silently fell back to structural success")
	}
}
