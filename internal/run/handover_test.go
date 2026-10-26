package run

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// DEC-073: a Finding naming a Definition the checked model lacks is refused in
// every tool. In Verify a reject Finding that names none is refused with the
// way to find one, and no judgment is recorded: the reject never becomes an
// approval and never reaches Implement as a Note only.
func TestFindingsMustNameACheckedDefinitionWhereImplementReceivesThem(t *testing.T) {
	x, root, a, _ := inboxExecution()
	defer x.cancel()
	root.a.phase, root.a.version = active, 1
	a.a.phase, a.a.version = active, 1
	unknown := []Finding{{Definition: "a/Missing", Text: "repair it"}}
	for name, call := range map[string]func() error{
		"judge":    func() error { return x.tools(a).Judge(context.Background(), Submitted, unknown) },
		"review":   func() error { return x.tools(root).Review(context.Background(), "a", "c", Reject, unknown) },
		"delegate": func() error { return x.tools(root).Delegate(context.Background(), "a", "task", unknown) },
	} {
		if err := call(); err == nil || !strings.Contains(err.Error(), "a/Missing, which is not a Definition of the checked model") || !strings.Contains(err.Error(), "then "+name+" again") {
			t.Fatal(name, err)
		}
	}
	x.request.Mandate = Verify
	unbound := []Finding{{Text: "revert the optional test file"}}
	err := x.tools(a).Judge(context.Background(), Reject, unbound)
	for _, want := range []string{"Finding 1 of your reject names no Definition", ".markitect/registry/", "requiredBy", "then judge again"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatal("unbound Verify reject not refused with its next step", err)
		}
	}
	if a.a.judgment != nil {
		t.Fatal("a refused reject left a judgment")
	}
	if err = x.tools(root).Review(context.Background(), "a", "c", Reject, unbound); err == nil || !strings.Contains(err.Error(), "then review again") {
		t.Fatal("unbound Verify review reject not refused", err)
	}
	// An unknown Definition in a Verify reject cannot be left empty either.
	if err = x.tools(a).Judge(context.Background(), Reject, unknown); err == nil || !strings.Contains(err.Error(), "Any Definition of the checked model is accepted") || strings.Contains(err.Error(), "leave definition empty") {
		t.Fatal("unknown Definition of a Verify reject without a usable next step", err)
	}
	// A Verify approval hands nothing to Implement: its Findings are Notes.
	bound := []Finding{{Definition: "a/Thing", Text: "could use a boundary test"}}
	if err = x.tools(a).Judge(context.Background(), Approve, bound); err == nil || !strings.Contains(err.Error(), "a Verify approval hands no Finding to Implement, so an optional addition is a Note") {
		t.Fatal("Definition in a Verify approval not refused", err)
	}
	if err = x.tools(root).Review(context.Background(), "a", "c", Approve, bound); err == nil || !strings.Contains(err.Error(), "then review again") {
		t.Fatal("Definition in a Verify review approval not refused", err)
	}
	for _, decision := range []Decision{Approve, ModelBlocker} {
		if err = x.tools(a).Judge(context.Background(), decision, unbound); err != nil && strings.Contains(err.Error(), "Definition") {
			t.Fatal(decision, "refused for its Definition", err)
		}
	}
}

// Verify Root receives a diff up to MaxChanges in full and a larger one as its
// complete file summary with the way to read it.
func TestVerifyChangesStayWithinOneAssignment(t *testing.T) {
	small := "diff --git a/out/a.txt b/out/a.txt\n--- a/out/a.txt\n+++ b/out/a.txt\n-initial\n+changed\n"
	if got := changes(small, "bound"); got != small {
		t.Fatal(got)
	}
	large := small + "diff --git a/big.txt b/big.txt\n--- /dev/null\n+++ b/big.txt\n" + strings.Repeat("+line\n", MaxChanges/6+1)
	got := changes(large, "bound")
	for _, want := range []string{"read it with git diff bound in your copy", "a/out/a.txt b/out/a.txt +1 -1\n", "a/big.txt b/big.txt +"} {
		if !strings.Contains(got, want) {
			t.Fatal("summary lacks", want, got)
		}
	}
	if len(got) > MaxChanges {
		t.Fatal("summary exceeds the bound", len(got))
	}
}

// Initial keeps no Finding that names no Definition of the checked model.
func TestInitialKeepsOnlyFindingsOfCheckedDefinitions(t *testing.T) {
	if got := Initial(project.Prepared{}, []Finding{{Text: "from old Notes"}, {Definition: "a/Thing", Text: "unchecked model"}}); got != nil {
		t.Fatal(got)
	}
}

// A Finding longer than one Note continues in further Notes, cut only at rune
// boundaries, so nothing is lost.
func TestNotePartsKeepEveryByteWithinTheNoteLimit(t *testing.T) {
	// The limit falls inside the last two-byte rune.
	text := "x" + strings.Repeat("ä", MaxNoteText/2)
	parts := noteParts(text)
	if len(parts) != 2 || strings.Join(parts, "") != text {
		t.Fatal(len(parts))
	}
	for _, part := range parts {
		if len(part) > MaxNoteText || !utf8.ValidString(part) {
			t.Fatal("part breaks the limit or a rune", len(part))
		}
	}
	if got := noteParts("short"); len(got) != 1 || got[0] != "short" {
		t.Fatal(got)
	}
}
