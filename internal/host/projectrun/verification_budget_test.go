package projectrun

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
	yaml "go.yaml.in/yaml/v3"
)

const verifierProcessHelperEnv = "MARKITECT_PROJECTRUN_VERIFY_HELPER"

// TestProjectRunVerifierHelperProcess is invoked as a real agentexec child by
// TestFailedVerifierAttemptIsDurableAndCannotReplayBudget. Its failed response
// is still protocol-valid, so the projectrun verifier's role/outcome validation
// runs without contacting a provider.
func TestProjectRunVerifierHelperProcess(t *testing.T) {
	if os.Getenv(verifierProcessHelperEnv) != "failed-verifier" {
		return
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(data, &invocation); err != nil {
		t.Fatal(err)
	}
	refs := append(append([]string(nil), invocation.Request.ScopeIDs...), invocation.Request.PolicyIDs...)
	sort.Strings(refs)
	unique := refs[:0]
	for _, ref := range refs {
		if len(unique) == 0 || unique[len(unique)-1] != ref {
			unique = append(unique, ref)
		}
	}
	inTokens, outTokens := int64(1_000_000), int64(0)
	response := agentexec.Response{
		APIVersion: invocation.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleVerifier, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeFailed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: unique,
		VerifierObservations: []agentexec.Observation{{Subject: "check-1", Outcome: "failed", Detail: "fixture verifier rejected the result"}},
		Uncertainty:          []string{"fixture failure"},
		Usage:                &agentexec.Usage{Source: "provider-reported", InputTokens: &inTokens, OutputTokens: &outTokens},
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

type countingProcessInvoker struct {
	calls int
}

func (i *countingProcessInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	return (ProcessInvoker{}).Run(ctx, config, request, options)
}

func (i *countingProcessInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return (ProcessInvoker{}).Fingerprint(config)
}

func TestFailedVerifierAttemptIsDurableAndCannotReplayBudget(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "--quiet")
	if err := os.WriteFile(filepath.Join(root, "fixture.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "fixture.txt")
	git(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "--quiet", "-m", "fixture")
	head := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	branch, err := resolveGitBranch(root)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, executableBytes, err := readPinnedExecutable(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(verifierProcessHelperEnv, "failed-verifier"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv(verifierProcessHelperEnv) })
	agent := Agent{
		Command: executable, Args: []string{"-test.run=^TestProjectRunVerifierHelperProcess$"}, Model: "fixture",
		ProviderVersion: "fixture-v1", Timeout: Duration(30 * time.Second), MaxStdoutBytes: 64 << 10, MaxStderrBytes: 16 << 10,
		Environment: []string{verifierProcessHelperEnv, "SystemRoot"}, Pricing: Pricing{InputMicrosPerMillion: 1000},
	}
	runtime := validRuntime()
	runtime.Limits.MaxStarts = 2 // one executable check and one verifier process
	runtime.Limits.MaxCostMicros = 1
	checkAgent := runtime.Agents["commerce"]
	checkAgent.Command = executable
	checkAgent.Args = []string{"-test.run=^TestProjectRunVerifierHelperProcess$"}
	checkAgent.Environment = []string{"SystemRoot"}
	checkAgent.ProviderVersion = "fixture-v1"
	checkAgent.Timeout = Duration(30 * time.Second)
	checkAgent.RuntimeFiles = []agentexec.RuntimeFile{}
	runtime.Agents["commerce"] = checkAgent
	agent.RuntimeFiles = []agentexec.RuntimeFile{}
	runtime.Verifier = &agent
	markitectDir := filepath.Join(root, ".markitect")
	if err := os.MkdirAll(markitectDir, 0o700); err != nil {
		t.Fatal(err)
	}
	runtimeData, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(RuntimePath)), runtimeData, 0o600); err != nil {
		t.Fatal(err)
	}

	snapshotBase := &snapshot.Snapshot{ID: head, Files: map[string][]byte{"fixture.txt": []byte("base")}, Modes: map[string]string{"fixture.txt": "100644"}}
	modelDigest := "sha256:" + strings.Repeat("a", 64)
	project := &Project{Root: root, Revision: head, Digest: "project-digest", Report: projectmodel.Report{
		ModelDigest: modelDigest, Digest: "project-report-digest", Status: "complete",
		Artifacts: []projectmodel.Artifact{}, Findings: []projectmodel.Finding{},
	}, Snapshot: snapshotBase}
	host := Host{
		Load: func(_, _ string) (*Project, error) { return project, nil },
		FromSnapshot: func(_ string, snap *Snapshot) (*Project, error) {
			copy := *project
			copy.Snapshot = snap
			return &copy, nil
		},
	}
	invoker := &countingProcessInvoker{}
	runtimeDigest, err := runtimeDigestWithInvoker(invoker, runtime)
	if err != nil {
		t.Fatal(err)
	}

	const runID = "11111111111111111111111111111111"
	const candidateID = "22222222222222222222222222222222"
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(root, filepath.FromSlash(RunsPath))); err != nil {
		t.Fatal(err)
	}
	dir, err := store.createRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	check := CheckPlan{ID: "check-1", Owner: "commerce", Command: []string{executable, "-test.run=^TestProjectRunVerifierHelperProcess$"}, ExecutablePath: executable, ExecutableDigest: rawContentDigest(executableBytes), Required: true}
	plan := PlanRecord{
		APIVersion: APIVersion, ID: runID, Status: StatusIntegrated, Goal: "verification budget fixture", ExecuteAuthorized: true,
		BaseRevision: head, TargetBranch: branch, TargetHead: head, RepositoryDigest: identity.Digest, BaseSnapshot: snapshotBase.Digest(), WorkingSnapshot: snapshotBase.Digest(),
		BaseProjectDigest: project.Digest, WorkingProjectDigest: project.Digest, BaseModelDigest: project.Report.ModelDigest,
		ModelDigest: project.Report.ModelDigest, ReportDigest: project.Report.Digest, RuntimeDigest: runtimeDigest,
		Checks: []CheckPlan{check}, InitialCandidateID: candidateID, RuntimeAgents: map[string]string{"commerce": "fixture"},
	}
	plan.Digest, err = planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.writePlan(plan); err != nil {
		t.Fatal(err)
	}
	candidate := candidateData{ID: candidateID, Files: map[string]File{}}
	if err := store.writeCandidate(dir, candidate); err != nil {
		t.Fatal(err)
	}
	candidate, err = store.readCandidate(dir, candidateID)
	if err != nil {
		t.Fatal(err)
	}
	run := RunReport{
		APIVersion: APIVersion, ID: runID, PlanID: runID, Status: StatusIntegrated,
		Mode: ModeControlledLocal, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		BaseRevision: head, BaseSnapshot: snapshotBase.Digest(), ModelDigest: project.Report.ModelDigest,
		RuntimeDigest: runtimeDigest, Candidate: candidateRef(candidate, true), Tasks: []ManagerTask{},
		Invocations: []InvocationLog{}, Checks: []CheckResult{},
	}
	if err := persistState(store, &run); err != nil {
		t.Fatal(err)
	}
	loadedRuntime, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	loadedRuntimeDigest, err := runtimeDigestWithInvoker(invoker, loadedRuntime)
	if err != nil {
		t.Fatal(err)
	}
	if loadedRuntimeDigest != plan.RuntimeDigest {
		originalJSON, _ := json.Marshal(runtime)
		loadedJSON, _ := json.Marshal(loadedRuntime)
		t.Fatalf("runtime digest mismatch before Verify: want %s got %s\noriginal=%s\nloaded=%s", plan.RuntimeDigest, loadedRuntimeDigest, originalJSON, loadedJSON)
	}

	report, err := Verify(context.Background(), host, invoker, root, runID)
	if err == nil || !strings.Contains(err.Error(), "did not pass") {
		t.Fatalf("expected failed verifier result, got report=%+v err=%v", report, err)
	}
	if report.Digest == "" {
		t.Fatal("failed verification report omitted its digest")
	}
	persisted, err := latestVerification(dir, candidateID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != "failed" || persisted.Digest != report.Digest {
		t.Fatalf("failed report digest was not durably persisted: %+v", persisted)
	}
	latest, err := store.readLatestState(runID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Status != StatusFailed || len(latest.Checks) != 1 || latest.Checks[0].Outcome != "passed" {
		t.Fatalf("failed verification attempt was not durably accounted: status=%s checks=%+v", latest.Status, latest.Checks)
	}
	if len(latest.Invocations) != 1 || latest.Invocations[0].Phase != "verify" || latest.Invocations[0].CostMicros != 1000 {
		t.Fatalf("failed verifier start/cost was not persisted: %+v", latest.Invocations)
	}
	if invoker.calls != 1 {
		t.Fatalf("expected exactly one real verifier subprocess, got %d", invoker.calls)
	}
	if _, err := Verify(context.Background(), host, invoker, root, runID); err == nil || !strings.Contains(err.Error(), StatusFailed) {
		t.Fatalf("failed verification replay was not rejected: %v", err)
	}
	if invoker.calls != 1 {
		t.Fatalf("retry bypassed the persisted verifier attempt budget: calls=%d", invoker.calls)
	}
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
