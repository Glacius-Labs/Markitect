package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

func managerStageRunID(sessionID string) string {
	return strings.TrimPrefix(rawContentDigest([]byte("projectrun-manager-session\x00"+sessionID)), "sha256:")[:32]
}

func validateManagerStageBudget(b ManagerStageBudget) error {
	if b.MaxStarts < 1 || b.MaxStarts > maxRoleStartBudget || b.MaxRetries < 0 || b.MaxRetries > maxRoleStartBudget || b.MaxDuration <= 0 || b.MaxDuration > 24*time.Hour || b.MaxCostMicros <= 0 || b.MaxCostMicros > 1_000_000_000_000 {
		return errors.New("Manager session requires finite supported resource ceilings")
	}
	return nil
}

func frozenManagerStageBudget(initial, current ManagerStageBudget) ManagerStageBudget {
	current.MaxStarts = min(initial.MaxStarts, current.MaxStarts)
	current.MaxRetries = min(initial.MaxRetries, current.MaxRetries)
	current.MaxDuration = min(initial.MaxDuration, current.MaxDuration)
	current.MaxCostMicros = min(initial.MaxCostMicros, current.MaxCostMicros)
	return current
}

func managerStageBounds(agent projectconfig.Agent, config agentexec.Config) ManagerStageBounds {
	return ManagerStageBounds{Timeout: config.Timeout, MaxStdoutBytes: config.MaxStdoutBytes, MaxStderrBytes: config.MaxStderrBytes, InputPriceMicrosPerMillion: agent.Pricing.InputMicrosPerMillion, OutputPriceMicrosPerMillion: agent.Pricing.OutputMicrosPerMillion}
}

// Brownfield preserves its stricter, separately rounded input/output estimates.
// Ordinary metered/unmetered accounting remains estimateAgentCost unchanged.
func managerStageCost(usage *agentexec.Usage, b ManagerStageBounds) (int64, bool, bool) {
	if usage == nil || usage.Source != "provider-reported" || usage.InputTokens == nil || usage.OutputTokens == nil || usage.ToolCalls != nil && *usage.ToolCalls != 0 {
		return 0, false, false
	}
	tokens := []int64{*usage.InputTokens, *usage.OutputTokens}
	prices := []int64{b.InputPriceMicrosPerMillion, b.OutputPriceMicrosPerMillion}
	var total int64
	for i, n := range tokens {
		price := prices[i]
		if n < 0 || price <= 0 || price > 1_000_000_000_000 {
			return 0, false, false
		}
		whole, remainder := n/1_000_000, n%1_000_000
		if whole > math.MaxInt64/price {
			return math.MaxInt64, false, true
		}
		partial := remainder * price / 1_000_000
		if remainder*price%1_000_000 != 0 {
			partial++
		}
		cost := whole * price
		if cost > math.MaxInt64-partial || total > math.MaxInt64-cost-partial {
			return math.MaxInt64, false, true
		}
		total += cost + partial
	}
	return total, true, false
}

func managerStageUsage(report RunReport) (starts, retries int, elapsed time.Duration, cost int64, unknown bool) {
	seen := map[string]bool{}
	for _, log := range report.Invocations {
		a := log.Attempt
		if a == nil || seen[a.ID] {
			unknown = true
			continue
		}
		seen[a.ID] = true
		if a.RetryOfAttemptID != "" {
			retries++
		}
		if a.Status == "started" || a.Status == "uncertain" || !log.CostKnown || log.CostOverflow || log.Receipt.RunID == "" {
			unknown = true
		}
		d := a.FinishedAt.Sub(a.StartedAt)
		if a.FinishedAt.IsZero() || d < 0 || d > time.Duration(math.MaxInt64)-elapsed {
			unknown = true
		} else {
			elapsed += d
		}
		if log.CostMicros < 0 || cost > math.MaxInt64-log.CostMicros {
			unknown = true
		} else {
			cost += log.CostMicros
		}
	}
	starts = reportStartCount(report)
	for _, r := range report.RoleStartReservations {
		if r.Kind == "root" && !seen[r.Request.RequestID] {
			unknown = true
			starts = max(starts, len(report.RoleStartReservations))
		}
	}
	return
}

func reportStartCount(report RunReport) int {
	observed, _ := reportStartAccounting(report)
	roots := map[string]bool{}
	for _, r := range report.RoleStartReservations {
		if r.Kind == "root" {
			roots[r.Key] = true
		}
	}
	return max(observed.Roots, len(roots)) + observed.ObservedNested
}

