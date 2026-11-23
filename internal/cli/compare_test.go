package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// copyTree copies a project folder; crlf turns every line end into CRLF.
func copyTree(t *testing.T, from string, crlf bool) string {
	t.Helper()
	to := t.TempDir()
	err := filepath.WalkDir(from, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		text := read(t, from, filepath.ToSlash(rel))
		if crlf && !strings.HasSuffix(p, ".png") {
			text = strings.ReplaceAll(text, "\n", "\r\n")
		}
		write(t, to, filepath.ToSlash(rel), text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return to
}

// before and after are the lending project at a confirmed state and after a
// series of edits, confirms and moves.
func beforeAndAfter(t *testing.T) (string, string) {
	t.Helper()
	clock(t)
	older := confirmedWith(t, withFlow())
	write(t, older, "docs/spec/lending/desk.md", "# Desk\n\nThe desk takes each [return](return.md).\n")
	write(t, older, "docs/spec/lending/holds.md", "# Holds\n\nA member may hold a book until its [return](return.md).\n")
	verdict(t, older, "lending/desk.md", "ok", "-e", "src/returns.go")
	verdict(t, older, "lending/holds.md", "ok", "-e", "src/returns.go")
	newer := copyTree(t, older, false)
	if code, _, errOut := run("move", "-C", newer, "lending/borrow.md", "loans/borrow.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	write(t, newer, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within four weeks.\n")
	if err := os.Remove(filepath.Join(newer, "docs", "spec", "lending", "holds.md")); err != nil {
		t.Fatal(err)
	}
	write(t, newer, "docs/spec/lending/fines.md", "# Fines\n\nA late book costs a fee.\n")
	write(t, newer, "src/fines.go", "package lending\n")
	verdict(t, newer, "lending/return.md", "fail", "--file", "src/returns.go", "-m", "still three weeks")
	return older, newer
}

func TestCompare(t *testing.T) {
	older, newer := beforeAndAfter(t)
	code, out, errOut := run("compare", "-C", newer, older, "--json")
	if code != ExitFound {
		t.Fatal(code, out, errOut)
	}
	var doc compareDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	same(t, "new", doc.New, []string{"lending/fines.md"})
	same(t, "gone", doc.Gone, []string{"lending/holds.md"})
	// The move rewrote the moved document and the member's link to it, and
	// nothing else: neither is changed, and neither makes a neighbour suspect.
	if len(doc.Moved) != 1 || doc.Moved[0].From != "lending/borrow.md" || doc.Moved[0].To != "loans/borrow.md" || !doc.Moved[0].Mechanical || doc.Moved[0].Diff != "" {
		t.Errorf("moved %+v", doc.Moved)
	}
	same(t, "rewritten", doc.Rewritten, []string{"roles/member.md"})
	var changed []string
	for _, c := range doc.Changed {
		changed = append(changed, c.Doc)
	}
	same(t, "changed", changed, []string{"lending/return.md"})
	if !strings.Contains(doc.Changed[0].Diff, "--- docs/spec/lending/return.md\n+++ docs/spec/lending/return.md\n@@ -3,4 +3,4 @@\n") ||
		!strings.Contains(doc.Changed[0].Diff, "-A book comes back within three weeks.\n+A book comes back within four weeks.\n") {
		t.Error(doc.Changed[0].Diff)
	}
	if len(doc.Suspect) != 1 || doc.Suspect[0].Doc != "lending/desk.md" || strings.Join(doc.Suspect[0].Because, " ") != "link:lending/return.md" {
		t.Errorf("suspect %+v", doc.Suspect)
	}
	if len(doc.Verdicts) != 1 || doc.Verdicts[0].Doc != "lending/return.md" || doc.Verdicts[0].OldVerdict != "ok" || doc.Verdicts[0].NewVerdict != "fail" {
		t.Errorf("verdicts %+v", doc.Verdicts)
	}
	same(t, "unexplained", doc.Unexplained, []string{"src/fines.go"})
	// The text says the same, and is the same at each run and with CRLF.
	_, text, _ := run("compare", older, newer)
	for _, part := range []string{"## New (1)\n\nlending/fines.md\n", "## Gone (1)\n\nlending/holds.md\n", "## Moved (1)\n\nlending/borrow.md -> loans/borrow.md (links rewritten only)\n", "## Rewritten (1)\n\nroles/member.md (links rewritten only)\n",
		"## Suspect (1)\n\nlending/desk.md: link:lending/return.md\n", "## Verdicts (1)\n\nlending/return.md: ok -> fail; artifacts src/returns.go -> src/returns.go\n", "## Unexplained (1)\n\nsrc/fines.go\n"} {
		if !strings.Contains(text, part) {
			t.Errorf("missing %q in\n%s", part, text)
		}
	}
	if _, again, _ := run("compare", older, newer); again != text {
		t.Error("two runs differ")
	}
	if _, crlf, _ := run("compare", copyTree(t, older, true), copyTree(t, newer, true)); crlf != text {
		t.Errorf("CRLF differs:\n%s", crlf)
	}
	// A project against itself, or its CRLF copy, differs in nothing.
	code, out, _ = run("compare", older, copyTree(t, older, true))
	if code != ExitOK || strings.Count(out, "(none)") != 9 {
		t.Error(code, out)
	}
	if code, _, errOut := run("compare", t.TempDir()); code != ExitFailed || !strings.Contains(errOut, "holds no .markitect/version") {
		t.Error(code, errOut)
	}
}

// The change signals of the history stand in the topic of the whole spec, after
// the signals of the spec itself.
func TestTheTopicShowsTheChangeSignalsOfTheLendingProject(t *testing.T) {
	_, newer := beforeAndAfter(t)
	code, out, _ := run("topic", "-C", newer)
	// The confirms of the confirmed project are one change, the confirms after
	// the edits two more; the move is left out.
	if code != ExitOK || !strings.HasPrefix(out, "# Signals of the spec\n") || !strings.Contains(out, "\n# Change signals of the history\n\n3 changes from 7 verdicts; moves left out, since they are mechanical: 1\n") ||
		!strings.Contains(out, "\n### wide-change (3)\n\nreference: median 2, range 2–4 documents per change\n4 documents changed together at 2026-11-20T09:00:01Z") ||
		!strings.Contains(out, "\n### hotspot (3)\n\nreference: changes per document: median 1; documents linking to one: median 0.5\nlending/return.md: changed 2 times × linked from 1 document = 2\n") ||
		!strings.Contains(out, "\n### weakening\n\nnot computed: a strong statement gathering conditions shows only in its old and new text, which an agent reads from Git; the history holds hashes\n") {
		t.Error(code, out)
	}
	_, again, _ := run("topic", "-C", copyTree(t, newer, true))
	if again != out {
		t.Errorf("CRLF differs:\n%s", again)
	}
	_, js, _ := run("topic", "-C", newer, "--json")
	var doc topicDocument
	if err := json.Unmarshal([]byte(js), &doc); err != nil || doc.Kind != "topic" || doc.Change == nil || doc.Change.Moves != 1 || len(doc.Change.Signals) != 5 || doc.Change.Weakening == "" {
		t.Error(js)
	}
	// The change signals belong to the topic of the whole spec alone.
	if _, js, _ := run("topic", "-C", newer, "lending/return", "--json"); strings.Contains(js, "change_signals") {
		t.Error(js)
	}
}

// A changed file of the spec folder makes the documents that link it
// suspect; a line either side cannot read is a note.
func TestCompareAssetsAndNotes(t *testing.T) {
	older := confirmedWith(t, withFlow())
	newer := copyTree(t, older, false)
	write(t, newer, "docs/spec/lending/flow.png", "\x89PNG another flow")
	write(t, newer, ".markitect/history.jsonl", read(t, newer, ".markitect/history.jsonl")+"not json\n")
	code, out, _ := run("compare", older, newer)
	if code != ExitFound || !strings.HasPrefix(out, "# Comparison\n\nnote: new: .markitect/history.jsonl:5: history.unreadable: ") ||
		!strings.Contains(out, "## Suspect (1)\n\nlending/borrow.md: asset:lending/flow.png\n") {
		t.Error(code, out)
	}
}

// Only a changed text counts: a document recorded fail stays in the changed
// list of later verdicts, but counts in no later change while its text stays.
func TestChangeSignalsCountOnlyChangedText(t *testing.T) {
	clock(t)
	dir := confirmed(t)
	verdict(t, dir, "roles/member.md", "fail", "--file", "src/member.go", "-m", "no card numbers yet")
	saved := now
	at := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	now = func() time.Time { at = at.Add(2 * time.Hour); return at }
	t.Cleanup(func() { now = saved })
	// An edited artifact changes no text: no change.
	write(t, dir, "src/returns.go", "package lending\n\nfunc Return() { late() }\n")
	verdict(t, dir, "lending/return.md", "ok", "--file", "src/returns.go", "-e", "src/returns.go")
	write(t, dir, "docs/spec/lending/return.md", read(t, dir, "docs/spec/lending/return.md")+"\nA late book costs a fee.\n")
	verdict(t, dir, "lending/return.md", "ok", "--file", "src/returns.go", "-e", "src/returns.go")
	write(t, dir, "docs/spec/types/operation.md", read(t, dir, "docs/spec/types/operation.md")+"\nIt happens at the desk.\n")
	verdict(t, dir, "types/operation.md", "ok", "-e", "docs/spec/types/operation.md")
	_, out, _ := run("topic", "-C", dir, "--all")
	for _, part := range []string{
		"3 changes from 8 verdicts;",
		"4 documents changed together at 2026-11-20T09:00:01Z (",
		"1 document changed together at 2026-11-21T13:00:00Z (median 1, range 1–4 documents per change): lending/return.md\n",
		"1 document changed together at 2026-11-21T15:00:00Z (median 1, range 1–4 documents per change): types/operation.md\n",
		"roles/member.md: changed 1 time × linked from 1 document = 1\n",
	} {
		if !strings.Contains(out, part) {
			t.Errorf("missing %q in\n%s", part, out)
		}
	}
}

// A document a move rewrote that also changed otherwise is changed, and
// makes its neighbours suspect.
func TestRewrittenAndChanged(t *testing.T) {
	clock(t)
	older := confirmedWith(t, withFlow())
	newer := copyTree(t, older, false)
	if code, _, errOut := run("move", "-C", newer, "lending/borrow.md", "loans/borrow.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	write(t, newer, "docs/spec/roles/member.md", read(t, newer, "docs/spec/roles/member.md")+"\nA member shows a card.\n")
	write(t, newer, "docs/spec/loans/borrow.md", read(t, newer, "docs/spec/loans/borrow.md")+"\nFive books at most.\n")
	code, out, _ := run("compare", older, newer, "--json")
	var doc compareDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitFound {
		t.Fatal(code, out)
	}
	var changed []string
	for _, c := range doc.Changed {
		changed = append(changed, c.Doc)
	}
	same(t, "changed", changed, []string{"roles/member.md"})
	if len(doc.Moved) != 1 || doc.Moved[0].Mechanical || !strings.Contains(doc.Moved[0].Diff, "+Five books at most.\n") {
		t.Errorf("moved %+v", doc.Moved)
	}
	same(t, "rewritten", doc.Rewritten, []string{})
}

// A document moved without a change of its own is suspect when a document
// it links changed, as status says.
func TestMovedDocumentIsSuspect(t *testing.T) {
	clock(t)
	older := confirmedWith(t, withFlow())
	newer := copyTree(t, older, false)
	if code, _, errOut := run("move", "-C", newer, "lending/borrow.md", "loans/borrow.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	write(t, newer, "docs/spec/types/operation.md", read(t, newer, "docs/spec/types/operation.md")+"\nIt happens at the desk.\n")
	code, out, _ := run("compare", older, newer, "--json")
	var doc compareDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitFound {
		t.Fatal(code, out)
	}
	var suspects []string
	for _, s := range doc.Suspect {
		suspects = append(suspects, s.Doc+": "+strings.Join(s.Because, ", "))
	}
	same(t, "suspect", suspects, []string{"lending/return.md: type:types/operation.md", "loans/borrow.md: type:types/operation.md"})
	if s := statusOf(t, newer).summary(); !strings.Contains(strings.Join(s, "\n"), "suspect loans/borrow.md: type:types/operation.md") {
		t.Error(s)
	}
}

// Like the signals of the spec, each change signal shows its first findings
// and says how many more there are; --all shows every one, and the JSON
// document always holds every one.
func TestAllShowsEveryChangeSignal(t *testing.T) {
	clock(t)
	dir := confirmed(t)
	saved := now
	at := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	now = func() time.Time { at = at.Add(2 * time.Hour); return at }
	t.Cleanup(func() { now = saved })
	// Twelve more changes, each of one document and an hour or more apart,
	// make thirteen with the first.
	for i := 0; i < 12; i++ {
		write(t, dir, "docs/spec/roles/member.md", read(t, dir, "docs/spec/roles/member.md")+"\nOne more line.\n")
		verdict(t, dir, "roles/member.md", "ok", "--file", "src/member.go", "-e", "src/member.go")
	}
	count := func(out string) int { return strings.Count(out, " changed together at ") }
	_, short, _ := run("topic", "-C", dir)
	if n := count(short); n != 10 || !strings.Contains(short, "… 3 more; --all shows them\n") {
		t.Errorf("without --all: %d lines\n%s", n, short)
	}
	_, long, _ := run("topic", "-C", dir, "--all")
	if n := count(long); n != 13 || strings.Contains(long[strings.Index(long, "# Change signals"):], "--all shows them") {
		t.Errorf("with --all: %d lines", n)
	}
	_, js, _ := run("topic", "-C", dir, "--json")
	var doc topicDocument
	if err := json.Unmarshal([]byte(js), &doc); err != nil || doc.Change == nil || len(doc.Change.Signals["wide-change"]) != 13 {
		t.Errorf("%v %s", err, js)
	}
}

// A document that NEW moves back to the path OLD's own history moved it away
// from is moved, not new and gone: only the moves OLD lacks count.
func TestCompareFindsADocumentMovedBack(t *testing.T) {
	clock(t)
	older := confirmedWith(t, withFlow())
	if code, _, errOut := run("move", "-C", older, "lending/borrow.md", "loans/borrow.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	newer := copyTree(t, older, false)
	if code, _, errOut := run("move", "-C", newer, "loans/borrow.md", "lending/borrow.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	code, out, _ := run("compare", older, newer, "--json")
	var doc compareDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitFound {
		t.Fatal(code, out)
	}
	if len(doc.New) != 0 || len(doc.Gone) != 0 || len(doc.Moved) != 1 || doc.Moved[0].From != "loans/borrow.md" || doc.Moved[0].To != "lending/borrow.md" {
		t.Errorf("new %v gone %v moved %+v", doc.New, doc.Gone, doc.Moved)
	}
}

// The moves of a history are followed in order, each applied to the document
// that held the path at that moment, also when a name is vacated and taken by
// another document: fines becomes late-fees, loans takes the name fines, and
// late-fees moves up a folder.
func TestCompareFollowsASwapOfNames(t *testing.T) {
	clock(t)
	older := project(t, map[string]string{
		"docs/spec/components/fines.md": "# Fines\n\nA late book costs a fee.\n",
		"docs/spec/components/loans.md": "# Loans\n\nA loan runs three weeks; see [fines](fines.md).\n",
		"src/x.go":                      "package lending\n",
	})
	verdict(t, older, "components/fines.md", "ok", "-e", "src/x.go")
	verdict(t, older, "components/loans.md", "ok", "-e", "src/x.go")
	newer := copyTree(t, older, false)
	for _, m := range [][2]string{
		{"components/fines.md", "components/late-fees.md"},
		{"components/loans.md", "components/fines.md"},
		{"components/late-fees.md", "late-fees.md"},
	} {
		if code, _, errOut := run("move", "-C", newer, m[0], m[1]); code != ExitOK {
			t.Fatal(errOut)
		}
	}
	code, out, _ := run("compare", older, newer, "--json")
	var doc compareDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitFound {
		t.Fatal(code, out)
	}
	var moved []string
	for _, m := range doc.Moved {
		moved = append(moved, m.From+" -> "+m.To)
	}
	same(t, "moved", moved, []string{"components/loans.md -> components/fines.md", "components/fines.md -> late-fees.md"})
	if len(doc.New) != 0 || len(doc.Gone) != 0 || len(doc.Changed) != 0 || len(doc.Suspect) != 0 {
		t.Errorf("new %v gone %v changed %v suspect %v", doc.New, doc.Gone, doc.Changed, doc.Suspect)
	}
	for _, m := range doc.Moved {
		if !m.Mechanical || m.Diff != "" {
			t.Errorf("a move that changed no text shows a diff: %+v", m)
		}
	}
	same(t, "rewritten", doc.Rewritten, []string{})
}
