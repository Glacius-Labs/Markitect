package examples

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

// These scenarios exercise the public binary, its durable state and the real
// exchange process. Scripted responses establish local method mechanics only.
func TestMethodExchangeCLIEndToEnd(t *testing.T) {
	t.Parallel()
	binaries := testkit.TempDir(t)
	cli := filepath.Join(binaries, executableName("markitect"))
	exchange := filepath.Join(binaries, executableName("markitect-exchange-executor"))
	for _, build := range []struct{ target, pkg string }{{cli, "./src/cmd/markitect"}, {exchange, "./src/cmd/markitect-exchange-executor"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", build.target, build.pkg)
		cmd.Dir = harnessRepositoryRoot(t)
		output, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("build %s: %v\n%s", build.pkg, err, output)
		}
	}
	for _, mode := range []string{"happy", "forgotten-file", "missing-required-file", "out-of-scope", "integration-defect", "hold-first-request"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			f := newMethodExchangeFixture(t, testkit.TempDir(t), mode)
			driver := &methodCLIDriver{t: t, binary: cli, root: f.root, logDir: filepath.Join(f.exchangeDir, "..", "cli")}
			driver.preserveEvidence(f, mode)
			driver.configure(exchange, f.exchangeDir)
			f.start(t)
			defer func() {
				entries, err := f.stop()
				if err != nil {
					t.Errorf("exchange responder: %v", err)
				}
				seen := map[string]bool{}
				for _, entry := range entries {
					if entry.ResponseError != "" {
						t.Errorf("exchange adapter rejected scripted response: %s", entry.ResponseError)
					}
					if entry.RunID == "" || entry.Nonce == "" || seen[entry.RunID+"/"+entry.Nonce] {
						t.Errorf("missing or duplicate exchange invocation identity: %+v", entry)
					}
					seen[entry.RunID+"/"+entry.Nonce] = true
				}
				t.Logf("archived exchange requests=%d", len(entries))
			}()
			plan := driver.must("plan", "--goal", "Deliver the declared orders and inventory artifacts and their parent integration.")
			plan = driver.must("plan", "--goal", "Deliver the declared orders and inventory artifacts and their parent integration.", "--expect", methodString(t, plan, "previewDigest"), "--write")
			id := methodString(t, plan, "id")
			if mode == "hold-first-request" {
				driver.killAndResume(f, id)
				return
			}
			run, code := driver.call("run", id, "--execute")
			if mode == "out-of-scope" {
				if code == 0 {
					t.Fatalf("out-of-scope run accepted: %s", run)
				}
				status := driver.must("status", id)
				if !bytes.Contains(status, []byte("outside")) && !bytes.Contains(status, []byte("scope")) {
					t.Fatalf("missing scope diagnostic: %s", status)
				}
				driver.refuseApply(id, driver.candidate(id))
				driver.assertSource(f, false)
				return
			}
			if code != 0 || methodString(t, run, "status") != "integrated" {
				t.Fatalf("run exit=%d: %s", code, run)
			}
			driver.assertSource(f, false)
			candidate := methodNestedString(t, run, "candidate", "id")
			driver.refuseApply(id, candidate)
			verification, verifyCode := driver.call("verify", id, "--execute")
			if mode == "missing-required-file" {
				if verifyCode != 2 || len(verification) != 0 {
					t.Fatalf("missing required artifact must fail structurally before a verification report: exit=%d %s", verifyCode, verification)
				}
				status := driver.must("status", id)
				var state struct {
					Run struct {
						Status string
						Checks []json.RawMessage
					}
				}
				if err := json.Unmarshal(status, &state); err != nil {
					t.Fatal(err)
				}
				if state.Run.Status != "integrated" || len(state.Run.Checks) != 0 {
					t.Fatalf("structural failure fabricated a process-check result: %s", status)
				}
				calls := len(f.snapshotEntries())
				if _, code := driver.call("repair", id, "--execute"); code == 0 {
					t.Fatal("structural omission was accepted as failed-check repair")
				}
				if len(f.snapshotEntries()) != calls {
					t.Fatal("structural repair rejection started another executor")
				}
				driver.refuseApply(id, candidate)
				driver.assertSource(f, false)
				t.Log("missing required file: structural Verify refusal, no declared check started, no repair replay, no Apply")
				return
			}
			if mode == "forgotten-file" {
				if verifyCode == 0 || methodString(t, verification, "status") != "failed" {
					t.Fatalf("forgotten file escaped the declared check: exit=%d %s", verifyCode, verification)
				}
				failedVerification := append([]byte(nil), verification...)
				driver.refuseApply(id, candidate)
				repaired := driver.must("repair", id, "--execute")
				if methodString(t, repaired, "id") != id || methodString(t, repaired, "status") != "integrated" {
					t.Fatalf("repair identity/status: %s", repaired)
				}
				newCandidate := methodNestedString(t, repaired, "candidate", "id")
				if newCandidate == candidate {
					t.Fatal("repair reused the failed candidate")
				}
				driver.assertRepair(f, candidate, newCandidate, failedVerification, repaired)
				driver.refuseApply(id, newCandidate)
				verification = driver.must("verify", id, "--execute")
				driver.refuseApply(id, candidate)
				candidate = newCandidate
			} else if verifyCode != 0 {
				t.Fatalf("verify exit=%d: %s", verifyCode, verification)
			}
			if methodString(t, verification, "status") != "verified" {
				t.Fatalf("verification status: %s", verification)
			}
			var evidence struct {
				CandidateID string
				Checks      []struct {
					CandidateID, Outcome string
					ExitCode             int
				}
			}
			if err := json.Unmarshal(verification, &evidence); err != nil {
				t.Fatal(err)
			}
			if evidence.CandidateID != candidate || len(evidence.Checks) != 3 {
				t.Fatalf("verification must bind all three literal checks to candidate %s: %s", candidate, verification)
			}
			for _, check := range evidence.Checks {
				if check.CandidateID != candidate || check.Outcome != "passed" || check.ExitCode != 0 {
					t.Fatalf("check did not pass exact candidate: %s", verification)
				}
			}
			driver.assertSource(f, false)
			preflight := driver.must("apply", "--plan", id, "--run", id, "--candidate", candidate)
			args := []string{"apply", "--plan", id, "--run", id, "--candidate", candidate, "--branch", methodString(t, preflight, "targetBranch"), "--head", methodString(t, preflight, "expectedHead"), "--worktree", methodString(t, preflight, "expectedWorktree"), "--expect", "sha256:" + strings.Repeat("0", 64), "--write"}
			if _, code := driver.call(args...); code == 0 {
				t.Fatal("stale verification digest was accepted")
			}
			driver.assertSource(f, false)
			args[len(args)-2] = methodString(t, preflight, "verificationDigest")
			applied := driver.must(args...)
			if methodString(t, applied, "status") != "applied" {
				t.Fatalf("apply status: %s", applied)
			}
			driver.assertSource(f, true)
			status := driver.must("status", id)
			if mode == "integration-defect" {
				driver.assertIntegrationReview(f, candidate, status)
			}
			t.Logf("%s: real CLI plan/run/review/verify/apply completed; run=%s candidate=%s", mode, id, candidate)
		})
	}
}

