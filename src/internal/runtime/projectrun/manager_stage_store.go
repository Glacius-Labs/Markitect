package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

type managerStageJournalKey struct{}

// Only journal mutation is serialized. Transport calls and the existing
// ordinary Manager batch remain concurrent. Budget locks precede this lock.
type managerStageJournal struct {
	mu     sync.Mutex
	store  *runStore
	report *RunReport
	budget *RoleStartBudget
}

func newManagerStageJournal(store *runStore, report *RunReport, limits projectconfig.Limits) (*managerStageJournal, error) {
	j := &managerStageJournal{store: store, report: report}
	budget, err := NewRoleStartBudget(limits.MaxStarts, j.snapshot, j.reserve)
	if err != nil {
		return nil, err
	}
	j.budget = budget
	return j, nil
}

func (j *managerStageJournal) snapshot() RunReport {
	j.mu.Lock()
	defer j.mu.Unlock()
	data, err := json.Marshal(j.report)
	if err != nil {
		panic("Runtime report cannot be snapshotted")
	}
	var copy RunReport
	if json.Unmarshal(data, &copy) != nil {
		panic("Runtime report snapshot cannot be read")
	}
	return copy
}

func (j *managerStageJournal) reserve(record RoleStartReservation) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i, prior := range j.report.RoleStartReservations {
		if prior.Key == record.Key {
			j.report.RoleStartReservations[i] = record
			return persistState(j.store, j.report)
		}
	}
	j.report.RoleStartReservations = append(j.report.RoleStartReservations, record)
	return persistState(j.store, j.report)
}

func journalFromContext(ctx context.Context) *managerStageJournal {
	journal, _ := ctx.Value(managerStageJournalKey{}).(*managerStageJournal)
	return journal
}

func withManagerStageJournal(ctx context.Context, journal *managerStageJournal) context.Context {
	return context.WithValue(ctx, managerStageJournalKey{}, journal)
}

func stageEvidencePath(store *runStore, id string, ref ManagerStageEvidenceRef) (string, error) {
	parts := strings.Split(ref.Path, "/")
	if len(parts) != 2 || parts[0] != "reports" || parts[1] == "" || filepath.Base(parts[1]) != parts[1] || strings.ContainsAny(parts[1], "\\:") {
		return "", errors.New("invalid Manager evidence reference")
	}
	dir, err := store.runDir(id)
	if err != nil {
		return "", err
	}
	reports, err := requireRealSubdirectory(dir, "reports", store)
	if err != nil {
		return "", err
	}
	path := filepath.Join(reports, parts[1])
	return path, nil
}

// stageEvidenceDirectoryParts preserves recordDirectoryOwner's lexical
// canonical-path checks while allowing the unheld read to keep its one
// ephemeral physical Store across the ordered List/List/Read operations.
func stageEvidenceDirectoryParts(absolute string, owner *runStore) (recordstore.Kind, string, string, bool, error) {
	slash := filepath.ToSlash(filepath.Clean(absolute))
	marker := "/" + recordstore.RootPath + "/"
	pos := strings.LastIndex(slash, marker)
	if pos < 0 {
		return "", "", "", false, nil
	}
	parts := strings.Split(slash[pos+len(marker):], "/")
	kind := recordstore.Kind(parts[0])
	id, role := "", ""
	if len(parts) > 1 {
		id = parts[1]
	}
	if len(parts) > 2 {
		role = strings.Join(parts[2:], "/")
	}
	root := filepath.FromSlash(slash[:pos])
	if !sameRecoveryPath(root, owner.root) {
		return kind, id, role, true, errors.New("runtime directory differs from owning root")
	}
	return kind, id, role, true, nil
}

// stageEvidenceRecordName preserves recordPathOwner's parse-before-owner
// ordering without opening a replacement physical Store.
func stageEvidenceRecordName(absolute string, owner *runStore) (recordstore.Name, error) {
	slash := filepath.ToSlash(filepath.Clean(absolute))
	marker := "/" + recordstore.RootPath + "/"
	pos := strings.LastIndex(slash, marker)
	if pos < 0 {
		return recordstore.Name{}, errors.New("runtime record outside canonical store")
	}
	root := filepath.FromSlash(slash[:pos])
	name, err := recordstore.Parse(slash[pos+1:])
	if err != nil {
		return name, err
	}
	if !sameRecoveryPath(root, owner.root) {
		return name, errors.New("runtime record differs from owning root")
	}
	return name, nil
}

