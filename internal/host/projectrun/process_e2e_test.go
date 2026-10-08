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
		ChildReports []struct {
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
		"directChildren": contextPayload.DirectChildren, "childReports": contextPayload.ChildReports,
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
		Delegations: []Delegation{}, Integrated: contextPayload.Phase == "integrate", Questions: []string{}, Risks: []string{},
		ResolvedQuestions: []string{}, ResolvedRisks: []string{}, EscalateTo: ""}
	files := []agentexec.CandidateFile{}
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
		}
	case "integrate":
		if len(contextPayload.ChildReports) != 2 {
			processExit(2, fmt.Sprintf("root integration saw %d child reports, want 2", len(contextPayload.ChildReports)))
		}
	default:
		processExit(2, "unknown phase "+contextPayload.Phase)
	}
	reportJSON, err := json.Marshal(response)
	if err != nil {
		processExit(2, "encode task report: "+err.Error())
	}
	inputTokens, outputTokens := int64(41), int64(13)
	result := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
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
			MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20, Environment: []string{"PATH", e2eExecutorEnv, e2eCheckEnv, e2eLogEnv},
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
