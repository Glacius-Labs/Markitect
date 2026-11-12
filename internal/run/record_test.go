package run_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/folder"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/spec"
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
	req := run.MarkRequest{Dir: p.dir, Item: id + "@" + at, Verdict: verdict, Evidence: evidence, Why: why}
	if verdict == run.VerdictOK || verdict == run.VerdictSkip {
		// The reviewer counts with an ok and a skip; one case, demonstrated.
		one := 1
		req.Cases, req.Covered = &one, &one
	}
	text, ok, err := p.engine.Mark(context.Background(), req)
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
	if listed(st) != "catalog/book,catalog/catalog,loan-limit,members/loans/loan,members/loans/loans,members/member,members/members,project" || st.Clean() {
		t.Fatalf("first status:\n%s", st.Text())
	}
	for _, it := range st.Items {
		if it.State != run.StateNew || len(it.Fingerprint) != 6 {
			t.Fatalf("%+v", it)
		}
	}
	has(t, st.Text(), "To verify (8 of 8 Definitions)", "members/loans/loan@"+p.item(st, "members/loans/loan").Fingerprint+"\n", "no record yet", "look at: .markitect/spec/members/loans/loan.md, src/members/loans/loan.go")
	p.verifyAll()
	has(t, p.status(false).Text(), "Nothing to verify: the spec compiles, and all 8 Definitions are verified at their current inputs.")
	rec := p.read(project.ReviewFile)
	lines := strings.Split(strings.TrimSuffix(rec, "\n"), "\n")
	if len(lines) != 8 || !strings.HasPrefix(lines[0], `{"definition":"catalog/book","verdict":"ok","inputs":{`) {
		t.Fatalf("record:\n%s", rec)
	}
	has(t, rec, `"file:src/members/loans/loan.go":"sha256:`, `"kind:std/Rule":"sha256:`, `"link:members/member":"sha256:`, `"evidence":["src/members/loans/loan.go:1"]`)
	parsed := run.ParseRecord([]byte(rec))
	if len(parsed.Unreadable) != 0 || len(parsed.Lines[0].Inputs[run.InputSelf]) != len("sha256:")+64 {
		t.Fatalf("%+v", parsed)
	}
	// --all lists every Definition again, and so is never clean while there is one.
	if all := p.status(true); len(all.Items) != 8 || all.Clean() || all.Items[0].State != run.StateOK {
		t.Fatalf("--all:\n%s", all.Text())
	}
}

// status counts its items per Package, with each Package's Definitions and the
// Packages it depends on, so that a driver can plan work per Package in the order
// of their dependencies (DEC-103). A spec-problem is counted apart, since it waits
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
	for name, want := range map[string][2]int{"Root": {2, 2}, "catalog": {2, 2}, "members": {2, 2}, "members/loans": {2, 2}} {
		if got[name].Definitions != want[0] || got[name].Items != want[1] || got[name].SpecProblems != 0 {
			t.Errorf("%s: %+v", name, got[name])
		}
	}
	// A loan links to a member, its ancestor: implicit, as list --deps has it.
	for _, pk := range st.Packages {
		if len(pk.DependsOn) != 0 {
			t.Errorf("%s depends on %v", pk.Package, pk.DependsOn)
		}
	}
	// A link across Packages is a dependency.
	q := newPlainOf(t, library())
	q.write(m+"catalog/book.md", def("std/Concept", "Book", "A book a [[members/member]] may borrow."))
	for _, pk := range q.status(false).Packages {
		if pk.Package == "catalog" && (len(pk.DependsOn) != 1 || pk.DependsOn[0] != "members") {
			t.Errorf("catalog depends on %v", pk.DependsOn)
		}
	}
	it := p.item(st, "members/loans/loan")
	if text, ok := p.review(it.Definition, run.VerdictSpecProblem, it.Fingerprint, nil, "which limit?"); !ok {
		t.Fatal(text)
	}
	for _, pk := range p.status(false).Packages {
		if pk.Package == "members/loans" && (pk.Items != 2 || pk.SpecProblems != 1) {
			t.Fatalf("after a spec-problem: %+v", pk)
		}
	}
	if broken := newPlainOf(t, map[string]string{project.ManifestFile: "libraries:\n  std: 1.2.0\n", project.SpecDir + "/x.md": "no front matter\n"}).status(false); len(broken.Packages) != 0 {
		t.Fatalf("a spec with errors has Packages: %+v", broken.Packages)
	}
}

