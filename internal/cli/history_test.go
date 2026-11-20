package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// clock makes the history's clock tick a second per event from a fixed time.
func clock(t *testing.T) {
	t.Helper()
	at := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	saved := now
	now = func() time.Time { at = at.Add(time.Second); return at }
	t.Cleanup(func() { now = saved })
}

func history(t *testing.T, dir string) []state.Event {
	t.Helper()
	events, problems := state.ParseHistory([]byte(read(t, dir, state.HistoryFile)))
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	return events
}

// withFlow is the lending project in which the member links borrowing back
// and borrowing shows a diagram of the spec folder.
func withFlow() map[string]string {
	files := map[string]string{}
	for p, text := range lending {
		files[p] = text
	}
	files["docs/spec/roles/member.md"] = "# Member\n\nA card holder who may [borrow](../lending/borrow.md) books.\n"
	files["docs/spec/lending/borrow.md"] += "\n![The flow](flow.png)\n"
	files["docs/spec/lending/flow.png"] = "\x89PNG flow"
	return files
}

func TestMoveCarriesTheState(t *testing.T) {
	clock(t)
	dir := markedWith(t, withFlow())
	code, out, errOut := run("move", "-C", dir, "lending/borrow.md", "loans/borrow.md")
	if code != ExitOK || out != "moved lending/borrow.md -> loans/borrow.md\nchanged loans/borrow.md\nchanged roles/member.md\n" {
		t.Fatal(code, out, errOut)
	}
	if !strings.Contains(read(t, dir, "docs/spec/loans/borrow.md"), "![The flow](../lending/flow.png)") {
		t.Error(read(t, dir, "docs/spec/loans/borrow.md"))
	}
	if l := statusOf(t, dir); l.code != ExitOK || len(l.summary()) != 0 {
		t.Errorf("not empty after the move: %v", l.summary())
	}
	rows, _ := state.ParseState([]byte(read(t, dir, state.StateFile)))
	sources := map[string]string{}
	for _, r := range rows {
		sources[r.Doc] = r.Source
	}
	if sources["loans/borrow.md"] != "move" || sources["roles/member.md"] != "move" || sources["lending/return.md"] != "mark" || len(sources) != 4 {
		t.Errorf("sources %v", sources)
	}
	events := history(t, dir)
	last := events[len(events)-1]
	if len(events) != 5 || last.Type != state.EventMove || last.From != "lending/borrow.md" || last.To != "loans/borrow.md" ||
		strings.Join(last.Rewritten, " ") != "loans/borrow.md roles/member.md" {
		t.Errorf("%+v", events)
	}
	lines := strings.Split(strings.TrimSpace(read(t, dir, state.HistoryFile)), "\n")
	if want := `{"time":"2026-11-20T09:00:05Z","type":"move","from":"lending/borrow.md","to":"loans/borrow.md","rewritten":["loans/borrow.md","roles/member.md"]}`; lines[4] != want {
		t.Errorf("got  %s\nwant %s", lines[4], want)
	}
	// A move into the same folder rewrites only the links to the document.
	if code, _, errOut := run("move", "-C", dir, "roles/member.md", "roles/card holder.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	if l := statusOf(t, dir); l.code != ExitOK {
		t.Errorf("not empty after the rename: %v", l.summary())
	}
}

