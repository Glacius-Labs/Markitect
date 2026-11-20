package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// lendingReport is a report on the lending project, from its sections in
// the order of the report type; an empty value leaves the section out.
func lendingReport(source, place, property, actions, closing string) string {
	text := "# A report\n\n## Source\n" + source + "\n\n## Place\n" + place + "\n\n## Finding\nThe fee for a late book is not stated.\n\n" +
		"## Occurred or risk\nrisk\n\n## Violated property\n" + property + "\n"
	if actions != "" {
		text += "\n## Actions\n" + actions + "\n"
	}
	if closing != "" {
		text += "\n## Closing\n" + closing + "\n"
	}
	return text
}

// waitingList is the waiting part of the JSON document of status.
type waitingList struct {
	Waiting []waitingJSON `json:"waiting"`
	Errors  []diagnostic  `json:"errors"`
}

func waitingOf(t *testing.T, dir string) ([]waitingJSON, []diagnostic, int) {
	t.Helper()
	code, out, errOut := run("status", "-C", dir, "--json")
	var l waitingList
	if err := json.Unmarshal([]byte(out), &l); err != nil {
		t.Fatalf("%v: %s %s", err, out, errOut)
	}
	return l.Waiting, l.Errors, code
}

func TestInitWritesTheSettings(t *testing.T) {
	dir := initialized(t)
	if got := read(t, dir, ".markitect/settings.md"); got != state.DefaultSettings {
		t.Error(got)
	}
	if code, out, _ := run("check", "-C", dir); code != ExitOK || out != "0 errors in 0 documents\n" {
		t.Error(code, out)
	}
}

