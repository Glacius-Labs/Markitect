package analysis

import (
	"strings"
	"testing"
)

// The name of what no file realizes, such as a Term or a group of people, is
// a word of the project: a sentence that uses it names no missing link.
func TestVetReportsNoMentionOfWhatIsNotRealized(t *testing.T) {
	r := library(t)
	r[m+"kinds/Group.md"] = "---\nkind: Kind\nrealized: false\n---\n# Group\n\nA group of people.\n"
	r[m+"readers.md"] = def("kinds/Group", "Reader", "People who read.")
	r[m+"catalog/desk.md"] = def("std/Concept", "Desk", "Where a reader asks for a book from the shelf.")
	p := loadRepo(t, r)
	for _, n := range Vet(p, History{}).Notes {
		if n.Code != "vet.unlinked_mention" || !strings.HasSuffix(n.File, "catalog/desk.md") {
			continue
		}
		// The book and the shelf are Concepts of its Package; the reader is not reported.
		if got := strings.Join(n.Targets, ","); got != "catalog/book,catalog/shelf" {
			t.Fatalf("the desk names %s", got)
		}
		return
	}
	t.Fatal("no note on the desk")
}
