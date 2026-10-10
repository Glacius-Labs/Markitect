package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

type recoveryWorkspaceFixture struct {
	root           string
	project        *Project
	request        projectworkspace.Request
	handle         projectworkspace.Handle
	invocation     agentexec.Invocation
	privateDir     string
	storage        string
	service        *projectworkspace.GitService
	journal        workspaceJournal
	journalPath    string
	recoveryHandle codexappserver.RecoveryHandle
}

type scriptedNativeRecoverer struct {
	result       agentexec.RunResult
	err          error
	recoverCalls int
	runCalls     int
}

type failOnceCloseWorkspaceService struct {
	*projectworkspace.GitService
	failClose bool
}

func (s *failOnceCloseWorkspaceService) Close(ctx context.Context, handle projectworkspace.Handle) error {
	if s.failClose {
		s.failClose = false
		return errors.New("simulated transient close failure")
	}
	return s.GitService.Close(ctx, handle)
}

func (i *scriptedNativeRecoverer) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	i.runCalls++
	return agentexec.RunResult{}, errors.New("new role run must not be called")
}

func (i *scriptedNativeRecoverer) Fingerprint(agentexec.Config) (string, error) {
	return "recovery-fingerprint", nil
}

func (i *scriptedNativeRecoverer) Recover(_ context.Context, _ agentexec.Config, handle codexappserver.RecoveryHandle, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.recoverCalls++
	if options.Workspace == nil || options.Workspace.ID != handle.Workspace.ID || options.PrivateLogDirectory == "" {
		return agentexec.RunResult{}, errors.New("recovery options lost the original workspace")
	}
	return i.result, i.err
}

