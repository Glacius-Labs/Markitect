package project_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestRegistryRejectsExpandedCustomTags(t *testing.T) {
	for _, tc := range []struct {
		name, source string
	}{
		{"root mapping", "!<tag:example.org,2026:registry>\napiVersion: markitect.registry/v1\nartifacts: []\n"},
		{"artifacts sequence", "apiVersion: markitect.registry/v1\nartifacts: !<tag:example.org,2026:registry> []\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := registryInputs(t, true)
			registryPath := path(t, ".markitect/registry/registry.yaml")
			replacedSource := false
			for i := range raw.Registry {
				if raw.Registry[i].RepositoryPath == registryPath {
					file, err := snapshot.NewEntry(raw.Registry[i].File.Path(), snapshot.Regular, []byte(tc.source))
					if err != nil {
						t.Fatal(err)
					}
					raw.Registry[i].File = file
					replacedSource = true
				}
			}
			if !replacedSource {
				t.Fatal("root registry source fixture missing")
			}
			entries := raw.Census.Entries()
			replaced := make([]snapshot.Entry, 0, len(entries))
			for _, entry := range entries {
				if entry.Path() != registryPath {
					replaced = append(replaced, entry)
				}
			}
			censusFile, err := snapshot.NewEntry(registryPath, snapshot.Regular, []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			replaced = append(replaced, censusFile)
			raw.Census, err = snapshot.NewTree(replaced)
			if err != nil {
				t.Fatal(err)
			}

			stage, err := captureService(captured(raw), nil).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
			if err != nil || stage.Checked().Status != model.Passed {
				t.Fatalf("model fixture failed: %v %+v", err, stage.Checked())
			}
			prepared := stage.PrepareRegistry()
			registry, available := prepared.Registry()
			if prepared.Status() != model.Failed || available || len(registry.Entries()) != 0 || !registryDiagnostic(prepared.Diagnostics(), "registry.yaml_feature") {
				t.Fatalf("expanded custom tag was not rejected: status=%s available=%v entries=%d diagnostics=%+v", prepared.Status(), available, len(registry.Entries()), prepared.Diagnostics())
			}
			if !reflect.DeepEqual(prepared.Model().Inputs(), stage.Inputs()) {
				t.Fatal("registry refusal lost its exact parent model/source observations")
			}
			found := false
			for _, source := range prepared.Model().Inputs().Raw.Registry {
				if source.RepositoryPath == registryPath {
					found = string(source.File.Bytes()) == tc.source
				}
			}
			if !found {
				t.Fatal("expanded-tag source bytes were not retained")
			}
		})
	}
}
