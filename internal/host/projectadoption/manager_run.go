package projectadoption

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

const (
	ManagerRunPhasePropose   = "propose"
	ManagerRunPhaseIntegrate = "integrate"
	managerRunVersion        = "markitect.example.org/brownfield-manager-run/v1alpha1"
	managerRunLedgerName     = "manager-runs.json"
	managerRunLockName       = "manager-runs.lock"
	managerRunRuntimePath    = ".markitect/runtime.yaml"
	managerRunMaxEvents      = 4096
	managerRunMaxLedger      = 32 << 20
	managerRunMaxTimeout     = time.Hour
	managerRunMaxDuration    = 24 * time.Hour
	managerRunMaxCost        = int64(1_000_000_000_000)
)

// ManagerRunLimits is copied from the target project's current Runtime. The
// Runtime file itself is hashed by the service at preview and invocation
// boundaries; this struct binds its session-wide numeric ceilings.
type ManagerRunLimits struct {
	MaxStarts                   int           `json:"maxStarts"`
	MaxRetries                  int           `json:"maxRetries"`
	MaxDuration                 time.Duration `json:"maxDuration"`
	MaxCostMicros               int64         `json:"maxCostMicros"`
	InputPriceMicrosPerMillion  int64         `json:"inputPriceMicrosPerMillion"`
	OutputPriceMicrosPerMillion int64         `json:"outputPriceMicrosPerMillion"`
	MaxTimeout                  time.Duration `json:"maxTimeout"`
	MaxStdoutBytes              int           `json:"maxStdoutBytes"`
	MaxStderrBytes              int           `json:"maxStderrBytes"`
	TempParent                  string        `json:"-"`
	PrivateLogDirectory         string        `json:"-"`
}

// ManagerRunInvoker is the only execution seam. The Host builds a bounded,
// typed report request and passes it to this caller-supplied executor once.
type ManagerRunInvoker interface {
	Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error)
	Fingerprint(agentexec.Config) (string, error)
}

type AgentExecManagerRunInvoker struct{}

func (AgentExecManagerRunInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	return agentexec.Run(ctx, config, request, options)
}
func (AgentExecManagerRunInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return agentexec.Fingerprint(config)
}

type ManagerRunPreview struct {
	SessionDigest               string        `json:"sessionDigest"`
	IterationID                 string        `json:"iterationId"`
	Phase                       string        `json:"phase"`
	ManagerID                   string        `json:"managerId"`
	AgentManagerID              string        `json:"agentManagerId"`
	ManagerOrigin               string        `json:"managerOrigin"`
	ProviderVersion             string        `json:"providerVersion"`
	Model                       string        `json:"model"`
	ConfigFingerprint           string        `json:"configFingerprint"`
	RuntimeFileDigest           string        `json:"runtimeFileDigest"`
	LimitsDigest                string        `json:"limitsDigest"`
	LedgerDigest                string        `json:"ledgerDigest"`
	ContextDigest               string        `json:"contextDigest"`
	RequestDigest               string        `json:"requestDigest"`
	RequestContractDigest       string        `json:"requestContractDigest"`
	PreviewDigest               string        `json:"previewDigest"`
	RetryOfAttemptID            string        `json:"retryOfAttemptId,omitempty"`
	EvidenceCount               int           `json:"evidenceCount"`
	ChildReportCount            int           `json:"childReportCount"`
	RemainingStarts             int           `json:"remainingStarts"`
	RemainingRetries            int           `json:"remainingRetries"`
	RemainingDuration           time.Duration `json:"remainingDuration"`
	RemainingCostMicros         int64         `json:"remainingCostMicros"`
	Timeout                     time.Duration `json:"timeout"`
	MaxStdoutBytes              int           `json:"maxStdoutBytes"`
	MaxStderrBytes              int           `json:"maxStderrBytes"`
	InputPriceMicrosPerMillion  int64         `json:"inputPriceMicrosPerMillion"`
	OutputPriceMicrosPerMillion int64         `json:"outputPriceMicrosPerMillion"`
	Completed                   bool          `json:"completed"`
}

// ManagerRunEvent is an immutable journal entry. A start event is flushed
// before invoking the child process; terminal events retain receipts and only
// safe rejection categories, never transcripts or private log bodies.
type ManagerRunEvent struct {
	Sequence                    int                 `json:"sequence"`
	AttemptID                   string              `json:"attemptId"`
	RetryOfAttemptID            string              `json:"retryOfAttemptId,omitempty"`
	Event                       string              `json:"event"` // started or terminal
	SessionDigest               string              `json:"sessionDigest"`
	IterationID                 string              `json:"iterationId"`
	Phase                       string              `json:"phase"`
	ManagerID                   string              `json:"managerId"`
	AgentManagerID              string              `json:"agentManagerId"`
	PreviewDigest               string              `json:"previewDigest"`
	ContextDigest               string              `json:"contextDigest"`
	RequestDigest               string              `json:"requestDigest"`
	RequestContractDigest       string              `json:"requestContractDigest"`
	ConfigFingerprint           string              `json:"configFingerprint"`
	RuntimeFileDigest           string              `json:"runtimeFileDigest"`
	LimitsDigest                string              `json:"limitsDigest"`
	Timeout                     time.Duration       `json:"timeout"`
	MaxStdoutBytes              int                 `json:"maxStdoutBytes"`
	MaxStderrBytes              int                 `json:"maxStderrBytes"`
	InputPriceMicrosPerMillion  int64               `json:"inputPriceMicrosPerMillion"`
	OutputPriceMicrosPerMillion int64               `json:"outputPriceMicrosPerMillion"`
	StartedAt                   time.Time           `json:"startedAt"`
	FinishedAt                  time.Time           `json:"finishedAt,omitempty"`
	Status                      string              `json:"status,omitempty"`
	Execution                   *agentexec.Receipt  `json:"execution,omitempty"`
	ExecutionDigest             string              `json:"executionDigest,omitempty"`
	EstimatedCostMicros         int64               `json:"estimatedCostMicros,omitempty"`
	CostKnown                   bool                `json:"costKnown"`
	SafeFailure                 string              `json:"safeFailure,omitempty"`
	Proposal                    *ManagerProposal    `json:"proposal,omitempty"`
	Integration                 *ManagerIntegration `json:"integration,omitempty"`
	Digest                      string              `json:"digest"`
}