func canonicalManagerStageSpec(root string, spec ManagerStageSpec, invoker Invoker) (ManagerStageSpec, error) {
	if invoker == nil || spec.Binding.SessionID == "" || spec.Binding.SourceIdentity == "" || spec.Binding.TargetIdentity == "" || spec.Binding.SourceRevision == "" || spec.Binding.DiscoveryDigest == "" || spec.Binding.TargetContextDigest == "" {
		return spec, errors.New("Manager stage requires fixed session/source/target facts and an invoker")
	}
	if spec.Validation.SessionID != spec.Binding.SessionID || spec.Validation.SessionDigest == "" || spec.Validation.IterationID == "" || spec.Validation.ManagerID == "" || spec.Validation.AgentManagerID == "" || spec.Validation.ContextDigest == "" || spec.Validation.RequestContractDigest == "" || spec.Validation.RuntimeFileDigest == "" || (spec.Validation.Phase != "propose" && spec.Validation.Phase != "integrate") {
		return spec, errors.New("Manager stage requires exact prepared workflow bindings")
	}
	sourceRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return spec, err
	}
	selected, err := filepath.EvalSymlinks(spec.SelectedSourceRoot)
	if err != nil {
		return spec, err
	}
	target, err := filepath.EvalSymlinks(spec.TargetRoot)
	if err != nil {
		return spec, err
	}
	sourceRoot, err = filepath.Abs(sourceRoot)
	if err != nil {
		return spec, err
	}
	selected, err = filepath.Abs(selected)
	if err != nil {
		return spec, err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return spec, err
	}
	if !sameRecoveryPath(sourceRoot, selected) {
		return spec, errors.New("selected source differs from its session store")
	}
	spec.SelectedSourceRoot, spec.TargetRoot = selected, target
	if spec.Validation.SourceRoot == "" || spec.Validation.TargetRoot == "" || spec.Validation.AcceptedModelDigest == "" {
		return spec, errors.New("Manager stage lacks trusted accepted context/root facts")
	}
	trustedSource, err := canonicalManagerStageRoot(spec.Validation.SourceRoot)
	if err != nil {
		return spec, err
	}
	trustedTarget, err := canonicalManagerStageRoot(spec.Validation.TargetRoot)
	if err != nil {
		return spec, err
	}
	if !sameRecoveryPath(selected, trustedSource) || !sameRecoveryPath(target, trustedTarget) || spec.Binding.SourceRootDigest != rawContentDigest([]byte(selected)) || spec.Binding.TargetRootDigest != rawContentDigest([]byte(target)) {
		return spec, errors.New("Manager stage roots differ from trusted session facts")
	}
	spec.Validation.SourceRoot, spec.Validation.TargetRoot = selected, target
	if err := validateManagerStageBudget(spec.Binding.Budget); err != nil {
		return spec, err
	}
	if err := validateManagerStageBudget(spec.Limits); err != nil {
		return spec, err
	}
	if spec.Agent.Unmetered() || spec.Agent.Pricing.InputMicrosPerMillion <= 0 || spec.Agent.Pricing.OutputMicrosPerMillion <= 0 {
		return spec, errors.New("Brownfield requires metered reported usage")
	}
	if spec.Config.Timeout <= 0 || spec.Config.Timeout > time.Hour || spec.Config.MaxStdoutBytes <= 0 || spec.Config.MaxStderrBytes <= 0 {
		return spec, errors.New("Manager invocation process bounds are invalid")
	}
	if spec.Request.Role != agentexec.RoleExecutor || spec.Request.SourceRevision != spec.Binding.SourceRevision || len(spec.Request.Context) == 0 {
		return spec, errors.New("Manager request differs from its fixed source binding")
	}
	fingerprint, err := fingerprintForRequest(invoker, spec.Config, spec.Request)
	if err != nil {
		return spec, errors.New("Manager configuration cannot be fingerprinted")
	}
	if fingerprint != spec.Validation.ConfigFingerprint {
		return spec, errors.New("Manager configuration fingerprint differs from prepared facts")
	}
	return spec, nil
}

