package run_test

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// What binds is an input of what it binds (DEC-113): a Definition of a Kind
// that declares binds reaches every unit a file realizes in its Package and
// below, and status names the binding Definition that changed.

// changes is what an item's changes say: each change as status prints it.
func changes(it run.Item) string {
	var out []string
	for _, c := range it.Changes {
		out = append(out, c.String())
	}
	return strings.Join(out, ", ")
}

// A changed Rule of Root lists every unit a file realizes, each with the Rule
// that changed, and the Rule itself with its own text; the Project is
// realized by no file, so nothing binds it. Once the Rule is verified, what it
// binds still names it, and a verdict at a fingerprint from before the change
// is refused.
func TestAChangedRuleReachesWhatItBinds(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	p.write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
	st := p.status(false)
	const bound = "catalog/book,catalog/catalog,members/loans/loan,members/loans/loans,members/member,members/members"
	if got := listed(st); got != "catalog/book,catalog/catalog,loan-limit,members/loans/loan,members/loans/loans,members/member,members/members" {
		t.Fatalf("after a change of the Rule of Root, listed %q:\n%s", got, st.Text())
	}
	for _, it := range st.Items {
		want := "bound:loan-limit"
		if it.Definition == "loan-limit" {
			want = "self"
		}
		if got := changes(it); it.State != run.StateChanged || got != want {
			t.Fatalf("%s changed: %s, want %s", it.Definition, got, want)
		}
	}
	has(t, st.Text(), "members/member@"+p.item(st, "members/member").Fingerprint+"\n  changed: bound:loan-limit\n", "loan-limit@"+p.item(st, "loan-limit").Fingerprint+"\n  changed: self\n")

	// The Rule is verified; what it binds still names it.
	before := p.item(st, "members/member").Fingerprint
	rule := p.item(st, "loan-limit")
	if text, ok := p.review(rule.Definition, run.VerdictOK, rule.Fingerprint, []string{"src/members/loans/loan.go:3"}, ""); !ok {
		t.Fatal(text)
	}
	st = p.status(false)
	if got := listed(st); got != bound {
		t.Fatalf("after the Rule is verified, listed %q:\n%s", got, st.Text())
	}
	for _, it := range st.Items {
		if got := changes(it); got != "bound:loan-limit" {
			t.Fatalf("%s changed: %s", it.Definition, got)
		}
	}

	// A verdict at the fingerprint status printed records; one at a
	// fingerprint from before another change of the Rule is refused.
	member := p.item(st, "members/member")
	if text, ok := p.review(member.Definition, run.VerdictOK, member.Fingerprint, []string{"src/members/member.go:1"}, ""); !ok {
		t.Fatal(text)
	}
	at := p.item(st, "members/loans/loan").Fingerprint
	p.write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `4` open loans."))
	for _, old := range []string{at, before} {
		text, ok := p.review("members/loans/loan", run.VerdictOK, old, []string{"src/members/loans/loan.go:3"}, "")
		if ok {
			t.Fatal("a fingerprint from before the Rule changed was taken")
		}
		has(t, text, "Run markitect status again")
	}
	p.verifyAll()
	if rec := p.read(project.ReviewFile); strings.Count(rec, `{"bound":`) != 1 {
		t.Fatalf("the record keeps a directory no line names:\n%s", rec)
	}
}

// A binding thing in a Package below Root reaches that Package and the
// Packages below it, not its neighbours and not Root.
func TestARuleBelowRootBindsItsPackage(t *testing.T) {
	files := library()
	files[m+"members/rules/cards.md"] = def("std/Rule", "Cards", "Every member carries a card.")
	files[project.RegistryDir+"/members.md"] += "| src/members/cards.go | [[members/rules/cards]] | Cards |\n"
	files["src/members/cards.go"] = "package members\n"
	p := newPlainOf(t, files)
	p.verifyAll()
	// Every unit of members and below carries the directory of members and
	// that of Root, the Rule itself included; catalog only Root's.
	for _, l := range run.ParseRecord([]byte(p.read(project.ReviewFile))).Lines {
		_, root := l.Inputs["bound:Root"]
		_, members := l.Inputs["bound:members"]
		inMembers := strings.HasPrefix(l.Definition, "members/")
		if root != (l.Definition != "project") || members != inMembers {
			t.Errorf("%s: bound:Root %v, bound:members %v", l.Definition, root, members)
		}
	}
	p.write(m+"members/rules/cards.md", def("std/Rule", "Cards", "Every member carries a card with a photo."))
	st := p.status(false)
	if got := listed(st); got != "members/loans/loan,members/loans/loans,members/member,members/members,members/rules/cards" {
		t.Fatalf("after a change of the Rule of members, listed %q:\n%s", got, st.Text())
	}
	for _, it := range st.Items {
		if got := changes(it); it.Definition != "members/rules/cards" && got != "bound:members/rules/cards" {
			t.Fatalf("%s changed: %s", it.Definition, got)
		}
	}
}

