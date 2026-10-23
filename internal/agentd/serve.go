// Package agentd is the session helper that runs inside every Agent
// container. It starts the provider CLI on the Orchestrator's request, relays the
// provider's standard streams, serves the Orchestrator tool catalogue to the CLI as
// an MCP server on a Unix socket, and reports the provider's exit. It adds no
// policy: the Orchestrator decides every tool call.
package agentd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// DefaultSocketPath is where agentd listens for MCP bridge connections. The
// container mounts a writable temporary directory at /tmp.
const DefaultSocketPath = "/tmp/markitect-agentd.sock"

// Config is agentd's fixed setup for one session.
type Config struct {
	// Version is agentd's build version, reported in its hello message.
	Version string
	// SocketPath is the Unix socket for MCP bridges.
	SocketPath string
	// StopGrace bounds the polite stop when the Orchestrator connection closes.
	StopGrace time.Duration
}

// Serve runs one session over the Orchestrator connection. It returns nil when the
// Orchestrator closes the connection cleanly and the provider, if started, has been
// stopped. An Orchestrator protocol violation returns an error after stopping the
// provider, so the container ends instead of guessing.
func Serve(ctx context.Context, orchestrator io.Reader, out io.Writer, cfg Config) error {
	if cfg.SocketPath == "" {
		cfg.SocketPath = DefaultSocketPath
	}
	if cfg.StopGrace <= 0 {
		cfg.StopGrace = 5 * time.Second
	}
	s := &server{cfg: cfg, out: &frameWriter{w: out}, pending: map[uint64]chan frame.ToolReply{}, stopping: make(chan struct{})}
	defer s.shutdown()
	if err := s.sendControl(frame.ControlMessage{Type: frame.TypeHello, Hello: &frame.Hello{Protocol: frame.ProtocolVersion, Version: cfg.Version}}); err != nil {
		return err
	}
	frames := make(chan frame.Frame)
	readErr := make(chan error, 1)
	go func() {
		for {
			f, err := frame.Read(orchestrator)
			if err != nil {
				readErr <- err
				return
			}
			frames <- f
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErr:
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read Orchestrator frame: %w", err)
		case f := <-frames:
			if err := s.handle(f); err != nil {
				_ = s.sendControl(frame.ControlMessage{Type: frame.TypeFailure, Failure: &frame.Failure{Message: err.Error()}})
				return err
			}
		}
	}
}

type server struct {
	cfg Config
	out *frameWriter

	mu       sync.Mutex
	provider *provider
	tools    map[string]frame.ToolSpec
	toolList []frame.ToolSpec
	listener net.Listener
	pending  map[uint64]chan frame.ToolReply
	nextID   uint64
	closed   bool
	stopping chan struct{}
	reports  sync.WaitGroup
}

func (s *server) handle(f frame.Frame) error {
	switch f.Channel {
	case frame.Control:
		c, err := frame.DecodeControl(f.Payload)
		if err != nil {
			return err
		}
		switch c.Type {
		case frame.TypeStart:
			return s.start(c.Request, *c.Start)
		case frame.TypeStop:
			return s.stop(time.Duration(c.Stop.GraceMillis) * time.Millisecond)
		case frame.TypeCensus:
			return s.census(c.Request)
		default:
			return fmt.Errorf("the Orchestrator may not send a %q message", c.Type)
		}
	case frame.ProviderIn:
		p := s.current()
		if p == nil {
			return errors.New("provider input before start")
		}
		return p.write(f.Payload)
	case frame.Tool:
		r, err := frame.DecodeToolReply(f.Payload)
		if err != nil {
			return err
		}
		return s.deliver(r)
	default:
		return fmt.Errorf("the Orchestrator may not send on the %s channel", f.Channel)
	}
}

