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

// assessmentPass is the operator's frozen declaration of the assessment Pass:
// its purpose, mandate, guidance and correction purposes.
type assessmentPass struct {
	Purpose     string   `json:"purpose"`
	Mandate     string   `json:"mandate"`
	Guidance    string   `json:"guidance"`
	Corrections []string `json:"corrections"`
}

func (p assessmentPass) spec() runport.PassSpec {
	spec := runport.PassSpec{Purpose: runport.Purpose(p.Purpose), Mandate: p.Mandate, Review: true, Guidance: p.Guidance}
	for _, c := range p.Corrections {
		spec.Corrections = append(spec.Corrections, runport.Purpose(c))
	}
	return spec
}

// evaluateAssessment is Execute with one read-only Model assessment of the
// captured stand instead of Apply: the same preparation, ports, evidence and
// report, never a publication.
func (s *executionService) evaluateAssessment(assessment runport.PassSpec) func(context.Context, cli.ApplyRequest) (cli.ApplyReport, error) {
	return func(ctx context.Context, request cli.ApplyRequest) (report cli.ApplyReport, err error) {
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
		ports, err := s.ports(session.cfg, prepared, session.lock, session.runID, session.owner)
		if err != nil {
			return report, err
		}
		report.EvidenceDirectory = ports.Evidence.Dir()
		defer func() {
			if err != nil {
				report.Error = err.Error()
			}
			if writeErr := ports.Evidence.WriteReport(cli.ReportValue(report)); writeErr != nil {
				err = errors.Join(err, writeErr)
				report.Status = model.NotCompleted
			}
			err = errors.Join(err, ports.Evidence.Close())
		}()
		attempt, ok := prepared.Prepared()
		if !ok {
			return report, errors.New("complete captured settings are required")
		}
		cfg := session.cfg
		// A closed Run: no owner channel and no owner decisions reach the assessment.
		result, err := apply.AssessCaptured(ctx, apply.ApplyRequest{
			RunID: ports.RunID, Prepared: attempt, Selection: request.Selection,
			OwnerRequest: cfg.OwnerRequest, Deadline: ports.Deadline,
			MaxEntries: ports.Harvest.MaxEntries, MaxBytes: ports.Harvest.MaxBytes,
			CheckRequest: func(selected apply.CheckSelection) (runport.CheckRequest, error) {
				return executionCheck(selected, cfg)
			},
			Agents: ports.Agents, Workspaces: ports.Workspaces, Runner: ports.Checks, Evidence: ports.Evidence,
			Clock: ports.Clock, RemoveHome: ports.RemoveHome, RequireVolumeSubpaths: ports.RequireVolumeSubpaths,
		}, assessment)
		if result.Outcome.Status != "" {
			report.Run = cli.NewRunReport(ports.RunID, result.Outcome)
		}
		// Nothing is published; the operator's assessment classifies the outcome.
		switch {
		case result.Outcome.Status == runport.Failed:
			report.Status = model.Failed
		case result.Outcome.Status == runport.Verified && err == nil:
			report.Status = model.Passed
		}
		return report, err
	}
}

// TestModelAssessmentHarness runs one read-only Model assessment of a captured
// stand through the product's real preparation and adapters, for the finite
// model assessment trial (handoff of 21 October 2026, Session C). It is no
// Command and never publishes. It runs only with an explicit grant in the
// environment: MARKITECT_MODEL_ASSESSMENT=run-model-assessment,
// MARKITECT_MODEL_ASSESSMENT_PASS, the path of the frozen assessment Pass
// declaration, and MARKITECT_MODEL_ASSESSMENT_ARGS, a JSON list of the apply
// command's arguments ("apply", "--repo", ...). The report is printed like
// apply's.
func TestModelAssessmentHarness(t *testing.T) {
	if os.Getenv("MARKITECT_MODEL_ASSESSMENT") != "run-model-assessment" {
		t.Skip("set MARKITECT_MODEL_ASSESSMENT=run-model-assessment with an assessment grant")
	}
	data, err := os.ReadFile(os.Getenv("MARKITECT_MODEL_ASSESSMENT_PASS"))
	if err != nil {
		t.Fatalf("MARKITECT_MODEL_ASSESSMENT_PASS must name the assessment Pass declaration: %v", err)
	}
	var pass assessmentPass
	if err := json.Unmarshal(data, &pass); err != nil {
		t.Fatalf("the assessment Pass declaration: %v", err)
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
	execution := newExecution(preparation, repository, agents, checker, os.Stderr)
	code := cli.New(preparation, agents, checker, repository, views, execution.evaluateAssessment(pass.spec()), answer).Run(context.Background(), args, os.Stdout, os.Stderr)
	fmt.Fprintf(os.Stderr, "model assessment exit %d\n", code)
}
