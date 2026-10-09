package projectrun

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
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

// invokeAgent keeps the current v1 candidate pipeline from silently discarding
// a new Host-observed delta. P03/P06 must implement its explicit consumer before
// replacing this guard. The receipt survives unsupported output and failures.
func invokeAgent(ctx context.Context, invoker Invoker, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	result, err := invoker.Run(ctx, config, request, options)
	if err == nil && result.Delta != nil {
		err = errors.New("Host workspace delta is not supported by the current candidate pipeline")
	}
	return result, err
}
