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
	g := BuildGraph(loadRepo(t, library(t)))
	if len(g.Edges) != 1 || g.Edges[0].From != "catalog" || g.Edges[0].To != "members" {
		t.Fatalf("edges = %+v", g.Edges)
	}
	catalog, members := g.node("catalog"), g.node("members")
	if catalog.Efferent != 1 || catalog.Instability != 1 || members.Afferent != 1 || members.Instability != 0 {
		t.Fatalf("catalog %+v members %+v", catalog, members)
	}
}

func vetCodes(t *testing.T, r memRepo) model.Diagnostics {
	t.Helper()
	return Vet(loadRepo(t, r))
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

func TestVetCoupling(t *testing.T) {
	r := library(t)
	var links []string
	for i := 0; i < MaxEfferent+1; i++ {
		name := fmt.Sprintf("p%d", i)
		r[m+name+"/x.md"] = def("std/Rule", "X", "")
		links = append(links, "[["+name+"/x]]")
	}
	r[m+"hub/h.md"] = def("std/Rule", "Hub", strings.Join(links, " "))
	expectWarning(t, vetCodes(t, r), "vet.coupling", "hub depends on 6 Packages")
}

func TestVetUnstableDependency(t *testing.T) {
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
	expectWarning(t, vetCodes(t, r), "vet.unstable_dependency", "members (instability 0.25) depends on the less stable extra (instability 0.67)")
}

func TestVetSizes(t *testing.T) {
	r := library(t)
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]]. "+strings.Repeat("word ", MaxDefinitionWords))
	for i := 0; i <= MaxPackageSize; i++ {
		r[m+fmt.Sprintf("big/d%02d.md", i)] = def("std/Rule", "D", "")
	}
	ds := vetCodes(t, r)
	expectWarning(t, ds, "vet.definition_size", "catalog/book.md: vet.definition_size")
	expectWarning(t, ds, "vet.package_size", "big has 26 Definitions")
}

func TestVetThinPackageAndReach(t *testing.T) {
	r := library(t)
	r[m+"wrapper/inner/i.md"] = def("std/Rule", "I", "")
	ds := vetCodes(t, r)
	expectWarning(t, ds, "vet.thin_package", "wrapper has no Definitions of its own and only one child, wrapper/inner")

	r = library(t)
	r[m+"orders/o.md"] = def("std/Rule", "O", "[[members/member]]")
	r[m+"fees/f.md"] = def("std/Rule", "F", "[[members/member]]")
	// members/member: its own Package, and loans, catalog, orders and fees, whose
	// Definitions link to it: 5 of 6.
	expectWarning(t, vetCodes(t, r), "vet.reach", "a change to members/member puts Definitions of 5 of 6 Packages back on the list")
}

func TestVetNoPaths(t *testing.T) {
	r := library(t)
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

// A change reports everything it could reach, with distance and reason, and only
// the first step is on a checklist.
// A registry row that drops a link still reaches the Definition it dropped, on a
// changed file and on an unchanged one.
// Rows that reach one Definition give their reasons in one order, so that the
// same change gives the same impact and the same prompt.
