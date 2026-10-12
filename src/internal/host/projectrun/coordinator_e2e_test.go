//go:build !runtime_unit

package projectrun

import (
	"context"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"testing"
)

func TestRunDispatchesIndependentManagersConcurrently(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) { runtime.Limits.MaxParallel = 2 })
	setupE2EProcess(t, "")
	plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{
		Goal: "Implement both independent owned artifacts.", Managers: []string{
			e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory"),
		}, ExecuteAuthorized: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	invoker := &barrierInvoker{inner: projectexecutor.ProcessInvoker{}, ready: make(chan struct{})}
	report, err := Run(context.Background(), projectworkHost(), invoker, root, plan.ID)
	if err != nil || report.Status != StatusIntegrated {
		t.Fatalf("parallel manager run failed: status=%s err=%v", report.Status, err)
	}
	if invoker.started != 2 {
		t.Fatalf("barrier observed %d independent starts, want 2", invoker.started)
	}
}
