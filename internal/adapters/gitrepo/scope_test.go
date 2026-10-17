package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestCommittedSourceProtectsSelectedBytesAndIncludesIgnoredSiblings(t *testing.T) {
	for _, kind := range []string{"tracked", "selected-assume-unchanged", "model-assume-unchanged", "ignored-model-extra", "selected-uncommitted-ignored", "nonignored-untracked", "staged"} {
		t.Run(kind, func(t *testing.T) {
			repo, selection, head := libraryFixture(t)
			switch kind {
			case "tracked":
				repo.Write("real/library.go", "changed\r\n")
			case "selected-assume-unchanged":
				repo.Git("update-index", "--assume-unchanged", "config/execution.data")
				repo.Write("config/execution.data", "hidden selected change")
			case "model-assume-unchanged":
				repo.Git("update-index", "--assume-unchanged", "inputs/local/model/lending/loan.yaml")
				repo.Write("inputs/local/model/lending/loan.yaml", "hidden model change")
			case "ignored-model-extra":
				repo.Write("inputs/local/model/lending/extra.ignored", "extra protected bytes")
			case "selected-uncommitted-ignored":
				repo.Write("config/extra.ignored", "not a HEAD source")
				selection.Execution = append(selection.Execution, apply.SourceLocation{Boundary: repo.Dir, Path: fixturePath(t, "config/extra.ignored")})
			case "nonignored-untracked":
				repo.Write("unknown.bin", "must not be silently ignored")
			case "staged":
				repo.Write("real/library.go", "staged change")
				repo.Git("add", "real/library.go")
			}
			result, err := adapter(t).Capture(context.Background(), apply.CaptureRequest{Selection: selection})
			if err == nil || result.Guard != "" || result.Head != head {
				t.Fatalf("committed protection failed: %v head=%s guard=%s", err, result.Head, result.Guard)
			}
		})
	}
	repo, selection, _ := libraryFixture(t)
	r := adapter(t)
	first, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	repo.Write("config/sibling.ignored", "not selected settings; still in census")
	second, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	if first.Guard == second.Guard || len(second.Raw.Execution) != 1 {
		t.Fatal("ignored sibling silently became settings or missed guard")
	}
	if _, found := second.Raw.Census.Lookup(fixturePath(t, "config/sibling.ignored")); !found {
		t.Fatal("ignored sibling lost from census")
	}
	repo.Write(".markitect/state/journal", "control-only change")
	third, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	if third.Guard != second.Guard {
		t.Fatal("excluded state changed source guard")
	}
	repo.Write(".markitect/stateful/retained.ignored", "ordinary censused source")
	fourth, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	if fourth.Guard == third.Guard {
		t.Fatal("state prefix was excluded broadly")
	}
}

func copyBoundary(t *testing.T, source, target string) {
	t.Helper()
	err := fs.WalkDir(os.DirFS(source), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destination := filepath.Join(target, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExplicitExternalSourcesAreCurrentObservationsAndFixedPreviewRefuses(t *testing.T) {
	repo, selection, head := libraryFixture(t)
	external := testkit.TempDir(t)
	copyBoundary(t, selection.Dependencies[0].Boundary, external)
	selection.Dependencies[0].Boundary = external
	externalHost := testkit.TempDir(t)
	if err := os.WriteFile(filepath.Join(externalHost, "profile.data"), []byte("\x00external\xff\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	selection.HostProfiles = []apply.SourceLocation{{Boundary: externalHost, Path: fixturePath(t, "profile.data")}}
	r := adapter(t)
	service := apply.NewModelService(r, r)
	actual, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || actual.Compilation().Status != model.Passed {
		t.Fatalf("external capture: %v %+v", err, actual.Compilation())
	}
	if actual.Inputs().Raw.Dependencies[0].RepositoryArea.String() != "" || actual.Inputs().Raw.HostProfiles[0].RepositoryPath.String() != "" || !bytes.Equal(actual.Inputs().Raw.HostProfiles[0].File.Bytes(), []byte("\x00external\xff\r\n")) {
		t.Fatal("external observation was rebound to repository or changed")
	}
	working, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil || working.Compilation().Status != model.Passed {
		t.Fatalf("external working: %v", err)
	}
	refused, err := r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection, Revision: head})
	if !errors.Is(err, ErrUnsafe) || refused.Revision != head || refused.Raw.Dependencies[0].Selection.Boundary != external || len(refused.Raw.Dependencies[0].Files.Entries()) != 0 {
		t.Fatalf("external source was claimed historical: %v %+v", err, refused)
	}
	selection.Dependencies[0].Boundary = filepath.Join(repo.Dir, "inputs", "vocabulary")
	refused, err = r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection, Revision: head})
	if !errors.Is(err, ErrUnsafe) || len(refused.Raw.HostProfiles) != 1 || refused.Raw.HostProfiles[0].Location.Boundary != externalHost {
		t.Fatalf("external host selection lost on refusal: %v", err)
	}
	selection.HostProfiles = []apply.SourceLocation{{Boundary: repo.Dir, Path: fixturePath(t, "host/profile.data")}}
	selection.Execution[0].Boundary = externalHost
	_, err = r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection})
	if !errors.Is(err, ErrUnsafe) {
		t.Fatal("external execution source accepted")
	}
}

func TestFixedRevisionUsesHistoricalDirectoriesAndRejectsSelectors(t *testing.T) {
	repo, selection, head := libraryFixture(t)
	if err := os.Rename(selection.Local.Boundary, selection.Local.Boundary+"-moved"); err != nil {
		t.Fatal(err)
	}
	r := adapter(t)
	service := apply.NewModelService(r, r)
	fixed, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: head}, model.Limits{})
	if err != nil || fixed.Compilation().Status != model.Passed {
		t.Fatalf("historical root required today's directory: %v", err)
	}
	for _, revision := range []string{"HEAD", head[:12], strings.ToUpper(head), "--help", strings.Repeat("0", len(head))} {
		result, err := r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision})
		if err == nil || result.Head != head || result.Revision != revision {
			t.Fatalf("revision substituted: %q %+v %v", revision, result, err)
		}
	}
	if repo.Git("rev-parse", "HEAD") != head {
		t.Fatal("capture modified repository")
	}
}
