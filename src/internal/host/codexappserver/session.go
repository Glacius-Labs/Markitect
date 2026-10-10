package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	if req.Role == agentexec.RoleExecutor {
		var requestContext struct {
			ResponseSchema json.RawMessage `json:"responseSchema"`
		}
		if json.Unmarshal(req.Context, &requestContext) == nil {
			if _, schemaErr := nativeFullVerifyReportSchema(inv, requestContext.ResponseSchema); schemaErr != nil {
				return result, schemaErr
			}
		}
	}
	if req.Role == agentexec.RoleVerifier {
		if _, err := nativeVerifierEvidenceAliases(inv); err != nil {
			return result, err
		}
		if _, err := nativeVerifierSubjectAliases(inv); err != nil {
			return result, err
		}
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
	prompt := nativeTurnPrompt(inv, wire, opts.Workspace.CWD)
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
	result.Response, err = decodeNativeFinal(s.final, inv)
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

func nativeTurnPrompt(inv agentexec.Invocation, wire []byte, workspaceCWD string) string {
	contract := "Wire response contract:\n"
	if inv.Request.Role == agentexec.RoleExecutor || inv.Request.Role == agentexec.RoleVerifier {
		contract += "- Return exactly one JSON object and no surrounding Markdown. Return only the role's semantic properties in the constrained schema; do not include invocation identity fields. The Host binds those from the trusted invocation.\n" +
			"- Always include candidateFiles, verifierObservations, and uncertainty as JSON arrays, including empty arrays when there are no entries. Each verifierObservations entry is an object with exactly subject, outcome, and detail string fields; observation outcome must be passed, failed, incomplete, or escalated. Never use strings in place of observation objects.\n" +
			"- Do not invent lifecycle or workspace delta.\n" +
			"- Do not include nativeWork or usage; the Host owns lifecycle, workspace delta, and provider telemetry when available.\n"
		if inv.Request.Role == agentexec.RoleVerifier {
			if subjectAliases, aliasErr := nativeVerifierSubjectAliases(inv); aliasErr == nil && subjectAliases != nil {
				keys := make([]string, 0, len(subjectAliases))
				for alias := range subjectAliases {
					keys = append(keys, alias)
				}
				sort.Strings(keys)
				pairs := make([][2]string, 0, len(keys))
				for _, alias := range keys {
					pairs = append(pairs, [2]string{alias, subjectAliases[alias]})
				}
				mapping, _ := json.Marshal(pairs)
				contract += "- This request has an exact required verifier subject set. Use the assigned aliases in verifierObservations[].subject exactly once each; do not omit, duplicate, paraphrase, or add subjects. The Host restores canonical subjects before its coverage check. Subject alias mapping (alias, canonical identifier): " + string(mapping) + ". Keep each observation's outcome and detail faithful to your assessment.\n"
			}
			aliases, err := nativeEvidenceRefAliases(inv)
			if err == nil {
				keys := make([]string, 0, len(aliases))
				for alias := range aliases {
					keys = append(keys, alias)
				}
				sort.Strings(keys)
				pairs := make([][2]string, 0, len(keys))
				for _, alias := range keys {
					pairs = append(pairs, [2]string{alias, aliases[alias]})
				}
				mapping, _ := json.Marshal(pairs)
				required, requiredErr := nativeVerifierEvidenceAliases(inv)
				if requiredErr == nil {
					requiredValues := make([]string, 0, len(required))
					for alias := range required {
						requiredValues = append(requiredValues, alias)
					}
					sort.Strings(requiredValues)
					requiredJSON, _ := json.Marshal(requiredValues)
					contract += "- For a verifier response, include exactly these evidenceRefs aliases once: " + string(requiredJSON) + ". The request-bound alias mapping is " + string(mapping) + ". Emit only aliases; never copy canonical reference strings. Listing an alias is bookkeeping, not proof of inspection or support.\n"
				}
			}
		} else {
			contract += "- Do not include evidenceRefs; the Host supplies an empty array for this executor invocation. Do not invent evidence.\n"
		}
	} else {
		contract += "- Return exactly one JSON object and no surrounding Markdown. Copy apiVersion, runId, nonce, inputDigest, and role exactly from this invocation. Do not invent lifecycle, workspace delta, or evidence.\n" +
			"- Always include candidateFiles, evidenceRefs, verifierObservations, and uncertainty as JSON arrays, including empty arrays when there are no entries. Each verifierObservations entry is an object with exactly subject, outcome, and detail string fields; observation outcome must be passed, failed, incomplete, or escalated. Never use strings in place of observation objects.\n" +
			"- evidenceRefs may contain only exact strings supplied in request.artifacts[].path, request.scopeIds, or request.policyIds. Do not invent references from context pointers, context field names, unsupplied paths, or unverified test/tool claims. Do not repeat a reference; [] is valid.\n" +
			"- Do not include nativeWork or usage; the Host owns lifecycle, workspace delta, and provider telemetry when available.\n"
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
			if context.Kind == "projectrun-full-verify/v1" {
				contract += "- Include reportJson as a JSON object matching the supplied full-verification report shape; do not encode the object as a string. Return every required property and use arrays for every declared array field. assessments[].subject uses the assigned short aliases described below; the Host maps them back to canonical identifiers.\n"
			} else {
				contract += "- Include reportJson as a JSON object matching request.context.responseSchema exactly; do not encode the object as a string. Return every required property and use arrays for every declared array field.\n"
			}
			contract += "- Return candidateFiles as an empty array for this native report invocation; the Host harvests your actual workspace delta. Do not reproduce file bytes or compute their hashes in the response.\n"
		} else {
			contract += "- Omit reportJson unless this invocation supplies request.context.responseSchema. A proposed response needs candidateFiles or reportJson.\n"
		}
		if context.Kind == "projectrun-task/v1" {
			contract += "- For projectrun-task/v1, reportJson.status is a task status (complete, partial, blocked, failed, or no-op) and is separate from outer outcome. If reportJson.escalateTo is empty, outer outcome is proposed; if escalateTo is nonempty, outer outcome is escalated. Never copy reportJson.status into outer outcome. Follow the task phase and typed response schema; preserve unresolved questions and risks.\n"
		}
		if context.Kind == "projectrun-review/v1" {
			contract += "- For projectrun-review/v1, return the exact typed report in reportJson and keep candidateFiles, verifierObservations, and uncertainty empty for this read-only assessment. The outer outcome is always proposed when returning a well-formed typed review report, whether reportJson.status is pass or fail. reportJson.status expresses the review conclusion; a valid fail finding is an assessment result, not an invocation failure. Preserve each actionable grounded finding in reportJson.findings and do not move it to the outer outcome or uncertainty.\n"
		}
		if context.Kind == "projectrun-full-verify/v1" {
			contract += "- For projectrun-full-verify/v1, perform a read-only, bounded Manager audit against the fixed snapshot, model, files, briefing, child assessments, and check results supplied in this invocation. The complete audit scope is exactly request.context.requiredSubjects. Use each assigned short alias below exactly once in assessments[].subject; do not omit, duplicate, paraphrase, or add subjects. The Host maps each alias back to its exact canonical subject. Treat canonical identifiers as identifiers and use adjacent supplied context to interpret them. Do not gate this scoped audit on unrelated Git inspection, Markitect CLI/MCP availability, or rerunning Host-supplied checks; report a capability limitation only when it prevents assessing a required subject. For a well-formed typed audit report, outer outcome is proposed; reportJson.status independently expresses pass, fail, or incomplete and proposed does not assert a pass. Mark only evidence-supported subjects pass; use fail for contradictory evidence and incomplete when relevant evidence for a required subject is unavailable. Never force pass or invent evidence. Put relevant uncertainty in the typed assessment detail and mark the subject and overall status incomplete when evidence is missing. Keep outer uncertainty empty; return candidateFiles and verifierObservations as empty arrays for this read-only report.\n"
			aliases, aliasErr := nativeFullVerifySubjectAliases(inv)
			if aliasErr == nil && aliases != nil {
				keys := make([]string, 0, len(aliases))
				for alias := range aliases {
					keys = append(keys, alias)
				}
				sort.Strings(keys)
				pairs := make([][2]string, 0, len(keys))
				for _, alias := range keys {
					pairs = append(pairs, [2]string{alias, aliases[alias]})
				}
				mapping, _ := json.Marshal(pairs)
				contract += "- Subject alias mapping (alias, canonical identifier): " + string(mapping) + ". Use only the alias in reportJson assessments[].subject; do not copy canonical strings into that field.\n"
			}
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
		} else if json.Unmarshal(inv.Request.Context, &requestContext) == nil && requestContext.Kind == "projectrun-full-verify/v1" {
			opening = "Perform only the read-only, bounded full-verification audit defined by the supplied requiredSubjects and fixed Host context. Do not edit repository artifacts, invoke Host helpers, or dispatch work.\n"
		}
	}
	opening += "Run every shell command from the exact Host-owned workspace CWD supplied here: " + workspaceCWD + ". Before using a shell to read, check, or write files, explicitly set and verify that working directory; shell processes may start elsewhere.\n"
	if nativeWritableManagerOrHelper(inv) {
		opening += "Within allowedWritePaths, ordinary project tools including scoped shell writes may edit repository files. On Windows, prefer the native file-change/editor tool when the workspace alias or a packaged AppData\\Local\\Packages\\...\\LocalCache path causes shell access problems. Do not switch to a path under that LocalCache tree even if a tool prints one. If a shell write is denied, do not retry through another filesystem path or request/add permissions; use the native file-change/editor operation, or report the observed limitation if that operation is unavailable.\n"
	}
	return opening +
		"When tests or tools need scratch space or caches, use a fresh directory you own under the inherited OS temporary directory and create, use, and clean it up within the same shell call because Windows temporary paths can differ between calls. Do not create scratch space when it is not needed. Review and verification must leave repository artifacts unchanged.\n" +
		contract + "\nInvocation:\n" + string(wire)
}

