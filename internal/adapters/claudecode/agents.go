package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const (
	// stopGrace is the polite-signal window before agentd kills the provider.
	stopGrace = 10 * time.Second
	// stopMargin bounds the wait for the exit observation beyond the grace.
	stopMargin = 10 * time.Second
)

// Agents runs one Claude Code turn per container. It implements run.Agents.
type Agents struct {
	start       RuntimeStarter
	profiles    map[string]Profile
	credentials CredentialResolver
}

var _ run.Agents = (*Agents)(nil)

func NewAgents(engine *container.Engine, profiles map[string]Profile, credentials CredentialResolver) (*Agents, error) {
	if engine == nil {
		return nil, ErrSession
	}
	return NewAgentsWithRuntime(func(ctx context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		return engine.Start(ctx, spec)
	}, profiles, credentials)
}

// NewAgentsWithRuntime permits a Host-supplied runtime peer. Production peers
// must preserve Engine.Start and Session's exit semantics.
func NewAgentsWithRuntime(start RuntimeStarter, profiles map[string]Profile, credentials CredentialResolver) (*Agents, error) {
	if start == nil || credentials == nil {
		return nil, ErrSession
	}
	checked := make(map[string]Profile, len(profiles))
	for name, profile := range profiles {
		if name == "" || profile.validate() != nil {
			return nil, fmt.Errorf("%w: profile %q is invalid", ErrSession, name)
		}
		profile.Session.BuiltinTools = slices.Clone(profile.Session.BuiltinTools)
		checked[name] = profile
	}
	return &Agents{start: start, profiles: checked, credentials: credentials}, nil
}

// Run implements run.Agents. Every failure after the container started stops it
// before returning; Stopped reports whether the provider exit was observed.
func (a *Agents) Run(ctx context.Context, turn run.Turn, handler run.ToolHandler) (res run.TurnResult, err error) {
	profile, tools, err := a.admit(turn, handler)
	if err != nil {
		return res, err
	}
	credential, err := a.credentials(ctx, profile.CredentialRef)
	if err != nil || credential.Token == "" {
		return res, fmt.Errorf("%w: credential %q unavailable", ErrSession, profile.CredentialRef)
	}
	redact := redactor{token: credential.Token}
	codec, err := NewCodec(int(profile.MaxWireBytes))
	if err != nil {
		return res, err
	}
	initID := ControlRequestID("markitect-initialize")
	initRaw, err := codec.EncodeInitialize(initID)
	if err != nil {
		return res, err
	}
	promptRaw, err := codec.EncodeUser(UserInput{Text: turn.Prompt})
	if err != nil {
		return res, redact.err(err)
	}
	log, err := openTranscript(turn.Transcript, redact)
	if err != nil {
		return res, err
	}
	defer func() { err = errors.Join(err, log.close()) }()

	if turn.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, turn.Timeout)
		defer cancel()
	}
	res.Started = time.Now().UTC()
	spec := turnSpec(turn, profile)
	peer, err := a.start(ctx, spec)
	if err == nil && peer == nil {
		err = ErrSession
	}
	if err != nil {
		res.Ended, res.Reason = time.Now().UTC(), "container did not start; no exit observed"
		return res, redact.err(err)
	}

	c := &conversation{turn: turn, peer: peer, codec: codec, handler: handler, log: log, redact: redact, tools: tools,
		initID: initID, initRaw: initRaw, promptRaw: promptRaw, res: &res, done: make(chan struct{})}
	runErr := c.run(ctx, frame.Start{Argv: turnArgv(turn, profile, tools), Dir: "/work", Env: []string{"CLAUDE_CODE_OAUTH_TOKEN=" + credential.Token}, Tools: tools})
	close(c.done)

	outcome := "provider result"
	switch {
	case errors.Is(runErr, context.DeadlineExceeded):
		outcome = "timeout"
	case errors.Is(runErr, context.Canceled):
		outcome = "cancelled"
	case runErr != nil:
		outcome = "failed: " + redact.text(runErr.Error())
	}
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopGrace+stopMargin)
	defer cancel()
	exit, stopErr := peer.Stop(stopCtx, stopGrace.Milliseconds())
	closeErr := peer.Close()
	res.Stopped = stopErr == nil
	res.Ended = time.Now().UTC()
	if res.Stopped {
		res.Reason = outcome + "; provider exit observed code=" + strconv.Itoa(exit.Code) + " signal=" + redact.text(exit.Signal)
	} else {
		res.Reason = outcome + "; provider exit not observed"
	}
	log.value("end", map[string]any{"reason": res.Reason, "stopped": res.Stopped})
	return res, redact.err(errors.Join(runErr, stopErr, closeErr))
}

