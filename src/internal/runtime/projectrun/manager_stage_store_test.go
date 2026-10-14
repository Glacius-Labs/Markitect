package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

type stageStoreInvoker struct {
	calls     int
	raw       []byte
	reviewRaw []byte
	options   agentexec.RunOptions
	observe   func()
	usage     *agentexec.Usage
}

func (i *stageStoreInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return rawContentDigest([]byte("stage-fixture-fingerprint/" + config.Command + "/" + config.Model)), nil
}
func (i *stageStoreInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	i.options = options
	if i.observe != nil {
		i.observe()
	}
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	fp, _ := i.Fingerprint(config)
	usage := i.usage
	if usage == nil {
		in, out := int64(1), int64(1)
		usage = &agentexec.Usage{Source: "provider-reported", InputTokens: &in, OutputTokens: &out}
	}
	raw := i.raw
	if bytes.Contains(request.Context, []byte(`"kind":"projectrun-review/v1"`)) && i.reviewRaw != nil {
		raw = i.reviewRaw
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest, Role: agentexec.RoleExecutor, Outcome: agentexec.OutcomeProposed, ReportJSON: append(json.RawMessage(nil), raw...), Usage: usage}
	receipt := agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest, ContextDigest: rawContentDigest(request.Context), ConfigDigest: fp, Outcome: agentexec.OutcomeProposed, Usage: usage}
	return agentexec.RunResult{Response: response, Receipt: receipt}, nil
}

func stageStoreFixture(t *testing.T) (string, ManagerStageSpec, *stageStoreInvoker, Host) {
	t.Helper()
	base := testkit.TempDir(t)
	sourceRoot, targetRoot := filepath.Join(base, "source"), filepath.Join(base, "target")
	for _, dir := range []string{sourceRoot, targetRoot} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	invoker := &stageStoreInvoker{raw: []byte("{\n  \"report\" : { \"value\" : 1 },  \"unused\": true\n}")}
	fp, _ := invoker.Fingerprint(agentexec.Config{Command: "fixture", Model: "fixture"})
	budget := ManagerStageBudget{MaxStarts: 4, MaxRetries: 2, MaxDuration: time.Minute, MaxCostMicros: 1000}
	binding := ManagerStageBinding{SessionID: "fixture-session", SourceIdentity: rawContentDigest([]byte("source")), TargetIdentity: rawContentDigest([]byte("target")), SourceRevision: strings.Repeat("a", 40), DiscoveryDigest: rawContentDigest([]byte("discovery")), InitialTargetRevision: strings.Repeat("b", 40), TargetContextDigest: rawContentDigest([]byte("accepted")), Budget: budget}
	binding.SourceRootDigest = rawContentDigest([]byte(sourceRoot))
	binding.TargetRootDigest = rawContentDigest([]byte(targetRoot))
	acceptedDigest, _ := digest(projectmodel.ManagerContext{Manager: projectmodel.Manager{ID: "accepted-ancestor"}})
	validation := ManagerStageValidationBinding{AcceptedModelDigest: acceptedDigest, SourceRoot: sourceRoot, TargetRoot: targetRoot, SessionID: binding.SessionID, SessionDigest: rawContentDigest([]byte("session")), IterationID: "iteration", ManagerID: "root", AgentManagerID: "root", Phase: "propose", AcceptedTargetHead: binding.InitialTargetRevision, ContextDigest: rawContentDigest([]byte("context")), RequestContractDigest: rawContentDigest([]byte("contract")), ConfigFingerprint: fp, RuntimeFileDigest: rawContentDigest([]byte("runtime"))}
	agent := projectconfig.Agent{Command: "fixture", Model: "fixture", ProviderVersion: "fixture/1", Timeout: projectconfig.Duration(time.Second), MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20, Pricing: projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}}
	config := agentexec.Config{Command: agent.Command, Model: agent.Model, ProviderVersion: agent.ProviderVersion, Timeout: time.Second, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20}
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: binding.SourceRevision, ModelDigest: rawContentDigest([]byte("model")), ModulePin: "fixture-module", ProjectionID: "fixture-stage", ScopeIDs: []string{"root"}, PolicyIDs: []string{"report-only"}, Context: json.RawMessage(`{"kind":"projectadoption-manager-proposal/v1","phase":"propose"}`), Artifacts: []agentexec.Artifact{}}
	spec := ManagerStageSpec{SelectedSourceRoot: sourceRoot, TargetRoot: targetRoot, Binding: binding, Validation: validation, Agent: agent, Config: config, Request: request, Limits: budget, RuntimeLimits: projectconfig.Limits{MaxStarts: 4, MaxRetries: 2, MaxDuration: projectconfig.Duration(time.Minute), MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 2 << 20}}
	return sourceRoot, spec, invoker, Host{ValidateManagerStage: func(ManagerStageValidationBinding) error { return nil }}
}

