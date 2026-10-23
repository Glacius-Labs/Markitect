package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestApplyParsingPreservesSelectionAndReportsUnavailable(t *testing.T) {
	dir := t.TempDir()
	config, runtimeProfile := filepath.Join(dir, "run.json"), filepath.Join(dir, "runtime.yaml")
	var observed ApplyRequest
	calls := 0
	c := New(nil, nil, nil, nil, nil, func(_ context.Context, request ApplyRequest) (ApplyReport, error) {
		calls++
		observed = request
		return ApplyReport{Status: model.NotCompleted}, errors.New("Apply unavailable until S3")
	})
	var out, stderr bytes.Buffer
	code := c.Run(context.Background(), []string{"apply", "--repo", dir, "--dependency", "vendor/model.yaml", "--runtime-profile", runtimeProfile, "--run-config", config}, &out, &stderr)
	if code != 3 || calls != 1 || observed.RunConfig != config || observed.Selection.Repository != dir || len(observed.Selection.Dependencies) != 1 || observed.Selection.Dependencies[0].ManifestPath.String() != "vendor/model.yaml" || len(observed.Selection.RuntimeProfiles) != 1 || observed.Build != buildinfo.Build() {
		t.Fatalf("code=%d calls=%d selected=%+v err=%s", code, calls, observed, stderr.String())
	}
	var report map[string]any
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report["operation"] != "apply" || report["status"] != "not_completed" || report["outcome"] != "incomplete" || report["error"] == "" {
		t.Fatal(report)
	}
	if _, exists := report["run"]; exists {
		t.Fatal("unavailable command invented a Run", report)
	}
}

func TestApplyRefusesInvalidInvocationsWithoutDispatch(t *testing.T) {
	dir := t.TempDir()
	config, runtimeProfile := filepath.Join(dir, "run.json"), filepath.Join(dir, "runtime.yaml")
	calls := 0
	c := New(nil, nil, nil, nil, nil, func(context.Context, ApplyRequest) (ApplyReport, error) {
		calls++
		return ApplyReport{Status: model.Passed}, nil
	})
	for _, args := range [][]string{
		{"apply"}, {"apply", "--repo", "absent"},
		{"apply", "--run-config", "relative.json", "--runtime-profile", runtimeProfile},
		{"apply", "--run-config", config, "--runtime-profile", runtimeProfile, "--revision", "HEAD"},
		{"apply", "--run-config", config, "--runtime-profile", runtimeProfile, "--metadata"},
		{"apply", "--run-config", config, "--runtime-profile", runtimeProfile, "--text", "x"},
		// The answer command is gone: a Run is closed.
		{"answer"}, {"answer", "q", "--run", "run-1", "--text", "x", "--run-config", config},
		{"apply", "--run-config", config, "--runtime-profile", runtimeProfile, "--decision", "x"},
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
		})
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
