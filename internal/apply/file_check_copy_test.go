package apply_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestFileCheckOwnedCopiesPreserveCompleteSetsAndParents(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	result := checkedFiles(t, selection, raw)
	classes, _ := result.Classifications()
	firstClass := classes[0]
	classes[0] = realization.Classification{}
	if fresh, _ := result.Classifications(); fresh[0] != firstClass {
		t.Fatal("classification accessor leaked its mutable slice")
	}
	protections, _ := result.Protections()
	firstProtection := protections[0]
	protections[0] = realization.Protection{}
	if fresh, _ := result.Protections(); fresh[0] != firstProtection {
		t.Fatal("protection accessor leaked its mutable slice")
	}
	accounting, _ := result.Accounting()
	files := accounting.Files()
	firstFile := files[0]
	files[0] = realization.File{}
	if fresh, _ := result.Accounting(); !reflect.DeepEqual(fresh.Files()[0], firstFile) {
		t.Fatal("accounting accessor leaked mutable state")
	}
	links := accounting.Artifacts(model.Identity{Package: "library", ID: "library"})
	if len(links) != 1 {
		t.Fatal("fixture lacks artifact link")
	}
	links[0] = path(t, "changed.txt")
	if accounting.Artifacts(model.Identity{Package: "library", ID: "library"})[0].String() != "realization/library.md" {
		t.Fatal("accounting links leaked")
	}
	inputs := result.RegistryPreparation().Model().Inputs()
	inputs.Raw.Execution[0].RepositoryPath = path(t, "changed.yaml")
	for i := range inputs.Raw.Local.Source.Entries {
		if len(inputs.Raw.Local.Source.Entries[i].Bytes) > 0 {
			inputs.Raw.Local.Source.Entries[i].Bytes[0] = '!'
			break
		}
	}
	if !reflect.DeepEqual(result.RegistryPreparation().Model().Inputs().Raw, raw) {
		t.Fatal("parent source accessor leaked")
	}
}

func TestFileCheckRetainedFailureCopiesAndPreviewBinding(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	policy := "apiVersion: markitect.ignore/v1\nfiles: []\nfiles: []\n"
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, policy, true)
	censusFile(t, &raw, "unexplained.txt", snapshot.Regular, []byte("retained observation"))
	result := checkedFiles(t, selection, raw)
	diagnostics := result.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Position == nil {
		t.Fatal("fixture lacks positioned diagnostics")
	}
	expectedLine, expectedCode := diagnostics[0].Position.Line, diagnostics[0].Code
	diagnostics[0].Position.Line, diagnostics[0].Code = 99, "changed"
	if fresh := result.Diagnostics(); fresh[0].Code != expectedCode || fresh[0].Position.Line != expectedLine {
		t.Fatal("diagnostic copy changed retained source observation")
	}
	accounting, ran := result.Accounting()
	if !ran || len(accounting.Findings()) != 1 {
		t.Fatal("actual failed accounting missing")
	}
	findings := accounting.Findings()
	expected := findings[0]
	findings[0].Code = "changed"
	if accounting.Findings()[0] != expected {
		t.Fatal("accounting finding slice leaked")
	}
	source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
		return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: fixtureHead}, nil
	}}
	preview, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	prepared := preview.PrepareRegistry().CheckFiles()
	if prepared.Status() != model.Failed || !reflect.DeepEqual(prepared.RegistryPreparation().Model().Inputs(), preview.Inputs()) || prepared.RegistryPreparation().Model().Inputs().Revision != fixtureHead {
		t.Fatal("preview failure/source/revision changed")
	}
}

func TestFileCheckZeroReportsExposeNoAccountingOrCompleteSets(t *testing.T) {
	var captured apply.CapturedFileCheck
	var preview apply.PreviewFileCheck
	var capturedParent apply.CapturedRegistryPreparation
	var previewParent apply.PreviewRegistryPreparation
	for _, result := range []interface {
		Status() model.Status
		Accounting() (realization.Accounting, bool)
		Classifications() ([]realization.Classification, bool)
		Protections() ([]realization.Protection, bool)
	}{captured, preview, capturedParent.CheckFiles(), previewParent.CheckFiles()} {
		if result.Status() != model.NotCompleted {
			t.Fatal("zero report passed")
		}
		if _, ok := result.Accounting(); ok {
			t.Fatal("zero report fabricated actual accounting")
		}
		if _, ok := result.Classifications(); ok {
			t.Fatal("zero report exposed complete classifications")
		}
		if _, ok := result.Protections(); ok {
			t.Fatal("zero report exposed complete protections")
		}
	}
}
