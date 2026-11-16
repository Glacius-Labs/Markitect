package analysis

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// typedSpec adds to the small library project an Operation, its Outcome, a
// Concept with three States, the refusal codes, the roles with one Role, a
// Quality and a Connector of a project Kind.
func typedSpec(t *testing.T) memRepo {
	t.Helper()
	r := library(t)
	r[m+"rules/refusal-codes.md"] = def("std/Rule", "Refusal codes", "A refused call answers with one of these:\n\n- `taken`: someone holds it.")
	r[m+"rules/roles/roles.md"] = def("std/Rule", "Roles", "Who may call what.")
	r[m+"rules/roles/member.md"] = def("std/Role", "Member", "A signed-in member.\n\n**Granted by:** signing in.")
	r[m+"catalog/copy/copy.md"] = def("std/Concept", "Copy", "One copy of a book.")
	for _, s := range []string{"free", "lent", "lost"} {
		r[m+"catalog/copy/states/"+s+".md"] = def("std/State", strings.ToUpper(s[:1])+s[1:], "A state of a [[catalog/copy]].")
	}
	r[m+"catalog/operations/lend/lend.md"] = def("std/Operation", "Lend", "Lends a copy.\n\n"+
		"**Who:** `member`.\n\n"+
		"**Checked in this order:**\n\n"+
		"1. [[catalog/operations/lend/conditions/free]]: `taken`.\n"+
		"2. [[catalog/operations/lend/conditions/free]]: `gone`.\n"+
		"3. [[catalog/operations/lend/conditions/free]].\n\n"+
		"**Satisfies:** [[loan-limit]].")
	r[m+"catalog/operations/lend/conditions/free.md"] = def("std/Condition", "Free", "Nobody holds the copy.")
	r[m+"catalog/operations/lend/outcomes/lent.md"] = def("std/Outcome", "Lent", "The copy is lent.\n\n**Leaves in:** [[catalog/copy/states/lent]].")
	r[m+"catalog/operations/give-back/give-back.md"] = def("std/Operation", "Give back", "Takes a copy back.\n\n"+
		"**Who:** [[rules/roles/member]].\n\n"+
		"**Checked in this order:** [[catalog/operations/lend/conditions/free]]: `taken`.\n\n"+
		"**Satisfies:** [[loan-limit]].")
	r[m+"quality/fast.md"] = def("std/Quality", "Fast", "Answers fast.\n\n**Measure:** `p95 < 300 ms` at `10` calls a second.")
	r[m+"quality/vague.md"] = def("std/Quality", "Vague", "Answers well.\n\n**Measure:** quickly enough.")
	r[m+"quality/open.md"] = def("std/Quality", "Open", "Asks nothing.\n\n**Measure:** none is promised.")
	return r
}

func notesOf(t *testing.T, r memRepo, codes ...string) []string {
	t.Helper()
	var got []string
	for _, n := range Vet(loadRepo(t, r), History{}).Notes {
		for _, c := range codes {
			if n.Code == c {
				got = append(got, strings.TrimPrefix(n.File, m)+": "+n.Code)
			}
		}
	}
	sort.Strings(got)
	return got
}

