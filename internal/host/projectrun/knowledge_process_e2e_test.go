package projectrun

import (
	"bufio"
	"bytes"
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
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

// This regression joins validated provider-free ProjectRun records to the
// source-built knowledge CLI and MCP stdio process. The child test-binary agents
// below are deterministic protocol fixtures; they are not model actors.
func TestKnowledgeProcessEndToEndStoredRunEvidence(t *testing.T) {
	root := makeProjectRunFixture(t)
	knowledgeProcessAddRootReviewFixture(t, root)
	writeE2E(t, root, "src/project-integration.txt", "baseline integration summary\n")
	gitE2E(t, root, "add", "src/project-integration.txt")
	gitE2E(t, root, "commit", "--amend", "--no-edit")
	setupE2EProcess(t, "integration-review-fix")
	// Review is part of the stored chain, and the longer bound is local to this
	// disposable fixture because all lifecycle steps include real validation.
	updateE2ERuntime(t, root, func(config *Runtime) {
		config.Limits.MaxDuration = Duration(15 * time.Minute)
	})
	enableE2EReviews(t, root, 500000)
	baseRevision := identityHead(t, root)
	ordersManager := e2eManagerID("orders", "orders")
	inventoryManager := e2eManagerID("inventory", "inventory")
	foreignPlan, err := Plan(projectworkHost(), root, baseRevision, PlanRequest{Goal: "Plan private inventory-only work.",
		Managers: []string{inventoryManager}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("foreign-only Plan: %v", err)
	}
	planned, err := Plan(projectworkHost(), root, baseRevision, PlanRequest{Goal: "Exercise partial planned-only evidence.",
		Managers: []string{ordersManager}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("planned-only Plan: %v", err)
	}
	plan, err := Plan(projectworkHost(), root, baseRevision, PlanRequest{
		Goal:              "Implement both owned artifacts and run their declared checks.",
		Managers:          []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")},
		ExecuteAuthorized: true,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	if len(run.Reviews) < 2 {
		t.Fatalf("provider-free reviewer processes did not produce the expected review records: %+v", run.Reviews)
	}
	verified, err := Verify(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("Verify: status=%s err=%v", verified.Status, err)
	}
	paths, err := ApplyPaths(projectworkHost(), root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	worktreeDigest, err := CaptureTarget(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	apply, err := Apply(projectworkHost(), ProcessInvoker{}, root, ApplyRequest{
		RunID: plan.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
		ExpectedVerificationDigest: verified.Digest, TargetBranch: gitE2E(t, root, "branch", "--show-current"),
		ExpectedHead: identityHead(t, root), ExpectedWorktree: worktreeDigest,
	})
	if err != nil || apply.Status != StatusApplied {
		t.Fatalf("Apply: status=%s err=%v", apply.Status, err)
	}
	persisted, err := KnowledgeRecords(root, plan.ID)
	if err != nil {
		t.Fatalf("load persisted complete record set after Apply: %v", err)
	}
	if persisted.PlanState != KnowledgeRecordPresent || persisted.RunState != KnowledgeRecordPresent || persisted.CandidateState != KnowledgeRecordPresent || persisted.VerificationState != KnowledgeRecordPresent || persisted.ApplyState != KnowledgeRecordPresent {
		t.Fatalf("positive lifecycle did not persist every expected record: %+v", persisted)
	}
	plan = persisted.Status.Plan
	run = persisted.Status.Run

	fixtureHead := identityHead(t, root)
	if fixtureHead != baseRevision {
		t.Fatalf("Apply changed the committed fixture base: base=%s current=%s", baseRevision, fixtureHead)
	}
	ledgerPath := filepath.Join(root, filepath.FromSlash(RunsPath), plan.ID)
	ledgerBefore, err := knowledgeProcessTreeDigest(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	bin, buildInfo := buildKnowledgeProcessCLI(t)
	sourceHead := knowledgeProcessSourceHead(knowledgeProcessRepoRoot(t))
	if sourceHead != "" && (buildInfo.VCSRevision == "" || buildInfo.VCSModified == "" || buildInfo.VCSRevision != sourceHead) {
		t.Fatalf("source-built CLI revision=%s differs from recorded source HEAD=%s", buildInfo.VCSRevision, sourceHead)
	}
	baseGraph := knowledgeProcessCLI(t, bin, root, fixtureHead, "project", "graph", plan.ID)
	assertKnowledgeProcessCompleteGraph(t, baseGraph, run, plan, verified, apply, persisted, "current")
	if status := knowledgeProcessModelBinding(baseGraph); status != "current" {
		t.Fatalf("historical base view model binding = %q, want current", status)
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)

	// A model-only commit after Apply leaves the run as valid historical
	// evidence while making its current source/model binding stale.
	statementPath := filepath.Join(root, ".markitect", "model", "orders", "statement.yaml")
	statementBytes, err := os.ReadFile(statementPath)
	if err != nil {
		t.Fatal(err)
	}
	originalStatementBytes := append([]byte(nil), statementBytes...)
	statementBytes = bytes.Replace(statementBytes, []byte("  description: Implement the orders artifact.\n"), []byte("  description: Implement the orders artifact with revised goals.\n"), 1)
	if bytes.Equal(statementBytes, originalStatementBytes) {
		t.Fatal("model-only fixture edit did not change the Orders statement")
	}
	if err := os.WriteFile(statementPath, statementBytes, 0600); err != nil {
		t.Fatal(err)
	}
	gitE2E(t, root, "add", filepath.ToSlash(filepath.Join(".markitect", "model", "orders", "statement.yaml")))
	gitE2E(t, root, "commit", "-m", "change model for stale-binding fixture")
	modelOnlyRevision := identityHead(t, root)
	currentGraph := knowledgeProcessCLI(t, bin, root, "", "project", "graph", plan.ID)
	assertKnowledgeProcessCompleteGraph(t, currentGraph, run, plan, verified, apply, persisted, "stale")
	if status := knowledgeProcessModelBinding(currentGraph); status != "stale" {
		t.Fatalf("post-Apply working view model/source binding = %q, want stale", status)
	}
	baseBinding := knowledgeProcessMap(knowledgeProcessMap(baseGraph, "graph"), "binding")
	currentBinding := knowledgeProcessMap(knowledgeProcessMap(currentGraph, "graph"), "binding")
	if baseBinding["modelDigest"] == currentBinding["modelDigest"] || baseBinding["snapshotDigest"] == currentBinding["snapshotDigest"] {
		t.Fatalf("model-only stale fixture did not change both model and source snapshot bindings: base=%+v current=%+v", baseBinding, currentBinding)
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)
	if runtime := knowledgeProcessRunRuntimeBinding(currentGraph); runtime != "not-compared" {
		t.Fatalf("current provider runtime was implied: binding=%q", runtime)
	}
	currentMCP := knowledgeProcessMCP(t, bin, root, "", plan.ID, []knowledgeProcessQuery{{action: "graph"}})[0]
	if !bytes.Equal(knowledgeProcessCanonicalJSON(t, currentGraph), knowledgeProcessCanonicalJSON(t, currentMCP)) {
		t.Fatal("current working-source CLI and MCP graph results differ")
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)

	ordersGraph := knowledgeProcessCLI(t, bin, root, fixtureHead, ordersManager, "graph", plan.ID)
	ordersText := knowledgeProcessJSON(t, ordersGraph)
	privateArtifact, _ := json.Marshal([]string{projectModelAPIVersionForKnowledge(), "Artifact", "inventory", "inventory-code"})
	privateCheck, _ := json.Marshal([]string{projectModelAPIVersionForKnowledge(), "Check", "inventory", "inventory-check"})
	for _, forbidden := range []string{
		inventoryManager, string(privateArtifact), string(privateCheck), "src/inventory/",
		"inventory implementation v1", "inventory implementation v2",
	} {
		if knowledgeProcessContainsString(ordersGraph, forbidden) {
			t.Fatalf("Orders graph exposed foreign/private evidence %q", forbidden)
		}
	}
	if !strings.Contains(ordersText, "record/candidate/"+plan.ID) || !strings.Contains(ordersText, "record/check/"+plan.ID) {
		t.Fatalf("Orders graph omitted its selected candidate/check evidence: %s", ordersText)
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)
	privateInventoryArtifactBytes, _ := json.Marshal([]string{projectModelAPIVersionForKnowledge(), "Artifact", "inventory", "inventory-code"})
	privateTarget := string(privateInventoryArtifactBytes)
	missingTarget := "missing-private-target"
	privateCLIError := knowledgeProcessCLIError(t, bin, root, fixtureHead, ordersManager, plan.ID, privateTarget)
	missingCLIError := knowledgeProcessCLIError(t, bin, root, fixtureHead, ordersManager, plan.ID, missingTarget)
	if privateCLIError != missingCLIError || strings.Contains(privateCLIError, privateTarget) {
		t.Fatalf("private target did not fail exactly like an absent target: private=%q missing=%q", privateCLIError, missingCLIError)
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)
	privateMCPError, missingMCPError := knowledgeProcessMCPDenials(t, bin, root, fixtureHead, plan.ID, ordersManager, privateTarget, missingTarget)
	if privateMCPError != missingMCPError || strings.Contains(privateMCPError, privateTarget) {
		t.Fatalf("private MCP target did not fail exactly like an absent target: private=%q missing=%q", privateMCPError, missingMCPError)
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)

	// An inventory-only plan is unavailable in Orders scope and must not reveal
	// its ID. A separate Orders-only plan demonstrates honest partial evidence.
	foreignView := knowledgeProcessCLI(t, bin, root, fixtureHead, ordersManager, "graph", foreignPlan.ID)
	foreignText := knowledgeProcessJSON(t, foreignView)
	if knowledgeProcessContainsString(foreignView, foreignPlan.ID) || knowledgeProcessContainsString(foreignView, inventoryManager) || !strings.Contains(foreignText, "selected run is unavailable") {
		t.Fatalf("foreign run did not collapse to a generic unknown in Orders scope: %s", foreignText)
	}
	missingOrdersView := knowledgeProcessCLI(t, bin, root, fixtureHead, ordersManager, "graph", strings.Repeat("f", 32))
	if !bytes.Equal(knowledgeProcessCanonicalJSON(t, foreignView), knowledgeProcessCanonicalJSON(t, missingOrdersView)) {
		t.Fatalf("foreign and missing selected runs did not expose the same generic unknown: foreign=%v", knowledgeProcessUnknownReasons(foreignView))
	}
	foreignMCP, missingMCP := knowledgeProcessMCPUnknownRuns(t, bin, root, fixtureHead, foreignPlan.ID, strings.Repeat("f", 32), ordersManager)
	foreignMCPText := knowledgeProcessJSON(t, foreignMCP)
	if knowledgeProcessContainsString(foreignMCP, foreignPlan.ID) || knowledgeProcessContainsString(foreignMCP, inventoryManager) || !bytes.Equal(knowledgeProcessCanonicalJSON(t, foreignMCP), knowledgeProcessCanonicalJSON(t, missingMCP)) {
		t.Fatalf("MCP foreign run was not generic and indistinguishable from missing: foreign=%s missing=%s", foreignMCPText, knowledgeProcessJSON(t, missingMCP))
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)
	missing := knowledgeProcessCLI(t, bin, root, fixtureHead, "project", "graph", strings.Repeat("f", 32))
	if !strings.Contains(knowledgeProcessJSON(t, missing), "selected run is unavailable") {
		t.Fatalf("missing run selector was not retained as explicit unknown: %s", knowledgeProcessJSON(t, missing))
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)
	plannedGraph := knowledgeProcessCLI(t, bin, root, fixtureHead, "project", "graph", planned.ID)
	plannedGraphData := knowledgeProcessMap(plannedGraph, "graph")
	plannedEdges := knowledgeProcessArray(plannedGraphData, "edges")
	plannedOrdersTask := ""
	var plannedOrders ManagerTask
	for _, task := range planned.Managers {
		if task.ManagerID == ordersManager {
			plannedOrders = task
			plannedOrdersTask = "record/task/" + planned.ID + "/" + task.ID
			break
		}
	}
	plannedTaskPresent := false
	for _, node := range knowledgeProcessArray(plannedGraphData, "nodes") {
		if node["id"] == plannedOrdersTask {
			plannedTaskPresent = true
			props := knowledgeProcessMap(node, "properties")
			wantChecks := append([]string(nil), plannedOrders.Checks...)
			sort.Strings(wantChecks)
			if props["present"] != false || !reflect.DeepEqual(knowledgeProcessStrings(props, "expectedChecks"), wantChecks) {
				t.Fatalf("planned-only task did not retain declared checks without claiming observation: %+v", props)
			}
		}
	}
	if !plannedTaskPresent || !knowledgeProcessHasEdge(plannedEdges, "record/plan/"+planned.ID, "record/run/"+planned.ID, "expectsRun") || !knowledgeProcessHasEdge(plannedEdges, "record/plan/"+planned.ID, plannedOrdersTask, "plansTask") {
		t.Fatalf("planned-only records did not preserve the expected-but-unobserved run/task edges: %+v", plannedGraph)
	}
	if !containsString(knowledgeProcessUnknownReasons(plannedGraph), "selected run state is unavailable") {
		t.Fatalf("planned-only graph did not preserve unknown run state: %+v", knowledgeProcessMap(plannedGraph, "evidence"))
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)
	if _, err := knowledgeProcessCLIResult(bin, root, fixtureHead, "project", "graph", "bad/run"); err == nil {
		t.Fatal("malformed run selector was accepted")
	}

	// Remove one actual persisted verification report temporarily, query the
	// incomplete chain, and restore the exact bytes before the ledger hash check.
	verificationDir := filepath.Join(ledgerPath, "verification")
	verificationEntries, err := os.ReadDir(verificationDir)
	if err != nil || len(verificationEntries) == 0 {
		t.Fatalf("locate stored verification record: entries=%d err=%v", len(verificationEntries), err)
	}
	verificationPath := filepath.Join(verificationDir, verificationEntries[0].Name())
	verificationBytes, err := os.ReadFile(verificationPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(verificationPath); err != nil {
		t.Fatal(err)
	}
	partialGraph := knowledgeProcessCLI(t, bin, root, fixtureHead, "project", "graph", plan.ID)
	partialText := knowledgeProcessJSON(t, partialGraph)
	if !strings.Contains(partialText, `"verificationStatus":"unknown"`) || !strings.Contains(partialText, "selected candidate has no validated verification record") {
		t.Fatalf("missing verification was promoted instead of partial/unknown: %s", partialText)
	}
	if err := os.WriteFile(verificationPath, verificationBytes, 0600); err != nil {
		t.Fatal(err)
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)

	// A corrupt selected record must fail closed through the actual CLI.
	planPath := filepath.Join(ledgerPath, "plan.json")
	originalPlan, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, append(append([]byte(nil), originalPlan...), []byte("!corrupt")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := knowledgeProcessCLIResult(bin, root, fixtureHead, "project", "graph", plan.ID); err == nil {
		t.Fatal("corrupt selected plan was accepted by the knowledge CLI")
	}
	if err := os.WriteFile(planPath, originalPlan, 0600); err != nil {
		t.Fatal(err)
	}
	knowledgeProcessAssertLedger(t, ledgerPath, ledgerBefore)

	// MCP is a separate binary stdio process. Exercise every closed read action,
	// and compare each result with the same CLI request.
	queries := []knowledgeProcessQuery{
		{action: "graph"},
		{action: "relations", target: "record/run/" + plan.ID},
		{action: "explain", target: "record/run/" + plan.ID},
		{action: "trace", target: "record/run/" + plan.ID, bidirectional: true, maxDepth: 4},
		{action: "history", target: knowledgeProcessStatementID(), manager: e2eManagerID("orders", "orders"), briefingHistory: true},
		{action: "coverage"},
	}
	mcpResults := knowledgeProcessMCP(t, bin, root, fixtureHead, plan.ID, queries)
	for i, q := range queries {
		cliArgs := []string{"project", "knowledge", "--repo", root, "--revision", fixtureHead,
			"--knowledge-action", q.action, "--run", plan.ID}
		if q.manager == "" {
			cliArgs = append(cliArgs, "--knowledge-scope", "project")
		} else {
			cliArgs = append(cliArgs, "--manager", q.manager)
		}
		if q.target != "" {
			cliArgs = append(cliArgs, "--node-id", q.target)
		}
		if q.bidirectional {
			cliArgs = append(cliArgs, "--bidirectional")
		}
		if q.maxDepth > 0 {
			cliArgs = append(cliArgs, "--max-depth", fmt.Sprint(q.maxDepth))
		}
		if q.briefingHistory {
			cliArgs = append(cliArgs, "--briefing-history")
		}
		cli, err := knowledgeProcessRunCLI(t, bin, cliArgs...)
		if err != nil {
			t.Fatalf("CLI %s: %v", q.action, err)
		}
		if !bytes.Equal(knowledgeProcessCanonicalJSON(t, cli), knowledgeProcessCanonicalJSON(t, mcpResults[i])) {
			t.Fatalf("CLI/MCP %s semantic result differs", q.action)
		}
	}
	ledgerAfter, err := knowledgeProcessTreeDigest(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if ledgerAfter != ledgerBefore {
		t.Fatalf("read-only CLI/MCP queries changed the selected run ledger: before=%s after=%s", ledgerBefore, ledgerAfter)
	}

	receipt := map[string]any{
		"schema": "markitect-knowledge-process-e2e/v1", "sourceHead": sourceHead,
		"sourceBindingState": map[bool]string{true: "git-bound", false: "vcs-unavailable"}[sourceHead != ""],
		"binarySHA256":       buildInfo.SHA256, "binaryVCSRevision": buildInfo.VCSRevision, "binaryVCSModified": buildInfo.VCSModified,
		"fixtureBaseRevision": baseRevision, "fixtureModelOnlyRevision": modelOnlyRevision,
		"runId": plan.ID, "planDigest": plan.Digest, "candidateId": run.Candidate.ID,
		"candidateDigest": run.Candidate.Snapshot, "verificationDigest": verified.Digest,
		"runDigestAfterApply": persisted.RunDigest, "applyContentDigest": persisted.ApplyContentDigest,
		"baseModelBinding":    knowledgeProcessModelBinding(baseGraph),
		"workingModelBinding": knowledgeProcessModelBinding(currentGraph),
		"runtimeBinding":      knowledgeProcessRunRuntimeBinding(currentGraph),
		"scenarios": []string{
			"complete-stored-chain-cli-mcp-parity", "model-only-stale-binding", "orders-scope-private-target-hidden",
			"foreign-and-missing-run-equivalence", "planned-only-partial-records", "missing-verification-unknown",
			"corrupt-plan-rejected", "ledger-read-only",
		},
		"actions":      []string{"graph", "relations", "explain", "trace", "history", "coverage"},
		"queryDigests": knowledgeProcessQueryDigests(mcpResults), "providerCalls": 0,
		"fakeProcessInvocations": knowledgeProcessInvocationCount(t), "runLedgerDigestBefore": ledgerBefore, "runLedgerDigestAfter": ledgerAfter,
		"outcome": "passed-provider-free-fixture",
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("KNOWLEDGE_PROCESS_E2E_RECEIPT %s", receiptBytes)
	if path := os.Getenv("MARKITECT_KG_E2E_RECEIPT"); path != "" {
		abs, err := filepath.Abs(path)
		repo := knowledgeProcessRepoRoot(t)
		clean := filepath.Clean(abs)
		if err != nil || clean == filepath.Clean(root) || strings.HasPrefix(clean, filepath.Clean(root)+string(os.PathSeparator)) || clean == filepath.Clean(repo) || strings.HasPrefix(clean, filepath.Clean(repo)+string(os.PathSeparator)) {
			t.Fatalf("MARKITECT_KG_E2E_RECEIPT must be outside the source checkout and disposable fixture")
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, append(receiptBytes, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

type knowledgeProcessQuery struct {
	action          string
	target          string
	manager         string
	bidirectional   bool
	maxDepth        int
	briefingHistory bool
}

type knowledgeProcessBinaryInfo struct {
	SHA256      string `json:"sha256"`
	VCSRevision string `json:"vcsRevision,omitempty"`
	VCSModified string `json:"vcsModified,omitempty"`
}

func buildKnowledgeProcessCLI(t *testing.T) (string, knowledgeProcessBinaryInfo) {
	t.Helper()
	repo := knowledgeProcessRepoRoot(t)
	binDir := t.TempDir()
	name := "markitect"
	if os.PathSeparator == '\\' {
		name += ".exe"
	}
	bin := filepath.Join(binDir, name)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/markitect")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build source CLI: %v\n%s", err, output)
	}
	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	info := knowledgeProcessBinaryInfo{SHA256: hex.EncodeToString(sum[:])}
	version, err := exec.Command("go", "version", "-m", bin).CombinedOutput()
	if err != nil {
		t.Fatalf("read CLI build metadata: %v\n%s", err, version)
	}
	for _, line := range strings.Split(string(version), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "build" {
			key, value, ok := strings.Cut(fields[1], "=")
			if !ok {
				continue
			}
			switch key {
			case "vcs.revision":
				info.VCSRevision = value
			case "vcs.modified":
				info.VCSModified = value
			}
		}
	}
	return bin, info
}

func knowledgeProcessCLI(t *testing.T, bin, root, revision, scope, action, runID string) map[string]any {
	t.Helper()
	args := []string{"project", "knowledge", "--repo", root, "--knowledge-action", action, "--run", runID}
	if revision != "" {
		args = append(args, "--revision", revision)
	}
	if scope == "project" {
		args = append(args, "--knowledge-scope", "project")
	} else {
		args = append(args, "--manager", scope)
	}
	result, err := knowledgeProcessRunCLI(t, bin, args...)
	if err != nil {
		t.Fatalf("knowledge CLI %s: %v", action, err)
	}
	return result
}

func knowledgeProcessExecReadOnly(bin, root string, stdin io.Reader, args ...string) ([]byte, error) {
	ledgerRoot := filepath.Join(root, filepath.FromSlash(RunsPath))
	before, err := knowledgeProcessTreeDigest(ledgerRoot)
	if err != nil {
		return nil, fmt.Errorf("hash run records before knowledge query: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = stdin
	output, commandErr := cmd.CombinedOutput()
	after, hashErr := knowledgeProcessTreeDigest(ledgerRoot)
	if hashErr != nil {
		return output, fmt.Errorf("hash run records after knowledge query: %w", hashErr)
	}
	if before != after {
		return output, fmt.Errorf("read-only knowledge query changed persisted run records: before=%s after=%s", before, after)
	}
	if commandErr != nil {
		return output, fmt.Errorf("command exited with %v: %s", commandErr, output)
	}
	return output, nil
}

func knowledgeProcessCLIResult(bin, root, revision, scope, action, runID string) (map[string]any, error) {
	args := []string{"project", "knowledge", "--repo", root, "--knowledge-action", action, "--run", runID}
	if revision != "" {
		args = append(args, "--revision", revision)
	}
	if scope == "project" {
		args = append(args, "--knowledge-scope", "project")
	} else {
		args = append(args, "--manager", scope)
	}
	output, err := knowledgeProcessExecReadOnly(bin, root, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("exit=%v output=%s", err, output)
	}
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("decode JSON: %w output=%s", err, output)
	}
	return result, nil
}

func knowledgeProcessCLIError(t *testing.T, bin, root, revision, manager, runID, target string) string {
	t.Helper()
	args := []string{"project", "knowledge", "--repo", root, "--revision", revision,
		"--manager", manager, "--knowledge-action", "explain", "--node-id", target, "--run", runID}
	output, err := knowledgeProcessExecReadOnly(bin, root, nil, args...)
	if err == nil {
		t.Fatalf("knowledge explain accepted forbidden/absent target %q: %s", target, output)
	}
	return string(bytes.TrimSpace(output))
}

func knowledgeProcessRunCLI(t *testing.T, bin string, args ...string) (map[string]any, error) {
	t.Helper()
	root := ""
	for index := 0; index+1 < len(args); index++ {
		if args[index] == "--repo" {
			root = args[index+1]
			break
		}
	}
	if root == "" {
		return nil, fmt.Errorf("knowledge CLI arguments omitted --repo")
	}
	output, err := knowledgeProcessExecReadOnly(bin, root, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, output)
	}
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("decode JSON: %w output=%s", err, output)
	}
	return result, nil
}

func knowledgeProcessMCP(t *testing.T, bin, root, revision, runID string, queries []knowledgeProcessQuery) []map[string]any {
	t.Helper()
	lines := []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "knowledge-e2e", "version": "1"}}},
		map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"},
		map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"},
	}
	toolNames := map[string]string{"graph": "knowledge_graph", "relations": "knowledge_relations", "explain": "knowledge_explain", "trace": "knowledge_trace", "history": "knowledge_history", "coverage": "knowledge_coverage"}
	for i, q := range queries {
		args := map[string]any{"runId": runID}
		if q.manager == "" {
			args["projectScope"] = true
		} else {
			args["managerId"] = q.manager
		}
		if q.target != "" {
			args["targetId"] = q.target
		}
		if q.bidirectional {
			args["bidirectional"] = true
		}
		if q.maxDepth > 0 {
			args["maxDepth"] = q.maxDepth
		}
		if q.briefingHistory {
			args["briefingHistory"] = true
		}
		lines = append(lines, map[string]any{"jsonrpc": "2.0", "id": i + 3, "method": "tools/call", "params": map[string]any{"name": toolNames[q.action], "arguments": args}})
	}
	var input strings.Builder
	for _, line := range lines {
		data, err := json.Marshal(line)
		if err != nil {
			t.Fatal(err)
		}
		input.Write(data)
		input.WriteByte('\n')
	}
	args := []string{"project", "knowledge-mcp", "--repo", root}
	if revision != "" {
		args = append(args, "--revision", revision)
	}
	output, err := knowledgeProcessExecReadOnly(bin, root, strings.NewReader(input.String()), args...)
	if err != nil {
		t.Fatalf("knowledge MCP stdio process: %v\n%s", err, output)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	responses := []map[string]any{}
	for scanner.Scan() {
		var response map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			t.Fatalf("decode MCP response: %v line=%s", err, scanner.Text())
		}
		responses = append(responses, response)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(responses) != len(queries)+2 {
		t.Fatalf("MCP response count=%d want=%d: %s", len(responses), len(queries)+2, output)
	}
	results := make([]map[string]any, 0, len(queries))
	for i := range queries {
		var envelope struct {
			Result struct {
				IsError bool `json:"isError"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				Structured map[string]any `json:"structuredContent"`
			} `json:"result"`
		}
		data, _ := json.Marshal(responses[i+2])
		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Result.IsError || envelope.Result.Structured == nil {
			t.Fatalf("MCP %s returned error: %s", queries[i].action, data)
		}
		results = append(results, envelope.Result.Structured)
	}
	return results
}

func knowledgeProcessMCPDenials(t *testing.T, bin, root, revision, runID, manager, privateTarget, missingTarget string) (string, string) {
	t.Helper()
	messages := []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "knowledge-denial-e2e", "version": "1"}}},
		map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"},
	}
	for i, target := range []string{privateTarget, missingTarget} {
		messages = append(messages, map[string]any{"jsonrpc": "2.0", "id": i + 2, "method": "tools/call",
			"params": map[string]any{"name": "knowledge_explain", "arguments": map[string]any{"managerId": manager, "targetId": target, "runId": runID}}})
	}
	var input strings.Builder
	for _, message := range messages {
		data, err := json.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}
		input.Write(data)
		input.WriteByte('\n')
	}
	output, err := knowledgeProcessExecReadOnly(bin, root, strings.NewReader(input.String()), "project", "knowledge-mcp", "--repo", root, "--revision", revision)
	if err != nil {
		t.Fatalf("MCP denial process: %v\n%s", err, output)
	}
	var responses []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var response map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			t.Fatalf("decode MCP denial response: %v line=%s", err, scanner.Text())
		}
		responses = append(responses, response)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(responses) != 3 {
		t.Fatalf("MCP denial response count=%d want=3: %s", len(responses), output)
	}
	errors := make([]string, 0, 2)
	for _, response := range responses[1:] {
		payload, _ := json.Marshal(response)
		var envelope struct {
			Result struct {
				IsError bool `json:"isError"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				Structured json.RawMessage `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatal(err)
		}
		if !envelope.Result.IsError || len(envelope.Result.Structured) != 0 || len(envelope.Result.Content) != 1 {
			t.Fatalf("MCP denied target did not return a generic error: %s", payload)
		}
		errors = append(errors, envelope.Result.Content[0].Text)
	}
	return errors[0], errors[1]
}

func knowledgeProcessMCPUnknownRuns(t *testing.T, bin, root, revision, foreignRun, missingRun, manager string) (map[string]any, map[string]any) {
	t.Helper()
	messages := []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "knowledge-unknown-e2e", "version": "1"}}},
		map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"},
	}
	for i, runID := range []string{foreignRun, missingRun} {
		messages = append(messages, map[string]any{"jsonrpc": "2.0", "id": i + 2, "method": "tools/call",
			"params": map[string]any{"name": "knowledge_graph", "arguments": map[string]any{"managerId": manager, "runId": runID}}})
	}
	var input strings.Builder
	for _, message := range messages {
		data, err := json.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}
		input.Write(data)
		input.WriteByte('\n')
	}
	args := []string{"project", "knowledge-mcp", "--repo", root, "--revision", revision}
	output, err := knowledgeProcessExecReadOnly(bin, root, strings.NewReader(input.String()), args...)
	if err != nil {
		t.Fatalf("MCP foreign/missing run process: %v\n%s", err, output)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	responses := []map[string]any{}
	for scanner.Scan() {
		var response map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			t.Fatalf("decode MCP unknown-run response: %v line=%s", err, scanner.Text())
		}
		responses = append(responses, response)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(responses) != 3 {
		t.Fatalf("MCP unknown-run response count=%d want=3: %s", len(responses), output)
	}
	results := make([]map[string]any, 0, 2)
	for _, response := range responses[1:] {
		data, _ := json.Marshal(response)
		var envelope struct {
			Result struct {
				IsError    bool           `json:"isError"`
				Structured map[string]any `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Result.IsError || envelope.Result.Structured == nil {
			t.Fatalf("MCP unknown-run capture failed: %s", data)
		}
		results = append(results, envelope.Result.Structured)
	}
	return results[0], results[1]
}

func assertKnowledgeProcessCompleteGraph(t *testing.T, result map[string]any, run RunReport, plan PlanRecord, verify VerifyReport, apply ApplyReport, persisted KnowledgeRecordSet, wantModelBinding string) {
	t.Helper()
	graph := knowledgeProcessMap(result, "graph")
	nodes := knowledgeProcessArray(graph, "nodes")
	edges := knowledgeProcessArray(graph, "edges")
	byID := map[string]map[string]any{}
	for _, node := range nodes {
		id, _ := node["id"].(string)
		byID[id] = node
	}
	runID := "record/run/" + run.ID
	planID := "record/plan/" + plan.ID
	candidateID := "record/candidate/" + run.ID + "/" + run.Candidate.ID
	verificationID := "record/verification/" + run.ID + "/selected"
	applyID := "record/apply/" + run.ID + "/selected"
	for _, id := range []string{runID, planID, candidateID, verificationID, applyID} {
		if byID[id] == nil {
			t.Fatalf("complete graph omitted %s", id)
		}
	}
	candidateProps := knowledgeProcessMap(byID[candidateID], "properties")
	if candidateProps["digest"] != persisted.CandidateHash || candidateProps["id"] != persisted.CandidateID {
		t.Fatalf("candidate graph node does not match persisted candidate digest/ID: %+v", candidateProps)
	}
	for _, check := range verify.Checks {
		id := "record/check/" + run.ID + "/" + check.ID
		props := knowledgeProcessMap(byID[id], "properties")
		if props["candidateId"] != run.Candidate.ID || props["checked"] != true || props["checkPassed"] != true || props["verificationStatus"] != "passed" {
			t.Fatalf("check did not bind to final candidate and Verify result: %s %+v", id, props)
		}
	}
	checks, reviews := 0, 0
	for id, node := range byID {
		kind, _ := node["kind"].(string)
		if strings.HasPrefix(id, "record/check/"+run.ID+"/") && kind == "CheckExecution" {
			checks++
		}
		if strings.HasPrefix(id, "record/review/"+run.ID+"/") && kind == "CandidateReview" {
			reviews++
		}
	}
	if checks != len(verify.Checks) || checks != len(plan.Checks) {
		t.Fatalf("graph check executions=%d plan=%d verification=%d", checks, len(plan.Checks), len(verify.Checks))
	}
	if reviews != len(run.Reviews) || reviews == 0 {
		t.Fatalf("graph review records=%d run review records=%d", reviews, len(run.Reviews))
	}
	for _, relation := range []struct{ from, to, property string }{
		{runID, planID, "usesPlan"}, {runID, candidateID, "producedCandidate"},
		{candidateID, verificationID, "hasVerification"}, {runID, applyID, "hasApplyReceipt"},
		{candidateID, applyID, "appliedBy"},
	} {
		if !knowledgeProcessHasEdge(edges, relation.from, relation.to, relation.property) {
			t.Fatalf("graph omitted witness edge %+v", relation)
		}
	}
	for _, node := range nodes {
		id, _ := node["id"].(string)
		if strings.HasPrefix(id, "record/check/"+run.ID+"/") {
			if !knowledgeProcessHasEdge(edges, id, candidateID, "checkedCandidate") {
				t.Fatalf("check %s has no candidate witness", id)
			}
		}
		if strings.HasPrefix(id, "record/review/"+run.ID+"/") {
			// Earlier work reviews may bind a predecessor candidate. The exact
			// final candidate assertion is made against the root integration review.
		}
	}
	rootManager := e2eManagerID("", "project-owner")
	rootFinalReview := false
	for _, review := range run.Reviews {
		if review.ManagerID != rootManager || review.CandidateID != run.Candidate.ID {
			continue
		}
		reviewID := fmt.Sprintf("record/review/%s/%s/%s/%d", run.ID, review.TaskID, review.Phase, review.Round)
		if byID[reviewID] != nil && knowledgeProcessHasEdge(edges, reviewID, candidateID, "reviewedCandidate") {
			props := knowledgeProcessMap(byID[reviewID], "properties")
			if props["candidateId"] != run.Candidate.ID || props["candidateDigest"] != run.Candidate.Snapshot || review.CandidateDigest != run.Candidate.Snapshot {
				t.Fatalf("root final review is bound to another candidate: review=%+v graph=%+v final=%s", review, props, run.Candidate.Snapshot)
			}
			rootFinalReview = true
		}
	}
	if !rootFinalReview {
		t.Fatalf("root integration review did not witness the final candidate %s", candidateID)
	}
	planProps := knowledgeProcessMap(byID[planID], "properties")
	if planProps["modelBindingStatus"] != wantModelBinding {
		t.Fatalf("selected plan binding=%v, want %s", planProps["modelBindingStatus"], wantModelBinding)
	}
	runProps := knowledgeProcessMap(byID[runID], "properties")
	if runProps["modelBindingStatus"] != wantModelBinding {
		t.Fatalf("selected run binding=%v, want %s", runProps["modelBindingStatus"], wantModelBinding)
	}
	if runProps["runtimeBindingStatus"] != "not-compared" || runProps["captureState"] != "consistent" {
		t.Fatalf("runtime/capture evidence was overstated: %+v", runProps)
	}
	verifyProps := knowledgeProcessMap(byID[verificationID], "properties")
	if verifyProps["verified"] != true || verifyProps["verificationStatus"] != "verified" {
		t.Fatalf("verification node is not verified: %+v", verifyProps)
	}
	applyProps := knowledgeProcessMap(byID[applyID], "properties")
	if applyProps["applied"] != true || applyProps["authenticated"] != false || applyProps["recordedStatus"] != StatusApplied || applyProps["candidateId"] != persisted.CandidateID || applyProps["contentDigest"] != persisted.ApplyContentDigest {
		t.Fatalf("Apply receipt overstated or incomplete: %+v", applyProps)
	}
	if !reflect.DeepEqual(knowledgeProcessStrings(applyProps, "writtenPaths"), apply.Written) {
		wantPaths := append([]string(nil), apply.Written...)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(knowledgeProcessStrings(applyProps, "writtenPaths"), wantPaths) {
			t.Fatalf("Apply written paths differ from persisted receipt: graph=%v receipt=%v", knowledgeProcessStrings(applyProps, "writtenPaths"), wantPaths)
		}
	}
	if apply.Status != StatusApplied || run.Candidate.Snapshot == "" || verify.Digest == "" {
		t.Fatal("fixture did not reach a complete validated lifecycle")
	}
	evidence := knowledgeProcessMap(result, "evidence")
	bindings := knowledgeProcessArray(evidence, "sourceBindings")
	wantBindings := map[string]struct {
		digest, recordID, schema, model, revision, runtime string
	}{
		"plan":         {plan.Digest, plan.ID, plan.APIVersion, plan.BaseModelDigest, plan.BaseRevision, persisted.RuntimeDigest},
		"run":          {persisted.RunDigest, run.ID, run.APIVersion, run.ModelDigest, run.BaseRevision, persisted.RuntimeDigest},
		"candidate":    {persisted.CandidateHash, run.ID, persisted.CandidateAPIVersion, run.ModelDigest, run.BaseRevision, persisted.RuntimeDigest},
		"verification": {persisted.VerificationDigest, run.ID, persisted.VerificationAPIVersion, run.ModelDigest, run.BaseRevision, persisted.RuntimeDigest},
		"apply":        {persisted.ApplyContentDigest, run.ID, persisted.ApplyAPIVersion, run.ModelDigest, run.BaseRevision, persisted.RuntimeDigest},
	}
	for kind, digest := range wantBindings {
		found := false
		for _, binding := range bindings {
			if binding["kind"] == kind && binding["digest"] == digest.digest && binding["recordId"] == digest.recordID && binding["schema"] == digest.schema && binding["modelDigest"] == digest.model && binding["revision"] == digest.revision && binding["runtimeDigest"] == digest.runtime && binding["source"] == "live-operational-record" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("source binding %s=%+v is absent or mismatched in evidence summary", kind, digest)
		}
	}
	applyDigest := knowledgeProcessString(applyProps, "contentDigest")
	if applyDigest != persisted.ApplyContentDigest || knowledgeProcessString(applyProps, "digestKind") != "derived-read-fence-only" {
		t.Fatalf("Apply content digest is not explicitly a read-fence digest: %+v", applyProps)
	}
}

func knowledgeProcessHasEdge(edges []map[string]any, from, to, property string) bool {
	for _, edge := range edges {
		if edge["from"] == from && edge["to"] == to && edge["property"] == property {
			return true
		}
	}
	return false
}

func knowledgeProcessMap(value map[string]any, key string) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	result, _ := value[key].(map[string]any)
	if result == nil {
		return map[string]any{}
	}
	return result
}

func knowledgeProcessArray(value map[string]any, key string) []map[string]any {
	items, _ := value[key].([]any)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if typed, ok := item.(map[string]any); ok {
			result = append(result, typed)
		}
	}
	return result
}

func knowledgeProcessStrings(value map[string]any, key string) []string {
	items, _ := value[key].([]any)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if typed, ok := item.(string); ok {
			result = append(result, typed)
		}
	}
	sort.Strings(result)
	return result
}

func knowledgeProcessString(value map[string]any, key string) string {
	text, _ := value[key].(string)
	return text
}

func knowledgeProcessJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func knowledgeProcessCanonicalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func knowledgeProcessContainsString(value any, needle string) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(typed, needle)
	case []any:
		for _, item := range typed {
			if knowledgeProcessContainsString(item, needle) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if knowledgeProcessContainsString(item, needle) {
				return true
			}
		}
	}
	return false
}

func knowledgeProcessStatementID() string {
	data, _ := json.Marshal([]string{projectModelAPIVersionForKnowledge(), "Statement", "orders", "orders-work"})
	return string(data)
}

func knowledgeProcessModelBinding(result map[string]any) string {
	graph := knowledgeProcessMap(result, "graph")
	for _, node := range knowledgeProcessArray(graph, "nodes") {
		if node["kind"] == "ExecutionPlan" {
			return knowledgeProcessString(knowledgeProcessMap(node, "properties"), "modelBindingStatus")
		}
	}
	return ""
}

func knowledgeProcessRunRuntimeBinding(result map[string]any) string {
	graph := knowledgeProcessMap(result, "graph")
	for _, node := range knowledgeProcessArray(graph, "nodes") {
		if node["kind"] == "ExecutionRun" {
			return knowledgeProcessString(knowledgeProcessMap(node, "properties"), "runtimeBindingStatus")
		}
	}
	return ""
}

func knowledgeProcessUnknownReasons(result map[string]any) []string {
	evidence := knowledgeProcessMap(result, "evidence")
	unknowns := knowledgeProcessArray(evidence, "unknown")
	reasons := make([]string, 0, len(unknowns))
	for _, unknown := range unknowns {
		reason, _ := unknown["reason"].(string)
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	return reasons
}

func knowledgeProcessAssertLedger(t *testing.T, path, expected string) {
	t.Helper()
	actual, err := knowledgeProcessTreeDigest(path)
	if err != nil || actual != expected {
		t.Fatalf("knowledge query changed run ledger: got=%s err=%v want=%s", actual, err, expected)
	}
}

func projectModelAPIVersionForKnowledge() string { return "project.markitect.example.org/v1alpha1" }

func knowledgeProcessTreeDigest(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("ledger contains unexpected symlink %s", path)
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(hash, "%s\x00", filepath.ToSlash(rel))
		_, _ = hash.Write(data)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func knowledgeProcessRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller did not return source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func knowledgeProcessSourceHead(repo string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = repo
	data, err := cmd.Output()
	if err != nil {
		return ""
	}
	top, err := filepath.Abs(strings.TrimSpace(string(data)))
	if err != nil {
		return ""
	}
	expected, err := filepath.Abs(repo)
	if err != nil || !strings.EqualFold(filepath.Clean(top), filepath.Clean(expected)) {
		return ""
	}
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repo
	data, err = cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func TestKnowledgeProcessSourceHeadRequiresExactGitRoot(t *testing.T) {
	noGitRoot := t.TempDir()
	if got := knowledgeProcessSourceHead(noGitRoot); got != "" {
		t.Fatalf("source metadata claimed an unrelated or unavailable Git HEAD: %s", got)
	}
	gitRoot := t.TempDir()
	gitE2E(t, gitRoot, "init", "-b", "codex/source-head-boundary")
	nestedSource := filepath.Join(gitRoot, "materialized-source")
	if err := os.Mkdir(nestedSource, 0700); err != nil {
		t.Fatal(err)
	}
	if got := knowledgeProcessSourceHead(nestedSource); got != "" {
		t.Fatalf("materialized subdirectory claimed its ancestor repository HEAD: %s", got)
	}
}

func knowledgeProcessAddRootReviewFixture(t *testing.T, root string) {
	t.Helper()
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "  - .markitect/model/manager.yaml\n", "  - .markitect/model/manager.yaml\n  - .markitect/model/root-statement.yaml\n", 1)
	if updated == string(manifest) {
		t.Fatal("project fixture did not contain root manager model-file anchor")
	}
	writeE2E(t, root, projectwork.ManifestPath, updated)
	writeE2E(t, root, ".markitect/model/root-statement.yaml", "apiVersion: "+projectModelAPIVersionForKnowledge()+"\nkind: Statement\nmetadata:\n  name: integration-quality\n  namespace: \"\"\npurpose: Parent integration must produce a clean summary.\nspec:\n  category: concept\n  description: Parent-owned integration files contain the corrected summary.\n")
	gitE2E(t, root, "add", projectwork.ManifestPath, ".markitect/model/root-statement.yaml")
	gitE2E(t, root, "commit", "--amend", "--no-edit")
}

func knowledgeProcessQueryDigests(results []map[string]any) []string {
	digests := make([]string, 0, len(results))
	for _, result := range results {
		graph := knowledgeProcessMap(result, "graph")
		digests = append(digests, knowledgeProcessString(graph, "queryDigest"))
	}
	return digests
}

func knowledgeProcessInvocationCount(t *testing.T) int {
	t.Helper()
	path := os.Getenv(e2eLogEnv)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
		if len(line) != 0 {
			count++
		}
	}
	return count
}
