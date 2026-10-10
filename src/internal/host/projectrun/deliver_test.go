package projectrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

func TestDeliverResumesIntegratedRunAndCompletesAcknowledgedScope(t *testing.T) {
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
	briefingState, _, err := projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	resolution := projectbriefing.EventResolutionStatus(briefingState, event.ID)
	if resolution.Status != "resolved" || resolution.Resolution == nil || resolution.Resolution.Evidence.RunID != plan.ID ||
		resolution.Resolution.Evidence.CandidateID != runCandidateID(got.Run) || !resolution.Resolution.Evidence.FullVerifyPassed {
		t.Fatalf("accepted model event was not resolved from the successful full Verify + Apply: %+v", resolution)
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
}

func TestDeliverRejectsStaleExistingRunBeforeResuming(t *testing.T) {
	root := makeProjectRunFixture(t)
	enableE2EReviews(t, root, 100000)
	setupE2EProcess(t, "normal")
	const explorationID, scopeID = "deliver-stale", "orders-change"
	createDeliverExploration(t, root, explorationID, scopeID)
	managers := []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}
	plan, err := Plan(projectworkHost(), root, "", PlanRequest{ExplorationID: explorationID, ScopeID: scopeID,
		Operation: "apply", Goal: "Update the orders implementation", Managers: managers, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	appendBlockingDecision(t, projectworkHost(), root, explorationID, scopeID)
	got, err := Deliver(context.Background(), projectworkHost(), ProcessInvoker{}, root, DeliverRequest{
		ExplorationID: explorationID, ScopeID: scopeID, RunID: plan.ID, ExecuteAuthorized: true,
	})
	if !errors.Is(err, ErrStale) || !strings.Contains(err.Error(), "readiness changed") {
		t.Fatalf("stale scope was not rejected before resuming: report=%+v err=%v", got, err)
	}
	if got.Run != nil && got.Run.Status != "" && got.Run.Status != StatusPlanned {
		t.Fatalf("stale delivery unexpectedly started the existing run: %+v", got.Run)
	}
}

func TestApplyRejectsReadinessEditAtGuardedWriteBoundary(t *testing.T) {
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

func TestDeliverRequiresExplicitAuthorizationAndScope(t *testing.T) {
	root := t.TempDir()
	for _, request := range []DeliverRequest{
		{ExplorationID: "x", ScopeID: "y"},
		{ScopeID: "y", ExecuteAuthorized: true},
		{ExplorationID: "x", ExecuteAuthorized: true},
	} {
		if _, err := Deliver(context.Background(), Host{}, nil, root, request); err == nil {
			t.Fatalf("Deliver accepted incomplete or unauthorized request %+v", request)
		}
	}
}

func TestDeliverCreatesOnePersistedRunAndDoesNotReplaceFailedRun(t *testing.T) {
	root := makeProjectRunFixture(t)
	enableE2EReviews(t, root, 100000)
	setupE2EProcess(t, "normal")
	const explorationID, scopeID = "deliver-new-run", "orders-change"
	createDeliverExploration(t, root, explorationID, scopeID)
	invoker := &failingDeliverInvoker{}
	request := DeliverRequest{ExplorationID: explorationID, ScopeID: scopeID, ExecuteAuthorized: true}
	first, err := Deliver(context.Background(), projectworkHost(), invoker, root, request)
	if err == nil || first.RunID == "" || first.Plan == nil || first.Run == nil || first.Run.Status != StatusFailed {
		t.Fatalf("fresh delivery did not preserve its failed run for explicit recovery: report=%+v err=%v", first, err)
	}
	if invoker.calls != 1 {
		t.Fatalf("fresh delivery invoked executor %d times, want one", invoker.calls)
	}
	second, err := Deliver(context.Background(), projectworkHost(), invoker, root, DeliverRequest{
		ExplorationID: explorationID, ScopeID: scopeID, RunID: first.RunID, ExecuteAuthorized: true,
	})
	if err == nil || second.RunID != first.RunID || invoker.calls != 1 {
		t.Fatalf("failed run was silently replaced or retried: report=%+v calls=%d err=%v", second, invoker.calls, err)
	}
}

type failingDeliverInvoker struct{ calls int }

func (i *failingDeliverInvoker) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	return agentexec.RunResult{}, errors.New("bounded fixture invocation failure")
}

func (*failingDeliverInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return (ProcessInvoker{}).Fingerprint(config)
}

func TestGuidedPlanCannotBypassAcknowledgedExploration(t *testing.T) {
	root := makeProjectRunFixture(t)
	enableGuidedWorkflow(t, root)
	_, err := Plan(projectworkHost(), root, "", PlanRequest{Goal: "Implement both artifacts", Operation: "apply",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err == nil || !strings.Contains(err.Error(), "guided implementation requires") {
		t.Fatalf("guided Plan bypassed the exploration and scope binding: %v", err)
	}
}

func enableGuidedWorkflow(t *testing.T, root string) {
	t.Helper()
	manifest := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "name: Process fixture\n", "name: Process fixture\nworkflowMode: guided\n", 1)
	if updated == string(data) {
		t.Fatal("could not enable guided workflow in fixture")
	}
	writeE2E(t, root, projectwork.ManifestPath, updated)
	gitE2E(t, root, "add", projectwork.ManifestPath)
	gitE2E(t, root, "commit", "-m", "enable guided workflow for deliver test")
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if project.Config.DocumentPath != "" {
		document, docErr := projectwork.Document(project, false)
		if docErr != nil {
			t.Fatal(docErr)
		}
		docPath := filepath.Join(root, filepath.FromSlash(project.Config.DocumentPath))
		prior, readErr := os.ReadFile(docPath)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(prior) != document {
			writeE2E(t, root, project.Config.DocumentPath, document)
			gitE2E(t, root, "add", project.Config.DocumentPath)
			gitE2E(t, root, "commit", "-m", "regenerate guided fixture project document")
		}
	}
}

func introduceAcceptedModelEvent(t *testing.T, root string) {
	t.Helper()
	statementPaths := []string{".markitect/model/orders/statement.yaml", ".markitect/model/inventory/statement.yaml"}
	for _, statementPath := range statementPaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(statementPath)))
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(data), "Implement the "+filepath.Base(filepath.Dir(statementPath))+" artifact.",
			"Implement the "+filepath.Base(filepath.Dir(statementPath))+" artifact with explicitly documented line-item behavior.", 1)
		if updated == string(data) {
			t.Fatalf("could not make a semantic Manager statement change in %s", statementPath)
		}
		writeE2E(t, root, statementPath, updated)
	}
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	document, err := projectwork.Document(project, false)
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, project.Config.DocumentPath, document)
	for _, statementPath := range statementPaths {
		gitE2E(t, root, "add", statementPath)
	}
	gitE2E(t, root, "add", project.Config.DocumentPath)
	gitE2E(t, root, "commit", "-m", "accept orders statement clarification")
	if _, err := projectbriefing.EnsureAcceptedHistory(root, gitE2E(t, root, "rev-parse", "HEAD")); err != nil {
		t.Fatal(err)
	}
}

