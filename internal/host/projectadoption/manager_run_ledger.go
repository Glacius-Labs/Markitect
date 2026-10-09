package projectadoption

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

func checkManagerRunBudget(ledger ManagerRunLedger, limits ManagerRunLimits, timeout time.Duration, retry bool) error {
	starts, retries, elapsed, cost, unknown := managerRunUsage(ledger)
	if unknown {
		return errors.New("prior manager attempt has unknown usage cost; session is blocked fail-closed")
	}
	if starts >= limits.MaxStarts {
		return errors.New("session-wide Manager start limit is exhausted")
	}
	if retry && retries >= limits.MaxRetries {
		return errors.New("session-wide Manager retry limit is exhausted")
	}
	if elapsed+timeout > limits.MaxDuration {
		return errors.New("session-wide Manager duration limit would be exceeded")
	}
	if cost >= limits.MaxCostMicros {
		return errors.New("session-wide Manager cost limit is exhausted")
	}
	return nil
}

func managerRunUsage(ledger ManagerRunLedger) (starts, retries int, elapsed time.Duration, cost int64, unknown bool) {
	started := map[string]ManagerRunEvent{}
	terminal := map[string]ManagerRunEvent{}
	for _, event := range ledger.Events {
		if event.Event == "started" {
			starts++
			started[event.AttemptID] = event
			if event.RetryOfAttemptID != "" {
				retries++
			}
		}
		if event.Event == "terminal" {
			terminal[event.AttemptID] = event
		}
	}
	for id, start := range started {
		result, exists := terminal[id]
		if !exists {
			unknown = true
			continue
		}
		if !result.CostKnown {
			unknown = true
		} else if result.EstimatedCostMicros > math.MaxInt64-cost {
			unknown = true
		} else {
			cost += result.EstimatedCostMicros
		}
		if result.FinishedAt.Before(start.StartedAt) {
			unknown = true
			continue
		}
		span := result.FinishedAt.Sub(start.StartedAt)
		if span > time.Duration(math.MaxInt64)-elapsed {
			unknown = true
		} else {
			elapsed += span
		}
	}
	return
}

func managerRunCost(usage *agentexec.Usage, limits ManagerRunLimits) (int64, bool) {
	if err := requireDistillationUsage(usage); err != nil {
		return 0, false
	}
	inCost, inErr := tokenCostMicros(*usage.InputTokens, limits.InputPriceMicrosPerMillion)
	outCost, outErr := tokenCostMicros(*usage.OutputTokens, limits.OutputPriceMicrosPerMillion)
	if inErr != nil || outErr != nil || inCost > math.MaxInt64-outCost {
		return 0, false
	}
	return inCost + outCost, true
}

func completedManagerStage(session BrownfieldSession, iterationID, phase string) bool {
	iteration, ok := findIteration(session, iterationID)
	if !ok {
		return false
	}
	if phase == ManagerRunPhasePropose {
		return iteration.Proposal != nil
	}
	return iteration.Integration != nil
}

func managerRunLimitsDigest(limits ManagerRunLimits) string {
	limits.TempParent, limits.PrivateLogDirectory = "", ""
	return digestValue(limits)
}

func limitsForDigest(limits ManagerRunLimits) ManagerRunLimits {
	limits.TempParent, limits.PrivateLogDirectory = "", ""
	return limits
}

func budgetProjection(limits ManagerRunLimits) ManagerRunLimits {
	limits.TempParent, limits.PrivateLogDirectory = "", ""
	return limits
}

