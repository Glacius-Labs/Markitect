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
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

type readOnlyRepositoryInvoker struct {
	call      func(agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error)
	workspace projectworkspace.Handle
	called    bool
}

func (i *readOnlyRepositoryInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.called = true
	if options.Workspace != nil {
		i.workspace = *options.Workspace
	}
	if i.call != nil {
		return i.call(config, request, options)
	}
	return agentexec.RunResult{}, nil
}

func (*readOnlyRepositoryInvoker) Fingerprint(agentexec.Config) (string, error) {
	return "fixture", nil
}

func TestRunReadOnlyRepositoryUsesSelectedSourceAndPreservesCurrentWIP(t *testing.T) {
	sourceRoot, sourceSHA, _, targetSHA := readOnlyRepositoryRoots(t)
	if sourceSHA == targetSHA {
		t.Fatal("source and target fixture commits unexpectedly match")
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "untracked-wip.txt"), []byte("current source WIP\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	identity, err := source.IdentifyGit(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	service := readOnlyRepositoryService(t)
	privateDir := t.TempDir()
	invoker := &readOnlyRepositoryInvoker{call: func(config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
		if config.Transport != TransportCodexAppServer || request.SourceRevision != sourceSHA {
			t.Fatalf("request did not preserve the selected native source revision: config=%+v request=%+v", config, request)
		}
		if options.Workspace == nil || sameWorkspacePath(options.Workspace.CWD, sourceRoot) {
			t.Fatal("native invoker did not receive a private owned source workspace")
		}
		if options.Workspace.RepositoryIdentity != identity.Digest || options.Workspace.BaseSHA != sourceSHA || options.Workspace.BaseSHA == targetSHA {
			t.Fatalf("workspace was not bound to the selected source repository: %+v", options.Workspace)
		}
		if got, err := os.ReadFile(filepath.Join(options.Workspace.CWD, "untracked-wip.txt")); err != nil || string(got) != "current source WIP\n" {
			t.Fatalf("complete source working inventory was not preserved: got=%q err=%v", got, err)
		}
		if options.PrivateLogDirectory != privateDir {
			t.Fatalf("native journal root = %q, want %q", options.PrivateLogDirectory, privateDir)
		}
		return agentexec.RunResult{Receipt: terminalWorkspaceReceipt("readonly-clean")}, nil
	}}
	request := agentexec.Request{Role: agentexec.RoleVerifier, SourceRevision: sourceSHA, ModelDigest: "target-model-digest"}
	result, err := RunReadOnlyRepository(context.Background(), service, invoker, sourceRoot, privateDir,
		agentexec.Config{Transport: TransportCodexAppServer}, request,
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20})
	if err != nil {
		t.Fatalf("read-only source invocation: %v", err)
	}
	if !invoker.called || result.Receipt.RunID != "readonly-clean" || result.Delta == nil || len(result.Delta.Changes) != 0 {
		t.Fatalf("read-only source result did not retain its receipt and empty host harvest: result=%+v called=%v", result, invoker.called)
	}
	if _, err := os.Stat(invoker.workspace.CWD); !os.IsNotExist(err) {
		t.Fatalf("terminal clean read-only workspace was not closed: %v", err)
	}
	journal := readReadOnlyWorkspaceJournal(t, privateDir, invoker.workspace.ID)
	if journal.State != "closed" || journal.Receipt.RunID != "readonly-clean" || journal.Request.BaseSHA != sourceSHA || len(journal.Request.AllowedPaths) != 0 || len(journal.Request.ExcludedPaths) != 0 {
		t.Fatalf("private journal does not bind a closed, full-source read-only invocation: %+v", journal)
	}
}

func TestRunReadOnlyRepositoryPreservesUnknownLifecycleAndOriginalError(t *testing.T) {
	sourceRoot, sourceSHA, _, _ := readOnlyRepositoryRoots(t)
	service := readOnlyRepositoryService(t)
	privateDir := t.TempDir()
	providerErr := errors.New("original native call error")
	invoker := &readOnlyRepositoryInvoker{call: func(_ agentexec.Config, _ agentexec.Request, _ agentexec.RunOptions) (agentexec.RunResult, error) {
		return agentexec.RunResult{Receipt: agentexec.Receipt{RunID: "readonly-unknown", Lifecycle: &agentexec.Lifecycle{
			Provider: TransportCodexAppServer, SessionID: "session", TurnID: "turn", State: "unknown",
			StartRequests: []agentexec.RoleStartRequest{{RequestID: "root", State: "unknown"}},
		}}}, providerErr
	}}
	result, err := RunReadOnlyRepository(context.Background(), service, invoker, sourceRoot, privateDir,
		agentexec.Config{Transport: TransportCodexAppServer}, agentexec.Request{SourceRevision: sourceSHA},
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20})
	if !errors.Is(err, providerErr) || err == nil || !strings.Contains(err.Error(), "termination is unknown") {
		t.Fatalf("unknown lifecycle must preserve the original failure and report uncertainty: %v", err)
	}
	if result.Receipt.RunID != "readonly-unknown" || invoker.workspace.CWD == "" {
		t.Fatalf("unknown lifecycle lost receipt or workspace ownership: %+v handle=%+v", result, invoker.workspace)
	}
	if _, err := os.Stat(invoker.workspace.CWD); err != nil {
		t.Fatalf("uncertain source workspace was removed: %v", err)
	}
	journal := readReadOnlyWorkspaceJournal(t, privateDir, invoker.workspace.ID)
	if journal.State != "preserved" || journal.Receipt.RunID != "readonly-unknown" {
		t.Fatalf("uncertain private journal lost provider evidence: %+v", journal)
	}
	if err := service.Close(context.Background(), invoker.workspace); err != nil {
		t.Fatalf("close fixture workspace after assertions: %v", err)
	}
}

