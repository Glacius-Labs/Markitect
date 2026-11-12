package project

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

func TestStandardLibrariesCompile(t *testing.T) {
	var src spec.Source
	want := map[string][]string{"std": {"Action.md", "Alternative.md", "Assumption.md", "Attribute.md", "Component.md", "Concept.md", "Concern.md", "Condition.md", "Connector.md", "Consequence.md", "Constraint.md", "Decision.md", "Deployment.md", "Display.md", "Error.md", "Exclusion.md", "Expectation.md", "ExternalSystem.md", "Goal.md", "Interface.md", "Invariant.md", "Limit.md", "Message.md", "Need.md", "Offer.md", "Operation.md", "Outcome.md", "Page.md", "Project.md", "Provision.md", "Quality.md", "Reaction.md", "Refusal.md", "Requirement.md", "Result.md", "Route.md", "Rule.md", "Scenario.md", "Shape.md", "Stakeholder.md", "State.md", "Strategy.md", "Task.md", "Term.md", "Wish.md"}}
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
		"std": "sha256:6c041f606dfb3b50bb302be17c566b268ef6f5ac9c7bba55709d7300c7f03f7d",
	}
	if libraries.Version != "2.1.0" {
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