// A binding thing is hashed with its parts: a changed part reaches what it
// binds. A binding Definition that is added is named (new), one that is
// removed (gone), and a Package that gets its first binding Definition is
// named (new) itself.
func TestWhatBindsIsNamedByWhatChanged(t *testing.T) {
	files := library()
	files[m+"rules/fees/fees.md"] = def("std/Rule", "Fees", "A fee per day late.")
	files[m+"rules/fees/per-day.md"] = def("std/Rule", "Per day", "`1` euro.")
	files[project.RootPage] += "| src/fees.go | [[rules/fees]] | Fees |\n"
	files["src/fees.go"] = "package src\n"
	p := newPlainOf(t, files)
	p.verifyAll()
	realized := "catalog/book,catalog/catalog,loan-limit,members/loans/loan,members/loans/loans,members/member,members/members"
	expect := func(what, list string, named map[string]string) {
		t.Helper()
		st := p.status(false)
		if got := listed(st); got != list {
			t.Fatalf("%s: listed %q, want %q:\n%s", what, got, list, st.Text())
		}
		for _, it := range st.Items {
			want, ok := named[it.Definition]
			if !ok {
				want = named[""]
			}
			if got := changes(it); got != want {
				t.Fatalf("%s: %s changed: %s, want %s", what, it.Definition, got, want)
			}
		}
	}

	p.write(m+"rules/fees/per-day.md", def("std/Rule", "Per day", "`2` euros."))
	expect("a changed part", realized+",rules/fees/fees", map[string]string{"": "bound:rules/fees/fees", "rules/fees/fees": "part:rules/fees/per-day"})
	p.verifyAll()

	p.write(m+"rules/quiet.md", def("std/Rule", "Quiet", "Pages load without sound."))
	p.write(project.RootPage, p.read(project.RootPage)+"| src/quiet.go | [[rules/quiet]] | Quiet |\n")
	p.write("src/quiet.go", "package src\n")
	st := p.status(false)
	if it := p.item(st, "rules/quiet"); it.State != run.StateNew {
		t.Fatalf("a new Rule: %+v", it)
	}
	for _, it := range st.Items {
		if got := changes(it); it.Definition != "rules/quiet" && got != "bound:rules/quiet (new)" {
			t.Fatalf("a new Rule: %s changed: %s", it.Definition, got)
		}
	}
	has(t, st.Text(), "changed: bound:rules/quiet (new)\n")
	p.verifyAll()

	p.remove(m + "rules/quiet.md")
	p.write(project.RootPage, strings.Replace(p.read(project.RootPage), "| src/quiet.go | [[rules/quiet]] | Quiet |\n", "", 1))
	p.remove("src/quiet.go")
	st = p.status(false)
	if it := p.item(st, "rules/quiet"); it.State != run.StateGone {
		t.Fatalf("a removed Rule: %+v", it)
	}
	for _, it := range st.Items {
		if got := changes(it); it.Definition != "rules/quiet" && got != "bound:rules/quiet (gone)" {
			t.Fatalf("a removed Rule: %s changed: %s", it.Definition, got)
		}
	}
	p.verifyAll()

	// catalog had nothing that binds it until now.
	p.write(m+"catalog/rules/shelves.md", def("std/Rule", "Shelves", "Every book has a shelf."))
	p.write(project.RegistryDir+"/catalog.md", p.read(project.RegistryDir+"/catalog.md")+"| src/catalog/shelves.go | [[catalog/rules/shelves]] | Shelves |\n")
	p.write("src/catalog/shelves.go", "package catalog\n")
	st = p.status(false)
	if got := listed(st); got != "catalog/book,catalog/catalog,catalog/rules/shelves" {
		t.Fatalf("a first Rule of catalog, listed %q:\n%s", got, st.Text())
	}
	for _, id := range []string{"catalog/book", "catalog/catalog"} {
		if got := changes(p.item(st, id)); got != "bound:catalog (new)" {
			t.Fatalf("a first Rule of catalog: %s changed: %s", id, got)
		}
	}
}

// A record written before what binds was an input has no bound: input: each
// unit that something binds is listed once with the Package that binds it,
// and verified again it is settled.
func TestARecordWithoutBoundInputs(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	rec := run.ParseRecord([]byte(p.read(project.ReviewFile)))
	for _, l := range rec.Lines {
		delete(l.Inputs, "bound:Root")
	}
	p.write(project.ReviewFile, string(run.Record{Lines: rec.Lines}.Render()))
	if got := p.read(project.ReviewFile); strings.Contains(got, "bound") {
		t.Fatalf("the record of an earlier markitect:\n%s", got)
	}
	st := p.status(false)
	if got := listed(st); got != "catalog/book,catalog/catalog,loan-limit,members/loans/loan,members/loans/loans,members/member,members/members" {
		t.Fatalf("listed %q:\n%s", got, st.Text())
	}
	for _, it := range st.Items {
		if got := changes(it); got != "bound:Root (new)" {
			t.Fatalf("%s changed: %s", it.Definition, got)
		}
	}
	p.verifyAll()
}

