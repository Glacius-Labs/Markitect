package claudecode

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

const stopGrace = 10 * time.Second
const stopTimeout = 20 * time.Second

// maxAuthStatusBytes bounds the output of `claude auth status`.
const maxAuthStatusBytes = 64 << 10

// Agents is one prepared native profile implementing the actual Run session consumer.
// Outer composition selects the profile; Project/Run never resolve runtime profiles.
type Agents struct {
	// TranscriptRoot is an optional explicit absolute evidence directory, fixed before Open.
	TranscriptRoot string
	start          RuntimeStarter
	prepared       runtimeconfig.PreparedProfile
	profile        Profile
	identity       string
	credentials    CredentialResolver
	retry          retryPolicy
	quiet          quiescencePolicy
	hostTime       func(context.Context) func()
	now            func() time.Time
	retries        atomic.Int64
}

var _ run.Sessions = (*Agents)(nil)
var _ run.ActingSession = (*nativeSession)(nil)

func NewAgents(engine *container.Engine, prepared runtimeconfig.PreparedProfile, credentials CredentialResolver) (*Agents, error) {
	if engine == nil {
		return nil, ErrSession
	}
	return NewAgentsWithRuntime(func(ctx context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		return startRuntime(ctx, engine, spec)
	}, prepared, credentials)
}

// The supplied starter owns a fresh private home, confirms the actual running
// image and returns partial handles on error. It never borrows a host login home.
func NewAgentsWithRuntime(start RuntimeStarter, prepared runtimeconfig.PreparedProfile, credentials CredentialResolver) (*Agents, error) {
	p, err := ProfileFromPrepared(prepared)
	id, ok := prepared.Identity()
	if err != nil || !ok || start == nil || credentials == nil {
		return nil, errors.Join(ErrSession, err)
	}
	return &Agents{start: start, prepared: prepared, profile: p, identity: id, credentials: credentials, retry: defaultRetry, quiet: defaultQuiescence, hostTime: run.HostTime, now: time.Now}, nil
}

type ownedRuntime struct {
	RuntimeSession
	engine    *container.Engine
	home      string
	labels    map[string]string
	image     string
	closeOnce sync.Once
	closeErr  error
}

func (r *ownedRuntime) ImageID() string { return r.image }
func (r *ownedRuntime) Close() error {
	r.closeOnce.Do(func() {
		if r.RuntimeSession != nil {
			r.closeErr = r.RuntimeSession.Close()
		}
		ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
		defer cancel()
		v, err := r.engine.InspectVolume(ctx, r.home)
		r.closeErr = errors.Join(r.closeErr, err)
		if err != nil || !v.Exists {
			return
		}
		for k, want := range r.labels {
			if v.Labels[k] != want {
				r.closeErr = errors.Join(r.closeErr, fmt.Errorf("private home ownership unconfirmed"))
				return
			}
		}
		mounts, err := r.engine.VolumeMounts(ctx, r.home)
		r.closeErr = errors.Join(r.closeErr, err)
		if err != nil || len(mounts) != 0 {
			r.closeErr = errors.Join(r.closeErr, fmt.Errorf("private home still mounted or census failed"))
			return
		}
		r.closeErr = errors.Join(r.closeErr, r.engine.RemoveVolume(ctx, r.home))
		v, err = r.engine.InspectVolume(ctx, r.home)
		if err != nil || v.Exists {
			r.closeErr = errors.Join(r.closeErr, err, fmt.Errorf("private home closure unconfirmed"))
		}
	})
	return r.closeErr
}
func startRuntime(ctx context.Context, engine *container.Engine, spec container.ContainerSpec) (RuntimeSession, error) {
	r := &ownedRuntime{engine: engine, home: spec.Mounts[1].Volume, labels: spec.Labels}
	if _, err := engine.CreateVolumeExclusive(ctx, r.home, r.labels); err != nil {
		return r, err
	}
	peer, err := engine.Start(ctx, spec)
	if peer != nil {
		r.RuntimeSession = &enginePeer{Session: peer}
	}
	if err != nil {
		return r, err
	}
	r.image, err = engine.RunningImage(ctx, spec.Name)
	if err == nil && r.image != spec.Image {
		err = fmt.Errorf("native container image differs from prepared image")
	}
	return r, err
}

