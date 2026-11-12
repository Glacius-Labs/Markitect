package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// doc --read prints the Definitions as one Markdown text: a heading for
// each folder, a thing under the folder it lies in, its parts one level
// below it, each with its ID and Kind and its text without the title.
func TestRenderReadIsOneTextToRead(t *testing.T) {
	got := renderRead("members", []analysis.ReadEntry{
		{ID: "members/members", Title: "Members", Kind: "Package", Folder: "members", Text: "# Members\n\nWho may borrow.\n"},
		{ID: "members/member/member", Title: "Member", Kind: "std/Concept", Folder: "members/member", Text: "# Member\n\nA person with a [[members/member/attributes/card]].\n", Shown: "# Member\n\nA person with a Card.\n"},
		{ID: "members/member/attributes/card", Title: "Card", Kind: "std/Attribute", Folder: "members/member/attributes", Whole: "members/member/member", Text: "# Card\n\nThe number.\n"},
		{ID: "members/rules/quiet", Kind: "std/Rule", Folder: "members/rules", Text: "\n"},
	})
	want := "# The spec: members\n" +
		"\n4 Definitions, as markitect doc --read prints them. The files under .markitect/spec are the spec; this text is a view of them.\n" +
		"\n### Members\n\n`members/members`, Package\n\nWho may borrow.\n" +
		"\n## members\n" +
		"\n### Member\n\n`members/member/member`, std/Concept\n\nA person with a Card.\n" +
		"\n#### Card\n\n`members/member/attributes/card`, std/Attribute\n\nThe number.\n" +
		"\n## members/rules\n" +
		"\n### members/rules/quiet\n\n`members/rules/quiet`, std/Rule\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if whole := renderRead("", []analysis.ReadEntry{{ID: "project", Title: "Library", Kind: "std/Project", Text: "# Library\n\nBooks.\n"}}); whole != "# The spec\n\n1 Definition, as markitect doc --read prints them. The files under .markitect/spec are the spec; this text is a view of them.\n\n### Library\n\n`project`, std/Project\n\nBooks.\n" {
		t.Fatalf("the whole spec:\n%s", whole)
	}
}

// list --kinds prints the Kinds the spec has Definitions of, with their
// folders, and then the Kinds it has none of.
func TestRenderKinds(t *testing.T) {
	got := renderKinds([]analysis.KindCount{
		{Kind: "std/Goal", Definitions: 2, Folders: []string{"purpose"}},
		{Kind: "std/Rule", Definitions: 3, Folders: []string{"architecture", "domain"}},
		{Kind: "std/Scenario"},
		{Kind: "std/Task"},
	})
	want := "Kinds and how many Definitions the spec has of each:\n\n" +
		"  std/Goal: 2, in purpose\n  std/Rule: 3, in architecture, domain\n" +
		"\nNo Definition yet: std/Scenario, std/Task\n" +
		"\nmarkitect doc KIND prints what a Kind is for.\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
