package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/documents"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/modeldocs"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

// evaluateAssessment is Execute with one read-only Model assessment of the
// captured stand instead of Apply: the same preparation, ports, evidence and
// report, never a publication.
func (s *executionService) evaluateAssessment(ctx context.Context, request cli.ApplyRequest) (report cli.ApplyReport, err error) {
	report.FormatVersion, report.Operation, report.Status = 1, "model-assessment", model.NotCompleted
	session, failed, err := s.open(ctx, request.Selection.Repository, request.RunConfig)
	if err != nil {
		if failed {
			report.Status = model.Failed
		}
		return report, err
	}
	defer func() { err = errors.Join(err, session.lock.Release()) }()
	prepared, err := s.preparation.CaptureSettings(ctx, apply.CaptureRequest{Selection: request.Selection}, model.Limits{}, s.agents, s.checks)
	report.Preparation = capturedExecutionReport(prepared, session.cfg.ProductBuild)
	if prepared.Status() != model.Passed || err != nil {
		report.Status = prepared.Status()
		return report, err
	}
	ports, err := s.ports(session.cfg, prepared, session.lock, session.runID)
	if err != nil {
		return report, err
	}
	report.EvidenceDirectory = ports.Evidence.Dir()
	defer func() {
		err = finishAssessmentReport(&report, err, ports.Evidence.Close, session.lock.Release, ports.Evidence.WriteReport)
	}()
	attempt, ok := prepared.Prepared()
	if !ok {
		return report, errors.New("complete captured settings are required")
	}
	// The same request as the product's Apply in a closed Run: no owner channel
	// and no owner decisions reach the assessment, which never publishes.
	req := applyRequest(request, attempt, session.cfg, ports)
	req.Publisher, req.Control = nil, nil
	result, err := apply.AssessCaptured(ctx, req)
	if result.Binding.Entry != "" {
		report.Binding = &result.Binding
	}
	if result.Outcome.Status != "" {
		report.Run = cli.NewRunReport(ports.RunID, result.Outcome)
	}
	// Nothing is published; the operator's assessment classifies the outcome.
	switch {
	case result.Outcome.Status == runport.Failed:
		report.Status = model.Failed
	case result.Outcome.Status == runport.Incomplete && result.Outcome.Reason == apply.AssessmentAccepted && err == nil:
		report.Status = model.Passed
	}
	return report, err
}

// Finalize after the event log and lock have closed, so deferred errors cannot
// leave a successful on-disk assessment report. WriteReport is independent of
// the closed event stream; a write failure also reaches the final CLI output.
func finishAssessmentReport(report *cli.ApplyReport, err error, closeEvidence, releaseLock func() error, writeReport func(any) error) error {
	err = errors.Join(err, closeEvidence(), releaseLock())
	if err != nil {
		report.Error, report.Status = err.Error(), model.NotCompleted
	}
	if writeErr := writeReport(cli.ReportValue(*report)); writeErr != nil {
		err = errors.Join(err, writeErr)
		report.Error, report.Status = err.Error(), model.NotCompleted
	}
	return err
}

func TestModelAssessmentReportIncludesDeferredFailures(t *testing.T) {
	for _, failed := range []string{"", "close", "release", "write"} {
		t.Run("failure-"+failed, func(t *testing.T) {
			report := cli.ApplyReport{Status: model.Passed, Run: cli.NewRunReport("assessment-test", runport.Outcome{Status: runport.Incomplete, Reason: apply.AssessmentAccepted})}
			var order []string
			step := func(name string) error {
				order = append(order, name)
				if failed == name {
					return fmt.Errorf("%s failed", name)
				}
				return nil
			}
			var persisted map[string]any
			err := finishAssessmentReport(&report, nil, func() error { return step("close") }, func() error { return step("release") }, func(value any) error {
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &persisted); err != nil {
					t.Fatal(err)
				}
				return step("write")
			})
			if fmt.Sprint(order) != "[close release write]" {
				t.Fatalf("finalization order %v", order)
			}
			if failed == "" {
				if err != nil || report.Status != model.Passed || persisted["error"] != nil {
					t.Fatalf("clean report %+v, %v", report, err)
				}
			} else if err == nil || report.Status != model.NotCompleted || report.Error == "" || failed != "write" && (persisted["status"] != "not_completed" || persisted["error"] == nil) {
				t.Fatalf("failed report %+v, persisted %+v, %v", report, persisted, err)
			}
		})
	}
}

// TestModelAssessmentHarness runs one read-only Model assessment of a captured
// stand through the product's real preparation and adapters, for the finite
// model assessment trial (handoff of 21 October 2026, Session C). It is no
// Command and never publishes. It runs only with an explicit grant in the
// environment: MARKITECT_MODEL_ASSESSMENT=run-model-assessment,
// and MARKITECT_MODEL_ASSESSMENT_ARGS, a JSON list of the apply
// command's arguments ("apply", "--repo", ...). The report is printed like
// apply's.
func TestModelAssessmentHarness(t *testing.T) {
	if os.Getenv("MARKITECT_MODEL_ASSESSMENT") != "run-model-assessment" {
		t.Skip("set MARKITECT_MODEL_ASSESSMENT=run-model-assessment with an assessment grant")
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("MARKITECT_MODEL_ASSESSMENT_ARGS")), &args); err != nil || len(args) == 0 || args[0] != "apply" {
		t.Fatalf("MARKITECT_MODEL_ASSESSMENT_ARGS must be a JSON list of apply arguments: %v", err)
	}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	models := apply.NewModelService(repository, repository)
	preparation := apply.NewPreparationService(models)
	views := modeldocs.New(models, preparation, documents.Writer{})
	agents := agentMetadataRouter{codex: codexappserver.NewMetadataPreparer(codexappserver.NewEngineImageReader), claude: claudecode.NewMetadataPreparer(claudecode.NewEngineImageReader)}
	checker := checks.NewMetadataPreparer(checks.NewEngineImageReaders().Reader)
	execution := newExecution(preparation, repository, agents, checker)
	code := cli.New(preparation, agents, checker, repository, views, execution.evaluateAssessment).Run(context.Background(), args, os.Stdout, os.Stderr)
	fmt.Fprintf(os.Stderr, "model assessment exit %d\n", code)
}
