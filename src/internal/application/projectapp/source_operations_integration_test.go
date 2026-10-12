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
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
	"go.yaml.in/yaml/v3"
)

const (
	sourceOperationsWorkerEnv      = "MARKITECT_SOURCE_OPERATIONS_WORKER"
	sourceOperationsChangePathEnv  = "MARKITECT_SOURCE_OPERATIONS_CHANGE_PATH"
	sourceOperationsChangeBodyEnv  = "MARKITECT_SOURCE_OPERATIONS_CHANGE_BODY"
	sourceOperationsCheckEnv       = "MARKITECT_SOURCE_OPERATIONS_CHECK"
	sourceOperationsCheckModeEnv   = "MARKITECT_SOURCE_OPERATIONS_CHECK_MODE"
	sourceOperationsVerifyLogEnv   = "MARKITECT_SOURCE_OPERATIONS_VERIFY_LOG"
	sourceOperationsLegacyPath     = "src/orders/legacy.txt"
	sourceOperationsLegacyContents = "legacy orders bytes\n"
	sourceOperationsManifestPath   = ".markitect/project.yaml"
)

// TestSourceOperationsWorker is a provider-free process fixture for the real
// Operations composition. Each test gets its own descriptor and child env.
func TestSourceOperationsWorker(t *testing.T) {
	if os.Getenv(sourceOperationsWorkerEnv) != "1" {
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
		Kind    string `json:"kind"`
		Phase   string `json:"phase"`
		Manager struct {
			Manager struct {
				ID string `json:"id"`
			} `json:"manager"`
		} `json:"manager"`
		ManagerID      string   `json:"managerId"`
		DirectChildren []string `json:"directChildren"`
		Required       []string `json:"requiredSubjects"`
		Strictness     struct {
			Counterexamples int `json:"counterexamples"`
		} `json:"strictness"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &request); err != nil {
		t.Fatal(err)
	}
	managerID := request.Manager.Manager.ID
	if managerID == "" {
		managerID = request.ManagerID
	}
	if logPath := os.Getenv(sourceOperationsVerifyLogEnv); logPath != "" && request.Kind == "projectrun-full-verify/v1" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewEncoder(f).Encode(map[string]string{"managerId": managerID}); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}

	var report any
	candidates := []agentexec.CandidateFile{}
	switch request.Kind {
	case "projectrun-review/v1":
		report = map[string]any{"status": "pass", "summary": "bounded fixture review passed", "findings": []string{}}
	case "projectrun-full-verify/v1":
		if len(request.Required) == 0 {
			t.Fatal("full verification omitted required subjects")
		}
		status := "pass"
		findings := []string{}
		if managerID == deliveryManagerID("inventory", "inventory") {
			for _, artifact := range invocation.Request.Artifacts {
				if strings.Contains(string(artifact.Content), "UNIMPACTED_DRIFT") {
					status = "fail"
					findings = append(findings, "unimpacted inventory artifact contains drift")
					break
				}
			}
		}
		assessments := make([]map[string]string, 0, len(request.Required))
		for _, subject := range request.Required {
			outcome := "pass"
			if status == "fail" {
				outcome = "fail"
			}
			assessments = append(assessments, map[string]string{"subject": subject, "outcome": outcome, "detail": "bounded snapshot audit"})
		}
		counterexamples := make([]map[string]any, 0, request.Strictness.Counterexamples)
		for i := 0; i < request.Strictness.Counterexamples; i++ {
			ref := "manager:" + managerID
			if len(request.Required) > 0 {
				ref = request.Required[i%len(request.Required)]
			}
			counterexamples = append(counterexamples, map[string]any{"expected": fmt.Sprintf("accepted obligation %d", i+1), "observed": fmt.Sprintf("fixture inspection evidence %d", i+1), "evidenceRefs": []string{ref}})
		}
		report = map[string]any{"status": status, "summary": "bounded fixture audit", "assessments": assessments, "findings": findings, "counterexamples": counterexamples}
	default:
		delegations := []map[string]string{}
		status := "no-op"
		if request.Phase == "work" && managerID == deliveryManagerID("", "project-owner") {
			status = "complete"
			for _, child := range request.DirectChildren {
				delegations = append(delegations, map[string]string{"managerId": child, "goal": "Implement the owned artifact and report evidence."})
			}
		}
		if request.Phase == "integrate" {
			status = "complete"
		}
		changeOwner := deliveryManagerID("orders", "orders")
		if os.Getenv(sourceOperationsChangePathEnv) == sourceOperationsManifestPath {
			changeOwner = deliveryManagerID("", "project-owner")
		}
		if request.Phase == "work" && managerID == changeOwner {
			path, body := os.Getenv(sourceOperationsChangePathEnv), os.Getenv(sourceOperationsChangeBodyEnv)
			if path != "" {
				candidates = []agentexec.CandidateFile{{Path: path, Mode: "0644", Content: body}}
				status = "complete"
			}
		}
		integrated := request.Phase == "integrate"
		summary := managerID + " completed " + request.Phase
		if status == "no-op" {
			summary = "no change was warranted for the accepted scope"
		}
		if integrated {
			summary = "integrated child results"
		}
		report = map[string]any{"status": status, "summary": summary, "delegations": delegations, "reworkRequests": []any{}, "integrated": integrated, "escalateTo": "",
			"questions": []any{}, "risks": []any{}, "resolvedQuestions": []any{}, "resolvedRisks": []any{}}
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	inTokens, outTokens := int64(10), int64(5)
	response, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: candidates, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, ReportJSON: reportJSON,
		Uncertainty: []string{}, Usage: &agentexec.Usage{Source: "provider-reported", InputTokens: &inTokens, OutputTokens: &outTokens}})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(response, '\n'))
	os.Exit(0)
}

// TestSourceOperationsCheckWorker provides check behavior that the shared
// fixture's default check does not need: stdout with its cwd or a file-content
// assertion for the transitional-file scenario.
func TestSourceOperationsCheckWorker(t *testing.T) {
	if os.Getenv(sourceOperationsCheckEnv) != "1" {
		return
	}
	if os.Getenv(sourceOperationsCheckModeEnv) == "legacy" {
		content, err := os.ReadFile(sourceOperationsLegacyPath)
		if err != nil || string(content) != sourceOperationsLegacyContents {
			fmt.Fprintf(os.Stderr, "modelled file check failed: content=%q error=%v\n", content, err)
			os.Exit(1)
		}
		fmt.Println("observed " + sourceOperationsLegacyPath)
	} else {
		for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
			if _, err := os.Stat(path); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("check cwd: " + cwd)
	}
	os.Exit(0)
}

func sourceOperationsConfigureWorker(t *testing.T, root, changePath, changeBody, verifyLog string) {
	t.Helper()
	fixture := newDeliveryProcessFixture(t)
	fixture.env[sourceOperationsWorkerEnv] = "1"
	fixture.env[sourceOperationsChangePathEnv] = changePath
	fixture.env[sourceOperationsChangeBodyEnv] = changeBody
	fixture.env[sourceOperationsVerifyLogEnv] = verifyLog
	writeDeliveryProcessFixture(t, fixture)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := deliveryProcessArgs(t, []string{"-test.run=^TestSourceOperationsWorker$"})
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	configure := func(agent projectconfig.Agent) projectconfig.Agent {
		agent.Command = filepath.Base(executable)
		agent.Args = args
		agent.Environment = append(agent.Environment, sourceOperationsWorkerEnv, sourceOperationsChangePathEnv, sourceOperationsChangeBodyEnv, sourceOperationsVerifyLogEnv)
		return agent
	}
	for id, agent := range runtime.Agents {
		runtime.Agents[id] = configure(agent)
	}
	if runtime.Review != nil {
		for id, agent := range runtime.Review.Agents {
			runtime.Review.Agents[id] = configure(agent)
		}
	}
	encoded, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, projectconfig.RuntimePath, string(encoded))
}

func sourceOperationsConfigureNoOpStrictness(t *testing.T, root string) {
	t.Helper()
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	runtime.Strictness = &projectconfig.StrictnessConfig{
		Default:  projectconfig.StrictnessProfile{Evidence: []string{"project checks reviewed"}, Counterexamples: 1},
		Managers: map[string]projectconfig.StrictnessProfile{deliveryManagerID("orders", "orders"): {Evidence: []string{"orders contract review"}, Counterexamples: 1}},
	}
	encoded, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, projectconfig.RuntimePath, string(encoded))
}

func sourceOperationsConfigureChecks(t *testing.T, root, mode string) {
	t.Helper()
	fixture := newDeliveryProcessFixture(t)
	fixture.env[sourceOperationsCheckEnv] = "1"
	fixture.env[sourceOperationsCheckModeEnv] = mode
	writeDeliveryProcessFixture(t, fixture)
	for _, path := range []string{".markitect/model/orders/check.yaml", ".markitect/model/inventory/check.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(data), "TestDeliveryCheckWorker", "TestSourceOperationsCheckWorker", 1)
		if updated == string(data) {
			t.Fatalf("could not replace fixture check process in %s", path)
		}
		deliveryWrite(t, root, path, updated)
	}
}

func sourceOperationsRegenerateDocument(t *testing.T, operations Operations, root string) {
	t.Helper()
	selection := Selection{Root: root}
	preview, err := operations.Document(DocumentOperation{Selection: selection})
	if err != nil {
		t.Fatalf("document preview: %v", err)
	}
	written, err := operations.Document(DocumentOperation{Selection: selection, Write: true, ExpectedDigest: preview.Digest})
	if err != nil || !written.Written {
		t.Fatalf("write generated document: result=%+v err=%v", written, err)
	}
}

func sourceOperationsPlan(t *testing.T, operations Operations, root string, request projectrun.PlanRequest) projectrun.PlanRecord {
	t.Helper()
	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: request})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	return plan
}

func sourceOperationsCommitInputs(t *testing.T, root, message string) {
	t.Helper()
	deliveryGit(t, root, "add", ".")
	deliveryGit(t, root, "commit", "-m", message)
}

func sourceOperationsRun(t *testing.T, operations Operations, root string, plan projectrun.PlanRecord) projectrun.RunReport {
	t.Helper()
	run, err := operations.Run(context.Background(), RunOperation{Root: root, RunID: plan.ID})
	if err != nil || run.Status != projectrun.StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	return run
}

func sourceOperationsDigest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sourceOperationsContains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func TestVerifyChecksSeeFilesModelledInPlace(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	operations := deliveryOperations()
	manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourceOperationsManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	transitional := strings.Replace(string(manifest), "exclusions: []\n", "exclusions: []\ntransitionalExclusions:\n  - path: "+sourceOperationsLegacyPath+"\n    reason: Existing file awaits explicit modeling\n", 1)
	if transitional == string(manifest) {
		t.Fatal("could not add a transitional exclusion")
	}
	deliveryWrite(t, root, sourceOperationsManifestPath, transitional)
	deliveryWrite(t, root, sourceOperationsLegacyPath, sourceOperationsLegacyContents)
	artifactPath := ".markitect/model/orders/artifact.yaml"
	artifact, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifactPath)))
	if err != nil {
		t.Fatal(err)
	}
	updatedArtifact := strings.Replace(string(artifact), "paths: [src/orders/implementation.txt]", "paths: [src/orders/implementation.txt, "+sourceOperationsLegacyPath+"]", 1)
	if updatedArtifact == string(artifact) {
		t.Fatal("could not add the transitional file to the artifact")
	}
	deliveryWrite(t, root, artifactPath, updatedArtifact)
	sourceOperationsConfigureChecks(t, root, "legacy")
	sourceOperationsRegenerateDocument(t, operations, root)
	deliveryGit(t, root, "add", ".")
	deliveryGit(t, root, "commit", "-m", "introduce transitional legacy orders file")

	// Commit the explicit model transition before planning: the legacy file is
	// now owned by Orders and no longer excluded. The worker change below is a
	// normal implementation edit to another Orders-owned source file.
	deliveryWrite(t, root, sourceOperationsManifestPath, string(manifest))
	sourceOperationsRegenerateDocument(t, operations, root)
	sourceOperationsCommitInputs(t, root, "model legacy orders file in place")
	sourceOperationsConfigureWorker(t, root, "src/orders/implementation.txt", "orders implementation v2\n", "")
	sourceOperationsCommitInputs(t, root, "configure modelled-file process fixture")
	request := projectrun.PlanRequest{Operation: projectrun.OperationApply, Goal: "Model the legacy orders file in place.", Managers: deliveryManagers(), ExecuteAuthorized: true}
	sourceOperationsCreateAcknowledgedScope(t, operations, root, &request, "legacy-file", "legacy-file-change")
	plan := sourceOperationsPlan(t, operations, root, request)
	run := sourceOperationsRun(t, operations, root, plan)
	verified, err := operations.Verify(context.Background(), RunOperation{Root: root, RunID: run.ID})
	if err != nil || verified.Status != projectrun.StatusVerified {
		t.Fatalf("Verify: report=%+v err=%v", verified, err)
	}
	assertSourceOperationsChecksObservedLegacy(t, verified.Checks)
	if verified.ManagerVerification == nil || verified.ManagerVerification.Status != "passed" ||
		verified.ManagerVerification.CandidateID != run.Candidate.ID || verified.ManagerVerification.CandidateID != verified.CandidateID {
		t.Fatalf("verification did not bind a passing whole-project audit: %+v", verified.ManagerVerification)
	}
	preflight, err := operations.PreflightApply(PreflightOperation{Root: root, RunID: run.ID, CandidateID: run.Candidate.ID})
	if err != nil {
		t.Fatalf("modelled-file candidate preflight: %v", err)
	}
	applied, err := operations.Apply(ApplyOperation{Root: root, Request: projectrun.ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree}})
	if err != nil || applied.Status != projectrun.StatusApplied {
		t.Fatalf("apply modelled-file candidate: report=%+v err=%v", applied, err)
	}
	deliveryGit(t, root, append([]string{"add", "--"}, applied.Written...)...)
	deliveryGit(t, root, "commit", "-m", "apply modelled-file candidate")
	appliedRevision := deliveryGit(t, root, "rev-parse", "HEAD")
	appliedProject, err := operations.Host.Load(root, appliedRevision)
	if err != nil {
		t.Fatalf("load committed applied project: %v", err)
	}
	if verified.ManagerVerification.SnapshotDigest != appliedProject.Snapshot.Digest() {
		t.Fatalf("whole-project audit snapshot %q differs from the committed applied project snapshot %q", verified.ManagerVerification.SnapshotDigest, appliedProject.Snapshot.Digest())
	}
	full, err := operations.FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: appliedRevision}})
	if err != nil || full.Status != "passed" {
		t.Fatalf("FullVerify: report=%+v err=%v", full, err)
	}
	if full.SnapshotDigest != appliedProject.Snapshot.Digest() {
		t.Fatalf("FullVerify snapshot %q differs from the committed applied project snapshot %q", full.SnapshotDigest, appliedProject.Snapshot.Digest())
	}
	assertSourceOperationsFullChecksObservedLegacy(t, full.Checks)
}

func assertSourceOperationsChecksObservedLegacy(t *testing.T, checks []projectrun.CheckResult) {
	t.Helper()
	if len(checks) == 0 {
		t.Fatal("no declared check ran")
	}
	for _, check := range checks {
		if check.Outcome != "passed" || !strings.Contains(check.Stdout, "observed "+sourceOperationsLegacyPath) {
			t.Fatalf("check %s did not observe the modelled file: %+v", check.ID, check)
		}
	}
}

func assertSourceOperationsFullChecksObservedLegacy(t *testing.T, checks []projectrun.CheckResult) {
	t.Helper()
	assertSourceOperationsChecksObservedLegacy(t, checks)
}

func TestVerifyRunsDeclaredChecksOutsideTheRunStore(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	operations := deliveryOperations()
	sourceOperationsConfigureChecks(t, root, "cwd")
	sourceOperationsConfigureWorker(t, root, "", "", "")
	sourceOperationsCommitInputs(t, root, "configure isolated check process fixture")
	request := projectrun.PlanRequest{Operation: projectrun.OperationApply, Goal: "Verify declared checks run in an isolated working directory.", Managers: deliveryManagers(), ExecuteAuthorized: true}
	sourceOperationsCreateAcknowledgedScope(t, operations, root, &request, "isolated-check", "isolated-check-change")
	plan := sourceOperationsPlan(t, operations, root, request)
	run := sourceOperationsRun(t, operations, root, plan)
	verified, err := operations.Verify(context.Background(), RunOperation{Root: root, RunID: run.ID})
	if err != nil || verified.Status != projectrun.StatusVerified {
		t.Fatalf("Verify: report=%+v err=%v", verified, err)
	}
	cleanRoot := strings.ToLower(filepath.Clean(root))
	for _, check := range verified.Checks {
		_, cwd, found := strings.Cut(strings.TrimSpace(check.Stdout), "check cwd: ")
		if !found {
			t.Fatalf("check did not report its working directory: %+v", check)
		}
		if strings.HasPrefix(strings.ToLower(filepath.Clean(cwd)), cleanRoot) {
			t.Fatalf("check ran inside the repository run store: %s", cwd)
		}
	}
}

func TestTargetedPlanStillFullVerifiesUnimpactedManagerDrift(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	logPath := filepath.Join(t.TempDir(), "full-verify.jsonl")
	deliveryWrite(t, root, "src/inventory/implementation.txt", "UNIMPACTED_DRIFT\n")
	deliveryGit(t, root, "add", "src/inventory/implementation.txt")
	deliveryGit(t, root, "commit", "-m", "introduce unimpacted fixture drift")
	deliveryProcess := newDeliveryProcessFixture(t)
	deliveryProcess.env[sourceOperationsVerifyLogEnv] = logPath
	writeDeliveryProcessFixture(t, deliveryProcess)
	operations := deliveryOperations()
	sourceOperationsConfigureWorker(t, root, "", "", logPath)
	sourceOperationsCommitInputs(t, root, "configure full verification worker")
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	plan := sourceOperationsPlan(t, operations, root, projectrun.PlanRequest{Goal: "Implement the requested Orders change", Managers: []string{deliveryManagerID("orders", "orders")}})
	if len(plan.Managers) != 2 || plan.Managers[0].ManagerID != deliveryManagerID("", "project-owner") || plan.Managers[1].ManagerID != deliveryManagerID("orders", "orders") {
		t.Fatalf("Plan should stay scoped to root and Orders: %+v", plan.Managers)
	}
	full, err := operations.FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err == nil || full.Status != "failed" {
		t.Fatalf("full Verify did not reject unimpacted drift: status=%s err=%v", full.Status, err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var auditedInventory bool
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		var entry struct {
			ManagerID string `json:"managerId"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("decode full-verification audit record: %v", err)
		}
		if entry.ManagerID == deliveryManagerID("inventory", "inventory") {
			auditedInventory = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if !auditedInventory {
		t.Fatalf("unimpacted Inventory was not independently audited: %s", data)
	}
}

func sourceOperationsCreateAcknowledgedScope(t *testing.T, operations Operations, root string, request *projectrun.PlanRequest, explorationID, scopeID string) {
	t.Helper()
	operation, err := projectrun.NormalizeOperation(request.Operation)
	if err != nil {
		t.Fatal(err)
	}
	request.Operation = operation
	request.ExplorationID, request.ScopeID = explorationID, scopeID
	// Keep full selection explicit as an empty array in both the plan request
	// and named scope; a nil scope ManagerIDs slice is invalid record data.
	request.Managers = append([]string{}, request.Managers...)
	selection := Selection{Root: root}
	record := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: explorationID, Status: projectexplore.StatusActive,
		Request:   request.Goal,
		Scopes:    []projectexplore.Scope{{ID: scopeID, Name: scopeID, Goal: request.Goal, Operation: operation, ManagerIDs: append([]string{}, request.Managers...)}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	preview, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: preview.Plan.Digest}); err != nil {
		t.Fatal(err)
	}
	ack := StructureAcknowledgementInput{Actor: "fixture-user", Authority: "approved for test", Provenance: "source-operations-test", RecordedAt: time.Now().UTC()}
	ready, err := operations.Readiness(ReadinessOperation{Selection: selection, ExplorationID: explorationID, ScopeID: scopeID, Acknowledgement: &ack})
	if err != nil {
		t.Fatal(err)
	}
	if !ready.Readiness.Ready {
		t.Fatalf("source-operation scope is not ready: %+v", ready.Readiness)
	}
	if _, err := operations.Readiness(ReadinessOperation{Selection: selection, ExplorationID: explorationID, ScopeID: scopeID, Acknowledgement: &ack, Write: true, ExpectedDigest: ready.WritePlan.Digest}); err != nil {
		t.Fatal(err)
	}
}