func readManagerStageState(store *runStore, b ManagerStageBinding) (RunReport, bool, error) {
	id := managerStageRunID(b.SessionID)
	dir, err := store.runDir(id)
	if err != nil {
		return RunReport{}, false, err
	}
	if _, err := os.Lstat(filepath.Join(dir, "plan.json")); err == nil {
		return RunReport{}, false, errors.New("Manager session identity collides with an ordinary plan")
	}
	state, err := store.readLatestState(id)
	if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
		return RunReport{}, false, nil
	}
	if err != nil {
		return state, false, err
	}
	if state.ManagerStageBinding == nil {
		return state, false, errors.New("Manager session has no stage authority")
	}
	before := *state.ManagerStageBinding
	after := b
	// The first ceiling is authoritative even when current runtime ceilings tighten.
	after.Budget = before.Budget
	x, _ := digest(before)
	y, _ := digest(after)
	if x != y {
		return state, false, errors.New("fixed Manager session/source/target binding changed")
	}
	return state, true, nil
}

func managerStageKey(session, iteration, phase, manager string) string {
	return session + "\x00" + iteration + "\x00" + phase + "\x00" + manager
}

func latestManagerStageAttempt(report RunReport, key string) (InvocationLog, bool) {
	for i := len(report.Invocations) - 1; i >= 0; i-- {
		if a := report.Invocations[i].Attempt; a != nil && a.StageKey == key && a.ParentAttemptID == "" {
			return report.Invocations[i], true
		}
	}
	return InvocationLog{}, false
}

func managerStageReviewProjection(invoker Invoker, spec ManagerStageSpec) (ManagerStageReviewProjection, error) {
	if spec.Review == nil {
		return ManagerStageReviewProjection{}, nil
	}
	agent, ok := spec.Review.Agents[spec.Validation.AgentManagerID]
	if !ok {
		return ManagerStageReviewProjection{}, errors.New("Manager assessment has no explicit configured reviewer")
	}
	config, err := agent.AgentConfig()
	if err != nil {
		return ManagerStageReviewProjection{}, errors.New("configured reviewer is invalid")
	}
	fp, err := invoker.Fingerprint(config)
	if err != nil {
		return ManagerStageReviewProjection{}, errors.New("configured reviewer cannot be fingerprinted")
	}
	projection := ManagerStageReviewProjection{Enabled: true, AgentManagerID: spec.Validation.AgentManagerID, ConfigFingerprint: fp, MaxRounds: spec.Review.MaxRounds, MaxManagerRounds: spec.Review.MaxManagerRounds}
	projection.PolicyDigest, err = digest(projection)
	return projection, err
}

// PreviewManagerStage is provider-free and never initializes the store.
func PreviewManagerStage(invoker Invoker, root string, spec ManagerStageSpec) (ManagerStagePreview, error) {
	spec, err := canonicalManagerStageSpec(root, spec, invoker)
	if err != nil {
		return ManagerStagePreview{}, err
	}
	store, err := newManagerStageStore(spec.SelectedSourceRoot)
	if err != nil {
		return ManagerStagePreview{}, err
	}
	report, exists, err := readManagerStageState(store, spec.Binding)
	if err != nil {
		return ManagerStagePreview{}, err
	}
	budget := frozenManagerStageBudget(spec.Binding.Budget, spec.Limits)
	if exists {
		budget = frozenManagerStageBudget(effectiveManagerStageBudget(report), budget)
	}
	starts, retries, elapsed, cost, _ := managerStageUsage(report)
	preview := ManagerStagePreview{RunID: managerStageRunID(spec.Binding.SessionID), StateDigest: report.Digest, Bounds: managerStageBounds(spec.Agent, spec.Config), RemainingStarts: max(0, budget.MaxStarts-starts), RemainingRetries: max(0, budget.MaxRetries-retries), RemainingDuration: max(0, budget.MaxDuration-elapsed), RemainingCostMicros: max(0, budget.MaxCostMicros-cost)}
	if !exists {
		preview.StateDigest = rawContentDigest([]byte("absent-manager-state\x00" + preview.RunID))
	}
	preview.RequestDigest, err = digest(spec.Request)
	if err != nil {
		return preview, err
	}
	preview.LimitsDigest, err = digest(struct {
		Budget ManagerStageBudget
		Bounds ManagerStageBounds
	}{budget, preview.Bounds})
	if err != nil {
		return preview, err
	}
	preview.Review, err = managerStageReviewProjection(invoker, spec)
	if err != nil {
		return preview, err
	}
	key := managerStageKey(spec.Binding.SessionID, spec.Validation.IterationID, spec.Validation.Phase, spec.Validation.ManagerID)
	if latest, ok := latestManagerStageAttempt(report, key); ok {
		copy := *latest.Attempt
		preview.LatestAttempt = &copy
	}
	// Explicit safe projection; never marshal spec, Agent, config or request here.
	preview.PreviewDigest, err = digest(struct {
		Binding                ManagerStageBinding
		Validation             ManagerStageValidationBinding
		Retry                  string
		State, Request, Limits string
		Review                 ManagerStageReviewProjection
	}{spec.Binding, spec.Validation, spec.RetryOfAttemptID, preview.StateDigest, preview.RequestDigest, preview.LimitsDigest, preview.Review})
	return preview, err
}

