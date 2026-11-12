package analysis

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

type memRepo map[string]string

func (r memRepo) Files() ([]string, error) {
	var out []string
	for f := range r {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

func (r memRepo) Read(f string) ([]byte, error) {
	if data, ok := r[f]; ok {
		return []byte(data), nil
	}
	return nil, fmt.Errorf("no %s", f)
}

func (memRepo) GitIgnored(string) bool { return false }

func (r memRepo) clone() memRepo {
	out := memRepo{}
	for k, v := range r {
		out[k] = v
	}
	return out
}

const (
	m     = project.SpecDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
)

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

// pkg makes a folder of a fixture a Package: it adds the anchor, the
// Definition of the folder's own name.
func pkg(r memRepo, paths ...string) {
	for _, path := range paths {
		name := path[strings.LastIndexByte(path, '/')+1:]
		r[m+path+"/"+name+".md"] = def("std/Package", "Part "+name, "")
	}
}

// library returns a small project: Root with a Rule and the Project, members
// with a sub-Package loans, and catalog, which depends on members.
func library(t *testing.T) memRepo {
	t.Helper()
	files, _ := libraries.Files("std")
	std := spec.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, spec.File{Path: f.Path, Data: f.Data})
	}
	return memRepo{
		project.ManifestFile:                      "libraries:\n  std: 1.2.0\n",
		project.LockFile:                          string(spec.RenderLock([]spec.LibrarySource{std})),
		m + "project.md":                          def("std/Project", "Library", "Parts: [[members]], [[catalog]]."),
		m + "loan-limit.md":                       def("std/Rule", "Loan limit", "At most `5` open loans."),
		m + "members/members.md":                  def("std/Package", "Members", "Who may borrow."),
		m + "members/member.md":                   def("std/Concept", "Member", "A person."),
		m + "members/loans/loans.md":              def("std/Package", "Loans", "What is lent out."),
		m + "members/loans/loan.md":               def("std/Concept", "Loan", "A [[members/member]] borrows."),
		m + "catalog/catalog.md":                  def("std/Package", "Catalog", "What can be borrowed."),
		m + "catalog/book.md":                     def("std/Concept", "Book", "Lent to a [[members/member]]."),
		m + "catalog/shelf.md":                    def("std/Concept", "Shelf", "Holds books."),
		project.RootPage:                          table + "| README.md | [[project]] [[members/member]] | Overview |\n",
		project.RegistryDir + "/members.md":       "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Type |\n",
		project.RegistryDir + "/members/loans.md": "---\npaths: [src/members/loans/**]\n---\n" + table + "| src/members/loans/loan.go | [[members/loans/loan]] | Type |\n",
		project.RegistryDir + "/catalog.md":       "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] [[catalog/shelf]] | Types |\n",
		"README.md":                               "# Library\n",
		"src/members/member.go":                   "package members\n",
		"src/members/loans/loan.go":               "package loans\n",
		"src/catalog/book.go":                     "package catalog\n",
	}
}

func loadRepo(t *testing.T, r memRepo) *project.Project {
	t.Helper()
	p, err := project.Load(r)
	if err != nil {
		t.Fatal(err)
	}
	if errs := p.SpecErrors(); len(errs) != 0 {
		t.Fatalf("fixture does not compile:\n%v", errs)
	}
	return p
}

func TestRefs(t *testing.T) {
	p := loadRepo(t, library(t))
	refs, err := Refs(p, "[[members/member]]")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range refs {
		got = append(got, r.From+" "+r.Via)
	}
	want := "catalog/book link,members/loans/loan link,README.md registry,src/members/member.go registry"
	if strings.Join(got, ",") != want {
		t.Fatalf("refs = %v", got)
	}
	kinds, _ := Refs(p, "std/Concept")
	if len(kinds) != 5 || kinds[0].Via != "kind" {
		t.Fatalf("kind refs = %+v", kinds)
	}
	pkg, _ := Refs(p, "members")
	if len(pkg) != 1 || pkg[0].From != "project" {
		t.Fatalf("package refs = %+v", pkg)
	}
	if _, err := Refs(p, "nope"); err == nil {
		t.Fatal("unknown target accepted")
	}
}

