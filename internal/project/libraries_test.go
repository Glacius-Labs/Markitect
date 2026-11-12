package project

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

func TestStandardLibrariesCompile(t *testing.T) {
	var src spec.Source
	want := map[string][]string{"std": {"Concept.md", "Decision.md", "Feature.md", "Package.md", "Project.md", "Rule.md"}, "rest": {"Endpoint.md"}, "cli": {"Command.md"}}
	for _, name := range libraries.Names() {
		files, ok := libraries.Files(name)
		if !ok {
			t.Fatalf("no Library %s", name)
		}
		if len(files) != len(want[name]) {
			t.Fatalf("%s has %d files, want %v", name, len(files), want[name])
		}
		lib := spec.LibrarySource{Name: name, Version: libraries.Version, Dir: "markitect:" + name}
		for i, f := range files {
			if f.Path != want[name][i] {
				t.Fatalf("%s file %d = %s", name, i, f.Path)
			}
			lib.Files = append(lib.Files, spec.File{Path: f.Path, Data: f.Data})
		}
		src.Libraries = append(src.Libraries, lib)
	}
	if _, ds := spec.Compile(src); len(ds) != 0 {
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
		"std":  "sha256:7a2d23e215f0c38ecf41bb09f45706fff9d3957cee5521564c4681186fa8156d",
		"rest": "sha256:adab445700aa3b4a4bd60a90d57facccd1841848945206acafa37630e12938c6",
		"cli":  "sha256:f4dd204b6c16a842d1b9725d382d517651cfea63dcf4c0180ef8164da04c0f2c",
	}
	if libraries.Version != "1.2.0" {
		t.Fatalf("version %s: update the pinned digests with the version", libraries.Version)
	}
	for _, name := range libraries.Names() {
		files, _ := libraries.Files(name)
		var lib []spec.File
		for _, f := range files {
			lib = append(lib, spec.File{Path: f.Path, Data: f.Data})
		}
		if got := spec.LibraryDigest(lib); got != want[name] {
			t.Errorf("%s %s changed: digest %s, pinned %s", name, libraries.Version, got, want[name])
		}
	}
}

func TestStdCompilesAlone(t *testing.T) {
	files, _ := libraries.Files("std")
	lib := spec.LibrarySource{Name: "std", Version: libraries.Version, Dir: "markitect:std"}
	for _, f := range files {
		lib.Files = append(lib.Files, spec.File{Path: f.Path, Data: f.Data})
	}
	if _, ds := spec.Compile(spec.Source{Libraries: []spec.LibrarySource{lib}}); len(ds) != 0 {
		t.Fatalf("std needs another Library:\n%v", ds)
	}
}
