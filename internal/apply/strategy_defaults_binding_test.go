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
		{"  reviewRounds: 3", "  reviewRounds: 2"},
		{"  profiles:", "  instructions: {manager: [one.md, two.md, one.md], reviewer: []}\n  profiles:"},
		{"packages:\n  lending:\n    stepTimeout: 8m", "packages:\n  lending:\n    profiles: {manager: unobserved-profile}\n    instructions: {manager: [third.md, one.md]}\n    stepTimeout: 8m\n    reviewRounds: 3\n    creationRoots: [realization/lending]\n  '.':\n    stepTimeout: 9m\n  lending/reminders: {}"},
		{"ownerApproval: []", "ownerApproval: [lending/reminders, ., lending]"},
	} {
		if strings.Count(content, edit[0]) != 1 {
			t.Fatal("explicit declaration fixture marker changed")
		}
		content = strings.Replace(content, edit[0], edit[1], 1)
	}
	return content
}

func TestDefaultsExplicitFalseEmptyListsAndExactOverridesKeepAuthoredSources(t *testing.T) {
	parent := executionResult(t, defaultsExplicitExecution(t))
	before, _ := parent.Document()
	prepared := parent.PrepareDefaults(apply.ProductStrategyDefaults())
	values, ok := prepared.Declarations()
	if !ok || prepared.Status() != model.Passed || values.Defaults.ReviewRounds != *before.Defaults.ReviewRounds || values.Defaults.BatchNoChangeReviews != *before.Defaults.BatchNoChangeReviews || !reflect.DeepEqual(values.Defaults.Reviewer.Instructions, *before.Defaults.Reviewer.Instructions) || !reflect.DeepEqual(values.OwnerApproval, *before.OwnerApproval) {
		t.Fatalf("explicit false/empty/value declarations were replaced: %+v", prepared.Diagnostics())
	}
	if values.Defaults.BatchNoChangeReviews.Value || values.Defaults.Reviewer.Instructions.Source.File != ".markitect/execution.yaml" || values.Defaults.Reviewer.Instructions.Source.Field != "/defaults/instructions/reviewer" || !reflect.DeepEqual(values.Defaults.Manager.Instructions.Value, []string{"one.md", "two.md", "one.md"}) {
		t.Fatal("explicit empty/repeated instructions lost their actual source")
	}
	if !reflect.DeepEqual(values.Overrides, before.Packages.Value) || len(values.Overrides) != 3 || values.Overrides[2].Package != "lending/reminders" || values.Overrides[2].StepTimeout != nil || values.Overrides[2].ReviewRounds != nil || values.Overrides[2].Manager.Profile != nil {
		t.Fatal("exact override was changed or acquired ancestor declarations")
	}
	if values.Defaults.ReviewRounds.Value != 2 || values.Overrides[0].ReviewRounds.Value != 3 || values.Overrides[0].Manager.Profile.Value != "unobserved-profile" || len(parent.Model().Inputs().Raw.HostProfiles) != 0 || len(parent.Model().Inputs().Raw.Instructions) != 0 {
		t.Fatal("project defaults became ceilings, or declarative names invented resolved profile/instruction facts")
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
	changed.Defaults.Manager.Instructions.Value[0] = "changed.md"
	changed.Defaults.CreationRoots.Value[0] = snapshot.RootDirectory()
	changed.Ceilings.CreationRoots.Value[0] = snapshot.RootDirectory()
	changed.Overrides[0].Manager.Profile.Value = "changed-profile"
	changed.Overrides[0].Manager.Instructions.Value[0] = "changed-override.md"
	changed.Overrides[0].CreationRoots.Value[0] = snapshot.RootDirectory()
	changed.OwnerApproval.Value[0] = "changed-package"
	after, _ := prepared.Declarations()
	if !reflect.DeepEqual(after, before) || string(prepared.ProductInput().Bytes) != original || prepared.ProductSource().ContentID != snapshot.ContentID(sha256.Sum256([]byte(original))) || !reflect.DeepEqual(prepared.Execution(), parent) {
		t.Fatal("returned/input declaration data aliases the bound preparation")
	}
	invalid := apply.ProductStrategyDefaults()
	invalid.Bytes = []byte(strings.Replace(string(invalid.Bytes), "reviewRounds: 3", "reviewRounds: null", 1))
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
