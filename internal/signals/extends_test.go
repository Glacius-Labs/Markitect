package signals

import "testing"

// derived is a spec of a base type, a type that extends it and that no
// document names, a document of the derived type and a document of its own.
func derived(t *testing.T) *Analysis {
	return Analyze(spec(t, map[string]string{
		"types/artifact.md": "# Artifact\n\nSomething a team makes.\n\n## Owner\nEvery artifact has one owner.\n",
		"types/website.md":  "# Website\n\nExtends: [Artifact](artifact.md)\n\nA site people open.\n\n## Pages\nEach page is a document of its own.\n",
		"sites/home.md":     "# Home page\n\nType: [Website](../types/website.md)\n\n## Owner\nAda\n\n## Pages\nOne.\n",
		"notes.md":          "# Notes\n\nA website holds pages and an artifact has an owner.\n",
	}), References{})
}

func TestDerivedTypesAreTypes(t *testing.T) {
	a := derived(t)
	// Only what stands before the first section of a type says what it is: the
	// guidance under the sections of the base and of the derived type is no
	// statement, and the extends line is none either.
	var got []string
	for _, st := range a.Statements {
		got = append(got, st.Doc+"|"+st.Text)
	}
	eq(t, got, []string{
		"notes.md|A website holds pages and an artifact has an owner.",
		"sites/home.md|Ada",
		"sites/home.md|One.",
		"types/artifact.md|Something a team makes.",
		"types/website.md|A site people open.",
	})
	// The titles of types are named by no one: neither of the base nor of a type
	// that extends it.
	eq(t, a.Named[0], []string(nil))
}

func TestTopicOfDerivedTypes(t *testing.T) {
	a := derived(t)
	for path, want := range map[string][]string{
		"types/artifact": {"sites/home.md"},
		"types/website":  {"sites/home.md"},
	} {
		topic, ok := a.Topic(path)
		if !ok || topic.Kind != "type" {
			t.Fatalf("%s: %+v", path, topic)
		}
		eq(t, topic.Documents, want)
	}
	// A derived type nobody uses yet is a type; a document of it is not.
	b := Analyze(spec(t, map[string]string{
		"a.md": "# A\n\nBase.\n",
		"b.md": "# B\n\nExtends: [A](a.md)\n",
		"c.md": "# C\n\nText.\n",
	}), References{})
	for path, kind := range map[string]string{"a": "type", "b": "type", "c": "document"} {
		if topic, _ := b.Topic(path); topic.Kind != kind {
			t.Errorf("%s is %s", path, topic.Kind)
		}
	}
}

func TestInvalidExtendsLineIsNoType(t *testing.T) {
	a := Analyze(spec(t, map[string]string{
		"rules.md": "# Rules\n\nExtends: the borrowing rules of the house\n\nThe rules.\n\n## Rule\nEvery loan ends.\n",
		"lost.md":  "# Lost\n\nExtends: [Nowhere](nowhere.md)\n\nA type.\n\n## Rule\nEvery loan ends.\n",
	}), References{})
	if topic, _ := a.Topic("rules"); topic.Kind != "document" {
		t.Errorf("rules is %s", topic.Kind)
	}
	if topic, _ := a.Topic("lost"); topic.Kind != "type" {
		t.Errorf("lost is %s", topic.Kind)
	}
	var got []string
	for _, st := range a.Statements {
		got = append(got, st.Doc+"|"+st.Text)
	}
	eq(t, got, []string{"lost.md|A type.", "rules.md|The rules.", "rules.md|Every loan ends."})
}
