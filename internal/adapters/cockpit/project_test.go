package cockpit

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func managerOf(t *testing.T, v View, path string) ManagerView {
	t.Helper()
	for _, m := range v.Managers {
		if m.Path == path {
			return m
		}
	}
	t.Fatalf("no Manager %s in the view", path)
	return ManagerView{}
}

func questionOf(v View, id string) *QuestionView {
	for i := range v.Questions {
		if v.Questions[i].ID == id {
			return &v.Questions[i]
		}
	}
	return nil
}

func gapCodes(v View) []string {
	var out []string
	for _, g := range v.Gaps {
		out = append(out, g.Code)
	}
	return out
}

func TestProjectShowsNothingAfterItsBoundary(t *testing.T) {
	s, at := library(t)
	records := s.records()
	report, err := decodeReport(mustJSON(t, applyReport("success", "applied")))
	if err != nil {
		t.Fatal(err)
	}
	before := Project("r", records, at["asked"], Source{}, report)
	if q := questionOf(before, "Q1"); q == nil || q.Status != "open" || q.Answer != "" {
		t.Fatalf("Q1 before its answer = %+v, want open without answer", q)
	}
	if len(before.Findings) != 0 || before.Status.Outcome != "" || before.Status.Report != nil {
		t.Fatalf("a later finding, outcome or report leaked into the view at %d: %+v %+v", at["asked"], before.Findings, before.Status)
	}
	for _, p := range before.Passes {
		if p.ID == "p3" || p.Sealed && p.ID == "p2" {
			t.Fatalf("Pass %s appears before it was recorded", p.ID)
		}
	}
	after := Project("r", records, at["answered"], Source{}, report)
	if q := questionOf(after, "Q1"); q == nil || q.Status != "answered" || q.Answer == "" || q.AnsweredBy != "." {
		t.Fatalf("Q1 after its answer = %+v", q)
	}
	end := Project("r", records, len(records), Source{Report: true}, report)
	if end.Status.Outcome != "verified" || end.Status.Report == nil || end.Status.Report.Outcome != "success" {
		t.Fatalf("final status = %+v", end.Status)
	}
	notFinal := Project("r", records, len(records)-1, Source{Report: true}, report)
	if notFinal.Status.Report != nil {
		t.Fatal("the report appears before the final boundary")
	}
}

func TestProjectCorrectionAndFreshVerifyOnChangedCandidate(t *testing.T) {
	s, at := library(t)
	records := s.records()
	v := Project("r", records, at["correction"], Source{}, nil)
	p4 := v.Passes[len(v.Passes)-1]
	if p4.ID != "p4" || !p4.Correction || !slices.Equal(p4.Addresses, []string{"F1"}) {
		t.Fatalf("correction Pass = %+v", p4)
	}
	if m := managerOf(t, v, "lending"); !slices.Equal(m.Findings, []string{"F1"}) {
		t.Fatalf("lending's open findings = %v", m.Findings)
	}
	end := Project("r", records, len(records), Source{}, nil)
	f := end.Findings[0]
	if !f.Closed || f.ClosedAs != "resolved" || f.ClosedSeq != at["closed"] || f.Candidate != "tree-a" {
		t.Fatalf("closed finding = %+v", f)
	}
	var p3, p5 PassView
	for _, p := range end.Passes {
		switch p.ID {
		case "p3":
			p3 = p
		case "p5":
			p5 = p
		}
	}
	if p3.Candidate != "tree-a" || p5.Candidate != "tree-b" || p3.Verdict != "needs_correction" || p5.Verdict != "satisfied" {
		t.Fatalf("Verify Passes = %+v / %+v", p3, p5)
	}
	if len(p3.Checks) != 1 || !p3.Checks[0].Passed || p3.StartSeq == 0 {
		t.Fatalf("checks recorded before pass.start belong to their Pass: %+v", p3)
	}
}

func TestProjectParallelBranchesAndCrossBranchRoute(t *testing.T) {
	s, at := library(t)
	records := s.records()
	v := Project("r", records, at["parallel"], Source{}, nil)
	catalog, lending := managerOf(t, v, "catalog"), managerOf(t, v, "lending")
	if catalog.State != "working" || lending.State != "working" || catalog.Turn == nil || lending.Turn == nil || !lending.Turn.ReadOnly || catalog.Turn.ReadOnly {
		t.Fatalf("parallel turns: catalog %+v, lending %+v", catalog, lending)
	}
	var route []string
	for _, e := range Entries(records[:at["answered"]]) {
		if e.Link != nil && strings.HasPrefix(e.Kind, "question.") {
			route = append(route, e.Link.From+">"+e.Link.To)
		}
	}
	want := []string{"lending>.", ".>catalog", "catalog>.", ".>lending"}
	if !slices.Equal(route, want) {
		t.Fatalf("question route = %v, want %v", route, want)
	}
	if q := questionOf(Project("r", records, at["answered"], Source{}, nil), "Q2"); q == nil || q.About != "Q1" {
		t.Fatalf("the forwarded question keeps its correlation: %+v", q)
	}
}

