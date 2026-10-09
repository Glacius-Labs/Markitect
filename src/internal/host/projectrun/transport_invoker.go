package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

// TransportInvoker routes a shared invocation to its configured product
// transport. Host hooks are passed to the App Server adapter at construction;
// the process transport remains the existing agentexec implementation.
type TransportInvoker struct {
	appServerOptions   codexappserver.Options
	helperHost         *TransportHelperHost
	suppressHostHelper bool
}

// TransportHelperHost supplies the per-run Host-owned seams for dynamic
// helper calls. Reserve is expected to persist and globally cap each request
// before returning a permit; the child protocol root attaches to that permit.
type TransportHelperHost struct {
	Workspaces       projectworkspace.Service
	Limits           Limits
	// MaxStartRequests bounds local Host tool attempts; Reserve enforces the
	// durable global run limit. Zero selects the supported ceiling of 256.
	MaxStartRequests int
	Reserve          HelperReserveFunc
}

func NewTransportInvoker(options codexappserver.Options) *TransportInvoker {
	return &TransportInvoker{appServerOptions: options}
}

// NewTransportInvokerWithoutHostHelpers is for depth-one child sessions. It
// retains the App Server's native helper policy but does not advertise the
// Host tool without a parent reservation scope.
func NewTransportInvokerWithoutHostHelpers(options codexappserver.Options) *TransportInvoker {
	return &TransportInvoker{appServerOptions: options, suppressHostHelper: true}
}

// NewTransportInvokerWithHelpers opts a native invoker into the stable Host
// helper tool and its durable reservation seam. Fingerprints include the tool
// specification; callback functions and journals remain outside the digest.
func NewTransportInvokerWithHelpers(options codexappserver.Options, helperHost TransportHelperHost) *TransportInvoker {
	return &TransportInvoker{appServerOptions: options, helperHost: &helperHost}
}

// WithHelperHost binds a per-run durable reservation callback without changing
// the static App Server options or their fingerprint-visible tool spec.
func (i *TransportInvoker) WithHelperHost(helperHost TransportHelperHost) Invoker {
	if i == nil {
		return NewTransportInvokerWithHelpers(codexappserver.Options{}, helperHost)
	}
	clone := *i
	clone.helperHost = &helperHost
	return &clone
}

func (i *TransportInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	switch config.Transport {
	case TransportProcess:
		return (ProcessInvoker{}).Run(ctx, config, request, options)
	case TransportCodexAppServer:
		appOptions := i.appServerOptions
		var helperSession *HelperSession
		var err error
		if !i.suppressHostHelper {
			appOptions, err = i.optionsWithHelperSpec(config, appOptions)
			if err != nil {
				return agentexec.RunResult{}, err
			}
		}
		if i.helperHost != nil && !i.suppressHostHelper {
			appOptions, helperSession, err = i.optionsWithHelper(config, request, options)
			if err != nil {
				return agentexec.RunResult{}, err
			}
		} else if !i.suppressHostHelper && helperToolPresent(appOptions.DynamicTools) {
			appOptions = rejectUnboundHelper(appOptions)
		}
		if options.Workspace != nil && options.PrivateLogDirectory != "" {
			journal, err := newNativeJournal(options.PrivateLogDirectory, options.Workspace.CWD, options.Workspace.ID)
			if err != nil {
				return agentexec.RunResult{}, err
			}
			appOptions = journal.wrapOptions(appOptions)
		}
		adapter, err := i.appServerAdapterWithOptions(config, appOptions)
		if err != nil {
			return agentexec.RunResult{}, err
		}
		result, runErr := adapter.Run(ctx, config, request, options)
		if helperSession != nil {
			mergeHelperAccounting(&result.Receipt, helperSession)
		}
		return result, runErr
	default:
		return agentexec.RunResult{}, fmt.Errorf("unsupported agent transport %q", config.Transport)
	}
}

