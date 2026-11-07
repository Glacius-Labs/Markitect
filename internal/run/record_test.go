package run_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/folder"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// The record of what was verified (DEC-099), on plain folders without Git: the
// test plays the builder and the reviewer, and status says what is to verify.

// plain is a project in a folder with no Git, and its engine.
type plain struct {
	t      *testing.T
	dir    string
	engine *run.Engine
}

func newPlain(t *testing.T) *plain {
	t.Helper()
	return newPlainOf(t, library())
}

func newPlainOf(t *testing.T, files map[string]string) *plain {
	t.Helper()
	p := &plain{t: t, dir: testkit.TempDir(t)}
	for file, content := range files {
		write(t, p.dir, file, content)
	}
	p.engine = &run.Engine{Folder: folder.Disk{}, Cache: testkit.TempDir(t)}
	return p
}

func (p *plain) write(file, content string) { write(p.t, p.dir, file, content) }

func (p *plain) remove(file string) {
	p.t.Helper()
	if err := os.Remove(filepath.Join(p.dir, filepath.FromSlash(file))); err != nil {
		p.t.Fatal(err)
	}
}

func (p *plain) read(file string) string {
	p.t.Helper()
	data, err := os.ReadFile(filepath.Join(p.dir, filepath.FromSlash(file)))
	if err != nil {
		p.t.Fatal(err)
	}
	return string(data)
}

func (p *plain) status(all bool) run.Status {
	p.t.Helper()
	st, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, All: all})
	if err != nil {
		p.t.Fatal(err)
	}
	return st
}

// item finds a Definition among what status lists.
func (p *plain) item(st run.Status, id string) run.Item {
	p.t.Helper()
	for _, it := range st.Items {
		if it.Definition == id {
			return it
		}
	}
	p.t.Fatalf("%s is not listed:\n%s", id, st.Text())
	return run.Item{}
}

func listed(st run.Status) string {
	var ids []string
	for _, it := range st.Items {
		ids = append(ids, it.Definition)
	}
	return strings.Join(ids, ",")
}

// review marks a verdict as the reviewer does, with markitect mark.
func (p *plain) review(id, verdict, at string, evidence []string, why string) (string, bool) {
	p.t.Helper()
	text, ok, err := p.engine.Mark(context.Background(), run.MarkRequest{Dir: p.dir, Item: id + "@" + at, Verdict: verdict, Evidence: evidence, Why: why})
	if err != nil {
		p.t.Fatalf("mark %s %s: %v", id, verdict, err)
	}
	return text, ok
}

// verifyAll records every listed item ok, with the first file to look at as
// evidence, the way a reviewer does, and checks that status is then clean.
func (p *plain) verifyAll() {
	p.t.Helper()
	st := p.status(false)
	for _, it := range st.Items {
		var evidence []string
		if it.State != run.StateGone {
			evidence = []string{it.Look[len(it.Look)-1] + ":1"}
		}
		if text, ok := p.review(it.Definition, run.VerdictOK, it.Fingerprint, evidence, ""); !ok {
			p.t.Fatalf("not recorded: %s", text)
		}
	}
	if st := p.status(false); !st.Clean() {
		p.t.Fatalf("not clean after verifying everything:\n%s", st.Text())
	}
}

