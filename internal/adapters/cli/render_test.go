package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// Plain vet counts its notes in one line: the unlinked mentions within the Package
// structure and their Definitions, those against or beyond it, and the others.
func TestNotesLine(t *testing.T) {
	mention := func(structure string, targets ...string) analysis.Note {
		return analysis.Note{Diagnostic: model.Diagnostic{Code: "vet.unlinked_mention"}, Targets: targets, Structure: structure}
	}
	other := analysis.Note{Diagnostic: model.Diagnostic{Code: "vet.statements"}}
	for _, c := range []struct {
		notes []analysis.Note
		want  string
	}{
		{[]analysis.Note{mention("", "a", "b"), mention("", "c"), mention(analysis.StructureNew, "d"), mention(analysis.StructureAgainst, "e", "f"), other},
			"notes: 3 unlinked mentions in 2 Definitions, 3 more against or beyond the Package structure, 1 other note; markitect vet --strict lists them"},
		{[]analysis.Note{mention("", "a")}, "notes: 1 unlinked mention in 1 Definition; markitect vet --strict lists them"},
		{[]analysis.Note{mention(analysis.StructureAgainst, "e")}, "notes: 1 unlinked mention against or beyond the Package structure; markitect vet --strict lists them"},
		{[]analysis.Note{other, other}, "notes: 2; markitect vet --strict lists them"},
	} {
		if got := notesLine(c.notes); got != c.want {
			t.Fatalf("got  %q\nwant %q", got, c.want)
		}
	}
}

// doc of a Package shows its public Definitions, what it publishes, apart from
// its internal ones, and leaves out a heading with nothing under it.
func TestRenderDocPackageSeparatesPublicFromInternal(t *testing.T) {
	got := renderDocPackage(docPackage{Name: "members", Parent: "Root", Definitions: []listedDefinition{
		{ID: "members/limit", Title: "Limit", Kind: "std/Rule", Internal: true},
		{ID: "members/member", Title: "Member", Kind: "std/Concept"},
	}})
	want := "Package members, in Root\n" +
		"\nPublic Definitions:\n  members/member: Member (std/Concept)\n" +
		"\nInternal Definitions:\n  members/limit: Limit (std/Rule)\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	got = renderDocPackage(docPackage{Name: "Root", Definitions: []listedDefinition{{ID: "project", Title: "Project", Kind: "std/Project"}}})
	if want := "Package Root\n\nPublic Definitions:\n  project: Project (std/Project)\n"; got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	got = renderDocPackage(docPackage{Name: "loans", Parent: "members", Definitions: []listedDefinition{{ID: "members/loans/count", Title: "Count", Kind: "std/Concept", Internal: true}}})
	if want := "Package loans, in members\n\nInternal Definitions:\n  members/loans/count: Count (std/Concept)\n"; got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

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