type methodCLIDriver struct {
	t                    *testing.T
	binary, root, logDir string
	sequence             int
}

func (d *methodCLIDriver) preserveEvidence(f *methodExchangeFixture, mode string) {
	d.t.Helper()
	base := os.Getenv("MARKITECT_METHOD_EVIDENCE_DIR")
	if base == "" {
		return
	}
	if !filepath.IsAbs(base) {
		d.t.Fatal("MARKITECT_METHOD_EVIDENCE_DIR must be absolute")
	}
	rel, err := filepath.Rel(harnessRepositoryRoot(d.t), base)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		d.t.Fatal("method evidence must live outside the source repository")
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		d.t.Fatal(err)
	}
	dir := filepath.Join(base, mode)
	if err := os.Mkdir(dir, 0700); err != nil {
		d.t.Fatalf("new evidence destination required (preserve earlier runs): %v", err)
	}
	d.logDir, f.ledgerPath = filepath.Join(dir, "cli"), filepath.Join(dir, "exchange-ledger.jsonl")
	d.t.Cleanup(func() {
		for name, source := range map[string]string{"durable-runs": filepath.Join(f.root, ".markitect", "records", "run"), "private-and-transport": filepath.Join(f.root, ".markitect", "runs"), "unaccepted-exchange": f.exchangeDir, "literal-expectations": f.expectedDir} {
			if _, err := os.Stat(source); os.IsNotExist(err) {
				if name == "durable-runs" {
					d.t.Errorf("required canonical run evidence missing: %v", err)
				}
				continue
			}
			if err := os.CopyFS(filepath.Join(dir, name), os.DirFS(source)); err != nil {
				d.t.Errorf("preserve %s: %v", name, err)
				continue
			}
			verifyMethodEvidenceBytes(d.t, source, filepath.Join(dir, name))
		}
		baseline, err := json.MarshalIndent(f.baseline, "", "  ")
		if err != nil {
			d.t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "fixture-baseline.json"), baseline, 0600); err != nil {
			d.t.Error(err)
		}
	})
}

