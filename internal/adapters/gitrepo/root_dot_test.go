package gitrepo

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
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
	service := project.NewModelService(r, r)
	captured, err := service.CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || captured.Checked().Status != model.Passed {
		t.Fatalf("root dot capture: %v %+v", err, captured.Checked().Diagnostics)
	}
	working, err := service.PreviewModel(context.Background(), project.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil || working.Checked().Status != model.Passed {
		t.Fatalf("root dot working: %v %+v", err, working.Checked().Diagnostics)
	}
	fixed, err := service.PreviewModel(context.Background(), project.PreviewRequest{Selection: selection, Revision: head}, model.Limits{})
	if err != nil || fixed.Checked().Status != model.Passed {
		t.Fatalf("root dot fixed: %v %+v", err, fixed.Checked().Diagnostics)
	}
	for _, source := range []project.ModelSource{captured.Inputs().Raw.Local, working.Inputs().Raw.Local, fixed.Inputs().Raw.Local} {
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

func TestRunRepositoryRootModelSnapshotsKeepEmptyPackagesOutsideGitAdministration(t *testing.T) {
	repo := testkit.NewRepo(t)
	repo.Write("model.yaml", "formatVersion: 1\nid: root-run\nroot: .\ndependencies: {markitect-core: '1.1.0'}\n")
	repo.Write("model-lock.yaml", "formatVersion: 1\nmodels:\n  markitect-core: {version: '1.1.0', digest: 'sha256:9552a7cdd560d8822854c286cd74d7dbb5d4277ee1fd16b25ef275213868e043'}\n")
	core := testkit.TempDir(t)
	original := "../../../testdata/core"
	if err := fs.WalkDir(os.DirFS(original), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(filepath.Join(original, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		target := filepath.Join(core, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	repo.Commit("Root model manifest")
	if err := os.MkdirAll(filepath.Join(repo.Dir, "empty/deep"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tree, err := withDirectories(ctx, repo.Dir, repo.Git("rev-parse", "HEAD^{tree}"), []string{"empty", "empty/deep"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := localCommand(ctx, repo.Dir, []byte("Preserve root Package inventory\n"), "commit-tree", tree, "-p", repo.Git("rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	repo.Git("reset", "--mixed", id)
	selection := project.InputSelection{Repository: repo.Dir, Local: project.ModelSelection{Boundary: ".", ManifestPath: fixturePath(t, "model.yaml")}, Dependencies: []project.ModelSelection{{Boundary: core, ManifestPath: fixturePath(t, "model.yaml")}}}
	reader := adapter(t)
	captured, err := project.NewModelService(reader, reader).CaptureModel(ctx, project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || captured.Checked().Status != model.Passed {
		t.Fatalf("actual root-repository model failed: %v %+v", err, captured.Checked().Diagnostics)
	}
	roots := modelDirectoryRoots(captured.Inputs().Raw)
	linuxRoots := t.Run("Linux root selects all descendants", func(t *testing.T) {
		if !reflect.DeepEqual(roots, []string{""}) {
			t.Fatalf("repository root not normalized for native descendant matching: %q", roots)
		}
	})
	var dirs []string
	localRoots := t.Run("Local inventory excludes private Git", func(t *testing.T) {
		dirs, err = localModelDirectories(ctx, repo.Dir, roots)
		if err != nil {
			t.Fatal(err)
		}
		for _, dir := range dirs {
			if technical(dir) {
				t.Fatalf("private Git administration became model inventory: %q", dir)
			}
		}
		if !reflect.DeepEqual(dirs, []string{"empty", "empty/deep"}) {
			t.Fatalf("empty Package inventory changed: %q", dirs)
		}
	})
	if !linuxRoots || !localRoots {
		return
	}
	fixed, err := snapshotGit(ctx, repo.Dir, localCommand, dirs, localExecutables(repo.Dir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = localCommand(ctx, repo.Dir, nil, "fsck", "--full", "--no-reflogs"); err != nil {
		t.Fatal(err)
	}
	destination := testkit.TempDir(t)
	if _, err = localCommand(ctx, destination, nil, "clone", "--quiet", repo.Dir, "."); err != nil {
		t.Fatal(err)
	}
	if err = restoreDirectories(ctx, destination, fixed.ID); err != nil {
		t.Fatal(err)
	}
	selection.Repository = destination
	reopened, err := project.NewModelService(reader, reader).PreviewModel(ctx, project.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil || reopened.Checked().Status != model.Passed || !reflect.DeepEqual(captured.Inputs().Raw.Local.Source, reopened.Inputs().Raw.Local.Source) {
		t.Fatalf("root model/empty Packages changed after Git reopening: %v %+v", err, reopened.Checked().Diagnostics)
	}
	for _, dir := range []string{"empty", "empty/deep"} {
		if info, err := os.Stat(filepath.Join(destination, dir)); err != nil || !info.IsDir() {
			t.Fatal("physical empty Package lost", dir, err)
		}
	}
}

func TestRunDirectoryInventoryUsesAcquisitionAdministrationExclusions(t *testing.T) {
	repo := testkit.NewRepo(t)
	for _, dir := range []string{"empty/child", "empty/.git/private", ".markitect/state/private", ".markitect/kept"} {
		if err := os.MkdirAll(filepath.Join(repo.Dir, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	dirs, err := localModelDirectories(context.Background(), repo.Dir, []string{""})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{".markitect", ".markitect/kept", "empty", "empty/child"}
	if !reflect.DeepEqual(dirs, expected) {
		t.Fatalf("administrative directory inventory differs from acquisition: %q", dirs)
	}
}
