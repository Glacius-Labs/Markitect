package projectrun

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

const operationsVerifyHelperEnv = "MARKITECT_OPERATIONS_VERIFY_HELPER"
const operationsVerifyLogEnv = "MARKITECT_OPERATIONS_VERIFY_LOG"
const operationsTaskHelperEnv = "MARKITECT_OPERATIONS_TASK_HELPER"
const operationsSourceChangeEnv = "MARKITECT_OPERATIONS_SOURCE_CHANGE"

// This subprocess implements the bounded typed audit protocol without invoking
// a model. It records every Manager reached and deliberately fails Inventory
// when its supplied snapshot contains the drift marker.
func TestOperationsFullVerifyProcess(t *testing.T) {
	if path := os.Getenv(operationsVerifyLogEnv); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			_ = json.NewEncoder(file).Encode(map[string]string{"helperStarted": os.Getenv(operationsVerifyHelperEnv)})
			_ = file.Close()
		}
	}
	if os.Getenv(operationsVerifyHelperEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		t.Fatal(err)
	}
	var discriminator struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &discriminator); err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv(operationsVerifyLogEnv); path != "" {
		file, openErr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if openErr != nil {
			t.Fatal(openErr)
		}
		_ = json.NewEncoder(file).Encode(map[string]string{"kind": discriminator.Kind})
		_ = file.Close()
	}
	if discriminator.Kind == "projectrun-review/v1" {
		report, err := json.Marshal(reviewResponse{Status: "pass", Summary: "no-op is supported by the complete accepted scope", Findings: []reviewFindingResponse{}})
		if err != nil {
			t.Fatal(err)
		}
		usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
		encoded, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
			Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
			CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
			ReportJSON: report, Uncertainty: []string{}, Usage: usage})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = os.Stdout.Write(append(encoded, '\n'))
		os.Exit(0)
	}
	var request struct {
		Kind    string `json:"kind"`
		Manager struct {
			Manager struct {
				ID string `json:"id"`
			} `json:"manager"`
		} `json:"manager"`
		Subjects   []string          `json:"requiredSubjects"`
		Strictness StrictnessProfile `json:"strictness"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &request); err != nil || request.Kind != "projectrun-full-verify/v1" {
		t.Fatalf("unexpected full-verify request: %s (%v)", invocation.Request.Context, err)
	}
	if path := os.Getenv(operationsVerifyLogEnv); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(file).Encode(map[string]string{"managerId": request.Manager.Manager.ID})
		_ = file.Close()
	}
	status := "pass"
	assessments := make([]FullAssessment, 0, len(request.Subjects))
	findings := []string{}
	if request.Manager.Manager.ID == e2eManagerID("inventory", "inventory") {
		for _, artifact := range invocation.Request.Artifacts {
			if strings.Contains(string(artifact.Content), "UNIMPACTED_DRIFT") {
				status = "fail"
				findings = append(findings, "unimpacted inventory artifact contains drift")
				break
			}
		}
	}
	for _, subject := range request.Subjects {
		outcome := "pass"
		if status == "fail" && len(assessments) == 0 {
			outcome = "fail"
		}
		assessments = append(assessments, FullAssessment{Subject: subject, Outcome: outcome, Detail: "bounded snapshot audit"})
	}
	counterexamples := make([]FullCounterexample, 0, request.Strictness.Counterexamples)
	for i := 0; i < request.Strictness.Counterexamples; i++ {
		ref := "manager:" + request.Manager.Manager.ID
		if len(request.Subjects) > 0 {
			ref = request.Subjects[i%len(request.Subjects)]
		}
		counterexamples = append(counterexamples, FullCounterexample{Expected: "accepted obligation " + string(rune('a'+i)), Observed: "fixture inspection evidence " + string(rune('a'+i)), EvidenceRefs: []string{ref}})
	}
	report, err := json.Marshal(fullAuditResponse{Status: status, Summary: "bounded fixture audit", Assessments: assessments, Findings: findings, Counterexamples: counterexamples})
	if err != nil {
		t.Fatal(err)
	}
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	response, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: report, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(response, '\n'))
	os.Exit(0)
}

// This helper returns a justified no-op from each leaf Manager while the root
// still delegates and performs an explicit integration phase.
func TestOperationsNoOpTaskProcess(t *testing.T) {
	if os.Getenv(operationsTaskHelperEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		t.Fatal(err)
	}
	var request struct {
		Kind           string   `json:"kind"`
		Phase          string   `json:"phase"`
		DirectChildren []string `json:"directChildren"`
		Manager        struct {
			Manager struct {
				ID string `json:"id"`
			} `json:"manager"`
		} `json:"manager"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &request); err != nil || request.Kind != "projectrun-task/v1" {
		t.Fatalf("unexpected task request: %s (%v)", invocation.Request.Context, err)
	}
	response := TaskResponse{Status: "no-op", Summary: "The complete owned realization was reviewed; no warranted change was found.",
		Delegations: []Delegation{}, ReworkRequests: []ReworkRequest{}, Integrated: request.Phase == "integrate",
		Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	if request.Phase == "work" && request.Manager.Manager.ID == e2eManagerID("", "project-owner") {
		response.Status = "complete"
		for _, child := range request.DirectChildren {
			response.Delegations = append(response.Delegations, Delegation{ManagerID: child, Goal: "Review your complete owned responsibility and report whether a change is warranted."})
		}
	}
	if request.Phase == "integrate" {
		response.Status = "complete"
		response.Summary = "Reviewed child reports and integrated the no-op findings."
	}
	candidateFiles := []agentexec.CandidateFile{}
	if os.Getenv(operationsSourceChangeEnv) == "1" && request.Phase == "work" && request.Manager.Manager.ID == e2eManagerID("orders", "orders") {
		response.Status = "complete"
		response.Summary = "Improved the Orders implementation while preserving its accepted contract."
		candidateFiles = []agentexec.CandidateFile{{Path: "src/orders/implementation.txt", Mode: "0644", Content: "orders implementation v2\n"}}
	}
	report, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	encoded, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: candidateFiles, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: report, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(encoded, '\n'))
	os.Exit(0)
}

