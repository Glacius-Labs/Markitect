//go:build !runtime_unit

package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
	yaml "go.yaml.in/yaml/v3"
)

func TestFailedVerifierAttemptIsDurableAndCannotReplayBudget(t *testing.T) {
	t.Parallel()
	repository := testkit.NewRepo(t)
	root := repository.Dir
	if err := os.WriteFile(filepath.Join(root, "fixture.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	repository.Commit("fixture")
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
	setRuntimeFixtureEnv(t, verifierProcessHelperEnv, "failed-verifier")
	agent := projectconfig.Agent{
		Command: executable, Args: runtimeFixtureArgs(t, []string{"-test.run=^TestProjectRunVerifierHelperProcess$"}), Model: "fixture",
		ProviderVersion: "fixture-v1", Timeout: projectconfig.Duration(30 * time.Second), MaxStdoutBytes: 64 << 10, MaxStderrBytes: 16 << 10,
		Environment: []string{verifierProcessHelperEnv, "SystemRoot"}, Pricing: projectconfig.Pricing{InputMicrosPerMillion: 1000},
	}
	runtime := validRuntime()
	runtime.Limits.MaxStarts = 2 // one executable check and one verifier process
	runtime.Limits.MaxCostMicros = 1
	checkAgent := runtime.Agents["commerce"]
	checkAgent.Command = executable
	checkAgent.Args = []string{"-test.run=^TestProjectRunVerifierHelperProcess$"}
	// Only the verifier receives the failed-response fixture descriptor. The
	// check's allowlist intentionally excludes it; when SystemRoot is absent,
	// this also exercises an allowlist with no inherited values.
	checkAgent.Environment = []string{"SystemRoot"}
	checkAgent.ProviderVersion = "fixture-v1"
	checkAgent.Timeout = projectconfig.Duration(30 * time.Second)
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
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(projectconfig.RuntimePath)), runtimeData, 0o600); err != nil {
		t.Fatal(err)
	}

	snapshotBase := &snapshot.Snapshot{ID: head, Files: map[string][]byte{"fixture.txt": []byte("base")}, Modes: map[string]string{"fixture.txt": "100644"}}
	modelDigest := "sha256:" + strings.Repeat("a", 64)
	project := &projectvalue.Project{Root: root, Revision: head, Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Report: projectmodel.Report{
		ModelDigest: modelDigest, Digest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Status: "complete",
		Artifacts: []projectmodel.Artifact{}, Findings: []projectmodel.Finding{},
	}, Snapshot: snapshotBase}
	host := Host{
		Load: func(_, _ string) (*projectvalue.Project, error) { return project, nil },
		FromSnapshot: func(_ string, snap *snapshot.Snapshot) (*projectvalue.Project, error) {
			copy := *project
			copy.Snapshot = snap
			return &copy, nil
		},
		ClassifyCandidate: func(base, candidate *projectvalue.Project, _ ...string) (*projectvalue.Project, error) {
			if base == nil || base.Revision != head || candidate == nil || candidate.Revision != head || candidate.Snapshot == nil {
				return nil, fmt.Errorf("verification-budget fixture received an unbound candidate")
			}
			return candidate, nil
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
		RuntimeBindingVersion: methodRuntimeBindingVersion,
		Checks:                []CheckPlan{check}, InitialCandidateID: candidateID,
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
		Mode: projectconfig.ModeControlledLocal, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		BaseRevision: head, BaseSnapshot: snapshotBase.Digest(), ModelDigest: project.Report.ModelDigest,
		RuntimeDigest: runtimeDigest, Candidate: candidateRef(candidate, true), Tasks: []ManagerTask{},
		Invocations: []InvocationLog{}, Checks: []CheckResult{},
	}
	if err := persistState(store, &run); err != nil {
		t.Fatal(err)
	}
	loadedRuntime, err := projectconfig.LoadRuntime(root)
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
	if _, err := Verify(context.Background(), host, invoker, root, runID); err == nil || !strings.Contains(err.Error(), "no exact pending native verifier to recover") {
		t.Fatalf("failed verification replay was not rejected: %v", err)
	}
	if invoker.calls != 1 {
		t.Fatalf("retry bypassed the persisted verifier attempt budget: calls=%d", invoker.calls)
	}
}
