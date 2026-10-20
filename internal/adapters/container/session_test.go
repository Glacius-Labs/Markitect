package container

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// fakeAgentd plays agentd's side of one connection with the frame codec.
type fakeAgentd struct {
	t   *testing.T
	in  io.Reader      // frames from the Orchestrator
	out io.WriteCloser // frames to the Orchestrator
}

func connect(t *testing.T, protocol int) (*Session, *fakeAgentd, error) {
	t.Helper()
	orchestratorR, orchestratorW := io.Pipe()
	agentR, agentW := io.Pipe()
	a := &fakeAgentd{t: t, in: orchestratorR, out: agentW}
	ended := make(chan struct{})
	go func() {
		a.control(frame.ControlMessage{Type: frame.TypeHello, Hello: &frame.Hello{Protocol: protocol, Version: "fake"}})
	}()
	wait := func() error {
		close(ended)
		return nil
	}
	s, err := newSession(context.Background(), orchestratorW, agentR, func() error {
		_ = agentW.Close()
		return wait()
	}, func() { _ = agentW.Close() })
	return s, a, err
}

func (a *fakeAgentd) send(f frame.Frame) {
	if err := frame.Write(a.out, f); err != nil {
		a.t.Errorf("fake agentd send: %v", err)
	}
}

func (a *fakeAgentd) control(c frame.ControlMessage) {
	payload, err := frame.EncodeControl(c)
	if err != nil {
		a.t.Error(err)
		return
	}
	a.send(frame.Frame{Channel: frame.Control, Payload: payload})
}

// finish behaves like agentd after the Orchestrator closes the connection.
func (a *fakeAgentd) finish() {
	for {
		if _, err := frame.Read(a.in); err != nil {
			_ = a.out.Close()
			return
		}
	}
}

func (a *fakeAgentd) read() frame.Frame {
	f, err := frame.Read(a.in)
	if err != nil {
		a.t.Errorf("fake agentd read: %v", err)
	}
	return f
}

func TestSessionCarriesAProviderLifetime(t *testing.T) {
	s, a, err := connect(t, frame.ProtocolVersion)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		start := a.read()
		c, _ := frame.DecodeControl(start.Payload)
		if c.Type != frame.TypeStart || c.Start.Argv[0] != "codex" {
			t.Errorf("start = %+v", c)
		}
		a.control(frame.ControlMessage{Type: frame.TypeStarted, Request: c.Request, Started: &frame.Started{PID: 42}})
		if in := a.read(); in.Channel != frame.ProviderIn || string(in.Payload) != "turn\n" {
			t.Errorf("input = %s %q", in.Channel, in.Payload)
		}
		a.send(frame.Frame{Channel: frame.ProviderErr, Payload: []byte("warning: slow\n")})
		call, _ := frame.EncodeToolCall(frame.ToolCall{ID: 1, Name: "submit_work", Arguments: json.RawMessage(`{"summary":"done"}`)})
		a.send(frame.Frame{Channel: frame.Tool, Payload: call})
		reply := a.read()
		r, err := frame.DecodeToolReply(reply.Payload)
		if err != nil || r.ID != 1 {
			t.Errorf("reply = %+v, %v", r, err)
		}
		a.send(frame.Frame{Channel: frame.ProviderOut, Payload: []byte(`{"result":"D-1"}` + "\n")})
		a.control(frame.ControlMessage{Type: frame.TypeExited, Exited: &frame.Exited{Code: 0}})
		a.finish()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	started, err := s.StartProvider(ctx, frame.Start{Argv: []string{"codex", "app-server"}, Dir: "/work"})
	if err != nil || started.PID != 42 {
		t.Fatalf("start = %+v, %v", started, err)
	}
	if err := s.WriteInput([]byte("turn\n")); err != nil {
		t.Fatal(err)
	}
	call := <-s.ToolCalls()
	if call.Name != "submit_work" {
		t.Fatalf("tool call = %+v", call)
	}
	if err := s.Reply(frame.ToolReply{ID: call.ID, Result: json.RawMessage(`{"content":[]}`)}); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(s.Stdout())
	if err != nil || !strings.Contains(string(out), "D-1") {
		t.Fatalf("stdout = %q, %v", out, err)
	}
	if exit := <-s.Exited(); exit.Code != 0 {
		t.Fatalf("exit = %+v", exit)
	}
	if !strings.Contains(string(s.Diagnostics()), "warning: slow") {
		t.Fatalf("diagnostics = %q", s.Diagnostics())
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionRefusesAnotherFrameProtocol(t *testing.T) {
	if _, _, err := connect(t, frame.ProtocolVersion+1); err == nil || !strings.Contains(err.Error(), "frame protocol") {
		t.Fatalf("connect = %v", err)
	}
}

func TestRefusedStartCarriesTheReason(t *testing.T) {
	s, a, err := connect(t, frame.ProtocolVersion)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	go func() {
		req, _ := frame.DecodeControl(a.read().Payload)
		a.control(frame.ControlMessage{Type: frame.TypeFailure, Request: req.Request, Failure: &frame.Failure{Message: "start provider: exec: no such file"}})
		a.finish()
	}()
	_, err = s.StartProvider(context.Background(), frame.Start{Argv: []string{"missing"}, Dir: "/work"})
	if err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("start = %v", err)
	}
}

func TestEndedConnectionIsNotAnExit(t *testing.T) {
	s, a, err := connect(t, frame.ProtocolVersion)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		a.read() // stop request
		_ = a.out.Close()
	}()
	_, err = s.Stop(context.Background(), 100)
	if err == nil || errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "before the provider's exit was observed") {
		t.Fatalf("stop on a lost connection = %v, want an uncertain end", err)
	}
	_ = s.Close()
}