type ManagerRunLedger struct {
	APIVersion string            `json:"apiVersion"`
	SessionID  string            `json:"sessionId"`
	Budget     *ManagerRunLimits `json:"budget,omitempty"`
	Events     []ManagerRunEvent `json:"events"`
	Digest     string            `json:"digest"`
}

type ManagerRunContextBudget struct {
	RemainingStarts     int           `json:"remainingStarts"`
	RemainingRetries    int           `json:"remainingRetries"`
	RemainingDuration   time.Duration `json:"remainingDuration"`
	RemainingCostMicros int64         `json:"remainingCostMicros"`
	RequestedTimeout    time.Duration `json:"requestedTimeout"`
}

type ManagerRunResult struct {
	Status             string              `json:"status"`
	PriorSessionDigest string              `json:"priorSessionDigest"`
	SessionDigest      string              `json:"sessionDigest"`
	IterationID        string              `json:"iterationId"`
	Phase              string              `json:"phase"`
	AttemptID          string              `json:"attemptId,omitempty"`
	AttemptStatus      string              `json:"attemptStatus,omitempty"`
	Execution          *agentexec.Receipt  `json:"execution,omitempty"`
	ExecutionDigest    string              `json:"executionDigest,omitempty"`
	LedgerDigest       string              `json:"ledgerDigest"`
	Proposal           *ManagerProposal    `json:"proposal,omitempty"`
	Integration        *ManagerIntegration `json:"integration,omitempty"`
}

type ManagerProposalDraft struct {
	Report          DistillationDraft       `json:"report"`
	Hierarchy       []ProposedManager       `json:"hierarchy"`
	PublicContracts []ManagerPublicContract `json:"publicContracts"`
}

type ManagerIntegrationDraft struct {
	Report    DistillationDraft `json:"report"`
	Conflicts []SessionConflict `json:"conflicts"`
}

type managerRunRequestContext struct {
	Kind            string                     `json:"kind"`
	Instructions    string                     `json:"instructions"`
	Phase           string                     `json:"phase"`
	SessionDigest   string                     `json:"sessionDigest"`
	DiscoveryDigest string                     `json:"discoveryDigest"`
	TargetDigest    string                     `json:"targetContextDigest"`
	ManagerContext  ManagerReverseContext      `json:"managerContext"`
	Integration     *ManagerIntegrationContext `json:"integration,omitempty"`
	ModelSchema     any                        `json:"modelSchema"`
	Schema          json.RawMessage            `json:"responseSchema"`
	Budget          ManagerRunContextBudget    `json:"budget"`
}

// PreviewManagerStage constructs the exact bounded request without invoking
// an agent. expectedSessionDigest and retryOfAttemptID are optional for a new
// preview; either becomes part of the preview binding when supplied.
func PreviewManagerStage(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedSessionDigest, retryOfAttemptID string, config agentexec.Config, limits ManagerRunLimits) (ManagerRunPreview, error) {
	return buildManagerRunPreview(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedSessionDigest, retryOfAttemptID, config, limits, AgentExecManagerRunInvoker{})
}

// PreviewManagerStageWithInvoker binds preview to the same explicitly selected
// transport fingerprint used by guarded execution. It never invokes a role.
func PreviewManagerStageWithInvoker(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedSessionDigest, retryOfAttemptID string, config agentexec.Config, limits ManagerRunLimits, invoker ManagerRunInvoker) (ManagerRunPreview, error) {
	if invoker == nil {
		return ManagerRunPreview{}, errors.New("manager preview requires an invoker")
	}
	return buildManagerRunPreview(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedSessionDigest, retryOfAttemptID, config, limits, invoker)
}

