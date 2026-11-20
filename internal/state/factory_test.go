package state

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// lendingFiles is a small lending system: a spec of operations, the code that
// realizes them, and the files of the factory a test adds.
func lendingFiles(extra map[string]string) fstest.MapFS {
	files := map[string]string{
		".markitect/version":           "0.35.0\n",
		".markitect/spec":              "docs/spec\n",
		"docs/spec/types/operation.md": "# Operation\n\nSomething a member does.\n\n## Rule\nWhat holds.\n",
		"docs/spec/lending/borrow.md":  "# Borrow\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA member borrows at most five books.\n",
		"docs/spec/lending/return.md":  "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within three weeks.\n",
		"docs/spec/README.md":          "# Library\n",
		"src/loans.go":                 "package lending\n",
		"src/__init__.py":              "\n",
	}
	for p, text := range extra {
		files[p] = text
	}
	fsys := fstest.MapFS{}
	for p, text := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(text)}
	}
	return fsys
}

func loadFactory(t *testing.T, extra map[string]string) (*Project, *Factory) {
	t.Helper()
	p, err := Load(lendingFiles(extra))
	if err != nil {
		t.Fatal(err)
	}
	f, err := ReadFactory(p)
	if err != nil {
		t.Fatal(err)
	}
	return p, f
}

// report writes a report from its sections, in order; an empty value leaves
// the section out.
func report(sections ...string) string {
	heads := []string{"Source", "Place", "Finding", "Occurred or risk", "Violated property", "Rating", "Actions", "Prevention", "Closing"}
	b := "# A report\n"
	for i, v := range sections {
		if v != "" {
			b += "\n## " + heads[i] + "\n" + v + "\n"
		}
	}
	return b
}

// diagnostics are the factory's diagnostics as "path:line: code".
func diagnostics(f *Factory) []string {
	var out []string
	for _, d := range f.Diagnostics {
		out = append(out, d.Path+":"+two(d.Line)+": "+d.Code)
	}
	return out
}

// two is a number of two digits, so that lines sort as numbers.
func two(n int) string { return fmt.Sprintf("%02d", n) }

func TestReportForms(t *testing.T) {
	_, f := loadFactory(t, map[string]string{
		// Every value of its form: a link for the place, a list for the
		// actions, the value in code.
		".markitect/reports/late-fee.md": report("job build reviewer, 2026-11-18", "[Return](../../docs/spec/lending/return.md)",
			"The fee for a late book is not stated.", "`risk`", "complete", "impact high, likelihood medium",
			"- escalated\n- Charge per day.\n- Charge once.", "A question on fees.", ""),
		// A place may be an artifact, with underscores kept, or a folder.
		".markitect/reports/init-file.md": report("intake, 2026-11-19", "src/__init__.py", "It is empty.", "occurred", "none",
			"", "branch fix-init", "", "2026-11-20: done\nThe file is gone."),
		".markitect/reports/lending.md": report("owner, 2026-11-19", "docs/spec/lending/", "The area is thin.", "risk", "necessary",
			"", "", "", "2026-11-20: duplicate of late-fee\nSame gap."),
	})
	if got := diagnostics(f); len(got) != 0 {
		t.Fatal(got)
	}
	r := f.Reports[1]
	if r.Name != "late-fee" || r.By != "job" || r.Job != "build" || r.Role != "reviewer" || r.Raised != "2026-11-18" ||
		r.Place != "docs/spec/lending/return.md" || r.Occurred != "risk" || r.Property != "complete" || r.Impact != "high" ||
		r.Likelihood != "medium" || !r.Escalated || !r.Open() {
		t.Errorf("%+v", r)
	}
	if r := f.Reports[0]; r.Place != "src/__init__.py" || r.Outcome != ClosedDone || r.ClosedOn != "2026-11-20" || strings.Join(r.Branches, " ") != "fix-init" {
		t.Errorf("%+v", r)
	}
	if r := f.Reports[2]; r.Place != "docs/spec/lending" || r.Outcome != ClosedDuplicate || r.DuplicateOf != "late-fee" {
		t.Errorf("%+v", r)
	}
}

