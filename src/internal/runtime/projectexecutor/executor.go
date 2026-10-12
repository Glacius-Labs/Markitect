package projectexecutor

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

const HelperToolName = "markitect_start_helper"

type Executor struct {
	appServerOptions codexappserver.Options
	helperPolicy     HelperPolicy
}

func New(options codexappserver.Options, helperPolicy HelperPolicy) *Executor {
	return &Executor{appServerOptions: options, helperPolicy: helperPolicy}
}

// NewChild constructs a native child executor without the method helper tool.
// The provider's native helper policy remains part of the configured adapter.
func NewChild(options codexappserver.Options) *Executor {
	return New(withoutHelperTool(options), nil)
}

func (e *Executor) WithHelperPolicy(policy HelperPolicy) Invoker {
	clone := *e
	clone.helperPolicy = policy
	return &clone
}

func (e *Executor) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, runOptions agentexec.RunOptions) (agentexec.RunResult, error) {
	if config.Transport == projectconfig.TransportProcess {
		return agentexec.Run(ctx, config, request, runOptions)
	}
	if err := validateNativeInvocationConfig(config); err != nil {
		return agentexec.RunResult{}, err
	}
	options := e.appServerOptions
	var merge func(*agentexec.Receipt)
	if e.helperPolicy != nil {
		var err error
		options, err = e.helperPolicy.ToolOptions(config, options)
		if err != nil {
			return agentexec.RunResult{}, err
		}
		if e.helperPolicy.Eligible(request) {
			bound, err := e.helperPolicy.BindRun(config, request, runOptions, options)
			if err != nil {
				return agentexec.RunResult{}, err
			}
			options, merge = bound.Options, bound.Merge
		} else {
			options = withoutHelperTool(options)
		}
	}
	if runOptions.Workspace != nil && runOptions.PrivateLogDirectory != "" {
		journal, err := OpenNativeJournal(runOptions.PrivateLogDirectory, runOptions.Workspace.CWD, runOptions.Workspace.ID)
		if err != nil {
			return agentexec.RunResult{}, err
		}
		options = journal.WrapOptions(options)
	}
	adapter, err := newAdapter(config, options)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	result, runErr := adapter.Run(ctx, config, request, runOptions)
	if merge != nil {
		merge(&result.Receipt)
	}
	return result, runErr
}

// Recover inspects only the exact trusted dispatched turn. It cannot fall back
// to Run, bind a fresh helper session, or reserve another role start.
func (e *Executor) Recover(ctx context.Context, config agentexec.Config, handle codexappserver.RecoveryHandle, runOptions agentexec.RunOptions) (agentexec.RunResult, error) {
	if config.Transport != projectconfig.TransportCodexAppServer {
		return agentexec.RunResult{}, fmt.Errorf("recovery requires transport %q", projectconfig.TransportCodexAppServer)
	}
	if err := validateNativeInvocationConfig(config); err != nil {
		return agentexec.RunResult{}, err
	}
	if err := codexappserver.ValidateRecoveryProtocol(handle); err != nil {
		return agentexec.RunResult{}, err
	}
	if runOptions.Workspace == nil || *runOptions.Workspace != handle.Workspace || runOptions.PrivateLogDirectory == "" {
		return agentexec.RunResult{}, errors.New("native recovery requires its original owned workspace and private journal")
	}
	if handle.Invocation.APIVersion != agentexec.APIVersion || handle.Invocation.RunID == "" || handle.Invocation.Nonce == "" ||
		handle.Invocation.InputDigest == "" || !sameInvocationRequest(handle.Invocation.Request, handle.Invocation) ||
		handle.Invocation.Request.SourceRevision != handle.Workspace.BaseSHA || handle.ThreadID == "" ||
		handle.SessionID == "" || handle.TurnID == "" || !handle.TurnDispatched {
		return agentexec.RunResult{}, errors.New("native recovery requires the exact trusted dispatched turn and invocation")
	}
	options := e.appServerOptions
	if e.helperPolicy != nil {
		var err error
		options, err = e.helperPolicy.ToolOptions(config, options)
		if err != nil {
			return agentexec.RunResult{}, err
		}
		if e.helperPolicy.Eligible(handle.Invocation.Request) {
			options, err = e.helperPolicy.BindRecovery(config, handle.Invocation.Request, options)
			if err != nil {
				return agentexec.RunResult{}, err
			}
		} else {
			options = withoutHelperTool(options)
		}
	}
	adapter, err := newAdapter(config, options)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	fingerprint, err := adapter.Fingerprint(config)
	if err != nil || fingerprint != handle.Fingerprint {
		return agentexec.RunResult{}, errors.New("native recovery configuration does not match the trusted original handle")
	}
	journal, err := OpenNativeJournal(runOptions.PrivateLogDirectory, runOptions.Workspace.CWD, runOptions.Workspace.ID)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	if err := journal.BindRecoveryHandle(handle); err != nil {
		return agentexec.RunResult{}, err
	}
	adapter, err = newAdapter(config, journal.WrapOptions(options))
	if err != nil {
		return agentexec.RunResult{}, err
	}
	return adapter.Recover(ctx, config, handle)
}

func (e *Executor) Fingerprint(config agentexec.Config) (string, error) {
	if config.Transport == projectconfig.TransportProcess {
		return agentexec.Fingerprint(config)
	}
	if err := validateNativeInvocationConfig(config); err != nil {
		return "", err
	}
	options := e.appServerOptions
	if e.helperPolicy != nil {
		var err error
		options, err = e.helperPolicy.ToolOptions(config, options)
		if err != nil {
			return "", err
		}
	}
	adapter, err := newAdapter(config, options)
	if err != nil {
		return "", err
	}
	return adapter.Fingerprint(config)
}

// FingerprintForRequest binds the exact request-sensitive helper tool set.
func (e *Executor) FingerprintForRequest(config agentexec.Config, request agentexec.Request) (string, error) {
	if config.Transport == projectconfig.TransportProcess {
		return agentexec.Fingerprint(config)
	}
	if err := validateNativeInvocationConfig(config); err != nil {
		return "", err
	}
	options := e.appServerOptions
	if e.helperPolicy != nil {
		var err error
		options, err = e.helperPolicy.ToolOptions(config, options)
		if err != nil {
			return "", err
		}
		if !e.helperPolicy.Eligible(request) {
			options = withoutHelperTool(options)
		}
	}
	adapter, err := newAdapter(config, options)
	if err != nil {
		return "", err
	}
	return adapter.Fingerprint(config)
}

func newAdapter(config agentexec.Config, options codexappserver.Options) (*codexappserver.Adapter, error) {
	adapterConfig, err := appServerConfig(config)
	if err != nil {
		return nil, err
	}
	return codexappserver.NewAdapter(adapterConfig, options)
}

func withoutHelperTool(options codexappserver.Options) codexappserver.Options {
	filtered := make([]codexappserver.DynamicTool, 0, len(options.DynamicTools))
	for _, tool := range options.DynamicTools {
		if tool.Name != HelperToolName {
			filtered = append(filtered, tool)
		}
	}
	options.DynamicTools = filtered
	if len(filtered) == 0 {
		options.HandleToolCall, options.MaxToolCalls, options.ToolTimeout = nil, 0, 0
	}
	return options
}

func sameInvocationRequest(request agentexec.Request, invocation agentexec.Invocation) bool {
	expected, _, err := agentexec.PrepareInvocation(request)
	return err == nil && expected.InputDigest == invocation.InputDigest && reflect.DeepEqual(expected.Request, invocation.Request)
}
