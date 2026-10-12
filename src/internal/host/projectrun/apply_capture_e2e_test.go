//go:build !runtime_unit

package projectrun

import (
	"bytes"
	"context"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreflightApplyAndApplyPreserveRawCRLFCAS(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	gitE2E(t, root, "config", "core.autocrlf", "true")
	for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		baseBytes, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("read base target %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, bytes.ReplaceAll(baseBytes, []byte("\n"), []byte("\r\n")), 0o644); err != nil {
			t.Fatalf("materialize CRLF target %s: %v", path, err)
		}
	}
	project := runtimeLifecycleProject(t, root, false)
	for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read finite CRLF source %s: %v", path, err)
		}
		project.Snapshot.Files[path] = append([]byte(nil), content...)
	}
	host := processRuntimeHost(root, project)
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and run the declared checks.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("Verify: status=%s err=%v", verified.Status, err)
	}
	paths, err := ApplyPaths(host, root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("PreflightApply rejected clean CRLF targets: %v", err)
	}
	request := applyRequestFromPreflight(preflight)
	firstTarget := filepath.Join(root, filepath.FromSlash(paths[0]))
	lfBytes, err := os.ReadFile(firstTarget)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(firstTarget, bytes.ReplaceAll(lfBytes, []byte("\r\n"), []byte("\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	if err == nil || len(stale.Written) != 0 {
		t.Fatalf("Apply did not reject raw-byte change after PreflightApply without writing: report=%+v err=%v", stale, err)
	}
	if err := os.WriteFile(firstTarget, lfBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	if err != nil || applied.Status != "applied" {
		t.Fatalf("Apply rejected unchanged CRLF targets from preflight: status=%s err=%v", applied.Status, err)
	}
}

func TestApplyRechecksTargetIndexAtGuardedWriteBoundary(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	host := runtimeLifecycleHost(t, root)
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and run the declared checks.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("Verify: status=%s err=%v", verified.Status, err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("PreflightApply: %v", err)
	}
	request := applyRequestFromPreflight(preflight)
	path := "src/orders/implementation.txt"
	target := filepath.Join(root, filepath.FromSlash(path))
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	baseLoad := host.Load
	var injectIndexDrift bool
	var baseLoads int
	host.Load = func(loadRoot, revision string) (*projectvalue.Project, error) {
		project, loadErr := baseLoad(loadRoot, revision)
		if loadErr != nil {
			return project, loadErr
		}
		if injectIndexDrift && revision == plan.BaseRevision {
			baseLoads++
			// The first fixed-base load is Apply's pre-capture check. The second
			// happens inside its guarded-write callback, after repositoryMatches.
			if baseLoads == 2 {
				injectIndexDrift = false
				stageApplyTargetDrift(t, root, target, original)
			}
		}
		return project, nil
	}
	injectIndexDrift = true
	report, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	if err == nil || !strings.Contains(err.Error(), "staged index content differs") {
		t.Fatalf("Apply did not reject index drift at the guarded-write boundary: report=%+v err=%v", report, err)
	}
	if len(report.Written) != 0 {
		t.Fatalf("Apply wrote target files despite guarded-boundary index drift: %+v", report.Written)
	}
	current, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(current, original) {
		t.Fatalf("Apply changed working target before rejecting staged index drift: bytes=%q err=%v", current, err)
	}
	if baseLoads != 2 {
		t.Fatalf("test did not inject drift after the initial check and before guarded write: base loads=%d", baseLoads)
	}
}

func TestApplyRefusedByHeldWriterLockKeepsRunVerified(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	host := runtimeLifecycleHost(t, root)
	plan, request := verifiedApplyRequest(t, host, root)
	lockPath := filepath.Join(root, ".markitect", "write.lock")
	if err := os.WriteFile(lockPath, []byte("held by another Host writer\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	assertApplyRejectedKeepsRunVerified(t, root, plan.ID, report, err)
	assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	retry, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	if err != nil || retry.Status != StatusApplied {
		t.Fatalf("retry after releasing the writer lock did not apply: status=%s err=%v", retry.Status, err)
	}
}

func TestApplyStaleUnderWriterLockIsRejectedLikeEarlyStaleness(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	host := runtimeLifecycleHost(t, root)
	plan, request := verifiedApplyRequest(t, host, root)
	baseLoad := host.Load
	staleModel := false
	changedProject, err := baseLoad(root, plan.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	changedProject = runtimeProjectWithDigest(changedProject, ":model-change")
	host.Load = func(loadRoot, revision string) (*projectvalue.Project, error) {
		project, loadErr := baseLoad(loadRoot, revision)
		if loadErr != nil {
			return nil, loadErr
		}
		if revision == "" && staleModel {
			return changedProject, nil
		}
		return project, nil
	}
	modelPath := filepath.Join(root, ".markitect", "model", "orders", "statement.yaml")
	original, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := append(append([]byte(nil), original...), []byte("# concurrent editor save\n")...)
	if err := os.WriteFile(modelPath, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	staleModel = true
	early, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	assertApplyRejectedKeepsRunVerified(t, root, plan.ID, early, err)
	if err := os.WriteFile(modelPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	staleModel = false

	// The same edit first observed by the guarded-write precondition, while
	// the Host writer lock is held, is the same rejection.
	writerLock := filepath.Join(root, ".markitect", "write.lock")
	injected := false
	guardedHost := host
	guardedHost.Load = func(loadRoot, revision string) (*projectvalue.Project, error) {
		if revision == "" && !injected {
			if _, statErr := os.Stat(writerLock); statErr == nil {
				injected = true
				staleModel = true
				if err := os.WriteFile(modelPath, edited, 0o644); err != nil {
					t.Errorf("inject concurrent edit: %v", err)
				}
			}
		}
		return baseLoad(loadRoot, revision)
	}
	late, err := Apply(guardedHost, projectexecutor.ProcessInvoker{}, root, request)
	if !injected {
		t.Fatal("test did not reach the guarded-write precondition")
	}
	assertApplyRejectedKeepsRunVerified(t, root, plan.ID, late, err)
	if err := os.WriteFile(modelPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	staleModel = false
	retry, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	if err != nil || retry.Status != StatusApplied {
		t.Fatalf("retry after reverting the transient edit did not apply: status=%s err=%v", retry.Status, err)
	}
}

func TestApplyGenuinePartialWriteFailsRunAndKeepsReceipt(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores the read-only directory that blocks the second target")
	}
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	host := runtimeLifecycleHost(t, root)
	plan, request := verifiedApplyRequest(t, host, root)
	paths, err := ApplyPaths(host, root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("partial-write fixture needs two sorted targets, got %v", paths)
	}
	first, second := paths[0], paths[1]
	secondTarget := filepath.Join(root, filepath.FromSlash(second))
	original, err := os.ReadFile(secondTarget)
	if err != nil {
		t.Fatal(err)
	}
	blockApplyTargetReplace(t, secondTarget)
	report, applyErr := Apply(host, projectexecutor.ProcessInvoker{}, root, request)
	if applyErr == nil || report.Status != "partial" || report.Error == "" ||
		!stringSlicesEqual(report.Written, []string{first}) || !stringSlicesEqual(report.Journal, []string{first}) {
		t.Fatalf("Apply that wrote only %s did not report a partial write: report=%+v err=%v", first, report, applyErr)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.readLatestState(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != StatusFailed {
		t.Fatalf("partial Apply left run %s instead of failed", state.Status)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := os.ReadDir(filepath.Join(dir, "apply"))
	if err != nil || len(receipts) != 1 {
		t.Fatalf("partial Apply did not journal exactly one Apply receipt: receipts=%d err=%v", len(receipts), err)
	}
	var receipt ApplyReport
	if err := readJSON(filepath.Join(dir, "apply", receipts[0].Name()), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "partial" || !stringSlicesEqual(receipt.Written, []string{first}) {
		t.Fatalf("journaled Apply receipt does not name the written path: %+v", receipt)
	}
	candidate, err := store.readCandidate(dir, request.CandidateID)
	if err != nil {
		t.Fatal(err)
	}
	written, ok := candidate.Files[first]
	if !ok || written.Delete {
		t.Fatalf("candidate does not write %s", first)
	}
	assertFileContents(t, root, first, string(written.Content))
	assertFileContents(t, root, second, string(original))
}