func (s *server) start(request uint64, req frame.Start) error {
	s.mu.Lock()
	if s.provider != nil {
		select {
		case <-s.provider.reported:
			// The previous provider and its exit report ended. The Agent's
			// container and connection stay alive for another provider process.
		default:
			s.mu.Unlock()
			return errors.New("the provider was already started")
		}
	}
	tools := make(map[string]frame.ToolSpec, len(req.Tools))
	for _, t := range req.Tools {
		if t.Name == "" || len(t.InputSchema) == 0 {
			s.mu.Unlock()
			return errors.New("every tool needs a name and an input schema")
		}
		if _, dup := tools[t.Name]; dup {
			s.mu.Unlock()
			return fmt.Errorf("tool %q is offered twice", t.Name)
		}
		tools[t.Name] = t
	}
	needsListener := s.listener == nil
	s.provider = nil
	s.tools, s.toolList = tools, append([]frame.ToolSpec(nil), req.Tools...)
	s.mu.Unlock()
	if needsListener {
		if err := os.Remove(s.cfg.SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("prepare MCP socket: %w", err)
		}
		l, err := net.Listen("unix", s.cfg.SocketPath)
		if err != nil {
			return fmt.Errorf("listen for MCP bridges: %w", err)
		}
		s.mu.Lock()
		s.listener = l
		s.mu.Unlock()
		go s.acceptBridges(l)
	}

	cmd := exec.Command(req.Argv[0], req.Argv[1:]...)
	cmd.Dir = req.Dir
	cmd.Env = append(os.Environ(), req.Env...)
	// Hold the frame writer while starting, so the started report precedes
	// any provider output on the connection.
	s.out.mu.Lock()
	p, err := startProvider(cmd, s.out)
	if err != nil {
		s.out.mu.Unlock()
		// A refused start is reported to the Orchestrator as an observation, not a
		// protocol violation; the session stays open for inspection.
		return s.sendControl(frame.ControlMessage{Type: frame.TypeFailure, Request: request, Failure: &frame.Failure{Message: "start provider: " + err.Error()}})
	}
	payload, err := frame.EncodeControl(frame.ControlMessage{Type: frame.TypeStarted, Request: request, Started: &frame.Started{PID: p.pid()}})
	if err == nil {
		err = frame.Write(s.out.w, frame.Frame{Channel: frame.Control, Payload: payload})
	}
	s.out.mu.Unlock()
	s.mu.Lock()
	s.provider = p
	p.reported = make(chan struct{})
	s.mu.Unlock()
	s.reports.Add(1)
	go func() {
		defer s.reports.Done()
		exit := p.wait()
		close(p.reported)
		_ = s.sendControl(frame.ControlMessage{Type: frame.TypeExited, Exited: &exit})
	}()
	return err
}

func (s *server) stop(grace time.Duration) error {
	p := s.current()
	if p == nil {
		return errors.New("stop before start")
	}
	go p.stop(grace)
	return nil
}

func (s *server) current() *provider {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.provider
}

func (s *server) deliver(r frame.ToolReply) error {
	s.mu.Lock()
	ch, ok := s.pending[r.ID]
	delete(s.pending, r.ID)
	s.mu.Unlock()
	if !ok {
		s.mu.Lock()
		issued := r.ID != 0 && r.ID <= s.nextID
		s.mu.Unlock()
		if issued {
			// The call's bridge already left; the protocol has no cancel
			// message, so a late reply is expected and dropped.
			return nil
		}
		return fmt.Errorf("reply to tool call %d, which agentd never issued", r.ID)
	}
	ch <- r
	return nil
}

// call forwards one tool call to the Orchestrator and waits for its reply, the
// bridge's departure (done) or the session's end.
func (s *server) call(done <-chan struct{}, name string, args, meta []byte) (frame.ToolReply, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return frame.ToolReply{}, errors.New("session is closing")
	}
	s.nextID++
	id := s.nextID
	ch := make(chan frame.ToolReply, 1)
	s.pending[id] = ch
	s.mu.Unlock()
	payload, err := frame.EncodeToolCall(frame.ToolCall{ID: id, Name: name, Arguments: args, Meta: meta})
	if err == nil {
		err = s.out.write(frame.Frame{Channel: frame.Tool, Payload: payload})
	}
	if err != nil {
		s.forget(id)
		return frame.ToolReply{}, err
	}
	select {
	case r := <-ch:
		return r, nil
	case <-done:
		s.forget(id)
		return frame.ToolReply{}, errors.New("the MCP bridge left")
	case <-s.stopping:
		s.forget(id)
		return frame.ToolReply{}, errors.New("session is closing")
	}
}

func (s *server) forget(id uint64) {
	s.mu.Lock()
	delete(s.pending, id)
	s.mu.Unlock()
}

func (s *server) catalogue() ([]frame.ToolSpec, map[string]frame.ToolSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.toolList, s.tools
}

func (s *server) sendControl(c frame.ControlMessage) error {
	payload, err := frame.EncodeControl(c)
	if err != nil {
		return err
	}
	return s.out.write(frame.Frame{Channel: frame.Control, Payload: payload})
}

// shutdown fails pending tool calls, stops the provider within the stop
// grace and sends its exit report before Serve returns.
func (s *server) shutdown() {
	s.mu.Lock()
	s.closed = true
	close(s.stopping)
	p, l := s.provider, s.listener
	s.mu.Unlock()
	if l != nil {
		_ = l.Close()
	}
	if p != nil {
		p.stop(s.cfg.StopGrace)
		p.wait()
	}
	s.reports.Wait()
	if l != nil {
		_ = os.Remove(s.cfg.SocketPath)
	}
}

// frameWriter serializes whole frames from concurrent senders.
type frameWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (fw *frameWriter) write(f frame.Frame) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return frame.Write(fw.w, f)
}

// streamWriter turns a provider stream into frames of bounded size.
type streamWriter struct {
	out     *frameWriter
	channel frame.Channel
}

const streamChunk = 64 << 10

func (sw streamWriter) Write(b []byte) (int, error) {
	for off := 0; off < len(b); off += streamChunk {
		end := min(off+streamChunk, len(b))
		chunk := append([]byte(nil), b[off:end]...)
		if err := sw.out.write(frame.Frame{Channel: sw.channel, Payload: chunk}); err != nil {
			return off, err
		}
	}
	return len(b), nil
}
