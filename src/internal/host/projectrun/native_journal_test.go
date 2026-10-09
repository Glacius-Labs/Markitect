package projectrun

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

func TestNativeJournalBeforeStartPersistsReservationBeforeCallback(t *testing.T) {
	journal := newTestNativeJournal(t)
	called := false
	options := journal.wrapOptions(codexappserver.Options{BeforeStart: func(_ context.Context, request agentexec.RoleStartRequest) error {
		called = true
		if request.RequestID != "protocol-root" {
			t.Fatalf("protocol request ID = %q", request.RequestID)
		}
		lines := readJournalLines(t, filepath.Join(journal.directory, "starts.jsonl"))
		if len(lines) != 1 || lines[0].State != "requested" {
			t.Fatalf("reservation was not durable before callback: %#v", lines)
		}
		if lines[0].ProtocolRequestID != request.RequestID || lines[0].RequestID == request.RequestID || !strings.HasPrefix(lines[0].RequestID, "native-") {
			t.Fatalf("reservation identity not independently bound: %#v", lines[0])
		}
		return nil
	}})
	if err := options.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: "protocol-root", Role: "executor", State: "requested"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("original callback did not run")
	}
}

func TestNativeJournalFailedStartRemainsAsAppendOnlyFailure(t *testing.T) {
	journal := newTestNativeJournal(t)
	wantErr := errors.New("reservation denied")
	called := 0
	options := journal.wrapOptions(codexappserver.Options{BeforeStart: func(context.Context, agentexec.RoleStartRequest) error {
		called++
		return wantErr
	}})
	err := options.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: "protocol-root", Role: "executor"})
	if !errors.Is(err, wantErr) || called != 1 {
		t.Fatalf("callback error/call count = %v/%d", err, called)
	}
	lines := readJournalLines(t, filepath.Join(journal.directory, "starts.jsonl"))
	if len(lines) != 2 || lines[0].State != "requested" || lines[1].State != "failed" || lines[0].RequestID != lines[1].RequestID {
		t.Fatalf("failed attempt was erased or double-counted: %#v", lines)
	}
}

func TestNativeJournalHandleIsBoundAndPersistedBeforeCallback(t *testing.T) {
	journal := newTestNativeJournal(t)
	if _, err := journal.reserveStart(agentexec.RoleStartRequest{RequestID: "protocol-root", Role: "executor"}); err != nil {
		t.Fatal(err)
	}
	called := false
	options := journal.wrapOptions(codexappserver.Options{OnHandle: func(_ context.Context, handle codexappserver.RecoveryHandle) error {
		called = true
		recovered, err := readTrustedRecoveryHandles(filepath.Dir(filepath.Dir(filepath.Dir(journal.directory))), journal.workspaceID)
		if err != nil || len(recovered) != 1 || recovered[0].ThreadID != handle.ThreadID {
			t.Fatalf("recovery handle was not persisted before callback: %#v, %v", recovered, err)
		}
		return nil
	}})
	handle := testNativeRecoveryHandle(journal.workspaceID, "protocol-root")
	handle.Workspace.CWD = filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(journal.directory)))), "workspace")
	handle.ThreadID = "thread-1"
	if err := options.OnHandle(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("original OnHandle callback did not run")
	}
	called = false
	wrongWorkspace := testNativeRecoveryHandle("other-workspace", "protocol-root")
	if err := options.OnHandle(context.Background(), wrongWorkspace); err == nil || called {
		t.Fatal("mismatched workspace handle reached original callback")
	}
}

func TestNativeJournalPersistsExactPrivateEventBeforeCallback(t *testing.T) {
	journal := newTestNativeJournal(t)
	called := false
	wire := []byte("{ \"method\":\"item/agentMessage/delta\", \"params\":{\"text\":\"private project bytes\"} }\r\n")
	options := journal.wrapOptions(codexappserver.Options{OnEvent: func(_ context.Context, event codexappserver.Event) error {
		called = true
		data, err := os.ReadFile(filepath.Join(journal.directory, "events.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		var record nativeEventRecord
		if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &record); err != nil {
			t.Fatal(err)
		}
		got, err := base64.StdEncoding.DecodeString(record.WireBase64)
		if err != nil || string(got) != string(event.Wire) {
			t.Fatalf("persisted wire bytes differ: %q, %v", got, err)
		}
		if strings.Contains(string(data), "private project bytes") {
			t.Fatal("event payload was written in plaintext")
		}
		return nil
	}})
	if err := options.OnEvent(context.Background(), codexappserver.Event{Method: "item/agentMessage/delta", Wire: wire}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("original OnEvent callback did not run")
	}
	info, err := os.Stat(filepath.Join(journal.directory, "events.jsonl"))
	if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0600) {
		t.Fatalf("event journal permissions = %v, err=%v", info, err)
	}
}

func TestNativeJournalWrappingPreservesFingerprintOptions(t *testing.T) {
	calls := 0
	original := codexappserver.Options{
		BeforeStart:  func(context.Context, agentexec.RoleStartRequest) error { calls++; return nil },
		OnHandle:     func(context.Context, codexappserver.RecoveryHandle) error { calls++; return nil },
		OnEvent:      func(context.Context, codexappserver.Event) error { calls++; return nil },
		DynamicTools: []codexappserver.DynamicTool{{Type: "function", Name: "inspect", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
			calls++
			return codexappserver.ToolResult{}, nil
		},
		MaxToolCalls: 3, ToolTimeout: 4 * time.Second,
	}
	journal := newTestNativeJournal(t)
	wrapped := journal.wrapOptions(original)
	invoker := NewTransportInvoker(original)
	config := nativeFingerprintConfig(t)
	before, err := invoker.Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := invoker.appServerAdapterWithOptions(config, wrapped)
	if err != nil {
		t.Fatal(err)
	}
	after, err := adapter.Fingerprint(config)
	if err != nil || after != before {
		t.Fatalf("journal hooks changed fingerprint: %q / %q, %v", before, after, err)
	}
	if len(wrapped.DynamicTools) != 1 || wrapped.MaxToolCalls != original.MaxToolCalls || wrapped.ToolTimeout != original.ToolTimeout || wrapped.HandleToolCall == nil {
		t.Fatal("wrapping did not preserve the original non-callback options")
	}
	if err := wrapped.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: "root"}); err != nil || calls != 1 {
		t.Fatalf("original callback not preserved, err=%v calls=%d", err, calls)
	}
}

func newTestNativeJournal(t *testing.T) *nativeJournal {
	t.Helper()
	root := t.TempDir()
	private := filepath.Join(root, "private")
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	journal, err := newNativeJournal(private, workspace, "workspace-handle")
	if err != nil {
		t.Fatal(err)
	}
	if pathIsWithin(workspace, journal.directory) || pathIsWithin(journal.directory, workspace) {
		t.Fatal("journal and provider workspace must be separate")
	}
	return journal
}

func testNativeRecoveryHandle(workspaceID, runID string) codexappserver.RecoveryHandle {
	return codexappserver.RecoveryHandle{
		Protocol: "codex-app-server/fixture", Fingerprint: "fingerprint",
		Invocation: agentexec.Invocation{RunID: runID},
		Workspace:  projectworkspace.Handle{ID: workspaceID, CWD: filepath.Join(os.TempDir(), "native-journal-fixture")},
		ThreadID:   "thread", SessionID: "session",
	}
}

func readJournalLines(t *testing.T, path string) []nativeStartRecord {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var records []nativeStartRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record nativeStartRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return records
}
