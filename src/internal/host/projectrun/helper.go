package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

const HelperToolName = "markitect_start_helper"

const helperToolSchema = `{"type":"object","properties":{"task":{"type":"string","minLength":1,"maxLength":8192},"paths":{"type":"array","minItems":1,"maxItems":64,"items":{"type":"string","minLength":1,"maxLength":1024}}},"required":["task","paths"],"additionalProperties":false}`

// HelperDynamicTool is stable and may be used in the parent App Server
// fingerprint before a HelperSession exists.
func HelperDynamicTool() codexappserver.DynamicTool {
	return codexappserver.DynamicTool{Type: "function", Name: HelperToolName,
		Description: "Start one bounded helper in a fresh owned workspace, restricted to a subset of this Manager's write paths. Finish any parent workspace edits before calling this tool. Invoke it alone, without parallel workspace-changing tools, and wait for this tool to return before changing any parent workspace files or starting another helper. Inspect the returned result: success proves Host validation and delivery; child turn completion alone is not delivery. After a definitive request rejection, correct the request before a sequential retry. Do not continue when delivery is unknown or interrupted. Changing the parent workspace during this call invalidates the captured baseline and rejects delivery.",
		InputSchema: json.RawMessage(helperToolSchema)}
}

type HelperResponsibility struct {
	ManagerID string   `json:"managerId"`
	Purpose   string   `json:"purpose,omitempty"`
	Owns      []string `json:"owns"`
}

// HelperParentScope is Host metadata extracted from the parent request. It is
// not supplied by tool arguments and cannot be expanded by the model.
type HelperParentScope struct {
	ManagerID              string                 `json:"managerId"`
	AllowedWritePaths      []string               `json:"allowedWritePaths"`
	ExcludedWritePaths     []string               `json:"excludedWritePaths"`
	ActiveResponsibilities []HelperResponsibility `json:"activeResponsibilities"`
}

type HelperStartAttempt struct {
	Request     agentexec.RoleStartRequest
	ToolCallID  string
	Task        string
	Paths       []string
	ParentRunID string
	ManagerID   string
	Phase       string
}

// HelperReservation is one durable parent-owned reservation. AttachProtocolStart
// binds the child adapter's root request to this permit; it must not reserve or
// charge another Host start. Update appends state to the same request identity.
type HelperReservation interface {
	AttachProtocolStart(context.Context, agentexec.RoleStartRequest) error
	Update(context.Context, agentexec.RoleStartRequest) error
	CompleteDelivery(context.Context, agentexec.RoleStartRequest, HelperDelivery) error
}

type HelperReserveFunc func(context.Context, HelperStartAttempt) (HelperReservation, error)
type HelperInvokerFactory func(codexappserver.Options) Invoker

type HelperSessionOptions struct {
	ParentWorkspace     projectworkspace.Handle
	ParentConfig        agentexec.Config
	ParentRequest       agentexec.Request
	ParentScope         HelperParentScope
	Workspaces          projectworkspace.Service
	Limits              Limits
	PrivateLogDirectory string
	ChildOptions        codexappserver.Options
	Reserve             HelperReserveFunc
	NewInvoker          HelperInvokerFactory
	MaxStartRequests    int
}

type HelperSession struct {
	options   HelperSessionOptions
	service   candidateWorkspaceService
	maxStarts int

	mu               sync.Mutex
	bound            codexappserver.RecoveryHandle
	hasBinding       bool
	attempts         int
	toolCalls        map[string]bool
	requests         []agentexec.RoleStartRequest
	receipts         []agentexec.Receipt
	protocolStarts   map[string]bool
	receiptByRequest map[string]bool
}

type helperToolArgs struct {
	Task  string   `json:"task"`
	Paths []string `json:"paths"`
}

type helperContext struct {
	Guidance               string                 `json:"guidance"`
	Kind                   string                 `json:"kind"`
	HelperDepth            int                    `json:"helperDepth"`
	HelperAccounting       string                 `json:"helperAccounting"`
	ManagerID              string                 `json:"managerId"`
	Task                   string                 `json:"task"`
	AllowedWritePaths      []string               `json:"allowedWritePaths"`
	ExcludedWritePaths     []string               `json:"excludedWritePaths"`
	ActiveResponsibilities []HelperResponsibility `json:"activeResponsibilities"`
	ParentContext          json.RawMessage        `json:"parentContext"`
}

type helperJournalRecord struct {
	RequestID  string                   `json:"requestId"`
	ToolCallID string                   `json:"toolCallId"`
	Task       string                   `json:"task,omitempty"`
	Paths      []string                 `json:"paths,omitempty"`
	State      string                   `json:"state"`
	Error      string                   `json:"error,omitempty"`
	Handle     *projectworkspace.Handle `json:"handle,omitempty"`
	Receipt    *agentexec.Receipt       `json:"receipt,omitempty"`
	Delta      *projectworkspace.Delta  `json:"delta,omitempty"`
	RecordedAt time.Time                `json:"recordedAt"`
}

