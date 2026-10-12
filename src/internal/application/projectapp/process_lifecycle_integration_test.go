//go:build !runtime_unit

package projectapp

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
	"go.yaml.in/yaml/v3"
)

const (
	processLifecycleWorkerEnv   = "MARKITECT_APP_PROCESS_LIFECYCLE_WORKER"
	processLifecycleLogEnv      = "MARKITECT_APP_PROCESS_LIFECYCLE_LOG"
	processLifecycleVerifierEnv = "MARKITECT_APP_PROCESS_LIFECYCLE_VERIFIER"
)

// TestProjectRunProcessEndToEndResumeVerifyApply retains the real frontend,
// subprocess, required-check, freshness, and guarded-Apply integration claim.
func TestProjectRunProcessEndToEndResumeVerifyApply(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	logPath := filepath.Join(t.TempDir(), "process-lifecycle.jsonl")
	setDeliveryProcessFixtureEnv(t, processLifecycleWorkerEnv, "1")
	setDeliveryProcessFixtureEnv(t, processLifecycleLogEnv, logPath)
	setDeliveryProcessFixtureEnv(t, processLifecycleVerifierEnv, "1")
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	bindProcessLifecycleAgents(t, root)
	setProcessLifecycleConfiguration(t, root)

	operations := deliveryOperations()
	if _, err := source.IdentifyGit(root); err != nil {
		t.Fatal(err)
	}
	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root, Revision: deliveryHead(t, root)}, Request: projectrun.PlanRequest{
		Goal:     "Implement both owned artifacts, integrate the child work, and run the declared checks.",
		Managers: []string{deliveryManagerID("orders", "orders"), deliveryManagerID("inventory", "inventory")}, ExecuteAuthorized: true,
	}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Status != projectrun.StatusPlanned || len(plan.Managers) != 3 || len(plan.Checks) != 2 {
		t.Fatalf("unexpected bounded plan: status=%s managers=%d checks=%d", plan.Status, len(plan.Managers), len(plan.Checks))
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancellation at the durable pre-dispatch boundary must not record an
	// invocation or cause a subprocess to start.
	cancel()
	first, runErr := operations.Run(ctx, RunOperation{Root: root, RunID: plan.ID})
	if runErr == nil || first.Status != projectrun.StatusInterrupted {
		diagnostic, _ := os.ReadFile(logPath)
		t.Fatalf("Run should interrupt before dispatch at a durable boundary; status=%s err=%v; process log=%s", first.Status, runErr, diagnostic)
	}
	if len(first.Invocations) != 0 {
		t.Fatalf("pre-dispatch interruption recorded %d invocations, want none", len(first.Invocations))
	}
	if !processLifecycleNoActiveActions(first.Tasks) {
		t.Fatalf("interrupted report does not preserve a safe pre-dispatch boundary: %+v", first.Tasks)
	}
	resumed, err := operations.Resume(context.Background(), RunOperation{Root: root, RunID: plan.ID})
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.Status != projectrun.StatusIntegrated || !resumed.Candidate.Integrated {
		t.Fatalf("resume did not integrate the staged candidates: status=%s integrated=%v", resumed.Status, resumed.Candidate.Integrated)
	}
	var executorInvocations, reviewerInvocations int
	for _, invocation := range resumed.Invocations {
		switch invocation.Role {
		case agentexec.RoleExecutor:
			executorInvocations++
		case "reviewer":
			reviewerInvocations++
		}
	}
	if len(resumed.Invocations) != 4 || executorInvocations != 4 || reviewerInvocations != 0 {
		t.Fatalf("resume has %d invocations (executor=%d reviewer=%d), want exactly four executor phases with no replay", len(resumed.Invocations), executorInvocations, reviewerInvocations)
	}
	if err := assertAppProcessTrace(t, logPath, resumed); err != nil {
		t.Fatal(err)
	}

	verified, err := operations.Verify(context.Background(), RunOperation{Root: root, RunID: plan.ID})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verified.Status != projectrun.StatusVerified || len(verified.Checks) != 2 {
		t.Fatalf("actual required checks did not pass: status=%s checks=%+v", verified.Status, verified.Checks)
	}
	for _, check := range verified.Checks {
		if check.Outcome != "passed" || check.ExitCode != 0 || check.ExecutableDigest == "" {
			t.Fatalf("check was not an independently executed pinned process: %+v", check)
		}
	}

	preflight, err := operations.PreflightApply(PreflightOperation{Root: root, RunID: plan.ID, CandidateID: resumed.Candidate.ID})
	if err != nil {
		t.Fatal(err)
	}
	targetDigest, err := projectrun.CaptureTarget(root, preflight.Paths)
	if err != nil {
		t.Fatal(err)
	}
	branch := deliveryGit(t, root, "branch", "--show-current")
	head := deliveryHead(t, root)
	request := projectrun.ApplyRequest{RunID: plan.ID, PlanID: plan.ID, CandidateID: resumed.Candidate.ID,
		ExpectedVerificationDigest: verified.Digest, TargetBranch: branch, ExpectedHead: head, ExpectedWorktree: targetDigest}
	ordersPath := filepath.Join(root, "src", "orders", "implementation.txt")
	originalOrders, err := os.ReadFile(ordersPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ordersPath, []byte("manual change after verification\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, staleErr := operations.Apply(ApplyOperation{Root: root, Request: request}); staleErr == nil {
		t.Fatal("Apply accepted a working-tree change after verification")
	}
	if err := os.WriteFile(ordersPath, originalOrders, 0644); err != nil {
		t.Fatal(err)
	}
	request.ExpectedWorktree, err = projectrun.CaptureTarget(root, preflight.Paths)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := operations.Apply(ApplyOperation{Root: root, Request: request})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.Status != projectrun.StatusApplied || len(applied.Written) != 2 {
		t.Fatalf("Apply did not write both candidate artifacts: %+v", applied)
	}
	assertAppFileContents(t, root, "src/orders/implementation.txt", "orders implementation v2\n")
	assertAppFileContents(t, root, "src/inventory/implementation.txt", "inventory implementation v2\n")
}

// TestAppProcessLifecycleWorker is the exact provider-free descriptor target
// for the lifecycle test above. It uses the same ProcessInvoker JSON boundary.
func TestAppProcessLifecycleWorker(t *testing.T) {
	if os.Getenv(processLifecycleWorkerEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		processLifecycleExit(2, "read invocation: "+err.Error())
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		processLifecycleExit(2, "decode invocation: "+err.Error())
	}
	var payload struct {
		Phase          string   `json:"phase"`
		DirectChildren []string `json:"directChildren"`
		Manager        struct {
			Manager struct {
				ID      string   `json:"id"`
				Purpose string   `json:"purpose"`
				Owns    []string `json:"owns"`
			} `json:"manager"`
		} `json:"manager"`
		ResponseSchema json.RawMessage   `json:"responseSchema"`
		ChildReports   []json.RawMessage `json:"childReports"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &payload); err != nil || payload.Phase == "" || payload.Manager.Manager.ID == "" || len(payload.ResponseSchema) == 0 {
		processLifecycleExit(2, "request omitted manager, phase, or typed response schema")
	}
	if err := appendProcessLifecycleLog(processLifecycleLogEnv, map[string]any{
		"phase": payload.Phase, "managerId": payload.Manager.Manager.ID,
		"sourceRevision": invocation.Request.SourceRevision, "modelDigest": invocation.Request.ModelDigest,
		"modulePin": invocation.Request.ModulePin, "projectionId": invocation.Request.ProjectionID,
		"scopeIds": invocation.Request.ScopeIDs, "policyIds": invocation.Request.PolicyIDs,
		"artifacts": invocation.Request.Artifacts, "schema": payload.ResponseSchema,
		"managerPurpose": payload.Manager.Manager.Purpose, "managerOwns": payload.Manager.Manager.Owns,
		"contextDigest": processLifecycleDigest(invocation.Request.Context), "inputDigest": invocation.InputDigest,
		"childReports": payload.ChildReports,
	}); err != nil {
		processLifecycleExit(2, "record invocation: "+err.Error())
	}
	privateLog, err := os.OpenFile(os.Getenv("MARKITECT_AGENT_PRIVATE_LOG"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		processLifecycleExit(2, "create private process log: "+err.Error())
	}
	if _, err := privateLog.Write([]byte("fixture process completed\n")); err != nil {
		_ = privateLog.Close()
		processLifecycleExit(2, "write private process log: "+err.Error())
	}
	if err := privateLog.Close(); err != nil {
		processLifecycleExit(2, "close private process log: "+err.Error())
	}

	response := projectrun.TaskResponse{Status: "complete", Summary: payload.Manager.Manager.ID + " completed " + payload.Phase,
		Delegations: []projectrun.Delegation{}, ReworkRequests: []projectrun.ReworkRequest{}, Integrated: payload.Phase == "integrate",
		Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	files := []agentexec.CandidateFile{}
	if payload.Phase == "work" && payload.Manager.Manager.ID == deliveryManagerID("", "project-owner") {
		response.Delegations = make([]projectrun.Delegation, 0, len(payload.DirectChildren))
		for _, child := range payload.DirectChildren {
			response.Delegations = append(response.Delegations, projectrun.Delegation{ManagerID: child, Goal: "Implement the owned source artifact and report its evidence."})
		}
	} else if payload.Phase == "work" {
		var path, content string
		switch payload.Manager.Manager.ID {
		case deliveryManagerID("orders", "orders"):
			path, content = "src/orders/implementation.txt", "orders implementation v2\n"
		case deliveryManagerID("inventory", "inventory"):
			path, content = "src/inventory/implementation.txt", "inventory implementation v2\n"
		default:
			processLifecycleExit(2, "unexpected work manager "+payload.Manager.Manager.ID)
		}
		files = []agentexec.CandidateFile{{Path: path, Mode: "0644", Content: content}}
	} else if payload.Phase != "integrate" {
		processLifecycleExit(2, "unknown phase "+payload.Phase)
	}
	if payload.Phase == "integrate" && len(payload.ChildReports) != len(payload.DirectChildren) {
		processLifecycleExit(2, fmt.Sprintf("Manager integration saw %d child reports, want %d", len(payload.ChildReports), len(payload.DirectChildren)))
	}
	reportJSON, err := json.Marshal(response)
	if err != nil {
		processLifecycleExit(2, "encode task report: "+err.Error())
	}
	inputTokens, outputTokens := int64(41), int64(13)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: &inputTokens, OutputTokens: &outputTokens}
	result, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: files, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: reportJSON, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		processLifecycleExit(2, "encode response: "+err.Error())
	}
	_, _ = os.Stdout.Write(result)
	processLifecycleExit(0, "")
}

// TestAppProcessLifecycleVerifier is the process-backed Runtime Verify role.
func TestAppProcessLifecycleVerifier(t *testing.T) {
	if os.Getenv(processLifecycleVerifierEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		processLifecycleExit(2, "read verifier invocation: "+err.Error())
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		processLifecycleExit(2, "decode verifier invocation: "+err.Error())
	}
	var contextPayload struct {
		RequiredSubjects     []string `json:"requiredSubjects"`
		RequiredEvidenceRefs []string `json:"requiredEvidenceRefs"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextPayload); err != nil || len(contextPayload.RequiredSubjects) == 0 {
		processLifecycleExit(2, "verifier request omitted required subjects")
	}
	observations := make([]agentexec.Observation, 0, len(contextPayload.RequiredSubjects))
	for _, subject := range contextPayload.RequiredSubjects {
		observations = append(observations, agentexec.Observation{Subject: subject, Outcome: agentexec.OutcomePassed, Detail: "fixture verifier accepted the declared subject"})
	}
	inTokens, outTokens := int64(17), int64(5)
	result, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleVerifier, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomePassed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: contextPayload.RequiredEvidenceRefs, VerifierObservations: observations,
		Uncertainty: []string{}, Usage: &agentexec.Usage{Source: "provider-reported", InputTokens: &inTokens, OutputTokens: &outTokens}})
	if err != nil {
		processLifecycleExit(2, "encode verifier response: "+err.Error())
	}
	_, _ = os.Stdout.Write(result)
	processLifecycleExit(0, "")
}

