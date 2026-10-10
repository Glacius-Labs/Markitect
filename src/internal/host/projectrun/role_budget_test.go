package projectrun

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

type roleBudgetMemoryStore struct {
	mu     sync.Mutex
	report RunReport
	fail   bool
	writes int
}

func (s *roleBudgetMemoryStore) snapshot() RunReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := s.report
	copy.RoleStartReservations = append([]RoleStartReservation(nil), s.report.RoleStartReservations...)
	copy.Tasks = append([]ManagerTask(nil), s.report.Tasks...)
	copy.Invocations = append([]InvocationLog(nil), s.report.Invocations...)
	return copy
}

func (s *roleBudgetMemoryStore) persist(reservation RoleStartReservation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("disk unavailable")
	}
	for i := range s.report.RoleStartReservations {
		if s.report.RoleStartReservations[i].Key == reservation.Key {
			s.report.RoleStartReservations[i] = reservation
			s.writes++
			return nil
		}
	}
	s.report.RoleStartReservations = append(s.report.RoleStartReservations, reservation)
	s.writes++
	return nil
}

func newRoleBudgetForTest(t *testing.T, store *roleBudgetMemoryStore, limit int) *RoleStartBudget {
	t.Helper()
	budget, err := NewRoleStartBudget(limit, store.snapshot, store.persist)
	if err != nil {
		t.Fatal(err)
	}
	return budget
}

func TestRoleStartBudgetCountsDurableRootsAndFailedHelperRequests(t *testing.T) {
	store := &roleBudgetMemoryStore{report: RunReport{Tasks: []ManagerTask{{ID: "orders", WorkAttempts: 1}}}}
	budget := newRoleBudgetForTest(t, store, 3)
	root, err := budget.ReserveRoot(context.Background(), "review-1", "review", agentexec.RoleStartRequest{RequestID: "review-root", Role: "reviewer"})
	if err != nil || root == nil {
		t.Fatalf("root start reservation failed: permit=%v err=%v", root, err)
	}
	malformed := HelperStartAttempt{Request: agentexec.RoleStartRequest{RequestID: "helper-bad", ParentSessionID: "parent-session", Role: "helper", State: "requested"},
		ManagerID: "orders", Phase: "work", ParentRunID: "parent-run"}
	helper, err := budget.ReserveHelper(context.Background(), malformed)
	if err != nil || helper == nil {
		t.Fatalf("malformed helper attempt did not receive a durable reservation: permit=%v err=%v", helper, err)
	}
	if err := helper.Update(context.Background(), agentexec.RoleStartRequest{RequestID: "helper-bad", ParentSessionID: "parent-session", Role: "helper", State: "failed"}); err != nil {
		t.Fatal(err)
	}
	if got := budget.Accounting(); got.Roots != 2 || got.ObservedNested != 1 || got.ObservedTotal != 3 {
		t.Fatalf("task-counter root, review root, and failed helper were not counted once: %+v", got)
	}
	_, err = budget.ReserveHelper(context.Background(), HelperStartAttempt{Request: agentexec.RoleStartRequest{RequestID: "helper-over", ParentSessionID: "parent-session", Role: "helper"}})
	if !errors.Is(err, ErrRoleStartBudgetExceeded) {
		t.Fatalf("exhausted budget allowed another helper dispatch: %v", err)
	}
	if got := len(store.snapshot().RoleStartReservations); got != 3 {
		t.Fatalf("denied helper request was not retained durably: records=%d", got)
	}
}

