package apply_test

import (
	"context"
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func defaultsExplicitExecution(t *testing.T) string {
	t.Helper()
	content := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	for _, edit := range [][2]string{
		{"profiles:\n  execute", "instructions: {execute: [one.md, two.md, one.md], verify: []}\nprofiles:\n  execute"},
		{"packages:\n  lending:\n    profiles: {verify: library-agent}", "packages:\n  lending:\n    profiles: {execute: unobserved-profile}\n    instructions: {execute: [third.md, one.md]}\n  '.':\n    profiles: {verify: other-profile}\n  lending/reminders: {}"},
	} {
		if strings.Count(content, edit[0]) != 1 {
			t.Fatal("explicit declaration fixture marker changed")
		}
		content = strings.Replace(content, edit[0], edit[1], 1)
	}
	return content
}

func TestDefaultsExplicitEmptyListsAndExactOverridesKeepAuthoredSources(t *testing.T) {
	parent := executionResult(t, defaultsExplicitExecution(t))
	before, _ := parent.Document()
	prepared := parent.PrepareDefaults(apply.ProductStrategyDefaults())
	values, ok := prepared.Declarations()
	if !ok || prepared.Status() != model.Passed || !reflect.DeepEqual(values.Defaults.Verify.Instructions, *before.Defaults.Verify.Instructions) {
		t.Fatalf("explicit empty declarations were replaced: %+v", prepared.Diagnostics())
	}
	if values.Defaults.Verify.Instructions.Source.File != ".markitect/execution.yaml" || values.Defaults.Verify.Instructions.Source.Field != "/instructions/verify" || !reflect.DeepEqual(values.Defaults.Execute.Instructions.Value, []string{"one.md", "two.md", "one.md"}) {
		t.Fatal("explicit empty/repeated instructions lost their actual source")
	}
	if !reflect.DeepEqual(values.Overrides, before.Packages.Value) || len(values.Overrides) != 3 || values.Overrides[2].Package != "lending/reminders" || values.Overrides[2].Execute.Profile != nil || values.Overrides[2].Verify.Profile != nil {
		t.Fatal("exact override was changed or acquired ancestor declarations")
	}
	if values.Overrides[0].Execute.Profile.Value != "unobserved-profile" || len(parent.Model().Inputs().Raw.HostProfiles) != 0 || len(parent.Model().Inputs().Raw.Instructions) != 0 {
		t.Fatal("declarative names invented resolved profile/instruction facts")
	}
	if !reflect.DeepEqual(prepared.Execution().Partial(), before) {
		t.Fatal("authored execution changed during default application")
	}
}

func TestDefaultsCaptureAndBothPreviewsRetainTheirOwnOpaqueParents(t *testing.T) {
	selection, raw := executionInputs(t, defaultsAbsentExecution(t))
	capture := registryModel(t, selection, raw).DecodeExecution()
	prepared := capture.PrepareDefaults(apply.ProductStrategyDefaults())
	expected, ok := prepared.Declarations()
	if !ok || !reflect.DeepEqual(prepared.Execution(), capture) || prepared.Execution().Model().Inputs().Guard == "" {
		t.Fatal("captured defaults lost their exact execution and guard")
	}
	for _, revision := range []string{"", fixtureHead} {
		source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
			return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: revision}, nil
		}}
		preview, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		parent := preview.DecodeExecution()
		result := parent.PrepareDefaults(apply.ProductStrategyDefaults())
		values, available := result.Declarations()
		if result.Status() != model.Passed || !available || !reflect.DeepEqual(values, expected) || !reflect.DeepEqual(result.Execution(), parent) || result.Execution().Model().Inputs().Revision != revision || result.Execution().Model().Request().Revision != revision || !reflect.DeepEqual(result.Execution().Model().Inputs(), preview.Inputs()) {
			t.Fatal("preview defaults changed their working/fixed source binding")
		}
	}
	var zero apply.PreviewDefaultsPreparation
	if _, ok := zero.Declarations(); ok || zero.Status() != model.NotCompleted {
		t.Fatal("zero preview defaults became usable")
	}
}

func TestDefaultsOwnEveryInputDeclarationAndDiagnosticCopy(t *testing.T) {
	input := apply.ProductStrategyDefaults()
	input.Bytes = []byte("# Original product source.\r\n" + strings.ReplaceAll(string(input.Bytes), "\n", "\r\n"))
	original := string(input.Bytes)
	parent := executionResult(t, defaultsExplicitExecution(t))
	prepared := parent.PrepareDefaults(input)
	before, ok := prepared.Declarations()
	if !ok {
		t.Fatalf("actual product bytes did not decode: %+v", prepared.Diagnostics())
	}
	input.Bytes[0] ^= 1
	input.Source = "mutated caller source"
	returned := prepared.ProductInput()
	returned.Bytes[0] ^= 1
	changed, _ := prepared.Declarations()
	changed.Defaults.Execute.Instructions.Value[0] = "changed.md"
	changed.Overrides[0].Execute.Profile.Value = "changed-profile"
	changed.Overrides[0].Execute.Instructions.Value[0] = "changed-override.md"
	after, _ := prepared.Declarations()
	if !reflect.DeepEqual(after, before) || string(prepared.ProductInput().Bytes) != original || prepared.ProductSource().ContentID != snapshot.ContentID(sha256.Sum256([]byte(original))) || !reflect.DeepEqual(prepared.Execution(), parent) {
		t.Fatal("returned/input declaration data aliases the bound preparation")
	}
	invalid := apply.ProductStrategyDefaults()
	invalid.Bytes = []byte(strings.Replace(string(invalid.Bytes), "execute: []", "execute: null", 1))
	refused := parent.PrepareDefaults(invalid)
	diagnostics := refused.Diagnostics()
	originalDiagnostics := refused.Diagnostics()
	for i := range diagnostics {
		diagnostics[i].Source.Field = "mutated field"
		if diagnostics[i].Position != nil {
			diagnostics[i].Position.Line = -1
		}
	}
	if refused.Status() != model.Failed || len(originalDiagnostics) == 0 || !reflect.DeepEqual(refused.Diagnostics(), originalDiagnostics) {
		t.Fatal("diagnostics alias or hide the original product refusal")
	}
	shipped := apply.ProductStrategyDefaults()
	shipped.Bytes[0] ^= 1
	if reflect.DeepEqual(shipped, apply.ProductStrategyDefaults()) {
		t.Fatal("shipped defaults accessor aliases embedded product storage")
	}
}
