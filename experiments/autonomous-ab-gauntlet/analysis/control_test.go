package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestDeadlineOverrunCannotCountAsTimelyPass(t *testing.T) {
	for _, integration := range []bool{false, true} {
		for _, completed := range []string{"2026-10-04T19:30:00Z", "2026-10-04T19:30:00.001Z"} {
			arena := t.TempDir()
			n := nativeRecord{RunID: "r", Project: "p", Arm: "a", Trial: 1, TaskID: "08", ActorStatus: "completed", ParallelIntegration: integration, ActorCompletedUTC: completed, DeadlineUTC: "2026-10-04T19:30:00Z"}
			evidenceFile(t, arena, "runs/r/08.evaluation.yaml", "task: '08'\nstatus: passed\nchecks: []\n")
			path := "raw/r/08/helper-attempt-0.jsonl"
			if integration {
				path = "raw/r/08.attempt-0.helper.jsonl"
			}
			evidenceFile(t, arena, path, "{\"task_id\":\"08\",\"exit_code\":0,\"actor_attempt\":1}\n")
			got, err := analyzeTask(arena, n, taskCard{ID: "08"}, true)
			if err != nil {
				t.Fatal(err)
			}
			if got.AggregateEvaluationStatus != "passed" || got.PublicHelper.FinalStatus != "success" {
				t.Fatalf("raw evidence changed: %#v", got)
			}
			if completed == n.DeadlineUTC {
				if got.CountedOutcome != "passed" || got.DeadlineCompliance != "within-deadline" {
					t.Fatalf("within: %#v", got)
				}
			} else if got.CountedOutcome != "incomplete" || got.DeadlineCompliance != "overrun" {
				t.Fatalf("late pass: %#v", got)
			}
		}
	}
	if deadlineCompliance(nativeRecord{TimeoutExceeded: true}) != "overrun" {
		t.Fatal("explicit timeout was ignored")
	}
	if deadlineCompliance(nativeRecord{ActorCompletedUTC: "invalid"}) != "unavailable" {
		t.Fatal("invalid timestamp manufactured compliance")
	}
}

func TestRunExclusionRequiresExactFreezeAndUniqueIdentities(t *testing.T) {
	arena := t.TempDir()
	freeze := strings.Repeat("f", 64)
	runs, status, hash, err := readRunExclusions(arena, freeze)
	if err != nil || status != "unavailable" || len(runs) != 0 || hash != "" {
		t.Fatalf("missing: %v %s %s %v", runs, status, hash, err)
	}
	evidenceFile(t, arena, "decisions/run-exclusions.yaml", "freeze_digest: "+freeze+"\nruns:\n  - run_id: p-a-t01\n    reason: asymmetric controller error\n  - run_id: p-b-t01\n    reason: paired comparison exclusion\n")
	runs, status, hash, err = readRunExclusions(arena, freeze)
	if err != nil || status != "matched" || len(runs) != 2 || len(hash) != 64 || runs["p-a-t01"] == "" || runs["p-a-t02"] != "" {
		t.Fatalf("matched: %v %s %s %v", runs, status, hash, err)
	}
	runs, status, staleHash, err := readRunExclusions(arena, strings.Repeat("0", 64))
	if err != nil || status != "unmatched-freeze" || len(runs) != 0 || staleHash != hash {
		t.Fatalf("wrong freeze: %v %s %s %v", runs, status, staleHash, err)
	}
	for _, entries := range []string{
		"  - run_id: p-a-t01\n    reason: one\n  - run_id: p-a-t01\n    reason: two\n",
		"  - run_id: ../outside\n    reason: unsafe\n",
		"  - run_id: p-a-t01\n    reason: ''\n",
	} {
		evidenceFile(t, arena, "decisions/run-exclusions.yaml", "freeze_digest: "+freeze+"\nruns:\n"+entries)
		if _, _, _, err := readRunExclusions(arena, freeze); err == nil {
			t.Fatalf("invalid exclusion accepted: %q", entries)
		}
	}
}

