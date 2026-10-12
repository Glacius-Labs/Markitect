package projectrun

import (
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
)

// Runtime binding version 2 separates the durable method policy from the
// executor implementation used to carry out an invocation. Older saved plans
// contain fingerprints in RuntimeDigest and cannot be reinterpreted safely.
const methodRuntimeBindingVersion = 2

type methodPolicy struct {
	APIVersion       string                          `json:"apiVersion"`
	Mode             string                          `json:"mode"`
	RequireIsolation bool                            `json:"requireIsolation"`
	Agents           map[string]methodAgentPolicy    `json:"agents"`
	Verifier         *methodAgentPolicy              `json:"verifier,omitempty"`
	Review           *methodReviewPolicy             `json:"review,omitempty"`
	Strictness       *projectconfig.StrictnessConfig `json:"strictness,omitempty"`
	Limits           projectconfig.Limits            `json:"limits"`
}

type methodReviewPolicy struct {
	Agents           map[string]methodAgentPolicy `json:"agents"`
	MaxRounds        int                          `json:"maxRounds"`
	MaxManagerRounds int                          `json:"maxManagerRounds"`
}

type methodAgentPolicy struct {
	Model            string                 `json:"model"`
	ModelOptions     any                    `json:"modelOptions,omitempty"`
	ProviderVersion  string                 `json:"providerVersion"`
	WorkspaceMode    string                 `json:"workspaceMode,omitempty"`
	InstructionPaths []string               `json:"instructionPaths,omitempty"`
	Timeout          projectconfig.Duration `json:"timeout"`
	MaxStdoutBytes   int                    `json:"maxStdoutBytes"`
	MaxStderrBytes   int                    `json:"maxStderrBytes"`
	Environment      []string               `json:"environment,omitempty"`
	CostMode         string                 `json:"costMode,omitempty"`
	Pricing          projectconfig.Pricing  `json:"pricing"`
	AppServer        *methodAppServerPolicy `json:"appServer,omitempty"`
}

// App Server settings that change the role's model or authority remain part
// of method policy. Transport, command, wrapper args, runtime file bytes and
// executor implementation fingerprints are bound at invocation time.
type methodAppServerPolicy struct {
	ReasoningEffort       string                                 `json:"reasoningEffort"`
	PermissionProfile     string                                 `json:"permissionProfile,omitempty"`
	WindowsSandboxBackend codexappserver.WindowsSandboxBackend   `json:"windowsSandboxBackend,omitempty"`
	EnvironmentMode       projectconfig.AppServerEnvironmentMode `json:"environmentMode,omitempty"`
	Helpers               projectconfig.AppServerHelpers         `json:"helpers"`
	MaxEventBytes         int64                                  `json:"maxEventBytes"`
}

func requireCurrentRuntimeBinding(plan PlanRecord) error {
	if plan.RuntimeBindingVersion != methodRuntimeBindingVersion {
		return ErrStale
	}
	return nil
}

func methodPolicyDigest(runtime projectconfig.Runtime) (string, error) {
	policy := methodPolicy{
		APIVersion: runtime.APIVersion, Mode: runtime.Mode, RequireIsolation: runtime.RequireIsolation,
		Agents: make(map[string]methodAgentPolicy, len(runtime.Agents)), Strictness: runtime.Strictness, Limits: runtime.Limits,
	}
	for role, agent := range runtime.Agents {
		policy.Agents[role] = methodPolicyForAgent(agent)
	}
	if runtime.Verifier != nil {
		verifier := methodPolicyForAgent(*runtime.Verifier)
		policy.Verifier = &verifier
	}
	if runtime.Review != nil {
		review := &methodReviewPolicy{
			Agents:    make(map[string]methodAgentPolicy, len(runtime.Review.Agents)),
			MaxRounds: runtime.Review.MaxRounds, MaxManagerRounds: runtime.Review.MaxManagerRounds,
		}
		for role, agent := range runtime.Review.Agents {
			review.Agents[role] = methodPolicyForAgent(agent)
		}
		policy.Review = review
	}
	return digest(policy)
}

func methodPolicyForAgent(agent projectconfig.Agent) methodAgentPolicy {
	environment := append([]string(nil), agent.Environment...)
	sort.Strings(environment)
	policy := methodAgentPolicy{
		Model: agent.Model, ModelOptions: agent.ModelOptions, ProviderVersion: agent.ProviderVersion,
		WorkspaceMode: agent.WorkspaceMode, InstructionPaths: append([]string(nil), agent.InstructionPaths...),
		Timeout: agent.Timeout, MaxStdoutBytes: agent.MaxStdoutBytes, MaxStderrBytes: agent.MaxStderrBytes,
		Environment: environment, CostMode: agent.CostMode, Pricing: agent.Pricing,
	}
	if agent.AppServer != nil {
		policy.AppServer = &methodAppServerPolicy{
			ReasoningEffort: agent.AppServer.ReasoningEffort, PermissionProfile: agent.AppServer.PermissionProfile,
			WindowsSandboxBackend: agent.AppServer.WindowsSandboxBackend, EnvironmentMode: agent.AppServer.EnvironmentMode,
			Helpers: agent.AppServer.Helpers, MaxEventBytes: agent.AppServer.MaxEventBytes,
		}
	}
	return policy
}
