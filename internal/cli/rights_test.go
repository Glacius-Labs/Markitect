package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
