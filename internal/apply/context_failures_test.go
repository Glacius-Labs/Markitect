package apply_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func TestStaticContextRejectsInvalidScopeThroughOrganization(t *testing.T) {
	for _, tc := range []struct{ name, target, scope string }{
		{"unknown Package", "missing", contextScope},
		{"imported Package", "software", contextScope},
		{"case differs", "Lending", contextScope},
		{"path normalization", "lending/../catalog", contextScope},
		{"Kind is Property", "lending", "scope: [{kind: policy/appliesTo, property: policy/appliesTo}]\n"},
		{"Property not declared", "lending", "scope: [{kind: policy/ScopedRule, property: lending/maxOpenLoans}]\n"},
		{"nontextual Property", "lending", "scope: [{kind: lending/LendingPolicy, property: lending/maxOpenLoans}]\n"},
		{"duplicate selector", "lending", "scope: [{kind: policy/ScopedRule, property: policy/appliesTo}, {kind: policy/ScopedRule, property: policy/appliesTo}]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := scopedContextInputs(t, tc.target, tc.scope)
			parent := registryModel(t, selection, raw).DecodeExecution()
			if parent.Status() != model.Passed {
				t.Fatalf("fixture did not reach organization consumer: %+v", parent.Diagnostics())
			}
			result := parent.SelectContexts()
			if got, ok := result.Selections(); ok || len(got) != 0 || result.Status() != model.Failed || len(result.Diagnostics()) == 0 {
				t.Fatalf("invalid selector/target exposed a usable partial set: %s %+v", result.Status(), result.Diagnostics())
			}
			for _, diagnostic := range result.Diagnostics() {
				if diagnostic.Code != "context.selection" || diagnostic.Package == "" || diagnostic.Message == "" {
					t.Fatal("organization failure lost its Manager/consumer observation")
				}
			}
			if !reflect.DeepEqual(result.Execution().Partial(), parent.Partial()) || !reflect.DeepEqual(result.Execution().Model().Inputs().Raw, raw) {
				t.Fatal("failed authored/source observations lost")
			}
		})
	}
}

func TestStaticContextsRetainFailedAndIncompleteExecutionParents(t *testing.T) {
	for _, tc := range []struct {
		name, replacement string
		status            model.Status
	}{
		{"known invalid", "agentTurns: 0", model.Failed},
		{"tool range", "agentTurns: 184467440737095516160", model.NotCompleted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := strings.Replace(documentedExecution(t), "agentTurns: 120", tc.replacement, 1)
			parent := executionResult(t, content)
			result := parent.SelectContexts()
			if got, ok := result.Selections(); ok || len(got) != 0 || result.Status() != tc.status || !reflect.DeepEqual(result.Execution().Diagnostics(), parent.Diagnostics()) || !reflect.DeepEqual(result.Execution().Model().Inputs(), parent.Model().Inputs()) {
				t.Fatalf("nonpassing parent promoted or lost: %+v", result.Diagnostics())
			}
			if len(result.Diagnostics()) != 1 || result.Diagnostics()[0].Code != "context.execution_not_passed" {
				t.Fatal("parent refusal lost")
			}
		})
	}
	selection, raw := executionInputs(t, documentedExecution(t))
	failure := errors.New("observed source interruption")
	stage, err := captureService(captured(raw), failure).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if !errors.Is(err, failure) {
		t.Fatal("acquisition error lost")
	}
	result := stage.DecodeExecution().SelectContexts()
	if got, ok := result.Selections(); ok || len(got) != 0 || result.Status() != model.NotCompleted || !reflect.DeepEqual(result.Execution().Model().Inputs().Raw, raw) {
		t.Fatal("partial acquisition promoted or discarded")
	}
	for _, limits := range []model.Limits{{Entries: 1}, {}} {
		selected := selection
		observed := raw
		if limits.Entries == 0 {
			selected.Dependencies, observed.Dependencies = nil, nil
		}
		stage, err := captureService(captured(observed), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selected}, limits)
		if err != nil || stage.Compilation().Status == model.Passed {
			t.Fatalf("fixture did not reach a nonpassing compiler: %v %+v", err, stage.Compilation())
		}
		result := stage.DecodeExecution().SelectContexts()
		if got, ok := result.Selections(); ok || len(got) != 0 || result.Status() != model.NotCompleted || !reflect.DeepEqual(result.Execution().Model().Compilation(), stage.Compilation()) {
			t.Fatal("nonpassing compiler result promoted or lost")
		}
	}
}

func TestStaticContextZeroStagesExposeNothing(t *testing.T) {
	var captured apply.CapturedContextSelections
	var preview apply.PreviewContextSelections
	var capturedParent apply.CapturedExecutionDecoding
	var previewParent apply.PreviewExecutionDecoding
	for _, result := range []interface {
		Status() model.Status
		Selections() ([]organization.ContextSelection, bool)
	}{captured, preview, capturedParent.SelectContexts(), previewParent.SelectContexts()} {
		if got, ok := result.Selections(); ok || len(got) != 0 || result.Status() != model.NotCompleted {
			t.Fatal("zero stage exposed usable contexts")
		}
	}
}

func TestStaticContextPreviewRetainsOrganizationRefusalAndRevision(t *testing.T) {
	selection, raw := scopedContextInputs(t, "unknown", contextScope)
	source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
		return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: fixtureHead}, nil
	}}
	parent, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	decoded := parent.DecodeExecution()
	if decoded.Status() != model.Passed {
		t.Fatal("preview did not reach organization consumer")
	}
	result := decoded.SelectContexts()
	if got, ok := result.Selections(); ok || len(got) != 0 || result.Status() != model.Failed || !reflect.DeepEqual(result.Execution().Model().Inputs(), parent.Inputs()) || result.Execution().Model().Inputs().Revision != fixtureHead {
		t.Fatal("preview failure promoted or lost exact revision/source")
	}
}