// With no record, every Definition is to verify; once each is recorded ok, status
// is empty, and the record holds one sorted line per Definition with full hashes.
func TestStatusEmptiesAsTheReviewerRecords(t *testing.T) {
	p := newPlain(t)
	st := p.status(false)
	if listed(st) != "catalog/book,loan-limit,members/loans/loan,members/member,project" || st.Clean() {
		t.Fatalf("first status:\n%s", st.Text())
	}
	for _, it := range st.Items {
		if it.State != run.StateNew || len(it.Fingerprint) != 6 {
			t.Fatalf("%+v", it)
		}
	}
	has(t, st.Text(), "To verify (5 of 5 Definitions)", "members/loans/loan@"+p.item(st, "members/loans/loan").Fingerprint+"\n", "no record yet", "look at: .markitect/model/members/loans/loan.md, src/members/loans/loan.go")
	p.verifyAll()
	has(t, p.status(false).Text(), "Nothing to verify: the spec compiles, and all 5 Definitions are verified at their current inputs.")
	rec := p.read(project.ReviewFile)
	lines := strings.Split(strings.TrimSuffix(rec, "\n"), "\n")
	if len(lines) != 5 || !strings.HasPrefix(lines[0], `{"definition":"catalog/book","verdict":"ok","inputs":{`) {
		t.Fatalf("record:\n%s", rec)
	}
	has(t, rec, `"file:src/members/loans/loan.go":"sha256:`, `"kind:std/Rule":"sha256:`, `"link:members/member":"sha256:`, `"evidence":["src/members/loans/loan.go:1"]`)
	parsed := run.ParseRecord([]byte(rec))
	if len(parsed.Unreadable) != 0 || len(parsed.Lines[0].Inputs[run.InputSelf]) != len("sha256:")+64 {
		t.Fatalf("%+v", parsed)
	}
	// --all lists every Definition again, and so is never clean while there is one.
	if all := p.status(true); len(all.Items) != 5 || all.Clean() || all.Items[0].State != run.StateOK {
		t.Fatalf("--all:\n%s", all.Text())
	}
}

// status counts its items per Package, with each Package's Definitions and the
// Packages it depends on, so that a driver can plan work per Package in the order
// of their dependencies (DEC-102). A spec-problem is counted apart, since it waits
// for the owner.
func TestStatusCountsItemsPerPackage(t *testing.T) {
	p := newPlain(t)
	st := p.status(false)
	got := map[string]run.PackageItems{}
	for _, pk := range st.Packages {
		got[pk.Package] = pk
	}
	if len(st.Packages) != 4 || st.Packages[0].Package != "Root" {
		t.Fatalf("Packages: %+v", st.Packages)
	}
	for name, want := range map[string][2]int{"Root": {2, 2}, "catalog": {1, 1}, "members": {1, 1}, "members/loans": {1, 1}} {
		if got[name].Definitions != want[0] || got[name].Items != want[1] || got[name].SpecProblems != 0 {
			t.Errorf("%s: %+v", name, got[name])
		}
	}
	// A loan links to a member: a link to an ancestor, which list --deps leaves
	// out, still orders the work.
	if deps := got["members/loans"].DependsOn; len(deps) != 1 || deps[0] != "members" {
		t.Errorf("members/loans depends on %v", deps)
	}
	it := p.item(st, "members/loans/loan")
	if text, ok := p.review(it.Definition, run.VerdictSpecProblem, it.Fingerprint, nil, "which limit?"); !ok {
		t.Fatal(text)
	}
	for _, pk := range p.status(false).Packages {
		if pk.Package == "members/loans" && (pk.Items != 1 || pk.SpecProblems != 1) {
			t.Fatalf("after a spec-problem: %+v", pk)
		}
	}
	if broken := newPlainOf(t, map[string]string{project.ManifestFile: "libraries:\n  std: 1.0.0\n", project.ModelDir + "/x.md": "no front matter\n"}).status(false); len(broken.Packages) != 0 {
		t.Fatalf("a spec with errors has Packages: %+v", broken.Packages)
	}
}

// What is to verify is what concept 4.0's checklist reached at distance 0 and 1.
func TestWhatAChangeReaches(t *testing.T) {
	cases := []struct {
		name   string
		change func(p *plain)
		want   string
	}{
		{"a changed Definition and what links to it, one step", func(p *plain) {
			p.write(m+"members/member.md", def("std/Concept", "Member", "A person with a card."))
		}, "members/loans/loan,members/member"},
		{"a changed Rule above a Package reaches only itself and what links to it", func(p *plain) {
			p.write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
		}, "loan-limit"},
		{"a changed file reaches what its row names", func(p *plain) {
			p.write("src/members/loans/loan.go", limit3)
		}, "loan-limit,members/loans/loan"},
		{"a row that gains a Definition reaches it", func(p *plain) {
			p.write(project.RegistryDir+"/members.md", "---\npaths: [src/members/**]\n---\n"+table+"| src/members/member.go | [[members/member]], [[loan-limit]] | Member |\n")
		}, "loan-limit"},
		{"a row that drops a Definition reaches it, though the file did not change", func(p *plain) {
			p.write(project.RegistryDir+"/members/loans.md", "---\npaths: [src/members/loans/**]\n---\n"+table+"| src/members/loans/loan.go | [[members/loans/loan]] | Loans |\n")
		}, "loan-limit"},
		{"a changed row reason reaches nothing", func(p *plain) {
			p.write(project.RootPage, table+"| README.md | [[project]] | The overview |\n")
		}, ""},
		{"a file the ignore file lists reaches nothing", func(p *plain) {
			p.write(project.IgnoreFile, "notes.txt\n")
			p.write("notes.txt", "x\n")
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newPlain(t)
			p.verifyAll()
			c.change(p)
			st := p.status(false)
			if got := listed(st); got != c.want {
				t.Fatalf("listed %q, want %q:\n%s", got, c.want, st.Text())
			}
			for _, it := range st.Items {
				if it.State != run.StateChanged || len(it.Changes) == 0 {
					t.Fatalf("%+v", it)
				}
			}
		})
	}
}