func TestRunReadOnlyRepositoryPreservesUnexpectedWritesAndInvocationError(t *testing.T) {
	sourceRoot, sourceSHA, _, _ := readOnlyRepositoryRoots(t)
	service := readOnlyRepositoryService(t)
	privateDir := t.TempDir()
	providerErr := errors.New("provider reported a partial failure")
	invoker := &readOnlyRepositoryInvoker{call: func(_ agentexec.Config, _ agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
		if err := os.WriteFile(filepath.Join(options.Workspace.CWD, "unexpected.txt"), []byte("must be preserved\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return agentexec.RunResult{Receipt: terminalWorkspaceReceipt("readonly-write")}, providerErr
	}}
	result, err := RunReadOnlyRepository(context.Background(), service, invoker, sourceRoot, privateDir,
		agentexec.Config{Transport: TransportCodexAppServer}, agentexec.Request{SourceRevision: sourceSHA},
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20})
	if !errors.Is(err, providerErr) || err == nil || !strings.Contains(err.Error(), "read-only repository invocation changed source files") {
		t.Fatalf("unexpected writes must preserve the provider failure and report the violation: %v", err)
	}
	if result.Receipt.RunID != "readonly-write" || result.Delta != nil {
		t.Fatalf("unexpected writes must not be accepted as a read-only delta: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(invoker.workspace.CWD, "unexpected.txt")); err != nil {
		t.Fatalf("workspace with unexpected writes was removed or changed: %v", err)
	}
	journal := readReadOnlyWorkspaceJournal(t, privateDir, invoker.workspace.ID)
	if journal.State != "preserved" || journal.Receipt.RunID != "readonly-write" {
		t.Fatalf("write violation did not preserve durable evidence: %+v", journal)
	}
	if err := service.Close(context.Background(), invoker.workspace); err != nil {
		t.Fatalf("close fixture workspace after assertions: %v", err)
	}
}

func TestRunReadOnlyRepositoryRequiresMatchingNativeSourceRevision(t *testing.T) {
	sourceRoot, sourceSHA, _, _ := readOnlyRepositoryRoots(t)
	invoker := &readOnlyRepositoryInvoker{}
	_, err := RunReadOnlyRepository(context.Background(), readOnlyRepositoryService(t), invoker, sourceRoot, t.TempDir(),
		agentexec.Config{Transport: TransportCodexAppServer}, agentexec.Request{SourceRevision: strings.Repeat("0", 40)},
		Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20})
	if err == nil || !strings.Contains(err.Error(), "does not match selected repository HEAD") || invoker.called {
		t.Fatalf("mismatched source revision was invoked: source=%s err=%v called=%v", sourceSHA, err, invoker.called)
	}
}

func readOnlyRepositoryRoots(t *testing.T) (string, string, string, string) {
	t.Helper()
	sourceRoot := t.TempDir()
	gitE2E(t, sourceRoot, "init", "--quiet", "-b", "codex/source-repository")
	gitE2E(t, sourceRoot, "config", "user.email", "readonly@example.test")
	gitE2E(t, sourceRoot, "config", "user.name", "Read Only Fixture")
	writeE2E(t, sourceRoot, "source.txt", "source repository\n")
	gitE2E(t, sourceRoot, "add", ".")
	gitE2E(t, sourceRoot, "commit", "--quiet", "-m", "source repository")
	sourceSHA := identityHead(t, sourceRoot)
	targetRoot := t.TempDir()
	gitE2E(t, targetRoot, "init", "--quiet", "-b", "codex/target-project")
	gitE2E(t, targetRoot, "config", "user.email", "target@example.test")
	gitE2E(t, targetRoot, "config", "user.name", "Target Fixture")
	writeE2E(t, targetRoot, "target.txt", "target model repository\n")
	gitE2E(t, targetRoot, "add", ".")
	gitE2E(t, targetRoot, "commit", "--quiet", "-m", "target repository")
	return sourceRoot, sourceSHA, targetRoot, identityHead(t, targetRoot)
}

func readOnlyRepositoryService(t *testing.T) *projectworkspace.GitService {
	t.Helper()
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func readReadOnlyWorkspaceJournal(t *testing.T, privateDir, workspaceID string) workspaceJournal {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(privateDir, "workspaces", workspaceID+".json"))
	if err != nil {
		t.Fatalf("read private read-only journal: %v", err)
	}
	var journal workspaceJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		t.Fatalf("decode private read-only journal: %v", err)
	}
	return journal
}
