package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

func nativeRepositoryPaths(t *testing.T, root string) (string, string) {
	t.Helper()
	convert := func(path string, call func(*uint16, *uint16, uint32) (uint32, error)) string {
		t.Helper()
		input, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			t.Fatal(err)
		}
		buffer := make([]uint16, 32768)
		n, err := call(input, &buffer[0], uint32(len(buffer)))
		if err != nil || n == 0 || n >= uint32(len(buffer)) {
			t.Fatalf("native path conversion: length=%d error=%v", n, err)
		}
		return filepath.Clean(syscall.UTF16ToString(buffer[:n]))
	}
	long := convert(root, syscall.GetLongPathName)
	short := convert(long, syscall.GetShortPathName)
	rel, err := filepath.Rel(long, short)
	if err != nil {
		t.Fatal(err)
	}
	if rel == "." {
		t.Skip("fixture filesystem provides no distinct native 8.3 path")
	}
	return long, short
}

func repositoryAliasSelection(t *testing.T, selection project.InputSelection, original, root string) project.InputSelection {
	t.Helper()
	rewrite := func(boundary string) string {
		rel, err := filepath.Rel(original, boundary)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return boundary // Explicit external Runtime inputs remain external.
		}
		return filepath.Join(root, rel)
	}
	selection.Repository = root
	selection.Local.Boundary = rewrite(selection.Local.Boundary)
	selection.Dependencies = append([]project.ModelSelection(nil), selection.Dependencies...)
	for i := range selection.Dependencies {
		selection.Dependencies[i].Boundary = rewrite(selection.Dependencies[i].Boundary)
	}
	for _, group := range []*[]project.SourceLocation{&selection.Execution, &selection.Registry, &selection.Ignore, &selection.Instructions, &selection.RuntimeProfiles} {
		*group = append([]project.SourceLocation(nil), (*group)...)
		for i := range *group {
			(*group)[i].Boundary = rewrite((*group)[i].Boundary)
		}
	}
	return selection
}

func TestNativeRepositoryAliasCaptureAndPreviews(t *testing.T) {
	repo, original, historical := fileCheckSeamFixture(t)
	const current = "Current committed instructions.\r\n"
	repo.Write("ops/manager.md", current)
	head := repo.Commit("Advance instructions after historical revision")
	_, alias := nativeRepositoryPaths(t, repo.Dir)
	for _, relative := range []bool{false, true} {
		for _, mode := range []string{"capture", "working", "fixed"} {
			name := mode + "/absolute-model"
			if relative {
				name = mode + "/relative-model"
			}
			t.Run(name, func(t *testing.T) {
				selection := repositoryAliasSelection(t, original, repo.Dir, alias)
				if relative {
					selection.Local.Boundary = "inputs/local"
				}
				r := adapter(t)
				var raw project.RawInputs
				if mode == "capture" {
					result, err := r.Capture(context.Background(), project.CaptureRequest{Selection: selection})
					if err != nil || result.Head != head || result.Guard == "" {
						t.Fatalf("alias Capture: head=%s guard=%q error=%v", result.Head, result.Guard, err)
					}
					raw = result.Raw
				} else {
					request := project.PreviewRequest{Selection: selection}
					if mode == "fixed" {
						request.Revision = historical
					}
					result, err := r.CapturePreview(context.Background(), request)
					if err != nil || result.Head != head || result.Revision != request.Revision {
						t.Fatalf("alias %s preview: head=%s revision=%s error=%v", mode, result.Head, result.Revision, err)
					}
					raw = result.Raw
				}
				if raw.Local.RepositoryArea.String() != "inputs/local" || len(raw.Execution) != 1 || raw.Execution[0].RepositoryPath.String() != ".markitect/execution.yaml" {
					t.Fatal("alias lost model or mandatory execution repository coordinates")
				}
				want := []byte(current)
				if mode == "fixed" {
					want = []byte(fileSeamInstruction("ops/manager.md"))
				}
				if len(raw.Instructions) == 0 || !bytes.Equal(raw.Instructions[0].File.Bytes(), want) {
					t.Fatal("alias preview lost historical/current instruction bytes")
				}
			})
		}
	}
}

func TestNativeRepositoryAliasKeepsExternalFixedBoundariesRefused(t *testing.T) {
	repo, original, head := libraryFixture(t)
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "runtime.yaml"), []byte("fixed external"), 0600); err != nil {
		t.Fatal(err)
	}
	original.RuntimeProfiles = []project.SourceLocation{{Boundary: external, Path: fixturePath(t, "runtime.yaml")}}
	long, alias := nativeRepositoryPaths(t, repo.Dir)
	r := adapter(t)
	selection := repositoryAliasSelection(t, original, repo.Dir, alias)
	result, err := r.CapturePreview(context.Background(), project.PreviewRequest{Selection: selection, Revision: head})
	if !errors.Is(err, ErrUnsafe) || len(result.Raw.RuntimeProfiles) != 1 || len(result.Raw.RuntimeProfiles[0].File.Bytes()) != 0 || result.Raw.RuntimeProfiles[0].RepositoryPath.String() != "" {
		t.Fatalf("external historical Runtime boundary was accepted: %v", err)
	}
	// A boundary under an unselected alias is not an inferred local coordinate.
	selection = repositoryAliasSelection(t, original, repo.Dir, long)
	selection.Local.Boundary = filepath.Join(alias, "inputs/local")
	if _, err := r.CapturePreview(context.Background(), project.PreviewRequest{Selection: selection, Revision: head}); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("unselected alias was inferred: %v", err)
	}
}