// A changed Kind reaches the Definitions of that Kind, and says so.
func TestAChangedKindReachesItsDefinitions(t *testing.T) {
	files := library()
	files[m+"catalog/Medium.md"] = def("Kind", "Medium", "Something that can be lent.")
	files[m+"catalog/dvd.md"] = def("catalog/Medium", "DVD", "A disc.")
	files[project.RegistryDir+"/catalog.md"] += "| src/catalog/dvd.go | [[catalog/dvd]] | DVD |\n"
	files["src/catalog/dvd.go"] = "package catalog\n"
	p := newPlainOf(t, files)
	p.verifyAll()
	p.write(m+"catalog/Medium.md", def("Kind", "Medium", "Something that can be lent, with a loan period."))
	st := p.status(false)
	if listed(st) != "catalog/Medium,catalog/dvd" {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), "changed: kind:catalog/Medium\n", "changed: self")
}

// A Library counts as one, by its digest in the lock, so that a newer binary
// changes nothing until the lock is written again: then every Definition of one
// of its Kinds is to verify.
func TestALibraryCountsByItsLockDigest(t *testing.T) {
	files := library()
	files[project.ManifestFile] = "libraries:\n  std: 1.0.0\n  web: 0.1.0\n"
	files[project.LibraryDir+"/web/Page.md"] = "---\nkind: Kind\n---\n# Page\n"
	files[project.LibraryDir+"/web/Form.md"] = "---\nkind: Kind\n---\n# Form\n"
	files[m+"catalog/search.md"] = def("web/Form", "Search", "Finds a [[catalog/book]].")
	files[project.RegistryDir+"/catalog.md"] += "| src/catalog/search.go | [[catalog/search]] | Search form |\n"
	files["src/catalog/search.go"] = "package catalog\n"
	p := newPlainOf(t, files)
	relock := func() {
		pr, err := project.Load(mustOpen(t, p.dir))
		if err != nil {
			t.Fatal(err)
		}
		p.write(project.LockFile, string(model.RenderLock(pr.Libraries)))
	}
	relock()
	p.verifyAll()
	p.write(project.LibraryDir+"/web/Page.md", "---\nkind: Kind\n---\n# Page\n\nNow with a title bar.\n")
	relock()
	st := p.status(false)
	if listed(st) != "catalog/search" {
		t.Fatalf("a Library counts as one by its digest:\n%s", st.Text())
	}
	has(t, st.Text(), "changed: kind:web/Form\n")
}

const limit3 = "package loans\n\nconst Limit = 3\n"

