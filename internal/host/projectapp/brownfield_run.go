package projectapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
)

type BrownfieldRunOperation struct {
	Root           string                    `json:"root"`
	SourceRoot     string                    `json:"sourceRoot,omitempty"`
	Revision       string                    `json:"revision,omitempty"`
	SessionID      string                    `json:"sessionId"`
	Write          bool                      `json:"write"`
	ExpectedDigest string                    `json:"expectedDigest,omitempty"`
	Request        BrownfieldManagerRunInput `json:"request"`
}
type BrownfieldManagerRunInput struct {
	IterationID      string `json:"iterationId"`
	Phase            string `json:"phase"`
	AgentManagerID   string `json:"agentManagerId"`
	RetryOfAttemptID string `json:"retryOfAttemptId,omitempty"`
}

type BrownfieldManagerRunOutput struct {
	Status        string                              `json:"status"`
	Action        string                              `json:"action"`
	SessionDigest string                              `json:"sessionDigest"`
	PreviewDigest string                              `json:"previewDigest"`
	Preview       *projectadoption.ManagerRunPreview  `json:"preview,omitempty"`
	Attempt       *BrownfieldManagerAttempt           `json:"attempt,omitempty"`
	Proposal      *projectadoption.ManagerProposal    `json:"proposal,omitempty"`
	Integration   *projectadoption.ManagerIntegration `json:"integration,omitempty"`
}

type BrownfieldManagerAttempt struct {
	ID              string             `json:"id"`
	Status          string             `json:"status,omitempty"`
	Execution       *agentexec.Receipt `json:"execution,omitempty"`
	ExecutionDigest string             `json:"executionDigest,omitempty"`
	LedgerDigest    string             `json:"ledgerDigest,omitempty"`
}

// BrownfieldRun shares the provider-free preview and one-shot execution
// path across transports. The service owns per-session lock,
// durable invocation events, budget accounting, validation, and CAS attach.
func (o Operations) BrownfieldRun(ctx context.Context, operation BrownfieldRunOperation, invoker projectadoption.ManagerRunInvoker) (BrownfieldManagerRunOutput, error) {
	if err := requireRoot(operation.Root); err != nil {
		return BrownfieldManagerRunOutput{}, err
	}
	if err := ctx.Err(); err != nil {
		return BrownfieldManagerRunOutput{}, err
	}
	request := operation.Request
	sourceRoot := operation.SourceRoot
	if sourceRoot == "" {
		sourceRoot = operation.Root
	}
	if operation.Root == "" || sourceRoot == "" || operation.SessionID == "" {
		return BrownfieldManagerRunOutput{}, errors.New("Brownfield run requires --repo, --session, and --input")
	}
	if operation.Revision != "" {
		return BrownfieldManagerRunOutput{}, errors.New("Brownfield run uses the session's fixed target revision; start a new session to select another revision")
	}
	if invoker == nil {
		return BrownfieldManagerRunOutput{}, errors.New("Brownfield run requires a manager invoker")
	}
	if operation.Write && operation.ExpectedDigest == "" {
		return BrownfieldManagerRunOutput{}, errors.New("Brownfield run --write requires --expect with the exact manager-run preview digest")
	}
	if request.IterationID == "" || request.Phase == "" || request.AgentManagerID == "" {
		return BrownfieldManagerRunOutput{}, errors.New("Brownfield manager-run input requires iterationId, phase, and agentManagerId")
	}
	runtime, err := projectrun.LoadRuntime(operation.Root)
	if err != nil {
		return BrownfieldManagerRunOutput{}, fmt.Errorf("load target project runtime: %w", err)
	}
	selectedAgent, err := projectrun.ReadOnlyAgent(runtime, request.AgentManagerID)
	if err != nil {
		return BrownfieldManagerRunOutput{}, fmt.Errorf("select read-only Brownfield binding for explicitly selected Manager %q: %w", request.AgentManagerID, err)
	}
	config, err := selectedAgent.AgentConfig()
	if err != nil {
		return BrownfieldManagerRunOutput{}, err
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
	if !operation.Write {
		// On a preview, --expect is an optional current-session precondition.
		expectedSessionDigest = operation.ExpectedDigest
	}
	preview, err := projectadoption.PreviewManagerStage(sourceRoot, operation.Root, operation.SessionID, request.IterationID,
		request.Phase, request.AgentManagerID, expectedSessionDigest, request.RetryOfAttemptID, config, limits)
	if err != nil {
		return BrownfieldManagerRunOutput{}, err
	}
	if !operation.Write {
		return BrownfieldManagerRunOutput{Status: "preview", Action: "run", SessionDigest: preview.SessionDigest, PreviewDigest: preview.PreviewDigest, Preview: &preview}, nil
	}
	if operation.ExpectedDigest != preview.PreviewDigest {
		return BrownfieldManagerRunOutput{}, errors.New("--expect does not match the exact manager-run preview digest; regenerate the preview and review the changed context, runtime, or budget")
	}
	result, runErr := projectadoption.RunManagerStage(ctx, sourceRoot, operation.Root, operation.SessionID, request.IterationID,
		request.Phase, request.AgentManagerID, operation.ExpectedDigest, request.RetryOfAttemptID, config, limits, invoker)
	output := BrownfieldManagerRunOutput{Status: result.Status, Action: "run", SessionDigest: result.SessionDigest,
		PreviewDigest: preview.PreviewDigest, Preview: &preview, Proposal: result.Proposal, Integration: result.Integration}
	if output.SessionDigest == "" {
		output.SessionDigest = preview.SessionDigest
	}
	if result.AttemptID != "" {
		output.Attempt = &BrownfieldManagerAttempt{ID: result.AttemptID, Status: result.AttemptStatus,
			Execution: result.Execution, ExecutionDigest: result.ExecutionDigest, LedgerDigest: result.LedgerDigest}
	}
	return output, runErr
}