func TestRecoverProjectAgentReopensOnlyTerminalOriginalAndCachesResult(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	if err := os.WriteFile(filepath.Join(fixture.handle.CWD, "src", "orders", "implementation.txt"), []byte("native recovered bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	invoker := &scriptedNativeRecoverer{result: recoveredWorkspaceResult(fixture, "completed")}
	result, found, err := recoverFixture(t, fixture, invoker)
	if err != nil || !found {
		t.Fatalf("recovery failed: found=%t err=%v", found, err)
	}
	if invoker.recoverCalls != 1 || invoker.runCalls != 0 || result.Delta == nil || result.Receipt.RunID != fixture.invocation.RunID {
		t.Fatalf("recovery did not inspect the original turn and harvest bytes: result=%+v invoker=%+v", result, invoker)
	}
	if _, err := os.Stat(fixture.handle.CWD); !os.IsNotExist(err) {
		t.Fatalf("successfully recovered workspace was not closed: %v", err)
	}
	closed := readWorkspaceJournal(t, fixture.root, fixture.handle.ID)
	if closed.State != "closed" || closed.CachedResult == nil || closed.CachedResult.Delta == nil {
		t.Fatalf("closed workspace did not retain the Host-validated cached result: %+v", closed)
	}
	second := &scriptedNativeRecoverer{}
	cached, found, err := recoverFixture(t, fixture, second)
	if err != nil || !found || second.recoverCalls != 0 || second.runCalls != 0 || cached.Delta == nil || cached.Delta.Digest != result.Delta.Digest {
		t.Fatalf("cached result caused provider work or failed validation: result=%+v found=%t err=%v invoker=%+v", cached, found, err, second)
	}
}

func TestRecoverProjectAgentRetriesCloseFromHarvestedCacheWithoutProviderWork(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	if err := os.WriteFile(filepath.Join(fixture.handle.CWD, "src", "orders", "implementation.txt"), []byte("terminal candidate bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	firstService, err := projectworkspace.NewGitService(fixture.storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	firstInvoker := &scriptedNativeRecoverer{result: recoveredWorkspaceResult(fixture, "completed")}
	_, found, err := recoverWithWorkspaceService(t, fixture, firstInvoker, &failOnceCloseWorkspaceService{GitService: firstService, failClose: true})
	if err == nil || !found || firstInvoker.recoverCalls != 1 || firstInvoker.runCalls != 0 {
		t.Fatalf("first recovery did not preserve the cached harvest after close failure: found=%t err=%v invoker=%+v", found, err, firstInvoker)
	}
	harvested := readWorkspaceJournal(t, fixture.root, fixture.handle.ID)
	if harvested.State != "harvested" || harvested.CachedResult == nil {
		t.Fatalf("close failure lost the durable cached result: %+v", harvested)
	}
	if _, err := os.Stat(fixture.handle.CWD); err != nil {
		t.Fatalf("close failure removed the workspace needed for safe retry: %v", err)
	}

	secondService, err := projectworkspace.NewGitService(fixture.storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	secondInvoker := &scriptedNativeRecoverer{}
	result, found, err := recoverWithWorkspaceService(t, fixture, secondInvoker, secondService)
	if err != nil || !found || secondInvoker.recoverCalls != 0 || secondInvoker.runCalls != 0 || result.Delta == nil {
		t.Fatalf("cleanup retry performed provider work or failed: result=%+v found=%t err=%v invoker=%+v", result, found, err, secondInvoker)
	}
	closed := readWorkspaceJournal(t, fixture.root, fixture.handle.ID)
	if closed.State != "closed" {
		t.Fatalf("cleanup retry did not persist closed state: %+v", closed)
	}
	if _, err := os.Stat(fixture.handle.CWD); !os.IsNotExist(err) {
		t.Fatalf("cleanup retry left the private workspace behind: %v", err)
	}
}

func TestRecoverProjectAgentUnknownLifecyclePreservesWorkspaceWithoutReplay(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	invoker := &scriptedNativeRecoverer{result: agentexec.RunResult{Receipt: agentexec.Receipt{
		RunID: fixture.invocation.RunID, InputDigest: fixture.invocation.InputDigest, ConfigDigest: "recovery-fingerprint",
		Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "session-original", TurnID: "turn-original", State: "unknown",
			StartRequests: []agentexec.RoleStartRequest{{RequestID: fixture.invocation.RunID, Role: agentexec.RoleExecutor, State: "unknown"}}},
	}}}
	result, found, err := recoverFixture(t, fixture, invoker)
	if err == nil || !found || invoker.recoverCalls != 1 || invoker.runCalls != 0 || result.Receipt.RunID != fixture.invocation.RunID {
		t.Fatalf("unknown turn was treated as complete or replayed: result=%+v found=%t err=%v calls=%+v", result, found, err, invoker)
	}
	if _, statErr := os.Stat(fixture.handle.CWD); statErr != nil {
		t.Fatalf("uncertain workspace was removed: %v", statErr)
	}
	journal := readWorkspaceJournal(t, fixture.root, fixture.handle.ID)
	if journal.State != "preserved" || journal.Receipt.RunID != fixture.invocation.RunID {
		t.Fatalf("unknown lifecycle evidence was not retained: %+v", journal)
	}
}

func TestRecoverPreparedWorkspaceWithTrustedDispatchedTurn(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	fixture.journal.State = "prepared"
	if err := persistWorkspaceJournal(fixture.journalPath, fixture.journal); err != nil {
		t.Fatal(err)
	}
	invoker := &scriptedNativeRecoverer{result: recoveredWorkspaceResult(fixture, "completed")}
	result, found, err := recoverFixture(t, fixture, invoker)
	if err != nil || !found || invoker.recoverCalls != 1 || invoker.runCalls != 0 || result.Delta == nil {
		t.Fatalf("trusted dispatched turn was not recovered from prepared workspace: found=%t err=%v result=%+v invoker=%+v", found, err, result, invoker)
	}
	if _, err := os.Stat(fixture.handle.CWD); !os.IsNotExist(err) {
		t.Fatalf("successfully recovered prepared workspace was not closed: %v", err)
	}
}

func TestRecoverPreparedWorkspaceWithKnownRunIDAndLegacyEmptyReceipt(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	fixture.journal.State = "prepared"
	fixture.journal.OwnerRunID = ""
	fixture.journal.Receipt = agentexec.Receipt{}
	if err := persistWorkspaceJournal(fixture.journalPath, fixture.journal); err != nil {
		t.Fatal(err)
	}
	service, err := projectworkspace.NewGitService(fixture.storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	in := &scriptedNativeRecoverer{result: recoveredWorkspaceResult(fixture, "completed")}
	binding := nativeRecoveryBinding{RunID: fixture.invocation.RunID, InputDigest: fixture.invocation.InputDigest, OwnerRunID: "fixture-run"}
	result, found, err := RecoverProjectAgent(context.Background(), Host{Workspaces: service, Load: projectwork.Load}, in,
		fixture.root, fixture.request.TaskID, agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20},
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, fixture.invocation.Request)
	if err != nil || !found || in.recoverCalls != 1 || in.runCalls != 0 || result.Receipt.RunID != fixture.invocation.RunID {
		t.Fatalf("known receipt run ID did not bind a legacy prepared journal to its exact trusted handle: result=%+v found=%t err=%v invoker=%+v", result, found, err, in)
	}
}

func TestRecoverPreparedWorkspaceWithoutReceiptUsesCurrentOwnerNotOlderSameInput(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	olderHandleID := addClosedSameInputSibling(t, fixture, "older-run")
	fixture.journal.State = "prepared"
	fixture.journal.Receipt = agentexec.Receipt{}
	if err := persistWorkspaceJournal(fixture.journalPath, fixture.journal); err != nil {
		t.Fatal(err)
	}
	service, err := projectworkspace.NewGitService(fixture.storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	in := &scriptedNativeRecoverer{result: recoveredWorkspaceResult(fixture, "completed")}
	binding := nativeRecoveryBinding{InputDigest: fixture.invocation.InputDigest, OwnerRunID: "fixture-run"}
	result, found, err := RecoverProjectAgent(context.Background(), Host{Workspaces: service, Load: projectwork.Load}, in,
		fixture.root, fixture.request.TaskID, agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20},
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, fixture.invocation.Request)
	if err != nil || !found || in.recoverCalls != 1 || in.runCalls != 0 || result.Receipt.RunID != fixture.invocation.RunID {
		t.Fatalf("prepared current-owner turn was not recovered exactly: result=%+v found=%t err=%#v invalidHandle=%t invoker=%+v", result, found, err, errors.Is(err, projectworkspace.ErrInvalidHandle), in)
	}
	if old := readWorkspaceJournal(t, fixture.root, olderHandleID); old.State != "closed" {
		t.Fatalf("older same-input workspace was selected or changed: %+v", old)
	}
}

func TestRecoverPreparedWorkspaceWithoutReceiptRejectsMissingCurrentOwnerHandle(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	fixture.journal.State = "prepared"
	fixture.journal.Receipt = agentexec.Receipt{}
	fixture.journal.OwnerRunID = "different-run"
	if err := persistWorkspaceJournal(fixture.journalPath, fixture.journal); err != nil {
		t.Fatal(err)
	}
	in := &scriptedNativeRecoverer{}
	binding := nativeRecoveryBinding{InputDigest: fixture.invocation.InputDigest, OwnerRunID: "current-run"}
	_, found, err := RecoverProjectAgent(context.Background(), Host{Workspaces: fixture.service, Load: projectwork.Load}, in,
		fixture.root, fixture.request.TaskID, agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20},
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, fixture.invocation.Request)
	if err == nil || !found || in.recoverCalls != 0 || in.runCalls != 0 {
		t.Fatalf("workspace without current-owner provenance was accepted: found=%t err=%v invoker=%+v", found, err, in)
	}
}

func TestRecoverPreparedWorkspaceWithoutReceiptRejectsDuplicateCurrentOwner(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	addClosedSameInputSibling(t, fixture, "fixture-run")
	fixture.journal.State = "prepared"
	fixture.journal.Receipt = agentexec.Receipt{}
	if err := persistWorkspaceJournal(fixture.journalPath, fixture.journal); err != nil {
		t.Fatal(err)
	}
	in := &scriptedNativeRecoverer{}
	binding := nativeRecoveryBinding{InputDigest: fixture.invocation.InputDigest, OwnerRunID: "fixture-run"}
	_, found, err := RecoverProjectAgent(context.Background(), Host{Workspaces: fixture.service, Load: projectwork.Load}, in,
		fixture.root, fixture.request.TaskID, agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20},
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, fixture.invocation.Request)
	if err == nil || !found || !strings.Contains(err.Error(), "multiple private workspace journals") || in.recoverCalls != 0 || in.runCalls != 0 {
		t.Fatalf("duplicate current-owner workspaces were guessed: found=%t err=%v invoker=%+v", found, err, in)
	}
}

func TestRecoverPreparedWorkspaceWithoutTrustedDispatchedTurnRemainsBlocked(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*codexappserver.RecoveryHandle)
	}{
		{name: "not dispatched", mutate: func(handle *codexappserver.RecoveryHandle) { handle.TurnDispatched = false; handle.TurnID = "" }},
		{name: "missing turn identity", mutate: func(handle *codexappserver.RecoveryHandle) { handle.TurnID = "" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newRecoveryWorkspaceFixture(t)
			fixture.journal.State = "prepared"
			if err := persistWorkspaceJournal(fixture.journalPath, fixture.journal); err != nil {
				t.Fatal(err)
			}
			replaceTrustedRecoveryHandles(t, fixture, test.mutate)
			invoker := &scriptedNativeRecoverer{}
			_, found, err := recoverFixture(t, fixture, invoker)
			if err == nil || !found || invoker.recoverCalls != 0 || invoker.runCalls != 0 {
				t.Fatalf("prepared workspace without a proven dispatched turn was inspected or replayed: found=%t err=%v invoker=%+v", found, err, invoker)
			}
			if _, statErr := os.Stat(fixture.handle.CWD); statErr != nil {
				t.Fatalf("unproven prepared workspace was removed: %v", statErr)
			}
			if got := readWorkspaceJournal(t, fixture.root, fixture.handle.ID).State; got != "prepared" {
				t.Fatalf("unproven prepared state was changed to %q", got)
			}
		})
	}
}

