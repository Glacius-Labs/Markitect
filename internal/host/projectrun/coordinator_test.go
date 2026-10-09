package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

type barrierInvoker struct {
	inner   Invoker
	mu      sync.Mutex
	started int
	ready   chan struct{}
}

func (i *barrierInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	var payload struct {
		ManagerID string `json:"managerId"`
		Phase     string `json:"phase"`
	}
	if err := json.Unmarshal(request.Context, &payload); err != nil {
		return agentexec.RunResult{}, err
	}
	if payload.Phase == "work" && (strings.Contains(payload.ManagerID, `"orders"`) || strings.Contains(payload.ManagerID, `"inventory"`)) {
		i.mu.Lock()
		i.started++
		if i.started == 2 {
			close(i.ready)
		}
		i.mu.Unlock()
		select {
		case <-i.ready:
		case <-ctx.Done():
			return agentexec.RunResult{}, ctx.Err()
		case <-time.After(3 * time.Second):
			return agentexec.RunResult{}, errors.New("independent manager work did not overlap")
		}
	}
	return i.inner.Run(ctx, config, request, options)
}

func (i *barrierInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return i.inner.Fingerprint(config)
}

func TestRunDispatchesIndependentManagersConcurrently(t *testing.T) {
	root := makeProjectRunFixture(t)
	updateE2ERuntime(t, root, func(runtime *Runtime) { runtime.Limits.MaxParallel = 2 })
	setupE2EProcess(t, "")
	plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{
		Goal: "Implement both independent owned artifacts.", Managers: []string{
			e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory"),
		}, ExecuteAuthorized: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	invoker := &barrierInvoker{inner: ProcessInvoker{}, ready: make(chan struct{})}
	report, err := Run(context.Background(), projectworkHost(), invoker, root, plan.ID)
	if err != nil || report.Status != StatusIntegrated {
		t.Fatalf("parallel manager run failed: status=%s err=%v", report.Status, err)
	}
	if invoker.started != 2 {
		t.Fatalf("barrier observed %d independent starts, want 2", invoker.started)
	}
}

func TestInvokeManagerBatchStartsTogetherAndReturnsStableOrder(t *testing.T) {
	var mu sync.Mutex
	started := 0
	release := make(chan struct{})
	calls := []managerInvocationCall{
		{managerID: "bravo", invoke: func(context.Context) (agentexec.RunResult, InvocationLog, error) {
			return waitForBatchRelease(t, &mu, &started, release), InvocationLog{}, nil
		}},
		{managerID: "alpha", invoke: func(context.Context) (agentexec.RunResult, InvocationLog, error) {
			return waitForBatchRelease(t, &mu, &started, release), InvocationLog{}, nil
		}},
	}
	results := invokeManagerBatch(context.Background(), calls)
	if len(results) != 2 || results[0].managerID != "alpha" || results[1].managerID != "bravo" {
		t.Fatalf("batch results = %+v, want stable [alpha bravo]", results)
	}
}

func waitForBatchRelease(t *testing.T, mu *sync.Mutex, started *int, release chan struct{}) agentexec.RunResult {
	t.Helper()
	mu.Lock()
	*started++
	if *started == 2 {
		close(release)
	}
	mu.Unlock()
	select {
	case <-release:
		return agentexec.RunResult{}
	case <-time.After(time.Second):
		t.Error("manager batch was dispatched serially")
		return agentexec.RunResult{}
	}
}

func TestInvokeManagerBatchCancelsAndCollectsEveryDispatchedResult(t *testing.T) {
	calls := []managerInvocationCall{
		{managerID: "alpha", invoke: func(ctx context.Context) (agentexec.RunResult, InvocationLog, error) {
			<-ctx.Done()
			return agentexec.RunResult{Receipt: agentexec.Receipt{RunID: "alpha-receipt"}}, InvocationLog{ReportID: "alpha-receipt"}, ctx.Err()
		}},
		{managerID: "bravo", invoke: func(context.Context) (agentexec.RunResult, InvocationLog, error) {
			return agentexec.RunResult{Receipt: agentexec.Receipt{RunID: "bravo-receipt"}}, InvocationLog{ReportID: "bravo-receipt"}, errors.New("dispatch failed")
		}},
	}
	results := invokeManagerBatch(context.Background(), calls)
	if len(results) != 2 || results[0].managerID != "alpha" || results[1].managerID != "bravo" {
		t.Fatalf("batch results = %+v, want both results in stable order", results)
	}
	if results[0].log.ReportID != "alpha-receipt" || results[1].log.ReportID != "bravo-receipt" || results[0].err == nil || results[1].err == nil {
		t.Fatalf("failed batch lost a result receipt: %+v", results)
	}
}

func TestBatchInvocationErrorPreservesOriginalFailureOverEarlierCancellation(t *testing.T) {
	actualFailure := errors.New("native turn failed")
	results := invokeManagerBatch(context.Background(), []managerInvocationCall{
		{managerID: "a-cancelled", invoke: func(ctx context.Context) (agentexec.RunResult, InvocationLog, error) {
			<-ctx.Done()
			return agentexec.RunResult{}, InvocationLog{}, ctx.Err()
		}},
		{managerID: "z-original-failure", invoke: func(context.Context) (agentexec.RunResult, InvocationLog, error) {
			return agentexec.RunResult{Receipt: agentexec.Receipt{RunID: "failed-receipt"}}, InvocationLog{ReportID: "failed-receipt"}, actualFailure
		}},
	})
	got := batchInvocationError(results)
	if !errors.Is(got, actualFailure) {
		t.Fatalf("batch error = %v, want preserve original failure", got)
	}
	if len(results) != 2 || results[1].log.ReportID != "failed-receipt" {
		t.Fatalf("batch did not collect failure receipt: %+v", results)
	}
}

func TestReadyManagerActionsAdmitsRetryAndSeparatesOverlappingWork(t *testing.T) {
	tasks := []ManagerTask{
		schedulerTask("root", "", 0, "worked", "."),
		schedulerTask("alpha-retry", "root", 1, "work-retry-ready", "src/shared/"),
		schedulerTask("bravo-overlap", "root", 1, "queued", "src/shared/bravo/"),
		schedulerTask("charlie-independent", "root", 1, "queued", "tests/"),
	}
	actions, err := readyManagerActions(tasks, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []managerAction{{managerID: "alpha-retry", phase: "work"}, {managerID: "charlie-independent", phase: "work"}}
	if len(actions) != len(want) {
		t.Fatalf("retry batch = %+v, want %+v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("retry batch = %+v, want %+v", actions, want)
		}
	}
}

func TestNativeTaskAttemptIDChangesForRetries(t *testing.T) {
	task := ManagerTask{ID: "manager-task", WorkAttempts: 2, IntegrationAttempts: 1}
	if got, want := nativeTaskAttemptID(task, "work"), "manager-task-work-2"; got != want {
		t.Fatalf("work attempt ID = %q, want %q", got, want)
	}
	if got, want := nativeTaskAttemptID(task, "integrate"), "manager-task-integrate-1"; got != want {
		t.Fatalf("integration attempt ID = %q, want %q", got, want)
	}
}

func TestMergeDependencyInputsCarriesProviderBytesWithoutChangingBranchBase(t *testing.T) {
	store := &runStore{}
	dir := t.TempDir()
	if err := ensureDirectory(filepath.Join(dir, "candidates")); err != nil {
		t.Fatal(err)
	}
	baseID := "00000000000000000000000000000000"
	workID := "11111111111111111111111111111111"
	base := candidateData{ID: baseID, Files: map[string]File{
		"provider/api.md": {Path: "provider/api.md", Mode: "100644", Content: []byte("old")},
		"consumer/use.md": {Path: "consumer/use.md", Mode: "100644", Content: []byte("consumer")},
	}}
	if err := store.writeCandidate(dir, base); err != nil {
		t.Fatal(err)
	}
	work := candidateData{ID: workID, Parents: []string{baseID}, Files: map[string]File{
		"provider/api.md": {Path: "provider/api.md", Mode: "100644", Content: []byte("new contract")},
		"consumer/use.md": {Path: "consumer/use.md", Mode: "100644", Content: []byte("consumer")},
	}}
	if err := store.writeCandidate(dir, work); err != nil {
		t.Fatal(err)
	}
	input, err := mergeDependencyInputs(store, dir, []ManagerTask{{ManagerID: "provider", CandidateID: workID}}, base, []string{"provider"})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(input.Files["provider/api.md"].Content); got != "new contract" {
		t.Fatalf("dependent input provider bytes = %q, want new contract", got)
	}
	if got := string(base.Files["provider/api.md"].Content); got != "old" {
		t.Fatalf("branch base was mutated: %q", got)
	}
}
