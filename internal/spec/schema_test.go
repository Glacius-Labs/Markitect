package spec

import (
	"reflect"
	"testing"
)

// typedKinds is a Library whose Kinds declare fields and parts.
var typedKinds = LibrarySource{Name: "typed", Version: "1.0.0", Dir: "typed", Files: []File{
	{Path: "Goal.md", Data: []byte("---\nkind: Kind\n---\n# Goal\n")},
	{Path: "Condition.md", Data: []byte("---\nkind: Kind\n---\n# Condition\n")},
	{Path: "Outcome.md", Data: []byte("---\nkind: Kind\n---\n# Outcome\n")},
	{Path: "Operation.md", Data: []byte(`---
kind: Kind
fields:
  Who: {required: true}
  Given:
  Checked in this order: {links: typed/Condition}
  Serves:
    required: true
    links: [typed/Goal, Package]
  Notes: {links: }
parts:
  conditions: typed/Condition
  outcomes: [typed/Outcome]
---
# Operation

An Operation states **Who:** may call it, in its text.
`)},
	{Path: "Note.md", Data: []byte("---\nkind: Kind\nfields:\nparts:\n  .: [typed/Note]\n---\n# Note\n")},
	{Path: "Leaf.md", Data: []byte("---\nkind: Kind\nparts:\n---\n# Leaf\n")},
}}

func typed(t *testing.T, files map[string]string) (*Spec, Diagnostics) {
	t.Helper()
	return compile(t, files, typedKinds)
}

const lend = "# Lend\n\nLends a copy, as [[members/conditions/shared]] allows.\n\n**Who:** `anyone`.\n\n**Given:** `copyId`, see [[members/conditions/shared]].\n\n**Checked in this order:**\n\n1. [[members/operations/lend/conditions/in-stock]]: `out`.\n2. [[members/conditions/shared]]: `no`.\n\n**Serves:** [[goals/lend-books]] and [[members]].\n"

func lending(change map[string]string) map[string]string {
	files := map[string]string{
		"goals/lend-books.md":                            def("typed/Goal", "# Lend books\n"),
		"members/members.md":                             anchor("Members"),
		"members/conditions/shared.md":                   def("typed/Condition", "# Shared\n"),
		"members/operations/lend/lend.md":                def("typed/Operation", lend),
		"members/operations/lend/conditions/in-stock.md": def("typed/Condition", "# In stock\n"),
		"members/operations/lend/outcomes/lent/lent.md":  def("typed/Outcome", "# Lent\n"),
		"members/operations/lend/outcomes/lent/until.md": def("typed/Condition", "# Until\n\nA part of an Outcome, whose Kind declares no parts.\n"),
	}
	for name, data := range change {
		if data == "" {
			delete(files, name)
		} else {
			files[name] = data
		}
	}
	return files
}

func TestCompileAKindDeclaresFieldsAndParts(t *testing.T) {
	m, ds := typed(t, lending(nil))
	clean(t, ds)
	op := m.Definitions["members/operations/lend/lend"]
	var labels []string
	for _, f := range op.Fields {
		labels = append(labels, f.Label)
	}
	if want := []string{"Who", "Given", "Checked in this order", "Serves"}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("fields = %q", labels)
	}
	// The kind first, then the links of the text, each with its field.
	var in []string
	for _, l := range op.Links[1:] {
		in = append(in, l.Field)
	}
	if want := []string{"", "Given", "Checked in this order", "Checked in this order", "Serves", "Serves"}; !reflect.DeepEqual(in, want) {
		t.Fatalf("fields of the links = %q", in)
	}
	kind := m.Definitions["typed/Operation"]
	if kind.Schema == nil || !kind.Schema.HasFields || !kind.Schema.HasParts || len(kind.Fields) != 0 {
		t.Fatalf("schema of the Kind = %+v, its own fields %v", kind.Schema, kind.Fields)
	}
	if f := kind.Schema.Field("Serves"); f == nil || !f.Required || !f.Limited || !reflect.DeepEqual(f.Links, []string{"typed/Goal", "Package"}) {
		t.Fatalf("Serves = %+v", f)
	}
	if f := kind.Schema.Field("Given"); f == nil || f.Required || f.Limited {
		t.Fatalf("Given = %+v", f)
	}
	if p := kind.Schema.Part("outcomes"); p == nil || !reflect.DeepEqual(p.Kinds, []string{"typed/Outcome"}) {
		t.Fatalf("outcomes = %+v", p)
	}
	if m.Definitions["typed/Goal"].Schema != nil {
		t.Fatal("a Kind that declares nothing has a schema")
	}
}

