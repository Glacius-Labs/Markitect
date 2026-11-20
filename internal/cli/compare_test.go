package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
// series of edits, marks and moves.
func beforeAndAfter(t *testing.T) (string, string) {
	t.Helper()
	clock(t)
	older := markedWith(t, withFlow())
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
	if len(doc.Moved) != 1 || doc.Moved[0].From != "lending/borrow.md" || doc.Moved[0].To != "loans/borrow.md" ||
		!strings.Contains(doc.Moved[0].Diff, "-![The flow](flow.png)\n+![The flow](../lending/flow.png)\n") {
		t.Errorf("moved %+v", doc.Moved)
	}
	var changed []string
	for _, c := range doc.Changed {
		changed = append(changed, c.Doc)
	}
	same(t, "changed", changed, []string{"lending/return.md", "roles/member.md"})
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
	for _, part := range []string{"## New (1)\n\nlending/fines.md\n", "## Gone (1)\n\nlending/holds.md\n", "## Moved (1)\n\nlending/borrow.md -> loans/borrow.md\n--- docs/spec/lending/borrow.md\n+++ docs/spec/loans/borrow.md\n",
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
	if code != ExitOK || strings.Count(out, "(none)") != 7 {
		t.Error(code, out)
	}
	if code, _, errOut := run("compare", t.TempDir()); code != ExitFailed || !strings.Contains(errOut, "holds no .markitect/version") {
		t.Error(code, errOut)
	}
}

func TestFiguresOfTheLendingProject(t *testing.T) {
	_, newer := beforeAndAfter(t)
	code, out, _ := run("figures", "-C", newer)
	// The marks of the confirmed project are one change, the marks after
	// the edits two more; the move is left out.
	if code != ExitOK || !strings.HasPrefix(out, "# Figures from the history\n\n3 changes from 7 verdicts; 1 moves left out, since they are mechanical.\n") ||
		!strings.Contains(out, "\n### wide-change (3)\n\nreference: median 3, range 2–4 documents per change\n4 documents changed together at 2026-11-20T09:00:01Z") ||
		!strings.Contains(out, "\n### hotspot (3)\n\nreference: changes per document: median 1; documents linking to one: median 0.5\nlending/return.md: changed 2 times, linked from 1 documents, 2\n") ||
		!strings.HasSuffix(out, "\n### weakening\n\nnot computed: a strong statement gathering conditions shows only in its old and new text, which an agent reads from Git; the history holds hashes\n") {
		t.Error(code, out)
	}
	_, again, _ := run("figures", "-C", copyTree(t, newer, true))
	if again != out {
		t.Errorf("CRLF differs:\n%s", again)
	}
	_, js, _ := run("figures", "-C", newer, "--json")
	var doc figuresDocument
	if err := json.Unmarshal([]byte(js), &doc); err != nil || doc.Kind != "figures" || doc.Moves != 1 || len(doc.Signals) != 5 || doc.Weakening == "" {
		t.Error(js)
	}
}
