package projectrun

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

const (
	checkDescendantHelperEnv = "MARKITECT_PROJECTRUN_CHECK_DESCENDANT"
	modelledFileCheckEnv     = "MARKITECT_PROJECTRUN_CHECK_MODELLED_FILE"
	legacyOrdersBytes        = "legacy orders bytes\n"
)

// A candidate may model a transitional file in place: it drops the exclusion
// and leaves the bytes unchanged. Closure reads those bytes from the base, so
// the declared checks Verify runs must see them too, and so must full
// verification when it runs checks itself.
func TestVerifyChecksSeeFilesModelledInPlace(t *testing.T) {
	root := makeFullVerifyFixture(t)
	configureOperationsFullVerify(t, root)
	t.Setenv(modelledFileCheckEnv, "1")
	updateE2ERuntime(t, root, func(config *Runtime) {
		for id, agent := range config.Agents {
			agent.Environment = append(agent.Environment, modelledFileCheckEnv)
			config.Agents[id] = agent
		}
	})
	manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	transitional := strings.Replace(string(manifest), "exclusions: []\n", "exclusions: []\ntransitionalExclusions:\n  - path: "+fixtureLegacyOrdersFile+"\n    reason: Existing file awaits explicit modeling\n", 1)
	if transitional == string(manifest) {
		t.Fatal("could not add a transitional exclusion")
	}
	writeE2E(t, root, projectwork.ManifestPath, transitional)
	writeE2E(t, root, fixtureLegacyOrdersFile, legacyOrdersBytes)
	writeE2E(t, root, fixtureOrdersArtifact, e2eArtifact("orders", "orders-code", "orders-work", "orders-check", fixtureOrdersFile+", "+fixtureLegacyOrdersFile))
	for _, check := range []string{".markitect/model/orders/check.yaml", ".markitect/model/inventory/check.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(check)))
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(data), "TestProjectRunCheckProcess", "TestModelledFileCheckProcess", 1)
		if updated == string(data) {
			t.Fatalf("could not point %s at the modelled-file check", check)
		}
		writeE2E(t, root, check, updated)
	}
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "-m", "model the transitional legacy orders file")
	host := projectworkHost()
	plan, err := Plan(host, root, gitE2E(t, root, "rev-parse", "HEAD"), PlanRequest{Goal: "Model the legacy orders file in place.", ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	// The integrated candidate only drops the exclusion and carries the
	// document the run would generate for it.
	final := candidateData{Parents: []string{plan.InitialCandidateID}, Files: map[string]File{projectwork.ManifestPath: candidateWrite(projectwork.ManifestPath, string(manifest))}}
	compiled, err := finalProjectForCandidate(host, root, base, final)
	if err != nil {
		t.Fatal(err)
	}
	document, err := projectwork.Document(compiled, false)
	if err != nil {
		t.Fatal(err)
	}
	documentPath := projectwork.DocumentPath(compiled.Config)
	final.Files[documentPath] = candidateWrite(documentPath, document)
	if final.ID, err = newID(); err != nil {
		t.Fatal(err)
	}
	if err := store.writeCandidate(dir, final); err != nil {
		t.Fatal(err)
	}
	if final, err = store.readCandidate(dir, final.ID); err != nil {
		t.Fatal(err)
	}
	if compiled, err = finalProjectForCandidate(host, root, base, final); err != nil {
		t.Fatal(err)
	}
	if string(compiled.Snapshot.Files[fixtureLegacyOrdersFile]) != legacyOrdersBytes {
		t.Fatal("precondition: closure did not bind the modelled file's base bytes")
	}
	report := RunReport{APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Operation: plan.Operation, Status: StatusIntegrated, Mode: ModeControlledLocal,
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot,
		ModelDigest: plan.ModelDigest, RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers),
		Candidate: CandidateRef{ID: final.ID, Snapshot: final.Digest, Files: map[string]string{}, Integrated: true}, Revision: 1}
	for i := range report.Tasks {
		report.Tasks[i].ReportStatus = "complete"
	}
	var reviews []ReviewRecord
	for _, task := range report.Tasks {
		phase := "work"
		if len(activeChildren(report.Tasks, task.ManagerID)) > 0 {
			phase = "integrate"
		}
		if !phaseReviewRequired(compiled, task, report.Tasks, phase) {
			continue
		}
		scope, err := reviewScopeDigest(plan, compiled, task, phase, report)
		if err != nil {
			t.Fatal(err)
		}
		reviews = append(reviews, ReviewRecord{TaskID: task.ID, ManagerID: task.ManagerID, Phase: phase, CandidateID: final.ID, CandidateDigest: final.Digest,
			ScopeDigest: scope, InputDigest: "sha256:review-input", Outcome: "pass", Findings: []ReviewFinding{}, Receipt: agentexec.Receipt{RunID: "review-" + task.ID}})
	}
	report.Reviews = reviews
	if err := persistState(store, &report); err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified {
		t.Fatalf("Verify: report=%+v err=%v", verified, err)
	}
	assertChecksObservedLegacyFile(t, verified.Checks)
	if verified.ManagerVerification == nil || verified.ManagerVerification.SnapshotDigest != compiled.Snapshot.Digest() {
		t.Fatal("verification did not bind the closure snapshot that holds the modelled file's bytes")
	}
	// Full verification materializes its own check tree when no planned check
	// result covers a declared check.
	runtimeConfig, err := LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	full, err := FullVerifyProject(context.Background(), host, ProcessInvoker{}, root, compiled, runtimeConfig, FullVerifyBinding{ExpectedSnapshot: compiled.Snapshot.Digest()})
	if err != nil || full.Status != "passed" {
		t.Fatalf("FullVerifyProject: report=%+v err=%v", full, err)
	}
	assertChecksObservedLegacyFile(t, full.Checks)
}

