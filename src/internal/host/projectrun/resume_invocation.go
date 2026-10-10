package projectrun

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

type nativeResumeContextKey struct{}
type nativeRecoveryRequiredKey struct{}
type nativeRecoveryLedgerKey struct{}
type nativeRecoveryOwnerKey struct{}

type nativeRecoveryBinding struct {
	RunID       string
	InputDigest string
	OwnerRunID  string
}

func requireNativeRecovery(ctx context.Context, ownerRunID string, invocations []InvocationLog) context.Context {
	ctx = context.WithValue(withNativeResume(ctx), nativeRecoveryRequiredKey{}, true)
	ctx = context.WithValue(ctx, nativeRecoveryLedgerKey{}, append([]InvocationLog(nil), invocations...))
	return context.WithValue(ctx, nativeRecoveryOwnerKey{}, ownerRunID)
}

func nativeRecoveryLedger(ctx context.Context) []InvocationLog {
	invocations, _ := ctx.Value(nativeRecoveryLedgerKey{}).([]InvocationLog)
	return invocations
}

// nativeRequestInputDigest uses the same normalized request digest that the
// agent transport records in its receipt. Preparing an invocation is local
// protocol work only; it does not start a provider call.
func nativeRequestInputDigest(request agentexec.Request) (string, error) {
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return "", err
	}
	return invocation.InputDigest, nil
}

// originalInvocationIndex reconciles a recovered result with the durable
// start row for the same task, role, and phase. A known receipt RunID is an
// exact discriminator, while a competing unreceipted start with the same raw
// digest remains ambiguous. The raw digest also selects a start row when no
// receipt exists.
func originalInvocationIndex(invocations []InvocationLog, expected InvocationLog) (int, error) {
	var matches []int
	for i, prior := range invocations {
		if prior.TaskID != expected.TaskID || prior.Role != expected.Role || prior.Phase != expected.Phase {
			continue
		}
		if expected.Receipt.RunID != "" {
			if prior.Receipt.RunID == expected.Receipt.RunID || prior.Receipt.RunID == "" && prior.InputDigest == expected.InputDigest {
				matches = append(matches, i)
			}
			continue
		}
		if prior.InputDigest == expected.InputDigest {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		return -1, errors.New("original invocation is absent from the durable ledger")
	}
	if len(matches) != 1 {
		return -1, errors.New("multiple durable ledger rows match the original invocation")
	}
	return matches[0], nil
}

func originalNativeRecoveryBinding(ownerRunID string, invocations []InvocationLog, taskID, role, phase string, request agentexec.Request) (nativeRecoveryBinding, error) {
	if ownerRunID == "" || taskID == "" || role == "" || phase == "" {
		return nativeRecoveryBinding{}, errors.New("original native recovery identity is incomplete")
	}
	inputDigest, err := nativeRequestInputDigest(request)
	if err != nil {
		return nativeRecoveryBinding{}, errors.New("original native request cannot be normalized")
	}
	rawDigest, err := digest(request)
	if err != nil {
		return nativeRecoveryBinding{}, errors.New("original native request cannot be encoded")
	}
	var matches []InvocationLog
	for _, invocation := range invocations {
		if invocation.TaskID != taskID || invocation.Role != role || invocation.Phase != phase {
			continue
		}
		ledgerDigestMatches := invocation.InputDigest == inputDigest || invocation.InputDigest == rawDigest || invocation.Receipt.InputDigest == inputDigest
		if !ledgerDigestMatches {
			continue
		}
		if invocation.Receipt.RunID != "" && invocation.Receipt.InputDigest != inputDigest {
			return nativeRecoveryBinding{}, errors.New("durable ledger receipt does not match the original input digest")
		}
		matches = append(matches, invocation)
	}
	if len(matches) == 0 {
		return nativeRecoveryBinding{InputDigest: inputDigest, OwnerRunID: ownerRunID}, nil
	}
	if len(matches) != 1 {
		return nativeRecoveryBinding{}, errors.New("multiple original native invocations match the durable request")
	}
	invocation := matches[0]
	if invocation.Receipt.RunID == "" {
		if invocation.Receipt.InputDigest != "" && invocation.Receipt.InputDigest != inputDigest {
			return nativeRecoveryBinding{}, errors.New("durable ledger receipt does not match the original input digest")
		}
		return nativeRecoveryBinding{InputDigest: inputDigest, OwnerRunID: ownerRunID}, nil
	}
	if invocation.Receipt.InputDigest != inputDigest {
		return nativeRecoveryBinding{}, errors.New("durable ledger receipt does not match the original input digest")
	}
	return nativeRecoveryBinding{RunID: invocation.Receipt.RunID, InputDigest: inputDigest, OwnerRunID: ownerRunID}, nil
}

// withNativeResume authorizes inspection of exact existing journals, never a
// replay of a previously reserved native call.
func withNativeResume(ctx context.Context) context.Context {
	return context.WithValue(ctx, nativeResumeContextKey{}, true)
}

func recoverInvocationOnResume(ctx context.Context, host Host, invoker Invoker, root, taskID string, config agentexec.Config, limits Limits, request agentexec.Request, binding nativeRecoveryBinding) (agentexec.RunResult, bool, error) {
	enabled, _ := ctx.Value(nativeResumeContextKey{}).(bool)
	if !enabled || config.Transport != TransportCodexAppServer {
		return agentexec.RunResult{}, false, nil
	}
	required, _ := ctx.Value(nativeRecoveryRequiredKey{}).(bool)
	if !required {
		return agentexec.RunResult{}, false, nil
	}
	if required && (binding.InputDigest == "" || binding.OwnerRunID == "") {
		return agentexec.RunResult{}, true, errors.New("exact original native invocation owner and input digest are required; replay is prohibited")
	}
	result, found, err := RecoverProjectAgent(ctx, host, invoker, root, taskID, config, limits, binding, request)
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
