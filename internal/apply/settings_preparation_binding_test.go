package apply_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestHostSettingsRejectsMixedCaptureAndPreviewParents(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	agent, check := hostPorts(t)
	for _, change := range []string{"guard", "head", "request", "raw"} {
		t.Run(change, func(t *testing.T) {
			otherSelection := selection
			otherRaw := raw
			inputs := m.Inputs()
			switch change {
			case "guard":
				inputs.Guard += "different"
			case "head":
				inputs.Head = strings.Repeat("b", 40)
			case "request":
				otherSelection.Repository += "different"
			case "raw":
				addHostDocument(t, &otherSelection, &otherRaw, "# Other original bytes\n"+testHostDocument)
				inputs.Raw = otherRaw
			}
			source := controlledSource{capture: func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) { return inputs, nil }}
			other, err := apply.NewModelService(source, nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: otherSelection}, model.Limits{})
			if err != nil {
				t.Fatal(err)
			}
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), other.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
			if err != nil || result.Status() != model.Failed || len(result.ProfileObservations()) != 0 {
				t.Fatalf("mixed capture accepted: %v %+v", err, result.Diagnostics())
			}
			if _, ok := result.Settings(); ok {
				t.Fatal("mixed capture settings exposed")
			}
		})
	}
	var previews []apply.PreviewedModel
	for _, revision := range []string{"", fixtureHead} {
		source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
			return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: revision}, nil
		}}
		p, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		previews = append(previews, p)
	}
	result, err := previews[0].PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), previews[1].DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	if err != nil || result.Status() != model.Failed {
		t.Fatal("mixed preview revision accepted")
	}
}

func TestHostInstructionsNeverExtendOldInputsAndFailedFileCheckStaysFailed(t *testing.T) {
	for _, failed := range []bool{false, true} {
		selection, raw := hostSettingsInputs(t)
		if failed {
			censusFile(t, &raw, "unexplained.txt", snapshot.Regular, []byte("unaccounted"))
		}
		m := registryModel(t, selection, raw)
		files := m.PrepareRegistry().CheckFiles()
		missing := apply.SourceLocation{Boundary: "external-host-instructions", Path: path(t, "new.md")}
		agent, check := hostPorts(t)
		agent = func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
			o := testObservation(t, true)
			o.Instructions = []apply.SourceLocation{missing}
			return o, nil
		}
		result, err := files.PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
		want := model.NotCompleted
		if failed {
			want = model.Failed
		}
		if err != nil || result.Status() != want || !reflect.DeepEqual(result.RequiredInstructions(), []apply.SourceLocation{missing}) || !reflect.DeepEqual(result.FileCheck(), files) || !reflect.DeepEqual(result.FileCheck().RegistryPreparation().Model().Inputs(), m.Inputs()) {
			t.Fatalf("instruction closure/status lost: %v %s %+v", err, result.Status(), result.Diagnostics())
		}
		if _, ok := result.Settings(); ok {
			t.Fatal("missing Host instruction exposed Settings")
		}
		if failed {
			if _, ok := result.Checks(); ok {
				t.Fatal("failed FileCheck exposed CheckSelections")
			}
		}
	}
}

func TestHostSettingsResolvesCapturedProjectInstructionsWithRepeats(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	content := strings.Replace(defaultsAbsentExecution(t), "profiles:\n  execute", "instructions: {execute: [ops/one.md, ops/one.md], verify: []}\nprofiles:\n  execute", 1)
	rewriteFileCheckSource(t, &raw, &raw.Execution[0], content)
	fileCheckSource(t, &selection, &raw, "instruction", "ops/one.md", snapshot.Regular, "Original instruction bytes.\r\n", true)
	m := registryModel(t, selection, raw)
	agent, check := hostPorts(t)
	agent = func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		o := testObservation(t, true)
		if !reflect.DeepEqual(input.Instructions(), raw.Instructions) {
			t.Fatal("port did not receive exact captured instructions")
		}
		o.Instructions = []apply.SourceLocation{raw.Instructions[0].Location}
		return o, nil
	}
	result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	settings, ok := result.Settings()
	if err != nil || result.Status() != model.Passed || !ok {
		t.Fatalf("captured instructions unresolved: %v %+v", err, result.Diagnostics())
	}
	root, _ := settings.Manager(".")
	if len(root.Execute.Instructions) != 2 || !reflect.DeepEqual(root.Execute.Instructions[0], root.Execute.Instructions[1]) || string(root.Execute.Instructions[0].Value.Bytes) != "Original instruction bytes.\r\n" {
		t.Fatal("instruction bytes or authored repeats lost")
	}
	if !reflect.DeepEqual(result.Defaults().Execution().Partial(), m.DecodeExecution().Partial()) {
		t.Fatal("authored lists changed")
	}
}
