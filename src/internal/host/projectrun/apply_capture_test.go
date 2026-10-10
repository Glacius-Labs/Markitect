package projectrun

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/guardedwrite"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func applyCaptureFixture(t *testing.T) (string, string, *Snapshot, []byte) {
	t.Helper()
	root := t.TempDir()
	gitE2E(t, root, "init", "-b", "codex/apply-capture-test")
	gitE2E(t, root, "config", "user.email", "apply-capture@example.test")
	gitE2E(t, root, "config", "user.name", "Apply Capture Test")
	gitE2E(t, root, "config", "core.autocrlf", "true")
	writeE2E(t, root, "README.md", "# Greeting\n\nFixed base text.\n")
	gitE2E(t, root, "add", "README.md")
	gitE2E(t, root, "commit", "-m", "fixed base")
	revision := identityHead(t, root)
	fixed, err := source.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	base := fixed.Files["README.md"]
	if len(base) == 0 || bytes.Contains(base, []byte("\r\n")) {
		t.Fatalf("test fixed Git blob is not LF-only: %q", base)
	}
	crlf := bytes.ReplaceAll(base, []byte("\n"), []byte("\r\n"))
	if err := os.WriteFile(filepath.Join(root, "README.md"), crlf, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, revision, fixed, crlf
}

func captureApplyTarget(t *testing.T, root string) *guardedwrite.Capture {
	t.Helper()
	capture, err := guardedwrite.CaptureFiles(root, []string{"README.md"})
	if err != nil {
		t.Fatal(err)
	}
	return capture
}

func TestCompareCaptureToBaseAcceptsCleanCRLFCheckout(t *testing.T) {
	root, revision, fixed, crlf := applyCaptureFixture(t)
	capture := captureApplyTarget(t, root)
	if !bytes.Equal(capture.Files["README.md"].Bytes, crlf) {
		t.Fatal("guarded capture did not preserve exact CRLF working bytes")
	}
	if err := compareCaptureToBase(root, revision, capture, fixed, []string{"README.md"}); err != nil {
		t.Fatalf("clean CRLF checkout was rejected against its LF Git base: %v", err)
	}
	if err := requireCleanSelectedBasisAtRevision(root, revision,
		&Snapshot{Files: map[string][]byte{"README.md": fixed.Files["README.md"]}, Modes: map[string]string{"README.md": fixed.Modes["README.md"]}},
		&Snapshot{Files: map[string][]byte{"README.md": crlf}, Modes: map[string]string{"README.md": fixed.Modes["README.md"]}}); err != nil {
		t.Fatalf("existing plan clean-filter contract rejected the same CRLF checkout: %v", err)
	}
}

func TestCompareCaptureToBaseRejectsEditedBytesAndChangedAttributes(t *testing.T) {
	t.Run("real edit", func(t *testing.T) {
		root, revision, fixed, _ := applyCaptureFixture(t)
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Edited\r\n\r\nFixed base text.\r\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := compareCaptureToBase(root, revision, captureApplyTarget(t, root), fixed, []string{"README.md"}); err == nil {
			t.Fatal("real selected-file edit was accepted as a clean-filter-equivalent base")
		}
	})
	t.Run("attribute drift", func(t *testing.T) {
		root, revision, fixed, _ := applyCaptureFixture(t)
		writeE2E(t, root, ".gitattributes", "README.md -text\n")
		err := compareCaptureToBase(root, revision, captureApplyTarget(t, root), fixed, []string{"README.md"})
		if err == nil || !strings.Contains(err.Error(), "Git attributes") {
			t.Fatalf("changed clean-filter attributes were not rejected explicitly: %v", err)
		}
	})
	t.Run("fixed nontext attribute does not normalize CRLF", func(t *testing.T) {
		root, _, _, _ := applyCaptureFixture(t)
		writeE2E(t, root, ".gitattributes", "README.md -text\n")
		gitE2E(t, root, "add", ".gitattributes")
		gitE2E(t, root, "commit", "-m", "fix README as nontext")
		revision := identityHead(t, root)
		fixed, err := source.Load(root, revision)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "README.md"), bytes.ReplaceAll(fixed.Files["README.md"], []byte("\n"), []byte("\r\n")), 0o644); err != nil {
			t.Fatal(err)
		}
		err = compareCaptureToBase(root, revision, captureApplyTarget(t, root), fixed, []string{"README.md"})
		if err == nil || !strings.Contains(err.Error(), "fixed base bytes or mode") {
			t.Fatalf("CRLF was accepted although fixed -text attributes do not normalize it: %v", err)
		}
	})
}

