package projectapp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

const (
	managerRunHelperEnv = "MARKITECT_APP_MANAGER_RUN_TEST_HELPER"
	managerRunCountEnv  = "MARKITECT_APP_MANAGER_RUN_TEST_COUNT"
	managerRunModeEnv   = "MARKITECT_APP_MANAGER_RUN_TEST_MODE"
)

func brownfieldEngineInt64(n int64) *int64 { return &n }
func TestBrownfieldEngineExecutorHelper(t *testing.T) {
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
	var kind struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(invocation.Request.Context, &kind)
	if kind.Kind == "projectrun-review/v1" {
		var reviewContext struct {
			Strictness projectconfig.StrictnessProfile `json:"strictness"`
		}
		if err := json.Unmarshal(invocation.Request.Context, &reviewContext); err != nil || reviewContext.Strictness.Counterexamples != 3 || len(reviewContext.Strictness.Evidence) != 2 {
			os.Exit(35)
		}
		usage := &agentexec.Usage{Source: "provider-reported", InputTokens: brownfieldEngineInt64(10), OutputTokens: brownfieldEngineInt64(5)}
		response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce, InputDigest: invocation.InputDigest, Role: agentexec.RoleExecutor, Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}, ReportJSON: json.RawMessage(`{"status":"pass","summary":"The exact report matches supplied evidence and accepted context.","findings":[]}`), Usage: usage}
		if os.Getenv(managerRunModeEnv) == "review-fail" {
			response.ReportJSON = json.RawMessage(`{"status":"fail","summary":"Evidence does not justify this proposal.","findings":[{"path":"evidence/implementation.txt","expectation":"Ground the exact statement","grounding":"evidence:implementation"}]}`)
		}
		_ = json.NewEncoder(os.Stdout).Encode(response)
		os.Exit(0)
	}
	var contextData struct {
		Phase          string `json:"phase"`
		ManagerContext struct {
			Manager  projectadoption.DistillationTargetManager `json:"manager"`
			Evidence []projectadoption.ManagerReverseEvidence  `json:"evidence"`
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
	draft := projectadoption.DistillationDraft{
		Claims: []projectadoption.DistillationDraftClaim{{ID: claimID, ScopeID: scopeID, Kind: "observation", Method: "static-source", Statement: "The selected source defines order behavior.",
			Evidence: []projectadoption.EvidenceRef{{EvidenceID: evidence.EvidenceID, StartLine: line, EndLine: line, Excerpt: excerpt}}, Uncertainty: []string{}, RuntimeObservationJSON: ""}},
		Terms: []projectadoption.Term{}, Contradictions: []projectadoption.Contradiction{}, Questions: []projectadoption.DistillationDraftQuestion{},
		Scopes: []projectadoption.DistillationDraftScope{{ID: scopeID, Name: "Order behavior", ParentID: "", ClaimIDs: []string{claimID}, OwnerCandidate: ""}},
		Proposal: projectadoption.ModelProposal{Goal: "Represent observed order behavior", Files: []projectadoption.ProposedFile{{ScopeID: scopeID, Path: ".markitect/model/orders/statement.yaml",
			Content: "apiVersion: " + projectvalue.APIVersion + "\nkind: Statement\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Order behavior\nspec:\n  category: concept\n  description: Order behavior\n  public: false\n  uses: []\n  requires: []\n"}}},
	}
	// These modes return a child that passes proposal checks but that the
	// session ledger rejects: an invalid ID, or an accepted Manager ID whose
	// fixed parent differs.
	hierarchy := []projectadoption.ProposedManager{}
	parentID := contextData.ManagerContext.Manager.ID
	switch os.Getenv(managerRunModeEnv) {
	case "child-proposal":
		hierarchy = []projectadoption.ProposedManager{{ID: "orders-manager", Name: "Orders Manager", Purpose: "Model order behavior", ParentID: parentID, EvidenceIDs: []string{evidence.EvidenceID}, DelegationEvidenceIDs: []string{}}}
	case "invalid-child-id":
		hierarchy = []projectadoption.ProposedManager{{ID: "Orders_Manager", Name: "Orders Manager", Purpose: "Model order behavior", ParentID: parentID, EvidenceIDs: []string{evidence.EvidenceID}, DelegationEvidenceIDs: []string{}}}
	case "accepted-child-mismatch":
		hierarchy = []projectadoption.ProposedManager{{ID: parentID, Name: "Orders Manager", Purpose: "Model order behavior", ParentID: parentID, EvidenceIDs: []string{evidence.EvidenceID}, DelegationEvidenceIDs: []string{}}}
	}
	var report any
	if contextData.Phase == projectadoption.ManagerRunPhasePropose {
		report = projectadoption.ManagerProposalDraft{Report: draft, Hierarchy: hierarchy, PublicContracts: []projectadoption.ManagerPublicContract{}}
	} else {
		report = projectadoption.ManagerIntegrationDraft{Report: draft, Conflicts: []projectadoption.SessionConflict{}}
	}
	reportBytes, _ := json.Marshal(report)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: brownfieldEngineInt64(10), OutputTokens: brownfieldEngineInt64(5), ToolCalls: brownfieldEngineInt64(0)}
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

type brownfieldEngineFixture struct {
	sourceRoot, targetRoot, counter string
	session                         projectadoption.BrownfieldSession
	operation                       BrownfieldRunOperation
	operations                      Operations
}

func newBrownfieldEngineFixture(t *testing.T, review bool) brownfieldEngineFixture {
	t.Helper()
	targetRoot := makePlanFixture(t)
	runtime, err := projectconfig.LoadRuntime(targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	counter := filepath.Join(t.TempDir(), "calls.txt")
	t.Setenv(managerRunHelperEnv, "1")
	t.Setenv(managerRunCountEnv, counter)
	t.Setenv(managerRunModeEnv, "")
	for id, agent := range runtime.Agents {
		agent.Command = executable
		agent.Args = []string{"-test.run=^TestBrownfieldEngineExecutorHelper$"}
		agent.Environment = []string{managerRunHelperEnv, managerRunCountEnv, managerRunModeEnv}
		agent.Timeout = projectconfig.Duration(8 * time.Second)
		agent.MaxStdoutBytes = 2 << 20
		agent.MaxStderrBytes = 128 << 10
		agent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 10000, OutputMicrosPerMillion: 10000}
		runtime.Agents[id] = agent
	}
	runtime.Limits.MaxStarts = 8
	runtime.Limits.MaxRetries = 2
	runtime.Limits.MaxDuration = projectconfig.Duration(2 * time.Minute)
	if review {
		runtime.Strictness = &projectconfig.StrictnessConfig{Default: projectconfig.StrictnessProfile{Evidence: []string{"exact report"}, Counterexamples: 1}, Managers: map[string]projectconfig.StrictnessProfile{}}
		runtime.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{}, MaxRounds: 2, MaxManagerRounds: 3}
		for id, agent := range runtime.Agents {
			agent.Model = "fixture-independent-reviewer"
			runtime.Review.Agents[id] = agent
			runtime.Strictness.Managers[id] = projectconfig.StrictnessProfile{Evidence: []string{"selected source"}, Counterexamples: 2}
		}
	}
	encoded, err := json.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetRoot, filepath.FromSlash(projectconfig.RuntimePath)), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, targetRoot, "add", projectconfig.RuntimePath)
	runGit(t, targetRoot, "commit", "-m", "configure finite Brownfield fixture")
	target, err := projectwork.Load(targetRoot, gitOutput(t, targetRoot, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	sourceRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sourceRoot, "src", "orders"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "src", "orders", "cancel.go"), []byte("package orders\nfunc Cancel() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, sourceRoot, "init", "-b", "brownfield-source")
	runGit(t, sourceRoot, "config", "user.email", "brownfield@example.test")
	runGit(t, sourceRoot, "config", "user.name", "Brownfield Engine Test")
	runGit(t, sourceRoot, "add", ".")
	runGit(t, sourceRoot, "commit", "-m", "fixed selected evidence")
	discovery, err := projectadoption.Discover(sourceRoot, projectadoption.DiscoveryRequest{APIVersion: projectadoption.DiscoveryVersion, ID: "engine-source", Purpose: "Model the selected order behavior", Review: "human-evidence-review", Commit: gitOutput(t, sourceRoot, "rev-parse", "HEAD"), ScopeRoots: []string{"src/orders"}, Selected: []projectadoption.SelectedPath{{ID: "implementation", Path: "src/orders/cancel.go", Reason: "Selected implementation", Basis: "code"}}, Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := projectadoption.StartBrownfieldSession(sourceRoot, target, discovery, []projectadoption.ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	session, err = projectadoption.BeginReverseIteration(sourceRoot, target, session, projectadoption.ReverseIterationRequest{ID: "root-pass", ManagerID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"implementation"}, DelegationEvidenceIDs: []string{}, Purpose: "Map exactly the selected source behavior", Review: "human-manager-review"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(sourceRoot, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	operations := NewOperations()
	operations.Invoker = projectexecutor.ProcessInvoker{}
	operation := BrownfieldRunOperation{Root: targetRoot, SourceRoot: sourceRoot, SessionID: session.ID, Request: BrownfieldManagerRunInput{IterationID: "root-pass", Phase: projectadoption.ManagerRunPhasePropose, AgentManagerID: session.TargetContext.RootManagerID}}
	return brownfieldEngineFixture{sourceRoot: sourceRoot, targetRoot: targetRoot, counter: counter, session: session, operation: operation, operations: operations}
}
func previewAndRunBrownfield(t *testing.T, f brownfieldEngineFixture, operation BrownfieldRunOperation) (BrownfieldManagerRunOutput, error) {
	t.Helper()
	operation.Write = false
	operation.ExpectedDigest = ""
	preview, err := f.operations.RunBrownfield(context.Background(), operation)
	if err != nil {
		t.Fatal(err)
	}
	operation.Write = true
	operation.ExpectedDigest = preview.PreviewDigest
	return f.operations.RunBrownfield(context.Background(), operation)
}

func TestBrownfieldEngineOneCallPerStageAndCompletedReplay(t *testing.T) {
	f := newBrownfieldEngineFixture(t, false)
	proposal, err := previewAndRunBrownfield(t, f, f.operation)
	if err != nil || proposal.Proposal == nil || proposal.Attempt == nil || proposal.Attempt.Status != "accepted" {
		t.Fatalf("App proposal did not attach exact Runtime terminal: %+v %v", proposal, err)
	}
	again, err := previewAndRunBrownfield(t, f, f.operation)
	if err != nil || again.Proposal == nil {
		t.Fatalf("completed proposal recovery failed: %v", err)
	}
	operation := f.operation
	operation.Request.Phase = projectadoption.ManagerRunPhaseIntegrate
	integrated, err := previewAndRunBrownfield(t, f, operation)
	if err != nil || integrated.Integration == nil || integrated.Attempt.Status != "accepted" {
		t.Fatalf("App integration failed: %+v %v", integrated, err)
	}
	if _, err := previewAndRunBrownfield(t, f, operation); err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(f.counter)
	if err != nil || string(calls) != "xx" {
		t.Fatalf("stage replay dispatched again: %q %v", calls, err)
	}
	if _, err := os.Stat(filepath.Join(f.sourceRoot, projectadoption.BrownfieldSessionPath, f.session.ID, "manager-runs.json")); !os.IsNotExist(err) {
		t.Fatal("App created another attempt authority")
	}
}

func TestBrownfieldEngineConfiguredReviewBeforeSessionCAS(t *testing.T) {
	f := newBrownfieldEngineFixture(t, true)
	preview, err := f.operations.RunBrownfield(context.Background(), f.operation)
	if err != nil || preview.Preview.Review == nil || !preview.Preview.Review.Enabled || preview.Preview.Review.MaxManagerRounds != 3 {
		t.Fatalf("review policy missing from exact preview: %v", err)
	}
	output, err := previewAndRunBrownfield(t, f, f.operation)
	if err != nil || output.Proposal == nil || output.Attempt.Status != "accepted" {
		t.Fatalf("configured report review did not permit Session CAS: %+v %v", output, err)
	}
	calls, err := os.ReadFile(f.counter)
	if err != nil || string(calls) != "xx" {
		t.Fatalf("Manager and independent reviewer did not run once each: %q %v", calls, err)
	}
}

func TestBrownfieldEngineRecoversCompletionBeforeSessionCAS(t *testing.T) {
	f := newBrownfieldEngineFixture(t, false)
	runtime, err := projectconfig.LoadRuntime(f.targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := projectconfig.ReadOnlyAgent(runtime, f.operation.Request.AgentManagerID)
	if err != nil {
		t.Fatal(err)
	}
	config, err := agent.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	limits := projectadoption.ManagerRunLimits{MaxStarts: runtime.Limits.MaxStarts, MaxRetries: runtime.Limits.MaxRetries, MaxDuration: time.Duration(runtime.Limits.MaxDuration), MaxCostMicros: runtime.Limits.MaxCostMicros, InputPriceMicrosPerMillion: agent.Pricing.InputMicrosPerMillion, OutputPriceMicrosPerMillion: agent.Pricing.OutputMicrosPerMillion, MaxTimeout: config.Timeout, MaxStdoutBytes: config.MaxStdoutBytes, MaxStderrBytes: config.MaxStderrBytes}
	unlock, err := projectadoption.AcquireManagerStageLease(f.sourceRoot, f.session.ID)
	if err != nil {
		t.Fatal(err)
	}
	prepared, spec, _, preview, err := f.operations.prepareBrownfieldRuntimeStage(f.sourceRoot, f.operation, runtime, agent, config, limits, "", f.operations.Invoker)
	if err != nil {
		t.Fatal(err)
	}
	host := f.operations.brownfieldStageHost(f.sourceRoot, f.operation, f.operations.Invoker)
	manager, err := projectrun.RunManagerStage(context.Background(), host, f.operations.Invoker, f.sourceRoot, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	proposal, integration, err := projectadoption.AcceptManagerStageDraft(prepared.Session, f.operation.Request.IterationID, f.operation.Request.Phase, prepared.Preview.ManagerID, manager.Result.Response.ReportJSON, config, manager.Result.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := projectrun.CompleteManagerStage(f.sourceRoot, manager.RunID, manager.AttemptID, manager.StateDigest, projectrun.ManagerStageValidation{Accepted: true, TypedResultDigest: brownfieldStageValueDigest(struct {
		Proposal    *projectadoption.ManagerProposal
		Integration *projectadoption.ManagerIntegration
	}{proposal, integration})})
	if err != nil || completed.Status != "accepted" {
		t.Fatal(err)
	}
	unlock() // Simulated crash: Runtime terminal accepted, Session CAS did not happen.
	session, err := projectadoption.LoadBrownfieldSession(f.sourceRoot, f.session.ID)
	if err != nil || session.Digest != f.session.Digest {
		t.Fatal("fixture unexpectedly attached Session")
	}
	recovered, err := previewAndRunBrownfield(t, f, f.operation)
	if err != nil || recovered.Proposal == nil || recovered.Attempt.ID != manager.AttemptID {
		t.Fatalf("same Runtime completion did not recover Session CAS: %+v %v", recovered, err)
	}
	calls, _ := os.ReadFile(f.counter)
	if string(calls) != "x" {
		t.Fatalf("CAS recovery replayed provider: %q", calls)
	}
}

func TestBrownfieldEnginePricedInvalidDraftRequiresExplicitRetry(t *testing.T) {
	for _, mode := range []string{"invalid-report", "invalid-child-id", "accepted-child-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			f := newBrownfieldEngineFixture(t, false)
			t.Setenv(managerRunModeEnv, mode)
			failed, err := previewAndRunBrownfield(t, f, f.operation)
			if err == nil || failed.Attempt == nil || failed.Attempt.Status != "rejected" || failed.Attempt.Execution == nil {
				t.Fatalf("invalid typed result did not retain priced rejection: %+v %v", failed, err)
			}
			if _, err := previewAndRunBrownfield(t, f, f.operation); err == nil {
				t.Fatal("invalid draft automatically replayed")
			}
			calls, _ := os.ReadFile(f.counter)
			if string(calls) != "x" {
				t.Fatal("failed draft replayed")
			}
			t.Setenv(managerRunModeEnv, "")
			operation := f.operation
			operation.Request.RetryOfAttemptID = failed.Attempt.ID
			retried, err := previewAndRunBrownfield(t, f, operation)
			if err != nil || retried.Proposal == nil || retried.Attempt.ID == failed.Attempt.ID {
				t.Fatalf("explicit priced retry failed: %+v %v", retried, err)
			}
			calls, _ = os.ReadFile(f.counter)
			if string(calls) != "xx" {
				t.Fatal("explicit retry did not dispatch exactly once")
			}
		})
	}
}

type brownfieldAuditProbeInvoker struct {
	t     *testing.T
	inner projectrun.Invoker
}

func (i brownfieldAuditProbeInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return i.inner.Fingerprint(config)
}
func (i brownfieldAuditProbeInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	for _, root := range options.InputRoots {
		_ = filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				if _, err := os.ReadFile(name); err != nil {
					relative, _ := filepath.Rel(root, name)
					i.t.Logf("exact input audit read failure at %s: %v", relative, err)
				}
			}
			return nil
		})
	}
	return i.inner.Run(ctx, config, request, options)
}
func TestBrownfieldEngineInputAuditReadsHeldSessionLock(t *testing.T) {
	f := newBrownfieldEngineFixture(t, false)
	f.operations.Invoker = brownfieldAuditProbeInvoker{t: t, inner: f.operations.Invoker}
	_, err := previewAndRunBrownfield(t, f, f.operation)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBrownfieldEngineFailedConfiguredReviewCannotAttachOrReworkAutomatically(t *testing.T) {
	f := newBrownfieldEngineFixture(t, true)
	t.Setenv(managerRunModeEnv, "review-fail")
	failed, err := previewAndRunBrownfield(t, f, f.operation)
	if err == nil || failed.Status != "review-failed" || failed.Attempt == nil || failed.Attempt.Status != "rejected" {
		t.Fatalf("failed review attached or lost its verdict: %+v %v", failed, err)
	}
	session, err := projectadoption.LoadBrownfieldSession(f.sourceRoot, f.session.ID)
	if err != nil || session.Digest != f.session.Digest {
		t.Fatal("failed review performed Session CAS")
	}
	if _, err := previewAndRunBrownfield(t, f, f.operation); err == nil {
		t.Fatal("failed review automatically reworked/replayed Manager")
	}
	calls, _ := os.ReadFile(f.counter)
	if string(calls) != "xx" {
		t.Fatalf("failed review launched more than one Manager and one reviewer: %q", calls)
	}
}

func TestBrownfieldEngineLeafIntegrationUnderIntegratedParentRetainsFailure(t *testing.T) {
	f := newBrownfieldEngineFixture(t, false)
	t.Setenv(managerRunModeEnv, "child-proposal")
	rootProposal, err := previewAndRunBrownfield(t, f, f.operation)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(managerRunModeEnv, "")
	session, err := projectadoption.LoadBrownfieldSession(f.sourceRoot, f.session.ID)
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(f.targetRoot, session.Target.Revision)
	if err != nil {
		t.Fatal(err)
	}
	priorDigest := session.Digest
	session, err = projectadoption.BeginReverseIteration(f.sourceRoot, target, session, projectadoption.ReverseIterationRequest{ID: "orders-pass", ParentIterationID: "root-pass", ManagerID: rootProposal.Proposal.Hierarchy[0].ID, EvidenceIDs: []string{"implementation"}, DelegationEvidenceIDs: []string{}, Purpose: "Model delegated order evidence", Review: "human-child-review"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(f.sourceRoot, session, priorDigest); err != nil {
		t.Fatal(err)
	}
	childOperation := f.operation
	childOperation.Request.IterationID = "orders-pass"
	if _, err := previewAndRunBrownfield(t, f, childOperation); err != nil {
		t.Fatal(err)
	}
	rootIntegration := f.operation
	rootIntegration.Request.Phase = projectadoption.ManagerRunPhaseIntegrate
	if _, err := previewAndRunBrownfield(t, f, rootIntegration); err != nil {
		t.Fatal(err)
	}
	childOperation.Request.Phase = projectadoption.ManagerRunPhaseIntegrate
	failed, err := previewAndRunBrownfield(t, f, childOperation)
	if err == nil || failed.Attempt == nil || failed.Attempt.Status != "rejected" || failed.Attempt.Execution == nil {
		t.Fatalf("already integrated parent accepted a new child integration: %+v %v", failed, err)
	}
	if _, err := previewAndRunBrownfield(t, f, childOperation); err == nil {
		t.Fatal("unrecordable child integration automatically replayed")
	}
	calls, _ := os.ReadFile(f.counter)
	if string(calls) != "xxxx" {
		t.Fatalf("child integration failure did not preserve exactly one attempt: %q", calls)
	}
}
