package project

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestStandardLibrariesCompile(t *testing.T) {
	var src model.Source
	want := map[string][]string{"std": {"Concept.md", "Decision.md", "Feature.md", "Project.md", "Rule.md"}, "rest": {"Endpoint.md"}, "cli": {"Command.md"}}
	for _, name := range libraries.Names() {
		files, ok := libraries.Files(name)
		if !ok {
			t.Fatalf("no Library %s", name)
		}
		if len(files) != len(want[name]) {
			t.Fatalf("%s has %d files, want %v", name, len(files), want[name])
		}
		lib := model.LibrarySource{Name: name, Version: libraries.Version, Dir: "markitect:" + name}
		for i, f := range files {
			if f.Path != want[name][i] {
				t.Fatalf("%s file %d = %s", name, i, f.Path)
			}
			lib.Files = append(lib.Files, model.File{Path: f.Path, Data: f.Data})
		}
		src.Libraries = append(src.Libraries, lib)
	}
	if _, ds := model.Compile(src); len(ds) != 0 {
		t.Fatalf("standard Libraries do not compile:\n%v", ds)
	}
	if _, ok := libraries.Files("nope"); ok {
		t.Fatal("unknown Library found")
	}
}

// TestStandardLibraryDigests pins the content of each standard Library at its
// version: every project's lock pins these digests, so a change to a Kind
// needs a new Library version and a new expected digest here.
func TestStandardLibraryDigests(t *testing.T) {
	want := map[string]string{
		"std":  "sha256:116d53f28a01acee77087130cf3f999f42274e36b0cdb949cbd766a2885a5b39",
		"rest": "sha256:adab445700aa3b4a4bd60a90d57facccd1841848945206acafa37630e12938c6",
		"cli":  "sha256:216b8df758a5b9177b0e8dbe6524451782bb6e40cd8067558a4ede1b00186597",
	}
	if libraries.Version != "1.0.0" {
		t.Fatalf("version %s: update the pinned digests with the version", libraries.Version)
	}
	for _, name := range libraries.Names() {
		files, _ := libraries.Files(name)
		var lib []model.File
		for _, f := range files {
			lib = append(lib, model.File{Path: f.Path, Data: f.Data})
		}
		if got := model.LibraryDigest(lib); got != want[name] {
			t.Errorf("%s %s changed: digest %s, pinned %s", name, libraries.Version, got, want[name])
		}
	}
}

func TestStdCompilesAlone(t *testing.T) {
	files, _ := libraries.Files("std")
	lib := model.LibrarySource{Name: "std", Version: libraries.Version, Dir: "markitect:std"}
	for _, f := range files {
		lib.Files = append(lib.Files, model.File{Path: f.Path, Data: f.Data})
	}
	if _, ds := model.Compile(model.Source{Libraries: []model.LibrarySource{lib}}); len(ds) != 0 {
		t.Fatalf("std needs another Library:\n%v", ds)
	}
}
