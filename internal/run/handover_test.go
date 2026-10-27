package run

import (
	"context"
	"errors"
	"fmt"
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
	for _, want := range []string{"Finding 1 of your reject names no Definition", "record it with append_note", ".markitect/registry/", "requiredBy", "then judge again"} {
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

// DEC-073: when the host states that the bound input was already verified and
// the fixed input differs from it, Verify Root's approval names the defect the
// change corrects. A plain approval, and one naming a Definition the checked
// model lacks, are refused with the next step and record no judgment. Other
// Managers, reviews and other decisions keep the ordinary rule. The admitted,
// recorded approval needs a checked model; the real-Git Apply test
// TestApplyStatesTheVerifiedFactOnlyForAStoredApprovalOfTheSameInput covers it.
func TestVerifiedChangeApprovalNamesTheCorrectedDefect(t *testing.T) {
	x, root, a, _ := inboxExecution()
	defer x.cancel()
	x.request.Mandate = Verify
	x.request.Input = project.CandidateRef{Commit: "changed", Tree: "changed-tree"}
	x.request.Origin = Origin{Input: project.CandidateRef{Commit: "bound", Tree: "bound-tree"}, Verified: "first-3-verify"}
	x.changed = true
	root.a.phase, root.a.version = active, 1
	a.a.phase, a.a.version = active, 1
	approve := func(s *managerState, findings []Finding) error {
		return x.tools(s).Judge(context.Background(), Approve, findings)
	}
	err := approve(root, []Finding{{Text: "the candidate conforms"}})
	for _, want := range []string{"candidate changed differs from the command's bound input bound", "Verify Run first-3-verify already verified", "names the defect it corrects", ".markitect/registry/", "judge reject instead"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatal("plain approval of a change on a verified input not refused with its next step", want, err)
		}
	}
	err = approve(root, []Finding{{Definition: "a/Missing", Text: "the bound input violated it"}})
	for _, want := range []string{"a/Missing, which is not a Definition of the checked model", "name as definition the current Definition that the bound input violated", "then judge again"} {
		if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "leave definition empty") {
			t.Fatal("unknown corrected Definition not refused with a usable next step", want, err)
		}
	}
	if root.a.judgment != nil {
		t.Fatal("a refused approval left a judgment")
	}
	// The rule binds the Run's judgment only: a child judges its own
	// assignment, and reviews and Definition-free rejects stay as they were.
	if err = x.findingsErr(a, "judge", Approve, nil); err != nil {
		t.Fatal("a child's plain approval was refused", err)
	}
	bound := []Finding{{Definition: "a/Thing", Text: "corrects a/Thing"}}
	if err = approve(a, bound); err == nil || !strings.Contains(err.Error(), "a Verify approval hands no Finding to Implement") {
		t.Fatal("a child's approval with a Definition was admitted", err)
	}
	if err = x.tools(root).Review(context.Background(), "a", "c", Approve, bound); err == nil || !strings.Contains(err.Error(), "then review again") {
		t.Fatal("a review approval with a Definition was admitted", err)
	}
	if err = x.tools(root).Judge(context.Background(), Reject, []Finding{{Text: "unbound"}}); err == nil || !strings.Contains(err.Error(), "of your reject names no Definition") {
		t.Fatal("the reject rule changed", err)
	}
	// Without a change, or without the fact, a plain approval is admitted and
	// an approval still names no Definition.
	for name, setup := range map[string]func(){
		"unchanged": func() { x.changed, x.request.Origin.Verified = false, "first-3-verify" },
		"no fact":   func() { x.changed, x.request.Origin.Verified = true, "" },
	} {
		setup()
		if err = x.findingsErr(root, "judge", Approve, []Finding{{Text: "conforms"}}); err != nil {
			t.Fatal(name, "plain approval refused", err)
		}
		if err = approve(root, bound); err == nil || !strings.Contains(err.Error(), "a Verify approval hands no Finding to Implement") {
			t.Fatal(name, "approval with a Definition admitted", err)
		}
	}
}

