package projectadoption

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
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
			Content: "apiVersion: " + projectvalue.APIVersion + "\nkind: Statement\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Order behavior\nspec:\n  category: concept\n  description: Order behavior\n  public: false\n  uses: []\n  requires: []\n"}}},
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

func TestManagerRunLedgerChangeInvalidatesPreview(t *testing.T) {
	root, targetRoot, session, iterationID := managerRunFixture(t)
	counter := filepath.Join(t.TempDir(), "calls.txt")
	config, limits := managerRunTestConfig(t, counter), managerRunTestLimits()
	_, err := PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
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
	before, err := os.ReadFile(filepath.Join(ledgerDir, managerRunLedgerName))
	if err != nil {
		t.Fatal(err)
	}
	_, err = PrepareManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose, session.TargetContext.RootManagerID, "", "", config, limits, AgentExecManagerRunInvoker{})
	if unlock, leaseErr := AcquireManagerStageLease(root, session.ID); leaseErr == nil {
		unlock()
		t.Fatal("nonempty legacy ledger admitted Runtime lease")
	}
	after, readErr := os.ReadFile(filepath.Join(ledgerDir, managerRunLedgerName))
	if readErr != nil || string(before) != string(after) {
		t.Fatal("legacy bytes changed during failed admission")
	}
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
	_, err = PreviewManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose,
		session.TargetContext.RootManagerID, session.Digest, "", config, limits)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(ledgerDir, managerRunLedgerName))
	if err != nil {
		t.Fatal(err)
	}
	if unlock, err := AcquireManagerStageLease(root, session.ID); err == nil {
		unlock()
		t.Fatal("pending legacy attempt acquired Runtime lease")
	}
	if _, err := PrepareManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose, session.TargetContext.RootManagerID, "", "", config, limits, AgentExecManagerRunInvoker{}); err == nil {
		t.Fatal("pending legacy attempt prepared Runtime dispatch")
	}
	after, readErr := os.ReadFile(filepath.Join(ledgerDir, managerRunLedgerName))
	if readErr != nil || string(before) != string(after) {
		t.Fatal("pending historical evidence changed")
	}
	if calls, _ := os.ReadFile(counter); len(calls) != 0 {
		t.Fatalf("pending attempt was implicitly replayed: %q", calls)
	}
}

func TestLegacyManagerRunSealedBudgetCannotStartRuntimeAuthority(t *testing.T) {
	root, targetRoot, session, iterationID := managerRunFixture(t)
	config := managerRunTestConfig(t, filepath.Join(t.TempDir(), "calls.txt"))
	limits := managerRunTestLimits()
	dir, err := sessionDirectory(root, session.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := loadManagerRunLedger(dir, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	ledger.Budget = ptrManagerRunLimits(budgetProjection(limits))
	sealManagerRunLedger(&ledger)
	if err := writeManagerRunLedger(dir, ledger); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, managerRunLedgerName))
	if err != nil {
		t.Fatal(err)
	}
	limits.MaxStarts = 8
	if _, err := PrepareManagerStage(root, targetRoot, session.ID, iterationID, ManagerRunPhasePropose, session.TargetContext.RootManagerID, "", "", config, limits, AgentExecManagerRunInvoker{}); err == nil {
		t.Fatal("sealed empty legacy budget was refilled/imported")
	}
	if unlock, err := AcquireManagerStageLease(root, session.ID); err == nil {
		unlock()
		t.Fatal("sealed empty legacy budget acquired Runtime lease")
	}
	after, err := os.ReadFile(filepath.Join(dir, managerRunLedgerName))
	if err != nil || string(before) != string(after) {
		t.Fatal("sealed budget bytes changed")
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
	repo, sourceCommit := committedRepository(t, map[string]string{
		"src/orders.go":          "package orders\nfunc Order() {}\n",
		"docs/orders.md":         "Orders are managed by the order module.\n",
		"private/unassigned.txt": "UNASSIGNED-PRIVATE-EVIDENCE-SENTINEL\n",
	})
	root := repo.Dir
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "manager-integration-context", Purpose: "Map order behavior", Review: "review-context",
		Commit: sourceCommit, ScopeRoots: []string{"."}, Selected: []SelectedPath{
			{ID: "orders-code", Path: "src/orders.go", Reason: "Implementation evidence", Basis: "code"},
			{ID: "orders-doc", Path: "docs/orders.md", Reason: "Documented intent", Basis: "documentation"},
			{ID: "unassigned", Path: "private/unassigned.txt", Reason: "Retained but not assigned", Basis: "code"},
		}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	targetRepo, _ := committedRepository(t, map[string]string{"README.md": "target\n"})
	targetRoot := targetRepo.Dir
	targetRepo.Git("checkout", "-b", "codex/manager-run-context-target")
	if _, err := projectwork.Init(targetRoot, "Manager context target", true); err != nil {
		t.Fatal(err)
	}
	revision := targetRepo.Commit("initialize target")
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

// Historical journal fixture writer; production contains no ledger writer.
func writeManagerRunLedger(dir string, ledger ManagerRunLedger) error {
	if err := ensureNoIncompleteAtomicWrite(dir, managerRunLedgerName); err != nil {
		return err
	}
	data, err := encodeClosedJSON(ledger)
	if err != nil {
		return err
	}
	temp := filepath.Join(dir, ".manager-runs.json.tmp")
	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(temp)
		return err
	}
	path := filepath.Join(dir, managerRunLedgerName)
	if err := atomicReplaceFile(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}
