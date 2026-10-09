package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectworkspace"
)

// Options supplies Host integration hooks. Hooks run synchronously, must be
// bounded, and must not call back into this adapter. BeforeStart reserves a root
// request BEFORE dispatch; native child events are observations AFTER dispatch.
type Options struct {
	BeforeStart    func(context.Context, agentexec.RoleStartRequest) error
	OnHandle       func(context.Context, RecoveryHandle) error
	OnEvent        func(context.Context, Event) error
	DynamicTools   []DynamicTool
	HandleToolCall func(context.Context, ToolCall) (ToolResult, error)
	MaxToolCalls   int
	ToolTimeout    time.Duration
}

// RecoveryHandle is private Host state, not agent-authored data. Persist exactly
// the returned value in the invocation's trusted journal. It authorizes inspection
// of that invocation only, never a new turn, fork, or another Manager's context.
type RecoveryHandle struct {
	Protocol       string                  `json:"protocol"`
	Fingerprint    string                  `json:"fingerprint"`
	Invocation     agentexec.Invocation    `json:"invocation"`
	Workspace      projectworkspace.Handle `json:"workspace"`
	ThreadID       string                  `json:"threadId"`
	SessionID      string                  `json:"sessionId"`
	TurnID         string                  `json:"turnId,omitempty"`
	TurnDispatched bool                    `json:"turnDispatched"`
}

type Adapter struct {
	config  Config
	options Options
}

func NewAdapter(cfg Config, options Options) (*Adapter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.ProviderVersion != SupportedProviderVersion {
		return nil, errors.New("unsupported App Server provider/protocol version")
	}
	if cfg.Timeout > 30*time.Minute || cfg.MaxEventBytes > 256<<20 {
		return nil, errors.New("App Server time/event budget exceeds supported bounds")
	}
	if err := options.validateTools(cfg.Timeout); err != nil {
		return nil, err
	}
	options.DynamicTools = append([]DynamicTool(nil), options.DynamicTools...)
	for i := range options.DynamicTools {
		options.DynamicTools[i].InputSchema = append(json.RawMessage(nil), options.DynamicTools[i].InputSchema...)
	}
	if !strings.Contains(" minimal low medium high xhigh max ultra ", " "+cfg.ReasoningEffort+" ") {
		return nil, errors.New("unsupported reasoning effort")
	}
	return &Adapter{cfg, options}, nil
}

func (a *Adapter) Fingerprint(cfg agentexec.Config) (string, error) {
	if cfg.Command != a.config.Command || cfg.ProviderVersion != a.config.ProviderVersion || cfg.Model != a.config.Model || cfg.Timeout != a.config.Timeout || len(cfg.Args) != 0 || cfg.WorkspaceMode != "" {
		return "", errors.New("shared invocation configuration differs from App Server configuration")
	}
	var opts map[string]json.RawMessage
	if len(cfg.ModelOptions) > 0 && (json.Unmarshal(cfg.ModelOptions, &opts) != nil || len(opts) != 0) {
		return "", errors.New("App Server model options belong in explicit transport configuration")
	}
	base, err := agentexec.Fingerprint(cfg)
	if err != nil {
		return "", err
	}
	settings, err := a.config.Digest()
	if err != nil {
		return "", err
	}
	tools, _ := json.Marshal(struct {
		Tools    []DynamicTool
		MaxCalls int
		Timeout  time.Duration
	}{a.options.DynamicTools, a.options.MaxToolCalls, a.options.ToolTimeout})
	return digest(append([]byte(protocolIdentity+base+settings), tools...)), nil
}

func digest(b []byte) string { return digestBytes(b) }

