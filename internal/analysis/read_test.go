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
