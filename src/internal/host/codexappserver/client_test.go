package codexappserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func TestClientEnvelopeGuards(t *testing.T) {
	for _, wire := range []string{`{"id":7,"result":{}}`, `{"result":{}}`, `{"id":1,"result":{},"error":{}}`, `{"id":1}`, `{broken`, `{"method":"turn/started"}`} {
		t.Run(wire, func(t *testing.T) {
			local, remote := net.Pipe()
			defer remote.Close()
			c, err := NewClient(local, 4096, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			go func() {
				var request map[string]any
				_ = json.NewDecoder(remote).Decode(&request)
				_, _ = remote.Write([]byte(wire + "\n"))
			}()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err = c.call(ctx, "initialize", map[string]any{}, nil); err == nil {
				t.Fatal("malformed/mismatched envelope accepted")
			}
		})
	}
}
func TestClientEventBudgetAndRightsEvidence(t *testing.T) {
	local, remote := net.Pipe()
	defer remote.Close()
	seen := false
	c, err := NewClient(local, 1024, func(e Event) error {
		if e.Method == "rpc/response" && strings.Contains(string(e.Wire), "approvalPolicy") {
			seen = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	go func() {
		var request map[string]any
		_ = json.NewDecoder(remote).Decode(&request)
		_ = json.NewEncoder(remote).Encode(map[string]any{"id": 1, "result": map[string]any{"approvalPolicy": "never", "sandbox": map[string]string{"type": "workspaceWrite"}}})
		_, _ = remote.Write([]byte(strings.Repeat("x", 1024) + "\n"))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err = c.call(ctx, "thread/start", map[string]any{}, nil); err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("actual rights RPC evidence lost")
	}
	_, err = c.next(ctx)
	if !errors.Is(err, ErrEventLimit) {
		t.Fatalf("budget not enforced: %v", err)
	}
}

func TestClientRecordsExactSuccessfullyWrittenTurnStartSchema(t *testing.T) {
	local, remote := net.Pipe()
	defer remote.Close()
	var events []Event
	c, err := NewClient(local, 16<<10, func(e Event) error {
		events = append(events, e)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	params := map[string]any{
		"threadId": "thread-1",
		"input":    []any{map[string]any{"type": "text", "text": "semantic output only"}},
		"outputSchema": map[string]any{
			"type": "object", "additionalProperties": false,
			"required":   []string{"outcome", "candidateFiles", "evidenceRefs"},
			"properties": map[string]any{"evidenceRefs": map[string]any{"type": "array", "items": map[string]any{"enum": []string{"evidence-000000-000000"}}}},
		},
	}
	gotWire := make(chan []byte, 1)
	go func() {
		line, _ := bufio.NewReader(remote).ReadBytes('\n')
		gotWire <- line
	}()
	if err := c.send(context.Background(), map[string]any{"id": 7, "method": "turn/start", "params": params}); err != nil {
		t.Fatal(err)
	}
	wire := <-gotWire
	if len(events) != 1 || events[0].Method != "rpc/request" {
		t.Fatalf("successful outgoing request was not journaled: %#v", events)
	}
	if string(events[0].Wire) != string(wire) || events[0].Wire[len(events[0].Wire)-1] != '\n' {
		t.Fatalf("journal did not retain the exact successfully written frame: event=%q wire=%q", events[0].Wire, wire)
	}
	var sent envelope
	if err := json.Unmarshal(wire, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Method != "turn/start" || !bytes.Equal(events[0].Params, sent.Params) || !bytes.Contains(events[0].Params, []byte(`"outputSchema"`)) {
		t.Fatalf("captured request differs from transmitted turn/start schema: event=%s sent=%s", events[0].Params, sent.Params)
	}
}

func TestClientDoesNotJournalFailedOutgoingRequest(t *testing.T) {
	local, remote := net.Pipe()
	remote.Close()
	var events []Event
	c, err := NewClient(local, 4096, func(e Event) error {
		events = append(events, e)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	err = c.send(context.Background(), map[string]any{"id": 1, "method": "turn/start", "params": map[string]any{"outputSchema": map[string]any{"type": "object"}}})
	if err == nil {
		t.Fatal("write to closed peer unexpectedly succeeded")
	}
	if len(events) != 0 {
		t.Fatalf("failed outgoing write was represented as dispatched: %#v", events)
	}
}

type writeCompletesOnClose struct {
	started chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (c *writeCompletesOnClose) Read([]byte) (int, error) {
	<-c.closed
	return 0, io.EOF
}
func (c *writeCompletesOnClose) Write(p []byte) (int, error) {
	close(c.started)
	<-c.closed
	return len(p), nil
}
func (c *writeCompletesOnClose) Close() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

func TestClientJournalsFullFrameWhenCancellationWinsWriteSelect(t *testing.T) {
	conn := &writeCompletesOnClose{started: make(chan struct{}), closed: make(chan struct{})}
	var events []Event
	c, err := NewClient(conn, 4096, func(e Event) error { events = append(events, e); return nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- c.send(ctx, map[string]any{"id": 9, "method": "turn/start", "params": map[string]any{"outputSchema": map[string]any{"type": "object"}}})
	}()
	<-conn.started
	cancel()
	err = <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request returned %v", err)
	}
	if len(events) != 1 || events[0].Method != "rpc/request" || !bytes.Contains(events[0].Wire, []byte(`"outputSchema"`)) {
		t.Fatalf("successful write lost its dispatched request evidence when cancellation won select: %#v", events)
	}
}

func TestHandleSaveFailureBeforeDispatchIsCertain(t *testing.T) {
	a, cfg, req, opts := fixture(t, "success", Options{OnHandle: func(_ context.Context, h RecoveryHandle) error {
		if h.TurnDispatched {
			return errors.New("journal unavailable")
		}
		return nil
	}})
	r, err := a.Run(context.Background(), cfg, req, opts)
	if err == nil || errors.Is(err, ErrUncertain) {
		t.Fatalf("unsent turn became uncertain: %v %+v", err, r.Receipt)
	}
}

func TestFingerprintBindsToolsAndRejectsMismatch(t *testing.T) {
	a, cfg, _, _ := fixture(t, "success", Options{})
	base, err := a.Fingerprint(cfg)
	if err != nil {
		t.Fatal(err)
	}
	changed := cfg
	changed.Args = []string{"--dangerously-bypass-approvals-and-sandbox"}
	if _, err = a.Fingerprint(changed); err == nil {
		t.Fatal("unsupported args accepted")
	}
	changed = cfg
	changed.Model = "other"
	if _, err = a.Fingerprint(changed); err == nil {
		t.Fatal("model mismatch accepted")
	}
	options := Options{DynamicTools: []DynamicTool{{Type: "function", Name: "host_helper", InputSchema: json.RawMessage(`{"type":"object"}`)}}, MaxToolCalls: 1, ToolTimeout: time.Second, HandleToolCall: func(context.Context, ToolCall) (ToolResult, error) { return ToolResult{}, nil }}
	other, err := NewAdapter(a.config, options)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := other.Fingerprint(cfg)
	if err != nil || pin == base {
		t.Fatal("dynamic tool identity not pinned")
	}
}

func TestToolDuplicateAndTimeoutAreBounded(t *testing.T) {
	a, _, _, _ := fixture(t, "success", Options{DynamicTools: []DynamicTool{{Type: "function", Name: "host_helper", InputSchema: json.RawMessage(`{"type":"object"}`)}}, MaxToolCalls: 1, ToolTimeout: 10 * time.Millisecond, HandleToolCall: func(ctx context.Context, _ ToolCall) (ToolResult, error) {
		<-ctx.Done()
		return ToolResult{}, ctx.Err()
	}})
	s := &session{a: a, h: RecoveryHandle{ThreadID: "t", TurnID: "u"}, toolCalls: map[string]bool{}, life: &agentexec.Lifecycle{}}
	params, _ := json.Marshal(ToolCall{ThreadID: "t", TurnID: "u", CallID: "c", Tool: "host_helper", Arguments: json.RawMessage(`{}`)})
	err := s.toolRequest(context.Background(), envelope{Method: "item/tool/call", Params: params})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("tool timeout not bounded: %v", err)
	}
	if err = s.toolRequest(context.Background(), envelope{Method: "item/tool/call", Params: params}); err == nil {
		t.Fatal("duplicate tool could execute twice")
	}
}

func TestExplicitProfileMustBeConfirmed(t *testing.T) {
	for _, mode := range []string{"success", "profile-unconfirmed", "policy-unconfirmed", "readonly-sandbox", "extra-writable-root", "network-enabled", "temp-disabled"} {
		t.Run(mode, func(t *testing.T) {
			a, cfg, req, opts := fixture(t, mode, Options{})
			a.config.PermissionProfile = ":workspace"
			r, err := a.Run(context.Background(), cfg, req, opts)
			if mode == "success" {
				if err != nil || r.Receipt.Lifecycle.Effective.PermissionProfile != ":workspace" {
					t.Fatalf("explicit profile missing: %v %+v", err, r.Receipt)
				}
			} else if err == nil || r.Receipt.Lifecycle.TurnID != "" {
				t.Fatal("unconfirmed explicit rights started a turn")
			}
		})
	}
}

func TestChildThreadIdentityDoesNotInventSession(t *testing.T) {
	a, _, _, _ := fixture(t, "success", Options{})
	s := &session{a: a, h: RecoveryHandle{ThreadID: "root", SessionID: "root-session"}, life: &agentexec.Lifecycle{}, children: map[string]int{}, spawns: map[string]int{}, sessions: map[string]string{"root": "root-session"}, receivers: map[string]int{}}
	v := item{ID: "spawn", Type: "collabAgentToolCall", Tool: "spawnAgent", ReceiverThreadIDs: []string{"child"}, Status: "completed"}
	if err := s.takeItem("root", v, true); err != nil {
		t.Fatal(err)
	}
	if s.life.StartRequests[0].SessionID != "" || s.life.StartRequests[0].ParentSessionID != "root-session" {
		t.Fatal("thread IDs falsely recorded as sessions")
	}
	p, _ := json.Marshal(map[string]any{"thread": thread{ID: "child", SessionID: "child-session", ParentThreadID: "root"}})
	if err := s.observe(Event{Method: "thread/started", Params: p}); err != nil {
		t.Fatal(err)
	}
	if s.life.StartRequests[0].SessionID != "child-session" {
		t.Fatal("observed child session was not bound")
	}
}
