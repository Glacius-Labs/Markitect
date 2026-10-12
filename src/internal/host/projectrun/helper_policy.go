package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

// NativeHelperPolicy is the method-owned policy/session factory supplied to
// the transport executor. Its stable tool definition is available before a
// run-specific durable budget has been bound.
type NativeHelperPolicy struct {
	Workspaces       projectworkspace.Service
	Limits           projectconfig.Limits
	MaxStartRequests int
	Reserve          HelperReserveFunc
}

func (NativeHelperPolicy) ToolOptions(config agentexec.Config, options codexappserver.Options) (codexappserver.Options, error) {
	settings, err := projectexecutor.DecodeAppServerSettings(config.TransportConfig)
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

func (NativeHelperPolicy) Eligible(request agentexec.Request) bool {
	if request.Role != agentexec.RoleExecutor {
		return false
	}
	var task struct {
		Kind              string   `json:"kind"`
		ManagerID         string   `json:"managerId"`
		Phase             string   `json:"phase"`
		AllowedWritePaths []string `json:"allowedWritePaths"`
	}
	if json.Unmarshal(request.Context, &task) != nil || task.Kind != "projectrun-task/v1" ||
		strings.TrimSpace(task.ManagerID) == "" || strings.TrimSpace(task.ManagerID) != task.ManagerID ||
		(task.Phase != "work" && task.Phase != "integrate") {
		return false
	}
	for _, path := range task.AllowedWritePaths {
		if strings.TrimSpace(path) != "" {
			return true
		}
	}
	return false
}

func (p NativeHelperPolicy) BindRun(config agentexec.Config, request agentexec.Request, runOptions agentexec.RunOptions, options codexappserver.Options) (projectexecutor.BoundHelper, error) {
	options, err := p.ToolOptions(config, options)
	if err != nil {
		return projectexecutor.BoundHelper{}, err
	}
	if !helperToolPresent(options.DynamicTools) {
		return projectexecutor.BoundHelper{Options: options}, nil
	}
	if p.Reserve == nil {
		return projectexecutor.BoundHelper{Options: rejectUnboundHelper(options)}, nil
	}
	var task nativeTaskContext
	if err := json.Unmarshal(request.Context, &task); err != nil || task.ManagerID == "" || task.Phase == "" {
		return projectexecutor.BoundHelper{}, errors.New("Host helpers require valid typed parent context")
	}
	if runOptions.Workspace == nil || runOptions.PrivateLogDirectory == "" || p.Workspaces == nil {
		return projectexecutor.BoundHelper{}, errors.New("Manager helpers require an owned parent workspace, private journal and candidate workspace service")
	}
	parentWorkspace := *runOptions.Workspace
	parentScope := HelperParentScope{ManagerID: task.ManagerID, AllowedWritePaths: task.AllowedWritePaths,
		ExcludedWritePaths: task.ExcludedWritePaths, ActiveResponsibilities: task.ActiveResponsibilities}
	maxStarts := p.MaxStartRequests
	if maxStarts <= 0 {
		maxStarts = 256
	}
	if options.MaxToolCalls <= 0 {
		options.MaxToolCalls = maxStarts
		if options.MaxToolCalls > 256 {
			options.MaxToolCalls = 256
		}
	}
	if options.ToolTimeout <= 0 {
		options.ToolTimeout = config.Timeout
	}
	session, err := NewHelperSession(HelperSessionOptions{
		ParentWorkspace: parentWorkspace, ParentConfig: config, ParentRequest: request, ParentScope: parentScope,
		Workspaces: p.Workspaces, Limits: p.Limits,
		PrivateLogDirectory: runOptions.PrivateLogDirectory, ChildOptions: options, Reserve: p.Reserve,
		MaxStartRequests: maxStarts,
	})
	if err != nil {
		return projectexecutor.BoundHelper{}, err
	}
	previousHandler := options.HandleToolCall
	options.HandleToolCall = func(ctx context.Context, call codexappserver.ToolCall) (codexappserver.ToolResult, error) {
		if call.Tool == HelperToolName {
			return session.HandleToolCall(ctx, call)
		}
		if previousHandler == nil {
			return codexappserver.ToolResult{}, fmt.Errorf("no Host handler for native tool %q", call.Tool)
		}
		return previousHandler(ctx, call)
	}
	previousOnHandle := options.OnHandle
	options.OnHandle = func(ctx context.Context, handle codexappserver.RecoveryHandle) error {
		if err := session.BindHandle(handle); err != nil {
			return err
		}
		if previousOnHandle != nil {
			return previousOnHandle(ctx, handle)
		}
		return nil
	}
	return projectexecutor.BoundHelper{Options: options, Merge: func(receipt *agentexec.Receipt) {
		mergeHelperAccounting(receipt, session)
	}}, nil
}

type nativeTaskContext struct {
	Kind                   string                 `json:"kind"`
	ManagerID              string                 `json:"managerId"`
	Phase                  string                 `json:"phase"`
	AllowedWritePaths      []string               `json:"allowedWritePaths"`
	ExcludedWritePaths     []string               `json:"excludedWritePaths"`
	ActiveResponsibilities []HelperResponsibility `json:"activeResponsibilities"`
}

func (NativeHelperPolicy) BindRecovery(config agentexec.Config, request agentexec.Request, options codexappserver.Options) (codexappserver.Options, error) {
	options, err := (NativeHelperPolicy{}).ToolOptions(config, options)
	if err != nil {
		return codexappserver.Options{}, err
	}
	if helperToolPresent(options.DynamicTools) && (NativeHelperPolicy{}).Eligible(request) {
		return rejectUnboundHelper(options), nil
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
		lifecycle = &agentexec.Lifecycle{Provider: projectconfig.TransportCodexAppServer, State: "unknown", Accounting: "partial"}
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
			if index == 0 {
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