func TestRecoverPreparedWorkspaceWithMismatchedRequestRemainsBlocked(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	fixture.journal.State = "prepared"
	if err := persistWorkspaceJournal(fixture.journalPath, fixture.journal); err != nil {
		t.Fatal(err)
	}
	request := fixture.invocation.Request
	request.Context = json.RawMessage(`{"different":true}`)
	invoker := &scriptedNativeRecoverer{}
	binding := nativeRecoveryBinding{RunID: fixture.invocation.RunID, InputDigest: fixture.invocation.InputDigest, OwnerRunID: "fixture-run"}
	_, found, err := RecoverProjectAgent(context.Background(), Host{Workspaces: fixture.service, Load: projectwork.Load}, invoker, fixture.root, fixture.request.TaskID,
		agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20}, Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, request)
	if err == nil || !found || invoker.recoverCalls != 0 || invoker.runCalls != 0 {
		t.Fatalf("prepared workspace with a mismatched request reached native recovery: found=%t err=%v invoker=%+v", found, err, invoker)
	}
	if _, statErr := os.Stat(fixture.handle.CWD); statErr != nil {
		t.Fatalf("mismatched prepared workspace was removed: %v", statErr)
	}
}

func replaceTrustedRecoveryHandles(t *testing.T, fixture recoveryWorkspaceFixture, mutate func(*codexappserver.RecoveryHandle)) {
	t.Helper()
	workspaceJournalDir := filepath.Join(fixture.privateDir, nativeJournalDirectory, workspaceJournalKey(fixture.handle.ID))
	entries, err := os.ReadDir(workspaceJournalDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(workspaceJournalDir, entry.Name(), "handles")); err != nil {
			t.Fatal(err)
		}
	}
	journal, err := newNativeJournal(fixture.privateDir, fixture.handle.CWD, fixture.handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := journal.wrapOptions(codexappserver.Options{})
	if err := wrapped.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: fixture.invocation.RunID, Role: agentexec.RoleExecutor}); err != nil {
		t.Fatal(err)
	}
	handle := fixture.recoveryHandle
	mutate(&handle)
	if err := wrapped.OnHandle(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverProjectAgentRejectsChangedSourceBeforeQuery(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	writeE2E(t, fixture.root, "source-wip.txt", "changed after the original invocation\n")
	invoker := &scriptedNativeRecoverer{}
	_, found, err := recoverFixture(t, fixture, invoker)
	if err == nil || !found || invoker.recoverCalls != 0 || invoker.runCalls != 0 {
		t.Fatalf("stale source reached native recovery: found=%t err=%v invoker=%+v", found, err, invoker)
	}
	if _, statErr := os.Stat(fixture.handle.CWD); statErr != nil {
		t.Fatalf("stale-source candidate was deleted: %v", statErr)
	}
}

func TestRecoverProjectAgentRejectsConflictingSavedTurns(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	second := fixture.recoveryHandle
	second.TurnID = "turn-conflict"
	second.TurnDispatched = true
	journal, err := newNativeJournal(fixture.privateDir, fixture.handle.CWD, fixture.handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := journal.wrapOptions(codexappserver.Options{})
	if err := wrapped.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: fixture.invocation.RunID, Role: agentexec.RoleExecutor}); err != nil {
		t.Fatal(err)
	}
	if err := wrapped.OnHandle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	invoker := &scriptedNativeRecoverer{}
	_, found, err := recoverFixture(t, fixture, invoker)
	if err == nil || !found || !strings.Contains(err.Error(), "conflicting dispatched turn") || invoker.recoverCalls != 0 || invoker.runCalls != 0 {
		t.Fatalf("conflicting turn identities were guessed: found=%t err=%v invoker=%+v", found, err, invoker)
	}
}

