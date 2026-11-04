// Package agents starts Markitect Desktop's chats on the user's own token, each
// in a hardened container of its own (Chat). It knows no provider; the Claude
// Code and Codex adapters supply the launch contract.
package agents

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// Fixed container paths and the session user. The image owns /home/agent and
// /work by that user, so fresh volumes mounted there are writable for it.
const (
	homeDir  = "/home/agent"
	workDir  = "/work"
	gitPath  = "/usr/bin/git"
	user     = "1000:1000"
	stopWait = 10 * time.Second
	// maxLine bounds one line of provider output.
	maxLine = 8 << 20
)

// settle bounds how long a process the turn's end cut off may take to end in the
// container before the census counts it.
var settle = 30 * time.Second

// Runtime is the container engine as the agent runtime uses it.
// EngineRuntime adapts the container client.
type Runtime interface {
	CreateVolume(ctx context.Context, name string, labels map[string]string) error
	RemoveVolume(ctx context.Context, name string) error
	Start(ctx context.Context, spec container.ContainerSpec) (Session, error)
	Exec(ctx context.Context, name, dir string, argv []string, stdin io.Reader, stdout io.Writer) error
}

// Session is one agentd connection (container.Session).
type Session interface {
	Initialize(context.Context, frame.Initialize) error
	ClearPrivateHome(context.Context) error
	StartProvider(context.Context, frame.Start) (frame.Started, error)
	Stdout() io.Reader
	WriteInput([]byte) error
	CloseInput() error
	ToolCalls() <-chan frame.ToolCall
	Reply(frame.ToolReply) error
	Stop(context.Context, int64) (frame.Exited, error)
	Census(context.Context) ([]frame.Process, error)
	Diagnostics() []byte
	Close() error
}

type engineRuntime struct{ e *container.Engine }

// EngineRuntime is the Runtime of a real container engine.
func EngineRuntime(e *container.Engine) Runtime { return engineRuntime{e} }

func (r engineRuntime) CreateVolume(ctx context.Context, name string, labels map[string]string) error {
	return r.e.CreateVolume(ctx, name, labels)
}
func (r engineRuntime) RemoveVolume(ctx context.Context, name string) error {
	return r.e.RemoveVolume(ctx, name)
}
func (r engineRuntime) Start(ctx context.Context, spec container.ContainerSpec) (Session, error) {
	s, err := r.e.Start(ctx, spec)
	if err != nil {
		return nil, err
	}
	return s, nil
}
func (r engineRuntime) Exec(ctx context.Context, name, dir string, argv []string, stdin io.Reader, stdout io.Writer) error {
	return r.e.Exec(ctx, name, dir, argv, stdin, stdout)
}

// Limits bound one agent's container.
type Limits struct {
	CPUs        float64
	MemoryBytes int64
	Pids        int
	TmpBytes    int64
}

// Spec is everything one chat's container needs.
type Spec struct {
	// Name is the container's unique name; its volumes carry it as prefix.
	Name string
	// Labels mark the container and its volumes, so that a sweep finds them.
	Labels map[string]string
	Image  string
	Limits Limits
	// Network reaches the provider. Hosts and DNS are passed to the engine as
	// they are.
	Network    string
	Hosts, DNS []string
	// Deadline bounds the next turn.
	Deadline time.Time
	// Token is the user's account token; it reaches the provider only through
	// its environment.
	Token  string
	Model  string
	Effort string
	// Transcript, when set, receives the agent's provider output and tool
	// calls as JSON lines, with every secret redacted.
	Transcript io.Writer
	// Now is the clock; nil is time.Now.
	Now func() time.Time
}

// Agent is one started agent.
type Agent struct {
	spec     Spec
	provider Provider
	rt       Runtime
	session  Session
	doc      harness.Document
	secrets  [][]byte
	volumes  []string

	mu           sync.Mutex
	conversation string
	turns        int
	closed       bool

	// progress, when set, hears each line's progress while a turn runs, if
	// the provider's reader can tell it.
	progress func(Progress)
}

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// StartError is a start that failed in a way worth telling apart: a refused
// or expired login is the provider's, everything else is Other.
type StartError struct {
	Class Class
	Err   error
}

func (e *StartError) Error() string { return fmt.Sprintf("agent start (%s): %v", e.Class, e.Err) }
func (e *StartError) Unwrap() error { return e.Err }

// checkLogin runs the CLI's own login report before any inference.
func (a *Agent) checkLogin(ctx context.Context) error {
	argv, env := a.provider.AuthStatus(homeDir)
	env = append(env, a.provider.TokenVariable()+"="+a.spec.Token)
	if _, err := a.session.StartProvider(ctx, frame.Start{Argv: argv, Dir: workDir, Env: env, Tools: []frame.ToolSpec{}}); err != nil {
		return err
	}
	if err := a.session.CloseInput(); err != nil {
		return err
	}
	read := make(chan []byte, 1)
	go func() {
		out, _ := io.ReadAll(io.LimitReader(a.session.Stdout(), 1<<20))
		read <- out
	}()
	var out []byte
	var readErr error
	select {
	case out = <-read:
	case <-ctx.Done():
		readErr = ctx.Err()
	}
	exit, err := a.session.Stop(context.WithoutCancel(ctx), stopWait.Milliseconds())
	if readErr != nil {
		<-read
	}
	if err := errors.Join(readErr, err); err != nil {
		return err
	}
	if exit.Code != 0 {
		return fmt.Errorf("%s reported no usable login (exit %d)", a.provider.Name(), exit.Code)
	}
	return a.provider.CheckAuth(out)
}