// admit resolves the turn's profile and validates everything that can be
// checked before a container starts. It returns the Host tools in agentd form.
func (a *Agents) admit(turn run.Turn, handler run.ToolHandler) (Profile, []frame.ToolSpec, error) {
	profile, ok := a.profiles[turn.Profile]
	if !ok {
		return Profile{}, nil, fmt.Errorf("%w: unknown profile %q", ErrSession, turn.Profile)
	}
	bad := func(reason string) (Profile, []frame.ToolSpec, error) {
		return Profile{}, nil, fmt.Errorf("%w: %s", ErrSession, reason)
	}
	switch {
	case handler == nil:
		return bad("a tool handler is required")
	case !volumeName.MatchString(turn.ID) || !volumeName.MatchString(turn.Workspace) || !volumeName.MatchString(turn.Home) || turn.Workspace == turn.Home:
		return bad("turn ID, workspace and home must be distinct plain engine names")
	case turn.Kind != run.ExecuteTurn && turn.Kind != run.VerifyTurn, turn.Kind == run.VerifyTurn && (!turn.ReadOnly || turn.Resume != ""):
		return bad("a verify turn is read-only and never resumes")
	case strings.TrimSpace(turn.Prompt) == "" || turn.Timeout < 0:
		return bad("a prompt and a non-negative timeout are required")
	}
	var tools []frame.ToolSpec
	seen := map[string]bool{}
	for _, t := range turn.Tools {
		if !volumeName.MatchString(t.Name) || seen[t.Name] || strings.TrimSpace(t.Description) == "" {
			return bad("invalid or duplicate Host tool " + strconv.Quote(t.Name))
		}
		if _, err := object(t.InputSchema); err != nil {
			return bad("Host tool " + strconv.Quote(t.Name) + " has no JSON object schema")
		}
		seen[t.Name] = true
		tools = append(tools, frame.ToolSpec{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
	}
	return profile, tools, nil
}

func turnSpec(turn run.Turn, profile Profile) container.ContainerSpec {
	s := profile.Session
	return container.ContainerSpec{Name: "markitect-" + turn.ID, Image: profile.ImageID,
		Labels: map[string]string{"markitect.store": s.Store}, User: "1000:1000", Workdir: "/work",
		Mounts: []container.Mount{{Volume: turn.Workspace, Target: "/work", ReadOnly: turn.ReadOnly, NoCopy: true}, {Volume: turn.Home, Target: "/home/agent"}},
		CPUs:   s.CPUs, MemoryBytes: s.MemoryBytes, Pids: s.Pids, TmpBytes: s.TmpBytes, Network: s.Network}
}

// turnTools returns the builtin tools a turn may use: a read-only workspace
// takes away the tools that exist to change files.
func turnTools(turn run.Turn, profile Profile) []string {
	var builtin []string
	for _, name := range profile.Session.BuiltinTools {
		if !turn.ReadOnly || name != "Write" && name != "Edit" {
			builtin = append(builtin, name)
		}
	}
	return builtin
}

func turnArgv(turn run.Turn, profile Profile, tools []frame.ToolSpec) []string {
	builtin := turnTools(turn, profile)
	allowed := slices.Clone(builtin)
	for _, tool := range tools {
		allowed = append(allowed, "mcp__markitect__"+tool.Name)
	}
	s := profile.Session
	args := []string{"/usr/local/bin/claude", "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{"markitect":{"type":"stdio","command":"/usr/local/bin/markitect-agentd","args":["mcp"]}}}`,
		"--tools", strings.Join(builtin, ","), "--allowedTools", strings.Join(allowed, ","), "--permission-mode", "dontAsk", "--model", s.Model, "--effort", s.Effort,
		"--setting-sources", "", "--system-prompt", s.SystemPrompt}
	if turn.Resume != "" {
		args = append(args, "--resume", turn.Resume)
	}
	return args
}

// conversation is the state of one running turn. Only run's goroutine touches it
// besides the reader goroutine, which owns the codec.
type conversation struct {
	turn      run.Turn
	peer      RuntimeSession
	codec     *Codec
	handler   run.ToolHandler
	log       *transcript
	redact    redactor
	tools     []frame.ToolSpec
	initID    ControlRequestID
	initRaw   []byte
	promptRaw []byte
	res       *run.TurnResult
	done      chan struct{}
}

type streamEvent struct {
	message Message
	err     error
}

func (c *conversation) run(ctx context.Context, start frame.Start) error {
	if err := doCtx(ctx, func() error { _, err := c.peer.StartProvider(ctx, start); return err }); err != nil {
		return err
	}
	events := make(chan streamEvent)
	go c.read(events)
	if err := c.write(ctx, c.initRaw); err != nil {
		return err
	}
	calls := c.peer.ToolCalls()
	seen := map[uint64]bool{}
	acked, initialized := false, false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case call, ok := <-calls:
			if !ok {
				calls = nil
				continue
			}
			if call.ID == 0 || seen[call.ID] {
				return ErrProtocol
			}
			seen[call.ID] = true
			if err := c.answer(ctx, call); err != nil {
				return err
			}
		case event := <-events:
			if event.err != nil {
				c.log.line("provider-error", event.message.Raw())
				return event.err
			}
			m := event.message
			c.log.line("provider", m.Raw())
			if !acked && m.kind != ControlResponseMessage {
				return ErrProtocol
			}
			switch m.kind {
			case ControlResponseMessage:
				if acked {
					continue
				}
				ack, err := m.ReadInitializeResponse(c.initID)
				if err != nil || ack.Status != ControlSuccess || ack.Version != "" && ack.Version != "2.1.301" {
					return ErrProtocol
				}
				acked = true
				if err := c.write(ctx, c.promptRaw); err != nil {
					return err
				}
			case SystemInitMessage:
				init, err := m.Init()
				if err != nil || initialized || init.Version != "2.1.301" {
					return ErrProtocol
				}
				initialized = true
				c.res.Conversation, c.res.Usage.Model = string(init.Session), initModel(m.Raw())
			case ResultMessage:
				return c.finish(m)
			case UnsupportedRequest, RejectedMessage:
				return ErrUnsupported
			}
			// Other messages are the provider's own progress and carry no outcome.
		}
	}
}