func NewHelperSession(options HelperSessionOptions) (*HelperSession, error) {
	workspace := options.ParentWorkspace
	noAuthority := options.ParentScope.ManagerID == ""
	if !noAuthority && (workspace.ID == "" || !filepath.IsAbs(workspace.CWD) || workspace.BaseSHA == "" || options.ParentRequest.SourceRevision != workspace.BaseSHA) {
		return nil, errors.New("helper session requires a parent request bound to its owned workspace")
	}
	if options.ParentConfig.Transport != TransportCodexAppServer {
		return nil, errors.New("Host helper tool requires a native App Server parent")
	}
	if (!noAuthority && options.PrivateLogDirectory == "") || options.Reserve == nil || (!noAuthority && (options.Limits.MaxCandidateFileBytes <= 0 || options.Limits.MaxCandidateBytes <= 0)) {
		return nil, errors.New("helper session requires private journaling, reservation callback, and positive candidate limits")
	}
	settings, err := decodeAppServerSettings(options.ParentConfig.TransportConfig)
	if err != nil {
		return nil, err
	}
	if !settings.Helpers.Enabled || settings.Helpers.MaxStartRequests < 1 || settings.Helpers.MaxDepth < 1 {
		return nil, errors.New("native App Server helper policy is disabled or unbounded")
	}
	if !json.Valid(options.ParentRequest.Context) {
		return nil, errors.New("helper session requires valid parent context")
	}
	parentManagerKnown := false
	for _, responsibility := range options.ParentScope.ActiveResponsibilities {
		if responsibility.ManagerID == options.ParentScope.ManagerID {
			parentManagerKnown = true
			break
		}
	}
	if options.ParentScope.ManagerID != "" && !parentManagerKnown {
		return nil, errors.New("helper parent Manager is absent from Host active responsibilities")
	}
	if options.ParentScope.ManagerID == "" && (len(options.ParentScope.AllowedWritePaths) != 0 || len(options.ParentScope.ActiveResponsibilities) != 0) {
		return nil, errors.New("unbound helper scope cannot carry Manager ownership")
	}
	service, serviceOK := options.Workspaces.(candidateWorkspaceService)
	if !noAuthority && !serviceOK {
		return nil, errors.New("helper session requires a candidate Git workspace service")
	}
	for _, scope := range options.ParentScope.AllowedWritePaths {
		if !safeRepoPath(strings.TrimSuffix(scope, "/")) {
			return nil, fmt.Errorf("invalid Host helper write scope %q", scope)
		}
	}
	for _, scope := range options.ParentScope.ExcludedWritePaths {
		if !safeRepoPath(strings.TrimSuffix(scope, "/")) {
			return nil, fmt.Errorf("invalid Host helper deny scope %q", scope)
		}
	}
	maxStarts := options.MaxStartRequests
	if maxStarts <= 0 {
		maxStarts = settings.Helpers.MaxStartRequests
	}
	var candidateService candidateWorkspaceService
	if serviceOK {
		candidateService = service
	}
	return &HelperSession{options: options, service: candidateService, maxStarts: maxStarts,
		toolCalls: map[string]bool{}, protocolStarts: map[string]bool{}, receiptByRequest: map[string]bool{}}, nil
}

func (s *HelperSession) DynamicTool() codexappserver.DynamicTool { return HelperDynamicTool() }

// BindHandle binds accepted tool calls to the parent's currently observed
// server thread and turn. Partial pre-turn handles are retained but cannot
// authorize a tool call until a non-empty turn ID has been observed.
func (s *HelperSession) BindHandle(handle codexappserver.RecoveryHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.options.ParentScope.ManagerID == "" && s.options.ParentWorkspace.ID == "" {
		if handle.Invocation.RunID == "" || handle.ThreadID == "" || handle.SessionID == "" {
			return errors.New("unscoped parent App Server handle is incomplete")
		}
		if s.hasBinding && (s.bound.Invocation.RunID != handle.Invocation.RunID || s.bound.ThreadID != handle.ThreadID || s.bound.SessionID != handle.SessionID) {
			return errors.New("parent App Server handle changed thread or invocation")
		}
		if s.hasBinding && s.bound.TurnID != "" && handle.TurnID != "" && s.bound.TurnID != handle.TurnID {
			return errors.New("parent App Server handle changed active turn")
		}
		if handle.TurnID == "" && s.hasBinding {
			handle.TurnID = s.bound.TurnID
		}
		s.bound, s.hasBinding = handle, true
		return nil
	}
	if handle.Workspace.ID != s.options.ParentWorkspace.ID || handle.Workspace.CWD != s.options.ParentWorkspace.CWD || handle.Invocation.RunID == "" || handle.ThreadID == "" || handle.SessionID == "" {
		return errors.New("parent App Server handle is not bound to the owned helper workspace")
	}
	if s.hasBinding && (s.bound.Invocation.RunID != handle.Invocation.RunID || s.bound.ThreadID != handle.ThreadID || s.bound.SessionID != handle.SessionID) {
		return errors.New("parent App Server handle changed thread or invocation")
	}
	if s.hasBinding && s.bound.TurnID != "" && handle.TurnID != "" && s.bound.TurnID != handle.TurnID {
		return errors.New("parent App Server handle changed active turn")
	}
	if handle.TurnID == "" && s.hasBinding {
		handle.TurnID = s.bound.TurnID
	}
	s.bound, s.hasBinding = handle, true
	return nil
}

