package projectexecutor

import (
	"context"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
)

// ProcessInvoker is the real process transport composition for method tests
// and explicitly process-backed operations.
type ProcessInvoker struct{}

func (ProcessInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	return agentexec.Run(ctx, config, request, options)
}

func (ProcessInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return agentexec.Fingerprint(config)
}