func TestGraphMeasures(t *testing.T) {
	r := library(t)
	// A Kind of catalog's own makes it a quarter abstract: A 1/4, I 1, D 1/4.
	r[m+"catalog/Medium.md"] = def("Kind", "Medium", "A thing to lend.")
	g := BuildGraph(loadRepo(t, r))
	if len(g.Edges) != 1 || g.Edges[0].From != "catalog" || g.Edges[0].To != "members" {
		t.Fatalf("edges = %+v", g.Edges)
	}
	catalog, members := g.node("catalog"), g.node("members")
	if catalog.Efferent != 1 || catalog.Instability != 1 || members.Afferent != 1 || members.Instability != 0 {
		t.Fatalf("catalog %+v members %+v", catalog, members)
	}
	if fmt.Sprintf("%.3f %.3f", catalog.Abstractness, catalog.Distance) != "0.250 0.250" || members.Abstractness != 0 || members.Distance != 1 {
		t.Fatalf("abstractness and distance: catalog %+v members %+v", catalog, members)
	}
	// A Package without a Kind of its own is not abstract.
	if loans := g.node("members/loans"); loans.Abstractness != 0 || loans.Distance != 1 {
		t.Fatalf("members/loans %+v", loans)
	}
}

func vetCodes(t *testing.T, r memRepo) spec.Diagnostics {
	t.Helper()
	return Vet(loadRepo(t, r), History{}).Warnings
}

// vetNotes returns vet's notes on a fixture, as text lines, after the Revisions
// the record would give.
func vetNotes(t *testing.T, r memRepo, revisions map[string]int) []Note {
	t.Helper()
	return Vet(loadRepo(t, r), History{Revisions: revisions}).Notes
}

func expectWarning(t *testing.T, ds spec.Diagnostics, code, fragment string) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code && strings.Contains(d.String(), fragment) {
			return
		}
	}
	t.Fatalf("no %s containing %q in:\n%v", code, fragment, ds)
}

// noteLines are the notes of one code, as vet --strict prints them.
func noteLines(notes []Note, code string) []string {
	var out []string
	for _, n := range notes {
		if n.Code == code {
			out = append(out, n.String())
		}
	}
	return out
}

func expectLines(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestVetCoupling(t *testing.T) {
	r := library(t)
	var links []string
	for i := 0; i < MaxEfferent+1; i++ {
		name := fmt.Sprintf("p%d", i)
		pkg(r, name)
		r[m+name+"/x.md"] = def("std/Rule", "X", "")
		links = append(links, "[["+name+"/x]]")
	}
	pkg(r, "hub")
	r[m+"hub/h.md"] = def("std/Rule", "Hub", strings.Join(links, " "))
	expectWarning(t, vetCodes(t, r), "vet.coupling", "hub depends on 6 Packages")
}

func TestVetUnstableDependency(t *testing.T) {
	r := library(t)
	// shelf-tools depends on catalog, which also depends on members: catalog's
	// instability drops below shelf-tools'... and members depends on extra.
	pkg(r, "extra", "reports", "audits", "x1", "x2")
	r[m+"extra/e.md"] = def("std/Rule", "E", "")
	r[m+"members/member.md"] = def("std/Concept", "Member", "Uses [[extra/e]].")
	r[m+"reports/r.md"] = def("std/Rule", "R", "Counts [[members/member]].")
	r[m+"audits/a.md"] = def("std/Rule", "A", "Checks [[members/member]].")
	// members: Ca 3 (catalog, reports, audits), Ce 1 -> 0.25; extra: Ca 1, Ce 0 -> 0.
	// Make extra unstable: it depends on two more Packages.
	r[m+"x1/a.md"] = def("std/Rule", "X1", "")
	r[m+"x2/a.md"] = def("std/Rule", "X2", "")
	r[m+"extra/e.md"] = def("std/Rule", "E", "[[x1/a]] [[x2/a]]")
	expectWarning(t, vetCodes(t, r), "vet.unstable_dependency", "members (instability 0.25) depends on the less stable extra (instability 0.67)")
}

func TestVetSizes(t *testing.T) {
	r := library(t)
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]]. "+strings.Repeat("word ", MaxDefinitionWords))
	// With its anchor, big has one Definition more than the limit.
	pkg(r, "big")
	for i := 0; i < MaxPackageSize; i++ {
		r[m+fmt.Sprintf("big/d%02d.md", i)] = def("std/Rule", "D", "")
	}
	ds := vetCodes(t, r)
	expectWarning(t, ds, "vet.definition_size", "catalog/book.md: vet.definition_size")
	expectWarning(t, ds, "vet.package_size", "big has 26 Definitions")
}

