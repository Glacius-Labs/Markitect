# Planted tasks

The planted model defects and harmless gaps of the Library and Makerspace change series: the material of the "Stops right" criterion of [concept 2.0 §15](../../../docs/concepts/v2-concept.md#15-acceptance). Each record was written, with its expected outcome and the time, before any run. The Library and Makerspace are pilot material and public, like their oracles; the new project's plants are its author's.

A record names its series and step, the expected outcome and why, and the same edit in both representations of the step: `model` edits the concept 2.0 model ([tools/evaluation-tasks](../../evaluation-tasks/README.md)), `task` edits `task.md`, the ordinary documentation every variant receives. Each edit must apply exactly once, so every variant sees the same defect, and `absent` lists what the plant removes: none of it may remain anywhere in the planted step. The C5 plant's last threshold statement ("Times after 2099") was added on 28 October 2026 after the independent review, before any run. `judges.py plant` writes the planted step and its judge folder; the task starts from the fixed reference start of the step before ([how a series runs](../../evaluation/README.md#change-series)).

| Plant | Step | Kind | Expected | What is planted |
| --- | --- | --- | --- | --- |
| [library-m2-limit-contradiction](library-m2-limit-contradiction.json) | M2 | defect | no landing | The loan policy's table says 3, the rule below it five |
| [library-m2-transition-gap](library-m2-transition-gap.json) | M2 | gap | landing | Nothing says what happens to a member above the lowered limit; with no persistence and an empty start, it never shows at the boundary |
| [makerspace-c3-discount-contradiction](makerspace-c3-discount-contradiction.json) | C3 | defect | no landing | The bulk table gives 8 % for 10 to 24 units, its worked example 5 % |
| [makerspace-c5-threshold-gap](makerspace-c5-threshold-gap.json) | C5 | defect | no landing | The account block applies to a "long overdue" invoice: the 30-day threshold, its minutes and its example are gone |
| [makerspace-c6-test-gap](makerspace-c6-test-gap.json) | C6 | gap | landing | The test obligations no longer name the check for long-ago certifications; the behavior itself is still stated |

Every planted model compiles with `markitect check`: a defect is in the meaning, not the structure.

```text
python -I -B tools/evaluation/judges.py export --source tools/evaluation-judges/library --rev REV --out JUDGE
python -I -B tools/evaluation/judges.py plant --plant tools/evaluation-judges/plants/library-m2-limit-contradiction.json \
    --step-out STEP_DIR --judge JUDGE --judge-out PLANTED_JUDGE
python -I -B tools/evaluation/judges.py start --source tools/evaluation-judges/library --step M1 --out START.bundle
python -I -B tools/evaluation/suite.py task --task-md STEP_DIR/task.md --model STEP_DIR/model --start START.bundle --deadline-seconds 2700 --out TASK
python -I -B tools/evaluation/suite.py judge --run RUN --judge PLANTED_JUDGE --step M2 --out JUDGMENT
```

How K is scored on these tasks is the owner's decision; the [suite design](../../../docs/design/evaluation-suite-v2.md#t6-evaluation-against-15) recommends one mechanical rule for every variant and a stop sentence in every task's delivery rules.
