package projectrun

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
)

type requestFingerprinter interface {
	FingerprintForRequest(agentexec.Config, agentexec.Request) (string, error)
}

func fingerprintForRequest(invoker Invoker, config agentexec.Config, request agentexec.Request) (string, error) {
	if aware, ok := invoker.(requestFingerprinter); ok {
		return aware.FingerprintForRequest(config, request)
	}
	return invoker.Fingerprint(config)
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
