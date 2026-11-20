package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReportsList(t *testing.T) {
	dir := marked(t)
	write(t, dir, ".markitect/reports/late-fee.md", lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete",
		"escalated\nCharge per day.\nCharge once.", "")+"\n## Rating\nimpact high, likelihood medium\n")
	write(t, dir, ".markitect/reports/slow-loans.md", lendingReport("intake, 2026-11-17", "src/loans.go", "none", "branch markitect/slow-loans",
		"2026-11-19: done\nLoans are fast again."))
	code, out, _ := run("reports", "-C", dir)
	want := "late-fee: open, escalated\n  source: job read reader, 2026-11-18\n  place: docs/spec/lending/return.md\n  occurred or risk: risk\n" +
		"  property: complete\n  rating: impact high, likelihood medium\n  branches: (none)\n  proposal: Charge per day.\n  proposal: Charge once.\n  closing: (none)\n" +
		"slow-loans: closed\n  source: intake, 2026-11-17\n  place: src/loans.go\n  occurred or risk: risk\n  property: none\n  rating: (none)\n" +
		"  branches: markitect/slow-loans\n  closing: 2026-11-19: done\n2 reports, 1 open, 1 of them escalated\n"
	if code != ExitOK || out != want {
		t.Errorf("%d\n%s", code, out)
	}
	code, out, _ = run("reports", "-C", dir, "--open", "--json")
	var doc reportsDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitOK || doc.Kind != "reports" || len(doc.Reports) != 1 {
		t.Fatalf("%v %d %s", err, code, out)
	}
	if r := doc.Reports[0]; r.Name != "late-fee" || r.Source != "job" || r.Job != "read" || r.Role != "reader" || r.Day != "2026-11-18" ||
		!r.Escalated || len(r.Proposals) != 2 || r.Impact != "high" || !r.Open {
		t.Errorf("%+v", r)
	}
	// A report not of its form is printed first, and the command exits 1.
	write(t, dir, ".markitect/reports/bad.md", lendingReport("someone, 2026-11-18", "src/loans.go", "none", "", ""))
	if code, out, _ := run("reports", "-C", dir, "--open"); code != ExitFound || !strings.HasPrefix(out, ".markitect/reports/bad.md:4: report.form: source:") {
		t.Errorf("%d %s", code, out)
	}
	if code, _, errOut := run("status", "-C", dir, "--open"); code != ExitFailed || !strings.Contains(errOut, "--open is a flag of reports") {
		t.Error(code, errOut)
	}
}

// A report on the spec is answered on a branch of its own and closed there,
// so that the closing lands with its answer: from a job, switched to
// architect, and from the owner's choice between proposals.
func TestAReportAnsweredOnItsBranchLands(t *testing.T) {
	land := func(branch, target string) (int, string) {
		t.Helper()
		code, out, _ := run("status", "-C", branch, "--target", target)
		return code, out
	}
	answer := func(branch string) {
		t.Helper()
		write(t, branch, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within three weeks; a late one costs a fee per day.\n")
		write(t, branch, "src/returns.go", "package lending\n\nfunc Return() { fee() }\n")
		verdict(t, branch, "lending/return.md", "ok", "--file", "src/returns.go", "-e", "src/returns.go:3")
	}
	report := func(actions, closing string) string {
		return lendingReport("job read reader, 2026-11-18", "docs/spec/lending/return.md", "complete", actions, closing)
	}
	closing := "2026-11-20: done\nThe spec states the fee, and it is built."

	// From a job: switched to architect and answered on the job's branch.
	target := marked(t)
	branch := copyTree(t, target, false)
	write(t, branch, ".markitect/reports/late-fee.md", report("branch markitect/read-2026-11-20", ""))
	answer(branch)
	if code, out := land(branch, target); code != ExitFound || !strings.Contains(out, "adds waiting: lending/return.md") {
		t.Errorf("open and not escalated: %d %s", code, out)
	}
	write(t, branch, ".markitect/reports/late-fee.md", report("branch markitect/read-2026-11-20", closing))
	if code, out := land(branch, target); code != ExitOK {
		t.Errorf("closed with its answer: %d %s", code, out)
	}

	// From the owner's choice: escalated on the target, the chosen branch
	// in place of escalated, built and closed there.
	target = marked(t)
	write(t, target, ".markitect/reports/late-fee.md", report("escalated\nCharge per day.\nCharge once.", ""))
	branch = copyTree(t, target, false)
	write(t, branch, ".markitect/reports/late-fee.md", report("branch markitect/late-fee", ""))
	answer(branch)
	if code, out := land(branch, target); code != ExitFound {
		t.Errorf("chosen and not closed: %d %s", code, out)
	}
	write(t, branch, ".markitect/reports/late-fee.md", report("branch markitect/late-fee", closing))
	if code, out := land(branch, target); code != ExitOK {
		t.Errorf("chosen, built and closed: %d %s", code, out)
	}
}
