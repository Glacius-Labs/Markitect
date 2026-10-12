package projectapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projecthistory"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	projectbriefing "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
	"go.yaml.in/yaml/v3"
)

const deliveryRuntimeFixtureArg = "-markitect-delivery-runtime-fixture="
const deliveryWorkerEnv = "MARKITECT_DELIVERY_WORKER"
const deliveryWorkerLogEnv = "MARKITECT_DELIVERY_WORKER_LOG"
const deliveryFullVerifyCheckEnv = "MARKITECT_FULL_VERIFY_CHECK"
const deliveryCandidateCheckEnv = "MARKITECT_DELIVERY_CANDIDATE_CHECK"
const deliveryFullVerifyBadEnv = "MARKITECT_FULL_VERIFY_BAD_MANAGER"
const deliveryFullVerifyFailEnv = "MARKITECT_FULL_VERIFY_FAIL_MANAGER"
const deliveryFullVerifyNoUsageEnv = "MARKITECT_FULL_VERIFY_NO_USAGE"
const deliveryFullVerifyLogEnv = "MARKITECT_FULL_VERIFY_LOG"

type deliveryProcessFixture struct {
	path string
	env  map[string]string
}

var deliveryProcessFixtures sync.Map

// init loads per-fixture child environment before testing parses flags. Each
// process descriptor belongs to one test, so parallel tests never call Setenv.
func init() {
	args := os.Args[:1]
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, deliveryRuntimeFixtureArg) {
			args = append(args, arg)
			continue
		}
		data, err := os.ReadFile(strings.TrimPrefix(arg, deliveryRuntimeFixtureArg))
		if err != nil {
			fmt.Fprintln(os.Stderr, "read delivery process fixture:", err)
			os.Exit(2)
		}
		var env map[string]string
		if err := json.Unmarshal(data, &env); err != nil {
			fmt.Fprintln(os.Stderr, "decode delivery process fixture:", err)
			os.Exit(2)
		}
		for key, value := range env {
			if err := os.Setenv(key, value); err != nil {
				fmt.Fprintln(os.Stderr, "set delivery process fixture:", err)
				os.Exit(2)
			}
		}
	}
	// Check executable pins resolve the bare test-binary name through PATH.
	// Match the Runtime process fixture while keeping one package-level value,
	// so parallel App tests never mutate their parent's environment.
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve delivery test executable:", err)
		os.Exit(2)
	}
	path := filepath.Dir(executable) + string(os.PathListSeparator) + os.Getenv("PATH")
	if err := os.Setenv("PATH", path); err != nil {
		fmt.Fprintln(os.Stderr, "extend delivery test PATH:", err)
		os.Exit(2)
	}
	os.Args = args
}

func newDeliveryProcessFixture(t *testing.T) *deliveryProcessFixture {
	t.Helper()
	if value, ok := deliveryProcessFixtures.Load(t); ok {
		return value.(*deliveryProcessFixture)
	}
	dir := testkit.TempDir(t)
	fixture := &deliveryProcessFixture{path: filepath.Join(dir, "process.json"), env: map[string]string{
		deliveryWorkerEnv: "1", deliveryWorkerLogEnv: "", deliveryFullVerifyLogEnv: "", deliveryFullVerifyCheckEnv: "", deliveryCandidateCheckEnv: "", deliveryFullVerifyBadEnv: "", deliveryFullVerifyFailEnv: "", deliveryFullVerifyNoUsageEnv: "",
	}}
	deliveryProcessFixtures.Store(t, fixture)
	t.Cleanup(func() { deliveryProcessFixtures.Delete(t) })
	writeDeliveryProcessFixture(t, fixture)
	return fixture
}

func writeDeliveryProcessFixture(t *testing.T, fixture *deliveryProcessFixture) {
	t.Helper()
	data, err := json.Marshal(fixture.env)
	if err != nil {
		t.Fatal(err)
	}
	tmp := fixture.path + ".new"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, fixture.path); err != nil {
		t.Fatal(err)
	}
}

