package projectrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

var (
	ErrRoleStartBudgetExceeded  = errors.New("runtime role-start budget exhausted")
	ErrRoleStartAlreadyReserved = errors.New("role-start request was already durably reserved")
)

const maxRoleStartBudget = 256

func cloneRoleStartReservation(reservation RoleStartReservation) RoleStartReservation {
	if reservation.HelperDelivery != nil {
		delivery := cloneHelperDelivery(*reservation.HelperDelivery)
		reservation.HelperDelivery = &delivery
	}
	return reservation
}

func cloneRoleStartDeliveries(report RunReport) RunReport {
	reservations := make([]RoleStartReservation, len(report.RoleStartReservations))
	for index, reservation := range report.RoleStartReservations {
		reservations[index] = cloneRoleStartReservation(reservation)
	}
	report.RoleStartReservations = reservations
	return report
}

// RoleStartBudget serializes request reservation and state updates. Snapshot
// and Persist callbacks belong to the coordinator and must read/update the
// durable RunReport; Persist must upsert by reservation Key and durably save it.
// The limit is the run's maxStarts, which recorded checks also consume.
// Generic native observations remain a lower bound when their accounting is
// partial; this cannot impose a provider-side hard cap on unobserved starts.
type RoleStartBudget struct {
	mu               sync.Mutex
	limit            int
	snapshot         func() RunReport
	persist          func(RoleStartReservation) error
	initialRootFloor int
	initialRootKeys  map[string]bool
}

type RoleStartPermit struct {
	budget *RoleStartBudget
	key    string
}

func NewRoleStartBudget(limit int, snapshot func() RunReport, persist func(RoleStartReservation) error) (*RoleStartBudget, error) {
	if limit < 1 || limit > maxRoleStartBudget || snapshot == nil || persist == nil {
		return nil, fmt.Errorf("role-start budget requires a limit in 1..%d and durable snapshot/persist callbacks", maxRoleStartBudget)
	}
	initial := cloneRoleStartDeliveries(snapshot())
	accounting, _ := reportStartAccounting(initial)
	rootKeys := map[string]bool{}
	for _, reservation := range initial.RoleStartReservations {
		if reservation.Kind == "root" && reservation.Request.RequestID != "" {
			rootKeys[reservation.Key] = true
		}
	}
	rootFloor := accounting.Roots
	if len(rootKeys) > rootFloor {
		rootFloor = len(rootKeys)
	}
	return &RoleStartBudget{limit: limit, snapshot: snapshot, persist: persist, initialRootFloor: rootFloor, initialRootKeys: rootKeys}, nil
}

// ReserveRoot durably reserves one manager, reviewer, or verifier root before
// its transport dispatch. Root attempts already reflected in task counters or
// InvocationLog records are deduplicated by taking the larger root count.
func (b *RoleStartBudget) ReserveRoot(ctx context.Context, taskID, phase string, request agentexec.RoleStartRequest) (*RoleStartPermit, error) {
	if ctx == nil || strings.TrimSpace(taskID) == "" || strings.TrimSpace(phase) == "" {
		return nil, errors.New("root role reservation requires context, task ID, and phase")
	}
	return b.reserve(ctx, RoleStartReservation{Key: rootReservationKey(taskID, phase, request), Kind: "root", TaskID: taskID, Phase: phase, Request: request})
}

// ReserveHelper satisfies HelperReserveFunc. It persists the Host request even
// when the global budget is exhausted, so denied and malformed helper attempts
// remain charged and auditable before argument validation or dispatch.
func (b *RoleStartBudget) ReserveHelper(ctx context.Context, attempt HelperStartAttempt) (HelperReservation, error) {
	request := attempt.Request
	if ctx == nil || request.RequestID == "" || request.ParentSessionID == "" || request.Role != "helper" {
		return nil, errors.New("helper role reservation lacks its stable Host request identity")
	}
	permit, err := b.reserve(ctx, RoleStartReservation{Key: helperReservationKey(request), Kind: "helper",
		ManagerID: attempt.ManagerID, Phase: attempt.Phase, ParentRunID: attempt.ParentRunID, Request: request})
	return permit, err
}

func (b *RoleStartBudget) reserve(ctx context.Context, record RoleStartReservation) (*RoleStartPermit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if record.Key == "" || record.Request.RequestID == "" {
		return nil, errors.New("role-start reservation requires a stable request ID")
	}
	report := cloneRoleStartDeliveries(b.snapshot())
	for _, existing := range report.RoleStartReservations {
		if existing.Key == record.Key {
			return &RoleStartPermit{budget: b, key: record.Key}, ErrRoleStartAlreadyReserved
		}
	}
	if err := ctx.Err(); err != nil {
		record.Request.State = "failed"
	} else if record.Request.State == "" {
		record.Request.State = "requested"
	}
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	}
	// Checks are not role starts, but they consume the run's shared start limit.
	if b.accounting(report).ObservedTotal+len(report.Checks) >= b.limit {
		record.Request.State = "failed"
		if err := b.persist(cloneRoleStartReservation(record)); err != nil {
			return nil, errors.Join(ErrRoleStartBudgetExceeded, fmt.Errorf("persist rejected role-start request: %w", err))
		}
		return &RoleStartPermit{budget: b, key: record.Key}, errors.Join(ErrRoleStartBudgetExceeded, ctx.Err())
	}
	if err := b.persist(cloneRoleStartReservation(record)); err != nil {
		return nil, fmt.Errorf("persist role-start reservation before dispatch: %w", err)
	}
	permit := &RoleStartPermit{budget: b, key: record.Key}
	if err := ctx.Err(); err != nil {
		return permit, err
	}
	return permit, nil
}

