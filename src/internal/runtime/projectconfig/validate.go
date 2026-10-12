package projectconfig

import (
	"encoding/json"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var forbiddenEnvironmentPrefixes = []string{
	"GIT_", "SSH_", "AWS_", "AZURE_", "GH_", "GITHUB_", "MCP_",
}

var forbiddenEnvironmentNames = map[string]bool{
	"CODEX_HOME": true, "CLAUDE_CONFIG_DIR": true, "HOME": true,
	"USERPROFILE": true, "APPDATA": true, "LOCALAPPDATA": true,
}

// ValidateRuntime checks configuration shape and finite bounds independently
// of the project model. Executable identity is checked later by Fingerprint.
func ValidateRuntime(config Runtime) error {
	if err := ValidateStrictness(config.Strictness); err != nil {
		return err
	}
	if config.APIVersion != APIVersion {
		return fmt.Errorf("runtime apiVersion must be %q", APIVersion)
	}
	if config.Mode != ModeControlledLocal && config.Mode != ModeIsolated {
		return fmt.Errorf("runtime mode must be %q or %q", ModeControlledLocal, ModeIsolated)
	}
	if config.RequireIsolation || config.Mode == ModeIsolated {
		return fmt.Errorf("requested isolated execution is unavailable: no verified host-controlled launcher is configured")
	}
	if len(config.Agents) == 0 {
		return fmt.Errorf("runtime must declare at least one manager agent")
	}
	for managerID, agent := range config.Agents {
		if strings.TrimSpace(managerID) == "" {
			return fmt.Errorf("runtime agent manager ID must not be empty")
		}
		if strings.TrimSpace(agent.Command) == "" {
			return fmt.Errorf("runtime agent %q must declare command", managerID)
		}
		if strings.TrimSpace(agent.Model) == "" || strings.TrimSpace(agent.ProviderVersion) == "" {
			return fmt.Errorf("runtime agent %q must declare model and providerVersion", managerID)
		}
		if agent.Timeout <= 0 || agent.Timeout > Duration(time.Hour) || agent.MaxStdoutBytes <= 0 || agent.MaxStderrBytes <= 0 {
			return fmt.Errorf("runtime agent %q must declare positive timeout (at most 60m) and output limits", managerID)
		}
		if agent.ModelOptions != nil {
			if _, err := json.Marshal(agent.ModelOptions); err != nil {
				return fmt.Errorf("runtime agent %q modelOptions must be JSON-compatible: %w", managerID, err)
			}
		}
		if err := validateAgentTransport(managerID, agent); err != nil {
			return err
		}
		if agent.WorkspaceMode != "" && agent.WorkspaceMode != "scoped" && agent.WorkspaceMode != "git" {
			return fmt.Errorf("runtime agent %q has unsupported workspaceMode %q", managerID, agent.WorkspaceMode)
		}
		if agent.WorkspaceMode == "git" && agent.Transport != TransportCodexAppServer {
			return fmt.Errorf("runtime agent %q workspaceMode git requires transport %q", managerID, TransportCodexAppServer)
		}
		if agent.Transport == TransportCodexAppServer && agent.WorkspaceMode == "scoped" {
			return fmt.Errorf("runtime agent %q native App Server transport requires workspaceMode git or an empty workspaceMode", managerID)
		}
		if agent.WorkspaceMode == "" && len(agent.InstructionPaths) != 0 {
			return fmt.Errorf("runtime agent %q instructionPaths require scoped workspaceMode", managerID)
		}
		if agent.WorkspaceMode == "scoped" || agent.WorkspaceMode == "git" {
			if len(agent.InstructionPaths) == 0 || len(agent.InstructionPaths) > MaxInstructionFiles {
				return fmt.Errorf("runtime agent %q workspace requires 1..%d instructionPaths", managerID, MaxInstructionFiles)
			}
			if err := ValidateInstructionPaths(agent.InstructionPaths); err != nil {
				return fmt.Errorf("runtime agent %q instructionPaths: %w", managerID, err)
			}
		}
		if err := validateCostMode(managerID, agent); err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, name := range agent.Environment {
			if !environmentName.MatchString(name) || seen[name] {
				return fmt.Errorf("runtime agent %q has an invalid or duplicate environment variable name %q", managerID, name)
			}
			upper := strings.ToUpper(name)
			standardNativeHome := (agent.WorkspaceMode == "scoped" || (agent.WorkspaceMode == "git" && agent.Transport == TransportCodexAppServer)) && (upper == "HOME" || upper == "USERPROFILE" || upper == "APPDATA" || upper == "LOCALAPPDATA")
			if forbiddenEnvironmentNames[upper] && !standardNativeHome {
				return fmt.Errorf("runtime agent %q may not inherit environment variable %q", managerID, name)
			}
			for _, prefix := range forbiddenEnvironmentPrefixes {
				if strings.HasPrefix(upper, prefix) {
					return fmt.Errorf("runtime agent %q may not inherit environment variable %q", managerID, name)
				}
			}
			seen[name] = true
		}
	}
	if config.Verifier != nil {
		if config.Verifier.WorkspaceMode != "" && !supportsReadOnlyGitWorkspace(*config.Verifier) {
			return fmt.Errorf("native workspace mode is supported only for Manager executors, not runtime verifier")
		}
		if _, err := config.Verifier.AgentConfig(); err != nil {
			return fmt.Errorf("invalid runtime verifier: %w", err)
		}
	}
	if config.Review != nil {
		if len(config.Review.Agents) == 0 || config.Review.MaxRounds < 1 || config.Review.MaxRounds > 3 ||
			config.Review.MaxManagerRounds < 1 || config.Review.MaxManagerRounds > 3 {
			return fmt.Errorf("runtime review must declare reviewer agents and rounds within 1..3")
		}
		for managerID, agent := range config.Review.Agents {
			if agent.WorkspaceMode != "" && !supportsReadOnlyGitWorkspace(agent) {
				return fmt.Errorf("native workspace mode is supported only for Manager executors, not reviewer %q", managerID)
			}
			if strings.TrimSpace(managerID) == "" {
				return fmt.Errorf("runtime reviewer Manager ID must not be empty")
			}
			if _, err := agent.AgentConfig(); err != nil {
				return fmt.Errorf("invalid runtime reviewer %q: %w", managerID, err)
			}
		}
	}
	l := config.Limits
	if l.MaxDepth < 1 || l.MaxDepth > 32 || l.MaxStarts < 1 || l.MaxStarts > 256 ||
		l.MaxRetries < 0 || l.MaxRetries > 3 || l.MaxParallel < 1 || l.MaxParallel > 16 ||
		l.MaxDuration <= 0 || l.MaxDuration > Duration(24*time.Hour) || l.MaxCostMicros <= 0 ||
		l.MaxCandidateFileBytes <= 0 || l.MaxCandidateFileBytes > 8<<20 ||
		l.MaxCandidateBytes < l.MaxCandidateFileBytes || l.MaxCandidateBytes > 32<<20 {
		return fmt.Errorf("runtime limits must be finite and within supported bounds (depth 1..32, starts 1..256, retries 0..3, parallel 1..16, duration <=24h, candidate <=32 MiB)")
	}
	return nil
}

// validateCostMode requires rates for a metered agent and none for an
// unmetered one. Only process executors may be unmetered.
func validateCostMode(managerID string, agent Agent) error {
	switch agent.CostMode {
	case "", CostModeMetered:
		if agent.Pricing.InputMicrosPerMillion < 0 || agent.Pricing.OutputMicrosPerMillion < 0 ||
			(agent.Pricing.InputMicrosPerMillion == 0 && agent.Pricing.OutputMicrosPerMillion == 0) {
			return fmt.Errorf("runtime agent %q must declare nonnegative input/output pricing with at least one positive rate", managerID)
		}
	case CostModeUnmetered:
		if agent.Pricing != (Pricing{}) {
			return fmt.Errorf("runtime agent %q is unmetered and must not declare pricing", managerID)
		}
		if agent.Transport == TransportCodexAppServer {
			// The App Server reports token usage; leaving it unpriced would
			// silently remove the known-cost budget for billed work.
			return fmt.Errorf("runtime agent %q uses the Codex App Server, which reports token usage; it must be metered", managerID)
		}
	default:
		return fmt.Errorf("runtime agent %q has unsupported costMode %q; use %q or %q", managerID, agent.CostMode, CostModeMetered, CostModeUnmetered)
	}
	return nil
}

func validateAgentTransport(managerID string, agent Agent) error {
	if agent.AppServer != nil && agent.AppServer.EnvironmentMode != "" && agent.AppServer.EnvironmentMode != AppServerEnvironmentModeInherit {
		return fmt.Errorf("runtime agent %q appServer environmentMode must be empty or inherit", managerID)
	}
	if agent.AppServer != nil && agent.AppServer.EnvironmentMode == AppServerEnvironmentModeInherit &&
		(agent.Transport != TransportCodexAppServer || agent.WorkspaceMode != "git") {
		return fmt.Errorf("runtime agent %q appServer environmentMode inherit requires native codex-app-server with owned git workspace", managerID)
	}
	switch agent.Transport {
	case TransportProcess:
		if agent.AppServer != nil {
			return fmt.Errorf("runtime agent %q appServer settings require transport %q", managerID, TransportCodexAppServer)
		}
	case TransportCodexAppServer:
		if agent.AppServer == nil {
			return fmt.Errorf("runtime agent %q transport %q requires explicit appServer settings", managerID, TransportCodexAppServer)
		}
		if !filepath.IsAbs(agent.Command) {
			return fmt.Errorf("runtime agent %q App Server command must be an absolute executable path", managerID)
		}
		if len(agent.Args) != 0 {
			return fmt.Errorf("runtime agent %q App Server transport does not accept command arguments", managerID)
		}
		if agent.ModelOptions != nil {
			return fmt.Errorf("runtime agent %q App Server transport does not accept modelOptions", managerID)
		}
		settings := agent.AppServer
		adapterConfig := codexappserver.Config{
			Command: agent.Command, ProviderVersion: agent.ProviderVersion, Model: agent.Model,
			ReasoningEffort: settings.ReasoningEffort, PermissionProfile: settings.PermissionProfile,
			WindowsSandboxBackend: settings.WindowsSandboxBackend,
			Helpers:               codexappserver.HelperPolicy{Enabled: settings.Helpers.Enabled, MaxStartRequests: settings.Helpers.MaxStartRequests, MaxDepth: settings.Helpers.MaxDepth},
			Timeout:               time.Duration(agent.Timeout), MaxEventBytes: settings.MaxEventBytes,
		}
		if err := adapterConfig.Validate(); err != nil {
			return fmt.Errorf("runtime agent %q appServer settings: %w", managerID, err)
		}
	default:
		return fmt.Errorf("runtime agent %q has unsupported transport %q", managerID, agent.Transport)
	}
	return nil
}

func supportsReadOnlyGitWorkspace(agent Agent) bool {
	return agent.Transport == TransportCodexAppServer && agent.WorkspaceMode == "git" && len(agent.InstructionPaths) > 0
}
