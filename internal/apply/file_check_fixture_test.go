package apply_test

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const emptyFilePolicy = "apiVersion: markitect.ignore/v1\nfiles: []\n"
const editorFilePolicy = "apiVersion: markitect.ignore/v1\nfiles:\n  - path: .editor/library-session.json\n    reason: '  Local editor layout.  '\n"

func fileCheckSource(t *testing.T, selection *apply.InputSelection, raw *apply.RawInputs, group, name string, mode snapshot.Mode, content string, capture bool) {
	t.Helper()
	censusFile(t, raw, name, mode, []byte(content))
	if !capture {
		return
	}
	location := apply.SourceLocation{Boundary: "opaque-" + group, Path: path(t, "acquired/"+group+"/"+name+".data")}
	entry, err := snapshot.NewEntry(location.Path, mode, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	source := apply.SourceFile{Location: location, File: entry, RepositoryPath: path(t, name)}
	switch group {
	case "ignore":
		selection.Ignore = append(selection.Ignore, location)
		raw.Ignore = append(raw.Ignore, source)
	case "instruction":
		selection.Instructions = append(selection.Instructions, location)
		raw.Instructions = append(raw.Instructions, source)
	case "runtime":
		selection.RuntimeProfiles = append(selection.RuntimeProfiles, location)
		raw.RuntimeProfiles = append(raw.RuntimeProfiles, source)
	default:
		t.Fatal("unknown fixture source group")
	}
}

func rewriteFileCheckSource(t *testing.T, raw *apply.RawInputs, source *apply.SourceFile, content string) {
	t.Helper()
	entry, err := snapshot.NewEntry(source.Location.Path, source.File.Mode(), []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	source.File = entry
	var census []snapshot.Entry
	for _, original := range raw.Census.Entries() {
		if original.Path() == source.RepositoryPath {
			original, err = snapshot.NewEntry(original.Path(), original.Mode(), []byte(content))
			if err != nil {
				t.Fatal(err)
			}
		}
		census = append(census, original)
	}
	raw.Census, err = snapshot.NewTree(census)
	if err != nil {
		t.Fatal(err)
	}
}

func checkedFiles(t *testing.T, selection apply.InputSelection, raw apply.RawInputs) apply.CapturedFileCheck {
	t.Helper()
	return registryModel(t, selection, raw).PrepareRegistry().CheckFiles()
}

func fileDiagnostic(result apply.CapturedFileCheck, code string) bool {
	for _, diagnostic := range result.Diagnostics() {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
