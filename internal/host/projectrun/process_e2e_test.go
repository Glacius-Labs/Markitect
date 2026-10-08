package projectrun

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

const (
	e2eExecutorEnv = "MARKITECT_E2E_EXECUTOR"
	e2eCheckEnv    = "MARKITECT_E2E_CHECK"
	e2eLogEnv      = "MARKITECT_E2E_LOG"
	e2eBehaviorEnv = "MARKITECT_E2E_BEHAVIOR"
	e2eRootEnv     = "MARKITECT_E2E_ROOT"
)

// These process entry points are re-executed by ProcessInvoker and by the
// repository's declared independent checks. They deliberately exercise the
// same JSON/stdin/stdout boundary as a provider adapter without using a mock
// Invoker or a paid provider.
func TestProjectRunExecutorProcess(t *testing.T) {
	if os.Getenv(e2eExecutorEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		processExit(2, "read invocation: "+err.Error())
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		processExit(2, "decode invocation: "+err.Error())
	}
	var discriminator struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &discriminator); err != nil {
		processExit(2, "decode invocation context: "+err.Error())
	}
	if discriminator.Kind == "projectrun-review/v1" {
		runE2EReview(invocation)
		return
	}
	var contextPayload struct {
		Phase          string   `json:"phase"`
		DirectChildren []string `json:"directChildren"`
		Manager        struct {
			Manager struct {
				ID      string   `json:"id"`
				Purpose string   `json:"purpose"`
				Owns    []string `json:"owns"`
			} `json:"manager"`
		} `json:"manager"`
		GlobalGoal        string                `json:"globalGoal"`
		OwnTask           string                `json:"ownTask"`
		RepairDiagnostic  string                `json:"repairDiagnostic"`
		RepairRound       int                   `json:"repairRound"`
		RepairChecks      []RepairCheckFeedback `json:"repairChecks"`
		CandidateDigest   string                `json:"candidateDigest"`
		AllowedWritePaths []string              `json:"allowedWritePaths"`
		ChildReports      []struct {
			ManagerID string `json:"managerId"`
			Summary   string `json:"summary"`
			Status    string `json:"status"`
		} `json:"childReports"`
		ResponseSchema json.RawMessage `json:"responseSchema"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextPayload); err != nil {
		processExit(2, "decode project context: "+err.Error())
	}
	if len(contextPayload.ResponseSchema) == 0 || contextPayload.Manager.Manager.ID == "" || contextPayload.Phase == "" {
		processExit(2, "request omitted manager, phase, or typed response schema")
	}
	if err := appendE2ELog(os.Getenv(e2eLogEnv), map[string]any{
		"phase": contextPayload.Phase, "managerId": contextPayload.Manager.Manager.ID,
		"globalGoal": contextPayload.GlobalGoal, "ownTask": contextPayload.OwnTask,
		"repairDiagnostic": contextPayload.RepairDiagnostic, "repairRound": contextPayload.RepairRound,
		"repairChecks": contextPayload.RepairChecks, "candidateDigest": contextPayload.CandidateDigest,
		"allowedWritePaths": contextPayload.AllowedWritePaths,
		"directChildren":    contextPayload.DirectChildren, "childReports": contextPayload.ChildReports,
		"schema": contextPayload.ResponseSchema, "sourceRevision": invocation.Request.SourceRevision,
		"modelDigest": invocation.Request.ModelDigest, "modulePin": invocation.Request.ModulePin,
		"projectionId": invocation.Request.ProjectionID, "scopeIds": invocation.Request.ScopeIDs,
		"policyIds": invocation.Request.PolicyIDs, "artifacts": invocation.Request.Artifacts,
		"managerPurpose": contextPayload.Manager.Manager.Purpose, "managerOwns": contextPayload.Manager.Manager.Owns,
		"contextDigest": rawContentDigest(invocation.Request.Context), "inputDigest": invocation.InputDigest,
	}); err != nil {
		processExit(2, "record invocation: "+err.Error())
	}
	privateLog, err := os.OpenFile(os.Getenv("MARKITECT_AGENT_PRIVATE_LOG"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		processExit(2, "create private process log: "+err.Error())
	}
	if _, err := privateLog.Write([]byte("fixture process completed\n")); err != nil {
		_ = privateLog.Close()
		processExit(2, "write private process log: "+err.Error())
	}
	if err := privateLog.Close(); err != nil {
		processExit(2, "close private process log: "+err.Error())
	}

	response := TaskResponse{Status: "complete", Summary: contextPayload.Manager.Manager.ID + " completed " + contextPayload.Phase,
		Delegations: []Delegation{}, ReworkRequests: []ReworkRequest{}, Integrated: contextPayload.Phase == "integrate", Questions: []string{}, Risks: []string{},
		ResolvedQuestions: []string{}, ResolvedRisks: []string{}, EscalateTo: ""}
	files := []agentexec.CandidateFile{}
	if os.Getenv(e2eBehaviorEnv) == "manager-rework-invalid-noop" && contextPayload.Phase == "work" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") && contextPayload.RepairDiagnostic != "" {
		response.Status = "no-op"
	}
	switch contextPayload.Phase {
	case "work":
		if contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") {
			response.Delegations = make([]Delegation, 0, len(contextPayload.DirectChildren))
			for _, child := range contextPayload.DirectChildren {
				response.Delegations = append(response.Delegations, Delegation{ManagerID: child, Goal: "Implement the owned source artifact and report its evidence."})
			}
		} else {
			artifactPath, content := e2eArtifactForManager(contextPayload.Manager.Manager.ID)
			if artifactPath == "" {
				processExit(2, "unexpected work manager "+contextPayload.Manager.Manager.ID)
			}
			files = []agentexec.CandidateFile{{Path: artifactPath, Mode: "0644", Content: content}}
			if os.Getenv(e2eBehaviorEnv) == "review-defect-fix" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
				files = []agentexec.CandidateFile{{Path: artifactPath, Mode: "0644", Content: "DEFECT: orders implementation v2\n"}}
			}
			if os.Getenv(e2eBehaviorEnv) == "review-always-fail" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") {
				files = []agentexec.CandidateFile{{Path: artifactPath, Mode: "0644", Content: "DEFECT: orders implementation v2\n"}}
			}
			if (os.Getenv(e2eBehaviorEnv) == "repair-check-fail" && contextPayload.RepairRound == 0 || os.Getenv(e2eBehaviorEnv) == "repair-check-no-change") && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") {
				files = []agentexec.CandidateFile{{Path: artifactPath, Mode: "0644", Content: "orders implementation v1\n"}}
			}
			if os.Getenv(e2eBehaviorEnv) == "repair-foreign-first" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
				files = []agentexec.CandidateFile{{Path: "src/inventory/foreign.txt", Mode: "0644", Content: "unauthorized"}}
			}
			if os.Getenv(e2eBehaviorEnv) == "out-of-scope" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") {
				files = []agentexec.CandidateFile{{Path: "src/inventory/foreign.txt", Mode: "0644", Content: "unauthorized"}}
			}
		}
	case "integrate":
		if len(contextPayload.ChildReports) != 2 {
			processExit(2, fmt.Sprintf("root integration saw %d child reports, want 2", len(contextPayload.ChildReports)))
		}
		if os.Getenv(e2eBehaviorEnv) == "failed-integration" {
			response.Status = "failed"
		}
		if (os.Getenv(e2eBehaviorEnv) == "manager-directed-rework" || os.Getenv(e2eBehaviorEnv) == "manager-rework-invalid-noop") && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
			response.ReworkRequests = []ReworkRequest{{ManagerID: e2eManagerID("orders", "orders"), Goal: "Correct the orders artifact after integration review.", Reason: "The initial orders implementation needs a focused correction."}}
			files = []agentexec.CandidateFile{{Path: "src/project-integration.txt", Mode: "0644", Content: "parent integration edit survives child rework\n"}}
		}
		if os.Getenv(e2eBehaviorEnv) == "integration-review-fix" && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") {
			content := "corrected parent integration\n"
			if countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
				content = "BAD parent integration\n"
			}
			files = []agentexec.CandidateFile{{Path: "src/project-integration.txt", Mode: "0644", Content: content}}
		}
		if os.Getenv(e2eBehaviorEnv) == "repair-foreign-first" && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
			files = []agentexec.CandidateFile{{Path: "src/orders/implementation.txt", Mode: "0644", Content: "unauthorized"}}
		}
	default:
		processExit(2, "unknown phase "+contextPayload.Phase)
	}
	reportJSON, err := json.Marshal(response)
	if err != nil {
		processExit(2, "encode task report: "+err.Error())
	}
	inputTokens, outputTokens := int64(41), int64(13)
	nonce := invocation.Nonce
	if os.Getenv(e2eBehaviorEnv) == "stale-nonce" {
		nonce = "stale-" + nonce
	}
	result := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: files, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: reportJSON, Uncertainty: []string{}, Usage: &agentexec.Usage{Source: "provider-reported", InputTokens: &inputTokens, OutputTokens: &outputTokens}}
	encoded, err := json.Marshal(result)
	if err != nil {
		processExit(2, "encode response: "+err.Error())
	}
	_, _ = os.Stdout.Write(encoded)
	processExit(0, "")
}

func runE2EReview(invocation agentexec.Invocation) {
	var contextPayload struct {
		ManagerID       string `json:"managerId"`
		CandidateID     string `json:"candidateId"`
		CandidateDigest string `json:"candidateDigest"`
		AcceptedModel   struct {
			Statements []projectmodel.Statement `json:"statements"`
		} `json:"acceptedModel"`
		CandidateFiles []reviewFileRef `json:"candidateFiles"`
		ResponseSchema json.RawMessage `json:"responseSchema"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextPayload); err != nil || contextPayload.ManagerID == "" || contextPayload.CandidateDigest == "" || len(contextPayload.ResponseSchema) == 0 {
		processExit(2, "review request omitted its identity, candidate binding, or schema")
	}
	if err := appendE2ELog(os.Getenv(e2eLogEnv), map[string]any{"phase": "review", "managerId": contextPayload.ManagerID,
		"candidateId": contextPayload.CandidateID, "candidateDigest": contextPayload.CandidateDigest,
		"candidateFiles": contextPayload.CandidateFiles, "artifacts": invocation.Request.Artifacts,
		"scopeIds": invocation.Request.ScopeIDs, "inputDigest": invocation.InputDigest, "schema": contextPayload.ResponseSchema,
	}); err != nil {
		processExit(2, "record review invocation: "+err.Error())
	}
	privateLog, err := os.OpenFile(os.Getenv("MARKITECT_AGENT_PRIVATE_LOG"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		processExit(2, "create private process log: "+err.Error())
	}
	if _, err := privateLog.Write([]byte("review fixture completed\n")); err != nil {
		_ = privateLog.Close()
		processExit(2, "write private process log: "+err.Error())
	}
	_ = privateLog.Close()
	response := reviewResponse{Status: "pass", Summary: "scoped candidate satisfies its accepted model", Findings: []reviewFindingResponse{}}
	if os.Getenv(e2eBehaviorEnv) == "review-defect-fix" && contextPayload.ManagerID == e2eManagerID("orders", "orders") {
		for _, artifact := range invocation.Request.Artifacts {
			if strings.Contains(string(artifact.Content), "DEFECT") {
				grounding := ""
				if len(contextPayload.AcceptedModel.Statements) > 0 {
					grounding = "statement:" + contextPayload.AcceptedModel.Statements[0].ID
				}
				response.Status = "fail"
				response.Summary = "candidate does not satisfy the accepted statement"
				response.Findings = []reviewFindingResponse{{Path: artifact.Path, Expectation: "implementation must not contain the injected defect", Grounding: grounding}}
			}
		}
	}
	if os.Getenv(e2eBehaviorEnv) == "review-always-fail" && contextPayload.ManagerID == e2eManagerID("orders", "orders") {
		grounding := ""
		if len(contextPayload.AcceptedModel.Statements) > 0 {
			grounding = "statement:" + contextPayload.AcceptedModel.Statements[0].ID
		}
		response.Status = "fail"
		response.Summary = "fixture candidate remains defective"
		response.Findings = []reviewFindingResponse{{Path: "src/orders/implementation.txt", Expectation: "implementation must not contain the injected defect", Grounding: grounding}}
	}
	if os.Getenv(e2eBehaviorEnv) == "integration-review-fix" && contextPayload.ManagerID == e2eManagerID("", "project-owner") {
		for _, artifact := range invocation.Request.Artifacts {
			if artifact.Path == "src/project-integration.txt" && strings.HasPrefix(string(artifact.Content), "BAD") {
				grounding := ""
				if len(contextPayload.AcceptedModel.Statements) > 0 {
					grounding = "statement:" + contextPayload.AcceptedModel.Statements[0].ID
				}
				response.Status = "fail"
				response.Summary = "parent integration contains a marked defect"
				response.Findings = []reviewFindingResponse{{Path: artifact.Path, Expectation: "integration must contain the corrected parent summary", Grounding: grounding}}
			}
		}
	}
	if contextPayload.ManagerID == e2eManagerID("orders", "orders") && os.Getenv(e2eBehaviorEnv) == "review-ungrounded" {
		response.Status = "fail"
		response.Summary = "fixture finding with invented grounding"
		response.Findings = []reviewFindingResponse{{Path: "src/orders/implementation.txt", Expectation: "invented expectation", Grounding: "statement:not-accepted"}}
	}
	if contextPayload.ManagerID == e2eManagerID("orders", "orders") && os.Getenv(e2eBehaviorEnv) == "review-working-drift" {
		root := os.Getenv(e2eRootEnv)
		if root == "" {
			processExit(2, "review drift fixture omitted project root")
		}
		if err := os.WriteFile(filepath.Join(root, "src", "orders", "implementation.txt"), []byte("changed during reviewer invocation\n"), 0o644); err != nil {
			processExit(2, "mutate review drift fixture: "+err.Error())
		}
	}
	report, err := json.Marshal(response)
	if err != nil {
		processExit(2, "encode review report: "+err.Error())
	}
	inputTokens, outputTokens := int64(31), int64(7)
	outcome := agentexec.OutcomeProposed
	candidateFiles := []agentexec.CandidateFile{}
	if contextPayload.ManagerID == e2eManagerID("orders", "orders") && os.Getenv(e2eBehaviorEnv) == "review-proposes-write" {
		candidateFiles = []agentexec.CandidateFile{{Path: "src/orders/implementation.txt", Mode: "0644", Content: "reviewer must not write"}}
	}
	if os.Getenv(e2eBehaviorEnv) == "review-incomplete" {
		outcome = agentexec.OutcomeIncomplete
	}
	if os.Getenv(e2eBehaviorEnv) == "review-over-budget" {
		inputTokens = 2_000_000
	}
	result := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: outcome,
		CandidateFiles: candidateFiles, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: report, Uncertainty: []string{}, Usage: &agentexec.Usage{Source: "provider-reported", InputTokens: &inputTokens, OutputTokens: &outputTokens}}
	encoded, err := json.Marshal(result)
	if err != nil {
		processExit(2, "encode review response: "+err.Error())
	}
	_, _ = os.Stdout.Write(encoded)
	processExit(0, "")
}

