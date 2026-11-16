package spec

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseMeasure(t *testing.T) {
	for text, want := range map[string]Measure{
		"`p95 < 300 ms`":     {Number: 300, Unit: "ms", Op: "<", Of: "p95"},
		"p99<=2s under load": {Number: 2, Unit: "s", Op: "<=", Of: "p99"},
		"`30 days`":          {Number: 30, Unit: "days"},
		"reachable `99` % of each calendar month":                {Number: 99, Unit: "%"},
		"the server answers within `1` second for `95` of `100`": {Number: 1, Unit: "second", Op: "<="},
		"at most `4` pages of choices":                           {Number: 4, Unit: "pages", Op: "<="},
		"at least 3,000 appointments a month":                    {Number: 3000, Unit: "appointments", Op: ">="},
		"`95` of `100` calls":                                    {Number: 100, Unit: "calls"},
		"no more than 5 per minute":                              {Number: 5, Unit: "per minute", Op: "<="},
		"a request body of at most `64` KB.":                     {Number: 64, Unit: "KB", Op: "<="},
		"meets WCAG `2.1` level `AA`":                            {Number: 2.1, Unit: "level"},
		"a hover of 2 s":                                         {Number: 2, Unit: "s"},
		// A number with a comparison counts before one without: the load
		// stays text.
		"At `100` users, `p95 < 300 ms`": {Number: 300, Unit: "ms", Op: "<", Of: "p95"},
		"a 404 page within 2 s":          {Number: 2, Unit: "s", Op: "<="},
		// Without a comparison the first number with a unit counts.
		"a 404 page in 2 s": {Number: 404, Unit: "page"},
	} {
		got, ok := ParseMeasure(text)
		if !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("ParseMeasure(%q) = %+v, %v; want %+v", text, got, ok, want)
		}
	}
	for _, text := range []string{
		"started on a fresh Linux server with Docker by the steps of the install Task alone.",
		"at `03:30` every night",
		"the p95 of the answers",
		"version 1.2.3",
		"ISO 25010 only",
		"",
	} {
		if got, ok := ParseMeasure(text); ok {
			t.Errorf("ParseMeasure(%q) = %+v; want no measure", text, got)
		}
	}
	if !SaysNoMeasure("none is promised: the host decides") || SaysNoMeasure("started by the steps alone") {
		t.Error("SaysNoMeasure reads the words that say none is promised")
	}
}

func TestEnumValue(t *testing.T) {
	values := []string{"sync", "async"}
	for text, want := range map[string]string{
		"`sync`.":                             "sync",
		"async, the mail goes out later":      "async",
		"Sync: the page waits for the answer": "sync",
		"`async`":                             "async",
		"**sync**, the page waits":            "sync",
		"\"sync\"":                            "sync",
		"Async!":                              "async",
		"(async)":                             "async",
	} {
		if got, ok := EnumValue(text, values); !ok || got != want {
			t.Errorf("EnumValue(%q) = %q, %v; want %q", text, got, ok, want)
		}
	}
	for _, text := range []string{"sometimes", "it is sync", "synchronous", ""} {
		if got, ok := EnumValue(text, values); ok {
			t.Errorf("EnumValue(%q) = %q; want none", text, got)
		}
	}
}

func TestCodeReadsTheFirstCodeInBackticks(t *testing.T) {
	if got := Code("[[x]] and [[y]]: `too_late`, then `other`."); got != "too_late" {
		t.Fatalf("Code = %q", got)
	}
	if got := Codes("`a`, b and ` c `"); !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatalf("Codes = %q", got)
	}
	// A code in a link's label is no code of the text.
	owned := "[[catalog/conditions/isbn|`isbn`]] is owned: `book_unknown`."
	if got := Code(owned); got != "book_unknown" {
		t.Fatalf("Code of a text with a label in backticks = %q", got)
	}
	if got := Codes(owned); !reflect.DeepEqual(got, []string{"book_unknown"}) {
		t.Fatalf("Codes of a text with a label in backticks = %q", got)
	}
}

