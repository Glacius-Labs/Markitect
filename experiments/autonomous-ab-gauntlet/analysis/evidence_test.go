package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func evidenceFile(t *testing.T, arena, path, content string) {
	t.Helper()
	full := filepath.Join(arena, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationCollectsActualInitialAndRepairHelperLocations(t *testing.T) {
	arena := t.TempDir()
	n := nativeRecord{RunID: "p-a-t01-integration", Project: "p", Arm: "a", TaskID: "08", EvaluationTaskID: "08", Trial: 1, ActorStatus: "completed", ParallelIntegration: true, RepairIterations: 1}
	evidenceFile(t, arena, "runs/"+n.RunID+"/08.evaluation.yaml", "task: '08'\nstatus: passed\nchecks:\n  - name: combined-vector\n    passed: true\n")
	for index, exit := range []int{1, 0} {
		evidenceFile(t, arena, fmt.Sprintf("raw/%s/08.attempt-%d.helper.jsonl", n.RunID, index), fmt.Sprintf("{\"task_id\":\"08\",\"exit_code\":%d,\"actor_attempt\":%d}\n", exit, index+1))
	}
	got, err := analyzeTask(arena, n, taskCard{ID: "08"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.PublicHelper.Calls != 2 || got.PublicHelper.Failure != 1 || got.PublicHelper.Success != 1 || got.PublicHelper.BeforeRepairStatus != "failed" || got.PublicHelper.FinalStatus != "success" || got.CountedOutcome != "passed" {
		t.Fatalf("integration evidence = %#v", got)
	}
	if got.RunKind != "parallel-integration" {
		t.Fatalf("integration kind = %s", got.RunKind)
	}
}

func TestUnnamedUnresolvedCheckCannotCountAsPassing(t *testing.T) {
	for _, status := range []string{"failed", "manual", "unknown", "owner_decision_required"} {
		t.Run(status, func(t *testing.T) {
			arena := t.TempDir()
			n := nativeRecord{RunID: "r", Project: "p", Arm: "a", Trial: 1, TaskID: "01", ActorStatus: "completed"}
			evidenceFile(t, arena, "runs/r/01.evaluation.yaml", "task: '01'\nstatus: passed\nchecks:\n  - status: "+status+"\n")
			evidenceFile(t, arena, "raw/r/01/helper.jsonl", "{\"task_id\":\"01\",\"exit_code\":0,\"actor_attempt\":1}\n")
			got, err := analyzeTask(arena, n, taskCard{ID: "01"}, true)
			if err != nil {
				t.Fatal(err)
			}
			if got.CountedOutcome == "passed" || len(got.RawChecks) != 1 || len(got.Checks) != 1 || got.Checks[0].Name != "unnamed-check-1" || len(got.EvaluationEvidenceConflicts) == 0 {
				t.Fatalf("unnamed %s evidence = %#v", status, got)
			}
		})
	}
}

func TestWorkflowElapsedUsesOnlyRecordedClockAndNeverAttention(t *testing.T) {
	arena := t.TempDir()
	n := nativeRecord{RunID: "r", Project: "p", Arm: "a", Trial: 1, TaskID: "01", ActorStartedUTC: "2026-10-04T18:13:41Z", ActorCompletedUTC: "2026-10-04T18:23:21Z", DeadlineUTC: "2026-10-04T18:43:41Z"}
	got, err := analyzeTask(arena, n, taskCard{ID: "01"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkflowTimeStatus != "available" || got.ElapsedWorkflowSeconds == nil || *got.ElapsedWorkflowSeconds != 580 || got.AttentionStatus != "unavailable" || got.AttentionSeconds != nil || got.WorkflowDeadlineUTC != n.DeadlineUTC {
		t.Fatalf("workflow/attention = %#v", got)
	}
	for _, times := range [][2]string{{"", ""}, {n.ActorStartedUTC, ""}, {"bad-date", n.ActorCompletedUTC}, {n.ActorCompletedUTC, n.ActorStartedUTC}} {
		seconds, why := workflowElapsed(times[0], times[1])
		if seconds != nil || why == "" {
			t.Fatalf("unavailable timestamps %v = %v / %q", times, seconds, why)
		}
	}
}

func TestEvaluatorTaskBindingAndParallelRecordFilename(t *testing.T) {
	arena := t.TempDir()
	n := nativeRecord{RunID: "r", Project: "p", Arm: "a", Trial: 1, TaskID: "P01", EvaluationTaskID: "07", ActorStatus: "completed", Parallel: true}
	evidenceFile(t, arena, "raw/r/P01/helper.jsonl", "{\"task_id\":\"P01\",\"exit_code\":0,\"actor_attempt\":1}\n")
	evidenceFile(t, arena, "runs/r/P01.evaluation.yaml", "task: '08'\nstatus: passed\nchecks: []\n")
	got, err := analyzeTask(arena, n, taskCard{ID: "P01"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.AggregateEvaluationStatus != "passed" || got.EvaluationRecordStatus != "task-identity-mismatch" || got.CountedOutcome == "passed" {
		t.Fatalf("mismatched evaluation = %#v", got)
	}
	evidenceFile(t, arena, "runs/r/P01.evaluation.yaml", "task: '07'\nstatus: passed\nchecks: []\n")
	got, err = analyzeTask(arena, n, taskCard{ID: "P01"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.CountedOutcome != "passed" || got.EvaluationRecordStatus != "matched" || got.RunKind != "parallel-fork" {
		t.Fatalf("matched parallel evaluation = %#v", got)
	}
}

func TestUnexpectedEscalationCannotEscapeImplementationDenominator(t *testing.T) {
	arena := t.TempDir()
	n := nativeRecord{RunID: "r", Project: "p", Arm: "a", Trial: 1, TaskID: "01"}
	evidenceFile(t, arena, "runs/r/01.evaluation.yaml", "task: '01'\nstatus: owner_decision_required\n")
	got, err := analyzeTask(arena, n, taskCard{ID: "01"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.OwnerDecisionRequired || !got.UnexpectedOwnerDecision || got.UnexpectedOwnerAssessment != "unavailable" {
		t.Fatalf("unexpected owner handling = %#v", got)
	}
	c := counts{}
	countOutcome(&c, got.CountedOutcome, got.OwnerDecisionRequired)
	if c.OwnerDecisionRequired != 0 || c.Incomplete != 1 || c.UnexpectedOwnerDecision != 1 {
		t.Fatalf("unexpected counts = %#v", c)
	}
	g := makeGroups([]taskReport{got})[0]
	if g.Outcomes.Tasks != 1 || g.Outcomes.ImplementationEligibleTasks != 1 || g.Outcomes.OwnerDecisionRequired != 0 || g.Outcomes.UnexpectedOwnerDecision != 1 || g.Outcomes.Incomplete != 1 {
		t.Fatalf("unexpected group = %#v", g)
	}
}

func TestBeforeRepairContradictionDoesNotContaminateFinalEvidence(t *testing.T) {
	for _, status := range []string{"failed", "manual", "unknown"} {
		t.Run(status, func(t *testing.T) {
			arena := t.TempDir()
			n := nativeRecord{RunID: "r", Project: "p", Arm: "a", Trial: 1, TaskID: "01", ActorStatus: "completed"}
			evidenceFile(t, arena, "runs/r/01.before-repair.evaluation.yaml", "task: '01'\nstatus: passed\nchecks:\n  - status: "+status+"\n")
			evidenceFile(t, arena, "runs/r/01.evaluation.yaml", "task: '01'\nstatus: passed\nchecks:\n  - name: final-vector\n    passed: true\n")
			evidenceFile(t, arena, "raw/r/01/helper.jsonl", "{\"task_id\":\"01\",\"exit_code\":0,\"actor_attempt\":1}\n")
			got, err := analyzeTask(arena, n, taskCard{ID: "01"}, true)
			if err != nil {
				t.Fatal(err)
			}
			if got.BeforeRepairAggregateStatus != "passed" || got.BeforeRepairEvaluationStatus != "incomplete" || got.BeforeRepairChecks[0].Name != "unnamed-check-1" || got.CountedOutcome != "passed" || len(got.EvaluationEvidenceConflicts) != 1 {
				t.Fatalf("independent before/final evidence = %#v", got)
			}
		})
	}
}

func TestOwnerTaskRemainsInTotalButNotImplementationDenominator(t *testing.T) {
	rows := []taskReport{
		{RunID: "r", RunKind: "sequential", Project: "p", Arm: "A", Trial: 1, TaskID: "01", CountedOutcome: "passed"},
		{RunID: "r", RunKind: "sequential", Project: "p", Arm: "A", Trial: 1, TaskID: "02", CountedOutcome: "owner_decision_required", OwnerDecisionRequired: true, Repairs: 1},
	}
	got := makeGroups(rows)[0]
	if got.Outcomes.Tasks != 2 || got.Outcomes.ImplementationEligibleTasks != 1 || got.Outcomes.OwnerDecisionRequired != 1 || got.Outcomes.Repairs != 1 || got.Outcomes.Passed != 1 {
		t.Fatalf("owner/implementation denominator = %#v", got)
	}
}

func TestParallelTaskSetUsesExplicitOverlayInsteadOfSequentialLocation(t *testing.T) {
	arena := t.TempDir()
	evidenceFile(t, arena, "projects/p/project.yaml", "shared:\n  taskSet: tasks/task-set.yaml\n")
	evidenceFile(t, arena, "projects/p/tasks/task-set.yaml", "tasks:\n  - id: '01'\n")
	evidenceFile(t, arena, "projects/p/parallel-task-set.yaml", "tasks:\n  - id: P01\n    origin_task_id: '07'\n")
	cards, err := loadCardsForRun(arena, "p", "a", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards["P01"].OriginTaskID != "07" {
		t.Fatalf("parallel cards = %#v", cards)
	}
}

func TestFirstStreakBoundaryIsRetainedAndParallelIsSeparate(t *testing.T) {
	rows := []taskReport{
		{RunID: "r", RunKind: "sequential", Project: "p", Arm: "A", Trial: 1, TaskID: "01", CountedOutcome: "passed"},
		{RunID: "r", RunKind: "sequential", Project: "p", Arm: "A", Trial: 1, TaskID: "02", CountedOutcome: "failed"},
		{RunID: "r", RunKind: "sequential", Project: "p", Arm: "A", Trial: 1, TaskID: "03", CountedOutcome: "failed"},
		{RunID: "fork", RunKind: "parallel-fork", Project: "p", Arm: "A", Trial: 1, TaskID: "P01", CountedOutcome: "passed"},
	}
	groups := makeGroups(rows)
	if len(groups) != 2 {
		t.Fatalf("groups = %#v", groups)
	}
	for _, g := range groups {
		if g.RunKind == "sequential" && (g.PassingStreak != 1 || g.StreakBoundary != "first failure at task 02" || g.Outcomes.Tasks != 3) {
			t.Fatalf("main streak = %#v", g)
		}
		if g.RunKind == "parallel-fork" && (g.StreakStatus != "not-longitudinal" || g.PassingStreak != 0 || g.Outcomes.Tasks != 1) {
			t.Fatalf("fork streak = %#v", g)
		}
	}
}

func TestTraversalAndLinkedInputOutputBoundaries(t *testing.T) {
	arena := t.TempDir()
	for _, path := range []string{"../outside", "..\\outside", "C:/outside"} {
		if _, err := arenaPath(arena, path); err == nil {
			t.Fatalf("unsafe path accepted: %s", path)
		}
	}
	n := nativeRecord{RunID: "r", Project: "p", Arm: "a", Trial: 1, TaskID: "01", FinalSnapshot: "../outside"}
	if _, err := analyzeTask(arena, n, taskCard{ID: "01"}, true); err == nil {
		t.Fatal("snapshot traversal accepted")
	}
	outside := t.TempDir()
	if err := os.Symlink(arena, filepath.Join(outside, "alias")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	canonical, err := filepath.EvalSymlinks(arena)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := canonicalOutputPath(canonical, filepath.Join(outside, "alias", "report")); err == nil {
		t.Fatal("output alias into arena accepted")
	}
	if err := os.Symlink(outside, filepath.Join(arena, "outside")); err != nil {
		t.Fatal(err)
	}
	if _, err := arenaPath(arena, "outside/file.yaml"); err == nil {
		t.Fatal("linked input outside arena accepted")
	}
}

func TestFreezeAggregateIsRecomputed(t *testing.T) {
	arena := t.TempDir()
	evidenceFile(t, arena, "protocol.yaml", "id: fixture\n")
	sum, err := hashFile(filepath.Join(arena, "protocol.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	inputs := []freezeInput{{Path: "protocol.yaml", SHA256: sum}}
	manifest := struct {
		Inputs []freezeInput `yaml:"inputs"`
		Digest string        `yaml:"digest"`
	}{inputs, freezeDigest(inputs)}
	bytes, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	evidenceFile(t, arena, "freeze.yaml", string(bytes))
	if err := verifyFrozenInputs(arena); err != nil {
		t.Fatal(err)
	}
	manifest.Digest = strings.Repeat("0", 64)
	bytes, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	evidenceFile(t, arena, "freeze.yaml", string(bytes))
	if err := verifyFrozenInputs(arena); err == nil || !strings.Contains(err.Error(), "aggregate digest mismatch") {
		t.Fatalf("forged aggregate accepted: %v", err)
	}
}
