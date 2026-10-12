//go:build !runtime_unit

package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
)

func TestDeliverResumesIntegratedRunAndCompletesAcknowledgedScope(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	enableGuidedWorkflow(t, root)
	introduceAcceptedModelEvent(t, root)
	configureOperationsFullVerify(t, root)
	// This end-to-end delivery runs Manager work, process checks, and a complete
	// whole-repository audit. Keep the cumulative fixture window generous enough
	// for the same provider-free subprocess workload under a loaded Windows suite;
	// per-process timeout and start/cost bounds remain unchanged.
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		runtime.Limits.MaxDuration = Duration(15 * time.Minute)
	})
	setupE2EProcess(t, "normal")
	const explorationID, scopeID = "deliver-orders", "orders-change"
	event := dismissFirstUnresolvedEvent(t, root)
	createDeliverExploration(t, root, explorationID, scopeID)
	managers := []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}
	assertAcceptedHistoryForManagers(t, root)
	plan, err := Plan(projectworkHost(), root, "", PlanRequest{ExplorationID: explorationID, ScopeID: scopeID,
		Operation: "apply", Goal: "Update the orders implementation", Managers: managers, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || initial.Status != StatusIntegrated {
		t.Fatalf("setup run = %+v, err=%v", initial, err)
	}
	priorInvocations := len(initial.Invocations)

	got, err := Deliver(context.Background(), projectworkHost(), ProcessInvoker{}, root, DeliverRequest{
		ExplorationID: explorationID, ScopeID: scopeID, RunID: plan.ID, ExecuteAuthorized: true,
	})
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if got.Status != StatusApplied || got.RunID != plan.ID || got.Verification == nil || got.Apply == nil || got.Apply.Status != StatusApplied {
		t.Fatalf("delivery did not reach guarded Apply: %+v", got)
	}
	if got.Verification.VerificationScope != "full" || got.Verification.ManagerVerification == nil || len(got.Verification.ManagerVerification.Managers) != 3 {
		t.Fatalf("guided delivery did not retain whole-repository Manager verification: %+v", got.Verification)
	}
	if got.Run == nil || got.Run.ID != plan.ID || len(got.Run.Invocations) != priorInvocations {
		t.Fatalf("delivery replayed manager proposals instead of continuing the existing run: prior=%d got=%+v", priorInvocations, got.Run)
	}
	record, err := projectexplore.Load(root, explorationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Completions) != 1 || record.Completions[0].RunID != plan.ID || record.Completions[0].BindingDigest == "" {
		t.Fatalf("successful Apply did not complete the exact scope binding: %+v", record.Completions)
	}
	// The resolution counts once the applied result is committed; until then
	// the event is reported as delivered but not committed.
	briefingState, _, err := projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolution := projectbriefing.EventResolutionStatus(briefingState, event.ID); resolution.Status != projectbriefing.StatusDeliveredUncommitted || resolution.Resolution == nil || resolution.Resolution.Evidence.RunID != plan.ID {
		t.Fatalf("uncommitted delivery status = %+v, want %s", resolution, projectbriefing.StatusDeliveredUncommitted)
	}
	// Re-entering a completed delivery must recover from the immutable Apply
	// receipt, not make another proposal, verification, or write.
	second, err := Deliver(context.Background(), projectworkHost(), ProcessInvoker{}, root, DeliverRequest{
		ExplorationID: explorationID, ScopeID: scopeID, RunID: plan.ID, ExecuteAuthorized: true,
	})
	if err != nil || second.Status != StatusApplied {
		t.Fatalf("idempotent delivery = %+v, err=%v", second, err)
	}
	if second.Verification != nil || second.Apply != nil {
		t.Fatalf("already-applied recovery repeated later stages: %+v", second)
	}
	if len(record.Completions) != 1 {
		t.Fatalf("completion receipt duplicated: %+v", record.Completions)
	}
	gitE2E(t, root, append([]string{"add", "--"}, got.Apply.Written...)...)
	gitE2E(t, root, "commit", "-m", "commit the delivery")
	briefingState, _, err = projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	resolution := projectbriefing.EventResolutionStatus(briefingState, event.ID)
	if resolution.Status != "resolved" || resolution.Resolution == nil || resolution.Resolution.Evidence.RunID != plan.ID ||
		resolution.Resolution.Evidence.CandidateID != runCandidateID(got.Run) || !resolution.Resolution.Evidence.FullVerifyPassed ||
		len(resolution.Resolution.Evidence.Delivered) != len(got.Apply.Written) {
		t.Fatalf("accepted model event was not resolved from the committed full Verify + Apply result: %+v", resolution)
	}
}

func TestApplyRejectsReadinessEditAtGuardedWriteBoundary(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	enableE2EReviews(t, root, 100000)
	setupE2EProcess(t, "normal")
	const explorationID, scopeID = "deliver-race", "orders-change"
	createDeliverExploration(t, root, explorationID, scopeID)
	managers := uniqueSorted([]string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")})
	plan, err := Plan(projectworkHost(), root, "", PlanRequest{ExplorationID: explorationID, ScopeID: scopeID,
		Operation: "apply", Goal: "Update the orders implementation", Managers: managers, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run = %s, %v", run.Status, err)
	}
	verified, err := Verify(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("Verify = %s, %v", verified.Status, err)
	}
	preflight, err := PreflightApply(projectworkHost(), root, plan.ID, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}

	baseHost := projectworkHost()
	guardedHost := baseHost
	mutated := false
	guardedHost.Load = func(path, revision string) (*Project, error) {
		if revision == "" && !mutated {
			mutated = true
			appendBlockingDecision(t, baseHost, root, explorationID, scopeID)
		}
		return baseHost.Load(path, revision)
	}
	apply, err := Apply(guardedHost, ProcessInvoker{}, root, ApplyRequest{
		RunID: preflight.RunID, PlanID: preflight.PlanID, CandidateID: preflight.CandidateID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch,
		ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree,
	})
	if err == nil || !mutated {
		t.Fatalf("Apply did not reject concurrent readiness edit: report=%+v err=%v", apply, err)
	}
	if apply.Status == StatusApplied || len(apply.Written) != 0 {
		t.Fatalf("stale readiness edit wrote candidate source: %+v", apply)
	}
	if data, readErr := os.ReadFile(filepath.Join(root, "src", "orders", "implementation.txt")); readErr != nil || string(data) != "orders implementation v1\n" {
		t.Fatalf("guarded Apply changed source despite stale readiness: content=%q err=%v", data, readErr)
	}
}

func TestGuidedPreviewsMatchExecuteWithoutWritingHistory(t *testing.T) {
	t.Parallel()
	for _, laterChange := range []bool{false, true} {
		name := "first model change"
		if laterChange {
			name = "later model change"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := makeFullVerifyFixture(t)
			enableGuidedWorkflow(t, root)
			configureOperationsFullVerify(t, root)
			if laterChange {
				introduceAcceptedModelEvent(t, root)
				commitManagerStatementChange(t, root, "%s artifact with explicitly documented line-item behavior.", "%s artifact with explicitly documented line-item and tax behavior.")
			} else {
				commitManagerStatementChange(t, root, "Implement the %s artifact.", "Implement the %s artifact with explicitly documented line-item behavior.")
			}
			historyPath := filepath.Join(root, filepath.FromSlash(".markitect/state/briefings/history.json"))
			_, before, err := projectbriefing.Read(root)
			if err != nil {
				t.Fatal(err)
			}
			// The readiness acknowledgement and the plan below come from previews
			// taken before any accepted history for this change was persisted.
			const explorationID, scopeID = "preview-parity", "orders-change"
			createDeliverExploration(t, root, explorationID, scopeID)
			request := PlanRequest{ExplorationID: explorationID, ScopeID: scopeID, Operation: "apply", Goal: "Update the orders implementation",
				Managers: uniqueSorted([]string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")})}
			preview, err := Plan(projectworkHost(), root, "", request)
			if err != nil {
				t.Fatalf("plan preview: %v", err)
			}
			if !laterChange {
				report, _ := FullVerify(context.Background(), projectworkHost(), ProcessInvoker{}, root, FullVerifyRequest{Revision: gitE2E(t, root, "rev-parse", "HEAD")})
				if !stringMapsEqual(report.BriefingDigests, preview.BriefingDigests) {
					t.Fatalf("full Verify preview briefings %v differ from the plan preview %v", report.BriefingDigests, preview.BriefingDigests)
				}
				if _, err := os.Stat(historyPath); !os.IsNotExist(err) {
					t.Fatalf("previews wrote accepted history on a fresh guided repository: %v", err)
				}
			}
			if _, after, err := projectbriefing.Read(root); err != nil || after != before {
				t.Fatalf("previews changed the briefing store: before=%s after=%s err=%v", before, after, err)
			}
			request.ExecuteAuthorized = true
			executed, err := Plan(projectworkHost(), root, "", request)
			if err != nil {
				t.Fatalf("execute rejected the readiness acknowledged from a preview: %v", err)
			}
			briefed := false
			for _, digest := range executed.BriefingDigests {
				briefed = briefed || digest != ""
			}
			if !briefed || !stringMapsEqual(executed.BriefingDigests, preview.BriefingDigests) {
				t.Fatalf("execute briefings %v differ from the preview %v", executed.BriefingDigests, preview.BriefingDigests)
			}
			if _, err := os.Stat(historyPath); err != nil {
				t.Fatalf("execute did not persist accepted history: %v", err)
			}
		})
	}
}
