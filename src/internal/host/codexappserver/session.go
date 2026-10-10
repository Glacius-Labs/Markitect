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

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
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
	if cfg.Timeout > time.Hour || cfg.MaxEventBytes > 256<<20 {
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
	ID                string             `json:"id"`
	Type              string             `json:"type"`
	Text              string             `json:"text"`
	Phase             string             `json:"phase"`
	Tool              string             `json:"tool"`
	Status            string             `json:"status"`
	Changes           []fileUpdateChange `json:"changes"`
	SenderThreadID    string             `json:"senderThreadId"`
	ReceiverThreadIDs []string           `json:"receiverThreadIds"`
	Model             string             `json:"model"`
	ReasoningEffort   string             `json:"reasoningEffort"`
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
	a                    *Adapter
	ctx                  context.Context
	c                    *Client
	h                    RecoveryHandle
	inv                  agentexec.Invocation
	life                 *agentexec.Lifecycle
	output               map[string]string
	final                string
	terminal             bool
	usage                *agentexec.Usage
	children             map[string]int
	spawns               map[string]int
	outputLimit          int
	toolCalls            map[string]bool
	instructionPins      map[string]string
	sessions             map[string]string
	receivers            map[string]int
	pendingFileChanges   map[string]trackedFileChange
	handledApprovals     map[string]bool
	effectiveSandboxType string
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
	s := &session{a: a, ctx: ctx, h: RecoveryHandle{Protocol: protocolIdentity, Fingerprint: fp, Invocation: inv, Workspace: *opts.Workspace}, inv: inv, output: map[string]string{}, children: map[string]int{}, spawns: map[string]int{}, outputLimit: cfg.MaxStdoutBytes, toolCalls: map[string]bool{}, pendingFileChanges: map[string]trackedFileChange{}, handledApprovals: map[string]bool{}}
	s.sessions = map[string]string{}
	s.receivers = map[string]int{}
	s.life = &agentexec.Lifecycle{Provider: "codex-app-server", State: "unknown", Accounting: "partial", Requested: agentexec.SessionSettings{Model: a.config.Model, ReasoningEffort: a.config.ReasoningEffort, PermissionProfile: a.config.PermissionProfile, WindowsSandboxBackend: string(a.config.WindowsSandboxBackend), CWD: opts.Workspace.CWD, InstructionDigest: digest(wire)}}
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
	if err = bindReceipt(&result.Receipt, cfg, env, appServerArgs(a.config)); err != nil {
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
	s.c.request = s.serverRequest
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
	if a.config.PermissionProfile == ":workspace" {
		startParams["approvalPolicy"] = "never"
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
	prompt := nativeTurnPrompt(inv, wire)
	turnParams := map[string]any{"threadId": s.h.ThreadID, "input": []any{map[string]any{"type": "text", "text": prompt}}, "model": a.config.Model, "effort": a.config.ReasoningEffort, "cwd": opts.Workspace.CWD}
	if schema := nativeTurnOutputSchema(inv); schema != nil {
		turnParams["outputSchema"] = schema
	}
	if err = s.c.call(ctx, "turn/start", turnParams, &begun); err != nil {
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

func nativeTurnPrompt(inv agentexec.Invocation, wire []byte) string {
	contract := "Wire response contract:\n" +
		"- Return exactly one JSON object and no surrounding Markdown. Copy apiVersion, runId, nonce, inputDigest, and role exactly from this invocation. Do not invent lifecycle, workspace delta, or evidence.\n" +
		"- Always include candidateFiles, evidenceRefs, verifierObservations, and uncertainty as JSON arrays, including empty arrays when there are no entries. Each verifierObservations entry is an object with exactly subject, outcome, and detail string fields; observation outcome must be passed, failed, incomplete, or escalated. Never use strings in place of observation objects.\n" +
		"- evidenceRefs may contain only exact strings supplied in request.artifacts[].path, request.scopeIds, or request.policyIds. Do not invent references from context pointers, context field names, unsupplied paths, or unverified test/tool claims. Do not repeat a reference; [] is valid.\n" +
		"- Do not include nativeWork or usage; the Host owns lifecycle, workspace delta, and provider telemetry when available.\n"
	if (inv.Request.Role == agentexec.RoleExecutor || inv.Request.Role == agentexec.RoleVerifier) && nativeEvidenceRefsOnlyEmpty(inv) {
		contract += "- This invocation's constrained schema permits only an empty evidenceRefs array; return [].\n"
	}
	contract += "- CandidateFile output entries have exactly path, mode, and content; mode is 0644 or 0755 for this Git workspace, and content is plain UTF-8 text, not base64. Input Artifact digest/base64 fields are not the output format. Copy identifiers as decoded JSON string values, without adding escaping characters.\n"
	switch inv.Request.Role {
	case agentexec.RoleExecutor:
		contract += "- For executor responses, outer outcome must be one of proposed, failed, incomplete, or escalated. Never use a task report status such as blocked, complete, or partial as the outer outcome. verifierObservations must be an empty array; omit candidateJson.\n"
		var context struct {
			Kind           string          `json:"kind"`
			ResponseSchema json.RawMessage `json:"responseSchema"`
		}
		if json.Unmarshal(inv.Request.Context, &context) == nil && len(context.ResponseSchema) > 0 {
			contract += "- Include reportJson as a JSON object matching request.context.responseSchema exactly; do not encode the object as a string. Return every required property and use arrays for every declared array field.\n"
			contract += "- Return candidateFiles as an empty array for this native report invocation; the Host harvests your actual workspace delta. Do not reproduce file bytes or compute their hashes in the response.\n"
		} else {
			contract += "- Omit reportJson unless this invocation supplies request.context.responseSchema. A proposed response needs candidateFiles or reportJson.\n"
		}
		if context.Kind == "projectrun-task/v1" {
			contract += "- For projectrun-task/v1, reportJson.status is a task status (complete, partial, blocked, failed, or no-op) and is separate from outer outcome. If reportJson.escalateTo is empty, outer outcome is proposed; if escalateTo is nonempty, outer outcome is escalated. Never copy reportJson.status into outer outcome. Follow the task phase and typed response schema; preserve unresolved questions and risks.\n"
		}
	case agentexec.RoleVerifier:
		contract += "- For verifier responses, outer outcome must be one of passed, failed, incomplete, or escalated. candidateFiles must be empty; omit candidateJson and reportJson. A passed or failed result requires concrete verifierObservations as objects with subject, outcome, and detail.\n"
	case agentexec.RoleInfer:
		contract += "- For inference responses, outer outcome must be one of proposed, failed, incomplete, or escalated. candidateFiles and verifierObservations must be empty; omit reportJson. A proposed result requires candidateJson as a JSON object.\n"
	default:
		contract += "- The request role is unsupported; return incomplete with empty arrays and explain the limitation in uncertainty.\n"
	}
	opening := "Implement/assess the supplied Host invocation in this real workspace using ordinary project tools and guidance.\n"
	if inv.Request.Role == agentexec.RoleExecutor {
		var requestContext struct {
			Kind string `json:"kind"`
		}
		if json.Unmarshal(inv.Request.Context, &requestContext) == nil && requestContext.Kind == "projectrun-review/v1" {
			opening = "Assess the exact supplied review candidate using ordinary read-only project tools and cited context. This is an assessment only; do not implement the overall RunGoal, edit repository artifacts, invoke Host helpers, or dispatch work. The Manager task, accepted model, and child task definitions are review context only.\n"
		}
	}
	return opening +
		"Use a fresh temporary directory you own under the inherited OS temporary directory for test scratch and caches; create, use and clean it up within the same shell call because Windows MXC temp paths can differ between calls. Review and verification must leave repository artifacts unchanged.\n" +
		contract + "\nInvocation:\n" + string(wire)
}

// Constrain the native final message to the existing closed wire DTO. The Host
// still validates the response and harvests real bytes; schema is not evidence.
func nativeTurnOutputSchema(inv agentexec.Invocation) map[string]any {
	if inv.Request.Role != agentexec.RoleExecutor && inv.Request.Role != agentexec.RoleVerifier {
		return nil
	}
	text := map[string]any{"type": "string"}
	boundedText := map[string]any{"type": "string", "minLength": 1, "maxLength": nativeResponseTextMaxLength}
	list := func(items any) map[string]any {
		return map[string]any{"type": "array", "items": items, "maxItems": nativeResponseArrayMaxItems}
	}
	fixed := func(value string) map[string]any { return map[string]any{"type": "string", "enum": []string{value}} }
	candidates := list(map[string]any{"type": "object", "additionalProperties": false, "required": []string{"path", "mode", "content"}, "properties": map[string]any{"path": boundedText, "mode": map[string]any{"type": "string", "enum": []string{"0644", "0755"}}, "content": text}})
	observations := list(map[string]any{"type": "object", "additionalProperties": false, "required": []string{"subject", "outcome", "detail"}, "properties": map[string]any{"subject": boundedText, "outcome": map[string]any{"type": "string", "enum": []string{"passed", "failed", "incomplete", "escalated"}}, "detail": boundedText}})
	outcomes := []string{"proposed", "failed", "incomplete", "escalated"}
	if inv.Request.Role == agentexec.RoleVerifier {
		outcomes = []string{"passed", "failed", "incomplete", "escalated"}
		candidates["maxItems"] = 0
	} else {
		observations["maxItems"] = 0
	}
	properties := map[string]any{"apiVersion": fixed(inv.APIVersion), "runId": fixed(inv.RunID), "nonce": fixed(inv.Nonce), "inputDigest": fixed(inv.InputDigest), "role": fixed(inv.Request.Role), "outcome": map[string]any{"type": "string", "enum": outcomes}, "candidateFiles": candidates, "evidenceRefs": nativeEvidenceRefsSchema(inv), "verifierObservations": observations, "uncertainty": list(boundedText)}
	required := []string{"apiVersion", "runId", "nonce", "inputDigest", "role", "outcome", "candidateFiles", "evidenceRefs", "verifierObservations", "uncertainty"}
	var context struct {
		ResponseSchema json.RawMessage `json:"responseSchema"`
	}
	if inv.Request.Role == agentexec.RoleExecutor && json.Unmarshal(inv.Request.Context, &context) == nil && len(context.ResponseSchema) > 0 {
		properties["reportJson"] = context.ResponseSchema
		required = append(required, "reportJson")
		candidates["maxItems"] = 0
	}
	return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": properties}
}

const (
	// Keep native response-schema bounds aligned with agentexec's wire limits.
	nativeResponseArrayMaxItems = 128
	// Structured Outputs bounds string characters; agentexec's final decoder
	// remains authoritative for UTF-8 validity and the 4096-byte wire bound.
	nativeResponseTextMaxLength = 4096

	// Match the shared response count bound, but keep the constant local because
	// evidence references are not verifier observations.
	nativeEvidenceRefMaxItems = 128
	// OpenAI Structured Outputs supports at most 1000 enum values total and
	// limits a string enum over 250 values to 15,000 characters. Keep this one
	// request-derived enum comfortably bounded; larger unions legally fall back
	// to [] rather than making the native turn schema unrepresentable.
	nativeEvidenceEnumMaxValues = 250
	nativeEvidenceEnumMaxBytes  = 15_000
)

func nativeEvidenceRefs(inv agentexec.Invocation) []string {
	set := make(map[string]struct{}, len(inv.Request.Artifacts)+len(inv.Request.ScopeIDs)+len(inv.Request.PolicyIDs))
	for _, artifact := range inv.Request.Artifacts {
		set[artifact.Path] = struct{}{}
	}
	for _, id := range inv.Request.ScopeIDs {
		set[id] = struct{}{}
	}
	for _, id := range inv.Request.PolicyIDs {
		set[id] = struct{}{}
	}
	refs := make([]string, 0, len(set))
	for ref := range set {
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	sort.Strings(refs)
	return refs
}

func nativeEvidenceRefsSchema(inv agentexec.Invocation) map[string]any {
	refs := nativeEvidenceRefs(inv)
	item := map[string]any{"type": "string", "minLength": 1, "maxLength": nativeResponseTextMaxLength}
	array := map[string]any{"type": "array", "items": item, "maxItems": nativeEvidenceRefMaxItems}
	if !nativeEvidenceEnumFits(refs) {
		array["maxItems"] = 0
		return array
	}
	item["enum"] = refs
	return array
}

func nativeEvidenceRefsOnlyEmpty(inv agentexec.Invocation) bool {
	return !nativeEvidenceEnumFits(nativeEvidenceRefs(inv))
}

func nativeEvidenceEnumFits(refs []string) bool {
	if len(refs) == 0 || len(refs) > nativeEvidenceEnumMaxValues {
		return false
	}
	bytes := 0
	for _, ref := range refs {
		bytes += len(ref) // byte count is conservative for the provider's character budget
		if bytes > nativeEvidenceEnumMaxBytes {
			return false
		}
	}
	return true
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
	var effectiveSandbox struct {
		Type                string   `json:"type"`
		WritableRoots       []string `json:"writableRoots"`
		NetworkAccess       bool     `json:"networkAccess"`
		ExcludeTmpdirEnvVar bool     `json:"excludeTmpdirEnvVar"`
		ExcludeSlashTmp     bool     `json:"excludeSlashTmp"`
	}
	if json.Unmarshal(r.Sandbox, &effectiveSandbox) == nil {
		s.effectiveSandboxType = effectiveSandbox.Type
	}
	if s.a.config.PermissionProfile != "" && (r.ActivePermissionProfile == nil || r.ActivePermissionProfile.ID != s.a.config.PermissionProfile) {
		return errors.New("explicit permission profile was not confirmed by server")
	}
	if s.a.config.PermissionProfile == ":workspace" {
		var approval string
		if json.Unmarshal(r.ApprovalPolicy, &approval) != nil || approval != "never" || s.effectiveSandboxType != "workspaceWrite" || len(effectiveSandbox.WritableRoots) != 0 || effectiveSandbox.NetworkAccess || effectiveSandbox.ExcludeTmpdirEnvVar || effectiveSandbox.ExcludeSlashTmp {
			return errors.New("owned workspace startup settings were not confirmed by server")
		}
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