func writeStageEvidence(store *runStore, id, attempt, suffix string, data []byte) (*ManagerStageEvidenceRef, error) {
	name := strings.TrimPrefix(rawContentDigest([]byte(attempt)), "sha256:") + suffix
	ref := ManagerStageEvidenceRef{Path: "reports/" + name, Digest: rawContentDigest(data)}
	path, err := stageEvidencePath(store, id, ref)
	if err != nil {
		return nil, err
	}
	if old, readErr := readRecordBytes(path, store); readErr == nil {
		if rawContentDigest(old) != ref.Digest {
			return nil, errors.New("immutable Manager evidence changed")
		}
		return &ref, nil
	} else if !errors.Is(readErr, errRuntimeRecordMissing) {
		return nil, readErr
	}
	if err := writeImmutable(path, data, store); err != nil {
		return nil, err
	}
	return &ref, nil
}

func readStageEvidence(store *runStore, id string, ref *ManagerStageEvidenceRef) ([]byte, error) {
	if ref == nil || !strings.HasPrefix(ref.Digest, "sha256:") {
		return nil, errors.New("Manager evidence reference is missing")
	}
	if store.storage == nil {
		parts := strings.Split(ref.Path, "/")
		if len(parts) != 2 || parts[0] != "reports" || parts[1] == "" || filepath.Base(parts[1]) != parts[1] || strings.ContainsAny(parts[1], "\\:") {
			return nil, errors.New("invalid Manager evidence reference")
		}
		if !validID(id) {
			return nil, fmt.Errorf("invalid run ID %q", id)
		}
		data, err := func() ([]byte, error) {
			physical, done, err := store.physical()
			if err != nil {
				return nil, err
			}
			defer done()
			if _, err := physical.List(recordstore.Run, id, ""); err != nil {
				return nil, err
			}

			directoryName := "reports"
			if directoryName == "" || filepath.Base(directoryName) != directoryName {
				return nil, fmt.Errorf("invalid runtime state directory %q", directoryName)
			}
			runPath := filepath.Join(store.base, id)
			reportsPath := filepath.Join(runPath, directoryName)
			kind, directoryID, role, canonical, err := stageEvidenceDirectoryParts(reportsPath, store)
			if err != nil {
				return nil, err
			}
			if canonical {
				if err := physical.CheckRootBinding(store.root); err != nil {
					return nil, err
				}
				if _, err := physical.List(kind, directoryID, role); err != nil {
					return nil, err
				}
			} else {
				info, err := os.Lstat(reportsPath)
				if err != nil {
					return nil, err
				}
				if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					return nil, fmt.Errorf("runtime state directory %s must be real", reportsPath)
				}
			}

			path := filepath.Join(reportsPath, parts[1])
			if !canonical {
				// recordPathOwner rejects this same outside path before opening a
				// replacement Store; retain that terminal error after the old Lstat.
				return nil, errors.New("runtime record outside canonical store")
			}
			name, err := stageEvidenceRecordName(path, store)
			if err != nil {
				return nil, err
			}
			if err := physical.CheckRootBinding(store.root); err != nil {
				return nil, err
			}
			file, err := physical.Read(name, 0)
			if err != nil {
				return nil, err
			}
			if !file.Exists {
				return nil, errRuntimeRecordMissing
			}
			return file.Bytes, nil
		}()
		if err != nil {
			return nil, err
		}
		if rawContentDigest(data) != ref.Digest {
			return nil, errors.New("Manager evidence byte digest differs")
		}
		return data, nil
	}
	parts := strings.Split(ref.Path, "/")
	if len(parts) != 2 || parts[0] != "reports" || parts[1] == "" || filepath.Base(parts[1]) != parts[1] || strings.ContainsAny(parts[1], "\\:") {
		return nil, errors.New("invalid Manager evidence reference")
	}
	if !validID(id) {
		return nil, fmt.Errorf("invalid run ID %q", id)
	}
	baseRelativeToRoot, err := filepath.Rel(store.root, store.base)
	if err != nil || filepath.ToSlash(baseRelativeToRoot) != RunsPath {
		return nil, errors.New("runtime directory differs from owning root")
	}
	name := recordstore.Name{Kind: recordstore.Run, ID: id, Role: ref.Path}
	if _, err := name.Path(); err != nil {
		return nil, err
	}
	physical, done, err := store.physical()
	if err != nil {
		return nil, err
	}
	defer done()
	file, err := physical.ReadWithDirectoryInspection(name, 0)
	if err != nil {
		return nil, err
	}
	if !file.Exists {
		return nil, errRuntimeRecordMissing
	}
	data := file.Bytes
	if rawContentDigest(data) != ref.Digest {
		return nil, errors.New("Manager evidence byte digest differs")
	}
	return data, nil
}