func TestFullCoverageBlocksUnclassifiedOrdinaryFile(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	deliveryWrite(t, root, "unclassified.txt", "ordinary repository content\n")
	deliveryGit(t, root, "add", "unclassified.txt")
	deliveryGit(t, root, "commit", "-m", "add unclassified ordinary file")
	operations := deliveryOperations()
	coverage, err := operations.Coverage(Selection{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Accounted {
		t.Fatalf("full coverage accepted an unclassified ordinary file: %+v", coverage)
	}
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	full, err := operations.FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err == nil || full.Status == "passed" {
		t.Fatalf("full Verify closed despite an unclassified ordinary file: report=%+v err=%v", full, err)
	}
}

func TestCleanupAndReconcileNoOpVerifyFullTreeAndGuardedApply(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{projectrun.OperationCleanup, projectrun.OperationReconcile} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			root := deliveryFixture(t)
			operations := deliveryOperations()
			sourceOperationsConfigureWorker(t, root, "", "", "")
			sourceOperationsConfigureNoOpStrictness(t, root)
			sourceOperationsCommitInputs(t, root, "configure no-op operation workers")
			request := projectrun.PlanRequest{Operation: operation, Goal: "Review the complete project and improve only if warranted", ExecuteAuthorized: true}
			sourceOperationsCreateAcknowledgedScope(t, operations, root, &request, "no-op-"+operation, "no-op-scope")
			plan := sourceOperationsPlan(t, operations, root, request)
			if len(plan.Managers) != 3 || len(plan.Checks) != 2 {
				t.Fatalf("%s plan did not include every Manager and check: managers=%d checks=%d", operation, len(plan.Managers), len(plan.Checks))
			}
			ordersStrictness := plan.Strictness[deliveryManagerID("orders", "orders")]
			if ordersStrictness.Counterexamples != 2 || !sourceOperationsContains(ordersStrictness.Evidence, "project checks reviewed") || !sourceOperationsContains(ordersStrictness.Evidence, "orders contract review") {
				t.Fatalf("%s bypassed or replaced additive Orders strictness: %+v", operation, ordersStrictness)
			}
			run := sourceOperationsRun(t, operations, root, plan)
			for _, task := range run.Tasks {
				if task.ManagerID != deliveryManagerID("", "project-owner") && task.ReportStatus != "no-op" {
					t.Errorf("Manager %s did not report a justified no-op: %s", task.ManagerID, task.ReportStatus)
				}
			}
			verified, err := operations.Verify(context.Background(), RunOperation{Root: root, RunID: run.ID})
			if err != nil || verified.Status != projectrun.StatusVerified || verified.VerificationScope != "full" {
				t.Fatalf("%s no-op did not pass checks and full Verify: status=%s err=%v scope=%s", operation, verified.Status, err, verified.VerificationScope)
			}
			preflight, err := operations.PreflightApply(PreflightOperation{Root: root, RunID: run.ID, CandidateID: run.Candidate.ID})
			if err != nil || len(preflight.Paths) != 0 {
				t.Fatalf("%s no-op preflight = %+v, %v; want guarded empty-path apply", operation, preflight, err)
			}
			untracked := filepath.Join(root, "untracked-ordinary.txt")
			if err := os.WriteFile(untracked, []byte("new ordinary file\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, staleErr := operations.Apply(ApplyOperation{Root: root, Request: projectrun.ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
				ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree}})
			if staleErr == nil {
				t.Fatal("Apply accepted a changed repository after no-op preflight")
			}
			if err := os.Remove(untracked); err != nil {
				t.Fatal(err)
			}
			preflight, err = operations.PreflightApply(PreflightOperation{Root: root, RunID: run.ID, CandidateID: run.Candidate.ID})
			if err != nil {
				t.Fatalf("fresh no-op preflight: %v", err)
			}
			applied, err := operations.Apply(ApplyOperation{Root: root, Request: projectrun.ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
				ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree}})
			if err != nil || applied.Status != projectrun.StatusApplied || len(applied.Written) != 0 {
				t.Fatalf("%s verified no-op Apply = %+v, %v", operation, applied, err)
			}
		})
	}
}

