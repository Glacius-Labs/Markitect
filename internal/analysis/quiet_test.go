package analysis

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func has(ds []string, code, fragment string) bool {
	for _, d := range ds {
		if strings.Contains(d, code) && strings.Contains(d, fragment) {
			return true
		}
	}
	return false
}

func warnings(t *testing.T, r memRepo) []string {
	t.Helper()
	var out []string
	for _, d := range vetCodes(t, r) {
		out = append(out, d.String())
	}
	return out
}

// Root holds what binds the whole project. In a spec with Packages it is not
// measured by its size; a spec without any Package is, since Root is then all
// of it.
func TestVetMeasuresRootOnlyInASpecWithoutPackages(t *testing.T) {
	r := library(t)
	for i := 0; i < MaxPackageSize+5; i++ {
		r[m+fmt.Sprintf("rules/r%02d.md", i)] = def("std/Rule", fmt.Sprintf("R%02d", i), "")
	}
	if ds := warnings(t, r); has(ds, "vet.package_size", "Root") {
		t.Fatalf("Root of a spec with Packages:\n%v", ds)
	}
	flat := memRepo{project.ManifestFile: r[project.ManifestFile], project.LockFile: r[project.LockFile], project.RootPage: table}
	for i := 0; i < MaxPackageSize+1; i++ {
		flat[m+fmt.Sprintf("r%02d.md", i)] = def("std/Rule", fmt.Sprintf("R%02d", i), "")
	}
	if ds := warnings(t, flat); !has(ds, "vet.package_size", "Root has 26 Definitions of its own") {
		t.Fatalf("Root of a spec without Packages:\n%v", ds)
	}
}

// Until something is built the registry is empty and the builder writes its
// pages, so no Package is told that it owns no file.
func TestVetAsksForPathsOnlyOnceSomethingIsBuilt(t *testing.T) {
	r := library(t)
	pkg(r, "orders")
	r[m+"orders/o.md"] = def("std/Rule", "O", "[[members/member]]")
	if ds := warnings(t, r); !has(ds, "vet.no_paths", "orders") {
		t.Fatalf("a Package without a page in a built project:\n%v", ds)
	}
	fresh := memRepo{project.ManifestFile: r[project.ManifestFile], project.LockFile: r[project.LockFile], project.RootPage: table}
	for file, text := range r {
		if strings.HasPrefix(file, m) {
			fresh[file] = text
		}
	}
	p, err := project.Load(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if errs := p.SpecErrors(); len(errs) != 0 {
		t.Fatalf("fixture does not compile:\n%v", errs)
	}
	for _, d := range Vet(p, History{}).Warnings {
		if d.Code == "vet.no_paths" {
			t.Fatalf("nothing is built: %v", d)
		}
	}
}

// An unlinked mention names what a Definition relies on without linking it.
// A part's name is a word of its whole and is not reported; what one part of
// a thing links is linked for all of it; and of several Definitions with one
// name, a mention names the nearest.
func TestVetUnlinkedMentionsOfThings(t *testing.T) {
	r := library(t)
	// A Concept with a part named Title, and a page named like the Concept.
	r[m+"catalog/book/book.md"] = def("std/Concept", "Book", "A book.")
	delete(r, m+"catalog/book.md")
	r[m+"catalog/book/attributes/title.md"] = def("std/Attribute", "Title", "`title` is its title.")
	pkg(r, "site")
	r[m+"site/pages/book/book.md"] = def("std/Rule", "Book", "The page of a [[catalog/book]].")
	// A thing whose whole links the member and whose part names the member.
	r[m+"catalog/rules/lend/lend.md"] = def("std/Rule", "Lend", "To a [[members/member]].")
	r[m+"catalog/rules/lend/twice.md"] = def("std/Rule", "Twice", "A member borrows a book with a title twice.")
	// A Definition of another Package that names the book and links nothing.
	r[m+"members/note.md"] = def("std/Rule", "Note", "A book has a title, and a shelf holds it.")
	p := loadRepo(t, r)
	got := map[string][]string{}
	for _, n := range Vet(p, History{}).Notes {
		if n.Code == "vet.unlinked_mention" {
			got[n.File] = append(got[n.File], n.Targets...)
		}
	}
	// twice: the member is linked by its whole; the book is the one of its own
	// Package, not the page; the title is a part and is not reported.
	if want := []string{"catalog/book/book"}; fmt.Sprint(got[m+"catalog/rules/lend/twice.md"]) != fmt.Sprint(want) {
		t.Fatalf("twice names %v", got[m+"catalog/rules/lend/twice.md"])
	}
	// note, in another Package: of the two books the first by ID, and the shelf.
	if want := []string{"catalog/book/book", "catalog/shelf"}; fmt.Sprint(sorted(got[m+"members/note.md"])) != fmt.Sprint(want) {
		t.Fatalf("note names %v", got[m+"members/note.md"])
	}
}