func managerStageAdmission(report RunReport, budget ManagerStageBudget, timeout time.Duration, retry bool) error {
	starts, retries, elapsed, cost, unknown := managerStageUsage(report)
	if unknown {
		return errors.New("prior Manager reservation or attempt has unknown terminal accounting; replay prohibited")
	}
	if starts >= budget.MaxStarts || retry && retries >= budget.MaxRetries {
		return errors.New("Manager session start or retry ceiling is exhausted")
	}
	if timeout > time.Duration(math.MaxInt64)-elapsed || elapsed+timeout > budget.MaxDuration {
		return errors.New("Manager invocation exceeds remaining cumulative duration")
	}
	if cost >= budget.MaxCostMicros {
		return errors.New("Manager session reported cost ceiling is exhausted")
	}
	return nil
}

func initializeManagerStage(store *runStore, b ManagerStageBinding) (RunReport, error) {
	id := managerStageRunID(b.SessionID)
	dir, err := store.runDir(id)
	if err != nil {
		return RunReport{}, err
	}
	if _, err := os.Lstat(dir); os.IsNotExist(err) {
		if _, err := store.createRun(id); err != nil {
			return RunReport{}, err
		}
	} else if err != nil {
		return RunReport{}, err
	}
	for _, name := range []string{"states", "reports", "candidates"} {
		if err := ensureDirectory(filepath.Join(dir, name)); err != nil {
			return RunReport{}, err
		}
	}
	now := time.Now().UTC()
	report := RunReport{APIVersion: APIVersion, ID: id, Status: StatusRunning, Mode: projectconfig.ModeControlledLocal, StartedAt: now, UpdatedAt: now, BaseRevision: b.SourceRevision, ModelDigest: b.TargetContextDigest, ManagerStageBinding: &b, ManagerStageLimits: &b.Budget, Revision: 1}
	if err := store.appendState(report); err != nil {
		return report, err
	}
	return store.readLatestState(id)
}

func newStageInvocation(spec ManagerStageSpec, preview ManagerStagePreview, id string) InvocationLog {
	a := &ManagerStageAttempt{ID: id, StageKey: managerStageKey(spec.Binding.SessionID, spec.Validation.IterationID, spec.Validation.Phase, spec.Validation.ManagerID), RetryOfAttemptID: spec.RetryOfAttemptID, SessionDigest: spec.Validation.SessionDigest, IterationID: spec.Validation.IterationID, ManagerID: spec.Validation.ManagerID, AgentManagerID: spec.Validation.AgentManagerID, Phase: spec.Validation.Phase, PreviewDigest: preview.PreviewDigest, ContextDigest: spec.Validation.ContextDigest, RequestDigest: preview.RequestDigest, RequestContractDigest: spec.Validation.RequestContractDigest, ConfigFingerprint: spec.Validation.ConfigFingerprint, RuntimeFileDigest: spec.Validation.RuntimeFileDigest, LimitsDigest: preview.LimitsDigest, AcceptedTargetHead: spec.Validation.AcceptedTargetHead, AcceptedModelDigest: spec.Validation.AcceptedModelDigest, Bounds: preview.Bounds, StartedAt: time.Now().UTC(), Status: "started", ReviewRequired: preview.Review.Enabled, ReviewPolicyDigest: preview.Review.PolicyDigest}
	return InvocationLog{TaskID: spec.Validation.IterationID, Role: agentexec.RoleExecutor, Phase: spec.Validation.Phase, InputDigest: preview.RequestDigest, Outcome: "started", Attempt: a}
}