type thread struct {
	ID             string `json:"id"`
	SessionID      string `json:"sessionId"`
	ParentThreadID string `json:"parentThreadId"`
	ForkedFromID   string `json:"forkedFromId"`
	CLIVersion     string `json:"cliVersion"`
	CWD            string `json:"cwd"`
	Turns          []turn `json:"turns"`
}
type turn struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Items  []item          `json:"items"`
	Error  json.RawMessage `json:"error"`
}
type item struct {
	ID                string   `json:"id"`
	Type              string   `json:"type"`
	Text              string   `json:"text"`
	Phase             string   `json:"phase"`
	Tool              string   `json:"tool"`
	Status            string   `json:"status"`
	SenderThreadID    string   `json:"senderThreadId"`
	ReceiverThreadIDs []string `json:"receiverThreadIds"`
	Model             string   `json:"model"`
	ReasoningEffort   string   `json:"reasoningEffort"`
	AgentsStates      map[string]struct {
		Status string `json:"status"`
	} `json:"agentsStates"`
}
type threadResponse struct {
	Thread                  thread          `json:"thread"`
	Model                   string          `json:"model"`
	ReasoningEffort         string          `json:"reasoningEffort"`
	CWD                     string          `json:"cwd"`
	InstructionSources      []string        `json:"instructionSources"`
	ApprovalPolicy          json.RawMessage `json:"approvalPolicy"`
	Sandbox                 json.RawMessage `json:"sandbox"`
	ActivePermissionProfile *struct {
		ID string `json:"id"`
	} `json:"activePermissionProfile"`
}

type session struct {
	a               *Adapter
	ctx             context.Context
	c               *Client
	h               RecoveryHandle
	life            *agentexec.Lifecycle
	output          map[string]string
	final           string
	terminal        bool
	usage           *agentexec.Usage
	children        map[string]int
	spawns          map[string]int
	outputLimit     int
	toolCalls       map[string]bool
	instructionPins map[string]string
	sessions        map[string]string
	receivers       map[string]int
}