func TestProjectRunCheckProcess(t *testing.T) {
	if os.Getenv(e2eCheckEnv) != "1" {
		return
	}
	orders, err := os.ReadFile("src/orders/implementation.txt")
	if err != nil || string(orders) != "orders implementation v2\n" {
		processExit(1, fmt.Sprintf("orders check failed: content=%q error=%v", orders, err))
	}
	inventory, err := os.ReadFile("src/inventory/implementation.txt")
	if err != nil || string(inventory) != "inventory implementation v2\n" {
		processExit(1, fmt.Sprintf("inventory check failed: content=%q error=%v", inventory, err))
	}
	processExit(0, "")
}

func TestProjectRunProcessEndToEndResumeVerifyApply(t *testing.T) {
	root := makeProjectRunFixture(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pathDir := filepath.Dir(executable)
	pathValue := pathDir + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", pathValue)
	t.Setenv(e2eExecutorEnv, "1")
	t.Setenv(e2eCheckEnv, "1")
	logPath := filepath.Join(t.TempDir(), "requests.jsonl")
	t.Setenv(e2eLogEnv, logPath)

	host := projectworkHost()
	if _, err := source.IdentifyGit(root); err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts, integrate the child work, and run the declared checks.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Status != StatusPlanned || len(plan.Managers) != 3 || len(plan.Checks) != 2 {
		t.Fatalf("unexpected bounded plan: status=%s managers=%d checks=%d", plan.Status, len(plan.Managers), len(plan.Checks))
	}

	ctx, cancel := context.WithCancel(context.Background())
	interrupting := &cancelAfterSuccessfulProcess{cancel: cancel}
	first, runErr := Run(ctx, host, interrupting, root, plan.ID)
	cancel()
	if runErr == nil || first.Status != StatusInterrupted {
		diagnostic, _ := os.ReadFile(logPath)
		t.Fatalf("Run should interrupt after two completed subprocesses; status=%s err=%v; process log=%s", first.Status, runErr, diagnostic)
	}
	if len(first.Invocations) != 2 {
		t.Fatalf("first run recorded %d invocations, want root work plus one completed child", len(first.Invocations))
	}
	resumed, err := Resume(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.Status != StatusIntegrated || !resumed.Candidate.Integrated {
		t.Fatalf("resume did not integrate the staged candidates: status=%s integrated=%v", resumed.Status, resumed.Candidate.Integrated)
	}
	if len(resumed.Invocations) != 4 {
		t.Fatalf("resume has %d invocations, want exactly four with no replay", len(resumed.Invocations))
	}
	if err := assertProcessTrace(t, logPath, resumed); err != nil {
		t.Fatal(err)
	}

	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verified.Status != "verified" || len(verified.Checks) != 2 {
		t.Fatalf("actual required checks did not pass: status=%s checks=%+v", verified.Status, verified.Checks)
	}
	for _, check := range verified.Checks {
		if check.Outcome != "passed" || check.ExitCode != 0 || check.ExecutableDigest == "" {
			t.Fatalf("check was not an independently executed pinned process: %+v", check)
		}
	}

	paths, err := ApplyPaths(host, root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	targetDigest, err := CaptureTarget(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	branch := gitE2E(t, root, "branch", "--show-current")
	head := identityHead(t, root)
	request := ApplyRequest{RunID: plan.ID, PlanID: plan.ID, CandidateID: resumed.Candidate.ID,
		ExpectedVerificationDigest: verified.Digest, TargetBranch: branch, ExpectedHead: head, ExpectedWorktree: targetDigest}
	ordersPath := filepath.Join(root, "src", "orders", "implementation.txt")
	originalOrders, err := os.ReadFile(ordersPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ordersPath, []byte("manual change after verification\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, staleErr := Apply(host, ProcessInvoker{}, root, request); staleErr == nil {
		t.Fatal("Apply accepted a working-tree change after verification")
	}
	if err := os.WriteFile(ordersPath, originalOrders, 0644); err != nil {
		t.Fatal(err)
	}
	request.ExpectedWorktree, err = CaptureTarget(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := Apply(host, ProcessInvoker{}, root, request)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.Status != StatusApplied || len(applied.Written) != 2 {
		t.Fatalf("Apply did not write both candidate artifacts: %+v", applied)
	}
	assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v2\n")
	assertFileContents(t, root, "src/inventory/implementation.txt", "inventory implementation v2\n")
}

func TestProjectRunExecutorRejectsForeignPathProposal(t *testing.T) {
	// The main flow proves the positive protocol; this table-level process check
	// protects the actual subprocess response parser and authority validator.
	base := candidateData{Files: map[string]File{}}
	project := &Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "orders", Owns: []string{"src/orders/"}}}}}
	limits := Limits{MaxCandidateFileBytes: 1024, MaxCandidateBytes: 2048}
	_, err := applyProposal(base, []agentexec.CandidateFile{{Path: "src/inventory/foreign.txt", Mode: "0644", Content: "x"}},
		projectwork.Config{InventoryRoots: []string{"src"}}, project.Report,
		ManagerTask{ManagerID: "orders", Owns: []string{"src/orders/"}}, "work", nil, limits)
	if err == nil || !strings.Contains(err.Error(), "proposed path") {
		t.Fatalf("foreign-path candidate was not rejected: %v", err)
	}
}

func TestProjectRunReviewerFindingsTriggerTargetedImplementerRepair(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-defect-fix")
	updateE2ERuntime(t, root, func(config *Runtime) {
		config.Review = &ReviewConfig{Agents: map[string]Agent{}, MaxRounds: 2, MaxManagerRounds: 2}
		for id, agent := range config.Agents {
			config.Review.Agents[id] = agent
		}
		config.Limits.MaxStarts = 32
	})
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and reconcile the result.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != StatusIntegrated || !run.Candidate.Integrated {
		t.Fatalf("review loop did not integrate: %+v", run)
	}
	var ordersReviews []ReviewRecord
	for _, review := range run.Reviews {
		if review.ManagerID == e2eManagerID("orders", "orders") && review.Phase == "work" {
			ordersReviews = append(ordersReviews, review)
		}
	}
	if len(ordersReviews) != 2 || ordersReviews[0].Outcome != "fail" || ordersReviews[1].Outcome != "pass" || ordersReviews[0].CandidateDigest == ordersReviews[1].CandidateDigest {
		t.Fatalf("expected a defect-bound failed review followed by a fresh passing candidate review: %+v", ordersReviews)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, review := range ordersReviews {
		candidate, err := store.readCandidate(dir, review.CandidateID)
		if err != nil || candidate.Digest != review.CandidateDigest {
			t.Fatalf("review %d is not bound to its retained candidate: candidate=%+v err=%v", i, candidate, err)
		}
		if got := string(candidate.Files["src/orders/implementation.txt"].Content); i == 0 && !strings.Contains(got, "DEFECT") || i == 1 && got != "orders implementation v2\n" {
			t.Fatalf("review %d candidate bytes were %q", i, got)
		}
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("orders", "orders"), "work"); calls != 2 {
		t.Fatalf("implementer ran %d times after review findings, want 2", calls)
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("inventory", "inventory"), "work"); calls != 1 {
		t.Fatalf("unaffected sibling ran %d times, want 1", calls)
	}
	latest, err := store.readLatestState(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	staleCandidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	staleCandidate.ID, err = newID()
	if err != nil {
		t.Fatal(err)
	}
	staleCandidate.Parents = []string{run.Candidate.ID}
	ordersFile := staleCandidate.Files["src/orders/implementation.txt"]
	ordersFile.Content = []byte("changed after review\n")
	staleCandidate.Files[ordersFile.Path] = ordersFile
	if err := store.writeCandidate(dir, staleCandidate); err != nil {
		t.Fatal(err)
	}
	staleCandidate, err = store.readCandidate(dir, staleCandidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	latest.Candidate = candidateRef(staleCandidate, true)
	if err := persistState(store, &latest); err != nil {
		t.Fatal(err)
	}
	if _, staleErr := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID); staleErr == nil || !strings.Contains(staleErr.Error(), "fresh passed work review") {
		t.Fatalf("Verify accepted bytes outside the passed review scope: %v", staleErr)
	}
	latest.Candidate = run.Candidate
	if err := persistState(store, &latest); err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("fresh scoped reviews did not permit independent verification: report=%+v err=%v", verified, err)
	}
	paths, err := ApplyPaths(host, root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	worktreeDigest, err := CaptureTarget(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	request := ApplyRequest{RunID: plan.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID, ExpectedVerificationDigest: verified.Digest,
		TargetBranch: gitE2E(t, root, "branch", "--show-current"), ExpectedHead: identityHead(t, root), ExpectedWorktree: worktreeDigest}
	if _, err := Apply(host, ProcessInvoker{}, root, request); err != nil {
		t.Fatalf("freshly reviewed candidate could not pass guarded Apply: %v", err)
	}
}

func TestProjectRunReviewerRoundLimitBlocksRepeatedDefect(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-always-fail")
	enableE2EReviews(t, root, 100000)
	updateE2ERuntime(t, root, func(config *Runtime) { config.Review.MaxRounds = 2 })
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement and review the owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err == nil || run.Status != StatusBlocked {
		t.Fatalf("persistent review defect did not block at the configured round limit: status=%s err=%v", run.Status, err)
	}
	orders := e2eManagerID("orders", "orders")
	if reviews := reviewCount(run.Reviews, orders, "work"); reviews != 2 {
		t.Fatalf("work review rounds=%d, want bounded limit 2", reviews)
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), orders, "work"); calls != 2 {
		t.Fatalf("bounded reviewer loop launched %d implementer calls, want 2", calls)
	}
}

func TestManagerDirectedReworkRerunsOnlyRequestedLeafAndReintegratesAncestors(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "manager-directed-rework")
	updateE2ERuntime(t, root, func(config *Runtime) {
		config.Review = &ReviewConfig{Agents: map[string]Agent{}, MaxRounds: 3, MaxManagerRounds: 2}
		for id, agent := range config.Agents {
			config.Review.Agents[id] = agent
		}
		config.Limits.MaxStarts = 32
	})
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and reconcile the result.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != StatusIntegrated || len(run.ManagerReworkRounds) != 1 || run.ManagerReworkRounds[0].Status != "completed" {
		t.Fatalf("manager-directed rework did not close: status=%s rounds=%+v", run.Status, run.ManagerReworkRounds)
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("orders", "orders"), "work"); calls != 2 {
		t.Fatalf("requested leaf work ran %d times, want initial plus targeted rework", calls)
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("inventory", "inventory"), "work"); calls != 1 {
		t.Fatalf("unaffected sibling work ran %d times, want 1", calls)
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("", "project-owner"), "integrate"); calls != 2 {
		t.Fatalf("affected root integrations ran %d times, want initial plus reintegration", calls)
	}
	if len(run.Reviews) < 5 {
		t.Fatalf("expected work and reintegration reviews, got %d", len(run.Reviews))
	}
	ordersTask := findTask(run.Tasks, e2eManagerID("orders", "orders"))
	if ordersTask == nil || ordersTask.State != "worked" {
		t.Fatalf("successfully reworked leaf was not durably marked worked: %+v", ordersTask)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	finalCandidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(finalCandidate.Files["src/project-integration.txt"].Content); got != "parent integration edit survives child rework\n" {
		t.Fatalf("targeted child rework discarded parent-owned integration edit: %q", got)
	}
	// Model a durable interruption just after the completed rework round and
	// before the caller persists final integration closure. Resume must trust
	// the worked task and its reviews without replaying its subprocesses.
	latest, err := store.readLatestState(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	latest.Status = StatusInterrupted
	if err := persistState(store, &latest); err != nil {
		t.Fatal(err)
	}
	resumed, resumeErr := Resume(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	resumedOrders := findTask(resumed.Tasks, ordersTask.ManagerID)
	if resumeErr != nil || resumed.Status != StatusIntegrated || resumedOrders == nil || resumedOrders.State != "worked" || len(resumed.Invocations) != len(run.Invocations) {
		t.Fatalf("known-complete targeted rework did not resume without losing state: status=%s task=%+v err=%v", resumed.Status, resumedOrders, resumeErr)
	}
}

func TestPureRoutingManagerSkipsReviewerButImplementationManagersDoNot(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	enableE2EReviews(t, root, 100000)
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("pure routing fixture failed: status=%s err=%v", run.Status, err)
	}
	rootID := e2eManagerID("", "project-owner")
	rootTask := findTask(run.Tasks, rootID)
	if rootTask == nil || rootTask.ReviewStatus != "not-required" {
		t.Fatalf("pure router did not retain explicit not-required state: %+v", rootTask)
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), rootID, "review"); calls != 0 {
		t.Fatalf("pure routing Manager invoked reviewer %d times, want zero", calls)
	}
	for _, managerID := range []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")} {
		if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), managerID, "review"); calls == 0 {
			t.Fatalf("implementation Manager %s skipped mandatory review", managerID)
		}
	}
}

func TestTargetedWorkRejectsNoOpWithCandidateWrites(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "manager-rework-invalid-noop")
	enableE2EReviews(t, root, 100000)
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err == nil || run.Status != StatusBlocked || !strings.Contains(err.Error(), "claimed no-op while proposing files") {
		t.Fatalf("targeted work accepted no-op alongside candidate writes: status=%s err=%v", run.Status, err)
	}
	orders := e2eManagerID("orders", "orders")
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), orders, "work"); calls != 2 {
		t.Fatalf("targeted work calls=%d, want one normal and one rejected rework response", calls)
	}
}

