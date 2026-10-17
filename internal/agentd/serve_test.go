package agentd_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// The test binary doubles as the provider CLI and as its MCP bridge, selected
// by AGENTD_TEST_PROVIDER, so tests exercise real processes and sockets.
func TestMain(m *testing.M) {
	if mode := os.Getenv("AGENTD_TEST_PROVIDER"); mode != "" {
		os.Exit(fakeProvider(mode))
	}
	os.Exit(m.Run())
}

func fakeProvider(mode string) int {
	socket := os.Getenv("AGENTD_TEST_SOCKET")
	switch mode {
	case "echo":
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			fmt.Printf("echo: %s\n", in.Text())
		}
		return 0
	case "bridge":
		if err := agentd.Bridge(context.Background(), socket, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "client", "client-unknown":
		tool := "submit_work"
		if mode == "client-unknown" {
			tool = "delete_everything"
		}
		return mcpClient(tool)
	case "reject":
		fmt.Fprintln(os.Stderr, "error: unexpected argument '--bogus' found")
		return 2
	case "stubborn-tree":
		signal.Ignore(syscall.SIGTERM)
		child := exec.Command(os.Args[0])
		child.Env = append(os.Environ(), "AGENTD_TEST_PROVIDER=sleep")
		if err := child.Start(); err != nil {
			return 1
		}
		fmt.Printf("child %d\n", child.Process.Pid)
		time.Sleep(time.Hour)
		return 0
	case "sleep":
		time.Sleep(time.Hour)
		return 0
	}
	return 3
}

// mcpClient behaves like a CLI: it starts its MCP server as a subprocess and
// speaks newline-delimited JSON-RPC to it over stdio.
func mcpClient(tool string) int {
	bridge := exec.Command(os.Args[0])
	bridge.Env = append(os.Environ(), "AGENTD_TEST_PROVIDER=bridge")
	bridge.Stderr = os.Stderr
	in, _ := bridge.StdinPipe()
	outPipe, _ := bridge.StdoutPipe()
	if err := bridge.Start(); err != nil {
		return 1
	}
	out := bufio.NewScanner(outPipe)
	request := func(id int, method string, params any) map[string]any {
		b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
		_, _ = in.Write(append(b, '\n'))
		if !out.Scan() {
			return nil
		}
		var r map[string]any
		_ = json.Unmarshal(out.Bytes(), &r)
		return r
	}
	request(1, "initialize", map[string]any{"protocolVersion": "2025-06-18"})
	_, _ = in.Write([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"))
	list := request(2, "tools/list", map[string]any{})
	result := request(3, "tools/call", map[string]any{"name": tool, "arguments": map[string]any{"summary": "done"}, "_meta": map[string]any{"turn": "T1"}})
	listed, _ := json.Marshal(list["result"])
	called, _ := json.Marshal(result)
	fmt.Printf("listed %s\n", listed)
	fmt.Printf("called %s\n", called)
	_ = in.Close()
	_ = bridge.Wait()
	return 0
}

// shortSocketPath stays within the Unix socket path limit on every host.
func shortSocketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ad")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "s")
}

type harness struct {
	t      *testing.T
	socket string
	toAgnt *io.PipeWriter
	frames chan frame.Frame
	served chan error
	stderr syncBuffer
}

// syncBuffer collects provider diagnostics for failure messages.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	hostR, hostW := io.Pipe()
	agentR, agentW := io.Pipe()
	h := &harness{
		t:      t,
		socket: shortSocketPath(t),
		toAgnt: hostW,
		frames: make(chan frame.Frame, 64),
		served: make(chan error, 1),
	}
	go func() {
		h.served <- agentd.Serve(context.Background(), hostR, agentW, agentd.Config{Version: "test", SocketPath: h.socket, StopGrace: 2 * time.Second})
		_ = agentW.Close()
	}()
	go func() {
		defer close(h.frames)
		for {
			f, err := frame.Read(agentR)
			if err != nil {
				return
			}
			if f.Channel == frame.ProviderErr {
				h.stderr.Write(f.Payload)
			}
			h.frames <- f
		}
	}()
	t.Cleanup(func() {
		_ = hostW.Close()
		select {
		case <-h.served:
		case <-time.After(10 * time.Second):
			t.Error("agentd did not end after the Host closed")
		}
	})
	hello := h.control(frame.TypeHello)
	if hello.Hello.Protocol != frame.ProtocolVersion {
		t.Fatalf("hello protocol = %d", hello.Hello.Protocol)
	}
	return h
}

