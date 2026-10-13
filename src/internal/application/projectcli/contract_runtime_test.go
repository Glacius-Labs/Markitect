package projectcli

import "go.yaml.in/yaml/v3"

import "github.com/Glacius-Labs/Markitect/src/internal/application/projecthistory"

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"context"
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

const runtimeContractCheck = "TestContractRuntimeRequiredCheckProcess"

func TestContractRuntimeRequiredCheckProcess(t *testing.T) {
	if !containsArg(os.Args, "contract-runtime-required-check") {
		return
	}
	content, err := os.ReadFile(filepath.Join("src", "value.txt"))
	if err != nil {
		t.Fatalf("candidate check could not read src/value.txt: %v", err)
	}
	if string(content) != "valid\n" {
		t.Fatalf("candidate check rejected %q", content)
	}
}

type contractRuntimeInvoker struct {
	calls        int
	invalidFirst bool
}

func (i *contractRuntimeInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return agentexec.Fingerprint(config)
}

func (i *contractRuntimeInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, _ agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	var contextData struct {
		Phase string `json:"phase"`
	}
	if err := json.Unmarshal(request.Context, &contextData); err != nil {
		return agentexec.RunResult{}, err
	}
	response := projectrun.TaskResponse{Status: "complete", Summary: "completed " + contextData.Phase,
		Delegations: []projectrun.Delegation{}, ReworkRequests: []projectrun.ReworkRequest{}, Integrated: contextData.Phase == "integrate",
		Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	files := []agentexec.CandidateFile{}
	if contextData.Phase == "work" {
		content := "valid\n"
		if i.invalidFirst && i.calls == 1 {
			content = "invalid\n"
		}
		files = append(files, agentexec.CandidateFile{Path: "src/value.txt", Mode: "0644", Content: content})
	}
	report, err := json.Marshal(response)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	fingerprint, err := i.Fingerprint(config)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	return agentexec.RunResult{
		Response: agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
			Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
			CandidateFiles: files, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, ReportJSON: report, Uncertainty: []string{}},
		Receipt: agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest,
			ConfigDigest: fingerprint, ProviderVersion: config.ProviderVersion, Outcome: agentexec.OutcomeProposed},
	}, nil
}

func runtimeContractCases() map[string]contractCase {
	return map[string]contractCase{
		"plan":    {contractPlanHappy, contractPlanNoWrite},
		"run":     {contractRunHappy, contractRunNoWrite},
		"resume":  {contractResumeHappy, contractResumeNoWrite},
		"repair":  {contractRepairHappy, contractRepairNoWrite},
		"verify":  {contractVerifyHappy, contractVerifyNoWrite},
		"apply":   {contractApplyHappy, contractApplyNoWrite},
		"deliver": {contractDeliverHappy, contractDeliverNoWrite},
	}
}

