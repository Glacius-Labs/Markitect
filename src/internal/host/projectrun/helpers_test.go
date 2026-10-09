package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func TestHelperDynamicToolIsStableAndStrict(t *testing.T) {
	first, second := HelperDynamicTool(), HelperDynamicTool()
	if first.Name != HelperToolName || first.Name != second.Name || first.Type != "function" || string(first.InputSchema) != string(second.InputSchema) {
		t.Fatalf("helper tool schema is not stable: %#v %#v", first, second)
	}
	var schema map[string]any
	if err := json.Unmarshal(first.InputSchema, &schema); err != nil || schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Fatalf("helper schema is not a strict object: %#v err=%v", schema, err)
	}
}

func TestHelperRunsFreshScopedChildAndAppliesOnlyObservedDelta(t *testing.T) {
	fixture := newHelperFixture(t)
	var capturedOptions codexappserver.Options
	var attached agentexec.RoleStartRequest
	behavior := helperFakeBehavior{path: "src/child.go", content: []byte("package child\n// helper bytes\n"), mode: "0644"}
	session := fixture.session(t, func(options codexappserver.Options) Invoker {
		capturedOptions = options
		return &helperFakeInvoker{options: options, behavior: behavior}
	})
	fixture.reserver.onAttach = func(request agentexec.RoleStartRequest) { attached = request }
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	result, err := session.HandleToolCall(context.Background(), helperCall("call-1", `{"task":"add one helper-owned source file","paths":["src/child.go"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || strings.Contains(result.Text, string(behavior.content)) || !strings.Contains(result.Text, "status=proposed") {
		t.Fatalf("helper returned unbounded or raw child output: %#v", result)
	}
	got, err := os.ReadFile(filepath.Join(fixture.repo, "src", "child.go"))
	if err != nil || string(got) != string(behavior.content) {
		t.Fatalf("observed child bytes were not applied to parent: %q err=%v", got, err)
	}
	if capturedOptions.BeforeStart == nil || len(capturedOptions.DynamicTools) != 1 || capturedOptions.DynamicTools[0].Name == HelperToolName {
		t.Fatalf("child retained recursive Host helper tool or lost start hook: %#v", capturedOptions)
	}
	if attached.RequestID == "" || fixture.reserver.attachCount != 1 {
		t.Fatalf("child protocol root was not attached to one Host permit: %#v / %d", attached, fixture.reserver.attachCount)
	}
	requests := session.Requests()
	if len(requests) != 1 || requests[0].RequestID != "helper-call-1" || requests[0].ParentSessionID != "parent-session" || requests[0].SessionID != "child-session" || requests[0].State != "completed" {
		t.Fatalf("Host-owned helper request was not bound/accounted: %#v", requests)
	}
	if len(session.Receipts()) != 1 || fixture.reserver.reserveCount != 1 || fixture.reserver.updateCount != 2 || len(fixture.reserver.updateStates) != 2 || fixture.reserver.updateStates[0] != "unknown" || fixture.reserver.updateStates[1] != "completed" {
		t.Fatalf("helper receipt/reservation counts differ: receipts=%d reserve=%d update=%d", len(session.Receipts()), fixture.reserver.reserveCount, fixture.reserver.updateCount)
	}
}

func TestHelperCountsMalformedAndOutOfScopeRequestsBeforeValidation(t *testing.T) {
	fixture := newHelperFixture(t)
	invocations := 0
	session := fixture.session(t, func(options codexappserver.Options) Invoker {
		invocations++
		return &helperFakeInvoker{options: options, behavior: helperFakeBehavior{path: "src/child.go", content: []byte("x"), mode: "0644"}}
	})
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id   string
		args string
	}{
		{id: "bad-json", args: `{"task":`},
		{id: "out-of-scope", args: `{"task":"change docs","paths":["docs/"]}`},
	} {
		if _, err := session.HandleToolCall(context.Background(), helperCall(tc.id, tc.args)); err == nil {
			t.Fatalf("helper request %s unexpectedly succeeded", tc.id)
		}
	}
	if invocations != 0 || fixture.reserver.reserveCount != 2 || len(session.Requests()) != 2 {
		t.Fatalf("invalid requests were not counted before validation: invocations=%d reserves=%d requests=%#v", invocations, fixture.reserver.reserveCount, session.Requests())
	}
	for _, request := range session.Requests() {
		if request.State != "failed" {
			t.Fatalf("failed prevalidation request was dropped or left requested: %#v", request)
		}
	}
}

func TestHelperAppliesBinaryModifyDeleteAndRenameDeltas(t *testing.T) {
	t.Run("binary modify", func(t *testing.T) {
		fixture := newHelperFixture(t)
		binary := []byte{0, 1, 2, 0xff, '\n'}
		session := fixture.session(t, func(options codexappserver.Options) Invoker {
			return &helperFakeInvoker{options: options, behavior: helperFakeBehavior{path: "src/main.go", content: binary, mode: "0644"}}
		})
		if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
			t.Fatal(err)
		}
		if _, err := session.HandleToolCall(context.Background(), helperCall("binary-modify", `{"task":"replace source bytes","paths":["src/main.go"]}`)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(fixture.repo, "src", "main.go"))
		if err != nil || !bytes.Equal(got, binary) {
			t.Fatalf("binary modification differs: %v %v", got, err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		fixture := newHelperFixture(t)
		session := fixture.session(t, func(options codexappserver.Options) Invoker {
			return &helperFakeInvoker{options: options, behavior: helperFakeBehavior{deletePath: "src/main.go"}}
		})
		if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
			t.Fatal(err)
		}
		if _, err := session.HandleToolCall(context.Background(), helperCall("delete", `{"task":"remove obsolete source","paths":["src/main.go"]}`)); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(fixture.repo, "src", "main.go")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("delete was not applied: %v", err)
		}
	})
	t.Run("rename", func(t *testing.T) {
		fixture := newHelperFixture(t)
		fixture.options.ParentScope.ExcludedWritePaths = nil
		session := fixture.session(t, func(options codexappserver.Options) Invoker {
			return &helperFakeInvoker{options: options, behavior: helperFakeBehavior{path: "src/renamed.go", renameOld: "src/main.go", content: []byte("package src\n"), mode: "0644"}}
		})
		if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
			t.Fatal(err)
		}
		if _, err := session.HandleToolCall(context.Background(), helperCall("rename", `{"task":"rename source","paths":["src/"]}`)); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(fixture.repo, "src", "main.go")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("rename source remains: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(fixture.repo, "src", "renamed.go"))
		if err != nil || string(got) != "package src\n" {
			t.Fatalf("rename destination differs: %q %v", got, err)
		}
	})
}

func TestHelperRejectsForeignParentThreadBeforeReservation(t *testing.T) {
	fixture := newHelperFixture(t)
	session := fixture.session(t, func(options codexappserver.Options) Invoker { return &helperFakeInvoker{options: options} })
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	call := helperCall("foreign-call", `{"task":"do work","paths":["src/child.go"]}`)
	call.ThreadID = "foreign-thread"
	if _, err := session.HandleToolCall(context.Background(), call); err == nil {
		t.Fatal("foreign-thread helper call was accepted")
	}
	if fixture.reserver.reserveCount != 0 || len(session.Requests()) != 0 {
		t.Fatal("foreign thread consumed the parent's helper reservation")
	}
}

func TestHelperRejectsUnobservedModelWritesAndPreservesChildWorkspace(t *testing.T) {
	fixture := newHelperFixture(t)
	behavior := helperFakeBehavior{path: "src/child.go", content: []byte("actual bytes"), modelContent: "claimed bytes", mode: "0644"}
	session := fixture.session(t, func(options codexappserver.Options) Invoker {
		return &helperFakeInvoker{options: options, behavior: behavior}
	})
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	if _, err := session.HandleToolCall(context.Background(), helperCall("contradiction", `{"task":"write file","paths":["src/child.go"]}`)); err == nil || !strings.Contains(err.Error(), "contradicts observed child") {
		t.Fatalf("contradictory model write accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fixture.repo, "src", "child.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("untrusted model bytes reached parent workspace: err=%v", err)
	}
	entries, err := os.ReadDir(fixture.storage)
	if err != nil || len(entries) == 0 {
		t.Fatalf("child workspace was deleted despite failed delta review: entries=%v err=%v", entries, err)
	}
}

func TestHelperUnknownChildStatePreservesWorkspaceWithoutHarvestOrClose(t *testing.T) {
	fixture := newHelperFixture(t)
	behavior := helperFakeBehavior{path: "src/child.go", content: []byte("x"), mode: "0644", lifecycleState: "running"}
	session := fixture.session(t, func(options codexappserver.Options) Invoker {
		return &helperFakeInvoker{options: options, behavior: behavior}
	})
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	if _, err := session.HandleToolCall(context.Background(), helperCall("uncertain", `{"task":"write file","paths":["src/child.go"]}`)); err == nil || !strings.Contains(err.Error(), "termination is unknown") {
		t.Fatalf("running child did not fail closed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fixture.repo, "src", "child.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unknown child bytes reached parent workspace: err=%v", err)
	}
	entries, err := os.ReadDir(fixture.storage)
	if err != nil || len(entries) == 0 {
		t.Fatalf("unknown child workspace was removed: entries=%v err=%v", entries, err)
	}
	if len(session.Requests()) != 1 || session.Requests()[0].State != "unknown" {
		t.Fatalf("unknown helper start not retained: %#v", session.Requests())
	}
}

func TestHelperFailedParentLifecycleKeepsUnresolvedChildBlocking(t *testing.T) {
	fixture := newHelperFixture(t)
	behavior := helperFakeBehavior{path: "src/child.go", content: []byte("x"), mode: "0644", lifecycleState: "failed", nestedLifecycleState: "started"}
	service := &helperTrackingWorkspaceService{candidateWorkspaceService: fixture.options.Workspaces.(candidateWorkspaceService)}
	fixture.options.Workspaces = service
	session := fixture.session(t, func(options codexappserver.Options) Invoker {
		return &helperFakeInvoker{options: options, behavior: behavior}
	})
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	_, err := session.HandleToolCall(context.Background(), helperCall("failed-parent-active-child", `{"task":"write child source","paths":["src/child.go"]}`))
	if err == nil || !strings.Contains(err.Error(), "termination is unknown") {
		t.Fatalf("failed parent lifecycle did not preserve unresolved child: %v", err)
	}
	if service.closeCalls != 0 {
		t.Fatalf("workspace with unresolved child was closed: calls=%d", service.closeCalls)
	}
	if requests := session.Requests(); len(requests) != 1 || requests[0].State != "unknown" {
		t.Fatalf("terminal parent state hid nonterminal child state: %#v", requests)
	}
	if _, err := os.Stat(filepath.Join(fixture.repo, "src", "child.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unresolved child delta reached the parent workspace: %v", err)
	}
}

func TestHelperAppliedDeltaCleanupAndPersistenceFailuresRemainRecoverable(t *testing.T) {
	fixture := newHelperFixture(t)
	closeFailure := errors.New("injected child cleanup failure")
	reservationFailure := errors.New("injected reservation persistence failure")
	service := &helperTrackingWorkspaceService{candidateWorkspaceService: fixture.options.Workspaces.(candidateWorkspaceService), closeErr: closeFailure}
	fixture.options.Workspaces = service
	fixture.reserver.updateErr = reservationFailure
	injected := helperFakeBehavior{path: "src/child.go", content: []byte("package child\n// applied before cleanup\n"), mode: "0644"}
	session := fixture.session(t, func(options codexappserver.Options) Invoker {
		return &helperFakeInvoker{options: options, behavior: injected}
	})
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	_, err := session.HandleToolCall(context.Background(), helperCall("cleanup-pending", `{"task":"write child source","paths":["src/child.go"]}`))
	if !errors.Is(err, closeFailure) || !errors.Is(err, reservationFailure) {
		t.Fatalf("cleanup and reservation failures were not surfaced: %v", err)
	}
	if service.closeCalls != 1 || service.lastHandle.ID == "" {
		t.Fatalf("terminal helper cleanup was not attempted with its known handle: calls=%d handle=%+v", service.closeCalls, service.lastHandle)
	}
	if _, err := os.Stat(service.lastHandle.CWD); err != nil {
		t.Fatalf("failed cleanup discarded the child workspace needed for recovery: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(fixture.repo, "src", "child.go")); err != nil || string(got) != string(injected.content) {
		t.Fatalf("observed delta was not applied before cleanup: %q err=%v", got, err)
	}
	if requests := session.Requests(); len(requests) != 1 || requests[0].State != "unknown" {
		t.Fatalf("cleanup uncertainty was overwritten by terminal provider lifecycle: %#v", requests)
	}
	if len(fixture.reserver.updateStates) < 2 || fixture.reserver.updateStates[len(fixture.reserver.updateStates)-1] != "unknown" {
		t.Fatalf("Host reservation does not preserve cleanup-pending state: %v", fixture.reserver.updateStates)
	}
	assertHelperJournalState(t, fixture.private, "cleanup-pending", service.lastHandle.ID)
}

func TestHelperFailRequestSurfacesCleanupFailureAndPersistsHandle(t *testing.T) {
	fixture := newHelperFixture(t)
	closeFailure := errors.New("injected prelaunch cleanup failure")
	service := &helperTrackingWorkspaceService{candidateWorkspaceService: fixture.options.Workspaces.(candidateWorkspaceService), closeErr: closeFailure}
	fixture.options.Workspaces = service
	session := fixture.session(t, func(codexappserver.Options) Invoker { return nil })
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	_, err := session.HandleToolCall(context.Background(), helperCall("prelaunch-cleanup", `{"task":"write child source","paths":["src/child.go"]}`))
	if !errors.Is(err, closeFailure) || !strings.Contains(err.Error(), "invoker factory returned nil") {
		t.Fatalf("prelaunch cleanup error was discarded: %v", err)
	}
	if service.closeCalls != 1 || service.lastHandle.ID == "" {
		t.Fatalf("prepared helper workspace cleanup was not attempted: calls=%d handle=%+v", service.closeCalls, service.lastHandle)
	}
	if requests := session.Requests(); len(requests) != 1 || requests[0].State != "unknown" {
		t.Fatalf("failed cleanup did not remain blocking: %#v", requests)
	}
	assertHelperJournalState(t, fixture.private, "cleanup-pending", service.lastHandle.ID)
}

func TestHelperScopeCannotOverlapAnotherActiveManagerOwnership(t *testing.T) {
	fixture := newHelperFixture(t)
	fixture.options.ParentScope.ActiveResponsibilities = append(fixture.options.ParentScope.ActiveResponsibilities,
		HelperResponsibility{ManagerID: "docs", Owns: []string{"src/private/"}})
	invocations := 0
	session := fixture.session(t, func(options codexappserver.Options) Invoker {
		invocations++
		return &helperFakeInvoker{options: options}
	})
	if err := session.BindHandle(fixture.parentRecoveryHandle()); err != nil {
		t.Fatal(err)
	}
	if _, err := session.HandleToolCall(context.Background(), helperCall("foreign-owner", `{"task":"touch path","paths":["src/private/"]}`)); err == nil || !strings.Contains(err.Error(), "active Manager") {
		t.Fatalf("other Manager's path was accepted: %v", err)
	}
	if invocations != 0 || len(session.Requests()) != 1 || session.Requests()[0].State != "failed" {
		t.Fatalf("ownership conflict was not durably counted before provider start: %d %#v", invocations, session.Requests())
	}
}

type helperFixture struct {
	repo      string
	storage   string
	private   string
	workspace projectworkspace.Handle
	options   HelperSessionOptions
	reserver  *helperFakeReserver
}

type helperFakeBehavior struct {
	path                 string
	deletePath           string
	renameOld            string
	content              []byte
	modelContent         string
	mode                 string
	lifecycleState       string
	nestedLifecycleState string
}

type helperFakeInvoker struct {
	options   codexappserver.Options
	behavior  helperFakeBehavior
	workspace projectworkspace.Handle
}

func (f *helperFakeInvoker) Fingerprint(agentexec.Config) (string, error) { return "fake", nil }

func (f *helperFakeInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	if options.Workspace == nil {
		return agentexec.RunResult{}, errors.New("helper did not receive owned workspace")
	}
	f.workspace = *options.Workspace
	inv, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	rootRequest := agentexec.RoleStartRequest{RequestID: inv.RunID, Role: "executor", State: "requested", Model: config.Model}
	if f.options.BeforeStart == nil {
		return agentexec.RunResult{}, errors.New("helper child has no Host permit attachment hook")
	}
	if err := f.options.BeforeStart(ctx, rootRequest); err != nil {
		return agentexec.RunResult{}, err
	}
	if f.behavior.path != "" {
		path := filepath.Join(options.Workspace.CWD, filepath.FromSlash(f.behavior.path))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return agentexec.RunResult{}, err
		}
		if err := os.WriteFile(path, f.behavior.content, 0644); err != nil {
			return agentexec.RunResult{}, err
		}
	}
	if f.behavior.renameOld != "" {
		if err := os.Remove(filepath.Join(options.Workspace.CWD, filepath.FromSlash(f.behavior.renameOld))); err != nil {
			return agentexec.RunResult{}, err
		}
	}
	if f.behavior.deletePath != "" {
		if err := os.Remove(filepath.Join(options.Workspace.CWD, filepath.FromSlash(f.behavior.deletePath))); err != nil {
			return agentexec.RunResult{}, err
		}
	}
	state := f.behavior.lifecycleState
	if state == "" {
		state = "completed"
	}
	modelContent := f.behavior.modelContent
	if modelContent == "" {
		modelContent = string(f.behavior.content)
	}
	var candidates []agentexec.CandidateFile
	if f.behavior.path != "" {
		candidates = []agentexec.CandidateFile{{Path: f.behavior.path, Mode: f.behavior.mode, Content: modelContent}}
	}
	result := agentexec.RunResult{Response: agentexec.Response{Outcome: agentexec.OutcomeProposed, CandidateFiles: candidates},
		Receipt: agentexec.Receipt{RunID: inv.RunID, InputDigest: inv.InputDigest, Outcome: agentexec.OutcomeProposed,
			Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "child-session", TurnID: "child-turn", State: state,
				Accounting: "partial", StartRequests: []agentexec.RoleStartRequest{{RequestID: inv.RunID, Role: "executor", State: "started"}}}}}
	if f.behavior.nestedLifecycleState != "" {
		result.Receipt.Lifecycle.StartRequests = append(result.Receipt.Lifecycle.StartRequests,
			agentexec.RoleStartRequest{RequestID: inv.RunID + "-nested", ParentSessionID: "child-session", Role: "helper", State: f.behavior.nestedLifecycleState})
	}
	return result, nil
}

type helperFakeReserver struct {
	reserveCount int
	attachCount  int
	updateCount  int
	requests     []HelperStartAttempt
	updateErr    error
	updateStates []string
	onAttach     func(agentexec.RoleStartRequest)
}

func (r *helperFakeReserver) reserve(_ context.Context, attempt HelperStartAttempt) (HelperReservation, error) {
	r.reserveCount++
	r.requests = append(r.requests, attempt)
	return &helperFakeReservation{owner: r}, nil
}

type helperFakeReservation struct{ owner *helperFakeReserver }

func (r *helperFakeReservation) AttachProtocolStart(_ context.Context, request agentexec.RoleStartRequest) error {
	r.owner.attachCount++
	if r.owner.onAttach != nil {
		r.owner.onAttach(request)
	}
	return nil
}

func (r *helperFakeReservation) Update(_ context.Context, request agentexec.RoleStartRequest) error {
	r.owner.updateCount++
	r.owner.updateStates = append(r.owner.updateStates, request.State)
	if r.owner.updateErr != nil {
		err := r.owner.updateErr
		r.owner.updateErr = nil
		return err
	}
	return nil
}

type helperTrackingWorkspaceService struct {
	candidateWorkspaceService
	closeErr   error
	closeCalls int
	lastHandle projectworkspace.Handle
}

func (s *helperTrackingWorkspaceService) Close(ctx context.Context, handle projectworkspace.Handle) error {
	s.closeCalls++
	s.lastHandle = handle
	if s.closeErr != nil {
		return s.closeErr
	}
	return s.candidateWorkspaceService.Close(ctx, handle)
}

func newHelperFixture(t *testing.T) *helperFixture {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "main.go"), []byte("package src\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "init", "-q")
	gitTest(t, repo, "config", "user.email", "helper@example.invalid")
	gitTest(t, repo, "config", "user.name", "Helper Test")
	gitTest(t, repo, "add", "src/main.go")
	gitTest(t, repo, "commit", "-qm", "base")
	head := strings.TrimSpace(string(gitTest(t, repo, "rev-parse", "HEAD")))
	identity, err := source.IdentifyGit(repo)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := projectworkspace.InspectRepository(context.Background(), repo, head)
	if err != nil {
		t.Fatal(err)
	}
	workspace := projectworkspace.Handle{ID: "parent-workspace", CWD: repo, RepositoryRoot: repo, RepositoryIdentity: identity.Digest,
		BaseSHA: head, OverlayDigest: binding.OverlayDigest, TaskID: "manager-work", BaseDigest: binding.InventoryDigest}
	storage := filepath.Join(root, "workspace-storage")
	service, err := projectworkspace.NewGitService(storage, projectworkspace.Limits{MaxFiles: 1000, MaxFileBytes: 1 << 20, MaxTotalBytes: 16 << 20})
	if err != nil {
		t.Fatal(err)
	}
	reserver := &helperFakeReserver{}
	options := HelperSessionOptions{
		ParentWorkspace: workspace,
		ParentConfig: agentexec.Config{Transport: TransportCodexAppServer, ProviderVersion: codexappserver.SupportedProviderVersion,
			Command: "codex", Model: "gpt-6-luna", Timeout: time.Minute, TransportConfig: json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":true,"maxStartRequests":2,"maxDepth":1},"maxEventBytes":1048576}`)},
		ParentRequest: agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: head, ModelDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000", ModulePin: "module-pin", ProjectionID: "projection", Context: json.RawMessage(`{"kind":"projectrun-task/v1","managerId":"orders"}`), ScopeIDs: []string{"orders"}, PolicyIDs: []string{}, Artifacts: []agentexec.Artifact{}},
		ParentScope: HelperParentScope{ManagerID: "orders", AllowedWritePaths: []string{"src/"}, ExcludedWritePaths: []string{"src/generated/"},
			ActiveResponsibilities: []HelperResponsibility{{ManagerID: "orders", Owns: []string{"src/"}}, {ManagerID: "docs", Owns: []string{"docs/"}}}},
		Workspaces: service, Limits: Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20},
		PrivateLogDirectory: filepath.Join(root, "private"), ChildOptions: codexappserver.Options{DynamicTools: []codexappserver.DynamicTool{HelperDynamicTool(), {Type: "function", Name: "native_read", InputSchema: json.RawMessage(`{"type":"object"}`)}}, HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
			return codexappserver.ToolResult{}, nil
		}, MaxToolCalls: 4, ToolTimeout: time.Minute},
		Reserve: reserver.reserve,
	}
	return &helperFixture{repo: repo, storage: storage, private: options.PrivateLogDirectory, workspace: workspace, options: options, reserver: reserver}
}