func (h *harness) send(f frame.Frame) {
	h.t.Helper()
	if err := frame.Write(h.toAgnt, f); err != nil {
		h.t.Fatalf("send %s: %v", f.Channel, err)
	}
}

func (h *harness) sendControl(c frame.ControlMessage) {
	h.t.Helper()
	payload, err := frame.EncodeControl(c)
	if err != nil {
		h.t.Fatal(err)
	}
	h.send(frame.Frame{Channel: frame.Control, Payload: payload})
}

func (h *harness) start(mode string, tools ...frame.ToolSpec) {
	h.t.Helper()
	h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Start: &frame.Start{
		Argv:  []string{os.Args[0]},
		Dir:   h.t.TempDir(),
		Env:   []string{"AGENTD_TEST_PROVIDER=" + mode, "AGENTD_TEST_SOCKET=" + h.socket},
		Tools: tools,
	}})
	h.control(frame.TypeStarted)
}

// next returns the next frame, failing after a bound.
func (h *harness) next() frame.Frame {
	h.t.Helper()
	select {
	case f, ok := <-h.frames:
		if !ok {
			h.t.Fatal("agentd closed its output")
		}
		return f
	case <-time.After(15 * time.Second):
		h.t.Fatalf("no frame from agentd; provider stderr so far: %q", h.stderr.String())
	}
	return frame.Frame{}
}

// control returns the next control message of the wanted type and fails on
// any other control message. Provider output in between is collected.
func (h *harness) control(want frame.ControlType) frame.ControlMessage {
	h.t.Helper()
	for {
		f := h.next()
		if f.Channel != frame.Control {
			continue
		}
		c, err := frame.DecodeControl(f.Payload)
		if err != nil {
			h.t.Fatal(err)
		}
		if c.Type != want {
			detail := ""
			if c.Failure != nil {
				detail = c.Failure.Message
			}
			h.t.Fatalf("control %q %s, want %q", c.Type, detail, want)
		}
		return c
	}
}

// output reads provider standard output until it contains want.
func (h *harness) output(channel frame.Channel, want string) string {
	h.t.Helper()
	var got strings.Builder
	for !strings.Contains(got.String(), want) {
		f := h.next()
		if f.Channel == channel {
			got.Write(f.Payload)
		}
	}
	return got.String()
}

var submitWork = frame.ToolSpec{Name: "submit_work", Description: "Submit the delivery.", InputSchema: json.RawMessage(`{"type":"object"}`)}

func TestProviderStreamsRelayAndExit(t *testing.T) {
	h := newHarness(t)
	h.start("echo")
	h.send(frame.Frame{Channel: frame.ProviderIn, Payload: []byte("hello\n")})
	h.output(frame.ProviderOut, "echo: hello\n")
	h.send(frame.Frame{Channel: frame.ProviderIn}) // end of input
	if exit := h.control(frame.TypeExited); exit.Exited.Code != 0 || exit.Exited.Signal != "" {
		t.Fatalf("exit = %+v", exit.Exited)
	}
}