func frozenManagerRunLimits(ledger ManagerRunLedger, current ManagerRunLimits) (ManagerRunLimits, error) {
	if len(ledger.Events) == 0 && ledger.Budget == nil {
		return current, nil
	}
	if ledger.Budget == nil {
		return ManagerRunLimits{}, errors.New("manager ledger has attempts but no immutable initial budget")
	}
	initial := *ledger.Budget
	if initial.MaxStarts <= 0 || initial.MaxRetries < 0 || initial.MaxDuration <= 0 || initial.MaxCostMicros <= 0 || initial.InputPriceMicrosPerMillion <= 0 || initial.OutputPriceMicrosPerMillion <= 0 {
		return ManagerRunLimits{}, errors.New("manager ledger initial budget is invalid")
	}
	if current.MaxStarts < initial.MaxStarts {
		initial.MaxStarts = current.MaxStarts
	}
	if current.MaxRetries < initial.MaxRetries {
		initial.MaxRetries = current.MaxRetries
	}
	if current.MaxDuration < initial.MaxDuration {
		initial.MaxDuration = current.MaxDuration
	}
	if current.MaxCostMicros < initial.MaxCostMicros {
		initial.MaxCostMicros = current.MaxCostMicros
	}
	// Runtime ceilings are session-wide. Token prices and per-invocation
	// process bounds are selected for each Manager call and are already bound
	// into that call's preview and sealed terminal estimate.
	initial.InputPriceMicrosPerMillion = current.InputPriceMicrosPerMillion
	initial.OutputPriceMicrosPerMillion = current.OutputPriceMicrosPerMillion
	initial.MaxTimeout = current.MaxTimeout
	initial.MaxStdoutBytes = current.MaxStdoutBytes
	initial.MaxStderrBytes = current.MaxStderrBytes
	initial.TempParent = current.TempParent
	initial.PrivateLogDirectory = current.PrivateLogDirectory
	return initial, nil
}

func managerRunCompletedResult(session BrownfieldSession, ledger ManagerRunLedger, iterationID, phase string) ManagerRunResult {
	iteration, _ := findIteration(session, iterationID)
	result := ManagerRunResult{Status: "already-completed", PriorSessionDigest: session.Digest, SessionDigest: session.Digest,
		IterationID: iterationID, Phase: phase, LedgerDigest: ledger.Digest}
	if phase == ManagerRunPhasePropose {
		result.Proposal = iteration.Proposal
	} else {
		result.Integration = iteration.Integration
	}
	if attempt, ok := latestManagerAttempt(ledger, iterationID, phase); ok {
		result = managerRunAttemptResult(session, ledger, attempt, "already-completed")
		result.Proposal = iteration.Proposal
		result.Integration = iteration.Integration
	}
	return result
}

func managerRunAttemptResult(session BrownfieldSession, ledger ManagerRunLedger, event ManagerRunEvent, status string) ManagerRunResult {
	return ManagerRunResult{Status: status, PriorSessionDigest: session.Digest, SessionDigest: session.Digest, IterationID: event.IterationID, Phase: event.Phase,
		AttemptID: event.AttemptID, AttemptStatus: event.Status, Execution: event.Execution, ExecutionDigest: event.ExecutionDigest, LedgerDigest: ledger.Digest,
		Proposal: event.Proposal, Integration: event.Integration}
}

func recoverManagerRunStage(sourceRoot, targetRoot string, session BrownfieldSession, ledger ManagerRunLedger, event ManagerRunEvent, preview ManagerRunPreview, invoker ManagerRunInvoker, config agentexec.Config) (ManagerRunResult, error) {
	result := managerRunAttemptResult(session, ledger, event, "stage-recovery")
	if event.Status != "succeeded" || event.SessionDigest != session.Digest || event.ContextDigest != preview.ContextDigest || event.RequestContractDigest != preview.RequestContractDigest || event.ConfigFingerprint != preview.ConfigFingerprint ||
		event.RuntimeFileDigest != preview.RuntimeFileDigest || event.LimitsDigest != preview.LimitsDigest || event.Execution == nil || event.ExecutionDigest != digestValue(*event.Execution) {
		return result, errors.New("prior successful Manager receipt cannot be recovered against the current session")
	}
	var next BrownfieldSession
	var err error
	if event.Phase == ManagerRunPhasePropose && event.Proposal != nil {
		next, err = RecordManagerProposal(session, event.IterationID, *event.Proposal)
	} else if event.Phase == ManagerRunPhaseIntegrate && event.Integration != nil {
		next, err = IntegrateManagerProposal(session, event.IterationID, event.ManagerID, *event.Integration)
	} else {
		return result, errors.New("prior Manager receipt has no recoverable typed stage report")
	}
	if err != nil {
		return result, fmt.Errorf("recover validated Manager stage: %w", err)
	}
	if err := revalidateManagerRunBases(sourceRoot, targetRoot, session.ID, session.Digest, preview.RuntimeFileDigest, invoker, config, preview.ConfigFingerprint); err != nil {
		return result, fmt.Errorf("revalidate Manager bases before receipt recovery: %w", err)
	}
	if _, err := WriteBrownfieldSession(sourceRoot, next, session.Digest); err != nil {
		return result, fmt.Errorf("persist recovered Manager stage: %w", err)
	}
	result.Status, result.SessionDigest = "recovered", next.Digest
	result.Proposal, result.Integration = event.Proposal, event.Integration
	return result, nil
}

