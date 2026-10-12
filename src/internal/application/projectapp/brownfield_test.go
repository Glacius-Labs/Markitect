package projectapp

import (
	"context"
	"errors"
	"testing"
)

func TestBrownfieldRejectsMixedStagePayloadsBeforeReadingRepositories(t *testing.T) {
	operations := Operations{}
	for _, input := range []BrownfieldInput{
		{},
		{Start: &BrownfieldStartInput{}, Plan: &BrownfieldPlanInput{}},
		{Plan: &BrownfieldPlanInput{}},
	} {
		_, err := operations.Brownfield(BrownfieldOperation{Root: t.TempDir(), Action: "start", Input: input})
		if err == nil || err.Error() != "adopt input must contain exactly the typed payload for the selected action" {
			t.Fatalf("mixed or missing stage input reached filesystem: %v", err)
		}
	}
	if _, err := operations.Brownfield(BrownfieldOperation{Root: t.TempDir(), Action: "status", Input: BrownfieldInput{Plan: &BrownfieldPlanInput{}}}); err == nil {
		t.Fatal("status accepted a mutation payload")
	}
	for _, removed := range []string{"resume", "apply-adoption", "record-adoption"} {
		if _, err := operations.Brownfield(BrownfieldOperation{Root: t.TempDir(), Action: removed}); err == nil {
			t.Fatalf("removed stage %q accepted", removed)
		}
	}
}

func TestBrownfieldRunCancellationPrecedesRuntimeLoading(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := (Operations{}).BrownfieldRun(ctx, BrownfieldRunOperation{Root: t.TempDir()}, nil)
	if !errors.Is(err, context.Canceled) || result.Attempt != nil {
		t.Fatalf("cancelled operation loaded runtime or started an attempt: result=%+v err=%v", result, err)
	}
}