func TestVetThinPackage(t *testing.T) {
	r := library(t)
	pkg(r, "wrapper", "wrapper/inner")
	r[m+"wrapper/inner/i.md"] = def("std/Rule", "I", "")
	// A folder that only sorts adds no level of Packages: members/rules is none.
	r[m+"members/rules/late.md"] = def("std/Rule", "Late", "")
	ds := vetCodes(t, r)
	expectWarning(t, ds, "vet.thin_package", "wrapper holds only its anchor and one child, wrapper/inner")
	for _, d := range ds {
		if d.Code == "vet.thin_package" && !strings.Contains(d.Message, "wrapper holds") {
			t.Fatalf("unexpected %v", d)
		}
	}
}

// vet.hub counts Definitions, Root included, and leaves Kinds out: it warns when
// a change to one Definition alone puts more than a fifth of the spec, and more
// than 10, back on the list.
func TestVetHub(t *testing.T) {
	r := library(t)
	r[m+"Order.md"] = def("Kind", "Order", "An order.")
	// members/member is linked by loan and book; nine orders more make 11 linkers,
	// so a change lists 12 of 20 Definitions. loan-limit at Root is linked by 9
	// orders, which lists 10: not more than 10.
	pkg(r, "orders")
	for i := 0; i < 9; i++ {
		r[m+fmt.Sprintf("orders/o%d.md", i)] = def("Order", "O", "For a [[members/member]], under the [[loan-limit]].")
	}
	ds := vetCodes(t, r)
	expectWarning(t, ds, "vet.hub", "members/member.md: vet.hub: 11 Definitions link to members/member or are of it as Kind, so a change to it alone puts 12 of the spec's 20 back on the list")
	for _, d := range ds {
		if d.Code == "vet.hub" && !strings.Contains(d.File, "members/member") {
			t.Fatalf("unexpected %v", d)
		}
	}
	// One order more puts loan-limit, at Root, over the line; the Kind Order, with
	// as many Definitions of it, stays no hub.
	r[m+"orders/o9.md"] = def("Order", "O", "Under the [[loan-limit]].")
	ds = vetCodes(t, r)
	expectWarning(t, ds, "vet.hub", "loan-limit.md: vet.hub: 10 Definitions link to loan-limit")
	for _, d := range ds {
		if d.Code == "vet.hub" && strings.Contains(d.File, "Order") {
			t.Fatalf("a Kind is no hub: %v", d)
		}
	}
}

// vet.statements notes a Definition of more than 20 list items and table rows.
func TestVetStatements(t *testing.T) {
	r := library(t)
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]].\n\n"+strings.Repeat("- a case\n", 18)+"\n| Field | Format |\n|---|---|\n| title | text |\n")
	if got := noteLines(vetNotes(t, r, nil), "vet.statements"); len(got) != 0 {
		t.Fatalf("20 statements noted: %v", got)
	}
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]].\n\n"+strings.Repeat("- a case\n", 18)+"  - a nested case\n\n| Field | Format |\n|---|---|\n| title | text |\n")
	expectLines(t, noteLines(vetNotes(t, r, nil), "vet.statements"),
		m+"catalog/book.md: vet.statements: 21 statements, list items and table rows (more than 20); a reviewer verifies each, so split it along its ideas")
}

// vet.placement notes a Definition whose linkers all lie below it in one Package,
// the lowest that holds them all.
func TestVetPlacement(t *testing.T) {
	r := library(t)
	r[m+"fees.md"] = def("std/Rule", "Fees", "A fee per day.")
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]] for the [[fees]].")
	r[m+"catalog/shelf.md"] = def("std/Concept", "Shelf", "Holds books; [[fees]] apply.")
	r[m+"renewal.md"] = def("std/Rule", "Renewal", "Twice.")
	r[m+"members/loans/loan.md"] = def("std/Concept", "Loan", "A [[members/member]] borrows, under the [[renewal]].")
	// members/member is linked from catalog and from members/loans: it stays.
	expectLines(t, noteLines(vetNotes(t, r, nil), "vet.placement"),
		m+"fees.md: vet.placement: every Definition that links to fees lies in catalog; it could move down there, unless it binds Root by scope",
		m+"renewal.md: vet.placement: every Definition that links to renewal lies in members/loans; it could move down there, unless it binds Root by scope")
}