func mustOpen(t *testing.T, dir string) project.Repository {
	t.Helper()
	files, err := folder.Disk{}.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// mark binds a verdict to the state the reviewer read: a fingerprint whose
// inputs changed since is refused, and so is evidence that does not exist.
func TestMarkRefusesAStaleFingerprintAndMissingEvidence(t *testing.T) {
	p := newPlain(t)
	at := p.item(p.status(false), "members/loans/loan").Fingerprint
	p.write("src/members/loans/loan.go", limit3)
	text, ok := p.review("members/loans/loan", run.VerdictOK, at, []string{"src/members/loans/loan.go:3"}, "")
	if ok {
		t.Fatal("a stale fingerprint was taken")
	}
	has(t, text, "Run markitect status again", "Nothing is recorded.")
	at = p.item(p.status(false), "members/loans/loan").Fingerprint
	text, ok = p.review("members/loans/loan", run.VerdictOK, at, []string{"src/members/loans/loan.go:3", "tests/loan_test.go:9 (TestLimit)"}, "")
	if ok {
		t.Fatal("evidence that does not exist was taken")
	}
	has(t, text, "tests/loan_test.go")
	text, ok = p.review("members/gone", run.VerdictOK, at, []string{"README.md"}, "")
	if ok {
		t.Fatal("an unknown Definition was recorded")
	}
	has(t, text, "no Definition of the spec")
	if _, err := os.Stat(filepath.Join(p.dir, project.ReviewFile)); !os.IsNotExist(err) {
		t.Fatalf("a refusal wrote the record: %v", err)
	}
	member := "members/member@" + p.item(p.status(false), "members/member").Fingerprint
	for _, req := range []run.MarkRequest{
		{Item: member, Verdict: "wrong"},
		{Item: member, Verdict: run.VerdictFail},
		{Item: "members/member", Verdict: run.VerdictOK, Evidence: []string{"README.md"}},
		{Item: "members/member@" + at[:4], Verdict: run.VerdictOK, Evidence: []string{"README.md"}},
		{Item: member, Verdict: run.VerdictOK},
		{Item: member, Verdict: run.VerdictOK, Evidence: []string{"/etc/passwd"}},
	} {
		req.Dir = p.dir
		if _, _, err := p.engine.Mark(context.Background(), req); err == nil || !strings.Contains(err.Error(), "usage") {
			t.Fatalf("%+v: %v", req, err)
		}
	}
	// A longer prefix of the fingerprint is taken too, in either case.
	full := ""
	for _, line := range strings.Split(p.status(false).Text(), "\n") {
		if strings.HasPrefix(line, "members/loans/loan@") {
			full = strings.TrimPrefix(line, "members/loans/loan@")
		}
	}
	if full != at {
		t.Fatalf("status prints %q, the item %q", full, at)
	}
	text, ok = p.review("[[members/loans/loan]]", run.VerdictOK, strings.ToUpper(at), []string{`src\members\loans\loan.go:3`}, "")
	if !ok {
		t.Fatalf("%s", text)
	}
	has(t, p.read(project.ReviewFile), `"evidence":["src/members/loans/loan.go:3"]`)
}

// The evidence is an input: a changed test that served as evidence puts its
// Definition back on the list.
func TestChangedEvidenceIsToVerify(t *testing.T) {
	p := newPlain(t)
	p.write(project.RegistryDir+"/members.md", "---\npaths: [src/members/**]\n---\n"+table+"| src/members/member.go | [[members/member]] | Member |\n| src/members/member_test.go | [[members/member]] | Test |\n")
	p.write("src/members/member_test.go", "package members\n")
	p.write("README.md", "# Library\n\nThe loan limit is 5.\n")
	p.verifyAll()
	at := p.item(p.status(true), "loan-limit").Fingerprint
	if _, ok := p.review("loan-limit", run.VerdictOK, at, []string{"README.md:3 (the overview says so)"}, ""); !ok {
		t.Fatal("not recorded")
	}
	p.write("README.md", "# Library\n\nThe loan limit is 6.\n")
	st := p.status(false)
	if listed(st) != "loan-limit,project" {
		t.Fatalf("%s", st.Text())
	}
	has(t, p.item(st, "loan-limit").Changes[0].String(), "evidence:README.md")
	p.remove("README.md")
	has(t, p.status(false).Text(), "evidence:README.md (gone)")
}

// fail and spec-problem stay on the list with their reason; skip is a note, and a
// note alone leaves status clean.
func TestVerdictsOnTheList(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	st := p.status(true)
	if _, ok := p.review("loan-limit", run.VerdictFail, p.item(st, "loan-limit").Fingerprint, []string{"src/members/loans/loan.go:3"}, "allows a sixth\nloan"); !ok {
		t.Fatal("fail not recorded")
	}
	if _, ok := p.review("members/member", run.VerdictSpecProblem, p.item(st, "members/member").Fingerprint, nil, "is a member a person or a household?"); !ok {
		t.Fatal("spec-problem not recorded")
	}
	if _, ok := p.review("catalog/book", run.VerdictSkip, p.item(st, "catalog/book").Fingerprint, nil, "no test can reach the printer"); !ok {
		t.Fatal("skip not recorded")
	}
	st = p.status(false)
	if listed(st) != "loan-limit,members/member" || st.Clean() {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), "  fail: allows a sixth loan\n", "  spec-problem: is a member a person or a household?\n", "Notes: skipped", "catalog/book: no test can reach the printer")
	// A changed input of a failed Definition says what was recorded before.
	p.write("src/members/loans/loan.go", limit3)
	has(t, p.status(false).Text(), "recorded before: fail: allows a sixth loan")
	// Reviewed ok, the skip leaves only the note's absence behind.
	p.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 5\n")
	for _, id := range []string{"loan-limit", "members/member"} {
		if _, ok := p.review(id, run.VerdictOK, p.item(p.status(false), id).Fingerprint, []string{"README.md"}, ""); !ok {
			t.Fatal("not recorded")
		}
	}
	if st := p.status(false); !st.Clean() || len(st.Skipped) != 1 {
		t.Fatalf("%s", st.Text())
	}
}