func TestCompileFieldsAreChecked(t *testing.T) {
	_, ds := typed(t, lending(map[string]string{
		"members/operations/lend/lend.md": def("typed/Operation", "# Lend\n\nLends.\n\n**Who:** `anyone`.\n\n**Returns:** nothing.\n\n**Who:** again.\n\n**Checked in this order:** [[goals/lend-books]].\n\n**Notes:** see [[members]].\n\n**Who**: not a field.\n\n- **Given:** nor this.\n"),
	}))
	file := ".markitect/spec/members/operations/lend/lend.md"
	if d := expect(t, ds, "field.unknown", `no field "Returns"; it has Who, Given, Checked in this order, Serves, Notes`); d.File != file || d.Line != 10 {
		t.Fatalf("where: %v", d)
	}
	expect(t, ds, "field.repeated", `"Who" stands twice`)
	if d := expect(t, ds, "field.missing", `a typed/Operation states "Serves": add a line that starts with **Serves:**`); d.Line != 2 {
		t.Fatalf("where: %v", d)
	}
	expect(t, ds, "field.target", `"Checked in this order" of a typed/Operation links a typed/Condition; goals/lend-books is a typed/Goal`)
	expect(t, ds, "field.target", `"Notes" of a typed/Operation links nothing; members is a Package`)
	if len(ds) != 5 {
		t.Fatalf("want five findings:\n%v", ds)
	}
}

func TestCompileAFieldLinksAThingByItsFolder(t *testing.T) {
	_, ds := typed(t, lending(map[string]string{
		"goals/lend-books.md":            "",
		"goals/lend-books/lend-books.md": def("typed/Goal", "# Lend books\n"),
	}))
	clean(t, ds)
	_, ds = typed(t, lending(map[string]string{
		"members/operations/lend/lend.md": def("typed/Operation", "# Lend\n\n**Who:** x.\n\n**Serves:** [[members/operations/lend/outcomes/lent]].\n"),
	}))
	expect(t, ds, "field.target", "links one of Package, typed/Goal; members/operations/lend/outcomes/lent/lent is a typed/Outcome")
}

func TestCompilePartsAreChecked(t *testing.T) {
	_, ds := typed(t, lending(map[string]string{
		"members/operations/lend/notes/why.md":         def("typed/Condition", "# Why\n"),
		"members/operations/lend/beside.md":            def("typed/Condition", "# Beside\n"),
		"members/operations/lend/conditions/wrong.md":  def("typed/Outcome", "# Wrong\n"),
		"members/operations/lend/outcomes/deep/x/y.md": def("typed/Condition", "# Deep\n"),
		"members/notes/remark/remark.md":               def("typed/Note", "# Remark\n"),
		"members/notes/remark/more.md":                 def("typed/Note", "# More\n"),
		"members/notes/remark/sub/sub.md":              def("typed/Note", "# Sub, a part with parts\n"),
		"members/notes/remark/sub/x.md":                def("typed/Note", "# X\n"),
		"members/notes/remark/odd.md":                  def("typed/Goal", "# Odd\n"),
		"members/notes/remark/folder/in.md":            def("typed/Note", "# In a folder\n"),
		"members/leaves/leaf/leaf.md":                  def("typed/Leaf", "# Leaf\n"),
		"members/leaves/leaf/part.md":                  def("typed/Leaf", "# Part\n"),
	}))
	expect(t, ds, "part.folder", "members/operations/lend/notes/why is a part of members/operations/lend, and a typed/Operation has parts only under conditions/, under outcomes/")
	expect(t, ds, "part.folder", "members/operations/lend/beside is a part of members/operations/lend")
	if d := expect(t, ds, "part.kind", "what lies under conditions/ of a typed/Operation is a typed/Condition; this is a typed/Outcome"); d.Line != 2 {
		t.Fatalf("where: %v", d)
	}
	expect(t, ds, "part.kind", "what lies under outcomes/ of a typed/Operation is a typed/Outcome; this is a typed/Condition")
	expect(t, ds, "part.kind", "what lies directly in the folder of a typed/Note is a typed/Note; this is a typed/Goal")
	expect(t, ds, "part.folder", "members/notes/remark/folder/in is a part of members/notes/remark, and a typed/Note has parts only directly in the folder")
	expect(t, ds, "part.folder", "a typed/Leaf has parts only nowhere: it has none")
	if len(ds) != 7 {
		t.Fatalf("want seven findings:\n%v", ds)
	}
}

