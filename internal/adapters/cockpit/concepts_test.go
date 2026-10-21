package cockpit

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// The tests below follow the current Apply concept: the bound strategy and
// each Manager's model context, the Model assessment as a verdict, the kinds
// of observation, probes with receipts and dispositions, lookups, and the
// closed Run ending. The records are synthetic and shaped like the run
// engine's.

func startWithBinding(s *scenario) {
	s.add("run.start", "", "", "", map[string]any{
		"run": "synthetic", "managers": []string{".", "stock"}, "start": "tree-start", "agentTurns": 60, "passes": 12, "deadline": t0.Add(time.Hour),
		"binding": map[string]any{
			"entry": "apply", "head": "abc123", "strategy": "staged", "build": "0.14.1+b9cdd6d8",
			"work": []string{"assess", "plan", "prepare", "implement"}, "corrections": []string{"plan", "prepare", "implement"},
			"budget": map[string]any{"executionTime": "40m0s", "turnTimeout": "8m0s", "agentTurns": 60, "passes": 12},
			"managers": []map[string]any{
				{"manager": ".", "context": map[string]any{"brief": "sha256:aa", "full": []string{"stock/interface"}, "reference": []string{"stock/add", "stock/list"}}},
				{"manager": "stock", "context": map[string]any{"brief": "sha256:bb", "full": []string{"stock/add", "stock/list", "stock/interface"}, "reference": []string{}}},
			},
		},
	})
}

func TestProjectDecodesTheBindingAndEachManagersContext(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	s.add("pass.start", "p1", "", "", map[string]any{"purpose": "assess", "review": true, "writes": false, "tree": "tree-start"})
	s.add("turn.start", "p1", ".", "r-p1-root-r1", map[string]any{"role": "reviewer", "kind": "verify", "mode": "review", "context": "brief", "readOnly": true})
	s.add("lookup.definition", "p1", ".", "r-p1-root-r1", map[string]any{"id": "stock/add", "owner": "stock", "bytes": 585, "answer": "sha256:cc"})
	s.add("lookup.brief", "p1", ".", "r-p1-root-r1", map[string]any{"bytes": 11000, "answer": "sha256:dd"})
	s.add("turn.end", "p1", ".", "r-p1-root-r1", map[string]any{"role": "reviewer", "kind": "verify", "mode": "review", "stopped": true, "reports": []string{"basis"}, "reason": "provider result", "probes": 0, "compacted": false,
		"usage": run.Usage{Model: "m", OutputTokens: 100, CostUSD: 0.5, ProviderTurns: 3}})
	s.add("turn.start", "p1", ".", "r-p1-root-r2", map[string]any{"role": "reviewer", "kind": "verify", "mode": "review", "context": "history", "readOnly": true})
	s.add("turn.end", "p1", ".", "r-p1-root-r2", map[string]any{"role": "reviewer", "stopped": true, "reports": []string{}, "usage": run.Usage{OutputTokens: 20, CostUSD: 0.25}})
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	h := v.Run
	if h.Strategy != "staged" || h.Build != "0.14.1+b9cdd6d8" || h.Source != "abc123" || h.ExecutionTime != "40m0s" || !slices.Equal(h.Work, []string{"assess", "plan", "prepare", "implement"}) {
		t.Fatalf("binding = %+v", h)
	}
	root := managerOf(t, v, ".")
	if root.Context == nil || root.Context.Brief != "sha256:aa" || !slices.Equal(root.Context.Full, []string{"stock/interface"}) || len(root.Context.Reference) != 2 {
		t.Fatalf("Root's context = %+v", root.Context)
	}
	if stock := managerOf(t, v, "stock"); stock.Context == nil || stock.Context.Reference == nil || len(stock.Context.Reference) != 0 {
		t.Fatalf("a Manager without reference-only Definitions keeps an empty list: %+v", stock.Context)
	}
	if len(v.Lookups) != 2 || v.Lookups[0].Kind != "definition" || v.Lookups[0].ID != "stock/add" || v.Lookups[0].Owner != "stock" || v.Lookups[1].Kind != "brief" {
		t.Fatalf("lookups = %+v", v.Lookups)
	}
	if got := v.Turns[0]; got.Mode != "review" || got.Context != "brief" || v.Turns[1].Context != "history" {
		t.Fatalf("turns = %+v", v.Turns)
	}
	pass := v.Passes[0]
	if pass.Turns != 2 || pass.CostUSD != 0.75 || pass.OutputTokens != 120 {
		t.Fatalf("the Pass sums its turns' usage: %+v", pass)
	}
	texts := entryTexts(records)
	for _, want := range []string{". looked up", "Root looked up stock/add", "Root looked up its Brief again", "context brief"} {
		if !anyContains(texts, want) && want != ". looked up" {
			t.Errorf("no entry text contains %q: %v", want, texts)
		}
	}
	if len(v.Gaps) == 0 || slices.Contains(gapCodes(v), "record.kind") {
		t.Fatalf("lookups and the binding are known kinds: %v", gapCodes(v))
	}
}

