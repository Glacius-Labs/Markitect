package analysis

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
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
	m     = project.ModelDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
)

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

// library returns a small project: Root with a Rule and the anchor, members
// with a sub-Package loans, and catalog, which depends on members.
func library(t *testing.T) memRepo {
	t.Helper()
	files, _ := libraries.Files("std")
	std := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, model.File{Path: f.Path, Data: f.Data})
	}
	return memRepo{
		project.ManifestFile:                      "libraries:\n  std: 1.0.0\n",
		project.LockFile:                          string(model.RenderLock([]model.LibrarySource{std})),
		project.ChecksFile:                        "checks: []\n",
		m + "project.md":                          def("std/Project", "Library", "Parts: [[members]], [[catalog]]."),
		m + "loan-limit.md":                       def("std/Rule", "Loan limit", "At most `5` open loans."),
		m + "members/member.md":                   def("std/Concept", "Member", "A person."),
		m + "members/loans/loan.md":               def("std/Concept", "Loan", "A [[members/member]] borrows."),
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
	if len(p.Errors) != 0 {
		t.Fatalf("fixture does not compile:\n%v", p.Errors)
	}
	return p
}

func impact(t *testing.T, base, cur memRepo, changed []Change, findings []Finding, full bool) Impact {
	t.Helper()
	return ComputeImpact(ImpactInput{Base: loadRepo(t, base), Current: loadRepo(t, cur), Changed: changed, Findings: findings, Full: full})
}

func checklist(im Impact, pkg string) (Checklist, bool) {
	for _, c := range im.Checklists {
		if c.Package == pkg {
			return c, true
		}
	}
	return Checklist{}, false
}

func ids(items []Item) string {
	var out []string
	for _, it := range items {
		out = append(out, it.Definition)
	}
	return strings.Join(out, ",")
}

func TestImpactNothingChanged(t *testing.T) {
	base := library(t)
	im := impact(t, base, base.clone(), nil, nil, false)
	if len(im.Packages) != 0 || len(im.Reached) != 0 {
		t.Fatalf("impact of no change = %+v", im)
	}
}

func TestImpactOwnItems(t *testing.T) {
	base := library(t)
	cur := base.clone()
	cur[m+"members/member.md"] = def("std/Concept", "Member", "A person with a card.")
	im := impact(t, base, cur, nil, nil, false)

	if got := strings.Join(packages(im), ","); got != ",catalog,members,members/loans" {
		t.Fatalf("working = %q", got)
	}
	members, _ := checklist(im, "members")
	if ids(members.Own) != "members/member" || members.Own[0].Reasons[0] != Modified || len(members.Inherited) != 0 {
		t.Fatalf("members = %+v", members)
	}
	catalog, _ := checklist(im, "catalog")
	if ids(catalog.Own) != "catalog/book" || !strings.Contains(catalog.Own[0].Reasons[0], "links [[members/member]], which changed") {
		t.Fatalf("catalog = %+v", catalog)
	}
	loans, _ := checklist(im, "members/loans")
	if ids(loans.Own) != "members/loans/loan" || len(loans.Inherited) != 1 || loans.Inherited[0].Definition != "members/member" {
		t.Fatalf("loans = %+v (a changed parent Definition is inherited)", loans)
	}
	root, _ := checklist(im, "")
	if ids(root.Own) != "" || !root.Empty() {
		t.Fatalf("root = %+v (Root only integrates)", root)
	}
	var files []string
	for _, f := range im.ReachedFiles {
		files = append(files, f.File+"@"+f.Owner)
	}
	if got := strings.Join(files, " "); got != "README.md@ src/catalog/book.go@catalog src/members/loans/loan.go@members/loans src/members/member.go@members" {
		t.Fatalf("reached files = %s", got)
	}
}

func TestImpactInheritedRootRule(t *testing.T) {
	base := library(t)
	cur := base.clone()
	cur[m+"loan-limit.md"] = def("std/Rule", "Loan limit", "At most `3` open loans.")
	im := impact(t, base, cur, nil, nil, false)
	if got := strings.Join(packages(im), ","); got != ",catalog,members,members/loans" {
		t.Fatalf("a changed Root Rule must reach every Manager: %q", got)
	}
	for _, pkg := range []string{"catalog", "members", "members/loans"} {
		c, _ := checklist(im, pkg)
		if ids(c.Inherited) != "loan-limit" || len(c.Own) != 0 {
			t.Fatalf("%s = %+v", pkg, c)
		}
	}
	root, _ := checklist(im, "")
	if ids(root.Own) != "loan-limit" {
		t.Fatalf("root = %+v", root)
	}
}