// Recover inspects an already dispatched App Server turn. It never routes to
// the process adapter and does not reserve or start another role request.
func (i *TransportInvoker) Recover(ctx context.Context, config agentexec.Config, handle codexappserver.RecoveryHandle, options agentexec.RunOptions) (agentexec.RunResult, error) {
	if config.Transport != TransportCodexAppServer {
		return agentexec.RunResult{}, fmt.Errorf("recovery requires transport %q", TransportCodexAppServer)
	}
	appOptions := i.appServerOptions
	if options.Workspace == nil || options.Workspace.ID != handle.Workspace.ID || options.PrivateLogDirectory == "" {
		return agentexec.RunResult{}, errors.New("native recovery requires its original owned workspace and private journal")
	}
	journal, err := newNativeJournal(options.PrivateLogDirectory, options.Workspace.CWD, options.Workspace.ID)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	if !i.suppressHostHelper {
		appOptions, err = i.optionsWithHelperSpec(config, appOptions)
		if err != nil {
			return agentexec.RunResult{}, err
		}
	}
	if !i.suppressHostHelper && helperToolPresent(appOptions.DynamicTools) {
		appOptions = rejectUnboundHelper(appOptions)
	}
	appOptions = journal.wrapOptions(appOptions)
	adapter, err := i.appServerAdapterWithOptions(config, appOptions)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	return adapter.Recover(ctx, config, handle)
}

func (i *TransportInvoker) Fingerprint(config agentexec.Config) (string, error) {
	switch config.Transport {
	case TransportProcess:
		return (ProcessInvoker{}).Fingerprint(config)
	case TransportCodexAppServer:
		options := i.appServerOptions
		var err error
		if !i.suppressHostHelper {
			options, err = i.optionsWithHelperSpec(config, options)
			if err != nil {
				return "", err
			}
		}
		adapter, err := i.appServerAdapterWithOptions(config, options)
		if err != nil {
			return "", err
		}
		return adapter.Fingerprint(config)
	default:
		return "", fmt.Errorf("unsupported agent transport %q", config.Transport)
	}
}

type nativeTaskContext struct {
	Kind                   string                 `json:"kind"`
	ManagerID              string                 `json:"managerId"`
	Phase                  string                 `json:"phase"`
	AllowedWritePaths      []string               `json:"allowedWritePaths"`
	ExcludedWritePaths     []string               `json:"excludedWritePaths"`
	ActiveResponsibilities []HelperResponsibility `json:"activeResponsibilities"`
}

func helperParentMetadata(raw json.RawMessage) (string, string) {
	var value struct {
		Kind      string `json:"kind"`
		ManagerID string `json:"managerId"`
		Phase     string `json:"phase"`
	}
	_ = json.Unmarshal(raw, &value)
	if value.Kind != "projectrun-task/v1" {
		return "", ""
	}
	return value.ManagerID, value.Phase
}

