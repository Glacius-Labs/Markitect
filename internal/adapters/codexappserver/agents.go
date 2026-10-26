package codexappserver

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
	retryWaits     []time.Duration
	hostTime       func(context.Context) func()
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
	return &Agents{start: start, prepared: prepared, profile: p, identity: id, credentials: credentials, retryWaits: retryWaits, hostTime: run.HostTime}, nil
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
	ready                             bool
	agents                            *Agents
	spec                              run.SessionSpec
	profile                           Profile
	document                          harness.Document
	peer                              RuntimeSession
	enc                               *Codec
	events                            <-chan event
	done                              chan struct{}
	tr                                *transcript
	secrets                           [][]byte
	loginInstalled, providerAttempted bool
	thread                            ThreadID
	turnID                            TurnID
	model, final, prompt              string
	acted                             bool // the current Exchange observed a native execution
	served                            int  // Run tool calls the current Exchange served
	running                           map[string]string
	runningBytes                      int
	nextRequest                       int64
	children                          map[ThreadID]bool
	childIDBytes                      int
	mu                                sync.Mutex
	abortMu                           sync.Mutex
	cancel                            context.CancelFunc
	closing, closed                   bool
	closeErr                          error
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
	name := "markitect-native-" + hex.EncodeToString(nonce[:])
	cs, err := sessionSpec(name, spec.Workspace.Location(), name+"-home", a.profile)
	if err != nil {
		return nil, err
	}
	t := &nativeSession{agents: a, spec: spec, profile: a.profile, document: doc, done: make(chan struct{})}
	// Resolve only after all model/workspace/harness admission. Never log resolver errors.
	credential, err := a.credentials(ctx, a.profile.CredentialRef)
	if err != nil {
		return nil, fmt.Errorf("%w: credential resolution failed", ErrSession)
	}
	env, err := t.credential(ctx, credential)
	if err != nil {
		return nil, err
	}
	if t.enc, err = NewCodec(int(a.profile.MaxWireBytes)); err != nil {
		return nil, err
	}
	dec, err := NewCodec(int(a.profile.MaxWireBytes))
	if err != nil {
		return nil, err
	}
	if a.TranscriptRoot != "" {
		if !filepath.IsAbs(a.TranscriptRoot) {
			return nil, ErrSession
		}
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
	if err = t.installCredential(ctx, credential); err != nil {
		return t, err
	}
	t.providerAttempted = true
	if err = t.io(ctx, func() error {
		_, e := t.peer.StartProvider(ctx, frame.Start{Argv: providerArgv(), Dir: "/work", Env: env, Tools: toolSpecs()})
		return e
	}); err != nil {
		return t, err
	}
	t.events = readEvents(t.peer.Stdout(), dec, t.done)
	if err = t.handshake(ctx); err != nil {
		return t, err
	}
	t.nextRequest = 2
	t.ready = true
	return t, nil
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
	t.prompt, t.final, t.turnID, t.acted, t.served = string(payload), "", "", false, 0
	if err = t.runTurn(active); err != nil {
		return err
	}
	return t.drainTerminals(active)
}

// Acted reports whether the last Exchange observed a native execution, any
// completed item but text, also on a native subagent thread.
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
		observed := !t.providerAttempted
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
		defer cancel()
		if t.providerAttempted {
			_, e := t.peer.Stop(cleanup, stopGrace.Milliseconds())
			observed = e == nil
			err = errors.Join(err, t.safe(e))
		}
		if t.loginInstalled && observed {
			err = errors.Join(err, t.removeCredential(cleanup))
		}
		err = errors.Join(err, t.safe(t.peer.Close()))
	}
	return errors.Join(err, t.tr.close())
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
func (t *nativeSession) startThread(ctx context.Context) error {
	id := IntegerRequestID(2)
	raw, err := t.enc.EncodePersistentThread(id, ThreadStartParams{Model: t.profile.Session.Model, Cwd: "/work", ApprovalPolicy: "never", Sandbox: "danger-full-access", BaseInstructions: t.document.Base, DeveloperInstructions: t.document.Developer}, "")
	if err != nil {
		return err
	}
	if err = t.send(ctx, raw); err != nil {
		return err
	}
	r, err := t.response(ctx, id, ThreadStart)
	if err != nil {
		return err
	}
	if t.thread != "" && t.thread != r.Thread {
		return ErrProtocol
	}
	t.thread = r.Thread
	return nil
}

func (t *nativeSession) answer(ctx context.Context, call frame.ToolCall) error {
	var meta struct {
		Thread string `json:"threadId"`
	}
	if json.Unmarshal(call.Meta, &meta) != nil || meta.Thread != string(t.thread) {
		return ErrProtocol
	}
	if raw, err := frame.EncodeToolCall(call); err == nil {
		t.tr.record("tool-call", raw)
	}
	result, err := dispatch(ctx, t.spec.Tools, call.Name, t.redact(call.Arguments))
	content := toolResult(string(t.redact([]byte(result))), err != nil)
	if errors.Is(err, errUnknownTool) {
		return ErrProtocol
	}
	reply := frame.ToolReply{ID: call.ID, Result: content}
	if raw, e := frame.EncodeToolReply(reply); e == nil {
		t.tr.record("tool-reply", raw)
	}
	return t.io(ctx, func() error { return t.peer.Reply(reply) })
}