func TestCompareCaptureToBaseRejectsStagedIndexDrift(t *testing.T) {
	root, revision, fixed, crlf := applyCaptureFixture(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Staged replacement\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitE2E(t, root, "add", "README.md")
	if err := os.WriteFile(filepath.Join(root, "README.md"), crlf, 0o644); err != nil {
		t.Fatal(err)
	}
	err := compareCaptureToBase(root, revision, captureApplyTarget(t, root), fixed, []string{"README.md"})
	if err == nil || !strings.Contains(err.Error(), "staged index content") {
		t.Fatalf("staged target content differing from the fixed base was not rejected: %v", err)
	}
}

func TestCompareCaptureToBaseRejectsStagedIndexDeletion(t *testing.T) {
	root, revision, fixed, _ := applyCaptureFixture(t)
	gitE2E(t, root, "rm", "--cached", "--", "README.md")
	err := compareCaptureToBase(root, revision, captureApplyTarget(t, root), fixed, []string{"README.md"})
	if err == nil || !strings.Contains(err.Error(), "index existence") {
		t.Fatalf("staged deletion of a fixed target path was not rejected: %v", err)
	}
}

func TestCompareCaptureToBaseRejectsAttributeDriftForAbsentTarget(t *testing.T) {
	root, revision, fixed, _ := applyCaptureFixture(t)
	writeE2E(t, root, ".gitattributes", "new.txt text\n")
	capture, err := guardedwrite.CaptureFiles(root, []string{"new.txt"})
	if err != nil {
		t.Fatal(err)
	}
	err = compareCaptureToBase(root, revision, capture, fixed, []string{"new.txt"})
	if err == nil || !strings.Contains(err.Error(), "Git attributes") {
		t.Fatalf("attribute drift for an absent target was not rejected: %v", err)
	}
}

func TestTargetDigestKeepsRawCaptureCASAcrossLineEndingChange(t *testing.T) {
	root, revision, fixed, crlf := applyCaptureFixture(t)
	preflightCapture := captureApplyTarget(t, root)
	preflightDigest, err := TargetDigest(preflightCapture)
	if err != nil {
		t.Fatal(err)
	}
	if err := compareCaptureToBase(root, revision, preflightCapture, fixed, []string{"README.md"}); err != nil {
		t.Fatalf("clean CRLF target failed preflight comparison: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), fixed.Files["README.md"], 0o644); err != nil {
		t.Fatal(err)
	}
	postChangeCapture := captureApplyTarget(t, root)
	postChangeDigest, err := TargetDigest(postChangeCapture)
	if err != nil {
		t.Fatal(err)
	}
	if preflightDigest == postChangeDigest {
		t.Fatal("raw CRLF-to-LF change did not invalidate the exact target capture token")
	}
	if !bytes.Equal(preflightCapture.Files["README.md"].Bytes, crlf) {
		t.Fatal("preflight capture was not retained as its original raw byte snapshot")
	}
}

func TestPreflightApplyAndApplyPreserveRawCRLFCAS(t *testing.T) {
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
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and run the declared checks.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
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
	stale, err := Apply(host, ProcessInvoker{}, root, request)
	if err == nil || len(stale.Written) != 0 {
		t.Fatalf("Apply did not reject raw-byte change after PreflightApply without writing: report=%+v err=%v", stale, err)
	}
	if err := os.WriteFile(firstTarget, lfBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := Apply(host, ProcessInvoker{}, root, request)
	if err != nil || applied.Status != "applied" {
		t.Fatalf("Apply rejected unchanged CRLF targets from preflight: status=%s err=%v", applied.Status, err)
	}
}

func TestApplyRechecksTargetIndexAtGuardedWriteBoundary(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	host := projectworkHost()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and run the declared checks.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
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
	host.Load = func(loadRoot, revision string) (*Project, error) {
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
	report, err := Apply(host, ProcessInvoker{}, root, request)
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
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	host := projectworkHost()
	plan, request := verifiedApplyRequest(t, host, root)
	lockPath := filepath.Join(root, ".markitect", "write.lock")
	if err := os.WriteFile(lockPath, []byte("held by another Host writer\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Apply(host, ProcessInvoker{}, root, request)
	assertApplyRejectedKeepsRunVerified(t, root, plan.ID, report, err)
	assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	retry, err := Apply(host, ProcessInvoker{}, root, request)
	if err != nil || retry.Status != StatusApplied {
		t.Fatalf("retry after releasing the writer lock did not apply: status=%s err=%v", retry.Status, err)
	}
}

func TestApplyStaleUnderWriterLockIsRejectedLikeEarlyStaleness(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	host := projectworkHost()
	plan, request := verifiedApplyRequest(t, host, root)
	modelPath := filepath.Join(root, ".markitect", "model", "orders", "statement.yaml")
	original, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := append(append([]byte(nil), original...), []byte("# concurrent editor save\n")...)
	if err := os.WriteFile(modelPath, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	early, err := Apply(host, ProcessInvoker{}, root, request)
	assertApplyRejectedKeepsRunVerified(t, root, plan.ID, early, err)
	if err := os.WriteFile(modelPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	// The same edit first observed by the guarded-write precondition, while
	// the Host writer lock is held, is the same rejection.
	writerLock := filepath.Join(root, ".markitect", "write.lock")
	baseLoad := host.Load
	injected := false
	guardedHost := host
	guardedHost.Load = func(loadRoot, revision string) (*Project, error) {
		if revision == "" && !injected {
			if _, statErr := os.Stat(writerLock); statErr == nil {
				injected = true
				if err := os.WriteFile(modelPath, edited, 0o644); err != nil {
					t.Errorf("inject concurrent edit: %v", err)
				}
			}
		}
		return baseLoad(loadRoot, revision)
	}
	late, err := Apply(guardedHost, ProcessInvoker{}, root, request)
	if !injected {
		t.Fatal("test did not reach the guarded-write precondition")
	}
	assertApplyRejectedKeepsRunVerified(t, root, plan.ID, late, err)
	if err := os.WriteFile(modelPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	retry, err := Apply(host, ProcessInvoker{}, root, request)
	if err != nil || retry.Status != StatusApplied {
		t.Fatalf("retry after reverting the transient edit did not apply: status=%s err=%v", retry.Status, err)
	}
}

func verifiedApplyRequest(t *testing.T, host Host, root string) (PlanRecord, ApplyRequest) {
	t.Helper()
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and run the declared checks.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), host, ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("Verify: status=%s err=%v", verified.Status, err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("PreflightApply: %v", err)
	}
	return plan, applyRequestFromPreflight(preflight)
}

// assertApplyRejectedKeepsRunVerified requires a refusal that wrote nothing
// to leave the run verified and journal no Apply receipt.
func assertApplyRejectedKeepsRunVerified(t *testing.T, root, runID string, report ApplyReport, applyErr error) {
	t.Helper()
	if applyErr == nil || report.Status != "rejected" || len(report.Written) != 0 || len(report.Journal) != 0 {
		t.Fatalf("zero-write Apply refusal was not a plain rejection: report=%+v err=%v", report, applyErr)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.readLatestState(runID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != StatusVerified {
		t.Fatalf("zero-write Apply refusal moved run to %s: %v", state.Status, applyErr)
	}
	dir, err := store.runDir(runID)
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := os.ReadDir(filepath.Join(dir, "apply"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(receipts) != 0 {
		t.Fatalf("zero-write Apply refusal journaled %d Apply receipts", len(receipts))
	}
}

func stageApplyTargetDrift(t *testing.T, root, target string, original []byte) {
	t.Helper()
	changed := append(append([]byte(nil), original...), []byte("staged-only apply drift\n")...)
	if err := os.WriteFile(target, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	gitE2E(t, root, "add", "--", filepath.ToSlash(strings.TrimPrefix(target, root+string(os.PathSeparator))))
	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatal(err)
	}
}
