package projectconfig

import (
	"encoding/json"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"sort"
	"strings"
	"time"
)

// ReadOnlyAgent selects the explicit transport for a Manager assessment or
// Brownfield model proposal. Native Manager bindings stay exclusive to scoped
// implementation tasks; their separately configured review binding serves a
// fresh read-only invocation, not a reused reviewer conversation.
func ReadOnlyAgent(config Runtime, managerID string) (Agent, error) {
	manager, ok := config.Agents[managerID]
	if !ok || strings.TrimSpace(managerID) == "" {
		return Agent{}, fmt.Errorf("runtime has no agent mapped to Manager %q", managerID)
	}
	if manager.WorkspaceMode == "" {
		// Explicit custom adapters can perform context-only work without a
		// native workspace. This is a current transport contract.
		return manager, nil
	}
	if manager.WorkspaceMode != "scoped" && manager.WorkspaceMode != "git" {
		return Agent{}, fmt.Errorf("Manager %q has unsupported workspace mode %q", managerID, manager.WorkspaceMode)
	}
	if config.Review == nil {
		return Agent{}, fmt.Errorf("native Manager %q requires a configured read-only assessment binding", managerID)
	}
	assessment, ok := config.Review.Agents[managerID]
	if !ok || ((assessment.WorkspaceMode != "" || len(assessment.InstructionPaths) != 0) && !supportsReadOnlyGitWorkspace(assessment)) {
		return Agent{}, fmt.Errorf("native Manager %q requires a separate read-only assessment binding", managerID)
	}
	return assessment, nil
}

// AgentConfig builds the explicit subprocess config for one manager. The
// configured environment contains names only; agentexec resolves values.
func (a Agent) AgentConfig() (agentexec.Config, error) {
	if err := ValidateRuntime(Runtime{APIVersion: APIVersion, Mode: ModeControlledLocal,
		Agents: map[string]Agent{"manager": a},
		Limits: Limits{MaxDepth: 1, MaxStarts: 1, MaxParallel: 1, MaxDuration: Duration(time.Minute), MaxCostMicros: 1,
			MaxCandidateFileBytes: 1, MaxCandidateBytes: 1}}); err != nil {
		return agentexec.Config{}, err
	}
	var modelOptions []byte
	if a.ModelOptions != nil {
		encoded, err := json.Marshal(a.ModelOptions)
		if err != nil {
			return agentexec.Config{}, fmt.Errorf("encode modelOptions: %w", err)
		}
		modelOptions = encoded
	}
	env := append([]string{}, a.Environment...)
	sort.Strings(env)
	var transportConfig json.RawMessage
	if a.Transport == TransportCodexAppServer {
		encoded, err := json.Marshal(a.AppServer)
		if err != nil {
			return agentexec.Config{}, fmt.Errorf("encode App Server settings: %w", err)
		}
		transportConfig = encoded
	}
	workspaceMode := a.WorkspaceMode
	if workspaceMode == "git" {
		// The native adapter binds Git worktree semantics from the source runtime
		// config. Its shared agentexec fingerprint remains transport-neutral.
		workspaceMode = ""
	}
	var environmentAllowlist *[]string
	if a.Transport == TransportCodexAppServer && a.AppServer != nil && a.AppServer.EnvironmentMode == AppServerEnvironmentModeInherit {
		// Agent.Environment remains the selected-name policy for declared
		// checks. Only the native App Server child inherits all caller values.
	} else {
		environmentAllowlist = &env
	}
	return agentexec.Config{
		Command: a.Command, Args: append([]string(nil), a.Args...), Model: a.Model,
		ModelOptions: modelOptions, ProviderVersion: a.ProviderVersion, Transport: a.Transport,
		TransportConfig: transportConfig,
		WorkspaceMode:   workspaceMode,
		Timeout:         time.Duration(a.Timeout), MaxStdoutBytes: a.MaxStdoutBytes,
		MaxStderrBytes: a.MaxStderrBytes, RuntimeFiles: append([]agentexec.RuntimeFile(nil), a.RuntimeFiles...),
		EnvironmentAllowlist: environmentAllowlist,
	}, nil
}

// Unmetered reports whether the agent declares that its invocations have no Markitect price.
func (a Agent) Unmetered() bool { return a.CostMode == CostModeUnmetered }
