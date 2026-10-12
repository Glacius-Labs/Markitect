package projectrun

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
)

// These end-to-end tests drive the delegated method through bring-your-own
// process executors with a separate profile per role: metered Managers that
// report usage, and unmetered reviewers and verifier that report none. The
// scripted executor injects faults; scope, review, Verify and Apply guards
// must hold exactly as for the native runtime.

const (
	e2eManagerModel  = "fixture-manager-model"
	e2eReviewerModel = "fixture-review-model"
	e2eVerifierModel = "fixture-verifier-model"
)

func configureMixedRoleRuntime(t *testing.T, root string) {
	t.Helper()
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
		reviewers := make(map[string]projectconfig.Agent, len(config.Agents))
		for id, agent := range config.Agents {
			agent.Model, agent.ModelOptions = e2eManagerModel, map[string]any{"reasoningEffort": "high"}
			agent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 1_000_000, OutputMicrosPerMillion: 1_000_000}
			config.Agents[id] = agent
			reviewer := agent
			reviewer.Model, reviewer.ModelOptions = e2eReviewerModel, map[string]any{"reasoningEffort": "medium"}
			reviewer.CostMode, reviewer.Pricing = projectconfig.CostModeUnmetered, projectconfig.Pricing{}
			reviewers[id] = reviewer
		}
		verifier := config.Agents[e2eManagerID("", "project-owner")]
		verifier.Model, verifier.ModelOptions = e2eVerifierModel, nil
		verifier.CostMode, verifier.Pricing = projectconfig.CostModeUnmetered, projectconfig.Pricing{}
		config.Verifier = &verifier
		config.Review = &projectconfig.ReviewConfig{Agents: reviewers, MaxRounds: 3, MaxManagerRounds: 2}
		config.Limits.MaxStarts = 64
		// One bounded repair round for check failures found by Verify.
		config.Limits.MaxRetries = 1
		// A loaded Windows host needs minutes per scenario for its Git calls;
		// no scenario here tests the duration budget.
		config.Limits.MaxDuration = projectconfig.Duration(30 * time.Minute)
	})
	setRuntimeFixtureEnv(t, e2eOmitUsageEnv, "review,verify")
}

func runtimeLifecycleProject(t *testing.T, root string, nested bool) *projectvalue.Project {
	t.Helper()
	revision := identityHead(t, root)
	return processRuntimeProject(t, root, revision, nested)
}

func runtimeLifecycleHost(t *testing.T, root string) Host {
	return processRuntimeHost(root, runtimeLifecycleProject(t, root, false))
}

func includeFiniteRuntimeFiles(t *testing.T, project *projectvalue.Project, root string, paths ...string) {
	t.Helper()
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read explicit Runtime fixture file %s: %v", path, err)
		}
		project.Snapshot.Files[path] = append([]byte(nil), content...)
		project.Snapshot.Modes[path] = "100644"
	}
}

func runtimeProjectWithDigest(project *projectvalue.Project, suffix string) *projectvalue.Project {
	copy := *project
	copy.Digest = project.Digest + suffix
	return &copy
}

func runtimeProjectWithFile(project *projectvalue.Project, path string, content []byte) *projectvalue.Project {
	copy := *project
	files := make(map[string][]byte, len(project.Snapshot.Files)+1)
	for existing, bytes := range project.Snapshot.Files {
		files[existing] = append([]byte(nil), bytes...)
	}
	files[path] = append([]byte(nil), content...)
	modes := make(map[string]string, len(project.Snapshot.Modes)+1)
	for existing, mode := range project.Snapshot.Modes {
		modes[existing] = mode
	}
	if modes[path] == "" {
		modes[path] = "100644"
	}
	copy.Snapshot = &snapshot.Snapshot{ID: project.Snapshot.ID, Provisional: project.Snapshot.Provisional, Files: files, Modes: modes}
	copy.Digest = project.Digest + ":source-change"
	return &copy
}