func TestRecoverProjectAgentSelectsExactReceiptAmongRepeatedTaskJournals(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	for _, text := range []string{`{"prior":"one"}`, `{"prior":"two"}`} {
		addSiblingRecoveryJournal(t, fixture, text)
	}

	invoker := &scriptedNativeRecoverer{result: recoveredWorkspaceResult(fixture, "completed")}
	result, found, err := recoverFixture(t, fixture, invoker)
	if err != nil || !found || invoker.recoverCalls != 1 || invoker.runCalls != 0 || result.Receipt.RunID != fixture.invocation.RunID {
		t.Fatalf("exact original receipt did not select its journal: found=%t err=%v result=%+v invoker=%+v", found, err, result, invoker)
	}
	if got := readWorkspaceJournal(t, fixture.root, fixture.handle.ID); got.State != "closed" || got.CachedResult == nil {
		t.Fatalf("selected original workspace was not closed and cached: %+v", got)
	}
}

func TestRecoverProjectAgentRequiresExactReceiptWhenTaskJournalsRepeat(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	addSiblingRecoveryJournal(t, fixture, `{"prior":true}`)
	invoker := &scriptedNativeRecoverer{}
	binding := nativeRecoveryBinding{RunID: "another-run", InputDigest: fixture.invocation.InputDigest, OwnerRunID: "other-current-run"}
	_, found, err := RecoverProjectAgent(context.Background(), Host{Workspaces: fixture.service, Load: projectwork.Load}, invoker, fixture.root, fixture.request.TaskID,
		agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20}, Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, fixture.invocation.Request)
	if !found || err == nil || !strings.Contains(err.Error(), "no private workspace journal matches") || invoker.recoverCalls != 0 || invoker.runCalls != 0 {
		t.Fatalf("recovery guessed a repeated task journal without exact run identity: found=%t err=%v invoker=%+v", found, err, invoker)
	}
}

