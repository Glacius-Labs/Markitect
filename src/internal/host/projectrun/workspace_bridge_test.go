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
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

type workspaceBridgeInvoker struct {
	result    agentexec.RunResult
	err       error
	call      func(agentexec.RunOptions) (agentexec.RunResult, error)
	called    bool
	workspace projectworkspace.Handle
}

func (i *workspaceBridgeInvoker) Run(_ context.Context, _ agentexec.Config, _ agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.called = true
	if options.Workspace != nil {
		i.workspace = *options.Workspace
	}
	if i.call != nil {
		return i.call(options)
	}
	return i.result, i.err
}

func (*workspaceBridgeInvoker) Fingerprint(agentexec.Config) (string, error) { return "fixture", nil }

type tamperingWorkspaceService struct {
	*candidateWorkspaceServiceAdapter
	mutate func(*projectworkspace.Delta)
}

// candidateWorkspaceServiceAdapter permits a small Harvest-only wrapper while
// retaining the real private Git service for preparation, identity and close.
type candidateWorkspaceServiceAdapter struct{ service *projectworkspace.GitService }

func (s *candidateWorkspaceServiceAdapter) Prepare(ctx context.Context, request projectworkspace.Request) (projectworkspace.Handle, error) {
	return s.service.Prepare(ctx, request)
}
func (s *candidateWorkspaceServiceAdapter) PrepareCandidate(ctx context.Context, request projectworkspace.Request, overlay []projectworkspace.Change, digest string) (projectworkspace.Handle, error) {
	return s.service.PrepareCandidate(ctx, request, overlay, digest)
}
func (s *candidateWorkspaceServiceAdapter) Harvest(ctx context.Context, handle projectworkspace.Handle) (projectworkspace.Delta, error) {
	delta, err := s.service.Harvest(ctx, handle)
	return delta, err
}
func (s *candidateWorkspaceServiceAdapter) Close(ctx context.Context, handle projectworkspace.Handle) error {
	return s.service.Close(ctx, handle)
}
func (s *tamperingWorkspaceService) Harvest(ctx context.Context, handle projectworkspace.Handle) (projectworkspace.Delta, error) {
	delta, err := s.service.Harvest(ctx, handle)
	if err == nil {
		s.mutate(&delta)
	}
	return delta, err
}

func TestInvokeProjectAgentUsesPrivateCandidateWithSourceWIPAndHarvestsByteDelta(t *testing.T) {
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	writeE2E(t, root, "src/orders/obsolete.txt", "delete this tracked file\n")
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "--quiet", "-m", "add workspace deletion fixture")
	revision := identityHead(t, root)
	base, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, "source-wip.txt", "source WIP that must remain in the private workspace\n")

	// The manager receives a parent candidate overlay on top of the complete
	// source-WIP inventory. The unrelated README WIP must survive preparation.
	candidate := *base
	candidateSnapshot := *base.Snapshot
	candidateSnapshot.Files = cloneWorkspaceSnapshotFiles(base.Snapshot.Files)
	candidateSnapshot.Modes = cloneWorkspaceSnapshotModes(base.Snapshot.Modes)
	candidateBytes := []byte("parent candidate change to the owned implementation\n")
	candidateSnapshot.Files["src/orders/implementation.txt"] = candidateBytes
	candidate.Snapshot = &candidateSnapshot

	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	host := Host{Workspaces: service, Load: projectwork.Load}
	providerErr := errors.New("original native invocation failure")
	invoker := &workspaceBridgeInvoker{call: func(options agentexec.RunOptions) (agentexec.RunResult, error) {
		if options.Workspace == nil {
			t.Fatal("native invoker did not receive the owned workspace handle")
		}
		handle := *options.Workspace
		if sameWorkspacePath(handle.CWD, root) {
			t.Fatal("native invoker received the source checkout as CWD")
		}
		if handle.RepositoryIdentity != identity.Digest || handle.BaseSHA != revision {
			t.Fatalf("workspace handle is not bound to selected source: %+v", handle)
		}
		read := func(path string) []byte {
			t.Helper()
			data, readErr := os.ReadFile(filepath.Join(handle.CWD, filepath.FromSlash(path)))
			if readErr != nil {
				t.Fatalf("read private workspace %s: %v", path, readErr)
			}
			return data
		}
		if got := string(read("source-wip.txt")); got != "source WIP that must remain in the private workspace\n" {
			t.Fatalf("source WIP was not preserved in private workspace: %q", got)
		}
		if got := read("src/orders/implementation.txt"); string(got) != string(candidateBytes) {
			t.Fatalf("parent candidate overlay was not applied over source WIP: %q", got)
		}
		if err := os.Remove(filepath.Join(handle.CWD, "src", "orders", "obsolete.txt")); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(handle.CWD, "src", "orders", "implementation.txt"), filepath.Join(handle.CWD, "src", "orders", "renamed.txt")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(handle.CWD, "src", "orders", "binary.bin"), []byte{0, 1, 0xff, 0, 0x80}, 0o644); err != nil {
			t.Fatal(err)
		}
		return agentexec.RunResult{Receipt: terminalWorkspaceReceipt("workspace-run")}, providerErr
	}}
	limits := Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}
	result, runErr := invokeProjectAgent(context.Background(), host, invoker, root, &candidate,
		workspaceBridgeAgent(t, root), "workspace-owner-run", "orders-work", []string{"src/orders"}, nil, limits,
		agentexec.Config{Transport: TransportCodexAppServer}, agentexec.Request{Role: agentexec.RoleExecutor})
	if !errors.Is(runErr, providerErr) {
		t.Fatalf("original invocation error was not retained: %v", runErr)
	}
	if result.Receipt.RunID != "workspace-run" || result.Delta == nil {
		t.Fatalf("receipt or Host-harvested delta was lost: result=%+v err=%v", result, runErr)
	}
	kinds := map[string]projectworkspace.Change{}
	for _, change := range result.Delta.Changes {
		kinds[string(change.Kind)+":"+change.Path] = change
	}
	if _, ok := kinds["delete:src/orders/obsolete.txt"]; !ok {
		t.Fatalf("observed deletion missing from delta: %+v", result.Delta.Changes)
	}
	rename, ok := kinds["rename:src/orders/renamed.txt"]
	if !ok || rename.OldPath != "src/orders/implementation.txt" || string(rename.Content) != string(candidateBytes) {
		t.Fatalf("observed rename missing or incorrect: %+v", result.Delta.Changes)
	}
	binary, ok := kinds["add:src/orders/binary.bin"]
	if !ok || string(binary.Content) != string([]byte{0, 1, 0xff, 0, 0x80}) {
		t.Fatalf("binary bytes missing from observed delta: %+v", result.Delta.Changes)
	}
	if _, err := os.Stat(invoker.workspace.CWD); !os.IsNotExist(err) {
		t.Fatalf("terminal harvested workspace was not closed: %v", err)
	}
	journal := readWorkspaceJournal(t, root, invoker.workspace.ID)
	if journal.State != "closed" || journal.Receipt.RunID != "workspace-run" || journal.Delta == nil {
		t.Fatalf("closed journal did not retain receipt and delta: %+v", journal)
	}
}