func TestReportFormErrors(t *testing.T) {
	_, f := loadFactory(t, map[string]string{
		".markitect/reports/bad.md": report("job build boss, 2026-11-18", "docs/spec/lending/ghost.md", "x", "maybe", "wrong",
			"impact huge", "merge it", "", ""),
		".markitect/reports/worse.md": report("someone, 18 November", "[a](../../docs/spec/a.md) and [b](../../docs/spec/b.md)", "x", "risk", "correct",
			"", "escalated", "", "2026-11-17: duplicate of nothing-here\nNo reason."),
		".markitect/reports/closed.md": report("owner, 2026-11-18", "docs/spec/lending/ghost.md", "x", "risk", "correct",
			"", "", "", "2026-11-19: done"),
		".markitect/reports/early.md": report("owner, 2026-11-18", "src/loans.go", "x", "risk", "none",
			"", "", "", "2026-11-17: nothing to do\nNo reason."),
		".markitect/reports/short.md":    "# Short\n\n## Source\nowner, 2026-11-18\n\n## Place\n.markitect/state.jsonl\n",
		".markitect/reports/Late_Fee.md": report("owner, 2026-11-18"),
		".markitect/reports/notes.txt":   "",
		".markitect/reports/.gitkeep":    "",
	})
	want := []string{
		".markitect/reports/Late_Fee.md:01: report.form",
		// The role, the place that is gone while the report is open, and
		// every value not of its form.
		".markitect/reports/bad.md:04: report.form",
		".markitect/reports/bad.md:07: report.place",
		".markitect/reports/bad.md:13: report.form",
		".markitect/reports/bad.md:16: report.form",
		".markitect/reports/bad.md:19: report.form",
		".markitect/reports/bad.md:22: report.form",
		// A closed report may name a place that is gone; its closing needs a
		// reason.
		".markitect/reports/closed.md:19: report.form",
		// Closed before it was raised.
		".markitect/reports/early.md:19: report.form",
		".markitect/reports/notes.txt:01: report.form",
		".markitect/reports/short.md:01: section.missing",
		".markitect/reports/short.md:01: section.missing",
		".markitect/reports/short.md:01: section.missing",
		".markitect/reports/short.md:07: report.form",
		".markitect/reports/worse.md:04: report.form",
		".markitect/reports/worse.md:07: report.form",
		".markitect/reports/worse.md:19: report.form",
		".markitect/reports/worse.md:22: report.duplicate",
	}
	if got := diagnostics(f); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, d := range f.Diagnostics {
		if d.Path == ".markitect/reports/early.md" && !strings.Contains(d.Message, "before it was raised") {
			t.Error(d)
		}
	}
}

func TestSettings(t *testing.T) {
	_, f := loadFactory(t, map[string]string{SettingsFile: DefaultSettings})
	if got := diagnostics(f); len(got) != 0 || f.Settings == nil {
		t.Fatal(got)
	}
	s := f.Settings
	if s.CounterChecks != "tests/counter-checks" || len(s.Models) != 8 || s.Models["architect"] != "opus" || s.Models["builder"] != "sonnet" {
		t.Errorf("%+v", s)
	}
	_, f = loadFactory(t, map[string]string{SettingsFile: "# Settings\n\n## Models\nbuilder: sonnet\nbuilder: opus\npainter: opus\nreviewer: a model\n\n" +
		"## Counter-checks\ndocs/spec/checks\n\n## Cadence\nreview: daily\nreview: weekly\nBug Hunt: weekly\n\n" +
		"## Write table\nbuilder: `src/**/*.go`\nbuilder: !src/vendor/\nreviewer: [a-\nreader: [[:word:]]\narchitect: \\\nbuilder\n\n## Notes\nOur own section.\n"})
	want := []string{
		".markitect/settings.md:05: settings.form", // builder twice
		".markitect/settings.md:06: settings.form", // no such role
		".markitect/settings.md:07: settings.form", // a model with a space
		".markitect/settings.md:10: settings.form", // in the spec folder
		".markitect/settings.md:14: settings.form", // review twice
		".markitect/settings.md:15: settings.form", // not a job's name
		".markitect/settings.md:20: settings.form", // an open bracket
		".markitect/settings.md:21: settings.form", // an unknown class
		".markitect/settings.md:22: settings.form", // a backslash at the end
		".markitect/settings.md:23: settings.form", // no pattern
	}
	if got := diagnostics(f); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if s := f.Settings; s.CounterChecks != "" || len(s.WriteTable) != 2 || s.WriteTable[0].Pattern != "src/**/*.go" || s.WriteTable[1].Pattern != "!src/vendor/" {
		t.Errorf("%+v", s)
	}
	for _, folder := range []string{".markitect/checks", "a/.git/checks", "../checks", "/checks", "two folders"} {
		_, f = loadFactory(t, map[string]string{SettingsFile: "# Settings\n\n## Counter-checks\n" + folder + "\n"})
		if len(f.Diagnostics) != 1 || f.Diagnostics[0].Code != CodeSettingsForm {
			t.Errorf("%s: %v", folder, f.Diagnostics)
		}
	}
	// A project without settings has none, and no error.
	if _, f = loadFactory(t, nil); f.Settings != nil || len(f.Diagnostics) != 0 {
		t.Errorf("%+v", f)
	}
}