// RunManagerStage repeats preview construction under a per-session execution
// lock and invokes exactly one explicitly requested stage only when its digest
// matches. A repeated completed or failed request returns its durable result;
// explicit retry requires the latest failed attempt ID and a fresh preview.
func RunManagerStage(ctx context.Context, sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedPreviewDigest, retryOfAttemptID string, config agentexec.Config, limits ManagerRunLimits, invoker ManagerRunInvoker) (ManagerRunResult, error) {
	var empty ManagerRunResult
	if ctx == nil || invoker == nil {
		return empty, errors.New("manager stage requires context and an invoker")
	}
	dir, err := sessionDirectory(sourceRoot, sessionID, false)
	if err != nil {
		return empty, err
	}
	unlock, err := acquireNamedLock(dir, managerRunLockName)
	if err != nil {
		return empty, err
	}
	defer unlock()
	preview, session, request, _, err := buildManagerRunPreviewInternal(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, "", retryOfAttemptID, config, limits, invoker)
	if err != nil {
		return empty, err
	}
	if expectedPreviewDigest == "" || expectedPreviewDigest != preview.PreviewDigest {
		return ManagerRunResult{Status: "stale-preview", PriorSessionDigest: session.Digest, SessionDigest: session.Digest, IterationID: iterationID, Phase: phase}, errors.New("manager stage preview changed; regenerate preview before invocation")
	}
	ledger, err := loadManagerRunLedger(dir, sessionID)
	if err != nil {
		return empty, err
	}
	limits, err = frozenManagerRunLimits(ledger, limits)
	if err != nil {
		return empty, err
	}
	if complete := completedManagerStage(session, iterationID, phase); complete {
		return managerRunCompletedResult(session, ledger, iterationID, phase), nil
	}
	if prior, ok := latestManagerAttempt(ledger, iterationID, phase); ok {
		if retryOfAttemptID == "" {
			if prior.Status == "succeeded" {
				return recoverManagerRunStage(sourceRoot, targetRoot, session, ledger, prior, preview, invoker, config)
			}
			result := managerRunAttemptResult(session, ledger, prior, "attempt-exists")
			if prior.Status == "failed" && prior.Execution != nil && prior.CostKnown {
				return result, fmt.Errorf("prior Manager attempt failed (%s); retry explicitly with attempt ID %s after reviewing the recorded receipt", prior.SafeFailure, prior.AttemptID)
			}
			return result, fmt.Errorf("prior Manager attempt %s has no recoverable priced terminal receipt; no automatic replay is allowed", prior.Status)
		}
		if prior.AttemptID != retryOfAttemptID || prior.Status != "failed" || prior.Execution == nil || !prior.CostKnown {
			return empty, errors.New("explicit retry must name the latest failed attempt with a known receipt and cost")
		}
	}
	if err := checkManagerRunBudget(ledger, limits, config.Timeout, retryOfAttemptID != ""); err != nil {
		return empty, err
	}
	_, _, elapsedBefore, _, unknownBefore := managerRunUsage(ledger)
	if unknownBefore {
		return empty, errors.New("prior manager invocation has unknown cost; session is blocked fail-closed")
	}
	if ledger.Digest != preview.LedgerDigest {
		return ManagerRunResult{Status: "stale-preview", PriorSessionDigest: session.Digest, SessionDigest: session.Digest, IterationID: iterationID, Phase: phase}, errors.New("manager run ledger changed; regenerate preview before invocation")
	}
	if err := revalidateManagerRunBases(sourceRoot, targetRoot, sessionID, session.Digest, preview.RuntimeFileDigest, invoker, config, preview.ConfigFingerprint); err != nil {
		return empty, err
	}
	attemptID, err := newSessionRunID()
	if err != nil {
		return empty, err
	}
	started := time.Now().UTC()
	privateLogs := limits.PrivateLogDirectory
	if privateLogs == "" {
		privateLogs, err = newPrivateLogDirectory(os.TempDir())
		if err != nil {
			return empty, fmt.Errorf("prepare private invocation log directory: %w", err)
		}
	}
	tempParent := limits.TempParent
	if tempParent == "" {
		tempParent = os.TempDir()
	}
	if ledger.Budget == nil {
		budget := budgetProjection(limits)
		ledger.Budget = &budget
	}
	start := ManagerRunEvent{Sequence: len(ledger.Events) + 1, AttemptID: attemptID, RetryOfAttemptID: retryOfAttemptID, Event: "started", SessionDigest: session.Digest,
		IterationID: iterationID, Phase: phase, ManagerID: preview.ManagerID, AgentManagerID: agentManagerID,
		PreviewDigest: preview.PreviewDigest, ContextDigest: preview.ContextDigest, RequestDigest: preview.RequestDigest, RequestContractDigest: preview.RequestContractDigest,
		ConfigFingerprint: preview.ConfigFingerprint, RuntimeFileDigest: preview.RuntimeFileDigest, LimitsDigest: preview.LimitsDigest,
		Timeout: preview.Timeout, MaxStdoutBytes: preview.MaxStdoutBytes, MaxStderrBytes: preview.MaxStderrBytes,
		InputPriceMicrosPerMillion: preview.InputPriceMicrosPerMillion, OutputPriceMicrosPerMillion: preview.OutputPriceMicrosPerMillion, StartedAt: started}
	sealManagerRunEvent(&start)
	ledger.Events = append(ledger.Events, start)
	sealManagerRunLedger(&ledger)
	if err := writeManagerRunLedger(dir, ledger); err != nil {
		return empty, fmt.Errorf("persist manager invocation start: %w", err)
	}
	result := ManagerRunResult{Status: "invoking", PriorSessionDigest: session.Digest, SessionDigest: session.Digest, IterationID: iterationID, Phase: phase, AttemptID: attemptID}
	runResult, invokeErr := invoker.Run(ctx, config, request, agentexec.RunOptions{InputRoots: []string{}, TempParent: tempParent, PrivateLogDirectory: privateLogs})
	terminal := ManagerRunEvent{Sequence: len(ledger.Events) + 1, AttemptID: attemptID, Event: "terminal", SessionDigest: session.Digest,
		IterationID: iterationID, Phase: phase, ManagerID: preview.ManagerID, AgentManagerID: agentManagerID,
		PreviewDigest: preview.PreviewDigest, ContextDigest: preview.ContextDigest, RequestDigest: preview.RequestDigest, RequestContractDigest: preview.RequestContractDigest,
		ConfigFingerprint: preview.ConfigFingerprint, RuntimeFileDigest: preview.RuntimeFileDigest, LimitsDigest: preview.LimitsDigest,
		Timeout: preview.Timeout, MaxStdoutBytes: preview.MaxStdoutBytes, MaxStderrBytes: preview.MaxStderrBytes,
		InputPriceMicrosPerMillion: preview.InputPriceMicrosPerMillion, OutputPriceMicrosPerMillion: preview.OutputPriceMicrosPerMillion,
		StartedAt: started, FinishedAt: time.Now().UTC(), Status: "failed", SafeFailure: "invocation-failed"}
	if runResult.Receipt.RunID != "" {
		receipt := runResult.Receipt
		terminal.Execution = &receipt
		terminal.ExecutionDigest = digestValue(receipt)
		if cost, known := managerRunCost(receipt.Usage, limits); known {
			terminal.CostKnown, terminal.EstimatedCostMicros = true, cost
		}
	}
	if invokeErr == nil && terminal.Execution == nil {
		invokeErr = errors.New("manager executor returned no execution receipt")
	}
	if invokeErr == nil {
		if err := validateManagerExecutorEnvelope(runResult, request, attemptID); err != nil {
			invokeErr = err
			terminal.SafeFailure = "invalid-executor-envelope"
		}
	}
	if invokeErr == nil && !terminal.CostKnown {
		invokeErr = errors.New("manager executor omitted provider-reported token usage; session cost is unknown")
		terminal.SafeFailure = "usage-unknown"
	}
	var proposal *ManagerProposal
	var integration *ManagerIntegration
	if invokeErr == nil {
		proposal, integration, invokeErr = acceptManagerStageDraft(session, iterationID, phase, preview.ManagerID, runResult.Response.ReportJSON, config, *terminal.Execution)
		if invokeErr != nil {
			terminal.SafeFailure = "invalid-manager-report"
		}
	}
	if invokeErr == nil && terminal.EstimatedCostMicros > preview.RemainingCostMicros {
		invokeErr = errors.New("manager stage exceeded the session-wide reported cost budget")
		terminal.SafeFailure = "cost-budget-exceeded"
	}
	terminal.FinishedAt = time.Now().UTC()
	if invokeErr == nil && (terminal.FinishedAt.Sub(started) > time.Duration(math.MaxInt64)-elapsedBefore || elapsedBefore+terminal.FinishedAt.Sub(started) > limits.MaxDuration) {
		invokeErr = errors.New("manager stage exceeded the session-wide elapsed-duration budget")
		terminal.SafeFailure = "duration-budget-exceeded"
	}
	terminal.Status = "failed"
	if invokeErr == nil {
		terminal.Status = "succeeded"
		terminal.Proposal, terminal.Integration = proposal, integration
	}
	sealManagerRunEvent(&terminal)
	ledger.Events = append(ledger.Events, terminal)
	sealManagerRunLedger(&ledger)
	if err := writeManagerRunLedger(dir, ledger); err != nil {
		return result, fmt.Errorf("retain manager execution receipt: %w", err)
	}
	result.AttemptStatus, result.Execution, result.ExecutionDigest, result.LedgerDigest = terminal.Status, terminal.Execution, terminal.ExecutionDigest, ledger.Digest
	if invokeErr != nil {
		result.Status = "failed"
		return result, invokeErr
	}
	if err := revalidateManagerRunBases(sourceRoot, targetRoot, sessionID, session.Digest, preview.RuntimeFileDigest, invoker, config, preview.ConfigFingerprint); err != nil {
		result.Status = "stale-after-invocation"
		return result, err
	}
	var next BrownfieldSession
	if proposal != nil {
		next, err = RecordManagerProposal(session, iterationID, *proposal)
	} else {
		next, err = IntegrateManagerProposal(session, iterationID, preview.ManagerID, *integration)
	}
	if err != nil {
		result.Status = "stage-not-recorded"
		return result, err
	}
	if _, err = WriteBrownfieldSession(sourceRoot, next, session.Digest); err != nil {
		result.Status = "stage-not-recorded"
		return result, fmt.Errorf("persist validated manager stage after retaining receipt: %w", err)
	}
	result.Status, result.SessionDigest = "recorded", next.Digest
	result.Proposal, result.Integration = proposal, integration
	return result, nil
}