// AttachProtocolStart binds an adapter-generated child root to its already
// counted Host helper permit; it records identity without consuming quota.
func (p *RoleStartPermit) AttachProtocolStart(ctx context.Context, request agentexec.RoleStartRequest) error {
	if p == nil || p.budget == nil || request.RequestID == "" || request.Role == "" {
		return errors.New("protocol root cannot attach to an invalid role-start permit")
	}
	return p.budget.update(ctx, p.key, func(record *RoleStartReservation) error {
		if record.Kind != "helper" {
			return errors.New("only a Host helper permit can attach a child protocol root")
		}
		if record.ProtocolRequestID != "" && record.ProtocolRequestID != request.RequestID {
			return errors.New("helper permit is already bound to another child protocol root")
		}
		record.ProtocolRequestID = request.RequestID
		return nil
	})
}

// Update records the Host-owned request lifecycle. Child protocol root state
// is intentionally attached separately and never creates a second reservation.
func (p *RoleStartPermit) Update(ctx context.Context, request agentexec.RoleStartRequest) error {
	if p == nil || p.budget == nil || request.RequestID == "" {
		return errors.New("role-start lifecycle update has no reservation")
	}
	return p.budget.update(ctx, p.key, func(record *RoleStartReservation) error {
		if request.RequestID != record.Request.RequestID || request.ParentSessionID != record.Request.ParentSessionID {
			return errors.New("role-start lifecycle update changed its Host request identity")
		}
		record.Request = request
		if request.State != "completed" {
			record.HelperDelivery = nil
		}
		return nil
	})
}

// CompleteDelivery atomically records the terminal Host request and the
// validated, applied, and closed helper result on its original reservation.
func (p *RoleStartPermit) CompleteDelivery(ctx context.Context, request agentexec.RoleStartRequest, delivery HelperDelivery) error {
	if p == nil || p.budget == nil || request.RequestID == "" || request.ParentSessionID == "" || request.SessionID == "" || delivery.State != "applied-and-closed" || delivery.RequestID != request.RequestID || delivery.DeltaDigest == "" {
		return errors.New("helper delivery completion lacks its durable identity or applied state")
	}
	return p.budget.update(ctx, p.key, func(record *RoleStartReservation) error {
		if record.Kind != "helper" || request.RequestID != record.Request.RequestID || request.ParentSessionID != record.Request.ParentSessionID || request.State != "completed" {
			return errors.New("helper delivery completion changed its Host request identity or terminal state")
		}
		if record.ParentRunID == "" || record.ManagerID == "" || record.Phase == "" {
			return errors.New("helper delivery completion is missing its parent binding")
		}
		record.Request = request
		cloned := cloneHelperDelivery(delivery)
		record.HelperDelivery = &cloned
		return nil
	})
}

func (b *RoleStartBudget) update(ctx context.Context, key string, change func(*RoleStartReservation) error) error {
	if ctx == nil {
		return errors.New("role-start lifecycle update requires context")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	report := cloneRoleStartDeliveries(b.snapshot())
	for _, existing := range report.RoleStartReservations {
		if existing.Key != key {
			continue
		}
		updated := existing
		if err := change(&updated); err != nil {
			return err
		}
		updated.RecordedAt = time.Now().UTC()
		if err := b.persist(cloneRoleStartReservation(updated)); err != nil {
			return fmt.Errorf("persist role-start lifecycle update: %w", err)
		}
		return nil
	}
	return errors.New("role-start reservation is not present in the durable report")
}

// Accounting reports known roots and the union of observed nested requests and
// durable Host helper reservations. Partial native accounting stays partial.
func (b *RoleStartBudget) Accounting() StartAccounting {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.accounting(cloneRoleStartDeliveries(b.snapshot()))
}

func (b *RoleStartBudget) accounting(report RunReport) StartAccounting {
	observed, nested := reportStartAccounting(report)
	newRootIDs := map[string]bool{}
	for _, reservation := range report.RoleStartReservations {
		if reservation.Kind == "root" && reservation.Request.RequestID != "" && !b.initialRootKeys[reservation.Key] {
			newRootIDs[reservation.Key] = true
		}
	}
	roots := observed.Roots
	reservedRootFloor := b.initialRootFloor + len(newRootIDs)
	if reservedRootFloor > roots {
		roots = reservedRootFloor
	}
	nestedIDs := map[string]bool{}
	unkeyed := 0
	for _, request := range nested {
		key := startRequestIdentity(request)
		if key == "" {
			unkeyed++
		} else {
			nestedIDs[key] = true
		}
	}
	for _, reservation := range report.RoleStartReservations {
		if reservation.Kind == "helper" && reservation.Request.RequestID != "" {
			nestedIDs[startRequestIdentity(reservation.Request)] = true
		}
	}
	observed.ObservedNested = len(nestedIDs) + unkeyed
	observed.Roots = roots
	observed.ObservedTotal = roots + observed.ObservedNested
	return observed
}

func startRequestIdentity(request agentexec.RoleStartRequest) string {
	if request.RequestID == "" {
		return ""
	}
	return request.ParentSessionID + "\x00" + request.RequestID
}

func rootReservationKey(taskID, phase string, request agentexec.RoleStartRequest) string {
	if request.RequestID == "" {
		return ""
	}
	return "root\x00" + taskID + "\x00" + phase + "\x00" + request.RequestID
}

func helperReservationKey(request agentexec.RoleStartRequest) string {
	if request.RequestID == "" || request.ParentSessionID == "" {
		return ""
	}
	return "helper\x00" + request.ParentSessionID + "\x00" + request.RequestID
}

var _ HelperReservation = (*RoleStartPermit)(nil)
