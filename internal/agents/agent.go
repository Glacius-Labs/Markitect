// Package agents starts and drives one agent of a Manager (concept 2.0
// §6.2, §9): a maker or a checker, each in its own hardened Linux container
// that clones the Run's remote, with a provider CLI logged in through an
// access-only copy of the account login. It implements the engine's agent
// port: a start renders the harness from the project at the Run's base, runs
// the provider and serves the Manager tools: submit and review push to the
// Run's remote with the verdict trailers, the host tools reach the engine.
// It knows no provider; the Claude Code and Codex adapters supply the launch
// contract.
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
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
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
	// loginMargin is how long an access token must outlive the deadline: a
	// CLI may try to refresh shortly before expiry, which the access-only
	// copy cannot.
	loginMargin = 10 * time.Minute
)

// settle bounds how long a Git command of a tool call that the turn's end
// cut off may take to end in the container before the census counts it.
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

// Spec is everything one agent needs: the engine's start and host, and the
// adapter's configuration.
type Spec struct {
	// Name is the container's unique name; its volumes carry it as prefix.
	Name string
	// Labels mark the container and its volumes, so that a sweep finds them.
	Labels map[string]string
	Image  string
	Limits Limits
	// Network reaches the provider. It must be a user-defined network: the
	// Docker CLI on Linux refuses to combine the engine's default "bridge"
	// with the Run's network.
	// The container also joins the Run's network, Start.Git.Network; without
	// a Network it reaches the remote only. Hosts and DNS are passed to the
	// engine as they are.
	Network    string
	Hosts, DNS []string
	Start      agentport.Start
	Host       agentport.Host
	// Login is the operator's login file for the provider.
	Login  []byte
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
	pkg      string
	doc      harness.Document
	tools    []harness.Tool
	secrets  [][]byte
	volumes  []string

	mu           sync.Mutex
	conversation string
	turns        int
	closed       bool
}

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// Start starts the agent's container, clones the Run's remote at the
// Manager's branch, renders the harness from the project at the Run's base,
// installs the login and checks that the CLI uses the account login. It
// starts no turn. A failed start removes what it created.
func Start(ctx context.Context, rt Runtime, p Provider, spec Spec) (*Agent, error) {
	now := time.Now
	if spec.Now != nil {
		now = spec.Now
	}
	s, g := spec.Start, spec.Start.Git
	if !namePattern.MatchString(spec.Name) || len(spec.Labels) == 0 || spec.Host == nil || g.URL == "" || g.User == "" || g.Token == "" || g.Network == "" || g.Base == "" || spec.Model == "" || spec.Effort == "" || s.Deadline.IsZero() || s.Manager.ID == "" {
		return nil, errors.New("agents: an agent needs a plain name, labels, a host, its remote access with the Run's network and base, a model, an effort, its Manager and a deadline")
	}
	if strings.ContainsAny(g.User+g.Token, ":@/\n\r ") || !commitPattern.MatchString(g.Base) {
		return nil, errors.New("agents: the remote user, token and base must be plain")
	}
	pkg := harness.Package(s.Manager.ID)
	if g.Branch != runremote.Branch(pkg) {
		return nil, fmt.Errorf("agents: the branch %q is not Manager %s's branch %q", g.Branch, s.Manager.ID, runremote.Branch(pkg))
	}
	login, err := p.Login(spec.Login, now())
	if err != nil {
		return nil, &StartError{Class: LoginFailed, Err: err}
	}
	if !login.Expires.IsZero() && !login.Expires.After(s.Deadline.Add(loginMargin)) {
		return nil, &StartError{Class: LoginFailed, Err: fmt.Errorf("the %s login's access token expires at %s, before the deadline %s; refresh the login on the host first", p.Name(), login.Expires.UTC().Format(time.RFC3339), s.Deadline.UTC().Format(time.RFC3339))}
	}
	a := &Agent{spec: spec, provider: p, rt: rt, pkg: pkg, secrets: append(login.Secrets, []byte(g.Token))}
	if err := a.start(ctx, login); err != nil {
		if cerr := a.Close(context.WithoutCancel(ctx)); cerr != nil {
			err = errors.Join(err, fmt.Errorf("%w: %v", errCleanup, cerr))
		}
		return nil, err
	}
	return a, nil
}

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// StartError is a start that failed in a way the evaluation sorts: a refused
// or expired login is the provider's, everything else is Other.
type StartError struct {
	Class Class
	Err   error
}

func (e *StartError) Error() string { return fmt.Sprintf("agent start (%s): %v", e.Class, e.Err) }
func (e *StartError) Unwrap() error { return e.Err }