func (f *helperFixture) session(t *testing.T, factory HelperInvokerFactory) *HelperSession {
	t.Helper()
	options := f.options
	options.NewInvoker = factory
	session, err := NewHelperSession(options)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func (f *helperFixture) parentRecoveryHandle() codexappserver.RecoveryHandle {
	return codexappserver.RecoveryHandle{Protocol: "fixture", Invocation: agentexec.Invocation{RunID: "parent-run"}, Workspace: f.workspace,
		ThreadID: "parent-thread", SessionID: "parent-session", TurnID: "parent-turn", TurnDispatched: true}
}

func helperCall(id, arguments string) codexappserver.ToolCall {
	return codexappserver.ToolCall{ThreadID: "parent-thread", TurnID: "parent-turn", CallID: id, Tool: HelperToolName,
		Arguments: json.RawMessage(arguments)}
}

func assertHelperJournalState(t *testing.T, privateDirectory, state, handleID string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(privateDirectory, "codex-app-server", "*", "*", "helpers.jsonl"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("locate helper journal: paths=%v err=%v", paths, err)
	}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var record helperJournalRecord
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				t.Fatalf("decode helper journal record: %v", err)
			}
			if record.State == state && record.Handle != nil && record.Handle.ID == handleID {
				return
			}
		}
	}
	t.Fatalf("helper journal has no %q record retaining handle %q", state, handleID)
}

func gitTest(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return output
}