func bindProcessLifecycleAgents(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(projectconfig.RuntimePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config projectconfig.Runtime
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := deliveryProcessArgs(t, []string{"-test.run=^TestAppProcessLifecycleWorker$"})
	bind := func(agent projectconfig.Agent) projectconfig.Agent {
		agent.Command = executable
		agent.Args = append([]string(nil), args...)
		agent.Environment = []string{"PATH", processLifecycleWorkerEnv, processLifecycleLogEnv}
		return agent
	}
	for id, agent := range config.Agents {
		config.Agents[id] = bind(agent)
	}
	config.Review = nil
	verifier := config.Agents[deliveryManagerID("", "project-owner")]
	verifier.Args = deliveryProcessArgs(t, []string{"-test.run=^TestAppProcessLifecycleVerifier$"})
	verifier.Environment = []string{"PATH", processLifecycleVerifierEnv}
	config.Verifier = &verifier
	updated, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, updated, 0644); err != nil {
		t.Fatal(err)
	}
	deliveryGit(t, root, "add", projectconfig.RuntimePath)
	deliveryGit(t, root, "commit", "-m", "bind the exact process lifecycle fixture")
}

func setProcessLifecycleConfiguration(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(".markitect/project.yaml"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), "workflowMode: guided\n", "workflowMode: empty\n", 1)
	if text == string(data) {
		t.Fatal("delivery fixture workflow mode was not guided")
	}
	beforeCoverageChange := text
	text = strings.Replace(text, "coverageMode: full\n", "coverageMode: selected\n", 1)
	if text == beforeCoverageChange {
		t.Fatal("delivery fixture coverage mode was not full")
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	deliveryGit(t, root, "add", ".markitect/project.yaml")
	deliveryGit(t, root, "commit", "-m", "select the bounded process lifecycle model")
}