// Verify Root receives a diff up to MaxChanges in full and a larger one as its
// complete file summary with the way to read it.
func TestVerifyChangesStayWithinOneAssignment(t *testing.T) {
	// A removed SQL comment and an added "++" line are content, not headers.
	small := "diff --git a/q.sql b/q.sql\n--- a/q.sql\n+++ b/q.sql\n@@ -1,2 +1,2 @@\n--- old comment\n+++ new\n context\n"
	if got := changes(small, "bound"); got != small {
		t.Fatal(got)
	}
	large := small + "diff --git a/big.txt b/big.txt\n--- /dev/null\n+++ b/big.txt\n@@ -0,0 +1,43691 @@\n" + strings.Repeat("+line\n", MaxChanges/6+1)
	got := changes(large, "bound")
	for _, want := range []string{"read it with git diff bound HEAD in your copy", "a/q.sql b/q.sql +1 -1\n", fmt.Sprintf("a/big.txt b/big.txt +%d -0\n", MaxChanges/6+1)} {
		if !strings.Contains(got, want) {
			t.Fatal("summary lacks", want, got)
		}
	}
	if len(got) > MaxChanges {
		t.Fatal("summary exceeds the bound", len(got))
	}
	// A summary too long for one assignment says what it leaves out.
	var many strings.Builder
	for i := range 20000 {
		fmt.Fprintf(&many, "diff --git a/f%d b/f%d\n--- /dev/null\n+++ b/f%d\n@@ -0,0 +1 @@\n+x\n", i, i, i)
	}
	got = changes(many.String(), "bound")
	if len(got) > MaxChanges || !strings.Contains(got, "further changed files are not listed here; git diff --stat bound HEAD lists them all.") || !strings.Contains(got, "a/f0 b/f0 +1 -0\n") {
		t.Fatal("unbounded or silent summary", len(got))
	}
}

type noteRecords struct {
	Records
	notes []Note
	fail  error
}

func (r *noteRecords) Append(_ context.Context, n Note) (Note, error) {
	if r.fail != nil {
		return Note{}, r.fail
	}
	r.notes = append(r.notes, n)
	return n, nil
}

// Only an Assessment approval and a Verify judgment keep their unbound
// Findings as Notes; a long Finding continues in further Notes, and a failed
// append is an error, never a silent loss.
func TestOnlyHandedOverDecisionsKeepUnboundFindingsAsNotes(t *testing.T) {
	findings := []Finding{{Manager: "a", Text: "an optional idea"}, {Manager: "a", Text: " "}}
	note := func(mandate Mandate, decision Decision, records *noteRecords, findings []Finding) error {
		x := &execution{executor: &Executor{Records: records}, request: Request{ID: "r", Project: "p", Mandate: mandate}, result: Result{Decision: decision, Findings: findings}}
		return x.noteUnbound(context.Background())
	}
	for _, tc := range []struct {
		mandate  Mandate
		decision Decision
		notes    int
	}{{Assessment, Approve, 1}, {Assessment, Reject, 0}, {Assessment, ModelBlocker, 0}, {Verify, Approve, 1}, {Verify, Reject, 1}, {Verify, ModelBlocker, 0}, {Implement, Submitted, 0}} {
		records := &noteRecords{}
		if err := note(tc.mandate, tc.decision, records, findings); err != nil || len(records.notes) != tc.notes {
			t.Fatal(tc.mandate, tc.decision, len(records.notes), err)
		}
		for _, n := range records.notes {
			if n.Run != "r" || n.Project != "p" || n.Manager != "a" || n.Text != "an optional idea" || !strings.Contains(n.Context, "noted, not implemented") {
				t.Fatal("Note lost its origin or content", n)
			}
		}
	}
	if err := note(Verify, Approve, &noteRecords{fail: errors.New("disk full")}, findings); err == nil {
		t.Fatal("a failed append was ignored")
	}
	records := &noteRecords{}
	if err := note(Verify, Approve, records, []Finding{{Manager: "a", Text: strings.Repeat("x", MaxNoteText+1)}}); err != nil || len(records.notes) != 2 || !strings.Contains(records.notes[1].Context, "(part 2 of 2)") {
		t.Fatal("a long Finding was not continued", len(records.notes), err)
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