func assertChecksObservedLegacyFile(t *testing.T, checks []CheckResult) {
	t.Helper()
	if len(checks) == 0 {
		t.Fatal("no declared check ran")
	}
	for _, check := range checks {
		if check.Outcome != "passed" || !strings.Contains(check.Stdout, "observed "+fixtureLegacyOrdersFile) {
			t.Fatalf("check %s did not observe the modelled file: %+v", check.ID, check)
		}
	}
}

// TestModelledFileCheckProcess is a declared check that passes only when the
// check tree holds the legacy orders file with its committed bytes.
func TestModelledFileCheckProcess(t *testing.T) {
	if os.Getenv(modelledFileCheckEnv) != "1" {
		return
	}
	content, err := os.ReadFile(fixtureLegacyOrdersFile)
	if err != nil || string(content) != legacyOrdersBytes {
		processExit(1, fmt.Sprintf("modelled file check failed: content=%q error=%v", content, err))
	}
	fmt.Println("observed " + fixtureLegacyOrdersFile)
	processExit(0, "")
}

// Declared checks once ran in a materialized candidate under the repository's
// run store. Below a deep repository (or GOTMPDIR) that directory exceeded the
// Windows working-directory limit and every check failed with "The directory
// name is invalid". Checks now run in a fresh short temporary directory.
func TestVerifyRunsDeclaredChecksOutsideTheRunStore(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	plan := planBothManagers(t, root)
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified {
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

func TestCheckWorkingDirectoryLengthNamesTheWindowsLimit(t *testing.T) {
	short := strings.Repeat("d", maxWindowsWorkingDirectory)
	long := strings.Repeat("d", maxWindowsWorkingDirectory+1)
	if err := checkWorkingDirectoryLength(short); err != nil {
		t.Fatalf("a directory at the limit was rejected: %v", err)
	}
	err := checkWorkingDirectoryLength(long)
	if runtime.GOOS != "windows" {
		if err != nil {
			t.Fatalf("the Windows limit applied on %s: %v", runtime.GOOS, err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), "at most 258 characters") {
		t.Fatalf("over-long directory error = %v", err)
	}
}

// Candidate paths are checked lexically. Windows also resolves an existing
// entry through its 8.3 short name or another case, so MARKIT~1/project.yaml
// once overwrote .markitect/project.yaml in the copy that checks run in.
func TestMaterializeCandidateRefusesWindowsAliasesOfExistingEntries(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("short names and case aliases are resolved by Windows")
	}
	base := &Snapshot{
		Files: map[string][]byte{".markitect/project.yaml": []byte("project\n"), "Docs/guide.md": []byte("guide\n")},
		Modes: map[string]string{".markitect/project.yaml": "100644", "Docs/guide.md": "100644"},
	}
	materialize := func(t *testing.T, path string) (string, error) {
		t.Helper()
		dest := t.TempDir()
		// .markitect exists first, as when its files sort first, and so owns
		// the short name MARKIT~1 on a volume that generates 8.3 names.
		if err := os.Mkdir(filepath.Join(dest, ".markitect"), 0o700); err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(path, "MARKIT~1/") && !resolvesAsShortName(dest, ".markitect", "MARKIT~1") {
			t.Skip("volume generates no 8.3 short names")
		}
		candidate := candidateData{Files: map[string]File{path: {Path: path, Mode: "100644", Content: []byte("alias\n")}}}
		return dest, materializeCandidate(dest, base, candidate)
	}
	// Every NTFS volume resolves case variants, so only the 8.3 case skips.
	for _, tc := range []struct{ name, path string }{
		{"8.3 short name", "MARKIT~1/project.yaml"},
		{"case variant", "docs/guide.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest, err := materialize(t, tc.path)
			if err == nil {
				t.Errorf("materializeCandidate accepted alias %s", tc.path)
			}
			for name, want := range base.Files {
				if got, readErr := os.ReadFile(filepath.Join(dest, filepath.FromSlash(name))); readErr != nil || string(got) != string(want) {
					t.Errorf("alias %s changed %s to %q (%v)", tc.path, name, got, readErr)
				}
			}
		})
	}
	// Stored names and new 8.3-shaped names stay writable.
	for _, path := range []string{"Docs/guide.md", "notes~1/new.md"} {
		dest, err := materialize(t, path)
		if err != nil {
			t.Fatalf("materializeCandidate refused %s: %v", path, err)
		}
		if got, readErr := os.ReadFile(filepath.Join(dest, filepath.FromSlash(path))); readErr != nil || string(got) != "alias\n" {
			t.Fatalf("%s = %q (%v)", path, got, readErr)
		}
	}
}

// resolvesAsShortName reports whether Windows resolves alias in dir to the
// existing entry long, which needs a volume that generates 8.3 short names.
func resolvesAsShortName(dir, long, alias string) bool {
	longInfo, longErr := os.Stat(filepath.Join(dir, long))
	aliasInfo, aliasErr := os.Stat(filepath.Join(dir, alias))
	return longErr == nil && aliasErr == nil && os.SameFile(longInfo, aliasInfo)
}

// A check once killed only its direct child on timeout and set no WaitDelay,
// so a descendant holding the inherited output pipes kept runCheck blocked
// past the check timeout (and forever if it never exited). The timeout now
// stops the whole process tree.
func TestCheckTimeoutStopsDescendantsHoldingOutput(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	resolved, raw, err := readPinnedExecutable(executable)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(checkDescendantHelperEnv, "spawn")
	check := CheckPlan{ID: "descendant-timeout", Owner: "orders", Required: true,
		Command:        []string{filepath.Base(executable), "-test.run=^TestCheckDescendantHelperProcess$"},
		ExecutablePath: resolved, ExecutableDigest: rawContentDigest(raw)}
	agent := Agent{Timeout: Duration(time.Second), Environment: []string{checkDescendantHelperEnv, "PATH", "SystemRoot"}}
	started := time.Now()
	result := runCheck(context.Background(), t.TempDir(), check, agent, Duration(time.Minute), nil)
	elapsed := time.Since(started)
	if result.Outcome != "failed" || result.Error != "check timed out" {
		t.Fatalf("timed-out check = outcome %q error %q, want failed / check timed out", result.Outcome, result.Error)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("runCheck with a 1s check timeout returned after %s; a descendant holding its output kept it blocked", elapsed.Round(10*time.Millisecond))
	}
	_, pidText, found := strings.Cut(strings.TrimSpace(result.Stdout), "spawned descendant ")
	pid, err := strconv.Atoi(pidText)
	if !found || err != nil {
		t.Fatalf("check did not report its descendant: stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		alive, err := processAlive(pid)
		if err != nil {
			t.Fatal(err)
		}
		if !alive {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("check descendant %d outlived the check timeout", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestCheckDescendantHelperProcess is the re-executed check for
// TestCheckTimeoutStopsDescendantsHoldingOutput. It is a no-op unless the
// helper environment selects a role.
func TestCheckDescendantHelperProcess(t *testing.T) {
	switch os.Getenv(checkDescendantHelperEnv) {
	case "spawn":
		executable, err := os.Executable()
		if err != nil {
			os.Exit(3)
		}
		descendant := exec.Command(executable, "-test.run=^TestCheckDescendantHelperProcess$")
		descendant.Env = append(os.Environ(), checkDescendantHelperEnv+"=descendant")
		descendant.Stdout = os.Stdout
		descendant.Stderr = os.Stderr
		if err := descendant.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "start descendant:", err)
			os.Exit(3)
		}
		fmt.Println("spawned descendant", descendant.Process.Pid)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "descendant":
		time.Sleep(8 * time.Second)
		os.Exit(0)
	}
}