func (d *methodCLIDriver) assertRepair(f *methodExchangeFixture, before, after string, failed, repaired []byte) {
	d.t.Helper()
	const checkID = `["project.markitect.example.org/v1alpha1","Check","orders","orders-check"]`
	var verification struct {
		Digest string
		Checks []struct {
			ID, CandidateID, Outcome, Stdout string
			ExitCode                         int
		}
	}
	if err := json.Unmarshal(failed, &verification); err != nil {
		d.t.Fatal(err)
	}
	diagnostic := ""
	for _, check := range verification.Checks {
		if check.ID == checkID && check.CandidateID == before && check.Outcome == "failed" && check.ExitCode == 1 && strings.Contains(check.Stdout, "orders implementation v1") && strings.Contains(check.Stdout, "orders implementation v2") {
			diagnostic = check.Stdout
		}
	}
	if diagnostic == "" {
		d.t.Fatalf("forgotten replacement did not fail its exact declared literal check: %s", failed)
	}
	var run struct {
		RepairRounds []struct {
			PriorCandidateID, CandidateID, VerificationDigest string
			CheckFeedback                                     []struct {
				ID, Stdout string
				ExitCode   int
			}
		}
	}
	if err := json.Unmarshal(repaired, &run); err != nil {
		d.t.Fatal(err)
	}
	if len(run.RepairRounds) != 1 || run.RepairRounds[0].PriorCandidateID != before || run.RepairRounds[0].CandidateID != after || run.RepairRounds[0].VerificationDigest != verification.Digest {
		d.t.Fatalf("repair lost the failed check/candidate binding: %s", repaired)
	}
	feedback := false
	for _, entry := range f.snapshotEntries() {
		if entry.Kind != methodExchangeTaskKind || entry.ManagerID != methodExchangeOrdersID || entry.RepairRound != 1 {
			continue
		}
		var request struct {
			Request struct {
				Context struct {
					RepairChecks []struct {
						ID, Stdout string
						ExitCode   int
					}
				}
			}
		}
		if err := json.Unmarshal(entry.RawRequest, &request); err != nil {
			d.t.Fatal(err)
		}
		for _, check := range request.Request.Context.RepairChecks {
			if check.ID == checkID && check.ExitCode == 1 && check.Stdout == diagnostic {
				feedback = true
			}
		}
	}
	if !feedback {
		d.t.Fatal("Orders repair invocation did not receive the exact failed-check diagnostic")
	}
}

func (d *methodCLIDriver) assertIntegrationReview(f *methodExchangeFixture, candidate string, status []byte) {
	d.t.Helper()
	var state struct {
		Run struct {
			Reviews []struct {
				ManagerID, Phase, CandidateID, CandidateDigest, InputDigest, Outcome string
				Findings                                                             []struct{ Path, Grounding, Expectation string }
			}
		}
	}
	if err := json.Unmarshal(status, &state); err != nil {
		d.t.Fatal(err)
	}
	rejected, accepted := "", false
	for _, review := range state.Run.Reviews {
		if review.ManagerID != methodExchangeRootID || review.Phase != "integrate" {
			continue
		}
		if review.Outcome == "pass" && review.CandidateID == candidate && len(review.Findings) == 0 {
			accepted = true
		}
		if review.Outcome != "fail" || review.CandidateID == candidate || review.CandidateDigest == "" {
			continue
		}
		for _, finding := range review.Findings {
			if finding.Path != "src/project-integration.txt" || finding.Expectation == "" || finding.Grounding == "" {
				continue
			}
			for _, entry := range f.snapshotEntries() {
				if entry.Kind != methodExchangeReviewKind || entry.ManagerID != methodExchangeRootID || entry.Phase != "integrate" {
					continue
				}
				var request struct {
					InputDigest string
					Request     struct {
						Context struct {
							CandidateID, CandidateDigest string
							CandidateFiles               []struct {
								Path      string
								Grounding []string
							}
						}
					}
				}
				if err := json.Unmarshal(entry.RawRequest, &request); err != nil {
					d.t.Fatal(err)
				}
				if request.InputDigest != review.InputDigest || request.Request.Context.CandidateID != review.CandidateID || request.Request.Context.CandidateDigest != review.CandidateDigest {
					continue
				}
				for _, file := range request.Request.Context.CandidateFiles {
					for _, grounding := range file.Grounding {
						if file.Path == finding.Path && grounding == finding.Grounding {
							rejected = review.CandidateID
						}
					}
				}
			}
		}
	}
	if rejected == "" || !accepted {
		d.t.Fatalf("missing exact grounded integration rejection and fresh candidate acceptance: %s", status)
	}
}