type enginePeer struct{ *container.Session }

func (*enginePeer) ImageID() string { return "" } // only ownedRuntime reports a confirmed image

type nativeSession struct {
	ready        bool
	agents       *Agents
	spec         run.SessionSpec
	profile      Profile
	document     harness.Document
	peer         RuntimeSession
	done         chan struct{}
	tr           *transcript
	login        login
	secrets      [][]byte
	env          []string
	session      string // the Claude Code session ID the adapter chose
	persisted    bool   // the conversation exists; later processes resume it
	homeSetup    bool   // agentd may hold the private files
	live         bool   // a provider process may be running
	final        string
	prompt       string
	acted        bool // the current Exchange observed a native execution
	served       int  // Run tool calls the current Exchange served
	running      map[string]string
	runningBytes int
	lastError    string // the error category the current attempt reported
	result       *Result
	mu           sync.Mutex
	abortMu      sync.Mutex
	cancel       context.CancelFunc
	closing      bool
	closed       bool
	closeErr     error
}

func (a *Agents) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	if ctx == nil || ctx.Err() != nil || a.TranscriptRoot != "" && !filepath.IsAbs(a.TranscriptRoot) || !spec.Prepared.Valid() || spec.Workspace == nil || strings.TrimSpace(spec.RunID) == "" {
		return nil, ErrSession
	}
	actual, ok := spec.Prepared.Topology().Manager(spec.Manager.Package)
	if !ok || !reflect.DeepEqual(actual, spec.Manager) || !validTools(spec.Tools) {
		return nil, ErrSession
	}
	doc, err := a.documentFor(spec)
	if err != nil {
		return nil, err
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	name := "markitect-claude-" + hex.EncodeToString(nonce[:])
	cs, err := sessionSpec(name, spec.Workspace.Location(), name+"-home", a.profile)
	if err != nil {
		return nil, err
	}
	session, err := newSessionID()
	if err != nil {
		return nil, err
	}
	t := &nativeSession{agents: a, spec: spec, profile: a.profile, document: doc, session: session, done: make(chan struct{})}
	// Resolve only after all model/workspace/harness admission. Never log resolver errors.
	credential, err := a.credentials(ctx, a.profile.CredentialRef)
	if err != nil {
		return nil, fmt.Errorf("%w: credential resolution failed", ErrSession)
	}
	if t.login, err = parseLogin(credential, a.now()); err != nil {
		return nil, err
	}
	t.secrets, t.env = t.login.secrets, providerEnvironment(t.login.token)
	home := frame.Initialize{Claude: &frame.ClaudeHome{Credentials: t.login.credentials, SystemPrompt: doc.Base, AppendSystemPrompt: doc.Developer}}
	if err = fitsSetup(home); err != nil {
		return nil, err
	}
	if a.TranscriptRoot != "" {
		if t.tr, err = openTranscript(filepath.Join(a.TranscriptRoot, name+".jsonl"), t.redact); err != nil {
			return nil, err
		}
	}
	t.peer, err = a.start(ctx, cs)
	if err != nil {
		return t, t.safe(err)
	}
	if t.peer == nil {
		return t, ErrSession
	}
	if t.peer.ImageID() != a.profile.ImageID {
		return t, fmt.Errorf("%w: actual native image not confirmed", ErrSession)
	}
	t.recordSetup()
	// A lost setup reply can still have installed files; always attempt cleanup.
	t.homeSetup = true
	if err = t.peer.Initialize(ctx, home); err != nil {
		return t, t.safe(err)
	}
	if err = t.checkLogin(ctx); err != nil {
		return t, err
	}
	t.ready = true
	return t, nil
}