func entryTexts(records []Record) []string {
	var out []string
	for _, e := range Entries(records) {
		out = append(out, e.Text)
	}
	return out
}

func anyContains(texts []string, want string) bool {
	return slices.ContainsFunc(texts, func(t string) bool { return strings.Contains(t, want) })
}

func TestProjectFoldsProbesWithReceiptsAndDispositions(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	s.add("candidate", "p5", "", "", map[string]any{"tree": "tree-a"})
	s.add("pass.start", "p5", "", "", map[string]any{"purpose": "verify", "review": true, "writes": false, "tree": "tree-a"})
	s.turn("p5", "stock", "r-p5-stock-r1", "reviewer", true)
	req := func(id string) {
		s.add("probe.requested", "p5", "stock", "r-p5-stock-r1", map[string]any{"id": id, "obligation": "stock/add", "claim": "add refuses a duplicate", "expected": "exit 2", "check": "tests", "standKind": "candidate", "stand": "tree-a", "path": "probe.py", "args": []string{"-B"}, "contentDigest": "d", "proposal": "", "rerun": ""})
	}
	req("R1")
	req("R2")
	req("R3")
	s.add("probe.receipt", "p5", "stock", "r-p5-stock-r1", run.ProbeReceipt{ID: "R1", Pass: "p5", Obligation: "stock/add", Claim: "add refuses a duplicate", StandKind: "candidate", Outcome: run.ProbePassed, Started: true, Exited: true, Content: "print(1)", Output: "ok"})
	s.add("probe.receipt", "p5", "stock", "r-p5-stock-r1", run.ProbeReceipt{ID: "R2", Pass: "p5", Obligation: "stock/add", StandKind: "candidate", Outcome: run.ProbeFailed, ExitCode: 1, Started: true, Exited: true})
	s.add("probe.receipt", "p5", "stock", "r-p5-stock-r1", run.ProbeReceipt{ID: "R3", Pass: "p5", Obligation: "stock/add", StandKind: "candidate", Outcome: run.ProbeError, Uncertain: true, Reason: "the probe did not start"})
	records := s.records()
	before := Project("r", records, len(records), Source{}, nil)
	if len(before.Probes) != 3 {
		t.Fatalf("probes = %+v", before.Probes)
	}
	if p := before.Probes[0]; p.Outcome != "passed" || p.Adverse || p.RequestSeq == 0 || p.ReceiptSeq <= p.RequestSeq || p.Claim == "" || p.Expected != "exit 2" || p.StandKind != "candidate" {
		t.Fatalf("R1 = %+v", p)
	}
	if p := before.Probes[1]; !p.Adverse || p.ExitCode != 1 || p.Disposition != nil {
		t.Fatalf("R2 before its disposition = %+v", p)
	}
	if p := before.Probes[2]; !p.Adverse || !p.Uncertain || p.Reason == "" {
		t.Fatalf("R3 = %+v", p)
	}
	s.add("probe.disposed", "p5", "stock", "", map[string]any{"receipt": "R2", "disposition": run.ProbeDisposition{As: run.DisposedViolation, By: "stock", Pass: "p5", Proposal: "P9", Finding: "F1"}})
	s.add("probe.disposed", "p5", ".", "", map[string]any{"receipt": "R3", "disposition": run.ProbeDisposition{As: run.DisposedExpectationError, By: ".", Pass: "p5", Reason: "wrong flag", Pending: true}})
	records = s.records()
	v := Project("r", records, len(records), Source{}, nil)
	if d := v.Probes[1].Disposition; d == nil || d.As != "violation" || d.Finding != "F1" || v.Probes[1].DisposedSeq == 0 {
		t.Fatalf("R2 disposition = %+v", v.Probes[1])
	}
	if d := v.Probes[2].Disposition; d == nil || !d.Pending {
		t.Fatalf("R3 disposition = %+v", v.Probes[2])
	}
	s.add("probe.disposed", "p5", ".", "", map[string]any{"receipt": "R3", "voided": run.ProbeDisposition{As: run.DisposedExpectationError, By: "."}, "why": "its submission was reopened"})
	records = s.records()
	v = Project("r", records, len(records), Source{}, nil)
	if p := v.Probes[2]; p.Disposition != nil || p.VoidedSeq == 0 || p.VoidedWhy == "" {
		t.Fatalf("a withdrawn disposition leaves the receipt undisposed: %+v", p)
	}
	texts := entryTexts(records)
	for _, want := range []string{"Probe R1 requested by stock on the candidate for stock/add", "Probe R1 on the candidate: passed", "Probe R2 on the candidate: failed", "Probe R3 on the candidate: error (uncertain)",
		"Probe R2 disposed as violation by stock", "Probe R3 disposed as expectation error by Root, pending", "The disposition of probe R3 was withdrawn"} {
		if !anyContains(texts, want) {
			t.Errorf("no entry text contains %q: %v", want, texts)
		}
	}
	for _, e := range Entries(records) {
		if strings.HasPrefix(e.Kind, "probe.") && (e.Class != "evidence" || !strings.HasPrefix(e.Ref, "probe:")) {
			t.Errorf("probe entry %+v", e)
		}
	}
	// Select returns the probe's request, receipt and dispositions with their data.
	d := Select(records, len(records), "probe:R3")
	var kinds []string
	for _, r := range d.Records {
		kinds = append(kinds, r.Kind)
	}
	if !slices.Equal(kinds, []string{"probe.requested", "probe.receipt", "probe.disposed", "probe.disposed"}) {
		t.Fatalf("probe:R3 selects %v", kinds)
	}
}

