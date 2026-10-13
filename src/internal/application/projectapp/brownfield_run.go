package projectapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
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
func (o Operations) BrownfieldRun(ctx context.Context, operation BrownfieldRunOperation, invoker projectrun.Invoker) (BrownfieldManagerRunOutput, error) {
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
		return BrownfieldManagerRunOutput{}, errors.New("adopt run requires --repo, --session, and --input")
	}
	if operation.Revision != "" {
		return BrownfieldManagerRunOutput{}, errors.New("adopt run uses the session's fixed target revision; start a new session to select another revision")
	}
	if invoker == nil {
		return BrownfieldManagerRunOutput{}, errors.New("adopt run requires a manager invoker")
	}
	if operation.Write && operation.ExpectedDigest == "" {
		return BrownfieldManagerRunOutput{}, errors.New("adopt run --write requires --expect with the exact manager-run preview digest")
	}
	if request.IterationID == "" || request.Phase == "" || request.AgentManagerID == "" {
		return BrownfieldManagerRunOutput{}, errors.New("adopt manager-run input requires iterationId, phase, and agentManagerId")
	}
	runtime, err := projectconfig.LoadRuntime(operation.Root)
	if err != nil {
		return BrownfieldManagerRunOutput{}, fmt.Errorf("load target project runtime: %w", err)
	}
	selectedAgent, err := projectconfig.ReadOnlyAgent(runtime, request.AgentManagerID)
	if err != nil {
		return BrownfieldManagerRunOutput{}, fmt.Errorf("select read-only adoption binding for explicitly selected Manager %q: %w", request.AgentManagerID, err)
	}
	if selectedAgent.Unmetered() {
		// The Brownfield ledger seals a priced estimate for every attempt. It has
		// no unmetered accounting yet, so it rejects rather than invent a price.
		return BrownfieldManagerRunOutput{}, fmt.Errorf("adopt manager stages require a metered agent; the read-only binding for Manager %q is unmetered", request.AgentManagerID)
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
	if operation.Write {
		unlock, err := projectadoption.AcquireManagerStageLease(sourceRoot, operation.SessionID)
		if err != nil {
			return BrownfieldManagerRunOutput{}, err
		}
		defer unlock()
	}
	prepared, spec, accepted, runtimePreview, err := o.prepareBrownfieldRuntimeStage(sourceRoot, operation, runtime, selectedAgent, config, limits, expectedSessionDigest, invoker)
	if err != nil {
		return BrownfieldManagerRunOutput{}, err
	}
	preview := prepared.Preview
	mapBrownfieldRuntimePreview(&preview, runtimePreview)
	output := BrownfieldManagerRunOutput{Status: "preview", Action: "run", SessionDigest: prepared.Session.Digest, PreviewDigest: preview.PreviewDigest, Preview: &preview}
	if !operation.Write {
		return output, nil
	}
	if operation.ExpectedDigest != preview.PreviewDigest {
		return BrownfieldManagerRunOutput{}, errors.New("--expect does not match the exact manager-run preview digest; regenerate the preview")
	}
	if preview.Completed {
		output.Status = "completed"
		for _, iteration := range prepared.Session.Iterations {
			if iteration.ID == request.IterationID {
				if request.Phase == projectadoption.ManagerRunPhasePropose {
					output.Proposal = iteration.Proposal
				} else {
					output.Integration = iteration.Integration
				}
			}
		}
		if runtimePreview.LatestAttempt != nil {
			output.Attempt = &BrownfieldManagerAttempt{ID: runtimePreview.LatestAttempt.ID, Status: runtimePreview.LatestAttempt.Status, LedgerDigest: runtimePreview.StateDigest}
		}
		return output, nil
	}
	host := o.brownfieldStageHost(sourceRoot, operation, invoker)
	result, err := projectrun.RunManagerStage(ctx, host, invoker, spec.SelectedSourceRoot, spec, runtimePreview.PreviewDigest)
	projectBrownfieldAttempt(&output, result)
	if err != nil {
		output.Status = "failed"
		return output, err
	}
	proposal, integration, err := projectadoption.AcceptManagerStageDraft(prepared.Session, request.IterationID, request.Phase, preview.ManagerID, result.Result.Response.ReportJSON, config, result.Result.Receipt)
	if err != nil {
		completion, completionErr := projectrun.CompleteManagerStage(spec.SelectedSourceRoot, result.RunID, result.AttemptID, result.StateDigest, projectrun.ManagerStageValidation{SafeFailure: "invalid-manager-report"})
		if completionErr == nil {
			projectBrownfieldAttempt(&output, completion)
		}
		output.Status = "failed"
		return output, errors.Join(err, completionErr)
	}
	if runtime.Review != nil {
		reviewSpec, err := brownfieldRuntimeReview(prepared, spec, accepted, runtime)
		if err != nil {
			return output, err
		}
		review, err := projectrun.ReviewManagerStage(ctx, host, invoker, spec.SelectedSourceRoot, result.RunID, result.AttemptID, reviewSpec, result.StateDigest)
		if err != nil {
			output.Status = "review-failed"
			return output, err
		}
		result.StateDigest = review.StateDigest
		if review.Status != "pass" {
			completion, err := projectrun.CompleteManagerStage(spec.SelectedSourceRoot, result.RunID, result.AttemptID, result.StateDigest, projectrun.ManagerStageValidation{SafeFailure: "configured-review-failed"})
			if err == nil {
				projectBrownfieldAttempt(&output, completion)
			}
			output.Status = "review-failed"
			return output, errors.Join(errors.New("configured Manager review did not pass"), err)
		}
	}
	typedDigest := brownfieldStageValueDigest(struct {
		Proposal    *projectadoption.ManagerProposal
		Integration *projectadoption.ManagerIntegration
	}{proposal, integration})
	completed, err := projectrun.CompleteManagerStage(spec.SelectedSourceRoot, result.RunID, result.AttemptID, result.StateDigest, projectrun.ManagerStageValidation{Accepted: true, TypedResultDigest: typedDigest})
	projectBrownfieldAttempt(&output, completed)
	if err != nil {
		output.Status = "stage-not-recorded"
		return output, err
	}
	if err := host.ValidateManagerStage(spec.Validation); err != nil {
		output.Status = "stale-after-invocation"
		return output, err
	}
	var next projectadoption.BrownfieldSession
	if proposal != nil {
		next, err = projectadoption.RecordManagerProposal(prepared.Session, request.IterationID, *proposal)
	} else {
		next, err = projectadoption.IntegrateManagerProposal(prepared.Session, request.IterationID, preview.ManagerID, *integration)
	}
	if err != nil {
		output.Status = "stage-not-recorded"
		return output, err
	}
	if _, err := projectadoption.WriteBrownfieldSession(spec.SelectedSourceRoot, next, prepared.Session.Digest); err != nil {
		output.Status = "stage-not-recorded"
		return output, err
	}
	output.Status, output.SessionDigest, output.Proposal, output.Integration = "recorded", next.Digest, proposal, integration
	if runtimePreview.LatestAttempt != nil && runtimePreview.LatestAttempt.ID == result.AttemptID {
		output.Status = "recovered"
	}
	return output, nil
}

