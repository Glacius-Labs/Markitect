package projectadoption

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

const (
	managerRunHelperEnv = "MARKITECT_MANAGER_RUN_TEST_HELPER"
	managerRunCountEnv  = "MARKITECT_MANAGER_RUN_TEST_COUNT"
	managerRunModeEnv   = "MARKITECT_MANAGER_RUN_TEST_MODE"
)

func TestManagerRunExecutorHelper(t *testing.T) {
	if os.Getenv(managerRunHelperEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		os.Exit(31)
	}
	countPath := os.Getenv(managerRunCountEnv)
	if countPath != "" {
		previous, _ := os.ReadFile(countPath)
		_ = os.WriteFile(countPath, []byte(strings.TrimSpace(string(previous))+"x"), 0o600)
	}
	var contextData struct {
		Phase          string `json:"phase"`
		ManagerContext struct {
			Manager  DistillationTargetManager `json:"manager"`
			Evidence []ManagerReverseEvidence  `json:"evidence"`
		} `json:"managerContext"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextData); err != nil || len(contextData.ManagerContext.Evidence) != 1 {
		os.Exit(32)
	}
	evidence := contextData.ManagerContext.Evidence[0]
	var evidenceContent string
	for _, artifact := range invocation.Request.Artifacts {
		if artifact.Path == "evidence/"+evidence.EvidenceID+".txt" {
			evidenceContent = string(artifact.Content)
			break
		}
	}
	if evidenceContent == "" {
		os.Exit(33)
	}
	line := 0
	for i, value := range strings.Split(evidenceContent, "\n") {
		if value != "" {
			line = i + 1
			break
		}
	}
	if line == 0 {
		os.Exit(34)
	}
	scopeID := "orders"
	claimID := "orders-claim"
	excerpt := strings.Split(evidenceContent, "\n")[line-1]
	draft := DistillationDraft{
		Claims: []DistillationDraftClaim{{ID: claimID, ScopeID: scopeID, Kind: "observation", Method: "static-source", Statement: "The selected source defines order behavior.",
			Evidence: []EvidenceRef{{EvidenceID: evidence.EvidenceID, StartLine: line, EndLine: line, Excerpt: excerpt}}, Uncertainty: []string{}, RuntimeObservationJSON: ""}},
		Terms: []Term{}, Contradictions: []Contradiction{}, Questions: []DistillationDraftQuestion{},
		Scopes: []DistillationDraftScope{{ID: scopeID, Name: "Order behavior", ParentID: "", ClaimIDs: []string{claimID}, OwnerCandidate: ""}},
		Proposal: ModelProposal{Goal: "Represent observed order behavior", Files: []ProposedFile{{ScopeID: scopeID, Path: ".markitect/model/orders/statement.yaml",
			Content: "apiVersion: " + projectwork.APIVersion + "\nkind: Statement\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Order behavior\nspec:\n  category: concept\n  description: Order behavior\n  public: false\n  uses: []\n  requires: []\n"}}},
	}
	// These modes return a child that passes proposal checks but that the
	// session ledger rejects: an invalid ID, or an accepted Manager ID whose
	// fixed parent differs.
	hierarchy := []ProposedManager{}
	parentID := contextData.ManagerContext.Manager.ID
	switch os.Getenv(managerRunModeEnv) {
	case "invalid-child-id":
		hierarchy = []ProposedManager{{ID: "Orders_Manager", Name: "Orders Manager", Purpose: "Model order behavior", ParentID: parentID, EvidenceIDs: []string{evidence.EvidenceID}, DelegationEvidenceIDs: []string{}}}
	case "accepted-child-mismatch":
		hierarchy = []ProposedManager{{ID: parentID, Name: "Orders Manager", Purpose: "Model order behavior", ParentID: parentID, EvidenceIDs: []string{evidence.EvidenceID}, DelegationEvidenceIDs: []string{}}}
	}
	var report any
	if contextData.Phase == ManagerRunPhasePropose {
		report = ManagerProposalDraft{Report: draft, Hierarchy: hierarchy, PublicContracts: []ManagerPublicContract{}}
	} else {
		report = ManagerIntegrationDraft{Report: draft, Conflicts: []SessionConflict{}}
	}
	reportBytes, _ := json.Marshal(report)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Pointer(10), OutputTokens: int64Pointer(5), ToolCalls: int64Pointer(0)}
	if os.Getenv(managerRunModeEnv) == "missing-usage" {
		usage = nil
	}
	if os.Getenv(managerRunModeEnv) == "invalid-report" {
		reportBytes = []byte(`{"not":"a manager report"}`)
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: reportBytes, Uncertainty: []string{}, Usage: usage}
	_ = json.NewEncoder(os.Stdout).Encode(response)
	os.Exit(0)
}

func TestManagerRunPersistsOneExplicitCallPerStageAndRecoversReplay(t *testing.T) {
	root, targetRoot, session, iterationID := managerRunFixture(t)
	counter := filepath.Join(t.TempDir(), "calls.txt")
	config := managerRunTestConfig(t, counter)
	limits := managerRunTestLimits()

	proposalPreview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	if proposalPreview.PreviewDigest == "" || proposalPreview.RemainingStarts != 4 || proposalPreview.EvidenceCount != 1 {
		t.Fatalf("preview omitted its guard or bounded evidence/budget: %+v", proposalPreview)
	}
	proposed, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, proposalPreview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err != nil {
		t.Fatalf("proposal stage: %v", err)
	}
	if proposed.Status != "recorded" || proposed.Proposal == nil || proposed.Proposal.Report.Method != "agent-assisted" || proposed.Proposal.Report.RunnerIdentity == "" || proposed.Proposal.Report.SchemaDigest == "" {
		t.Fatalf("proposal stage did not bind its executed report: %+v", proposed)
	}

	// A fresh preview of the already persisted proposal is replay-safe: it
	// returns completion without making a second executor call.
	replayPreview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, proposed.SessionDigest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	if !replayPreview.Completed {
		t.Fatal("proposal replay preview did not report the persisted stage complete")
	}
	replayed, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, replayPreview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err != nil || replayed.Status != "already-completed" {
		t.Fatalf("completed proposal replay = %+v, err=%v", replayed, err)
	}

	integrationPreview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhaseIntegrate,
		session.TargetContext.RootManagerID, proposed.SessionDigest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	integrated, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhaseIntegrate,
		session.TargetContext.RootManagerID, integrationPreview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err != nil {
		t.Fatalf("integration stage: %v", err)
	}
	if integrated.Status != "recorded" || integrated.Integration == nil || integrated.Integration.ManagerID != session.TargetContext.RootManagerID {
		t.Fatalf("integration stage was not independently recorded: %+v", integrated)
	}
	calls, _ := os.ReadFile(counter)
	if string(calls) != "xx" {
		t.Fatalf("expected exactly one distinct process per explicit phase, calls=%q", calls)
	}
}

func TestManagerRunRecoversReceiptAfterSessionCASCrash(t *testing.T) {
	root, targetRoot, session, iterationID := managerRunFixture(t)
	counter := filepath.Join(t.TempDir(), "calls.txt")
	config, limits := managerRunTestConfig(t, counter), managerRunTestLimits()
	preview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a crash after the immutable terminal receipt was stored but
	// before the session CAS became durable.
	if _, err := WriteBrownfieldSession(root, session, completed.SessionDigest); err != nil {
		t.Fatal(err)
	}
	recoveryPreview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, recoveryPreview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err != nil || recovered.Status != "recovered" || recovered.Proposal == nil {
		t.Fatalf("stored stage receipt did not recover: %+v err=%v", recovered, err)
	}
	if calls, _ := os.ReadFile(counter); string(calls) != "x" {
		t.Fatalf("recovery launched a second executor: %q", calls)
	}
}

func TestManagerRunStalePreviewAndExplicitFailureRetry(t *testing.T) {
	t.Run("runtime change before guarded start", func(t *testing.T) {
		root, targetRoot, session, iterationID := managerRunFixture(t)
		counter := filepath.Join(t.TempDir(), "calls.txt")
		config := managerRunTestConfig(t, counter)
		limits := managerRunTestLimits()
		preview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, session.Digest, "", config, limits)
		if err != nil {
			t.Fatal(err)
		}
		runtimePath := filepath.Join(targetRoot, filepath.FromSlash(managerRunRuntimePath))
		original, err := os.ReadFile(runtimePath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(runtimePath, append(original, []byte("# changed\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err = RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
		if err == nil {
			t.Fatal("runtime changed after preview must invalidate the invocation guard")
		}
		calls, _ := os.ReadFile(counter)
		if len(calls) != 0 {
			t.Fatalf("stale runtime launched an executor: %q", calls)
		}
	})

	t.Run("executable configuration change before guarded start", func(t *testing.T) {
		root, targetRoot, session, iterationID := managerRunFixture(t)
		counter := filepath.Join(t.TempDir(), "calls.txt")
		config := managerRunTestConfig(t, counter)
		limits := managerRunTestLimits()
		preview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, session.Digest, "", config, limits)
		if err != nil {
			t.Fatal(err)
		}
		config.Args = append(config.Args, "changed-after-preview")
		_, err = RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
		if err == nil {
			t.Fatal("changed executable invocation configuration must invalidate the preview")
		}
		calls, _ := os.ReadFile(counter)
		if len(calls) != 0 {
			t.Fatalf("stale executable configuration launched an executor: %q", calls)
		}
	})

	t.Run("known failure only retries explicitly", func(t *testing.T) {
		root, targetRoot, session, iterationID := managerRunFixture(t)
		counter := filepath.Join(t.TempDir(), "calls.txt")
		config := managerRunTestConfig(t, counter)
		limits := managerRunTestLimits()
		t.Setenv(managerRunModeEnv, "invalid-report")
		preview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, session.Digest, "", config, limits)
		if err != nil {
			t.Fatal(err)
		}
		failed, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
		if err == nil || failed.AttemptStatus != "failed" || failed.Execution == nil {
			t.Fatalf("bad report should retain a priced execution receipt: %+v err=%v", failed, err)
		}
		ledgerDir, _ := sessionDirectory(root, session.ID, false)
		ledger, err := loadManagerRunLedger(ledgerDir, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(ledger.Events) != 2 || ledger.Events[1].SafeFailure != "invalid-manager-report" || !ledger.Events[1].CostKnown {
			t.Fatalf("failed attempt ledger incomplete: %+v", ledger)
		}
		// Repeating the same stage is a durable replay and must not invoke again.
		replayPreview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, session.Digest, "", config, limits)
		if err != nil {
			t.Fatal(err)
		}
		replayed, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, replayPreview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
		if err == nil || replayed.Status != "attempt-exists" {
			t.Fatalf("failed replay=%+v err=%v", replayed, err)
		}
		if b, _ := os.ReadFile(counter); string(b) != "x" {
			t.Fatalf("implicit replay made an extra call: %q", b)
		}

		// Only an explicit retry ID can authorize another invocation.
		retryPreview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, session.Digest, failed.AttemptID, config, limits)
		if err != nil {
			t.Fatal(err)
		}
		_, err = RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
			session.TargetContext.RootManagerID, retryPreview.PreviewDigest, failed.AttemptID, config, limits, AgentExecManagerRunInvoker{})
		if err == nil {
			t.Fatal("explicit retry still returns the intentionally invalid report")
		}
		if b, _ := os.ReadFile(counter); string(b) != "xx" {
			t.Fatalf("explicit retry invocation count=%q", b)
		}
	})
}

func TestManagerRunUnrecordableHierarchyIsRetryableFailure(t *testing.T) {
	for _, mode := range []string{"invalid-child-id", "accepted-child-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			root, targetRoot, session, iterationID := managerRunFixture(t)
			counter := filepath.Join(t.TempDir(), "calls.txt")
			config := managerRunTestConfig(t, counter)
			limits := managerRunTestLimits()
			t.Setenv(managerRunModeEnv, mode)
			rootID := session.TargetContext.RootManagerID
			preview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose, rootID, session.Digest, "", config, limits)
			if err != nil {
				t.Fatal(err)
			}
			failed, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
				rootID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
			if err == nil || failed.Status != "failed" || failed.AttemptStatus != "failed" || failed.Execution == nil {
				t.Fatalf("a hierarchy the session rejects must seal a failed attempt with its receipt: %+v err=%v", failed, err)
			}
			ledgerDir, _ := sessionDirectory(root, session.ID, false)
			ledger, err := loadManagerRunLedger(ledgerDir, session.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(ledger.Events) != 2 || ledger.Events[1].SafeFailure != "invalid-manager-report" || ledger.Events[1].Proposal != nil {
				t.Fatalf("failed attempt must be recorded as invalid-manager-report without a recoverable proposal: %+v", ledger.Events)
			}
			replayPreview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose, rootID, session.Digest, "", config, limits)
			if err != nil {
				t.Fatal(err)
			}
			replayed, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
				rootID, replayPreview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
			if err == nil || replayed.Status != "attempt-exists" {
				t.Fatalf("plain re-run must report the failed attempt, not recover it: %+v err=%v", replayed, err)
			}
			if _, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose, rootID, session.Digest, failed.AttemptID, config, limits); err != nil {
				t.Fatalf("explicit retry of the failed attempt must be allowed: %v", err)
			}
			if b, _ := os.ReadFile(counter); string(b) != "x" {
				t.Fatalf("invocation count=%q", b)
			}
		})
	}
}

func TestManagerRunLeafIntegrationUnderIntegratedParentIsRetryableFailure(t *testing.T) {
	root, target, session, report := leafChildSession(t)
	session, err := integrateLeafParent(session, report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBrownfieldSession(root, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	counter := filepath.Join(t.TempDir(), "calls.txt")
	config := managerRunTestConfig(t, counter)
	limits := managerRunTestLimits()
	rootID := session.TargetContext.RootManagerID
	preview, err := PreviewManagerStage(root, target.Root, session.ID, "orders-pass", ManagerRunPhaseIntegrate, rootID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := RunManagerStage(context.Background(), root, target.Root, session.ID, "orders-pass", ManagerRunPhaseIntegrate,
		rootID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err == nil || failed.Status != "failed" || failed.AttemptStatus != "failed" || failed.Execution == nil {
		t.Fatalf("an integration the session rejects must seal a failed attempt with its receipt: %+v err=%v", failed, err)
	}
	ledgerDir, _ := sessionDirectory(root, session.ID, false)
	ledger, err := loadManagerRunLedger(ledgerDir, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Events) != 2 || ledger.Events[1].SafeFailure != "invalid-manager-report" || ledger.Events[1].Integration != nil {
		t.Fatalf("failed attempt must be recorded as invalid-manager-report without a recoverable integration: %+v", ledger.Events)
	}
	replayPreview, err := PreviewManagerStage(root, target.Root, session.ID, "orders-pass", ManagerRunPhaseIntegrate, rootID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := RunManagerStage(context.Background(), root, target.Root, session.ID, "orders-pass", ManagerRunPhaseIntegrate,
		rootID, replayPreview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err == nil || replayed.Status != "attempt-exists" {
		t.Fatalf("plain re-run must report the failed attempt, not recover it: %+v err=%v", replayed, err)
	}
	if _, err := PreviewManagerStage(root, target.Root, session.ID, "orders-pass", ManagerRunPhaseIntegrate, rootID, session.Digest, failed.AttemptID, config, limits); err != nil {
		t.Fatalf("explicit retry of the failed attempt must be allowed: %v", err)
	}
	if b, _ := os.ReadFile(counter); string(b) != "x" {
		t.Fatalf("invocation count=%q", b)
	}
}

func TestManagerRunLedgerChangeInvalidatesPreview(t *testing.T) {
	root, targetRoot, session, iterationID := managerRunFixture(t)
	counter := filepath.Join(t.TempDir(), "calls.txt")
	config, limits := managerRunTestConfig(t, counter), managerRunTestLimits()
	preview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	ledgerDir, err := sessionDirectory(root, session.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := loadManagerRunLedger(ledgerDir, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-time.Millisecond)
	start := ManagerRunEvent{Sequence: 1, AttemptID: "external-attempt", Event: "started", SessionDigest: session.Digest,
		IterationID: "other-iteration", Phase: ManagerRunPhasePropose, ManagerID: "other-manager", AgentManagerID: "other-manager",
		StartedAt: started}
	sealManagerRunEvent(&start)
	terminal := ManagerRunEvent{Sequence: 2, AttemptID: start.AttemptID, Event: "terminal", SessionDigest: start.SessionDigest,
		IterationID: start.IterationID, Phase: start.Phase, ManagerID: start.ManagerID, AgentManagerID: start.AgentManagerID,
		StartedAt: started, FinishedAt: time.Now().UTC(), Status: "failed", CostKnown: true}
	sealManagerRunEvent(&terminal)
	ledger.Budget = ptrManagerRunLimits(budgetProjection(limits))
	ledger.Events = []ManagerRunEvent{start, terminal}
	sealManagerRunLedger(&ledger)
	if err := writeManagerRunLedger(ledgerDir, ledger); err != nil {
		t.Fatal(err)
	}
	_, err = RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err == nil {
		t.Fatal("changed attempt ledger must invalidate the preview")
	}
	if calls, _ := os.ReadFile(counter); len(calls) != 0 {
		t.Fatalf("stale ledger launched an executor: %q", calls)
	}
}

func TestManagerRunPendingAttemptIsFailClosedAndNotReplayed(t *testing.T) {
	root, targetRoot, session, iterationID := managerRunFixture(t)
	counter := filepath.Join(t.TempDir(), "calls.txt")
	config, limits := managerRunTestConfig(t, counter), managerRunTestLimits()
	ledgerDir, err := sessionDirectory(root, session.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := loadManagerRunLedger(ledgerDir, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	ledger.Budget = ptrManagerRunLimits(budgetProjection(limits))
	start := ManagerRunEvent{Sequence: 1, AttemptID: "uncertain-attempt", Event: "started", SessionDigest: session.Digest,
		IterationID: iterationID, Phase: ManagerRunPhasePropose, ManagerID: session.TargetContext.RootManagerID,
		AgentManagerID: session.TargetContext.RootManagerID, StartedAt: time.Now().UTC()}
	sealManagerRunEvent(&start)
	ledger.Events = []ManagerRunEvent{start}
	sealManagerRunLedger(&ledger)
	if err := writeManagerRunLedger(ledgerDir, ledger); err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunManagerStage(context.Background(), root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, preview.PreviewDigest, "", config, limits, AgentExecManagerRunInvoker{})
	if err == nil || result.AttemptStatus != "uncertain" {
		t.Fatalf("pending attempt should block with its durable status: %+v err=%v", result, err)
	}
	if calls, _ := os.ReadFile(counter); len(calls) != 0 {
		t.Fatalf("pending attempt was implicitly replayed: %q", calls)
	}
}

func TestManagerRunBudgetCannotRefillFromChangedRuntime(t *testing.T) {
	initial := managerRunTestLimits()
	startedAt := time.Now().UTC().Add(-time.Second)
	ledger := ManagerRunLedger{Events: []ManagerRunEvent{
		{Event: "started", AttemptID: "prior", StartedAt: startedAt},
		{Event: "terminal", AttemptID: "prior", Status: "failed", CostKnown: true, EstimatedCostMicros: 9, StartedAt: startedAt, FinishedAt: time.Now().UTC()},
	}, Budget: ptrManagerRunLimits(budgetProjection(initial))}
	changed := initial
	changed.MaxStarts += 3
	changed.MaxRetries += 1
	changed.MaxDuration += time.Hour
	changed.MaxCostMicros *= 2
	changed.InputPriceMicrosPerMillion *= 2
	effective, err := frozenManagerRunLimits(ledger, changed)
	if err != nil {
		t.Fatal(err)
	}
	if effective.MaxStarts != initial.MaxStarts || effective.MaxRetries != initial.MaxRetries || effective.MaxDuration != initial.MaxDuration || effective.MaxCostMicros != initial.MaxCostMicros || effective.InputPriceMicrosPerMillion != changed.InputPriceMicrosPerMillion {
		t.Fatalf("changed runtime refilled the session budget: got %+v initial %+v", effective, initial)
	}
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Pointer(100), OutputTokens: int64Pointer(50), ToolCalls: int64Pointer(0)}
	_, _, _, historicalCost, unknown := managerRunUsage(ledger)
	if unknown || historicalCost != 9 {
		t.Fatalf("sealed historical cost was recomputed or lost: cost=%d unknown=%v", historicalCost, unknown)
	}
	newCost, known := managerRunCost(usage, effective)
	if !known || newCost <= 0 {
		t.Fatalf("current Manager's explicit price was not used for its own call: cost=%d known=%v", newCost, known)
	}
}

func TestManagerRequestContractDigestCoversStaticInputsAndIgnoresOnlyRemainingBudget(t *testing.T) {
	base := managerRunRequestContext{Kind: "proposal", Instructions: "instructions", Phase: ManagerRunPhasePropose,
		SessionDigest: "session", DiscoveryDigest: "discovery", TargetDigest: "target",
		ManagerContext: ManagerReverseContext{APIVersion: "context-v1", SessionDigest: "session", IterationID: "root", Digest: "manager-context"},
		Integration:    &ManagerIntegrationContext{APIVersion: "integration-v1", SessionDigest: "session", IterationID: "root", Digest: "child-context"},
		ModelSchema:    map[string]any{"kind": "Statement", "version": 1}, Schema: json.RawMessage(`{"type":"object"}`),
		Budget: ManagerRunContextBudget{RemainingStarts: 4, RemainingRetries: 2, RemainingDuration: time.Minute, RemainingCostMicros: 100, RequestedTimeout: 5 * time.Second}}
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: "source-revision", ModelDigest: "sha256:source",
		ModulePin: "module-v1", ProjectionID: "brownfield-manager-propose", ScopeIDs: []string{"root"}, PolicyIDs: []string{"report-only"},
		Artifacts: []agentexec.Artifact{{Path: "evidence/one.txt", Mode: "0644", Digest: "evidence-digest", Content: []byte("assigned evidence")}}}
	request.Context, _ = json.Marshal(base)
	baseDigest, err := managerRequestContractDigest(request, base)
	if err != nil {
		t.Fatal(err)
	}
	baseFullRequestDigest := digestValue(request)

	volatile := base
	volatile.Budget.RemainingStarts = 3
	volatile.Budget.RemainingRetries = 1
	volatile.Budget.RemainingDuration = 30 * time.Second
	volatile.Budget.RemainingCostMicros = 90
	request.Context, _ = json.Marshal(volatile)
	volatileDigest, err := managerRequestContractDigest(request, volatile)
	if err != nil {
		t.Fatal(err)
	}
	if volatileDigest != baseDigest {
		t.Fatal("remaining budget counters changed the stable request contract")
	}
	if digestValue(request) == baseFullRequestDigest {
		t.Fatal("full request digest should remain sensitive to current remaining budget")
	}

	mutations := []struct {
		name string
		edit func(*managerRunRequestContext, *agentexec.Request)
	}{
		{"instructions", func(ctx *managerRunRequestContext, _ *agentexec.Request) { ctx.Instructions += " changed" }},
		{"response schema", func(ctx *managerRunRequestContext, _ *agentexec.Request) {
			ctx.Schema = json.RawMessage(`{"type":"array"}`)
		}},
		{"active model schema", func(ctx *managerRunRequestContext, _ *agentexec.Request) {
			ctx.ModelSchema = map[string]any{"kind": "different"}
		}},
		{"integration report binding", func(ctx *managerRunRequestContext, _ *agentexec.Request) {
			ctx.Integration.Digest = "different-child-context"
		}},
		{"artifact content", func(_ *managerRunRequestContext, req *agentexec.Request) {
			req.Artifacts[0].Content = []byte("different assigned evidence")
		}},
		{"semantic envelope", func(_ *managerRunRequestContext, req *agentexec.Request) {
			req.PolicyIDs = []string{"different-policy"}
		}},
		{"requested timeout", func(ctx *managerRunRequestContext, _ *agentexec.Request) { ctx.Budget.RequestedTimeout++ }},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			contextCopy := base
			if base.Integration != nil {
				integrationCopy := *base.Integration
				contextCopy.Integration = &integrationCopy
			}
			requestCopy := request
			requestCopy.Artifacts = append([]agentexec.Artifact{}, request.Artifacts...)
			requestCopy.PolicyIDs = append([]string{}, request.PolicyIDs...)
			test.edit(&contextCopy, &requestCopy)
			got, err := managerRequestContractDigest(requestCopy, contextCopy)
			if err != nil {
				t.Fatal(err)
			}
			if got == baseDigest {
				t.Fatal("static request mutation did not change the contract digest")
			}
		})
	}
}

func TestManagerIntegrationRequestCarriesChildReportWithoutUnassignedEvidence(t *testing.T) {
	root, sourceCommit := committedRepository(t, map[string]string{
		"src/orders.go":          "package orders\nfunc Order() {}\n",
		"docs/orders.md":         "Orders are managed by the order module.\n",
		"private/unassigned.txt": "UNASSIGNED-PRIVATE-EVIDENCE-SENTINEL\n",
	})
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "manager-integration-context", Purpose: "Map order behavior", Review: "review-context",
		Commit: sourceCommit, ScopeRoots: []string{"."}, Selected: []SelectedPath{
			{ID: "orders-code", Path: "src/orders.go", Reason: "Implementation evidence", Basis: "code"},
			{ID: "orders-doc", Path: "docs/orders.md", Reason: "Documented intent", Basis: "documentation"},
			{ID: "unassigned", Path: "private/unassigned.txt", Reason: "Retained but not assigned", Basis: "code"},
		}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	targetRoot, _ := committedRepository(t, map[string]string{"README.md": "target\n"})
	gitRun(t, targetRoot, "checkout", "-b", "codex/manager-run-context-target")
	if _, err := projectwork.Init(targetRoot, "Manager context target", true); err != nil {
		t.Fatal(err)
	}
	gitRun(t, targetRoot, "add", "--all")
	gitRun(t, targetRoot, "commit", "--quiet", "-m", "initialize target")
	revision := gitRun(t, targetRoot, "rev-parse", "HEAD")
	target, err := projectwork.Load(targetRoot, revision)
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	rootID := session.TargetContext.RootManagerID
	session, err = BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "root-pass", ManagerID: rootID,
		EvidenceIDs: []string{"orders-doc"}, DelegationEvidenceIDs: []string{"orders-code"}, Purpose: "Propose order structure", Review: "root-review"})
	if err != nil {
		t.Fatal(err)
	}
	rootIteration, _ := findIteration(session, "root-pass")
	childManager := ProposedManager{ID: "orders-manager", Name: "Orders Manager", Purpose: "Model order implementation", ParentID: rootID, EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}}
	rootProposal := ManagerProposal{ManagerID: rootID, EvidenceIDs: []string{"orders-doc"}, Hierarchy: []ProposedManager{childManager},
		PublicContracts: []ManagerPublicContract{}, Report: sessionReport(discovery, "orders-doc", "orders", "Orders are managed by the order module.", "Documented order intent.")}
	session, err = RecordManagerProposal(session, rootIteration.ID, rootProposal)
	if err != nil {
		t.Fatal(err)
	}
	session, err = BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "orders-pass", ParentIterationID: "root-pass", ManagerID: childManager.ID,
		EvidenceIDs: childManager.EvidenceIDs, DelegationEvidenceIDs: []string{}, Purpose: childManager.Purpose, Review: "child-review"})
	if err != nil {
		t.Fatal(err)
	}
	childReport := sessionReport(discovery, "orders-code", "orders", "func Order() {}", "The selected module defines order behavior.")
	childProposal := ManagerProposal{ManagerID: childManager.ID, EvidenceIDs: []string{"orders-code"}, Hierarchy: []ProposedManager{}, PublicContracts: []ManagerPublicContract{}, Report: childReport}
	session, err = RecordManagerProposal(session, "orders-pass", childProposal)
	if err != nil {
		t.Fatal(err)
	}
	integration, err := BuildManagerIntegrationContext(session, "root-pass")
	if err != nil {
		t.Fatal(err)
	}
	childIteration, _ := findIteration(session, "orders-pass")
	childContext, err := BuildManagerReverseContext(session, "orders-pass")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateManagerAgentMapping(session, childIteration, childContext, rootID); err != nil {
		t.Fatalf("proposed child should use its nearest accepted ancestor's mapping: %v", err)
	}
	if err := validateManagerAgentMapping(session, childIteration, childContext, "unrelated-manager"); err == nil {
		t.Fatal("proposed child accepted an unrelated Manager runtime mapping")
	}
	rootIteration, _ = findIteration(session, "root-pass")
	ctxData, _, childCount, err := makeManagerRunContext(session, rootIteration, ManagerRunPhaseIntegrate, integration.Parent, &integration,
		ManagerRunContextBudget{RemainingStarts: 2, RemainingDuration: time.Minute, RemainingCostMicros: 100})
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := json.Marshal(ctxData)
	if err != nil {
		t.Fatal(err)
	}
	if childCount != 1 || !strings.Contains(string(serialized), "The selected module defines order behavior.") || strings.Contains(string(serialized), "UNASSIGNED-PRIVATE-EVIDENCE-SENTINEL") {
		t.Fatalf("integration context omitted child report or leaked unassigned evidence: childCount=%d context=%s", childCount, serialized)
	}
	artifacts, err := managerRunArtifacts(session, integration.Parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].Path != "evidence/orders-doc.txt" || strings.Contains(string(artifacts[0].Content), "UNASSIGNED-PRIVATE-EVIDENCE-SENTINEL") {
		t.Fatalf("parent request artifacts do not contain only parent-assigned evidence: %+v", artifacts)
	}
}

func managerRunFixture(t *testing.T) (string, string, BrownfieldSession, string) {
	t.Helper()
	root, discovery, _, target := distillationDiscovery(t)
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	session, err = BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "root-pass", ManagerID: session.TargetContext.RootManagerID,
		EvidenceIDs: []string{"implementation"}, DelegationEvidenceIDs: []string{}, Purpose: "Map selected source behavior", Review: "review-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBrownfieldSession(root, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	return root, target.Root, session, "root-pass"
}

func managerRunTestConfig(t *testing.T, counterPath string) agentexec.Config {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	allow := []string{managerRunHelperEnv, managerRunCountEnv, managerRunModeEnv}
	t.Setenv(managerRunHelperEnv, "1")
	t.Setenv(managerRunCountEnv, counterPath)
	t.Setenv(managerRunModeEnv, "")
	return agentexec.Config{Command: executable, Args: []string{"-test.run=^TestManagerRunExecutorHelper$"}, Model: "fixture-manager-model",
		ModelOptions: json.RawMessage(`{"temperature":0}`), ProviderVersion: "fixture-agentexec-v1", EnvironmentAllowlist: &allow,
		Timeout: 8 * time.Second, MaxStdoutBytes: 2 << 20, MaxStderrBytes: 128 << 10}
}

func managerRunTestLimits() ManagerRunLimits {
	return ManagerRunLimits{MaxStarts: 4, MaxRetries: 1, MaxDuration: time.Minute, MaxCostMicros: 1_000,
		InputPriceMicrosPerMillion: 10_000, OutputPriceMicrosPerMillion: 10_000, MaxTimeout: 10 * time.Second,
		MaxStdoutBytes: 2 << 20, MaxStderrBytes: 128 << 10, TempParent: os.TempDir()}
}

func TestManagerRunLimitsAcceptProductDefaultStartsAndFourHourJob(t *testing.T) {
	root := t.TempDir()
	config := managerRunTestConfig(t, filepath.Join(t.TempDir(), "counter"))
	config.Timeout = time.Hour
	limits := managerRunTestLimits()
	limits.MaxStarts = 256
	limits.MaxDuration = 4 * time.Hour
	limits.MaxTimeout = time.Hour
	limits.PrivateLogDirectory = os.TempDir()
	if err := validateManagerRunLimits(root, config, limits); err != nil {
		t.Fatalf("product default manager bounds should validate without starting an agent: %v", err)
	}
}

func ptrManagerRunLimits(value ManagerRunLimits) *ManagerRunLimits { return &value }
