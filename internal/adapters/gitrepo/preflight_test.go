package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestManifestFailureRetainsActualResultAndNeverReadsDefinitions(t *testing.T) {
	for _, mode := range []string{"working", "committed", "fixed"} {
		for _, kind := range []string{"failed", "not-completed"} {
			t.Run(mode+"/"+kind, func(t *testing.T) {
				repo, selection, _ := libraryFixture(t)
				text := "# original coordinates\r\nformatVersion: 1\r\nid: broken\r\nroot: 7\r\ndependencies: {}\r\n"
				limits := Limits{}
				if kind == "not-completed" {
					text = "formatVersion: 1\nid: broken\nroot: model\ndependencies: {}\nextra: {a: {b: {c: x}}}\n"
					limits.Manifest.YAMLDepth = 3
				}
				repo.Write("inputs/vocabulary/model.yaml", text)
				head := repo.Commit("Actual preflight failure")
				r, err := New(limits)
				if err != nil {
					t.Fatal(err)
				}
				var reads []string
				r.observe = func(event, p string) {
					if event == "before-file-open" || event == "before-git-blob" {
						reads = append(reads, p)
					}
				}
				service := apply.NewModelService(r, r)
				var raw apply.RawInputs
				var status model.Status
				if mode == "committed" {
					result, e := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
					err = e
					raw = result.Inputs().Raw
					status = result.Compilation().Status
					if result.Inputs().Guard != "" || result.Inputs().Head != head {
						t.Fatal("refusal issued guard or discarded observed HEAD")
					}
				} else {
					revision := ""
					if mode == "fixed" {
						revision = head
					}
					result, e := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
					err = e
					raw = result.Inputs().Raw
					status = result.Compilation().Status
				}
				var failure *apply.ManifestPreflightError
				if !errors.As(err, &failure) || status != model.NotCompleted {
					t.Fatalf("missing actual manifest failure: %v status=%s", err, status)
				}
				inspection := failure.Inspection()
				want := model.InspectManifest(model.ManifestInput{Origin: raw.Dependencies[0].Source.Origin, ManifestPath: "model.yaml", Bytes: []byte(text), Limits: limits.Manifest})
				if failure.Selection() != selection.Dependencies[0] || !reflect.DeepEqual(inspection, want) || raw.Dependencies[0].Root.String() != "" {
					t.Fatal("manifest result, coordinates, selection or absent Root changed")
				}
				if !bytes.Equal(raw.Dependencies[0].Files.Entries()[0].Bytes(), []byte(text)) {
					t.Fatal("partial manifest bytes discarded")
				}
				if len(reads) != 2 {
					t.Fatalf("unexpected source reads before failed root preflight: %v", reads)
				}
				for _, p := range reads {
					if !strings.HasSuffix(p, "model.yaml") {
						t.Fatalf("nonmanifest read before preflight: %s", p)
					}
				}
			})
		}
	}
}

func TestEveryPhysicalRootIsValidatedBeforeDefinitionBytes(t *testing.T) {
	for _, kind := range []string{"missing", "file", "case-alias", "full-fold-collision", "normalization-collision", "same-root", "nested-root", "excluded-state"} {
		t.Run(kind, func(t *testing.T) {
			repo, selection, _ := libraryFixture(t)
			root := filepath.Join(selection.Local.Boundary, "model")
			switch kind {
			case "missing":
				if err := os.Rename(root, root+"-absent"); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.Rename(root, root+"-away"); err != nil {
					t.Fatal(err)
				}
				repo.Write("inputs/local/model", "not a directory")
			case "case-alias":
				if err := os.Rename(root, filepath.Join(selection.Local.Boundary, "Model")); err != nil {
					t.Fatal(err)
				}
			case "full-fold-collision", "normalization-collision":
				name, other := "straße", "strasse"
				if kind == "normalization-collision" {
					name, other = "über", "u\u0308ber"
				}
				if err := os.Rename(root, filepath.Join(selection.Local.Boundary, name)); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(selection.Local.Boundary, other), 0o755); err != nil {
					t.Skipf("filesystem cannot represent collision: %v", err)
				}
				data, err := os.ReadFile(filepath.Join(selection.Local.Boundary, "model.yaml"))
				if err != nil {
					t.Fatal(err)
				}
				repo.Write("inputs/local/model.yaml", strings.Replace(string(data), "root: model", "root: "+name, 1))
			case "same-root":
				selection.Dependencies[0] = selection.Local
			case "nested-root":
				repo.Write("inputs/local/model/nested/model.yaml", "formatVersion: 1\nid: nested\nroot: model\ndependencies: {}\n")
				if err := os.MkdirAll(filepath.Join(root, "nested", "model"), 0o755); err != nil {
					t.Fatal(err)
				}
				selection.Dependencies[0] = apply.ModelSelection{Boundary: filepath.Join(root, "nested"), ManifestPath: fixturePath(t, "model.yaml")}
			case "excluded-state":
				repo.Write(".markitect/model.yaml", "formatVersion: 1\nid: state\nroot: state\ndependencies: {}\n")
				if err := os.MkdirAll(filepath.Join(repo.Dir, ".markitect", "state"), 0o755); err != nil {
					t.Fatal(err)
				}
				selection.Local = apply.ModelSelection{Boundary: filepath.Join(repo.Dir, ".markitect"), ManifestPath: fixturePath(t, "model.yaml")}
			}
			r := adapter(t)
			var nonmanifest []string
			r.observe = func(event, p string) {
				if (event == "before-file-open" || event == "before-git-blob") && !strings.HasSuffix(p, "model.yaml") {
					nonmanifest = append(nonmanifest, p)
				}
			}
			result, err := r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection})
			if err == nil || len(nonmanifest) != 0 || result.Head == "" {
				t.Fatalf("root preflight: err=%v nonmanifest=%v head=%s", err, nonmanifest, result.Head)
			}
			var manifest *apply.ManifestPreflightError
			if errors.As(err, &manifest) {
				t.Fatalf("physical failure became language failure: %v", err)
			}
		})
	}
}

func TestWorkingEmptyModelDirectoriesAreExplicitAndCannotBecomeCapture(t *testing.T) {
	_, selection, head := libraryFixture(t)
	if err := os.MkdirAll(filepath.Join(selection.Local.Boundary, "model", "empty", "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := adapter(t)
	working, err := apply.NewModelService(r, r).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil || working.Compilation().Status != model.Passed {
		t.Fatalf("working empty dirs: %v %+v", err, working.Compilation())
	}
	var dirs []string
	for _, entry := range working.Inputs().Raw.Local.Source.Entries {
		if entry.Kind == model.Directory {
			dirs = append(dirs, entry.Path)
		}
	}
	if !contains(dirs, "model/empty") || !contains(dirs, "model/empty/child") {
		t.Fatalf("empty directory entries dropped: %v", dirs)
	}
	refused, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if !errors.Is(err, ErrUnrepresentable) || refused.Guard != "" || refused.Head != head {
		t.Fatalf("nonrevision empty dirs promoted: %v %+v", err, refused)
	}
	fixed, err := r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection, Revision: head})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range fixed.Raw.Local.Source.Entries {
		if strings.HasPrefix(entry.Path, "model/empty") {
			t.Fatal("working-only directory leaked into fixed revision")
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
