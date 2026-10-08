package projectrun

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

// ProcessInvoker delegates transport, input binding and receipts to the shared
// agentexec package. ProjectRun adds task scope and durable orchestration.
type ProcessInvoker struct{}

func (ProcessInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	return agentexec.Run(ctx, config, request, options)
}

func (ProcessInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return agentexec.Fingerprint(config)
}