func TestRecoverProjectAgentRejectsDuplicateExactReceiptJournals(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	addDuplicateExactRecoveryJournal(t, fixture)
	invoker := &scriptedNativeRecoverer{}
	_, found, err := recoverFixture(t, fixture, invoker)
	if !found || err == nil || !strings.Contains(err.Error(), "multiple private workspace journals match") || invoker.recoverCalls != 0 || invoker.runCalls != 0 {
		t.Fatalf("recovery guessed between duplicate exact journals: found=%t err=%v invoker=%+v", found, err, invoker)
	}
}

func addSiblingRecoveryJournal(t *testing.T, fixture recoveryWorkspaceFixture, contextJSON string) {
	t.Helper()
	request := fixture.invocation.Request
	request.Context = json.RawMessage(contextJSON)
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := fixture.service.PrepareCandidate(context.Background(), fixture.request, nil, overlayDigest)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := newNativeJournal(fixture.privateDir, handle.CWD, handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	options := journal.wrapOptions(codexappserver.Options{})
	if err := options.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: invocation.RunID, Role: agentexec.RoleExecutor}); err != nil {
		t.Fatal(err)
	}
	recoveryHandle := codexappserver.RecoveryHandle{Protocol: "codex-app-server/0.162.0", Fingerprint: "recovery-fingerprint",
		Invocation: invocation, Workspace: handle, ThreadID: "thread-sibling", SessionID: "session-sibling", TurnID: "turn-sibling", TurnDispatched: true}
	if err := options.OnHandle(context.Background(), recoveryHandle); err != nil {
		t.Fatal(err)
	}
	delta, err := fixture.service.Harvest(context.Background(), handle)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.Close(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}}
	receipt := agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest,
		ConfigDigest: "recovery-fingerprint", Outcome: agentexec.OutcomeProposed, Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer,
			SessionID: "session-sibling", TurnID: "turn-sibling", State: "completed", StartRequests: []agentexec.RoleStartRequest{{RequestID: invocation.RunID, Role: invocation.Request.Role, State: "completed"}}}}
	cached := agentexec.RunResult{Response: response, Receipt: receipt, Delta: &delta}
	state := workspaceJournal{OwnerRunID: "prior-run", Request: fixture.request, Handle: handle, OverlayDigest: overlayDigest, State: "closed", Receipt: receipt, Delta: &delta, CachedResult: &cached}
	if err := persistWorkspaceJournal(filepath.Join(fixture.privateDir, "workspaces", handle.ID+".json"), state); err != nil {
		t.Fatal(err)
	}
}

