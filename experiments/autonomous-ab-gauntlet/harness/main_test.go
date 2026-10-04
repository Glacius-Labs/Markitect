package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

type harnessFixture struct {
	arena     string
	protocol  string
	project   string
	workspace string
	runID     string
	baseA     string
	baseB     string
}

func newHarnessFixture(t *testing.T, maximumRepairs int) harnessFixture {
	t.Helper()
	arena := t.TempDir()
	project := filepath.Join(arena, "projects", "demo")
	for _, dir := range []string{"arm-a", "arm-b"} {
		if err := os.MkdirAll(filepath.Join(project, dir), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, dir, "source.txt"), []byte("seed\r\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	protocol := filepath.Join(arena, "protocol.yaml")
	protocolData := fmt.Sprintf(`agent:
  model: test-model
  reasoning_effort: medium
execution:
  maximum_self_repair_iterations: %d
  task_timeout_seconds: 600
projects:
  - id: demo
    path: projects/demo
    arms:
      a:
        path: arm-a
        base_revision: HEAD
        validator:
          - [git, rev-parse, '${BASE_REVISION}']
      b:
        path: arm-b
        base_revision: HEAD
        validator:
          - [git, rev-parse, '${CANDIDATE_REVISION}']
        full_acceptance:
          - [git, rev-parse, '${BASE_REVISION}']
`, maximumRepairs)
	if err := os.WriteFile(protocol, []byte(protocolData), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := `id: demo
arms:
  a:
    path: arm-a
    base_revision: HEAD
    validator:
      - [git, rev-parse, '${BASE_REVISION}']
  b:
    path: arm-b
    base_revision: HEAD
    validator:
      - [git, rev-parse, '${CANDIDATE_REVISION}']
    full_acceptance:
      - [git, rev-parse, '${BASE_REVISION}']
`
	if err := os.WriteFile(filepath.Join(project, "project.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	tasks := `project: demo
tasks:
  - id: '01'
    prompt: Update the source for the first task.
    allowed_paths: [common]
    allowed_paths_by_arm:
      a: [src/a]
      b: [src/b]
    validator: [git, rev-parse, '${BASE_REVISION}']
  - id: '02'
    prompt: Continue with the next task.
    allowed_paths: [src]
    validator: [git, rev-parse, HEAD]
  - id: '09'
    prompt: Holdout task.
    allowed_paths: [src]
    validator: [git, rev-parse, HEAD]
`
	if err := os.WriteFile(filepath.Join(project, "task-set.yaml"), []byte(tasks), 0644); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return prepare([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return freeze([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	return harnessFixture{arena: arena, protocol: protocol, project: project,
		workspace: filepath.Join(arena, "snapshots", "demo-a-t01"), runID: "demo-a-t01"}
}

func (f harnessFixture) taskArgs(arm, task string, extra ...string) []string {
	args := []string{"--arena", f.arena, "--protocol", f.protocol, "--project", "demo", "--arm", arm, "--trial", "1", "--task-id", task}
	return append(args, extra...)
}

func prepareEnvelope(t *testing.T, args []string) NativeEnvelope {
	t.Helper()
	if err := quietCall(t, func() error { return prepareTask(args) }); err != nil {
		t.Fatal(err)
	}
	var arena, project, arm, taskID, attempt string
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "--arena":
			arena = args[i+1]
		case "--project":
			project = args[i+1]
		case "--arm":
			arm = strings.ToLower(strings.TrimPrefix(args[i+1], "arm-"))
		case "--task-id":
			taskID = args[i+1]
		case "--repair-iteration":
			attempt = args[i+1]
		}
	}
	if attempt == "" {
		attempt = "0"
	}
	runID := fmt.Sprintf("%s-%s-t01", project, arm)
	for _, arg := range args {
		if arg == "--parallel" {
			runID += "-parallel-" + taskID
			break
		}
	}
	path := filepath.Join(arena, "runs", runID, taskID+".attempt-"+attempt+".envelope.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope NativeEnvelope
	if err := yaml.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}

func finishFixtureTask(t *testing.T, f harnessFixture, env NativeEnvelope, status string, attempt int) error {
	t.Helper()
	response := filepath.Join(t.TempDir(), "response.txt")
	if err := os.WriteFile(response, []byte("actor response"), 0600); err != nil {
		t.Fatal(err)
	}
	return finishTask([]string{"--arena", f.arena, "--run-id", env.RunID, "--task-id", env.TaskID,
		"--input-digest", env.InputDigest, "--attempt", fmt.Sprint(attempt), "--response-file", response, "--status", status})
}

func TestFreezeIsWriteOnceAndRejectsChangedOrAddedInputs(t *testing.T) {
	f := newHarnessFixture(t, 0)
	if err := verifyArenaFreeze(f.arena); err != nil {
		t.Fatalf("fresh freeze should verify: %v", err)
	}
	if err := freeze([]string{"--arena", f.arena}); err == nil {
		t.Fatal("second freeze unexpectedly replaced the write-once record")
	}
	late := filepath.Join(f.arena, "late-input.yaml")
	if err := os.WriteFile(late, []byte("late"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyArenaFreeze(f.arena); err == nil || !strings.Contains(err.Error(), "new file after freeze") {
		t.Fatalf("expected added input rejection, got %v", err)
	}
	if err := os.Remove(late); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(f.project, "task-set.yaml")
	if err := os.WriteFile(input, []byte("project: changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyArenaFreeze(f.arena); err == nil || !strings.Contains(err.Error(), "frozen input changed") {
		t.Fatalf("expected changed input rejection, got %v", err)
	}
}

func TestTaskOrderCarriesFailedWorkspaceForwardAndCapturesSnapshot(t *testing.T) {
	f := newHarnessFixture(t, 0)
	if err := prepareTask(f.taskArgs("A", "02")); err == nil || !strings.Contains(err.Error(), "before task 01") {
		t.Fatalf("task order was not enforced: %v", err)
	}
	first := prepareEnvelope(t, f.taskArgs("A", "01"))
	if err := os.WriteFile(filepath.Join(first.Workspace, "source.txt"), []byte("changed by failed task\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := finishFixtureTask(t, f, first, "failed", 0); err != nil {
		t.Fatal(err)
	}
	second := prepareEnvelope(t, f.taskArgs("A", "02"))
	if second.Workspace != first.Workspace {
		t.Fatalf("task chain switched workspace: %q != %q", second.Workspace, first.Workspace)
	}
	got, err := os.ReadFile(filepath.Join(second.Workspace, "source.txt"))
	if err != nil || string(got) != "changed by failed task\r\n" {
		t.Fatalf("failed task state was not carried forward: %q, %v", got, err)
	}
	var state NativeTaskRecord
	if err := readYAML(filepath.Join(f.arena, "runs", f.runID, "01.native.yaml"), &state); err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(f.arena, state.FinalSnapshot, "tree.yaml")); statErr != nil {
		t.Fatalf("failed task snapshot missing: %v", statErr)
	}
}

func TestArmScopeValidatorsAndRevisionPlaceholdersReachPublicCheck(t *testing.T) {
	f := newHarnessFixture(t, 0)
	for _, arm := range []string{"A", "B"} {
		env := prepareEnvelope(t, f.taskArgs(arm, "01"))
		wantPath := "src/a"
		wantBase := "${BASE_REVISION}"
		if arm == "B" {
			wantPath = "src/b"
			wantBase = "${BASE_REVISION}"
			containsCandidate := false
			for _, validator := range env.PublicValidators {
				if strings.Contains(strings.Join(validator, " "), "${CANDIDATE_REVISION}") {
					containsCandidate = true
				}
			}
			if !containsCandidate {
				t.Fatalf("arm B full-acceptance validator missing: %#v", env.PublicValidators)
			}
		}
		if len(env.AllowedPaths) != 1 || env.AllowedPaths[0] != wantPath {
			t.Fatalf("arm %s received wrong path boundary: %v", arm, env.AllowedPaths)
		}
		foundBase := false
		for _, validator := range env.PublicValidators {
			if strings.Contains(strings.Join(validator, " "), wantBase) {
				foundBase = true
			}
		}
		if !foundBase {
			t.Fatalf("arm %s validators lost revision placeholder: %#v", arm, env.PublicValidators)
		}
		manifest := filepath.Join(f.arena, "raw", env.RunID, env.TaskID, "validators.yaml")
		if err := runPublicCheckForTest(t, f, env, manifest); err != nil {
			t.Fatalf("arm %s actual public validator: %v", arm, err)
		}
	}
	for _, runID := range []string{"demo-a-t01", "demo-b-t01"} {
		output, err := os.ReadFile(filepath.Join(f.arena, "raw", runID, "01", "attempt-0.public-validator.txt"))
		if err != nil {
			t.Fatal(err)
		}
		resolved := 0
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "resolved: ") {
				resolved++
				if strings.Contains(line, "${") {
					t.Fatalf("public check left arm %s revision placeholder unresolved: %s", runID, line)
				}
			}
		}
		if resolved == 0 {
			t.Fatalf("public check did not record resolved arm %s validators: %s", runID, output)
		}
		recordPath := filepath.Join(f.arena, "raw", runID, "01", "helper.jsonl")
		count, exitCode, _, attempt := helperResults(recordPath)
		if count != 1 || exitCode != 0 || attempt != 0 {
			t.Fatalf("arm %s helper receipt lost attempt/outcome: count=%d exit=%d attempt=%d", runID, count, exitCode, attempt)
		}
		var event helperEvent
		if err := json.Unmarshal(mustReadFile(t, recordPath), &event); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(event.Snapshot); err != nil {
			t.Fatalf("arm %s helper receipt references missing before-check snapshot %q: %v", runID, event.Snapshot, err)
		}
	}
}

func runPublicCheckForTest(t *testing.T, f harnessFixture, env NativeEnvelope, manifest string) error {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(env.Workspace); err != nil {
		return err
	}
	defer os.Chdir(oldDir)
	oldEnv := map[string]string{}
	for key, value := range map[string]string{
		"GAUNTLET_RUN_ID":                  env.RunID,
		"GAUNTLET_TASK_ID":                 env.TaskID,
		"GAUNTLET_HELPER_RECORD":           filepath.Join(f.arena, "raw", env.RunID, env.TaskID, "helper.jsonl"),
		"GAUNTLET_SNAPSHOT_ROOT":           filepath.Join(f.arena, "raw", env.RunID, env.TaskID, "snapshots"),
		"GAUNTLET_PUBLIC_VALIDATOR_OUTPUT": filepath.Join(f.arena, "raw", env.RunID, env.TaskID, "attempt-0.public-validator.txt"),
	} {
		oldEnv[key] = os.Getenv(key)
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	defer func() {
		for key, value := range oldEnv {
			_ = os.Setenv(key, value)
		}
	}()
	return quietCall(t, func() error { return publicCheck([]string{"--task-id", env.TaskID, "--manifest", manifest}) })
}

func runPublicCheckWithEnvelope(t *testing.T, env NativeEnvelope) error {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(env.Workspace); err != nil {
		return err
	}
	defer os.Chdir(oldDir)
	oldEnv := make(map[string]string, len(env.HelperEnvironment))
	for key, value := range env.HelperEnvironment {
		oldEnv[key] = os.Getenv(key)
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	defer func() {
		for key, value := range oldEnv {
			_ = os.Setenv(key, value)
		}
	}()
	return quietCall(t, func() error {
		return publicCheck([]string{"--task-id", env.TaskID, "--manifest", env.ValidatorManifest})
	})
}

func TestNativeRepairFeedbackDigestBindingRestartRefusalAndExhaustion(t *testing.T) {
	f := newHarnessFixture(t, 2)
	initial := prepareEnvelope(t, f.taskArgs("A", "01"))
	if err := finishFixtureTask(t, f, initial, "repair-needed", 0); err != nil {
		t.Fatal(err)
	}
	feedback := filepath.Join(f.arena, "raw", initial.RunID, initial.TaskID, "attempt-0.public-validator.txt")
	if err := os.WriteFile(feedback, []byte("exact failure output\n"), 0600); err != nil {
		t.Fatal(err)
	}
	repair1 := prepareEnvelope(t, f.taskArgs("A", "01", "--repair-iteration", "1", "--feedback-file", feedback))
	if repair1.InputDigest == initial.InputDigest || !strings.Contains(repair1.ActorPrompt, "exact failure output\n") {
		t.Fatalf("repair digest or public feedback not bound into fresh turn: digest=%s prompt=%s", repair1.InputDigest, repair1.ActorPrompt)
	}
	if repair1.DeadlineUTC == "" || repair1.DeadlineUTC != initial.DeadlineUTC {
		t.Fatalf("repair actor did not retain original task deadline: initial=%q repair=%q", initial.DeadlineUTC, repair1.DeadlineUTC)
	}
	if err := prepareTask(f.taskArgs("A", "01", "--repair-iteration", "1", "--feedback-file", feedback)); err == nil {
		t.Fatal("restarting an already active repair attempt was not refused")
	}
	wrong := repair1.InputDigest + "wrong"
	response := filepath.Join(t.TempDir(), "response.txt")
	if err := os.WriteFile(response, []byte("response"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := finishTask([]string{"--arena", f.arena, "--run-id", repair1.RunID, "--task-id", repair1.TaskID,
		"--input-digest", wrong, "--attempt", "1", "--response-file", response, "--status", "repair-needed"}); err == nil || !strings.Contains(err.Error(), "input digest does not match") {
		t.Fatalf("stale digest was accepted: %v", err)
	}
	if err := finishFixtureTask(t, f, repair1, "repair-needed", 1); err != nil {
		t.Fatal(err)
	}
	feedback2 := filepath.Join(f.arena, "raw", repair1.RunID, repair1.TaskID, "attempt-1.public-validator.txt")
	if err := os.WriteFile(feedback2, []byte("second exact failure\n"), 0600); err != nil {
		t.Fatal(err)
	}
	repair2 := prepareEnvelope(t, f.taskArgs("A", "01", "--repair-iteration", "2", "--feedback-file", feedback2))
	if repair2.InputDigest == repair1.InputDigest || !strings.Contains(repair2.ActorPrompt, "second exact failure\n") {
		t.Fatal("second repair turn did not bind its own feedback")
	}
	if err := finishFixtureTask(t, f, repair2, "failed", 2); err != nil {
		t.Fatal(err)
	}
	if err := prepareTask(f.taskArgs("A", "01", "--repair-iteration", "3", "--feedback-file", feedback2)); err == nil || !strings.Contains(err.Error(), "exceeds frozen protocol limit") {
		t.Fatalf("repair exhaustion was not enforced: %v", err)
	}
	if err := finishFixtureTask(t, f, repair2, "failed", 2); err == nil {
		t.Fatal("terminal task record was overwritten")
	}
}

func TestExpiredTaskDeadlineForcesTerminalFailure(t *testing.T) {
	f := newHarnessFixture(t, 2)
	env := prepareEnvelope(t, f.taskArgs("A", "01"))
	statePath := filepath.Join(f.arena, "runs", env.RunID, "01.native.yaml")
	var state NativeTaskRecord
	if err := readYAML(statePath, &state); err != nil {
		t.Fatal(err)
	}
	if state.DeadlineUTC == "" {
		t.Fatal("prepared task omitted its fixed deadline")
	}
	state.DeadlineUTC = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	stateBytes, err := yaml.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, stateBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := finishFixtureTask(t, f, env, "repair-needed", 0); err != nil {
		t.Fatal(err)
	}
	if err := readYAML(statePath, &state); err != nil {
		t.Fatal(err)
	}
	if !state.TimeoutExceeded || state.ActorStatus != "failed" || state.ActorCompletedUTC == "" {
		t.Fatalf("overdue repair-needed response did not become a terminal timeout: %#v", state)
	}
	feedback := filepath.Join(f.arena, "raw", env.RunID, env.TaskID, "attempt-0.public-validator.txt")
	if err := os.WriteFile(feedback, []byte("failure after deadline"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareTask(f.taskArgs("A", "01", "--repair-iteration", "1", "--feedback-file", feedback)); err == nil {
		t.Fatal("expired task deadline allowed a repair actor")
	}
}

func TestNativePreparationRequiresPositiveFrozenTaskTimeout(t *testing.T) {
	for _, timeoutField := range []string{"", "  task_timeout_seconds: 0\n"} {
		label := "missing"
		if timeoutField != "" {
			label = "zero"
		}
		t.Run(label, func(t *testing.T) {
			arena := t.TempDir()
			projectRoot := filepath.Join(arena, "projects", "demo")
			for _, arm := range []string{"arm-a", "arm-b"} {
				if err := os.MkdirAll(filepath.Join(projectRoot, arm), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectRoot, arm, "source.txt"), []byte("seed\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			protocol := filepath.Join(arena, "protocol.yaml")
			data := "agent:\n  model: test\nexecution:\n  maximum_self_repair_iterations: 0\n" + timeoutField + "projects: [demo]\n"
			if err := os.WriteFile(protocol, []byte(data), 0644); err != nil {
				t.Fatal(err)
			}
			projectManifest := "id: demo\narms:\n  a: {path: arm-a, base_revision: HEAD, validator: [[git, rev-parse, HEAD]]}\n"
			if err := os.WriteFile(filepath.Join(projectRoot, "project.yaml"), []byte(projectManifest), 0644); err != nil {
				t.Fatal(err)
			}
			taskSet := "project: demo\ntasks:\n  - id: '01'\n    prompt: timed work\n    allowed_paths: [src]\n    validator: [git, rev-parse, HEAD]\n"
			if err := os.WriteFile(filepath.Join(projectRoot, "task-set.yaml"), []byte(taskSet), 0644); err != nil {
				t.Fatal(err)
			}
			if err := quietCall(t, func() error { return prepare([]string{"--arena", arena}) }); err != nil {
				t.Fatal(err)
			}
			if err := quietCall(t, func() error { return freeze([]string{"--arena", arena}) }); err != nil {
				t.Fatal(err)
			}
			err := prepareTask([]string{"--arena", arena, "--protocol", protocol, "--project", "demo", "--arm", "A", "--trial", "1", "--task-id", "01"})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "task_timeout_seconds") {
				t.Fatalf("%s task timeout was accepted: %v", label, err)
			}
		})
	}
}

func TestParallelForkUsesCompletedMainSnapshotAndFrozenLineage(t *testing.T) {
	f := newParallelFixture(t)
	parallelArgs := f.taskArgs("A", "P01", "--parallel", "--task-set", "parallel-task-set.yaml", "--fork-from-task", "06")
	if err := prepareTask(parallelArgs); err == nil || !strings.Contains(err.Error(), "parallel fork requires completed task 06") {
		t.Fatalf("parallel fork started without a completed main snapshot: %v", err)
	}
	main := prepareEnvelope(t, f.taskArgs("A", "06"))
	if err := finishFixtureTask(t, f, main, "completed", 0); err != nil {
		t.Fatal(err)
	}
	parallel := prepareEnvelope(t, parallelArgs)
	if parallel.RunID != "demo-a-t01-parallel-P01" || parallel.Workspace == main.Workspace {
		t.Fatalf("parallel actor did not receive isolated run/workspace: run=%s workspace=%s", parallel.RunID, parallel.Workspace)
	}
	if len(parallel.AllowedPaths) != 1 || parallel.AllowedPaths[0] != "src/parallel" {
		t.Fatalf("parallel card path boundary was not loaded: %v", parallel.AllowedPaths)
	}
	var state NativeTaskRecord
	if err := readYAML(filepath.Join(f.arena, "runs", parallel.RunID, "P01.native.yaml"), &state); err != nil {
		t.Fatal(err)
	}
	var mainState NativeTaskRecord
	if err := readYAML(filepath.Join(f.arena, "runs", f.runID, "06.native.yaml"), &mainState); err != nil {
		t.Fatal(err)
	}
	mainSnapshot := filepath.Join(f.arena, mainState.FinalSnapshot, "workspace")
	base := gitOutput(t, mainSnapshot, "rev-parse", "HEAD")
	if !state.Parallel || state.PriorTasksThrough != "06" || state.BaseRevision != base || gitOutput(t, parallel.Workspace, "rev-parse", "HEAD") != base {
		t.Fatalf("parallel fork lineage did not match frozen main snapshot: state=%#v base=%s", state, base)
	}
}

func TestParallelIntegrationPreservesLexicalGitConflictAndInitialEvidence(t *testing.T) {
	result := createConflictIntegration(t)
	if result.integration.ActorStatus != "repair-needed" || result.integration.FinalSnapshot == "" {
		t.Fatalf("integration failure lacks terminal record/snapshot: %#v", result.integration)
	}
	for taskID, commit := range result.commits {
		if got := gitOutput(t, result.integration.Workspace, "cat-file", "-e", commit+"^{commit}"); got != "" {
			t.Fatalf("unexpected git cat-file output for %s: %s", taskID, got)
		}
	}
	status := gitOutput(t, result.integration.Workspace, "status", "--short")
	if !strings.Contains(status, "UU shared.txt") {
		history := gitOutput(t, result.integration.Workspace, "log", "--format=%B", result.integration.BaseRevision+"..HEAD")
		t.Fatalf("integration did not preserve the forced second-commit conflict: status=%q history=%q public feedback=%q", status, history, string(result.feedback))
	}
	history := gitOutput(t, result.integration.Workspace, "log", "--format=%B", result.integration.BaseRevision+"..HEAD")
	p01Trailer := "(cherry picked from commit " + result.commits["P01"] + ")"
	p02Trailer := "(cherry picked from commit " + result.commits["P02"] + ")"
	if !strings.Contains(history, p01Trailer) || strings.Contains(history, p02Trailer) {
		t.Fatalf("integration conflict did not leave exactly the lexical P01 commit applied and P02 pending: history=%q status=%q public feedback=%q", history, status, string(result.feedback))
	}
	if len(result.feedback) == 0 {
		t.Fatal("initial public integration failure output is empty")
	}
	if _, err := os.Stat(filepath.Join(result.f.arena, result.initialSnapshot, "workspace", ".git", "HEAD")); err != nil {
		t.Fatalf("initial conflict snapshot did not preserve Git state: %v", err)
	}
}

func TestIntegrationRepairPreservesFailureSnapshotAndEnforcesTwoFreshAttempts(t *testing.T) {
	result := createConflictIntegration(t)
	initialState := result.integration
	initialFeedback := append([]byte(nil), result.feedback...)
	initialTree := mustReadFile(t, filepath.Join(result.f.arena, result.initialSnapshot, "tree.yaml"))
	initialWorkspaceFile := mustReadFile(t, filepath.Join(result.f.arena, result.initialSnapshot, "workspace", "shared.txt"))
	initialGitIndex := mustReadFile(t, filepath.Join(result.f.arena, result.initialSnapshot, "workspace", ".git", "index"))
	integrationID := initialState.RunID
	feedback0 := filepath.Join(result.f.arena, "raw", integrationID, "08.attempt-0.feedback.txt")
	protocol := result.f.protocol
	prepare := func(attempt int, feedbackPath string) NativeEnvelope {
		t.Helper()
		args := []string{"--arena", result.f.arena, "--project", "demo", "--arm", "A", "--trial", "1", "--protocol", protocol,
			"--attempt", fmt.Sprint(attempt), "--feedback-file", feedbackPath}
		if err := quietCall(t, func() error { return prepareIntegrationRepair(args) }); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(result.f.arena, "runs", integrationID, fmt.Sprintf("08.attempt-%d.envelope.yaml", attempt))
		var env NativeEnvelope
		if err := readYAML(path, &env); err != nil {
			t.Fatal(err)
		}
		return env
	}
	finishRepair := func(env NativeEnvelope, status string, attempt int) error {
		t.Helper()
		response := filepath.Join(t.TempDir(), "response.txt")
		if err := os.WriteFile(response, []byte("repair actor response"), 0600); err != nil {
			return err
		}
		return finishTask([]string{"--arena", result.f.arena, "--run-id", integrationID, "--task-id", "08", "--input-digest", env.InputDigest,
			"--attempt", fmt.Sprint(attempt), "--response-file", response, "--status", status})
	}

	tampered := filepath.Join(t.TempDir(), "tampered-feedback.txt")
	if err := os.WriteFile(tampered, append(initialFeedback, []byte("tampered")...), 0600); err != nil {
		t.Fatal(err)
	}
	badArgs := []string{"--arena", result.f.arena, "--project", "demo", "--arm", "A", "--trial", "1", "--protocol", protocol,
		"--attempt", "1", "--feedback-file", tampered}
	if err := prepareIntegrationRepair(badArgs); err == nil || !strings.Contains(err.Error(), "byte-match") {
		t.Fatalf("integration repair accepted altered feedback bytes: %v", err)
	}

	attempt1 := prepare(1, feedback0)
	if attempt1.InputDigest == initialState.InputDigest || attempt1.DeadlineUTC != initialState.DeadlineUTC {
		t.Fatalf("repair digest/deadline did not bind the current failed integration: initial=%#v repair=%#v", initialState, attempt1)
	}
	if want := expectedIntegrationRepairDigest(t, result, initialState, initialFeedback, 1); attempt1.InputDigest != want {
		t.Fatalf("repair digest does not bind the current workspace, saved feedback, frozen plan, protocol, and attempt: got %s want %s", attempt1.InputDigest, want)
	}
	if !strings.Contains(attempt1.ActorPrompt, string(initialFeedback)) {
		t.Fatal("first repair prompt did not contain the exact saved initial integration feedback")
	}
	if len(attempt1.IntegrationTaskIDs) != 2 || strings.Join(attempt1.IntegrationTaskIDs, ",") != "P01,P02" {
		t.Fatalf("repair envelope omitted the frozen integration task set: %v", attempt1.IntegrationTaskIDs)
	}
	if !strings.Contains(attempt1.ActorPrompt, "P01: Write the P01 value") || !strings.Contains(attempt1.ActorPrompt, "P02: Write the P02 value") {
		t.Fatalf("repair prompt omitted the two original task requests: %s", attempt1.ActorPrompt)
	}
	if len(attempt1.AllowedPaths) != 1 || attempt1.AllowedPaths[0] != "shared.txt" {
		t.Fatalf("repair envelope did not use the exact union of same-arm scopes: %v", attempt1.AllowedPaths)
	}
	if len(attempt1.IntegrationPlan) != 2 || attempt1.IntegrationPlan[0].TaskID != "P01" || attempt1.IntegrationPlan[1].TaskID != "P02" {
		t.Fatalf("repair envelope lost the frozen lexical commit plan: %#v", attempt1.IntegrationPlan)
	}
	if len(attempt1.IntegrationRemainingCommits) != 1 || attempt1.IntegrationRemainingCommits[0].TaskID != "P02" || attempt1.IntegrationRemainingCommits[0].Commit != result.commits["P02"] {
		t.Fatalf("repair envelope did not show only the remaining unmerged commit: %#v", attempt1.IntegrationRemainingCommits)
	}
	for key, want := range map[string]string{
		"GAUNTLET_ACTOR_ATTEMPT":           "2",
		"GAUNTLET_HELPER_RECORD":           filepath.Join(result.f.arena, "raw", integrationID, "08.attempt-1.helper.jsonl"),
		"GAUNTLET_SNAPSHOT_ROOT":           filepath.Join(result.f.arena, "raw", integrationID, "snapshots", "attempt-1"),
		"GAUNTLET_PUBLIC_VALIDATOR_OUTPUT": filepath.Join(result.f.arena, "raw", integrationID, "08.attempt-1.public-validator.txt"),
	} {
		if attempt1.HelperEnvironment[key] != want {
			t.Fatalf("attempt 1 helper environment %s=%q, want %q", key, attempt1.HelperEnvironment[key], want)
		}
	}
	if err := runPublicCheckWithEnvelope(t, attempt1); err == nil {
		t.Fatal("the unresolved Git conflict unexpectedly passed the combined public validator")
	}
	output1 := mustReadFile(t, attempt1.HelperEnvironment["GAUNTLET_PUBLIC_VALIDATOR_OUTPUT"])
	if err := finishRepair(attempt1, "repair-needed", 1); err != nil {
		t.Fatal(err)
	}
	feedback1 := mustReadFile(t, filepath.Join(result.f.arena, "raw", integrationID, "08.attempt-1.feedback.txt"))
	if !bytes.Equal(feedback1, output1) {
		t.Fatal("repair 1 did not save the exact public helper output as next-attempt feedback")
	}

	var afterAttempt1 NativeTaskRecord
	if err := readYAML(filepath.Join(result.f.arena, "runs", integrationID, "08.native.yaml"), &afterAttempt1); err != nil {
		t.Fatal(err)
	}
	if afterAttempt1.ActiveAttempt != 1 || afterAttempt1.LastTurnDigest != attempt1.InputDigest || afterAttempt1.ActorStatus != "repair-needed" {
		t.Fatalf("repair 1 did not remain the active preserved state before preparing repair 2: %#v", afterAttempt1)
	}
	attempt2 := prepare(2, filepath.Join(result.f.arena, "raw", integrationID, "08.attempt-1.feedback.txt"))
	if attempt2.DeadlineUTC != initialState.DeadlineUTC || !strings.Contains(attempt2.ActorPrompt, string(feedback1)) {
		t.Fatal("repair 2 did not retain the task deadline and exact repair-1 public output")
	}
	if want := expectedIntegrationRepairDigest(t, result, afterAttempt1, feedback1, 2); attempt2.InputDigest != want {
		workspaceDigest, _ := treeDigest(afterAttempt1.Workspace)
		feedbackHash := sha256.Sum256(feedback1)
		protocolHash, _ := hashFile(result.f.protocol)
		planBytes := mustReadFile(t, filepath.Join(result.f.arena, filepath.FromSlash(afterAttempt1.IntegrationPlanPath)))
		planHash := sha256.Sum256(planBytes)
		t.Fatalf("repair 2 digest omitted current integration state or prior public output: got %s want %s (last_turn=%s workspace=%s feedback=%s plan=%s protocol=%s active_attempt=%d)", attempt2.InputDigest, want, afterAttempt1.LastTurnDigest, workspaceDigest, hex.EncodeToString(feedbackHash[:]), hex.EncodeToString(planHash[:]), protocolHash, afterAttempt1.ActiveAttempt)
	}
	if len(attempt2.IntegrationRemainingCommits) != 1 || attempt2.IntegrationRemainingCommits[0].TaskID != "P02" {
		t.Fatalf("repair 2 lost the still-pending P02 commit: %#v", attempt2.IntegrationRemainingCommits)
	}
	if err := runPublicCheckWithEnvelope(t, attempt2); err == nil {
		t.Fatal("the unresolved Git conflict unexpectedly passed the second public validator")
	}
	if err := finishRepair(attempt2, "failed", 2); err != nil {
		t.Fatal(err)
	}
	thirdArgs := []string{"--arena", result.f.arena, "--project", "demo", "--arm", "A", "--trial", "1", "--protocol", protocol,
		"--attempt", "3", "--feedback-file", filepath.Join(result.f.arena, "raw", integrationID, "08.attempt-2.public-validator.txt")}
	if err := prepareIntegrationRepair(thirdArgs); err == nil || !strings.Contains(err.Error(), "attempt must be 1 or 2") {
		t.Fatalf("third integration repair exceeded the frozen two-attempt budget: %v", err)
	}
	var finalState NativeTaskRecord
	if err := readYAML(filepath.Join(result.f.arena, "runs", integrationID, "08.native.yaml"), &finalState); err != nil {
		t.Fatal(err)
	}
	if finalState.ActorStatus != "failed" || finalState.ActorCompletedUTC == "" || len(finalState.Turns) != 3 || finalState.Turns[0] != initialState.Turns[0] {
		t.Fatalf("integration repair history did not preserve the terminal record and original turn: %#v", finalState)
	}
	if finalState.InitialSnapshot != initialState.InitialSnapshot || finalState.FinalSnapshot == finalState.InitialSnapshot {
		t.Fatalf("terminal repair overwrote or lost the initial integration snapshot: initial=%#v final=%#v", initialState, finalState)
	}
	if got := mustReadFile(t, filepath.Join(result.f.arena, finalState.InitialSnapshot, "tree.yaml")); !bytes.Equal(got, initialTree) {
		t.Fatal("initial integration snapshot manifest changed during repairs")
	}
	if got := mustReadFile(t, filepath.Join(result.f.arena, finalState.InitialSnapshot, "workspace", "shared.txt")); !bytes.Equal(got, initialWorkspaceFile) {
		t.Fatal("initial integration conflict workspace changed during repairs")
	}
	if got := mustReadFile(t, filepath.Join(result.f.arena, finalState.InitialSnapshot, "workspace", ".git", "index")); !bytes.Equal(got, initialGitIndex) {
		t.Fatal("initial integration Git index changed during repairs")
	}
	if got := mustReadFile(t, filepath.Join(result.f.arena, "raw", integrationID, "08.attempt-0.feedback.txt")); !bytes.Equal(got, initialFeedback) {
		t.Fatal("initial integration feedback bytes changed during repairs")
	}
}

type conflictIntegrationResult struct {
	f               harnessFixture
	integration     NativeTaskRecord
	commits         map[string]string
	feedback        []byte
	initialSnapshot string
}

func createConflictIntegration(t *testing.T) conflictIntegrationResult {
	t.Helper()
	// The integration workspace is a materialized Git snapshot, so it must not
	// depend on this machine's global/system committer identity. Parallel actor
	// commits set identity explicitly in gitCommit; the harness must do the same
	// for its mechanical cherry-picks.
	emptyGitConfig := filepath.Join(t.TempDir(), "empty-git-config")
	if err := os.WriteFile(emptyGitConfig, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", emptyGitConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	f := newConflictIntegrationFixture(t)
	main := prepareEnvelope(t, f.taskArgs("A", "06"))
	if err := finishFixtureTask(t, f, main, "completed", 0); err != nil {
		t.Fatal(err)
	}
	commits := map[string]string{}
	for _, taskID := range []string{"P02", "P01"} {
		args := f.taskArgs("A", taskID, "--parallel", "--task-set", "parallel-task-set.yaml", "--fork-from-task", "06")
		env := prepareEnvelope(t, args)
		if err := os.WriteFile(filepath.Join(env.Workspace, "shared.txt"), []byte(taskID+" change\n"), 0644); err != nil {
			t.Fatal(err)
		}
		gitCommit(t, env.Workspace, "parallel "+taskID)
		commits[taskID] = gitOutput(t, env.Workspace, "rev-parse", "HEAD")
		if err := finishFixtureTask(t, f, env, "completed", 0); err != nil {
			t.Fatal(err)
		}
	}
	args := []string{"--arena", f.arena, "--project", "demo", "--arm", "A", "--trial", "1", "--protocol", f.protocol, "--main-task-id", "06", "--parallel-task-ids", "P02,P01"}
	err := quietCall(t, func() error { return integrateParallel(args) })
	if err == nil || !strings.Contains(err.Error(), "preserved with failed status") {
		t.Fatalf("conflicting lexical integration did not preserve a failed record: %v", err)
	}
	integrationID := "demo-a-t01-integration"
	var integration NativeTaskRecord
	if err := readYAML(filepath.Join(f.arena, "runs", integrationID, "08.native.yaml"), &integration); err != nil {
		t.Fatal(err)
	}
	initialSnapshot := integration.InitialSnapshot
	if initialSnapshot == "" {
		initialSnapshot = integration.FinalSnapshot
	}
	feedbackPath := filepath.Join(f.arena, "raw", integrationID, "08.attempt-0.feedback.txt")
	feedback, err := os.ReadFile(feedbackPath)
	if err != nil {
		t.Fatalf("initial public integration failure output missing: %v", err)
	}
	return conflictIntegrationResult{f: f, integration: integration, commits: commits, feedback: feedback, initialSnapshot: initialSnapshot}
}

func expectedIntegrationRepairDigest(t *testing.T, result conflictIntegrationResult, state NativeTaskRecord, feedback []byte, attempt int) string {
	t.Helper()
	workspaceDigest, err := treeDigest(state.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	feedbackHash := sha256.Sum256(feedback)
	protocolHash, err := hashFile(result.f.protocol)
	if err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(result.f.arena, filepath.FromSlash(state.IntegrationPlanPath))
	planBytes := mustReadFile(t, planPath)
	planHash := sha256.Sum256(planBytes)
	combined := sha256.Sum256([]byte(state.LastTurnDigest + "\x00" + workspaceDigest + "\x00" + hex.EncodeToString(feedbackHash[:]) + "\x00" + hex.EncodeToString(planHash[:]) + "\x00" + protocolHash + "\x00" + fmt.Sprint(attempt)))
	return hex.EncodeToString(combined[:])
}

func TestHoldout09BlockedBeforeDecisionAndOpensAfterRecordedDecision(t *testing.T) {
	f := newHarnessFixture(t, 0)
	if err := prepareTask(f.taskArgs("A", "09")); err == nil || !strings.Contains(err.Error(), "remain sealed") {
		t.Fatalf("holdout task was not blocked before decision: %v", err)
	}
	for _, taskID := range []string{"01", "02"} {
		env := prepareEnvelope(t, f.taskArgs("A", taskID))
		if err := finishFixtureTask(t, f, env, "completed", 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := recordHoldoutDecision([]string{"--arena", f.arena, "--decision", "fix", "--candidate", "abc123", "--rationale", "public tasks reviewed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.arena, "runs", f.runID, "09.native.yaml")); !os.IsNotExist(err) {
		t.Fatalf("blocked holdout created a task record early: %v", err)
	}
	prepareEnvelope(t, f.taskArgs("A", "09"))
}

func TestSeedUsesCodexBranchAndPreservesLocalNewlines(t *testing.T) {
	f := newHarnessFixture(t, 0)
	env := prepareEnvelope(t, f.taskArgs("A", "01"))
	branch := gitOutput(t, env.Workspace, "branch", "--show-current")
	if !strings.HasPrefix(branch, "codex/") || branch == "main" || branch == "master" {
		t.Fatalf("seed initialized on unexpected branch %q", branch)
	}
	config := gitOutput(t, env.Workspace, "config", "--local", "core.autocrlf")
	if config != "false" {
		t.Fatalf("seed core.autocrlf = %q, want false", config)
	}
	data, err := os.ReadFile(filepath.Join(env.Workspace, "source.txt"))
	if err != nil || string(data) != "seed\r\n" {
		t.Fatalf("seed newline bytes changed: %q, %v", data, err)
	}
}

func TestSnapshotKeepsGitBaseButDigestExcludesMetadataAndBuildTrees(t *testing.T) {
	f := newHarnessFixture(t, 0)
	env := prepareEnvelope(t, f.taskArgs("A", "01"))
	if err := os.WriteFile(filepath.Join(env.Workspace, ".gitignore"), []byte("ignored-source.txt\nsrc/commerce/bin/\nchecks/obj/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.Workspace, "ignored-source.txt"), []byte("ignored source is still input"), 0644); err != nil {
		t.Fatal(err)
	}
	generatedBin := filepath.Join(env.Workspace, "src", "commerce", "bin")
	if err := os.MkdirAll(generatedBin, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generatedBin, "generated.dll"), []byte("generated"), 0644); err != nil {
		t.Fatal(err)
	}
	generatedObj := filepath.Join(env.Workspace, "checks", "obj")
	if err := os.MkdirAll(generatedObj, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generatedObj, "project.assets.json"), []byte("generated"), 0644); err != nil {
		t.Fatal(err)
	}
	base := gitOutput(t, env.Workspace, "rev-parse", "HEAD")
	if err := finishFixtureTask(t, f, env, "completed", 0); err != nil {
		t.Fatal(err)
	}
	var state NativeTaskRecord
	if err := readYAML(filepath.Join(f.arena, "runs", f.runID, "01.native.yaml"), &state); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(f.arena, state.FinalSnapshot, "workspace")
	if got := gitOutput(t, snapshot, "rev-parse", "HEAD"); got != base {
		t.Fatalf("snapshot Git base changed: %s != %s", got, base)
	}
	manifestData, err := os.ReadFile(filepath.Join(f.arena, state.FinalSnapshot, "tree.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]string
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if _, ok := manifest[".git/HEAD"]; ok {
		t.Fatal("Git metadata leaked into snapshot digest")
	}
	if _, ok := manifest["src/commerce/bin/generated.dll"]; ok {
		t.Fatalf("generated build output leaked into snapshot digest: %#v", manifest)
	}
	if _, ok := manifest["checks/obj/project.assets.json"]; ok {
		t.Fatalf("generated restore metadata leaked into snapshot digest: %#v", manifest)
	}
	if _, ok := manifest["ignored-source.txt"]; !ok {
		t.Fatal("ordinary Git-ignored source input was omitted from snapshot digest")
	}
	if _, err := os.Stat(filepath.Join(snapshot, "src", "commerce", "bin", "generated.dll")); !os.IsNotExist(err) {
		t.Fatalf("build output leaked into final workspace snapshot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(snapshot, "ignored-source.txt")); err != nil {
		t.Fatalf("ordinary ignored source was not preserved for evaluation: %v", err)
	}
}

func TestNativeEnvelopeDoesNotExposeEvaluationOracle(t *testing.T) {
	envelope := NativeEnvelope{Prompt: "public request", ActorPrompt: "public request", AllowedPaths: []string{"src"}}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"expected_classification", "expected_affected", "evaluator", "oracle"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("native envelope exposed %q", forbidden)
		}
	}
}

func TestCheckedInProjectTaskSetsProduceScopedNativeEnvelopes(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate checked-in fixture projects")
	}
	fixtureRoot := filepath.Join(filepath.Dir(sourceFile), "..", "projects")
	arena := t.TempDir()
	for _, project := range []string{"vertical-slices", "modular-service", "engineering-ops"} {
		source := filepath.Join(fixtureRoot, project)
		dest := filepath.Join(arena, "projects", project)
		if err := copyTree(source, dest); err != nil {
			t.Fatalf("copy checked-in %s project: %v", project, err)
		}
	}
	protocol := filepath.Join(arena, "protocol.yaml")
	if err := os.WriteFile(protocol, []byte(`agent:
  model: test-model
  reasoning_effort: medium
execution:
  maximum_self_repair_iterations: 2
  task_timeout_seconds: 600
projects: [vertical-slices, modular-service, engineering-ops]
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return prepare([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return freeze([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	for _, project := range []string{"vertical-slices", "modular-service", "engineering-ops"} {
		for _, arm := range []string{"A", "B"} {
			runID := fmt.Sprintf("%s-%s-t01", project, strings.ToLower(arm))
			args := []string{"--arena", arena, "--protocol", protocol, "--project", project, "--arm", arm,
				"--trial", "1", "--task-id", "01"}
			env := prepareEnvelope(t, args)
			if env.RunID != runID || len(env.AllowedPaths) == 0 {
				t.Fatalf("%s arm %s did not load task 01 path scope: run=%s paths=%v", project, arm, env.RunID, env.AllowedPaths)
			}
			if len(env.PublicValidators) == 0 {
				t.Fatalf("%s arm %s resolved no public validator commands", project, arm)
			}
			for index, command := range env.PublicValidators {
				if len(command) == 0 || command[0] == "" {
					t.Fatalf("%s arm %s validator %d is empty: %#v", project, arm, index, command)
				}
			}
			manifestPath := filepath.Join(arena, "raw", runID, "01", "validators.yaml")
			var manifest ValidatorManifest
			if err := readYAML(manifestPath, &manifest); err != nil {
				t.Fatalf("%s arm %s validator manifest: %v", project, arm, err)
			}
			if len(manifest.Commands) != len(env.PublicValidators) {
				t.Fatalf("%s arm %s envelope/manifest validator count differs: %d != %d", project, arm, len(env.PublicValidators), len(manifest.Commands))
			}
			for i := range manifest.Commands {
				if strings.Join(manifest.Commands[i], "\x00") != strings.Join(env.PublicValidators[i], "\x00") {
					t.Fatalf("%s arm %s validator %d differs between envelope and public manifest", project, arm, i)
				}
			}
		}
	}
}

func TestConfiguredProjectAndTaskSetErrorsAreNotSilentlyIgnored(t *testing.T) {
	t.Run("malformed project manifest", func(t *testing.T) {
		f := newMalformedFixture(t, "id: demo\narms: [broken\n", "project: demo\ntasks: []\n")
		err := prepareTask(f.taskArgs("A", "01"))
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "project manifest") {
			t.Fatalf("malformed existing project manifest was ignored: %v", err)
		}
	})
	t.Run("malformed configured task set", func(t *testing.T) {
		f := newMalformedFixture(t, "id: demo\ncontrols:\n  task_set: task-set.yaml\narms:\n  a: {path: arm-a, base_revision: HEAD, validator: [[git, rev-parse, HEAD]]}\n", "project: demo\ntasks: [broken\n")
		err := prepareTask(f.taskArgs("A", "01"))
		if err == nil {
			t.Fatal("malformed configured task set was accepted")
		}
	})
	t.Run("missing configured path does not fall back", func(t *testing.T) {
		f := newMalformedFixture(t, "id: demo\ncontrols:\n  task_set: missing-task-set.yaml\narms:\n  a: {path: arm-a, base_revision: HEAD, validator: [[git, rev-parse, HEAD]]}\n", "project: demo\ntasks:\n  - id: '01'\n    prompt: fallback must not load\n    allowed_paths: [src]\n    validator: [git, rev-parse, HEAD]\n")
		err := prepareTask(f.taskArgs("A", "01"))
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "missing-task-set.yaml") {
			t.Fatalf("missing configured task-set path silently fell back to a default: %v", err)
		}
	})
}

func newMalformedFixture(t *testing.T, projectManifest, taskSet string) harnessFixture {
	t.Helper()
	arena := t.TempDir()
	projectRoot := filepath.Join(arena, "projects", "demo")
	for _, arm := range []string{"arm-a", "arm-b"} {
		if err := os.MkdirAll(filepath.Join(projectRoot, arm), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projectRoot, arm, "source.txt"), []byte("seed\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	protocol := filepath.Join(arena, "protocol.yaml")
	if err := os.WriteFile(protocol, []byte("agent:\n  model: test\nexecution:\n  maximum_self_repair_iterations: 0\n  task_timeout_seconds: 600\nprojects: [demo]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "project.yaml"), []byte(projectManifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "task-set.yaml"), []byte(taskSet), 0644); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return prepare([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return freeze([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	return harnessFixture{arena: arena, protocol: protocol, project: projectRoot, runID: "demo-a-t01"}
}

func newParallelFixture(t *testing.T) harnessFixture {
	t.Helper()
	arena := t.TempDir()
	projectRoot := filepath.Join(arena, "projects", "demo")
	if err := os.MkdirAll(filepath.Join(projectRoot, "arm-a", "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "arm-a", "src", "base.txt"), []byte("main snapshot\n"), 0644); err != nil {
		t.Fatal(err)
	}
	protocol := filepath.Join(arena, "protocol.yaml")
	if err := os.WriteFile(protocol, []byte(`agent:
  model: test-model
  reasoning_effort: medium
execution:
  maximum_self_repair_iterations: 2
  task_timeout_seconds: 600
projects: [demo]
`), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := `id: demo
controls:
  task_set: task-set.yaml
arms:
  a:
    path: arm-a
    validator:
      - [git, rev-parse, HEAD]
`
	if err := os.WriteFile(filepath.Join(projectRoot, "project.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	mainTasks := `project: demo
tasks:
  - id: '06'
    prompt: Record the main fork base.
    allowed_paths: [src]
    validator: [git, rev-parse, HEAD]
  - id: '08'
    prompt: Complete combined validation.
    allowed_paths: [src]
    validator: [git, rev-parse, HEAD]
`
	parallelTasks := `project: demo
fork_from:
  main_task_id: '06'
  main_trial: 1
  prior_tasks_through: '06'
tasks:
  - id: P01
    prompt: Change a parallel path.
    allowed_paths_by_arm:
      a: [src/parallel]
    origin_task_id: '06'
    prior_tasks_through: '06'
    validator: [git, rev-parse, HEAD]
`
	for path, data := range map[string]string{"task-set.yaml": mainTasks, "parallel-task-set.yaml": parallelTasks} {
		if err := os.WriteFile(filepath.Join(projectRoot, path), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := quietCall(t, func() error { return prepare([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return freeze([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	return harnessFixture{arena: arena, protocol: protocol, project: projectRoot, runID: "demo-a-t01"}
}

func newConflictIntegrationFixture(t *testing.T) harnessFixture {
	t.Helper()
	arena := t.TempDir()
	projectRoot := filepath.Join(arena, "projects", "demo")
	for _, arm := range []string{"arm-a", "arm-b"} {
		if err := os.MkdirAll(filepath.Join(projectRoot, arm), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projectRoot, arm, "shared.txt"), []byte("base state\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	conflictValidator := `import subprocess
result = subprocess.run(["git", "diff", "--name-only", "--diff-filter=U"], check=True, capture_output=True, text=True)
print(result.stdout, end="")
raise SystemExit(1 if result.stdout.strip() else 0)
`
	if err := os.WriteFile(filepath.Join(projectRoot, "arm-a", "check-no-conflicts.py"), []byte(conflictValidator), 0644); err != nil {
		t.Fatal(err)
	}
	protocol := filepath.Join(arena, "protocol.yaml")
	if err := os.WriteFile(protocol, []byte(`agent:
  model: test-model
  reasoning_effort: medium
execution:
  maximum_self_repair_iterations: 2
  task_timeout_seconds: 600
projects: [demo]
`), 0644); err != nil {
		t.Fatal(err)
	}
	projectManifest := `id: demo
controls:
  task_set: task-set.yaml
arms:
  a:
    path: arm-a
    validator:
      - [python, check-no-conflicts.py]
`
	if err := os.WriteFile(filepath.Join(projectRoot, "project.yaml"), []byte(projectManifest), 0644); err != nil {
		t.Fatal(err)
	}
	mainTasks := `project: demo
tasks:
  - id: '06'
    prompt: Main-chain fork base.
    allowed_paths: [shared.txt]
    validator: [git, diff, --check]
  - id: '08'
    prompt: Run combined public validation after importing parallel changes.
    allowed_paths: [shared.txt]
    validator: [git, diff, --exit-code]
`
	parallelTasks := `project: demo
fork_from:
  main_task_id: '06'
  main_trial: 1
  prior_tasks_through: '06'
tasks:
  - id: P01
    prompt: Write the P01 value to the shared file.
    allowed_paths: [shared.txt]
    origin_task_id: '06'
    prior_tasks_through: '06'
    validator: [git, diff, --check]
  - id: P02
    prompt: Write the P02 value to the same shared file.
    allowed_paths: [shared.txt]
    origin_task_id: '06'
    prior_tasks_through: '06'
    validator: [git, diff, --check]
`
	for path, data := range map[string]string{"task-set.yaml": mainTasks, "parallel-task-set.yaml": parallelTasks} {
		if err := os.WriteFile(filepath.Join(projectRoot, path), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := buildGauntletHelper(t, arena); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return prepare([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	if err := quietCall(t, func() error { return freeze([]string{"--arena", arena}) }); err != nil {
		t.Fatal(err)
	}
	return harnessFixture{arena: arena, protocol: protocol, project: projectRoot, runID: "demo-a-t01"}
}

func buildGauntletHelper(t *testing.T, arena string) error {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("could not locate harness source directory")
	}
	binDir := filepath.Join(arena, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	name := "gauntlet"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, name), ".")
	cmd.Dir = filepath.Dir(sourceFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build local gauntlet helper: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func gitCommit(t *testing.T, workspace, message string) {
	t.Helper()
	cmd := exec.Command("git", "add", "--", "shared.txt")
	cmd.Dir = workspace
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, output)
	}
	cmd = exec.Command("git", "-c", "user.name=Gauntlet Test", "-c", "user.email=gauntlet-test@invalid", "commit", "-m", message)
	cmd.Dir = workspace
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, output)
	}
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func quietCall(t *testing.T, action func() error) error {
	t.Helper()
	previous := os.Stdout
	discard, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	os.Stdout = discard
	defer func() {
		os.Stdout = previous
		_ = discard.Close()
	}()
	return action()
}