func sample(source, planted, verdicts string) string {
	return "# A sample\n\n## Source\n" + source + "\n\n## Copy\nbranch sample-1\n\n## Planted\n" + planted + "\n\n## Verdicts\n" + verdicts + "\n"
}

func TestSamplesAndTheHitRate(t *testing.T) {
	p, f := loadFactory(t, map[string]string{
		".markitect/samples/first.md": sample("job control-sample sampler, 2026-11-18",
			"- lending/borrow.md: six books allowed\n- lending/borrow.md: no member check\n- lending/return.md: four weeks",
			"- lending/borrow.md: fail\n- lending/return.md: ok\n- types/operation.md: ok"),
		".markitect/samples/second.md": sample("job control-sample sampler, 2026-11-19", "lending/return.md: no fee", "lending/return.md: fail"),
	})
	if got := diagnostics(f); len(got) != 0 {
		t.Fatal(got)
	}
	ff := ComputeFactoryFigures(p, f, nil)
	if ff.Planted != 3 || ff.Hit != 2 || len(ff.Samples) != 2 || ff.Samples[0].Name != "first" || ff.Samples[0].Planted != 2 || ff.Samples[0].Hit != 1 ||
		strings.Join(ff.Samples[0].Missed, " ") != "lending/return.md" || ff.Samples[1].Hit != 1 {
		t.Errorf("%+v", ff)
	}
	if ref := ff.Reference["hit-rate"]; ref != "2 of 3 planted documents (66%) over 2 samples" {
		t.Error(ref)
	}
	_, f = loadFactory(t, map[string]string{
		".markitect/samples/bad.md": sample("job build reviewer, 2026-11-18", "lending/borrow.md: six books\nborrow: x\nlending/return.md: four weeks",
			"lending/borrow.md: maybe\nlending/borrow.md: fail\nlending/borrow.md: ok"),
	})
	want := []string{
		".markitect/samples/bad.md:04: sample.form",    // not the sampler's source
		".markitect/samples/bad.md:11: sample.form",    // no document
		".markitect/samples/bad.md:12: sample.verdict", // planted without a verdict
		".markitect/samples/bad.md:15: sample.form",    // no verdict
		".markitect/samples/bad.md:17: sample.form",    // a second verdict
	}
	if got := diagnostics(f); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSpotCheckContradictions(t *testing.T) {
	t0 := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	in1, in2 := Hash([]byte("inputs 1")), Hash([]byte("inputs 2"))
	v := func(min int, doc, verdict, inputs string) Event {
		return Event{Time: t0.Add(time.Duration(min) * time.Minute), Type: EventVerdict, Doc: doc, Verdict: verdict, Inputs: inputs}
	}
	events := []Event{
		v(0, "lending/borrow.md", "ok", in1),
		v(1, "lending/borrow.md", "ok", in1),   // a re-verdict that agrees
		v(2, "lending/borrow.md", "fail", in1), // contradicts the ok before
		v(3, "lending/borrow.md", "fail", in2), // other inputs: no re-verdict
		v(4, "lending/return.md", "ok", ""),    // written before inputs: left out
		v(5, "lending/return.md", "fail", ""),
		v(6, "lending/return.md", "ok", in1), // another document
	}
	ff := ComputeFactoryFigures(&Project{}, &Factory{}, events)
	if ff.Reverdicts != 2 || len(ff.Contradictions) != 1 || ff.Contradictions[0].Was != "ok" || ff.Contradictions[0].Now != "fail" ||
		ff.Reference["contradictions"] != "2 re-verdicts at unchanged inputs, of 5 verdicts that name their inputs" {
		t.Errorf("%+v", ff)
	}
}

func TestInputsDigestLeavesOutTheEvidence(t *testing.T) {
	a := Inputs{"self": hashA, "file:src/loans.go": hashA, "evidence:tests/loans_test.go": hashA}
	b := Inputs{"self": hashA, "file:src/loans.go": hashA}
	if InputsDigest(a) != InputsDigest(b) || !validHash(InputsDigest(a)) {
		t.Error("the evidence counts")
	}
	b["check:tests/counter-checks/loans_test.go"] = hashA
	if InputsDigest(a) == InputsDigest(b) {
		t.Error("a counter-check does not count")
	}
	e := Event{Time: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC), Type: EventVerdict, Doc: "a.md", Verdict: "ok", Self: hashA, Inputs: InputsDigest(a)}
	if got := string(e.Line()); got != `{"time":"2026-11-20T09:00:00Z","type":"verdict","doc":"a.md","verdict":"ok","self":"`+hashA+`","inputs":"`+InputsDigest(a)+`","changed":[]}`+"\n" {
		t.Error(got)
	}
	events, problems := ParseHistory(e.Line())
	if len(events) != 1 || events[0].Inputs != InputsDigest(a) || len(problems) != 0 {
		t.Error(events, problems)
	}
	if _, problems := ParseHistory([]byte(`{"time":"2026-11-20T09:00:00Z","type":"verdict","doc":"a.md","verdict":"ok","inputs":"x"}`)); len(problems) != 1 {
		t.Error("a bad inputs hash reads")
	}
}