// fitsSetup refuses native instructions and login that exceed one private
// setup frame before any container starts. Mandatory context is reported,
// never truncated.
func fitsSetup(home frame.Initialize) error {
	payload, err := frame.EncodeControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1 << 62, Initialize: &home})
	if err != nil || len(payload) > frame.MaxPayload {
		size := len(home.Claude.SystemPrompt) + len(home.Claude.AppendSystemPrompt)
		return fmt.Errorf("%w: the native instructions (%d bytes) and login exceed the private setup frame of %d bytes", ErrSession, size, frame.MaxPayload)
	}
	return nil
}

// recordSetup keeps the native instructions and login mode as evidence.
func (t *nativeSession) recordSetup() {
	data, _ := json.Marshal(struct {
		Session            string `json:"session"`
		LoginMode          string `json:"loginMode"`
		HarnessDigest      string `json:"harnessDigest"`
		SystemPrompt       string `json:"systemPrompt"`
		AppendSystemPrompt string `json:"appendSystemPrompt"`
	}{t.session, t.login.mode, t.document.Digest(), t.document.Base, t.document.Developer})
	t.tr.record("setup", data)
}

// checkLogin asks the CLI which login it would use, before any inference. It
// must be the selected subscription login: an API key, an API-key helper or
// another provider, for example from the image environment, is refused.
func (t *nativeSession) checkLogin(ctx context.Context) error {
	out, exit, err := t.runProcess(ctx, authStatusArgv(), privateHome, maxAuthStatusBytes)
	if err != nil {
		return err
	}
	method, err := authMethod(out)
	data, _ := json.Marshal(struct {
		AuthMethod string `json:"authMethod"`
		Exit       int    `json:"exit"`
	}{method, exit.Code})
	t.tr.record("login", data) // the status output itself names the account and stays out
	if err != nil {
		return fmt.Errorf("%w: claude auth status reported no readable authMethod", ErrSession)
	}
	if method != t.login.method {
		return fmt.Errorf("%w: Claude Code would authenticate with %q instead of the selected subscription login (%q); API keys, API-key helpers and other providers are refused", ErrSession, method, t.login.method)
	}
	return nil
}

// runProcess runs one short provider process with closed input and returns
// its bounded output and exit.
func (t *nativeSession) runProcess(ctx context.Context, argv []string, dir string, limit int64) ([]byte, frame.Exited, error) {
	if err := t.startProcess(ctx, argv, dir, nil); err != nil {
		return nil, frame.Exited{}, err
	}
	if err := t.io(ctx, t.peer.CloseInput); err != nil {
		return nil, frame.Exited{}, err
	}
	var out []byte
	err := t.io(ctx, func() error {
		var e error
		out, e = io.ReadAll(io.LimitReader(t.peer.Stdout(), limit+1))
		return e
	})
	if err != nil {
		return nil, frame.Exited{}, err
	}
	if int64(len(out)) > limit {
		return nil, frame.Exited{}, fmt.Errorf("%w: %s output exceeds %d bytes", ErrSession, filepath.Base(argv[0]), limit)
	}
	exit, err := t.collect(ctx)
	return out, exit, err
}

// startProcess starts one provider process. A lost start reply can still have
// started it, so it counts as live until its exit is collected.
func (t *nativeSession) startProcess(ctx context.Context, argv []string, dir string, tools []frame.ToolSpec) error {
	t.live = true
	data, _ := json.Marshal(struct {
		Argv []string `json:"argv"`
		Dir  string   `json:"dir"`
		Env  []string `json:"env"`
	}{argv, dir, environmentNames(t.env)})
	t.tr.record("start", data)
	return t.io(ctx, func() error {
		_, e := t.peer.StartProvider(ctx, frame.Start{Argv: argv, Dir: dir, Env: t.env, Tools: tools})
		return e
	})
}

// collect observes the exit of the provider process whose output ended. Stop
// also releases the container session for the next process.
func (t *nativeSession) collect(ctx context.Context) (frame.Exited, error) {
	exit, err := t.peer.Stop(ctx, stopGrace.Milliseconds())
	if err != nil {
		return exit, t.safe(err)
	}
	t.live = false
	if data, e := json.Marshal(exit); e == nil {
		t.tr.record("exit", data)
	}
	return exit, nil
}