func buildManagerRunPreview(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedSessionDigest, retryOfAttemptID string, config agentexec.Config, limits ManagerRunLimits, invoker ManagerRunInvoker) (ManagerRunPreview, error) {
	preview, _, _, _, err := buildManagerRunPreviewInternal(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedSessionDigest, retryOfAttemptID, config, limits, invoker)
	return preview, err
}

func buildManagerRunPreviewInternal(sourceRoot, targetRoot, sessionID, iterationID, phase, agentManagerID, expectedSessionDigest, retryOfAttemptID string, config agentexec.Config, limits ManagerRunLimits, invoker ManagerRunInvoker) (ManagerRunPreview, BrownfieldSession, agentexec.Request, managerRunRequestContext, error) {
	var empty ManagerRunPreview
	var emptySession BrownfieldSession
	var emptyRequest agentexec.Request
	var emptyContext managerRunRequestContext
	if phase != ManagerRunPhasePropose && phase != ManagerRunPhaseIntegrate {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("manager stage phase must be propose or integrate")
	}
	if invoker == nil || sourceRoot == "" || targetRoot == "" || sessionID == "" || iterationID == "" || agentManagerID == "" {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("manager stage requires source/target, session, iteration, phase, agent mapping, and invoker")
	}
	session, readiness, err := ResumeBrownfieldSession(sourceRoot, targetRoot, sessionID)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	if !readiness.SourceCurrent || !readiness.TargetCurrent {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("Brownfield source or accepted target basis is stale")
	}
	if expectedSessionDigest != "" && expectedSessionDigest != session.Digest {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("Brownfield session changed; reload before preview")
	}
	iteration, ok := findIteration(session, iterationID)
	if !ok {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("unknown Brownfield iteration")
	}
	if !iterationInLatestActiveTree(session, iterationID) {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("requested Manager iteration belongs to a superseded root tree")
	}
	if phase == ManagerRunPhaseIntegrate && iteration.Proposal == nil {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("requested manager stage does not match the iteration's persisted stage")
	}
	var managerContext ManagerReverseContext
	var integrationContext *ManagerIntegrationContext
	if phase == ManagerRunPhasePropose {
		managerContext, err = BuildManagerReverseContext(session, iterationID)
	} else {
		built, buildErr := BuildManagerIntegrationContext(session, iterationID)
		err = buildErr
		if err == nil {
			integrationContext = &built
			managerContext = built.Parent
		}
	}
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	if err := validateManagerAgentMapping(session, iteration, managerContext, agentManagerID); err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	if err := validateManagerRunLimits(targetRoot, config, limits); err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	fingerprint, err := invoker.Fingerprint(config)
	if err != nil || fingerprint == "" {
		return empty, emptySession, emptyRequest, emptyContext, fmt.Errorf("fingerprint manager runtime: %w", err)
	}
	runtimeDigest, err := managerRuntimeFileDigest(targetRoot)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	ledgerDir, err := sessionDirectory(sourceRoot, sessionID, false)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	ledger, err := loadManagerRunLedger(ledgerDir, sessionID)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	limits, err = frozenManagerRunLimits(ledger, limits)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	stageCompleted := completedManagerStage(session, iterationID, phase)
	_, hasPriorAttempt := latestManagerAttempt(ledger, iterationID, phase)
	if !stageCompleted && (!hasPriorAttempt || retryOfAttemptID != "") {
		if err := checkManagerRunBudget(ledger, limits, config.Timeout, retryOfAttemptID != ""); err != nil {
			return empty, emptySession, emptyRequest, emptyContext, err
		}
	}
	usedStarts, usedRetries, usedDuration, usedCost, unknownCost := managerRunUsage(ledger)
	if unknownCost && (!hasPriorAttempt || retryOfAttemptID != "") && !stageCompleted {
		return empty, emptySession, emptyRequest, emptyContext, errors.New("prior manager invocation has unknown cost; session is blocked fail-closed")
	}
	contextBudget := ManagerRunContextBudget{RemainingStarts: limits.MaxStarts - usedStarts, RemainingRetries: limits.MaxRetries - usedRetries,
		RemainingDuration: limits.MaxDuration - usedDuration, RemainingCostMicros: limits.MaxCostMicros - usedCost, RequestedTimeout: config.Timeout}
	ctxData, request, childReports, err := makeManagerRunContext(session, iteration, phase, managerContext, integrationContext, contextBudget)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	ctxBytes, err := json.Marshal(ctxData)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	request.Context = ctxBytes
	request.Role = agentexec.RoleExecutor
	request.SourceRevision = session.Source.Commit
	request.ModelDigest = "sha256:" + session.Source.Digest
	request.ModulePin = "project-adoption/" + managerRunVersion
	request.ProjectionID = "brownfield-manager-" + phase
	request.ScopeIDs = []string{iteration.ManagerID}
	request.PolicyIDs = []string{"selected-evidence-only", "report-only", "no-source-or-target-writes"}
	request.Artifacts, err = managerRunArtifacts(session, managerContext)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	requestDigest := digestValue(request)
	requestContractDigest, err := managerRequestContractDigest(request, ctxData)
	if err != nil {
		return empty, emptySession, emptyRequest, emptyContext, err
	}
	limitsDigest := digestValue(limitsForDigest(limits))
	preview := ManagerRunPreview{SessionDigest: session.Digest, IterationID: iterationID, Phase: phase, ManagerID: iteration.ManagerID,
		AgentManagerID: agentManagerID, ManagerOrigin: managerContext.ManagerOrigin, ProviderVersion: config.ProviderVersion, Model: config.Model,
		ConfigFingerprint: fingerprint, RuntimeFileDigest: runtimeDigest, LimitsDigest: limitsDigest, LedgerDigest: ledger.Digest,
		ContextDigest: managerContext.Digest, RequestDigest: requestDigest, RequestContractDigest: requestContractDigest, RetryOfAttemptID: retryOfAttemptID,
		EvidenceCount: len(managerContext.Evidence), ChildReportCount: childReports, Timeout: config.Timeout, MaxStdoutBytes: config.MaxStdoutBytes,
		MaxStderrBytes: config.MaxStderrBytes, InputPriceMicrosPerMillion: limits.InputPriceMicrosPerMillion, OutputPriceMicrosPerMillion: limits.OutputPriceMicrosPerMillion}
	preview.RemainingStarts = limits.MaxStarts - usedStarts
	preview.RemainingRetries = limits.MaxRetries - usedRetries
	preview.RemainingDuration = limits.MaxDuration - usedDuration
	preview.RemainingCostMicros = limits.MaxCostMicros - usedCost
	if preview.RemainingStarts < 0 {
		preview.RemainingStarts = 0
	}
	if preview.RemainingRetries < 0 {
		preview.RemainingRetries = 0
	}
	if preview.RemainingDuration < 0 {
		preview.RemainingDuration = 0
	}
	if preview.RemainingCostMicros < 0 {
		preview.RemainingCostMicros = 0
	}
	preview.Completed = completedManagerStage(session, iterationID, phase)
	if retryOfAttemptID != "" {
		latest, exists := latestManagerAttempt(ledger, iterationID, phase)
		if !exists || latest.AttemptID != retryOfAttemptID || latest.Status != "failed" || latest.Execution == nil || !latest.CostKnown {
			return empty, emptySession, emptyRequest, emptyContext, errors.New("retry must name the latest failed attempt with a known receipt and cost")
		}
	}
	copy := preview
	copy.PreviewDigest = ""
	preview.PreviewDigest = digestValue(copy)
	return preview, session, request, ctxData, nil
}