func TestInvokeProjectAgentPreservesUnknownRootOrChildLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name      string
		lifecycle *agentexec.Lifecycle
	}{
		{name: "missing lifecycle"},
		{name: "unknown child", lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "session", TurnID: "turn", State: "completed", StartRequests: []agentexec.RoleStartRequest{{RequestID: "root", SessionID: "session", Role: "root", State: "completed"}, {RequestID: "child", ParentSessionID: "session", SessionID: "child-session", Role: "worker", State: "unknown"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, project, service := workspaceBridgeBase(t)
			invoker := &workspaceBridgeInvoker{result: agentexec.RunResult{Receipt: agentexec.Receipt{RunID: "unknown-run", Lifecycle: tc.lifecycle}}}
			result, err := invokeProjectAgent(context.Background(), Host{Workspaces: service, Load: projectwork.Load}, invoker, root, project,
				workspaceBridgeAgent(t, root), "workspace-owner-run", "orders-work", []string{"src/orders"}, nil,
				Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, agentexec.Config{}, agentexec.Request{Role: agentexec.RoleExecutor})
			if err == nil || !strings.Contains(err.Error(), "termination is unknown") {
				t.Fatalf("unknown lifecycle was not reported: %v", err)
			}
			if result.Receipt.RunID != "unknown-run" || invoker.workspace.CWD == "" {
				t.Fatalf("durable receipt or workspace handle was lost: result=%+v handle=%+v", result, invoker.workspace)
			}
			if _, err := os.Stat(invoker.workspace.CWD); err != nil {
				t.Fatalf("uncertain workspace was cleaned up: %v", err)
			}
			journal := readWorkspaceJournal(t, root, invoker.workspace.ID)
			if journal.State != "preserved" || journal.Receipt.RunID != "unknown-run" {
				t.Fatalf("uncertain journal did not preserve receipt/workspace identity: %+v", journal)
			}
			_ = service.Close(context.Background(), invoker.workspace)
		})
	}
}

func TestInvokeProjectAgentRejectsReadOnlyNativeBindingBeforeInvocation(t *testing.T) {
	root, project, service := workspaceBridgeBase(t)
	invoker := &workspaceBridgeInvoker{}
	_, err := invokeProjectAgent(context.Background(), Host{Workspaces: service, Load: projectwork.Load}, invoker, root, project,
		Agent{Transport: TransportCodexAppServer, WorkspaceMode: ""}, "workspace-owner-run", "orders-review", []string{"src/orders"}, nil,
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, agentexec.Config{}, agentexec.Request{Role: agentexec.RoleExecutor})
	if err == nil || invoker.called {
		t.Fatalf("native read-only binding was allowed into a writable Git workspace: called=%t err=%v", invoker.called, err)
	}
}

