package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestExtractParsingFixesTheManifestAndDispatches(t *testing.T) {
	dir := t.TempDir()
	config, runtimeProfile := filepath.Join(dir, "run.json"), filepath.Join(dir, "runtime.yaml")
	var observed ExtractRequest
	calls := 0
	c := New(nil, nil, nil, nil, nil, nil, nil).WithExtract(func(_ context.Context, request ExtractRequest) (ExtractReport, error) {
		calls++
		observed = request
		return ExtractReport{Status: model.NotCompleted}, nil
	})
	var out, stderr bytes.Buffer
	code := c.Run(context.Background(), []string{"extract", "--repo", dir, "--dependency", ".markitect/dependencies/core/model.yaml", "--dependency", ".markitect/dependencies/software/model.yaml", "--runtime-profile", runtimeProfile, "--run-config", config}, &out, &stderr)
	if code != 3 || calls != 1 || observed.RunConfig != config || observed.Selection.Repository != dir || observed.Selection.Local.ManifestPath.String() != ".markitect/model.yaml" || observed.Selection.Local.Boundary != "." ||
		len(observed.Selection.Dependencies) != 2 || observed.Selection.Dependencies[1].ManifestPath.String() != ".markitect/dependencies/software/model.yaml" || len(observed.Selection.RuntimeProfiles) != 1 || observed.Selection.RuntimeProfiles[0].Boundary != dir {
		t.Fatalf("extract arguments changed: code=%d calls=%d request=%+v stderr=%s", code, calls, observed, stderr.String())
	}
	var report struct {
		Operation     string `json:"operation"`
		Status        string `json:"status"`
		ModelComplete bool   `json:"modelComplete"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.Operation != "extract" || report.Status != "not_completed" || report.ModelComplete {
		t.Fatalf("%v: %s", err, out.String())
	}
}

func TestExtractRefusesInvalidInvocationsWithoutDispatch(t *testing.T) {
	dir := t.TempDir()
	config, runtimeProfile := filepath.Join(dir, "run.json"), filepath.Join(dir, "runtime.yaml")
	calls := 0
	c := New(nil, nil, nil, nil, nil, nil, nil).WithExtract(func(context.Context, ExtractRequest) (ExtractReport, error) {
		calls++
		return ExtractReport{Status: model.Passed}, nil
	})
	for _, args := range [][]string{
		{"extract"}, {"extract", "--repo", "absent"},
		{"extract", "--run-config", "relative.json", "--runtime-profile", runtimeProfile},
		{"extract", "--run-config", config},
		{"extract", "--run-config", config, "--runtime-profile", "relative.yaml"},
		// There is no --model flag: the manifest path is fixed.
		{"extract", "--run-config", config, "--runtime-profile", runtimeProfile, "--model", ".markitect/model.yaml"},
		{"extract", "--run-config", config, "--runtime-profile", runtimeProfile, "--revision", "HEAD"},
		{"extract", "--run-config", config, "--runtime-profile", runtimeProfile, "--text", "x"},
	} {
		var out, stderr bytes.Buffer
		if code := c.Run(context.Background(), args, &out, &stderr); code == 0 {
			t.Fatalf("incomplete or invalid invocation passed: %v", args)
		}
	}
	if calls != 0 {
		t.Fatal("invalid invocation reached execution")
	}
	// Without the operation installed, Extract is unavailable.
	var out, stderr bytes.Buffer
	if code := New(nil, nil, nil, nil, nil, nil, nil).Run(context.Background(), []string{"extract", "--run-config", config, "--runtime-profile", runtimeProfile}, &out, &stderr); code != 3 || !strings.Contains(out.String(), "unavailable") {
		t.Fatalf("%d %s", code, out.String())
	}
}

func TestExtractReportShape(t *testing.T) {
	started := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	structure := run.Outcome{Status: run.Verified, Passes: []run.PassRecord{{ID: "structure-1", Purpose: "structure", Verdict: run.Satisfied, Sealed: true}, {ID: "verify-2", Purpose: run.Verify, Review: true, Verdict: run.Satisfied, Sealed: true}}, Turns: []run.TurnRecord{{ID: "t1", Manager: ".", Kind: run.ExecuteTurn, Pass: "structure-1"}}, Usage: run.Usage{InputTokens: 3}}
	modelOutcome := run.Outcome{Status: run.Verified,
		Passes: []run.PassRecord{
			{ID: "model-1", Purpose: "model", Verdict: run.Satisfied, Sealed: true},
			{ID: "verify-2", Purpose: run.Verify, Review: true, Verdict: run.NeedsCorrection, Sealed: true, Managers: []run.ManagerRecord{{Manager: ".", State: run.Done, Verdict: run.NeedsCorrection}}},
			{ID: "model-3", Purpose: "model", Verdict: run.Satisfied, Sealed: true},
			{ID: "verify-4", Purpose: run.Verify, Review: true, Verdict: run.Satisfied, Sealed: true, Managers: []run.ManagerRecord{{Manager: ".", State: run.Done, Verdict: run.Satisfied, Summary: "fine"}, {Manager: "lending", State: run.Done, Verdict: run.Satisfied, Summary: "fine"}}}},
		Findings: []run.Finding{
			{ID: "f1", Owner: "lending", RaisedBy: "lending", Subject: "artifact", Definition: "lending/loan-limit", Obligation: "At most five loans", Observed: "borrow.go admits a sixth loan.", Blocking: false},
			{ID: "f4", Owner: "lending/reminders", RaisedBy: "lending", Subject: "artifact", Definition: "lending/loan-limit", Obligation: "At most five loans", Observed: "remind.go counts six.", Blocking: false},
			{ID: "f5", Owner: "lending", RaisedBy: "lending", Subject: "artifact", Definition: "lending/closed", Obligation: "Closed", Observed: "x", Closed: true},
			{ID: "f2", Owner: "lending", RaisedBy: "lending", Obligation: "Missing statement", Observed: "Refusal.", Blocking: true, Closed: true},
			{ID: "f3", Owner: "lending", RaisedBy: "lending", Obligation: "Better wording", Observed: "x", Blocking: false},
		},
		Questions: []run.Question{{ID: "q1", Text: "Which currency?", Asked: started}, {ID: "q2", Text: "Answered.", Asked: started, Answered: true, Answer: "euro"}},
		Turns:     []run.TurnRecord{{ID: "t2", Manager: ".", Kind: run.VerifyTurn, Pass: "verify-4"}}, Usage: run.Usage{InputTokens: 7},
	}
	receipt := &apply.PublicationReceipt{ID: "x-1", Outcome: apply.PublicationApplied}
	proposed := []apply.ProposedCheck{{ID: "project-tests", Profile: "go-tests", Directory: "realization", Args: []string{"test", "./..."}}}
	result := apply.ExtractResult{Structure: structure, Model: &modelOutcome, Receipt: receipt, ProposedChecks: proposed}
	report := NewExtractReport("x-1", result)
	report.FormatVersion, report.Operation, report.Status = 1, "extract", ExtractStatus(result, nil)
	if report.Status != model.Passed || !report.ModelComplete || report.Structure.ID != "x-1-structure" || report.Model.ID != "x-1-model" || report.Model.Usage.InputTokens != 7 || report.Structure.Usage.InputTokens != 3 {
		t.Fatalf("%+v", report)
	}
	if report.Verify == nil || report.Verify.Pass != "verify-4" || report.Verify.Verdict != run.Satisfied || len(report.Verify.Managers) != 2 {
		t.Fatalf("verify %+v", report.Verify)
	}
	if len(report.Deviations) != 1 || report.Deviations[0].Definition != "lending/loan-limit" || len(report.Deviations[0].Observed) != 2 || !slices.Equal(report.Deviations[0].Owners, []string{"lending", "lending/reminders"}) || len(report.OpenQuestions) != 1 || report.OpenQuestions[0].ID != "q1" {
		t.Fatalf("deviations %+v questions %+v", report.Deviations, report.OpenQuestions)
	}
	if len(report.ProposedChecks) != 1 || report.ProposedChecks[0].ID != "project-tests" {
		t.Fatalf("proposed checks %+v", report.ProposedChecks)
	}
	data, err := json.Marshal(ReportValue(report))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"formatVersion", "operation", "status", "modelComplete", "structure", "model", "verify", "deviations", "proposedChecks", "openQuestions", "publication"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("report lacks %q: %s", key, data)
		}
	}
	// Lists are never null, even for an Extract that stopped in the first phase.
	stopped := NewExtractReport("x-2", apply.ExtractResult{Structure: run.Outcome{Status: run.Incomplete, Reason: "budget"}})
	if stopped.Model != nil || stopped.Verify != nil || stopped.ModelComplete || stopped.Deviations == nil || stopped.ProposedChecks == nil || stopped.OpenQuestions == nil || stopped.Structure.Passes == nil || stopped.Structure.Problems == nil {
		t.Fatalf("%+v", stopped)
	}
}

func TestExtractStatusIsPassedOnlyForAPublishedVerifiedModel(t *testing.T) {
	verified := run.Outcome{Status: run.Verified}
	structure := run.Outcome{Status: run.Verified}
	for _, tc := range []struct {
		name   string
		result apply.ExtractResult
		err    error
		want   model.Status
	}{
		{"published", apply.ExtractResult{Structure: structure, Model: &verified, Receipt: &apply.PublicationReceipt{Outcome: apply.PublicationApplied}}, nil, model.Passed},
		{"no change", apply.ExtractResult{Structure: structure, Model: &verified, Receipt: &apply.PublicationReceipt{Outcome: apply.PublicationNoChange}}, nil, model.Passed},
		{"stale", apply.ExtractResult{Structure: structure, Model: &verified, Receipt: &apply.PublicationReceipt{Outcome: apply.PublicationStale}}, nil, model.NotCompleted},
		{"unpublished", apply.ExtractResult{Structure: structure, Model: &verified}, nil, model.NotCompleted},
		{"error", apply.ExtractResult{Structure: structure, Model: &verified, Receipt: &apply.PublicationReceipt{Outcome: apply.PublicationApplied}}, errors.New("read-back"), model.NotCompleted},
		{"publication failed", apply.ExtractResult{Structure: structure, Model: &verified, Receipt: &apply.PublicationReceipt{Outcome: apply.PublicationFailed}}, nil, model.Failed},
		{"structure incomplete", apply.ExtractResult{Structure: run.Outcome{Status: run.Incomplete}}, nil, model.NotCompleted},
		{"structure failed", apply.ExtractResult{Structure: run.Outcome{Status: run.Failed}}, errors.New("port"), model.Failed},
		{"model incomplete", apply.ExtractResult{Structure: structure, Model: &run.Outcome{Status: run.Incomplete}}, nil, model.NotCompleted},
	} {
		if got := ExtractStatus(tc.result, tc.err); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestExtractReportTakesOpenQuestionsFromTheFinalPhase(t *testing.T) {
	asked := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	answered := run.Question{ID: "Q1", Text: "Structure?", Asked: asked, Answered: true, Answer: "yes"}
	open := run.Question{ID: "Q2", Text: "Currency?", Asked: asked}
	// The model phase carries the structure phase's questions: Q1 is reported once or not at all.
	carried := NewExtractReport("x-1", apply.ExtractResult{Structure: run.Outcome{Status: run.Verified, Questions: []run.Question{answered}},
		Model: &run.Outcome{Status: run.Incomplete, Questions: []run.Question{answered, open}}})
	if len(carried.OpenQuestions) != 1 || carried.OpenQuestions[0].ID != "Q2" {
		t.Errorf("questions %+v", carried.OpenQuestions)
	}
	// A model phase that never started has none of its own: the structure's stay.
	unstarted := NewExtractReport("x-2", apply.ExtractResult{Structure: run.Outcome{Status: run.Verified, Questions: []run.Question{answered, open}},
		Model: &run.Outcome{Status: run.Failed}})
	if len(unstarted.OpenQuestions) != 1 || unstarted.OpenQuestions[0].ID != "Q2" {
		t.Errorf("questions %+v", unstarted.OpenQuestions)
	}
}

// libraryExtract scripts the result of an Extract of the small library: a
// verified, published model with one realization deviation and one proposed
// check.
func libraryExtract(t *testing.T) apply.ExtractResult {
	t.Helper()
	asked := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	var published []apply.PublishedPath
	for _, name := range []string{".markitect/execution.yaml", ".markitect/model/library/library.yaml", ".markitect/registry/registry.yaml"} {
		path, err := snapshot.NewPath(name)
		if err != nil {
			t.Fatal(err)
		}
		published = append(published, apply.PublishedPath{Path: path, State: apply.PathAfter})
	}
	var managers []run.ManagerRecord
	for _, m := range []string{".", "catalog", "lending", "lending/reminders", "library", "members"} {
		managers = append(managers, run.ManagerRecord{Manager: m, State: run.Done, Verdict: run.Satisfied, Summary: "fine"})
	}
	return apply.ExtractResult{
		Structure: run.Outcome{Status: run.Verified, Passes: []run.PassRecord{{ID: "verify-2", Purpose: run.Verify, Review: true, Verdict: run.Satisfied, Sealed: true}}},
		Model: &run.Outcome{Status: run.Verified,
			Passes: []run.PassRecord{{ID: "verify-1", Purpose: run.Verify, Review: true}, {ID: "verify-2", Purpose: run.Verify, Review: true, Verdict: run.Satisfied, Sealed: true, Managers: managers}},
			Findings: []run.Finding{
				{ID: "f1", Owner: "lending", RaisedBy: "lending", Subject: "artifact", Definition: "lending/lending", Observed: "borrow.go admits a sixth loan although the Definition limits a member to five loans at a time; the limit is only checked when a loan is renewed.", Blocking: false},
				{ID: "f2", Owner: "members", RaisedBy: "members", Subject: "model", Observed: "The status wording could be tighter.", Blocking: false},
				{ID: "f3", Owner: "catalog", RaisedBy: "catalog", Subject: "model", Observed: "Fixed.", Closed: true},
			},
			Questions: []run.Question{{ID: "Q1", Text: "Which currency do fines use?", Asked: asked, Answered: true, Answer: "euro"}},
		},
		Receipt:        &apply.PublicationReceipt{ID: "x-1", Outcome: apply.PublicationApplied, Paths: published},
		ProposedChecks: []apply.ProposedCheck{{ID: "project-vet", Profile: "go-tests", Directory: "realization", Args: []string{"vet", "./..."}, Timeout: "2m"}},
		Packages:       []apply.ExtractedPackage{{Path: "catalog"}, {Path: "lending"}, {Path: "lending/reminders"}, {Path: "library", Title: "Small library"}, {Path: "members"}},
	}
}

func accountOf(result apply.ExtractResult, err error, evidence string) string {
	report := NewExtractReport("x-1", result)
	report.Status, report.EvidenceDirectory = ExtractStatus(result, err), evidence
	if err != nil {
		report.Error = err.Error()
	}
	return RenderExtractAccount(report)
}

const libraryAccount = `Model complete and published.

Packages (5)
  catalog
  lending
  lending/reminders
  library            Small library
  members

Owner questions (1)
  Q1 Which currency do fines use? -> euro

Realization deviates from the model (1)
  lending/lending (owners: lending)
    borrow.go admits a sixth loan although the Definition limits a member to five loans at a time; the limit is only chec...

Proposed checks (1)
  project-vet: vet ./... in realization

Verify (Pass verify-2, verdict satisfied)
  satisfied: ., catalog, lending, lending/reminders, library, members
  open advisory findings: 1

Publication: applied, 3 files changed
Evidence: /state/x-1

Next: review and commit .markitect/model, .markitect/registry and
  .markitect/execution.yaml to adopt the model. Extract commits nothing.
`

func TestRenderExtractAccount(t *testing.T) {
	library := libraryExtract(t)
	asked := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

	incomplete := libraryExtract(t)
	incomplete.Receipt, incomplete.ProposedChecks, incomplete.Packages = nil, nil, nil
	incomplete.Model.Status, incomplete.Model.Reason = run.Incomplete, "owner question Q2 is still open"
	incomplete.Model.Passes = []run.PassRecord{{ID: "verify-1", Purpose: run.Verify, Review: true, Verdict: run.NeedsCorrection, Managers: []run.ManagerRecord{{Manager: ".", State: run.Done, Verdict: run.Satisfied}, {Manager: "lending", State: run.Done, Verdict: run.NeedsCorrection}}}}
	incomplete.Model.Questions = append(incomplete.Model.Questions, run.Question{ID: "Q2", Text: "May a member hold a reserved copy?", Asked: asked})
	incomplete.RetainedWork = "x-1-model-work"

	stale := libraryExtract(t)
	stale.Receipt = &apply.PublicationReceipt{ID: "x-1", Outcome: apply.PublicationStale, Reason: "the repository changed", Paths: slices.Clone(library.Receipt.Paths)}
	for i := range stale.Receipt.Paths {
		stale.Receipt.Paths[i].State = apply.PathBefore
	}

	failed := apply.ExtractResult{Structure: run.Outcome{Status: run.Failed, Reason: "the agent port broke"}}

	many := libraryExtract(t)
	for i := range 20 {
		many.Packages = append(many.Packages, apply.ExtractedPackage{Path: fmt.Sprintf("extra/p%02d", i)})
		many.Model.Findings = append(many.Model.Findings, run.Finding{ID: fmt.Sprintf("d%d", i), Owner: "lending", RaisedBy: "lending", Subject: "artifact", Definition: fmt.Sprintf("lending/d%02d", i), Observed: "x"})
	}

	for _, tc := range []struct {
		name     string
		result   apply.ExtractResult
		err      error
		evidence string
		want     []string // lines in this order
		absent   []string
	}{
		{"complete", library, nil, "/state/x-1", []string{
			"Model complete and published.", "Packages (5)", "  library            Small library", "Owner questions (1)", "  Q1 Which currency do fines use? -> euro",
			"Realization deviates from the model (1)", "  lending/lending (owners: lending)", "    borrow.go admits a sixth loan although the Definition limits a member to five loans at a time; the limit is only chec...",
			"Proposed checks (1)", "  project-vet: vet ./... in realization", "Verify (Pass verify-2, verdict satisfied)", "  satisfied: ., catalog, lending, lending/reminders, library, members", "  open advisory findings: 1",
			"Publication: applied, 3 files changed", "Evidence: /state/x-1", "Next: review and commit .markitect/model, .markitect/registry and", "  .markitect/execution.yaml to adopt the model. Extract commits nothing.",
		}, []string{"not satisfied", "Retained work"}},
		{"incomplete", incomplete, nil, "/state/x-1", []string{
			"Model incomplete: owner question Q2 is still open.", "Owner questions (2)", "  Q2 May a member hold a reserved copy? -> unanswered",
			"Verify (Pass verify-1, verdict needs_correction)", "  satisfied: .", "  not satisfied: lending", "Publication: none", "Retained work: x-1-model-work (kept for inspection)", "Evidence: /state/x-1",
		}, []string{"Next:", "Packages", "Proposed checks"}},
		{"verified but stale", stale, nil, "", []string{
			"Model incomplete: the verified model was not published (stale: the repository changed).", "Publication: stale, 0 of 3 files written: the repository changed",
		}, []string{"Next:", "Evidence"}},
		{"failed", failed, errors.New("agent port: closed"), "", []string{
			"Extract failed: structure phase: the agent port broke; agent port: closed.", "Verify: no Pass finished", "Publication: none",
		}, []string{"Next:", "Owner questions", "Realization deviations"}},
		{"capped lists", many, nil, "", []string{"Packages (25)", "  and 13 more", "Realization deviates from the model (21)", "  and 16 more"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := accountOf(tc.result, tc.err, tc.evidence)
			lines := strings.Split(strings.TrimSuffix(account, "\n"), "\n")
			next := 0
			for _, want := range tc.want {
				for next < len(lines) && lines[next] != want {
					next++
				}
				if next == len(lines) {
					t.Fatalf("line %q is missing or out of order in:\n%s", want, account)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(account, absent) {
					t.Errorf("account contains %q:\n%s", absent, account)
				}
			}
			if !strings.HasSuffix(account, "\n") || strings.Contains(account, "\x1b") {
				t.Errorf("account must be plain text ending in a newline: %q", account)
			}
		})
	}

	if got := accountOf(library, nil, "/state/x-1"); got != libraryAccount {
		t.Errorf("library account changed:\n%s", got)
	}
	if n := strings.Count(libraryAccount, "\n"); n > 40 {
		t.Errorf("the library account has %d lines", n)
	}
}

func TestExtractPrintsTheAccountToStderrAndKeepsStdoutJSON(t *testing.T) {
	dir := t.TempDir()
	config, runtimeProfile := filepath.Join(dir, "run.json"), filepath.Join(dir, "runtime.yaml")
	result := libraryExtract(t)
	var report ExtractReport
	c := New(nil, nil, nil, nil, nil, nil, nil).WithExtract(func(context.Context, ExtractRequest) (ExtractReport, error) {
		report = NewExtractReport("x-1", result)
		report.Status, report.EvidenceDirectory = model.Passed, "/state/x-1"
		return report, nil
	})
	var out, stderr bytes.Buffer
	if code := c.Run(context.Background(), []string{"extract", "--repo", dir, "--runtime-profile", runtimeProfile, "--run-config", config}, &out, &stderr); code != 0 {
		t.Fatalf("code %d: %s", code, stderr.String())
	}
	// stdout is exactly the JSON report, which does not carry the account's data.
	report.FormatVersion, report.Operation = 1, "extract"
	var want bytes.Buffer
	encoder := json.NewEncoder(&want)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ReportValue(report)); err != nil {
		t.Fatal(err)
	}
	if out.String() != want.String() || strings.Contains(out.String(), "Small library") {
		t.Errorf("stdout changed:\n%s", out.String())
	}
	// stderr ends with the account, after everything else the command wrote.
	if !strings.HasSuffix(stderr.String(), RenderExtractAccount(report)) || !strings.HasPrefix(stderr.String(), "Model complete and published.\n") {
		t.Errorf("stderr:\n%s", stderr.String())
	}
}