func makeManagerRunContext(session BrownfieldSession, iteration ReverseIteration, phase string, managerContext ManagerReverseContext, integration *ManagerIntegrationContext, budget ManagerRunContextBudget) (managerRunRequestContext, agentexec.Request, int, error) {
	var draftSchema any
	kind := "projectadoption-manager-proposal/v1"
	instructions := `Analyze only this Manager's supplied fixed evidence artifacts, accepted public-neighbor contracts, and the active project-model schema. Do not infer behavior from filenames or unsupplied files. Distinguish static observations, documented intent, submitted runtime records, and synthesis hypotheses. Cite exact selected evidence IDs and exact excerpts with inclusive one-based line bounds. Never present runtime records as authenticated execution or claim human acceptance. Return a closed typed report only; do not edit source or target files, invoke tools, or include private transcripts.

IDs must match ^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$; use simple local IDs. Claim, question, contradiction, and scope references must be valid and same-scope where applicable. Every claim and term occurrence must cite selected evidence and quote exact contiguous source text. Use only the exact claim kind/method pairs observation/static-source, documented-intent/documentation, submitted-runtime-record/submitted-record, and hypothesis/synthesis. Copy submitted runtime record fields exactly from selected evidence; never fabricate runtime metadata.

Use modelSchema as authoritative. Do not invent fields or silently normalize YAML. Model proposal files belong only in report.proposal.files, one YAML document per file, under canonical .markitect/model paths, with namespace matching parent folders. For Statement proposals, require apiVersion, kind: Statement, metadata.name, metadata.namespace, top-level purpose, and spec.category, spec.description, spec.public, spec.uses, and spec.requires. uses and requires are arrays of {namespace,name} references. Empty arrays are valid.`
	if phase == ManagerRunPhaseIntegrate {
		kind = "projectadoption-manager-integration/v1"
		instructions += ` Integrate this Manager report with every supplied direct-child proposal and each non-leaf child's final integrated report. Preserve the exact child proposal, public-contract, integration, and report bindings supplied by the Host; do not invent or omit a child. Report unresolved conflicts only; owner resolution is separate. Return a closed ManagerIntegrationDraft with report and conflicts.`
		draftSchema = jsonSchemaForType(reflectTypeOfManagerIntegrationDraft())
	} else {
		draftSchema = jsonSchemaForType(reflectTypeOfManagerProposalDraft())
	}
	schemaBytes, err := json.Marshal(draftSchema)
	if err != nil {
		return managerRunRequestContext{}, agentexec.Request{}, 0, err
	}
	requestContext := managerRunRequestContext{Kind: kind, Instructions: instructions, Phase: phase,
		SessionDigest: session.Digest, DiscoveryDigest: session.Source.Digest, TargetDigest: session.TargetContext.Digest,
		ManagerContext: managerContext, Integration: integration, ModelSchema: projectmodel.Schema(), Schema: schemaBytes, Budget: budget}
	children := 0
	if integration != nil {
		children = len(integration.Children)
	}
	return requestContext, agentexec.Request{}, children, nil
}

