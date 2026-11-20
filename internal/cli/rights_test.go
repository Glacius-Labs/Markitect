package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// stepCheck runs the step check of role on dir, after the step, against
// before, and returns the exit code and the messages of the violations.
func stepCheck(t *testing.T, role, before, after string, flags ...string) (int, []string) {
	t.Helper()
	args := append([]string{"status", "-C", after, "--step", role, "--before", before, "--json"}, flags...)
	code, out, errOut := run(args...)
	var doc checksDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("%v: %s %s", err, out, errOut)
	}
	var msgs []string
	for _, v := range doc.Violations {
		msgs = append(msgs, v.File+": "+v.Message)
	}
	return code, msgs
}

// allowed and denied run the step check and expect no violation, or
// violations that contain each of want.
func allowed(t *testing.T, role, before, after string, flags ...string) {
	t.Helper()
	if code, msgs := stepCheck(t, role, before, after, flags...); code != ExitOK || len(msgs) != 0 {
		t.Errorf("%s: %d %v", role, code, msgs)
	}
}

func denied(t *testing.T, role, before, after string, want ...string) {
	t.Helper()
	code, msgs := stepCheck(t, role, before, after)
	joined := strings.Join(msgs, "\n")
	for _, w := range want {
		if !strings.Contains(joined, w) {
			t.Errorf("%s: missing %q in\n%s", role, w, joined)
		}
	}
	if code != ExitFound || len(msgs) != len(want) {
		t.Errorf("%s: %d %v", role, code, msgs)
	}
}

// stepped is the marked lending project after a step, and a copy of it
// before.
func stepped(t *testing.T) (before, after string) {
	t.Helper()
	after = marked(t)
	write(t, after, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete", "", ""))
	return copyTree(t, after, false), after
}

func remove(t *testing.T, dir, p string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
		t.Fatal(err)
	}
}

