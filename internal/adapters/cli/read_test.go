package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// doc --read prints the Definitions as one Markdown text: a heading for
// each folder, a Package's anchor under its own folder, a thing under the
// folder it lies in, each part one heading below its whole, each with its ID
// and Kind and its text without the title. A heading inside a text moves
// below the Definition's own.
func TestRenderReadIsOneTextToRead(t *testing.T) {
	got := renderRead("members", []analysis.ReadEntry{
		{ID: "members/members", Title: "Members", Kind: "Package", Folder: "members", Text: "# Members\n\nWho may borrow.\n"},
		{ID: "members/member/member", Title: "Member", Kind: "std/Concept", Folder: "members", Text: "# Member\n\nA person with a [[members/member/attributes/card]].\n", Shown: "# Member\n\nA person with a Card.\n\n## Why\n\nSo that loans have a holder.\n\n```\n# no heading\n```\n"},
		{ID: "members/member/attributes/card", Title: "Card", Kind: "std/Attribute", Folder: "members/member/attributes", Whole: "members/member/member", Depth: 1, Text: "# Card\n\nThe number.\n"},
		{ID: "members/member/attributes/card/values/lost", Title: "Lost", Kind: "std/State", Folder: "members/member/attributes/card/values", Whole: "members/member/attributes/card", Depth: 2, Text: "# Lost\n\nReported lost.\n"},
		{ID: "members/loans/loans", Title: "Loans", Kind: "Package", Folder: "members/loans", Text: "# Loans\n"},
		{ID: "members/rules/quiet", Kind: "std/Rule", Folder: "members/rules", Text: "\n"},
	})
	want := "# The spec: members\n" +
		"\n6 Definitions, as markitect doc --read prints them. The files under .markitect/spec are the spec; this text is a view of them.\n" +
		"\n## members\n" +
		"\n### Members\n\n`members/members`, Package\n\nWho may borrow.\n" +
		"\n### Member\n\n`members/member/member`, std/Concept\n\nA person with a Card.\n\n#### Why\n\nSo that loans have a holder.\n\n```\n# no heading\n```\n" +
		"\n#### Card\n\n`members/member/attributes/card`, std/Attribute\n\nThe number.\n" +
		"\n##### Lost\n\n`members/member/attributes/card/values/lost`, std/State\n\nReported lost.\n" +
		"\n## members/loans\n" +
		"\n### Loans\n\n`members/loans/loans`, Package\n" +
		"\n## members/rules\n" +
		"\n### members/rules/quiet\n\n`members/rules/quiet`, std/Rule\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if whole := renderRead("", []analysis.ReadEntry{{ID: "project", Title: "Library", Kind: "std/Project", Text: "# Library\n\nBooks.\n"}}); whole != "# The spec\n\n1 Definition, as markitect doc --read prints them. The files under .markitect/spec are the spec; this text is a view of them.\n\n### Library\n\n`project`, std/Project\n\nBooks.\n" {
		t.Fatalf("the whole spec:\n%s", whole)
	}
}

// A heading of a text moves below the heading it stands under and never
// deeper than six; what only looks like one stays.
func TestHeadingsBelow(t *testing.T) {
	text := "## One\n\n#tag and # in a line\n\n    # indented code\n\n~~~\n## fenced\n~~~\n\n````\n```\n## in a long fence\n````\n\nUnderlined\n----------\n\n- an item\n---\n\n#### Deep\n"
	want := "#### One\n\n#tag and # in a line\n\n    # indented code\n\n~~~\n## fenced\n~~~\n\n````\n```\n## in a long fence\n````\n\n#### Underlined\n\n\n- an item\n---\n\n###### Deep\n"
	if got := headingsBelow(text, 3); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
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
