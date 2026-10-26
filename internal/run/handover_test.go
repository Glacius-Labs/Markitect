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
	for _, want := range []string{"Finding 1 of your reject names no Definition", ".markitect/registry/", "requiredBy", "Then judge again"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatal("unbound Verify reject not refused with its next step", err)
		}
	}
	if a.a.judgment != nil {
		t.Fatal("a refused reject left a judgment")
	}
	if err = x.tools(root).Review(context.Background(), "a", "c", Reject, unbound); err == nil || !strings.Contains(err.Error(), "Then review again") {
		t.Fatal("unbound Verify review reject not refused", err)
	}
	// Approvals and model blockers hand nothing to Implement.
	for _, decision := range []Decision{Approve, ModelBlocker} {
		if err = x.tools(a).Judge(context.Background(), decision, unbound); err != nil && strings.Contains(err.Error(), "Definition") {
			t.Fatal(decision, "refused for its Definition", err)
		}
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