// managerRequestContractDigest binds all stable semantic request inputs while
// excluding only cumulative remaining-budget counters, which change after each
// terminal attempt. RequestedTimeout remains part of the contract.
func managerRequestContractDigest(request agentexec.Request, requestContext managerRunRequestContext) (string, error) {
	staticContext := requestContext
	staticContext.Budget.RemainingStarts = 0
	staticContext.Budget.RemainingRetries = 0
	staticContext.Budget.RemainingDuration = 0
	staticContext.Budget.RemainingCostMicros = 0
	contextBytes, err := json.Marshal(staticContext)
	if err != nil {
		return "", err
	}
	contractRequest := request
	contractRequest.Context = contextBytes
	return digestValue(contractRequest), nil
}

func managerRunArtifacts(session BrownfieldSession, context ManagerReverseContext) ([]agentexec.Artifact, error) {
	byID := make(map[string]Evidence, len(session.Source.Evidence))
	for _, evidence := range session.Source.Evidence {
		byID[evidence.ID] = evidence
	}
	out := make([]agentexec.Artifact, 0, len(context.Evidence))
	for _, evidence := range context.Evidence {
		selected, ok := byID[evidence.EvidenceID]
		if !ok || selected.Path != evidence.Path || selected.Basis != evidence.Basis || selected.Digest != evidence.Digest || digestBytes([]byte(selected.Content)) != selected.Digest {
			return nil, fmt.Errorf("Manager context evidence %q differs from fixed Discovery", evidence.EvidenceID)
		}
		mode := "0644"
		if selected.Mode == "100755" {
			mode = "0755"
		}
		out = append(out, agentexec.Artifact{Path: "evidence/" + evidence.EvidenceID + ".txt", Mode: mode,
			Digest: "sha256:" + selected.Digest, Content: []byte(selected.Content)})
	}
	return out, nil
}

