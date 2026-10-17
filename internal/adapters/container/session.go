package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

const (
	// maxDiagnostics bounds the retained tail of the provider's standard error.
	maxDiagnostics = 64 << 10
	// closeTimeout bounds how long Close waits for agentd to stop the
	// provider and end before the container is removed by force.
	closeTimeout = 30 * time.Second
)

// Session is the Host side of one agentd connection. Provider output is
// buffered up to a bound, so control replies such as a census or the exit
// never wait for the consumer. Tool calls arrive on a short queue; the
// provider waits for their replies.
type Session struct {
	in    io.WriteCloser
	wmu   sync.Mutex
	hello frame.Hello
	wait  func() error
	kill  func()

	closeTimeout time.Duration

	stdout  *outputBuffer
	tools   chan frame.ToolCall
	exited  chan frame.Exited
	ended   chan struct{}
	closing chan struct{}

	mu          sync.Mutex
	nextRequest uint64
	replies     map[uint64]chan frame.ControlMessage
	diagnostics []byte
	failure     string
	err         error
	closeOnce   sync.Once
	closeErr    error
}

// newSession reads agentd's hello and starts demultiplexing. wait reports
// the end of the engine process that carries the connection; kill ends that
// process and removes the container by force.
func newSession(ctx context.Context, in io.WriteCloser, out io.Reader, wait func() error, kill func()) (*Session, error) {
	type helloResult struct {
		hello frame.Hello
		err   error
	}
	first := make(chan helloResult, 1)
	go func() {
		f, err := frame.Read(out)
		if err != nil {
			first <- helloResult{err: fmt.Errorf("read agentd hello: %w", err)}
			return
		}
		c, err := frame.DecodeControl(f.Payload)
		if err == nil && (f.Channel != frame.Control || c.Type != frame.TypeHello) {
			err = errors.New("agentd did not start with a hello message")
		}
		if err != nil {
			first <- helloResult{err: err}
			return
		}
		first <- helloResult{hello: *c.Hello}
	}()
	var hello frame.Hello
	abandon := func(err error) error {
		_ = in.Close()
		kill()
		return errors.Join(err, wait())
	}
	select {
	case r := <-first:
		if r.err != nil {
			return nil, abandon(r.err)
		}
		hello = r.hello
	case <-ctx.Done():
		return nil, abandon(ctx.Err())
	}
	if hello.Protocol != frame.ProtocolVersion {
		return nil, abandon(fmt.Errorf("agentd speaks frame protocol %d, the Host %d", hello.Protocol, frame.ProtocolVersion))
	}
	s := &Session{
		in: in, hello: hello, wait: wait, kill: kill, closeTimeout: closeTimeout,
		stdout:  newOutputBuffer(),
		tools:   make(chan frame.ToolCall, 16),
		replies: map[uint64]chan frame.ControlMessage{},
		exited:  make(chan frame.Exited, 1),
		ended:   make(chan struct{}),
		closing: make(chan struct{}),
	}
	go s.demux(out)
	return s, nil
}

// Hello returns agentd's announcement.
func (s *Session) Hello() frame.Hello { return s.hello }

func (s *Session) demux(out io.Reader) {
	defer close(s.ended)
	defer close(s.tools)
	for {
		f, err := frame.Read(out)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.fail(fmt.Errorf("read agentd frame: %w", err))
			}
			s.stdout.close()
			return
		}
		switch f.Channel {
		case frame.ProviderOut:
			if _, err := s.stdout.Write(f.Payload); err != nil {
				s.fail(fmt.Errorf("deliver provider output: %w", err))
			}
		case frame.ProviderErr:
			s.keepDiagnostics(f.Payload)
		case frame.Tool:
			call, err := frame.DecodeToolCall(f.Payload)
			if err != nil {
				s.fail(err)
				continue
			}
			select {
			case s.tools <- call:
			case <-s.closing:
			}
		case frame.Control:
			c, err := frame.DecodeControl(f.Payload)
			if err != nil {
				s.fail(err)
				continue
			}
			switch c.Type {
			case frame.TypeExited:
				s.exited <- *c.Exited
				s.stdout.close()
			case frame.TypeFailure:
				if c.Request == 0 {
					// A failure agentd sends on its own ends the session.
					s.mu.Lock()
					s.failure = c.Failure.Message
					s.mu.Unlock()
					continue
				}
				s.deliverReply(c)
			case frame.TypeStarted, frame.TypeCensus:
				s.deliverReply(c)
			default:
				s.fail(fmt.Errorf("agentd sent an unexpected %q message", c.Type))
			}
		default:
			s.fail(fmt.Errorf("agentd sent on the %s channel", f.Channel))
		}
	}
}

// deliverReply hands a reply to the request that is still waiting for it. A
// reply to an abandoned request is dropped, so it can never answer a later one.
func (s *Session) deliverReply(c frame.ControlMessage) {
	s.mu.Lock()
	ch, waiting := s.replies[c.Request]
	delete(s.replies, c.Request)
	s.mu.Unlock()
	if waiting {
		ch <- c
	}
}