// A Definition gone from the spec stays on the list with the files that realized
// it, until a reviewer finds them dealt with: then its line goes.
func TestAGoneDefinition(t *testing.T) {
	files := library()
	files[m+"catalog/shelf.md"] = def("std/Concept", "Shelf", "Holds a [[catalog/book]].")
	files[project.RegistryDir+"/catalog.md"] += "| src/catalog/shelf.go | [[catalog/shelf]] | Shelf |\n"
	files["src/catalog/shelf.go"] = "package catalog\n"
	p := newPlainOf(t, files)
	p.verifyAll()
	p.remove(m + "catalog/shelf.md")
	st := p.status(false)
	gone := p.item(st, "catalog/shelf")
	if gone.State != run.StateGone || strings.Join(gone.Look, ",") != "src/catalog/shelf.go" {
		t.Fatalf("%+v", gone)
	}
	has(t, st.Text(), "gone from the spec", "Errors in the realization", "stale.definition")
	if text, ok := p.review("catalog/shelf", run.VerdictSkip, gone.Fingerprint, nil, "later"); ok {
		t.Fatalf("a gone Definition was skipped: %s", text)
	}
	if _, ok := p.review("catalog/shelf", run.VerdictFail, gone.Fingerprint, nil, "src/catalog/shelf.go is still there"); !ok {
		t.Fatal("not recorded")
	}
	has(t, p.status(false).Text(), "recorded: fail: src/catalog/shelf.go is still there")
	p.remove("src/catalog/shelf.go")
	p.write(project.RegistryDir+"/catalog.md", "---\npaths: [src/catalog/**]\n---\n"+table+"| src/catalog/book.go | [[catalog/book]] | Book |\n")
	st = p.status(false)
	if listed(st) != "catalog/shelf" || len(st.Errors) != 0 {
		t.Fatalf("%s", st.Text())
	}
	text, ok := p.review("catalog/shelf", run.VerdictOK, p.item(st, "catalog/shelf").Fingerprint, nil, "")
	if !ok || !strings.Contains(text, "its line is removed") {
		t.Fatalf("%s", text)
	}
	if st := p.status(false); !st.Clean() || strings.Contains(p.read(project.ReviewFile), "catalog/shelf") {
		t.Fatalf("%s", st.Text())
	}
}