func TestIntegrationReviewFindingsRepairAndRereviewParentCandidate(t *testing.T) {
	root := makeProjectRunFixture(t)
	projectPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	projectYAML, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	projectYAML = []byte(strings.Replace(string(projectYAML), "  - .markitect/model/manager.yaml\n", "  - .markitect/model/manager.yaml\n  - .markitect/model/root-statement.yaml\n", 1))
	writeE2E(t, root, projectwork.ManifestPath, string(projectYAML))
	writeE2E(t, root, ".markitect/model/root-statement.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Statement\nmetadata:\n  name: integration-quality\n  namespace: \"\"\npurpose: Parent integration must produce a clean summary.\nspec:\n  category: concept\n  description: Parent-owned integration files contain the corrected summary.\n")
	gitE2E(t, root, "add", projectwork.ManifestPath, ".markitect/model/root-statement.yaml")
	gitE2E(t, root, "commit", "--amend", "--no-edit")
	setupE2EProcess(t, "integration-review-fix")
	enableE2EReviews(t, root, 100000)
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement and reconcile the two owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("integration reviewer findings did not lead to corrected closure: status=%s err=%v", run.Status, err)
	}
	var rootReviews []ReviewRecord
	for _, review := range run.Reviews {
		if review.ManagerID == e2eManagerID("", "project-owner") && review.Phase == "integrate" {
			rootReviews = append(rootReviews, review)
		}
	}
	if len(rootReviews) != 2 || rootReviews[0].Outcome != "fail" || rootReviews[1].Outcome != "pass" {
		t.Fatalf("expected failed and fresh passed root integration reviews, got %+v", rootReviews)
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
	if got := string(candidate.Files["src/project-integration.txt"].Content); got != "corrected parent integration\n" {
		t.Fatalf("final candidate retained reviewer finding: %q", got)
	}
	if calls := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("", "project-owner"), "integrate"); calls != 2 {
		t.Fatalf("parent integration ran %d times, want initial plus bounded correction", calls)
	}
}

