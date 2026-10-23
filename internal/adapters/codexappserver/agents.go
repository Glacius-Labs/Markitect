package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const (
	stopGrace   = 10 * time.Second
	stopTimeout = 20 * time.Second
)

// Agents supports single turns and persistent assignments. Each assignment
// retains its own container and provider thread until Close.
type Agents struct {
	start       RuntimeStarter
	profiles    map[string]Profile
	credentials CredentialResolver
	stopGrace   time.Duration
	stopTimeout time.Duration
}

var _ run.AssignmentAgents = (*Agents)(nil)

// NewAgents runs turns in containers of engine. profiles maps the Runtime profile
// names a Turn refers to.
func NewAgents(engine *container.Engine, profiles map[string]Profile, credentials CredentialResolver) (*Agents, error) {
	if engine == nil {
		return nil, ErrSession
	}
	return NewAgentsWithRuntime(func(ctx context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		s, err := engine.Start(ctx, spec)
		if err != nil {
			return nil, err
		}
		return s, nil
	}, profiles, credentials)
}

// NewAgentsWithRuntime permits an Orchestrator-supplied runtime peer. Production peers
// must preserve Engine.Start and Session's exit semantics.
func NewAgentsWithRuntime(start RuntimeStarter, profiles map[string]Profile, credentials CredentialResolver) (*Agents, error) {
	if start == nil || credentials == nil {
		return nil, ErrSession
	}
	copied := make(map[string]Profile, len(profiles))
	for name, p := range profiles {
		if name == "" || validateProfile(p) != nil {
			return nil, ErrSession
		}
		copied[name] = p
	}
	return &Agents{start: start, profiles: copied, credentials: credentials, stopGrace: stopGrace, stopTimeout: stopTimeout}, nil
}

// turnRun is the state of one Run call.
type turnRun struct {
	agents  *Agents
	turn    run.Turn
	profile Profile
	handler run.ToolHandler
	name    string
	offered map[string]bool
	tr      *transcript

	secrets           [][]byte
	loginInstalled    bool
	providerAttempted bool
	attempted         bool
	peer              RuntimeSession
	enc               *Codec
	events            <-chan event
	done              chan struct{}

	thread      ThreadID
	turnID      TurnID
	model       string
	final       string
	commands    []run.CommandObservation
	usage       run.Usage
	nextRequest int64
	// Native children share this provider runtime, not the outer Manager
	// assignment. Only an observed subAgentActivity establishes their identity.
	children     map[ThreadID]bool
	childIDBytes int
}

// Run runs one turn. It always stops the container before it returns; Stopped
// is true when the provider's exit and the end of the container were observed,
// or when no container was ever started. Known secrets are removed from tool
// call arguments before the handler sees them.
func (a *Agents) Run(ctx context.Context, turn run.Turn, handler run.ToolHandler) (res run.TurnResult, err error) {
	res.Started = time.Now().UTC()
	defer func() { res.Ended = time.Now().UTC() }()
	profile, offered, err := a.admit(turn, handler)
	if err != nil {
		res.Stopped, res.Reason = true, "turn refused before any container was started"
		return res, err
	}
	t := &turnRun{agents: a, turn: turn, profile: profile, handler: handler, name: "markitect-" + turn.ID, offered: offered, done: make(chan struct{})}
	if t.tr, err = openTranscript(turn.Transcript, t.redact); err != nil {
		res.Stopped, res.Reason = true, "turn refused before any container was started"
		return res, err
	}
	runCtx := ctx
	if turn.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, turn.Timeout)
		defer cancel()
	}
	err = t.exchange(runCtx, &res)
	if err != nil {
		switch {
		case ctx.Err() != nil:
			res.Reason = "cancelled"
		case runCtx.Err() != nil:
			res.Reason = "timeout"
			err = fmt.Errorf("%w: turn exceeded its %s timeout: %w", ErrSession, turn.Timeout, err)
		case res.Reason == "":
			res.Reason = "failed"
		}
	}
	res.Conversation, res.Final, res.Usage = string(t.thread), string(t.redact([]byte(t.final))), t.usage
	res.Commands = t.commands
	if res.Usage.ProviderTurns > 0 {
		res.Usage.Model = t.model
	}
	return res, t.finish(ctx, &res, err)
}