// After a merge, Git keeps the lines of both sides. status takes the one that
// matches the folder, lists two that match with different verdicts as a
// conflict, and the next review writes the file clean.
func TestMergedLines(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	before := p.read(project.ReviewFile)
	p.write("src/members/loans/loan.go", limit3)
	for _, id := range []string{"loan-limit", "members/loans/loan"} {
		if _, ok := p.review(id, run.VerdictOK, p.item(p.status(false), id).Fingerprint, []string{"src/members/loans/loan.go:3"}, ""); !ok {
			t.Fatal("not recorded")
		}
	}
	after := p.read(project.ReviewFile)
	p.write(project.ReviewFile, before+after)
	if st := p.status(false); !st.Clean() {
		t.Fatalf("the matching line does not count:\n%s", st.Text())
	}
	// Two matching lines with different verdicts are a conflict.
	var failed []string
	for _, line := range strings.Split(after, "\n") {
		if strings.Contains(line, `"definition":"loan-limit"`) {
			r := strings.Replace(line, `"verdict":"ok"`, `"verdict":"fail"`, 1)
			failed = append(failed, r[:len(r)-1]+`,"why":"a sixth loan"}`)
		}
	}
	p.write(project.ReviewFile, before+after+strings.Join(failed, "\n")+"\n")
	st := p.status(false)
	if listed(st) != "loan-limit" || p.item(st, "loan-limit").State != run.StateConflict {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), "conflict: lines with ok and fail match")
	if _, ok := p.review("loan-limit", run.VerdictOK, p.item(st, "loan-limit").Fingerprint, []string{"README.md"}, ""); !ok {
		t.Fatal("not recorded")
	}
	rec := p.read(project.ReviewFile)
	if n := len(strings.Split(strings.TrimSuffix(rec, "\n"), "\n")); n != 5 {
		t.Fatalf("not written clean, %d lines:\n%s", n, rec)
	}
	if !p.status(false).Clean() {
		t.Fatal("not clean")
	}
}

// A line markitect cannot read is an error status names, and review records
// nothing over it.
func TestAnUnreadableLine(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	p.write(project.ReviewFile, p.read(project.ReviewFile)+"{\"definition\": \"x\", \"verdict\": \"maybe\", \"inputs\": {}}\nnot json\n")
	st := p.status(false)
	if st.Clean() || len(st.Unreadable) != 2 {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), ".markitect/review.sum:6: record.unreadable: ", ".markitect/review.sum:7: record.unreadable: ")
	at := p.item(p.status(true), "loan-limit").Fingerprint
	if text, ok := p.review("loan-limit", run.VerdictOK, at, []string{"README.md"}, ""); ok || !strings.Contains(text, "cannot read every line") {
		t.Fatalf("%s", text)
	}
}

// An error in the spec stops the list and every verdict; an error in the
// realization is listed for whoever builds.
func TestErrors(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	p.write("src/catalog/new.go", "package catalog\n")
	st := p.status(false)
	if st.Clean() || len(st.Errors) != 1 || st.Errors[0].Code != project.CodeUnregistered || listed(st) != "" {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), "Errors in the realization (1)", "src/catalog/new.go")
	at := p.item(p.status(true), "catalog/book").Fingerprint
	p.write(m+"catalog/book.md", def("std/Concept", "Book", "A book of [[catalog/nothing]]."))
	st = p.status(false)
	if len(st.SpecErrors) == 0 || len(st.Items) != 0 {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), "there is no list until it compiles")
	if text, ok := p.review("catalog/book", run.VerdictOK, at, []string{"README.md"}, ""); ok || !strings.Contains(text, "The spec has errors") {
		t.Fatalf("%s", text)
	}
}

// Reviewers in parallel, such as one per Package, lose no line: each writes
// under the lock, as one whole file.
func TestParallelReviewersLoseNoLine(t *testing.T) {
	p := newPlain(t)
	st := p.status(false)
	var wg sync.WaitGroup
	errs := make(chan string, len(st.Items))
	for _, it := range st.Items {
		wg.Add(1)
		go func(it run.Item) {
			defer wg.Done()
			text, ok, err := p.engine.Mark(context.Background(), run.MarkRequest{Dir: p.dir, Item: it.Definition + "@" + it.Fingerprint, Verdict: run.VerdictOK, Evidence: []string{"README.md"}})
			if err != nil || !ok {
				errs <- it.Definition + ": " + text
				if err != nil {
					errs <- err.Error()
				}
			}
		}(it)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	if st := p.status(false); !st.Clean() {
		t.Fatalf("%s", st.Text())
	}
}

// The record joins the fixed layout of .markitect, and so does its Git
// attributes file.
func TestTheRecordIsPartOfTheLayout(t *testing.T) {
	p := newPlain(t)
	p.write(project.AttributesFile, project.Attributes)
	p.verifyAll()
	pr, err := project.Load(mustOpen(t, p.dir))
	if err != nil {
		t.Fatal(err)
	}
	if len(pr.Errors) != 0 {
		t.Fatalf("%v", pr.Errors)
	}
}