func (s *HelperSession) HandleToolCall(ctx context.Context, call codexappserver.ToolCall) (codexappserver.ToolResult, error) {
	if ctx == nil || call.Tool != HelperToolName || call.Namespace != "" || call.CallID == "" || len(call.CallID) > 256 || !utf8.ValidString(call.CallID) || len(call.Arguments) > 64<<10 {
		return codexappserver.ToolResult{}, errors.New("invalid Host helper tool call")
	}
	s.mu.Lock()
	if !s.hasBinding || s.bound.ThreadID == "" || s.bound.TurnID == "" || call.ThreadID != s.bound.ThreadID || call.TurnID != s.bound.TurnID {
		s.mu.Unlock()
		return codexappserver.ToolResult{}, errors.New("Host helper call is not bound to the active parent thread and turn")
	}
	if s.toolCalls[call.CallID] {
		s.mu.Unlock()
		return codexappserver.ToolResult{}, errors.New("duplicate Host helper tool call ID")
	}
	s.toolCalls[call.CallID] = true
	s.attempts++ // malformed and denied requests consume the per-parent allowance
	ordinal := s.attempts
	parentHandle := s.bound
	s.mu.Unlock()

	args, parseErr := decodeHelperArgs(call.Arguments)
	request := agentexec.RoleStartRequest{RequestID: "helper-" + call.CallID, ParentSessionID: parentHandle.SessionID,
		Role: "helper", Model: s.options.ParentConfig.Model, State: "requested"}
	managerID, phase := helperParentMetadata(s.options.ParentRequest.Context)
	attempt := HelperStartAttempt{Request: request, ToolCallID: call.CallID, ParentRunID: parentHandle.Invocation.RunID, ManagerID: managerID, Phase: phase}
	if parseErr == nil {
		attempt.Task = args.Task
		attempt.Paths = append([]string(nil), args.Paths...)
	}
	index := s.recordRequest(request)
	reservation, reserveErr := s.options.Reserve(ctx, attempt)
	if reserveErr != nil {
		if errors.Is(reserveErr, ErrRoleStartAlreadyReserved) {
			s.mu.Lock()
			s.requests[index].State = "failed"
			s.mu.Unlock()
			return codexappserver.ToolResult{}, fmt.Errorf("Host helper request was already reserved: %w", reserveErr)
		}
		return s.failRequest(ctx, index, reservation, nil, call.CallID, attempt, "failed", fmt.Errorf("reserve Host helper start: %w", reserveErr))
	}
	if ordinal > s.maxStarts {
		return s.failRequest(ctx, index, reservation, nil, call.CallID, attempt, "failed", fmt.Errorf("parent helper start limit %d exceeded", s.maxStarts))
	}
	if parseErr != nil {
		return s.failRequest(ctx, index, reservation, nil, call.CallID, attempt, "failed", parseErr)
	}
	if err := validateHelperArgs(args, s.options.ParentScope); err != nil {
		// A correctly bound, uniquely identified helper request that fails the
		// parent's explicit path-scope check is a definitive pre-dispatch
		// rejection. Keep the reservation as a counted failed attempt, then
		// return bounded feedback to this same native turn so it may correct its
		// request. Any persistence uncertainty remains a transport error.
		if updateErr := s.updateRequest(ctx, index, reservation, "failed", agentexec.Receipt{}, ""); updateErr != nil {
			return codexappserver.ToolResult{}, errors.Join(err, updateErr)
		}
		return codexappserver.ToolResult{Success: false, Text: truncateHelperText(err.Error(), 4096)}, nil
	}
	return s.runHelper(ctx, call, parentHandle, index, reservation, attempt, args)
}

func (s *HelperSession) Requests() []agentexec.RoleStartRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]agentexec.RoleStartRequest(nil), s.requests...)
}

func (s *HelperSession) Receipts() []agentexec.Receipt {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]agentexec.Receipt, len(s.receipts))
	for i, receipt := range s.receipts {
		out[i] = cloneHelperReceipt(receipt)
	}
	return out
}