// RunBrownfield uses the invoker selected by the application composition root.
func (o Operations) RunBrownfield(ctx context.Context, operation BrownfieldRunOperation) (BrownfieldManagerRunOutput, error) {
	return o.BrownfieldRun(ctx, operation, o.Invoker)
}

func brownfieldStageValueDigest(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("Application stage facts cannot be encoded")
	}
	return brownfieldStageRawDigest(encoded)
}
func brownfieldStageRawDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func brownfieldStageCanonicalRoot(root string) (string, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	return filepath.Abs(resolved)
}

func (o Operations) prepareBrownfieldRuntimeStage(sourceRoot string, operation BrownfieldRunOperation, runtime projectconfig.Runtime, agent projectconfig.Agent, config agentexec.Config, limits projectadoption.ManagerRunLimits, expectedSession string, invoker projectrun.Invoker) (projectadoption.PreparedManagerStage, projectrun.ManagerStageSpec, projectmodel.ManagerContext, projectrun.ManagerStagePreview, error) {
	var emptySpec projectrun.ManagerStageSpec
	var emptyModel projectmodel.ManagerContext
	var emptyPreview projectrun.ManagerStagePreview
	prepared, err := projectadoption.PrepareManagerStage(sourceRoot, operation.Root, operation.SessionID, operation.Request.IterationID, operation.Request.Phase, operation.Request.AgentManagerID, expectedSession, operation.Request.RetryOfAttemptID, config, limits, invoker)
	if err != nil {
		return prepared, emptySpec, emptyModel, emptyPreview, err
	}
	sourceRoot, err = brownfieldStageCanonicalRoot(sourceRoot)
	if err != nil {
		return prepared, emptySpec, emptyModel, emptyPreview, err
	}
	targetRoot, err := brownfieldStageCanonicalRoot(operation.Root)
	if err != nil {
		return prepared, emptySpec, emptyModel, emptyPreview, err
	}
	if o.Host.Load == nil {
		return prepared, emptySpec, emptyModel, emptyPreview, errors.New("Brownfield stages require the accepted target project Host")
	}
	target, err := o.Host.Load(targetRoot, prepared.Session.Target.Revision)
	if err != nil {
		return prepared, emptySpec, emptyModel, emptyPreview, err
	}
	accepted, err := projectmodel.Context(target.Report, operation.Request.AgentManagerID)
	if err != nil {
		return prepared, emptySpec, emptyModel, emptyPreview, err
	}
	fingerprint := prepared.Preview.ConfigFingerprint
	if aware, ok := invoker.(interface {
		FingerprintForRequest(agentexec.Config, agentexec.Request) (string, error)
	}); ok {
		fingerprint, err = aware.FingerprintForRequest(config, prepared.Request)
		if err != nil {
			return prepared, emptySpec, emptyModel, emptyPreview, err
		}
	}
	budget := projectrun.ManagerStageBudget{MaxStarts: limits.MaxStarts, MaxRetries: limits.MaxRetries, MaxDuration: limits.MaxDuration, MaxCostMicros: limits.MaxCostMicros}
	validation := projectrun.ManagerStageValidationBinding{SessionID: prepared.Session.ID, SessionDigest: prepared.Session.Digest, IterationID: operation.Request.IterationID, ManagerID: prepared.Preview.ManagerID, AgentManagerID: operation.Request.AgentManagerID, Phase: operation.Request.Phase, AcceptedTargetHead: prepared.Session.Target.Revision, ContextDigest: prepared.Preview.ContextDigest, RequestContractDigest: prepared.Preview.RequestContractDigest, ConfigFingerprint: fingerprint, RuntimeFileDigest: prepared.Preview.RuntimeFileDigest, AcceptedModelDigest: brownfieldStageValueDigest(accepted), SourceRoot: sourceRoot, TargetRoot: targetRoot}
	binding := projectrun.ManagerStageBinding{SessionID: prepared.Session.ID, SourceIdentity: prepared.Session.Source.Identity.Digest, TargetIdentity: brownfieldStageRawDigest([]byte(targetRoot)), SourceRevision: prepared.Session.Source.Commit, DiscoveryDigest: prepared.Session.Source.Digest, InitialTargetRevision: prepared.Session.Target.Revision, TargetContextDigest: prepared.Session.TargetContext.Digest, Budget: budget, SourceRootDigest: brownfieldStageRawDigest([]byte(sourceRoot)), TargetRootDigest: brownfieldStageRawDigest([]byte(targetRoot))}
	spec := projectrun.ManagerStageSpec{SelectedSourceRoot: sourceRoot, TargetRoot: targetRoot, Binding: binding, Validation: validation, RetryOfAttemptID: operation.Request.RetryOfAttemptID, Agent: agent, Config: config, Request: prepared.Request, Limits: budget, RuntimeLimits: runtime.Limits, Review: runtime.Review}
	preview, err := projectrun.PreviewManagerStage(invoker, sourceRoot, spec)
	if err != nil {
		return prepared, spec, accepted, preview, err
	}
	if err := projectadoption.BindPreparedManagerStageBudget(&prepared, projectadoption.ManagerRunContextBudget{RemainingStarts: preview.RemainingStarts, RemainingRetries: preview.RemainingRetries, RemainingDuration: preview.RemainingDuration, RemainingCostMicros: preview.RemainingCostMicros, RequestedTimeout: config.Timeout}); err != nil {
		return prepared, spec, accepted, preview, err
	}
	spec.Request = prepared.Request
	prepared.Preview.ConfigFingerprint = fingerprint
	preview, err = projectrun.PreviewManagerStage(invoker, sourceRoot, spec)
	return prepared, spec, accepted, preview, err
}