func TestCloseRemovesAContainerThatDoesNotEnd(t *testing.T) {
	orchestratorR, orchestratorW := io.Pipe()
	agentR, agentW := io.Pipe()
	a := &fakeAgentd{t: t, in: orchestratorR, out: agentW}
	go a.control(frame.ControlMessage{Type: frame.TypeHello, Hello: &frame.Hello{Protocol: frame.ProtocolVersion}})
	go func() { _, _ = io.Copy(io.Discard, orchestratorR) }() // ignores the Orchestrator closing
	killed := make(chan struct{})
	s, err := newSession(context.Background(), orchestratorW, agentR, func() error { return nil }, func() {
		close(killed)
		_ = agentW.Close()
	})
	if err != nil {
		t.Fatal(err)
	}
	s.closeTimeout = 50 * time.Millisecond
	done := make(chan error, 1)
	go func() { done <- s.Close() }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrForcedClose) {
			t.Fatalf("Close = %v, want ErrForcedClose", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close waited for a container that never ended")
	}
	select {
	case <-killed:
	default:
		t.Fatal("the container was not removed by force")
	}
}

func TestALateReplyNeverAnswersALaterRequest(t *testing.T) {
	s, a, err := connect(t, frame.ProtocolVersion)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	firstAsked := make(chan uint64, 1)
	go func() {
		first, _ := frame.DecodeControl(a.read().Payload)
		firstAsked <- first.Request
		second, _ := frame.DecodeControl(a.read().Payload)
		// Answer the abandoned request first, then the current one.
		a.control(frame.ControlMessage{Type: frame.TypeCensus, Request: first.Request, Census: &frame.Census{Processes: []frame.Process{{PID: 1001, Command: "stale"}}}})
		a.control(frame.ControlMessage{Type: frame.TypeCensus, Request: second.Request, Census: &frame.Census{Processes: []frame.Process{{PID: 2002, Command: "current"}}}})
		a.finish()
	}()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { <-firstAsked; cancel() }()
	if _, err := s.Census(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("abandoned census = %v", err)
	}
	processes, err := s.Census(context.Background())
	if err != nil || len(processes) != 1 || processes[0].Command != "current" {
		t.Fatalf("second census = %+v, %v; the late reply must be dropped", processes, err)
	}
}