func addClosedSameInputSibling(t *testing.T, fixture recoveryWorkspaceFixture, ownerRunID string) string {
	t.Helper()
	siblingService, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	invocation, _, err := agentexec.PrepareInvocation(fixture.invocation.Request)
	if err != nil {
		t.Fatal(err)
	}
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := siblingService.PrepareCandidate(context.Background(), fixture.request, nil, overlayDigest)
	if err != nil {
		t.Fatal(err)
	}
	native, err := newNativeJournal(fixture.privateDir, handle.CWD, handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	options := native.wrapOptions(codexappserver.Options{})
	if err := options.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: invocation.RunID, Role: agentexec.RoleExecutor}); err != nil {
		t.Fatal(err)
	}
	recoveryHandle := codexappserver.RecoveryHandle{Protocol: "codex-app-server/0.162.0", Fingerprint: "recovery-fingerprint",
		Invocation: invocation, Workspace: handle, ThreadID: "older-thread", SessionID: "older-session", TurnID: "older-turn", TurnDispatched: true}
	if err := options.OnHandle(context.Background(), recoveryHandle); err != nil {
		t.Fatal(err)
	}
	delta, err := siblingService.Harvest(context.Background(), handle)
	if err != nil {
		t.Fatal(err)
	}
	if err := siblingService.Close(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}}
	receipt := agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest,
		ConfigDigest: "recovery-fingerprint", ProviderVersion: "0.162.0", Outcome: agentexec.OutcomeProposed,
		Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "older-session", TurnID: "older-turn", State: "completed",
			StartRequests: []agentexec.RoleStartRequest{{RequestID: invocation.RunID, Role: agentexec.RoleExecutor, State: "completed"}}}}
	cached := agentexec.RunResult{Response: response, Receipt: receipt, Delta: &delta}
	state := workspaceJournal{OwnerRunID: ownerRunID, Request: fixture.request, Handle: handle, OverlayDigest: overlayDigest,
		State: "closed", Receipt: receipt, Delta: &delta, CachedResult: &cached}
	if err := persistWorkspaceJournal(filepath.Join(fixture.privateDir, "workspaces", handle.ID+".json"), state); err != nil {
		t.Fatal(err)
	}
	return handle.ID
}