func (s *HelperSession) runHelper(ctx context.Context, call codexappserver.ToolCall, parentHandle codexappserver.RecoveryHandle, requestIndex int, reservation HelperReservation, attempt HelperStartAttempt, args helperToolArgs) (codexappserver.ToolResult, error) {
	if s.service == nil {
		return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", errors.New("Host helper has no writable Manager workspace"))
	}
	childConfig := s.options.ParentConfig
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", context.DeadlineExceeded)
		}
		if remaining < childConfig.Timeout {
			childConfig.Timeout = remaining
		}
	}
	identity, err := source.IdentifyGit(parentHandle.Workspace.CWD)
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", fmt.Errorf("identify parent workspace repository: %w", err))
	}
	baseline, err := projectworkspace.InspectRepository(ctx, parentHandle.Workspace.CWD, parentHandle.Workspace.BaseSHA)
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", fmt.Errorf("capture parent workspace baseline: %w", err))
	}
	taskID, err := nativeRandomID()
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", err)
	}
	request := projectworkspace.Request{RepositoryRoot: parentHandle.Workspace.CWD, RepositoryIdentity: identity.Digest,
		BaseSHA: parentHandle.Workspace.BaseSHA, OverlayDigest: baseline.OverlayDigest, TaskID: "helper-" + taskID,
		AllowedPaths: append([]string(nil), args.Paths...), ExcludedPaths: append([]string(nil), s.options.ParentScope.ExcludedWritePaths...)}
	if err := request.Validate(); err != nil {
		return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", err)
	}
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(nil)
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", err)
	}
	handle, err := s.service.PrepareCandidate(ctx, request, nil, overlayDigest)
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, nil, call.CallID, attempt, "failed", fmt.Errorf("prepare helper workspace: %w", err))
	}
	privateLogs, err := filepath.Abs(s.options.PrivateLogDirectory)
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, &handle, call.CallID, attempt, "failed", err)
	}
	journal, err := newNativeJournal(privateLogs, handle.CWD, handle.ID)
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, &handle, call.CallID, attempt, "failed", fmt.Errorf("create helper private journal: %w", err))
	}
	if err := createPrivateFile(filepath.Join(journal.directory, "helpers.jsonl")); err != nil {
		return s.failRequest(ctx, requestIndex, reservation, &handle, call.CallID, attempt, "failed", err, journal)
	}
	if err := s.appendHelperRecord(journal, helperJournalRecord{RequestID: attempt.Request.RequestID, ToolCallID: call.CallID,
		Task: args.Task, Paths: args.Paths, State: "prepared", Handle: &handle, RecordedAt: time.Now().UTC()}); err != nil {
		return s.failRequest(ctx, requestIndex, reservation, &handle, call.CallID, attempt, "failed", err, journal)
	}

	options := s.childOptions(reservation, attempt.Request)
	newInvoker := s.options.NewInvoker
	if newInvoker == nil {
		newInvoker = func(options codexappserver.Options) Invoker { return NewTransportInvokerWithoutHostHelpers(options) }
	}
	invoker := newInvoker(options)
	if invoker == nil {
		return s.failRequest(ctx, requestIndex, reservation, &handle, call.CallID, attempt, "failed", errors.New("helper invoker factory returned nil"), journal)
	}
	childRequest, err := s.childRequest(args)
	if err != nil {
		return s.failRequest(ctx, requestIndex, reservation, &handle, call.CallID, attempt, "failed", err, journal)
	}
	privateDirectory := filepath.Join(privateLogs, "helpers")
	if err := ensurePrivateDirectory(privateDirectory); err != nil {
		return s.failRequest(ctx, requestIndex, reservation, &handle, call.CallID, attempt, "failed", err, journal)
	}
	result, runErr := invoker.Run(ctx, childConfig, childRequest, agentexec.RunOptions{Workspace: &handle, PrivateLogDirectory: privateDirectory})
	s.storeReceipt(attempt.Request.RequestID, result.Receipt)
	requestRecord := helperJournalRecord{RequestID: attempt.Request.RequestID, ToolCallID: call.CallID, Task: args.Task,
		Paths: args.Paths, State: "running", Handle: &handle, Receipt: &result.Receipt, RecordedAt: time.Now().UTC()}
	terminal := helperInvocationTerminal(result.Receipt.Lifecycle)
	if !terminal {
		state := "unknown"
		if !s.protocolStartSeen(attempt.Request.RequestID) {
			state = "failed"
		}
		updateErr := s.updateRequest(ctx, requestIndex, reservation, state, result.Receipt, "")
		requestRecord.State, requestRecord.Error = state, boundedError(runErr)
		journalErr := s.appendHelperRecord(journal, requestRecord)
		return codexappserver.ToolResult{}, errors.Join(fmt.Errorf("helper child termination is %s; workspace preserved for recovery", state), runErr, updateErr, journalErr)
	}

	harvestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	delta, harvestErr := s.service.Harvest(harvestCtx, handle)
	cancel()
	if harvestErr == nil {
		var normalized projectworkspace.Delta
		normalized, harvestErr = projectworkspace.NormalizeDelta(request, handle, delta.Changes, helperWorkspaceLimits(s.options.Limits))
		if harvestErr == nil && (!sameHelperDeltaBinding(delta, normalized, request, handle) || normalized.Digest != delta.Digest) {
			harvestErr = errors.New("helper workspace delta binding or canonical digest differs")
		}
		if harvestErr == nil {
			harvestErr = validateHelperDeltaScope(normalized, args.Paths, s.options.ParentScope)
		}
		if harvestErr == nil {
			delta = normalized
		}
	}
	requestRecord.Delta = &delta
	if harvestErr != nil {
		requestRecord.State, requestRecord.Error = "failed", boundedError(errors.Join(runErr, harvestErr))
		journalErr := s.appendHelperRecord(journal, requestRecord)
		updateErr := s.updateRequest(ctx, requestIndex, reservation, "failed", result.Receipt, "")
		return codexappserver.ToolResult{}, errors.Join(fmt.Errorf("harvest helper child delta: %w", harvestErr), runErr, journalErr, updateErr)
	}
	requestRecord.State = "delta-harvested"
	if err := s.appendHelperRecord(journal, requestRecord); err != nil {
		updateErr := s.updateRequest(ctx, requestIndex, reservation, "unknown", result.Receipt, "")
		return codexappserver.ToolResult{}, errors.Join(fmt.Errorf("persist harvested helper delta: %w", err), updateErr)
	}
	if proposalErr := validateHelperProposals(delta, result.Response.CandidateFiles); proposalErr != nil {
		requestRecord.State, requestRecord.Error = "failed", boundedError(errors.Join(runErr, proposalErr))
		journalErr := s.appendHelperRecord(journal, requestRecord)
		updateErr := s.updateRequest(ctx, requestIndex, reservation, "failed", result.Receipt, "")
		return codexappserver.ToolResult{}, errors.Join(proposalErr, runErr, journalErr, updateErr)
	}
	if runErr != nil || result.Receipt.Outcome != agentexec.OutcomeProposed {
		requestRecord.State = "failed"
		requestRecord.Error = boundedError(runErr)
		journalErr := s.appendHelperRecord(journal, requestRecord)
		updateErr := s.updateRequest(ctx, requestIndex, reservation, lifecycleHostState(result.Receipt.Lifecycle), result.Receipt, "")
		return codexappserver.ToolResult{}, errors.Join(errors.New("helper child did not complete with a proposed result"), runErr, journalErr, updateErr)
	}
	if err := applyHelperDelta(ctx, parentHandle.Workspace, baseline, delta); err != nil {
		requestRecord.State, requestRecord.Error = "failed", boundedError(err)
		journalErr := s.appendHelperRecord(journal, requestRecord)
		updateErr := s.updateRequest(ctx, requestIndex, reservation, lifecycleHostState(result.Receipt.Lifecycle), result.Receipt, "")
		return codexappserver.ToolResult{}, errors.Join(fmt.Errorf("apply observed helper delta: %w", err), journalErr, updateErr)
	}
	requestRecord.State = "delta-applied"
	var persistenceErrs []error
	if err := s.appendHelperRecord(journal, requestRecord); err != nil {
		persistenceErrs = append(persistenceErrs, fmt.Errorf("persist applied helper delta: %w", err))
	}
	// Keep the durable Host reservation blocking while the terminal child
	// workspace is being closed. A completed provider turn is not equivalent to
	// a completed Host operation until cleanup has succeeded.
	if err := s.updateRequest(ctx, requestIndex, reservation, "unknown", result.Receipt, ""); err != nil {
		persistenceErrs = append(persistenceErrs, err)
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 30*time.Second)
	closeErr := s.service.Close(closeCtx, handle)
	closeCancel()
	if closeErr != nil {
		requestRecord.State, requestRecord.Error = "cleanup-pending", boundedError(closeErr)
		journalErr := s.appendHelperRecord(journal, requestRecord)
		stateErr := s.updateRequest(ctx, requestIndex, reservation, "unknown", result.Receipt, "")
		return codexappserver.ToolResult{}, errors.Join(errors.New("helper delta was applied but terminal workspace cleanup is pending"), closeErr,
			errors.Join(persistenceErrs...), journalErr, stateErr)
	}
	requestRecord.State, requestRecord.Error = "closed", ""
	if err := s.appendHelperRecord(journal, requestRecord); err != nil {
		persistenceErrs = append(persistenceErrs, fmt.Errorf("persist closed helper workspace: %w", err))
	}
	state := "completed"
	if len(persistenceErrs) != 0 {
		state = "unknown"
	}
	var completionErr error
	if state == "completed" {
		completionErr = s.completeRequest(ctx, requestIndex, reservation, result.Receipt, helperDeliveryFromDelta(attempt.Request.RequestID, args, delta))
	} else {
		completionErr = s.updateRequest(ctx, requestIndex, reservation, state, result.Receipt, "")
	}
	if completionErr != nil {
		persistenceErrs = append(persistenceErrs, completionErr)
		if state != "unknown" {
			if unknownErr := s.updateRequest(ctx, requestIndex, reservation, "unknown", result.Receipt, ""); unknownErr != nil {
				persistenceErrs = append(persistenceErrs, unknownErr)
			}
		}
	}
	if len(persistenceErrs) != 0 {
		return codexappserver.ToolResult{}, errors.Join(errors.New("helper workspace closed but Host completion persistence is uncertain"), errors.Join(persistenceErrs...))
	}
	text := fmt.Sprintf("Helper completed. task=%s status=proposed paths=%s delta=%s", request.TaskID, strings.Join(deltaPaths(delta), ","), delta.Digest)
	return codexappserver.ToolResult{Success: true, Text: truncateHelperText(text, 4096)}, nil
}

