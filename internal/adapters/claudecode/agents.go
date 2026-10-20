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

// NewAgentsWithRuntime permits an Orchestrator-supplied runtime peer. Production peers
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

// Run implements run.Agents. A turn that fails before a container starts is
// reported as stopped, with the error. Every failure after the container
// started stops it before returning; Stopped reports whether the provider exit
// and the end of the container were observed. Usage is accumulated from the
// stream and replaced by the result message's totals; a turn without them
// reports a lower bound and says so in Reason. The known token is removed from tool-call arguments before the
// handler sees them.
func (a *Agents) Run(ctx context.Context, turn run.Turn, handler run.ToolHandler) (res run.TurnResult, err error) {
	// Before a container starts there is nothing to stop: the turn is over and
	// the error is the whole story.
	redact := redactor{}
	notStarted := func(err error) (run.TurnResult, error) {
		now := time.Now().UTC()
		return run.TurnResult{Started: now, Ended: now, Stopped: true, Reason: "no container started: " + redact.text(err.Error())}, redact.err(err)
	}
	profile, tools, err := a.admit(turn, handler)
	if err != nil {
		return notStarted(err)
	}
	credential, err := a.credentials(ctx, profile.CredentialRef)
	if err != nil || credential.Token == "" {
		return notStarted(fmt.Errorf("%w: credential %q unavailable", ErrSession, profile.CredentialRef))
	}
	redact = redactor{token: credential.Token}
	codec, err := NewCodec(int(profile.MaxWireBytes))
	if err != nil {
		return notStarted(err)
	}
	initID := ControlRequestID("markitect-initialize")
	initRaw, err := codec.EncodeInitialize(initID)
	if err != nil {
		return notStarted(err)
	}
	promptRaw, err := codec.EncodeUser(UserInput{Text: turn.Prompt})
	if err != nil {
		return notStarted(err)
	}
	log, err := openTranscript(turn.Transcript, redact)
	if err != nil {
		return notStarted(err)
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
	// The container really ends in Close: Stopped needs the provider's exit and
	// a Close that did not have to give up on the container.
	res.Stopped = stopErr == nil && closeErr == nil
	res.Ended = time.Now().UTC()
	switch {
	case res.Stopped:
		res.Reason = outcome + "; provider exit observed code=" + strconv.Itoa(exit.Code) + " signal=" + redact.text(exit.Signal)
	case stopErr == nil:
		res.Reason = outcome + "; provider exit observed but the container end was not: " + redact.text(closeErr.Error())
	default:
		res.Reason = outcome + "; provider exit not observed"
	}
	if c.usage != usageFinal {
		res.Reason += "; " + c.usage.partial()
	}
	log.value("end", map[string]any{"reason": res.Reason, "stopped": res.Stopped})
	return res, redact.err(errors.Join(runErr, stopErr, closeErr))
}

// admit resolves the turn's profile and validates everything that can be
// checked before a container starts. It returns the Orchestrator tools in agentd form.
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
	case !validMounts(turn.Mounts) || turn.Kind == run.VerifyTurn && writable(turn.Mounts):
		return bad("workspace mounts must be distinct clean relative directories, none writable in a verify turn")
	}
	var tools []frame.ToolSpec
	seen := map[string]bool{}
	for _, t := range turn.Tools {
		if !volumeName.MatchString(t.Name) || seen[t.Name] || strings.TrimSpace(t.Description) == "" {
			return bad("invalid or duplicate Orchestrator tool " + strconv.Quote(t.Name))
		}
		if _, err := object(t.InputSchema); err != nil {
			return bad("Orchestrator tool " + strconv.Quote(t.Name) + " has no JSON object schema")
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
		Mounts: append(workspaceMounts(turn), container.Mount{Volume: turn.Home, Target: "/home/agent"}),
		CPUs:   s.CPUs, MemoryBytes: s.MemoryBytes, Pids: s.Pids, TmpBytes: s.TmpBytes, Network: s.Network}
}

// workspaceMounts is the workspace volume at /work followed by the turn's
// overlays: each a directory of the same volume at its own path below /work.
func workspaceMounts(turn run.Turn) []container.Mount {
	mounts := []container.Mount{{Volume: turn.Workspace, Target: "/work", ReadOnly: turn.ReadOnly, NoCopy: true}}
	for _, m := range turn.Mounts {
		mounts = append(mounts, container.Mount{Volume: turn.Workspace, Target: "/work/" + m.Path, Subpath: m.Path, ReadOnly: !m.Writable, NoCopy: true})
	}
	return mounts
}