func TestCompileADeclarationIsChecked(t *testing.T) {
	kind := func(front string) string { return "---\nkind: Kind\n" + front + "---\n# K\n" }
	for front, want := range map[string]string{
		"fields: [Who]\n":                     "fields is a mapping of field labels",
		"fields:\n  Who: yes\n":               `the field "Who" is declared with nothing after it`,
		"fields:\n  Who: {must: true}\n":      `a field declares only required and links, not "must"`,
		"fields:\n  Who: {required: 1}\n":     "required is true or false",
		"fields:\n  Who:\n  Who:\n":           `the field "Who" is declared twice`,
		"parts:\n  a: k/K\n  a: k/K\n":        `the folder "a" is declared twice`,
		"fields:\n  \"W:\":\n":                "a field is named by its label as written between ** and :",
		"fields:\n  Who: {links: [a b]}\n":    "links names Kinds by path",
		"fields:\n  Who: {links: nothing}\n":  `"nothing" names no Kind`,
		"fields:\n  Who: {links: k/other}\n":  `"k/other" names no Kind`,
		"parts: conditions\n":                 "parts is a mapping of folders",
		"parts:\n  a/b: typed/Goal\n":         "parts are declared by the folder they lie in",
		"parts:\n  conditions: [typed/No]\n":  `"typed/No" names no Kind`,
		"parts:\n  conditions: {a: b}\n":      "parts names Kinds by path",
		"fields:\n  Who: {links: [Kind]}\n":   "",
		"parts:\n  .: [k/K, Package]\n":       "",
		"fields:\n  Who:\nparts:\n  x: k/K\n": "",
	} {
		_, ds := typed(t, map[string]string{
			"k/K.md":     kind(front),
			"k/other.md": def("typed/Goal", "# Other\n"),
		})
		switch {
		case want == "":
			clean(t, ds)
		default:
			if d := expect(t, ds, "kind.schema", want); d.File != ".markitect/spec/k/K.md" || d.Line < 3 {
				t.Errorf("%q: where: %v", front, d)
			}
		}
	}
	// Only a Kind declares.
	_, ds := typed(t, map[string]string{"goal.md": "---\nkind: typed/Goal\nfields:\n  Who:\n---\n# Goal\n"})
	expect(t, ds, "definition.front_matter", "only a Kind declares fields, parts, links, realized and binds")
	if len(ds) != 1 {
		t.Fatalf("want one finding:\n%v", ds)
	}
}

func TestCompileAProjectKindDeclares(t *testing.T) {
	_, ds := typed(t, map[string]string{
		"kinds/Step.md": "---\nkind: Kind\nfields:\n  After: {links: kinds/Step}\n---\n# Step\n",
		"one.md":        def("kinds/Step", "# One\n"),
		"two.md":        def("kinds/Step", "# Two\n\n**After:** [[one]].\n"),
		"three.md":      def("kinds/Step", "# Three\n\n**After:** [[kinds/Step]].\n"),
	})
	expect(t, ds, "field.target", `"After" of a kinds/Step links a kinds/Step; kinds/Step is a Kind`)
	if len(ds) != 1 {
		t.Fatalf("want one finding:\n%v", ds)
	}
}