func validateManagerAgentMapping(session BrownfieldSession, iteration ReverseIteration, context ManagerReverseContext, agentManagerID string) error {
	if context.ManagerOrigin == "accepted-target" {
		if agentManagerID != iteration.ManagerID {
			return errors.New("accepted Manager must use its exact configured agent mapping")
		}
		return nil
	}
	if context.ManagerOrigin != "proposed-by-parent" {
		return errors.New("unknown Manager origin")
	}
	currentIteration := iteration
	for hops := 0; hops <= len(session.Iterations)+len(session.TargetContext.Managers); hops++ {
		if currentIteration.ParentIterationID == "" {
			return errors.New("proposed Manager has no accepted ancestor for runtime mapping")
		}
		parent, ok := findIteration(session, currentIteration.ParentIterationID)
		if !ok {
			return errors.New("proposed Manager parent iteration is unavailable")
		}
		if _, accepted := targetManager(session.TargetContext, parent.ManagerID); accepted {
			if parent.ManagerID != agentManagerID {
				return errors.New("proposed Manager runtime mapping must use its nearest accepted ancestor")
			}
			return nil
		}
		currentIteration = parent
	}
	return errors.New("proposed Manager ancestry exceeds the bounded hierarchy")
}

func iterationInLatestActiveTree(session BrownfieldSession, iterationID string) bool {
	latestRoot := ""
	for _, candidate := range session.Iterations {
		if candidate.ParentIterationID == "" {
			latestRoot = candidate.ID
		}
	}
	if latestRoot == "" {
		return false
	}
	current, ok := findIteration(session, iterationID)
	if !ok {
		return false
	}
	for current.ParentIterationID != "" {
		parent, exists := findIteration(session, current.ParentIterationID)
		if !exists {
			return false
		}
		current = parent
	}
	return current.ID == latestRoot
}

func validateManagerRunLimits(targetRoot string, config agentexec.Config, limits ManagerRunLimits) error {
	if limits.MaxStarts <= 0 || limits.MaxStarts > 10_000 || limits.MaxRetries < 0 || limits.MaxRetries >= limits.MaxStarts || limits.MaxDuration <= 0 || limits.MaxDuration > managerRunMaxDuration || limits.MaxCostMicros <= 0 || limits.MaxCostMicros > managerRunMaxCost {
		return errors.New("manager runtime limits must be finite positive session-wide bounds")
	}
	if limits.InputPriceMicrosPerMillion <= 0 || limits.OutputPriceMicrosPerMillion <= 0 || limits.InputPriceMicrosPerMillion > managerRunMaxCost || limits.OutputPriceMicrosPerMillion > managerRunMaxCost {
		return errors.New("manager runtime requires finite positive input and output token prices")
	}
	options := DistillationRunOptions{MaxTimeout: limits.MaxTimeout, MaxStdoutBytes: limits.MaxStdoutBytes, MaxStderrBytes: limits.MaxStderrBytes,
		MaxCostMicros: limits.MaxCostMicros, InputPriceMicrosPerMillion: limits.InputPriceMicrosPerMillion,
		OutputPriceMicrosPerMillion: limits.OutputPriceMicrosPerMillion, TempParent: limits.TempParent,
		PrivateLogDirectory: limits.PrivateLogDirectory}
	if err := validateDistillationRunnerOptions(targetRoot, config, options); err != nil {
		return err
	}
	return nil
}