func TestExcludedRunRetainsRawPassAndDoesNotExcludeOtherTrial(t *testing.T) {
	arena, outParent := t.TempDir(), t.TempDir()
	static := map[string]string{
		"protocol.yaml":            "id: fixture\n",
		"analysis-contract.yaml":   "id: descriptive\n",
		"projects/p/project.yaml":  "shared:\n  taskSet: task-set.yaml\n",
		"projects/p/task-set.yaml": "tasks:\n  - id: '01'\n",
	}
	var inputs []freezeInput
	for path, content := range static {
		evidenceFile(t, arena, path, content)
		hash, err := hashFile(filepath.Join(arena, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, freezeInput{Path: path, SHA256: hash})
	}
	frozen := struct {
		Inputs []freezeInput `yaml:"inputs"`
		Digest string        `yaml:"digest"`
	}{inputs, freezeDigest(inputs)}
	bytes, err := yaml.Marshal(frozen)
	if err != nil {
		t.Fatal(err)
	}
	evidenceFile(t, arena, "freeze.yaml", string(bytes))
	evidenceFile(t, arena, "decisions/run-exclusions.yaml", "freeze_digest: "+frozen.Digest+"\nruns:\n  - run_id: p-a-t01\n    reason: paired controller exclusion\n")
	for index, run := range []string{"p-a-t01", "p-a-t02"} {
		n := nativeRecord{RunID: run, Project: "p", Arm: "a", Trial: index + 1, TaskID: "01", ActorStatus: "completed"}
		bytes, err = yaml.Marshal(n)
		if err != nil {
			t.Fatal(err)
		}
		evidenceFile(t, arena, "runs/"+run+"/01.native.yaml", string(bytes))
		evidenceFile(t, arena, "runs/"+run+"/01.evaluation.yaml", "task: '01'\nstatus: passed\nchecks: []\n")
		evidenceFile(t, arena, "raw/"+run+"/01/helper-attempt-0.jsonl", "{\"task_id\":\"01\",\"exit_code\":0,\"actor_attempt\":1}\n")
	}
	out := filepath.Join(outParent, "report")
	if err := analyze(arena, out); err != nil {
		t.Fatal(err)
	}
	var got report
	if err := readYAML(filepath.Join(out, "report.yaml"), &got); err != nil {
		t.Fatal(err)
	}
	if got.RunExclusionRecordStatus != "matched" || len(got.Tasks) != 2 {
		t.Fatalf("report: %#v", got)
	}
	for _, task := range got.Tasks {
		if task.AggregateEvaluationStatus != "passed" || task.PublicHelper.FinalStatus != "success" {
			t.Fatalf("raw: %#v", task)
		}
		if task.Trial == 1 {
			if task.CountedOutcome != "invalidated" || task.ComparisonExclusionReason == "" || task.RawReferenceSHA256["decisions/run-exclusions.yaml"] == "" {
				t.Fatalf("excluded: %#v", task)
			}
		} else if task.CountedOutcome != "passed" || task.ComparisonExclusionReason != "" {
			t.Fatalf("other trial excluded: %#v", task)
		}
	}
	if _, err := os.Stat(filepath.Join(arena, "report.yaml")); !os.IsNotExist(err) {
		t.Fatal("collector wrote into arena")
	}
	evidenceFile(t, arena, "decisions/oracle-run-exclusions.yaml", "freeze_digest: "+frozen.Digest+"\nruns:\n  - run_id: p-a-t02\n    reason: oracle contract exclusion\n")
	oracleOut := filepath.Join(outParent, "oracle")
	if err := analyze(arena, oracleOut); err != nil {
		t.Fatal(err)
	}
	if err := readYAML(filepath.Join(oracleOut, "report.yaml"), &got); err != nil {
		t.Fatal(err)
	}
	if got.OracleExclusionRecordStatus != "matched" || len(got.OracleExclusionRecordSHA256) != 64 {
		t.Fatalf("oracle record not separately bound: %#v", got)
	}
	for _, task := range got.Tasks {
		ref := "decisions/run-exclusions.yaml"
		if task.Trial == 2 {
			ref = "decisions/oracle-run-exclusions.yaml"
		}
		if task.CountedOutcome != "invalidated" || task.AggregateEvaluationStatus != "passed" || task.RawReferenceSHA256[ref] == "" {
			t.Fatalf("merged exclusion lost raw evidence or exact source: %#v", task)
		}
	}
	evidenceFile(t, arena, "decisions/oracle-run-exclusions.yaml", "freeze_digest: "+frozen.Digest+"\nruns:\n  - run_id: p-a-t01\n    reason: conflicting second record\n")
	if err := analyze(arena, filepath.Join(outParent, "overlap")); err == nil || !strings.Contains(err.Error(), "more than one control record") {
		t.Fatalf("hidden exclusion precedence accepted: %v", err)
	}
	evidenceFile(t, arena, "decisions/oracle-run-exclusions.yaml", "freeze_digest: "+frozen.Digest+"\nruns:\n  - run_id: p-a-t99\n    reason: absent oracle run\n")
	if err := analyze(arena, filepath.Join(outParent, "absent-oracle")); err == nil || !strings.Contains(err.Error(), "absent native run IDs: p-a-t99") {
		t.Fatalf("dangling oracle exclusion accepted: %v", err)
	}
	evidenceFile(t, arena, "decisions/oracle-run-exclusions.yaml", "freeze_digest: "+frozen.Digest+"\nruns: []\n")
	evidenceFile(t, arena, "decisions/run-exclusions.yaml", "freeze_digest: "+frozen.Digest+"\nruns:\n  - run_id: p-a-t99\n    reason: absent run\n")
	if err := analyze(arena, filepath.Join(outParent, "unmatched")); err == nil || !strings.Contains(err.Error(), "absent native run IDs: p-a-t99") {
		t.Fatalf("dangling exclusion accepted: %v", err)
	}
}

func TestExclusionRecordsDoNotEnableArbitrarySourcePaths(t *testing.T) {
	if _, _, _, err := readNamedRunExclusions(t.TempDir(), strings.Repeat("f", 64), "../external.yaml"); err == nil {
		t.Fatal("arbitrary control path accepted")
	}
}

func TestInvalidatedOwnerTaskCannotCountAsReviewedDecision(t *testing.T) {
	c := counts{}
	countOutcome(&c, "invalidated", true)
	if c.Invalidated != 1 || c.OwnerDecisionRequired != 0 {
		t.Fatalf("counts: %#v", c)
	}
	rows := []taskReport{{RunID: "r", Project: "p", Arm: "A", Trial: 1, TaskID: "01", OwnerDecisionRequired: true, CountedOutcome: "invalidated"}}
	g := makeGroups(rows)[0]
	if g.Outcomes.Invalidated != 1 || g.Outcomes.OwnerDecisionRequired != 0 || g.Outcomes.ImplementationEligibleTasks != 0 || g.StreakStatus != "unavailable" {
		t.Fatalf("group: %#v", g)
	}
}