func finding(id, kind string, blocking bool, class run.Purpose) run.Finding {
	return run.Finding{ID: id, Owner: "stock", RaisedBy: "stock", Obligation: "stock/add", Observed: "observed", Expected: "expected", Blocking: blocking, Correction: class,
		Kind: run.Observation(kind), Pass: "p1", Review: true}
}

func TestFindingKindsDecideWhatBlocks(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	s.add("candidate", "p1", "", "", map[string]any{"tree": "tree-a"})
	s.add("pass.start", "p1", "", "", map[string]any{"purpose": "assess", "review": true, "writes": false, "tree": "tree-a"})
	s.add("finding.recorded", "p1", "stock", "", finding("F1", "model", true, "model_gap"))
	s.add("finding.recorded", "p1", "stock", "", finding("F2", "suspected", true, ""))
	s.add("finding.recorded", "p1", "stock", "", finding("F3", "suggestion", false, ""))
	s.add("finding.recorded", "p1", "stock", "", finding("F4", "", false, ""))
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	effect := map[string]string{}
	for _, f := range v.Findings {
		effect[f.ID] = f.Effect
	}
	if effect["F1"] != "blocks" || effect["F2"] != "investigate" || effect["F3"] != "advisory" || effect["F4"] != "advisory" {
		t.Fatalf("effects = %v", effect)
	}
	if v.Findings[0].Class != "model_gap" || v.Findings[1].Class != "" {
		t.Fatalf("classes = %q %q: the class of a Model assessment finding is its correction field", v.Findings[0].Class, v.Findings[1].Class)
	}
	if got := managerOf(t, v, "stock").Findings; !slices.Equal(got, []string{"F1"}) {
		t.Fatalf("only the confirmed finding is a blocking badge: %v", got)
	}
	var open []string
	for _, sentence := range v.Account {
		open = append(open, sentenceText(sentence))
	}
	text := strings.Join(open, "\n")
	if !strings.Contains(text, "Open findings that block: F1 ") || !strings.Contains(text, "Suspicions to investigate") || strings.Contains(text, "F3") {
		t.Fatalf("account = %q", text)
	}
	texts := entryTexts(records)
	for _, want := range []string{"Model problem F1 raised", ", class model_gap", "Suspicion F2 raised", "Suggestion F3 raised", "Note F4 raised"} {
		if !anyContains(texts, want) {
			t.Errorf("no entry text contains %q: %v", want, texts)
		}
	}
}