func nativeWritableManagerOrHelper(inv agentexec.Invocation) bool {
	if inv.Request.Role != agentexec.RoleExecutor {
		return false
	}
	var context struct {
		Kind              string   `json:"kind"`
		Phase             string   `json:"phase"`
		AllowedWritePaths []string `json:"allowedWritePaths"`
	}
	if json.Unmarshal(inv.Request.Context, &context) != nil || len(context.AllowedWritePaths) == 0 {
		return false
	}
	switch context.Kind {
	case "projectrun-task/v1":
		return context.Phase == "work" || context.Phase == "integrate"
	case "projectrun-helper/v1":
		return true
	default:
		return false
	}
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
	candidates := list(map[string]any{"type": "object", "additionalProperties": false, "required": []string{"path", "mode", "content"}, "properties": map[string]any{"path": boundedText, "mode": map[string]any{"type": "string", "enum": []string{"0644", "0755"}}, "content": text}})
	observations := list(map[string]any{"type": "object", "additionalProperties": false, "required": []string{"subject", "outcome", "detail"}, "properties": map[string]any{"subject": boundedText, "outcome": map[string]any{"type": "string", "enum": []string{"passed", "failed", "incomplete", "escalated"}}, "detail": boundedText}})
	outcomes := []string{"proposed", "failed", "incomplete", "escalated"}
	if inv.Request.Role == agentexec.RoleVerifier {
		outcomes = []string{"passed", "failed", "incomplete", "escalated"}
		candidates["maxItems"] = 0
		aliases, err := nativeVerifierSubjectAliases(inv)
		if err != nil {
			return nil
		}
		if aliases != nil {
			subject := observations["items"].(map[string]any)["properties"].(map[string]any)["subject"].(map[string]any)
			values := make([]string, 0, len(aliases))
			for alias := range aliases {
				values = append(values, alias)
			}
			sort.Strings(values)
			if len(values) == 0 {
				observations["maxItems"] = 0
			} else {
				subject["enum"] = values
			}
		}
	} else {
		observations["maxItems"] = 0
	}
	properties := map[string]any{"outcome": map[string]any{"type": "string", "enum": outcomes}, "candidateFiles": candidates, "verifierObservations": observations, "uncertainty": list(boundedText)}
	required := []string{"outcome", "candidateFiles", "verifierObservations", "uncertainty"}
	if inv.Request.Role == agentexec.RoleVerifier {
		aliases, err := nativeVerifierEvidenceAliases(inv)
		if err != nil {
			return nil
		}
		values := make([]string, 0, len(aliases))
		for alias := range aliases {
			values = append(values, alias)
		}
		sort.Strings(values)
		items := map[string]any{"type": "string", "minLength": 1, "maxLength": 64}
		evidence := list(items)
		if len(values) > 0 {
			items["enum"] = values
		} else {
			evidence["maxItems"] = 0
		}
		properties["evidenceRefs"] = evidence
		required = append(required, "evidenceRefs")
	}
	var context struct {
		ResponseSchema json.RawMessage `json:"responseSchema"`
	}
	if inv.Request.Role == agentexec.RoleExecutor && json.Unmarshal(inv.Request.Context, &context) == nil && len(context.ResponseSchema) > 0 {
		if transformed, err := nativeFullVerifyReportSchema(inv, context.ResponseSchema); err != nil {
			return nil
		} else if transformed != nil {
			var schemaValue any
			if json.Unmarshal(transformed, &schemaValue) != nil {
				return nil
			}
			properties["reportJson"] = schemaValue
		} else {
			properties["reportJson"] = context.ResponseSchema
		}
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
)

// decodeNativeFinal treats executor/verifier output as semantic data, with
// verifier references retained as request-bound aliases for coverage checks.
// Invocation identity is always Host-owned. Executor evidenceRefs are empty;
// verifier aliases are mapped back to supplied references before the shared
// decoder and projectrun's exact-coverage check run.
func decodeNativeFinal(final string, inv agentexec.Invocation) (agentexec.Response, error) {
	if inv.Request.Role != agentexec.RoleExecutor && inv.Request.Role != agentexec.RoleVerifier {
		return agentexec.DecodeResponse([]byte(final), inv, "")
	}
	allowed := map[string]bool{"outcome": true, "candidateFiles": true, "verifierObservations": true, "uncertainty": true}
	var context struct {
		Kind           string          `json:"kind"`
		ResponseSchema json.RawMessage `json:"responseSchema"`
	}
	if inv.Request.Role == agentexec.RoleExecutor && json.Unmarshal(inv.Request.Context, &context) == nil && len(context.ResponseSchema) > 0 {
		allowed["reportJson"] = true
	}
	if inv.Request.Role == agentexec.RoleVerifier {
		allowed["evidenceRefs"] = true
	}
	semantic, err := decodeNativeSemanticObject([]byte(final), allowed)
	if err != nil {
		return agentexec.Response{}, err
	}
	for _, required := range []string{"outcome", "candidateFiles", "verifierObservations", "uncertainty"} {
		if _, ok := semantic[required]; !ok {
			return agentexec.Response{}, errors.New("native response omitted a required semantic field")
		}
	}
	if allowed["reportJson"] {
		if _, ok := semantic["reportJson"]; !ok {
			return agentexec.Response{}, errors.New("native response omitted the required task report")
		}
		if context.Kind == "projectrun-full-verify/v1" {
			aliases, aliasErr := nativeFullVerifySubjectAliases(inv)
			if aliasErr != nil || aliases == nil {
				return agentexec.Response{}, errors.New("native full-verification subject bindings are invalid")
			}
			semantic["reportJson"], err = decodeNativeFullVerifyReportSubjects(semantic["reportJson"], aliases)
			if err != nil {
				return agentexec.Response{}, err
			}
		}
	}
	canonicalRefs := []string{}
	if inv.Request.Role == agentexec.RoleVerifier {
		refsRaw, ok := semantic["evidenceRefs"]
		if !ok {
			return agentexec.Response{}, errors.New("native verifier response omitted evidence references")
		}
		var aliases []string
		if err := json.Unmarshal(refsRaw, &aliases); err != nil || aliases == nil {
			return agentexec.Response{}, errors.New("native verifier evidence references must be an array")
		}
		canonicalRefs, err = decodeNativeEvidenceAliases(inv, aliases)
		if err != nil {
			return agentexec.Response{}, err
		}
		subjectAliases, aliasErr := nativeVerifierSubjectAliases(inv)
		if aliasErr != nil {
			return agentexec.Response{}, aliasErr
		}
		if subjectAliases != nil {
			observationRaw, ok := semantic["verifierObservations"]
			if !ok {
				return agentexec.Response{}, errors.New("native verifier response omitted observations")
			}
			semantic["verifierObservations"], err = decodeNativeVerifierObservationSubjects(observationRaw, subjectAliases)
			if err != nil {
				return agentexec.Response{}, err
			}
		}
	}
	full := map[string]json.RawMessage{}
	for key, value := range semantic {
		full[key] = value
	}
	for key, value := range map[string]string{"apiVersion": inv.APIVersion, "runId": inv.RunID, "nonce": inv.Nonce, "inputDigest": inv.InputDigest, "role": inv.Request.Role} {
		encoded, _ := json.Marshal(value)
		full[key] = encoded
	}
	refsWire, _ := json.Marshal(canonicalRefs)
	full["evidenceRefs"] = refsWire
	composed, err := json.Marshal(full)
	if err != nil {
		return agentexec.Response{}, errors.New("native response metadata could not be composed")
	}
	return agentexec.DecodeResponse(composed, inv, "")
}

func nativeEvidenceRefAliases(inv agentexec.Invocation) (map[string]string, error) {
	set := map[string]struct{}{}
	for _, artifact := range inv.Request.Artifacts {
		if artifact.Path != "" {
			set[artifact.Path] = struct{}{}
		}
	}
	for _, value := range inv.Request.ScopeIDs {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	for _, value := range inv.Request.PolicyIDs {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	refs := make([]string, 0, len(set))
	for value := range set {
		refs = append(refs, value)
	}
	sort.Strings(refs)
	for generation := 0; generation <= len(refs); generation++ {
		prefix := fmt.Sprintf("evidence-%06d-", generation)
		aliases := make(map[string]string, len(refs))
		collision := false
		for index, ref := range refs {
			alias := fmt.Sprintf("%s%06d", prefix, index)
			if _, exists := set[alias]; exists {
				collision = true
				break
			}
			aliases[alias] = ref
		}
		if !collision {
			return aliases, nil
		}
	}
	return nil, errors.New("could not construct a collision-free evidence alias namespace")
}

// nativeVerifierEvidenceAliases distinguishes allowed evidence identities from
// the exact references the project verifier must cover. Projectrun supplies
// requiredEvidenceRefs in its trusted context; other verifier callers retain
// the bounded compatibility behavior of requiring the full supplied union.
func nativeVerifierEvidenceAliases(inv agentexec.Invocation) (map[string]string, error) {
	allowed, err := nativeEvidenceRefAliases(inv)
	if err != nil {
		return nil, err
	}
	var context struct {
		RequiredEvidenceRefs *[]string `json:"requiredEvidenceRefs"`
	}
	if err := json.Unmarshal(inv.Request.Context, &context); err != nil {
		return nil, errors.New("native verifier request context is invalid")
	}
	refs := make([]string, 0, len(allowed))
	if context.RequiredEvidenceRefs == nil {
		for _, ref := range allowed {
			refs = append(refs, ref)
		}
	} else {
		refs = append(refs, (*context.RequiredEvidenceRefs)...)
	}
	if len(refs) > nativeResponseArrayMaxItems {
		return nil, fmt.Errorf("native verifier required evidence references exceed the supported %d values", nativeResponseArrayMaxItems)
	}
	seen := map[string]bool{}
	result := map[string]string{}
	for _, ref := range refs {
		if seen[ref] {
			return nil, errors.New("native verifier required evidence references contain a duplicate")
		}
		seen[ref] = true
		alias := ""
		for candidate, canonical := range allowed {
			if canonical == ref {
				alias = candidate
				break
			}
		}
		if alias == "" {
			return nil, errors.New("native verifier required evidence reference was not supplied")
		}
		result[alias] = ref
	}
	return result, nil
}

func decodeNativeEvidenceAliases(inv agentexec.Invocation, aliases []string) ([]string, error) {
	allowed, err := nativeEvidenceRefAliases(inv)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	refs := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		if seen[alias] {
			return nil, errors.New("native verifier evidence references contain a duplicate alias")
		}
		seen[alias] = true
		ref, ok := allowed[alias]
		if !ok {
			return nil, errors.New("native verifier evidence references contain an unknown alias")
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// decodeNativeSemanticObject accepts one closed top-level object and preserves
// raw nested values for the shared decoder, which checks their duplicate keys
// and role-specific semantics after Host metadata is composed.
func decodeNativeSemanticObject(data []byte, allowed map[string]bool) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	tok, err := decoder.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, errors.New("native response must be one JSON object")
	}
	fields := map[string]json.RawMessage{}
	for decoder.More() {
		tok, err = decoder.Token()
		key, ok := tok.(string)
		if err != nil || !ok || !allowed[key] {
			return nil, errors.New("native response contains an unsupported semantic field")
		}
		if _, exists := fields[key]; exists {
			return nil, errors.New("native response contains a duplicate semantic field")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, errors.New("native response contains invalid JSON")
		}
		fields[key] = value
	}
	if _, err = decoder.Token(); err != nil {
		return nil, errors.New("native response object is incomplete")
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, errors.New("native response contains trailing JSON")
	}
	return fields, nil
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