func newRuntimeContract(t *testing.T, surface string, invalidFirst bool) (*contractCaller, projectapp.Operations, *testkit.Repo, *contractRuntimeInvoker) {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("switch", "-c", "dev/runtime-contract")
	checkExe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	initPlan, err := projectwork.Init(repo.Dir, "runtime-contract", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range initPlan.Files {
		if file.Path == projectvalue.ManifestPath {
			repo.Write(file.Path, strings.Replace(file.Content, "coverageMode: full", "coverageMode: selected", 1))
		} else if file.Path == projectconfig.RuntimePath {
			repo.Write(file.Path, runtimeContractYAML(checkExe))
		} else {
			repo.Write(file.Path, file.Content)
		}
	}
	model := []struct{ path, body string }{
		{".markitect/model/statement.yaml", "apiVersion: " + projectmodel.APIVersion + "\nkind: Statement\nmetadata:\n  name: value-behavior\n  namespace: \"\"\npurpose: The value file contains the accepted behavior.\nspec:\n  category: concept\n  description: The value file contains the accepted behavior.\n"},
		{".markitect/model/artifact.yaml", "apiVersion: " + projectmodel.APIVersion + "\nkind: Artifact\nmetadata:\n  name: value-file\n  namespace: \"\"\npurpose: Implements the modeled value behavior.\nspec:\n  role: implementation\n  realizes:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Statement\n      namespace: \"\"\n      name: value-behavior\n  paths: [src/value.txt]\n  checks:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Check\n      namespace: \"\"\n      name: value-check\n  required: true\n"},
		{".markitect/model/check.yaml", "apiVersion: " + projectmodel.APIVersion + "\nkind: Check\nmetadata:\n  name: value-check\n  namespace: \"\"\npurpose: Check the materialized candidate.\nspec:\n  command: [go, tool, test2json, -t, " + yamlQuote(checkExe) + ", \"-test.run=^" + runtimeContractCheck + "$\", contract-runtime-required-check]\n  uses:\n    - apiVersion: " + projectmodel.APIVersion + "\n      kind: Statement\n      namespace: \"\"\n      name: value-behavior\n  limitation: This fixture check validates only the materialized value file.\n"},
	}
	for _, file := range model {
		repo.Write(file.path, file.body)
	}
	manifestPath := filepath.Join(repo.Dir, filepath.FromSlash(projectvalue.ManifestPath))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest projectvalue.Config
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.CoverageMode = "selected"
	manifest.WorkflowMode = projectvalue.WorkflowModeEmpty
	manifest.ModelFiles = append(manifest.ModelFiles,
		".markitect/model/statement.yaml", ".markitect/model/artifact.yaml", ".markitect/model/check.yaml")
	manifest.InventoryRoots = []string{"src"}
	manifestBytes, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(projectvalue.ManifestPath, string(manifestBytes))
	repo.Write("src/value.txt", "initial\n")
	repo.Commit("runtime contract fixture")
	invoker := &contractRuntimeInvoker{invalidFirst: invalidFirst}
	ops := projectapp.NewOperations()
	ops.Invoker = invoker
	return newContractCaller(t, surface, repo.Dir, ops), ops, repo, invoker
}

func runtimeContractYAML(executable string) string {
	config := projectconfig.Runtime{
		APIVersion: projectconfig.APIVersion,
		Mode:       "controlled-local",
		Agents: map[string]projectconfig.Agent{
			contractManagerID(): {
				Command: executable, Model: "contract-fixture", ProviderVersion: "contract-runtime-v1",
				CostMode: "unmetered", Timeout: projectconfig.Duration(30 * time.Second),
				MaxStdoutBytes: 1048576, MaxStderrBytes: 1048576, RuntimeFiles: []agentexec.RuntimeFile{},
				Environment: []string{"PATH", "PATHEXT", "TEMP", "TMP", "TMPDIR", "GOCACHE"}, Pricing: projectconfig.Pricing{},
			},
		},
		Limits: projectconfig.Limits{
			MaxDepth: 2, MaxStarts: 8, MaxRetries: 1, MaxParallel: 1,
			MaxDuration: projectconfig.Duration(2 * time.Minute), MaxCostMicros: 100000,
			MaxCandidateFileBytes: 4096, MaxCandidateBytes: 8192,
		},
	}
	encoded, _ := yaml.Marshal(config)
	return string(encoded)
}

func yamlQuote(value string) string {
	encoded, _ := yaml.Marshal(value)
	return strings.TrimSpace(string(encoded))
}

func containsArg(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted {
			return true
		}
	}
	return false
}