func managerRuntimeFileDigest(targetRoot string) (string, error) {
	path := filepath.Join(targetRoot, filepath.FromSlash(managerRunRuntimePath))
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("read current target Runtime: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("target Runtime must be a regular nonsymlink file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func revalidateManagerRunBases(sourceRoot, targetRoot, sessionID, sessionDigest, runtimeDigest string, invoker ManagerRunInvoker, config agentexec.Config, configFingerprint string) error {
	current, readiness, err := ResumeBrownfieldSession(sourceRoot, targetRoot, sessionID)
	if err != nil {
		return fmt.Errorf("revalidate fixed Brownfield bases: %w", err)
	}
	if current.Digest != sessionDigest || !readiness.SourceCurrent || !readiness.TargetCurrent {
		return errors.New("Brownfield session or fixed bases changed during manager invocation")
	}
	gotRuntime, err := managerRuntimeFileDigest(targetRoot)
	if err != nil || gotRuntime != runtimeDigest {
		return errors.New("target Runtime changed during manager invocation")
	}
	fingerprint, err := invoker.Fingerprint(config)
	if err != nil || fingerprint != configFingerprint {
		return errors.New("manager runtime fingerprint changed during invocation")
	}
	return nil
}

func validateManagerExecutorEnvelope(result agentexec.RunResult, request agentexec.Request, attemptID string) error {
	response := result.Response
	if response.APIVersion != agentexec.APIVersion || response.Role != agentexec.RoleExecutor || response.Outcome != agentexec.OutcomeProposed || response.InputDigest != result.Receipt.InputDigest || response.InputDigest == "" || response.RunID != result.Receipt.RunID {
		return errors.New("manager executor did not return a proposed response bound to its execution receipt")
	}
	if len(response.CandidateFiles) != 0 || len(response.CandidateJSON) != 0 || len(response.VerifierObservations) != 0 || len(response.ReportJSON) == 0 {
		return errors.New("manager executor must return a report-only response without candidate files or JSON")
	}
	if result.Receipt.Outcome != agentexec.OutcomeProposed || result.Receipt.ContextDigest == "" || result.Receipt.ConfigDigest == "" || len(request.Context) == 0 {
		return errors.New("manager executor receipt does not bind the requested context and configuration")
	}
	if digestValue(response.Usage) != digestValue(result.Receipt.Usage) {
		return errors.New("manager executor receipt usage differs from the returned response")
	}
	_ = attemptID
	return nil
}

func acceptManagerStageDraft(session BrownfieldSession, iterationID, phase, managerID string, reportJSON json.RawMessage, config agentexec.Config, receipt agentexec.Receipt) (*ManagerProposal, *ManagerIntegration, error) {
	iteration, ok := findIteration(session, iterationID)
	if !ok {
		return nil, nil, errors.New("unknown manager iteration")
	}
	if phase == ManagerRunPhasePropose {
		var draft ManagerProposalDraft
		if err := decodeClosedJSON(reportJSON, &draft); err != nil {
			return nil, nil, fmt.Errorf("decode manager proposal draft: %w", err)
		}
		report, err := bindManagerDistillationDraft(session, iterationID, draft.Report, config, receipt)
		if err != nil {
			return nil, nil, err
		}
		proposal := ManagerProposal{ManagerID: iteration.ManagerID, EvidenceIDs: append([]string{}, iteration.EvidenceIDs...),
			Hierarchy: draft.Hierarchy, PublicContracts: draft.PublicContracts, Report: report}
		next, err := RecordManagerProposal(session, iterationID, proposal)
		if err != nil {
			return nil, nil, fmt.Errorf("validate Manager proposal: %w", err)
		}
		accepted, _ := findIteration(next, iterationID)
		return accepted.Proposal, nil, nil
	}
	var draft ManagerIntegrationDraft
	if err := decodeClosedJSON(reportJSON, &draft); err != nil {
		return nil, nil, fmt.Errorf("decode manager integration draft: %w", err)
	}
	report, err := bindManagerDistillationDraft(session, iterationID, draft.Report, config, receipt)
	if err != nil {
		return nil, nil, err
	}
	context, err := BuildManagerIntegrationContext(session, iterationID)
	if err != nil {
		return nil, nil, err
	}
	childDigests := []string{}
	childContracts := []IntegratedChildContracts{}
	childIntegrations := []ChildIntegrationDigest{}
	for _, child := range context.Children {
		childDigests = append(childDigests, child.ProposalDigest)
		childContracts = append(childContracts, IntegratedChildContracts{ManagerID: child.ManagerID, ProposalDigest: child.ProposalDigest, Contracts: append([]ManagerPublicContract{}, child.PublicContracts...)})
		if child.IntegrationDigest != "" {
			childIntegrations = append(childIntegrations, ChildIntegrationDigest{ManagerID: child.ManagerID, ProposalDigest: child.ProposalDigest,
				IntegrationDigest: child.IntegrationDigest, ReportDigest: child.ReportDigest})
		}
	}
	sort.Strings(childDigests)
	integration := ManagerIntegration{ManagerID: managerID, ChildProposalDigests: childDigests, ChildContracts: childContracts,
		ChildIntegrationDigests: childIntegrations, Report: report, Conflicts: draft.Conflicts}
	next, err := IntegrateManagerProposal(session, iterationID, managerID, integration)
	if err != nil {
		return nil, nil, fmt.Errorf("validate Manager integration: %w", err)
	}
	accepted, _ := findIteration(next, iterationID)
	return nil, accepted.Integration, nil
}

func bindManagerDistillationDraft(session BrownfieldSession, iterationID string, draft DistillationDraft, config agentexec.Config, receipt agentexec.Receipt) (Distillation, error) {
	iteration, ok := findIteration(session, iterationID)
	if !ok {
		return Distillation{}, errors.New("unknown manager iteration")
	}
	schemaDigest, _, err := CurrentBindings(projectmodel.Schema())
	if err != nil {
		return Distillation{}, err
	}
	_ = iteration
	report := Distillation{APIVersion: DistillationVersion, DiscoveryDigest: session.Source.Digest, TargetBasis: session.Target.ProjectDigest,
		TargetRevision: session.Target.Revision, TargetContextDigest: session.TargetContext.Digest, Method: "agent-assisted", RunnerIdentity: distillationRunnerIdentity(config),
		RunnerDigest: digestWithoutPrefix(receipt.ConfigDigest), SchemaDigest: schemaDigest,
		Claims: []Claim{}, Terms: draft.Terms, Contradictions: draft.Contradictions, Questions: []Question{}, Scopes: []ScopeProposal{}, Proposal: draft.Proposal}
	for _, claim := range draft.Claims {
		var runtime *RuntimeObservation
		if claim.RuntimeObservationJSON != "" {
			var parsed RuntimeObservation
			if err := decodeClosedJSON([]byte(claim.RuntimeObservationJSON), &parsed); err != nil {
				return Distillation{}, err
			}
			runtime = &parsed
		}
		report.Claims = append(report.Claims, Claim{ID: claim.ID, ScopeID: claim.ScopeID, Kind: claim.Kind, Method: claim.Method, Statement: claim.Statement,
			Evidence: claim.Evidence, Uncertainty: claim.Uncertainty, Runtime: runtime})
	}
	for _, question := range draft.Questions {
		blocking := question.Blocking
		report.Questions = append(report.Questions, Question{ID: question.ID, ScopeID: question.ScopeID,
			Prompt: question.Prompt, Alternatives: question.Alternatives, ClaimIDs: question.ClaimIDs, Blocking: &blocking})
	}
	for _, scope := range draft.Scopes {
		report.Scopes = append(report.Scopes, ScopeProposal{ID: scope.ID, Name: scope.Name, ParentID: scope.ParentID, ClaimIDs: scope.ClaimIDs, OwnerCandidate: scope.OwnerCandidate})
	}
	SealDistillation(&report)
	if err := ValidateDistillation(session.Source, report); err != nil {
		return Distillation{}, fmt.Errorf("validate manager distillation report: %w", err)
	}
	if err := validateSessionReportTarget(report, session); err != nil {
		return Distillation{}, err
	}
	return report, nil
}
