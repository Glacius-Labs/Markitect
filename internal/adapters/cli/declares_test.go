package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// doc of a Kind prints what it declares below its text, since the
// declaration stands in the front matter, which the text leaves out.
func TestRenderDocShowsWhatAKindDeclares(t *testing.T) {
	kind := func(declares *analysis.DocDeclares) string {
		return renderDocDefinition(analysis.DocDefinition{
			ListedDefinition: analysis.ListedDefinition{ID: "std/Operation", Title: "Operation", Kind: "Kind", File: "std/Operation.md", Links: []string{"std/Goal"}},
			Library:          "std", Text: "# Operation\n\nOne thing a caller can do.\n", Declares: declares,
		})
	}
	head := "std/Operation: Operation (Kind)\nLibrary std\n\n# Operation\n\nOne thing a caller can do.\n"
	tail := "\nLinks to: std/Goal\n"
	for _, c := range []struct {
		declares *analysis.DocDeclares
		want     string
	}{
		{nil, ""},
		{&analysis.DocDeclares{
			Fields: []analysis.DocDeclaredField{
				{Name: "Who", Required: true},
				{Name: "Given"},
				{Name: "Serves", Required: true, Links: []string{"std/Goal", "std/Rule"}},
				{Name: "Alone", Links: []string{}},
			},
			Parts: []analysis.DocDeclaredPart{
				{Folder: "conditions", Kinds: []string{"std/Condition"}},
				{Folder: ".", Kinds: []string{"std/Operation", "std/Rule"}},
			},
		}, "\nDeclares the fields:\n  Who, required\n  Given\n  Serves, required, links std/Goal, std/Rule\n  Alone, links nothing\n" +
			"\nDeclares the parts:\n  under conditions/: std/Condition\n  in the folder itself: std/Operation, std/Rule\n"},
		{&analysis.DocDeclares{Fields: []analysis.DocDeclaredField{}, Parts: []analysis.DocDeclaredPart{}},
			"\nDeclares the fields:\n  none\n\nDeclares the parts:\n  none\n"},
		{&analysis.DocDeclares{Parts: []analysis.DocDeclaredPart{}}, "\nDeclares the parts:\n  none\n"},
	} {
		if got, want := kind(c.declares), head+c.want+tail; got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	}
}