// OpenAssignment starts one container and one Codex App Server process for the
// complete assignment, initializing only the connection. The first Message
// starts the provider thread; subsequent Messages use that same thread.
func (a *Agents) OpenAssignment(ctx context.Context, spec run.AssignmentSpec, handler run.ToolHandler) (run.AssignmentSession, error) {
	turn := run.Turn{ID: spec.ID, Kind: run.ExecuteTurn, Profile: spec.Profile, Workspace: spec.Workspace,
		Home: spec.Home, Tools: spec.Tools, Transcript: spec.Transcript}
	profile, offered, err := a.admit(turn, handler)
	if err != nil {
		return nil, err
	}
	t := &turnRun{agents: a, turn: turn, profile: profile, handler: handler, name: "markitect-" + spec.ID,
		offered: offered, done: make(chan struct{})}
	if t.tr, err = openTranscript(spec.Transcript, t.redact); err != nil {
		return nil, err
	}
	if err = t.open(ctx); err != nil {
		res := run.TurnResult{Reason: "assignment start failed"}
		return nil, t.finish(ctx, &res, err)
	}
	return &assignmentSession{turn: t}, nil
}

type assignmentSession struct {
	mu           sync.Mutex
	abortMu      sync.Mutex
	activeCancel context.CancelFunc
	closing      bool
	turn         *turnRun
	closed       bool
	terminal     run.TurnResult
	terminalErr  error
}

func (s *assignmentSession) Message(ctx context.Context, text string, timeout time.Duration) (res run.TurnResult, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || strings.TrimSpace(text) == "" || timeout < 0 {
		return res, ErrSession
	}
	res.Started = time.Now().UTC()
	defer func() { res.Ended = time.Now().UTC() }()
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.abortMu.Lock()
	if s.closing {
		s.abortMu.Unlock()
		return res, context.Canceled
	}
	s.activeCancel = cancel
	s.abortMu.Unlock()
	defer func() {
		s.abortMu.Lock()
		s.activeCancel = nil
		s.abortMu.Unlock()
	}()
	t := s.turn
	t.turn.Prompt, t.turnID, t.final = text, "", ""
	t.commands, t.usage = nil, run.Usage{}
	if timeout > 0 {
		var timeoutCancel context.CancelFunc
		runCtx, timeoutCancel = context.WithTimeout(runCtx, timeout)
		defer timeoutCancel()
	}
	err = t.runTurn(runCtx, &res)
	res.Conversation, res.Final, res.Commands, res.Usage = string(t.thread), string(t.redact([]byte(t.final))), t.commands, t.usage
	if res.Usage.ProviderTurns > 0 {
		res.Usage.Model = t.model
	}
	if errors.Is(err, run.ErrContextLimit) {
		res.Reason = "message exceeds provider channel"
		return res, err
	}
	if err != nil {
		s.closed = true
		if runCtx.Err() != nil {
			res.Reason = runCtx.Err().Error()
		}
		err = t.finish(ctx, &res, err)
		s.terminal, s.terminalErr = res, err
	}
	return res, err
}

func (s *assignmentSession) Close(ctx context.Context) (res run.TurnResult, err error) {
	s.abortMu.Lock()
	s.closing = true
	cancel := s.activeCancel
	s.abortMu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.terminal, s.terminalErr
	}
	s.closed = true
	res.Reason = "assignment closed"
	err = s.turn.finish(ctx, &res, nil)
	res.Ended = time.Now().UTC()
	s.terminal, s.terminalErr = res, err
	return res, err
}

// admit checks the turn against the Runtime profile before anything runs.
func (a *Agents) admit(turn run.Turn, handler run.ToolHandler) (Profile, map[string]bool, error) {
	profile, ok := a.profiles[turn.Profile]
	if !ok || handler == nil || !volumeName.MatchString(turn.ID) || len(turn.ID) > 100 || !volumeName.MatchString(turn.Workspace) || !volumeName.MatchString(turn.Home) || turn.Workspace == turn.Home || turn.Timeout < 0 {
		return Profile{}, nil, ErrSession
	}
	switch turn.Kind {
	case run.ExecuteTurn:
	case run.VerifyTurn:
		// A verify turn may continue only its own reviewer's conversation of the
		// same review Pass; the Orchestrator gives every review Pass fresh homes.
		if !turn.ReadOnly || writable(turn.Mounts) {
			return Profile{}, nil, ErrSession
		}
	default:
		return Profile{}, nil, ErrSession
	}
	if !validMounts(turn.Mounts) {
		return Profile{}, nil, ErrSession
	}
	offered := map[string]bool{}
	for _, tool := range turn.Tools {
		if tool.Name == "" || offered[tool.Name] || strings.TrimSpace(tool.Description) == "" {
			return Profile{}, nil, ErrSession
		}
		if _, err := protocolObject(tool.InputSchema); err != nil {
			return Profile{}, nil, ErrSession
		}
		offered[tool.Name] = true
	}
	return profile, offered, nil
}