func latestManagerAttempt(ledger ManagerRunLedger, iterationID, phase string) (ManagerRunEvent, bool) {
	var latest ManagerRunEvent
	found := false
	terminals := map[string]ManagerRunEvent{}
	for _, event := range ledger.Events {
		if event.IterationID != iterationID || event.Phase != phase {
			continue
		}
		if event.Event == "started" {
			if !found || event.Sequence > latest.Sequence {
				latest, found = event, true
				latest.Status = "uncertain"
			}
		}
		if event.Event == "terminal" {
			terminals[event.AttemptID] = event
			if !found || event.Sequence > latest.Sequence {
				latest, found = event, true
			}
		}
	}
	if found && latest.Event == "started" {
		if terminal, ok := terminals[latest.AttemptID]; ok {
			latest = terminal
		} else {
			latest.Status = "uncertain"
		}
	}
	return latest, found
}

func loadManagerRunLedger(dir, sessionID string) (ManagerRunLedger, error) {
	if err := ensureNoIncompleteAtomicWrite(dir, managerRunLedgerName); err != nil {
		return ManagerRunLedger{}, err
	}
	path := filepath.Join(dir, managerRunLedgerName)
	data, err := readRegularLedger(path)
	if errors.Is(err, os.ErrNotExist) {
		empty := ManagerRunLedger{APIVersion: managerRunVersion, SessionID: sessionID, Events: []ManagerRunEvent{}}
		sealManagerRunLedger(&empty)
		return empty, nil
	}
	if err != nil {
		return ManagerRunLedger{}, err
	}
	if len(data) > managerRunMaxLedger {
		return ManagerRunLedger{}, errors.New("manager run ledger exceeds the bounded size")
	}
	var ledger ManagerRunLedger
	if err := decodeClosedJSON(data, &ledger); err != nil {
		return ManagerRunLedger{}, fmt.Errorf("decode manager run ledger: %w", err)
	}
	if ledger.APIVersion != managerRunVersion || ledger.SessionID != sessionID || ledger.Events == nil || len(ledger.Events) > managerRunMaxEvents {
		return ManagerRunLedger{}, errors.New("manager run ledger identity or bounds are invalid")
	}
	copy := ledger
	copy.Digest = ""
	if ledger.Digest == "" || digestValue(copy) != ledger.Digest {
		return ManagerRunLedger{}, errors.New("manager run ledger digest mismatch")
	}
	for i, event := range ledger.Events {
		copyEvent := event
		copyEvent.Digest = ""
		if event.Sequence != i+1 || event.Digest == "" || digestValue(copyEvent) != event.Digest {
			return ManagerRunLedger{}, errors.New("manager run ledger event chain is invalid")
		}
	}
	return ledger, nil
}

func writeManagerRunLedger(dir string, ledger ManagerRunLedger) error {
	if err := ensureNoIncompleteAtomicWrite(dir, managerRunLedgerName); err != nil {
		return err
	}
	data, err := encodeClosedJSON(ledger)
	if err != nil {
		return err
	}
	temp := filepath.Join(dir, ".manager-runs.json.tmp")
	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(temp)
		return err
	}
	path := filepath.Join(dir, managerRunLedgerName)
	if err := atomicReplaceFile(temp, path); err != nil {
		return err
	}
	return nil
}

func acquireNamedLock(dir, name string) (func(), error) {
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("manager run lock directory must be a regular nonsymlink directory")
	}
	unlock, err := acquireProcessFileLock(filepath.Join(dir, name))
	if err != nil {
		return nil, fmt.Errorf("manager execution is already active: %w", err)
	}
	return unlock, nil
}

func sealManagerRunEvent(event *ManagerRunEvent) {
	copy := *event
	copy.Digest = ""
	event.Digest = digestValue(copy)
}
func sealManagerRunLedger(ledger *ManagerRunLedger) {
	copy := *ledger
	copy.Digest = ""
	ledger.Digest = digestValue(copy)
}

func newSessionRunID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func reflectTypeOfManagerProposalDraft() reflect.Type { return reflect.TypeOf(ManagerProposalDraft{}) }
func reflectTypeOfManagerIntegrationDraft() reflect.Type {
	return reflect.TypeOf(ManagerIntegrationDraft{})
}
