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

func fileCheckInputs(t *testing.T) (apply.InputSelection, apply.RawInputs) {
	t.Helper()
	selection, raw := executionInputs(t, documentedExecution(t))
	censusFile(t, &raw, "realization/library.md", snapshot.Regular, []byte("Library usage.\r\n"))
	censusFile(t, &raw, "realization/lending.go", snapshot.Executable, []byte("Lending implementation."))
	registrySource(t, &selection, &raw, ".markitect/registry/registry.yaml", rootRegistry, true)
	registrySource(t, &selection, &raw, ".markitect/registry/lending/registry.yaml", childRegistry, true)
	return selection, raw
}

func TestM1FileCheckAccountsActualCensusAndRetainsBoundParents(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	parent := registryModel(t, selection, raw).PrepareRegistry()
	result := parent.CheckFiles()
	accounting, observed := result.Accounting()
	classes, complete := result.Classifications()
	protections, protected := result.Protections()
	if result.Status() != model.Passed || !observed || accounting.Status() != realization.Accounted || len(accounting.Findings()) != 0 || len(result.Diagnostics()) != 0 || !complete || !protected || len(protections) == 0 {
		t.Fatalf("actual M1 file check failed: %s %+v %+v", result.Status(), result.Diagnostics(), accounting.Findings())
	}
	if accounting.TreeID() != raw.Census.Digest() || len(accounting.Files()) != len(raw.Census.Entries()) || len(classes)+2 != len(accounting.Files()) {
		t.Fatal("accounting/classification did not cover the actual fixed census")
	}
	for _, file := range accounting.Files() {
		name := file.Entry().Path().String()
		want := realization.CanonicalInput
		switch name {
		case ".markitect/execution.yaml", ".markitect/registry/registry.yaml", ".markitect/registry/lending/registry.yaml":
			want = realization.MarkitectInput
		case "realization/library.md", "realization/lending.go":
			want = realization.RegisteredArtifact
		}
		if file.Class() != want {
			t.Fatalf("%s classified as %s, expected %s", name, file.Class(), want)
		}
	}
	if !reflect.DeepEqual(result.RegistryPreparation().Model().Inputs(), parent.Model().Inputs()) || result.RegistryPreparation().Model().Inputs().Guard == "" || !reflect.DeepEqual(result.RegistryPreparation().Records(), parent.Records()) {
		t.Fatal("capture model/registry/raw/guard binding changed")
	}
	for _, revision := range []string{"", fixtureHead} {
		source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
			return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: revision}, nil
		}}
		preview, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		prepared := preview.PrepareRegistry().CheckFiles()
		got, available := prepared.Accounting()
		if prepared.Status() != model.Passed || !available || !reflect.DeepEqual(got, accounting) || !reflect.DeepEqual(prepared.RegistryPreparation().Model().Inputs(), preview.Inputs()) || prepared.RegistryPreparation().Model().Inputs().Revision != revision {
			t.Fatalf("preview binding changed: %+v", prepared.Diagnostics())
		}
	}
}