// request sends one numbered control request and waits for its own reply.
func (s *Session) request(ctx context.Context, c frame.ControlMessage, purpose string) (frame.ControlMessage, error) {
	s.mu.Lock()
	s.nextRequest++
	c.Request = s.nextRequest
	reply := make(chan frame.ControlMessage, 1)
	s.replies[c.Request] = reply
	s.mu.Unlock()
	abandon := func() {
		s.mu.Lock()
		delete(s.replies, c.Request)
		s.mu.Unlock()
	}
	if err := s.sendControl(c); err != nil {
		abandon()
		return frame.ControlMessage{}, err
	}
	select {
	case r := <-reply:
		if r.Type == frame.TypeFailure {
			return frame.ControlMessage{}, fmt.Errorf("agentd refused the %s: %s", purpose, r.Failure.Message)
		}
		return r, nil
	case <-s.ended:
		abandon()
		return frame.ControlMessage{}, s.endError("before the " + purpose + " was answered")
	case <-ctx.Done():
		abandon()
		return frame.ControlMessage{}, ctx.Err()
	}
}

// StartProvider asks agentd to start the provider and returns its process ID.
// A refused start returns the provider's reported reason.
func (s *Session) StartProvider(ctx context.Context, req frame.Start) (frame.Started, error) {
	r, err := s.request(ctx, frame.ControlMessage{Type: frame.TypeStart, Start: &req}, "start")
	if err != nil {
		return frame.Started{}, err
	}
	if r.Type != frame.TypeStarted {
		return frame.Started{}, fmt.Errorf("agentd answered the start with %q", r.Type)
	}
	return *r.Started, nil
}

// Census returns the processes running in the container besides agentd and
// its init. It is a snapshot for a quiescence judgment: the Host decides
// which of the listed processes, if any, may remain.
func (s *Session) Census(ctx context.Context) ([]frame.Process, error) {
	r, err := s.request(ctx, frame.ControlMessage{Type: frame.TypeCensus, Census: &frame.Census{}}, "census")
	if err != nil {
		return nil, err
	}
	if r.Type != frame.TypeCensus {
		return nil, fmt.Errorf("agentd answered the census with %q", r.Type)
	}
	return r.Census.Processes, nil
}

// Stdout is the provider's standard output. It ends when the provider exits
// or the connection ends.
func (s *Session) Stdout() io.Reader { return s.stdout }

// WriteInput sends bytes to the provider's standard input.
func (s *Session) WriteInput(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	for off := 0; off < len(b); off += frame.MaxPayload {
		end := min(off+frame.MaxPayload, len(b))
		if err := s.write(frame.Frame{Channel: frame.ProviderIn, Payload: b[off:end]}); err != nil {
			return err
		}
	}
	return nil
}

// CloseInput ends the provider's standard input.
func (s *Session) CloseInput() error {
	return s.write(frame.Frame{Channel: frame.ProviderIn})
}

// ToolCalls delivers the provider's Host tool calls. It closes when the
// connection ends.
func (s *Session) ToolCalls() <-chan frame.ToolCall { return s.tools }

// Reply answers one tool call.
func (s *Session) Reply(r frame.ToolReply) error {
	payload, err := frame.EncodeToolReply(r)
	if err != nil {
		return err
	}
	return s.write(frame.Frame{Channel: frame.Tool, Payload: payload})
}

// Exited delivers the provider's observed exit once.
func (s *Session) Exited() <-chan frame.Exited { return s.exited }

// Stop asks agentd to stop the provider's process group within grace and
// waits for the observed exit.
func (s *Session) Stop(ctx context.Context, graceMillis int64) (frame.Exited, error) {
	if err := s.sendControl(frame.ControlMessage{Type: frame.TypeStop, Stop: &frame.Stop{GraceMillis: graceMillis}}); err != nil {
		return frame.Exited{}, err
	}
	select {
	case e := <-s.exited:
		return e, nil
	case <-s.ended:
		return frame.Exited{}, s.endError("before the provider's exit was observed")
	case <-ctx.Done():
		return frame.Exited{}, ctx.Err()
	}
}

// Diagnostics returns the retained tail of the provider's standard error.
func (s *Session) Diagnostics() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.diagnostics...)
}

// Err returns the first connection or protocol failure, if any.
func (s *Session) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close ends the connection. agentd then stops the provider and exits, which
// ends the container. If that does not happen within the close timeout, the
// container is removed by force. Close returns the engine process's result;
// it depends neither on the consumer reading Stdout or ToolCalls nor on a
// writer that is blocked.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.closing)
		s.stdout.detach()
		_ = s.in.Close()
		select {
		case <-s.ended:
		case <-time.After(s.closeTimeout):
			s.kill()
			<-s.ended
		}
		s.closeErr = s.wait()
	})
	return s.closeErr
}

func (s *Session) sendControl(c frame.ControlMessage) error {
	payload, err := frame.EncodeControl(c)
	if err != nil {
		return err
	}
	return s.write(frame.Frame{Channel: frame.Control, Payload: payload})
}

func (s *Session) write(f frame.Frame) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if err := frame.Write(s.in, f); err != nil {
		return fmt.Errorf("send %s frame to agentd: %w", f.Channel, err)
	}
	return nil
}

func (s *Session) keepDiagnostics(b []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diagnostics = append(s.diagnostics, b...)
	if over := len(s.diagnostics) - maxDiagnostics; over > 0 {
		s.diagnostics = append([]byte(nil), s.diagnostics[over:]...)
	}
}

func (s *Session) fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

func (s *Session) endError(when string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := "the agentd connection ended " + when
	if s.failure != "" {
		msg += ": " + s.failure
	}
	return errors.Join(errors.New(msg), s.err)
}