// read frames provider output into messages until the stream ends or the turn
// is over.
func (c *conversation) read(events chan<- streamEvent) {
	send := func(e streamEvent) bool {
		select {
		case events <- e:
			return true
		case <-c.done:
			return false
		}
	}
	out, buf := c.peer.Stdout(), make([]byte, 8192)
	for {
		n, readErr := out.Read(buf)
		if n > 0 {
			messages, err := c.codec.Feed(buf[:n])
			for _, m := range messages {
				if m.kind == RejectedMessage {
					break
				}
				if !send(streamEvent{message: m}) {
					return
				}
			}
			if err != nil {
				send(streamEvent{message: Message{raw: c.codec.Pending()}, err: err})
				return
			}
		}
		if readErr != nil {
			err := fmt.Errorf("%w: provider output ended before its result", ErrIncomplete)
			if readErr != io.EOF {
				err = fmt.Errorf("%w: %v", ErrIncomplete, readErr)
			}
			if endErr := c.codec.End(); endErr != nil {
				err = endErr
			}
			send(streamEvent{err: err})
			return
		}
	}
}

// answer hands one Host tool call to the handler and returns its answer to the
// provider as an MCP text result, an error result when the Host refused.
func (c *conversation) answer(ctx context.Context, call frame.ToolCall) error {
	c.log.value("tool_call", call)
	var reply frame.ToolReply
	if !slices.ContainsFunc(c.tools, func(t frame.ToolSpec) bool { return t.Name == call.Name }) {
		reply = frame.ToolReply{ID: call.ID, Error: &frame.ToolError{Code: -32601, Message: "unknown Host tool"}}
	} else {
		a := c.handler(ctx, call.Name, call.Arguments)
		if err := ctx.Err(); err != nil {
			return err
		}
		c.log.value("tool_answer", map[string]any{"id": call.ID, "text": a.Text, "refused": a.Refused})
		result, err := json.Marshal(struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}{[]struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{{"text", a.Text}}, a.Refused})
		if err != nil {
			return err
		}
		reply = frame.ToolReply{ID: call.ID, Result: result}
	}
	return doCtx(ctx, func() error { return c.peer.Reply(reply) })
}

func (c *conversation) write(ctx context.Context, raw []byte) error {
	c.log.line("sent", raw)
	return doCtx(ctx, func() error { return c.peer.WriteInput(raw) })
}

// finish records the result message. An error result is still a complete
// observation, reported with the error.
func (c *conversation) finish(m Message) error {
	result, err := m.Result()
	if err != nil {
		return err
	}
	if c.res.Conversation == "" {
		c.res.Conversation = string(result.Session)
	}
	c.res.Final = result.Text
	var body struct {
		Cost  float64 `json:"total_cost_usd"`
		Turns int     `json:"num_turns"`
		Usage struct {
			Input         int64 `json:"input_tokens"`
			Output        int64 `json:"output_tokens"`
			CacheRead     int64 `json:"cache_read_input_tokens"`
			CacheCreation int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(m.Raw(), &body) != nil {
		return ErrProtocol
	}
	u := &c.res.Usage
	u.InputTokens, u.OutputTokens, u.CacheReadTokens, u.CacheCreationTokens = body.Usage.Input, body.Usage.Output, body.Usage.CacheRead, body.Usage.CacheCreation
	u.CostUSD, u.ProviderTurns = body.Cost, body.Turns
	if result.IsError {
		return fmt.Errorf("%w: provider reported %s (%s) %s", ErrSession, result.Subtype, result.TerminalReason, strings.Join(result.Errors, "; "))
	}
	return nil
}

func initModel(raw []byte) string {
	var init struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(raw, &init)
	return init.Model
}

// doCtx runs a blocking runtime call and gives up on it when ctx ends. The call
// itself ends when the runtime is closed.
func doCtx(ctx context.Context, fn func() error) error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