func (d *methodCLIDriver) assertSource(f *methodExchangeFixture, applied bool) {
	d.t.Helper()
	want := f.baseline
	if applied {
		want = f.expected
	}
	for path, content := range want {
		actual, err := os.ReadFile(filepath.Join(d.root, filepath.FromSlash(path)))
		if err != nil || string(actual) != content {
			d.t.Fatalf("source %s applied=%t: got=%q want=%q err=%v", path, applied, actual, content, err)
		}
	}
	if applied {
		for path, content := range f.baseline {
			if _, changed := f.expected[path]; changed {
				continue
			}
			actual, err := os.ReadFile(filepath.Join(d.root, filepath.FromSlash(path)))
			if err != nil || string(actual) != content {
				d.t.Fatalf("Apply changed baseline outside declared outputs: %s err=%v", path, err)
			}
		}
	}
	if !applied {
		for path := range f.expected {
			if _, existed := f.baseline[path]; existed {
				continue
			}
			if _, err := os.Stat(filepath.Join(d.root, filepath.FromSlash(path))); !os.IsNotExist(err) {
				d.t.Fatalf("candidate escaped into source before Apply: %s err=%v", path, err)
			}
		}
	}
	for path, content := range f.expected {
		actual, err := os.ReadFile(filepath.Join(f.expectedDir, filepath.FromSlash(path)))
		if err != nil || string(actual) != content {
			d.t.Fatalf("literal check expectation changed: %s got=%q want=%q err=%v", path, actual, content, err)
		}
	}
}