func (a *Adapter) Run(parent context.Context, cfg agentexec.Config, req agentexec.Request, opts agentexec.RunOptions) (result agentexec.RunResult, err error) {
	if parent == nil {
		return result, errors.New("context required")
	}
	fp, err := a.Fingerprint(cfg)
	if err != nil {
		return result, err
	}
	if opts.Workspace == nil || opts.Workspace.ID == "" || !filepath.IsAbs(opts.Workspace.CWD) || opts.Workspace.BaseSHA != req.SourceRevision {
		return result, errors.New("owned workspace bound to request revision required")
	}
	info, err := os.Stat(opts.Workspace.CWD)
	if err != nil || !info.IsDir() {
		return result, errors.New("workspace CWD unavailable")
	}
	inv, wire, err := agentexec.PrepareInvocation(req)
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(parent, a.config.Timeout)
	defer cancel()
	start := time.Now()
	s := &session{a: a, ctx: ctx, h: RecoveryHandle{Protocol: protocolIdentity, Fingerprint: fp, Invocation: inv, Workspace: *opts.Workspace}, output: map[string]string{}, children: map[string]int{}, spawns: map[string]int{}, outputLimit: cfg.MaxStdoutBytes, toolCalls: map[string]bool{}}
	s.sessions = map[string]string{}
	s.receivers = map[string]int{}
	s.life = &agentexec.Lifecycle{Provider: "codex-app-server", State: "unknown", Accounting: "partial", Requested: agentexec.SessionSettings{Model: a.config.Model, ReasoningEffort: a.config.ReasoningEffort, PermissionProfile: a.config.PermissionProfile, CWD: opts.Workspace.CWD, InstructionDigest: digest(wire)}}
	result.Receipt = agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: inv.RunID, InputDigest: inv.InputDigest, ContextDigest: digest(req.Context), ConfigDigest: fp, ProviderVersion: a.config.ProviderVersion, ProviderVersionDigest: digest([]byte(a.config.ProviderVersion)), Lifecycle: s.life, Outcome: agentexec.OutcomeIncomplete}
	defer func() {
		result.Receipt.WallTimeMilliseconds = time.Since(start).Milliseconds()
		result.Receipt.Usage = s.usage
		if err != nil && s.h.TurnDispatched && !s.terminal {
			s.life.State = "unknown"
			err = errors.Join(err, ErrUncertain)
		}
	}()
	env := selectedEnvironment(cfg.EnvironmentAllowlist)
	if err = bindReceipt(&result.Receipt, cfg, env); err != nil {
		return result, err
	}
	if err = verifyVersion(ctx, a.config, env); err != nil {
		return result, err
	}
	p, err := startProcess(a.config, opts.Workspace.CWD, env, cfg.MaxStderrBytes)
	if err != nil {
		return result, err
	}
	defer func() {
		closeErr := p.Close()
		result.Receipt.StderrDigest = p.stderr.digest()
		if closeErr != nil {
			err = errors.Join(err, closeErr, ErrUncertain)
			s.life.State = "unknown"
		}
		if p.stderr.overflow {
			err = errors.Join(err, agentexec.ErrOutputTooLarge)
		}
	}()
	s.c, err = NewClient(p, a.config.MaxEventBytes, s.observe)
	if err != nil {
		return result, err
	}
	s.c.request = s.toolRequest
	s.c.experimental = len(a.options.DynamicTools) > 0 || a.config.PermissionProfile != ""
	defer func() {
		_ = s.c.Close()
		result.Receipt.StdoutDigest = digest([]byte(s.final))
		s.life.EventLogDigest = s.c.EventDigest()
	}()
	if err = s.c.initialize(ctx); err != nil {
		return result, err
	}
	root := agentexec.RoleStartRequest{RequestID: inv.RunID, Role: req.Role, Model: a.config.Model, ReasoningEffort: a.config.ReasoningEffort, State: "requested"}
	s.life.StartRequests = append(s.life.StartRequests, root)
	if a.options.BeforeStart != nil {
		if err = a.options.BeforeStart(ctx, root); err != nil {
			s.life.State = "failed"
			s.life.StartRequests[0].State = "failed"
			return result, err
		}
	}
	config := map[string]any{"model_reasoning_effort": a.config.ReasoningEffort}
	if a.config.Helpers.Enabled {
		config["agents.max_concurrent_threads_per_session"] = a.config.Helpers.MaxStartRequests
	}
	var started threadResponse
	startParams := map[string]any{"model": a.config.Model, "cwd": opts.Workspace.CWD, "config": config, "serviceName": "markitect"}
	if a.config.PermissionProfile != "" {
		startParams["permissions"] = a.config.PermissionProfile
	}
	if len(a.options.DynamicTools) > 0 {
		startParams["dynamicTools"] = a.options.DynamicTools
	}
	if err = s.c.call(ctx, "thread/start", startParams, &started); err != nil {
		s.life.StartRequests[0].State = "unknown"
		return result, err
	}
	if err = s.bind(started); err != nil {
		return result, err
	}
	s.life.StartRequests[0].SessionID = s.h.SessionID
	s.life.StartRequests[0].State = "started"
	if err = s.save(); err != nil {
		return result, err
	}
	// Persist uncertainty BEFORE sending. A lost reply must never be replayed.
	s.h.TurnDispatched = true
	if err = s.save(); err != nil {
		s.h.TurnDispatched = false
		return result, err
	}
	var begun struct {
		Turn turn `json:"turn"`
	}
	prompt := "Implement/assess the supplied Host invocation in this real workspace using ordinary project tools and guidance. Return exactly one JSON agent-execution Response bound to apiVersion, runId, nonce, inputDigest and request role. Do not invent lifecycle, workspace delta, or evidence. Response fields: apiVersion, runId, nonce, role, inputDigest, outcome, candidateFiles, evidenceRefs, verifierObservations, uncertainty; optional candidateJson/reportJson/usage. Invocation:\n" + string(wire)
	if err = s.c.call(ctx, "turn/start", map[string]any{"threadId": s.h.ThreadID, "input": []any{map[string]any{"type": "text", "text": prompt}}, "model": a.config.Model, "effort": a.config.ReasoningEffort, "cwd": opts.Workspace.CWD}, &begun); err != nil {
		s.interrupt()
		return result, err
	}
	if begun.Turn.ID == "" || (s.h.TurnID != "" && s.h.TurnID != begun.Turn.ID) {
		err = ErrProtocol
		s.interrupt()
		return result, err
	}
	s.h.TurnID = begun.Turn.ID
	s.life.TurnID = begun.Turn.ID
	if !s.terminal {
		s.life.State = "running"
	}
	if err = s.save(); err != nil {
		s.interrupt()
		return result, err
	}
	for !s.terminal {
		if _, err = s.c.next(ctx); err != nil {
			s.interrupt()
			return result, err
		}
	}
	if s.life.State != "completed" {
		return result, fmt.Errorf("App Server turn %s", s.life.State)
	}
	result.Response, err = agentexec.DecodeResponse([]byte(s.final), inv, "")
	if err != nil {
		return result, err
	}
	result.Receipt.Outcome = result.Response.Outcome
	if checkErr := s.verifyInstructions(); checkErr != nil {
		return result, checkErr
	}
	after, checkErr := a.Fingerprint(cfg)
	if checkErr != nil || after != fp {
		return result, agentexec.ErrInputChanged
	}
	return result, nil
}