func TestReviewerContractFailuresAndUncertainInvocationNeverReplay(t *testing.T) {
	for _, behavior := range []string{"review-proposes-write", "review-ungrounded", "review-incomplete"} {
		t.Run(behavior, func(t *testing.T) {
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, behavior)
			enableE2EReviews(t, root, 100000)
			host := projectworkHost()
			plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both fixture artifacts.",
				Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatal(err)
			}
			run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
			if err == nil || run.Status != StatusFailed {
				t.Fatalf("invalid or uncertain review was accepted: status=%s err=%v", run.Status, err)
			}
			if len(run.Invocations) == 0 || run.Invocations[len(run.Invocations)-1].Role != "reviewer" {
				t.Fatalf("failed reviewer invocation was not retained: %+v", run.Invocations)
			}
			before := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("orders", "orders"), "review") + countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("", "project-owner"), "review")
			_, resumeErr := Resume(context.Background(), host, ProcessInvoker{}, root, plan.ID)
			if resumeErr == nil {
				t.Fatal("uncertain review invocation was automatically replayed")
			}
			after := countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("orders", "orders"), "review") + countE2EProcessCalls(os.Getenv(e2eLogEnv), e2eManagerID("", "project-owner"), "review")
			if after != before {
				t.Fatalf("Resume replayed reviewer process: before=%d after=%d", before, after)
			}
		})
	}
}