func configureOperationsNoOpRuntime(t *testing.T, root string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(operationsTaskHelperEnv, "1")
	t.Setenv(operationsVerifyHelperEnv, "1")
	t.Setenv(operationsVerifyLogEnv, filepath.Join(t.TempDir(), "operations-review.jsonl"))
	setupFullVerifyProcesses(t)
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		for id, agent := range runtime.Agents {
			agent.Command = executable
			agent.Args = []string{"-test.run=^TestOperationsNoOpTaskProcess$"}
			agent.Environment = append(agent.Environment, operationsTaskHelperEnv, operationsSourceChangeEnv)
			runtime.Agents[id] = agent
		}
		reviewers := make(map[string]Agent, len(runtime.Agents))
		for id, agent := range runtime.Agents {
			agent.Args = []string{"-test.run=^TestOperationsFullVerifyProcess$"}
			agent.Environment = append(agent.Environment, operationsVerifyHelperEnv, operationsVerifyLogEnv)
			reviewers[id] = agent
		}
		runtime.Review = &ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
		runtime.Strictness = &StrictnessConfig{Default: StrictnessProfile{Evidence: []string{"project checks reviewed"}, Counterexamples: 1},
			Managers: map[string]StrictnessProfile{e2eManagerID("orders", "orders"): {Evidence: []string{"orders contract review"}, Counterexamples: 1}}}
	})
}

func configureOperationsFullVerify(t *testing.T, root string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "operations-full-verify.jsonl")
	t.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(operationsVerifyHelperEnv, "1")
	t.Setenv(operationsVerifyLogEnv, logPath)
	setupFullVerifyProcesses(t)
	updateE2ERuntime(t, root, func(runtime *Runtime) {
		reviewers := make(map[string]Agent, len(runtime.Agents))
		for id, agent := range runtime.Agents {
			agent.Command = executable
			agent.Args = []string{"-test.run=^TestOperationsFullVerifyProcess$"}
			agent.Environment = append(agent.Environment, operationsVerifyHelperEnv, operationsVerifyLogEnv)
			reviewers[id] = agent
		}
		runtime.Review = &ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
	})
	return logPath
}

