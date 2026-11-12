package analysis

import (
	"github.com/Glacius-Labs/Markitect/internal/project"
	"reflect"
	"strings"
	"testing"
)

// list shows each Package with its Definitions, and with deps what it depends
// on and through which links.
func TestListPackages(t *testing.T) {
	p := loadRepo(t, library(t))
	pkgs := ListPackages(p, true)
	byName := map[string]ListedPackage{}
	for _, pkg := range pkgs {
		byName[pkg.Name] = pkg
	}
	catalog := byName["catalog"]
	if catalog.Parent != "Root" || len(catalog.Definitions) != 3 || catalog.Deps == nil || len(catalog.Deps.DependsOn) != 1 {
		t.Fatalf("catalog: %+v", catalog)
	}
	dep := catalog.Deps.DependsOn[0]
	if dep.Package != "members" || !reflect.DeepEqual(dep.Uses[0], Use{From: "catalog/book", Target: "members/member", Via: "link", File: dep.Uses[0].File, Line: dep.Uses[0].Line}) {
		t.Fatalf("catalog depends on %+v", dep)
	}
	if book := catalog.Definitions[0]; book.ID != "catalog/book" || !reflect.DeepEqual(book.Links, []string{"members/member"}) {
		t.Fatalf("book: %+v", book)
	}
	for _, pkg := range ListPackages(p, false) {
		if pkg.Deps != nil {
			t.Fatalf("%s has deps without --deps", pkg.Name)
		}
		for _, d := range pkg.Definitions {
			if d.Links != nil {
				t.Fatalf("%s has links without --deps", d.ID)
			}
		}
	}
}

