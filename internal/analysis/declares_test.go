package analysis

import (
	"reflect"
	"testing"
)

// doc shows the labelled fields of a Definition with what each links, and for
// a Kind what it declares: null where it declares nothing, an empty list
// where it declares that there is none.
func TestDocShowsFieldsAndWhatAKindDeclares(t *testing.T) {
	r := library(t)
	r[m+"kinds/Step.md"] = "---\nkind: Kind\nfields:\n  After: {required: true, links: [kinds/Step, std/Rule]}\n  Note:\n  Alone: {links: }\nparts:\n  .: kinds/Step\n---\n# Step\n\nOne step.\n"
	r[m+"kinds/Bare.md"] = "---\nkind: Kind\nfields:\nparts:\n---\n# Bare\n\nNothing but its sentence.\n"
	r[m+"kinds/Open.md"] = "---\nkind: Kind\nfields:\n  Why:\n---\n# Open\n\nDeclares fields and says nothing of parts.\n"
	r[m+"kinds/Plain.md"] = def("Kind", "Plain", "Says nothing of fields or parts.")
	r[m+"steps/first.md"] = def("kinds/Step", "First", "Starts, see [[loan-limit]].\n\n**After:** [[loan-limit]].\n\n**Note:** none.")
	r[m+"steps/second.md"] = def("kinds/Step", "Second", "**After:** [[steps/first]] and [[loan-limit]].")
	p := loadRepo(t, r)

	second := DocDefinitionOf(p, p.Spec.Definitions["steps/second"])
	if want := []DocField{{Label: "After", Links: []string{"steps/first", "loan-limit"}}}; !reflect.DeepEqual(second.Fields, want) || second.Declares != nil {
		t.Fatalf("fields of second = %+v, declares %+v", second.Fields, second.Declares)
	}
	first := DocDefinitionOf(p, p.Spec.Definitions["steps/first"])
	if want := []DocField{{Label: "After", Links: []string{"loan-limit"}}, {Label: "Note", Links: []string{}}}; !reflect.DeepEqual(first.Fields, want) {
		t.Fatalf("fields of first = %+v", first.Fields)
	}

	step := DocDefinitionOf(p, p.Spec.Definitions["kinds/Step"])
	want := &DocDeclares{
		Fields: []DocDeclaredField{
			{Name: "After", Required: true, Links: []string{"kinds/Step", "std/Rule"}},
			{Name: "Note"},
			{Name: "Alone", Links: []string{}},
		},
		Parts: []DocDeclaredPart{{Folder: ".", Kinds: []string{"kinds/Step"}}},
	}
	if !reflect.DeepEqual(step.Declares, want) {
		t.Fatalf("Step declares %+v", step.Declares)
	}
	bare := DocDefinitionOf(p, p.Spec.Definitions["kinds/Bare"]).Declares
	if bare == nil || bare.Fields == nil || len(bare.Fields) != 0 || bare.Parts == nil || len(bare.Parts) != 0 {
		t.Fatalf("Bare declares %+v", bare)
	}
	open := DocDefinitionOf(p, p.Spec.Definitions["kinds/Open"]).Declares
	if open == nil || len(open.Fields) != 1 || open.Parts != nil {
		t.Fatalf("Open declares %+v", open)
	}
	if plain := DocDefinitionOf(p, p.Spec.Definitions["kinds/Plain"]); plain.Declares != nil {
		t.Fatalf("a Kind that declares nothing: %+v", plain.Declares)
	}
}