func TestWaitingOnReports(t *testing.T) {
	dir := marked(t)
	place := "docs/spec/lending/return.md"
	write(t, dir, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", place, "complete", "", ""))
	// A confirmed document waits on an open report about the spec, which
	// counts like any entry.
	code, out, _ := run("status", "-C", dir)
	if code != ExitFound || !strings.Contains(out, "waiting: lending/return.md@") || !strings.HasSuffix(out, ": report late-fee\n0 changed, 0 suspect, 0 unexplained, 1 waiting\n") {
		t.Error(code, out)
	}
	// A changed document waits too, and is listed only as waiting, with what
	// moved.
	write(t, dir, place, "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within four weeks.\n")
	l := statusOf(t, dir)
	waiting, _, _ := waitingOf(t, dir)
	if len(l.Changed) != 0 || len(waiting) != 1 || strings.Join(waiting[0].Changes, " ") != "self" || len(waiting[0].Fingerprint) != 8 {
		t.Errorf("%v %+v", l.summary(), waiting)
	}
	// A verdict event leaves it out of the documents listed as changed.
	verdict(t, dir, "roles/member.md", "ok", "--file", "src/member.go", "-e", "src/member.go")
	if events := history(t, dir); len(events[len(events)-1].Changed) != 0 {
		t.Errorf("%+v", events[len(events)-1])
	}
	// Escalated, it waits on the owner, which does not make status fail.
	write(t, dir, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", place, "complete", "escalated\nCharge per day.\nCharge once.", ""))
	code, out, _ = run("status", "-C", dir)
	if code != ExitOK || !strings.Contains(out, ": report late-fee (escalated: waits on the owner): self\n") || !strings.HasSuffix(out, ", 1 waiting, 1 of them on the owner\n") {
		t.Error(code, out)
	}
	// Waiting excuses only building it: a changed artifact counts, also on
	// the owner.
	write(t, dir, "src/returns.go", "package lending\n\nfunc Return() { late() }\n")
	code, out, _ = run("status", "-C", dir)
	if code != ExitFound || !strings.Contains(out, ": report late-fee (escalated: waits on the owner): self, file:src/returns.go, evidence:src/returns.go\n") {
		t.Error(code, out)
	}
	write(t, dir, "src/returns.go", lending["src/returns.go"])
	// A second open report that is not escalated makes it count again.
	write(t, dir, ".markitect/reports/fee-limit.md", lendingReport("owner, 2026-11-19", "[Return](../../"+place+")", "consistent", "", ""))
	waiting, _, code = waitingOf(t, dir)
	if code != ExitFound || len(waiting) != 1 || strings.Join(waiting[0].Reports, " ") != "fee-limit late-fee" || waiting[0].Escalated {
		t.Errorf("%d %+v", code, waiting)
	}
	// Closed, or about an artifact that deviates from a spec that holds, a
	// report keeps nothing waiting.
	write(t, dir, ".markitect/reports/fee-limit.md", lendingReport("owner, 2026-11-19", place, "consistent", "", "2026-11-20: nothing to do\nThe limit is stated."))
	write(t, dir, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", place, "none", "", ""))
	if waiting, _, _ = waitingOf(t, dir); len(waiting) != 0 {
		t.Errorf("%+v", waiting)
	}
	if l := statusOf(t, dir); strings.Join(l.summary(), " ") != "changed lending/return.md: self" {
		t.Error(l.summary())
	}
	// The neighbours of a waiting document are listed as before.
	write(t, dir, ".markitect/reports/card.md", lendingReport("owner, 2026-11-20", "docs/spec/roles/member.md", "correct", "", ""))
	write(t, dir, "docs/spec/roles/member.md", "# Member\n\nA card holder with a library card.\n")
	l = statusOf(t, dir)
	if e, kind := l.entry("lending/borrow.md"); kind != "suspect" || strings.Join(e.Changes, " ") != "link:roles/member.md" {
		t.Errorf("%v", l.summary())
	}
	if _, kind := l.entry("roles/member.md"); kind != "" {
		t.Errorf("member is %s", kind)
	}
}

func TestMoveRewritesThePlaceOfReports(t *testing.T) {
	clock(t)
	dir := marked(t)
	write(t, dir, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "- `docs/spec/lending/return.md`", "complete", "", ""))
	write(t, dir, ".markitect/reports/fee-limit.md", strings.ReplaceAll(
		lendingReport("owner, 2026-11-19", "[Return](../../docs/spec/lending/return.md#rule)", "consistent", "", "2026-11-20: nothing to do\nStated."), "\n", "\r\n"))
	write(t, dir, ".markitect/reports/borrow-limit.md", lendingReport("owner, 2026-11-19", "docs/spec/lending/borrow.md", "consistent", "", ""))
	code, out, errOut := run("move", "-C", dir, "lending/return.md", "loans/give-back.md")
	if code != ExitOK || !strings.Contains(out, "changed .markitect/reports/fee-limit.md\nchanged .markitect/reports/late-fee.md\n") || strings.Contains(out, "borrow-limit") {
		t.Fatal(code, out, errOut)
	}
	// A path keeps its form; a link takes the path relative to the report,
	// with its section, and the line ends stay.
	if got := read(t, dir, ".markitect/reports/late-fee.md"); !strings.Contains(got, "\n- `docs/spec/loans/give-back.md`\n") {
		t.Error(got)
	}
	if got := read(t, dir, ".markitect/reports/fee-limit.md"); !strings.Contains(got, "\r\n[Return](../../docs/spec/loans/give-back.md#rule)\r\n") {
		t.Error(got)
	}
	events := history(t, dir)
	if last := events[len(events)-1]; last.Type != state.EventMove || strings.Join(last.Reports, " ") != ".markitect/reports/fee-limit.md .markitect/reports/late-fee.md" ||
		strings.Contains(strings.Join(last.Rewritten, " "), ".markitect") {
		t.Errorf("%+v", last)
	}
	if waiting, errs, _ := waitingOf(t, dir); len(errs) != 0 || len(waiting) != 2 || waiting[1].Doc != "loans/give-back.md" {
		t.Errorf("%+v %v", waiting, errs)
	}
}

func TestAGonePlaceIsANote(t *testing.T) {
	dir := marked(t)
	// A build may delete the artifact a report names, and a place may be
	// written in another case: neither blocks status or mark.
	write(t, dir, ".markitect/reports/ghost.md", lendingReport("job build reviewer, 2026-11-18", "src/ghost.go", "none", "", ""))
	write(t, dir, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/Lending/return.md", "complete", "", ""))
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Error(code, out)
	}
	code, out, _ := run("status", "-C", dir)
	if code != ExitOK || out != "note: report ghost: place gone: src/ghost.go\nnote: report late-fee: place gone: docs/spec/Lending/return.md\n0 changed, 0 suspect, 0 unexplained\n" {
		t.Error(code, out)
	}
	verdict(t, dir, "roles/member.md", "ok", "--file", "src/member.go", "-e", "src/member.go")
	// A closed report notes nothing.
	write(t, dir, ".markitect/reports/ghost.md", lendingReport("job build reviewer, 2026-11-18", "src/ghost.go", "none", "", "2026-11-20: done\nRemoved."))
	if _, out, _ := run("status", "-C", dir); strings.Contains(out, "ghost") {
		t.Error(out)
	}
}

func TestFactoryErrorsBlockStatusAndMark(t *testing.T) {
	dir := marked(t)
	write(t, dir, ".markitect/reports/slow.md", lendingReport("job build reviewer, 2026-11-18", "src/loans.go", "slow", "", ""))
	code, out, _ := run("check", "-C", dir)
	if code != ExitFound || out != ".markitect/reports/slow.md:16: report.form: violated property: write one of necessary, complete, consistent, unambiguous and verifiable, structured, correct, none\n1 error in 4 documents, 1 report and 0 samples\n" {
		t.Error(code, out)
	}
	_, errs, code := waitingOf(t, dir)
	if code != ExitFound || len(errs) != 1 || errs[0].Code != state.CodeReportForm {
		t.Error(code, errs)
	}
	at := "roles/member.md@" + fingerprint(t, dir, "roles/member.md")
	if code, _, errOut := run("mark", "-C", dir, at, "ok", "-e", "src/member.go"); code != ExitFound || !strings.Contains(errOut, "report.form") {
		t.Error(code, errOut)
	}
	// move is not stopped by errors of the reports.
	if code, _, errOut := run("move", "-C", dir, "roles/member.md", "roles/card-holder.md"); code != ExitOK {
		t.Error(code, errOut)
	}
}

func TestMarkTakesCounterChecksOnlyFromTheFolder(t *testing.T) {
	dir := project(t, lending)
	write(t, dir, "tests/loans_test.go", "package lending\n")
	at := "lending/borrow.md@" + fingerprint(t, dir, "lending/borrow.md")
	code, _, errOut := run("mark", "-C", dir, at, "ok", "--file", "src/loans.go", "--check", "tests/loans_test.go", "-e", "tests/loans_test.go")
	if code != ExitFound || !strings.Contains(errOut, "tests/loans_test.go lies outside tests/counter-checks, the counter-check folder the settings name") {
		t.Error(code, errOut)
	}
	// Without the folder in the settings, any file outside the spec does.
	write(t, dir, ".markitect/settings.md", "# Settings\n\n## Models\n- builder: sonnet\n")
	verdict(t, dir, "lending/borrow.md", "ok", "--file", "src/loans.go", "--check", "tests/loans_test.go", "-e", "tests/loans_test.go")
	// A folder that does not exist yet is no error.
	write(t, dir, ".markitect/settings.md", "# Settings\n\n## Counter-checks\nchecks/\n")
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Error(code, out)
	}
}

func TestSpotChecksFromTheHistory(t *testing.T) {
	clock(t)
	dir := marked(t)
	events := history(t, dir)
	if len(events) != 4 || events[0].Inputs == "" {
		t.Fatalf("%+v", events)
	}
	// A fresh reviewer judges member again at the same inputs, and differs;
	// new evidence does not change the inputs.
	verdict(t, dir, "roles/member.md", "fail", "--file", "src/member.go", "-m", "a member has no card")
	verdict(t, dir, "roles/member.md", "fail", "--file", "src/member.go", "-m", "still no card")
	code, out, _ := run("figures", "-C", dir, "--json")
	var doc struct {
		Reverdicts     int                 `json:"reverdicts"`
		Contradictions []contradictionJSON `json:"contradictions"`
		Reference      map[string]string   `json:"reference"`
		StoppedJobs    *int                `json:"stopped_jobs"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitOK {
		t.Fatal(err, out)
	}
	if doc.Reverdicts != 2 || len(doc.Contradictions) != 1 || doc.Contradictions[0].Doc != "roles/member.md" || doc.Contradictions[0].Was != "ok" ||
		doc.Contradictions[0].Now != "fail" || doc.StoppedJobs != nil || doc.Reference["stopped-jobs"] != state.StoppedJobs {
		t.Errorf("%+v", doc)
	}
}

func TestFiguresOfReportsAndSamples(t *testing.T) {
	dir := marked(t)
	write(t, dir, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete", "", ""))
	write(t, dir, ".markitect/reports/slow-loans.md", lendingReport("intake, 2026-11-19", "src/loans.go", "none", "branch fix-loans", "2026-11-20: done\nLanded."))
	write(t, dir, ".markitect/samples/first.md", "# First sample\n\n## Source\njob control-sample sampler, 2026-11-19\n\n## Copy\nbranch sample-first\n\n"+
		"## Planted\n- lending/borrow.md: lends a sixth book\n- lending/return.md: four weeks\n\n## Verdicts\n- lending/borrow.md: fail\n- lending/return.md: ok\n")
	if code, out, _ := run("check", "-C", dir); code != ExitOK || out != "0 errors in 4 documents, 2 reports and 1 sample\n" {
		t.Error(code, out)
	}
	code, out, _ := run("figures", "-C", dir)
	for _, part := range []string{
		"\n# Figures of the factory\n\n### hit-rate (1)\n\nreference: 1 of 2 planted documents (50%) over 1 sample\nfirst (2026-11-19): 1 of 2 found; missed lending/return.md\n",
		"\n### open-reports (1)\n\nreference: 2 reports, 1 open; open at the end of a day with reports raised or closed: median 1, range 1–2\n2026-11-18: 1 raised, 0 closed, 1 open\n2026-11-19: 1 raised, 0 closed, 2 open\n2026-11-20: 0 raised, 1 closed, 1 open\n",
		"\n### findings-per-area (2)\n\nreference: 2 areas; reports per area: median 1\nartifacts: 1 report, 0 open\nlending: 1 report, 1 open\n",
		"\n### recurring-reports (0)\n",
		"\n### stopped-jobs\n\n" + state.StoppedJobs + "\n",
	} {
		if code != ExitOK || !strings.Contains(out, part) {
			t.Errorf("missing %q in\n%s", part, out)
		}
	}
}

func TestCompareReports(t *testing.T) {
	older := marked(t)
	write(t, older, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete", "", ""))
	write(t, older, ".markitect/reports/slow-loans.md", lendingReport("intake, 2026-11-19", "src/loans.go", "none", "", ""))
	write(t, older, ".markitect/reports/old-idea.md", lendingReport("owner, 2026-11-19", "docs/spec/roles/member.md", "necessary", "", ""))
	newer := copyTree(t, older, false)
	write(t, newer, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete", "", "2026-11-20: done\nThe fee is stated."))
	write(t, newer, ".markitect/reports/slow-loans.md", lendingReport("intake, 2026-11-19", "src/loans.go", "none", "branch fix-loans", ""))
	write(t, newer, ".markitect/reports/new-wish.md", lendingReport("intake, 2026-11-20", "src/member.go", "none", "", ""))
	if err := os.Remove(filepath.Join(newer, ".markitect", "reports", "old-idea.md")); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run("compare", older, newer)
	if code != ExitFound || !strings.Contains(out, "## Reports (4)\n\nclosed: late-fee: 2026-11-20: done\n--- .markitect/reports/late-fee.md\n+++ .markitect/reports/late-fee.md\n") ||
		!strings.Contains(out, "\nnew: new-wish\ngone: old-idea\nchanged: slow-loans\n--- .markitect/reports/slow-loans.md\n") || !strings.Contains(out, "+branch fix-loans\n") {
		t.Error(code, out)
	}
	// The same reports in both differ in nothing.
	if code, out, _ := run("compare", older, copyTree(t, older, true)); code != ExitOK {
		t.Error(code, out)
	}
}

func TestSettingsCommand(t *testing.T) {
	dir := initialized(t)
	write(t, dir, ".markitect/settings.md", state.DefaultSettings+"\n## Tests\ngo test ./...\n\n## Write table\nbuilder: src/\n")
	code, out, _ := run("settings", "-C", dir)
	want := "Models:\n  architect: opus\n  builder: sonnet\n  counter-checker: sonnet\n  intake: sonnet\n  reader: sonnet\n  reviewer: sonnet\n  sampler: sonnet\n  triage: sonnet\n" +
		"Counter-checks: tests/counter-checks\nTests: go test ./...\nCadence: (none)\nWrite table:\n  builder: src/\nCI: .github/workflows/\n"
	if code != ExitOK || out != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
	code, out, _ = run("settings", "-C", dir, "--json")
	var doc settingsDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitOK {
		t.Fatalf("%d %v\n%s", code, err, out)
	}
	if doc.Kind != "settings" || !doc.Present || doc.Tests != "go test ./..." || doc.CounterChecks != "tests/counter-checks" ||
		doc.Models["reviewer"] != "sonnet" || len(doc.WriteTable) != 1 || doc.WriteTable[0].Role != "builder" || len(doc.Errors) != 0 {
		t.Errorf("%+v", doc)
	}
	// Errors in the settings are printed, and the command exits 1.
	write(t, dir, ".markitect/settings.md", "# Settings\n\n## Tests\ngo test\nnpm test\n")
	code, out, _ = run("settings", "-C", dir)
	if code != ExitFound || !strings.HasPrefix(out, ".markitect/settings.md:5: settings.form: tests: one command, on one line\n") || !strings.Contains(out, "Tests: go test\n") {
		t.Errorf("got %d\n%s", code, out)
	}
	// Without the file there are no settings, and the CI definition is the default.
	if err := os.Remove(filepath.Join(dir, ".markitect", "settings.md")); err != nil {
		t.Fatal(err)
	}
	code, out, _ = run("settings", "-C", dir, "--json")
	doc = settingsDocument{}
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitOK || doc.Present || doc.Tests != "" || len(doc.Models) != 0 || strings.Join(doc.CI, " ") != ".github/workflows/" {
		t.Errorf("%d %v %+v", code, err, doc)
	}
}