// doc shows a Definition's text without its front matter, who links to it and
// the files that realize it apart, and a Package's page, paths and files.
func TestDocViews(t *testing.T) {
	p := loadRepo(t, library(t))
	dd := DocDefinitionOf(p, p.Spec.Definitions["members/member"])
	if dd.Text != "# Member\n\nA person.\n" || dd.Package != "members" || dd.Record != nil {
		t.Fatalf("member: %+v", dd)
	}
	var from, files []string
	for _, r := range dd.LinkedFrom {
		from = append(from, r.From)
	}
	for _, r := range dd.RealizedBy {
		files = append(files, r.File)
	}
	if !reflect.DeepEqual(from, []string{"catalog/book", "members/loans/loan"}) || !reflect.DeepEqual(files, []string{"README.md", "src/members/member.go"}) {
		t.Fatalf("linked from %v, realized by %v", from, files)
	}
	// A thing is shown whole: its parts with their text, a folder's parts
	// together, in the order the thing links them and then by ID.
	r := library(t)
	r[m+"feature.md"] = def("Kind", "Feature", "One thing a user can do.")
	r[m+"catalog/operations/lend/lend.md"] = def("feature", "Lend", "First [[catalog/operations/lend/conditions/free]].")
	r[m+"catalog/operations/lend/conditions/free.md"] = def("std/Rule", "Free", "Nobody holds it.")
	r[m+"catalog/operations/lend/conditions/an-earlier-name.md"] = def("std/Rule", "An earlier name", "By name it comes first.")
	r[m+"catalog/operations/lend/outcomes/lent.md"] = def("std/Rule", "Lent", "Somebody holds it.")
	q := loadRepo(t, r)
	lend := DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/lend/lend"])
	var parts []string
	for _, part := range lend.Parts {
		parts = append(parts, part.Group+" "+part.Title+": "+part.Text)
	}
	if want := []string{"conditions Free: # Free\n\nNobody holds it.\n", "conditions An earlier name: # An earlier name\n\nBy name it comes first.\n", "outcomes Lent: # Lent\n\nSomebody holds it.\n"}; !reflect.DeepEqual(parts, want) || lend.Whole != "" {
		t.Fatalf("parts of lend: %q", parts)
	}
	if free := DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/lend/conditions/free"]); free.Whole != "catalog/operations/lend/lend" || len(free.Parts) != 0 {
		t.Fatalf("free: %+v", free)
	}
	// Folders come in the order the thing links their parts, and a part that
	// has parts of its own sits with its siblings, its parts after it.
	r[m+"catalog/operations/lend/lend.md"] = def("feature", "Lend", "Afterwards [[catalog/operations/lend/outcomes/lent]]; first [[catalog/operations/lend/conditions/free]].")
	r[m+"catalog/operations/lend/outcomes/due/due.md"] = def("std/Rule", "Due", "It is due.")
	r[m+"catalog/operations/lend/outcomes/due/after.md"] = def("std/Rule", "After", "After `14` days.")
	q = loadRepo(t, r)
	parts = nil
	for _, part := range DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/lend/lend"]).Parts {
		parts = append(parts, part.Group+" "+part.Title)
	}
	// What lies directly in the thing's folder comes first, under no name.
	r[m+"catalog/operations/lend/note.md"] = def("std/Rule", "Note", "A note.")
	q = loadRepo(t, r)
	parts = nil
	for _, part := range DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/lend/lend"]).Parts {
		parts = append(parts, part.Group+" "+part.Title)
	}
	if want := []string{" Note", "outcomes Lent", "outcomes Due", "outcomes/due After", "conditions Free", "conditions An earlier name"}; !reflect.DeepEqual(parts, want) {
		t.Fatalf("order of folders and parts: %q", parts)
	}
	// A part shows no parts of the thing it belongs to: neither its whole nor
	// a part beside it, though it links both.
	r[m+"catalog/operations/lend/outcomes/due/after.md"] = def("std/Rule", "After", "After `14` days, see [[catalog/operations/lend/outcomes/due]] and [[catalog/operations/lend/conditions/free]].")
	q = loadRepo(t, r)
	if after := DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/lend/outcomes/due/after"]); len(after.Parts) != 0 || after.Whole != "catalog/operations/lend/outcomes/due/due" {
		t.Fatalf("a part shows its own thing: %+v", after.Parts)
	}
	delete(r, m+"catalog/operations/lend/note.md")
	delete(r, m+"catalog/operations/lend/outcomes/due/due.md")
	delete(r, m+"catalog/operations/lend/outcomes/due/after.md")
	// A part of another thing that the Definition links is read with it, in
	// its place in the order and with the thing it is a part of; and the text
	// is shown with each link by its label or by the title of what it names.
	r[m+"catalog/book/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]].")
	r[m+"catalog/book/conditions/exists.md"] = def("std/Condition", "Book exists", "A [[catalog/book]] with this `isbn` exists.")
	r[m+"catalog/operations/lend/lend.md"] = def("feature", "Lend", "Checked: [[catalog/book/conditions/exists]], then [[catalog/operations/lend/conditions/free|that it is free]]; see `[[catalog/book]]`.\n\n```\n[[catalog/book]]\n```\n\nIn [[members]], a `[[catalog/book]]` is a [[ catalog/book ]].\n\n~~~\n[[catalog/book]]\n~~~")
	delete(r, m+"catalog/book.md")
	r[project.RegistryDir+"/catalog.md"] = "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] [[catalog/shelf]] [[catalog/catalog]] | Types |\n"
	q = loadRepo(t, r)
	lend = DocDefinitionOf(q, q.Spec.Definitions["catalog/operations/lend/lend"])
	parts = nil
	for _, part := range lend.Parts {
		parts = append(parts, part.Group+" "+part.Title+" of "+part.Of+": "+part.Shown)
	}
	if want := []string{
		"conditions Book exists of catalog/book/book: # Book exists\n\nA Book with this `isbn` exists.\n",
		"conditions Free of : # Free\n\nNobody holds it.\n",
		"conditions An earlier name of : # An earlier name\n\nBy name it comes first.\n",
		"outcomes Lent of : # Lent\n\nSomebody holds it.\n",
	}; !reflect.DeepEqual(parts, want) {
		t.Fatalf("parts of lend with a part of another thing:\n%q", parts)
	}
	if want := "# Lend\n\nChecked: Book exists, then that it is free; see `[[catalog/book]]`.\n\n```\n[[catalog/book]]\n```\n\nIn Members, a `[[catalog/book]]` is a Book.\n\n~~~\n[[catalog/book]]\n~~~\n"; lend.Shown != want || !strings.Contains(lend.Text, "[[catalog/book/conditions/exists]]") {
		t.Fatalf("shown:\n%q\ntext:\n%q", lend.Shown, lend.Text)
	}
	dp := DocPackageOf(p, p.Spec.Packages[p.Spec.Definitions["members/member"].Package])
	if dp.Anchor != "members/members" || dp.Text != "# Members\n\nWho may borrow.\n" {
		t.Fatalf("anchor of members: %+v", dp)
	}
	if root := DocPackageOf(p, p.Spec.Packages[""]); root.Anchor != "" || root.Text != "" {
		t.Fatalf("Root has an anchor: %+v", root)
	}
	if dp.Name != "members" || dp.Parent != "Root" || dp.Page == "" || !reflect.DeepEqual(dp.Paths, []string{"src/members/**"}) || !reflect.DeepEqual(dp.Files, []string{"src/members/member.go"}) {
		t.Fatalf("members: %+v", dp)
	}
}