// status limited to a Package lists the items whose Definition lies directly in
// it, Root's by its name, so that a strand of a parallel run asks the tool for
// its part and does not work it out. The errors in the realization belong to no
// Package: it leaves them out and says how many there are. A Package verified is
// clean though others are not, and a Package the spec does not have is a usage
// error.
func TestStatusOfOnePackage(t *testing.T) {
	p := newPlain(t)
	p.write("src/unlisted.go", "package src\n")
	whole := p.status(false)
	if len(whole.Errors) != 1 {
		t.Fatalf("want one error in the realization:\n%s", whole.Text())
	}
	of := func(name string, all bool) run.Status {
		t.Helper()
		st, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, All: all, Package: name})
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	for name, want := range map[string]string{"Root": "loan-limit,project", "catalog": "catalog/book,catalog/catalog", "members": "members/member,members/members", "members/loans": "members/loans/loan,members/loans/loans"} {
		st := of(name, false)
		n := strings.Count(want, ",") + 1
		if listed(st) != want || st.Definitions != n || st.Package != name || len(st.Errors) != 0 || st.ErrorsElsewhere != 1 || len(st.Packages) != len(whole.Packages) || st.Clean() {
			t.Errorf("%s: %+v", name, st)
		}
		has(t, st.Text(), "To verify in Package "+name+" (", "Not listed: 1 error in the realization, which belongs to no Package; markitect status lists it.")
		if strings.Contains(st.Text(), "src/unlisted.go") {
			t.Errorf("%s lists the errors in the realization:\n%s", name, st.Text())
		}
	}
	for _, it := range of("catalog", false).Items {
		if text, ok := p.review(it.Definition, run.VerdictOK, it.Fingerprint, []string{"src/catalog/book.go:1"}, ""); !ok {
			t.Fatal(text)
		}
	}
	st := of("catalog", false)
	if !st.Clean() || p.status(false).Clean() {
		t.Fatalf("after verifying the Package:\n%s", st.Text())
	}
	has(t, st.Text(), "Nothing to verify in Package catalog: the spec compiles, and all 2 Definitions are verified at their current inputs.", "Not listed: 1 error")
	if all := of("catalog", true); listed(all) != "catalog/book,catalog/catalog" || all.Items[0].State != run.StateOK {
		t.Fatalf("--all in a Package:\n%s", all.Text())
	}
	// A line of the record markitect cannot read belongs to no Package either:
	// the Package stays clean, and the whole list names the line.
	p.write(project.ReviewFile, p.read(project.ReviewFile)+"not a line\n")
	st = of("catalog", false)
	if !st.Clean() || len(st.Unreadable) != 0 || st.UnreadableElsewhere != 1 || len(p.status(false).Unreadable) != 1 {
		t.Fatalf("with an unreadable line:\n%s", st.Text())
	}
	has(t, st.Text(), "Nothing to verify in Package catalog", "Not listed: 1 error in the realization and 1 line of the record that markitect cannot read, which belong to no Package; markitect status lists them.")
	if _, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Package: "loans"}); !errors.Is(err, run.ErrUsage) || !strings.Contains(err.Error(), `no Package "loans"`) {
		t.Fatalf("a Package the spec does not have: %v", err)
	}
	// A spec with errors has no list to limit, whatever Package is asked for.
	broken := newPlainOf(t, map[string]string{project.ManifestFile: "libraries:\n  std: 1.2.0\n", project.SpecDir + "/x.md": "no front matter\n"})
	if st, err := broken.engine.Status(context.Background(), run.StatusRequest{Dir: broken.dir, Package: "nowhere"}); err != nil || len(st.SpecErrors) == 0 {
		t.Fatalf("a spec with errors: %v\n%s", err, st.Text())
	}
}

