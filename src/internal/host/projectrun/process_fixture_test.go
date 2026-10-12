package projectrun

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
	"go.yaml.in/yaml/v3"
)

const (
	e2eExecutorEnv = "MARKITECT_E2E_EXECUTOR"
	e2eCheckEnv    = "MARKITECT_E2E_CHECK"
	e2eLogEnv      = "MARKITECT_E2E_LOG"
	e2eBehaviorEnv = "MARKITECT_E2E_BEHAVIOR"
	e2eRootEnv     = "MARKITECT_E2E_ROOT"
	// e2eOmitUsageEnv lists the fixture phases (work, review, verify) whose
	// responses omit provider usage, like an executor that cannot report it.
	e2eOmitUsageEnv        = "MARKITECT_E2E_OMIT_USAGE"
	e2eIntegrationCheckEnv = "MARKITECT_E2E_INTEGRATION_CHECK"
)

// These process entry points are re-executed by projectexecutor.ProcessInvoker and by the
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
	if invocation.Request.Role == agentexec.RoleVerifier {
		runE2EVerifier(invocation)
		return
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
	if discriminator.Kind != "projectrun-task/v1" {
		processExit(2, "expected explicit Manager task context")
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
		"agentModel": e2eAgentModel(),
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
			if len(contextPayload.DirectChildren) > 0 {
				response.Delegations = make([]Delegation, 0, len(contextPayload.DirectChildren))
				for _, child := range contextPayload.DirectChildren {
					response.Delegations = append(response.Delegations, Delegation{ManagerID: child, Goal: "Implement the nested owned artifact and report its evidence."})
				}
			}
			artifactPath, content := e2eArtifactForManager(contextPayload.Manager.Manager.ID)
			if artifactPath == "" {
				processExit(2, "unexpected work manager "+contextPayload.Manager.Manager.ID)
			}
			files = []agentexec.CandidateFile{{Path: artifactPath, Mode: "0644", Content: content}}
			if os.Getenv(e2eBehaviorEnv) == "integration-review-child-rework" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
				files[0].Content = "orders implementation DEFECT v1\n"
			}
			if os.Getenv(e2eBehaviorEnv) == "nested-child-rework" && contextPayload.Manager.Manager.ID == e2eManagerID("orders.commerce", "commerce") {
				switch countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) {
				case 1:
					files[0].Content = "commerce implementation DEFECT v1\n"
				case 2:
					files[0].Content = "commerce implementation DEFECT v2\n"
				}
			}
			if os.Getenv(e2eBehaviorEnv) == "provenance-escalation" && contextPayload.Manager.Manager.ID == e2eManagerID("orders.commerce", "commerce") {
				response.Status = "partial"
				response.Questions = []string{"Commerce deployment question"}
				response.Risks = []string{"Commerce deployment risk"}
				response.EscalateTo = e2eManagerID("orders", "orders")
			}
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
			if os.Getenv(e2eBehaviorEnv) == "forgotten-file" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") && contextPayload.RepairRound == 0 {
				// The executor reports completion but forgets its owned artifact.
				files = []agentexec.CandidateFile{}
			}
		}
	case "integrate":
		if os.Getenv(e2eBehaviorEnv) == "integration-stale-summary" && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") {
			// Both children are correct, but the first integration summarizes the
			// base revision instead of the integrated child outputs.
			summary := "orders implementation v1\ninventory implementation v1\n"
			if contextPayload.RepairRound > 0 {
				summary = "orders implementation v2\ninventory implementation v2\n"
			}
			files = []agentexec.CandidateFile{{Path: "src/project-integration.txt", Mode: "0644", Content: summary}}
		}
		if len(contextPayload.ChildReports) != len(contextPayload.DirectChildren) {
			processExit(2, fmt.Sprintf("Manager integration saw %d child reports, want %d", len(contextPayload.ChildReports), len(contextPayload.DirectChildren)))
		}
		if os.Getenv(e2eBehaviorEnv) == "failed-integration" {
			response.Status = "failed"
		}
		if (os.Getenv(e2eBehaviorEnv) == "manager-directed-rework" || os.Getenv(e2eBehaviorEnv) == "manager-rework-invalid-noop") && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
			response.ReworkRequests = []ReworkRequest{{ManagerID: e2eManagerID("orders", "orders"), Goal: "Correct the orders artifact after integration review.", Reason: "The initial orders implementation needs a focused correction."}}
			files = []agentexec.CandidateFile{{Path: "src/project-integration.txt", Mode: "0644", Content: "parent integration edit survives child rework\n"}}
		}
		if os.Getenv(e2eBehaviorEnv) == "integration-review-child-rework" && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") {
			switch countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) {
			case 1:
				files = []agentexec.CandidateFile{{Path: "src/project-integration.txt", Mode: "0644", Content: "parent integration edit survives child rework\n"}}
			case 2:
				response.ReworkRequests = []ReworkRequest{{ManagerID: e2eManagerID("orders", "orders"), Goal: "Correct the orders artifact after integration review.", Reason: "The independent integration review found a defect in the orders artifact."}}
			}
		}
		if os.Getenv(e2eBehaviorEnv) == "nested-child-rework" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) <= 2 {
			response.ReworkRequests = []ReworkRequest{{ManagerID: e2eManagerID("orders.commerce", "commerce"), Goal: "Correct the nested Commerce artifact.", Reason: "The parent found a defect in the nested child output."}}
		}
		if os.Getenv(e2eBehaviorEnv) == "nested-child-rework" && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") && countE2EProcessCalls(os.Getenv(e2eLogEnv), contextPayload.Manager.Manager.ID, contextPayload.Phase) == 1 {
			files = []agentexec.CandidateFile{{Path: "src/project-integration.txt", Mode: "0644", Content: "root integration after nested rework\n"}}
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
		if os.Getenv(e2eBehaviorEnv) == "provenance-escalation" && contextPayload.Manager.Manager.ID == e2eManagerID("orders", "orders") {
			response.Status = "partial"
			response.Questions = []string{"Commerce deployment question"}
			response.Risks = []string{"Commerce deployment risk"}
			response.EscalateTo = e2eManagerID("", "project-owner")
		}
		if os.Getenv(e2eBehaviorEnv) == "provenance-escalation" && contextPayload.Manager.Manager.ID == e2eManagerID("", "project-owner") {
			response.ResolvedQuestions = []string{"Commerce deployment question"}
			response.ResolvedRisks = []string{"Commerce deployment risk"}
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
	outcome := agentexec.OutcomeProposed
	if response.EscalateTo != "" {
		outcome = agentexec.OutcomeEscalated
	}
	result := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: outcome,
		CandidateFiles: files, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: reportJSON, Uncertainty: []string{}, Usage: e2eUsage("work", inputTokens, outputTokens)}
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
		"agentModel": e2eAgentModel(),
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
	if os.Getenv(e2eBehaviorEnv) == "integration-review-child-rework" && contextPayload.ManagerID == e2eManagerID("", "project-owner") {
		for _, artifact := range invocation.Request.Artifacts {
			if artifact.Path == "src/orders/implementation.txt" && strings.Contains(string(artifact.Content), "DEFECT") {
				grounding := ""
				if len(contextPayload.AcceptedModel.Statements) > 0 {
					grounding = "statement:" + contextPayload.AcceptedModel.Statements[0].ID
				}
				response.Status = "fail"
				response.Summary = "integration candidate contains a defect in the orders child artifact"
				response.Findings = []reviewFindingResponse{{Path: artifact.Path, Expectation: "orders artifact must contain the corrected implementation", Grounding: grounding}}
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
		ReportJSON: report, Uncertainty: []string{}, Usage: e2eUsage("review", inputTokens, outputTokens)}
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
	if wd, err := os.Getwd(); err == nil {
		fmt.Println("check cwd: " + wd)
	}
	processExit(0, "")
}

// TestProjectRunIntegrationCheckProcess is a root-owned declared check: the
// integration summary must match the integrated child artifacts.
func TestProjectRunIntegrationCheckProcess(t *testing.T) {
	if os.Getenv(e2eIntegrationCheckEnv) != "1" {
		return
	}
	var expected strings.Builder
	for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
		content, err := os.ReadFile(path)
		if err != nil {
			processExit(1, "integration check could not read "+path)
		}
		expected.Write(content)
	}
	summary, err := os.ReadFile("src/project-integration.txt")
	if err != nil || string(summary) != expected.String() {
		processExit(1, fmt.Sprintf("integration summary %q does not match the integrated child artifacts %q", summary, expected.String()))
	}
	processExit(0, "")
}