func TestReviewerCostOverrunBlocksBeforeAcceptingReview(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-over-budget")
	enableE2EReviews(t, root, 1)
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both fixture artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err == nil || run.Status != StatusBlocked || len(run.Invocations) == 0 || run.Invocations[len(run.Invocations)-1].CostMicros <= 1 {
		t.Fatalf("review cost overrun was not durably blocked: status=%s last=%+v err=%v", run.Status, lastInvocation(run.Invocations), err)
	}
}

func TestReviewerWorkingTreeDriftRetainsUncertainCallWithoutReplay(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-working-drift")
	t.Setenv(e2eRootEnv, root)
	enableE2EReviews(t, root, 100000)
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both fixture artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, runErr := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if runErr == nil || run.Status != StatusFailed {
		t.Fatalf("reviewer changed the working snapshot but its review was consumed: status=%s err=%v", run.Status, runErr)
	}
	orders := e2eManagerID("orders", "orders")
	task := findTask(run.Tasks, orders)
	if task == nil || task.ReviewStatus != "uncertain" {
		t.Fatalf("post-review binding failure was not retained as uncertain: %+v", task)
	}
	if len(run.Invocations) == 0 || run.Invocations[len(run.Invocations)-1].Role != "reviewer" {
		t.Fatalf("reviewer call receipt was not retained after binding drift: %+v", run.Invocations)
	}
	before := countE2EProcessCalls(os.Getenv(e2eLogEnv), orders, "review")
	if _, err := Resume(context.Background(), host, ProcessInvoker{}, root, plan.ID); err == nil {
		t.Fatal("uncertain reviewer invocation was resumed automatically")
	}
	if after := countE2EProcessCalls(os.Getenv(e2eLogEnv), orders, "review"); after != before {
		t.Fatalf("Resume replayed reviewer invocation after binding drift: before=%d after=%d", before, after)
	}
}

