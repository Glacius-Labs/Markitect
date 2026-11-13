package analysis

import (
	"strings"
	"testing"
)

// The reading view gives the Definitions at or below a folder in the order
// one reads them: the Definition that carries its folder's name first, then
// the other files and the things with parts, each thing with its parts
// behind it, then the folders below, and a link by its title.
func TestReadGivesTheSpecInReadingOrder(t *testing.T) {
	r := library(t)
	r[m+"members/member/member.md"] = def("std/Concept", "Member", "A person with a [[members/member/attributes/card]].")
	delete(r, m+"members/member.md")
	r[m+"members/member/attributes/card.md"] = def("std/Attribute", "Card", "The number on the card.")
	r[m+"members/aaa.md"] = def("std/Concept", "Aaa", "Sorts before the anchor by name.")
	p := loadRepo(t, r)

	var ids []string
	for _, e := range Read(p, "members") {
		ids = append(ids, e.ID)
	}
	want := "members/members,members/aaa,members/member/member,members/member/attributes/card,members/loans/loans,members/loans/loan"
	if got := strings.Join(ids, ","); got != want {
		t.Fatalf("order:\n%s\nwant:\n%s", got, want)
	}
	whole := Read(p, "")
	if len(whole) != len(p.Spec.OwnDefinitions()) || whole[0].ID != "loan-limit" && whole[0].ID != "project" {
		t.Fatalf("the whole spec: %d entries, first %s", len(whole), whole[0].ID)
	}
	for _, e := range Read(p, "members/member") {
		switch e.ID {
		case "members/member/member":
			// A thing is read under the folder its own folder lies in.
			if e.Whole != "" || e.Depth != 0 || e.Folder != "members" || !strings.Contains(e.Shown, "A person with a Card.") || !strings.Contains(e.Text, "[[members/member/attributes/card]]") {
				t.Errorf("the whole: %+v", e)
			}
		case "members/member/attributes/card":
			if e.Whole != "members/member/member" || e.Depth != 1 || e.Kind != "std/Attribute" {
				t.Errorf("the part: %+v", e)
			}
		default:
			t.Errorf("not below the folder: %s", e.ID)
		}
	}
	if got := Read(p, "member"); len(got) != 0 {
		t.Fatalf("a folder the spec does not have: %+v", got)
	}
}

