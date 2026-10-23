package project_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func singleRegistry(t *testing.T, name, content string) (project.InputSelection, project.RawInputs) {
	t.Helper()
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, "realization/lending.go", snapshot.Regular, []byte("actual artifact"))
	registrySource(t, &selection, &raw, name, content, true)
	return selection, raw
}

func TestRegistryKnownFailuresRetainFactsAndNeverExposePartialRegistry(t *testing.T) {
	for _, kind := range []string{"stale owner", "imported owner", "duplicate basis", "duplicate ownership", "unknown key", "payload owner", "unknown form", "dangling basis", "short basis", "case changed basis", "artifact missing", "artifact path", "empty basis", "blank why", "layout", "duplicate source", "wrong source area"} {
		t.Run(kind, func(t *testing.T) {
			name, content, code := ".markitect/registry/lending/registry.yaml", childRegistry, "registry.artifact"
			switch kind {
			case "stale owner":
				name, code = ".markitect/registry/removed/registry.yaml", "registry.unknown_owner"
			case "imported owner":
				name, code = ".markitect/registry/software/registry.yaml", "registry.unknown_owner"
			case "duplicate basis":
				content = strings.Replace(content, "[lending/borrow]", "[lending/borrow, lending/borrow]", 1)
			case "duplicate ownership":
				content += strings.TrimPrefix(childRegistry, "apiVersion: markitect.registry/v1\nartifacts:\n")
				code = "registry.records_invalid"
			case "unknown key":
				content += "unrecognized: value\n"
				code = "registry.unknown_key"
			case "payload owner":
				content += "owner: library\n"
				code = "registry.unknown_key"
			case "unknown form":
				content = strings.Replace(content, "form: code", "form: invented", 1)
			case "dangling basis":
				content = strings.Replace(content, "lending/borrow", "lending/absent", 1)
				code = "registry.definition_missing"
			case "short basis":
				content = strings.Replace(content, "lending/borrow", "borrow", 1)
				code = "registry.definition_address"
			case "case changed basis":
				content = strings.Replace(content, "lending/borrow", "Lending/borrow", 1)
				code = "registry.definition_missing"
			case "artifact missing":
				content = strings.Replace(content, "realization/lending.go", "absent.go", 1)
				code = "registry.artifact_missing"
			case "artifact path":
				content = strings.Replace(content, "realization/lending.go", "../outside.go", 1)
				code = "registry.artifact_path"
			case "empty basis":
				content = strings.Replace(content, "[lending/borrow]", "[]", 1)
			case "blank why":
				content = strings.Replace(content, "Lending operation.", "'  '", 1)
			case "layout":
				name, code = ".markitect/registry/notes.yaml", "registry.layout"
			case "duplicate source":
				code = "registry.duplicate_source"
			case "wrong source area":
				name, code = "inputs/registry.yaml", "registry.source_path"
			}
			selection, raw := singleRegistry(t, name, content)
			if kind == "duplicate source" {
				selection.Registry = append(selection.Registry, selection.Registry[0])
				raw.Registry = append(raw.Registry, raw.Registry[0])
			}
			result := registryModel(t, selection, raw).PrepareRegistry()
			registry, available := result.Registry()
			if result.Status() != model.Failed || available || len(registry.Entries()) != 0 || !registryDiagnostic(result.Diagnostics(), code) {
				t.Fatalf("failure was hidden: %s %+v", result.Status(), result.Diagnostics())
			}
			if !reflect.DeepEqual(result.Model().Inputs().Raw, raw) {
				t.Fatal("original invalid/partial input lost")
			}
			if kind == "duplicate basis" && (len(result.Records()) != 1 || len(result.Records()[0].Realizes) != 2) {
				t.Fatal("duplicate basis collapsed before diagnostics")
			}
		})
	}
}