func enableE2EReviews(t *testing.T, root string, maxCost int64) {
	t.Helper()
	updateE2ERuntime(t, root, func(config *Runtime) {
		config.Review = &ReviewConfig{Agents: map[string]Agent{}, MaxRounds: 3, MaxManagerRounds: 2}
		for id, agent := range config.Agents {
			config.Review.Agents[id] = agent
		}
		config.Limits.MaxStarts = 32
		config.Limits.MaxCostMicros = maxCost
	})
}

func lastInvocation(invocations []InvocationLog) InvocationLog {
	if len(invocations) == 0 {
		return InvocationLog{}
	}
	return invocations[len(invocations)-1]
}

func TestProjectRunProcessRejectsStaleNonceOutOfScopeAndFailedIntegration(t *testing.T) {
	for _, behavior := range []string{"stale-nonce", "out-of-scope", "failed-integration"} {
		t.Run(behavior, func(t *testing.T) {
			root := makeProjectRunFixture(t)
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv(e2eExecutorEnv, "1")
			t.Setenv(e2eCheckEnv, "1")
			t.Setenv(e2eBehaviorEnv, behavior)
			t.Setenv(e2eLogEnv, filepath.Join(t.TempDir(), "requests.jsonl"))
			plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
				Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			run, runErr := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
			if runErr == nil || run.Status == StatusIntegrated || run.Status == StatusVerified || run.Status == StatusApplied {
				t.Fatalf("behavior %s unexpectedly closed successfully: status=%s err=%v", behavior, run.Status, runErr)
			}
		})
	}
}

func TestProjectRunRepairsKnownValidationFailuresInWorkAndIntegration(t *testing.T) {
	root := makeProjectRunFixture(t)
	updateE2ERuntime(t, root, func(config *Runtime) {
		config.Limits.MaxRetries = 1
		config.Limits.MaxDuration += Duration(time.Second)
	})
	setupE2EProcess(t, "repair-foreign-first")
	logPath := os.Getenv(e2eLogEnv)
	plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Run did not repair known validation failures: status=%s err=%v", run.Status, err)
	}
	if run.Status != StatusIntegrated || len(run.Invocations) != 6 {
		t.Fatalf("repair did not finish a fully integrated run: status=%s invocations=%d", run.Status, len(run.Invocations))
	}
	orders := findTask(run.Tasks, e2eManagerID("orders", "orders"))
	rootTask := findTask(run.Tasks, e2eManagerID("", "project-owner"))
	if orders == nil || orders.WorkAttempts != 2 || orders.Attempts != 2 || rootTask == nil || rootTask.IntegrationAttempts != 2 || rootTask.Attempts != 3 {
		t.Fatalf("attempt counts were not retained: orders=%+v root=%+v", orders, rootTask)
	}
	if err := assertRepairTrace(t, logPath, orders.ManagerID, "work"); err != nil {
		t.Fatal(err)
	}
	if err := assertRepairTrace(t, logPath, rootTask.ManagerID, "integrate"); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := store.readLatestState(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != StatusIntegrated || findTask(persisted.Tasks, orders.ManagerID).WorkAttempts != 2 || findTask(persisted.Tasks, rootTask.ManagerID).IntegrationAttempts != 2 {
		t.Fatalf("retry ledger was not durable in latest state: %+v", persisted.Tasks)
	}
}

func TestProjectRunRetryCeilingAndCostAccountingAreDurable(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		retries     int
		maxCost     int64
		priceOrders int64
		wantStatus  string
	}{
		{name: "retry ceiling", retries: 0, maxCost: 100000, priceOrders: 1, wantStatus: StatusFailed},
		{name: "cost ceiling before retry", retries: 1, maxCost: 54, priceOrders: 1_000_000, wantStatus: StatusBlocked},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := makeProjectRunFixture(t)
			updateE2ERuntime(t, root, func(config *Runtime) {
				config.Limits.MaxRetries = scenario.retries
				config.Limits.MaxCostMicros = scenario.maxCost
				config.Limits.MaxDuration += Duration(time.Second)
				orders := e2eManagerID("orders", "orders")
				agent := config.Agents[orders]
				agent.Pricing = Pricing{InputMicrosPerMillion: scenario.priceOrders, OutputMicrosPerMillion: scenario.priceOrders}
				config.Agents[orders] = agent
			})
			setupE2EProcess(t, "repair-foreign-first")
			logPath := os.Getenv(e2eLogEnv)
			plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Implement the owned orders artifact.",
				Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			run, runErr := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
			if runErr == nil || run.Status != scenario.wantStatus {
				t.Fatalf("invalid proposal did not stop at the expected bounded state: status=%s err=%v", run.Status, runErr)
			}
			orders := findTask(run.Tasks, e2eManagerID("orders", "orders"))
			if orders == nil || orders.WorkAttempts != 1 || orders.Attempts != 1 || len(run.Invocations) != 2 {
				t.Fatalf("first failed attempt accounting missing: orders=%+v invocations=%d", orders, len(run.Invocations))
			}
			if got := countE2EProcessCalls(logPath, orders.ManagerID, "work"); got != 1 {
				t.Fatalf("runtime started %d orders attempts, want exactly one", got)
			}
			store, err := newRunStore(root)
			if err != nil {
				t.Fatal(err)
			}
			persisted, err := store.readLatestState(run.ID)
			if err != nil {
				t.Fatal(err)
			}
			persistedOrders := findTask(persisted.Tasks, orders.ManagerID)
			if persisted.Status != scenario.wantStatus || persistedOrders.WorkAttempts != 1 || persistedOrders.Attempts != 1 || strings.TrimSpace(persistedOrders.RepairDiagnostic) == "" {
				t.Fatalf("failed-attempt and retry diagnostic were not durably recorded: status=%s task=%+v", persisted.Status, persistedOrders)
			}
			if scenario.name == "cost ceiling before retry" && totalCost(persisted.Invocations) != scenario.maxCost {
				t.Fatalf("reported spend %d, want bound %d", totalCost(persisted.Invocations), scenario.maxCost)
			}
		})
	}
}

func projectworkHost() Host {
	return Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot, PlanEdit: projectwork.PlanEdit, ApplyEdit: projectwork.ApplyEdit}
}

func e2eManagerID(namespace, name string) string {
	data, _ := json.Marshal([]string{projectmodel.APIVersion, "Manager", namespace, name})
	return string(data)
}