func RunManagerStage(ctx context.Context, host Host, invoker Invoker, root string, spec ManagerStageSpec, expectedPreviewDigest string) (ManagerStageResult, error) {
	if ctx == nil || host.ValidateManagerStage == nil {
		return ManagerStageResult{}, errors.New("Manager stage requires context and Application fact validation")
	}
	if err := ctx.Err(); err != nil {
		return ManagerStageResult{}, err
	}
	spec, err := canonicalManagerStageSpec(root, spec, invoker)
	if err != nil {
		return ManagerStageResult{}, err
	}
	store, err := newManagerStageStore(spec.SelectedSourceRoot)
	if err != nil {
		return ManagerStageResult{}, err
	}
	unlock, err := store.lock()
	if err != nil {
		return ManagerStageResult{}, err
	}
	defer unlock()
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		return ManagerStageResult{}, err
	}
	if expectedPreviewDigest == "" || expectedPreviewDigest != preview.PreviewDigest {
		return ManagerStageResult{}, errors.New("Manager stage preview changed; no dispatch")
	}
	if err := host.ValidateManagerStage(spec.Validation); err != nil {
		return ManagerStageResult{}, err
	}
	report, exists, err := readManagerStageState(store, spec.Binding)
	if err != nil {
		return ManagerStageResult{}, err
	}
	if !exists {
		report, err = initializeManagerStage(store, spec.Binding)
		if err != nil {
			return ManagerStageResult{}, err
		}
	}
	budget, err := tightenManagerStageBudget(store, &report, spec.Limits)
	if err != nil {
		return ManagerStageResult{}, err
	}
	limits := spec.RuntimeLimits
	limits.MaxStarts = budget.MaxStarts
	journal, err := newManagerStageJournal(store, &report, limits)
	if err != nil {
		return ManagerStageResult{}, err
	}
	key := managerStageKey(spec.Binding.SessionID, spec.Validation.IterationID, spec.Validation.Phase, spec.Validation.ManagerID)
	if previous, ok := latestManagerStageAttempt(report, key); ok {
		if spec.RetryOfAttemptID == "" {
			return recoverManagerStage(ctx, host, invoker, journal, spec, preview, previous)
		}
		if previous.Attempt.ID != spec.RetryOfAttemptID || (previous.Attempt.Status != "rejected" && previous.Attempt.Status != "transport-failed") || previous.Receipt.RunID == "" || !previous.CostKnown || previous.CostOverflow {
			return ManagerStageResult{}, errors.New("retry must name the latest priced failed Manager attempt")
		}
	} else if spec.RetryOfAttemptID != "" {
		return ManagerStageResult{}, errors.New("retry parent is absent")
	}
	if err := managerStageAdmission(report, budget, spec.Config.Timeout, spec.RetryOfAttemptID != ""); err != nil {
		return ManagerStageResult{}, err
	}
	id, err := newID()
	if err != nil {
		return ManagerStageResult{}, err
	}
	log := newStageInvocation(spec, preview, id)
	result, log, invokeErr := invokeManagerAttempt(ctx, journal, log, spec.Request, spec.Agent, func(call context.Context) (agentexec.RunResult, error) {
		return dispatchReadOnlyManagerStage(call, host, invoker, spec.SelectedSourceRoot, spec.TargetRoot, report.ID, id, spec.Config, spec.Request, spec.RuntimeLimits)
	})
	if log.Attempt == nil || log.Attempt.ResponseRef == nil {
		return ManagerStageResult{RunID: report.ID, AttemptID: id, Status: "uncertain"}, invokeErr
	}
	out, err := journal.result(log, result)
	if err != nil {
		return out, errors.Join(invokeErr, err)
	}
	if invokeErr != nil {
		return out, invokeErr
	}
	if err := validateManagerStageEnvelope(result, spec.Request, spec.Validation.ConfigFingerprint); err != nil {
		return out, err
	}
	if err := host.ValidateManagerStage(spec.Validation); err != nil {
		return out, err
	}
	return out, managerStageAttachability(report, budget)
}