func (s *session) save() error {
	if s.a.options.OnHandle != nil {
		wire, err := json.Marshal(s.h)
		if err != nil {
			return err
		}
		var copy RecoveryHandle
		if err = json.Unmarshal(wire, &copy); err != nil {
			return err
		}
		return s.a.options.OnHandle(s.ctx, copy)
	}
	return nil
}
func (s *session) bind(r threadResponse) error {
	if r.Thread.ID == "" || r.Thread.SessionID == "" || r.Thread.ParentThreadID != "" || r.Thread.ForkedFromID != "" || r.Thread.CLIVersion != "0.162.0" || r.Model != s.a.config.Model || r.ReasoningEffort != s.a.config.ReasoningEffort || filepath.Clean(r.CWD) != filepath.Clean(s.h.Workspace.CWD) {
		return errors.New("App Server fresh thread/effective settings mismatch")
	}
	if len(r.ApprovalPolicy) == 0 || !json.Valid(r.ApprovalPolicy) || len(r.Sandbox) == 0 || !json.Valid(r.Sandbox) {
		return errors.New("effective approval/sandbox settings unavailable")
	}
	if s.a.config.PermissionProfile != "" && (r.ActivePermissionProfile == nil || r.ActivePermissionProfile.ID != s.a.config.PermissionProfile) {
		return errors.New("explicit permission profile was not confirmed by server")
	}
	s.h.ThreadID = r.Thread.ID
	s.h.SessionID = r.Thread.SessionID
	s.sessions[r.Thread.ID] = r.Thread.SessionID
	s.life.SessionID = r.Thread.SessionID
	s.life.Effective = &agentexec.SessionSettings{Model: r.Model, ReasoningEffort: r.ReasoningEffort, CWD: r.CWD}
	if r.ActivePermissionProfile != nil {
		s.life.Effective.PermissionProfile = r.ActivePermissionProfile.ID
	}
	// Observe current bytes at the server-reported loaded paths. The protocol
	// reports paths, not loaded byte hashes; this is not proof of loaded bytes.
	paths := append([]string(nil), r.InstructionSources...)
	if len(paths) > 64 {
		return errors.New("instruction source count exceeds bound")
	}
	sort.Strings(paths)
	var pins []byte
	var instructionBytes int64
	s.instructionPins = map[string]string{}
	for _, path := range paths {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		if !filepath.IsAbs(path) {
			return ErrProtocol
		}
		b, err := readPinned(path, 4<<20)
		if err != nil {
			return err
		}
		instructionBytes += int64(len(b))
		if instructionBytes > 16<<20 {
			return errors.New("instruction sources exceed aggregate byte bound")
		}
		pins = append(pins, []byte(path+digest(b))...)
		s.instructionPins[path] = digest(b)
	}
	if len(paths) > 0 {
		// The bytes were observed after thread/start, not returned by the server.
		// Preserve this distinction in the private event stream rather than claim
		// an exact loaded-byte identity in Effective.InstructionDigest.
		observed, _ := json.Marshal(map[string]any{"sources": s.instructionPins, "currentBytesDigest": digest(pins), "loadedBytes": "unknown"})
		if s.a.options.OnEvent != nil {
			if err := s.a.options.OnEvent(s.ctx, Event{Method: "markitect/instructionSources/observed", Params: observed}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *session) verifyInstructions() error {
	for path, pin := range s.instructionPins {
		b, err := readPinned(path, 4<<20)
		if err != nil || digest(b) != pin {
			return agentexec.ErrInputChanged
		}
	}
	return nil
}

func (s *session) interrupt() {
	if s.h.ThreadID == "" || s.h.TurnID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Send interrupt without waiting for a possibly lost prior RPC reply.
	s.c.nextID++
	_ = s.c.send(ctx, map[string]any{"id": s.c.nextID, "method": "turn/interrupt", "params": map[string]string{"threadId": s.h.ThreadID, "turnId": s.h.TurnID}})
	for !s.terminal {
		msg, err := s.c.next(ctx)
		if err != nil {
			return
		}
		if msg.Method == "" {
			continue
		}
	}
}