func addRuntimeIntegrationFacts(t *testing.T, project *projectvalue.Project) {
	t.Helper()
	rootID := e2eManagerID("", "project-owner")
	statementID := e2eFixtureIdentity("Statement", "", "integration-summary")
	artifactID := e2eFixtureIdentity("Artifact", "", "integration-summary-file")
	checkID := e2eFixtureIdentity("Check", "", "integration-check")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := append([]string{filepath.Base(executable)}, runtimeFixtureArgs(t, []string{"-test.run=^TestProjectRunIntegrationCheckProcess$"})...)
	project.Report.Statements = append(project.Report.Statements, projectmodel.Statement{
		ID: statementID, Name: "integration-summary", Owner: rootID, Category: "concept", Description: "The integration summary lists the integrated child artifacts.",
	})
	project.Report.Artifacts = append(project.Report.Artifacts, projectmodel.Artifact{
		ID: artifactID, Name: "integration-summary-file", Owner: rootID, Role: "implementation", Realizes: []string{statementID}, Paths: []string{"src/project-integration.txt"}, Checks: []string{checkID}, Required: true,
	})
	project.Report.Checks = append(project.Report.Checks, projectmodel.Check{
		ID: checkID, Name: "integration-check", Owner: rootID, Command: command, Uses: []string{statementID}, Limitation: "This fixture check compares candidate bytes only.",
	})
	project.Report.Files = append(project.Report.Files, projectmodel.FileEntry{
		Path: "src/project-integration.txt", Owner: rootID, Class: "source", Statements: []string{statementID}, Artifacts: []string{artifactID}, Checks: []string{checkID}, Exists: true,
	})
	project.Snapshot.Files["src/project-integration.txt"] = []byte("pending integration\n")
	project.Snapshot.Modes["src/project-integration.txt"] = "100644"
	project.Report.Digest = "sha256:5555555555555555555555555555555555555555555555555555555555555555"
}
func planBothManagers(t *testing.T, root string) PlanRecord {
	t.Helper()
	plan, err := Plan(runtimeLifecycleHost(t, root), root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	return plan
}

func verifyAndApply(t *testing.T, root string, plan PlanRecord, candidateID string, hosts ...Host) {
	t.Helper()
	host := runtimeLifecycleHost(t, root)
	if len(hosts) > 1 {
		t.Fatalf("verifyAndApply received %d Host values", len(hosts))
	}
	if len(hosts) == 1 {
		host = hosts[0]
	}
	verified, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified || verified.Verifier == nil {
		t.Fatalf("Verify did not pass with the independent verifier: report=%+v err=%v", verified, err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, candidateID)
	if err != nil {
		t.Fatalf("PreflightApply: %v", err)
	}
	applied, err := Apply(host, projectexecutor.ProcessInvoker{}, root, applyRequestFromPreflight(preflight))
	if err != nil || applied.Status != StatusApplied {
		t.Fatalf("guarded Apply failed: report=%+v err=%v", applied, err)
	}
}

// failedCheckWith finds a failed declared check by its diagnostic. The fixture
// checks inspect the whole candidate, so the diagnostic names the defect.
func failedCheckWith(checks []CheckResult, diagnostic string) bool {
	for _, check := range checks {
		if check.Outcome != "passed" && strings.Contains(check.Stderr, diagnostic) {
			return true
		}
	}
	return false
}

func planHasCheck(plan PlanRecord, name string) bool {
	for _, check := range plan.Checks {
		if strings.Contains(check.ID, name) {
			return true
		}
	}
	return false
}

// addRootIntegrationCheck gives the root Manager an integration artifact with
// its own declared check, which only the aggregate candidate can satisfy.
func addRootIntegrationCheck(t *testing.T, root string) {
	t.Helper()
	manifestPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "  - .markitect/model/manager.yaml\n",
		"  - .markitect/model/manager.yaml\n  - .markitect/model/integration-statement.yaml\n  - .markitect/model/integration-artifact.yaml\n  - .markitect/model/integration-check.yaml\n", 1)
	if updated == string(manifest) {
		t.Fatal("could not add the root integration model files")
	}
	writeE2E(t, root, projectvalue.ManifestPath, updated)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, ".markitect/model/integration-statement.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Statement\nmetadata:\n  name: integration-summary\n  namespace: \"\"\npurpose: The integration summary lists the integrated child artifacts.\nspec:\n  category: concept\n  description: The integration summary lists the integrated child artifacts.\n")
	writeE2E(t, root, ".markitect/model/integration-artifact.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Artifact\nmetadata:\n  name: integration-summary-file\n  namespace: \"\"\npurpose: Summarizes the integrated child artifacts.\nspec:\n  role: implementation\n  realizes:\n    - apiVersion: "+projectmodel.APIVersion+"\n      kind: Statement\n      namespace: \"\"\n      name: integration-summary\n  paths: [src/project-integration.txt]\n  checks:\n    - apiVersion: "+projectmodel.APIVersion+"\n      kind: Check\n      namespace: \"\"\n      name: integration-check\n  required: true\n")
	writeE2E(t, root, ".markitect/model/integration-check.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Check\nmetadata:\n  name: integration-check\n  namespace: \"\"\npurpose: Check that the integration summary matches the integrated children.\nspec:\n  command: "+runtimeFixtureCommand(t, filepath.Base(executable), "-test.run=^TestProjectRunIntegrationCheckProcess$")+"\n  uses:\n    - apiVersion: "+projectmodel.APIVersion+"\n      kind: Statement\n      namespace: \"\"\n      name: integration-summary\n  limitation: This fixture check compares candidate bytes only.\n")
	writeE2E(t, root, "src/project-integration.txt", "pending integration\n")
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "-m", "add root integration check")
}

