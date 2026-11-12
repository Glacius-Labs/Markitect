package cli

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// doc lists what binds a Definition without a link below what it links, and
// for a Kind what it declares about binding and about what its Definitions
// may link.
func TestRenderDocShowsWhatBinds(t *testing.T) {
	got := renderDocDefinition(analysis.DocDefinition{
		ListedDefinition: analysis.ListedDefinition{ID: "members/loan", Title: "Loan", Kind: "std/Concept", File: "loan.md", Links: []string{"members/member"}},
		Package:          "members", Text: "# Loan\n",
		BoundBy: []analysis.ListedDefinition{
			{ID: "rules/logging/logging", Title: "Logging", Kind: "std/Rule"},
			{ID: "members/rules/cards", Title: "Cards", Kind: "std/Rule"},
		},
	})
	want := "members/loan: Loan (std/Concept)\nPackage members, loan.md\n\n# Loan\n" +
		"\nLinks to: members/member\n" +
		"\nBound by, without a link, as everything in its Package:\n  rules/logging/logging: Logging (std/Rule)\n  members/rules/cards: Cards (std/Rule)\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	kind := renderDocDefinition(analysis.DocDefinition{
		ListedDefinition: analysis.ListedDefinition{ID: "std/Constraint", Title: "Constraint", Kind: "Kind", File: "std/Constraint.md"},
		Library:          "std", Text: "# Constraint\n",
		Declares: &analysis.DocDeclares{Links: []string{"std/Term", "std/Constraint"}, Unrealized: true, Binds: true},
	})
	for _, line := range []string{
		"\nDeclares that it links only: std/Term, std/Constraint\n",
		"\nDeclares that a Definition of this Kind binds its Package and those below it without a link.\n",
		"\nDeclares that no file realizes a Definition of this Kind.\n",
	} {
		if !strings.Contains(kind, line) {
			t.Errorf("doc of the Kind lacks %q:\n%s", line, kind)
		}
	}
	if none := renderDocDefinition(analysis.DocDefinition{Declares: &analysis.DocDeclares{Links: []string{}}}); !strings.Contains(none, "\nDeclares that it links nothing.\n") {
		t.Errorf("a Kind that links nothing:\n%s", none)
	}
}