// Exchange serializes one native message; direct Run tool callbacks drain
// before return. Close first cancels the active callback/message, then joins
// its closure. The judgment is a Run tool call, not part of the response.
func (t *nativeSession) Exchange(ctx context.Context, message run.Message) (err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ready || t.closed || ctx == nil || len(message.Events) == 0 && len(message.Feedback) == 0 {
		return ErrSession
	}
	active, cancel := context.WithCancel(ctx)
	defer cancel()
	t.abortMu.Lock()
	if t.closing {
		t.abortMu.Unlock()
		return context.Canceled
	}
	t.cancel = cancel
	t.abortMu.Unlock()
	defer func() {
		t.abortMu.Lock()
		t.cancel = nil
		t.abortMu.Unlock()
		if err != nil {
			t.closed = true
			t.closeErr = t.finish(active)
			err = errors.Join(err, t.closeErr)
		}
	}()
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	t.prompt, t.final, t.acted, t.served = string(payload), "", false, 0
	return t.runTurn(active)
}

// Acted reports whether the last Exchange observed a native execution: any
// tool use that completed, a Run tool call included.
func (t *nativeSession) Acted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.acted
}
func (t *nativeSession) Close(ctx context.Context) error {
	t.abortMu.Lock()
	t.closing = true
	cancel := t.cancel
	t.abortMu.Unlock()
	if cancel != nil {
		cancel()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return t.closeErr
	}
	t.closed = true
	t.closeErr = t.finish(ctx)
	return t.closeErr
}
func (t *nativeSession) finish(ctx context.Context) error {
	close(t.done)
	var err error
	if t.peer != nil {
		observed := !t.live
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
		defer cancel()
		if t.live {
			_, e := t.peer.Stop(cleanup, stopGrace.Milliseconds())
			observed = e == nil
			err = errors.Join(err, t.safe(e))
		}
		if t.homeSetup && observed {
			err = errors.Join(err, t.clearHome(cleanup))
		}
		err = errors.Join(err, t.safe(t.peer.Close()))
	}
	return errors.Join(err, t.tr.close())
}

// clearHome removes the installed login and instructions after observed
// provider exit.
func (t *nativeSession) clearHome(ctx context.Context) error {
	if err := t.peer.ClearPrivateHome(ctx); err != nil {
		return t.safe(err)
	}
	t.homeSetup = false
	return nil
}

// io cancels a blocked transport write by closing the owned peer, then waits for
// that write to drain. Runtime peers must release writes when Close returns.
func (t *nativeSession) io(ctx context.Context, f func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- f() }()
	select {
	case err := <-done:
		return t.safe(err)
	case <-ctx.Done():
		closeErr := t.peer.Close()
		writeErr := <-done
		return errors.Join(ctx.Err(), t.safe(closeErr), t.safe(writeErr))
	}
}

func (t *nativeSession) answer(ctx context.Context, call frame.ToolCall) error {
	if raw, err := frame.EncodeToolCall(call); err == nil {
		t.tr.record("tool-call", raw)
	}
	result, err := dispatch(ctx, t.spec.Tools, call.Name, t.redact(call.Arguments))
	if errors.Is(err, errUnknownTool) {
		return ErrProtocol
	}
	content := toolResult(string(t.redact([]byte(result))), err != nil)
	reply := frame.ToolReply{ID: call.ID, Result: content}
	if raw, e := frame.EncodeToolReply(reply); e == nil {
		t.tr.record("tool-reply", raw)
	}
	return t.io(ctx, func() error { return t.peer.Reply(reply) })
}

