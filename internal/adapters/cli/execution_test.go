package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestApplyParsingPreservesSelectionAndReportsTheRun(t *testing.T) {
	dir := t.TempDir()
	config, runtimeProfile := filepath.Join(dir, "run.json"), filepath.Join(dir, "runtime.yaml")
	var observed ApplyRequest
	calls := 0
	c := New(nil, nil, nil, nil, nil, func(_ context.Context, request ApplyRequest) (ApplyReport, error) {
		calls++
		observed = request
		started := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
		outcome := run.Outcome{
			Status: run.Incomplete, Reason: "owner question stayed open",
			Passes:    []run.PassRecord{{ID: "verify-1", Purpose: run.Verify, Review: true, Tree: "tree-1", Verdict: run.Satisfied, Sealed: true, Checks: []run.CheckResult{{Name: "unit", Passed: true}}, Managers: []run.ManagerRecord{{Manager: ".", State: run.Done, Result: "r1", Verdict: run.Satisfied, Summary: "fine"}}}},
			Questions: []run.Question{{ID: "q1", Text: "Which currency?", Asked: started}},
			Problems:  []run.Problem{{ID: "p1", Manager: "lending", Pass: "plan-1", Category: "model", Text: "The loan limit contradicts itself.", Escalated: true}},
			Turns:     []run.TurnRecord{{ID: "t1", Manager: ".", Kind: run.VerifyTurn, Pass: "verify-1", Started: started, Ended: started.Add(time.Minute), Usage: run.Usage{InputTokens: 7}}},
			Usage:     run.Usage{InputTokens: 7},
		}
		return ApplyReport{Status: ApplyStatus(outcome, nil, nil), Run: NewRunReport("run-1", outcome), EvidenceDirectory: filepath.Join(dir, "runs", "run-1")}, nil
	}, nil)
	var out, stderr bytes.Buffer
	code := c.Run(context.Background(), []string{"apply", "--repo", dir, "--model", ".markitect/model.yaml", "--dependency", "vendor/model.yaml", "--runtime-profile", runtimeProfile, "--run-config", config}, &out, &stderr)
	if code != 3 || calls != 1 || observed.RunConfig != config || observed.Selection.Repository != dir || len(observed.Selection.Dependencies) != 1 || observed.Selection.Dependencies[0].ManifestPath.String() != "vendor/model.yaml" || len(observed.Selection.RuntimeProfiles) != 1 {
		t.Fatalf("apply arguments changed: code=%d calls=%d stderr=%s", code, calls, stderr.String())
	}
	var report struct {
		Operation string `json:"operation"`
		Status    string `json:"status"`
		Outcome   string `json:"outcome"`
		Run       struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Reason string `json:"reason"`
			Passes []struct {
				ID       string
				Verdict  string
				Checks   []struct{ Name string }
				Managers []struct{ Manager, State, Verdict string }
			}
			Problems  []struct{ Manager, Category string }
			Questions []struct {
				ID    string `json:"id"`
				Asked string `json:"asked"`
			}
			Turns []struct {
				Started string
				Usage   struct{ InputTokens int }
			}
			Usage struct{ InputTokens int }
		} `json:"run"`
		EvidenceDirectory string `json:"evidenceDirectory"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if report.Outcome != "incomplete" {
		t.Fatalf("a not completed Apply must report the incomplete outcome: %s", out.String())
	}
	if report.Operation != "apply" || report.Status != "not_completed" || report.Run.ID != "run-1" || report.Run.Status != "incomplete" || report.Run.Reason == "" ||
		len(report.Run.Passes) != 1 || report.Run.Passes[0].ID != "verify-1" || report.Run.Passes[0].Verdict != "satisfied" || report.Run.Passes[0].Checks[0].Name != "unit" || report.Run.Passes[0].Managers[0].State != "done" || len(report.Run.Problems) != 1 || report.Run.Problems[0].Manager != "lending" || report.Run.Questions[0].ID != "q1" || report.Run.Questions[0].Asked != "2026-10-19T12:00:00Z" ||
		report.Run.Turns[0].Started != "2026-10-19T12:00:00Z" || report.Run.Turns[0].Usage.InputTokens != 7 || report.Run.Usage.InputTokens != 7 || report.EvidenceDirectory == "" {
		t.Fatalf("run was not reported: %s", out.String())
	}
}

func TestApplyAndAnswerRefuseInvalidInvocationsWithoutDispatch(t *testing.T) {
	dir := t.TempDir()
	config, runtimeProfile := filepath.Join(dir, "run.json"), filepath.Join(dir, "runtime.yaml")
	calls := 0
	c := New(nil, nil, nil, nil, nil, func(context.Context, ApplyRequest) (ApplyReport, error) {
		calls++
		return ApplyReport{Status: model.Passed}, nil
	}, func(context.Context, AnswerRequest) (AnswerReport, error) {
		calls++
		return AnswerReport{Status: model.Passed}, nil
	})
	for _, args := range [][]string{
		{"apply"}, {"apply", "--repo", "absent"},
		{"apply", "--run-config", "relative.json", "--runtime-profile", runtimeProfile},
		{"apply", "--run-config", config, "--runtime-profile", runtimeProfile, "--revision", "HEAD"},
		{"apply", "--run-config", config, "--runtime-profile", runtimeProfile, "--metadata"},
		{"apply", "--run-config", config, "--runtime-profile", runtimeProfile, "--text", "x"},
		{"answer"}, {"answer", "q"}, {"answer", "q", "--run-config", config}, {"answer", "q", "--run", "run-1", "--text", "x"},
		{"answer", "q", "--run", "run-1", "--text", "x", "--run-config", "relative.json"},
		{"answer", "q", "--run", "run-1", "--text", "x", "--run-config", config, "--runtime-profile", runtimeProfile},
		{"answer", "--run", "run-1", "--text", "x", "--run-config", config},
		// The run is required: an answer never reaches a run it was not made for.
		{"answer", "q", "--text", "x", "--run-config", config},
		{"answer", "q", "--run", "run-1", "--run-config", config},
	} {
		var out, stderr bytes.Buffer
		if code := c.Run(context.Background(), args, &out, &stderr); code == 0 {
			t.Fatalf("incomplete or invalid invocation passed: %v", args)
		}
	}
	if calls != 0 {
		t.Fatal("invalid invocation reached execution")
	}
}

func TestAnswerPassesQuestionTextAndConfigAndReportsTheWrite(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "run.json")
	var observed AnswerRequest
	c := New(nil, nil, nil, nil, nil, nil, func(_ context.Context, request AnswerRequest) (AnswerReport, error) {
		observed = request
		return AnswerReport{Status: model.Passed, Path: filepath.Join(dir, "answers", "q1.txt"), Bytes: 14}, nil
	})
	var out, stderr bytes.Buffer
	if code := c.Run(context.Background(), []string{"answer", "q1", "--run", "run-1", "--text", "literal answer", "--run-config", config}, &out, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if observed != (AnswerRequest{Question: "q1", Run: "run-1", Text: "literal answer", RunConfig: config}) {
		t.Fatalf("answer arguments changed: %+v", observed)
	}
	if !strings.Contains(out.String(), `"operation": "answer"`) || !strings.Contains(out.String(), `"question": "q1"`) || !strings.Contains(out.String(), "q1.txt") {
		t.Fatal(out.String())
	}
}

func TestApplyStatusNeverConvertsErrorOrUnpublishedRunsIntoSuccess(t *testing.T) {
	receipt := func(outcome apply.PublicationOutcome) *apply.PublicationReceipt {
		return &apply.PublicationReceipt{Outcome: outcome}
	}
	for _, tc := range []struct {
		name    string
		outcome run.Outcome
		receipt *apply.PublicationReceipt
		err     error
		want    model.Status
	}{
		{"applied", run.Outcome{Status: run.Verified}, receipt(apply.PublicationApplied), nil, model.Passed},
		{"no change", run.Outcome{Status: run.Verified}, receipt(apply.PublicationNoChange), nil, model.Passed},
		{"applied with error", run.Outcome{Status: run.Verified}, receipt(apply.PublicationApplied), errors.New("read-back"), model.NotCompleted},
		{"verified without receipt", run.Outcome{Status: run.Verified}, nil, nil, model.NotCompleted},
		{"stale", run.Outcome{Status: run.Verified}, receipt(apply.PublicationStale), nil, model.NotCompleted},
		{"partial", run.Outcome{Status: run.Verified}, receipt(apply.PublicationPartial), nil, model.NotCompleted},
		{"unknown", run.Outcome{Status: run.Verified}, receipt(apply.PublicationUnknown), nil, model.NotCompleted},
		{"publication failed", run.Outcome{Status: run.Verified}, receipt(apply.PublicationFailed), nil, model.Failed},
		{"incomplete", run.Outcome{Status: run.Incomplete}, nil, nil, model.NotCompleted},
		{"failed", run.Outcome{Status: run.Failed}, nil, nil, model.Failed},
		{"failed with the error that always comes with it", run.Outcome{Status: run.Failed}, nil, errors.New("rejection limit reached"), model.Failed},
		{"incomplete with error", run.Outcome{Status: run.Incomplete}, nil, errors.New("budget"), model.NotCompleted},
		{"zero", run.Outcome{}, nil, nil, model.NotCompleted},
	} {
		if got := ApplyStatus(tc.outcome, tc.receipt, tc.err); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestExecutionExitCodesFollowTheReportedStatus(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		status model.Status
		err    error
		code   int
	}{
		{model.Passed, nil, 0}, {model.Failed, nil, 1}, {model.Failed, errors.New("run failed"), 1}, {model.NotCompleted, nil, 3},
		{model.Passed, errors.New("actual execution refusal"), 3}, {"", nil, 3},
	} {
		c := New(nil, nil, nil, nil, nil, func(context.Context, ApplyRequest) (ApplyReport, error) {
			return ApplyReport{Status: tc.status}, tc.err
		}, nil)
		var out, stderr bytes.Buffer
		if code := c.Run(context.Background(), []string{"apply", "--runtime-profile", filepath.Join(dir, "runtime.yaml"), "--run-config", filepath.Join(dir, "run.json")}, &out, &stderr); code != tc.code {
			t.Fatalf("status=%s err=%v code=%d", tc.status, tc.err, code)
		}
	}
}

func TestRunOutcomeNamesSuccessFailedAndIncompleteOnly(t *testing.T) {
	for status, want := range map[model.Status]string{model.Passed: "success", model.Failed: "failed", model.NotCompleted: "incomplete", "": "incomplete"} {
		if got := RunOutcome(status); got != want {
			t.Errorf("RunOutcome(%q) = %q, want %q", status, got, want)
		}
	}
}