// validMounts reports whether every overlay path is a distinct clean relative
// directory path.
func validMounts(mounts []run.PathMount) bool {
	seen := map[string]bool{}
	for _, m := range mounts {
		if !container.ValidSubpath(m.Path) || seen[m.Path] {
			return false
		}
		seen[m.Path] = true
	}
	return true
}

func writable(mounts []run.PathMount) bool {
	return slices.ContainsFunc(mounts, func(m run.PathMount) bool { return m.Writable })
}

// turnTools returns the builtin tools a turn may use: a verify turn, or a turn
// that can write nowhere in its workspace, loses the tools that exist to change
// files. An Execute turn with a read-only base and writable mounts keeps them.
func turnTools(turn run.Turn, profile Profile) []string {
	cannotWrite := turn.Kind == run.VerifyTurn || turn.ReadOnly && !writable(turn.Mounts)
	var builtin []string
	for _, name := range profile.Session.BuiltinTools {
		if !cannotWrite || name != "Write" && name != "Edit" {
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

	// usage says what res.Usage holds; streamed holds the usage of each
	// assistant message by its ID, which the provider repeats per content block.
	usage    usageState
	streamed map[string]run.Usage
}

// usageState tells how complete the usage of a turn is.
type usageState int

const (
	// usageNone: no usage was observed.
	usageNone usageState = iota
	// usageStreamed: only the assistant messages' usage was observed.
	usageStreamed
	// usageNoTotals: a result message arrived without usage totals.
	usageNoTotals
	// usageFinal: the result message's totals.
	usageFinal
)

func (u usageState) partial() string {
	if u == usageNoTotals {
		return "usage partial: the result message carries no usage"
	}
	return "usage partial: no result message"
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
			case AssistantMessage:
				c.addStreamed(m.Raw())
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

// answer hands one Orchestrator tool call to the handler and returns its answer to the
// provider as an MCP text result, an error result when the Orchestrator refused.
func (c *conversation) answer(ctx context.Context, call frame.ToolCall) error {
	c.log.value("tool_call", call)
	var reply frame.ToolReply
	if !slices.ContainsFunc(c.tools, func(t frame.ToolSpec) bool { return t.Name == call.Name }) {
		reply = frame.ToolReply{ID: call.ID, Error: &frame.ToolError{Code: -32601, Message: "unknown Orchestrator tool"}}
	} else {
		a := c.handler(ctx, call.Name, c.redact.bytes(call.Arguments))
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
		Cost  float64     `json:"total_cost_usd"`
		Turns int         `json:"num_turns"`
		Usage *tokenUsage `json:"usage"`
	}
	if json.Unmarshal(m.Raw(), &body) != nil {
		return ErrProtocol
	}
	c.usage = usageNoTotals
	if body.Usage != nil {
		// The result's totals replace what the stream added up.
		u := &c.res.Usage
		u.InputTokens, u.OutputTokens, u.CacheReadTokens, u.CacheCreationTokens = body.Usage.Input, body.Usage.Output, body.Usage.CacheRead, body.Usage.CacheCreation
		u.CostUSD, u.ProviderTurns = body.Cost, body.Turns
		c.usage = usageFinal
	}
	if result.IsError {
		return fmt.Errorf("%w: provider reported %s (%s) %s", ErrSession, result.Subtype, result.TerminalReason, strings.Join(result.Errors, "; "))
	}
	return nil
}

// tokenUsage is the provider's token report of a result or assistant message.
type tokenUsage struct {
	Input         int64 `json:"input_tokens"`
	Output        int64 `json:"output_tokens"`
	CacheRead     int64 `json:"cache_read_input_tokens"`
	CacheCreation int64 `json:"cache_creation_input_tokens"`
}

// addStreamed adds one assistant message's usage to the turn's running total,
// so a turn that ends without a result still reports what it consumed. A
// message repeated per content block counts once, with its latest usage. A
// message without readable usage counts as none.
func (c *conversation) addStreamed(raw []byte) {
	var body struct {
		Message struct {
			ID    string      `json:"id"`
			Usage *tokenUsage `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal(raw, &body) != nil || body.Message.Usage == nil || c.usage >= usageNoTotals {
		return
	}
	if c.streamed == nil {
		c.streamed = map[string]run.Usage{}
	}
	id := body.Message.ID
	if id == "" {
		id = "#" + strconv.Itoa(len(c.streamed))
	}
	u := body.Message.Usage
	c.streamed[id] = run.Usage{InputTokens: u.Input, OutputTokens: u.Output, CacheReadTokens: u.CacheRead, CacheCreationTokens: u.CacheCreation, ProviderTurns: 1}
	total := run.Usage{Model: c.res.Usage.Model}
	for _, one := range c.streamed {
		total = total.Add(one)
	}
	c.res.Usage, c.usage = total, usageStreamed
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