func TestProjectWaitCauseAndOutdatedWait(t *testing.T) {
	s, at := library(t)
	records := s.records()
	v := Project("r", records, at["dependency"], Source{}, nil)
	w := managerOf(t, v, "lending").Wait
	if w == nil || w.Kind != "dependency" || !slices.Equal(w.On, []string{"catalog"}) || w.Outdated != "" {
		t.Fatalf("lending's wait = %+v", w)
	}
	if len(w.Cause) == 0 || w.Cause[0].Node != "catalog" || !strings.Contains(w.Cause[0].Doing, "turn m1") {
		t.Fatalf("cause = %+v", w.Cause)
	}
	if d := managerOf(t, v, "lending").Declared; len(d) != 1 || d[0].Owner != "catalog" || d[0].Accepted {
		t.Fatalf("declared dependencies = %+v", d)
	}
	stale := Project("r", records, at["stale"], Source{}, nil)
	w = managerOf(t, stale, "lending").Wait
	if w == nil || !strings.Contains(w.Outdated, "accepted at record") || len(w.On) != 0 {
		t.Fatalf("a wait whose dependency was accepted later is outdated, not current: %+v", w)
	}
	if !slices.Contains(gapCodes(stale), "wait.unrecorded") {
		t.Fatalf("gaps = %v", gapCodes(stale))
	}
	early := Project("r", records, at["released"]-2, Source{}, nil)
	if w := managerOf(t, early, "lending").Wait; w == nil || w.Kind != "parent-basis" || w.Outdated != "" {
		t.Fatalf("before release the parent-basis wait holds: %+v", w)
	}
	released := Project("r", records, at["released"], Source{}, nil)
	if w := managerOf(t, released, "lending").Wait; w == nil || !strings.Contains(w.Outdated, "released at record") {
		t.Fatalf("after release the parent-basis wait is outdated: %+v", w)
	}
}

func TestProjectLapsedOwnerQuestion(t *testing.T) {
	s, _ := library(t)
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	q := questionOf(v, "Q3")
	if q == nil || q.Status != "lapsed" || q.To != "user" || q.LapsedSeq == 0 {
		t.Fatalf("Q3 = %+v", q)
	}
	for _, sentence := range v.Account {
		for _, seg := range sentence {
			if seg.Ref == "question:Q3" {
				t.Fatalf("a lapsed question stays in the account as open: %+v", sentence)
			}
		}
	}
}

func TestProjectNoChangeAndInterruptedRuns(t *testing.T) {
	s, _ := library(t)
	records := s.records()
	report, _ := decodeReport(mustJSON(t, applyReport("success", "no_change")))
	v := Project("r", records, len(records), Source{Report: true}, report)
	if text := sentenceText(v.Account[0]); !strings.Contains(text, "no change") || !strings.Contains(text, "nothing to write") || strings.Contains(text, "0 of 0") {
		t.Fatalf("no-change account = %q", text)
	}
	cut := records[:len(records)-1]
	v = Project("r", cut, len(cut), Source{}, nil)
	if v.Status.Outcome != "" || !slices.Contains(gapCodes(v), "run.no_outcome") {
		t.Fatalf("a record without an outcome says so: %+v %v", v.Status, gapCodes(v))
	}
	v = Project("r", records, len(records), Source{}, nil)
	if !slices.Contains(gapCodes(v), "run.no_report") {
		t.Fatalf("an outcome without report.json says so: %v", gapCodes(v))
	}
}

func TestProjectKeepsUnreadableAndUnknownRecords(t *testing.T) {
	records := []Record{
		decodeRecord(1, []byte(`{"at":"2026-10-20T12:00:00Z","kind":"run.start","data":{"managers":["."]}}`)),
		decodeRecord(2, []byte(`{not json`)),
		decodeRecord(3, []byte(`{"at":"2026-10-20T12:00:01Z","kind":"something.new","data":{}}`)),
		decodeRecord(4, []byte(`{"at":"2026-10-20T12:00:02Z","kind":"pass.start","pass":"p1","data":"unexpected"}`)),
	}
	v := Project("r", records, 4, Source{}, nil)
	for _, code := range []string{"record.invalid", "record.kind", "record.data"} {
		if !slices.Contains(gapCodes(v), code) {
			t.Fatalf("gap %s missing: %v", code, gapCodes(v))
		}
	}
	if e := Entries(records)[1]; e.Seq != 2 || e.Class != "unreadable" {
		t.Fatalf("an unreadable line keeps its place: %+v", e)
	}
}

func TestEntriesSeparateAgentAndOrchestratorChannels(t *testing.T) {
	s, _ := library(t)
	channels := map[string]string{}
	for _, e := range Entries(s.records()) {
		if e.Link != nil {
			channels[e.Kind+" "+e.Link.From] = e.Link.Channel
		}
	}
	for key, want := range map[string]string{
		"basis.accepted @orchestrator": "orchestrator", "result.accepted .": "agent", "proposal.submitted catalog": "submission",
		"question.asked lending": "agent", "question.asked .": "owner", "manager.release @orchestrator": "orchestrator",
	} {
		if channels[key] != want {
			t.Fatalf("%s channel = %q, want %q (all: %v)", key, channels[key], want, channels)
		}
	}
}

func TestSelectBoundsDetailToTheBoundary(t *testing.T) {
	s, at := library(t)
	records := s.records()
	d := Select(records, at["asked"], "question:Q1")
	if len(d.Records) != 1 || d.Records[0].Kind != "question.asked" {
		t.Fatalf("Q1 detail before its answer = %+v", d.Records)
	}
	d = Select(records, len(records), "finding:F1")
	if len(d.Records) != 2 {
		t.Fatalf("F1 detail = %+v", d.Records)
	}
	if d := Select(records, len(records), "turn:r-p2-lending-m1"); len(d.Records) < 3 {
		t.Fatalf("a turn's detail lists what it recorded: %+v", d.Records)
	}
	if d := Select(records, len(records), "unknown:x"); len(d.Records) != 0 {
		t.Fatal("an unknown reference selects nothing")
	}
}

func sentenceText(s Sentence) string {
	var b strings.Builder
	for _, seg := range s {
		b.WriteString(seg.Text)
	}
	return b.String()
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
