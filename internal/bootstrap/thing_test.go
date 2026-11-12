package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A thing with parts through the commands: check holds it to what its Kind
// declares, status lists it as one item, mark records one verdict on it and
// refuses one on a part, and doc of a part shows the record of its whole.
func TestAThingWithPartsThroughTheCommands(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Library")
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	const spec = ".markitect/spec/"
	remove := func(file string) {
		t.Helper()
		if err := os.Remove(filepath.Join(r.Dir, filepath.FromSlash(spec+file))); err != nil {
			t.Fatal(err)
		}
	}
	r.Write(spec+"purpose/goals/lend-books.md", def("std/Goal", "Lend books", "**Result:** books go home.\n**Changes:** shelves empty.\n**Obstacle:** paper cards."))
	r.Write(spec+"purpose/exclusions/fines.md", def("std/Exclusion", "Fines", "Fines are left out."))
	r.Write(spec+"domain/rules/roles.md", def("std/Rule", "Roles", "`anyone` is any caller."))
	r.Write(spec+"domain/operations/lend/lend.md", def("std/Operation", "Lend", "Lends a copy.\n\n**Who:** `anyone`.\n**Checked in this order:**\n\n1. [[domain/operations/lend/conditions/in-stock]]: `out`.\n\n**Satisfies:** [[purpose/goals/lend-books]]."))
	r.Write(spec+"domain/operations/lend/conditions/in-stock.md", def("std/Condition", "In stock", "A copy is on the shelf."))
	r.Write(spec+"domain/operations/lend/outcomes/lent.md", def("std/Outcome", "Lent", "The copy is lent."))
	r.Write(".markitect/registry.md", "# Registry: Root\n\n| File | Realizes | Why |\n|---|---|---|\n| `lend.go` | [[domain/operations/lend]] [[domain/rules/roles]] | Lending |\n")
	r.Write("lend.go", "package library\n")
	if out := mustRun(t, r.Dir, "check"); !strings.HasPrefix(out, "ok:") {
		t.Fatalf("check: %s", out)
	}

	// What the Kind declares is checked: a part in a folder it does not name, a
	// field it does not have, a required field left out and a link a field may
	// not name.
	r.Write(spec+"domain/operations/lend/notes/why.md", def("std/Outcome", "Why", "A note."))
	r.Write(spec+"domain/operations/odd/odd.md", def("std/Operation", "Odd", "Odd.\n\n**Who:** `anyone`.\n**Because:** no such field.\n**Checked in this order:** [[domain/rules/roles]]."))
	_, out, _ := run(t, r.Dir, "check", "--spec")
	wantAll(t, out,
		"domain/operations/lend/notes/why.md: part.folder: domain/operations/lend/notes/why is a part of domain/operations/lend, and a std/Operation has parts only under conditions/, under outcomes/, under results/",
		`field.unknown: a std/Operation has no field "Because"`,
		`field.missing: a std/Operation states "Satisfies": add a line that starts with **Satisfies:**`,
		`field.target: "Checked in this order" of a std/Operation links a std/Condition; domain/rules/roles is a std/Rule`,
		"4 errors: 4 in the spec")
	remove("domain/operations/lend/notes/why.md")
	remove("domain/operations/odd/odd.md")

	// status: one item for the thing, none for its parts, and none of the
	// Exclusion is missing from the registry.
	_, out, _ = run(t, r.Dir, "status")
	wantAll(t, out, "To verify (5 of 5 items)", "domain/operations/lend/lend@", "with its 2 parts, verified as one; markitect doc domain/operations/lend shows it whole")
	if strings.Contains(out, "in-stock@") || strings.Contains(out, "definition.unused") {
		t.Fatalf("status:\n%s", out)
	}
	var st struct {
		Definitions, Units int
		Items              []struct {
			Definition, Fingerprint string
			Parts                   []string
		}
	}
	_, out, _ = run(t, r.Dir, "status", "--json")
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatalf("status --json: %v\n%s", err, out)
	}
	item := ""
	for _, it := range st.Items {
		if it.Definition == "domain/operations/lend/lend" {
			item = it.Definition + "@" + it.Fingerprint
			if strings.Join(it.Parts, ",") != "domain/operations/lend/conditions/in-stock,domain/operations/lend/outcomes/lent" {
				t.Fatalf("parts in JSON: %v", it.Parts)
			}
		}
	}
	if item == "" || st.Units != 5 || st.Definitions != 7 {
		t.Fatalf("status --json: %d units of %d Definitions, item %q", st.Units, st.Definitions, item)
	}

	// mark: refused on a part, recorded on the thing.
	_, out, _ = run(t, r.Dir, "mark", "domain/operations/lend/conditions/in-stock@"+item[strings.Index(item, "@")+1:], "ok", "-e", "lend.go", "--cases", "1", "--covered", "1")
	wantAll(t, out, "domain/operations/lend/conditions/in-stock is a part of domain/operations/lend/lend, which is verified as one with its parts", "Nothing is recorded.")
	mustRun(t, r.Dir, "mark", item, "ok", "-e", "lend.go", "--cases", "3", "--covered", "3")

	// doc: the thing whole, and the part with the record of its whole.
	wantAll(t, mustRun(t, r.Dir, "doc", "domain/operations/lend"), "conditions:\n\n  In stock", "outcomes:\n\n  Lent", "Record: "+item+", ok")
	wantAll(t, mustRun(t, r.Dir, "doc", "domain/operations/lend/outcomes/lent"), "Part of domain/operations/lend", "Record: "+item+", ok")

	// A changed part puts the thing back on the list.
	r.Write(spec+"domain/operations/lend/outcomes/lent.md", def("std/Outcome", "Lent", "The copy is lent for `14` days."))
	_, out, _ = run(t, r.Dir, "status")
	wantAll(t, out, "domain/operations/lend/lend@", "changed: part:domain/operations/lend/outcomes/lent")

	// list --packages prints the Packages alone.
	if out := mustRun(t, r.Dir, "list", "--packages", "--deps"); !strings.HasPrefix(out, "Root (Ca 0, Ce 0, ") || !strings.Contains(out, ", 7 Definitions\n") || strings.Contains(out, "domain/operations/lend/lend") {
		t.Fatalf("list --packages:\n%s", out)
	}

	// doc of a Kind prints what it declares.
	wantAll(t, mustRun(t, r.Dir, "doc", "std/Operation"), "Declares the fields:\n  Who, required\n", "Declares the parts:\n  under conditions/: std/Condition\n")
	wantAll(t, mustRun(t, r.Dir, "doc", "std/Exclusion"), "Declares that no file realizes a Definition of this Kind.")
	wantAll(t, mustRun(t, r.Dir, "doc", "std/Term"), "Declares that it links only: std/Term")
}
