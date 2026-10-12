// Package projectexecutor composes the process and native role transports.
package projectexecutor

import (
	"context"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
)

// Invoker is the role execution boundary consumed by method orchestration.
type Invoker interface {
	Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error)
	Fingerprint(agentexec.Config) (string, error)
}

// BoundHelper is the per-invocation method helper binding. Merge accounts for
// completed nested requests on the parent transport receipt.
type BoundHelper struct {
	Options codexappserver.Options
	Merge   func(*agentexec.Receipt)
}

// HelperPolicy keeps request-specific helper authority and execution in the
// method layer while the executor owns native transport construction.
type HelperPolicy interface {
	ToolOptions(agentexec.Config, codexappserver.Options) (codexappserver.Options, error)
	Eligible(agentexec.Request) bool
	BindRun(agentexec.Config, agentexec.Request, agentexec.RunOptions, codexappserver.Options) (BoundHelper, error)
	BindRecovery(agentexec.Config, agentexec.Request, codexappserver.Options) (codexappserver.Options, error)
}

// HelperBinder binds method-owned policy/session state without changing the
// stable helper tool specification used by planned fingerprints.
type HelperBinder interface {
	WithHelperPolicy(HelperPolicy) Invoker
}