func (t *nativeSession) handshake(ctx context.Context) error {
	id := IntegerRequestID(1)
	raw, err := t.enc.EncodeInitialize(id, ClientInfo{Name: "markitect", Version: "1"})
	if err != nil {
		return err
	}
	if err = t.send(ctx, raw); err != nil {
		return err
	}
	if _, err = t.response(ctx, id, Initialize); err != nil {
		return err
	}
	if raw, err = t.enc.EncodeInitialized(); err != nil {
		return err
	}
	return t.send(ctx, raw)
}

func (t *nativeSession) observe(m Message, request RequestID, responseSeen *bool) (TurnStatus, error) {
	switch m.kind {
	case ResponseMessage, ErrorMessage:
		if got, _ := m.ID(); got != request || *responseSeen {
			return "", ErrProtocol
		}
		if m.kind == ErrorMessage {
			return "", t.rpcError(m)
		}
		r, err := m.ReadResponse(request, TurnStart)
		if err != nil {
			return "", err
		}
		if t.turnID != "" && t.turnID != r.Turn.Turn {
			return "", ErrProtocol
		}
		t.turnID = r.Turn.Turn
		*responseSeen = true
		if r.Turn.Status != TurnInProgress {
			return r.Turn.Status, nil
		}
	case TurnStartedMessage, TurnCompletedMessage:
		o := m.Turn()
		if o.Thread != t.thread {
			if t.children[o.Thread] {
				return "", nil
			}
			return "", ErrProtocol
		}
		if t.turnID != "" && t.turnID != o.Turn {
			return "", ErrProtocol
		}
		t.turnID = o.Turn
		if m.kind == TurnCompletedMessage {
			return o.Status, nil
		}
	case ThreadStartedMessage:
		if m.Thread() != t.thread && !t.children[m.Thread()] {
			return "", ErrProtocol
		}
	case UnknownNotification:
		return "", t.notification(m)
	default:
		return "", fmt.Errorf("%w: %s", ErrProtocolUnsupported, m.kind)
	}
	return "", nil
}

// notification reads the item and usage notifications a turn reports; any
// other notification is ignored, except a provider error that will not retry.
func (t *nativeSession) notification(m Message) error {
	switch m.method {
	case "error":
		var e struct {
			Thread ThreadID `json:"threadId"`
			Error  struct {
				Message string          `json:"message"`
				Info    json.RawMessage `json:"codexErrorInfo"`
			} `json:"error"`
			WillRetry bool `json:"willRetry"`
		}
		if err := json.Unmarshal(m.body, &e); err != nil {
			return ErrProtocol
		}
		if e.Thread != "" && e.Thread != t.thread {
			if t.children[e.Thread] {
				return nil
			}
			return ErrProtocol
		}
		if !e.WillRetry {
			return &providerError{kind: errorKind(e.Error.Info), message: string(t.redact([]byte(e.Error.Message)))}
		}
	case "item/started", "item/completed":
		var n struct {
			Thread string `json:"threadId"`
			Turn   string `json:"turnId"`
			Item   struct {
				ID       string   `json:"id"`
				Type     string   `json:"type"`
				Text     string   `json:"text"`
				Phase    string   `json:"phase"`
				Command  string   `json:"command"`
				Status   string   `json:"status"`
				ExitCode *int     `json:"exitCode"`
				Output   string   `json:"aggregatedOutput"`
				Kind     string   `json:"kind"`
				Child    ThreadID `json:"agentThreadId"`
			} `json:"item"`
		}
		if json.Unmarshal(m.body, &n) != nil {
			return ErrProtocol
		}
		main := n.Thread == string(t.thread)
		if main {
			if t.turnID != "" && n.Turn != string(t.turnID) {
				return ErrProtocol
			}
		} else if !t.children[ThreadID(n.Thread)] {
			return ErrProtocol
		}
		if n.Item.Type == "subAgentActivity" && n.Item.Kind == "started" {
			if n.Turn == "" || main && t.turnID == "" || n.Item.Child == "" || n.Item.Child == t.thread || n.Item.Child == ThreadID(n.Thread) {
				return ErrProtocol
			}
			if !t.children[n.Item.Child] {
				// Bound retained native identities by the existing protocol byte
				// budget; no unbounded notification stream can grow this set.
				if len(n.Item.Child) > maxProtocolMessage-t.childIDBytes {
					return ErrProtocolLimit
				}
				if t.children == nil {
					t.children = make(map[ThreadID]bool)
				}
				t.children[n.Item.Child] = true
				t.childIDBytes += len(n.Item.Child)
			}
		}
		// A native item that started and has not completed may be running. Text
		// runs nothing, and the host counts the Run tool calls it serves itself.
		key := n.Thread + "/" + n.Item.ID
		_, started := t.running[key]
		switch {
		case n.Item.Type == "userMessage" || n.Item.Type == "agentMessage" || n.Item.Type == "reasoning" || n.Item.Type == "mcpToolCall":
		case m.method == "item/started" && !started:
			if len(key) > maxProtocolMessage-t.runningBytes {
				return ErrProtocolLimit
			}
			if t.running == nil {
				t.running = make(map[string]string)
			}
			t.running[key] = n.Item.Type
			t.runningBytes += len(key)
		case m.method == "item/completed" && started:
			delete(t.running, key)
			t.runningBytes -= len(key)
		}
		if m.method != "item/completed" {
			return nil
		}
		// Every item but text ran something: a command, a file change, a tool
		// call, a web search or a subagent. Its outcome does not matter.
		switch n.Item.Type {
		case "userMessage", "agentMessage", "reasoning":
		default:
			t.acted = true
		}
		if !main {
			return nil
		}
		if n.Item.Type == "agentMessage" && n.Item.Phase == "final_answer" {
			t.final = n.Item.Text
		}
	}
	return nil
}