func (j *managerStageJournal) start(log InvocationLog, request agentexec.Request) error {
	if log.Attempt == nil || log.Attempt.ID == "" {
		return errors.New("Manager start lacks its stable attempt")
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, prior := range j.report.Invocations {
		if prior.Attempt != nil && prior.Attempt.ID == log.Attempt.ID {
			return errors.New("Manager attempt already exists; replay prohibited")
		}
	}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	ref, err := writeStageEvidence(j.store, j.report.ID, log.Attempt.ID, ".request.json", data)
	if err != nil {
		return err
	}
	log.Attempt.RequestRef = ref
	j.report.Invocations = append(j.report.Invocations, log)
	return persistState(j.store, j.report)
}

func (j *managerStageJournal) terminal(log InvocationLog, result agentexec.RunResult, invokeErr error) (InvocationLog, error) {
	if log.Attempt == nil {
		return log, errors.New("Manager terminal lacks its original attempt")
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	index := -1
	for i, prior := range j.report.Invocations {
		if prior.Attempt != nil && prior.Attempt.ID == log.Attempt.ID {
			index = i
			break
		}
	}
	if index < 0 {
		return log, errors.New("original Manager start is absent")
	}
	prior := j.report.Invocations[index]
	if result.Receipt.RunID == "" && prior.Receipt.RunID != "" {
		return prior, errors.New("original Manager receipt retained; recovery remains unresolved")
	}
	if prior.Attempt.Status != "started" && prior.Attempt.Status != "uncertain" {
		if prior.Receipt.RunID != result.Receipt.RunID {
			return log, errors.New("terminal Manager receipt identity changed")
		}
		return prior, nil
	}
	attempt := *prior.Attempt
	attempt.FinishedAt = time.Now().UTC()
	attempt.Status = "transport-failed"
	if result.Receipt.RunID == "" || result.Receipt.Lifecycle != nil && !workspaceInvocationTerminal(result.Receipt.Lifecycle) {
		attempt.Status = "uncertain"
	}
	if invokeErr == nil && attempt.Status != "uncertain" {
		attempt.Status = "transport-succeeded"
	}
	if invokeErr != nil {
		attempt.SafeFailure = "invocation-failed"
	}
	if len(result.Response.ReportJSON) > 0 {
		if prior.Attempt.ReportRef != nil {
			original, err := readStageEvidence(j.store, j.report.ID, prior.Attempt.ReportRef)
			if err != nil {
				return log, err
			}
			if !sameOpaqueManagerReport(original, result.Response.ReportJSON) {
				return log, errors.New("original Manager report differs on recovery")
			}
			result.Response.ReportJSON = append(json.RawMessage(nil), original...)
		}
		ref, err := writeStageEvidence(j.store, j.report.ID, attempt.ID, ".report.json", append([]byte(nil), result.Response.ReportJSON...))
		if err != nil {
			return log, err
		}
		attempt.ReportRef = ref
	}
	// Response is protected protocol output evidence. The metadata row exposes
	// only its digest/reference; no invocation configuration is encoded here.
	output := result.Response
	output.ReportJSON = nil
	encoded, err := json.Marshal(output)
	if err != nil {
		return log, err
	}
	ref, err := writeStageEvidence(j.store, j.report.ID, attempt.ID, ".response-"+strings.TrimPrefix(rawContentDigest(encoded), "sha256:")+".json", encoded)
	if err != nil {
		return log, err
	}
	attempt.ResponseRef = ref
	log.Attempt = &attempt
	log.Receipt = result.Receipt
	log.ReportID = result.Receipt.RunID
	log.Outcome = result.Receipt.Outcome
	if result.Receipt.RunID == "" {
		log.Outcome = "uncertain"
	}
	if log.Role == agentexec.RoleExecutor && result.Receipt.InputDigest != "" {
		log.InputDigest = result.Receipt.InputDigest
	}
	j.report.Invocations[index] = log
	if err := persistState(j.store, j.report); err != nil {
		return log, err
	}
	return log, nil
}

func readStageResult(store *runStore, report RunReport, log InvocationLog) (agentexec.RunResult, error) {
	if log.Attempt == nil || log.Attempt.ResponseRef == nil {
		return agentexec.RunResult{}, errors.New("Manager terminal output is unavailable")
	}
	data, err := readStageEvidence(store, report.ID, log.Attempt.ResponseRef)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	var response agentexec.Response
	if err := json.Unmarshal(data, &response); err != nil {
		return agentexec.RunResult{}, err
	}
	if log.Attempt.ReportRef != nil {
		data, err = readStageEvidence(store, report.ID, log.Attempt.ReportRef)
		if err != nil {
			return agentexec.RunResult{}, err
		}
		response.ReportJSON = append(json.RawMessage(nil), data...)
	}
	if response.RunID != log.Receipt.RunID || response.InputDigest != log.Receipt.InputDigest {
		return agentexec.RunResult{}, errors.New("Manager output differs from its terminal receipt")
	}
	return agentexec.RunResult{Response: response, Receipt: log.Receipt}, nil
}

func validateManagerStageState(store *runStore, state, prior RunReport) error {
	binding := state.ManagerStageBinding
	if binding != nil {
		if state.PlanID != "" || state.Candidate.ID != "" || len(state.Tasks) > 0 || binding.SessionID == "" || managerStageRunID(binding.SessionID) != state.ID {
			return errors.New("planless Manager stage has invalid authority")
		}
		if err := validateManagerStageBudget(binding.Budget); err != nil {
			return err
		}
		if state.ManagerStageLimits == nil || validateManagerStageBudget(*state.ManagerStageLimits) != nil || frozenManagerStageBudget(binding.Budget, *state.ManagerStageLimits) != *state.ManagerStageLimits {
			return errors.New("Manager effective ceilings are invalid")
		}
		if prior.ManagerStageBinding != nil {
			if frozenManagerStageBudget(effectiveManagerStageBudget(prior), *state.ManagerStageLimits) != *state.ManagerStageLimits {
				return errors.New("Manager effective ceilings were refilled")
			}
			a, _ := digest(prior.ManagerStageBinding)
			b, _ := digest(binding)
			if a != b {
				return errors.New("Manager session authority or frozen ceiling changed")
			}
		} else if prior.ID != "" {
			return errors.New("ordinary run cannot become a Manager stage")
		}
	} else if prior.ManagerStageBinding != nil {
		return errors.New("Manager stage cannot become an ordinary run")
	}
	seen := map[string]bool{}
	old := map[string]InvocationLog{}
	for _, log := range prior.Invocations {
		if log.Attempt != nil {
			old[log.Attempt.ID] = log
		}
	}
	for _, log := range state.Invocations {
		a := log.Attempt
		if a == nil {
			if binding != nil {
				return errors.New("Manager stage contains an unbound invocation")
			}
			continue
		}
		if a.ID == "" || a.StageKey == "" || a.RequestDigest == "" || a.StartedAt.IsZero() || seen[a.ID] {
			return errors.New("Manager attempt identity is invalid or repeated")
		}
		seen[a.ID] = true
		switch a.Status {
		case "started", "uncertain", "transport-succeeded", "transport-failed", "accepted", "rejected":
		default:
			return errors.New("Manager attempt status is invalid")
		}
		if previous, ok := old[a.ID]; ok {
			before := *previous.Attempt
			after := *a
			before.FinishedAt = after.FinishedAt
			before.Status = after.Status
			before.SafeFailure = after.SafeFailure
			before.ValidationDigest = after.ValidationDigest
			before.ResponseRef = after.ResponseRef
			before.ReportRef = after.ReportRef
			x, _ := digest(before)
			y, _ := digest(after)
			if x != y {
				return errors.New("immutable Manager attempt bindings changed")
			}
			if previous.Receipt.RunID != "" && previous.Receipt.RunID != log.Receipt.RunID {
				return errors.New("Manager receipt changed")
			}
			if previous.Attempt.Status == "accepted" && (a.Status != "accepted" || a.ValidationDigest != previous.Attempt.ValidationDigest) {
				return errors.New("accepted Manager attempt was rewritten")
			}
			delete(old, a.ID)
		}
		if a.RequestRef != nil {
			if _, err := readStageEvidence(store, state.ID, a.RequestRef); err != nil {
				return err
			}
		}
		if a.ResponseRef != nil {
			if _, err := readStageEvidence(store, state.ID, a.ResponseRef); err != nil {
				return err
			}
		}
		if a.ReportRef != nil {
			if _, err := readStageEvidence(store, state.ID, a.ReportRef); err != nil {
				return err
			}
		}
		if a.Status != "started" && a.Status != "uncertain" && (a.FinishedAt.IsZero() || a.FinishedAt.Before(a.StartedAt)) {
			return errors.New("Manager terminal duration is invalid")
		}
		if binding != nil && a.IterationID == "" {
			return errors.New("Manager stage iteration is absent")
		}
	}
	if len(old) > 0 {
		return fmt.Errorf("durable Manager attempts were removed")
	}
	return validateStageReviewRecords(store, state)
}

// invokeManagerAttempt is the shared attempt lifecycle for ordinary manager,
// independent reviewer and Brownfield calls. Workflow adapters own requests,
// dispatch options and typed acceptance. No configuration is persisted.
func invokeManagerAttempt(ctx context.Context, j *managerStageJournal, log InvocationLog, request agentexec.Request, agent projectconfig.Agent, dispatch func(context.Context) (agentexec.RunResult, error)) (agentexec.RunResult, InvocationLog, error) {
	if j == nil || log.Attempt == nil {
		return agentexec.RunResult{}, log, errors.New("Manager invocation requires its durable Runtime journal")
	}
	if err := ctx.Err(); err != nil {
		return agentexec.RunResult{}, log, err
	}
	if _, _, err := agentexec.PrepareInvocation(request); err != nil {
		return agentexec.RunResult{}, log, errors.New("Manager request violates the bounded protocol")
	}
	roleRequest := agentexec.RoleStartRequest{RequestID: log.Attempt.ID, ParentSessionID: j.report.ID, Role: log.Role, State: "requested"}
	var permit *RoleStartPermit
	var err error
	if j.report.ManagerStageBinding == nil && log.Role == agentexec.RoleExecutor && (log.Phase == "work" || log.Phase == "integrate") {
		permit, err = j.budget.ReserveRecordedTaskRoot(ctx, log.TaskID, log.Phase, roleRequest)
	} else {
		permit, err = j.budget.ReserveRoot(ctx, log.TaskID, log.Phase, roleRequest)
	}
	if err != nil {
		return agentexec.RunResult{}, log, err
	}
	if err = j.start(log, request); err != nil {
		return agentexec.RunResult{}, log, fmt.Errorf("retain Manager start after role reservation: %w", err)
	}
	result, invokeErr := dispatch(ctx)
	cost, known, overflow := estimateAgentCost(result.Receipt.Usage, agent)
	if j.report.ManagerStageBinding != nil {
		cost, known, overflow = managerStageCost(result.Receipt.Usage, log.Attempt.Bounds)
	}
	log.CostMicros, log.CostKnown, log.CostOverflow = cost, known, overflow
	if overflow {
		invokeErr = errors.Join(invokeErr, errors.New("Manager cost overflow"))
	}
	if !known && requiresReportedUsage(agent) {
		invokeErr = errors.Join(invokeErr, errors.New("Manager reported usage is missing"))
	}
	terminal, persistErr := j.terminal(log, result, invokeErr)
	if persistErr != nil {
		return result, terminal, errors.Join(invokeErr, fmt.Errorf("retain Manager terminal receipt: %w", persistErr))
	}
	roleRequest.State = "completed"
	if terminal.Attempt.Status == "uncertain" {
		roleRequest.State = "unknown"
	}
	if terminal.Attempt.Status == "transport-failed" {
		roleRequest.State = "failed"
	}
	if err := permit.Update(context.Background(), roleRequest); err != nil {
		return result, terminal, errors.Join(invokeErr, err)
	}
	return result, terminal, invokeErr
}

func (j *managerStageJournal) result(log InvocationLog, result agentexec.RunResult) (ManagerStageResult, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	state, err := j.store.readLatestState(j.report.ID)
	if err != nil {
		return ManagerStageResult{}, err
	}
	return ManagerStageResult{RunID: state.ID, AttemptID: log.Attempt.ID, StateDigest: state.Digest, Status: log.Attempt.Status, Invocation: log, Result: result}, nil
}

// Stage storage uses the existing runStore primitives in the source session
// repository. Runtime configuration belongs to the independently bound target;
// ordinary newRunStore still requires its normal runtime-file admission.
func newManagerStageStore(root string) (*runStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	physical, err := recordstore.Open(abs)
	if err != nil {
		return nil, err
	}
	defer physical.Close()
	if err := physical.CheckCutover(recordstore.Run); err != nil {
		return nil, err
	}
	return &runStore{root: abs, base: filepath.Join(abs, filepath.FromSlash(RunsPath)), identity: physical.Identity()}, nil
}

// A terminal row may have been flushed just before its reservation update.
// Reconcile the SAME permit idempotently; never reserve or dispatch a new role.
func (j *managerStageJournal) reconcileTerminalReservation(log InvocationLog) error {
	if log.Attempt == nil {
		return errors.New("terminal reservation lacks its attempt")
	}
	snapshot := j.snapshot()
	request := agentexec.RoleStartRequest{RequestID: log.Attempt.ID, ParentSessionID: snapshot.ID, Role: log.Role}
	key := rootReservationKey(log.TaskID, log.Phase, request)
	found := false
	for _, r := range snapshot.RoleStartReservations {
		if r.Key == key {
			found = true
			request = r.Request
			break
		}
	}
	if !found {
		return errors.New("original role reservation is absent; no replacement permitted")
	}
	request.Role = log.Role
	request.State = "completed"
	if log.Attempt.Status == "started" || log.Attempt.Status == "uncertain" {
		request.State = "unknown"
	}
	if log.Attempt.Status == "transport-failed" || log.Attempt.Status == "rejected" {
		request.State = "failed"
	}
	for _, r := range snapshot.RoleStartReservations {
		if r.Key == key && r.Request.State == request.State && r.Request.Role == request.Role {
			return nil
		}
	}
	permit := RoleStartPermit{budget: j.budget, key: key}
	return permit.Update(context.Background(), request)
}

// Ordinary adapters retain their request builders, transport options and
// combined-floor pricing. Only attempt authority and role admission are shared.
func ordinaryManagerAttemptLog(journal *managerStageJournal, invoker Invoker, config agentexec.Config, agent projectconfig.Agent, task ManagerTask, phase, id string, request agentexec.Request) (InvocationLog, error) {
	if journal == nil || journal.report.ManagerStageBinding != nil {
		return InvocationLog{}, errors.New("ordinary Manager requires its bound Runtime journal")
	}
	id = ordinaryRunAttemptID(*journal.report, id)
	requestDigest, err := digest(request)
	if err != nil {
		return InvocationLog{}, err
	}
	fingerprint, err := fingerprintForRequest(invoker, config, request)
	if err != nil {
		return InvocationLog{}, err
	}
	return InvocationLog{TaskID: task.ID, Role: agentexec.RoleExecutor, Phase: phase, InputDigest: requestDigest, Outcome: "started", Attempt: &ManagerStageAttempt{ID: id, StageKey: journal.report.ID + "/" + id, IterationID: task.ID, ManagerID: task.ManagerID, Phase: phase, ContextDigest: rawContentDigest(request.Context), RequestDigest: requestDigest, ConfigFingerprint: fingerprint, Bounds: managerStageBounds(agent, config), StartedAt: time.Now().UTC(), Status: "started"}}, nil
}

func invocationAttemptIndex(rows []InvocationLog, log InvocationLog) int {
	if log.Attempt == nil {
		return -1
	}
	for i, row := range rows {
		if row.Attempt != nil && row.Attempt.ID == log.Attempt.ID {
			return i
		}
	}
	return -1
}

func sameOpaqueManagerReport(first, second []byte) bool {
	decode := func(data []byte) ([]byte, error) {
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		return json.Marshal(value)
	}
	a, err := decode(first)
	if err != nil {
		return false
	}
	b, err := decode(second)
	return err == nil && string(a) == string(b)
}

func recoverOrdinaryManagerAttempt(journal *managerStageJournal, attemptID string, request agentexec.Request, result agentexec.RunResult, agent projectconfig.Agent, recoverErr error) (InvocationLog, error) {
	if journal == nil {
		return InvocationLog{}, errors.New("recovery requires the original Runtime journal")
	}
	index := invocationAttemptIndex(journal.report.Invocations, InvocationLog{Attempt: &ManagerStageAttempt{ID: attemptID}})
	if index < 0 {
		return InvocationLog{}, errors.New("original durable Manager attempt is absent; no replay")
	}
	log := journal.report.Invocations[index]
	requestDigest, err := digest(request)
	if err != nil || log.Attempt.RequestDigest != requestDigest {
		return log, errors.New("original Manager request changed; no replay")
	}
	cost, known, overflow := estimateAgentCost(result.Receipt.Usage, agent)
	log.CostMicros, log.CostKnown, log.CostOverflow = cost, known, overflow
	log, err = journal.terminal(log, result, recoverErr)
	if err != nil {
		return log, err
	}
	return log, journal.reconcileTerminalReservation(log)
}

// A lost plain-process caller cannot establish terminal output. Preserve its
// exact charged attempt and protected request; reconcile no replacement ticket.
func (j *managerStageJournal) markOrdinaryProcessAttemptUncertain(task ManagerTask, phase string) error {
	j.mu.Lock()
	log, index, err := j.originalUnfinishedProcessAttempt(task, phase)
	if err == nil && log.Attempt.Status == "started" {
		attempt := *log.Attempt
		attempt.Status = "uncertain"
		log.Attempt = &attempt
		j.report.Invocations[index] = log
		err = persistState(j.store, j.report)
	}
	j.mu.Unlock()
	if err != nil {
		return err
	}
	// Budget mutation takes the budget lock before the journal lock.
	return j.reconcileTerminalReservation(log)
}
func (j *managerStageJournal) originalUnfinishedProcessAttempt(task ManagerTask, phase string) (InvocationLog, int, error) {
	fail := func() (InvocationLog, int, error) {
		return InvocationLog{}, -1, errors.New("original process attempt/request/reservation authority is missing, ambiguous or incompatible; no replay")
	}
	if j.report.ManagerStageBinding != nil || j.report.ID == "" || task.ID == "" || task.ManagerID == "" || task.RepairPhase != phase || phase != "work" && phase != "integrate" {
		return fail()
	}
	id := ordinaryRunAttemptID(*j.report, nativeTaskAttemptID(task, phase))
	index, count := -1, 0
	for i, row := range j.report.Invocations {
		if row.Attempt != nil && row.Attempt.ID == id {
			index = i
			count++
		}
	}
	if count != 1 {
		return fail()
	}
	log := j.report.Invocations[index]
	a := log.Attempt
	receipt, _ := digest(log.Receipt)
	emptyReceipt, _ := digest(agentexec.Receipt{})
	if log.TaskID != task.ID || log.Phase != phase || log.Role != agentexec.RoleExecutor || log.InputDigest != a.RequestDigest || log.ReportID != "" || receipt != emptyReceipt || a.ManagerID != task.ManagerID || a.IterationID != task.ID || a.Phase != phase || a.StageKey != j.report.ID+"/"+id || a.ParentAttemptID != "" || a.Status != "started" && a.Status != "uncertain" || !a.FinishedAt.IsZero() || a.ResponseRef != nil || a.ReportRef != nil || a.ValidationDigest != "" {
		return fail()
	}
	input, err := readStageEvidence(j.store, j.report.ID, a.RequestRef)
	if err != nil {
		return InvocationLog{}, -1, err
	}
	var original agentexec.Request
	if err := json.Unmarshal(input, &original); err != nil {
		return InvocationLog{}, -1, err
	}
	requestDigest, err := digest(original)
	if err != nil || requestDigest != a.RequestDigest || original.Role != log.Role || rawContentDigest(original.Context) != a.ContextDigest {
		return fail()
	}
	request := agentexec.RoleStartRequest{RequestID: id, ParentSessionID: j.report.ID, Role: log.Role}
	key := rootReservationKey(task.ID, phase, request)
	count = 0
	for _, reservation := range j.report.RoleStartReservations {
		if reservation.Key != key && reservation.Request.RequestID != id {
			continue
		}
		count++
		r := reservation.Request
		if reservation.Key != key || reservation.Kind != "root" || reservation.TaskID != task.ID || reservation.Phase != phase || r.RequestID != id || r.ParentSessionID != j.report.ID || r.Role != log.Role || r.State != "requested" && r.State != "unknown" {
			return fail()
		}
	}
	if count != 1 {
		return fail()
	}
	return log, index, nil
}