func addDuplicateExactRecoveryJournal(t *testing.T, fixture recoveryWorkspaceFixture) {
	t.Helper()
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := fixture.service.PrepareCandidate(context.Background(), fixture.request, nil, overlayDigest)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := newNativeJournal(fixture.privateDir, handle.CWD, handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	options := journal.wrapOptions(codexappserver.Options{})
	if err := options.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: fixture.invocation.RunID, Role: agentexec.RoleExecutor}); err != nil {
		t.Fatal(err)
	}
	recoveryHandle := codexappserver.RecoveryHandle{Protocol: "codex-app-server/0.162.0", Fingerprint: "recovery-fingerprint",
		Invocation: fixture.invocation, Workspace: handle, ThreadID: "thread-duplicate", SessionID: "session-duplicate", TurnID: "turn-duplicate", TurnDispatched: true}
	if err := options.OnHandle(context.Background(), recoveryHandle); err != nil {
		t.Fatal(err)
	}
	state := workspaceJournal{OwnerRunID: "prior-run", Request: fixture.request, Handle: handle, OverlayDigest: overlayDigest, State: "preserved",
		Receipt: agentexec.Receipt{RunID: fixture.invocation.RunID, InputDigest: fixture.invocation.InputDigest}}
	if err := persistWorkspaceJournal(filepath.Join(fixture.privateDir, "workspaces", handle.ID+".json"), state); err != nil {
		t.Fatal(err)
	}
}

func TestSelectRecoveryHandleChoosesMostProgressForOneOriginalInvocation(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	beforeDispatch := fixture.recoveryHandle
	beforeDispatch.TurnID = ""
	beforeDispatch.TurnDispatched = false
	selected, err := selectRecoveryHandle([]codexappserver.RecoveryHandle{beforeDispatch, fixture.recoveryHandle}, fixture.handle, "recovery-fingerprint")
	if err != nil || selected.TurnID != fixture.recoveryHandle.TurnID || !selected.TurnDispatched {
		t.Fatalf("recovery did not select the most advanced saved handle: selected=%+v err=%v", selected, err)
	}
}

func newRecoveryWorkspaceFixture(t *testing.T) recoveryWorkspaceFixture {
	t.Helper()
	root, project, _ := workspaceBridgeBase(t)
	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := projectworkspace.InspectRepository(context.Background(), root, project.Revision)
	if err != nil {
		t.Fatal(err)
	}
	request := projectworkspace.Request{RepositoryRoot: root, RepositoryIdentity: identity.Digest, BaseSHA: project.Revision,
		OverlayDigest: binding.OverlayDigest, TaskID: "orders-work", AllowedPaths: []string{"src/orders"}}
	storage := t.TempDir()
	service, err := projectworkspace.NewGitService(storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := service.PrepareCandidate(context.Background(), request, nil, overlayDigest)
	if err != nil {
		t.Fatal(err)
	}
	agentRequest := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: project.Revision,
		ModelDigest: "sha256:" + strings.Repeat("a", 64), ModulePin: "fixture-module", ProjectionID: "orders", Context: json.RawMessage(`{}`),
	}
	invocation, _, err := agentexec.PrepareInvocation(agentRequest)
	if err != nil {
		t.Fatal(err)
	}
	privateDir := filepath.Join(root, ".markitect", "runs", "private")
	journal, err := newNativeJournal(privateDir, handle.CWD, handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := journal.wrapOptions(codexappserver.Options{})
	if err := wrapped.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: invocation.RunID, Role: agentexec.RoleExecutor}); err != nil {
		t.Fatal(err)
	}
	recoveryHandle := codexappserver.RecoveryHandle{Protocol: "codex-app-server/0.162.0", Fingerprint: "recovery-fingerprint",
		Invocation: invocation, Workspace: handle, ThreadID: "thread-original", SessionID: "session-original", TurnID: "turn-original", TurnDispatched: true}
	if err := wrapped.OnHandle(context.Background(), recoveryHandle); err != nil {
		t.Fatal(err)
	}
	workspace := workspaceJournal{OwnerRunID: "fixture-run", Request: request, Handle: handle, OverlayDigest: overlayDigest, State: "preserved",
		Receipt: agentexec.Receipt{RunID: invocation.RunID, InputDigest: invocation.InputDigest}}
	journalPath := filepath.Join(privateDir, "workspaces", handle.ID+".json")
	if err := persistWorkspaceJournal(journalPath, workspace); err != nil {
		t.Fatal(err)
	}
	return recoveryWorkspaceFixture{root: root, project: project, request: request, handle: handle, invocation: invocation, privateDir: privateDir,
		storage: storage, service: service, journal: workspace, journalPath: journalPath, recoveryHandle: recoveryHandle}
}