func setDeliveryProcessFixtureEnv(t *testing.T, key, value string) {
	t.Helper()
	fixture := newDeliveryProcessFixture(t)
	fixture.env[key] = value
	writeDeliveryProcessFixture(t, fixture)
}

func deliveryProcessArgs(t *testing.T, args []string) []string {
	t.Helper()
	result := append([]string(nil), args...)
	for _, arg := range result {
		if strings.HasPrefix(arg, deliveryRuntimeFixtureArg) {
			return result
		}
	}
	return append(result, deliveryRuntimeFixtureArg+newDeliveryProcessFixture(t).path)
}

// deliveryFixture is shared with other projectapp boundary tests. It is a
// committed repository acquired by the real frontend, with three Managers,
// full coverage, a generated document, and process-backed checks and agents.
func deliveryFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, "README.md", "Delivery integration fixture\n")
	deliveryWrite(t, root, projectvalue.ManifestPath, "apiVersion: "+projectvalue.APIVersion+"\nname: Delivery fixture\nworkflowMode: guided\ndocumentPath: README.md\ncoverageMode: full\nmodelFiles:\n  - .markitect/model/manager.yaml\n  - .markitect/model/orders/manager.yaml\n  - .markitect/model/orders/statement.yaml\n  - .markitect/model/orders/artifact.yaml\n  - .markitect/model/orders/check.yaml\n  - .markitect/model/inventory/manager.yaml\n  - .markitect/model/inventory/statement.yaml\n  - .markitect/model/inventory/artifact.yaml\n  - .markitect/model/inventory/check.yaml\ninventoryRoots:\n  - src\nexclusions: []\n")
	deliveryWrite(t, root, ".markitect/model/manager.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Manager\nmetadata:\n  name: project-owner\n  namespace: \"\"\npurpose: Owns project-wide integration.\nspec:\n  owns: [.]\n")
	deliveryWrite(t, root, ".markitect/model/orders/manager.yaml", deliveryChildManager("orders"))
	deliveryWrite(t, root, ".markitect/model/orders/statement.yaml", deliveryStatement("orders", "orders-work", "Implement the orders artifact."))
	deliveryWrite(t, root, ".markitect/model/orders/artifact.yaml", deliveryArtifact("orders", "orders-code", "orders-work", "orders-check", "src/orders/implementation.txt"))
	deliveryWrite(t, root, ".markitect/model/orders/check.yaml", deliveryCheck(t, "orders", "orders-check", "orders-work", executable))
	deliveryWrite(t, root, ".markitect/model/inventory/manager.yaml", deliveryChildManager("inventory"))
	deliveryWrite(t, root, ".markitect/model/inventory/statement.yaml", deliveryStatement("inventory", "inventory-work", "Implement the inventory artifact."))
	deliveryWrite(t, root, ".markitect/model/inventory/artifact.yaml", deliveryArtifact("inventory", "inventory-code", "inventory-work", "inventory-check", "src/inventory/implementation.txt"))
	deliveryWrite(t, root, ".markitect/model/inventory/check.yaml", deliveryCheck(t, "inventory", "inventory-check", "inventory-work", executable))
	deliveryWrite(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
	deliveryWrite(t, root, "src/inventory/implementation.txt", "inventory implementation v1\n")
	deliveryWrite(t, root, projectcoverage.IgnorePath, "apiVersion: "+projectcoverage.IgnoreAPIVersion+"\nkind: RepositoryIgnore\nentries: []\n")
	args := deliveryProcessArgs(t, []string{"-test.run=^TestDeliveryProcessWorker$"})
	buildAgent := func() projectconfig.Agent {
		return projectconfig.Agent{Command: executable, Args: args, Model: "fixture-model", ProviderVersion: "delivery-fixture/1", Timeout: projectconfig.Duration(30 * time.Second), MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20, Environment: []string{"PATH", deliveryWorkerEnv, deliveryWorkerLogEnv, deliveryFullVerifyLogEnv, deliveryFullVerifyCheckEnv, deliveryCandidateCheckEnv, deliveryFullVerifyBadEnv, deliveryFullVerifyFailEnv, deliveryFullVerifyNoUsageEnv}, Pricing: projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}}
	}
	agents := map[string]projectconfig.Agent{
		deliveryManagerID("", "project-owner"): buildAgent(), deliveryManagerID("orders", "orders"): buildAgent(), deliveryManagerID("inventory", "inventory"): buildAgent(),
	}
	reviewAgents := map[string]projectconfig.Agent{}
	for id, agent := range agents {
		reviewAgents[id] = agent
	}
	config := projectconfig.Runtime{APIVersion: projectconfig.APIVersion, Mode: projectconfig.ModeControlledLocal, Agents: agents,
		Review: &projectconfig.ReviewConfig{Agents: reviewAgents, MaxRounds: 2, MaxManagerRounds: 2},
		Limits: projectconfig.Limits{MaxDepth: 4, MaxStarts: 48, MaxRetries: 0, MaxParallel: 1, MaxDuration: projectconfig.Duration(6 * time.Minute), MaxCostMicros: 100000, MaxCandidateFileBytes: 4096, MaxCandidateBytes: 16384}}
	encoded, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, projectconfig.RuntimePath, string(encoded))
	deliveryGit(t, root, "init", "-b", "dev/delivery-fixture")
	deliveryGit(t, root, "config", "user.email", "delivery@example.test")
	deliveryGit(t, root, "config", "user.name", "Delivery fixture")
	deliveryGit(t, root, "add", ".")
	deliveryGit(t, root, "commit", "-m", "create delivery fixture")
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	document, err := projectwork.Document(project, false)
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, projectvalue.DocumentPath(project.Config), document)
	deliveryGit(t, root, "add", projectvalue.DocumentPath(project.Config))
	deliveryGit(t, root, "commit", "-m", "generate project document")
	return root
}