func (i *TransportInvoker) optionsWithHelper(config agentexec.Config, request agentexec.Request, runOptions agentexec.RunOptions) (codexappserver.Options, *HelperSession, error) {
	settings, err := decodeAppServerSettings(config.TransportConfig)
	if err != nil {
		return codexappserver.Options{}, nil, err
	}
	if !settings.Helpers.Enabled {
		return i.appServerOptions, nil, nil
	}
	if i.helperHost.Reserve == nil {
		return codexappserver.Options{}, nil, errors.New("Host helpers require a durable reservation callback")
	}
	var taskContext nativeTaskContext
	if err := json.Unmarshal(request.Context, &taskContext); err != nil {
		return codexappserver.Options{}, nil, errors.New("Host helpers require valid typed parent context")
	}
	scope := HelperParentScope{}
	if taskContext.Kind == "projectrun-task/v1" && strings.TrimSpace(taskContext.ManagerID) != "" {
		if taskContext.Phase == "" {
			return codexappserver.Options{}, nil, errors.New("projectrun helper context has no phase")
		}
		if runOptions.Workspace == nil || runOptions.PrivateLogDirectory == "" || i.helperHost.Workspaces == nil {
			return codexappserver.Options{}, nil, errors.New("Manager helpers require an owned parent workspace, private journal and candidate workspace service")
		}
		scope = HelperParentScope{ManagerID: taskContext.ManagerID, AllowedWritePaths: taskContext.AllowedWritePaths,
			ExcludedWritePaths: taskContext.ExcludedWritePaths, ActiveResponsibilities: taskContext.ActiveResponsibilities}
	}
	parentWorkspace := projectworkspace.Handle{}
	if runOptions.Workspace != nil {
		parentWorkspace = *runOptions.Workspace
	}
	appOptions, err := i.optionsWithHelperSpec(config, i.appServerOptions)
	if err != nil {
		return codexappserver.Options{}, nil, err
	}
	maxStarts := i.helperHost.MaxStartRequests
	if maxStarts <= 0 {
		maxStarts = 256
	}
	if appOptions.MaxToolCalls <= 0 {
		appOptions.MaxToolCalls = maxStarts
		if appOptions.MaxToolCalls > 256 {
			appOptions.MaxToolCalls = 256
		}
	}
	if appOptions.ToolTimeout <= 0 {
		appOptions.ToolTimeout = config.Timeout
	}
	helperSession, err := NewHelperSession(HelperSessionOptions{
		ParentWorkspace: parentWorkspace, ParentConfig: config, ParentRequest: request,
		ParentScope: scope, Workspaces: i.helperHost.Workspaces, Limits: i.helperHost.Limits,
		PrivateLogDirectory: runOptions.PrivateLogDirectory, ChildOptions: appOptions,
		Reserve: i.helperHost.Reserve, MaxStartRequests: maxStarts,
	})
	if err != nil {
		return codexappserver.Options{}, nil, err
	}
	previousHandler := appOptions.HandleToolCall
	appOptions.HandleToolCall = func(ctx context.Context, call codexappserver.ToolCall) (codexappserver.ToolResult, error) {
		if call.Tool == HelperToolName {
			return helperSession.HandleToolCall(ctx, call)
		}
		if previousHandler == nil {
			return codexappserver.ToolResult{}, fmt.Errorf("no Host handler for native tool %q", call.Tool)
		}
		return previousHandler(ctx, call)
	}
	previousOnHandle := appOptions.OnHandle
	appOptions.OnHandle = func(ctx context.Context, handle codexappserver.RecoveryHandle) error {
		if err := helperSession.BindHandle(handle); err != nil {
			return err
		}
		if previousOnHandle != nil {
			return previousOnHandle(ctx, handle)
		}
		return nil
	}
	return appOptions, helperSession, nil
}

func (i *TransportInvoker) optionsWithHelperSpec(config agentexec.Config, options codexappserver.Options) (codexappserver.Options, error) {
	if i.suppressHostHelper {
		return options, nil
	}
	settings, err := decodeAppServerSettings(config.TransportConfig)
	if err != nil {
		return codexappserver.Options{}, err
	}
	if !settings.Helpers.Enabled {
		if helperToolPresent(options.DynamicTools) {
			return codexappserver.Options{}, errors.New("Host helper tool cannot be advertised when native helpers are disabled")
		}
		return options, nil
	}
	tool := HelperDynamicTool()
	found := false
	normalTools := 0
	for index, existing := range options.DynamicTools {
		if existing.Name != HelperToolName {
			normalTools++
			continue
		}
		if existing.Type != tool.Type || existing.Description != tool.Description || string(existing.InputSchema) != string(tool.InputSchema) {
			return codexappserver.Options{}, errors.New("native helper tool name is already bound to a different specification")
		}
		options.DynamicTools[index] = tool
		found = true
		break
	}
	if !found {
		options.DynamicTools = append(options.DynamicTools, tool)
	}
	if options.HandleToolCall == nil {
		if normalTools > 0 {
			return codexappserver.Options{}, errors.New("existing native tools have no Host call handler")
		}
		options.HandleToolCall = func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
			return codexappserver.ToolResult{}, errors.New("Host helper handler is unavailable")
		}
	}
	if options.MaxToolCalls == 0 {
		options.MaxToolCalls = 256
	}
	if options.ToolTimeout == 0 {
		options.ToolTimeout = config.Timeout
	}
	return options, nil
}

func helperToolPresent(tools []codexappserver.DynamicTool) bool {
	for _, tool := range tools {
		if tool.Name == HelperToolName {
			return true
		}
	}
	return false
}

func rejectUnboundHelper(options codexappserver.Options) codexappserver.Options {
	previous := options.HandleToolCall
	options.HandleToolCall = func(ctx context.Context, call codexappserver.ToolCall) (codexappserver.ToolResult, error) {
		if call.Tool == HelperToolName {
			return codexappserver.ToolResult{}, errors.New("Host helper reservation is unavailable for this invocation")
		}
		if previous == nil {
			return codexappserver.ToolResult{}, fmt.Errorf("no Host handler for native tool %q", call.Tool)
		}
		return previous(ctx, call)
	}
	return options
}

