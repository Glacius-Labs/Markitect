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

// evaluateVerify is Execute with an independent Verify of the captured stand
// instead of Apply: the same preparation, ports, evidence and report, never a
// publication.
func (s *executionService) evaluateVerify(ctx context.Context, request cli.ApplyRequest) (report cli.ApplyReport, err error) {
	report.FormatVersion, report.Operation, report.Status = 1, "verify-evaluation", model.NotCompleted
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
	result, err := apply.VerifyCaptured(ctx, apply.ApplyRequest{
		RunID: ports.RunID, Prepared: attempt, Selection: request.Selection,
		OwnerRequest: cfg.OwnerRequest, OwnerDecisions: cfg.OwnerDecisions, Deadline: ports.Deadline,
		MaxEntries: ports.Harvest.MaxEntries, MaxBytes: ports.Harvest.MaxBytes,
		CheckRequest: func(selected apply.CheckSelection) (runport.CheckRequest, error) {
			return executionCheck(selected, cfg)
		},
		Agents: ports.Agents, Workspaces: ports.Workspaces, Runner: ports.Checks, Owner: ports.Owner, Evidence: ports.Evidence,
		Clock: ports.Clock, RemoveHome: ports.RemoveHome, RequireVolumeSubpaths: ports.RequireVolumeSubpaths,
	})
	if result.Outcome.Status != "" {
		report.Run = cli.NewRunReport(ports.RunID, result.Outcome)
	}
	report.Status = cli.ApplyStatus(result.Outcome, &apply.PublicationReceipt{Outcome: apply.PublicationNoChange}, err)
	return report, err
}

// TestVerifyEvaluationHarness runs one independent Verify of a captured stand
// through the product's real preparation and adapters, for the finite
// evaluation of the Apply hardening (handoff of 20 October 2026). It is no
// Command and never publishes. It runs only with an explicit grant in the
// environment: MARKITECT_VERIFY_EVALUATION=run-verify-evaluation and
// MARKITECT_VERIFY_EVALUATION_ARGS, a JSON list of the apply command's
// arguments ("apply", "--repo", ...). The report is printed like apply's.
func TestVerifyEvaluationHarness(t *testing.T) {
	if os.Getenv("MARKITECT_VERIFY_EVALUATION") != "run-verify-evaluation" {
		t.Skip("set MARKITECT_VERIFY_EVALUATION=run-verify-evaluation with an evaluation grant")
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("MARKITECT_VERIFY_EVALUATION_ARGS")), &args); err != nil || len(args) == 0 || args[0] != "apply" {
		t.Fatalf("MARKITECT_VERIFY_EVALUATION_ARGS must be a JSON list of apply arguments: %v", err)
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
	code := cli.New(preparation, agents, checker, repository, views, execution.evaluateVerify, answer).Run(context.Background(), args, os.Stdout, os.Stderr)
	fmt.Fprintf(os.Stderr, "verify evaluation exit %d\n", code)
}