func contractPlanHappy(t *testing.T, surface string) {
	t.Helper()
	caller, _, _, invoker := newRuntimeContract(t, surface, false)
	goal := "Implement the modeled value behavior."
	preview := contractSuccess(t, caller, "plan", []string{"--goal", goal, "--manager", contractManagerID()}, map[string]any{
		"goal": goal, "manager": []string{contractManagerID()},
	})
	id, token := contractString(t, preview, "id"), contractString(t, preview, "previewDigest")
	if id == "" || token == "" || contractString(t, preview, "status") != projectrun.StatusPlanned {
		t.Fatalf("plan preview omitted its status or digest binding: %v", preview)
	}
	argv := []string{"--goal", goal, "--manager", contractManagerID(), "--expect", token, "--write"}
	var persisted map[string]any
	if surface == "cli" {
		args := append([]string{"plan", "--repo", caller.root}, argv...)
		code, output, stderr := runCLIWith(t, caller.ops, args...)
		if code != 0 || !json.Valid([]byte(output)) {
			t.Fatalf("CLI plan persist code=%d output=%s stderr=%s", code, output, stderr)
		}
		if err := json.Unmarshal([]byte(output), &persisted); err != nil {
			t.Fatal(err)
		}
	} else {
		persisted = contractSuccess(t, caller, "plan", argv, map[string]any{
			"goal": goal, "manager": []string{contractManagerID()}, "expect": token, "write": true,
		})
	}
	persistedID := contractString(t, persisted, "id")
	if persistedID == "" || contractString(t, persisted, "status") != projectrun.StatusPlanned || invoker.calls != 0 {
		t.Fatalf("persisted plan omitted its independent identity/status or invoked runtime: plan=%v calls=%d", persisted, invoker.calls)
	}
	planBytes, err := os.ReadFile(filepath.Join(caller.root, ".markitect", "records", "run", persistedID, "plan.json"))
	if err != nil {
		t.Fatalf("read persisted plan %s: %v", persistedID, err)
	}
	var stored projectrun.PlanRecord
	if err := json.Unmarshal(planBytes, &stored); err != nil {
		t.Fatalf("decode persisted plan %s: %v", persistedID, err)
	}
	baseRevision := contractGitHead(t, caller.root)
	if stored.ID != persistedID || stored.Status != projectrun.StatusPlanned || stored.Goal != goal || !stored.ExecuteAuthorized || stored.BaseRevision != baseRevision {
		t.Fatalf("plan response was not backed by the expected persisted record: got id=%q status=%q goal=%q execute=%v base=%q; want id=%q status=%q goal=%q execute=true base=%q",
			stored.ID, stored.Status, stored.Goal, stored.ExecuteAuthorized, stored.BaseRevision, persistedID, projectrun.StatusPlanned, goal, baseRevision)
	}
}

