package projectrun

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

type nativeResumeContextKey struct{}
type nativeRecoveryRequiredKey struct{}

func requireNativeRecovery(ctx context.Context) context.Context {
	return context.WithValue(withNativeResume(ctx), nativeRecoveryRequiredKey{}, true)
}

// withNativeResume authorizes inspection of exact existing journals, never a
// replay of a previously reserved native call.
func withNativeResume(ctx context.Context) context.Context {
	return context.WithValue(ctx, nativeResumeContextKey{}, true)
}

func recoverInvocationOnResume(ctx context.Context, host Host, invoker Invoker, root, taskID string, config agentexec.Config, limits Limits, request agentexec.Request) (agentexec.RunResult, bool, error) {
	enabled, _ := ctx.Value(nativeResumeContextKey{}).(bool)
	if !enabled || config.Transport != TransportCodexAppServer {
		return agentexec.RunResult{}, false, nil
	}
	result, found, err := RecoverProjectAgent(ctx, host, invoker, root, taskID, config, limits, request)
	required, _ := ctx.Value(nativeRecoveryRequiredKey{}).(bool)
	if required && !found && err == nil {
		return result, true, errors.New("original native invocation journal is missing; replay is prohibited")
	}
	return result, found, err
}

// retainOriginalReceipt keeps known lifecycle and usage evidence when recovery
// fails before the transport can return another bound receipt.
func retainOriginalReceipt(prior, recovered InvocationLog) InvocationLog {
	if recovered.Receipt.RunID == "" {
		recovered.Receipt = prior.Receipt
		recovered.ReportID = prior.ReportID
		recovered.CostMicros = prior.CostMicros
		recovered.Outcome = prior.Outcome
	}
	return recovered
}