func TestManagerStageStoresExactReportBeforeAttachAndRecoversWithoutDispatch(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, RunsPath)); !os.IsNotExist(err) {
		t.Fatal("preview initialized Runtime state")
	}
	invoker.observe = func() {
		store, _ := newManagerStageStore(root)
		report, err := store.readLatestState(preview.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Invocations) != 1 || report.Invocations[0].Attempt.Status != "started" || len(report.RoleStartReservations) != 1 {
			t.Fatal("dispatch preceded its durable start/reservation")
		}
	}
	result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newManagerStageStore(root)
	raw, err := readStageEvidence(store, result.RunID, result.Invocation.Attempt.ReportRef)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, invoker.raw) || !bytes.Equal(result.Result.Response.ReportJSON, invoker.raw) {
		t.Fatal("opaque report bytes were normalized")
	}
	if result.Invocation.CostMicros != 2 || !result.Invocation.CostKnown {
		t.Fatalf("Brownfield separate ceiling rounding changed: %+v", result.Invocation)
	}
	if len(invoker.options.InputRoots) != 2 || invoker.options.Workspace != nil || managerStagePathWithin(invoker.options.PrivateLogDirectory, root) || managerStagePathWithin(invoker.options.PrivateLogDirectory, spec.TargetRoot) {
		t.Fatal("Brownfield did not use its exact read-only audit/private-log policy")
	}
	invoker.observe = nil
	current, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := RunManagerStage(context.Background(), host, invoker, root, spec, current.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 1 || recovered.AttemptID != result.AttemptID || !bytes.Equal(recovered.Result.Response.ReportJSON, invoker.raw) {
		t.Fatal("receipt-before-CAS recovery dispatched or changed raw bytes")
	}
	accepted, err := CompleteManagerStage(root, result.RunID, result.AttemptID, recovered.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: rawContentDigest([]byte("typed-domain-validation"))})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != "accepted" {
		t.Fatal("validated completion not durable")
	}
	if _, err := CompleteManagerStage(root, result.RunID, result.AttemptID, recovered.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "changed"}); err == nil {
		t.Fatal("stale completion digest accepted")
	}
	runs, err := ListRuns(root)
	if err != nil || len(runs) != 0 {
		t.Fatalf("internal stage polluted ordinary runs: %+v %v", runs, err)
	}
	if _, err := Status(root, result.RunID); err == nil {
		t.Fatal("planless stage was presented as an ordinary plan")
	}
}

func TestManagerStageReservationBeforeInvocationLogCrashCannotReplay(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	store, _ := newManagerStageStore(root)
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	report, err := initializeManagerStage(store, spec.Binding)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := newManagerStageJournal(store, &report, spec.RuntimeLimits)
	if err != nil {
		t.Fatal(err)
	}
	_, err = journal.budget.ReserveRoot(context.Background(), spec.Validation.IterationID, spec.Validation.Phase, agentexec.RoleStartRequest{RequestID: "original-early-request", ParentSessionID: report.ID, Role: agentexec.RoleExecutor})
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
		t.Fatal("reservation-only uncertainty allowed a replacement attempt")
	}
	after, err := store.readLatestState(report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 0 || len(after.RoleStartReservations) != 1 || len(after.Invocations) != 0 {
		t.Fatal("early crash refilled/redispatched its charged reservation")
	}
}

