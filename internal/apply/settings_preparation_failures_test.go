package apply_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestRuntimeSettingsKeepsDefaultsFailuresAndExactSelectionSources(t *testing.T) {
	for _, tc := range []string{"missing product", "invalid product", "explicit declarations"} {
		t.Run(tc, func(t *testing.T) {
			selection, raw := runtimeSettingsInputs(t)
			input := apply.ProductStrategyDefaults()
			want := model.Failed
			switch tc {
			case "missing product":
				input.Bytes = nil
				want = model.NotCompleted
			case "invalid product":
				input.Bytes = []byte("invalid: true")
			case "explicit declarations":
				content := strings.Replace(defaultsAbsentExecution(t), "profiles:\n  execute", "instructions: {execute: [], verify: []}\nprofiles:\n  execute", 1)
				rewriteFileCheckSource(t, &raw, &raw.Execution[0], content)
				want = model.Passed
			}
			m := registryModel(t, selection, raw)
			defaults := m.DecodeExecution().PrepareDefaults(input)
			agent, check := runtimePorts(t)
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), defaults, agent, check)
			settings, available := result.Settings()
			if err != nil || result.Status() != want || !reflect.DeepEqual(result.Defaults(), defaults) || available != (want == model.Passed) {
				t.Fatalf("defaults seam: %v %s %+v", err, result.Status(), result.Diagnostics())
			}
			if available {
				root, _ := settings.Manager(".")
				original, _ := m.DecodeExecution().Document()
				if len(root.Execute.InstructionSources) != 1 || root.Execute.InstructionSources[0] != original.Defaults.Execute.Instructions.Source {
					t.Fatal("explicit empty instruction selection lost authored source")
				}
			}
		})
	}
}

func TestRuntimeSettingsAdapterOwnsClosedInlineValidation(t *testing.T) {
	selection, raw := runtimeSettingsInputs(t)
	addRuntimeDocument(t, &selection, &raw, strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 2, unknownAdapterField: true}", 1))
	m := registryModel(t, selection, raw)
	agent, check := runtimePorts(t)
	calls := 0
	agent = func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		calls++
		// A controlled closed adapter contract rejects fields/versions here.
		if input.Inline().Content[1].Value != "2" || input.Inline().Content[2].Value != "unknownAdapterField" {
			t.Fatal("Apply rewrote adapter-owned inline input")
		}
		return apply.ProfileObservation{Status: model.Failed, Contract: apply.ProfileContract{Name: "controlled-closed-contract", Version: 1}, Diagnostics: []apply.RuntimeProfileDiagnostic{{Code: "test.adapter_contract", Message: "Unsupported version and unknown adapter field."}}}, nil
	}
	result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	if err != nil || calls != 1 || result.Status() != model.Failed {
		t.Fatalf("adapter contract result hidden: %v %+v", err, result.Diagnostics())
	}
}

func TestRuntimeSettingsRetainsIndependentStaticComponentEvidence(t *testing.T) {
	selection, raw := runtimeSettingsInputs(t)
	m := registryModel(t, selection, raw)
	agent, check := runtimePorts(t)
	for _, missingAgent := range []bool{false, true} {
		var a apply.AgentProfilePreparer = agent
		var c apply.CheckProfilePreparer = check
		if missingAgent {
			a = nil
		} else {
			c = nil
		}
		result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), a, c)
		_, settingsAvailable := result.Settings()
		_, checksAvailable := result.Checks()
		if err != nil || result.Status() != model.NotCompleted || settingsAvailable == missingAgent || checksAvailable != missingAgent {
			t.Fatalf("independent static evidence lost: %v %+v", err, result.Diagnostics())
		}
	}
	var files apply.CapturedFileCheck
	var defaults apply.CapturedDefaultsPreparation
	zero, err := files.PrepareSettings(context.Background(), defaults, nil, nil)
	if err != nil || zero.Status() != model.NotCompleted {
		t.Fatal("zero parents became usable")
	}
	for _, d := range zero.Diagnostics() {
		if d.Source.File != "" {
			t.Fatal("missing source received an invented locator")
		}
	}
}