func e2eArtifactForManager(id string) (string, string) {
	switch id {
	case e2eManagerID("orders", "orders"):
		return "src/orders/implementation.txt", "orders implementation v2\n"
	case e2eManagerID("inventory", "inventory"):
		return "src/inventory/implementation.txt", "inventory implementation v2\n"
	default:
		return "", ""
	}
}

func makeProjectRunFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitE2E(t, root, "init", "-b", "codex/projectrun-e2e")
	gitE2E(t, root, "config", "user.email", "projectrun@example.test")
	gitE2E(t, root, "config", "user.name", "Project Run E2E")
	writeE2E(t, root, "README.md", "project-run process fixture\n")
	writeE2E(t, root, ".markitect/project.yaml", "apiVersion: "+projectwork.APIVersion+"\nname: Process fixture\nmodelFiles:\n  - .markitect/model/manager.yaml\n  - .markitect/model/orders/manager.yaml\n  - .markitect/model/orders/statement.yaml\n  - .markitect/model/orders/artifact.yaml\n  - .markitect/model/orders/check.yaml\n  - .markitect/model/inventory/manager.yaml\n  - .markitect/model/inventory/statement.yaml\n  - .markitect/model/inventory/artifact.yaml\n  - .markitect/model/inventory/check.yaml\ninventoryRoots:\n  - src\nexclusions: []\n")
	rootManager := "apiVersion: " + projectmodel.APIVersion + "\nkind: Manager\nmetadata:\n  name: project-owner\n  namespace: \"\"\npurpose: Owns integration and project-wide delivery.\nspec:\n  owns: [.]\n"
	writeE2E(t, root, ".markitect/model/manager.yaml", rootManager)
	writeE2E(t, root, ".markitect/model/orders/manager.yaml", e2eChildManager("orders"))
	writeE2E(t, root, ".markitect/model/orders/statement.yaml", e2eStatement("orders", "orders-work", "Implement the orders artifact."))
	writeE2E(t, root, ".markitect/model/orders/artifact.yaml", e2eArtifact("orders", "orders-code", "orders-work", "orders-check", "src/orders/implementation.txt"))
	writeE2E(t, root, ".markitect/model/orders/check.yaml", e2eCheck("orders", "orders-check", "orders-work"))
	writeE2E(t, root, ".markitect/model/inventory/manager.yaml", e2eChildManager("inventory"))
	writeE2E(t, root, ".markitect/model/inventory/statement.yaml", e2eStatement("inventory", "inventory-work", "Implement the inventory artifact."))
	writeE2E(t, root, ".markitect/model/inventory/artifact.yaml", e2eArtifact("inventory", "inventory-code", "inventory-work", "inventory-check", "src/inventory/implementation.txt"))
	writeE2E(t, root, ".markitect/model/inventory/check.yaml", e2eCheck("inventory", "inventory-check", "inventory-work"))
	writeE2E(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
	writeE2E(t, root, "src/inventory/implementation.txt", "inventory implementation v1\n")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"-test.run=^TestProjectRunExecutorProcess$"}
	checkCommand := filepath.Base(executable)
	checkArgs := []string{"-test.run=^TestProjectRunCheckProcess$"}
	buildAgent := func() Agent {
		return Agent{Command: executable, Args: args, Model: "fixture-model", ProviderVersion: "e2e-process-v1", Timeout: Duration(30 * time.Second),
			MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20, Environment: []string{"PATH", e2eExecutorEnv, e2eCheckEnv, e2eLogEnv, e2eBehaviorEnv, e2eRootEnv},
			Pricing: Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}}
	}
	config := Runtime{APIVersion: APIVersion, Mode: ModeControlledLocal, Agents: map[string]Agent{
		e2eManagerID("", "project-owner"):      buildAgent(),
		e2eManagerID("orders", "orders"):       buildAgent(),
		e2eManagerID("inventory", "inventory"): buildAgent(),
	}, Limits: Limits{MaxDepth: 4, MaxStarts: 16, MaxRetries: 0, MaxParallel: 1, MaxDuration: Duration(4 * time.Minute), MaxCostMicros: 100000,
		MaxCandidateFileBytes: 4096, MaxCandidateBytes: 16384}}
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, RuntimePath, string(configBytes))
	// The check argv is modeled as a literal bare executable plus fixed args.
	for _, file := range []string{".markitect/model/orders/check.yaml", ".markitect/model/inventory/check.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Replace(string(data), "command: [go, test, ./orders]", "command: ["+checkCommand+", "+strings.Join(checkArgs, ", ")+" ]", 1)
		text = strings.Replace(text, "command: [go, test, ./inventory]", "command: ["+checkCommand+", "+strings.Join(checkArgs, ", ")+" ]", 1)
		writeE2E(t, root, file, text)
	}
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "-m", "project run process fixture")
	return root
}

func updateE2ERuntime(t *testing.T, root string, update func(*Runtime)) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(RuntimePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var runtime Runtime
	if err := yaml.Unmarshal(data, &runtime); err != nil {
		t.Fatal(err)
	}
	update(&runtime)
	updated, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, RuntimePath, string(updated))
	gitE2E(t, root, "add", RuntimePath)
	gitE2E(t, root, "commit", "-m", "adjust projectrun test runtime")
}

func setupE2EProcess(t *testing.T, behavior string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(e2eExecutorEnv, "1")
	t.Setenv(e2eCheckEnv, "1")
	t.Setenv(e2eBehaviorEnv, behavior)
	t.Setenv(e2eLogEnv, filepath.Join(t.TempDir(), "requests.jsonl"))
}

func assertRepairTrace(t *testing.T, logPath, managerID, phase string) error {
	t.Helper()
	records, err := readE2ERecords(logPath)
	if err != nil {
		return err
	}
	var found []map[string]any
	for _, record := range records {
		if record["managerId"] == managerID && record["phase"] == phase {
			found = append(found, record)
		}
	}
	if len(found) != 2 {
		return fmt.Errorf("%s %s had %d process calls, want a failed validation and one repair", managerID, phase, len(found))
	}
	if strings.TrimSpace(fmt.Sprint(found[0]["repairDiagnostic"])) != "" || strings.TrimSpace(fmt.Sprint(found[1]["repairDiagnostic"])) == "" {
		return fmt.Errorf("repair diagnostic was not supplied only on retry: %+v", found)
	}
	if found[0]["candidateDigest"] != found[1]["candidateDigest"] {
		return fmt.Errorf("repair changed the scoped candidate input: %v != %v", found[0]["candidateDigest"], found[1]["candidateDigest"])
	}
	if !reflect.DeepEqual(found[0]["artifacts"], found[1]["artifacts"]) {
		return fmt.Errorf("repair changed the scoped source artifacts")
	}
	return nil
}