// exchange runs the container, the handshake and the turn until the provider
// reports its end. Cleanup is finish's job.
func (t *turnRun) exchange(ctx context.Context, res *run.TurnResult) error {
	if err := t.open(ctx); err != nil {
		return err
	}
	return t.runTurn(ctx, res)
}

func (t *turnRun) open(ctx context.Context) error {
	credential, err := t.agents.credentials(ctx, t.profile.CredentialRef)
	if err != nil {
		return fmt.Errorf("%w: credential resolution failed", ErrSession)
	}
	env, err := t.credential(ctx, credential)
	if err != nil {
		return err
	}
	limit := int(t.profile.MaxWireBytes)
	if t.enc, err = NewCodec(limit); err != nil {
		return err
	}
	dec, err := NewCodec(limit)
	if err != nil {
		return err
	}
	// Even a failed container creation can leave runtime effects.
	t.attempted = true
	peer, err := t.agents.start(ctx, turnSpec(t.name, t.turn, t.profile))
	if err != nil {
		return t.safe(err)
	}
	if peer == nil {
		return ErrSession
	}
	t.peer = peer
	if err := t.installCredential(ctx, credential); err != nil {
		return err
	}
	specs := make([]frame.ToolSpec, len(t.turn.Tools))
	for i, tool := range t.turn.Tools {
		specs[i] = frame.ToolSpec{Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema}
	}
	start := frame.Start{Argv: providerArgv(credential.Mode), Dir: "/work", Env: env, Tools: specs}
	t.providerAttempted = true // a lost start reply may still have started it
	if err = within(ctx, func() error { _, startErr := peer.StartProvider(ctx, start); return startErr }); err != nil {
		return t.safe(err)
	}
	t.events = readEvents(peer.Stdout(), dec, t.done)
	if err = t.handshake(ctx); err != nil {
		return err
	}
	t.nextRequest = 2
	return nil
}

// handshake initializes the connection without starting a conversation.
func (t *turnRun) handshake(ctx context.Context) error {
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

// startThread starts or resumes the conversation on its first message.
func (t *turnRun) startThread(ctx context.Context) error {
	session := t.profile.Session
	id := IntegerRequestID(2)
	params := ThreadStartParams{Model: session.Model, Cwd: "/work", ApprovalPolicy: "never", Sandbox: "danger-full-access",
		BaseInstructions: session.BaseInstructions, DeveloperInstructions: session.DeveloperInstructions}
	raw, err := t.enc.EncodePersistentThread(id, params, ThreadID(t.turn.Resume))
	if err != nil {
		return err
	}
	if err = t.send(ctx, raw); err != nil {
		return err
	}
	method := ThreadStart
	if t.turn.Resume != "" {
		method = ThreadResume
	}
	r, err := t.response(ctx, id, method)
	if err != nil {
		return err
	}
	if t.thread != "" && t.thread != r.Thread || t.turn.Resume != "" && string(r.Thread) != t.turn.Resume {
		return ErrProtocol
	}
	t.thread = r.Thread
	return nil
}

// runTurn sends the prompt and serves the turn until the provider's terminal
// status. Tool calls and provider output are separate queues; calls are
// answered one at a time.
func (t *turnRun) runTurn(ctx context.Context, res *run.TurnResult) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.thread == "" {
		if err := t.startThread(ctx); err != nil {
			return err
		}
	}
	t.nextRequest++
	id := IntegerRequestID(t.nextRequest)
	raw, err := t.enc.EncodeTurnStart(id, TurnStartParams{Thread: t.thread, Text: t.turn.Prompt, Effort: t.profile.Session.Effort})
	if errors.Is(err, ErrProtocolLimit) {
		return fmt.Errorf("%w: the prompt of %d bytes exceeds the %d bytes the provider channel carries: %w", run.ErrContextLimit, len(t.turn.Prompt), t.profile.MaxWireBytes, err)
	}
	if err != nil {
		return err
	}
	if err = t.send(ctx, raw); err != nil {
		return err
	}
	tools := t.peer.ToolCalls()
	responseSeen := false
	for {
		var m Message
		select {
		case <-ctx.Done():
			return ctx.Err()
		case call, ok := <-tools:
			if !ok {
				tools = nil
				continue
			}
			if err = t.answer(ctx, call); err != nil {
				return err
			}
			continue
		case e, ok := <-t.events:
			if m, err = t.accept(e, ok); err != nil {
				return err
			}
		}
		status, err := t.observe(m, id, &responseSeen)
		if err != nil {
			return err
		}
		if status == "" {
			continue
		}
		// The two queues have no cross-channel order: never accept the end of
		// the turn while a tool call is still waiting for its answer.
		select {
		case _, ok := <-tools:
			if ok {
				return fmt.Errorf("%w: provider ended the turn with a tool call unanswered", ErrProtocolIncomplete)
			}
		default:
		}
		res.Reason = "provider turn " + string(status)
		if status != TurnCompleted {
			return fmt.Errorf("%w: provider turn %s", ErrSession, status)
		}
		res.Completed = true
		return nil
	}
}

