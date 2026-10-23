package apply_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestRepositoryDependencyAreaBindsEveryFileToCensus(t *testing.T) {
	for _, kind := range []string{"external", "named area", "repository root", "missing file", "different bytes", "different mode"} {
		t.Run(kind, func(t *testing.T) {
			selection, raw := libraryInputs(t)
			dependency := &raw.Dependencies[0]
			if kind != "external" {
				dependency.RepositoryArea = area(t, "vendor/vocabulary")
				if kind == "repository root" {
					dependency.RepositoryArea = snapshot.RootDirectory()
				}
				entries := raw.Census.Entries()
				for i, file := range dependency.Files.Entries() {
					p := file.Path().String()
					if !dependency.RepositoryArea.IsRoot() {
						p = dependency.RepositoryArea.String() + "/" + p
					}
					mode, bytes := file.Mode(), file.Bytes()
					if i == 0 {
						switch kind {
						case "missing file":
							continue
						case "different bytes":
							bytes = []byte("different dependency census bytes")
						case "different mode":
							mode = snapshot.Executable
						}
					}
					entry, err := snapshot.NewEntry(path(t, p), mode, bytes)
					if err != nil {
						t.Fatal(err)
					}
					entries = append(entries, entry)
				}
				var err error
				raw.Census, err = snapshot.NewTree(entries)
				if err != nil {
					t.Fatal(err)
				}
			}
			result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
			switch kind {
			case "external", "named area", "repository root":
				if err != nil || result.Checked().Status != model.Passed {
					t.Fatalf("valid dependency acquisition refused: %v %+v", err, result.Checked())
				}
			default:
				if !errors.Is(err, apply.ErrIncompleteInputs) || result.Checked().Status != model.NotCompleted {
					t.Fatalf("dependency census mismatch accepted: %v %+v", err, result.Checked())
				}
			}
			if !reflect.DeepEqual(result.Inputs().Raw, raw) {
				t.Fatal("explicit areas or actual partial observations lost")
			}
		})
	}
}

func TestLocalModelRepositoryAreaMayBeRoot(t *testing.T) {
	selection, raw := libraryInputs(t)
	raw.Local.RepositoryArea = snapshot.RootDirectory()
	raw.Census = raw.Local.Files
	result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || result.Checked().Status != model.Passed || !result.Inputs().Raw.Local.RepositoryArea.IsRoot() {
		t.Fatalf("explicit root area lost: %v %+v", err, result.Checked())
	}
}