func TestSuspicionsNameNoCorrectionAndAnUnchangedCandidateIsAReVerify(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	s.add("candidate", "p3", "", "", map[string]any{"tree": "tree-a"})
	s.add("pass.start", "p3", "", "", map[string]any{"purpose": "verify", "review": true, "writes": false, "tree": "tree-a"})
	s.add("finding.recorded", "p3", "stock", "", finding("F1", "suspected", true, ""))
	s.add("pass.sealed", "p3", "", "", map[string]any{"purpose": "verify", "verdict": "needs_correction", "tree": "tree-a", "managers": []run.ManagerRecord{}})
	// Only a suspicion is open: a fresh review of the same candidate decides it.
	s.add("candidate", "p4", "", "", map[string]any{"tree": "tree-a"})
	s.add("pass.start", "p4", "", "", map[string]any{"purpose": "verify", "review": true, "writes": false, "tree": "tree-a"})
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	if v.Passes[1].Unchanged != "p3" || v.Passes[0].Unchanged != "" {
		t.Fatalf("passes = %+v", v.Passes)
	}
	if text := sentenceText(v.Account[0]); !strings.Contains(text, "on the unchanged candidate of Pass p3") {
		t.Fatalf("account = %q", text)
	}
	// A confirmed deviation sends a correction Pass; the later review judges a changed candidate.
	s.add("finding.recorded", "p4", "stock", "", finding("F2", "deviation", true, run.Implement))
	s.add("pass.sealed", "p4", "", "", map[string]any{"purpose": "verify", "verdict": "needs_correction", "tree": "tree-a", "managers": []run.ManagerRecord{}})
	s.add("pass.start", "p5", "", "", map[string]any{"purpose": "implement", "review": false, "writes": true, "tree": "tree-a"})
	s.add("candidate", "p6", "", "", map[string]any{"tree": "tree-b"})
	s.add("pass.start", "p6", "", "", map[string]any{"purpose": "verify", "review": true, "writes": false, "tree": "tree-b"})
	records = s.records()
	v = Project("r", records, len(records), Source{}, nil)
	byID := map[string]PassView{}
	for _, p := range v.Passes {
		byID[p.ID] = p
	}
	if p := byID["p5"]; !p.Correction || !slices.Equal(p.Addresses, []string{"F2"}) {
		t.Fatalf("the correction Pass addresses the deviation, not the suspicion: %+v", p)
	}
	if byID["p6"].Unchanged != "" {
		t.Fatalf("a review of a changed candidate is no re-Verify: %+v", byID["p6"])
	}
}