func contractPlanNoWrite(t *testing.T, surface string) {
	t.Helper()
	caller, _, repo, invoker := newRuntimeContract(t, surface, false)
	manifestBytes, err := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(projectvalue.ManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	var manifest projectvalue.Config
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.WorkflowMode = projectvalue.WorkflowModeGuided
	manifestBytes, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(projectvalue.ManifestPath, string(manifestBytes))
	managerPath := filepath.Join(repo.Dir, ".markitect", "model", "manager.yaml")
	managerBytes, err := os.ReadFile(managerPath)
	if err != nil {
		t.Fatal(err)
	}
	managerText := strings.Replace(string(managerBytes), "Owns the repository-wide engineering mandate and delegates bounded responsibility.", "Owns the repository-wide engineering mandate and delegates bounded responsibility with accepted context.", 1)
	if managerText == string(managerBytes) {
		t.Fatal("guided history fixture did not find the root Manager purpose")
	}
	repo.Write(".markitect/model/manager.yaml", managerText)
	repo.Commit("add guided history context")
	history, err := projecthistory.ReadAcceptedHistory(repo.Dir, contractGitHead(t, repo.Dir))
	if err != nil || len(history.Receipt.Bundles) != 1 {
		t.Fatalf("guided pending accepted history: bundles=%d err=%v", len(history.Receipt.Bundles), err)
	}
	caller.NoWrite(func() {
		preview := contractSuccess(t, caller, "plan", []string{"--goal", "Inspect the modeled value behavior.", "--manager", contractManagerID()}, map[string]any{
			"goal": "Inspect the modeled value behavior.", "manager": []string{contractManagerID()},
		})
		if contractString(t, preview, "status") != projectrun.StatusPlanned || contractString(t, preview, "previewDigest") == "" {
			t.Fatalf("plan preview is incomplete: %v", preview)
		}
	})
	if invoker.calls != 0 {
		t.Fatalf("plan preview invoked runtime %d times", invoker.calls)
	}
}

func contractRunHappy(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	report := contractSuccess(t, caller, "run", []string{plan.ID, "--execute"}, map[string]any{"plan": plan.ID, "execute": true})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if contractString(t, report, "status") != projectrun.StatusIntegrated || contractString(t, report, "candidate.id") == "" || invoker.calls != 1 {
		t.Fatalf("run did not integrate the one Manager's work: calls=%d report=%v", invoker.calls, report)
	}
}

func contractRunNoWrite(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	assertRuntimeContractTargetInitial(t, repo.Dir)
	before := invoker.calls
	caller.NoWrite(func() {
		contractUsageRejected(t, caller, "run", []string{plan.ID}, map[string]any{"plan": plan.ID})
	})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if invoker.calls != before {
		t.Fatalf("unauthorized run invoked runtime: before=%d after=%d", before, invoker.calls)
	}
}

func contractResumeHappy(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	interrupted, err := ops.Run(ctx, projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err == nil || interrupted.Status != projectrun.StatusInterrupted || invoker.calls != 0 {
		t.Fatalf("setup did not retain a safe queued interruption: status=%s calls=%d err=%v", interrupted.Status, invoker.calls, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	report := contractSuccess(t, caller, "resume", []string{plan.ID, "--execute"}, map[string]any{"run": plan.ID, "execute": true})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if contractString(t, report, "status") != projectrun.StatusIntegrated || invoker.calls != 1 {
		t.Fatalf("resume failed to continue the known queued run: calls=%d report=%v", invoker.calls, report)
	}
}

func contractResumeNoWrite(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	interrupted, err := ops.Run(ctx, projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err == nil || interrupted.Status != projectrun.StatusInterrupted || invoker.calls != 0 {
		t.Fatalf("setup did not retain a safe queued interruption: status=%s calls=%d err=%v", interrupted.Status, invoker.calls, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	before := invoker.calls
	caller.NoWrite(func() {
		contractUsageRejected(t, caller, "resume", []string{plan.ID}, map[string]any{"run": plan.ID})
	})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if invoker.calls != before {
		t.Fatalf("unauthorized resume invoked runtime: before=%d after=%d", before, invoker.calls)
	}
}

func contractRepairHappy(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, true)
	plan := contractPersistPlan(t, caller, ops)
	run, err := ops.Run(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err != nil || run.Status != projectrun.StatusIntegrated {
		t.Fatalf("setup run did not integrate: status=%s err=%v", run.Status, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	failed, err := ops.Verify(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err == nil || failed.Status != "failed" || len(failed.Checks) != 1 || failed.Checks[0].ExitCode == 0 {
		t.Fatalf("setup check did not fail as intended: report=%+v err=%v", failed, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	repaired := contractSuccess(t, caller, "repair", []string{plan.ID, "--execute"}, map[string]any{"run": plan.ID, "execute": true})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if contractString(t, repaired, "status") != projectrun.StatusIntegrated || invoker.calls != 2 {
		t.Fatalf("repair did not produce a fresh integrated candidate: calls=%d report=%v", invoker.calls, repaired)
	}
	verified := contractSuccess(t, caller, "verify", []string{plan.ID, "--execute"}, map[string]any{"run": plan.ID, "execute": true})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if contractString(t, verified, "status") != "verified" {
		t.Fatalf("repaired candidate did not pass fresh verification: %v", verified)
	}
}

func contractRepairNoWrite(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, true)
	plan := contractPersistPlan(t, caller, ops)
	run, err := ops.Run(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err != nil || run.Status != projectrun.StatusIntegrated {
		t.Fatalf("setup run did not integrate: status=%s err=%v", run.Status, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	failure, err := ops.Verify(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err == nil || failure.Status != projectrun.StatusFailed || len(failure.Checks) != 1 || failure.Checks[0].ExitCode == 0 {
		t.Fatalf("setup required-check failure was not recorded: report=%+v err=%v", failure, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	before := invoker.calls
	caller.NoWrite(func() {
		contractUsageRejected(t, caller, "repair", []string{plan.ID}, map[string]any{"run": plan.ID})
	})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if invoker.calls != before {
		t.Fatalf("unauthorized repair invoked runtime: before=%d after=%d", before, invoker.calls)
	}
}

func contractVerifyHappy(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	if _, err := ops.Run(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID}); err != nil {
		t.Fatal(err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	verified := contractSuccess(t, caller, "verify", []string{plan.ID, "--execute"}, map[string]any{"run": plan.ID, "execute": true})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if contractString(t, verified, "status") != "verified" || invoker.calls != 1 {
		t.Fatalf("verify did not pass the required check without a verifier call: calls=%d report=%v", invoker.calls, verified)
	}
}

func contractVerifyNoWrite(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	if _, err := ops.Run(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID}); err != nil {
		t.Fatal(err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	before := invoker.calls
	caller.NoWrite(func() {
		contractUsageRejected(t, caller, "verify", []string{plan.ID}, map[string]any{"run": plan.ID})
	})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if invoker.calls != before {
		t.Fatalf("unauthorized verify invoked runtime: before=%d after=%d", before, invoker.calls)
	}
}

func contractApplyHappy(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	run, err := ops.Run(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err != nil || run.Status != projectrun.StatusIntegrated {
		t.Fatalf("setup run failed: status=%s err=%v", run.Status, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	verification, err := ops.Verify(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err != nil || verification.Status != "verified" {
		t.Fatalf("setup verification failed: status=%s err=%v", verification.Status, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	preflight := contractSuccess(t, caller, "apply", []string{"--plan", plan.ID, "--run", plan.ID, "--candidate", run.Candidate.ID}, map[string]any{
		"plan": plan.ID, "run": plan.ID, "candidate": run.Candidate.ID,
	})
	assertRuntimeContractTargetInitial(t, repo.Dir) // Read-only preflight must leave it untouched.
	branch, head := contractString(t, preflight, "targetBranch"), contractString(t, preflight, "expectedHead")
	worktree, verifyDigest := contractString(t, preflight, "expectedWorktree"), contractString(t, preflight, "verificationDigest")
	if branch == "" || head == "" || worktree == "" || verifyDigest != verification.Digest {
		t.Fatalf("preflight omitted the reviewed verification bindings: %v", preflight)
	}
	result := contractSuccess(t, caller, "apply", []string{"--plan", plan.ID, "--run", plan.ID, "--candidate", run.Candidate.ID,
		"--branch", branch, "--head", head, "--worktree", worktree, "--expect", verifyDigest, "--write"}, map[string]any{
		"plan": plan.ID, "run": plan.ID, "candidate": run.Candidate.ID, "branch": branch, "head": head,
		"worktree": worktree, "expect": verifyDigest, "write": true,
	})
	content, err := os.ReadFile(filepath.Join(repo.Dir, "src", "value.txt"))
	if contractString(t, result, "status") != projectrun.StatusApplied || err != nil || string(content) != "valid\n" || invoker.calls != 1 {
		t.Fatalf("apply did not write the verified content: status=%s content=%q calls=%d err=%v", contractString(t, result, "status"), content, invoker.calls, err)
	}
}

func contractApplyNoWrite(t *testing.T, surface string) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	run, err := ops.Run(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	verification, err := ops.Verify(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err != nil || verification.Status != "verified" {
		t.Fatalf("setup verification failed: report=%+v err=%v", verification, err)
	}
	assertRuntimeContractTargetInitial(t, repo.Dir)
	beforeCalls := invoker.calls
	caller.NoWrite(func() {
		assertRuntimeContractTargetInitial(t, repo.Dir)
		preflight := contractSuccess(t, caller, "apply", []string{"--plan", plan.ID, "--run", plan.ID, "--candidate", run.Candidate.ID}, map[string]any{
			"plan": plan.ID, "run": plan.ID, "candidate": run.Candidate.ID,
		})
		assertRuntimeContractTargetInitial(t, repo.Dir)
		if contractString(t, preflight, "verificationDigest") != verification.Digest {
			t.Fatalf("read-only preflight is not bound to successful verification: %v", preflight)
		}
	})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if invoker.calls != beforeCalls {
		t.Fatalf("read-only preflight invoked runtime: before=%d after=%d", beforeCalls, invoker.calls)
	}
}

func contractDeliverHappy(t *testing.T, surface string) {
	t.Helper()
	caller, _, repo, invoker := newRuntimeContract(t, surface, false)
	contractAcknowledgeExploration(t, repo.Dir)
	assertRuntimeContractTargetInitial(t, repo.Dir)
	result := contractSuccess(t, caller, "deliver", []string{"--exploration", "runtime-delivery", "--scope", "value-scope", "--execute", "--write"}, map[string]any{
		"exploration": "runtime-delivery", "scope": "value-scope", "execute": true, "write": true,
	})
	content, err := os.ReadFile(filepath.Join(repo.Dir, "src", "value.txt"))
	if contractString(t, result, "status") != projectrun.StatusApplied || err != nil || string(content) != "valid\n" || invoker.calls != 1 {
		t.Fatalf("deliver did not execute, verify and apply its acknowledged scope: status=%s content=%q calls=%d err=%v", contractString(t, result, "status"), content, invoker.calls, err)
	}
}

func contractDeliverNoWrite(t *testing.T, surface string) {
	t.Helper()
	caller, _, repo, invoker := newRuntimeContract(t, surface, false)
	contractAcknowledgeExploration(t, repo.Dir)
	assertRuntimeContractTargetInitial(t, repo.Dir)
	before := invoker.calls
	caller.NoWrite(func() {
		contractUsageRejected(t, caller, "deliver", []string{"--exploration", "runtime-delivery", "--scope", "value-scope"}, map[string]any{
			"exploration": "runtime-delivery", "scope": "value-scope",
		})
		contractUsageRejected(t, caller, "deliver", []string{"--exploration", "runtime-delivery", "--scope", "value-scope", "--execute"}, map[string]any{
			"exploration": "runtime-delivery", "scope": "value-scope", "execute": true,
		})
		contractUsageRejected(t, caller, "deliver", []string{"--exploration", "runtime-delivery", "--scope", "value-scope", "--write"}, map[string]any{
			"exploration": "runtime-delivery", "scope": "value-scope", "write": true,
		})
	})
	assertRuntimeContractTargetInitial(t, repo.Dir)
	if invoker.calls != before {
		t.Fatalf("unauthorized deliver invoked runtime: before=%d after=%d", before, invoker.calls)
	}
}

func assertRuntimeContractTargetInitial(t *testing.T, root string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, "src", "value.txt"))
	if err != nil {
		t.Fatalf("read target artifact before guarded apply: %v", err)
	}
	if string(content) != "initial\n" {
		t.Fatalf("runtime wrote target artifact before guarded apply: got %q, want %q", content, "initial\n")
	}
}

func contractAcknowledgeExploration(t *testing.T, root string) {
	t.Helper()
	const id, scope, goal = "runtime-delivery", "value-scope", "Implement the modeled value behavior."
	binding, err := projectrun.ExplorationBinding(contractProjectHost(), root, contractGitHead(t, root), projectrun.PlanRequest{
		ScopeID: scope, Goal: goal, Operation: projectrun.OperationApply, Managers: []string{contractManagerID()},
	})
	if err != nil {
		t.Fatal(err)
	}
	record := projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: id, Status: projectexplore.StatusActive, Request: goal,
		Scopes:    []projectexplore.Scope{{ID: scope, Name: scope, Goal: goal, Operation: projectrun.OperationApply, ManagerIDs: []string{contractManagerID()}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
	preview, err := projectexplore.CreatePreview(root, record, binding)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := projectexplore.Write(root, preview, preview.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	readyResult, err := projectapp.NewOperations().Readiness(projectapp.ReadinessOperation{Selection: projectapp.Selection{Root: root}, ExplorationID: id, ScopeID: scope})
	scopeBinding, readiness := readyResult.Binding, readyResult.Readiness
	if err != nil {
		t.Fatal(err)
	}
	priorDigest := persisted.Digest
	if err := projectexplore.AcknowledgeStructure(&persisted, scope, scopeBinding, projectexplore.StructureAcknowledgement{
		ScopeID: scope, BindingDigest: readiness.BindingDigest, StructureDigest: readiness.StructureDigest,
		Actor: "fixture-user", Authority: "explicit contract test approval", Provenance: "contract_runtime_test", RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	ackPreview, err := projectexplore.UpdatePreview(root, persisted, priorDigest, scopeBinding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectexplore.Write(root, ackPreview, ackPreview.Digest, scopeBinding); err != nil {
		t.Fatal(err)
	}
}

func contractPersistPlan(t *testing.T, caller *contractCaller, ops projectapp.Operations) projectrun.PlanRecord {
	t.Helper()
	goal := "Implement the modeled value behavior."
	preview, err := ops.Plan(projectapp.PlanOperation{Selection: projectapp.Selection{Root: caller.root}, Request: projectrun.PlanRequest{
		Goal: goal, Managers: []string{contractManagerID()},
	}})
	if err != nil || preview.Status != projectrun.StatusPlanned {
		t.Fatalf("create plan preview: status=%s err=%v", preview.Status, err)
	}
	digest, err := ops.PlanPreviewDigest(preview)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ops.Plan(projectapp.PlanOperation{Selection: projectapp.Selection{Root: caller.root}, Request: projectrun.PlanRequest{
		Goal: goal, Managers: []string{contractManagerID()}, ExecuteAuthorized: true,
	}, ExpectedPreviewDigest: digest})
	if err != nil || plan.ID == "" || !plan.ExecuteAuthorized {
		t.Fatalf("persist plan: id=%q authorized=%v err=%v", plan.ID, plan.ExecuteAuthorized, err)
	}
	return plan
}

func contractSuccess(t *testing.T, caller *contractCaller, name string, argv []string, input any) map[string]any {
	t.Helper()
	result := caller.Invoke(name, argv, input)
	if result.Code != 0 || !json.Valid(result.Data) {
		t.Fatalf("%s %s: code=%d data=%s diagnostic=%+v", caller.surface, name, result.Code, result.Data, result.Diagnostic)
	}
	var value map[string]any
	if err := json.Unmarshal(result.Data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func contractUsageRejected(t *testing.T, caller *contractCaller, name string, argv []string, input any) {
	t.Helper()
	result := caller.Invoke(name, argv, input)
	if result.Code != 2 {
		t.Fatalf("%s %s without required execution/write authorization should be a usage rejection: code=%d data=%s diagnostic=%+v", caller.surface, name, result.Code, result.Data, result.Diagnostic)
	}
	if caller.surface == "mcp" && (result.Diagnostic == nil || result.Diagnostic.Code != "invalid_arguments") {
		t.Fatalf("%s %s did not expose invalid_arguments: %+v", caller.surface, name, result.Diagnostic)
	}
}

func contractString(t *testing.T, value map[string]any, path string) string {
	t.Helper()
	var current any = value
	for _, key := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("response path %q missing from %v", path, value)
		}
		current = object[key]
	}
	text, _ := current.(string)
	return text
}

func contractManagerID() string {
	id, _ := json.Marshal([]string{projectmodel.APIVersion, "Manager", "", "project-owner"})
	return string(id)
}

func contractGitHead(t *testing.T, root string) string {
	t.Helper()
	return strings.TrimSpace(testkit.Git(t, root, "rev-parse", "HEAD"))
}

func contractProjectHost() projectrun.Host {
	return projectapp.NewOperations().Host
}
