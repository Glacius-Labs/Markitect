package projectcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
)

type brownfieldManagerRunInput struct {
	IterationID      string `json:"iterationId"`
	Phase            string `json:"phase"`
	AgentManagerID   string `json:"agentManagerId"`
	RetryOfAttemptID string `json:"retryOfAttemptId,omitempty"`
}

type brownfieldManagerRunOutput struct {
	Status        string                              `json:"status"`
	Action        string                              `json:"action"`
	SessionDigest string                              `json:"sessionDigest"`
	PreviewDigest string                              `json:"previewDigest"`
	Preview       *projectadoption.ManagerRunPreview  `json:"preview,omitempty"`
	Attempt       *brownfieldManagerAttempt           `json:"attempt,omitempty"`
	Proposal      *projectadoption.ManagerProposal    `json:"proposal,omitempty"`
	Integration   *projectadoption.ManagerIntegration `json:"integration,omitempty"`
}

type brownfieldManagerAttempt struct {
	ID              string             `json:"id"`
	Status          string             `json:"status,omitempty"`
	Execution       *agentexec.Receipt `json:"execution,omitempty"`
	ExecutionDigest string             `json:"executionDigest,omitempty"`
	LedgerDigest    string             `json:"ledgerDigest,omitempty"`
}

// runBrownfieldManagerStage keeps the provider-free preview and one-shot
// execution path in the same CLI boundary. The service owns per-session lock,
// durable invocation events, budget accounting, validation, and CAS attach.
func runBrownfieldManagerStage(opts options, out io.Writer, invoker projectadoption.ManagerRunInvoker) error {
	sourceRoot := opts.sourceRepo
	if sourceRoot == "" {
		sourceRoot = opts.repo
	}
	if opts.repo == "" || sourceRoot == "" || opts.sessionID == "" || opts.input == "" {
		return errors.New("Brownfield run requires --repo, --session, and --input")
	}
	if opts.revision != "" {
		return errors.New("Brownfield run uses the session's fixed target revision; start a new session to select another revision")
	}
	if invoker == nil {
		return errors.New("Brownfield run requires a manager invoker")
	}
	if opts.write && opts.expect == "" {
		return errors.New("Brownfield run --write requires --expect with the exact manager-run preview digest")
	}
	data, err := readRecord(sourceRoot, opts.input)
	if err != nil {
		return err
	}
	var request brownfieldManagerRunInput
	if err := decodeClosedProjectJSON(data, &request); err != nil {
		return fmt.Errorf("decode Brownfield manager-run input: %w", err)
	}
	if request.IterationID == "" || request.Phase == "" || request.AgentManagerID == "" {
		return errors.New("Brownfield manager-run input requires iterationId, phase, and agentManagerId")
	}
	runtime, err := projectrun.LoadRuntime(opts.repo)
	if err != nil {
		return fmt.Errorf("load target project runtime: %w", err)
	}
	selectedAgent, err := projectrun.ReadOnlyAgent(runtime, request.AgentManagerID)
	if err != nil {
		return fmt.Errorf("select read-only Brownfield binding for explicitly selected Manager %q: %w", request.AgentManagerID, err)
	}
	config, err := selectedAgent.AgentConfig()
	if err != nil {
		return err
	}
	limits := projectadoption.ManagerRunLimits{
		MaxStarts: runtime.Limits.MaxStarts, MaxRetries: runtime.Limits.MaxRetries,
		MaxDuration: time.Duration(runtime.Limits.MaxDuration), MaxCostMicros: runtime.Limits.MaxCostMicros,
		InputPriceMicrosPerMillion:  selectedAgent.Pricing.InputMicrosPerMillion,
		OutputPriceMicrosPerMillion: selectedAgent.Pricing.OutputMicrosPerMillion,
		MaxTimeout:                  time.Duration(selectedAgent.Timeout), MaxStdoutBytes: selectedAgent.MaxStdoutBytes,
		MaxStderrBytes: selectedAgent.MaxStderrBytes,
	}
	expectedSessionDigest := ""
	if !opts.write {
		// On a preview, --expect is an optional current-session precondition.
		expectedSessionDigest = opts.expect
	}
	preview, err := projectadoption.PreviewManagerStage(sourceRoot, opts.repo, opts.sessionID, request.IterationID,
		request.Phase, request.AgentManagerID, expectedSessionDigest, request.RetryOfAttemptID, config, limits)
	if err != nil {
		return err
	}
	if !opts.write {
		return writeJSON(out, brownfieldManagerRunOutput{Status: "preview", Action: "run", SessionDigest: preview.SessionDigest,
			PreviewDigest: preview.PreviewDigest, Preview: &preview})
	}
	if opts.expect != preview.PreviewDigest {
		return errors.New("--expect does not match the exact manager-run preview digest; regenerate the preview and review the changed context, runtime, or budget")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, runErr := projectadoption.RunManagerStage(ctx, sourceRoot, opts.repo, opts.sessionID, request.IterationID,
		request.Phase, request.AgentManagerID, opts.expect, request.RetryOfAttemptID, config, limits, invoker)
	output := brownfieldManagerRunOutput{Status: result.Status, Action: "run", SessionDigest: result.SessionDigest,
		PreviewDigest: preview.PreviewDigest, Preview: &preview, Proposal: result.Proposal, Integration: result.Integration}
	if output.SessionDigest == "" {
		output.SessionDigest = preview.SessionDigest
	}
	if result.AttemptID != "" {
		output.Attempt = &brownfieldManagerAttempt{ID: result.AttemptID, Status: result.AttemptStatus,
			Execution: result.Execution, ExecutionDigest: result.ExecutionDigest, LedgerDigest: result.LedgerDigest}
	}
	if runErr != nil {
		if output.Attempt == nil {
			return runErr
		}
		if err := writeJSON(out, output); err != nil {
			return fmt.Errorf("manager attempt %s is durable but its CLI result could not be emitted; inspect the Brownfield run journal and do not replay automatically: %w", output.Attempt.ID, err)
		}
		return &projectOutcomeError{code: 1, message: fmt.Sprintf("Brownfield manager stage did not complete; attempt %s is retained with status %s and will not be replayed automatically", output.Attempt.ID, result.AttemptStatus)}
	}
	if err := writeJSON(out, output); err != nil {
		if output.Attempt != nil {
			return fmt.Errorf("Brownfield manager attempt %s is durable but its CLI result could not be emitted; inspect the run journal before continuing: %w", output.Attempt.ID, err)
		}
		return err
	}
	return nil
}