// Where the record no longer holds the directory a line names, status names
// the Package, as for any other input.
func TestADirectoryTheRecordLost(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	var kept []string
	for _, l := range strings.SplitAfter(p.read(project.ReviewFile), "\n") {
		if !strings.HasPrefix(l, `{"bound":`) {
			kept = append(kept, l)
		}
	}
	p.write(project.ReviewFile, strings.Join(kept, ""))
	p.write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
	st := p.status(false)
	if got := changes(p.item(st, "members/member")); got != "bound:Root" {
		t.Fatalf("without its directory, member changed: %s", got)
	}
	if got := changes(p.item(st, "loan-limit")); got != "bound:Root, self" {
		t.Fatalf("without its directory, the Rule changed: %s", got)
	}
}

// The record keeps one directory per Package and digest beside the lines:
// mark writes the one the new line names, every write drops one that no line
// names, a union of two records keeps both sides', a directory written by
// hand that does not match its digest cannot be read, and the file is
// written the same whatever the order it was given in.
func TestTheRecordKeepsTheDirectories(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	text := p.read(project.ReviewFile)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != 9 || !strings.HasPrefix(lines[8], `{"bound":"Root","digest":"sha256:`) || !strings.Contains(lines[8], `,"binds":{"loan-limit":"sha256:`) {
		t.Fatalf("the record with its directory:\n%s", text)
	}
	rec := run.ParseRecord([]byte(text))
	if len(rec.Lines) != 8 || len(rec.Directories) != 1 || len(rec.Unreadable) != 0 {
		t.Fatalf("read back: %+v", rec)
	}
	dir := rec.Directories[0]
	for _, l := range rec.Lines {
		if got, want := l.Inputs["bound:Root"], dir.Digest; l.Definition != "project" && got != want {
			t.Fatalf("%s names %s, the directory is %s", l.Definition, got, want)
		}
	}
	if len(dir.Binds) != 1 || dir.Bound != "Root" {
		t.Fatalf("directory: %+v", dir)
	}

	// The same lines in another order, a line twice and a directory no line
	// names: the same file.
	reversed := run.Record{Directories: append([]run.Directory{{Bound: "catalog", Digest: "sha256:0", Binds: map[string]string{"x": "sha256:1"}}}, rec.Directories...)}
	for i := len(rec.Lines) - 1; i >= 0; i-- {
		reversed.Lines = append(reversed.Lines, rec.Lines[i])
	}
	reversed.Lines = append(reversed.Lines, rec.Lines[3])
	reversed.Directories = append(reversed.Directories, rec.Directories...)
	if got := string(reversed.Render()); got != text {
		t.Fatalf("written again:\n%s\nwant:\n%s", got, text)
	}

	// Two sides of a merge, each with its own Rule: Git keeps both sides'
	// lines and directories, and status takes the lines that match.
	p.write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
	p.verifyAll()
	after := p.read(project.ReviewFile)
	p.write(project.ReviewFile, text+after)
	if got := run.ParseRecord([]byte(p.read(project.ReviewFile))); len(got.Directories) != 2 || len(got.Unreadable) != 0 {
		t.Fatalf("the union: %+v", got)
	}
	if st := p.status(false); !st.Clean() {
		t.Fatalf("the union at the new Rule:\n%s", st.Text())
	}
	p.write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `5` open loans."))
	if st := p.status(false); !st.Clean() {
		t.Fatalf("the union at the old Rule:\n%s", st.Text())
	}
	// The next verdict writes the file clean: the lines that match and the
	// directory they name.
	p.write("src/catalog/book.go", "package catalog\n\n// A book.\n")
	p.verifyAll()
	if got := p.read(project.ReviewFile); strings.Count(got, "\n") != 9 || strings.Count(got, `{"bound":`) != 1 || !strings.Contains(got, p.lineOf(text, `{"bound":"Root"`)+"\n") {
		t.Fatalf("written clean:\n%s", got)
	}

	// A directory that does not match its digest is a line markitect cannot
	// read, and so is anything else, each with its number.
	rec = run.ParseRecord([]byte(p.read(project.ReviewFile)))
	forged := strings.Replace(p.lineOf(p.read(project.ReviewFile), `"bound":"Root"`), `"loan-limit":"sha256:`, `"loan-limit":"sha256:0`, 1)
	p.write(project.ReviewFile, p.read(project.ReviewFile)+forged+"\nnot a line\n")
	st := p.status(false)
	if len(st.Unreadable) != 2 || st.Unreadable[0] != 10 || st.Unreadable[1] != 11 {
		t.Fatalf("unreadable: %v\n%s", st.Unreadable, st.Text())
	}
	has(t, st.Text(), ".markitect/review.sum:10: record.unreadable: ", ".markitect/review.sum:11: record.unreadable: ")
}

// lineOf returns the line of a record that holds a text.
func (p *plain) lineOf(record, holds string) string {
	p.t.Helper()
	for _, l := range strings.Split(record, "\n") {
		if strings.Contains(l, holds) {
			return l
		}
	}
	p.t.Fatalf("no line holds %s:\n%s", holds, record)
	return ""
}