func TestTargetedPlanStillFullVerifiesUnimpactedManagerDrift(t *testing.T) {
	root := makeFullVerifyFixture(t)
	writeE2E(t, root, "src/inventory/implementation.txt", "UNIMPACTED_DRIFT\n")
	gitE2E(t, root, "add", "src/inventory/implementation.txt")
	gitE2E(t, root, "commit", "-m", "introduce unimpacted fixture drift")
	logPath := configureOperationsFullVerify(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot, PlanEdit: projectwork.PlanEdit}
	plan, err := Plan(host, root, head, PlanRequest{Goal: "Implement the requested Orders change", Managers: []string{e2eManagerID("orders", "orders")}})
	if err != nil {
		t.Fatalf("targeted Plan: %v", err)
	}
	if len(plan.Managers) != 2 || plan.Managers[0].ManagerID != e2eManagerID("", "project-owner") || plan.Managers[1].ManagerID != e2eManagerID("orders", "orders") {
		t.Fatalf("Plan should stay scoped to root and Orders: %+v", plan.Managers)
	}
	report, err := FullVerify(context.Background(), host, ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "failed" {
		t.Fatalf("full Verify did not reject the unimpacted drift: status=%s err=%v", report.Status, err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "inventory") {
		t.Fatalf("unimpacted Inventory was not independently audited: %s", data)
	}
}

func TestFullCoverageBlocksUnclassifiedOrdinaryFile(t *testing.T) {
	root := makeFullVerifyFixture(t)
	writeE2E(t, root, "unclassified.txt", "ordinary repository content\n")
	gitE2E(t, root, "add", "unclassified.txt")
	gitE2E(t, root, "commit", "-m", "add unclassified ordinary file")
	project, err := projectwork.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	if project.Coverage == nil || project.Coverage.Accounted {
		t.Fatalf("full coverage accepted an unclassified ordinary file: %+v", project.Coverage)
	}
	configureOperationsFullVerify(t, root)
	report, err := FullVerify(context.Background(), Host{Load: projectwork.Load}, ProcessInvoker{}, root, FullVerifyRequest{Revision: project.Revision})
	if err == nil || report.Status == "passed" {
		t.Fatalf("full Verify closed despite an unclassified ordinary file: report=%+v err=%v", report, err)
	}
}

func TestDismissedAcceptedModelEventStillLoadsForManagerContext(t *testing.T) {
	root := makeFullVerifyFixture(t)
	base := gitE2E(t, root, "rev-parse", "HEAD")
	statementPath := ".markitect/model/orders/statement.yaml"
	statement, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(statementPath)))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(statement), "Implement the orders artifact.", "Implement the orders artifact with audited invariants.", 1)
	if updated == string(statement) {
		t.Fatal("Orders statement fixture text was not found")
	}
	writeE2E(t, root, statementPath, updated)
	gitE2E(t, root, "add", statementPath)
	gitE2E(t, root, "commit", "-m", "accept Orders model clarification")
	revision := gitE2E(t, root, "rev-parse", "HEAD")
	bundle, err := projectbriefing.Generate(root, base, revision, projectbriefing.Provenance{
		DecisionReference: "fixture-decision-17", Actor: "fixture-owner", Authority: "project-model-owner",
	})
	if err != nil {
		t.Fatalf("generate accepted-model briefing: %v", err)
	}
	state, digest, err := projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Briefings) != 0 || len(bundle.Events) == 0 {
		t.Fatalf("fixture did not create a new accepted event: prior=%d events=%d", len(state.Briefings), len(bundle.Events))
	}
	if _, err := projectbriefing.Write(root, bundle, digest); err != nil {
		t.Fatalf("write accepted-model briefing: %v", err)
	}
	ordersID := e2eManagerID("orders", "orders")
	var ordersEvent string
	for _, event := range bundle.Events {
		for _, managerID := range event.AffectedManagers {
			if managerID == ordersID {
				ordersEvent = event.ID
			}
		}
	}
	if ordersEvent == "" {
		t.Fatalf("accepted change did not affect Orders: %+v", bundle.Events)
	}
	_, digest, err = projectbriefing.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectbriefing.Dismiss(root, ordersEvent, ordersID, digest); err != nil {
		t.Fatalf("dismiss accepted event: %v", err)
	}
	current, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	context, err := managerBriefing(root, current.Report.ModelDigest, ordersID)
	if err != nil {
		t.Fatalf("load later Orders manager context: %v", err)
	}
	if len(context.Events) != 1 || context.Events[0].ID != ordersEvent {
		t.Fatalf("dismissal removed accepted event from later Manager context: %+v", context.Events)
	}
}

