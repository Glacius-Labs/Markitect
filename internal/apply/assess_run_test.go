package apply_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// assessedApply runs an Apply of the library fixture with the scripted Agents
// and returns its result, the Agents, the check runner and the publisher.
func assessedApply(t *testing.T, strategy, runID string, agents *preservingAgents) (apply.ApplyResult, *passingRunner, *fakePublisher) {
	t.Helper()
	req := strategyRequestWithTurns(t, strategy, 400)
	req.RunID = runID
	cfg, err := apply.BuildApplyConfig(req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	agents.t, agents.obligations = t, map[string][]string{}
	for _, m := range cfg.Team {
		agents.obligations[m.Path] = m.Obligations
	}
	runner, publisher := &passingRunner{}, &fakePublisher{}
	req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = agents, &treeWorkspaces{}, runner, &memoryEvidence{}, publisher
	result, _ := apply.Apply(context.Background(), req)
	return result, runner, publisher
}

// Every Apply begins with the Model assessment; a gap it confirms ends the run
// Failed with its class before any work: no Execute turn, no check, no
// publication, in both strategies.
func TestAConfirmedAssessmentGapEndsTheApplyBeforeAnyWork(t *testing.T) {
	for _, strategy := range []string{"direct", "staged"} {
		t.Run(strategy, func(t *testing.T) {
			agents := &preservingAgents{finds: reminders, gap: true}
			result, runner, publisher := assessedApply(t, strategy, "run-1", agents)
			out := result.Outcome
			if out.Status != run.Failed || !strings.HasPrefix(out.Reason, run.ModelAssessmentReason) || !strings.Contains(out.Reason, "["+reminders+", model_gap]") {
				t.Fatalf("outcome %s: %s", out.Status, out.Reason)
			}
			if len(out.Passes) != 1 || out.Passes[0].Purpose != run.Assess || out.Passes[0].Verdict != run.NeedsCorrection {
				t.Fatalf("passes %+v", out.Passes)
			}
			if slices.ContainsFunc(agents.turns, func(turn run.Turn) bool { return turn.Kind == run.ExecuteTurn }) || len(runner.runs) != 0 || len(publisher.requests) != 0 || result.Receipt != nil {
				t.Fatalf("work, checks or publication after a confirmed gap: %d checks, %d publications", len(runner.runs), len(publisher.requests))
			}
		})
	}
}

// A model problem Root confirms during realization, after an accepted
// assessment, ends the run Failed as well: no Verify and no publication.
func TestAModelProblemDuringTheWorkEndsTheApply(t *testing.T) {
	agents := &preservingAgents{abort: "."}
	result, _, publisher := assessedApply(t, "direct", "run-1", agents)
	out := result.Outcome
	if out.Status != run.Failed || !strings.HasPrefix(out.Reason, run.ModelErrorReason) || len(publisher.requests) != 0 || result.Receipt != nil {
		t.Fatalf("outcome %s: %s; %d publications", out.Status, out.Reason, len(publisher.requests))
	}
	if out.Passes[0].Purpose != run.Assess || out.Passes[0].Verdict != run.Satisfied || slices.ContainsFunc(out.Passes, func(p run.PassRecord) bool { return p.Purpose == run.Verify }) {
		t.Fatalf("passes %+v", out.Passes)
	}
}

// A failed run stays what it was; the run after a model correction outside it
// is a new run with its own assessment, and nothing of the failed run reaches
// it.
func TestARunAfterAFailedAssessmentAssessesAgain(t *testing.T) {
	failed, _, _ := assessedApply(t, "direct", "run-1", &preservingAgents{finds: reminders, gap: true})
	if failed.Outcome.Status != run.Failed {
		t.Fatalf("first run %s: %s", failed.Outcome.Status, failed.Outcome.Reason)
	}
	agents := &preservingAgents{}
	next, _, publisher := assessedApply(t, "direct", "run-2", agents)
	if next.Outcome.Status != run.Verified || len(publisher.requests) != 1 || next.Outcome.Passes[0].Purpose != run.Assess || len(next.Outcome.Findings) != 0 {
		t.Fatalf("second run %s: %s; passes %+v", next.Outcome.Status, next.Outcome.Reason, next.Outcome.Passes)
	}
	for _, turn := range agents.turns {
		if !strings.HasPrefix(turn.ID, "run-2-") || strings.Contains(turn.Prompt, "A scripted counterexample.") {
			t.Fatalf("turn %s carries the failed run", turn.ID)
		}
	}
}

// The Model assessment runs on the capture its entry checks passed: no gate,
// no required check and no probe tool; its reviewers start fresh and read-only.
func TestTheAssessmentRunsNoCheckAndOffersNoProbe(t *testing.T) {
	agents := &preservingAgents{}
	result, runner, _ := assessedApply(t, "direct", "run-1", agents)
	if result.Outcome.Status != run.Verified || len(runner.runs) != 1 {
		t.Fatalf("outcome %s: %s; %d checks", result.Outcome.Status, result.Outcome.Reason, len(runner.runs))
	}
	for _, turn := range agents.turns {
		if !strings.Contains(turn.ID, "-p1-") {
			continue
		}
		if turn.Kind != run.VerifyTurn || !turn.ReadOnly || slices.ContainsFunc(turn.Tools, func(s run.ToolSpec) bool { return s.Name == "check" }) {
			t.Fatalf("assessment turn %s: kind %s, read-only %v", turn.ID, turn.Kind, turn.ReadOnly)
		}
		if turn.Resume == "" && (!strings.Contains(turn.Prompt, "You assess the bound model") || !strings.Contains(turn.Prompt, "Pass p1: assess")) {
			t.Fatalf("assessment turn %s is not framed as the Model assessment", turn.ID)
		}
	}
}
