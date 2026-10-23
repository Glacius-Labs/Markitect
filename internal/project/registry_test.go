package project_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const rootRegistry = "apiVersion: markitect.registry/v1\nartifacts:\n  - path: realization/library.md\n    form: doc\n    realizes: [library/library]\n    why: Library usage.\n"
const childRegistry = "apiVersion: markitect.registry/v1\nartifacts:\n  - path: realization/lending.go\n    form: code\n    realizes: [lending/borrow]\n    why: Lending operation.\n"

func censusFile(t *testing.T, raw *project.RawInputs, name string, mode snapshot.Mode, content []byte) {
	t.Helper()
	entry, err := snapshot.NewEntry(path(t, name), mode, content)
	if err != nil {
		t.Fatal(err)
	}
	raw.Census, err = snapshot.NewTree(append(raw.Census.Entries(), entry))
	if err != nil {
		t.Fatal(err)
	}
}

func registrySource(t *testing.T, selection *project.InputSelection, raw *project.RawInputs, name, content string, captured bool) {
	t.Helper()
	censusFile(t, raw, name, snapshot.Regular, []byte(content))
	if !captured {
		return
	}
	// Location is deliberately unrelated to the owner. Only the physically bound
	// RepositoryPath establishes the registry shard's Manager coordinate.
	location := project.SourceLocation{Boundary: "opaque-registry-capture", Path: path(t, "acquired/"+strings.TrimPrefix(name, ".markitect/registry/"))}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	selection.Registry = append(selection.Registry, location)
	raw.Registry = append(raw.Registry, project.SourceFile{Location: location, File: entry, RepositoryPath: path(t, name)})
}

func registryInputs(t *testing.T, captureChild bool) (project.InputSelection, project.RawInputs) {
	t.Helper()
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, "realization/library.md", snapshot.Regular, []byte("usage\r\n"))
	censusFile(t, &raw, "realization/lending.go", snapshot.Executable, []byte("implementation"))
	registrySource(t, &selection, &raw, ".markitect/registry/registry.yaml", rootRegistry, true)
	registrySource(t, &selection, &raw, ".markitect/registry/lending/registry.yaml", childRegistry, captureChild)
	return selection, raw
}

func registryModel(t *testing.T, selection project.InputSelection, raw project.RawInputs) project.CapturedModel {
	t.Helper()
	stage, err := captureService(captured(raw), nil).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || stage.Checked().Status != model.Passed {
		t.Fatalf("bound model stage failed: %v %+v", err, stage.Checked())
	}
	return stage
}

func registryDiagnostic(codes []project.RegistryDiagnostic, code string) bool {
	for _, diagnostic := range codes {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestRegistryPreparationBindsActualRootAndChildShards(t *testing.T) {
	selection, raw := registryInputs(t, true)
	stage := registryModel(t, selection, raw)
	result := stage.PrepareRegistry()
	registry, available := result.Registry()
	if result.Status() != model.Passed || !available || len(result.Diagnostics()) != 0 || len(result.Discovered()) != 2 || len(result.Records()) != 2 || len(registry.Entries()) != 2 {
		t.Fatalf("registry preparation failed: %s %+v", result.Status(), result.Diagnostics())
	}
	root, found := registry.Lookup(path(t, "realization/library.md"))
	if !found || !root.Owner().IsRoot() {
		t.Fatal("root owner was inferred incorrectly")
	}
	child, found := registry.Lookup(path(t, "realization/lending.go"))
	if name, ok := child.Owner().Package(); !found || !ok || name != "lending" {
		t.Fatal("child owner not bound to physical shard path")
	}
	if !result.Model().Inputs().Raw.Census.Equal(raw.Census) || result.Model().Inputs().Guard != stage.Inputs().Guard || child.Artifact().Form() != realization.Code {
		t.Fatal("model/raw/capture binding changed")
	}

	source := controlledSource{preview: func(context.Context, project.PreviewRequest) (project.PreviewedInputs, error) {
		return project.PreviewedInputs{Raw: raw, Head: fixtureHead}, nil
	}}
	preview, err := project.NewModelService(nil, source).PreviewModel(context.Background(), project.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	prepared := preview.PrepareRegistry()
	if prepared.Status() != model.Passed || prepared.Model().Inputs().Revision != "" {
		t.Fatalf("preview preparation failed: %+v", prepared.Diagnostics())
	}
}

func TestRegistryPreparationCannotOmitPresentCensusShard(t *testing.T) {
	selection, raw := registryInputs(t, false)
	result := registryModel(t, selection, raw).PrepareRegistry()
	registry, available := result.Registry()
	if result.Status() != model.NotCompleted || available || len(registry.Entries()) != 0 || !registryDiagnostic(result.Diagnostics(), "registry.source_uncaptured") {
		t.Fatalf("omitted shard accepted: %s %+v", result.Status(), result.Diagnostics())
	}
	if len(result.Discovered()) != 2 || len(result.Records()) != 1 || !result.Model().Inputs().Raw.Census.Equal(raw.Census) {
		t.Fatal("partial observations or discovered shard disappeared")
	}
}