func (d *methodCLIDriver) killAndResume(f *methodExchangeFixture, id string) {
	d.t.Helper()
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		d.t.Skip("hard-kill tree proof is currently implemented for Windows and Linux")
	}
	if err := os.MkdirAll(d.logDir, 0700); err != nil {
		d.t.Fatal(err)
	}
	stdout, err := os.Create(filepath.Join(d.logDir, "killed-run.stdout.json"))
	if err != nil {
		d.t.Fatal(err)
	}
	defer stdout.Close()
	stderr, err := os.Create(filepath.Join(d.logDir, "killed-run.stderr.txt"))
	if err != nil {
		d.t.Fatal(err)
	}
	defer stderr.Close()
	cmd := exec.Command(d.binary, "run", "--repo", d.root, id, "--execute")
	cmd.Dir, cmd.Stdout, cmd.Stderr = d.root, stdout, stderr
	if err := cmd.Start(); err != nil {
		d.t.Fatal(err)
	}
	finished := false
	var cleanupPIDs []int
	defer func() {
		if !finished {
			if err := methodKillCLI(cmd, cleanupPIDs); err != nil {
				d.t.Errorf("clean up owned CLI/executor processes: %v", err)
			}
			_ = cmd.Wait()
		}
	}()
	select {
	case held := <-f.held:
		d.t.Logf("hard-kill raw request: %s", held.RawRequest)
	case <-time.After(2 * time.Minute):
		d.t.Fatal("real exchange request was not reached")
	}
	status := d.must("status", id)
	d.t.Logf("hard-kill durable status before kill: %s", status)
	var state projectrun.StatusReport
	if err := json.Unmarshal(status, &state); err != nil {
		d.t.Fatal(err)
	}
	invoking := 0
	for _, task := range state.Run.Tasks {
		if task.State == "invoking" {
			invoking++
		}
	}
	if state.Run.Status != "running" || invoking != 1 {
		d.t.Fatalf("held request lacks durable in-flight state: %s", status)
	}
	originalAttempt, originalReservation := assertHeldMethodAttempt(d.t, state.Run, "started", "requested")
	if runtime.GOOS == "windows" {
		kill := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
		if output, err := kill.CombinedOutput(); err != nil {
			d.t.Fatalf("kill exact CLI tree: %v %s", err, output)
		}
	} else if runtime.GOOS == "linux" {
		// Process executors get their own process group. Observe the actual child
		// before killing the Host; never select processes by a broad name match.
		threads, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", cmd.Process.Pid))
		if err != nil {
			d.t.Fatal(err)
		}
		var pids []string
		for _, thread := range threads {
			children, err := os.ReadFile(thread)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				d.t.Fatal(err)
			}
			pids = append(pids, strings.Fields(string(children))...)
		}
		if len(pids) != 1 {
			d.t.Fatalf("held CLI must have one exchange child, got %v", pids)
		}
		pid, err := strconv.Atoi(pids[0])
		if err != nil {
			d.t.Fatal(err)
		}
		cleanupPIDs = append(cleanupPIDs, pid)
		exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if err != nil || filepath.Base(exe) != executableName("markitect-exchange-executor") {
			d.t.Fatalf("held child executable=%s err=%v", exe, err)
		}
		if err := cmd.Process.Kill(); err != nil {
			d.t.Fatal(err)
		}
		kill := exec.Command("kill", "-KILL", "--", "-"+strconv.Itoa(pid))
		if output, err := kill.CombinedOutput(); err != nil {
			d.t.Fatalf("kill exact exchange group: %v %s", err, output)
		}
	} else {
		d.t.Skip("hard-kill tree proof is currently implemented for Windows and Linux")
	}
	if err := cmd.Wait(); err == nil {
		d.t.Fatal("hard-killed CLI exited successfully")
	}
	finished = true
	d.t.Logf("hard-killed CLI pid=%d process=%s exit=%d", cmd.Process.Pid, cmd.ProcessState.String(), cmd.ProcessState.ExitCode())
	resumed, code := d.call("resume", id, "--execute")
	d.t.Logf("hard-kill resume raw stdout (exit=%d): %s", code, resumed)
	if code == 0 || methodString(d.t, resumed, "status") != "blocked" || !bytes.Contains(resumed, []byte("uncertain")) {
		d.t.Fatalf("resume replayed/accepted unknown process outcome: exit=%d %s", code, resumed)
	}
	status = d.must("status", id)
	d.t.Logf("hard-kill durable status after resume: %s", status)
	if err := json.Unmarshal(status, &state); err != nil {
		d.t.Fatal(err)
	}
	if state.Run.Status != "blocked" {
		d.t.Fatalf("blocked state/empty receipts not durable: %s", status)
	}
	recoveredAttempt, recoveredReservation := assertHeldMethodAttempt(d.t, state.Run, "uncertain", "unknown")
	// Only knowledge of the original start changed; no terminal receipt,
	// replacement attempt, request or charged ticket may appear.
	recoveredAttempt.Status = originalAttempt.Status
	recoveredReservation.Request.State = originalReservation.Request.State
	if recoveredReservation.RecordedAt.Before(originalReservation.RecordedAt) {
		d.t.Fatal("reservation lifecycle timestamp regressed")
	}
	recoveredReservation.RecordedAt = originalReservation.RecordedAt
	if !bytes.Equal(methodMarshal(d.t, originalAttempt), methodMarshal(d.t, recoveredAttempt)) || !bytes.Equal(methodMarshal(d.t, originalReservation), methodMarshal(d.t, recoveredReservation)) {
		d.t.Fatalf("resume replaced or changed original attempt/request/reservation authority: %s", status)
	}
	uncertain := 0
	for _, task := range state.Run.Tasks {
		if task.State == "uncertain" {
			uncertain++
		}
	}
	if uncertain != 1 {
		d.t.Fatalf("unknown task state was not durably uncertain: %s", status)
	}
	d.refuseApply(id, d.candidate(id))
	d.assertSource(f, false)
	entries, err := f.stop()
	if err != nil {
		d.t.Fatal(err)
	}
	if len(entries) != 1 {
		d.t.Fatalf("resume issued additional exchange requests: %d", len(entries))
	}
	d.t.Logf("real hard kill: reclaimed writer lock; resume blocked unknown outcome; requests=1 receipts=0 run=%s", id)
}

