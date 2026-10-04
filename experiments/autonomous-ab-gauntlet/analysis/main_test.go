package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluationOutcomeIsSeparateFromPublicHelper(t *testing.T) {
	arena := t.TempDir()
	runID, taskID := "fixture-a-t01", "01"
	if err := os.MkdirAll(filepath.Join(arena, "runs", runID), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(arena, "raw", runID, taskID), 0700); err != nil {
		t.Fatal(err)
	}
	eval := "task: \"01\"\nstatus: failed\nchecks:\n  - name: hidden-vector\n    passed: false\n  - name: other-component\n    passed: true\n"
	if err := os.WriteFile(filepath.Join(arena, "runs", runID, taskID+".evaluation.yaml"), []byte(eval), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(arena, "raw", runID, taskID, "helper.jsonl"), []byte(`{"task_id":"01","commands":[["go","test","./..."]],"exit_code":0,"snapshot":"snapshots/01","actor_attempt":1}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := analyzeTask(arena, nativeRecord{RunID: runID, Project: "modular-service", Arm: "a", Trial: 1, TaskID: taskID}, taskCard{ID: taskID}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.EvaluationStatus != "failed" {
		t.Fatalf("evaluation status = %q, want failed", got.EvaluationStatus)
	}
	if len(got.EvaluationEvidenceConflicts) != 0 {
		t.Fatalf("failed aggregate with a passed component was called contradictory: %#v", got.EvaluationEvidenceConflicts)
	}
	if got.PublicHelper.Status != "success" || got.PublicHelper.Calls != 1 {
		t.Fatalf("helper receipt = %#v", got.PublicHelper)
	}
}

func TestPassedAggregateConflictsWithUnresolvedRequiredComponent(t *testing.T) {
	for _, component := range []string{"failed", "manual", "unknown"} {
		t.Run(component, func(t *testing.T) {
			arena := t.TempDir()
			runID, taskID := "fixture", "01"
			if err := os.MkdirAll(filepath.Join(arena, "runs", runID), 0700); err != nil {
				t.Fatal(err)
			}
			record := "task: '01'\nstatus: passed\nchecks:\n  - name: required-component\n    status: " + component + "\n"
			if err := os.WriteFile(filepath.Join(arena, "runs", runID, taskID+".evaluation.yaml"), []byte(record), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := analyzeTask(arena, nativeRecord{RunID: runID, Project: "p", Arm: "a", Trial: 1, TaskID: taskID}, taskCard{ID: taskID}, true)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.EvaluationEvidenceConflicts) != 1 {
				t.Fatalf("conflicts = %#v, want one", got.EvaluationEvidenceConflicts)
			}
			if component != "failed" && (got.EvaluationStatus != component || got.CountedOutcome != component) {
				t.Fatalf("%s component was normalized to evaluation=%s outcome=%s", component, got.EvaluationStatus, got.CountedOutcome)
			}
		})
	}
}

func TestContradictoryPassIsIncompleteDespiteCompletedActorAndSuccessfulHelper(t *testing.T) {
	arena := t.TempDir()
	runID, taskID := "fixture", "01"
	for _, dir := range []string{filepath.Join(arena, "runs", runID), filepath.Join(arena, "raw", runID, taskID)} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	eval := "task: '01'\nstatus: passed\nchecks:\n  - name: required-vector\n    status: failed\n"
	if err := os.WriteFile(filepath.Join(arena, "runs", runID, taskID+".evaluation.yaml"), []byte(eval), 0600); err != nil {
		t.Fatal(err)
	}
	helper := `{"task_id":"01","commands":[["test"]],"exit_code":0,"snapshot":"snapshots/attempt-0","actor_attempt":1}` + "\n"
	if err := os.WriteFile(filepath.Join(arena, "raw", runID, taskID, "helper-attempt-0.jsonl"), []byte(helper), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := analyzeTask(arena, nativeRecord{RunID: runID, Project: "p", Arm: "a", Trial: 1, TaskID: taskID, ActorStatus: "completed"}, taskCard{ID: taskID}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.AggregateEvaluationStatus != "passed" || got.Checks[0].Status != "failed" || len(got.EvaluationEvidenceConflicts) != 1 {
		t.Fatalf("raw aggregate/component evidence was lost: %#v", got)
	}
	if got.PublicHelper.Status != "success" || got.CountedOutcome != "incomplete" {
		t.Fatalf("helper/outcome = %#v / %q", got.PublicHelper, got.CountedOutcome)
	}
	count := counts{}
	countOutcome(&count, got.CountedOutcome, false)
	if count.Passed != 0 || count.Incomplete != 1 {
		t.Fatalf("contradictory outcome counts = %#v", count)
	}
}

func TestUnknownAndOwnerDecisionStayOutOfPassCounts(t *testing.T) {
	if got := normalizeStatus("unavailable"); got != "unknown" {
		t.Fatalf("unavailable normalized to %q", got)
	}
	c := counts{}
	countOutcome(&c, "owner_decision_required", true)
	if c.OwnerDecisionRequired != 1 || c.Passed != 0 || c.Unknown != 0 {
		t.Fatalf("owner-decision counts = %#v", c)
	}
	rows := []taskReport{
		{Project: "p", Arm: "A", Trial: 1, TaskID: "01", EvaluationStatus: "passed", CountedOutcome: "passed"},
		{Project: "p", Arm: "A", Trial: 1, TaskID: "02", EvaluationStatus: "unknown", CountedOutcome: "unknown"},
		{Project: "p", Arm: "A", Trial: 1, TaskID: "03", EvaluationStatus: "passed", CountedOutcome: "passed"},
	}
	g := makeGroups(rows)[0]
	if g.PassingStreak != 1 || g.StreakStatus != "unavailable" || g.StreakBoundary != "unknown outcome at task 02" {
		t.Fatalf("streak = %#v", g)
	}
	arena := t.TempDir()
	unknown, err := analyzeTask(arena, nativeRecord{RunID: "run", Project: "p", Arm: "a", Trial: 1, TaskID: "01"}, taskCard{ID: "01"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if unknown.EvaluationStatus != "unknown" || unknown.NativeTokensStatus != "unavailable" || unknown.AttentionStatus != "unavailable" {
		t.Fatalf("missing evidence = %#v", unknown)
	}
	if err := os.MkdirAll(filepath.Join(arena, "runs", "run"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(arena, "runs", "run", "02.evaluation.yaml"), []byte("task: '02'\nstatus: owner_decision_required\nowner_decision: external assessment required\n"), 0600); err != nil {
		t.Fatal(err)
	}
	owner, err := analyzeTask(arena, nativeRecord{RunID: "run", Project: "p", Arm: "a", Trial: 1, TaskID: "02"}, taskCard{ID: "02", RequiresOwnerDecision: true}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !owner.OwnerDecisionRequired || owner.EvaluationStatus != "owner_decision_required" {
		t.Fatalf("owner decision record = %#v", owner)
	}
	if invalidationApplies(invalidation{FreezeDigest: "abc", Scope: "all-native-records", Reason: "relative paths", ComparisonEligible: false}, "abc") == false {
		t.Fatal("matching frozen invalidation did not apply")
	}
	if invalidationApplies(invalidation{FreezeDigest: "other", Scope: "all-native-records", ComparisonEligible: false}, "abc") {
		t.Fatal("invalidation from a different freeze applied")
	}
	if err := os.WriteFile(filepath.Join(arena, "runs", "run", "03.evaluation.yaml"), []byte("task: '03'\nstatus: passed\nchecks: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	withoutHelper, err := analyzeTask(arena, nativeRecord{RunID: "run", Project: "p", Arm: "a", Trial: 1, TaskID: "03", ActorStatus: "completed"}, taskCard{ID: "03"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if withoutHelper.EvaluationStatus != "passed" || withoutHelper.CountedOutcome != "incomplete" {
		t.Fatalf("missing helper counted as %q; raw evaluation %q", withoutHelper.CountedOutcome, withoutHelper.EvaluationStatus)
	}
}

func TestChurnPathOwnersKeepCategoriesSeparate(t *testing.T) {
	cases := []struct{ project, arm, path, want string }{
		{"vertical-slices", "B", ".agents/skills/orders.md", "generated-projections"},
		{"vertical-slices", "B", "resources/orders.usecase.yaml", "authored-governance"},
		{"vertical-slices", "B", "markitect-artifacts.yaml", "check-configuration"},
		{"modular-service", "A", "internal/modules/orders/orders_test.go", "tests"},
		{"modular-service", "B", ".markitect/exceptions/waiver.yaml", "exceptions"},
	}
	for _, tc := range cases {
		if got := categoryFor(tc.project, tc.arm, tc.path); got != tc.want {
			t.Errorf("categoryFor(%s): got %s, want %s", tc.path, got, tc.want)
		}
	}
	ba, bd, la, ld := lineDelta([]byte("old\nkeep\n"), []byte("new\nkeep\nextra\n"))
	if ba != 10 || bd != 4 || la != 2 || ld != 1 {
		t.Fatalf("delta = bytes +%d/-%d lines +%d/-%d", ba, bd, la, ld)
	}
	for _, tc := range []struct {
		before, after      string
		addBytes, delBytes int64
		addLines, delLines int
	}{
		{"a\nb\nc\n", "a\nx\nc\n", 2, 2, 1, 1},
		{"a\nb\n", "b\n", 0, 2, 0, 1},
		{"b\n", "a\nb\n", 2, 0, 1, 0},
		{"same\n", "same\n", 0, 0, 0, 0},
	} {
		ab, db, al, dl := lineDelta([]byte(tc.before), []byte(tc.after))
		if ab != tc.addBytes || db != tc.delBytes || al != tc.addLines || dl != tc.delLines {
			t.Errorf("delta %q -> %q = %d/%d bytes %d/%d lines", tc.before, tc.after, ab, db, al, dl)
		}
	}
}