func TestStepBuilder(t *testing.T) {
	before, after := stepped(t)
	write(t, after, "src/loans.go", "package lending\n\nfunc Borrow() { count() }\n")
	write(t, after, "src/fines.go", "package lending\n")
	remove(t, after, "src/member.go")
	// Files .gitignore excludes do not land; .markitect/ignore only says what
	// the project does not explain, and an artifact it names still lands.
	write(t, after, "bin/lending.exe", "binary")
	allowed(t, "builder", before, after)
	write(t, after, "notes.txt", "a note")
	write(t, after, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nFour weeks.\n")
	write(t, after, "tests/counter-checks/loans_test.go", "package lending\n")
	write(t, after, ".markitect/settings.md", "# Settings\n")
	write(t, after, ".github/workflows/ci.yml", "on: push\n")
	denied(t, "builder", before, after,
		".github/workflows/ci.yml: builder may not change a control file",
		".markitect/settings.md: builder may not change a control file",
		"docs/spec/lending/return.md: builder may not change the spec",
		"tests/counter-checks/loans_test.go: builder may not change the counter-check folder")
	// The owner may change all of it.
	allowed(t, "owner", before, after)
}

func TestStepCounterCheckerAndReaders(t *testing.T) {
	before, after := stepped(t)
	write(t, after, "tests/counter-checks/return_test.go", "package lending\n")
	allowed(t, "counter-checker", before, after)
	write(t, after, "src/returns.go", "package lending\n")
	denied(t, "counter-checker", before, after, "src/returns.go: counter-checker may not change an artifact")
	for _, role := range []string{"reader", "sampler", "intake"} {
		denied(t, role, before, after, "src/returns.go: "+role+" may not change an artifact", "tests/counter-checks/return_test.go: "+role+" may not change the counter-check folder")
	}
}

func TestStepReviewer(t *testing.T) {
	before, after := stepped(t)
	verdict(t, after, "roles/member.md", "fail", "--file", "src/member.go", "-m", "a member has no card")
	allowed(t, "reviewer", before, after)
	// The same verdict is no step of the architect or the owner.
	denied(t, "architect", before, after,
		".markitect/history.jsonl: architect may not add a verdict event: roles/member.md",
		".markitect/state.jsonl: architect may not write a row with source mark: roles/member.md")
	denied(t, "owner", before, after,
		".markitect/history.jsonl: owner may not add a verdict event: roles/member.md",
		".markitect/state.jsonl: owner may not write a row with source mark: roles/member.md")
	// A row with source move, a history line removed, and a file.
	state := read(t, after, ".markitect/state.jsonl")
	write(t, after, ".markitect/state.jsonl", strings.Replace(state, `"doc":"roles/member.md","verdict":"fail","source":"mark"`, `"doc":"roles/member.md","verdict":"fail","source":"move"`, 1))
	history := read(t, after, ".markitect/history.jsonl")
	write(t, after, ".markitect/history.jsonl", history[strings.Index(history, "\n")+1:])
	write(t, after, "src/member.go", "package lending\n")
	code, msgs := stepCheck(t, "reviewer", before, after)
	joined := strings.Join(msgs, "\n")
	for _, w := range []string{".markitect/state.jsonl: reviewer may not write a row with source move: roles/member.md",
		".markitect/history.jsonl: reviewer may not remove a line of the history", "src/member.go: reviewer may not change an artifact"} {
		if code != ExitFound || !strings.Contains(joined, w) {
			t.Errorf("missing %q in %d\n%s", w, code, joined)
		}
	}
}

func TestStepArchitect(t *testing.T) {
	before, after := stepped(t)
	if code, _, errOut := run("move", "-C", after, "lending/return.md", "lending/give-back.md"); code != ExitOK {
		t.Fatal(code, errOut)
	}
	write(t, after, ".markitect/ignore", ".gitignore\nnotes.txt\ndrafts/\n")
	// The spec, the ignore file, the move's rows and event, and the place
	// the move rewrote.
	allowed(t, "architect", before, after)
	denied(t, "builder", before, after,
		".markitect/history.jsonl: builder may not change the history", ".markitect/ignore: builder may not change the ignore file",
		".markitect/reports/late-fee.md: builder may not change a report", ".markitect/state.jsonl: builder may not change the state",
		"docs/spec/lending/give-back.md: builder may not change the spec", "docs/spec/lending/return.md: builder may not change the spec")
	// A place the step's move did not rewrite.
	before, after = stepped(t)
	write(t, after, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/roles/member.md", "complete", "", ""))
	denied(t, "architect", before, after, ".markitect/reports/late-fee.md: architect may change a report only where a move of the step rewrote its place")
}

func TestStepTriageAndJob(t *testing.T) {
	before, after := stepped(t)
	report := lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete", "escalated\nCharge per day.", "")
	write(t, after, ".markitect/reports/late-fee.md", report+"\n## Rating\nimpact high, likelihood low\n")
	allowed(t, "triage", before, after)
	write(t, after, ".markitect/reports/fee-again.md", "# Again\n\n## Source\nintake, 2026-11-19\n")
	denied(t, "triage", before, after, ".markitect/reports/fee-again.md: triage may not write a new report")
	// The finding is the job's.
	before, after = stepped(t)
	write(t, after, ".markitect/reports/late-fee.md", strings.Replace(read(t, after, ".markitect/reports/late-fee.md"), "is not stated", "is unclear", 1))
	denied(t, "triage", before, after, ".markitect/reports/late-fee.md: triage may not change finding of a report")
	// The job writes new reports with its own source, the prevention and the
	// samples.
	before, after = stepped(t)
	write(t, after, ".markitect/reports/slow-loans.md", lendingReport("job build reviewer, 2026-11-19", "src/loans.go", "none", "", ""))
	write(t, after, ".markitect/reports/late-fee.md", read(t, after, ".markitect/reports/late-fee.md")+"\n## Prevention\nA question on fees.\n")
	write(t, after, ".markitect/samples/first.md", "# First\n")
	allowed(t, "job", before, after)
	write(t, after, ".markitect/reports/an-idea.md", lendingReport("owner, 2026-11-19", "src/loans.go", "none", "", ""))
	denied(t, "job", before, after, ".markitect/reports/an-idea.md: job may not write a report whose source is not a job or intake")
	denied(t, "owner", before, after, ".markitect/reports/slow-loans.md: owner may not write a report whose source is not the owner")
}

func TestStepNarrowedByTheTarget(t *testing.T) {
	before, after := stepped(t)
	target := copyTree(t, before, false)
	write(t, target, ".markitect/settings.md", "# Settings\n\n## Counter-checks\ntests/counter-checks\n\n## Write table\nbuilder: src/*\nbuilder: !src/vendor/\n")
	// Settings changed on the branch are not read: they would widen nothing.
	write(t, after, ".markitect/settings.md", "# Settings\n\n## Counter-checks\nsrc\n")
	write(t, after, "src/loans.go", "package lending\n\nfunc Borrow() { count() }\n")
	write(t, after, "src/vendor/lib.go", "package vendor\n")
	write(t, after, "scripts/build.sh", "make\n")
	code, msgs := stepCheck(t, "builder", before, after, "--target", target)
	want := []string{
		".markitect/settings.md: builder may not change a control file",
		"scripts/build.sh: builder may not change scripts/build.sh: the settings narrow builder to src/*, !src/vendor/",
		"src/vendor/lib.go: builder may not change src/vendor/lib.go: the settings narrow builder to src/*, !src/vendor/",
	}
	if code != ExitFound || strings.Join(msgs, "\n") != strings.Join(want, "\n") {
		t.Errorf("%d\n%s", code, strings.Join(msgs, "\n"))
	}
	// Without --target, the settings before the step count.
	code, msgs = stepCheck(t, "builder", before, after)
	if code != ExitFound || len(msgs) != 1 {
		t.Errorf("%d %v", code, msgs)
	}
	if code, _, errOut := run("status", "-C", after, "--step", "painter", "--before", before); code != ExitFailed || !strings.Contains(errOut, "painter is no role") {
		t.Error(code, errOut)
	}
	if code, _, errOut := run("status", "-C", after, "--step", "builder"); code != ExitFailed || !strings.Contains(errOut, "go together") {
		t.Error(code, errOut)
	}
}

func TestLandingCheck(t *testing.T) {
	target := marked(t)
	land := func(branch string) (int, string) {
		t.Helper()
		code, out, _ := run("status", "-C", branch, "--target", target)
		return code, out
	}
	// What the target shows too is no reason to stay.
	write(t, target, "src/fines.go", "package lending\n")
	branch := copyTree(t, target, false)
	if code, out := land(branch); code != ExitOK || out != "landing: 0 entries the target does not show; the tests and the counter-checks are the runner's to run\n" {
		t.Error(code, out)
	}
	// A changed document and its suspect neighbour, and an unexplained file.
	write(t, branch, "docs/spec/roles/member.md", "# Member\n\nA card holder with a card.\n")
	write(t, branch, "src/holds.go", "package lending\n")
	code, out := land(branch)
	if code != ExitFound || out != "adds changed: roles/member.md: self\nadds suspect: lending/borrow.md: link:roles/member.md\nadds unexplained: src/holds.go\n"+
		"landing: 3 entries the target does not show; the tests and the counter-checks are the runner's to run\n" {
		t.Error(code, out)
	}
	// A document waiting on the owner lands unbuilt.
	branch = copyTree(t, target, false)
	write(t, branch, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete", "escalated\nCharge per day.", ""))
	write(t, branch, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within four weeks.\n")
	if code, out := land(branch); code != ExitOK {
		t.Error(code, out)
	}
	// Unless something changed that the report does not excuse.
	write(t, branch, "src/returns.go", "package lending\n\nfunc Return() { late() }\n")
	if code, out := land(branch); code != ExitFound || !strings.Contains(out, "adds waiting: lending/return.md: self, file:src/returns.go") {
		t.Error(code, out)
	}
	// Errors give no answer; the step check and the landing check combine.
	before := copyTree(t, branch, false)
	write(t, branch, "docs/spec/lending/borrow.md", "# Borrow\n\nSee [fines](fines.md).\n")
	code, out, _ = run("status", "-C", branch, "--target", target, "--step", "builder", "--before", before)
	if code != ExitFound || !strings.Contains(out, "docs/spec/lending/borrow.md: write.denied: builder may not change the spec\nstep builder: 1 change the write table denies\n") ||
		!strings.Contains(out, "landing: no answer while check finds errors or the state has errors\n") {
		t.Error(code, out)
	}
}

func TestMoveRowsMustComeFromAMove(t *testing.T) {
	before, after := stepped(t)
	verdict(t, before, "lending/return.md", "fail", "--file", "src/returns.go", "-m", "a book comes back late")
	after = copyTree(t, before, false)
	if code, _, errOut := run("move", "-C", after, "lending/borrow.md", "loans/borrow.md"); code != ExitOK {
		t.Fatal(code, errOut)
	}
	allowed(t, "architect", before, after)
	// A fail turned into ok, and a row for a new document, each with source
	// move beside a real move.
	state := read(t, after, ".markitect/state.jsonl")
	state = strings.Replace(state, `"doc":"lending/return.md","verdict":"fail","source":"mark"`, `"doc":"lending/return.md","verdict":"ok","source":"move"`, 1)
	write(t, after, "docs/spec/lending/n.md", "# N\n")
	self := strings.SplitN(strings.SplitN(state, `"self":"`, 2)[1], `"`, 2)[0]
	state += `{"doc":"lending/n.md","verdict":"ok","source":"move","inputs":{"self":"` + self + `"},"evidence":[],"reason":""}` + "\n"
	write(t, after, ".markitect/state.jsonl", state)
	denied(t, "architect", before, after,
		".markitect/state.jsonl: architect may not write a row with source move that is not what the moves of the step make of the rows there were: lending/n.md",
		".markitect/state.jsonl: architect may not write a row with source move that is not what the moves of the step make of the rows there were: lending/return.md")
	// Without a move in the step, no row with source move.
	before, after = stepped(t)
	state = read(t, after, ".markitect/state.jsonl")
	write(t, after, ".markitect/state.jsonl", strings.Replace(state, `"doc":"roles/member.md","verdict":"ok","source":"mark"`, `"doc":"roles/member.md","verdict":"ok","source":"move"`, 1))
	denied(t, "owner", before, after, ".markitect/state.jsonl: owner may not write a row with source move that is not what the moves of the step make of the rows there were: roles/member.md")
}

func TestMarkRowsNeedAVerdictEvent(t *testing.T) {
	before, after := stepped(t)
	state := read(t, after, ".markitect/state.jsonl")
	write(t, after, ".markitect/state.jsonl", strings.Replace(state, `"evidence":["src/member.go"]`, `"evidence":["src/member.go:1"]`, 1))
	denied(t, "reviewer", before, after, ".markitect/state.jsonl: reviewer may not write a row with source mark without a verdict event of the step on it with its verdict and inputs: roles/member.md")
	// A verdict event that names other inputs does not explain it either.
	before, after = stepped(t)
	verdict(t, after, "roles/member.md", "ok", "--file", "src/member.go", "-e", "src/member.go:1")
	state = read(t, after, ".markitect/state.jsonl")
	write(t, after, ".markitect/state.jsonl", strings.Replace(state, `"doc":"roles/member.md","verdict":"ok"`, `"doc":"roles/member.md","verdict":"fail"`, 1))
	if code, msgs := stepCheck(t, "reviewer", before, after); code != ExitFound || len(msgs) != 1 {
		t.Error(code, msgs)
	}
}

func TestANewReportOfTheJobHoldsNoTriage(t *testing.T) {
	before, after := stepped(t)
	write(t, after, ".markitect/reports/slow-loans.md", lendingReport("job build reviewer, 2026-11-19", "src/loans.go", "none", "escalated\nFaster loans.", "")+"\n## Rating\nimpact low, likelihood low\n")
	denied(t, "job", before, after, ".markitect/reports/slow-loans.md: job may not write actions, rating in a new report; triage sets them")
}

// repository makes a nested repository at dir/p whose HEAD names
// refs/heads/main at commit.
func repository(t *testing.T, dir, p, commit string) {
	t.Helper()
	for _, sub := range []string{"objects", "refs/heads"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(p), ".git", filepath.FromSlash(sub)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// An empty folder would not survive a copy of the project.
	write(t, dir, p+"/.git/objects/pack/keep", "")
	write(t, dir, p+"/.git/HEAD", "ref: refs/heads/main\n")
	write(t, dir, p+"/.git/refs/heads/main", commit+"\n")
	write(t, dir, p+"/lib.go", "package lib\n")
}

func TestNestedRepositoriesCountByTheirCommit(t *testing.T) {
	_, after := stepped(t)
	repository(t, after, "vendor/lib", strings.Repeat("a", 40))
	write(t, after, ".markitect/ignore", ".gitignore\nnotes.txt\nvendor/\n")
	before := copyTree(t, after, false)
	// What is not committed in it does not land.
	write(t, after, "vendor/lib/lib.go", "package lib\n\nfunc New() {}\n")
	allowed(t, "reviewer", before, after)
	// A new commit, loose or packed, is a change only the builder and the
	// owner make.
	write(t, after, "vendor/lib/.git/refs/heads/main", strings.Repeat("b", 40)+"\n")
	denied(t, "reviewer", before, after, "vendor/lib: reviewer may not change the commit of a nested repository")
	allowed(t, "builder", before, after)
	remove(t, after, "vendor/lib/.git/refs/heads/main")
	write(t, after, "vendor/lib/.git/packed-refs", "# pack-refs with: peeled\n"+strings.Repeat("c", 40)+" refs/heads/main\n")
	denied(t, "architect", before, after, "vendor/lib: architect may not change the commit of a nested repository")
}

func TestControlFilesInAnyCase(t *testing.T) {
	before, after := stepped(t)
	write(t, after, ".Markitect/Settings.md", "# Settings\n")
	write(t, after, ".GitHub/Workflows/ci.yml", "on: push\n")
	code, msgs := stepCheck(t, "builder", before, after)
	joined := strings.Join(msgs, "\n")
	if code != ExitFound || len(msgs) != 2 || !strings.Contains(joined, "builder may not change a control file") || !strings.Contains(joined, "ci.yml: builder may not change a control file") {
		t.Errorf("%d\n%s", code, joined)
	}
}

// appendLine adds a line to a file of the project.
func appendLine(t *testing.T, dir, p, line string) {
	t.Helper()
	write(t, dir, p, read(t, dir, p)+line+"\n")
}

// selfOf is the hash of a document's text as the state holds it.
func selfOf(t *testing.T, dir, doc string) string {
	t.Helper()
	return state.Hash([]byte(read(t, dir, "docs/spec/"+doc)))
}

func TestMoveEventsMustBeMoves(t *testing.T) {
	const denial = "architect may not write a row with source move that is not what the moves of the step make of the rows there were: "
	moveEvent := func(from, to string, rewritten ...string) string {
		list, _ := json.Marshal(append([]string{}, rewritten...))
		return `{"time":"2099-01-01T00:00:00Z","type":"move","from":"` + from + `","to":"` + to + `","rewritten":` + string(list) + `}`
	}
	reself := func(dir, doc, was, now string) {
		state := read(t, dir, ".markitect/state.jsonl")
		line := state[strings.Index(state, `{"doc":"`+doc+`"`):]
		line = line[:strings.Index(line, "\n")]
		fixed := strings.Replace(strings.Replace(line, `"source":"mark"`, `"source":"move"`, 1), was, now, 1)
		write(t, dir, ".markitect/state.jsonl", strings.Replace(state, line, fixed, 1))
	}
	// A: a move of a document onto itself, with an edit and its self.
	before, after := stepped(t)
	was := selfOf(t, after, "lending/return.md")
	write(t, after, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within four weeks.\n")
	appendLine(t, after, ".markitect/history.jsonl", moveEvent("lending/return.md", "lending/return.md", "lending/return.md"))
	reself(after, "lending/return.md", was, selfOf(t, after, "lending/return.md"))
	code, msgs := stepCheck(t, "architect", before, after)
	joined := strings.Join(msgs, "\n")
	if code != ExitFound || !strings.Contains(joined, "lending/return.md -> lending/return.md: it moves a document onto itself") || !strings.Contains(joined, denial+"lending/return.md") {
		t.Errorf("A: %d\n%s", code, joined)
	}
	// B: a copy presented as a move.
	before, after = stepped(t)
	write(t, after, "docs/spec/lending/give-back.md", read(t, after, "docs/spec/lending/return.md"))
	appendLine(t, after, ".markitect/history.jsonl", moveEvent("lending/return.md", "lending/give-back.md"))
	state := read(t, after, ".markitect/state.jsonl")
	line := state[strings.Index(state, `{"doc":"lending/return.md"`):]
	line = line[:strings.Index(line, "\n")]
	write(t, after, ".markitect/state.jsonl", state+strings.Replace(strings.Replace(line, "lending/return.md", "lending/give-back.md", 1), `"source":"mark"`, `"source":"move"`, 1)+"\n")
	code, msgs = stepCheck(t, "architect", before, after)
	joined = strings.Join(msgs, "\n")
	if code != ExitFound || !strings.Contains(joined, "lending/return.md is still there, and a copy is no move") || !strings.Contains(joined, denial+"lending/give-back.md") {
		t.Errorf("B: %d\n%s", code, joined)
	}
	// C: a real move, and an edit of another document with its self
	// refreshed as if the move had made it.
	before, after = stepped(t)
	if code, _, errOut := run("move", "-C", after, "lending/borrow.md", "loans/borrow.md"); code != ExitOK {
		t.Fatal(code, errOut)
	}
	was = selfOf(t, after, "lending/return.md")
	write(t, after, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within four weeks.\n")
	// Without a forged row, the edit is an ordinary change of the spec, and
	// status lists what changed: here on the document that waits on the
	// open report.
	allowed(t, "architect", before, after)
	if waiting, _, _ := waitingOf(t, after); len(waiting) != 1 || waiting[0].Doc != "lending/return.md" || strings.Join(waiting[0].Changes, " ") != "self" {
		t.Errorf("C: %+v", waiting)
	}
	reself(after, "lending/return.md", was, selfOf(t, after, "lending/return.md"))
	denied(t, "architect", before, after, ".markitect/state.jsonl: "+denial+"lending/return.md")
}

func TestOkRowsNeedEvidence(t *testing.T) {
	// D: an ok row without evidence, with its verdict event.
	before, after := stepped(t)
	verdict(t, after, "roles/member.md", "ok", "--file", "src/member.go", "-e", "src/member.go:1")
	write(t, after, ".markitect/state.jsonl", strings.Replace(read(t, after, ".markitect/state.jsonl"), `"evidence":["src/member.go:1"]`, `"evidence":[]`, 1))
	denied(t, "reviewer", before, after, ".markitect/state.jsonl: reviewer may not write an ok row without evidence: roles/member.md")
	// A fail without a reason.
	before, after = stepped(t)
	verdict(t, after, "roles/member.md", "fail", "--file", "src/member.go", "-m", "no card")
	write(t, after, ".markitect/state.jsonl", strings.Replace(read(t, after, ".markitect/state.jsonl"), `"reason":"no card"`, `"reason":""`, 1))
	denied(t, "reviewer", before, after, ".markitect/state.jsonl: reviewer may not write a fail row without a reason: roles/member.md")
}

func TestNestedRefChains(t *testing.T) {
	// E: HEAD names a ref that names another ref; the reviewer moves its end.
	_, after := stepped(t)
	repository(t, after, "vendor/lib", "ref: refs/heads/real")
	write(t, after, "vendor/lib/.git/refs/heads/real", strings.Repeat("a", 40)+"\n")
	before := copyTree(t, after, false)
	write(t, after, "vendor/lib/.git/refs/heads/real", strings.Repeat("b", 40)+"\n")
	denied(t, "reviewer", before, after, "vendor/lib: reviewer may not change the commit of a nested repository")
	// The end in packed-refs counts too.
	remove(t, after, "vendor/lib/.git/refs/heads/real")
	write(t, after, "vendor/lib/.git/packed-refs", strings.Repeat("a", 40)+" refs/heads/real\n")
	allowed(t, "reviewer", before, after)
}

func TestHonestMovesPass(t *testing.T) {
	mv := func(dir, from, to string) {
		t.Helper()
		if code, _, errOut := run("move", "-C", dir, from, to); code != ExitOK {
			t.Fatal(code, errOut)
		}
	}
	// L: a chain of moves; the path between them is nowhere after.
	before, after := stepped(t)
	mv(after, "lending/borrow.md", "middle/borrow.md")
	mv(after, "middle/borrow.md", "loans/borrow.md")
	allowed(t, "architect", before, after)
	// A chain with a forged refreshed row is still denied.
	rows := read(t, after, ".markitect/state.jsonl")
	line := rows[strings.Index(rows, `{"doc":"roles/member.md"`):]
	line = line[:strings.Index(line, "\n")]
	write(t, after, "docs/spec/roles/member.md", "# Member\n\nA card holder with a card.\n")
	forged := strings.Replace(strings.Replace(line, `"source":"mark"`, `"source":"move"`, 1),
		state.Hash([]byte("# Member\n\nA card holder.\n")), selfOf(t, after, "roles/member.md"), 1)
	write(t, after, ".markitect/state.jsonl", strings.Replace(rows, line, forged, 1))
	if code, msgs := stepCheck(t, "architect", before, after); code != ExitFound || len(msgs) != 1 || !strings.Contains(msgs[0], "roles/member.md") {
		t.Errorf("forged chain: %d %v", code, msgs)
	}
	// M: there and back again.
	before, after = stepped(t)
	mv(after, "lending/borrow.md", "lending/b2.md")
	mv(after, "lending/b2.md", "lending/borrow.md")
	allowed(t, "architect", before, after)
	// O: a move, and an edit of a document the move rewrote: an ordinary
	// change of the spec, which the landing check lists as changed.
	before, after = stepped(t)
	mv(after, "lending/borrow.md", "loans/borrow.md")
	write(t, after, "docs/spec/loans/borrow.md", strings.Replace(read(t, after, "docs/spec/loans/borrow.md"), "five", "six", 1))
	allowed(t, "architect", before, after)
	code, out, _ := run("status", "-C", after, "--target", before)
	if code != ExitFound || !strings.Contains(out, "adds changed: loans/borrow.md: self\n") {
		t.Error(code, out)
	}
}