func (d *methodCLIDriver) call(args ...string) ([]byte, int) {
	d.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	argv := append([]string{args[0], "--repo", d.root}, args[1:]...)
	cmd := exec.CommandContext(ctx, d.binary, argv...)
	cmd.Cancel = func() error { return methodKillCLI(cmd, nil) }
	cmd.WaitDelay = 5 * time.Second
	cmd.Dir = d.root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	d.sequence++
	if err := os.MkdirAll(d.logDir, 0700); err != nil {
		d.t.Fatal(err)
	}
	name := filepath.Join(d.logDir, fmt.Sprintf("%02d-%s", d.sequence, args[0]))
	for suffix, data := range map[string][]byte{".stdout.json": stdout, ".stderr.txt": stderr.Bytes()} {
		if err := os.WriteFile(name+suffix, data, 0600); err != nil {
			d.t.Fatal(err)
		}
	}
	encodedArgs, errArgs := json.Marshal(argv)
	if errArgs != nil {
		d.t.Fatal(errArgs)
	}
	if err := os.WriteFile(name+".argv.json", encodedArgs, 0600); err != nil {
		d.t.Fatal(err)
	}
	if ctx.Err() != nil {
		d.t.Fatalf("%v timed out: %v stderr=%s", args, ctx.Err(), stderr.String())
	}
	if err == nil {
		return stdout, 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		d.t.Logf("%s exit=%d stderr=%s", args[0], exit.ExitCode(), stderr.String())
		return stdout, exit.ExitCode()
	}
	d.t.Fatalf("start %v: %v", args, err)
	return nil, -1
}

// methodKillCLI limits cleanup to the CLI process owned by this test and its
// observed children. The real process executor has a separate Linux group.
func methodKillCLI(cmd *exec.Cmd, known []int) error {
	if cmd.Process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").CombinedOutput()
		if err != nil {
			killErr := cmd.Process.Kill()
			if errors.Is(killErr, os.ErrProcessDone) {
				killErr = nil
			}
			return errors.Join(fmt.Errorf("owned-tree taskkill failed: %v %s", err, out), killErr)
		}
		return nil
	}
	children := map[int]bool{}
	for _, pid := range known {
		children[pid] = true
	}
	if runtime.GOOS == "linux" {
		paths, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", cmd.Process.Pid))
		if err != nil {
			return err
		}
		for _, path := range paths {
			raw, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			for _, text := range strings.Fields(string(raw)) {
				pid, err := strconv.Atoi(text)
				if err != nil {
					return err
				}
				children[pid] = true
			}
		}
	}
	var problems []error
	for pid := range children {
		exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if filepath.Base(exe) == executableName("markitect-exchange-executor") {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			out, err := exec.CommandContext(ctx, "kill", "-KILL", "--", "-"+strconv.Itoa(pid)).CombinedOutput()
			cancel()
			if err != nil {
				problems = append(problems, fmt.Errorf("kill owned executor group %d: %v %s", pid, err, out))
			}
		} else {
			child, err := os.FindProcess(pid)
			if err == nil {
				err = child.Kill()
			}
			if err != nil && !errors.Is(err, os.ErrProcessDone) {
				problems = append(problems, err)
			}
		}
	}
	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		problems = append(problems, err)
	}
	return errors.Join(problems...)
}

func (d *methodCLIDriver) must(args ...string) []byte {
	d.t.Helper()
	out, code := d.call(args...)
	if code != 0 {
		d.t.Fatalf("%v exit=%d stdout=%s", args, code, out)
	}
	return out
}

func (d *methodCLIDriver) configure(exchange, dir string) {
	d.t.Helper()
	config := []string{"config", "--provider", "process", "--model", "scripted-method", "--provider-executable", exchange, "--provider-arg", "--dir", "--provider-arg", dir, "--provider-arg", "--poll", "--provider-arg", "100ms", "--provider-version", "exchange/test", "--cost-mode", "unmetered", "--max-cost-micros", "1000000"}
	for _, args := range [][]string{config, {"docs"}} {
		preview := d.must(args...)
		var decoded struct {
			Digest   string
			EditPlan struct{ Digest string }
		}
		if err := json.Unmarshal(preview, &decoded); err != nil {
			d.t.Fatal(err)
		}
		digest := decoded.Digest
		if digest == "" {
			digest = decoded.EditPlan.Digest
		}
		if digest == "" {
			d.t.Fatalf("preview missing digest: %s", preview)
		}
		d.must(append(args, "--expect", digest, "--write")...)
	}
	cmd := exec.Command("git", "-C", d.root, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		d.t.Fatalf("git add: %v %s", err, out)
	}
	cmd = exec.Command("git", "-C", d.root, "commit", "-m", "Freeze scripted process runtime")
	if out, err := cmd.CombinedOutput(); err != nil {
		d.t.Fatalf("git commit: %v %s", err, out)
	}
}

