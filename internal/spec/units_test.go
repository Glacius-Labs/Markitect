package spec

import (
	"reflect"
	"strings"
	"testing"
)

// What is built and verified as one: a Definition that is no part of another
// with the parts it covers. A part with parts of its own is a unit of its
// own, a part in a folder of its name with no parts is covered like any
// other, and a Package's anchor and what lies in a Package are units.
func TestUnits(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"rule.md":                      def("std/Rule", "# Alone\n"),
		"lend/lend.md":                 def("std/Rule", "# Lend\n"),
		"lend/free.md":                 def("std/Rule", "# Free\n"),
		"lend/alone/alone.md":          def("std/Rule", "# A part in a folder of its name\n"),
		"lend/due/due.md":              def("std/Rule", "# Due, a part with parts\n"),
		"lend/due/after.md":            def("std/Rule", "# After\n"),
		"lend/due/soon/soon.md":        def("std/Rule", "# Soon, a part of due with a part\n"),
		"lend/due/soon/tomorrow.md":    def("std/Rule", "# Tomorrow\n"),
		"lend/outcomes/lent.md":        def("std/Rule", "# Lent\n"),
		"members/members.md":           anchor("Members"),
		"members/member.md":            def("std/Rule", "# Member\n"),
		"members/cards/card/card.md":   def("std/Rule", "# Card\n"),
		"members/cards/card/number.md": def("std/Rule", "# Number\n"),
	})
	clean(t, ds)
	want := map[string][]string{
		"rule":                    {},
		"lend/lend":               {"lend/alone/alone", "lend/free", "lend/outcomes/lent"},
		"lend/due/due":            {"lend/due/after"},
		"lend/due/soon/soon":      {"lend/due/soon/tomorrow"},
		"members/members":         {},
		"members/member":          {},
		"members/cards/card/card": {"members/cards/card/number"},
	}
	if got := m.Units(); !reflect.DeepEqual(got, want) {
		t.Fatalf("units = %v", got)
	}
	for id, unit := range map[string]string{
		"rule": "rule", "lend/lend": "lend/lend", "lend/free": "lend/lend", "lend/alone/alone": "lend/lend",
		"lend/due/due": "lend/due/due", "lend/due/after": "lend/due/due", "lend/due/soon/tomorrow": "lend/due/soon/soon",
		"members/members": "members/members", "members/cards/card/number": "members/cards/card/card", "gone": "gone",
	} {
		if got := m.UnitOf(id); got != unit {
			t.Errorf("unit of %s = %s, want %s", id, got, unit)
		}
	}
}

// A field starts a line of a top-level paragraph, also the line after another
// field without an empty line between them, and in a file with Windows line
// ends. A label in a list, in a quotation or inside a sentence is none.
func TestFieldsStartALine(t *testing.T) {
	body := "# Lend\n\nLends, and **Who:** inside a sentence is no field.\n\n**Who:** `anyone`.\n**Given:** `copyId`,\nsee [[goals/lend-books]].\n**Serves:** [[goals/lend-books]].\n\n- **Notes:** in a list.\n\n> **Notes:** in a quotation.\n"
	for _, text := range []string{body, strings.ReplaceAll(body, "\n", "\r\n")} {
		m, ds := typed(t, lending(map[string]string{"members/operations/lend/lend.md": def("typed/Operation", text)}))
		clean(t, ds)
		op := m.Definitions["members/operations/lend/lend"]
		var labels []string
		for _, f := range op.Fields {
			labels = append(labels, f.Label)
		}
		if want := []string{"Who", "Given", "Serves"}; !reflect.DeepEqual(labels, want) {
			t.Fatalf("fields = %q", labels)
		}
		if got := op.Links[1].Field + "," + op.Links[2].Field; got != "Given,Serves" {
			t.Fatalf("fields of the links = %q", got)
		}
		if line := op.Fields[1].Line; line != 9 {
			t.Fatalf("Given stands on line %d", line)
		}
	}
}

// A declaration reads YAML's booleans however they are capitalised, refuses a
// key said twice, and a Kind whose declaration has an error holds its
// Definitions to nothing, so that one error stays one.
func TestCompileDeclarationsAreReadWithCare(t *testing.T) {
	m, ds := typed(t, map[string]string{
		"kinds/K.md": "---\nkind: Kind\nrealized: False\nfields:\n  Who: {required: True}\n  Why: {required: FALSE}\n---\n# K\n",
		"one.md":     "---\nkind: kinds/K\ninternal: True\n---\n# One\n",
	})
	expect(t, ds, "field.missing", `states "Who"`)
	if len(ds) != 1 {
		t.Fatalf("want one finding:\n%v", ds)
	}
	if s := m.Definitions["kinds/K"].Schema; !s.Unrealized || !s.Field("Who").Required || s.Field("Why").Required || !m.Definitions["one"].Internal {
		t.Fatalf("declares %+v, internal %v", s, m.Definitions["one"].Internal)
	}
	_, ds = typed(t, map[string]string{
		"kinds/K.md":       "---\nkind: Kind\nfields:\n  Who: {required: true, required: false}\n---\n# K\n",
		"kinds/P.md":       "---\nkind: Kind\nparts:\n  notes: [no kind]\n---\n# P\n",
		"one.md":           def("kinds/K", "# One\n\n**Why:** no field, and Who is missing.\n"),
		"thing/thing.md":   def("kinds/P", "# Thing\n"),
		"thing/notes/a.md": def("kinds/K", "# A\n"),
		"thing/b.md":       def("kinds/K", "# B\n"),
	})
	expect(t, ds, "kind.schema", `the field "Who" says required twice`)
	expect(t, ds, "kind.schema", "parts names Kinds by path")
	if len(ds) != 2 {
		t.Fatalf("a broken declaration checks nothing:\n%v", ds)
	}
	// So does one that names something that is no Kind.
	_, ds = typed(t, map[string]string{
		"kinds/K.md": "---\nkind: Kind\nfields:\n  Who: {required: true, links: one}\n---\n# K\n",
		"one.md":     def("kinds/K", "# One\n"),
	})
	expect(t, ds, "kind.schema", `"one" names no Kind`)
	if len(ds) != 1 {
		t.Fatalf("a declaration that names no Kind checks nothing:\n%v", ds)
	}
}

// A label may follow a hard line break and may begin with code or an escaped
// character: the field starts at the two characters that open the label.
func TestFieldsStartAtTheirLabel(t *testing.T) {
	body := "# T\n\nText.\n\n**Who:** a.  \n**`Given`:** b.\\\n__Serves:__ c.\n**\\*Notes:** d.\n"
	page, err := ParsePage([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range page.Fields() {
		got = append(got, f.Label+"="+string(page.Body[f.Start:f.Start+2]))
	}
	if want := []string{"Who=**", "Given=**", "Serves=__", `\*Notes=**`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %q", got)
	}
}