func TestCleanupAndReconcileNoOpVerifyFullTreeAndGuardedApply(t *testing.T) {
	for _, operation := range []string{OperationCleanup, OperationReconcile} {
		t.Run(operation, func(t *testing.T) {
			root := makeFullVerifyFixture(t)
			configureOperationsNoOpRuntime(t, root)
			head := gitE2E(t, root, "rev-parse", "HEAD")
			host := projectworkHost()
			plan, err := Plan(host, root, head, PlanRequest{Operation: operation, Goal: "Review the complete project and improve only if warranted", ExecuteAuthorized: true})
			if err != nil {
				t.Fatalf("plan %s: %v", operation, err)
			}
			if len(plan.Managers) != 3 || len(plan.Checks) != 2 {
				t.Fatalf("%s plan did not include every Manager and check: managers=%d checks=%d", operation, len(plan.Managers), len(plan.Checks))
			}
			ordersStrictness := plan.Strictness[e2eManagerID("orders", "orders")]
			if ordersStrictness.Counterexamples != 2 || !containsString(ordersStrictness.Evidence, "project checks reviewed") || !containsString(ordersStrictness.Evidence, "orders contract review") {
				t.Fatalf("%s bypassed or replaced additive Orders strictness: %+v", operation, ordersStrictness)
			}
			run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
			if err != nil || run.Status != StatusIntegrated {
				debug, _ := os.ReadFile(os.Getenv(operationsVerifyLogEnv))
				runtimeConfig, _ := LoadRuntime(root)
				reviewer := runtimeConfig.Review.Agents[e2eManagerID("inventory", "inventory")]
				t.Fatalf("%s no-op run did not integrate: status=%s err=%v reviewer trace=%s reviewer=%+v", operation, run.Status, err, debug, reviewer)
			}
			for _, task := range run.Tasks {
				if task.ManagerID != e2eManagerID("", "project-owner") && task.ReportStatus != "no-op" {
					t.Errorf("Manager %s did not report a justified no-op: %s", task.ManagerID, task.ReportStatus)
				}
			}
			verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, run.ID)
			if err != nil || verified.Status != "verified" {
				t.Fatalf("%s no-op did not pass checks and full Verify: status=%s err=%v scope=%s", operation, verified.Status, err, verified.VerificationScope)
			}
			preflight, err := PreflightApply(host, root, run.ID, run.Candidate.ID)
			if err != nil || len(preflight.Paths) != 0 {
				t.Fatalf("%s no-op preflight = %+v, %v; want guarded empty-path apply", operation, preflight, err)
			}
			untracked := filepath.Join(root, "untracked-ordinary.txt")
			if err := os.WriteFile(untracked, []byte("new ordinary file\n"), 0600); err != nil {
				t.Fatal(err)
			}
			_, staleErr := Apply(host, ProcessInvoker{}, root, ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
				ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree})
			if staleErr == nil {
				t.Fatal("Apply accepted a changed repository after no-op preflight")
			}
			if err := os.Remove(untracked); err != nil {
				t.Fatal(err)
			}
			preflight, err = PreflightApply(host, root, run.ID, run.Candidate.ID)
			if err != nil {
				t.Fatalf("fresh no-op preflight: %v", err)
			}
			applied, err := Apply(host, ProcessInvoker{}, root, ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
				ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree})
			if err != nil || applied.Status != "applied" || len(applied.Written) != 0 {
				t.Fatalf("%s verified no-op Apply = %+v, %v", operation, applied, err)
			}
		})
	}
}

func TestCleanupSourceChangeGeneratesAndAppliesExactVerifiedDocument(t *testing.T) {
	root := makeFullVerifyFixture(t)
	configureOperationsNoOpRuntime(t, root)
	t.Setenv(operationsSourceChangeEnv, "1")
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := projectworkHost()
	plan, err := Plan(host, root, head, PlanRequest{Operation: OperationCleanup,
		Goal: "Review the complete project and improve Orders where the accepted model permits it", ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("plan cleanup source change: %v", err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("integrate cleanup source change: status=%s err=%v", run.Status, err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	composed, err := snapshotWithCandidate(base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	project, err := host.FromSnapshot(root, composed)
	if err != nil {
		t.Fatal(err)
	}
	expectedDocument, err := projectwork.Document(project, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(project.Snapshot.Files["README.md"]) != expectedDocument {
		t.Fatal("integrated candidate README is not the exact Host-generated document for its source snapshot")
	}
	paths := candidateDeltaPaths(base.Snapshot, candidate)
	if !containsString(paths, "src/orders/implementation.txt") {
		t.Fatalf("candidate did not bind its implementation change: %v", paths)
	}
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, run.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("verify source and generated document together: status=%s err=%v", verified.Status, err)
	}
	preflight, err := PreflightApply(host, root, run.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("source-change preflight: %v", err)
	}
	if !containsString(preflight.Paths, "src/orders/implementation.txt") {
		t.Fatalf("guarded Apply omitted the exact verified implementation path: %+v", preflight.Paths)
	}
	applied, err := Apply(host, ProcessInvoker{}, root, ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree})
	if err != nil || applied.Status != "applied" {
		t.Fatalf("apply verified source and generated document: report=%+v err=%v", applied, err)
	}
	contents, err := os.ReadFile(filepath.Join(root, "src/orders/implementation.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "orders implementation v2\n" {
		t.Fatalf("applied Orders source = %q", contents)
	}
	final, err := host.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	finalDocument, err := projectwork.Document(final, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(final.Snapshot.Files["README.md"]) != finalDocument || finalDocument != expectedDocument {
		t.Fatal("applied README differs from the same Host-generated document verified in the candidate")
	}
}
