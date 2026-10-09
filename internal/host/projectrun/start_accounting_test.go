package projectrun

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

func TestReportStartAccountingCountsFailedHostAttemptsWithoutReceipt(t *testing.T) {
	report := RunReport{Tasks: []ManagerTask{{ID: "root", WorkAttempts: 2, IntegrationAttempts: 1}}}
	summary, nested := reportStartAccounting(report)
	if summary.Roots != 3 || summary.ObservedNested != 0 || summary.ObservedTotal != 3 || summary.Accounting != "unavailable" || len(nested) != 0 {
		t.Fatalf("failed attempts without receipts were misrepresented: %#v, %#v", summary, nested)
	}
}

func TestReportStartAccountingDeduplicatesCumulativeNestedRequests(t *testing.T) {
	root := agentexec.RoleStartRequest{RequestID: "root", Role: "executor", State: "completed"}
	nested := []agentexec.RoleStartRequest{
		{RequestID: "child-failed", ParentSessionID: "session", Role: "worker", State: "failed"},
		{RequestID: "child-requested", ParentSessionID: "session", Role: "worker", State: "requested"},
		{RequestID: "child-unknown", ParentSessionID: "session", Role: "worker", State: "unknown"},
	}
	first := append([]agentexec.RoleStartRequest{root}, nested...)
	second := append([]agentexec.RoleStartRequest{root}, nested...)
	second[1].State = "failed" // newest cumulative appearance retains failure state
	report := RunReport{
		Tasks: []ManagerTask{{ID: "root", WorkAttempts: 1, IntegrationAttempts: 1}},
		Invocations: []InvocationLog{
			{Role: agentexec.RoleExecutor, Phase: "work", Receipt: agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: "codex-app-server", SessionID: "session", Accounting: "complete", StartRequests: first}}},
			{Role: agentexec.RoleExecutor, Phase: "integrate", Receipt: agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: "codex-app-server", SessionID: "session", Accounting: "complete", StartRequests: second}}},
		},
	}
	summary, observed := reportStartAccounting(report)
	if summary.Roots != 2 || summary.ObservedNested != 3 || summary.ObservedTotal != 5 || summary.Accounting != "complete" || len(observed) != 3 {
		t.Fatalf("cumulative nested requests were not unioned: %#v, %#v", summary, observed)
	}
	states := map[string]string{}
	for _, request := range observed {
		states[request.RequestID] = request.State
	}
	for _, request := range nested {
		if states[request.RequestID] != request.State {
			t.Errorf("nested request %s state = %q, want %q", request.RequestID, states[request.RequestID], request.State)
		}
	}
}

func TestReportStartAccountingCountsReviewerVerifierAndInferRootsOnce(t *testing.T) {
	report := RunReport{
		Tasks: []ManagerTask{{ID: "manager", WorkAttempts: 1, IntegrationAttempts: 1}},
		Invocations: []InvocationLog{
			{Role: agentexec.RoleExecutor, Phase: "work", Receipt: completeLifecycle("work-session")},
			{Role: agentexec.RoleExecutor, Phase: "integrate", Receipt: completeLifecycle("integrate-session")},
			{Role: "reviewer", Phase: "review", Receipt: completeLifecycle("review-session")},
			{Role: agentexec.RoleInfer, Phase: "infer", Receipt: completeLifecycle("infer-session")},
			{Role: agentexec.RoleVerifier, Phase: "verify", Receipt: completeLifecycle("verify-session")},
		},
	}
	summary, nested := reportStartAccounting(report)
	if summary.Roots != 5 || summary.ObservedTotal != 5 || summary.Accounting != "complete" || len(nested) != 0 {
		t.Fatalf("non-task Host roles were not counted exactly once: %#v, %#v", summary, nested)
	}
}

func TestReportStartAccountingDoesNotCountChecksAsAgentStarts(t *testing.T) {
	report := RunReport{
		Tasks:       []ManagerTask{{ID: "manager", WorkAttempts: 1}},
		Checks:      []CheckResult{{ID: "build"}, {ID: "test"}, {ID: "lint"}},
		Invocations: []InvocationLog{{Role: agentexec.RoleExecutor, Phase: "work", Receipt: completeLifecycle("session")}},
	}
	summary, _ := reportStartAccounting(report)
	if summary.Roots != 1 || summary.ObservedTotal != 1 {
		t.Fatalf("checks were counted as agent starts: %#v", summary)
	}
}

func TestReportStartAccountingKeepsMissingAndPartialLifecycleHonest(t *testing.T) {
	t.Run("missing lifecycle", func(t *testing.T) {
		report := RunReport{
			Tasks:       []ManagerTask{{ID: "root", WorkAttempts: 1}},
			Invocations: []InvocationLog{{Role: agentexec.RoleExecutor, Phase: "work", Receipt: agentexec.Receipt{}}},
		}
		summary, nested := reportStartAccounting(report)
		if summary.Roots != 1 || summary.ObservedNested != 0 || summary.ObservedTotal != 1 || summary.Accounting != "unavailable" || len(nested) != 0 {
			t.Fatalf("missing lifecycle was treated as observed zero: %#v, %#v", summary, nested)
		}
	})

	t.Run("explicit unavailable", func(t *testing.T) {
		report := RunReport{Invocations: []InvocationLog{{Role: "reviewer", Receipt: agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: "codex-app-server", Accounting: "unavailable"}}}}}
		summary, _ := reportStartAccounting(report)
		if summary.Roots != 1 || summary.Accounting != "unavailable" {
			t.Fatalf("unavailable lifecycle was overclaimed: %#v", summary)
		}
	})

	t.Run("partial with observed child", func(t *testing.T) {
		life := &agentexec.Lifecycle{Provider: "codex-app-server", SessionID: "session", Accounting: "partial", StartRequests: []agentexec.RoleStartRequest{
			{RequestID: "root", State: "completed"}, {RequestID: "child", State: "unknown"},
		}}
		report := RunReport{Invocations: []InvocationLog{{Role: "reviewer", Receipt: agentexec.Receipt{Lifecycle: life}}}}
		summary, nested := reportStartAccounting(report)
		if summary.Roots != 1 || summary.ObservedNested != 1 || summary.ObservedTotal != 2 || summary.Accounting != "partial" || len(nested) != 1 || nested[0].State != "unknown" {
			t.Fatalf("partial lifecycle was misrepresented: %#v, %#v", summary, nested)
		}
	})
}

func completeLifecycle(session string) agentexec.Receipt {
	return agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: "codex-app-server", SessionID: session, Accounting: "complete", StartRequests: []agentexec.RoleStartRequest{{RequestID: "root", State: "completed"}}}}
}