func (a *Agent) start(ctx context.Context, login Login) error {
	s := a.spec
	for _, v := range []string{s.Name + "-home", s.Name + "-work"} {
		if err := a.rt.CreateVolume(ctx, v, s.Labels); err != nil {
			return err
		}
		a.volumes = append(a.volumes, v)
	}
	labels := map[string]string{}
	for k, v := range s.Labels {
		labels[k] = v
	}
	spec := container.ContainerSpec{
		Name: s.Name, Image: s.Image, Labels: labels, User: user, Workdir: workDir,
		Mounts: []container.Mount{{Volume: s.Name + "-home", Target: homeDir}, {Volume: s.Name + "-work", Target: workDir}},
		CPUs:   s.Limits.CPUs, MemoryBytes: s.Limits.MemoryBytes, Pids: s.Limits.Pids, TmpBytes: s.Limits.TmpBytes,
		Network: s.Start.Git.Network, Hosts: s.Hosts, DNS: s.DNS,
	}
	if s.Network != "" {
		spec.Network, spec.MoreNetworks = s.Network, []string{s.Start.Git.Network}
	}
	session, err := a.rt.Start(ctx, spec)
	if err != nil {
		return err
	}
	a.session = session
	if err := a.clone(ctx); err != nil {
		return err
	}
	in := harness.Input{Start: s.Start}
	if in.Project, err = a.loadBase(ctx); err != nil {
		return err
	}
	if s.Start.Role == agentport.Checker && s.Start.Mode == agentport.Apply {
		if in.Submission, err = a.submission(ctx); err != nil {
			return err
		}
	}
	if a.doc, err = harness.Render(in); err != nil {
		return err
	}
	a.tools = harness.Tools(s.Start)
	home, err := a.provider.Home(login, a.doc)
	if err != nil {
		return err
	}
	if err := session.Initialize(ctx, home); err != nil {
		return err
	}
	if err := a.checkLogin(ctx); err != nil {
		return &StartError{Class: LoginFailed, Err: err}
	}
	return nil
}

// checkLogin runs the CLI's own login report before any inference.
func (a *Agent) checkLogin(ctx context.Context) error {
	argv, env := a.provider.AuthStatus()
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

// clone stores the agent's remote credential in the private home, where
// only Git reads it, and clones the remote at the Manager's branch. A
// checker in apply starts at the branch's tip, the submission it reviews.
func (a *Agent) clone(ctx context.Context) error {
	s, g := a.spec, a.spec.Start.Git
	host, err := remoteHost(g.URL)
	if err != nil {
		return err
	}
	identity := fmt.Sprintf("%s %s", s.Start.Manager.ID, s.Start.Role)
	steps := []struct {
		argv  []string
		stdin string
	}{
		{[]string{gitPath, "config", "--global", "credential.helper", "store"}, ""},
		{[]string{gitPath, "config", "--global", "user.name", identity}, ""},
		{[]string{gitPath, "config", "--global", "user.email", g.User + "@run.markitect.invalid"}, ""},
		{[]string{gitPath, "config", "--global", "core.autocrlf", "false"}, ""},
		{[]string{gitPath, "credential", "approve"}, fmt.Sprintf("protocol=http\nhost=%s\nusername=%s\npassword=%s\n\n", host, g.User, g.Token)},
		{[]string{gitPath, "clone", "--quiet", "--origin", "origin", "--branch", g.Branch, g.URL, workDir}, ""},
	}
	for _, step := range steps {
		if err := a.git(ctx, step.argv, step.stdin, io.Discard); err != nil {
			return fmt.Errorf("prepare the clone: %w", err)
		}
	}
	return nil
}

func remoteHost(url string) (string, error) {
	rest, ok := strings.CutPrefix(url, "http://")
	host, _, found := strings.Cut(rest, "/")
	if !ok || !found || host == "" || strings.ContainsAny(host, "@\n\r ") {
		return "", fmt.Errorf("agents: the remote %q is not a plain http URL on the Run's network", url)
	}
	return host, nil
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

// Send runs one turn: the message goes to the provider conversation, and the
// turn ends when the provider ends it, at the start's deadline or when ctx
// ends. Tool calls are served while it runs. The first turn's message is the
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
	names := make([]string, len(a.tools))
	specs := make([]frame.ToolSpec, len(a.tools))
	for i, t := range a.tools {
		names[i] = t.Name
		specs[i] = frame.ToolSpec{Name: t.Name, Description: t.Description, InputSchema: t.Schema}
	}
	turn := Turn{First: first, Conversation: conversation, Message: message, Document: a.doc, Model: a.spec.Model, Effort: a.spec.Effort, Tools: names, Deadline: a.spec.Start.Deadline}
	argv, env, stdin, err := a.provider.Turn(turn)
	if err != nil {
		return Outcome{}, err
	}
	turnCtx, cancel := context.WithDeadline(ctx, a.spec.Start.Deadline)
	defer cancel()
	if _, err := a.session.StartProvider(turnCtx, frame.Start{Argv: argv, Dir: workDir, Env: env, Tools: specs}); err != nil {
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
	serving, served := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(served)
		a.serve(turnCtx, serving)
	}()

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
	// A tool call still running is cut off and awaited, so that nothing of
	// the turn reaches the host or the remote after Send returns.
	close(serving)
	cancel()
	<-served
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
	}
	return scanner.Err()
}

// quiet requires that no process of the turn remains in the container. A
// Git command of a cut-off tool call may take a moment to end after its
// engine client was stopped; anything still running after settle is an
// error.
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

// serve answers tool calls until the turn ends.
func (a *Agent) serve(ctx context.Context, done <-chan struct{}) {
	calls := a.session.ToolCalls()
	for {
		select {
		case <-done:
			return
		case call, ok := <-calls:
			if !ok {
				return
			}
			// A call queued when the turn ended is not served.
			select {
			case <-done:
				return
			default:
			}
			a.record("tool-call", call)
			reply := a.call(ctx, call)
			a.record("tool-reply", reply)
			_ = a.session.Reply(reply)
		}
	}
}

// Close stops the provider, removes the login from the private home, ends
// the container and removes the agent's volumes. Every failure is reported.
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
