package apply_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// All five selected source classes carry original bytes, modes and exact
// acquisition coordinates. Repository membership is explicit, not inferred
// from SourceModel.Origin or an external Host source's location spelling.
func withSelectedSources(t *testing.T, selection *apply.InputSelection, raw *apply.RawInputs) {
	t.Helper()
	files := raw.Census.Entries()
	for i, name := range []string{"execution", "registry", "ignore", "host-profile", "instructions"} {
		location := apply.SourceLocation{Boundary: "explicit-source-boundary", Path: path(t, name+".data")}
		file, err := snapshot.NewEntry(location.Path, snapshot.Executable, []byte("original\r\n"+name+"\x00"))
		if err != nil {
			t.Fatal(err)
		}
		source := apply.SourceFile{Location: location, File: file}
		if i < 3 {
			source.RepositoryPath = path(t, "inputs/"+name+".data")
			local, err := snapshot.NewEntry(source.RepositoryPath, file.Mode(), file.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, local)
		}
		switch i {
		case 0:
			selection.Execution, raw.Execution = []apply.SourceLocation{location}, []apply.SourceFile{source}
		case 1:
			selection.Registry, raw.Registry = []apply.SourceLocation{location}, []apply.SourceFile{source}
		case 2:
			selection.Ignore, raw.Ignore = []apply.SourceLocation{location}, []apply.SourceFile{source}
		case 3:
			selection.HostProfiles, raw.HostProfiles = []apply.SourceLocation{location}, []apply.SourceFile{source}
		case 4:
			selection.Instructions, raw.Instructions = []apply.SourceLocation{location}, []apply.SourceFile{source}
		}
	}
	var err error
	raw.Census, err = snapshot.NewTree(files)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExactOriginalTreesModesCoordinatesAndSelectedSourcesSurvive(t *testing.T) {
	selection, raw := libraryInputs(t)
	withSelectedSources(t, &selection, &raw)
	// The model's provenance deliberately bears no relation to its filesystem
	// selection; only explicit coordinates bind it to the repository census.
	raw.Local.Source.Origin = "opaque-origin:outside/../not-a-location"
	result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || result.Compilation().Status != model.Passed {
		t.Fatalf("%v %+v", err, result.Compilation())
	}
	got := result.Inputs()
	if !reflect.DeepEqual(got.Raw, raw) || got.Raw.Census.Digest() != raw.Census.Digest() {
		t.Fatal("acquisition was normalized or rebound")
	}
	for _, file := range []apply.SourceFile{got.Raw.Execution[0], got.Raw.Registry[0], got.Raw.Ignore[0], got.Raw.HostProfiles[0], got.Raw.Instructions[0]} {
		want := snapshot.ContentID(sha256.Sum256([]byte("original\r\n" + file.Location.Path.String()[:len(file.Location.Path.String())-5] + "\x00")))
		if file.File.Mode() != snapshot.Executable || file.File.ContentID() != want {
			t.Fatal("mode or original content changed")
		}
	}
}

func TestSelectedInputAndDependencyByteMismatchesRefuseBeforeCompile(t *testing.T) {
	for _, kind := range []string{"execution missing", "registry missing", "ignore missing", "host missing", "instruction missing", "extra input", "input location", "input path", "input bytes", "input mode", "input repo path", "dependency bytes", "local bytes"} {
		t.Run(kind, func(t *testing.T) {
			selection, raw := libraryInputs(t)
			withSelectedSources(t, &selection, &raw)
			switch kind {
			case "execution missing":
				raw.Execution = nil
			case "registry missing":
				raw.Registry = nil
			case "ignore missing":
				raw.Ignore = nil
			case "host missing":
				raw.HostProfiles = nil
			case "instruction missing":
				raw.Instructions = nil
			case "extra input":
				raw.Ignore = append(raw.Ignore, raw.Ignore[0])
			case "input location":
				raw.Execution[0].Location.Boundary = "other"
			case "input path":
				raw.Execution[0].File, _ = snapshot.NewEntry(path(t, "other"), snapshot.Regular, nil)
			case "input bytes":
				raw.Execution[0].File, _ = snapshot.NewEntry(raw.Execution[0].File.Path(), snapshot.Executable, []byte("changed"))
			case "input mode":
				raw.Execution[0].File, _ = snapshot.NewEntry(raw.Execution[0].File.Path(), snapshot.Regular, raw.Execution[0].File.Bytes())
			case "input repo path":
				raw.Execution[0].RepositoryPath = path(t, "other")
			case "dependency bytes":
				for i := range raw.Dependencies[0].Source.Entries {
					if raw.Dependencies[0].Source.Entries[i].Kind == model.Regular {
						raw.Dependencies[0].Source.Entries[i].Bytes = []byte("changed")
						break
					}
				}
			case "local bytes":
				for i := range raw.Local.Source.Entries {
					if raw.Local.Source.Entries[i].Kind == model.Regular {
						raw.Local.Source.Entries[i].Bytes = []byte("changed")
						break
					}
				}
			}
			result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
			if !errors.Is(err, apply.ErrIncompleteInputs) || result.Compilation().Status != model.NotCompleted || !reflect.DeepEqual(result.Inputs().Raw, raw) {
				t.Fatalf("%v %+v", err, result.Compilation())
			}
		})
	}
}

func TestPackageInventoryComesFromActualModelDirectoryEntries(t *testing.T) {
	selection, raw := libraryInputs(t)
	raw.Local.Source.Entries = append(raw.Local.Source.Entries, model.Entry{Path: "model/empty/leaf", Kind: model.Directory})
	result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || result.Compilation().Status != model.Passed {
		t.Fatalf("%v %+v", err, result.Compilation())
	}
	local := result.Compilation().Models[0]
	found := make(map[string]bool)
	for _, pkg := range local.Packages {
		found[pkg.Name] = true
	}
	if !found["empty"] || !found["empty/leaf"] || len(local.Definitions) != 22 {
		t.Fatal("empty directory inventory lost or inferred from Definitions")
	}
}