func TestReportFigures(t *testing.T) {
	p, f := loadFactory(t, map[string]string{
		".markitect/reports/late-fee.md": report("job read reader, 2026-11-18", "docs/spec/lending/return.md", "x", "risk", "complete"),
		".markitect/reports/fee-again.md": report("intake, 2026-11-19", "docs/spec/lending/return.md", "x", "occurred", "complete",
			"", "", "", "2026-11-19: duplicate of late-fee\nThe same gap."),
		".markitect/reports/slow-loans.md": report("intake, 2026-11-19", "src/loans.go", "x", "occurred", "none",
			"", "", "", "2026-11-20: done\nFixed."),
		".markitect/reports/library.md": report("owner, 2026-11-20", "docs/spec/README.md", "x", "risk", "structured"),
	})
	if got := diagnostics(f); len(got) != 0 {
		t.Fatal(got)
	}
	ff := ComputeFactoryFigures(p, f, nil)
	var days []string
	for _, d := range ff.Days {
		days = append(days, d.Day+" +"+two(d.Raised)+" -"+two(d.Closed)+" ="+two(d.Open))
	}
	if ff.Reports != 4 || ff.Open != 2 || strings.Join(days, ", ") != "2026-11-18 +01 -00 =01, 2026-11-19 +02 -01 =02, 2026-11-20 +01 -01 =02" {
		t.Errorf("%d %d %v", ff.Reports, ff.Open, days)
	}
	var areas []string
	for _, a := range ff.Areas {
		areas = append(areas, a.Name+" "+two(a.Reports)+" "+two(a.Open))
	}
	if strings.Join(areas, ", ") != "lending 02 01, (spec) 01 01, artifacts 01 00" {
		t.Error(areas)
	}
	var recurring []string
	for _, r := range ff.Recurring {
		recurring = append(recurring, r.Kind+": "+r.Detail)
	}
	if strings.Join(recurring, "\n") != "duplicates: late-fee recurred 1 time: fee-again\nplace: docs/spec/lending/return.md: 2 reports: fee-again, late-fee" {
		t.Error(recurring)
	}
	if ff.Reference["stopped-jobs"] != StoppedJobs {
		t.Error(ff.Reference)
	}
}
