package gitrepo

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

func fixturePath(t *testing.T, p string) snapshot.Path {
	t.Helper()
	result, err := snapshot.NewPath(p)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func libraryFixture(t *testing.T) (*testkit.Repo, project.InputSelection, string) {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("config", "core.filemode", "false")
	base := filepath.Join("..", "..", "..", "docs", "design", "canonical-model")
	for _, copy := range []struct{ from, to string }{
		{filepath.Join(base, "library-example", "m1"), "inputs/local"},
		{filepath.Join(base, "library-example", "vocabulary"), "inputs/vocabulary"},
		{filepath.Join(base, "minimal-language", "exchange-example", "dependencies", "core"), "inputs/core"},
	} {
		err := fs.WalkDir(os.DirFS(copy.from), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(filepath.Join(copy.from, filepath.FromSlash(p)))
			if err != nil {
				return err
			}
			repo.Write(copy.to+"/"+p, string(data))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	repo.Write(".gitignore", "ignored/\nnode_modules/\n.markitect/state/\n*.ignored\n")
	repo.Write("real/library.go", "\xef\xbb\xbfpackage library\r\n")
	selection := project.InputSelection{Repository: repo.Dir}
	selected := func(boundary string) project.ModelSelection {
		return project.ModelSelection{Boundary: filepath.Join(repo.Dir, filepath.FromSlash(boundary)), ManifestPath: fixturePath(t, "model.yaml")}
	}
	selection.Local = selected("inputs/local")
	selection.Dependencies = []project.ModelSelection{selected("inputs/vocabulary"), selected("inputs/core")}
	for i, p := range []string{"config/execution.data", "config/registry.data", "config/ignore.data", "runtime/profile.data", "runtime/instruction.txt"} {
		repo.Write(p, "\xef\xbb\xbforiginal\x00\r\n"+p)
		location := project.SourceLocation{Boundary: repo.Dir, Path: fixturePath(t, p)}
		switch i {
		case 0:
			selection.Execution = []project.SourceLocation{location}
		case 1:
			selection.Registry = []project.SourceLocation{location}
		case 2:
			selection.Ignore = []project.SourceLocation{location}
		case 3:
			selection.RuntimeProfiles = []project.SourceLocation{location}
		case 4:
			selection.Instructions = []project.SourceLocation{location}
		}
	}
	head := repo.Commit("Actual M1, vocabulary and Core source fixture")
	repo.Git("update-index", "--chmod=+x", "real/library.go")
	head = repo.Commit("Retain executable Git mode")
	return repo, selection, head
}

func adapter(t *testing.T) *Repository {
	t.Helper()
	r, err := New(Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