func recoveredWorkspaceResult(f recoveryWorkspaceFixture, state string) agentexec.RunResult {
	startState := state
	if state == "completed" {
		startState = "completed"
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: f.invocation.RunID, Nonce: f.invocation.Nonce,
		Role: f.invocation.Request.Role, InputDigest: f.invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{{Path: "src/orders/implementation.txt", Mode: "0644", Content: "recovered proposal"}},
		EvidenceRefs:   []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}}
	return agentexec.RunResult{Response: response, Receipt: agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: f.invocation.RunID,
		InputDigest: f.invocation.InputDigest, ConfigDigest: "recovery-fingerprint", ProviderVersion: "0.162.0", Outcome: agentexec.OutcomeProposed,
		Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "session-original", TurnID: "turn-original", State: state,
			Accounting: "partial", StartRequests: []agentexec.RoleStartRequest{{RequestID: f.invocation.RunID, Role: agentexec.RoleExecutor, State: startState}}}}}
}

func recoverFixture(t *testing.T, fixture recoveryWorkspaceFixture, invoker *scriptedNativeRecoverer) (agentexec.RunResult, bool, error) {
	t.Helper()
	service, err := projectworkspace.NewGitService(fixture.storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return recoverWithWorkspaceService(t, fixture, invoker, service)
}

func recoverWithWorkspaceService(t *testing.T, fixture recoveryWorkspaceFixture, invoker Invoker, service projectworkspace.Service) (agentexec.RunResult, bool, error) {
	t.Helper()
	config := agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20}
	binding := nativeRecoveryBinding{RunID: fixture.invocation.RunID, InputDigest: fixture.invocation.InputDigest, OwnerRunID: "fixture-run"}
	return RecoverProjectAgent(context.Background(), Host{Workspaces: service, Load: projectwork.Load}, invoker, fixture.root, fixture.request.TaskID,
		config, Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, fixture.invocation.Request)
}

func TestRecoverProjectAgentRejectsDifferentCurrentRequestBeforeInspection(t *testing.T) {
	fixture := newRecoveryWorkspaceFixture(t)
	invoker := &scriptedNativeRecoverer{result: recoveredWorkspaceResult(fixture, "completed")}
	request := fixture.invocation.Request
	request.SourceRevision = "different-selected-source"
	binding := nativeRecoveryBinding{RunID: fixture.invocation.RunID, InputDigest: fixture.invocation.InputDigest, OwnerRunID: "fixture-run"}
	_, found, err := RecoverProjectAgent(context.Background(), Host{Workspaces: fixture.service, Load: projectwork.Load}, invoker, fixture.root, fixture.request.TaskID,
		agentexec.Config{Transport: TransportCodexAppServer, Timeout: 20}, Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, binding, request)
	if !found || err == nil || invoker.recoverCalls != 0 || invoker.runCalls != 0 {
		t.Fatalf("mismatched current request inspected/replayed original: found=%t err=%v invoker=%+v", found, err, invoker)
	}
	if _, err := os.Stat(fixture.handle.CWD); err != nil {
		t.Fatalf("mismatched request closed owned workspace: %v", err)
	}
}