// observe applies one provider message to the turn. It returns the terminal
// status when the message ends the turn.
func (t *turnRun) observe(m Message, request RequestID, responseSeen *bool) (TurnStatus, error) {
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
func (t *turnRun) notification(m Message) error {
	switch m.method {
	case "error":
		var e struct {
			Thread ThreadID `json:"threadId"`
			Error  struct {
				Message string `json:"message"`
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
			return fmt.Errorf("%w: provider error: %s", ErrProtocol, t.redact([]byte(e.Error.Message)))
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
		if !main || m.method != "item/completed" {
			return nil
		}
		if n.Item.Type == "agentMessage" && n.Item.Phase == "final_answer" {
			t.final = n.Item.Text
		}
		if n.Item.Type == "commandExecution" {
			t.commands = append(t.commands, run.CommandObservation{ID: n.Item.ID, Command: string(t.redact([]byte(n.Item.Command))), Status: n.Item.Status, ExitCode: n.Item.ExitCode, Output: string(t.redact([]byte(n.Item.Output)))})
		}
	case "thread/tokenUsage/updated":
		var n struct {
			Thread string `json:"threadId"`
			Usage  struct {
				Last struct {
					Input      int64 `json:"inputTokens"`
					Cached     int64 `json:"cachedInputTokens"`
					Output     int64 `json:"outputTokens"`
					CacheWrite int64 `json:"cacheWriteInputTokens"`
				} `json:"last"`
			} `json:"tokenUsage"`
		}
		// Usage is best effort: a report this adapter cannot read counts as none.
		if json.Unmarshal(m.body, &n) != nil || n.Thread != string(t.thread) {
			return nil
		}
		// Codex counts cached input inside inputTokens; Usage keeps them apart.
		last := n.Usage.Last
		t.usage = t.usage.Add(run.Usage{InputTokens: max(last.Input-last.Cached, 0), OutputTokens: last.Output,
			CacheReadTokens: last.Cached, CacheCreationTokens: last.CacheWrite, ProviderTurns: 1})
	}
	return nil
}

// answer serves one Orchestrator tool call through the handler. A refusal is a tool
// error for the provider, not a failure of the turn.
func (t *turnRun) answer(ctx context.Context, call frame.ToolCall) error {
	var meta struct {
		Thread string `json:"threadId"`
	}
	if !t.offered[call.Name] || json.Unmarshal(call.Meta, &meta) != nil || meta.Thread != string(t.thread) {
		return ErrProtocol
	}
	if raw, err := frame.EncodeToolCall(call); err == nil {
		t.tr.record("tool-call", raw)
	}
	answered := make(chan run.ToolAnswer, 1)
	arguments := t.redact(call.Arguments)
	go func() { answered <- t.handler(ctx, call.Name, arguments) }()
	var answer run.ToolAnswer
	select {
	case answer = <-answered:
	case <-ctx.Done():
		return ctx.Err()
	}
	type text struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	result, err := json.Marshal(struct {
		Content []text `json:"content"`
		IsError bool   `json:"isError"`
	}{[]text{{"text", answer.Text}}, answer.Refused})
	if err != nil {
		return err
	}
	reply := frame.ToolReply{ID: call.ID, Result: result}
	if raw, err := frame.EncodeToolReply(reply); err == nil {
		t.tr.record("tool-reply", raw)
	}
	return t.safe(within(ctx, func() error { return t.peer.Reply(reply) }))
}

// send writes one NDJSON request. A failed write may have been partial and is
// never repeated.
func (t *turnRun) send(ctx context.Context, raw []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.tr.record("send", raw)
	return t.safe(within(ctx, func() error { return t.peer.WriteInput(raw) }))
}

// response waits for the response to request id, skipping notifications that
// carry no obligation during the handshake.
func (t *turnRun) response(ctx context.Context, id RequestID, method RequestMethod) (Response, error) {
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
			if t.thread != "" && t.thread != m.Thread() || t.turn.Resume != "" && string(m.Thread()) != t.turn.Resume {
				return Response{}, ErrProtocol
			}
			t.thread = m.Thread()
		case UnknownNotification:
		default:
			return Response{}, fmt.Errorf("%w: %s", ErrProtocolUnsupported, m.kind)
		}
	}
}

func (t *turnRun) rpcError(m Message) error {
	code, text, err := m.RPCError()
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: provider refused the request (%d): %s", ErrProtocol, code, t.redact([]byte(text)))
}

// accept records one provider message and reports a closed or failed stream.
func (t *turnRun) accept(e event, open bool) (Message, error) {
	if !open {
		return Message{}, fmt.Errorf("%w: provider output ended before the turn finished", ErrProtocolIncomplete)
	}
	if len(e.msg.raw) > 0 {
		t.tr.record("receive", e.msg.raw)
	}
	return e.msg, t.safe(e.err)
}

// finish removes private login data after observed provider exit, closes the
// container and transcript, and returns err joined with every
// cleanup failure.
func (t *turnRun) finish(ctx context.Context, res *run.TurnResult, err error) error {
	close(t.done)
	observed, closed := false, false
	if t.peer != nil {
		observed = !t.providerAttempted
		var stopErr error
		if t.providerAttempted {
			stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), t.agents.stopTimeout)
			var exit frame.Exited
			exit, stopErr = stopProvider(stopCtx, t.peer, t.agents.stopGrace)
			cancel()
			if stopErr == nil {
				observed = true
				t.tr.record("exit", []byte(fmt.Sprintf("code=%d signal=%s", exit.Code, exit.Signal)))
			}
		}
		if t.loginInstalled && observed {
			cleanCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), t.agents.stopTimeout)
			cleanErr := t.removeCredential(cleanCtx)
			cancel()
			if cleanErr != nil {
				res.Reason += "; runtime login cleanup incomplete"
				err = errors.Join(err, cleanErr)
			}
		}
		// The container really ends in Close: Stopped also needs a Close that did
		// not have to give up on it.
		closeErr := t.safe(t.peer.Close())
		closed = closeErr == nil
		err = errors.Join(err, t.safe(stopErr), closeErr)
		switch {
		case !observed:
			res.Reason = strings.TrimPrefix(res.Reason+"; provider exit not observed", "; ")
		case !closed:
			res.Reason = strings.TrimPrefix(res.Reason+"; provider exit observed but the container end was not", "; ")
		}
	}
	ended := observed && closed
	res.Stopped = ended || !t.attempted
	return errors.Join(err, t.tr.close())
}

func stopProvider(ctx context.Context, peer RuntimeSession, grace time.Duration) (frame.Exited, error) {
	type result struct {
		exit frame.Exited
		err  error
	}
	done := make(chan result, 1)
	go func() {
		exit, err := peer.Stop(ctx, grace.Milliseconds())
		done <- result{exit, err}
	}()
	select {
	case r := <-done:
		return r.exit, r.err
	case <-ctx.Done():
		return frame.Exited{}, ctx.Err()
	}
}

// within returns when f finished or ctx ended; f itself may outlive the call
// until the connection is closed.
func within(ctx context.Context, f func() error) error {
	done := make(chan error, 1)
	go func() { done <- f() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
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
func (t *turnRun) redact(data []byte) []byte {
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
func (t *turnRun) safe(err error) error {
	if err == nil {
		return nil
	}
	text := []byte(err.Error())
	if clean := t.redact(text); !bytes.Equal(clean, text) {
		return fmt.Errorf("%w: %s", ErrSession, clean)
	}
	return err
}

var _ run.Agents = (*Agents)(nil)