func TestCleanupSourceChangeGeneratesAndAppliesExactVerifiedDocument(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	operations := deliveryOperations()
	initialDocument, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatal(err)
	}
	// Keep a valid generated-document marker but make the committed view stale.
	// CandidateDocument should then add the exact current Host rendering to the
	// candidate alongside the source edit.
	deliveryWrite(t, root, initialDocument.Path, initialDocument.Content+"\nStale committed fixture content.\n")
	sourceOperationsConfigureWorker(t, root, "src/orders/implementation.txt", "orders implementation v2\n", "")
	sourceOperationsCommitInputs(t, root, "configure cleanup source-change worker")
	request := projectrun.PlanRequest{Operation: projectrun.OperationCleanup,
		Goal: "Review the complete project and improve Orders where the accepted model permits it", ExecuteAuthorized: true}
	sourceOperationsCreateAcknowledgedScope(t, operations, root, &request, "cleanup-source", "cleanup-source-change")
	plan := sourceOperationsPlan(t, operations, root, request)
	run := sourceOperationsRun(t, operations, root, plan)
	verified, err := operations.Verify(context.Background(), RunOperation{Root: root, RunID: run.ID})
	if err != nil || verified.Status != projectrun.StatusVerified {
		t.Fatalf("verify source and generated document together: status=%s err=%v", verified.Status, err)
	}
	preflight, err := operations.PreflightApply(PreflightOperation{Root: root, RunID: run.ID, CandidateID: run.Candidate.ID})
	if err != nil {
		t.Fatalf("source-change preflight: %v", err)
	}
	if !sourceOperationsContains(preflight.Paths, "src/orders/implementation.txt") {
		t.Fatalf("guarded Apply omitted the exact verified implementation path: %+v", preflight.Paths)
	}
	if !sourceOperationsContains(preflight.Paths, initialDocument.Path) {
		t.Fatalf("guarded Apply omitted the generated document: %+v", preflight.Paths)
	}
	applied, err := operations.Apply(ApplyOperation{Root: root, Request: projectrun.ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree}})
	if err != nil || applied.Status != projectrun.StatusApplied {
		t.Fatalf("apply verified source and generated document: report=%+v err=%v", applied, err)
	}
	contents, err := os.ReadFile(filepath.Join(root, "src/orders/implementation.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "orders implementation v2\n" {
		t.Fatalf("applied Orders source = %q", contents)
	}
	final, err := operations.Host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	document, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatal(err)
	}
	if string(final.Snapshot.Files[document.Path]) != document.Content {
		t.Fatal("applied README differs from the exact Host-generated document for the applied source snapshot")
	}
	if run.Candidate.Files[document.Path] != sourceOperationsDigest(document.Content) {
		t.Fatalf("candidate document digest %q does not bind the exact generated document %q", run.Candidate.Files[document.Path], sourceOperationsDigest(document.Content))
	}
	candidatePath := fmt.Sprintf(".markitect/runs/%s/candidates/%s.json", run.ID, run.Candidate.ID)
	storedBytes, err := operations.ReadRecord(root, candidatePath)
	if err != nil {
		t.Fatalf("read persisted candidate through the public Application record path: %v", err)
	}
	var stored struct {
		Files map[string]struct {
			Path    string `json:"path"`
			Content []byte `json:"content"`
			Delete  bool   `json:"delete,omitempty"`
		} `json:"files"`
	}
	if err := json.Unmarshal(storedBytes, &stored); err != nil {
		t.Fatalf("decode persisted candidate record: %v", err)
	}
	storedDocument, ok := stored.Files[document.Path]
	if !ok || storedDocument.Delete || storedDocument.Path != document.Path || string(storedDocument.Content) != document.Content {
		t.Fatalf("persisted candidate omitted the exact Application-generated document: %+v", storedDocument)
	}
	if run.Candidate.Files[document.Path] != sourceOperationsDigest(string(storedDocument.Content)) {
		t.Fatalf("persisted candidate document bytes do not match the public candidate digest: %q", run.Candidate.Files[document.Path])
	}
}