func helperDeliveryFromDelta(requestID string, args helperToolArgs, delta projectworkspace.Delta) HelperDelivery {
	delivery := HelperDelivery{State: "applied-and-closed", RequestID: requestID, Task: args.Task,
		RequestedPaths: append([]string(nil), args.Paths...), DeltaDigest: delta.Digest,
		Changes: make([]HelperDeliveryChange, 0, len(delta.Changes))}
	for _, change := range delta.Changes {
		fact := HelperDeliveryChange{Kind: string(change.Kind), Path: change.Path, OldPath: change.OldPath, Mode: protocolMode(change.Mode)}
		if change.Kind != projectworkspace.ChangeDelete {
			fact.ContentDigest = rawContentDigest(change.Content)
		}
		delivery.Changes = append(delivery.Changes, fact)
	}
	return delivery
}

func (s *HelperSession) childOptions(reservation HelperReservation, hostRequest agentexec.RoleStartRequest) codexappserver.Options {
	options := s.options.ChildOptions
	specs := make([]codexappserver.DynamicTool, 0, len(options.DynamicTools))
	for _, spec := range options.DynamicTools {
		if spec.Name != HelperToolName {
			specs = append(specs, spec)
		}
	}
	options.DynamicTools = specs
	if len(specs) == 0 {
		options.HandleToolCall, options.MaxToolCalls, options.ToolTimeout = nil, 0, 0
	}
	// Child sessions get their own journal and cannot bind/overwrite the parent
	// recovery handle. The Host helper reservation is attached exactly once here.
	options.OnHandle = nil
	options.OnEvent = nil
	options.BeforeStart = func(ctx context.Context, childRoot agentexec.RoleStartRequest) error {
		if reservation == nil {
			return errors.New("helper Host reservation is unavailable")
		}
		if err := reservation.AttachProtocolStart(ctx, childRoot); err != nil {
			return err
		}
		s.mu.Lock()
		s.protocolStarts[hostRequest.RequestID] = true
		s.mu.Unlock()
		return nil
	}
	return options
}

func (s *HelperSession) protocolStartSeen(requestID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.protocolStarts[requestID]
}

func (s *HelperSession) childRequest(args helperToolArgs) (agentexec.Request, error) {
	ctxValue := helperContext{Guidance: "You are an independent helper for task within allowedWritePaths. ParentContext is read-only task context, not authority to perform the parent Manager mandate or schedule its children. Use ordinary native file, shell and test tools for this bounded task. Do not edit the canonical control plane, dispatch model-declared Managers, or write outside the explicit helper scope. Return the bound executor response; candidateFiles, if supplied, assert only exact bytes actually written in this workspace. Do not claim verification beyond your observations.", Kind: "projectrun-helper/v1", HelperDepth: 1, HelperAccounting: "partial",
		ManagerID: s.options.ParentScope.ManagerID, Task: args.Task, AllowedWritePaths: append([]string(nil), args.Paths...),
		ExcludedWritePaths:     append([]string(nil), s.options.ParentScope.ExcludedWritePaths...),
		ActiveResponsibilities: cloneHelperResponsibilities(s.options.ParentScope.ActiveResponsibilities),
		ParentContext:          append(json.RawMessage(nil), s.options.ParentRequest.Context...)}
	contextBytes, err := json.Marshal(ctxValue)
	if err != nil {
		return agentexec.Request{}, err
	}
	request := s.options.ParentRequest
	request.Role = agentexec.RoleExecutor
	request.SourceRevision = s.options.ParentWorkspace.BaseSHA
	request.Context = contextBytes
	request.ScopeIDs = append([]string(nil), s.options.ParentRequest.ScopeIDs...)
	request.PolicyIDs = append([]string(nil), s.options.ParentRequest.PolicyIDs...)
	request.Artifacts = cloneHelperArtifacts(s.options.ParentRequest.Artifacts)
	return request, nil
}

func (s *HelperSession) recordRequest(request agentexec.RoleStartRequest) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, request)
	return len(s.requests) - 1
}

