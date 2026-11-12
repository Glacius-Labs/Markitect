package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// Plain vet counts its notes in one line: the unlinked mentions within the Package
// structure and their Definitions, those against or beyond it, and the others.
func TestNotesLine(t *testing.T) {
	mention := func(structure string, targets ...string) analysis.Note {
		return analysis.Note{Diagnostic: spec.Diagnostic{Code: "vet.unlinked_mention"}, Targets: targets, Structure: structure}
	}
	other := analysis.Note{Diagnostic: spec.Diagnostic{Code: "vet.statements"}}
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

// doc of a thing prints its parts below its text, each folder's under the
// folder's name, a part's title before its text; and a part names its whole.
func TestRenderDocDefinitionWithParts(t *testing.T) {
	got := renderDocDefinition(analysis.DocDefinition{
		ListedDefinition: analysis.ListedDefinition{ID: "catalog/operations/lend/lend", Title: "Lend", Kind: "std/Operation", File: "lend.md"},
		Package:          "catalog", Text: "# Lend\n\nA copy goes home.\n",
		Parts: []analysis.DocPart{
			{ID: "catalog/operations/lend/conditions/free", Title: "Free", Group: "conditions", Text: "# Free\n\nNobody holds it.\n\n**Refusal:** `held`.\n"},
			{ID: "catalog/book/conditions/exists", Title: "Book exists", Group: "conditions", Of: "catalog/book/book", Text: "# Book exists\n\nA [[catalog/book]] exists.\n", Shown: "# Book exists\n\nA Book exists.\n"},
			{ID: "catalog/operations/lend/outcomes/lent", Title: "Lent", Group: "outcomes", Text: "# Lent\n\nSomebody holds it.\n"},
		},
	})
	want := "catalog/operations/lend/lend: Lend (std/Operation)\nPackage catalog, lend.md\n\n# Lend\n\nA copy goes home.\n" +
		"\nconditions:\n\n  Free (catalog/operations/lend/conditions/free)\n    Nobody holds it.\n\n    **Refusal:** `held`.\n" +
		"\n  Book exists (catalog/book/conditions/exists), a part of catalog/book\n    A Book exists.\n" +
		"\noutcomes:\n\n  Lent (catalog/operations/lend/outcomes/lent)\n    Somebody holds it.\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	got = renderDocDefinition(analysis.DocDefinition{
		ListedDefinition: analysis.ListedDefinition{ID: "catalog/operations/lend/conditions/free", Title: "Free", Kind: "std/Rule", File: "free.md"},
		Package:          "catalog", Text: "# Free\n", Whole: "catalog/operations/lend/lend",
	})
	if want := "catalog/operations/lend/conditions/free: Free (std/Rule)\nPackage catalog, free.md\nPart of catalog/operations/lend\n\n# Free\n"; got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	for text, want := range map[string]string{
		"# Free\n\nNobody holds it.\n":     "\nNobody holds it.\n",
		"Free\n====\n\nNobody holds it.\n": "\nNobody holds it.\n",
		"Nobody holds it.\n":               "Nobody holds it.\n",
		"# Free":                           "",
	} {
		if got := partText(text); got != want {
			t.Errorf("partText(%q) = %q, want %q", text, got, want)
		}
	}
}

// doc of a Package opens with the text of its anchor, then shows its public
// Definitions, what it publishes, apart from its internal ones, and leaves out
// a heading with nothing under it.
func TestRenderDocPackageSeparatesPublicFromInternal(t *testing.T) {
	got := renderDocPackage(analysis.DocPackage{Name: "members", Parent: "Root", Anchor: "members/members", Text: "# Members\n\nWho may borrow.\n", Definitions: []analysis.ListedDefinition{
		{ID: "members/limit", Title: "Limit", Kind: "std/Rule", Internal: true},
		{ID: "members/member", Title: "Member", Kind: "std/Concept"},
	}})
	want := "Package members, in Root\n" +
		"\n# Members\n\nWho may borrow.\n" +
		"\nPublic Definitions:\n  members/member: Member (std/Concept)\n" +
		"\nInternal Definitions:\n  members/limit: Limit (std/Rule)\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	got = renderDocPackage(analysis.DocPackage{Name: "Root", Definitions: []analysis.ListedDefinition{{ID: "project", Title: "Project", Kind: "std/Project"}}})
	if want := "Package Root\n\nPublic Definitions:\n  project: Project (std/Project)\n"; got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	got = renderDocPackage(analysis.DocPackage{Name: "loans", Parent: "members", Definitions: []analysis.ListedDefinition{{ID: "members/loans/count", Title: "Count", Kind: "std/Concept", Internal: true}}})
	if want := "Package loans, in members\n\nInternal Definitions:\n  members/loans/count: Count (std/Concept)\n"; got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// list --deps shows each Package's coupling, abstractness and distance, and what
// it depends on, with the links that make the dependency, and each Definition's
// links.
func TestRenderListWithDeps(t *testing.T) {
	pkgs := []analysis.ListedPackage{
		{Name: "Root", Definitions: []analysis.ListedDefinition{{ID: "loan-limit", Title: "Loan limit", Kind: "std/Rule"}}, Deps: &analysis.PackageDeps{Afferent: 1, Abstractness: 0.5, Distance: 0.5}},
		{Name: "catalog", Parent: "Root", Definitions: []analysis.ListedDefinition{{ID: "catalog/book", Title: "Book", Kind: "std/Concept", Links: []string{"members/member"}}}, Deps: &analysis.PackageDeps{Efferent: 1, Instability: 1, DependsOn: []analysis.Dependency{
			{Package: "members", Uses: []analysis.Use{{From: "catalog/book", Target: "members/member", Via: "link"}, {From: "catalog/dvd", Target: "members/Medium", Via: "kind"}}},
		}}},
	}
	want := "Root (Ca 1, Ce 0, instability 0.00, abstractness 0.50, distance 0.50)\n" +
		"  loan-limit: Loan limit (std/Rule)\n" +
		"catalog (Ca 0, Ce 1, instability 1.00, abstractness 0.00, distance 0.00)\n" +
		"  depends on members: catalog/book links members/member; catalog/dvd is of Kind members/Medium\n" +
		"  catalog/book: Book (std/Concept)\n" +
		"    links members/member\n"
	if got := renderList(pkgs, false); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	// The Packages alone: a dependency is one line with the number of links.
	want = "Root (Ca 1, Ce 0, instability 0.00, abstractness 0.50, distance 0.50), 1 Definition\n" +
		"catalog (Ca 0, Ce 1, instability 1.00, abstractness 0.00, distance 0.00), 1 Definition\n" +
		"  depends on members (2 links)\n"
	if got := renderList(pkgs, true); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