// observe reads one output line of the current response. A provider error
// result returns a providerError; a transient one ends the attempt only once
// the process exits.
func (t *nativeSession) observe(m Message) error {
	if m.Session() != "" && m.Session() != t.session {
		return fmt.Errorf("%w: the provider reported another session than the adapter's", ErrProtocol)
	}
	switch m.Kind() {
	case SystemInitMessage:
		init := m.Init()
		t.persisted = true
		for _, server := range init.MCPServers {
			if server.Name == "markitect" {
				if !oneOf(server.Status, "connected", "pending") {
					return fmt.Errorf("%w: the markitect MCP server is %q in Claude Code, so the Run tools are unavailable", ErrSession, server.Status)
				}
				return nil
			}
		}
		return fmt.Errorf("%w: Claude Code started without the markitect MCP server, so the Run tools are unavailable", ErrSession)
	case APIRetryMessage:
		if kind := m.Retry().Error; kind != "" {
			t.lastError = kind
		}
	case AssistantMessage, UserMessage:
		if m.Parent() != "" {
			return fmt.Errorf("%w: the provider reported native subagent work, and native delegation is disabled", ErrProtocol)
		}
		if kind := m.ErrorKind(); kind != "" {
			t.lastError = kind
		}
		for _, b := range m.Blocks() {
			switch {
			case m.Kind() == UserMessage && b.Type == "tool_result", strings.HasSuffix(b.Type, "_tool_result"):
				t.complete(b.ToolUseID)
			case m.Kind() == AssistantMessage && strings.HasSuffix(b.Type, "tool_use"):
				if err := t.started(b.ID, b.Name); err != nil {
					return err
				}
			case m.Kind() == AssistantMessage && b.Type == "text":
				t.final = b.Text
			}
		}
	case ResultMessage:
		r := m.Result()
		t.result = &r
		if r.IsError {
			message := r.Text
			if message == "" {
				message = strings.Join(r.Errors, "; ")
			}
			return &providerError{kind: t.lastError, message: string(t.redact([]byte(message)))}
		}
		if r.Text != "" {
			t.final = r.Text
		}
	case ControlRequestMessage:
		return fmt.Errorf("%w: the provider sent a control request, which this adapter does not answer", ErrProtocolUnsupported)
	}
	return nil
}

// started records a tool use that may now run, also a Run tool call on its
// way to the host. The set is bounded by the protocol byte budget.
func (t *nativeSession) started(id, name string) error {
	if id == "" {
		return ErrProtocol
	}
	if _, ok := t.running[id]; ok {
		return nil
	}
	if len(id) > maxProtocolMessage-t.runningBytes {
		return ErrProtocolLimit
	}
	if t.running == nil {
		t.running = make(map[string]string)
	}
	t.running[id] = name
	t.runningBytes += len(id)
	return nil
}

// complete records a finished tool use: the response acted, whatever the
// outcome of the tool.
func (t *nativeSession) complete(id string) {
	if _, ok := t.running[id]; ok {
		delete(t.running, id)
		t.runningBytes -= len(id)
	}
	t.acted = true
}

// accept records one provider message and reports a failed stream.
func (t *nativeSession) accept(e event) (Message, error) {
	if len(e.msg.raw) > 0 {
		t.tr.record("receive", e.msg.raw)
	}
	return e.msg, t.safe(e.err)
}

type event struct {
	msg Message
	err error
}

// readEvents decodes provider output into messages. A malformed or oversized
// line ends the stream with an error on its last message; the end of output
// closes the channel.
func readEvents(r io.Reader, codec *Codec, done <-chan struct{}) <-chan event {
	out := make(chan event, 64)
	go func() {
		defer close(out)
		send := func(e event) bool {
			select {
			case out <- e:
				return true
			case <-done:
				return false
			}
		}
		buf := make([]byte, 64<<10)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				messages, err := codec.Feed(buf[:n])
				for i, m := range messages {
					e := event{msg: m}
					if err != nil && i == len(messages)-1 {
						e.err = err
					}
					if !send(e) {
						return
					}
				}
				if err != nil {
					if len(messages) == 0 {
						send(event{err: err})
					}
					return
				}
			}
			if readErr != nil {
				if err := codec.End(); err != nil {
					send(event{msg: Message{raw: codec.Pending(), kind: RejectedMessage}, err: err})
				} else if readErr != io.EOF {
					send(event{err: readErr})
				}
				return
			}
		}
	}()
	return out
}