func TestImpactFileReachAndNoInheritance(t *testing.T) {
	base := library(t)
	cur := base.clone()
	cur["README.md"] = "# Library, edited\n"
	im := impact(t, base, cur, []Change{{Path: "README.md", Change: Modified}}, nil, false)
	root, _ := checklist(im, "")
	if ids(root.Own) != "project" || !strings.Contains(root.Own[0].Reasons[0], "realized by README.md") {
		t.Fatalf("root = %+v", root)
	}
	members, _ := checklist(im, "members")
	if ids(members.Own) != "members/member" || len(members.Inherited) != 0 {
		t.Fatalf("members = %+v", members)
	}
	if _, ok := checklist(im, "members/loans"); ok {
		t.Fatal("a Root Definition reached only through a file must not be inherited")
	}
	if _, ok := checklist(im, "catalog"); ok {
		t.Fatal("catalog is not reached")
	}
}

func TestImpactRemovedFileUsesBaseRegistry(t *testing.T) {
	base := library(t)
	cur := base.clone()
	delete(cur, "src/catalog/book.go")
	im := impact(t, base, cur, []Change{{Path: "src/catalog/book.go", Change: Removed}}, nil, false)
	catalog, _ := checklist(im, "catalog")
	if ids(catalog.Own) != "catalog/book,catalog/shelf" {
		t.Fatalf("catalog own = %+v", catalog.Own)
	}
	if len(catalog.Stale) != 1 || catalog.Stale[0].Code != "stale.file" {
		t.Fatalf("catalog stale = %+v", catalog.Stale)
	}
}

func TestImpactKindChangeReachesItsDefinitions(t *testing.T) {
	base := library(t)
	base[m+"catalog/Medium.md"] = def("Kind", "Medium", "Something that can be lent.")
	base[m+"catalog/dvd.md"] = def("catalog/Medium", "DVD", "A disc.")
	base[project.RegistryDir+"/catalog.md"] += "| src/catalog/dvd.go | [[catalog/dvd]] [[catalog/Medium]] | DVD |\n"
	base["src/catalog/dvd.go"] = "package catalog\n"
	cur := base.clone()
	cur[m+"catalog/Medium.md"] = def("Kind", "Medium", "Something that can be lent, with a loan period.")
	im := impact(t, base, cur, nil, nil, false)
	catalog, _ := checklist(im, "catalog")
	if ids(catalog.Own) != "catalog/Medium,catalog/dvd" || !strings.Contains(catalog.Own[1].Reasons[0], "is of Kind [[catalog/Medium]]") {
		t.Fatalf("catalog = %+v", catalog.Own)
	}
}

func TestImpactFindings(t *testing.T) {
	base := library(t)
	findings := []Finding{
		{Definition: "members/loans/loan", Observed: "a fourth loan", Expected: "refused", Line: 3},
		{Definition: "members/gone", Observed: "x", Expected: "y", Line: 9},
	}
	im := impact(t, base, base.clone(), nil, findings, false)
	loans, _ := checklist(im, "members/loans")
	if len(loans.Findings) != 1 || loans.Findings[0].Definition != "members/loans/loan" {
		t.Fatalf("loans = %+v", loans)
	}
	if got := strings.Join(packages(im), ","); got != ",members,members/loans" {
		t.Fatalf("working = %q", got)
	}
	if len(im.Notes) != 1 || !strings.Contains(im.Notes[0], "[[members/gone]]") {
		t.Fatalf("notes = %v", im.Notes)
	}
}

func TestImpactStaleItems(t *testing.T) {
	base := library(t)
	cur := base.clone()
	delete(cur, m+"catalog/shelf.md")
	im := impact(t, base, cur, nil, nil, false)
	catalog, _ := checklist(im, "catalog")
	if len(catalog.Stale) != 1 || catalog.Stale[0].Target != "catalog/shelf" || catalog.Stale[0].File != "src/catalog/book.go" {
		t.Fatalf("catalog stale = %+v", catalog.Stale)
	}
	if len(im.Definitions) != 1 || im.Definitions[0] != (Change{Path: "catalog/shelf", Change: Removed}) {
		t.Fatalf("changed = %+v", im.Definitions)
	}
}

func TestImpactFull(t *testing.T) {
	base := library(t)
	im := impact(t, base, base.clone(), nil, nil, true)
	if got := strings.Join(packages(im), ","); got != ",catalog,members,members/loans" {
		t.Fatalf("working = %q", got)
	}
	loans, _ := checklist(im, "members/loans")
	if ids(loans.Own) != "members/loans/loan" || ids(loans.Inherited) != "members/member,loan-limit,project" {
		t.Fatalf("loans = %+v", loans)
	}
}