func countE2EProcessCalls(path, managerID, phase string) int {
	records, err := readE2ERecords(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, record := range records {
		if record["managerId"] == managerID && record["phase"] == phase {
			count++
		}
	}
	return count
}

func readE2ERecords(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func e2eChildManager(name string) string {
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Manager\nmetadata:\n  name: " + name + "\n  namespace: " + name + "\npurpose: Owns the " + name + " implementation.\nspec:\n  parent:\n    apiVersion: " + projectmodel.APIVersion + "\n    kind: Manager\n    namespace: \"\"\n    name: project-owner\n  owns: [src/" + name + "/]\n"
}

func e2eStatement(namespace, name, description string) string {
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Statement\nmetadata:\n  name: " + name + "\n  namespace: " + namespace + "\npurpose: " + description + "\nspec:\n  category: concept\n  description: " + description + "\n"
}

func e2eArtifact(namespace, name, statement, check, path string) string {
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Artifact\nmetadata:\n  name: " + name + "\n  namespace: " + namespace + "\npurpose: Implements the owned source behavior.\nspec:\n  role: implementation\n  realizes:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Statement\n      namespace: " + namespace + "\n      name: " + statement + "\n  paths: [" + path + "]\n  checks:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Check\n      namespace: " + namespace + "\n      name: " + check + "\n  required: true\n"
}

func e2eCheck(namespace, name, statement string) string {
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Check\nmetadata:\n  name: " + name + "\n  namespace: " + namespace + "\npurpose: Check the materialized project candidate.\nspec:\n  command: [go, test, ./orders]\n  uses:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Statement\n      namespace: " + namespace + "\n      name: " + statement + "\n  limitation: This fixture check validates only the candidate bytes.\n"
}

func assertProcessTrace(t *testing.T, logPath string, run RunReport) error {
	t.Helper()
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()
	var records []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return err
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	counts := map[string]int{}
	rootIntegration := map[string]any(nil)
	for _, record := range records {
		manager, _ := record["managerId"].(string)
		phase, _ := record["phase"].(string)
		counts[manager+"|"+phase]++
		for _, key := range []string{"sourceRevision", "modelDigest", "modulePin", "projectionId"} {
			if strings.TrimSpace(fmt.Sprint(record[key])) == "" {
				return fmt.Errorf("subprocess request omitted %s: %+v", key, record)
			}
		}
		if record["schema"] == nil || record["scopeIds"] == nil || record["artifacts"] == nil {
			return fmt.Errorf("subprocess request omitted schema, scope, or artifacts: %+v", record)
		}
		revision := fmt.Sprint(record["sourceRevision"])
		if len(revision) != 40 {
			return fmt.Errorf("request source revision is not a fixed full commit: %q", revision)
		}
		if _, err := hex.DecodeString(revision); err != nil {
			return fmt.Errorf("request source revision is not hexadecimal: %q", revision)
		}
		for _, key := range []string{"modelDigest", "modulePin", "projectionId"} {
			if value := fmt.Sprint(record[key]); !strings.HasPrefix(value, "sha256:") {
				return fmt.Errorf("request %s is not content-bound: %q", key, value)
			}
		}
		var artifacts []agentexec.Artifact
		encoded, _ := json.Marshal(record["artifacts"])
		if err := json.Unmarshal(encoded, &artifacts); err != nil {
			return fmt.Errorf("decode logged scoped artifacts: %w", err)
		}
		for _, artifact := range artifacts {
			sum := sha256.Sum256(artifact.Content)
			if artifact.Digest != "sha256:"+hex.EncodeToString(sum[:]) {
				return fmt.Errorf("artifact %s digest does not match its exact bytes: %s", artifact.Path, artifact.Digest)
			}
			if artifact.Mode != "0644" && artifact.Mode != "0755" && artifact.Mode != "0600" {
				return fmt.Errorf("artifact %s has unsupported protocol mode %s", artifact.Path, artifact.Mode)
			}
		}
		if record["contextDigest"] == nil || record["inputDigest"] == nil || strings.TrimSpace(fmt.Sprint(record["managerPurpose"])) == "" {
			return fmt.Errorf("request omitted context binding or manager responsibility: %+v", record)
		}
		if phase == "work" && strings.Contains(manager, `"orders"`) && !containsString(anyStringSlice(record["managerOwns"]), "src/orders/") {
			return fmt.Errorf("orders context omitted its owned repository path: %+v", record["managerOwns"])
		}
		if phase == "work" && strings.Contains(manager, `"inventory"`) && !containsString(anyStringSlice(record["managerOwns"]), "src/inventory/") {
			return fmt.Errorf("inventory context omitted its owned repository path: %+v", record["managerOwns"])
		}
		if manager == e2eManagerID("", "project-owner") && phase == "integrate" {
			rootIntegration = record
		}
	}
	if len(records) != len(run.Invocations) {
		return fmt.Errorf("process log has %d requests for %d durable invocation receipts", len(records), len(run.Invocations))
	}
	if counts[e2eManagerID("", "project-owner")+"|work"] != 1 || counts[e2eManagerID("orders", "orders")+"|work"] != 1 || counts[e2eManagerID("inventory", "inventory")+"|work"] != 1 || counts[e2eManagerID("", "project-owner")+"|integrate"] != 1 {
		return fmt.Errorf("resume replayed completed work or omitted a manager phase: %v", counts)
	}
	if rootIntegration == nil {
		return fmt.Errorf("root integration subprocess was not invoked")
	}
	reports, ok := rootIntegration["childReports"].([]any)
	if !ok || len(reports) != 2 {
		return fmt.Errorf("root integration did not receive exactly two child summaries: %#v", rootIntegration["childReports"])
	}
	return nil
}

func anyStringSlice(value any) []string {
	items, _ := value.([]any)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

type cancelAfterSuccessfulProcess struct {
	cancel context.CancelFunc
	calls  atomic.Int32
}

func (i *cancelAfterSuccessfulProcess) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	result, err := (ProcessInvoker{}).Run(ctx, config, request, options)
	if err == nil && i.calls.Add(1) == 2 {
		i.cancel()
	}
	return result, err
}

func (i *cancelAfterSuccessfulProcess) Fingerprint(config agentexec.Config) (string, error) {
	return (ProcessInvoker{}).Fingerprint(config)
}

func appendE2ELog(path string, value any) error {
	if path == "" {
		return fmt.Errorf("log path is missing")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(data, '\n'))
	return err
}

func processExit(code int, message string) {
	if message != "" {
		_, _ = fmt.Fprintln(os.Stderr, message)
		_ = appendE2ELog(os.Getenv(e2eLogEnv), map[string]string{"processError": message})
	}
	os.Exit(code)
}

func gitE2E(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func identityHead(t *testing.T, root string) string {
	t.Helper()
	return gitE2E(t, root, "rev-parse", "HEAD")
}

func writeE2E(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContents(t *testing.T, root, relative, expected string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil || string(data) != expected {
		t.Fatalf("%s = %q, %v; want %q", relative, data, err, expected)
	}
}
