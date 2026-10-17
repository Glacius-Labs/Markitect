package gitrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
)

func TestRedirectedManifestsRootsAndDefinitionFilesAreNotAcquired(t *testing.T) {
	for _, kind := range []string{"manifest", "root", "definition", "junction"} {
		t.Run(kind, func(t *testing.T) {
			if kind == "junction" && runtime.GOOS != "windows" {
				t.Skip("Windows junction")
			}
			_, selection, _ := libraryFixture(t)
			p := filepath.Join(selection.Local.Boundary, "model.yaml")
			if kind == "root" || kind == "junction" {
				p = filepath.Join(selection.Local.Boundary, "model")
			}
			if kind == "definition" {
				p = filepath.Join(selection.Local.Boundary, "model", "lending", "loan.yaml")
			}
			real := p + "-real"
			if err := os.Rename(p, real); err != nil {
				t.Fatal(err)
			}
			if kind == "junction" {
				if out, err := exec.Command("cmd", "/c", "mklink", "/J", p, real).CombinedOutput(); err != nil {
					t.Skipf("junction unavailable: %v %s", err, out)
				}
			} else if err := os.Symlink(real, p); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			result, err := adapter(t).CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection})
			if !errors.Is(err, ErrUnsafe) || result.Head == "" {
				t.Fatalf("redirect acquired: %v", err)
			}
			if kind == "manifest" && len(result.Raw.Local.Files.Entries()) != 0 {
				t.Fatal("redirected manifest bytes were acquired")
			}
		})
	}
}

func TestExecutionBoundaryCollisionIsRejectedInEveryView(t *testing.T) {
	repo, selection, _ := libraryFixture(t)
	if err := os.Rename(filepath.Join(repo.Dir, "config"), filepath.Join(repo.Dir, "straße")); err != nil {
		t.Fatal(err)
	}
	repo.Write("strasse/marker", "colliding acquisition area")
	selection.Execution[0].Boundary = filepath.Join(repo.Dir, "straße")
	selection.Execution[0].Path = fixturePath(t, "execution.data")
	selection.Registry[0].Path = fixturePath(t, "straße/registry.data")
	selection.Ignore[0].Path = fixturePath(t, "straße/ignore.data")
	head := repo.Commit("Explicit source area with D145 ambiguity")
	r := adapter(t)
	for _, mode := range []string{"working", "committed", "fixed"} {
		var err error
		if mode == "committed" {
			result, e := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
			err = e
			if result.Guard != "" {
				t.Fatal("ambiguous source issued guard")
			}
		} else {
			revision := ""
			if mode == "fixed" {
				revision = head
			}
			_, err = r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision})
		}
		if !errors.Is(err, ErrUnsafe) {
			t.Fatalf("%s source collision: %v", mode, err)
		}
	}
}