// The Kinds of a spec are its own, those of its Libraries and Package, each
// with how many Definitions the spec has of it and in which top-level
// folders: what it describes so far, and what it has nothing of.
func TestKindsCountsTheDefinitionsOfEachKind(t *testing.T) {
	r := library(t)
	r[m+"kinds/Note.md"] = "---\nkind: Kind\n---\n# Note\n\nA note.\n"
	p := loadRepo(t, r)
	got := map[string]KindCount{}
	var order []string
	for _, k := range Kinds(p) {
		got[k.Kind] = k
		order = append(order, k.Kind)
	}
	if !strings.HasPrefix(strings.Join(order, ","), "Kind,Package,kinds/Note,std/Action,") {
		t.Fatalf("order: %v", order)
	}
	for kind, want := range map[string]string{
		"std/Concept": "4 catalog,members",
		"std/Rule":    "1 .",
		"std/Project": "1 .",
		"Package":     "3 catalog,members",
		"Kind":        "1 kinds",
		"kinds/Note":  "0 ",
		"std/Goal":    "0 ",
	} {
		k, ok := got[kind]
		if !ok {
			t.Errorf("%s is missing", kind)
			continue
		}
		if have := strings.TrimSpace(strings.Join([]string{itoa(k.Definitions), strings.Join(k.Folders, ",")}, " ")); have != strings.TrimSpace(want) {
			t.Errorf("%s: %s, want %s", kind, have, want)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// doc --read --bound adds to the Definitions of a folder, each once and with
// its parts, what binds them from outside the folder, as doc lists it under
// Bound by, and what outside it they link; neither Kinds nor what a Library
// holds, and nothing of the folder itself.
func TestReadBoundAddsWhatBindsAndWhatIsLinked(t *testing.T) {
	r := library(t)
	r[m+"kinds/Law.md"] = "---\nkind: Kind\nbinds: true\n---\n# Law\n\nHolds below its place.\n"
	r[m+"laws/quiet/quiet.md"] = def("kinds/Law", "Quiet", "Be quiet.")
	r[m+"laws/quiet/at-night.md"] = def("kinds/Law", "At night", "At night.")
	r[m+"members/laws/cards.md"] = def("kinds/Law", "Cards", "Members carry a card.")
	r[m+"catalog/laws/shelves.md"] = def("kinds/Law", "Shelves", "Books stand on shelves.")
	r[m+"places/places.md"] = def("Package", "Places", "Where lending happens.")
	r[m+"places/desk.md"] = def("std/Concept", "Desk", "Where a book is handed over.")
	r[m+"places/rules/hours/hours.md"] = def("std/Rule", "Hours", "Open by day.")
	r[m+"places/rules/hours/morning.md"] = def("std/Rule", "Morning", "From `8`.")
	r[m+"members/member.md"] = def("std/Concept", "Member", "A person at the [[places/desk]] in [[places/rules/hours]], under [[loan-limit]] and [[laws/quiet/at-night]], a [[kinds/Law]] and a [[std/Rule]].")
	r[m+"members/loans/loan.md"] = def("std/Concept", "Loan", "A [[members/member]] borrows at the [[places/desk]].")
	// Operations and Reactions of members, of Root and of the Package below
	// members, loans.
	r[m+"members/operations/join/join.md"] = def("std/Operation", "Join", "A person becomes a [[members/member]].\n\n**Who:** `staff`\n\n**Satisfies:** [[loan-limit]]")
	r[m+"members/operations/join/outcomes/joined.md"] = def("std/Outcome", "Joined", "The [[members/member]] exists.")
	r[m+"members/reactions/remind.md"] = def("std/Reaction", "Remind", "Remind a [[members/member]].\n\n**When:** a week has passed.\n\n**Satisfies:** [[loan-limit]]")
	r[m+"members/loans/operations/lend.md"] = def("std/Operation", "Lend", "Lend a book.\n\n**Who:** `staff`\n\n**Satisfies:** [[loan-limit]]")
	r[m+"operations/open.md"] = def("std/Operation", "Open", "Open the library.\n\n**Who:** `staff`\n\n**Satisfies:** [[loan-limit]]")
	p := loadRepo(t, r)

	ids := func(entries []ReadEntry) string {
		var out []string
		for _, e := range entries {
			out = append(out, e.ID)
		}
		return strings.Join(out, ",")
	}
	entries := Read(p, "members")
	br := ReadBound(p, "members", entries)
	bound, linked := br.BoundBy, br.Linked
	walk := func(br BoundRead) string {
		var out []string
		for _, d := range br.Walk {
			out = append(out, d.ID)
		}
		return br.Package + ":" + strings.Join(out, ",")
	}
	// The Operations and Reactions whose own Package is members, not those of
	// loans below it, nor their parts.
	if got, want := walk(br), "members:members/operations/join/join,members/reactions/remind"; got != want {
		t.Errorf("walk:\n%s\nwant:\n%s", got, want)
	}
	if got, want := walk(ReadBound(p, "members/loans", Read(p, "members/loans"))), "members/loans:members/loans/operations/lend"; got != want {
		t.Errorf("walk of loans: %s", got)
	}
	// Root's own, for the whole spec and for a folder of Root's.
	if got, want := walk(ReadBound(p, "", Read(p, ""))), ":operations/open"; got != want {
		t.Errorf("walk of Root: %s", got)
	}
	if got, want := walk(ReadBound(p, "operations", Read(p, "operations"))), ":operations/open"; got != want {
		t.Errorf("walk of a folder of Root: %s", got)
	}
	// Each binding once, though it binds every Definition of the folder, a thing
	// with its parts, and not what binds only from inside the folder or what
	// binds elsewhere. The loan limit is linked too, and comes once, as binding.
	if got, want := ids(bound), "loan-limit,laws/quiet/quiet,laws/quiet/at-night"; got != want {
		t.Errorf("bound:\n%s\nwant:\n%s", got, want)
	}
	// Each linked Definition once, a thing with its parts, but no Kind, nothing
	// of a Library, and nothing that is printed already.
	if got, want := ids(linked), "places/desk,places/rules/hours/hours,places/rules/hours/morning"; got != want {
		t.Errorf("linked:\n%s\nwant:\n%s", got, want)
	}
	// Depth is how many wholes lie above a Definition, as Read has it.
	for _, e := range append(bound, linked...) {
		if (e.ID == "laws/quiet/at-night" || e.ID == "places/rules/hours/morning") != (e.Depth == 1) {
			t.Errorf("%s at depth %d", e.ID, e.Depth)
		}
		if e.Text == "" {
			t.Errorf("%s without its text", e.ID)
		}
	}

	// A folder that nothing binds from outside and that links nothing outside:
	// a Kind of the project binds nothing, as doc says.
	if br := ReadBound(p, "kinds", Read(p, "kinds")); len(br.BoundBy) != 0 || len(br.Linked) != 0 || len(br.Walk) != 0 {
		t.Errorf("kinds: bound %s, linked %s", ids(br.BoundBy), ids(br.Linked))
	}
	// The whole spec has nothing outside it.
	if br := ReadBound(p, "", Read(p, "")); len(br.BoundBy) != 0 || len(br.Linked) != 0 {
		t.Errorf("the whole spec: bound %s, linked %s", ids(br.BoundBy), ids(br.Linked))
	}
	// A part linked from outside its thing comes alone, as doc shows it, with
	// the depth it has.
	r[m+"catalog/book.md"] = def("std/Concept", "Book", "Lent to a [[members/member]] in the [[places/rules/hours/morning]].")
	p = loadRepo(t, r)
	if linked := ReadBound(p, "catalog", Read(p, "catalog")).Linked; ids(linked) != "members/member,places/rules/hours/morning" || linked[1].Depth != 1 {
		t.Errorf("catalog links %s", ids(linked))
	}
}