func TestManagerStageEvidenceTamperBlocksRecovery(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newManagerStageStore(root)
	file, err := stageEvidencePath(store, result.RunID, *result.Invocation.Attempt.ReportRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(`{"tampered":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewManagerStage(invoker, root, spec); err == nil {
		t.Fatal("tampered opaque evidence passed authority readback")
	}
	if invoker.calls != 1 {
		t.Fatal("tamper triggered a provider call")
	}
}

func TestReadStageEvidenceFreshStoreRetainsDerivedRootBinding(t *testing.T) {
	root := testkit.TempDir(t)
	foreignRoot := testkit.TempDir(t)
	store, err := newManagerStageStore(root)
	if err != nil {
		t.Fatal(err)
	}
	foreignStore, err := newManagerStageStore(foreignRoot)
	if err != nil {
		t.Fatal(err)
	}
	const id = "0123456789abcdef0123456789abcdef"
	if err := ensureDirectory(filepath.Join(store.base, id), store); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(store.base, id, "reports"), store); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(foreignStore.base, id), foreignStore); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(foreignStore.base, id, "reports"), foreignStore); err != nil {
		t.Fatal(err)
	}
	payload := []byte("foreign-root evidence")
	ref, err := writeStageEvidence(foreignStore, id, "attempt", ".report.json", payload)
	if err != nil {
		t.Fatal(err)
	}
	misbound := *store
	misbound.base = foreignStore.base
	if got, err := readStageEvidence(&misbound, id, ref); err == nil || strings.Contains(string(got), string(payload)) || !strings.Contains(err.Error(), "runtime directory differs from owning root") {
		t.Fatalf("fresh read crossed its owning root: bytes=%q err=%v", got, err)
	}
}

func TestReadStageEvidenceHeldStoreReadsCurrentGuardedBytes(t *testing.T) {
	root := testkit.TempDir(t)
	store, err := newManagerStageStore(root)
	if err != nil {
		t.Fatal(err)
	}
	const id = "fedcba9876543210fedcba9876543210"
	if err := ensureDirectory(filepath.Join(store.base, id), store); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(store.base, id, "reports"), store); err != nil {
		t.Fatal(err)
	}
	payload := []byte("current held-store evidence")
	ref, err := writeStageEvidence(store, id, "attempt", ".report.json", payload)
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := unlock(); err != nil {
			t.Errorf("release held Runtime store: %v", err)
		}
	}()
	misbound := *store
	misbound.base = filepath.Join(root, ".MARKITECT", "records", "run")
	if got, err := readStageEvidence(&misbound, id, ref); err == nil || bytes.Equal(got, payload) || !strings.Contains(err.Error(), "runtime directory differs from owning root") {
		t.Fatalf("held read accepted a noncanonical base spelling: bytes=%q err=%v", got, err)
	}
	got, err := readStageEvidence(store, id, ref)
	if err != nil || !bytes.Equal(got, payload) || ref.Digest != rawContentDigest(payload) {
		t.Fatalf("held read changed accepted bytes/digest: bytes=%q err=%v", got, err)
	}
	path, err := stageEvidencePath(store, id, *ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered held-store evidence"), 0600); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if got, err := readStageEvidence(store, id, ref); err == nil || bytes.Equal(got, payload) {
			t.Fatalf("held read reused prior evidence after tamper (attempt %d): bytes=%q err=%v", attempt+1, got, err)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := readStageEvidence(store, id, ref); !errors.Is(err, errRuntimeRecordMissing) || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("held validated absence lost its private Runtime classification: %v", err)
	}
}

func readStageEvidenceOriginalSequence(store *runStore, id string, ref *ManagerStageEvidenceRef) ([]byte, error) {
	if ref == nil || !strings.HasPrefix(ref.Digest, "sha256:") {
		return nil, errors.New("Manager evidence reference is missing")
	}
	path, err := stageEvidencePath(store, id, *ref)
	if err != nil {
		return nil, err
	}
	data, err := readRecordBytes(path, store)
	if err != nil {
		return nil, err
	}
	if rawContentDigest(data) != ref.Digest {
		return nil, errors.New("Manager evidence byte digest differs")
	}
	return data, nil
}

func BenchmarkManagerStageEvidenceReadHeld(b *testing.B) {
	root := testkit.TempDir(b)
	store, err := newManagerStageStore(root)
	if err != nil {
		b.Fatal(err)
	}
	const id = "1234567890abcdef1234567890abcdef"
	if err := ensureDirectory(filepath.Join(store.base, id), store); err != nil {
		b.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(store.base, id, "reports"), store); err != nil {
		b.Fatal(err)
	}
	payload := []byte("paired held-store benchmark evidence")
	ref, err := writeStageEvidence(store, id, "attempt", ".report.json", payload)
	if err != nil {
		b.Fatal(err)
	}
	unlock, err := store.lock()
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := unlock(); err != nil {
			b.Errorf("release held Runtime store: %v", err)
		}
	}()
	if got, err := readStageEvidenceOriginalSequence(store, id, ref); err != nil || !bytes.Equal(got, payload) {
		b.Fatalf("original-sequence fixture is invalid: bytes=%q err=%v", got, err)
	}
	if got, err := readStageEvidence(store, id, ref); err != nil || !bytes.Equal(got, payload) {
		b.Fatalf("combined-read fixture is invalid: bytes=%q err=%v", got, err)
	}
	benchmarks := []struct {
		name string
		read func(*runStore, string, *ManagerStageEvidenceRef) ([]byte, error)
	}{
		{name: "baseline-first", read: readStageEvidenceOriginalSequence},
		{name: "candidate-first", read: readStageEvidence},
		{name: "candidate-second", read: readStageEvidence},
		{name: "baseline-second", read: readStageEvidenceOriginalSequence},
	}
	for _, benchmark := range benchmarks {
		benchmark := benchmark
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			var last []byte
			for i := 0; i < b.N; i++ {
				var err error
				last, err = benchmark.read(store, id, ref)
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			if !bytes.Equal(last, payload) {
				b.Fatal("held evidence read returned changed bytes")
			}
		})
	}
}

func unheldStageEvidenceFixture(t testing.TB) (*runStore, string, *ManagerStageEvidenceRef, []byte) {
	t.Helper()
	root := testkit.TempDir(t)
	store, err := newManagerStageStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if store.storage != nil {
		t.Fatal("unheld fixture unexpectedly owns a persistent Store")
	}
	const id = "1234567890abcdef1234567890abcdef"
	if err := ensureDirectory(filepath.Join(store.base, id), store); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(store.base, id, "reports"), store); err != nil {
		t.Fatal(err)
	}
	payload := []byte("same unheld Manager evidence bytes")
	ref, err := writeStageEvidence(store, id, "attempt", ".report.json", payload)
	if err != nil {
		t.Fatal(err)
	}
	return store, id, ref, payload
}

func TestReadStageEvidenceUnheldValidatesDigestAndAbsence(t *testing.T) {
	store, id, ref, payload := unheldStageEvidenceFixture(t)
	got, err := readStageEvidence(store, id, ref)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("unheld read changed accepted bytes: got=%q err=%v", got, err)
	}

	badDigest := *ref
	badDigest.Digest = rawContentDigest([]byte("different evidence"))
	if got, err := readStageEvidence(store, id, &badDigest); err == nil || len(got) != 0 || !strings.Contains(err.Error(), "Manager evidence byte digest differs") {
		t.Fatalf("unheld read accepted a mismatched digest: bytes=%q err=%v", got, err)
	}
	badPath := *ref
	badPath.Path = "reports/../invalid.json"
	if got, err := readStageEvidence(store, "", &badPath); err == nil || len(got) != 0 || !strings.Contains(err.Error(), "invalid Manager evidence reference") {
		t.Fatalf("reference validation did not precede ID/open checks: bytes=%q err=%v", got, err)
	}
	if got, err := readStageEvidence(store, "invalid", ref); err == nil || len(got) != 0 || !strings.Contains(err.Error(), "invalid run ID") {
		t.Fatalf("unheld read accepted an invalid run ID: bytes=%q err=%v", got, err)
	}

	path, err := stageEvidencePath(store, id, *ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if got, err := readStageEvidence(store, id, ref); !errors.Is(err, errRuntimeRecordMissing) || errors.Is(err, fs.ErrNotExist) || len(got) != 0 {
		t.Fatalf("validated missing leaf lost its private Runtime classification: bytes=%q err=%v", got, err)
	}
	if err := os.Remove(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
	if got, err := readStageEvidence(store, id, ref); !errors.Is(err, errRuntimeRecordMissing) || errors.Is(err, fs.ErrNotExist) || len(got) != 0 {
		t.Fatalf("missing reports directory was not mapped after guarded absence: bytes=%q err=%v", got, err)
	}
	if err := os.Remove(filepath.Join(store.base, id)); err != nil {
		t.Fatal(err)
	}
	if got, err := readStageEvidence(store, id, ref); !errors.Is(err, errRuntimeRecordMissing) || errors.Is(err, fs.ErrNotExist) || len(got) != 0 {
		t.Fatalf("missing run directory was not mapped after guarded absence: bytes=%q err=%v", got, err)
	}
	if store.storage != nil {
		t.Fatal("unheld read changed runStore storage ownership")
	}
}

func TestReadStageEvidenceUnheldPreservesNoncanonicalDirectoryErrors(t *testing.T) {
	store, id, ref, _ := unheldStageEvidenceFixture(t)
	misbound := *store
	misbound.base = filepath.Join(store.root, "legacy-record-base")
	reports := filepath.Join(misbound.base, id, "reports")

	assertSameFailure := func(label string) {
		t.Helper()
		baselineBytes, baselineErr := readStageEvidenceUnheldBaseline(&misbound, id, ref)
		candidateBytes, candidateErr := readStageEvidence(&misbound, id, ref)
		if len(baselineBytes) != 0 || len(candidateBytes) != 0 || baselineErr == nil || candidateErr == nil || baselineErr.Error() != candidateErr.Error() {
			t.Fatalf("%s changed old path/error precedence: baseline=(%q,%v) candidate=(%q,%v)", label, baselineBytes, baselineErr, candidateBytes, candidateErr)
		}
	}

	assertSameFailure("missing noncanonical reports path")
	if err := os.MkdirAll(reports, 0700); err != nil {
		t.Fatal(err)
	}
	assertSameFailure("existing noncanonical reports directory")
	if err := os.Remove(reports); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reports, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	assertSameFailure("non-directory reports path")
	if err := os.Remove(reports); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(testkit.TempDir(t), "target-directory")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, reports); err != nil {
		t.Skipf("noncanonical reports symlink creation unavailable: %v", err)
	}
	assertSameFailure("symlinked noncanonical reports path")
}

func TestReadStageEvidenceUnheldRejectsReportsReparseEntry(t *testing.T) {
	store, id, ref, _ := unheldStageEvidenceFixture(t)
	target := filepath.Join(testkit.TempDir(t), "target")
	if err := os.WriteFile(target, []byte("target"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(store.base, id, "reports", "unexpected.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if got, err := readStageEvidence(store, id, ref); err == nil || len(got) != 0 || !strings.Contains(err.Error(), "record list contains reparse entry") {
		t.Fatalf("unheld read ignored a reports-directory reparse entry: bytes=%q err=%v", got, err)
	}
}

func TestReadStageEvidenceUnheldRejectsReparseAncestor(t *testing.T) {
	store, id, ref, payload := unheldStageEvidenceFixture(t)
	marker := filepath.Join(store.root, ".markitect")
	moved := marker + "-original"
	if err := os.Rename(marker, moved); err != nil {
		t.Skipf("record-root ancestor replacement unavailable: %v", err)
	}
	if err := os.Symlink(moved, marker); err != nil {
		_ = os.Rename(moved, marker)
		t.Skipf("directory symlink creation unavailable: %v", err)
	}
	if got, err := readStageEvidence(store, id, ref); err == nil || bytes.Equal(got, payload) || !strings.Contains(err.Error(), "unsafe record parent directory") {
		t.Fatalf("unheld read accepted a reparse ancestor: bytes=%q err=%v", got, err)
	}
}

func TestReadStageEvidenceUnheldRejectsRootReplacement(t *testing.T) {
	store, id, ref, payload := unheldStageEvidenceFixture(t)
	moved := store.root + "-original"
	if err := os.Rename(store.root, moved); err != nil {
		t.Skipf("root directory replacement unavailable: %v", err)
	}
	// TempDir cleanup only knows store.root; remove the moved original explicitly.
	t.Cleanup(func() { _ = os.RemoveAll(moved) })
	if err := os.Mkdir(store.root, 0700); err != nil {
		t.Fatal(err)
	}
	if got, err := readStageEvidence(store, id, ref); err == nil || bytes.Equal(got, payload) || !strings.Contains(err.Error(), "Runtime record root identity changed") {
		t.Fatalf("unheld read crossed a replaced repository root: bytes=%q err=%v", got, err)
	}
	if store.storage != nil {
		t.Fatal("root replacement path changed runStore storage ownership")
	}
}

// This is the byte-for-byte old unheld read sequence, kept test-only as the
// benchmark baseline: stageEvidencePath -> readRecordBytes -> digest check.
func readStageEvidenceUnheldBaseline(store *runStore, id string, ref *ManagerStageEvidenceRef) ([]byte, error) {
	if ref == nil || !strings.HasPrefix(ref.Digest, "sha256:") {
		return nil, errors.New("Manager evidence reference is missing")
	}
	path, err := stageEvidencePath(store, id, *ref)
	if err != nil {
		return nil, err
	}
	data, err := readRecordBytes(path, store)
	if err != nil {
		return nil, err
	}
	if rawContentDigest(data) != ref.Digest {
		return nil, errors.New("Manager evidence byte digest differs")
	}
	return data, nil
}

func BenchmarkManagerStageEvidenceReadUnheld(b *testing.B) {
	store, id, ref, payload := unheldStageEvidenceFixture(b)
	if store.storage != nil {
		b.Fatal("unheld benchmark fixture unexpectedly owns a persistent Store")
	}
	if got, err := readStageEvidenceUnheldBaseline(store, id, ref); err != nil || !bytes.Equal(got, payload) {
		b.Fatalf("copied pre-candidate baseline is invalid: bytes=%q err=%v", got, err)
	}
	if got, err := readStageEvidence(store, id, ref); err != nil || !bytes.Equal(got, payload) {
		b.Fatalf("candidate fixture is invalid: bytes=%q err=%v", got, err)
	}
	benchmarks := []struct {
		name              string
		read              func(*runStore, string, *ManagerStageEvidenceRef) ([]byte, error)
		physicalOpen      float64
		rootBinding       float64
		pathComponentOpen float64
		cutoverCheck      float64
	}{
		{name: "baseline-first", read: readStageEvidenceUnheldBaseline, physicalOpen: 3, rootBinding: 0, pathComponentOpen: 14, cutoverCheck: 3},
		{name: "candidate-first", read: readStageEvidence, physicalOpen: 1, rootBinding: 2, pathComponentOpen: 14, cutoverCheck: 3},
		{name: "candidate-second", read: readStageEvidence, physicalOpen: 1, rootBinding: 2, pathComponentOpen: 14, cutoverCheck: 3},
		{name: "baseline-second", read: readStageEvidenceUnheldBaseline, physicalOpen: 3, rootBinding: 0, pathComponentOpen: 14, cutoverCheck: 3},
	}
	for _, benchmark := range benchmarks {
		benchmark := benchmark
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			b.ResetTimer()
			var last []byte
			for i := 0; i < b.N; i++ {
				var err error
				last, err = benchmark.read(store, id, ref)
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			if !bytes.Equal(last, payload) {
				b.Fatal("unheld evidence read returned changed bytes")
			}
			if store.storage != nil {
				b.Fatal("unheld benchmark read changed runStore storage ownership")
			}
			b.ReportMetric(1, "read/op")
			b.ReportMetric(benchmark.physicalOpen, "physical-open/op")
			b.ReportMetric(benchmark.rootBinding, "root-binding/op")
			b.ReportMetric(benchmark.pathComponentOpen, "path-component-open/op")
			b.ReportMetric(benchmark.cutoverCheck, "cutover-check/op")
		})
	}
}

func TestManagerStageNewEvidenceNeverSerializesProviderSecrets(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	secrets := []string{"secret-command-argument", "secret-model-option", "secret-environment-value", "secret-runtime-file-content", "secret-transport-config"}
	spec.Agent.Args = []string{secrets[0]}
	spec.Config.Args = []string{secrets[0]}
	spec.Agent.ModelOptions = map[string]any{"credential": secrets[1]}
	spec.Config.ModelOptions, _ = json.Marshal(spec.Agent.ModelOptions)
	const environmentName = "ARCH15_TEST_SYNTHETIC_CREDENTIAL"
	t.Setenv(environmentName, secrets[2])
	names := []string{environmentName}
	spec.Agent.Environment = names
	spec.Config.EnvironmentAllowlist = &names
	runtimeFile := filepath.Join(testkit.TempDir(t), "runtime.fixture")
	if err := os.WriteFile(runtimeFile, []byte(secrets[3]), 0600); err != nil {
		t.Fatal(err)
	}
	spec.Config.RuntimeFiles = []agentexec.RuntimeFile{{Path: runtimeFile}}
	spec.Agent.RuntimeFiles = spec.Config.RuntimeFiles
	spec.Config.TransportConfig = json.RawMessage(`{"credential":"` + secrets[4] + `"}`)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	public, err := json.Marshal(struct {
		Preview    ManagerStagePreview
		Invocation InvocationLog
	}{preview, result.Invocation})
	if err != nil {
		t.Fatal(err)
	}
	scan := func(data []byte) {
		for _, secret := range secrets {
			if bytes.Contains(data, []byte(secret)) {
				t.Fatal("provider secret entered new evidence projection")
			}
		}
	}
	scan(public)
	if err := filepath.WalkDir(filepath.Join(root, RunsPath), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err == nil {
			scan(data)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestManagerStageBudgetCannotRefillAndUnpricedUsageCannotRetry(t *testing.T) {
	t.Run("ceiling", func(t *testing.T) {
		root, spec, invoker, host := stageStoreFixture(t)
		spec.Binding.Budget.MaxStarts = 1
		spec.Limits.MaxStarts = 1
		spec.RuntimeLimits.MaxStarts = 1
		preview, err := PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err != nil {
			t.Fatal(err)
		}
		spec.Binding.Budget.MaxStarts = 20
		spec.Limits.MaxStarts = 20
		spec.Validation.IterationID = "next-iteration"
		preview, err = PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		if preview.RemainingStarts != 0 {
			t.Fatal("higher runtime refilled first session ceiling")
		}
		if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
			t.Fatal("frozen start ceiling bypassed")
		}
		if invoker.calls != 1 {
			t.Fatal("exhaustion dispatched")
		}
	})
	t.Run("usage", func(t *testing.T) {
		root, spec, invoker, host := stageStoreFixture(t)
		in, out := int64(1), int64(1)
		invoker.usage = &agentexec.Usage{Source: "estimated", InputTokens: &in, OutputTokens: &out}
		preview, err := PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
		if err == nil {
			t.Fatal("non-provider usage became known Brownfield cost")
		}
		spec.RetryOfAttemptID = result.AttemptID
		preview, err = PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
			t.Fatal("unpriced failure allowed retry")
		}
		if invoker.calls != 1 {
			t.Fatal("unpriced failure replayed")
		}
	})
}

func TestManagerStageTerminalReservationCrashReconcilesSamePermit(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newManagerStageStore(root)
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.readLatestState(manager.RunID)
	if err != nil {
		t.Fatal(err)
	}
	// Model the durable boundary immediately before permit.Update, keeping receipt.
	report.RoleStartReservations[0].Request.State = "requested"
	if err := persistState(store, &report); err != nil {
		t.Fatal(err)
	}
	unlock()
	preview, err = PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.readLatestState(manager.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 1 || recovered.AttemptID != manager.AttemptID || len(after.RoleStartReservations) != 1 || after.RoleStartReservations[0].Request.State != "completed" {
		t.Fatal("terminal recovery replaced/recharged or left its original permit pending")
	}
}

func TestManagerStageOverrunCannotRecoverOrComplete(t *testing.T) {
	for _, kind := range []string{"cost", "duration"} {
		t.Run(kind, func(t *testing.T) {
			root, spec, invoker, host := stageStoreFixture(t)
			if kind == "cost" {
				spec.Binding.Budget.MaxCostMicros, spec.Limits.MaxCostMicros = 1, 1
			} else {
				spec.Config.Timeout = time.Millisecond
				spec.Binding.Budget.MaxDuration, spec.Limits.MaxDuration = 2*time.Millisecond, 2*time.Millisecond
				invoker.observe = func() { time.Sleep(10 * time.Millisecond) }
			}
			preview, err := PreviewManagerStage(invoker, root, spec)
			if err != nil {
				t.Fatal(err)
			}
			original, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
			if err == nil || original.Invocation.Attempt == nil || original.Invocation.Attempt.ReportRef == nil {
				t.Fatal("overrun was accepted or lost durable receipt")
			}
			store, _ := newManagerStageStore(root)
			raw, err := readStageEvidence(store, original.RunID, original.Invocation.Attempt.ReportRef)
			if err != nil || !bytes.Equal(raw, invoker.raw) {
				t.Fatal("overrun original evidence was lost")
			}
			preview, err = PreviewManagerStage(invoker, root, spec)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
				t.Fatal("overrun recovered as attachable")
			}
			state, err := store.readLatestState(original.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := CompleteManagerStage(root, original.RunID, original.AttemptID, state.Digest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
				t.Fatal("overrun completed directly")
			}
			if invoker.calls != 1 {
				t.Fatal("overrun replayed provider")
			}
		})
	}
}

func TestManagerStageAcceptedCompletionIsImmutableAndIdempotent(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	spec.Binding.Budget.MaxCostMicros, spec.Limits.MaxCostMicros = 2, 2
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, manager.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "digest-A"})
	if err != nil {
		t.Fatal(err)
	}
	again, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, accepted.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "digest-A"})
	if err != nil || again.StateDigest != accepted.StateDigest {
		t.Fatal("same completion appended or failed")
	}
	if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, again.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "digest-B"}); err == nil {
		t.Fatal("accepted result rewritten")
	}
}

func TestManagerStageTightenedCeilingCannotRefillOrComplete(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	spec.Limits.MaxCostMicros = 1
	preview, err = PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
		t.Fatal("tightened overrun recovered")
	}
	spec.Limits.MaxCostMicros = 1000
	preview, err = PreviewManagerStage(invoker, root, spec)
	if err != nil || preview.RemainingCostMicros != 0 {
		t.Fatal("tightened budget refilled")
	}
	if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, preview.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
		t.Fatal("completion bypassed tightened budget")
	}
	if invoker.calls != 1 {
		t.Fatal("tightening replayed")
	}
}

func TestManagerStageStrictReportedUsage(t *testing.T) {
	one, zero, negative, huge := int64(1), int64(0), int64(-1), int64(1<<63-1)
	valid := func() *agentexec.Usage {
		return &agentexec.Usage{Source: "provider-reported", InputTokens: &one, OutputTokens: &one}
	}
	bounds := ManagerStageBounds{InputPriceMicrosPerMillion: 1, OutputPriceMicrosPerMillion: 1}
	cases := []struct {
		name            string
		usage           *agentexec.Usage
		known, overflow bool
	}{
		{"nil", nil, false, false}, {"valid", valid(), true, false},
		{"estimated", &agentexec.Usage{Source: "estimated", InputTokens: &one, OutputTokens: &one}, false, false},
		{"input-missing", &agentexec.Usage{Source: "provider-reported", OutputTokens: &one}, false, false},
		{"output-missing", &agentexec.Usage{Source: "provider-reported", InputTokens: &one}, false, false},
		{"negative", &agentexec.Usage{Source: "provider-reported", InputTokens: &negative, OutputTokens: &one}, false, false},
		{"tools", &agentexec.Usage{Source: "provider-reported", InputTokens: &one, OutputTokens: &one, ToolCalls: &one}, false, false},
		{"zero-tools", &agentexec.Usage{Source: "provider-reported", InputTokens: &one, OutputTokens: &one, ToolCalls: &zero}, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, known, overflow := managerStageCost(c.usage, bounds)
			if known != c.known || overflow != c.overflow {
				t.Fatal("usage trust boundary changed")
			}
		})
	}
	_, known, overflow := managerStageCost(&agentexec.Usage{Source: "provider-reported", InputTokens: &huge, OutputTokens: &huge}, ManagerStageBounds{InputPriceMicrosPerMillion: 1_000_000_000_000, OutputPriceMicrosPerMillion: 1_000_000_000_000})
	if known || !overflow {
		t.Fatal("overflow is attachable")
	}
}

func ordinaryInterruptedJournalFixture(t *testing.T) (*managerStageJournal, ManagerTask, *stageStoreInvoker, projectconfig.Runtime) {
	t.Helper()
	root, _, invoker, _ := stageStoreFixture(t)
	store, err := newManagerStageStore(root)
	if err != nil {
		t.Fatal(err)
	}
	task := ManagerTask{ID: "task", ManagerID: "root", State: "invoking", RepairPhase: "work", WorkAttempts: 1, Attempts: 1}
	report := &RunReport{APIVersion: APIVersion, ID: strings.Repeat("c", 32), PlanID: strings.Repeat("c", 32), Status: StatusRunning, Tasks: []ManagerTask{task}, RepairRounds: []RepairRound{{Number: 3, Status: "completed"}}}
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := store.createRun(report.ID); err != nil {
		t.Fatal(err)
	}
	limits := projectconfig.Limits{MaxStarts: 1}
	journal, err := newManagerStageJournal(store, report, limits)
	if err != nil {
		t.Fatal(err)
	}
	request := agentexec.Request{Role: agentexec.RoleExecutor, Context: json.RawMessage(`{"goal":"original immutable work"}`)}
	agent := projectconfig.Agent{Command: "process-fixture", Model: "fixture", ProviderVersion: "fixture/1", Timeout: projectconfig.Duration(time.Second), MaxStdoutBytes: 4096, MaxStderrBytes: 4096, Pricing: projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}}
	config, err := agent.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	log, err := ordinaryManagerAttemptLog(journal, invoker, config, agent, task, "work", nativeTaskAttemptID(task, "work"), request)
	if err != nil {
		t.Fatal(err)
	}
	log.Attempt.SafeFailure = "unchanged-safe-marker"
	start := agentexec.RoleStartRequest{RequestID: log.Attempt.ID, ParentSessionID: report.ID, Role: log.Role, State: "requested"}
	if _, err := journal.budget.ReserveRecordedTaskRoot(context.Background(), task.ID, "work", start); err != nil {
		t.Fatal(err)
	}
	if err := journal.start(log, request); err != nil {
		t.Fatal(err)
	}
	return journal, task, invoker, projectconfig.Runtime{Agents: map[string]projectconfig.Agent{task.ManagerID: agent}, Limits: limits}
}
func TestOrdinaryProcessResumeRetainsOneChargedUnknownAttemptWithoutDispatch(t *testing.T) {
	journal, task, invoker, runtime := ordinaryInterruptedJournalFixture(t)
	before := journal.snapshot()
	original, err := readStageEvidence(journal.store, before.ID, before.Invocations[0].Attempt.RequestRef)
	if err != nil {
		t.Fatal(err)
	}
	execution := &runExecution{ctx: context.Background(), store: journal.store, report: before, runtime: runtime, invoker: invoker}
	continued, report, err := execution.resumeExisting()
	if err == nil || continued || report.Status != StatusBlocked || invoker.calls != 0 {
		t.Fatal("lost process outcome dispatched or was accepted")
	}
	after, err := journal.store.readLatestState(before.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Invocations) != 1 || len(after.RoleStartReservations) != 1 || after.Tasks[0].State != "uncertain" || after.Invocations[0].Attempt.Status != "uncertain" || after.RoleStartReservations[0].Request.State != "unknown" {
		t.Fatal("original charged uncertainty was not durably reconciled")
	}
	recovered := *after.Invocations[0].Attempt
	recovered.Status = before.Invocations[0].Attempt.Status
	a, _ := digest(recovered)
	b, _ := digest(before.Invocations[0].Attempt)
	if a != b || !after.Invocations[0].Attempt.FinishedAt.IsZero() {
		t.Fatal("metadata reconciliation changed immutable attempt fields or fabricated finish time")
	}
	reservation := after.RoleStartReservations[0]
	reservation.Request.State = before.RoleStartReservations[0].Request.State
	if reservation.RecordedAt.Before(before.RoleStartReservations[0].RecordedAt) {
		t.Fatal("lifecycle update timestamp regressed")
	}
	reservation.RecordedAt = before.RoleStartReservations[0].RecordedAt
	a, _ = digest(reservation)
	b, _ = digest(before.RoleStartReservations[0])
	if a != b {
		t.Fatal("metadata reconciliation replaced, refunded or changed original ticket")
	}
	raw, err := readStageEvidence(journal.store, after.ID, after.Invocations[0].Attempt.RequestRef)
	if err != nil || !bytes.Equal(original, raw) {
		t.Fatal("original protected request changed")
	}
	if after.Invocations[0].Receipt.RunID != "" || after.Invocations[0].Attempt.ResponseRef != nil || after.Invocations[0].Attempt.ReportRef != nil {
		t.Fatal("unknown process acquired invented output")
	}
	repeated := &runExecution{ctx: context.Background(), store: journal.store, report: after, runtime: runtime, invoker: invoker}
	if continued, _, err := repeated.resumeExisting(); err == nil || continued {
		t.Fatal("second resume accepted or dispatched unknown process")
	}
	latest, err := journal.store.readLatestState(before.ID)
	if err != nil {
		t.Fatal(err)
	}
	a, _ = digest(latest.Invocations)
	b, _ = digest(after.Invocations)
	if a != b || len(latest.RoleStartReservations) != 1 || invoker.calls != 0 || repeated.roleBudget.Accounting().ObservedTotal != 1 {
		t.Fatal("second resume changed attempt authority or refilled start quota")
	}
	a, _ = digest(latest.RoleStartReservations)
	b, _ = digest(after.RoleStartReservations)
	if a != b {
		t.Fatal("idempotent second resume rewrote original reservation lifecycle")
	}
	request := agentexec.RoleStartRequest{RequestID: "replacement", ParentSessionID: latest.ID, Role: agentexec.RoleExecutor}
	if _, err := repeated.roleBudget.ReserveRoot(context.Background(), task.ID, "work", request); err == nil {
		t.Fatal("unknown original start granted a replacement ticket")
	}
}
func TestOrdinaryProcessUncertaintyRejectsIncompatibleAuthority(t *testing.T) {
	cases := []string{"missing-attempt", "duplicate-attempt", "wrong-task", "wrong-phase", "wrong-role", "wrong-digest", "bad-protected-bytes", "missing-reservation", "duplicate-reservation", "wrong-key", "wrong-request-id", "wrong-parent", "wrong-reservation-role", "terminal-reservation", "terminal-attempt", "receipt", "response", "finished", "wrong-repair-round"}
	for _, defect := range cases {
		t.Run(defect, func(t *testing.T) {
			journal, task, invoker, _ := ordinaryInterruptedJournalFixture(t)
			row := &journal.report.Invocations[0]
			reservation := &journal.report.RoleStartReservations[0]
			switch defect {
			case "missing-attempt":
				journal.report.Invocations = nil
			case "duplicate-attempt":
				journal.report.Invocations = append(journal.report.Invocations, *row)
			case "wrong-task":
				row.TaskID = "other"
			case "wrong-phase":
				row.Phase = "integrate"
			case "wrong-role":
				row.Role = "reviewer"
			case "wrong-digest":
				row.Attempt.RequestDigest = "sha256:wrong"
			case "bad-protected-bytes":
				path, err := stageEvidencePath(journal.store, journal.report.ID, *row.Attempt.RequestRef)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(`{"changed":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-reservation":
				journal.report.RoleStartReservations = nil
			case "duplicate-reservation":
				journal.report.RoleStartReservations = append(journal.report.RoleStartReservations, *reservation)
			case "wrong-key":
				reservation.Key = "different"
			case "wrong-request-id":
				reservation.Request.RequestID = "different"
			case "wrong-parent":
				reservation.Request.ParentSessionID = "different"
			case "wrong-reservation-role":
				reservation.Request.Role = "reviewer"
			case "terminal-reservation":
				reservation.Request.State = "completed"
			case "terminal-attempt":
				row.Attempt.Status = "accepted"
			case "receipt":
				row.Receipt.RunID = "invented"
			case "response":
				row.Attempt.ResponseRef = &ManagerStageEvidenceRef{Path: "reports/invented.json", Digest: "sha256:invented"}
			case "finished":
				row.Attempt.FinishedAt = time.Now().UTC()
			case "wrong-repair-round":
				journal.report.RepairRounds[0].Number++
			}
			before := journal.snapshot()
			if err := journal.markOrdinaryProcessAttemptUncertain(task, "work"); err == nil {
				t.Fatal("incompatible or ambiguous process authority reconciled")
			}
			a, _ := digest(before)
			b, _ := digest(journal.snapshot())
			if a != b || invoker.calls != 0 {
				t.Fatal("failed reconciliation mutated authority or dispatched")
			}
		})
	}
}