func TestRoleStartBudgetAttachesChildRootWithoutAnotherCharge(t *testing.T) {
	store := &roleBudgetMemoryStore{}
	budget := newRoleBudgetForTest(t, store, 1)
	request := agentexec.RoleStartRequest{RequestID: "helper-call-1", ParentSessionID: "parent-session", Role: "helper", State: "requested"}
	permit, err := budget.ReserveHelper(context.Background(), HelperStartAttempt{Request: request, ParentRunID: "parent-run", ManagerID: "orders", Phase: "work"})
	if err != nil {
		t.Fatal(err)
	}
	childRoot := agentexec.RoleStartRequest{RequestID: "child-executor-root", Role: agentexec.RoleExecutor, State: "started"}
	if err := permit.AttachProtocolStart(context.Background(), childRoot); err != nil {
		t.Fatal(err)
	}
	request.State = "completed"
	if err := permit.Update(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	// The same Host helper request is now also visible in a parent lifecycle.
	snapshot := store.snapshot()
	snapshot.Tasks = []ManagerTask{{ID: "orders", WorkAttempts: 1}}
	snapshot.Invocations = []InvocationLog{{TaskID: "orders", Role: agentexec.RoleExecutor, Phase: "work", Receipt: agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{
		Provider: "codex-app-server", SessionID: "parent-session", Accounting: "partial",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: "parent-root", Role: "executor", State: "completed"}, request},
	}}}}
	store.mu.Lock()
	store.report = snapshot
	store.mu.Unlock()
	accounting := budget.Accounting()
	if accounting.Roots != 1 || accounting.ObservedNested != 1 || accounting.ObservedTotal != 2 {
		t.Fatalf("Host permit, parent lifecycle request, and attached child root double charged: %+v", accounting)
	}
	stored := store.snapshot().RoleStartReservations[0]
	if stored.ProtocolRequestID != childRoot.RequestID || stored.Request.State != "completed" {
		t.Fatalf("helper lifecycle/child binding was not persisted on the same permit: %+v", stored)
	}
}

func TestRoleStartBudgetCopiesCompletedHelperDelivery(t *testing.T) {
	store := &roleBudgetMemoryStore{}
	budget := newRoleBudgetForTest(t, store, 3)
	attempt := HelperStartAttempt{Request: agentexec.RoleStartRequest{RequestID: "helper-copy", ParentSessionID: "parent-session", SessionID: "helper-session", Role: "helper", State: "requested"},
		ManagerID: "orders", Phase: "work", ParentRunID: "parent-run"}
	reservation, err := budget.ReserveHelper(context.Background(), attempt)
	if err != nil {
		t.Fatal(err)
	}
	delivery := HelperDelivery{State: "applied-and-closed", RequestID: "helper-copy", DeltaDigest: "sha256:delta",
		RequestedPaths: []string{"tests/"}, Changes: []HelperDeliveryChange{{Kind: "modify", Path: "tests/a_test.go", Mode: "0644", ContentDigest: "sha256:content"}}}
	request := attempt.Request
	request.State = "completed"
	if err := reservation.CompleteDelivery(context.Background(), request, delivery); err != nil {
		t.Fatal(err)
	}
	delivery.RequestedPaths[0] = "outside/"
	delivery.Changes[0].Path = "outside/file"
	saved := store.snapshot().RoleStartReservations[0].HelperDelivery
	if saved == nil || saved.RequestedPaths[0] != "tests/" || saved.Changes[0].Path != "tests/a_test.go" {
		t.Fatalf("caller mutation changed the durable helper evidence: %+v", saved)
	}
}