// send writes one NDJSON request. A failed write may have been partial and is
// never repeated.
func (t *nativeSession) send(ctx context.Context, raw []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.tr.record("send", raw)
	return t.safe(t.io(ctx, func() error { return t.peer.WriteInput(raw) }))
}

// response waits for the response to request id, skipping notifications that
// carry no obligation during the handshake.
func (t *nativeSession) response(ctx context.Context, id RequestID, method RequestMethod) (Response, error) {
	for {
		var e event
		var open bool
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case e, open = <-t.events:
		}
		m, err := t.accept(e, open)
		if err != nil {
			return Response{}, err
		}
		switch m.kind {
		case ResponseMessage:
			r, err := m.ReadResponse(id, method)
			if err == nil && (method == ThreadStart || method == ThreadResume) {
				var body struct {
					Model string `json:"model"`
				}
				_ = json.Unmarshal(m.body, &body)
				t.model = body.Model
			}
			return r, err
		case ErrorMessage:
			return Response{}, t.rpcError(m)
		case ThreadStartedMessage:
			if t.thread != "" && t.thread != m.Thread() {
				return Response{}, ErrProtocol
			}
			t.thread = m.Thread()
		case UnknownNotification:
			if err := t.notification(m); err != nil {
				return Response{}, err
			}
		default:
			return Response{}, fmt.Errorf("%w: %s", ErrProtocolUnsupported, m.kind)
		}
	}
}

func (t *nativeSession) rpcError(m Message) error {
	code, text, err := m.RPCError()
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: provider refused the request (%d): %s", ErrProtocol, code, t.redact([]byte(text)))
}

// accept records one provider message and reports a closed or failed stream.
func (t *nativeSession) accept(e event, open bool) (Message, error) {
	if !open {
		return Message{}, fmt.Errorf("%w: provider output ended before the turn finished", ErrProtocolIncomplete)
	}
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
// line ends the stream with an error on its last message; a clean end of
// output closes the channel.
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
		buf := make([]byte, 8192)
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
func (t *nativeSession) drainTerminals(ctx context.Context) error {
	list := func() (Response, error) {
		t.nextRequest++
		id := IntegerRequestID(t.nextRequest)
		raw, err := t.enc.EncodeBackgroundList(id, t.thread)
		if err != nil {
			return Response{}, err
		}
		if err = t.send(ctx, raw); err != nil {
			return Response{}, err
		}
		return t.response(ctx, id, BackgroundList)
	}
	first, err := list()
	if err != nil {
		return fmt.Errorf("%w: native background terminals unavailable: %v", ErrSession, err)
	}
	for _, terminal := range first.Terminals {
		t.nextRequest++
		id := IntegerRequestID(t.nextRequest)
		raw, err := t.enc.EncodeBackgroundTerminate(id, t.thread, terminal.ProcessID)
		if err != nil {
			return err
		}
		if err = t.send(ctx, raw); err != nil {
			return err
		}
		if _, err = t.response(ctx, id, BackgroundTerminate); err != nil {
			return err
		}
	}
	final, err := list()
	if err != nil {
		return err
	}
	if len(final.Terminals) != 0 {
		return fmt.Errorf("%w: native background terminals remain after termination", ErrSession)
	}
	return nil
}

func (a *Agents) documentFor(spec run.SessionSpec) (harness.Document, error) {
	findings := make([]harness.Finding, len(spec.Findings))
	for i, f := range spec.Findings {
		findings[i] = harness.Finding{Manager: f.Manager, Text: f.Text}
	}
	doc, err := harness.Build(spec.Prepared, spec.Manager, string(spec.Mandate), findings)
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