func validateManagerStageEnvelope(result agentexec.RunResult, request agentexec.Request, fingerprint string) error {
	expected, err := nativeRequestInputDigest(request)
	if err != nil {
		return err
	}
	response := result.Response
	receipt := result.Receipt
	if response.APIVersion != agentexec.APIVersion || response.Role != agentexec.RoleExecutor || response.Outcome != agentexec.OutcomeProposed || response.RunID == "" || response.RunID != receipt.RunID || response.InputDigest != expected || receipt.InputDigest != expected || receipt.ConfigDigest != fingerprint || receipt.ContextDigest == "" || receipt.Outcome != response.Outcome {
		return errors.New("Manager response or receipt differs from its exact invocation")
	}
	if len(response.CandidateFiles) != 0 || len(response.CandidateJSON) != 0 || len(response.VerifierObservations) != 0 || len(response.ReportJSON) == 0 || result.Delta != nil && len(result.Delta.Changes) != 0 {
		return errors.New("Manager stage must return only its typed report without writes")
	}
	a, _ := digest(response.Usage)
	b, _ := digest(receipt.Usage)
	if a != b {
		return errors.New("Manager response usage differs from its retained receipt")
	}
	return nil
}

func recoverManagerStage(ctx context.Context, host Host, invoker Invoker, j *managerStageJournal, spec ManagerStageSpec, preview ManagerStagePreview, log InvocationLog) (ManagerStageResult, error) {
	a := log.Attempt
	if a.SessionDigest != spec.Validation.SessionDigest || a.ContextDigest != spec.Validation.ContextDigest || a.RequestContractDigest != spec.Validation.RequestContractDigest || a.ConfigFingerprint != spec.Validation.ConfigFingerprint || a.RuntimeFileDigest != spec.Validation.RuntimeFileDigest || a.LimitsDigest != preview.LimitsDigest || a.Bounds != preview.Bounds || a.ReviewPolicyDigest != preview.Review.PolicyDigest || a.AcceptedTargetHead != spec.Validation.AcceptedTargetHead || a.AcceptedModelDigest != spec.Validation.AcceptedModelDigest {
		return ManagerStageResult{}, errors.New("original Manager attempt bindings are stale; replay prohibited")
	}
	if a.Status == "rejected" || a.Status == "transport-failed" {
		out, err := j.result(log, agentexec.RunResult{Receipt: log.Receipt})
		return out, errors.Join(err, errors.New("prior Manager attempt failed; retry explicitly with its priced receipt"))
	}
	var result agentexec.RunResult
	var err error
	if a.Status == "started" || a.Status == "uncertain" {
		if spec.Config.Transport != projectconfig.TransportCodexAppServer {
			return ManagerStageResult{}, errors.New("Manager execution is uncertain; automatic replay prohibited")
		}
		data, err := readStageEvidence(j.store, j.report.ID, a.RequestRef)
		if err != nil {
			return ManagerStageResult{}, err
		}
		var original agentexec.Request
		if err := json.Unmarshal(data, &original); err != nil {
			return ManagerStageResult{}, err
		}
		result, err = recoverReadOnlyManagerStage(ctx, host, invoker, spec.SelectedSourceRoot, j.report.ID, a.ID, spec.Config, original, spec.RuntimeLimits, log.Receipt)
		cost, known, overflow := managerStageCost(result.Receipt.Usage, a.Bounds)
		log.CostMicros, log.CostKnown, log.CostOverflow = cost, known, overflow
		log, err = j.terminal(log, result, err)
		if err != nil {
			return ManagerStageResult{}, err
		}
		if log.Attempt.Status != "transport-succeeded" {
			return ManagerStageResult{}, errors.New("original Manager terminal cannot be recovered; no replay")
		}
	} else {
		result, err = readStageResult(j.store, *j.report, log)
		if err != nil {
			return ManagerStageResult{}, err
		}
	}
	data, err := readStageEvidence(j.store, j.report.ID, log.Attempt.RequestRef)
	if err != nil {
		return ManagerStageResult{}, err
	}
	var original agentexec.Request
	if err := json.Unmarshal(data, &original); err != nil {
		return ManagerStageResult{}, err
	}
	if err := validateManagerStageEnvelope(result, original, spec.Validation.ConfigFingerprint); err != nil {
		return ManagerStageResult{}, err
	}
	if err := host.ValidateManagerStage(spec.Validation); err != nil {
		return ManagerStageResult{}, err
	}
	if err := j.reconcileTerminalReservation(log); err != nil {
		return ManagerStageResult{}, err
	}
	out, err := j.result(log, result)
	return out, errors.Join(err, managerStageAttachability(*j.report, frozenManagerStageBudget(effectiveManagerStageBudget(*j.report), spec.Limits)))
}

