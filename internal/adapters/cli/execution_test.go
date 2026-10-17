package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestExecutionParsingPreservesSelectionAndActualResult(t *testing.T) {
	dir := t.TempDir()
	config, host := filepath.Join(dir, "run.json"), filepath.Join(dir, "host.yaml")
	var observed ExecutionRequest
	calls := 0
	c := New(nil, nil, nil, nil, nil, func(_ context.Context, request ExecutionRequest) (ExecutionReport, error) {
		calls++
		observed = request
		return ExecutionReport{Status: model.NotCompleted, OpenQuestions: []string{"actual-open-question"}}, nil
	})
	var out, stderr bytes.Buffer
	code := c.Run(context.Background(), []string{"answer", "question-1", "--text", "literal answer", "--repo", dir, "--model", ".markitect/model.yaml", "--dependency", "vendor/model.yaml", "--host-profile", host, "--run-config", config}, &out, &stderr)
	if code != 3 || calls != 1 || observed.RunConfig != config || observed.Question != "question-1" || observed.Answer != "literal answer" || observed.Selection.Repository != dir || len(observed.Selection.Dependencies) != 1 || observed.Selection.Dependencies[0].ManifestPath.String() != "vendor/model.yaml" || len(observed.Selection.HostProfiles) != 1 {
		t.Fatalf("execution arguments changed: code=%d calls=%d stderr=%s", code, calls, stderr.String())
	}
	if !strings.Contains(out.String(), "actual-open-question") || !strings.Contains(out.String(), `"operation": "answer"`) {
		t.Fatal("actual incomplete result was not reported")
	}
}

func TestExecutionRefusesPreviewFlagsAndMissingAuthorityWithoutDispatch(t *testing.T) {
	dir := t.TempDir()
	config, host := filepath.Join(dir, "run.json"), filepath.Join(dir, "host.yaml")
	calls := 0
	c := New(nil, nil, nil, nil, nil, func(context.Context, ExecutionRequest) (ExecutionReport, error) {
		calls++
		return ExecutionReport{Status: model.Passed}, nil
	})
	for _, args := range [][]string{
		{"apply"}, {"apply", "--repo", "absent"},
		{"apply", "--run-config", "relative.json", "--host-profile", host},
		{"apply", "--run-config", config, "--host-profile", host, "--revision", "HEAD"},
		{"apply", "--run-config", config, "--host-profile", host, "--metadata"},
		{"answer"}, {"answer", "q", "--run-config", config, "--host-profile", host},
	} {
		var out, stderr bytes.Buffer
		if code := c.Run(context.Background(), args, &out, &stderr); code == 0 {
			t.Fatal("incomplete or invalid invocation passed")
		}
	}
	if calls != 0 {
		t.Fatal("invalid invocation reached execution")
	}
}

func TestExecutionStatusesNeverConvertErrorOrIncompleteIntoSuccess(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		status model.Status
		err    error
		code   int
	}{
		{model.Passed, nil, 0}, {model.Failed, nil, 1}, {model.NotCompleted, nil, 3},
		{model.Passed, errors.New("actual execution refusal"), 3}, {"", nil, 3},
	} {
		c := New(nil, nil, nil, nil, nil, func(context.Context, ExecutionRequest) (ExecutionReport, error) {
			return ExecutionReport{Status: tc.status}, tc.err
		})
		var out, stderr bytes.Buffer
		if code := c.Run(context.Background(), []string{"apply", "--host-profile", filepath.Join(dir, "host.yaml"), "--run-config", filepath.Join(dir, "run.json")}, &out, &stderr); code != tc.code {
			t.Fatalf("status=%s err=%v code=%d", tc.status, tc.err, code)
		}
	}
}
