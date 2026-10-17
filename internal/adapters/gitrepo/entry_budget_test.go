package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestEntryBudgetIncludesDirectoriesAcrossExternalModels(t *testing.T) {
	_, selection, head := libraryFixture(t)
	for i := range selection.Dependencies {
		external := testkit.TempDir(t)
		copyBoundary(t, selection.Dependencies[i].Boundary, external)
		selection.Dependencies[i].Boundary = external
	}
	r, err := New(Limits{Entries: 200})
	if err != nil {
		t.Fatal(err)
	}
	service := apply.NewModelService(r, r)
	baseline, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || baseline.Compilation().Status != model.Passed {
		t.Fatalf("valid M1 capture within entry budget: %v", err)
	}
	preview, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil || preview.Compilation().Status != model.Passed {
		t.Fatalf("valid M1 preview within entry budget: %v", err)
	}
	// Each separate model scan remains below 200 entries. Their combined
	// empty directories alone exceed 200, without consuming any file bytes.
	for _, dependency := range selection.Dependencies {
		for i := range 110 {
			if err := os.Mkdir(filepath.Join(dependency.Boundary, "model", fmt.Sprintf("empty%03d", i)), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	manifest := func(source model.SourceModel) []byte {
		t.Helper()
		for _, entry := range source.Entries {
			if entry.Path == source.ManifestPath && entry.Kind == model.Regular {
				return entry.Bytes
			}
		}
		t.Fatal("acquired manifest missing")
		return nil
	}
	assertPartial := func(raw apply.RawInputs) {
		t.Helper()
		if len(raw.Dependencies) != 2 || !bytes.Equal(manifest(raw.Local.Source), manifest(baseline.Inputs().Raw.Local.Source)) {
			t.Fatal("entry refusal lost known model observations")
		}
		directories := 0
		for i, dependency := range raw.Dependencies {
			if !bytes.Equal(manifest(dependency.Source), manifest(baseline.Inputs().Raw.Dependencies[i].Source)) || len(dependency.Files.Entries()) == 0 {
				t.Fatal("entry refusal lost an acquired external manifest")
			}
			for _, entry := range dependency.Source.Entries {
				if entry.Kind == model.Directory {
					directories++
				}
			}
		}
		if directories <= 110 || directories > 200 {
			t.Fatalf("unexpected admitted partial directory inventory: %d", directories)
		}
	}
	t.Run("committed", func(t *testing.T) {
		result, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
		if !errors.Is(err, ErrLimit) || result.Compilation().Status != model.NotCompleted || result.Inputs().Head != head || result.Inputs().Guard != "" {
			t.Fatalf("aggregate model directories escaped budget: %v status=%s", err, result.Compilation().Status)
		}
		assertPartial(result.Inputs().Raw)
	})
	t.Run("working", func(t *testing.T) {
		result, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
		if !errors.Is(err, ErrLimit) || result.Compilation().Status != model.NotCompleted || result.Inputs().Head != head {
			t.Fatalf("aggregate preview directories escaped budget: %v status=%s", err, result.Compilation().Status)
		}
		assertPartial(result.Inputs().Raw)
	})
}