// vet.hotspot notes a Definition whose fan-in times the revisions the record
// counts goes above a tenth of the spec, and above 20.
func TestVetHotspot(t *testing.T) {
	r := library(t)
	// members/member has fan-in 2 (loan and book); the spec has 9 Definitions.
	if got := noteLines(vetNotes(t, r, map[string]int{"members/member": 10, "catalog/shelf": 50}), "vet.hotspot"); len(got) != 0 {
		t.Fatalf("2 × 10 is not above 20, and what nothing links is no hotspot: %v", got)
	}
	expectLines(t, noteLines(vetNotes(t, r, map[string]int{"members/member": 11}), "vet.hotspot"),
		m+"members/member.md: vet.hotspot: its own text changed 11 times between verdicts, and 2 Definitions link to it or are of it as Kind, so its changes put about 22 back on the list to verify, more than a tenth of the spec; split what changes from what relies on it")
}

// vet.risk notes a Definition whose ok or skip rests on thin evidence, by
// the reviewer's counts in the record: c² × (1 − e)³ + c above 30, with c one
// more than its cases and e the share of them the evidence demonstrates.
func TestVetRisk(t *testing.T) {
	r := library(t)
	risk := func(counts map[string]Count) []string {
		t.Helper()
		return noteLines(Vet(loadRepo(t, r), History{Counts: counts}).Notes, "vet.risk")
	}
	// 12 cases with 8 demonstrated is about 19; 7 cases all demonstrated is 8; a
	// skip of 3 cases is 20; a Definition the record has no counts for is not judged.
	if got := risk(map[string]Count{
		"members/member": {Cases: 12, Covered: 8},
		"catalog/book":   {Cases: 7, Covered: 7},
		"catalog/shelf":  {Cases: 3, Skip: true},
	}); len(got) != 0 {
		t.Fatalf("below the line: %v", got)
	}
	expectLines(t, risk(map[string]Count{
		"members/member": {Cases: 16, Covered: 10},
		"catalog/book":   {Cases: 9, Covered: 9, Skip: true},
		"catalog/shelf":  {Cases: 30, Covered: 30},
	}),
		m+"catalog/book.md: vet.risk: recorded skip with 9 cases, and a skip counts none of them as demonstrated (risk 110, more than 30); find a way to show them, or split the Definition",
		m+"catalog/shelf.md: vet.risk: 30 cases, too many to verify well however much evidence there is (risk 31, more than 30); split it along its ideas",
		m+"members/member.md: vet.risk: its ok rests on evidence for 10 of its 16 cases (risk 33, more than 30); the next review demonstrates the others, or the Definition is split")
	// Thirty cases and more are too many whatever is demonstrated, also when
	// some are not; and the risk is printed rounded up, never as the line itself.
	expectLines(t, risk(map[string]Count{
		"members/member": {Cases: 31, Covered: 30},
		"catalog/book":   {Cases: 28, Covered: 25},
	}),
		m+"catalog/book.md: vet.risk: its ok rests on evidence for 25 of its 28 cases (risk 31, more than 30); the next review demonstrates the others, or the Definition is split",
		m+"members/member.md: vet.risk: 31 cases, too many to verify well however much evidence there is (risk 33, more than 30); split it along its ideas")
	// The measure itself, at the values the guide gives.
	for _, c := range []struct {
		count Count
		want  float64
	}{
		{Count{Cases: 7, Covered: 2}, 31.3},
		{Count{Cases: 7, Covered: 4}, 13.0},
		{Count{Cases: 0}, 1},
		{Count{Cases: 0, Skip: true}, 2},
		// A hand-edited record may claim more than every case: no negative risk.
		{Count{Cases: 4, Covered: 9}, 5},
	} {
		if got := c.count.Risk(); got < c.want-0.05 || got > c.want+0.05 {
			t.Errorf("%+v: risk %.2f, want %.1f", c.count, got, c.want)
		}
	}
}