// vet notes a value out of the form its Kind leaves to vet, and what std's
// typed fields show across Definitions.
func TestVetNotesFormsAndStdFields(t *testing.T) {
	got := notesOf(t, typedSpec(t), "vet.field_value", "vet.refusal_code", "vet.unreached_state", "vet.who_role")
	want := []string{
		"catalog/copy/states/free.md: vet.unreached_state",
		"catalog/copy/states/lost.md: vet.unreached_state",
		"catalog/operations/lend/lend.md: vet.field_value",
		"catalog/operations/lend/lend.md: vet.refusal_code",
		"catalog/operations/lend/lend.md: vet.who_role",
		"quality/vague.md: vet.field_value",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notes:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, n := range Vet(loadRepo(t, typedSpec(t)), History{}).Notes {
		switch n.Code {
		case "vet.refusal_code":
			if !strings.Contains(n.Message, "entry 2 refuses with `gone`, which no Rule or Refusal of the spec states") || n.Line != 13 {
				t.Errorf("refusal code: %v", n.Diagnostic)
			}
		case "vet.who_role":
			if n.Line != 8 {
				t.Errorf("who: %v", n.Diagnostic)
			}
		}
	}
}

// A spec that has no Roles hears nothing of Who, and a Concept none of whose
// States an Outcome names hears nothing of its States: a spec written before
// these fields stays quiet.
func TestVetStdNotesWaitForTheSpecToUseTheFields(t *testing.T) {
	r := typedSpec(t)
	delete(r, m+"rules/roles/member.md")
	r[m+"catalog/operations/give-back/give-back.md"] = strings.Replace(r[m+"catalog/operations/give-back/give-back.md"], "[[rules/roles/member]]", "`member`", 1)
	r[m+"catalog/operations/lend/outcomes/lent.md"] = def("std/Outcome", "Lent", "The copy is lent.")
	if got := notesOf(t, r, "vet.unreached_state", "vet.who_role"); len(got) != 0 {
		t.Fatalf("notes on a spec without Roles and Leaves in: %v", got)
	}
}

// A code in backticks in a link's label is no refusal code: vet and doc read
// the code after the link.
func TestALinkLabelInBackticksIsNoCode(t *testing.T) {
	r := typedSpec(t)
	r[m+"catalog/operations/add/add.md"] = def("std/Operation", "Add", "Adds a book.\n\n"+
		"**Who:** [[rules/roles/member]].\n\n"+
		"**Checked in this order:**\n\n1. [[catalog/operations/add/conditions/isbn|`isbn`]] is owned: `book_unknown`.\n\n"+
		"**Satisfies:** [[loan-limit]].")
	r[m+"catalog/operations/add/conditions/isbn.md"] = def("std/Condition", "ISBN", "The ISBN names a book the library owns.")
	var found bool
	for _, n := range Vet(loadRepo(t, r), History{}).Notes {
		if n.File != m+"catalog/operations/add/add.md" {
			continue
		}
		switch n.Code {
		case "vet.refusal_code":
			found = true
			if !strings.Contains(n.Message, "entry 1 refuses with `book_unknown`,") {
				t.Errorf("refusal code: %v", n.Diagnostic)
			}
		case "vet.field_value":
			t.Errorf("the entry holds its link and its code: %v", n.Diagnostic)
		}
	}
	if !found {
		t.Fatal("no note of the unstated code book_unknown")
	}
	q := loadRepo(t, r)
	add := DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/add/add"])
	if items := add.Fields[1].Items; len(items) != 1 || items[0].Code != "book_unknown" {
		t.Fatalf("entries = %+v", items)
	}
	data, err := json.Marshal(add)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"code":"book_unknown"`) || strings.Contains(string(data), `"code":"isbn"`) {
		t.Fatalf("json of the entry: %s", data)
	}
}

// doc gives the typed value of a field whose Kind declares a type, and the
// code of each entry of a list whose Kind says each entry holds one.
func TestDocGivesTypedValues(t *testing.T) {
	r := typedSpec(t)
	r[m+"wire.md"] = "---\nkind: Kind\nfields:\n  Mode: {type: enum, values: [sync, async]}\n---\n# Wire\n\nHow two parts talk.\n"
	r[m+"catalog/mail.md"] = def("wire", "Mail", "Sends mail.\n\n**Mode:** `async`, nobody waits.")
	q := loadRepo(t, r)
	doc := func(id string) DocDefinition { return DocDefinitionOf(q, q.Spec.Definitions[id]) }

	if v := doc("catalog/mail").Fields[0].Value; v != "async" {
		t.Fatalf("Mode = %#v", v)
	}
	if v := doc("quality/fast").Fields[0].Value; !reflect.DeepEqual(v, spec.Measure{Number: 300, Unit: "ms", Op: "<", Of: "p95"}) {
		t.Fatalf("Measure = %#v", v)
	}
	if v := doc("quality/vague").Fields[0].Value; v != nil {
		t.Fatalf("a value out of form: %#v", v)
	}
	lend := doc("catalog/operations/lend/lend")
	var codes []string
	for _, it := range lend.Fields[1].Items {
		codes = append(codes, it.Code)
	}
	if !reflect.DeepEqual(codes, []string{"taken", "gone", ""}) {
		t.Fatalf("codes = %q", codes)
	}

	data, err := json.Marshal(doc("quality/fast"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"value":{"number":300,"unit":"ms","op":"\u003c","of":"p95"}`) {
		t.Fatalf("json of a measure: %s", data)
	}
	data, _ = json.Marshal(lend)
	if !strings.Contains(string(data), `"code":"taken"`) || strings.Contains(string(data), `"value"`) {
		t.Fatalf("json of the entries: %s", data)
	}

	// A Kind's declaration shows the form.
	op := doc("std/Operation").Declares
	var checked DocDeclaredField
	for _, f := range op.Fields {
		if f.Name == "Checked in this order" {
			checked = f
		}
	}
	if checked.Items == nil || !checked.Items.Code || !reflect.DeepEqual(checked.Items.Links, []string{"std/Condition"}) || !checked.Vet {
		t.Fatalf("Checked in this order declares %+v", checked)
	}
	if f := doc("wire").Declares.Fields[0]; f.Type != "enum" || !reflect.DeepEqual(f.Values, []string{"sync", "async"}) {
		t.Fatalf("Mode declares %+v", f)
	}
}