func TestRoleStartBudgetSerializesConcurrentHelperReservations(t *testing.T) {
	store := &roleBudgetMemoryStore{}
	budget := newRoleBudgetForTest(t, store, 2)
	const attempts = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed, denied := 0, 0
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			attempt := HelperStartAttempt{Request: agentexec.RoleStartRequest{RequestID: fmt.Sprintf("helper-%d", i), ParentSessionID: "parent", Role: "helper", State: "requested"}}
			permit, err := budget.ReserveHelper(context.Background(), attempt)
			mu.Lock()
			defer mu.Unlock()
			if permit == nil {
				t.Errorf("request %d was not durably recorded: %v", i, err)
				return
			}
			if err == nil {
				allowed++
			} else if errors.Is(err, ErrRoleStartBudgetExceeded) {
				denied++
			} else {
				t.Errorf("request %d failed unexpectedly: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	if allowed != 2 || denied != attempts-2 {
		t.Fatalf("concurrent requests exceeded or underused the limit: allowed=%d denied=%d", allowed, denied)
	}
	if accounting := budget.Accounting(); accounting.ObservedNested != attempts || accounting.ObservedTotal != attempts {
		t.Fatalf("all failed and allowed attempts were not retained in accounting: %+v", accounting)
	}
}

func TestRoleStartBudgetUsesObservedPartialStartsAsLowerBound(t *testing.T) {
	store := &roleBudgetMemoryStore{report: RunReport{Tasks: []ManagerTask{{ID: "orders", WorkAttempts: 1}}, Invocations: []InvocationLog{{TaskID: "orders", Role: agentexec.RoleExecutor, Phase: "work", Receipt: agentexec.Receipt{
		Lifecycle: &agentexec.Lifecycle{Provider: "codex-app-server", SessionID: "parent", Accounting: "partial",
			StartRequests: []agentexec.RoleStartRequest{{RequestID: "root", Role: agentexec.RoleExecutor, State: "completed"},
				{RequestID: "child-one", ParentSessionID: "parent", Role: "worker", State: "completed"},
				{RequestID: "child-two", ParentSessionID: "parent", Role: "worker", State: "unknown"}}},
	}}}}}
	budget := newRoleBudgetForTest(t, store, 2)
	_, err := budget.ReserveHelper(context.Background(), HelperStartAttempt{Request: agentexec.RoleStartRequest{RequestID: "host-next", ParentSessionID: "parent", Role: "helper"}})
	if !errors.Is(err, ErrRoleStartBudgetExceeded) {
		t.Fatalf("observed nested lower bound over budget did not block dispatch: %v", err)
	}
	accounting := budget.Accounting()
	if accounting.Accounting != "partial" || accounting.ObservedTotal != 4 {
		t.Fatalf("partial lifecycle was falsely promoted or its observed starts lost: %+v", accounting)
	}
}

func TestRoleStartBudgetDoesNotDispatchWhenDurableReservationFails(t *testing.T) {
	store := &roleBudgetMemoryStore{fail: true}
	budget := newRoleBudgetForTest(t, store, 1)
	permit, err := budget.ReserveRoot(context.Background(), "orders-work", "work", agentexec.RoleStartRequest{RequestID: "root", Role: agentexec.RoleExecutor})
	if err == nil || permit != nil {
		t.Fatalf("root dispatch could proceed without durable reservation: permit=%v err=%v", permit, err)
	}
}

func TestRunHelperBudgetCountsRecordedChecksAgainstMaxStarts(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(RunsPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, RuntimePath, "apiVersion: "+APIVersion+"\n")
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	const runID = "00000000000000000000000000000b01"
	if _, err := store.createRun(runID); err != nil {
		t.Fatal(err)
	}
	report := RunReport{APIVersion: APIVersion, ID: runID, PlanID: runID, Status: StatusRunning, Revision: 1,
		Tasks:  []ManagerTask{{ID: "orders-task", ManagerID: "orders", WorkAttempts: 1, Attempts: 1}},
		Checks: []CheckResult{{ID: "check-1", Outcome: "failed", ExitCode: 1}}}
	const maxStarts = 4
	_, budget, err := bindRunHelperBudget(ProcessInvoker{}, Host{}, Limits{MaxStarts: maxStarts}, store, &report)
	if err != nil {
		t.Fatal(err)
	}
	// A check recorded after binding must also count: the budget reads the live report.
	report.Checks = append(report.Checks, CheckResult{ID: "check-2", Outcome: "passed"})
	ledger := budget.Accounting().ObservedTotal + len(report.Checks) // the run's shared start ledger
	granted := 0
	for i := 0; i < maxStarts; i++ {
		_, reserveErr := budget.ReserveHelper(context.Background(), HelperStartAttempt{
			Request:   agentexec.RoleStartRequest{RequestID: fmt.Sprintf("helper-%d", i), ParentSessionID: "parent-session", Role: "helper"},
			ManagerID: "orders", Phase: "work", ParentRunID: "parent-run"})
		if reserveErr == nil {
			granted++
		} else if !errors.Is(reserveErr, ErrRoleStartBudgetExceeded) {
			t.Fatal(reserveErr)
		}
	}
	if total := ledger + granted; total != maxStarts {
		t.Errorf("helper budget granted %d helper starts after a shared ledger of %d (1 root + %d checks): total starts %d, want exactly MaxStarts %d", granted, ledger, len(report.Checks), total, maxStarts)
	}
}