func TestModelAssessmentShowsItsCoverageAndAConfirmedProblemEndsTheRun(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	s.add("candidate", "p1", "", "", map[string]any{"tree": "tree-start"})
	s.add("pass.start", "p1", "", "", map[string]any{"purpose": "assess", "review": true, "writes": false, "tree": "tree-start"})
	long := strings.Repeat("a long reason ", 40)
	s.add("proposal.submitted", "p1", "stock", "r-p1-stock-r1", map[string]any{"proposal": "P2", "kind": "result", "purpose": "assess", "findings": []any{}, "closed": []string{}, "disputed": []string{},
		"envelope": map[string]any{"summary": "summary", "verdict": "needs_correction", "obligations": []map[string]any{
			{"definition": "stock/add", "status": "done", "note": "Assessed.", "evidence": []map[string]any{{"claim": "the refusal is decided", "source": ".markitect/model/stock/add.yaml", "location": "stock/add statement", "reasoning": long}}},
			{"definition": "stock/inputs", "status": "open", "note": "the maximum name length is never given"},
		}}})
	s.add("finding.recorded", "p1", "stock", "", finding("F1", "model", true, "model_gap"))
	s.add("pass.sealed", "p1", "", "", map[string]any{"purpose": "assess", "verdict": "needs_correction", "tree": "tree-start", "managers": []run.ManagerRecord{{Manager: ".", State: run.Done, Verdict: run.NeedsCorrection, Summary: "confirmed"}}})
	reason := run.ModelAssessmentReason + "stock/inputs refers to a name maximum it never gives. " + long
	s.add("outcome", "", "", "", map[string]any{"status": "failed", "reason": reason})
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	var prop ProposalView
	for _, p := range v.Proposals {
		if p.ID == "P2" {
			prop = p
		}
	}
	if len(prop.Coverage) != 2 || prop.Coverage[0].Evidence[0].Reasoning != long || prop.Coverage[0].Evidence[0].Location != "stock/add statement" || prop.Coverage[1].Status != "open" {
		t.Fatalf("coverage = %+v", prop.Coverage)
	}
	if v.Status.Ending != "model_problem" {
		t.Fatalf("status = %+v", v.Status)
	}
	if text := sentenceText(v.Account[0]); text != "Model problem confirmed — nothing published. Correct the model and start a new run." {
		t.Fatalf("account = %q", text)
	}
	if text := sentenceText(v.Account[1]); !strings.HasPrefix(text, "Reason: stock/inputs refers to a name maximum") || !strings.Contains(text, long[:len(long)-1]) {
		t.Fatalf("account = %q: the assessment reason is shown whole", text)
	}
	// The same ending is recognized when Root ended the Run itself.
	s.events = s.events[:len(s.events)-1]
	s.add("outcome", "", "", "", map[string]any{"status": "failed", "reason": run.ModelErrorReason + "contradiction"})
	rec := s.records()
	if got := Project("r", rec, len(rec), Source{}, nil).Status.Ending; got != "model_problem" {
		t.Fatalf("ending = %q", got)
	}
	// A technical failure is no model problem.
	s2 := newScenario(t)
	startWithBinding(s2)
	s2.add("outcome", "", "", "", map[string]any{"status": "failed", "reason": "the Model check failed"})
	rec = s2.records()
	if got := Project("r", rec, len(rec), Source{}, nil).Status.Ending; got != "" {
		t.Fatalf("ending = %q", got)
	}
}

func TestWaitsOnAnotherManagersAcceptedSubmissionRecognizeTheNamedForm(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	s.add("pass.start", "p2", "", "", map[string]any{"purpose": "implement", "review": false, "writes": true, "tree": "tree-start"})
	s.add("manager.release", "p2", ".", "", map[string]any{"why": "the Pass started"})
	s.add("manager.release", "p2", "stock", "", map[string]any{"why": "the basis of its parent was accepted", "parent": "."})
	s.add("report.wait", "p2", "stock", "r-p2-stock-m1", map[string]any{"for": ".", "stage": "basis", "reason": "I need Root's basis"})
	s.state("p2", "stock", "working", "waiting", "blocked", "its turn ended", "the accepted basis of .")
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	w := managerOf(t, v, "stock").Wait
	if w == nil || w.Kind != "dependency" || !slices.Equal(w.On, []string{"."}) {
		t.Fatalf("wait = %+v", w)
	}
	if w.Agent == nil || w.Agent.Stage != "basis" || w.Agent.Reason != "I need Root's basis" {
		t.Fatalf("the Manager's own reason for a named wait is joined by its stage: %+v", w.Agent)
	}
	if !anyContains(entryTexts(records), "stock reported waiting for the accepted basis of Root") {
		t.Fatalf("entries = %v", entryTexts(records))
	}
	for _, text := range []string{"the accepted result of billing", "the accepted basis of a/b"} {
		var out Wait
		out.Text = text
		classify(&out, &ManagerView{}, map[string]*ManagerView{}, nil)
		if out.Kind != "dependency" {
			t.Errorf("%q = %+v", text, out)
		}
	}
}