// A folder without an anchor only sorts: the Definitions in it
// belong to the Package above, to its Definitions and its items, also once one
// is gone, and the folder is no Package of its own in status.
func TestStatusCountsASortFolderForThePackageAbove(t *testing.T) {
	files := library()
	files[m+"members/rules/limit.md"] = def("std/Rule", "Limit", "At most `5` open loans a [[members/member]].")
	files[project.RegistryDir+"/members.md"] += "| src/members/limit.go | [[members/rules/limit]] | Limit |\n"
	files["src/members/limit.go"] = "package members\n"
	p := newPlainOf(t, files)
	of := func(name string) run.Status {
		t.Helper()
		st, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Package: name})
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	members := func(st run.Status) run.PackageItems {
		t.Helper()
		for _, pk := range st.Packages {
			if pk.Package == "members" {
				return pk
			}
			if strings.HasPrefix(pk.Package, "members/") && pk.Package != "members/loans" {
				t.Fatalf("a folder that only sorts is a Package: %+v", st.Packages)
			}
		}
		t.Fatalf("no Package members: %+v", st.Packages)
		return run.PackageItems{}
	}
	st := p.status(false)
	if pk := members(st); pk.Definitions != 3 || pk.Items != 3 || len(st.Packages) != 4 {
		t.Fatalf("members with a sort folder: %+v of %+v", pk, st.Packages)
	}
	if of("members").Definitions != 3 || listed(of("members")) != "members/member,members/members,members/rules/limit" {
		t.Fatalf("status of members:\n%s", of("members").Text())
	}
	if _, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Package: "members/rules"}); !errors.Is(err, run.ErrUsage) {
		t.Fatalf("a sort folder is no Package to ask for: %v", err)
	}
	// Gone, the Definition is still counted for the Package above its folder.
	p.verifyAll()
	p.remove(m + "members/rules/limit.md")
	st = p.status(false)
	gone := p.item(st, "members/rules/limit")
	if gone.State != run.StateGone || len(st.Packages) != 4 {
		t.Fatalf("a gone Definition of a sort folder:\n%s", st.Text())
	}
	if pk := members(st); pk.Definitions != 2 || pk.Items != 1 {
		t.Fatalf("the gone item is not members': %+v", pk)
	}
	if only := of("members"); listed(only) != "members/rules/limit" || only.Definitions != 2 || listed(of("members/loans")) != "" || listed(of("Root")) != "" {
		t.Fatalf("status of members:\n%s", only.Text())
	}
}

// What is to verify is what a change touches at distance 0 and 1.
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
		}, "loan-limit,members/loans/loan,members/loans/loans"},
		{"a row that gains a Definition reaches it", func(p *plain) {
			p.write(project.RegistryDir+"/members.md", "---\npaths: [src/members/**]\n---\n"+table+"| src/members/member.go | [[members/member]], [[members/members]], [[loan-limit]] | Member |\n")
		}, "loan-limit"},
		{"a row that drops a Definition reaches it, though the file did not change", func(p *plain) {
			p.write(project.RegistryDir+"/members/loans.md", "---\npaths: [src/members/loans/**]\n---\n"+table+"| src/members/loans/loan.go | [[members/loans/loan]], [[members/loans/loans]] | Loans |\n")
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
	files[project.ManifestFile] = "libraries:\n  std: 1.2.0\n  web: 0.1.0\n"
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
		p.write(project.LockFile, string(spec.RenderLock(pr.Libraries)))
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
	p.write(project.RegistryDir+"/members.md", "---\npaths: [src/members/**]\n---\n"+table+"| src/members/member.go | [[members/member]], [[members/members]] | Member |\n| src/members/member_test.go | [[members/member]] | Test |\n")
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

// The architect's questions for the owner are a note: status says where they
// wait while the file exists, they leave it clean, they are no input of any
// Definition, and a list limited to a Package leaves them out.
func TestQuestionsAreANote(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	if st := p.status(false); st.Questions || strings.Contains(st.Text(), "questions") {
		t.Fatalf("questions without a file:\n%s", st.Text())
	}
	p.write(project.QuestionsFile, "# Questions\n\n1. Is a member a person or a household?\n")
	st := p.status(false)
	if !st.Questions || !st.Clean() || len(st.Items) != 0 {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), "Nothing to verify", "\nThe architect's questions for the owner wait in .markitect/questions.md.\n")
	in, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Package: "members"})
	if err != nil || in.Questions || strings.Contains(in.Text(), "questions") {
		t.Fatalf("%v\n%s", err, in.Text())
	}
	p.remove(project.QuestionsFile)
	if st := p.status(false); st.Questions {
		t.Fatalf("%s", st.Text())
	}
}

