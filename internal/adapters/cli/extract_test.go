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

func TestExtractParsingFixesTheManifestAndDispatches(t *testing.T) {
	dir := t.TempDir()
	config, host := filepath.Join(dir, "run.json"), filepath.Join(dir, "host.yaml")
	var observed ExtractRequest
	calls := 0
	c := New(nil, nil, nil, nil, nil, nil, nil).WithExtract(func(_ context.Context, request ExtractRequest) (ExtractReport, error) {
		calls++
		observed = request
		return ExtractReport{Status: model.NotCompleted}, nil
	})
	var out, stderr bytes.Buffer
	code := c.Run(context.Background(), []string{"extract", "--repo", dir, "--dependency", ".markitect/dependencies/core/model.yaml", "--dependency", ".markitect/dependencies/software/model.yaml", "--host-profile", host, "--run-config", config}, &out, &stderr)
	if code != 3 || calls != 1 || observed.RunConfig != config || observed.Selection.Repository != dir || observed.Selection.Local.ManifestPath.String() != ".markitect/model.yaml" || observed.Selection.Local.Boundary != "." ||
		len(observed.Selection.Dependencies) != 2 || observed.Selection.Dependencies[1].ManifestPath.String() != ".markitect/dependencies/software/model.yaml" || len(observed.Selection.HostProfiles) != 1 || observed.Selection.HostProfiles[0].Boundary != dir {
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
	config, host := filepath.Join(dir, "run.json"), filepath.Join(dir, "host.yaml")
	calls := 0
	c := New(nil, nil, nil, nil, nil, nil, nil).WithExtract(func(context.Context, ExtractRequest) (ExtractReport, error) {
		calls++
		return ExtractReport{Status: model.Passed}, nil
	})
	for _, args := range [][]string{
		{"extract"}, {"extract", "--repo", "absent"},
		{"extract", "--run-config", "relative.json", "--host-profile", host},
		{"extract", "--run-config", config},
		{"extract", "--run-config", config, "--host-profile", "relative.yaml"},
		// There is no --model flag: the manifest path is fixed.
		{"extract", "--run-config", config, "--host-profile", host, "--model", ".markitect/model.yaml"},
		{"extract", "--run-config", config, "--host-profile", host, "--revision", "HEAD"},
		{"extract", "--run-config", config, "--host-profile", host, "--text", "x"},
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
	if code := New(nil, nil, nil, nil, nil, nil, nil).Run(context.Background(), []string{"extract", "--run-config", config, "--host-profile", host}, &out, &stderr); code != 3 || !strings.Contains(out.String(), "unavailable") {
		t.Fatalf("%d %s", code, out.String())
	}
}

func TestExtractReportShape(t *testing.T) {
	started := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	structure := run.Outcome{Status: run.Verified, Rounds: []run.Round{{Number: 1, Green: true}}, Turns: []run.TurnRecord{{ID: "t1", Manager: ".", Kind: run.ExecuteTurn, Round: 1}}, Usage: run.Usage{InputTokens: 3}}
	modelOutcome := run.Outcome{Status: run.Verified,
		Rounds: []run.Round{{Number: 1, Assessments: []run.Assessment{{Manager: ".", Satisfied: false}}}, {Number: 2, Green: true, Assessments: []run.Assessment{{Manager: ".", Satisfied: true, Summary: "fine"}, {Manager: "lending", Satisfied: true, Summary: "fine"}}}},
		Findings: []run.Finding{
			{ID: "f1", Owner: "lending", RaisedBy: "lending", Obligation: "Realization deviation: lending/loan-limit", Observed: "borrow.go admits a sixth loan.", Blocking: false},
			{ID: "f2", Owner: "lending", RaisedBy: "lending", Obligation: "Missing statement", Observed: "Refusal.", Blocking: true, Closed: true},
			{ID: "f3", Owner: "lending", RaisedBy: "lending", Obligation: "Better wording", Observed: "x", Blocking: false},
		},
		Questions: []run.Question{{ID: "q1", Text: "Which currency?", Asked: started}, {ID: "q2", Text: "Answered.", Asked: started, Answered: true, Answer: "euro"}},
		Turns:     []run.TurnRecord{{ID: "t2", Manager: ".", Kind: run.VerifyTurn, Round: 2}}, Usage: run.Usage{InputTokens: 7},
	}
	receipt := &apply.PublicationReceipt{ID: "x-1", Outcome: apply.PublicationApplied}
	result := apply.ExtractResult{Structure: structure, Model: &modelOutcome, Receipt: receipt}
	report := NewExtractReport("x-1", result)
	report.FormatVersion, report.Operation, report.Status = 1, "extract", ExtractStatus(result, nil)
	if report.Status != model.Passed || !report.ModelComplete || report.Structure.ID != "x-1-structure" || report.Model.ID != "x-1-model" || report.Model.Usage.InputTokens != 7 || report.Structure.Usage.InputTokens != 3 {
		t.Fatalf("%+v", report)
	}
	if report.Verify == nil || report.Verify.Round != 2 || !report.Verify.Green || len(report.Verify.Assessments) != 2 {
		t.Fatalf("verify %+v", report.Verify)
	}
	if len(report.Deviations) != 1 || report.Deviations[0].ID != "f1" || len(report.OpenQuestions) != 1 || report.OpenQuestions[0].ID != "q1" {
		t.Fatalf("deviations %+v questions %+v", report.Deviations, report.OpenQuestions)
	}
	data, err := json.Marshal(ReportValue(report))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"formatVersion", "operation", "status", "modelComplete", "structure", "model", "verify", "deviations", "openQuestions", "publication"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("report lacks %q: %s", key, data)
		}
	}
	// Lists are never null, even for an Extract that stopped in the first phase.
	stopped := NewExtractReport("x-2", apply.ExtractResult{Structure: run.Outcome{Status: run.Incomplete, Reason: "budget"}})
	if stopped.Model != nil || stopped.Verify != nil || stopped.ModelComplete || stopped.Deviations == nil || stopped.OpenQuestions == nil || stopped.Structure.Rounds == nil {
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