func TestMoveKeepsAChangedDocumentChanged(t *testing.T) {
	dir := markedWith(t, withFlow())
	write(t, dir, "docs/spec/roles/member.md", "# Member\n\nA card holder who may [borrow](../lending/borrow.md) books, five at most.\n")
	before := statusOf(t, dir).summary()
	same(t, "before the move", before, []string{"changed roles/member.md: self", "suspect lending/borrow.md: link:roles/member.md"})
	if code, _, errOut := run("move", "-C", dir, "lending/borrow.md", "loans/borrow.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	same(t, "after the move", statusOf(t, dir).summary(), []string{"changed roles/member.md: self", "suspect loans/borrow.md: link:roles/member.md"})
}

func TestMarkAppendsOneVerdictEvent(t *testing.T) {
	clock(t)
	dir := project(t, lending)
	verdict(t, dir, "lending/borrow.md", "ok", "--file", "src/loans.go", "-e", "src/loans.go")
	lines := strings.Split(strings.TrimSpace(read(t, dir, state.HistoryFile)), "\n")
	if len(lines) != 1 {
		t.Fatalf("%d lines", len(lines))
	}
	var e struct {
		Time, Type, Doc, Verdict, Self string
		Changed                        []map[string]string
	}
	if err := json.Unmarshal([]byte(lines[0]), &e); err != nil {
		t.Fatal(err)
	}
	var changed []string
	for _, c := range e.Changed {
		if len(c["self"]) != 71 {
			t.Errorf("no self for %v", c)
		}
		changed = append(changed, c["doc"])
	}
	if e.Time != "2026-11-20T09:00:01Z" || e.Type != "verdict" || e.Doc != "lending/borrow.md" || e.Verdict != "ok" || len(e.Self) != 71 {
		t.Error(lines[0])
	}
	same(t, "the documents changed beside it", changed, []string{"lending/return.md", "roles/member.md", "types/operation.md"})
	if !strings.HasPrefix(lines[0], `{"time":"2026-11-20T09:00:01Z","type":"verdict","doc":"lending/borrow.md","verdict":"ok","self":"sha256:`) {
		t.Error(lines[0])
	}
	// A gone document's verdict has no self.
	if err := os.Remove(filepath.Join(specOf(dir), "lending", "borrow.md")); err != nil {
		t.Fatal(err)
	}
	verdict(t, dir, "lending/borrow.md", "ok")
	events := history(t, dir)
	if len(events) != 2 || events[1].Self != "" || events[1].Doc != "lending/borrow.md" {
		t.Errorf("%+v", events)
	}
}

func TestJoinedHistoriesReadAsOne(t *testing.T) {
	clock(t)
	dir := marked(t)
	base := read(t, dir, state.HistoryFile)
	verdict(t, dir, "roles/member.md", "skip", "--file", "src/member.go", "-m", "no members yet")
	ours := read(t, dir, state.HistoryFile)
	write(t, dir, state.HistoryFile, base)
	if code, _, errOut := run("move", "-C", dir, "lending/return.md", "returns/return.md"); code != ExitOK {
		t.Fatal(errOut)
	}
	theirs := read(t, dir, state.HistoryFile)
	write(t, dir, state.HistoryFile, ours+theirs)
	events := history(t, dir)
	var kinds []string
	for _, e := range events {
		kinds = append(kinds, e.Type+" "+e.Doc+e.From)
	}
	same(t, "the joined history", kinds, []string{
		"verdict lending/borrow.md", "verdict lending/return.md", "verdict roles/member.md", "verdict types/operation.md",
		"verdict roles/member.md", "move lending/return.md"})
}

// TestFailedMoveChangesNothing fails each step of a move in turn and checks
// that the spec, the state and the history are as they were.
func TestFailedMoveChangesNothing(t *testing.T) {
	args := func(dir string) []string { return []string{"move", "-C", dir, "lending/borrow.md", "loans/borrow.md"} }
	steps := 0
	t.Cleanup(func() { fault = nil })
	dir := markedWith(t, withFlow())
	fault = func(int) error { steps++; return nil }
	if code, _, errOut := run(args(dir)...); code != ExitOK {
		t.Fatal(errOut)
	}
	if steps < 8 {
		t.Fatalf("only %d steps", steps)
	}
	for k := 1; k <= steps; k++ {
		fault = nil
		dir := markedWith(t, withFlow())
		before := tree(t, dir)
		contents := map[string]string{}
		for _, p := range before {
			if !strings.HasSuffix(p, "/") {
				contents[p] = read(t, dir, p)
			}
		}
		fault = func(step int) error {
			if step == k {
				return errors.New("injected")
			}
			return nil
		}
		if code, out, errOut := run(args(dir)...); code != ExitFailed || out != "" || !strings.Contains(errOut, "injected") {
			t.Errorf("step %d: %d %q %q", k, code, out, errOut)
		}
		fault = nil
		if after := tree(t, dir); strings.Join(after, " ") != strings.Join(before, " ") {
			t.Errorf("step %d left %v", k, after)
		}
		for p, text := range contents {
			if read(t, dir, p) != text {
				t.Errorf("step %d changed %s", k, p)
			}
		}
	}
}

func TestMoveRefusesErrors(t *testing.T) {
	dir := marked(t)
	write(t, dir, state.StateFile, read(t, dir, state.StateFile)+"not json\n")
	before := read(t, dir, "docs/spec/roles/member.md")
	if code, _, errOut := run("move", "-C", dir, "roles/member.md", "people/member.md"); code != ExitFailed || !strings.Contains(errOut, "state.unreadable") {
		t.Error(code, errOut)
	}
	if read(t, dir, "docs/spec/roles/member.md") != before {
		t.Error("a refused move changed the spec")
	}
}

func TestLookAtNamesChangedAssets(t *testing.T) {
	dir := markedWith(t, withFlow())
	write(t, dir, "docs/spec/lending/flow.png", "\x89PNG new flow")
	e, _ := statusOf(t, dir).entry("lending/borrow.md")
	same(t, "where to look", e.LookAt, []string{"docs/spec/lending/borrow.md", "src/loans.go", "tests/counter-checks/loans_test.go", "tests/counter-checks/loans_test.go:3", "docs/spec/lending/flow.png"})
}

// mark writes the state and the history together: when the history cannot
// be written, neither is.
func TestMarkWritesStateAndHistoryTogether(t *testing.T) {
	dir := marked(t)
	fp := fingerprint(t, dir, "roles/member.md")
	file := filepath.Join(dir, filepath.FromSlash(state.HistoryFile))
	if err := os.Chmod(file, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(file, 0o644) })
	if f, err := os.OpenFile(file, os.O_WRONLY, 0); err == nil {
		f.Close()
		t.Skip("the file system lets this user write a read-only file")
	}
	stateBefore, historyBefore := read(t, dir, state.StateFile), read(t, dir, state.HistoryFile)
	code, out, errOut := run("mark", "-C", dir, "roles/member.md@"+fp, "skip", "--file", "src/member.go", "-m", "no members yet")
	if code != ExitFailed || out != "" || !strings.Contains(errOut, "history.jsonl: cannot be written") {
		t.Error(code, out, errOut)
	}
	if read(t, dir, state.StateFile) != stateBefore || read(t, dir, state.HistoryFile) != historyBefore {
		t.Error("a failed mark wrote the state or the history")
	}
}

// status names a line of the history it cannot read as a note.
func TestStatusNotesUnreadableHistory(t *testing.T) {
	dir := marked(t)
	write(t, dir, state.HistoryFile, read(t, dir, state.HistoryFile)+"not json\n")
	code, out, _ := run("status", "-C", dir)
	if code != ExitOK || !strings.HasPrefix(out, "note: .markitect/history.jsonl:5: history.unreadable: ") ||
		!strings.HasSuffix(out, "0 changed, 0 suspect, 0 unexplained\n") {
		t.Error(code, out)
	}
	_, out, _ = run("status", "-C", dir, "--json")
	var doc struct {
		History []diagnostic `json:"history"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil || len(doc.History) != 1 || doc.History[0].Code != "history.unreadable" {
		t.Error(out)
	}
}