func mapBrownfieldRuntimePreview(preview *projectadoption.ManagerRunPreview, runtime projectrun.ManagerStagePreview) {
	preview.PreviewDigest, preview.LedgerDigest, preview.LimitsDigest, preview.RequestDigest = runtime.PreviewDigest, runtime.StateDigest, runtime.LimitsDigest, runtime.RequestDigest
	preview.RemainingStarts, preview.RemainingRetries, preview.RemainingDuration, preview.RemainingCostMicros = runtime.RemainingStarts, runtime.RemainingRetries, runtime.RemainingDuration, runtime.RemainingCostMicros
	if runtime.Review.Enabled {
		review := runtime.Review
		preview.Review = &review
	}
}
func projectBrownfieldAttempt(output *BrownfieldManagerRunOutput, result projectrun.ManagerStageResult) {
	if result.AttemptID == "" {
		return
	}
	output.Attempt = &BrownfieldManagerAttempt{ID: result.AttemptID, Status: result.Status, LedgerDigest: result.StateDigest}
	if result.Result.Receipt.RunID != "" {
		receipt := result.Result.Receipt
		output.Attempt.Execution = &receipt
		output.Attempt.ExecutionDigest = brownfieldStageValueDigest(receipt)
	}
}

func brownfieldRuntimeReview(prepared projectadoption.PreparedManagerStage, spec projectrun.ManagerStageSpec, accepted projectmodel.ManagerContext, runtime projectconfig.Runtime) (projectrun.ManagerStageReviewSpec, error) {
	review := *runtime.Review
	agent, ok := review.Agents[spec.Validation.AgentManagerID]
	if !ok {
		return projectrun.ManagerStageReviewSpec{}, errors.New("configured review requires the explicit accepted Manager binding")
	}
	config, err := agent.AgentConfig()
	if err != nil {
		return projectrun.ManagerStageReviewSpec{}, err
	}
	strictness, err := projectconfig.ResolveStrictness(runtime, spec.Validation.AgentManagerID)
	if err != nil {
		return projectrun.ManagerStageReviewSpec{}, err
	}
	grounding := map[string][]string{}
	for _, evidence := range prepared.Context.Evidence {
		grounding["evidence/"+evidence.EvidenceID+".txt"] = []string{"evidence:" + evidence.EvidenceID}
	}
	draft, _ := json.Marshal(struct{ Classification, ManagerID, Phase, RequestContractDigest string }{"manager-proposed-draft", spec.Validation.ManagerID, spec.Validation.Phase, spec.Validation.RequestContractDigest})
	var children json.RawMessage
	if prepared.Integration != nil {
		children, _ = json.Marshal(prepared.Integration.Children)
	}
	return projectrun.ManagerStageReviewSpec{Validation: spec.Validation, Agent: agent, Config: config, AcceptedModel: accepted, AcceptedModelDigest: spec.Validation.AcceptedModelDigest, Purpose: prepared.Purpose + "\nDiscovery purpose: " + prepared.Session.Source.Purpose + "\nAccepted Manager purpose: " + prepared.Context.Manager.Purpose, ProposedContext: draft, Evidence: prepared.Request.Artifacts, Grounding: grounding, ChildBindings: children, SourceRoot: spec.SelectedSourceRoot, TargetRoot: spec.TargetRoot, Limits: spec.Limits, RuntimeLimits: spec.RuntimeLimits, Review: review, Strictness: strictness}, nil
}
