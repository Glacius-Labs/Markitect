package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"
)

// What binds is an input of what it binds (DEC-113), through the commands: doc
// lists what binds a Definition only where a file realizes it, and a changed
// Rule lists what it binds with the Rule's name, in the text and in status
// --json, also once the Rule itself is verified again.
func TestARuleBindsThroughTheCommands(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Library")
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	const spec = ".markitect/spec/"
	r.Write(spec+"purpose/goals/lend-books.md", def("std/Goal", "Lend books", "**Result:** books go home.\n**Changes:** shelves empty.\n**Obstacle:** paper cards."))
	r.Write(spec+"domain/rules/roles.md", def("std/Rule", "Roles", "`anyone` is any caller."))
	r.Write(spec+"domain/operations/lend.md", def("std/Operation", "Lend", "Lends a copy.\n\n**Who:** `anyone`.\n**Satisfies:** [[purpose/goals/lend-books]]."))
	r.Write(".markitect/registry.md", "# Registry: Root\n\n| File | Realizes | Why |\n|---|---|---|\n| `lend.go` | [[domain/operations/lend]] [[domain/rules/roles]] | Lending |\n")
	r.Write("lend.go", "package library\n")
	if out := mustRun(t, r.Dir, "check"); !strings.HasPrefix(out, "ok:") {
		t.Fatalf("check: %s", out)
	}

	// doc: the Rule binds the Operation, and nothing binds the Goal.
	wantAll(t, mustRun(t, r.Dir, "doc", "domain/operations/lend"), "\nBound by, without a link, as everything in its Package:\n  domain/rules/roles: Roles (std/Rule)\n")
	if out := mustRun(t, r.Dir, "doc", "purpose/goals/lend-books"); strings.Contains(out, "Bound by") {
		t.Fatalf("doc of a Goal:\n%s", out)
	}
	if out := mustRun(t, r.Dir, "doc", "purpose/goals/lend-books", "--json"); strings.Contains(out, "bound_by") {
		t.Fatalf("doc --json of a Goal:\n%s", out)
	}

	verifyAll(t, r.Dir)
	r.Write(spec+"domain/rules/roles.md", def("std/Rule", "Roles", "`anyone` is any caller, `staff` one who signed in."))
	_, out, _ := run(t, r.Dir, "status")
	wantAll(t, out, "To verify (2 of 4 items)", "\ndomain/operations/lend@", "  changed: bound:domain/rules/roles\n", "\ndomain/rules/roles@", "  changed: self\n")
	type item struct {
		Definition, Fingerprint string
		Changes                 []struct{ Key, How string }
	}
	listed := func() map[string]item {
		t.Helper()
		var st struct{ Items []item }
		_, out, _ := run(t, r.Dir, "status", "--json")
		if err := json.Unmarshal([]byte(out), &st); err != nil {
			t.Fatalf("status --json: %v\n%s", err, out)
		}
		items := map[string]item{}
		for _, it := range st.Items {
			items[it.Definition] = it
		}
		return items
	}
	items := listed()
	if lend := items["domain/operations/lend"]; len(items) != 2 || len(lend.Changes) != 1 || lend.Changes[0].Key != "bound:domain/rules/roles" || lend.Changes[0].How != "changed" {
		t.Fatalf("status --json: %+v", items)
	}

	// The Rule is verified; the Operation still names it, and is verified at
	// the fingerprint status printed.
	rule := items["domain/rules/roles"]
	mustRun(t, r.Dir, "mark", rule.Definition+"@"+rule.Fingerprint, "ok", "-e", "lend.go", "--cases", "1", "--covered", "1")
	_, out, _ = run(t, r.Dir, "status")
	wantAll(t, out, "To verify (1 of 4 items)", "  changed: bound:domain/rules/roles\n")
	if again := listed()["domain/operations/lend"]; len(again.Changes) != 1 || again.Changes[0].Key != "bound:domain/rules/roles" {
		t.Fatalf("after the Rule is verified: %+v", again)
	}
	lend := items["domain/operations/lend"]
	mustRun(t, r.Dir, "mark", lend.Definition+"@"+lend.Fingerprint, "ok", "-e", "lend.go", "--cases", "1", "--covered", "1")
	if code, out, _ := run(t, r.Dir, "status"); code != 0 || !strings.Contains(out, "Nothing to verify") {
		t.Fatalf("status at the end: %d\n%s", code, out)
	}
}