func assertAppProcessTrace(t *testing.T, logPath string, run projectrun.RunReport) error {
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
	var rootIntegration map[string]any
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
		if phase == "work" && manager == deliveryManagerID("orders", "orders") && !processLifecycleContains(processLifecycleStringSlice(record["managerOwns"]), "src/orders/") {
			return fmt.Errorf("orders context omitted its owned repository path: %+v", record["managerOwns"])
		}
		if phase == "work" && manager == deliveryManagerID("inventory", "inventory") && !processLifecycleContains(processLifecycleStringSlice(record["managerOwns"]), "src/inventory/") {
			return fmt.Errorf("inventory context omitted its owned repository path: %+v", record["managerOwns"])
		}
		if manager == deliveryManagerID("", "project-owner") && phase == "integrate" {
			rootIntegration = record
		}
	}
	executorReceipts := 0
	for _, invocation := range run.Invocations {
		if invocation.Role == agentexec.RoleExecutor {
			executorReceipts++
		}
	}
	if len(records) != executorReceipts {
		return fmt.Errorf("executor process log has %d requests for %d durable executor receipts", len(records), executorReceipts)
	}
	if counts[deliveryManagerID("", "project-owner")+"|work"] != 1 || counts[deliveryManagerID("orders", "orders")+"|work"] != 1 || counts[deliveryManagerID("inventory", "inventory")+"|work"] != 1 || counts[deliveryManagerID("", "project-owner")+"|integrate"] != 1 {
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

func processLifecycleNoActiveActions(tasks []projectrun.ManagerTask) bool {
	for _, task := range tasks {
		switch task.State {
		case "planned", "pending", "queued", "delegated", "integrated-child", "worked", "integrated", "complete":
		default:
			return false
		}
	}
	return len(tasks) > 0
}

func appendProcessLifecycleLog(key string, value any) error {
	path := os.Getenv(key)
	if path == "" {
		return fmt.Errorf("process lifecycle log path is missing")
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

func processLifecycleDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func processLifecycleContains(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func processLifecycleStringSlice(value any) []string {
	items, _ := value.([]any)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func processLifecycleExit(code int, message string) {
	if message != "" {
		_, _ = fmt.Fprintln(os.Stderr, message)
		_ = appendProcessLifecycleLog(processLifecycleLogEnv, map[string]string{"processError": message})
	}
	os.Exit(code)
}

func deliveryHead(t *testing.T, root string) string {
	t.Helper()
	return deliveryGit(t, root, "rev-parse", "HEAD")
}

func assertAppFileContents(t *testing.T, root, relative, expected string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil || string(data) != expected {
		t.Fatalf("%s = %q, %v; want %q", relative, data, err, expected)
	}
}
