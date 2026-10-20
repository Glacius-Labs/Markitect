package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestActualLibraryRootDotPreservesInventoryAndExcludesStateMarker(t *testing.T) {
	repo, selection, _ := libraryFixture(t)
	area := filepath.Join(repo.Dir, ".markitect")
	if err := os.Rename(selection.Local.Boundary, area); err != nil {
		t.Fatal(err)
	}
	selection.Local.Boundary = area
	children, err := os.ReadDir(filepath.Join(area, "model"))
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range children {
		if err := os.Rename(filepath.Join(area, "model", child.Name()), filepath.Join(area, child.Name())); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(area, "model")); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(area, "model.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(".markitect/model.yaml", strings.Replace(string(manifest), "root: model", "root: .", 1))
	repo.Write(".markitect/state/trace.yaml", "private journal must never become a model package")
	repo.Git("add", "--force", ".markitect/state/trace.yaml")
	head := repo.Commit("Same actual library model rooted at manifest directory")
	r := adapter(t)
	service := apply.NewModelService(r, r)
	captured, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || captured.Checked().Status != model.Passed {
		t.Fatalf("root dot capture: %v %+v", err, captured.Checked().Diagnostics)
	}
	working, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil || working.Checked().Status != model.Passed {
		t.Fatalf("root dot working: %v %+v", err, working.Checked().Diagnostics)
	}
	fixed, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: head}, model.Limits{})
	if err != nil || fixed.Checked().Status != model.Passed {
		t.Fatalf("root dot fixed: %v %+v", err, fixed.Checked().Diagnostics)
	}
	for _, source := range []apply.ModelSource{captured.Inputs().Raw.Local, working.Inputs().Raw.Local, fixed.Inputs().Raw.Local} {
		if !source.Root.IsRoot() || source.RepositoryArea.String() != ".markitect" {
			t.Fatal("root dot coordinates changed")
		}
		for _, entry := range source.Source.Entries {
			if entry.Path == ".markitect" || entry.Path == "state" || strings.HasPrefix(entry.Path, "state/") {
				t.Fatalf("invented boundary marker or captured state marker: %+v", entry)
			}
		}
	}
	if !reflect.DeepEqual(working.Inputs().Raw.Local.Source, fixed.Inputs().Raw.Local.Source) || !reflect.DeepEqual(working.Checked().Models[0].Packages, fixed.Checked().Models[0].Packages) {
		t.Fatal("root dot Working/Fixed inventories differ")
	}
}