// When every item is a spec-problem recorded at the inputs as they are, status
// says that only they are left: a builder settles none of them. A fail beside
// them, a changed input of one of them or an error in the realization ends that.
func TestOnlySpecProblemsAreForTheOwner(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	if st := p.status(false); st.ForTheOwner() {
		t.Fatalf("a clean status is for the owner:\n%s", st.Text())
	}
	st := p.status(true)
	if _, ok := p.review("members/member", run.VerdictSpecProblem, p.item(st, "members/member").Fingerprint, nil, "is a member a person or a household?"); !ok {
		t.Fatal("spec-problem not recorded")
	}
	st = p.status(false)
	if listed(st) != "members/member" || !st.ForTheOwner() || st.Clean() {
		t.Fatalf("%s", st.Text())
	}
	has(t, st.Text(), "\nOnly spec-problems are left: they wait for the owner, and no builder settles them.\n")
	in, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Package: "members"})
	if err != nil || !in.ForTheOwner() {
		t.Fatalf("%v\n%s", err, in.Text())
	}
	has(t, in.Text(), "Only spec-problems are left in Package members:")
	// An error in the realization is the builder's.
	p.write("src/stray.go", "package stray\n")
	if st := p.status(false); st.ForTheOwner() || strings.Contains(st.Text(), "Only spec-problems") {
		t.Fatalf("with an unlisted file:\n%s", st.Text())
	}
	p.remove("src/stray.go")
	if st := p.status(false); !st.ForTheOwner() {
		t.Fatalf("%s", st.Text())
	}
	// A changed input of the Definition puts it back to verify: the line says
	// what was recorded before, but nothing is recorded at the inputs as they are.
	p.write("src/members/member.go", "package members\n\n// changed\n")
	if st := p.status(false); st.ForTheOwner() || strings.Contains(st.Text(), "Only spec-problems") || !strings.Contains(st.Text(), "recorded before: spec-problem") {
		t.Fatalf("with a changed input:\n%s", st.Text())
	}
	p.write("src/members/member.go", "package members\n")
	if st := p.status(false); !st.ForTheOwner() {
		t.Fatalf("%s", st.Text())
	}
	// A fail beside it is the builder's.
	all := p.status(true)
	if _, ok := p.review("loan-limit", run.VerdictFail, p.item(all, "loan-limit").Fingerprint, []string{"src/members/loans/loan.go:3"}, "allows a sixth loan"); !ok {
		t.Fatal("fail not recorded")
	}
	if st := p.status(false); st.ForTheOwner() || strings.Contains(st.Text(), "Only spec-problems") {
		t.Fatalf("%s", st.Text())
	}
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
	if listed(st) != "loan-limit,members/member" || st.Clean() || st.ForTheOwner() || strings.Contains(st.Text(), "Only spec-problems") {
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
	p.write(project.RegistryDir+"/catalog.md", "---\npaths: [src/catalog/**]\n---\n"+table+"| src/catalog/book.go | [[catalog/book]], [[catalog/catalog]] | Book |\n")
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
	for _, id := range []string{"loan-limit", "members/loans/loan", "members/loans/loans"} {
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
	if n := len(strings.Split(strings.TrimSuffix(rec, "\n"), "\n")); n != 8 {
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
	has(t, st.Text(), ".markitect/review.sum:9: record.unreadable: ", ".markitect/review.sum:10: record.unreadable: ")
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
	one := 1
	var wg sync.WaitGroup
	errs := make(chan string, 2*len(st.Items))
	for _, it := range st.Items {
		wg.Add(1)
		go func(it run.Item) {
			defer wg.Done()
			text, ok, err := p.engine.Mark(context.Background(), run.MarkRequest{Dir: p.dir, Item: it.Definition + "@" + it.Fingerprint, Verdict: run.VerdictOK, Evidence: []string{"README.md"}, Cases: &one, Covered: &one})
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

// mark keeps the reviewer's counts of cases on the line, and the record counts
// how often a Definition's own text changed between verdicts (DEC-102): a verdict
// at another own text than every line before it adds one; a change of a link or a
// file does not.
func TestCountsAndRevisions(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	if rec := p.read(project.ReviewFile); strings.Contains(rec, "revisions") {
		t.Fatalf("a first verdict counts no revision:\n%s", rec)
	}
	mark := func(id, verdict string, cases, covered *int, why string, evidence ...string) string {
		t.Helper()
		text, ok, err := p.engine.Mark(context.Background(), run.MarkRequest{Dir: p.dir, Item: id + "@" + p.item(p.status(true), id).Fingerprint, Verdict: verdict, Evidence: evidence, Why: why, Cases: cases, Covered: covered})
		if err != nil || !ok {
			t.Fatalf("mark %s: %v %s", id, err, text)
		}
		return text
	}
	line := func(id string) run.Line {
		t.Helper()
		for _, l := range run.ParseRecord([]byte(p.read(project.ReviewFile))).Lines {
			if l.Definition == id {
				return l
			}
		}
		t.Fatalf("no line for %s", id)
		return run.Line{}
	}
	three, two, one := 3, 2, 1
	p.write(m+"members/member.md", def("std/Concept", "Member", "A person with a card."))
	mark("members/member", run.VerdictOK, &three, &two, "", "src/members/member.go:1")
	mark("members/loans/loan", run.VerdictOK, &one, &one, "", "src/members/loans/loan.go:1")
	if l := line("members/member"); l.Revisions != 1 || l.Cases == nil || *l.Cases != 3 || l.Covered == nil || *l.Covered != 2 {
		t.Fatalf("member: %+v", l)
	}
	if l := line("members/loans/loan"); l.Revisions != 0 || l.Cases == nil || *l.Cases != 1 {
		t.Fatalf("a changed link is no revision of loan: %+v", l)
	}
	has(t, p.read(project.ReviewFile), `"cases":3,"covered":2,"revisions":1`)
	// A changed file is no revision; a fail at a new text is one.
	p.write("src/members/member.go", "package members\n\ntype Member struct{}\n")
	mark("members/member", run.VerdictOK, &one, &one, "", "src/members/member.go:3")
	if l := line("members/member"); l.Revisions != 1 || l.Cases == nil || *l.Cases != 1 {
		t.Fatalf("after a changed file: %+v", l)
	}
	p.write(m+"members/member.md", def("std/Concept", "Member", "A person with a card and a locker."))
	zero := 0
	mark("members/member", run.VerdictFail, &zero, nil, "no locker", "src/members/member.go:3")
	if got := run.ParseRecord([]byte(p.read(project.ReviewFile))).Revisions(); len(got) != 1 || got["members/member"] != 2 {
		t.Fatalf("revisions = %v", got)
	}
	// After a merge, the next verdict at a text no line has counts on from the most.
	rec := p.read(project.ReviewFile)
	for _, l := range strings.Split(rec, "\n") {
		if strings.Contains(l, `"definition":"members/member"`) {
			p.write(project.ReviewFile, rec+strings.Replace(strings.Replace(l, `"revisions":2`, `"revisions":5`, 1), `"self":"sha256:`, `"self":"sha256:0`, 1)+"\n")
		}
	}
	// A verdict on another Definition writes the merged one's matching line with the
	// most revisions any of its lines counted.
	mark("catalog/book", run.VerdictOK, &one, &one, "", "src/catalog/book.go:1")
	if got := run.ParseRecord([]byte(p.read(project.ReviewFile))).Revisions()["members/member"]; got != 5 {
		t.Fatalf("a verdict on another Definition lost revisions of a merged one: %d", got)
	}
	p.write(m+"members/member.md", def("std/Concept", "Member", "A person."))
	mark("members/member", run.VerdictOK, &one, &one, "", "src/members/member.go:3")
	if l := line("members/member"); l.Revisions != 6 {
		t.Fatalf("after a merge: %+v", l)
	}
	// A gone Definition has no own text, so a verdict on it counts no revision.
	p.write(m+"catalog/shelf.md", def("std/Concept", "Shelf", "Holds books."))
	p.remove(m + "catalog/book.md")
	mark("catalog/book", run.VerdictFail, nil, nil, "its file is still there")
	if l := line("catalog/book"); l.Revisions != 0 || l.Verdict != run.VerdictFail {
		t.Fatalf("a gone Definition: %+v", l)
	}
	// Counts that cannot be are refused before anything is read, and an ok or a
	// skip without both counts is refused too: the reviewer counts with them.
	minus := -1
	for _, req := range []run.MarkRequest{
		{Covered: &two},
		{Cases: &two, Covered: &three},
		{Cases: &minus},
		{},
		{Cases: &two},
		{Verdict: run.VerdictSkip, Why: "no test can reach it"},
		{Verdict: run.VerdictSkip, Why: "no test can reach it", Cases: &two},
	} {
		if req.Verdict == "" {
			req.Verdict = run.VerdictOK
		}
		req.Dir, req.Item, req.Evidence = p.dir, "members/member@"+p.item(p.status(true), "members/member").Fingerprint, []string{"README.md"}
		if _, _, err := p.engine.Mark(context.Background(), req); !errors.Is(err, run.ErrUsage) {
			t.Fatalf("%+v: %v", req, err)
		}
	}
	// The record gives vet the counts of an ok or a skip at a Definition's text as
	// it is.
	self := func(ids ...string) map[string]string {
		out := map[string]string{}
		for _, id := range ids {
			out[id] = run.Hash([]byte(p.read(m + id + ".md")))
		}
		return out
	}
	now := self("members/loans/loan", "members/member")
	counts := run.ParseRecord([]byte(p.read(project.ReviewFile))).Counts(now)
	if c := counts["members/loans/loan"]; len(c) != 1 || c[0].Cases != 1 || c[0].Covered != 1 || c[0].Skip {
		t.Fatalf("counts of loan: %+v", c)
	}
	if c := counts["members/member"]; len(c) != 1 || c[0].Cases != 1 {
		t.Fatalf("counts of member, with an ok at its text: %+v", c)
	}
	// A line at another text of the Definition is not judged: the Definition has
	// changed since, and status lists it.
	p.write(m+"members/member.md", def("std/Concept", "Member", "Somebody."))
	if _, ok := run.ParseRecord([]byte(p.read(project.ReviewFile))).Counts(self("members/member"))["members/member"]; ok {
		t.Fatal("a line at another text has counts")
	}
	// After a merge the record may hold several lines at one text: all their
	// counts are returned, an ok's and a skip's alike, and a fail at that text
	// leaves none, whatever the order.
	line2 := func(verdict string, cases, covered int) string {
		return fmt.Sprintf(`{"definition":"x","verdict":%q,"inputs":{"self":"sha256:1"},"why":"w","cases":%d,"covered":%d}`, verdict, cases, covered)
	}
	at := map[string]string{"x": "sha256:1"}
	for _, lines := range [][]string{
		{line2("ok", 16, 16), line2("skip", 9, 9)},
		{line2("skip", 9, 9), line2("ok", 16, 16)},
	} {
		c := run.ParseRecord([]byte(strings.Join(lines, "\n") + "\n")).Counts(at)["x"]
		if len(c) != 2 || c[0].Skip == c[1].Skip {
			t.Fatalf("of two lines at one text: %+v", c)
		}
	}
	for _, lines := range [][]string{
		{line2("ok", 16, 16), line2("fail", 16, 2)},
		{line2("fail", 16, 2), line2("ok", 16, 16)},
	} {
		if c := run.ParseRecord([]byte(strings.Join(lines, "\n") + "\n")).Counts(at)["x"]; len(c) != 0 {
			t.Fatalf("an ok beside a fail at one text has counts: %v", lines)
		}
	}
	// A Definition gone from the spec has no cases: its ok needs no counts.
	if text, ok, err := p.engine.Mark(context.Background(), run.MarkRequest{Dir: p.dir, Item: "catalog/book@" + p.item(p.status(true), "catalog/book").Fingerprint, Verdict: run.VerdictOK}); err != nil || !ok {
		t.Fatalf("an ok without counts on a gone Definition: %v %s", err, text)
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
