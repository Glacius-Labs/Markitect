package analysis

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const lendOperation = "Lends a book.\n\n" +
	"**Who:** `member`.\n\n" +
	"**Checked in this order:**\n\n" +
	"1. [[catalog/operations/lend/conditions/free]]: `taken`.\n" +
	"2. [[members/member]] is in good standing, and [[loan-limit]] holds: `over_limit`.\n\n" +
	"**Returns:** the `loan`.\n\n" +
	"**Satisfies:** [[loan-limit]]."

// doc gives each labelled field of a Definition as the compiler reads it: the
// text after the label as written, the links in it, and the entries of a list
// that belongs to it, each with its own links. The sentences before the first
// field stay in the text.
func TestDocFieldsGiveTextLinksAndItems(t *testing.T) {
	r := library(t)
	r[m+"feature.md"] = def("Kind", "Feature", "One thing a user can do.")
	r[m+"catalog/operations/lend/lend.md"] = def("feature", "Lend", lendOperation)
	r[m+"catalog/operations/lend/conditions/free.md"] = def("feature", "Free", "Nobody holds it.\n\n**Note:** a *bold* word, see [[members/member]].")
	q := loadRepo(t, r)

	lend := DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/lend/lend"])
	want := []DocField{
		{Label: "Who", Text: "`member`.", Links: []string{}},
		{Label: "Checked in this order", Text: "", Links: []string{"catalog/operations/lend/conditions/free", "members/member", "loan-limit"}, Items: []DocItem{
			{Text: "[[catalog/operations/lend/conditions/free]]: `taken`.", Links: []string{"catalog/operations/lend/conditions/free"}},
			{Text: "[[members/member]] is in good standing, and [[loan-limit]] holds: `over_limit`.", Links: []string{"members/member", "loan-limit"}},
		}},
		{Label: "Returns", Text: "the `loan`.", Links: []string{}},
		{Label: "Satisfies", Text: "[[loan-limit]].", Links: []string{"loan-limit"}},
	}
	if !reflect.DeepEqual(lend.Fields, want) {
		t.Fatalf("fields of lend:\n%+v\nwant:\n%+v", lend.Fields, want)
	}
	if !strings.HasPrefix(lend.Text, "# Lend\n\nLends a book.\n\n**Who:**") {
		t.Fatalf("the text stays whole: %q", lend.Text)
	}

	// A part has its fields as a Definition has them.
	if len(lend.Parts) != 1 || !reflect.DeepEqual(lend.Parts[0].Fields, []DocField{{Label: "Note", Text: "a *bold* word, see [[members/member]].", Links: []string{"members/member"}}}) {
		t.Fatalf("parts of lend: %+v", lend.Parts)
	}

	// The reading view has them too.
	for _, e := range Read(q, "catalog/operations/lend") {
		if e.ID == "catalog/operations/lend/lend" && len(e.Fields) != 4 {
			t.Fatalf("read fields of lend: %+v", e.Fields)
		}
		if e.ID == "catalog/operations/lend/conditions/free" && len(e.Fields) != 1 {
			t.Fatalf("read fields of free: %+v", e.Fields)
		}
	}

	// No field, no key; a field without a list has no items key.
	plain := DocDefinitionOf(q, q.Spec.Definitions["loan-limit"])
	if plain.Fields != nil {
		t.Fatalf("a Definition without fields: %+v", plain.Fields)
	}
	data, err := json.Marshal(lend)
	if err != nil {
		t.Fatal(err)
	}
	var back struct {
		Fields []map[string]any `json:"fields"`
	}
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if _, has := back.Fields[0]["items"]; has || back.Fields[1]["items"] == nil || back.Fields[0]["label"] != "Who" {
		t.Fatalf("json of the fields: %v", back.Fields)
	}
}

// A bulleted list belongs to a field the same way, a label inside a list or
// after a sentence is none, and text after a list stays with the field.
func TestDocFieldsBulletsAndEdges(t *testing.T) {
	r := library(t)
	r[m+"steps.md"] = def("Kind", "Steps", "Steps.")
	r[m+"catalog/flow.md"] = def("steps", "Flow", "Opening words. **Not a field:** here.\n\n"+
		"**Steps:** Do these:\n\n- first, [[loan-limit]]\n- second\n  - nested, [[members/member]]\n\n"+
		"and then rest.\n\n"+
		"- **Inside:** a list\n\n"+
		"**Last:** a [[members/member|a member]].")
	q := loadRepo(t, r)
	flow := DocDefinitionOf(q, q.Spec.Definitions["catalog/flow"])
	want := []DocField{
		{Label: "Steps", Text: "Do these:\n\nand then rest.", Links: []string{"loan-limit", "members/member"}, Items: []DocItem{
			{Text: "first, [[loan-limit]]", Links: []string{"loan-limit"}},
			{Text: "second\n  - nested, [[members/member]]", Links: []string{"members/member"}},
		}},
		{Label: "Last", Text: "a [[members/member|a member]].", Links: []string{"members/member"}},
	}
	if len(flow.Fields) != 2 || flow.Fields[0].Label != "Steps" {
		t.Fatalf("fields of flow: %+v", flow.Fields)
	}
	// The list after "and then rest." lies in the field "Steps" too.
	want[0].Items = append(want[0].Items, DocItem{Text: "**Inside:** a list", Links: []string{}})
	if !reflect.DeepEqual(flow.Fields, want) {
		t.Fatalf("fields of flow:\n%+v\nwant:\n%+v", flow.Fields, want)
	}
}

// A link stands where it is written: the same link twice in one field is
// listed twice, and the links before the first field and in a field after it
// are not taken for another's.
func TestDocFieldsLinksStandWhereTheyAreWritten(t *testing.T) {
	r := library(t)
	r[m+"steps.md"] = def("Kind", "Steps", "Steps.")
	r[m+"catalog/twice.md"] = def("steps", "Twice", "Before, [[loan-limit]] and [[members/member]].\n\n"+
		"**First:** [[loan-limit]], then [[loan-limit]] again.\n\n"+
		"**Second:** [[members/member]] and [[loan-limit|the limit]].")
	q := loadRepo(t, r)
	got := DocDefinitionOf(q, q.Spec.Definitions["catalog/twice"]).Fields
	want := []DocField{
		{Label: "First", Text: "[[loan-limit]], then [[loan-limit]] again.", Links: []string{"loan-limit", "loan-limit"}},
		{Label: "Second", Text: "[[members/member]] and [[loan-limit|the limit]].", Links: []string{"members/member", "loan-limit"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fields:\n%+v\nwant:\n%+v", got, want)
	}
}