func TestOwnerChannelIsKnownOnlyWhenTheRecordHoldsOwnerRecords(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	records := s.records()
	if Project("r", records, len(records), Source{}, nil).OwnerChannel {
		t.Fatal("a Run has no owner channel")
	}
	s.add("question.asked", "p1", ".", "r-p1-root-r1", run.Question{ID: "Q1", From: ".", To: "user", Pass: "p1", Text: "?"})
	records = s.records()
	if !Project("r", records, len(records), Source{}, nil).OwnerChannel {
		t.Fatal("a legacy question to the owner keeps the owner in the record")
	}
	s2 := newScenario(t)
	startWithBinding(s2)
	s2.add("owner.error", "p1", ".", "", map[string]any{"question": "Q1"})
	records = s2.records()
	if !Project("r", records, len(records), Source{}, nil).OwnerChannel {
		t.Fatal("an owner error is an owner-channel record")
	}
}

func TestRunsDoNotPresentTheMissingReportOutcomeAsAGap(t *testing.T) {
	s := newScenario(t)
	startWithBinding(s)
	s.add("outcome", "", "", "", map[string]any{"status": "verified", "reason": ""})
	records := s.records()
	// The report as the producer writes it today: no outcome field, a no-change
	// publication whose Paths is null, and the Run's problems, turns and probes.
	report, err := decodeReport([]byte(`{"formatVersion":1,"operation":"apply","status":"passed","evidenceDirectory":"x",
		"publication":{"ID":"r","Outcome":"no_change","Paths":null,"Reason":""},
		"run":{"id":"r","status":"verified","passes":[],"findings":[],"questions":[],"problems":[{"id":"X1","manager":".","pass":"p1","category":"model","text":"t"}],
		"turns":[{"id":"t1"},{"id":"t2"}],
		"probes":[{"id":"R1","outcome":"passed"},{"id":"R2","outcome":"failed"},{"id":"R3","outcome":"error","disposition":{"as":"rerun","by":"x","pass":"p"}},{"id":"R4","outcome":"error"}],
		"usage":{"costUSD":1.5,"providerTurns":4}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if report.Run.Status != "verified" || report.Run.Turns != 2 || len(report.Run.Problems) != 1 || report.Run.Probes != (ProbeCounts{Total: 4, Passed: 1, Failed: 1, Error: 2, Undisposed: 2}) {
		t.Fatalf("run = %+v", report.Run)
	}
	if report.Publication == nil || report.Publication.Paths == nil || len(report.Publication.Paths) != 0 {
		t.Fatalf("publication = %+v", report.Publication)
	}
	v := Project("r", records, len(records), Source{Report: true}, report)
	for _, g := range v.Gaps {
		if strings.Contains(g.Code, "derived") || strings.Contains(g.Text, "no Run outcome field") {
			t.Fatalf("the producer's missing outcome is no gap of the record: %+v", g)
		}
		if g.Code == "publication.candidate" {
			t.Fatalf("a no-change publication writes nothing, so its missing candidate digest is no gap: %+v", g)
		}
	}
	if text := sentenceText(v.Account[0]); strings.Contains(text, "Run outcome:") || !strings.Contains(text, "nothing to write") {
		t.Fatalf("account = %q", text)
	}
}

func TestPassTitleNamesTheModelAssessment(t *testing.T) {
	if passTitle("assess") != "Model assessment" || passTitle("verify") != "Verify" || passTitle("") != "" {
		t.Fatal("pass titles")
	}
	s := newScenario(t)
	startWithBinding(s)
	s.add("candidate", "p1", "", "", map[string]any{"tree": "t"})
	s.add("pass.start", "p1", "", "", map[string]any{"purpose": "assess", "review": true, "writes": false, "tree": "t"})
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	if text := sentenceText(v.Account[0]); !strings.HasPrefix(text, "Model assessment Pass p1 is running") {
		t.Fatalf("account = %q", text)
	}
	s.add("pass.sealed", "p1", "", "", map[string]any{"purpose": "assess", "verdict": "satisfied", "tree": "t", "managers": []run.ManagerRecord{}})
	if !anyContains(entryTexts(s.records()), "Model assessment Pass p1 sealed satisfied") {
		t.Fatalf("entries = %v", entryTexts(s.records()))
	}
}