// vet.unlinked_mention finds a Definition's title or name in another one's prose
// without a link, from the text and the structure alone.
func TestVetUnlinkedMentions(t *testing.T) {
	r := library(t)
	r[m+"Medium.md"] = def("Kind", "Medium", "A thing to lend.")
	r[m+"members/fee.md"] = "---\nkind: std/Rule\ninternal: true\n---\n# Fee\n\nPer day.\n"
	// Root: the names it may not link, into a child, are no mentions.
	r[m+"loan-limit.md"] = def("std/Rule", "Loan limit", "At most `5` open loans per member.")
	// Within the structure: book in its own Package, member along catalog's
	// dependency on members, and the loan limit at Root, whose longer name covers
	// "loan". A Kind's name, an internal Definition outside its Package, a word in
	// code, in a link label or joined by a hyphen, and an ending of more than two
	// letters are none.
	r[m+"catalog/shelf.md"] = def("std/Concept", "Shelf", "Holds books for a\nmember, within the loan limit.\n\nAny medium, no fee, `book`, [[members|the member's]] own, member-only, bookings.")
	// Against the structure: members may not depend on catalog, which depends on
	// members. Beyond it: members/loans does not depend on catalog yet.
	r[m+"members/member.md"] = def("std/Concept", "Member", "A person who may borrow a Book.")
	r[m+"members/loans/loan.md"] = def("std/Concept", "Loan", "A [[members/member]] borrows a book\nfrom a shelf; the members wait.")
	notes := vetNotes(t, r, nil)
	expectLines(t, noteLines(notes, "vet.unlinked_mention"),
		m+"catalog/shelf.md:6: vet.unlinked_mention: names catalog/book, loan-limit and members/member without linking them",
		m+"members/loans/loan.md:6: vet.unlinked_mention: names catalog/book and catalog/shelf without linking them; links would add Package dependencies the spec does not have",
		m+"members/member.md:6: vet.unlinked_mention: names catalog/book without linking it; a link would close a Package cycle")
	for _, n := range notes {
		if n.Code != "vet.unlinked_mention" {
			continue
		}
		want := map[string]string{"catalog/shelf.md": "", "members/loans/loan.md": StructureNew, "members/member.md": StructureAgainst}[strings.TrimPrefix(n.File, m)]
		if n.Structure != want || len(n.Targets) == 0 {
			t.Fatalf("%s: structure %q, targets %v", n.File, n.Structure, n.Targets)
		}
	}
	// An anchor carries its Package's name, which prose uses freely: "the
	// catalog" names no Definition.
	r[m+"catalog/shelf.md"] = def("std/Concept", "Shelf", "A part of the Catalog.")
	for _, line := range noteLines(vetNotes(t, r, nil), "vet.unlinked_mention") {
		if strings.Contains(line, "catalog/shelf.md") {
			t.Fatalf("the name of a Package reported as a mention of its anchor: %s", line)
		}
	}
	// The anchor's name still covers its words: "book catalog" does not name
	// the book inside it.
	r[m+"catalog/catalog.md"] = def("std/Package", "Book catalog", "What can be borrowed.")
	r[m+"catalog/shelf.md"] = def("std/Concept", "Shelf", "A part of the book catalog.")
	for _, line := range noteLines(vetNotes(t, r, nil), "vet.unlinked_mention") {
		if strings.Contains(line, "catalog/shelf.md") {
			t.Fatalf("a word inside the name of a Package reported: %s", line)
		}
	}
	r[m+"catalog/catalog.md"] = def("std/Package", "Catalog", "What can be borrowed.")
	// A linked Definition's name covers the words: "members" beside a linked
	// members/member is no mention of members/member, nor of anything else.
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to [[members/member]]s; members queue.")
	for _, line := range noteLines(vetNotes(t, r, nil), "vet.unlinked_mention") {
		if strings.Contains(line, "catalog/book.md") {
			t.Fatalf("a covered mention reported: %s", line)
		}
	}
}

func TestVetNoPaths(t *testing.T) {
	r := library(t)
	pkg(r, "orders")
	r[m+"orders/o.md"] = def("std/Rule", "O", "[[members/member]]")
	// catalog's page names no paths, so Root owns its file.
	r[project.RegistryDir+"/catalog.md"] = table
	r[project.RootPage] += "| src/catalog/book.go | [[catalog/book]] [[catalog/shelf]] | Types |\n"
	ds := vetCodes(t, r)
	expectWarning(t, ds, "vet.no_paths", project.RegistryDir+"/orders.md: vet.no_paths: orders has no registry page")
	expectWarning(t, ds, "vet.no_paths", project.RegistryDir+"/catalog.md: vet.no_paths: catalog claims no path pattern")
	for _, d := range ds {
		if d.Code == "vet.no_paths" && !strings.Contains(d.Message, "orders") && !strings.Contains(d.Message, "catalog") {
			t.Fatalf("unexpected %v", d)
		}
	}
}