func deliveryManagerID(namespace, name string) string {
	identity := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Namespace: namespace, Name: name}
	return identity.Key()
}

func deliveryManagers() []string {
	managers := []string{deliveryManagerID("orders", "orders"), deliveryManagerID("inventory", "inventory")}
	sort.Strings(managers)
	return managers
}

// deliveryOperations preserves the real Application composition while
// selecting the concrete subprocess transport used by the provider-free
// protocol fixture.
func deliveryOperations() Operations {
	operations := NewOperations()
	operations.Invoker = projectexecutor.ProcessInvoker{}
	return operations
}

// TestDeliveryProcessWorker is a provider-free subprocess protocol worker.
func TestDeliveryProcessWorker(t *testing.T) {
	if os.Getenv(deliveryWorkerEnv) != "1" {
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
		ManagerID        string            `json:"managerId"`
		RequiredSubjects []string          `json:"requiredSubjects"`
		ChildAssessments []json.RawMessage `json:"childAssessments"`
		CheckResults     []json.RawMessage `json:"checkResults"`
		CandidateDigest  string            `json:"candidateDigest"`
		ResponseSchema   json.RawMessage   `json:"responseSchema"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &request); err != nil {
		t.Fatal(err)
	}
	managerID := request.Manager.Manager.ID
	if request.Kind == "projectrun-review/v1" {
		managerID = request.ManagerID
	}
	logPath := os.Getenv(deliveryWorkerLogEnv)
	if request.Kind == "projectrun-full-verify/v1" && os.Getenv(deliveryFullVerifyLogEnv) != "" {
		logPath = os.Getenv(deliveryFullVerifyLogEnv)
	}
	if logPath != "" {
		log, openErr := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if openErr != nil {
			t.Fatal(openErr)
		}
		entry := map[string]any{"kind": request.Kind, "phase": request.Phase, "managerId": managerID, "context": json.RawMessage(invocation.Request.Context)}
		if request.Kind == "projectrun-full-verify/v1" {
			entry = map[string]any{"managerId": managerID, "context": json.RawMessage(invocation.Request.Context)}
		}
		if encodeErr := json.NewEncoder(log).Encode(entry); encodeErr != nil {
			_ = log.Close()
			t.Fatal(encodeErr)
		}
		if closeErr := log.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	var report any
	candidates := []agentexec.CandidateFile{}
	if request.Kind == "projectrun-review/v1" {
		if managerID == "" || request.CandidateDigest == "" || len(request.ResponseSchema) == 0 {
			t.Fatalf("review request omitted its identity, candidate binding, or schema: %s", invocation.Request.Context)
		}
		report = map[string]any{"status": "pass", "summary": "scoped candidate satisfies its accepted model", "findings": []any{}}
	} else if request.Kind == "projectrun-full-verify/v1" {
		if managerID == "" {
			t.Fatalf("full verification context omitted manager: %s", invocation.Request.Context)
		}
		assessments := make([]map[string]string, 0, len(request.RequiredSubjects))
		for _, subject := range request.RequiredSubjects {
			assessments = append(assessments, map[string]string{"subject": subject, "outcome": "pass", "detail": "bound project snapshot inspected"})
		}
		status := "pass"
		if bad := os.Getenv(deliveryFullVerifyBadEnv); bad != "" && strings.Contains(managerID, bad) {
			status, assessments = "incomplete", nil
		}
		if failed := os.Getenv(deliveryFullVerifyFailEnv); failed != "" && strings.Contains(managerID, failed) {
			status = "fail"
		}
		report = map[string]any{"status": status, "summary": "completed bounded Manager audit", "assessments": assessments, "findings": []string{}, "counterexamples": []any{}}
	} else {
		delegations := []map[string]string{}
		status := "complete"
		summary := managerID + " completed " + request.Phase
		if request.Phase == "work" && managerID == deliveryManagerID("", "project-owner") {
			for _, id := range request.DirectChildren {
				delegations = append(delegations, map[string]string{"managerId": id, "goal": "Implement the owned source artifact and report its evidence."})
			}
		}
		if request.Phase == "work" && managerID == deliveryManagerID("orders", "orders") {
			candidates = []agentexec.CandidateFile{{Path: "src/orders/implementation.txt", Mode: "0644", Content: "orders implementation v2\n"}}
		}
		if request.Phase == "work" && managerID == deliveryManagerID("inventory", "inventory") {
			candidates = []agentexec.CandidateFile{{Path: "src/inventory/implementation.txt", Mode: "0644", Content: "inventory implementation v2\n"}}
		}
		report = map[string]any{"status": status, "summary": summary, "delegations": delegations, "reworkRequests": []any{}, "integrated": request.Phase == "integrate", "questions": []any{}, "risks": []any{}, "resolvedQuestions": []any{}, "resolvedRisks": []any{}, "escalateTo": ""}
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	inTokens, outTokens := int64(41), int64(13)
	if request.Kind == "projectrun-review/v1" {
		inTokens, outTokens = 31, 7
	} else if request.Kind == "projectrun-full-verify/v1" {
		inTokens, outTokens = 10, 5
	}
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: &inTokens, OutputTokens: &outTokens}
	if noUsage := os.Getenv(deliveryFullVerifyNoUsageEnv); request.Kind == "projectrun-full-verify/v1" && noUsage != "" && strings.Contains(managerID, noUsage) {
		usage = nil
	}
	response, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce, Role: invocation.Request.Role, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed, CandidateFiles: candidates, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, ReportJSON: reportJSON, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(response, '\n'))
	os.Exit(0)
}

func TestDeliveryCheckWorker(t *testing.T) {
	if os.Getenv(deliveryCandidateCheckEnv) == "1" {
		for _, item := range []struct{ path, expected string }{
			{"src/orders/implementation.txt", "orders implementation v2\n"},
			{"src/inventory/implementation.txt", "inventory implementation v2\n"},
		} {
			data, err := os.ReadFile(item.path)
			if err != nil || string(data) != item.expected {
				t.Fatalf("snapshot check failed for %s: content=%q error=%v", item.path, data, err)
			}
		}
	} else if os.Getenv(deliveryFullVerifyCheckEnv) == "1" {
		for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("full verification check could not find %s: %v", path, err)
			}
		}
	} else {
		return
	}
	if wd, err := os.Getwd(); err == nil {
		fmt.Println("check cwd: " + wd)
	}
}

func TestDeliverResumesIntegratedRunAndCompletesAcknowledgedScope(t *testing.T) {
	t.Parallel()
	setDeliveryProcessFixtureEnv(t, deliveryCandidateCheckEnv, "1")
	root := deliveryFixture(t)
	deliverySetMaxDuration(t, root, 15*time.Minute)
	operations := deliveryOperations()
	event := deliveryAcceptAndDismissFirstEvent(t, root)
	deliveryAssertAcceptedHistoryForManagers(t, root)
	deliveryCreateAcknowledgedScope(t, operations, root)
	request := projectrun.PlanRequest{ExplorationID: "delivery", ScopeID: "orders-change", Operation: "apply", Goal: "Update the orders implementation", Managers: deliveryManagers(), ExecuteAuthorized: true}
	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: request})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := operations.Run(context.Background(), RunOperation{Root: root, RunID: plan.ID})
	if err != nil || initial.Status != projectrun.StatusIntegrated {
		t.Fatalf("setup run = %+v, err=%v", initial, err)
	}
	priorInvocations := len(initial.Invocations)
	got, err := operations.Deliver(context.Background(), DeliverOperation{Root: root, Request: projectrun.DeliverRequest{ExplorationID: "delivery", ScopeID: "orders-change", RunID: plan.ID, ExecuteAuthorized: true}})
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if got.Status != projectrun.StatusApplied || got.RunID != plan.ID || got.Verification == nil || got.Apply == nil || got.Apply.Status != projectrun.StatusApplied {
		t.Fatalf("delivery did not reach guarded Apply: %+v", got)
	}
	if got.Verification.VerificationScope != "full" || got.Verification.ManagerVerification == nil || len(got.Verification.ManagerVerification.Managers) != 3 {
		t.Fatalf("guided delivery lost whole-repository Manager verification: %+v", got.Verification)
	}
	if got.Run == nil || got.Run.ID != plan.ID || len(got.Run.Invocations) != priorInvocations {
		t.Fatalf("delivery replayed proposals: prior=%d got=%+v", priorInvocations, got.Run)
	}
	exploration, err := operations.Explore(ExploreOperation{Selection: Selection{Root: root}, ExplorationID: "delivery"})
	if err != nil {
		t.Fatal(err)
	}
	if len(exploration.Record.Completions) != 1 || exploration.Record.Completions[0].RunID != plan.ID || exploration.Record.Completions[0].BindingDigest == "" {
		t.Fatalf("successful Apply did not complete exact scope binding: %+v", exploration.Record.Completions)
	}
	history, err := projecthistory.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	resolution := projectbriefing.EventResolutionStatus(history.StatusView, event.ID)
	if resolution.Status != projectbriefing.StatusDeliveredUncommitted || resolution.Resolution == nil || resolution.Resolution.Evidence.RunID != plan.ID {
		t.Fatalf("uncommitted delivery status = %+v", resolution)
	}
	second, err := operations.Deliver(context.Background(), DeliverOperation{Root: root, Request: projectrun.DeliverRequest{ExplorationID: "delivery", ScopeID: "orders-change", RunID: plan.ID, ExecuteAuthorized: true}})
	if err != nil || second.Status != projectrun.StatusApplied {
		t.Fatalf("idempotent delivery = %+v, err=%v", second, err)
	}
	if second.Verification != nil || second.Apply != nil {
		t.Fatalf("already-applied recovery repeated later stages: %+v", second)
	}
	if len(exploration.Record.Completions) != 1 {
		t.Fatalf("completion receipt duplicated: %+v", exploration.Record.Completions)
	}
	deliveryGit(t, root, append([]string{"add", "--"}, got.Apply.Written...)...)
	deliveryGit(t, root, "commit", "-m", "commit the delivery")
	history, err = projecthistory.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	resolution = projectbriefing.EventResolutionStatus(history.StatusView, event.ID)
	if resolution.Status != "resolved" || resolution.Resolution == nil || resolution.Resolution.Evidence.RunID != plan.ID || resolution.Resolution.Evidence.CandidateID != got.Run.Candidate.ID || !resolution.Resolution.Evidence.FullVerifyPassed || len(resolution.Resolution.Evidence.Delivered) != len(got.Apply.Written) {
		t.Fatalf("accepted model event was not resolved from committed full Verify + Apply: %+v", resolution)
	}
}

func deliveryAssertAcceptedHistoryForManagers(t *testing.T, root string) {
	t.Helper()
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	project, err := projectwork.Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	for _, manager := range project.Report.Managers {
		if _, _, _, err := projecthistory.LoadForManager(root, project.Report.ModelDigest, manager.ID, head); err != nil {
			t.Fatalf("accepted briefing history is not readable for manager %s at %s: %v", manager.ID, head, err)
		}
	}
}

func deliverySetMaxDuration(t *testing.T, root string, maximum time.Duration) {
	t.Helper()
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	runtime.Limits.MaxDuration = projectconfig.Duration(maximum)
	data, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, projectconfig.RuntimePath, string(data))
	deliveryGit(t, root, "add", projectconfig.RuntimePath)
	deliveryGit(t, root, "commit", "-m", "set delivery test runtime budget")
}

func TestDeliverRejectsStaleExistingRunBeforeResuming(t *testing.T) {
	root := deliveryFixture(t)
	operations := deliveryOperations()
	deliveryCreateAcknowledgedScope(t, operations, root)
	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: projectrun.PlanRequest{
		ExplorationID: "delivery", ScopeID: "orders-change", Operation: "apply", Goal: "Update the orders implementation", Managers: deliveryManagers(), ExecuteAuthorized: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	deliveryAppendBlockingDecision(t, operations, root)
	got, err := operations.Deliver(context.Background(), DeliverOperation{Root: root, Request: projectrun.DeliverRequest{
		ExplorationID: "delivery", ScopeID: "orders-change", RunID: plan.ID, ExecuteAuthorized: true,
	}})
	if !errors.Is(err, projectrun.ErrStale) || !strings.Contains(err.Error(), "readiness changed") {
		t.Fatalf("stale scope was not rejected before resuming: report=%+v err=%v", got, err)
	}
	if got.Run != nil && got.Run.Status != "" && got.Run.Status != projectrun.StatusPlanned {
		t.Fatalf("stale delivery unexpectedly started the existing run: %+v", got.Run)
	}
}

func TestDeliverCreatesOnePersistedRunAndDoesNotReplaceFailedRun(t *testing.T) {
	root := deliveryFixture(t)
	operations := deliveryOperations()
	deliveryCreateAcknowledgedScope(t, operations, root)
	invoker := &failingDeliveryInvoker{}
	operations.Invoker = invoker
	request := projectrun.DeliverRequest{ExplorationID: "delivery", ScopeID: "orders-change", ExecuteAuthorized: true}
	first, err := operations.Deliver(context.Background(), DeliverOperation{Root: root, Request: request})
	if err == nil || first.RunID == "" || first.Plan == nil || first.Run == nil || first.Run.Status != projectrun.StatusFailed {
		t.Fatalf("fresh delivery did not preserve its failed run for explicit recovery: report=%+v err=%v", first, err)
	}
	if invoker.calls != 1 {
		t.Fatalf("fresh delivery invoked executor %d times, want one", invoker.calls)
	}
	second, err := operations.Deliver(context.Background(), DeliverOperation{Root: root, Request: projectrun.DeliverRequest{
		ExplorationID: "delivery", ScopeID: "orders-change", RunID: first.RunID, ExecuteAuthorized: true,
	}})
	if err == nil || second.RunID != first.RunID || invoker.calls != 1 {
		t.Fatalf("failed run was silently replaced or retried: report=%+v calls=%d err=%v", second, invoker.calls, err)
	}
}

type failingDeliveryInvoker struct{ calls int }

func (i *failingDeliveryInvoker) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	return agentexec.RunResult{}, errors.New("bounded fixture invocation failure")
}

func (*failingDeliveryInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return projectexecutor.ProcessInvoker{}.Fingerprint(config)
}

func TestGuidedPlanCannotBypassAcknowledgedExploration(t *testing.T) {
	root := deliveryFixture(t)
	_, err := deliveryOperations().Plan(PlanOperation{Selection: Selection{Root: root}, Request: projectrun.PlanRequest{
		Goal: "Implement both artifacts", Operation: "apply", Managers: deliveryManagers(), ExecuteAuthorized: true,
	}})
	if err == nil || !strings.Contains(err.Error(), "guided implementation requires") {
		t.Fatalf("guided Plan bypassed the exploration and scope binding: %v", err)
	}
}

func deliveryAcceptAndDismissFirstEvent(t *testing.T, root string) projectbriefing.Event {
	t.Helper()
	revision := deliveryCommitStatementChange(t, root, "Implement the %s artifact.", "Implement the %s artifact with explicitly documented line-item behavior.")
	if _, err := projecthistory.EnsureAcceptedHistory(root, revision); err != nil {
		t.Fatal(err)
	}
	history, err := projecthistory.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, bundle := range history.Accepted.Briefings {
		for _, event := range bundle.Events {
			if projectbriefing.EventResolutionStatus(history.StatusView, event.ID).Status == "resolved" {
				continue
			}
			if len(event.AffectedManagers) == 0 {
				t.Fatalf("event has no affected Manager: %+v", event)
			}
			if _, err := projecthistory.Dismiss(root, event.ID, event.AffectedManagers[0], history.PersistedDigest); err != nil {
				t.Fatal(err)
			}
			after, err := projecthistory.Read(root)
			if err != nil {
				t.Fatal(err)
			}
			if status := projectbriefing.EventResolutionStatus(after.StatusView, event.ID).Status; status != "unresolved" {
				t.Fatalf("dismissal improperly resolved accepted event: %s", status)
			}
			return event
		}
	}
	t.Fatal("semantic model change did not create a briefing event")
	return projectbriefing.Event{}
}

func deliveryChildManager(name string) string {
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Manager\nmetadata:\n  name: " + name + "\n  namespace: " + name + "\npurpose: Owns the " + name + " implementation.\nspec:\n  parent:\n    apiVersion: " + projectmodel.APIVersion + "\n    kind: Manager\n    namespace: \"\"\n    name: project-owner\n  owns: [src/" + name + "/]\n"
}
func deliveryStatement(ns, name, desc string) string {
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Statement\nmetadata:\n  name: " + name + "\n  namespace: " + ns + "\npurpose: " + desc + "\nspec:\n  category: concept\n  description: " + desc + "\n"
}
func deliveryArtifact(ns, name, statement, check, path string) string {
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Artifact\nmetadata:\n  name: " + name + "\n  namespace: " + ns + "\npurpose: Implements the owned source behavior.\nspec:\n  role: implementation\n  realizes:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Statement\n      namespace: " + ns + "\n      name: " + statement + "\n  paths: [" + path + "]\n  checks:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Check\n      namespace: " + ns + "\n      name: " + check + "\n  required: true\n"
}
func deliveryCheck(t *testing.T, ns, name, statement, executable string) string {
	command := append([]string{filepath.Base(executable)}, deliveryProcessArgs(t, []string{"-test.run=^TestDeliveryCheckWorker$"})...)
	encoded, _ := json.Marshal(command)
	return "apiVersion: " + projectmodel.APIVersion + "\nkind: Check\nmetadata:\n  name: " + name + "\n  namespace: " + ns + "\npurpose: Check the materialized project candidate.\nspec:\n  command: " + string(encoded) + "\n  uses:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Statement\n      namespace: " + ns + "\n      name: " + statement + "\n  limitation: This fixture check validates only candidate bytes.\n"
}

func deliveryWrite(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
func deliveryGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}

// git is shared with the other projectapp integration test files.
func git(t *testing.T, root string, args ...string) string { return deliveryGit(t, root, args...) }
func deliveryFixtureScope() projectexplore.Record {
	goal := "Update the orders implementation"
	return projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: "delivery", Status: projectexplore.StatusActive, Request: goal, Scopes: []projectexplore.Scope{{ID: "orders-change", Name: "orders-change", Goal: goal, Operation: "apply", ManagerIDs: deliveryManagers()}}, Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
}
func deliveryCreateAcknowledgedScope(t *testing.T, operations Operations, root string) {
	t.Helper()
	selection := Selection{Root: root}
	record := deliveryFixtureScope()
	preview, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record})
	if err != nil {
		t.Fatal(err)
	}
	written, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: preview.Plan.Digest})
	if err != nil {
		t.Fatal(err)
	}
	ack := StructureAcknowledgementInput{Actor: "fixture-user", Authority: "approved for test", Provenance: "delivery-test", RecordedAt: time.Now().UTC()}
	ready, err := operations.Readiness(ReadinessOperation{Selection: selection, ExplorationID: record.ID, ScopeID: "orders-change", Acknowledgement: &ack})
	if err != nil {
		t.Fatal(err)
	}
	_, err = operations.Readiness(ReadinessOperation{Selection: selection, ExplorationID: record.ID, ScopeID: "orders-change", Acknowledgement: &ack, Write: true, ExpectedDigest: ready.WritePlan.Digest})
	if err != nil {
		t.Fatal(err)
	}
	_ = written
}

func deliveryCommitStatementChange(t *testing.T, root, oldText, newText string) string {
	t.Helper()
	for _, path := range []string{".markitect/model/orders/statement.yaml", ".markitect/model/inventory/statement.yaml"} {
		full := filepath.Join(root, filepath.FromSlash(path))
		data, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(filepath.Dir(path))
		updated := strings.Replace(string(data), fmt.Sprintf(oldText, name), fmt.Sprintf(newText, name), 1)
		if updated == string(data) {
			t.Fatalf("could not make semantic statement change in %s", path)
		}
		deliveryWrite(t, root, path, updated)
	}
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	document, err := projectwork.Document(project, false)
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, projectvalue.DocumentPath(project.Config), document)
	deliveryGit(t, root, "add", ".markitect/model/orders/statement.yaml", ".markitect/model/inventory/statement.yaml", projectvalue.DocumentPath(project.Config))
	deliveryGit(t, root, "commit", "-m", "clarify accepted orders behavior")
	return deliveryGit(t, root, "rev-parse", "HEAD")
}

func deliveryAppendBlockingDecision(t *testing.T, operations Operations, root string) {
	t.Helper()
	result, err := operations.Explore(ExploreOperation{Selection: Selection{Root: root}, ExplorationID: "delivery"})
	if err != nil {
		t.Fatal(err)
	}
	record := *result.Record
	record.Decisions = append(record.Decisions, projectexplore.Decision{ID: "blocking-change", ScopeIDs: []string{"orders-change"}, Question: "A new blocking decision was recorded after planning.", Blocking: true, Status: "open"})
	record.Digest = ""
	preview, err := operations.Explore(ExploreOperation{Selection: Selection{Root: root}, Record: &record})
	if err != nil {
		t.Fatal(err)
	}
	_, err = operations.Explore(ExploreOperation{Selection: Selection{Root: root}, Record: &record, Write: true, ExpectedDigest: preview.Plan.Digest})
	if err != nil {
		t.Fatal(err)
	}
}

func deliveryMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