// nativeRootInvoker answers native Manager turns in-process, as a completed
// App Server turn without edits that delegates to every direct child, and runs
// process Managers through the fixture executor.
type nativeRootInvoker struct {
	process projectexecutor.ProcessInvoker
}

func (i nativeRootInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	if config.Transport != projectconfig.TransportCodexAppServer {
		return i.process.Run(ctx, config, request, options)
	}
	var task struct {
		Phase          string   `json:"phase"`
		DirectChildren []string `json:"directChildren"`
	}
	if err := json.Unmarshal(request.Context, &task); err != nil {
		return agentexec.RunResult{}, err
	}
	response := TaskResponse{Status: "complete", Summary: "native root " + task.Phase, Integrated: task.Phase == "integrate", Delegations: []Delegation{},
		ReworkRequests: []ReworkRequest{}, Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	if task.Phase == "work" {
		for _, child := range task.DirectChildren {
			response.Delegations = append(response.Delegations, Delegation{ManagerID: child, Goal: "Implement the owned source artifact."})
		}
	}
	reportJSON, err := json.Marshal(response)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	fingerprint, err := i.Fingerprint(config)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	tokens := int64(1)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: &tokens, OutputTokens: &tokens}
	lifecycle := &agentexec.Lifecycle{Provider: projectconfig.TransportCodexAppServer, SessionID: "native-root-session", TurnID: "native-root-" + task.Phase, State: "completed", Accounting: "partial",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: invocation.RunID, Role: agentexec.RoleExecutor, State: "completed"}}}
	return agentexec.RunResult{
		Response: agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce, Role: agentexec.RoleExecutor,
			InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{},
			VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}, ReportJSON: reportJSON, Usage: usage},
		Receipt: agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest, ConfigDigest: fingerprint,
			ProviderVersion: config.ProviderVersion, Outcome: agentexec.OutcomeProposed, Usage: usage, Lifecycle: lifecycle},
	}, nil
}

func (nativeRootInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return projectexecutor.New(codexappserver.Options{}, NativeHelperPolicy{}).Fingerprint(config)
}