// formKinds is a Library whose Kinds declare the form of their fields' values.
var formKinds = LibrarySource{Name: "form", Version: "1.0.0", Dir: "form", Files: []File{
	{Path: "Condition.md", Data: []byte("---\nkind: Kind\n---\n# Condition\n")},
	{Path: "Connector.md", Data: []byte("---\nkind: Kind\nfields:\n  Mode: {type: enum, values: [sync, async]}\n---\n# Connector\n")},
	{Path: "Quality.md", Data: []byte("---\nkind: Kind\nfields:\n  Measure: {type: measure, vet: true}\n  Budget: {type: measure}\n---\n# Quality\n")},
	{Path: "Operation.md", Data: []byte("---\nkind: Kind\nfields:\n  Checked: {items: {links: form/Condition, code: true}, vet: true}\n  Steps: {items: {code: true}}\n---\n# Operation\n")},
	{Path: "Page.md", Data: []byte("---\nkind: Kind\nfields:\n  Path: {unique: true}\n---\n# Page\n")},
}}

func formed(t *testing.T, files map[string]string) (*Spec, Diagnostics) {
	t.Helper()
	return compile(t, files, formKinds)
}

// check holds a field's value to the form its Kind declares, unless the Kind
// leaves it to vet: then the problem is there for vet to note, and check
// passes.
func TestCompileAValueIsHeldToItsForm(t *testing.T) {
	files := map[string]string{
		"c/free.md":      def("form/Condition", "# Free\n"),
		"c/open.md":      def("form/Condition", "# Open\n"),
		"api.md":         def("form/Connector", "# API\n\n**Mode:** `sync`, the page waits.\n"),
		"mail.md":        def("form/Connector", "# Mail\n\n**Mode:** sometimes.\n"),
		"fast.md":        def("form/Quality", "# Fast\n\n**Measure:** `p95 < 300 ms`.\n\n**Budget:** `30` euros a month.\n"),
		"vague.md":       def("form/Quality", "# Vague\n\n**Measure:** quick enough.\n\n**Budget:** small.\n"),
		"free.md":        def("form/Quality", "# Free\n\n**Measure:** none is promised.\n"),
		"lend.md":        def("form/Operation", "# Lend\n\n**Checked:**\n\n1. [[c/free]]: `taken`.\n2. [[c/open]] and [[c/free]]: `closed`.\n3. [[c/open]].\n4. nothing: `odd`.\n\n**Steps:**\n\n- run `a`\n- then b\n"),
		"one-line.md":    def("form/Operation", "# One line\n\n**Checked:** [[c/free]]: `taken`.\n\n**Steps:** run `a`.\n"),
		"site/site.md":   anchor("Site"),
		"site/home.md":   def("form/Page", "# Home\n\n**Path:** `/`.\n"),
		"site/start.md":  def("form/Page", "# Start\n\n**Path:** `/`, the same.\n"),
		"site/about.md":  def("form/Page", "# About\n\n**Path:** `/about`.\n"),
		"other/other.md": anchor("Other"),
		"other/home.md":  def("form/Page", "# Home of another site\n\n**Path:** `/`.\n"),
	}
	sp, ds := formed(t, files)
	expect(t, ds, "field.value", `"Mode" of a form/Connector is one of `+"`sync`, `async`"+`, in backticks or plain`)
	if d := expect(t, ds, "field.value", `"Budget" of a form/Quality is a number with its unit`); d.File != ".markitect/spec/vague.md" {
		t.Fatalf("where: %v", d)
	}
	if d := expect(t, ds, "field.value", `entry 2 of "Steps" lacks a code in backticks`); d.Line != 16 {
		t.Fatalf("where: %v", d)
	}
	if d := expect(t, ds, "field.unique", `site/start gives the "Path" /, as site/home does; two Definitions of form/Page in site have the same value, and each needs a different one`); d.File != ".markitect/spec/site/start.md" {
		t.Fatalf("where: %v", d)
	}
	if len(ds) != 4 {
		t.Fatalf("want four errors, none for what vet notes:\n%v", ds)
	}
	var noted []string
	for _, p := range sp.ValueProblems() {
		if p.Vet {
			noted = append(noted, p.File[strings.LastIndex(p.File, "/")+1:]+": "+p.Message)
		}
	}
	want := []string{
		`lend.md: entry 3 of "Checked" lacks a code in backticks; each entry of it holds a link to a form/Condition and a code in backticks`,
		`lend.md: entry 4 of "Checked" lacks a link to a form/Condition; each entry of it holds a link to a form/Condition and a code in backticks`,
		`vague.md: "Measure" of a form/Quality is a number with its unit, such as ` + "`30 days` or `p95 < 300 ms`" + `, or says that none is promised`,
	}
	if !reflect.DeepEqual(noted, want) {
		t.Fatalf("left to vet:\n%s\nwant:\n%s", strings.Join(noted, "\n"), strings.Join(want, "\n"))
	}

	// The typed value of a field.
	connector := sp.Definitions["form/Connector"].Schema.Field("Mode")
	if v := FieldValue(connector, sp.Definitions["api"].Fields[0]); v != "sync" {
		t.Fatalf("Mode of api = %v", v)
	}
	quality := sp.Definitions["form/Quality"].Schema
	if v := FieldValue(quality.Field("Measure"), sp.Definitions["fast"].Fields[0]); !reflect.DeepEqual(v, Measure{Number: 300, Unit: "ms", Op: "<", Of: "p95"}) {
		t.Fatalf("Measure of fast = %#v", v)
	}
	if v := FieldValue(quality.Field("Measure"), sp.Definitions["vague"].Fields[0]); v != nil {
		t.Fatalf("a value out of form has no typed value: %#v", v)
	}
	if v := FieldValue(nil, sp.Definitions["api"].Fields[0]); v != nil {
		t.Fatalf("a field without a declaration has no typed value: %#v", v)
	}
}