func methodString(t *testing.T, raw []byte, key string) string {
	t.Helper()
	var v map[string]json.RawMessage
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	var result string
	if err := json.Unmarshal(v[key], &result); err != nil || result == "" {
		t.Fatalf("missing %s in %s: %v", key, raw, err)
	}
	return result
}
func methodNestedString(t *testing.T, raw []byte, parent, key string) string {
	t.Helper()
	var v map[string]json.RawMessage
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return methodString(t, v[parent], key)
}
func (d *methodCLIDriver) candidate(id string) string {
	d.t.Helper()
	var status struct {
		Run struct{ Candidate struct{ ID string } }
	}
	if err := json.Unmarshal(d.must("status", id), &status); err != nil {
		d.t.Fatal(err)
	}
	return status.Run.Candidate.ID
}
func (d *methodCLIDriver) refuseApply(id, candidate string) {
	d.t.Helper()
	if _, code := d.call("apply", "--plan", id, "--run", id, "--candidate", candidate); code == 0 {
		d.t.Fatal("unverified/failed/old candidate remained applicable")
	}
}

func assertHeldMethodAttempt(t *testing.T, run projectrun.RunReport, attemptStatus, reservationState string) (projectrun.ManagerStageAttempt, projectrun.RoleStartReservation) {
	t.Helper()
	if len(run.Invocations) != 1 || len(run.RoleStartReservations) != 1 {
		t.Fatalf("original charged attempt was absent, replaced or duplicated: %s", methodMarshal(t, run))
	}
	row := run.Invocations[0]
	a := row.Attempt
	r := run.RoleStartReservations[0]
	empty := projectrun.InvocationLog{}.Receipt
	if a == nil || a.ID == "" || a.Status != attemptStatus || a.RequestRef == nil || a.RequestRef.Digest == "" || a.RequestDigest == "" || row.InputDigest != a.RequestDigest || a.ResponseRef != nil || a.ReportRef != nil || !a.FinishedAt.IsZero() || !bytes.Equal(methodMarshal(t, row.Receipt), methodMarshal(t, empty)) || row.ReportID != "" {
		t.Fatalf("original pending request gained terminal output or lost its binding: %s", methodMarshal(t, row))
	}
	taskFound := false
	for _, task := range run.Tasks {
		if task.ID == row.TaskID && task.ManagerID == a.ManagerID && task.RepairPhase == row.Phase && task.WorkAttempts == 1 && task.Attempts == 1 {
			taskFound = true
		}
	}
	if !taskFound || a.IterationID != row.TaskID || a.Phase != row.Phase || row.Role != "executor" || a.StageKey != run.ID+"/"+a.ID || r.Kind != "root" || r.TaskID != row.TaskID || r.Phase != row.Phase || r.Request.RequestID != a.ID || r.Request.ParentSessionID != run.ID || r.Request.Role != row.Role || r.Request.State != reservationState || r.Key != "root\x00"+row.TaskID+"\x00"+row.Phase+"\x00"+a.ID {
		t.Fatalf("pending attempt has no exact original charged reservation: %s", methodMarshal(t, run))
	}
	return *a, r
}
func methodMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func verifyMethodEvidenceBytes(t *testing.T, source, destination string) {
	t.Helper()
	count := 0
	err := filepath.WalkDir(source, func(p string, e os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e.IsDir() {
			return nil
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular evidence %s", p)
		}
		rel, err := filepath.Rel(source, p)
		if err != nil {
			return err
		}
		original, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		copied, err := os.ReadFile(filepath.Join(destination, rel))
		if err != nil {
			return err
		}
		if sha256.Sum256(original) != sha256.Sum256(copied) {
			return fmt.Errorf("copied evidence hash differs: %s", rel)
		}
		count++
		return nil
	})
	if err != nil {
		t.Errorf("exact evidence copy: %v", err)
	}
	if filepath.Base(destination) == "durable-runs" && count == 0 {
		t.Error("canonical durable evidence copy was empty")
	}
}