func (s *HelperSession) failRequest(ctx context.Context, index int, reservation HelperReservation, handle *projectworkspace.Handle, callID string, attempt HelperStartAttempt, state string, cause error, journals ...*nativeJournal) (codexappserver.ToolResult, error) {
	var journalErr, closeErr error
	if handle != nil {
		// The parent reservation remains durable even if workspace preparation or
		// journal creation fails. No provider was launched, so cleanup is safe.
		var journal *nativeJournal
		if len(journals) != 0 {
			journal = journals[0]
		}
		if journal != nil {
			journalErr = s.appendHelperRecord(journal, helperJournalRecord{RequestID: attempt.Request.RequestID, ToolCallID: callID,
				Task: attempt.Task, Paths: attempt.Paths, State: "cleanup-pending", Handle: handle, RecordedAt: time.Now().UTC()})
		}
		closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		closeErr = s.service.Close(closeCtx, *handle)
		cancel()
		if closeErr != nil {
			state = "unknown"
			if journal != nil {
				journalErr = errors.Join(journalErr, s.appendHelperRecord(journal, helperJournalRecord{RequestID: attempt.Request.RequestID, ToolCallID: callID,
					Task: attempt.Task, Paths: attempt.Paths, State: "cleanup-pending", Error: boundedError(closeErr), Handle: handle, RecordedAt: time.Now().UTC()}))
			}
		} else if journal != nil {
			journalErr = errors.Join(journalErr, s.appendHelperRecord(journal, helperJournalRecord{RequestID: attempt.Request.RequestID, ToolCallID: callID,
				Task: attempt.Task, Paths: attempt.Paths, State: "closed", Handle: handle, RecordedAt: time.Now().UTC()}))
		}
	}
	updateErr := s.updateRequest(ctx, index, reservation, state, agentexec.Receipt{}, "")
	return codexappserver.ToolResult{}, errors.Join(cause, closeErr, journalErr, updateErr)
}

func (s *HelperSession) updateRequest(ctx context.Context, index int, reservation HelperReservation, state string, receipt agentexec.Receipt, errorText string) error {
	s.mu.Lock()
	if index < 0 || index >= len(s.requests) {
		s.mu.Unlock()
		return errors.New("helper reservation index is invalid")
	}
	request := s.requests[index]
	request.State = state
	if receipt.Lifecycle != nil {
		request.SessionID = receipt.Lifecycle.SessionID
		if request.State != "unknown" {
			if lifecycleState := lifecycleHostState(receipt.Lifecycle); lifecycleState != "unknown" {
				request.State = lifecycleState
			}
		}
	}
	s.requests[index] = request
	s.mu.Unlock()
	if reservation != nil {
		if err := reservation.Update(ctx, request); err != nil {
			return fmt.Errorf("persist Host helper request update: %w", err)
		}
	}
	return nil
}

func (s *HelperSession) completeRequest(ctx context.Context, index int, reservation HelperReservation, receipt agentexec.Receipt, delivery HelperDelivery) error {
	s.mu.Lock()
	if index < 0 || index >= len(s.requests) {
		s.mu.Unlock()
		return errors.New("helper reservation index is invalid")
	}
	request := s.requests[index]
	request.State = "completed"
	if receipt.Lifecycle != nil {
		request.SessionID = receipt.Lifecycle.SessionID
	}
	s.requests[index] = request
	s.mu.Unlock()
	if reservation == nil {
		return errors.New("completed helper delivery has no durable Host reservation")
	}
	if err := reservation.CompleteDelivery(ctx, request, delivery); err != nil {
		return fmt.Errorf("persist Host helper delivery: %w", err)
	}
	return nil
}

func (s *HelperSession) storeReceipt(requestID string, receipt agentexec.Receipt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts = append(s.receipts, cloneHelperReceipt(receipt))
	s.receiptByRequest[requestID] = true
}

func (s *HelperSession) requestHasReceipt(requestID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.receiptByRequest[requestID]
}

func (s *HelperSession) appendHelperRecord(journal *nativeJournal, record helperJournalRecord) error {
	return journal.appendJSONLine("helpers.jsonl", record)
}

func decodeHelperArgs(raw json.RawMessage) (helperToolArgs, error) {
	var args helperToolArgs
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&args); err != nil {
		return args, fmt.Errorf("decode Host helper arguments: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return args, errors.New("Host helper arguments must contain exactly one object")
	}
	return args, nil
}

func validateHelperArgs(args helperToolArgs, parent HelperParentScope) error {
	if strings.TrimSpace(args.Task) == "" || len(args.Task) > 8192 || !utf8.ValidString(args.Task) || len(args.Paths) == 0 || len(args.Paths) > 64 {
		return errors.New("helper requires a bounded task and at least one bounded path scope")
	}
	seen := map[string]string{}
	for _, requested := range args.Paths {
		base := strings.TrimSuffix(requested, "/")
		if strings.HasSuffix(base, "/") {
			return fmt.Errorf("helper path scope %q has repeated trailing separators", requested)
		}
		if !safeRepoPath(base) || forbiddenRuntimePath(base) {
			return fmt.Errorf("helper path scope %q is unsafe or reserved", requested)
		}
		key := strings.ToLower(requested)
		if prior, exists := seen[key]; exists {
			if prior != requested {
				return fmt.Errorf("helper path scopes %q and %q collide by case", prior, requested)
			}
			return fmt.Errorf("duplicate helper path scope %q", requested)
		}
		seen[key] = requested
		allowed := false
		for _, scope := range parent.AllowedWritePaths {
			if helperScopeContains(scope, requested) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("helper path scope %q exceeds the parent Manager's allowed paths", requested)
		}
		for _, deny := range parent.ExcludedWritePaths {
			if helperScopesOverlap(requested, deny) {
				return fmt.Errorf("helper path scope %q overlaps excluded path %q", requested, deny)
			}
		}
		for _, responsibility := range parent.ActiveResponsibilities {
			if responsibility.ManagerID == "" || responsibility.ManagerID == parent.ManagerID {
				continue
			}
			for _, owned := range responsibility.Owns {
				if helperScopesOverlap(requested, owned) {
					return fmt.Errorf("helper path scope %q overlaps active Manager %q ownership", requested, responsibility.ManagerID)
				}
			}
		}
	}
	return nil
}

func helperScopeContains(parent, child string) bool {
	parentDir := strings.HasSuffix(parent, "/")
	parent = strings.TrimSuffix(parent, "/")
	childBase := strings.TrimSuffix(child, "/")
	if strings.EqualFold(parent, childBase) {
		return parentDir || !strings.HasSuffix(child, "/")
	}
	return parentDir && strings.HasPrefix(strings.ToLower(childBase), strings.ToLower(parent)+"/")
}