func TestToolCallReachesTheHostThroughTheBridge(t *testing.T) {
	h := newHarness(t)
	h.start("client", submitWork)
	var call frame.ToolCall
	for {
		f := h.next()
		if f.Channel == frame.Tool {
			var err error
			if call, err = frame.DecodeToolCall(f.Payload); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if call.Name != "submit_work" || string(call.Arguments) != `{"summary":"done"}` || !strings.Contains(string(call.Meta), `"turn":"T1"`) {
		t.Fatalf("tool call = %+v", call)
	}
	payload, _ := frame.EncodeToolReply(frame.ToolReply{ID: call.ID, Result: json.RawMessage(`{"content":[{"type":"text","text":"accepted as D-1"}]}`)})
	h.send(frame.Frame{Channel: frame.Tool, Payload: payload})
	out := h.output(frame.ProviderOut, "accepted as D-1")
	if !strings.Contains(out, `"name":"submit_work"`) || !strings.Contains(out, "accepted as D-1") {
		t.Fatalf("provider saw %q", out)
	}
	h.control(frame.TypeExited)
}

func TestUnknownToolNeverReachesTheHost(t *testing.T) {
	h := newHarness(t)
	h.start("client-unknown", submitWork)
	var out strings.Builder
	for {
		f := h.next()
		switch f.Channel {
		case frame.Tool:
			t.Fatalf("unknown tool forwarded: %s", f.Payload)
		case frame.ProviderOut:
			out.Write(f.Payload)
		case frame.Control:
			c, _ := frame.DecodeControl(f.Payload)
			if c.Type == frame.TypeExited {
				if !strings.Contains(out.String(), "unknown tool delete_everything") {
					t.Fatalf("provider saw %q", out.String())
				}
				return
			}
		}
	}
}

func TestRejectedProviderStartIsVisible(t *testing.T) {
	h := newHarness(t)
	h.start("reject")
	h.output(frame.ProviderErr, "unexpected argument '--bogus'")
	if exit := h.control(frame.TypeExited); exit.Exited.Code != 2 {
		t.Fatalf("exit = %+v", exit.Exited)
	}
}

func TestUnstartableProviderIsReportedNotFatal(t *testing.T) {
	h := newHarness(t)
	h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Start: &frame.Start{Argv: []string{filepath.Join(t.TempDir(), "missing-cli")}, Dir: t.TempDir()}})
	if f := h.control(frame.TypeFailure); !strings.Contains(f.Failure.Message, "start provider") {
		t.Fatalf("failure = %q", f.Failure.Message)
	}
}

func TestHostProtocolViolationsEndTheSession(t *testing.T) {
	cases := map[string]func(h *harness){
		"input before start": func(h *harness) {
			h.send(frame.Frame{Channel: frame.ProviderIn, Payload: []byte("x")})
		},
		"start twice": func(h *harness) {
			h.start("sleep")
			h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Start: &frame.Start{Argv: []string{"x"}, Dir: "/"}})
		},
		"reply to unknown call": func(h *harness) {
			payload, _ := frame.EncodeToolReply(frame.ToolReply{ID: 99, Result: json.RawMessage(`{}`)})
			h.send(frame.Frame{Channel: frame.Tool, Payload: payload})
		},
		"agentd-only message": func(h *harness) {
			h.sendControl(frame.ControlMessage{Type: frame.TypeExited, Exited: &frame.Exited{}})
		},
	}
	for name, violate := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			violate(h)
			h.control(frame.TypeFailure)
			select {
			case err := <-h.served:
				if err == nil {
					t.Fatal("violation ended the session without an error")
				}
				h.served <- err // let cleanup observe the end
			case <-time.After(10 * time.Second):
				t.Fatal("session continued after a protocol violation")
			}
		})
	}
}

func TestHostDisconnectStopsTheProvider(t *testing.T) {
	h := newHarness(t)
	h.start("sleep")
	_ = h.toAgnt.Close()
	select {
	case err := <-h.served:
		if err != nil {
			t.Fatalf("serve = %v", err)
		}
		h.served <- err
	case <-time.After(10 * time.Second):
		t.Fatal("agentd kept running after the Host disconnected")
	}
}