func mergeHelperAccounting(receipt *agentexec.Receipt, session *HelperSession) {
	requests := session.Requests()
	childReceipts := session.Receipts()
	if len(requests) == 0 {
		return
	}
	lifecycle := receipt.Lifecycle
	if lifecycle == nil {
		lifecycle = &agentexec.Lifecycle{Provider: TransportCodexAppServer, State: "unknown", Accounting: "partial"}
		receipt.Lifecycle = lifecycle
	}
	if lifecycle.StartRequests == nil {
		lifecycle.StartRequests = []agentexec.RoleStartRequest{}
	}
	seen := make(map[string]bool, len(lifecycle.StartRequests)+len(requests))
	for _, request := range lifecycle.StartRequests {
		if key := helperStartKey(request); key != "" {
			seen[key] = true
		}
	}
	for _, request := range requests {
		key := helperStartKey(request)
		if key == "" || !seen[key] {
			lifecycle.StartRequests = append(lifecycle.StartRequests, request)
			if key != "" {
				seen[key] = true
			}
		}
	}
	partial := lifecycle.Accounting != "complete"
	for _, request := range requests {
		if request.State == "unknown" || request.State == "requested" || request.State == "started" {
			partial = true
		}
	}
	for _, child := range childReceipts {
		if child.Lifecycle == nil {
			partial = true
			continue
		}
		if child.Lifecycle.Accounting != "complete" {
			partial = true
		}
		for index, nested := range child.Lifecycle.StartRequests {
			if index == 0 { // attached child executor root is already represented by its Host reservation
				continue
			}
			key := helperStartKey(nested)
			if key == "" || !seen[key] {
				lifecycle.StartRequests = append(lifecycle.StartRequests, nested)
				if key != "" {
					seen[key] = true
				}
			}
		}
	}
	for _, request := range requests {
		if session.protocolStartSeen(request.RequestID) && !session.requestHasReceipt(request.RequestID) {
			partial = true
		}
	}
	if partial && lifecycle.Accounting != "unavailable" {
		lifecycle.Accounting = "partial"
	}
}

func helperStartKey(request agentexec.RoleStartRequest) string {
	if request.RequestID == "" {
		return ""
	}
	return request.ParentSessionID + "\x00" + request.SessionID + "\x00" + request.RequestID + "\x00" + request.Role
}

func (i *TransportInvoker) appServerAdapter(config agentexec.Config) (*codexappserver.Adapter, error) {
	return i.appServerAdapterWithOptions(config, i.appServerOptions)
}

func (i *TransportInvoker) appServerAdapterWithOptions(config agentexec.Config, options codexappserver.Options) (*codexappserver.Adapter, error) {
	if config.Transport != TransportCodexAppServer {
		return nil, fmt.Errorf("App Server adapter requires transport %q", TransportCodexAppServer)
	}
	settings, err := decodeAppServerSettings(config.TransportConfig)
	if err != nil {
		return nil, err
	}
	adapterConfig := codexappserver.Config{
		Command: config.Command, ProviderVersion: config.ProviderVersion, Model: config.Model,
		ReasoningEffort: settings.ReasoningEffort, PermissionProfile: settings.PermissionProfile,
		WindowsSandboxBackend: settings.WindowsSandboxBackend,
		Helpers: codexappserver.HelperPolicy{
			Enabled: settings.Helpers.Enabled, MaxStartRequests: settings.Helpers.MaxStartRequests, MaxDepth: settings.Helpers.MaxDepth,
		},
		Timeout: config.Timeout, MaxEventBytes: settings.MaxEventBytes,
	}
	return codexappserver.NewAdapter(adapterConfig, options)
}

func decodeAppServerSettings(raw json.RawMessage) (AppServerSettings, error) {
	var settings AppServerSettings
	if len(raw) == 0 {
		return settings, errors.New("codex-app-server requires explicit transportConfig")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&settings); err != nil {
		return settings, fmt.Errorf("decode codex-app-server transportConfig: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return settings, errors.New("codex-app-server transportConfig must contain exactly one object")
	}
	return settings, nil
}
