package analysis

import (
	"reflect"
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
	if catalog.Parent != "Root" || len(catalog.Definitions) != 2 || catalog.Deps == nil || len(catalog.Deps.DependsOn) != 1 {
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
	dp := DocPackageOf(p, p.Spec.Packages[p.Spec.Definitions["members/member"].Package])
	if dp.Name != "members" || dp.Parent != "Root" || dp.Page == "" || !reflect.DeepEqual(dp.Paths, []string{"src/members/**"}) || !reflect.DeepEqual(dp.Files, []string{"src/members/member.go"}) {
		t.Fatalf("members: %+v", dp)
	}
}
