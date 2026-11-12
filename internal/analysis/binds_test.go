package analysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// doc lists what binds a Definition without a link: the Definitions of a Kind
// that declares binds, in its Package and in those above it, a thing by its
// whole. What lies in a Package beside or below it does not bind it.
func TestDocListsWhatBindsADefinition(t *testing.T) {
	r := library(t)
	r[m+"kinds/Law.md"] = "---\nkind: Kind\nbinds: true\n---\n# Law\n\nHolds below its place.\n"
	r[m+"laws/quiet/quiet.md"] = def("kinds/Law", "Quiet", "Be quiet.")
	r[m+"laws/quiet/at-night.md"] = def("kinds/Law", "At night", "At night.")
	r[m+"members/laws/cards.md"] = def("kinds/Law", "Cards", "Members carry a card.")
	r[m+"members/loans/laws/due.md"] = def("kinds/Law", "Due", "Loans are due.")
	r[m+"catalog/laws/shelves.md"] = def("kinds/Law", "Shelves", "Books stand on shelves.")
	// A law with a part that has a part of its own.
	r[m+"laws/order/order.md"] = def("kinds/Law", "Order", "Keep order.")
	r[m+"laws/order/rows/rows.md"] = def("kinds/Law", "Rows", "In rows.")
	r[m+"laws/order/rows/straight.md"] = def("kinds/Law", "Straight", "Straight ones.")
	p := loadRepo(t, r)
	bound := func(id string) string {
		var ids []string
		for _, b := range DocDefinitionOf(p, p.Spec.Definitions[id]).BoundBy {
			ids = append(ids, b.ID)
		}
		return strings.Join(ids, ",")
	}
	// std's own Rule binds too: loan-limit is a std/Rule at Root.
	if got, want := bound("members/loans/loan"), "laws/order/order,laws/quiet/quiet,loan-limit,members/laws/cards,members/loans/laws/due"; got != want {
		t.Errorf("bound a loan: %s", got)
	}
	if got, want := bound("members/member"), "laws/order/order,laws/quiet/quiet,loan-limit,members/laws/cards"; got != want {
		t.Errorf("bound a member: %s", got)
	}
	if got, want := bound("catalog/book"), "catalog/laws/shelves,laws/order/order,laws/quiet/quiet,loan-limit"; got != want {
		t.Errorf("bound a book: %s", got)
	}
	// A Definition does not bind itself, nor does the thing it is a part of.
	if got, want := bound("laws/quiet/at-night"), "laws/order/order,loan-limit"; got != want {
		t.Errorf("bound a part of a law: %s", got)
	}
	// Nor does a thing above the thing it is a part of.
	for _, id := range []string{"laws/order/rows/rows", "laws/order/rows/straight"} {
		if got, want := bound(id), "laws/quiet/quiet,loan-limit"; got != want {
			t.Errorf("bound %s: %s", id, got)
		}
	}
	if got := bound("std/Rule"); got != "" {
		t.Errorf("bound a Kind of a Library: %s", got)
	}
	law := DocDefinitionOf(p, p.Spec.Definitions["kinds/Law"])
	if law.Declares == nil || !law.Declares.Binds {
		t.Fatalf("Law declares %+v", law.Declares)
	}
}

// Nothing binds a Definition that no file realizes, such as a Goal, nor a
// part of it, nor a Kind of the project (DEC-113); a Definition a file
// realizes is bound as before.
func TestDocListsNothingThatBindsWhatNoFileRealizes(t *testing.T) {
	r := library(t)
	r[m+"kinds/Medium.md"] = "---\nkind: Kind\n---\n# Medium\n\nSomething that can be lent.\n"
	r[m+"purpose/goals/lend.md"] = def("std/Goal", "Lend", "**Result:** books go home.\n\n**Changes:** shelves empty.\n\n**Obstacle:** paper cards.")
	r[m+"purpose/stakeholders/staff/staff.md"] = def("std/Stakeholder", "Staff", "The people at the desk.")
	r[m+"purpose/stakeholders/staff/wishes/lend.md"] = def("std/Wish", "Lend", "Lend a book while the member waits.")
	r[m+"context/constraints/law.md"] = def("std/Constraint", "Law", "Loans are recorded.")
	p := loadRepo(t, r)
	for _, id := range []string{"project", "purpose/goals/lend", "purpose/stakeholders/staff/staff", "purpose/stakeholders/staff/wishes/lend", "context/constraints/law", "kinds/Medium"} {
		dd := DocDefinitionOf(p, p.Spec.Definitions[id])
		if len(dd.BoundBy) != 0 {
			t.Errorf("%s is bound by %v", id, dd.BoundBy)
		}
		if data, err := json.Marshal(dd); err != nil || strings.Contains(string(data), "bound_by") {
			t.Errorf("%s in JSON: %s", id, data)
		}
	}
	// The Constraint binds without being bound itself.
	if got := DocDefinitionOf(p, p.Spec.Definitions["members/member"]).BoundBy; len(got) != 2 || got[0].ID != "context/constraints/law" || got[1].ID != "loan-limit" {
		t.Errorf("member is bound by %v", got)
	}
}

// vet.hub counts links: what a Definition binds by its place is no reason to
// call it a hub, however much of the spec that is.
func TestVetCountsNoBindingAsAHub(t *testing.T) {
	r := library(t)
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("catalog/item-%02d", i)
		r[m+id+".md"] = def("std/Concept", "Item", "One thing on a shelf.")
	}
	p := loadRepo(t, r)
	if n := len(DocDefinitionOf(p, p.Spec.Definitions["catalog/item-00"]).BoundBy); n != 1 {
		t.Fatalf("the loan limit binds an item: %d", n)
	}
	for _, w := range Vet(p, History{}).Warnings {
		if w.Code == "vet.hub" {
			t.Errorf("a Rule that binds by its place is a hub: %v", w)
		}
	}
}

// A mention of something the Kind of a Definition does not let it link is no
// missing link.
func TestVetReportsNoMentionTheKindMayNotLink(t *testing.T) {
	r := library(t)
	r[m+"kinds/Term.md"] = "---\nkind: Kind\nlinks: kinds/Term\nfields:\n  See:\n  By: {links: std/Concept}\n---\n# Term\n\nLinks only Terms, and in By a Concept.\n"
	r[m+"terms/shelf-mark.md"] = def("kinds/Term", "Shelf mark", "Where a book stands.")
	// The loan limit is a Rule, which no part of a Term may link; the book and the shelf are
	// Concepts, which its field By may.
	r[m+"catalog/terms/spine.md"] = def("kinds/Term", "Spine", "The back of a book, which bears the shelf mark and the loan limit, and faces out of the shelf.")
	p := loadRepo(t, r)
	for _, n := range Vet(p, History{}).Notes {
		if n.Code != "vet.unlinked_mention" || !strings.HasSuffix(n.File, "catalog/terms/spine.md") {
			continue
		}
		if got := strings.Join(n.Targets, ","); got != "catalog/book,catalog/shelf,terms/shelf-mark" {
			t.Fatalf("a Term names %s", got)
		}
		return
	}
	t.Fatal("no note on the Term that names another Term")
}
