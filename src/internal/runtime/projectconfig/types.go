// Package projectconfig owns the serialized project runtime configuration.
package projectconfig

import (
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
	"time"
)

const (
	APIVersion              = "markitect.example.org/project-run/v1alpha1"
	RuntimePath             = ".markitect/runtime.yaml"
	ModeControlledLocal     = "controlled-local"
	ModeIsolated            = "isolated"
	TransportProcess        = ""
	TransportCodexAppServer = "codex-app-server"
	CostModeMetered         = "metered"
	CostModeUnmetered       = "unmetered"
)

// Runtime binds one explicit agent invocation per manager role and finite
// resource ceilings. The configured mode is descriptive; only a verified
// host-controlled launcher can establish isolated execution.
type Runtime struct {
	APIVersion       string            `json:"apiVersion" yaml:"apiVersion"`
	Mode             string            `json:"mode" yaml:"mode"`
	RequireIsolation bool              `json:"requireIsolation" yaml:"requireIsolation"`
	Agents           map[string]Agent  `json:"agents" yaml:"agents"`
	Verifier         *Agent            `json:"verifier,omitempty" yaml:"verifier,omitempty"`
	Review           *ReviewConfig     `json:"review,omitempty" yaml:"review,omitempty"`
	Strictness       *StrictnessConfig `json:"strictness,omitempty" yaml:"strictness,omitempty"`
	Limits           Limits            `json:"limits" yaml:"limits"`
}

// ReviewConfig enables an independent bounded reviewer for each active
// Manager. Agent keys are Manager IDs; review invocations use their own
// runtime fingerprints and share the run's cumulative resource ceilings.
type ReviewConfig struct {
	Agents           map[string]Agent `json:"agents" yaml:"agents"`
	MaxRounds        int              `json:"maxRounds" yaml:"maxRounds"`
	MaxManagerRounds int              `json:"maxManagerRounds" yaml:"maxManagerRounds"`
}

type Agent struct {
	Command          string                  `json:"command" yaml:"command"`
	Args             []string                `json:"args" yaml:"args"`
	Transport        string                  `json:"transport,omitempty" yaml:"transport,omitempty"`
	AppServer        *AppServerSettings      `json:"appServer,omitempty" yaml:"appServer,omitempty"`
	Model            string                  `json:"model" yaml:"model"`
	ModelOptions     any                     `json:"modelOptions,omitempty" yaml:"modelOptions"`
	ProviderVersion  string                  `json:"providerVersion" yaml:"providerVersion"`
	WorkspaceMode    string                  `json:"workspaceMode,omitempty" yaml:"workspaceMode,omitempty"`
	InstructionPaths []string                `json:"instructionPaths,omitempty" yaml:"instructionPaths,omitempty"`
	Timeout          Duration                `json:"timeout" yaml:"timeout"`
	MaxStdoutBytes   int                     `json:"maxStdoutBytes" yaml:"maxStdoutBytes"`
	MaxStderrBytes   int                     `json:"maxStderrBytes" yaml:"maxStderrBytes"`
	RuntimeFiles     []agentexec.RuntimeFile `json:"runtimeFiles" yaml:"runtimeFiles"`
	Environment      []string                `json:"environment,omitempty" yaml:"environment,omitempty"`
	// CostMode is empty or "metered" for a priced agent and "unmetered" for an
	// executor whose invocations Markitect cannot price, such as a script or a
	// person behind an exchange adapter.
	CostMode string  `json:"costMode,omitempty" yaml:"costMode,omitempty"`
	Pricing  Pricing `json:"pricing" yaml:"pricing"`
}

// AppServerSettings explicitly selects the native Codex transport contract.
// Workspace authority remains in Agent.WorkspaceMode and the project model.
type AppServerSettings struct {
	ReasoningEffort       string                               `json:"reasoningEffort" yaml:"reasoningEffort"`
	PermissionProfile     string                               `json:"permissionProfile,omitempty" yaml:"permissionProfile,omitempty"`
	WindowsSandboxBackend codexappserver.WindowsSandboxBackend `json:"windowsSandboxBackend,omitempty" yaml:"windowsSandboxBackend,omitempty"`
	EnvironmentMode       AppServerEnvironmentMode             `json:"environmentMode,omitempty" yaml:"environmentMode,omitempty"`
	Helpers               AppServerHelpers                     `json:"helpers" yaml:"helpers"`
	MaxEventBytes         int64                                `json:"maxEventBytes" yaml:"maxEventBytes"`
}

// AppServerEnvironmentMode selects caller environment inheritance for the
// native App Server child only. Agent.Environment remains the explicit name
// policy used to resolve declared-check executables.
type AppServerEnvironmentMode string

const AppServerEnvironmentModeInherit AppServerEnvironmentMode = "inherit"

type AppServerHelpers struct {
	Enabled          bool `json:"enabled" yaml:"enabled"`
	MaxStartRequests int  `json:"maxStartRequests" yaml:"maxStartRequests"`
	MaxDepth         int  `json:"maxDepth" yaml:"maxDepth"`
}

// Pricing is an explicit estimate used to enforce the configured run budget.
// It does not claim to be a provider invoice or billing record.
type Pricing struct {
	InputMicrosPerMillion  int64 `json:"inputMicrosPerMillion" yaml:"inputMicrosPerMillion"`
	OutputMicrosPerMillion int64 `json:"outputMicrosPerMillion" yaml:"outputMicrosPerMillion"`
}

type Limits struct {
	MaxDepth              int      `json:"maxDepth" yaml:"maxDepth"`
	MaxStarts             int      `json:"maxStarts" yaml:"maxStarts"`
	MaxRetries            int      `json:"maxRetries" yaml:"maxRetries"`
	MaxParallel           int      `json:"maxParallel" yaml:"maxParallel"`
	MaxDuration           Duration `json:"maxDuration" yaml:"maxDuration"`
	MaxCostMicros         int64    `json:"maxCostMicros" yaml:"maxCostMicros"`
	MaxCandidateFileBytes int64    `json:"maxCandidateFileBytes" yaml:"maxCandidateFileBytes"`
	MaxCandidateBytes     int64    `json:"maxCandidateBytes" yaml:"maxCandidateBytes"`
}

// Duration accepts an explicit Go duration string in runtime.yaml.
type Duration time.Duration

// StrictnessConfig contains a project minimum and additive Manager-specific
// requirements. It changes review depth and evidence requested, never the
// accepted model or mandatory project checks.
type StrictnessConfig struct {
	Default  StrictnessProfile            `json:"default" yaml:"default"`
	Managers map[string]StrictnessProfile `json:"managers,omitempty" yaml:"managers,omitempty"`
}

// StrictnessProfile describes additional evidence requested from a Manager.
// Counterexamples is an additive count on top of the project default.
type StrictnessProfile struct {
	Evidence        []string `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Counterexamples int      `json:"counterexamples,omitempty" yaml:"counterexamples,omitempty"`
}