func TestCompileADeclarationOfAFormIsChecked(t *testing.T) {
	kind := func(front string) string { return "---\nkind: Kind\n" + front + "---\n# K\n" }
	for front, want := range map[string]string{
		"fields:\n  M: {type: number}\n":                                                      "type is enum, one of the values the field names, or measure",
		"fields:\n  M: {type: enum}\n":                                                        `the field "M" of type enum names its values`,
		"fields:\n  M: {type: enum, values: sync}\n":                                          "values is a list of words",
		"fields:\n  M: {type: enum, values: [a, a]}\n":                                        "each value is one word, named once",
		"fields:\n  M: {type: enum, values: [a b]}\n":                                         "each value is one word, named once",
		"fields:\n  M: {type: enum, values: [v1.0, v2]}\n":                                    `the value "v1.0" holds a mark that ends or wraps a word in the text`,
		"fields:\n  M: {type: enum, values: [read_only]}\n":                                   `the value "read_only" holds a mark`,
		"fields:\n  M: {values: [a, b]}\n":                                                    `the field "M" names values, which only a field of type enum has`,
		"fields:\n  M: {type: measure, items: {code: true}}\n":                                `the field "M" holds a value of a type or a list of items, not both`,
		"fields:\n  M: {vet: true}\n":                                                         `the field "M" says vet, which is for the form of its value`,
		"fields:\n  M: {links: k/K, vet: true}\n":                                             `the field "M" says vet, which is for the form of its value`,
		"fields:\n  M: {unique: 1}\n":                                                         "unique is true or false",
		"fields:\n  M: {items: code}\n":                                                       "items says what each entry holds",
		"fields:\n  M: {items: {code: yes please}}\n":                                         "code is true or false",
		"fields:\n  M: {items: {links: }}\n":                                                  "items names the Kinds each entry links",
		"fields:\n  M: {items: {size: 3}}\n":                                                  `items declares only links and code, not "size"`,
		"fields:\n  M: {items: {links: k/none}}\n":                                            `"k/none" names no Kind`,
		"fields:\n  M: {type: enum, values: [a, b], vet: true}\n":                             "",
		"fields:\n  M: {type: measure, unique: true}\n":                                       "",
		"fields:\n  M: {links: k/K, items: {links: [k/K, Package], code: true}, vet: true}\n": "",
	} {
		_, ds := formed(t, map[string]string{"k/K.md": kind(front)})
		switch {
		case want == "":
			clean(t, ds)
		default:
			if d := expect(t, ds, "kind.schema", want); d.File != ".markitect/spec/k/K.md" || d.Line < 3 {
				t.Errorf("%q: where: %v", front, d)
			}
		}
	}
}