func helperScopesOverlap(left, right string) bool {
	left, right = strings.TrimSuffix(left, "/"), strings.TrimSuffix(right, "/")
	if strings.EqualFold(left, right) {
		return true
	}
	if strings.HasPrefix(strings.ToLower(right), strings.ToLower(left)+"/") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(left), strings.ToLower(right)+"/")
}

func validateHelperDeltaScope(delta projectworkspace.Delta, requested []string, parent HelperParentScope) error {
	for _, change := range delta.Changes {
		for _, path := range []string{change.Path, change.OldPath} {
			if path == "" {
				continue
			}
			allowed := false
			for _, scope := range requested {
				if helperPathAllowed(scope, path) {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("helper child changed %q outside its exact requested scopes", path)
			}
			for _, scope := range parent.ExcludedWritePaths {
				if helperPathAllowed(scope, path) || helperScopesOverlap(path, scope) {
					return fmt.Errorf("helper child changed excluded path %q", path)
				}
			}
			for _, responsibility := range parent.ActiveResponsibilities {
				if responsibility.ManagerID == "" || responsibility.ManagerID == parent.ManagerID {
					continue
				}
				for _, owned := range responsibility.Owns {
					if helperPathAllowed(owned, path) || helperScopesOverlap(path, owned) {
						return fmt.Errorf("helper child changed path %q owned by active Manager %q", path, responsibility.ManagerID)
					}
				}
			}
		}
	}
	return nil
}

func helperPathAllowed(scope, file string) bool {
	if strings.HasSuffix(scope, "/") {
		base := strings.TrimSuffix(scope, "/")
		return strings.HasPrefix(strings.ToLower(file), strings.ToLower(base)+"/")
	}
	return strings.EqualFold(scope, file)
}

func helperWorkspaceLimits(limits Limits) projectworkspace.Limits {
	return projectworkspace.Limits{MaxFiles: 10000, MaxFileBytes: int(limits.MaxCandidateFileBytes), MaxTotalBytes: int(limits.MaxCandidateBytes)}
}

func sameHelperDeltaBinding(actual, normalized projectworkspace.Delta, request projectworkspace.Request, handle projectworkspace.Handle) bool {
	return actual.RepositoryIdentity == request.RepositoryIdentity && actual.BaseSHA == request.BaseSHA &&
		actual.OverlayDigest == request.OverlayDigest && actual.TaskID == request.TaskID && actual.BaseDigest == handle.BaseDigest &&
		actual.Digest == normalized.Digest
}

func validateHelperProposals(delta projectworkspace.Delta, proposals []agentexec.CandidateFile) error {
	writes := map[string]projectworkspace.Change{}
	for _, change := range delta.Changes {
		if change.Kind == projectworkspace.ChangeAdd || change.Kind == projectworkspace.ChangeModify || change.Kind == projectworkspace.ChangeRename {
			writes[change.Path] = change
		}
	}
	seen := map[string]string{}
	for _, proposal := range proposals {
		key := strings.ToLower(proposal.Path)
		if prior, ok := seen[key]; ok {
			if prior != proposal.Path {
				return fmt.Errorf("helper response contains case-alias writes %q and %q", prior, proposal.Path)
			}
			return fmt.Errorf("helper response repeats write %q", proposal.Path)
		}
		seen[key] = proposal.Path
		change, ok := writes[proposal.Path]
		if !ok {
			return fmt.Errorf("helper response write %q was not observed in child delta", proposal.Path)
		}
		mode := workspaceSnapshotMode(change.Mode)
		if mode == "" || snapshotMode(proposal.Mode) != mode || !bytes.Equal([]byte(proposal.Content), change.Content) {
			return fmt.Errorf("helper response write %q contradicts observed child bytes or mode", proposal.Path)
		}
	}
	return nil
}

func applyHelperDelta(ctx context.Context, workspace projectworkspace.Handle, baseline projectworkspace.Binding, delta projectworkspace.Delta) error {
	current, err := projectworkspace.InspectRepository(ctx, workspace.CWD, workspace.BaseSHA)
	if err != nil || current != baseline {
		return fmt.Errorf("parent workspace changed while helper was running: %w", errors.Join(err, ErrStale))
	}
	root, err := os.OpenRoot(workspace.CWD)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, change := range delta.Changes {
		if err := checkHelperPathNoReparse(root, change.Path, true); err != nil {
			return err
		}
		if change.Kind == projectworkspace.ChangeRename {
			if err := checkHelperPathNoReparse(root, change.OldPath, false); err != nil {
				return err
			}
		}
	}
	// Preflight the entire delta before changing the parent. The workspace
	// service supplies operations relative to its exact captured inventory.
	for _, change := range delta.Changes {
		if err := preflightHelperChange(root, change); err != nil {
			return err
		}
	}
	deletes := make([]projectworkspace.Change, 0, len(delta.Changes))
	writes := make([]projectworkspace.Change, 0, len(delta.Changes))
	for _, change := range delta.Changes {
		if change.Kind == projectworkspace.ChangeDelete || change.Kind == projectworkspace.ChangeRename {
			deletes = append(deletes, change)
		}
		if change.Kind != projectworkspace.ChangeDelete {
			writes = append(writes, change)
		}
	}
	sort.Slice(deletes, func(i, j int) bool { return strings.Count(deletes[i].Path, "/") > strings.Count(deletes[j].Path, "/") })
	for _, change := range deletes {
		path := change.Path
		if change.Kind == projectworkspace.ChangeRename {
			path = change.OldPath
		}
		if err := root.Remove(filepath.FromSlash(path)); err != nil {
			return fmt.Errorf("remove helper source %s: %w", path, err)
		}
	}
	for _, change := range writes {
		if err := writeHelperFile(root, change); err != nil {
			return err
		}
	}
	return nil
}

func preflightHelperChange(root *os.Root, change projectworkspace.Change) error {
	path := filepath.FromSlash(change.Path)
	info, err := root.Lstat(path)
	switch change.Kind {
	case projectworkspace.ChangeAdd:
		if err == nil && !info.IsDir() {
			return fmt.Errorf("helper add target already exists: %s", change.Path)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	case projectworkspace.ChangeModify, projectworkspace.ChangeDelete:
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("helper %s source is not a regular file: %s", change.Kind, change.Path)
		}
	case projectworkspace.ChangeRename:
		old, oldErr := root.Lstat(filepath.FromSlash(change.OldPath))
		if oldErr != nil || !old.Mode().IsRegular() || old.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("helper rename source is not a regular file: %s", change.OldPath)
		}
		if err == nil && !info.IsDir() {
			return fmt.Errorf("helper rename target already exists: %s", change.Path)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	default:
		return fmt.Errorf("unsupported helper delta operation %q", change.Kind)
	}
	return nil
}

func writeHelperFile(root *os.Root, change projectworkspace.Change) error {
	name := filepath.FromSlash(change.Path)
	if err := checkHelperPathNoReparse(root, change.Path, true); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	if info, err := root.Lstat(name); err == nil {
		if change.Kind == projectworkspace.ChangeModify {
			if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("helper modify target is not a regular file: %s", change.Path)
			}
		} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("helper write target became occupied: %s", change.Path)
		}
		if err := root.Remove(name); err != nil {
			return fmt.Errorf("remove existing helper target %s: %w", change.Path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create helper output %s: %w", change.Path, err)
	}
	for content := change.Content; len(content) > 0; {
		n, writeErr := file.Write(content)
		if writeErr != nil {
			file.Close()
			_ = root.Remove(name)
			return fmt.Errorf("write helper output %s: %w", change.Path, writeErr)
		}
		if n == 0 {
			file.Close()
			_ = root.Remove(name)
			return io.ErrShortWrite
		}
		content = content[n:]
	}
	mode := os.FileMode(0644)
	if change.Mode == "100755" {
		mode = 0755
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		_ = root.Remove(name)
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		_ = root.Remove(name)
		return err
	}
	return file.Close()
}

func checkHelperPathNoReparse(root *os.Root, value string, allowMissing bool) error {
	if !safeRepoPath(value) || forbiddenRuntimePath(value) {
		return fmt.Errorf("helper path is unsafe or control-plane reserved: %q", value)
	}
	parts := strings.Split(filepath.FromSlash(value), string(filepath.Separator))
	for i := 1; i < len(parts); i++ {
		prefix := filepath.Join(parts[:i]...)
		info, err := root.Lstat(prefix)
		if errors.Is(err, os.ErrNotExist) && allowMissing {
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("helper path parent is missing or reparsed: %s", prefix)
		}
	}
	info, err := root.Lstat(filepath.FromSlash(value))
	if errors.Is(err, os.ErrNotExist) && allowMissing {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("helper path is a reparse point: %s", value)
	}
	return nil
}

func lifecycleRootState(lifecycle *agentexec.Lifecycle) string {
	if lifecycle == nil || len(lifecycle.StartRequests) == 0 {
		return "unknown"
	}
	return lifecycle.StartRequests[0].State
}

func lifecycleHostState(lifecycle *agentexec.Lifecycle) string {
	if lifecycle == nil || !terminalWorkspaceState(lifecycle.State) {
		return "unknown"
	}
	return lifecycle.State
}

func helperInvocationTerminal(lifecycle *agentexec.Lifecycle) bool {
	if lifecycle == nil || lifecycle.Provider != TransportCodexAppServer || lifecycle.SessionID == "" || lifecycle.TurnID == "" || !terminalWorkspaceState(lifecycle.State) || len(lifecycle.StartRequests) == 0 {
		return false
	}
	// The transport's root request remains in the `started` state after its
	// turn completes. The enclosing lifecycle state is authoritative for that
	// root; every tracked nested request must independently be terminal.
	for index, request := range lifecycle.StartRequests {
		if index == 0 {
			continue
		}
		if !terminalWorkspaceState(request.State) {
			return false
		}
	}
	return true
}

func boundedError(err error) string {
	if err == nil {
		return ""
	}
	return truncateHelperText(err.Error(), 2048)
}

func truncateHelperText(text string, maximum int) string {
	if len(text) <= maximum {
		return text
	}
	text = strings.ToValidUTF8(text[:maximum], "�")
	return text
}

func deltaPaths(delta projectworkspace.Delta) []string {
	paths := make([]string, 0, len(delta.Changes)*2)
	for _, change := range delta.Changes {
		paths = append(paths, change.Path)
		if change.OldPath != "" {
			paths = append(paths, change.OldPath)
		}
	}
	sort.Strings(paths)
	return paths
}

func cloneHelperResponsibilities(values []HelperResponsibility) []HelperResponsibility {
	clone := make([]HelperResponsibility, len(values))
	for i, value := range values {
		clone[i] = value
		clone[i].Owns = append([]string(nil), value.Owns...)
	}
	return clone
}

func cloneHelperArtifacts(values []agentexec.Artifact) []agentexec.Artifact {
	clone := make([]agentexec.Artifact, len(values))
	for i, value := range values {
		clone[i] = value
		clone[i].Content = append([]byte(nil), value.Content...)
	}
	return clone
}

func cloneHelperReceipt(receipt agentexec.Receipt) agentexec.Receipt {
	clone := receipt
	if receipt.Usage != nil {
		usage := *receipt.Usage
		clone.Usage = &usage
	}
	if receipt.Lifecycle != nil {
		life := *receipt.Lifecycle
		life.StartRequests = append([]agentexec.RoleStartRequest(nil), receipt.Lifecycle.StartRequests...)
		if receipt.Lifecycle.Effective != nil {
			effective := *receipt.Lifecycle.Effective
			life.Effective = &effective
		}
		clone.Lifecycle = &life
	}
	if receipt.NativeWork != nil {
		work := *receipt.NativeWork
		work.ChangedPaths = append([]string(nil), receipt.NativeWork.ChangedPaths...)
		clone.NativeWork = &work
	}
	return clone
}