func TestInvokeProjectAgentRejectsForgedHarvestDigestOrBinding(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*projectworkspace.Delta)
	}{
		{name: "digest", mutate: func(delta *projectworkspace.Delta) { delta.Digest = "sha256:" + strings.Repeat("0", 64) }},
		{name: "binding", mutate: func(delta *projectworkspace.Delta) { delta.BaseSHA = strings.Repeat("0", 40) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, project, gitService := workspaceBridgeBase(t)
			service := &tamperingWorkspaceService{candidateWorkspaceServiceAdapter: &candidateWorkspaceServiceAdapter{service: gitService}, mutate: tc.mutate}
			invoker := &workspaceBridgeInvoker{result: agentexec.RunResult{Receipt: terminalWorkspaceReceipt("forged-run")}}
			result, err := invokeProjectAgent(context.Background(), Host{Workspaces: service, Load: projectwork.Load}, invoker, root, project,
				workspaceBridgeAgent(t, root), "workspace-owner-run", "orders-work", []string{"src/orders"}, nil,
				Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}, agentexec.Config{}, agentexec.Request{Role: agentexec.RoleExecutor})
			if err == nil || result.Delta != nil || result.Receipt.RunID != "forged-run" {
				t.Fatalf("forged harvested delta was accepted or receipt lost: result=%+v err=%v", result, err)
			}
			if invoker.workspace.CWD == "" {
				t.Fatal("invocation did not receive workspace handle")
			}
			journal := readWorkspaceJournal(t, root, invoker.workspace.ID)
			if journal.State != "preserved" || journal.Receipt.RunID != "forged-run" || journal.Delta != nil {
				t.Fatalf("invalid delta journal state is unsafe: %+v", journal)
			}
			_ = gitService.Close(context.Background(), invoker.workspace)
		})
	}
}

func workspaceBridgeBase(t *testing.T) (string, *Project, *projectworkspace.GitService) {
	t.Helper()
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	revision := identityHead(t, root)
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return root, project, service
}

func configureWorkspaceBridgeInstructions(t *testing.T, root string) {
	t.Helper()
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestText := strings.Replace(string(manifest), "name: Process fixture\n", "name: Process fixture\ndocumentPath: docs/agent-instructions.md\n", 1)
	if manifestText == string(manifest) {
		t.Fatal("could not set a fixed native instruction document path")
	}
	if err := os.WriteFile(manifestPath, []byte(manifestText), 0o644); err != nil {
		t.Fatal(err)
	}
	writeE2E(t, root, "docs/agent-instructions.md", "Follow the selected fixture contract.\n")
	gitE2E(t, root, "add", ".")
	gitE2E(t, root, "commit", "--quiet", "-m", "add pinned native instruction fixture")
}

func workspaceBridgeAgent(t *testing.T, root string) Agent {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(root, "docs", "agent-instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return Agent{Transport: TransportCodexAppServer, WorkspaceMode: "git", InstructionPaths: []string{"docs/agent-instructions.md"},
		RuntimeFiles: []agentexec.RuntimeFile{{Path: path, Mode: "0644", Digest: "sha256:" + digestBytes(content)}}}
}

func terminalWorkspaceReceipt(id string) agentexec.Receipt {
	return agentexec.Receipt{RunID: id, Lifecycle: &agentexec.Lifecycle{
		Provider: TransportCodexAppServer, SessionID: "session-" + id, TurnID: "turn-" + id, State: "completed",
		Accounting: "complete", StartRequests: []agentexec.RoleStartRequest{{RequestID: "root-" + id, Role: "root", State: "completed"}},
	}}
}

func cloneWorkspaceSnapshotFiles(files map[string][]byte) map[string][]byte {
	copy := make(map[string][]byte, len(files))
	for path, data := range files {
		copy[path] = append([]byte(nil), data...)
	}
	return copy
}

func cloneWorkspaceSnapshotModes(modes map[string]string) map[string]string {
	copy := make(map[string]string, len(modes))
	for path, mode := range modes {
		copy[path] = mode
	}
	return copy
}

func readWorkspaceJournal(t *testing.T, root, id string) workspaceJournal {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".markitect", "runs", "private", "workspaces", id+".json"))
	if err != nil {
		t.Fatalf("read workspace journal: %v", err)
	}
	var journal workspaceJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		t.Fatalf("decode workspace journal: %v", err)
	}
	return journal
}

func sameWorkspacePath(a, b string) bool { return filepath.Clean(a) == filepath.Clean(b) }
