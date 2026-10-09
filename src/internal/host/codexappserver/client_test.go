package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
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
