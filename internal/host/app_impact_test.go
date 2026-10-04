package host

import (
	"reflect"
	"testing"
)

func TestChangesIncludesOldAndNewDependencyClosures(t *testing.T) {
	before, err := Parse(fixtureFiles(t, "base", "Old policy.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	after, err := Parse(fixtureFiles(t, "candidate", "Revised policy.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	impact := Changes(before, after)
	want := []string{projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	if !reflect.DeepEqual(impact.Affected, want) {
		t.Fatalf("affected = %v, want dependency closure %v", impact.Affected, want)
	}
	if !reflect.DeepEqual(impact.Changed, []string{rulePath}) {
		t.Fatalf("changed = %v, want [%s]", impact.Changed, rulePath)
	}
}

func TestChangesInvalidatesAllForNonResourceMarkdown(t *testing.T) {
	beforeSnapshot := fixtureFiles(t, "base", "Stable policy.", projectNS)
	afterSnapshot := fixtureFiles(t, "candidate", "Stable policy.", projectNS)
	beforeSnapshot.Files["docs/general/notes.md"] = []byte("old note\n")
	afterSnapshot.Files["docs/general/notes.md"] = []byte("new note\n")
	before, err := Parse(beforeSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Parse(afterSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	impact := Changes(before, after)
	want := []string{"/Project/sample-project", projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	if !reflect.DeepEqual(impact.Affected, want) {
		t.Fatalf("non-resource Markdown change affected %v, want conservative invalidation %v", impact.Affected, want)
	}
}

func TestChangesInvalidatesInventoryOnSamePathNamespaceEdit(t *testing.T) {
	before, err := Parse(fixtureFiles(t, "base", "Stable policy.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	after, err := Parse(fixtureFiles(t, "candidate", "Stable policy.", "other-area"))
	if err != nil {
		t.Fatal(err)
	}
	impact := Changes(before, after)
	want := []string{"/Project/sample-project", "other-area/Skill/entry", projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	if !reflect.DeepEqual(impact.Affected, want) {
		t.Fatalf("namespace inventory edit affected %v, want old and new resource inventories %v", impact.Affected, want)
	}
}