// redact replaces every known secret and its JSON-escaped form.
func (t *nativeSession) redact(data []byte) []byte {
	data = bytes.Clone(data)
	marker := []byte("[REDACTED]")
	for _, secret := range t.secrets {
		if len(secret) == 0 {
			continue
		}
		data = bytes.ReplaceAll(data, secret, marker)
		encoded, _ := json.Marshal(string(secret))
		data = bytes.ReplaceAll(data, encoded[1:len(encoded)-1], marker)
	}
	return data
}

// safe keeps err's identity unless its text carries a known secret.
func (t *nativeSession) safe(err error) error {
	if err == nil {
		return nil
	}
	text := []byte(err.Error())
	if clean := t.redact(text); !bytes.Equal(clean, text) {
		return fmt.Errorf("%w: %s", ErrSession, clean)
	}
	return err
}

// Observation is supplied only by an actually constructed native session.
func (t *nativeSession) Observation() run.SessionObservation {
	if !t.ready {
		return run.SessionObservation{}
	}
	return run.SessionObservation{ProfileIdentity: t.agents.identity, ImageID: t.peer.ImageID(), HarnessDigest: t.document.Digest()}
}

// quiescencePolicy bounds the census that follows a provider's exit: agentd
// has killed the provider's process group, and a process that left the group
// may still be running.
type quiescencePolicy struct {
	attempts int
	interval time.Duration
}

var defaultQuiescence = quiescencePolicy{attempts: 20, interval: 100 * time.Millisecond}

// quiescent requires that no native process outlives the response: a
// detached process could still write to the working copy after Exchange
// returns. Processes that ended and await reaping do not count.
func (t *nativeSession) quiescent(ctx context.Context) error {
	var remaining []frame.Process
	for attempt := 0; attempt < max(1, t.agents.quiet.attempts); attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(t.agents.quiet.interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		processes, err := t.peer.Census(ctx)
		if err != nil {
			return fmt.Errorf("%w: the process census after the provider's exit failed: %v", ErrSession, t.safe(err))
		}
		remaining = remaining[:0]
		for _, p := range processes {
			if p.State != "Z" {
				remaining = append(remaining, p)
			}
		}
		if len(remaining) == 0 {
			return nil
		}
	}
	commands := make([]string, 0, min(len(remaining), 5))
	for _, p := range remaining[:min(len(remaining), 5)] {
		commands = append(commands, string(t.redact([]byte(p.Command))))
	}
	if data, err := json.Marshal(remaining); err == nil {
		t.tr.record("census", data)
	}
	return fmt.Errorf("%w: %d native processes remain after the provider exited (%s); detached background work is not supported", ErrSession, len(remaining), strings.Join(commands, ", "))
}

// newSessionID returns a random version 4 UUID, the form --session-id takes.
func newSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32], nil
}

func (a *Agents) documentFor(spec run.SessionSpec) (harness.Document, error) {
	findings := make([]harness.Finding, len(spec.Findings))
	for i, f := range spec.Findings {
		findings[i] = harness.Finding{Manager: f.Manager, Definition: f.Definition, Text: f.Text}
	}
	origin := harness.Origin{Commit: spec.Origin.Input.Commit, Tree: spec.Origin.Input.Tree, Verified: spec.Origin.Verified}
	doc, err := harness.Build(spec.Prepared, spec.Manager, string(spec.Mandate), findings, origin)
	if err != nil {
		return harness.Document{}, err
	}
	doc.Developer += "\nSelected provider profile instructions (subject to canonical rules):\n" + a.profile.Session.BaseInstructions + "\n" + a.profile.Session.DeveloperInstructions + "\n"
	for _, location := range a.prepared.Observation().Instructions {
		found := false
		for _, source := range a.prepared.Input().Instructions() {
			if source.Location == location {
				data := source.File.Bytes()
				if !utf8.Valid(data) {
					return harness.Document{}, ErrSession
				}
				doc.Developer += fmt.Sprintf("\nCaptured runtime instruction %s (%s, %d bytes):\n%s\n", location.Path, source.File.ContentID(), len(data), data)
				found = true
				break
			}
		}
		if !found {
			return harness.Document{}, fmt.Errorf("captured runtime instruction unavailable")
		}
	}
	return doc, nil
}