func dismissFirstUnresolvedEvent(t *testing.T, root string) projectbriefing.Event {
	t.Helper()
	state, digest, err := projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if projectbriefing.EventResolutionStatus(state, event.ID).Status == "resolved" {
				continue
			}
			if len(event.AffectedManagers) == 0 {
				t.Fatalf("event has no affected Manager: %+v", event)
			}
			if _, err := projectbriefing.Dismiss(root, event.ID, event.AffectedManagers[0], digest); err != nil {
				t.Fatal(err)
			}
			after, _, err := projectbriefing.Read(root)
			if err != nil {
				t.Fatal(err)
			}
			if status := projectbriefing.EventResolutionStatus(after, event.ID).Status; status != "unresolved" {
				t.Fatalf("dismissal improperly resolved accepted model event: %+v", status)
			}
			return event
		}
	}
	t.Fatal("semantic model change did not create a briefing event")
	return projectbriefing.Event{}
}

func assertAcceptedHistoryForManagers(t *testing.T, root string) {
	t.Helper()
	head := gitE2E(t, root, "rev-parse", "HEAD")
	project, err := projectwork.Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	for _, manager := range project.Report.Managers {
		if _, _, _, err := projectbriefing.LoadForManager(root, project.Report.ModelDigest, manager.ID, head); err != nil {
			t.Fatalf("accepted briefing history is not readable for manager %s at %s: %v", manager.ID, head, err)
		}
	}
}

func runCandidateID(run *RunReport) string {
	if run == nil {
		return ""
	}
	return run.Candidate.ID
}

func createDeliverExploration(t *testing.T, root, explorationID, scopeID string) {
	t.Helper()
	goal := "Update the orders implementation"
	managers := []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}
	managers = uniqueSorted(managers)
	revision := gitE2E(t, root, "rev-parse", "HEAD")
	binding, err := ExplorationBinding(projectworkHost(), root, revision, PlanRequest{ScopeID: scopeID, Goal: goal, Operation: "apply", Managers: managers})
	if err != nil {
		t.Fatal(err)
	}
	record := projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: explorationID, Status: projectexplore.StatusActive,
		Request: goal, Scopes: []projectexplore.Scope{{ID: scopeID, Name: scopeID, Goal: goal, Operation: "apply", ManagerIDs: managers}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
	preview, err := projectexplore.CreatePreview(root, record, binding)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := projectexplore.Write(root, preview, preview.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	_, _, binding, readiness, err := LoadExplorationReadiness(projectworkHost(), root, explorationID, scopeID)
	if err != nil {
		t.Fatal(err)
	}
	priorDigest := persisted.Digest
	if err := projectexplore.AcknowledgeStructure(&persisted, scopeID, binding, projectexplore.StructureAcknowledgement{
		ScopeID: scopeID, BindingDigest: readiness.BindingDigest, StructureDigest: readiness.StructureDigest,
		Actor: "fixture-user", Authority: "approved for test", Provenance: "deliver_test", RecordedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	preview, err = projectexplore.UpdatePreview(root, persisted, priorDigest, binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectexplore.Write(root, preview, preview.Digest, binding); err != nil {
		t.Fatal(err)
	}
}

func appendBlockingDecision(t *testing.T, host Host, root, explorationID, scopeID string) {
	t.Helper()
	record, _, binding, _, err := LoadExplorationReadiness(host, root, explorationID, scopeID)
	if err != nil {
		t.Fatal(err)
	}
	priorDigest := record.Digest
	record.Decisions = append(record.Decisions, projectexplore.Decision{ID: "blocking-change", ScopeIDs: []string{scopeID},
		Question: "A new blocking decision was recorded after planning.", Blocking: true, Status: "open"})
	record.Digest = ""
	encoded, err := projectexplore.EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	record, err = projectexplore.DecodeRecord(encoded)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := projectexplore.UpdatePreview(root, record, priorDigest, binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectexplore.Write(root, preview, preview.Digest, binding); err != nil {
		t.Fatal(err)
	}
}
