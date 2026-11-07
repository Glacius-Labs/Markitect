package cli

import "testing"

// list --deps shows each Package's coupling, abstractness and distance, and what
// it depends on, with the links that make the dependency, and each Definition's
// links.
func TestRenderListWithDeps(t *testing.T) {
	got := renderList([]listedPackage{
		{Name: "Root", Definitions: []listedDefinition{{ID: "loan-limit", Title: "Loan limit", Kind: "std/Rule"}}, Deps: &packageDeps{Afferent: 1, Abstractness: 0.5, Distance: 0.5}},
		{Name: "catalog", Parent: "Root", Definitions: []listedDefinition{{ID: "catalog/book", Title: "Book", Kind: "std/Concept", Links: []string{"members/member"}}}, Deps: &packageDeps{Efferent: 1, Instability: 1, DependsOn: []dependency{
			{Package: "members", Uses: []use{{From: "catalog/book", Target: "members/member", Via: "link"}, {From: "catalog/dvd", Target: "members/Medium", Via: "kind"}}},
		}}},
	})
	want := "Root (Ca 1, Ce 0, instability 0.00, abstractness 0.50, distance 0.50)\n" +
		"  loan-limit: Loan limit (std/Rule)\n" +
		"catalog (Ca 0, Ce 1, instability 1.00, abstractness 0.00, distance 0.00)\n" +
		"  depends on members: catalog/book links members/member; catalog/dvd is of Kind members/Medium\n" +
		"  catalog/book: Book (std/Concept)\n" +
		"    links members/member\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