func TestImpactLibraryUpgrade(t *testing.T) {
	// A new binary whose std differs from what the base commit locked: both
	// states read std from this binary, so only the lock shows the change.
	base := library(t)
	base[project.LockFile] = strings.Replace(base[project.LockFile], "digest: sha256:", "digest: sha256:0", 1)
	bp, err := project.Load(base)
	if err != nil {
		t.Fatal(err)
	}
	im := ComputeImpact(ImpactInput{Base: bp, Current: loadRepo(t, library(t))})
	if len(im.Definitions) != 5 || im.Definitions[0] != (Change{Path: "std/Concept", Change: Modified}) {
		t.Fatalf("changed = %+v", im.Definitions)
	}
	root, _ := checklist(im, "")
	if ids(root.Own) != "loan-limit,project" || !strings.Contains(root.Own[0].Reasons[0], "is of Kind [[std/Rule]]") {
		t.Fatalf("root = %+v", root)
	}
	if got := strings.Join(packages(im), ","); got != ",catalog,members,members/loans" {
		t.Fatalf("working = %q", got)
	}
}

func TestImpactLocalLibraryComparesKindByKind(t *testing.T) {
	withWeb := func(page string) memRepo {
		r := library(t)
		r[project.ManifestFile] = "libraries:\n  std: 1.0.0\n  web: 0.1.0\n"
		r[project.LibraryDir+"/web/Page.md"] = page
		r[project.LibraryDir+"/web/Form.md"] = "---\nkind: Kind\n---\n# Form\n"
		r[m+"catalog/search.md"] = def("web/Form", "Search", "Finds a [[catalog/book]].")
		r[project.RegistryDir+"/catalog.md"] += "| src/catalog/search.go | [[catalog/search]] | Search form |\n"
		r["src/catalog/search.go"] = "package catalog\n"
		p, err := project.Load(r)
		if err != nil {
			t.Fatal(err)
		}
		r[project.LockFile] = string(model.RenderLock(p.Libraries))
		return r
	}
	base := withWeb("---\nkind: Kind\n---\n# Page\n")
	cur := withWeb("---\nkind: Kind\n---\n# Page\n\nNow with a title bar.\n")
	im := impact(t, base, cur, nil, nil, false)
	if len(im.Definitions) != 1 || im.Definitions[0] != (Change{Path: "web/Page", Change: Modified}) {
		t.Fatalf("changed = %+v; only the edited Kind changed", im.Definitions)
	}
	if len(im.Packages) != 0 {
		t.Fatalf("no Definition is of Kind Page, yet %v work", im.Packages)
	}
}

func TestFindingsEmptyHeadingIsAnErrorNotACrash(t *testing.T) {
	_, ds := ParseFindings("f.md", []byte("# Findings\n\n##\n\nObserved: x\n"))
	expectWarning(t, ds, "findings.syntax", "exactly one link")
}

func packages(im Impact) []string { return im.Packages }

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
	g := BuildGraph(loadRepo(t, library(t)))
	if len(g.Edges) != 1 || g.Edges[0].From != "catalog" || g.Edges[0].To != "members" {
		t.Fatalf("edges = %+v", g.Edges)
	}
	catalog, members := g.node("catalog"), g.node("members")
	if catalog.Efferent != 1 || catalog.Instability != 1 || members.Afferent != 1 || members.Instability != 0 {
		t.Fatalf("catalog %+v members %+v", catalog, members)
	}
}

func lintCodes(t *testing.T, r memRepo) model.Diagnostics {
	t.Helper()
	return Lint(loadRepo(t, r))
}

func expectWarning(t *testing.T, ds model.Diagnostics, code, fragment string) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code && strings.Contains(d.String(), fragment) {
			return
		}
	}
	t.Fatalf("no %s containing %q in:\n%v", code, fragment, ds)
}

func TestLintCleanFixtureWarnsOnlyUnused(t *testing.T) {
	ds := lintCodes(t, library(t))
	if len(ds) != 1 || ds[0].Code != "lint.unused" || !strings.Contains(ds[0].Message, "loan-limit") {
		t.Fatalf("lint = %v", ds)
	}
}

func TestLintCoupling(t *testing.T) {
	r := library(t)
	var links []string
	for i := 0; i < MaxEfferent+1; i++ {
		name := fmt.Sprintf("p%d", i)
		r[m+name+"/x.md"] = def("std/Rule", "X", "")
		links = append(links, "[["+name+"/x]]")
	}
	r[m+"hub/h.md"] = def("std/Rule", "Hub", strings.Join(links, " "))
	expectWarning(t, lintCodes(t, r), "lint.coupling", "hub depends on 6 Packages")
}