func CompleteManagerStage(root, runID, attemptID, expectedStateDigest string, validation ManagerStageValidation) (ManagerStageResult, error) {
	store, err := newManagerStageStore(root)
	if err != nil {
		return ManagerStageResult{}, err
	}
	unlock, err := store.lock()
	if err != nil {
		return ManagerStageResult{}, err
	}
	defer unlock()
	report, err := store.readLatestState(runID)
	if err != nil {
		return ManagerStageResult{}, err
	}
	if report.ManagerStageBinding == nil || report.Digest != expectedStateDigest {
		return ManagerStageResult{}, errors.New("Manager completion state changed")
	}
	index := -1
	for i, log := range report.Invocations {
		if log.Attempt != nil && log.Attempt.ID == attemptID && log.Attempt.ParentAttemptID == "" {
			index = i
			break
		}
	}
	if index < 0 {
		return ManagerStageResult{}, errors.New("Manager completion attempt is absent")
	}
	log := report.Invocations[index]
	result, err := readStageResult(store, report, log)
	if err != nil {
		return ManagerStageResult{}, err
	}
	if validation.Accepted {
		if err := managerStageAttachability(report, effectiveManagerStageBudget(report)); err != nil {
			return ManagerStageResult{}, err
		}
		if log.Attempt.Status == "accepted" {
			if log.Attempt.ValidationDigest != validation.TypedResultDigest {
				return ManagerStageResult{}, errors.New("accepted typed Manager result is immutable")
			}
			return ManagerStageResult{RunID: runID, AttemptID: attemptID, StateDigest: report.Digest, Status: log.Attempt.Status, Invocation: log, Result: result}, nil
		}
		if log.Attempt.Status != "transport-succeeded" && log.Attempt.Status != "accepted" || !log.CostKnown || log.CostOverflow || validation.TypedResultDigest == "" || log.Attempt.ReportRef == nil {
			return ManagerStageResult{}, errors.New("Manager report has no attachable terminal evidence")
		}
		if log.Attempt.ReviewRequired && !passedManagerStageReview(report, log) {
			return ManagerStageResult{}, errors.New("Manager stage lacks its exact passing configured review")
		}
		attempt := *log.Attempt
		attempt.Status = "accepted"
		attempt.ValidationDigest = validation.TypedResultDigest
		log.Attempt = &attempt
	} else {
		if log.Attempt.Status == "accepted" || log.Attempt.Status == "started" || log.Attempt.Status == "uncertain" || log.Receipt.RunID == "" {
			return ManagerStageResult{}, errors.New("uncertain or accepted Manager attempt cannot be rewritten")
		}
		attempt := *log.Attempt
		attempt.Status = "rejected"
		attempt.SafeFailure = validation.SafeFailure
		log.Attempt = &attempt
	}
	report.Invocations[index] = log
	if err := persistState(store, &report); err != nil {
		return ManagerStageResult{}, err
	}
	current, err := store.readLatestState(runID)
	if err != nil {
		return ManagerStageResult{}, err
	}
	return ManagerStageResult{RunID: runID, AttemptID: attemptID, StateDigest: current.Digest, Status: log.Attempt.Status, Invocation: log, Result: result}, nil
}

func canonicalManagerStageRoot(root string) (string, error) {
	if root == "" {
		return "", errors.New("Manager audit root is empty")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("Manager audit root is not a directory")
	}
	return resolved, nil
}

func effectiveManagerStageBudget(report RunReport) ManagerStageBudget {
	if report.ManagerStageLimits == nil {
		return report.ManagerStageBinding.Budget
	}
	return frozenManagerStageBudget(report.ManagerStageBinding.Budget, *report.ManagerStageLimits)
}

func tightenManagerStageBudget(store *runStore, report *RunReport, current ManagerStageBudget) (ManagerStageBudget, error) {
	if err := validateManagerStageBudget(current); err != nil {
		return current, err
	}
	previous := effectiveManagerStageBudget(*report)
	next := frozenManagerStageBudget(previous, current)
	if report.ManagerStageLimits == nil || next != previous {
		report.ManagerStageLimits = &next
		if err := persistState(store, report); err != nil {
			return next, err
		}
	}
	return next, nil
}

func managerStageAttachability(report RunReport, budget ManagerStageBudget) error {
	starts, retries, elapsed, cost, unknown := managerStageUsage(report)
	if unknown || starts > budget.MaxStarts || retries > budget.MaxRetries || cost > budget.MaxCostMicros || elapsed > budget.MaxDuration {
		return errors.New("Manager terminal exceeds bounded session accounting")
	}
	return nil
}
