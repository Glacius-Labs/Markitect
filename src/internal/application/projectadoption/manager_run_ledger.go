package projectadoption

import (
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"math"
	"reflect"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
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

func loadManagerRunLedger(sourceRoot, sessionID string, owners ...*ManagerStageLease) (ManagerRunLedger, error) {
	store, release, err := borrowBrownfieldStore(sourceRoot, sessionID, owners...)
	if err != nil {
		return ManagerRunLedger{}, err
	}
	defer release()
	file, err := store.ReadLegacyBrownfieldManagerLedger(sessionID, managerRunMaxLedger)
	if err != nil {
		return ManagerRunLedger{}, err
	}
	if !file.Exists {
		empty := ManagerRunLedger{APIVersion: managerRunVersion, SessionID: sessionID, Events: []ManagerRunEvent{}}
		sealManagerRunLedger(&empty)
		return empty, nil
	}
	data := file.Bytes
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

func reflectTypeOfManagerProposalDraft() reflect.Type { return reflect.TypeOf(ManagerProposalDraft{}) }
func reflectTypeOfManagerIntegrationDraft() reflect.Type {
	return reflect.TypeOf(ManagerIntegrationDraft{})
}

// AcquireManagerStageLease holds the established per-session process-file
// lock through Runtime work, typed review and the existing Session CAS.
func AcquireManagerStageLease(sourceRoot, sessionID string) (*ManagerStageLease, error) {
	store, err := recordstore.Open(sourceRoot)
	if err != nil {
		return nil, err
	}
	if err = store.CheckWrite(recordstore.Brownfield); err != nil {
		store.Close()
		return nil, err
	}
	lease, err := store.Kernel(sessionID, managerRunLockName)
	if err != nil {
		store.Close()
		return nil, err
	}
	owner := &ManagerStageLease{store: store, lease: lease, sessionID: sessionID}
	ledger, err := loadManagerRunLedger(sourceRoot, sessionID, owner)
	if err == nil && (ledger.Budget != nil || len(ledger.Events) != 0) {
		err = errors.New("legacy Manager execution ledger blocks Runtime stage admission; historical bytes remain untouched")
	}
	if err != nil {
		return nil, errors.Join(err, lease.Close(), store.Close())
	}
	return owner, nil
}

// ManagerStageLease carries the same physical Store through Runtime work and
// Session CAS; a path check followed by a fresh Store open is not borrowing.
type ManagerStageLease struct {
	store     *recordstore.Store
	lease     *recordstore.Lease
	sessionID string
}

func (o *ManagerStageLease) Check() error {
	if o == nil || o.store == nil || o.lease == nil {
		return errors.New("Manager stage owner is missing")
	}
	return errors.Join(o.store.CheckIdentity(), o.lease.Check())
}
func (o *ManagerStageLease) Close() error {
	return errors.Join(o.lease.Close(), o.store.Close())
}

func borrowBrownfieldStore(sourceRoot, sessionID string, owners ...*ManagerStageLease) (*recordstore.Store, func() error, error) {
	if len(owners) > 1 {
		return nil, nil, errors.New("one Manager stage owner required")
	}
	if len(owners) == 1 && owners[0] != nil {
		owner := owners[0]
		if err := owner.Check(); err != nil {
			return nil, nil, err
		}
		if sessionID != owner.sessionID {
			return nil, nil, errors.New("Manager stage owner does not match source root and Session")
		}
		if err := owner.store.CheckRootBinding(sourceRoot); err != nil {
			return nil, nil, err
		}
		return owner.store, func() error { return nil }, nil
	}
	store, err := recordstore.Open(sourceRoot)
	if err != nil {
		return nil, nil, err
	}
	return store, store.Close, nil
}