func TestLintUnstableDependency(t *testing.T) {
	r := library(t)
	// shelf-tools depends on catalog, which also depends on members: catalog's
	// instability drops below shelf-tools'... and members depends on extra.
	r[m+"extra/e.md"] = def("std/Rule", "E", "")
	r[m+"members/member.md"] = def("std/Concept", "Member", "Uses [[extra/e]].")
	r[m+"reports/r.md"] = def("std/Rule", "R", "Counts [[members/member]].")
	r[m+"audits/a.md"] = def("std/Rule", "A", "Checks [[members/member]].")
	// members: Ca 3 (catalog, reports, audits), Ce 1 -> 0.25; extra: Ca 1, Ce 0 -> 0.
	// Make extra unstable: it depends on two more Packages.
	r[m+"x1/a.md"] = def("std/Rule", "X1", "")
	r[m+"x2/a.md"] = def("std/Rule", "X2", "")
	r[m+"extra/e.md"] = def("std/Rule", "E", "[[x1/a]] [[x2/a]]")
	expectWarning(t, lintCodes(t, r), "lint.unstable_dependency", "members (instability 0.25) depends on the less stable extra (instability 0.67)")
}

func TestLintSizes(t *testing.T) {
	r := library(t)
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]]. "+strings.Repeat("word ", MaxDefinitionWords))
	for i := 0; i <= MaxPackageSize; i++ {
		r[m+fmt.Sprintf("big/d%02d.md", i)] = def("std/Rule", "D", "")
	}
	ds := lintCodes(t, r)
	expectWarning(t, ds, "lint.definition_size", "catalog/book.md: lint.definition_size")
	expectWarning(t, ds, "lint.package_size", "big has 26 Definitions")
}

func TestLintThinPackageAndReach(t *testing.T) {
	r := library(t)
	r[m+"wrapper/inner/i.md"] = def("std/Rule", "I", "")
	ds := lintCodes(t, r)
	expectWarning(t, ds, "lint.thin_package", "wrapper has no Definitions of its own and only one child, wrapper/inner")

	r = library(t)
	r[m+"orders/o.md"] = def("std/Rule", "O", "[[members/member]]")
	r[m+"fees/f.md"] = def("std/Rule", "F", "[[members/member]]")
	// members/member: its own Package, loans below it (inherited), catalog,
	// orders and fees through links, and Root through the README: 6 of 6.
	expectWarning(t, lintCodes(t, r), "lint.reach", "a change to members/member reaches 6 of 6 Managers")
}

func TestLintNoPaths(t *testing.T) {
	r := library(t)
	r[m+"orders/o.md"] = def("std/Rule", "O", "[[members/member]]")
	// catalog's page names no paths, so Root owns its file.
	r[project.RegistryDir+"/catalog.md"] = table
	r[project.RootPage] += "| src/catalog/book.go | [[catalog/book]] [[catalog/shelf]] | Types |\n"
	ds := lintCodes(t, r)
	expectWarning(t, ds, "lint.no_paths", project.RegistryDir+"/orders.md: lint.no_paths: orders has no registry page")
	expectWarning(t, ds, "lint.no_paths", project.RegistryDir+"/catalog.md: lint.no_paths: catalog claims no path pattern")
	for _, d := range ds {
		if d.Code == "lint.no_paths" && !strings.Contains(d.Message, "orders") && !strings.Contains(d.Message, "catalog") {
			t.Fatalf("unexpected %v", d)
		}
	}
}

func TestLintStale(t *testing.T) {
	r := library(t)
	delete(r, m+"catalog/shelf.md")
	delete(r, "src/members/loans/loan.go")
	delete(r, m+"members/loans/loan.md")
	r[project.RegistryDir+"/gone.md"] = table + "| src/gone.go | [[gone/g]] | old |\n"
	ds := lintCodes(t, r)
	expectWarning(t, ds, "lint.stale", "src/catalog/book.go links [[catalog/shelf]], which no longer exists; catalog must realign")
	expectWarning(t, ds, "lint.stale", "the Package of this page no longer exists")
	expectWarning(t, ds, "lint.stale", "src/members/loans/loan.go no longer exists")
}

func TestParseFindings(t *testing.T) {
	data := "# Findings\n\n## [[members/member]]\n\nObserved: a fourth loan.\n\n**Expected:** refused with `limit`.\n\n## [[catalog/book|Book]] again\n\nObserved: x\n\nExpected: y\n"
	list, ds := ParseFindings("f.md", []byte(data))
	if len(ds) != 0 || len(list) != 2 {
		t.Fatalf("list %+v ds %v", list, ds)
	}
	if list[0] != (Finding{Definition: "members/member", Observed: "a fourth loan.", Expected: "refused with limit.", Line: 3}) {
		t.Fatalf("first = %+v", list[0])
	}
	_, ds = ParseFindings("f.md", []byte("## No link\n\nObserved: x\n\nExpected: y\n\n## [[a]] [[b]]\n\n## [[c]]\n\nObserved: only\n"))
	expectWarning(t, ds, "findings.syntax", "f.md:1: findings.syntax: a Finding's heading holds exactly one link")
	expectWarning(t, ds, "findings.syntax", "f.md:7")
	expectWarning(t, ds, "findings.syntax", "the Finding for c needs")
}