func TestRegistryVersionAndYAMLShapeAreClosed(t *testing.T) {
	for _, content := range []string{
		"", "null\n", "[]\n", "apiVersion: markitect.registry/v1\n", "apiVersion: markitect.registry/v1\nartifacts: null\n",
		"apiVersion: unsupported\nartifacts: []\n", "apiVersion: markitect.registry/v1\napiVersion: markitect.registry/v1\nartifacts: []\n",
		"apiVersion: markitect.registry/v1\nartifacts: []\n---\nartifacts: []\n",
		"apiVersion: markitect.registry/v1\nartifacts: &anchor []\n", "apiVersion: !custom markitect.registry/v1\nartifacts: []\n",
		"apiVersion: markitect.registry/v1\nartifacts: [null]\n", "apiVersion: markitect.registry/v1\nartifacts: [unterminated\n",
		strings.Replace(childRegistry, "path: realization/lending.go", "path: 123", 1),
		strings.Replace(childRegistry, "[lending/borrow]", "[123]", 1),
		strings.Replace(childRegistry, "form: code", "form: code\n    form: doc", 1),
		strings.Replace(childRegistry, "why: Lending operation.", "why: Lending operation.\n    unknown: value", 1),
	} {
		t.Run(content, func(t *testing.T) {
			selection, raw := singleRegistry(t, ".markitect/registry/lending/registry.yaml", content)
			result := registryModel(t, selection, raw).PrepareRegistry()
			if _, ok := result.Registry(); result.Status() != model.Failed || ok || len(result.Diagnostics()) == 0 {
				t.Fatalf("invalid registry accepted: %s %+v", result.Status(), result.Diagnostics())
			}
		})
	}
}

func TestEmptyRegistryIsOnlyARegistryStageResult(t *testing.T) {
	for _, emptyFile := range []bool{false, true} {
		selection, raw := libraryInputs(t)
		if emptyFile {
			registrySource(t, &selection, &raw, ".markitect/registry/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n", true)
		}
		result := registryModel(t, selection, raw).PrepareRegistry()
		registry, ok := result.Registry()
		if result.Status() != model.Passed || !ok || len(registry.Entries()) != 0 {
			t.Fatalf("empty registry stage refused: %+v", result.Diagnostics())
		}
	}
}

func TestFailedAndIncompleteModelStagesDoNotPrepareARegistry(t *testing.T) {
	if result := (project.CapturedModel{}).PrepareRegistry(); result.Status() != model.NotCompleted {
		t.Fatal("zero capture passed")
	}
	if result := (project.PreviewedModel{}).PrepareRegistry(); result.Status() != model.NotCompleted {
		t.Fatal("zero preview passed")
	}
	if result := (project.CapturedRegistryPreparation{}); result.Status() != model.NotCompleted {
		t.Fatal("zero registry stage passed")
	}
	for _, kind := range []string{"language", "budget", "source bytes", "source mode"} {
		t.Run(kind, func(t *testing.T) {
			selection, raw := registryInputs(t, true)
			limits := model.Limits{}
			switch kind {
			case "language":
				raw.Local.Source.Entries = append(raw.Local.Source.Entries, raw.Local.Source.Entries[0])
			case "budget":
				limits.Entries = 1
			case "source bytes":
				raw.Registry[0].File, _ = snapshot.NewEntry(raw.Registry[0].File.Path(), snapshot.Regular, []byte("drift"))
			case "source mode":
				raw.Registry[0].File, _ = snapshot.NewEntry(raw.Registry[0].File.Path(), snapshot.Executable, raw.Registry[0].File.Bytes())
			}
			stage, err := captureService(captured(raw), nil).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, limits)
			if strings.HasPrefix(kind, "source") && !errors.Is(err, project.ErrIncompleteInputs) {
				t.Fatal("source drift was not observed by bound model stage")
			}
			result := stage.PrepareRegistry()
			if _, ok := result.Registry(); result.Status() != model.NotCompleted || ok || !registryDiagnostic(result.Diagnostics(), "registry.model_not_passed") || !reflect.DeepEqual(result.Model().Inputs().Raw, raw) {
				t.Fatalf("non-passed model prepared registry: %+v", result.Diagnostics())
			}
		})
	}
}

func TestUncapturedCensusShardAndKnownInvalidOwnerRemainDistinct(t *testing.T) {
	selection, raw := registryInputs(t, false)
	// This controlled Census represents files even when Gitignore would exclude
	// them from cleanliness. It is not proof of actual Git repository acquisition.
	censusFile(t, &raw, ".gitignore", snapshot.Regular, []byte(".markitect/registry/lending/\n"))
	result := registryModel(t, selection, raw).PrepareRegistry()
	if result.Status() != model.NotCompleted || !registryDiagnostic(result.Diagnostics(), "registry.source_uncaptured") {
		t.Fatal("Gitignore text hid actual shard")
	}
	registrySource(t, &selection, &raw, ".markitect/registry/removed/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n", false)
	result = registryModel(t, selection, raw).PrepareRegistry()
	if result.Status() != model.Failed || !registryDiagnostic(result.Diagnostics(), "registry.unknown_owner") || !registryDiagnostic(result.Diagnostics(), "registry.source_uncaptured") {
		t.Fatalf("failure/incompleteness observations collapsed: %+v", result.Diagnostics())
	}
}
