package projectrun

import (
	"context"
	"errors"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

type observedDeltaInvoker struct {
	result agentexec.RunResult
	err    error
}

type invocationIdentityFixture struct {
	fingerprints []string
	reads        int
	receipt      string
}

func (i *invocationIdentityFixture) Fingerprint(agentexec.Config) (string, error) {
	index := i.reads
	if index >= len(i.fingerprints) {
		index = len(i.fingerprints) - 1
	}
	i.reads++
	return i.fingerprints[index], nil
}

func (i *invocationIdentityFixture) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	return agentexec.RunResult{Receipt: agentexec.Receipt{RunID: "observed-invocation", ConfigDigest: i.receipt}}, nil
}

func TestProjectRoleInvocationBindsExecutorReceiptAndRejectsDrift(t *testing.T) {
	for _, role := range []string{"manager", "reviewer", "verifier"} {
		for _, scenario := range []struct {
			name       string
			identities []string
			receipt    string
			stale      bool
		}{
			{"bound", []string{"executor-a", "executor-a"}, "executor-a", false},
			{"changed-executable", []string{"executor-a", "executor-b"}, "executor-a", true},
			{"wrong-receipt", []string{"executor-a", "executor-a"}, "executor-b", true},
		} {
			t.Run(role+"/"+scenario.name, func(t *testing.T) {
				invoker := &invocationIdentityFixture{fingerprints: scenario.identities, receipt: scenario.receipt}
				result, err := invokeProjectAgent(context.Background(), Host{}, invoker, "", nil, projectconfig.Agent{}, "", role, nil, nil, projectconfig.Limits{}, agentexec.Config{}, agentexec.Request{Role: role})
				if errors.Is(err, ErrStale) != scenario.stale || !scenario.stale && err != nil {
					t.Fatalf("receipt binding: err=%v, expected stale=%v", err, scenario.stale)
				}
				if result.Receipt.RunID != "observed-invocation" {
					t.Fatal("binding rejection discarded the actual invocation receipt")
				}
			})
		}
	}
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