// runE2EVerifier passes every required subject with the exact required
// evidence references, like a verifier that accepts the declared checks.
func runE2EVerifier(invocation agentexec.Invocation) {
	var contextPayload struct {
		RequiredSubjects     []string `json:"requiredSubjects"`
		RequiredEvidenceRefs []string `json:"requiredEvidenceRefs"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextPayload); err != nil || len(contextPayload.RequiredSubjects) == 0 {
		processExit(2, "verifier request omitted its required subjects")
	}
	if err := appendE2ELog(os.Getenv(e2eLogEnv), map[string]any{"phase": "verify", "managerId": "verifier", "agentModel": e2eAgentModel(), "inputDigest": invocation.InputDigest}); err != nil {
		processExit(2, "record verifier invocation: "+err.Error())
	}
	observations := make([]agentexec.Observation, 0, len(contextPayload.RequiredSubjects))
	for _, subject := range contextPayload.RequiredSubjects {
		observations = append(observations, agentexec.Observation{Subject: subject, Outcome: agentexec.OutcomePassed, Detail: "fixture verifier accepted the declared subject"})
	}
	result := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleVerifier, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomePassed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: contextPayload.RequiredEvidenceRefs, VerifierObservations: observations,
		Uncertainty: []string{}, Usage: e2eUsage("verify", 17, 5)}
	encoded, err := json.Marshal(result)
	if err != nil {
		processExit(2, "encode verifier response: "+err.Error())
	}
	_, _ = os.Stdout.Write(encoded)
	processExit(0, "")
}

// e2eAgentModel reports the model the Host configured for this role.
func e2eAgentModel() string {
	var config struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal([]byte(os.Getenv("MARKITECT_AGENT_CONFIG_JSON")), &config)
	return config.Model
}

func e2eUsage(phase string, inputTokens, outputTokens int64) *agentexec.Usage {
	for _, omitted := range strings.Split(os.Getenv(e2eOmitUsageEnv), ",") {
		if omitted == phase {
			return nil
		}
	}
	return &agentexec.Usage{Source: "provider-reported", InputTokens: &inputTokens, OutputTokens: &outputTokens}
}

func TestProjectRunExecutorRejectsForeignPathProposal(t *testing.T) {
	// The main flow proves the positive protocol; this table-level process check
	// protects the actual subprocess response parser and authority validator.
	base := candidateData{Files: map[string]File{}}
	project := &Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "orders", Owns: []string{"src/orders/"}}}}}
	limits := projectconfig.Limits{MaxCandidateFileBytes: 1024, MaxCandidateBytes: 2048}
	_, err := applyProposal(base, []agentexec.CandidateFile{{Path: "src/inventory/foreign.txt", Mode: "0644", Content: "x"}},
		projectwork.Config{InventoryRoots: []string{"src"}}, project.Report,
		ManagerTask{ManagerID: "orders", Owns: []string{"src/orders/"}}, "work", nil, limits)
	if err == nil || !strings.Contains(err.Error(), "proposed path") {
		t.Fatalf("foreign-path candidate was not rejected: %v", err)
	}
}

func enableE2EReviews(t *testing.T, root string, maxCost int64) {
	t.Helper()
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
		config.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{}, MaxRounds: 3, MaxManagerRounds: 2}
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

func makeMultilevelObligationFixture(t *testing.T) string {
	t.Helper()
	root := makeProjectRunFixture(t)
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	needle := "  - .markitect/model/orders/check.yaml\n"
	addition := needle + "  - .markitect/model/orders/commerce/manager.yaml\n  - .markitect/model/orders/commerce/statement.yaml\n  - .markitect/model/orders/commerce/artifact.yaml\n"
	updatedManifest := strings.Replace(string(manifest), needle, addition, 1)
	if updatedManifest == string(manifest) {
		t.Fatal("could not add Commerce model files to the multilevel fixture")
	}
	if err := os.WriteFile(manifestPath, []byte(updatedManifest), 0644); err != nil {
		t.Fatal(err)
	}
	commerceManager := strings.Replace(e2eChildManager("commerce"), "    namespace: \"\"\n    name: project-owner\n  owns: [src/commerce/]", "    namespace: orders\n    name: orders\n  owns: [src/orders/commerce/]", 1)
	commerceManager = strings.Replace(commerceManager, "  namespace: commerce\n", "  namespace: orders.commerce\n", 1)
	if strings.Contains(commerceManager, "namespace: \"\"\n    name: project-owner") {
		t.Fatal("failed to nest Commerce under Orders")
	}
	writeE2E(t, root, ".markitect/model/orders/commerce/manager.yaml", commerceManager)
	writeE2E(t, root, ".markitect/model/orders/commerce/statement.yaml", e2eStatement("orders.commerce", "commerce-work", "Resolve nested Commerce cancellation obligations."))
	commerceArtifact := "apiVersion: " + projectmodel.APIVersion + "\nkind: Artifact\nmetadata:\n  name: commerce-code\n  namespace: orders.commerce\npurpose: Implements the Commerce-owned nested source behavior.\nspec:\n  role: implementation\n  realizes:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Statement\n      namespace: orders.commerce\n      name: commerce-work\n  paths: [src/orders/commerce/implementation.txt]\n  required: false\n"
	writeE2E(t, root, ".markitect/model/orders/commerce/artifact.yaml", commerceArtifact)
	writeE2E(t, root, "src/orders/commerce/implementation.txt", "commerce implementation v1\n")
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "-m", "add nested Commerce obligation fixture")
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		commerceID := e2eManagerID("orders.commerce", "commerce")
		runtime.Agents[commerceID] = runtime.Agents[e2eManagerID("orders", "orders")]
		reviewAgents := make(map[string]projectconfig.Agent, len(runtime.Agents))
		for managerID, agent := range runtime.Agents {
			reviewAgents[managerID] = agent
		}
		runtime.Review = &projectconfig.ReviewConfig{Agents: reviewAgents, MaxRounds: 3, MaxManagerRounds: 2}
		runtime.Limits.MaxStarts = 24
	})
	return root
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
	case e2eManagerID("orders.commerce", "commerce"):
		return "src/orders/commerce/implementation.txt", "commerce implementation v2\n"
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
	repository := testkit.NewRepo(t)
	// Guarded Apply must exercise its successful path on a writable branch.
	// testkit's initial main branch is deliberately protected by the product.
	repository.Git("switch", "-c", "dev/process-fixture")
	root := repository.Dir
	setRuntimeFixtureEnv(t, e2eRootEnv, root)
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
	args := runtimeFixtureArgs(t, []string{"-test.run=^TestProjectRunExecutorProcess$"})
	checkCommand := filepath.Base(executable)
	checkCommandLine := runtimeFixtureCommand(t, checkCommand, "-test.run=^TestProjectRunCheckProcess$")
	buildAgent := func() projectconfig.Agent {
		return projectconfig.Agent{Command: executable, Args: args, Model: "fixture-model", ProviderVersion: "e2e-process-v1", Timeout: projectconfig.Duration(30 * time.Second),
			MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20, Environment: []string{"PATH", e2eExecutorEnv, e2eCheckEnv, e2eLogEnv, e2eBehaviorEnv, e2eRootEnv, e2eOmitUsageEnv, e2eIntegrationCheckEnv},
			Pricing: projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}}
	}
	config := projectconfig.Runtime{APIVersion: APIVersion, Mode: projectconfig.ModeControlledLocal, Agents: map[string]projectconfig.Agent{
		e2eManagerID("", "project-owner"):      buildAgent(),
		e2eManagerID("orders", "orders"):       buildAgent(),
		e2eManagerID("inventory", "inventory"): buildAgent(),
	}, Limits: projectconfig.Limits{MaxDepth: 4, MaxStarts: 16, MaxRetries: 0, MaxParallel: 1, MaxDuration: projectconfig.Duration(4 * time.Minute), MaxCostMicros: 100000,
		MaxCandidateFileBytes: 4096, MaxCandidateBytes: 16384}}
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, projectconfig.RuntimePath, string(configBytes))
	// The check argv is modeled as a literal bare executable plus fixed args.
	for _, file := range []string{".markitect/model/orders/check.yaml", ".markitect/model/inventory/check.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Replace(string(data), "command: [go, test, ./orders]", "command: "+checkCommandLine, 1)
		text = strings.Replace(text, "command: [go, test, ./inventory]", "command: "+checkCommandLine, 1)
		writeE2E(t, root, file, text)
	}
	repository.Commit("project run process fixture")
	return root
}

func updateE2ERuntime(t *testing.T, root string, update func(*projectconfig.Runtime)) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(projectconfig.RuntimePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var runtime projectconfig.Runtime
	if err := yaml.Unmarshal(data, &runtime); err != nil {
		t.Fatal(err)
	}
	update(&runtime)
	bind := func(agent projectconfig.Agent) projectconfig.Agent {
		if agent.Transport == projectconfig.TransportProcess {
			agent.Args = runtimeFixtureArgs(t, agent.Args)
		}
		return agent
	}
	for id, agent := range runtime.Agents {
		runtime.Agents[id] = bind(agent)
	}
	if runtime.Review != nil {
		for id, agent := range runtime.Review.Agents {
			runtime.Review.Agents[id] = bind(agent)
		}
	}
	if runtime.Verifier != nil {
		verifier := bind(*runtime.Verifier)
		runtime.Verifier = &verifier
	}
	updated, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, projectconfig.RuntimePath, string(updated))
	gitE2E(t, root, "add", projectconfig.RuntimePath)
	gitE2E(t, root, "commit", "-m", "adjust projectrun test runtime")
}

func setupE2EProcess(t *testing.T, behavior string) {
	t.Helper()
	setRuntimeFixtureEnv(t, e2eExecutorEnv, "1")
	setRuntimeFixtureEnv(t, e2eCheckEnv, "1")
	setRuntimeFixtureEnv(t, e2eBehaviorEnv, behavior)
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

func hasNoActiveManagerActions(tasks []ManagerTask) bool {
	for _, task := range tasks {
		switch task.State {
		case "planned", "pending", "queued", "delegated", "integrated-child", "worked", "integrated", "complete":
		default:
			return false
		}
	}
	return len(tasks) > 0
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
	return testkit.Git(t, root, args...)
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
