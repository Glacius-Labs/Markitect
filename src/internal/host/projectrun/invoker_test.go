package projectrun

import (
	"context"
	"errors"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

type observedDeltaInvoker struct {
	result agentexec.RunResult
	err    error
}

func (i observedDeltaInvoker) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	return i.result, i.err
}
func (observedDeltaInvoker) Fingerprint(agentexec.Config) (string, error) { return "", nil }

func TestUnwiredWorkspaceDeltaFailsAndPreservesReceipt(t *testing.T) {
	started := agentexec.RunResult{Delta: &projectworkspace.Delta{}, Receipt: agentexec.Receipt{RunID: "started"}}
	result, err := invokeAgent(context.Background(), observedDeltaInvoker{result: started}, agentexec.Config{}, agentexec.Request{}, agentexec.RunOptions{})
	if err == nil || result.Delta == nil || result.Receipt.RunID != "started" {
		t.Fatalf("delta silently discarded or receipt lost: %+v %v", result, err)
	}
	providerErr := errors.New("original failure")
	result, err = invokeAgent(context.Background(), observedDeltaInvoker{result: started, err: providerErr}, agentexec.Config{}, agentexec.Request{}, agentexec.RunOptions{})
	if !errors.Is(err, providerErr) || result.Receipt.RunID != "started" {
		t.Fatalf("original failure evidence lost: %+v %v", result, err)
	}
	if _, err := invokeAgent(context.Background(), observedDeltaInvoker{}, agentexec.Config{}, agentexec.Request{}, agentexec.RunOptions{}); err != nil {
		t.Fatalf("current no-delta result changed: %v", err)
	}
}