// Send runs one turn: the message goes to the provider conversation, and the
// turn ends when the provider ends it, at the deadline or when ctx ends. The first turn's message is the
// harness's task; message is then appended to it when not empty.
func (a *Agent) Send(ctx context.Context, message string) (Outcome, error) {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return Outcome{}, errors.New("agents: the agent is closed")
	}
	first := a.turns == 0
	a.turns++
	conversation := a.conversation
	a.mu.Unlock()
	if first {
		message = strings.TrimSpace(a.doc.Task + "\n\n" + message)
	}
	turn := Turn{First: first, Conversation: conversation, Message: message, Document: a.doc, Model: a.spec.Model, Effort: a.spec.Effort, Deadline: a.spec.Deadline}
	argv, env, stdin, err := a.provider.Turn(turn)
	if err != nil {
		return Outcome{}, err
	}
	env = append(env, a.provider.TokenVariable()+"="+a.spec.Token)
	turnCtx, cancel := context.WithDeadline(ctx, a.spec.Deadline)
	defer cancel()
	if _, err := a.session.StartProvider(turnCtx, frame.Start{Argv: argv, Dir: workDir, Env: env, Tools: []frame.ToolSpec{}}); err != nil {
		return Outcome{}, err
	}
	a.record("turn", map[string]any{"first": first, "conversation": conversation, "message": message})
	if err := errors.Join(a.session.WriteInput(stdin), a.session.CloseInput()); err != nil {
		_, stopErr := a.session.Stop(context.WithoutCancel(ctx), stopWait.Milliseconds())
		return Outcome{}, errors.Join(err, stopErr)
	}
	reader := a.provider.Reader(turn)
	lines := make(chan error, 1)
	go func() { lines <- a.read(reader) }()

	var lineErr error
	var end End
	select {
	case lineErr = <-lines:
	case <-turnCtx.Done():
		end = Deadline
		if ctx.Err() != nil {
			end = Stopped
		}
	}
	exit, stopErr := a.session.Stop(context.WithoutCancel(ctx), stopWait.Milliseconds())
	if end != "" {
		<-lines
	}
	cancel()
	if stopErr != nil {
		return Outcome{}, stopErr
	}
	if err := a.quiet(context.WithoutCancel(ctx)); err != nil {
		return Outcome{}, err
	}
	out := reader.Outcome()
	switch {
	case end != "":
		out.End, out.Class = end, ""
	case lineErr != nil:
		out.End, out.Class, out.Detail = Failed, Other, string(a.redact([]byte(lineErr.Error())))
	case out.End == "" && exit.Code != 0:
		out.End, out.Class, out.Detail = Failed, Other, fmt.Sprintf("the provider exited with %d without a result", exit.Code)
	case out.End == "":
		out.End, out.Class, out.Detail = Failed, Other, "the provider ended without a result"
	}
	out.Detail = string(a.redact([]byte(out.Detail)))
	a.mu.Lock()
	if out.Conversation != "" {
		a.conversation = out.Conversation
	}
	a.mu.Unlock()
	a.record("outcome", out)
	return out, nil
}

// read feeds the turn's output to the provider's reader line by line until
// the output ends. A line that breaks the launch contract ends the turn.
func (a *Agent) read(r Reader) error {
	scanner := bufio.NewScanner(a.session.Stdout())
	scanner.Buffer(make([]byte, 64<<10), maxLine)
	for scanner.Scan() {
		line := scanner.Bytes()
		a.write(line)
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if err := r.Line(line); err != nil {
			go io.Copy(io.Discard, a.session.Stdout())
			return err
		}
		if pr, ok := r.(ProgressReader); ok && a.progress != nil {
			if p := pr.Progress(line); p != (Progress{}) {
				a.progress(p)
			}
		}
	}
	return scanner.Err()
}

// quiet requires that no process of the turn remains in the container; anything
// still running after settle is an error.
func (a *Agent) quiet(ctx context.Context) error {
	limit := time.Now().Add(settle)
	for {
		left, err := a.session.Census(ctx)
		if err != nil {
			return err
		}
		if len(left) == 0 {
			return nil
		}
		if time.Now().After(limit) {
			return fmt.Errorf("agents: %d processes outlived the turn", len(left))
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// Close stops the provider, clears the private home, ends the container and removes the agent's volumes. Every failure is reported.
func (a *Agent) Close(ctx context.Context) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	a.mu.Unlock()
	var errs []error
	if a.session != nil {
		errs = append(errs, a.session.ClearPrivateHome(ctx), a.session.Close())
	}
	for _, v := range a.volumes {
		errs = append(errs, a.rt.RemoveVolume(ctx, v))
	}
	return errors.Join(errs...)
}

// redact replaces every secret, also JSON-escaped, with a marker.
func (a *Agent) redact(b []byte) []byte {
	for _, s := range a.secrets {
		if len(s) < 8 {
			continue
		}
		b = bytes.ReplaceAll(b, s, []byte("[redacted]"))
	}
	return b
}

// git runs one Git command in the container's clone and redacts its
// diagnostics.
//
// The clone is the agent's to change, so the host's Git runs no hooks, no
// file-system monitor and no replace refs the agent may have planted. A
// command that ran and failed stays an *container.ExecError.
func (a *Agent) git(ctx context.Context, argv []string, stdin string, stdout io.Writer) error {
	argv = append([]string{argv[0], "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "--no-replace-objects"}, argv[1:]...)
	err := a.rt.Exec(ctx, a.spec.Name, workDir, argv, strings.NewReader(stdin), stdout)
	var exit *container.ExecError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exit):
		return &container.ExecError{Code: exit.Code, Diagnostics: string(a.redact([]byte(exit.Diagnostics)))}
	default:
		return errors.New(string(a.redact([]byte(err.Error()))))
	}
}

// errCleanup marks a start whose cleanup after a failure did not finish.
var errCleanup = errors.New("cleanup")
