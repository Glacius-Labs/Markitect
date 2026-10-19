# Extract on the library: old/new comparison — 19 October 2026

**Scope:** the closing result of [#495](https://github.com/Glacius-Labs/Markitect/issues/495). It compares the first bounded Extract (Main `f15ea723`, "old") with the Extract after [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains) and the review fixes of PR #507 (build `cc89d18b`, "new"). Both use the same library realization, harness, scenarios, model (Sonnet `claude-sonnet-5-5`, high effort, every Execute and Verify turn) and assessor rubric. These are real runs in WSL. Source checks, Main integration and owner acceptance are reported separately at the end. Hosted CI was not run.

## Result

The new Extract is better on the two model-content error classes. The question result (3 of 5 against 4 of 4) is too small to count as an improvement, and unnecessary questions rose from 1 to 2. One new defect cost a run. The samples are small (old 3 plain, 3 seeded and 2 conflict runs; new 3 plain with one Incomplete, 2 seeded and 2 conflict), and no significance is claimed.

| Error class | Old (8 runs) | New (6 completed runs) |
| --- | --- | --- |
| Missing project and delivery obligations | Tests (P07) and documentation (P08) scored 0 in 7 of 8, "Go standard library only" (P12) in 7 of 8, "internals are free" (P14) in 8 of 8, agreement of code, tests and documentation with the model (P09) in 2 of 8 | P07, P08 and P12 scored 0 in no run; P14 in 5 of 6; P09 in 3 of 6 |
| Incidental detail made binding | 35–47 over-specifications per model (mean 42); evidence inside Definitions 5–14 | 17–27 (mean 22); evidence inside Definitions 0–2 |
| Escalation | The necessary owner question was asked in 3 of 5 seeded and conflict runs; 1 unnecessary question | Asked in 4 of 4; 2 unnecessary questions (one partly) |

- **Score:** old 66–72 of 92 (mean 70), new 77–81 (mean 79). No run of either build stated anything that contradicts the reference.
- **Completion:** old 8 of 8 completed. New 6 of 7. RXF-7 ended Incomplete and published nothing (see the defect below).
- **Deadline path (RXT-1, new, `executionTime` 3 minutes):** the structure phase verified, the model phase ended at the deadline. The run was Incomplete with the reason "the deadline was reached". Nothing was published. The model work volume was retained as `retainedWork`, the structure volume was closed and the homes were removed. The evidence was written and no container of the run remained. RXF-7's Incomplete end behaved the same.
- **Rebuild from the extracted model:** every plain run that completed rebuilt a program that passed the independent oracle 12/12 with discrimination. That was 3 of 3 old and 2 of 2 new.
- **Proposed checks:** 6 of 6 new runs proposed `go build` and `go test` from the README (2 also `go vet`). The Extract proposed them and the owner adopts them. Neither build runs them during Extract.
- **Project Package:** 3 new runs proposed a dedicated `project` Package. In the 3 others, `library` owns the project-wide obligations, as in the human model. The assessor credited both.

## Defect found: an unsatisfiable review finding (RXF-7)

The `project` Package has no artifact directory. Under the layout, `realization/README.md` and `go.mod` belong to Root, and Root registered them correctly. In each of four rounds, the fresh reviewer of `project` raised a blocking finding asking `project` to register both files in its own shard. Doing so is a duplicate registration, so the finding could not be satisfied. The Managers settled ownership correctly in their messages; the round budget ended and the run was Incomplete with retained work.

Cause: the model-phase Verify guidance asked reviewers to check that every artifact of their area is registered. The Host's `Account` already enforces complete registration exactly and addresses the owner who can act. **Smallest correction (this record's PR):** reviewers no longer check compilation, link resolution or complete registration; they judge only whether a registration's Definitions describe its artifact. The correction is not measured by a run.

## Residual weaknesses

- **"Internals are free" (P14)** is fully stated in no new model; RXF-1 states it partly. All bind some internal choice.
- **Agreement of code, tests and documentation with the model (P09)** scored 0 in 3 of 6 new runs, against 2 of 8 old ones.
- **Unnecessary questions:**
  - RXF-2 asked whether the documented limit of 3 is intended.
  - RXF-3 asked whether a whitespace-only identifier should be refused, which the code settles. The model left that point open after the neutral answer.
- **Questions only in prose:** In RXF-7 Root wrote owner questions into its summaries, which reach nobody. The new answer formats no longer ask for open questions, but this still happened.
- **Policy values** live once, at their owner, in 5 of 6 new models (RXF-4 repeats the 14-day period in `lending/calendar-dates`), always as prose. None models them as a structured policy Kind as the reference does.
- **"Agreed with the library board":** The seeded LENDING.md says this. Old runs read it as a recorded decision and did not ask 2 of 3 times. That is defensible. The conflict scenario without that phrase is the clean test.

## Runs

| Attempt | Build | Scenario | Extract | Score | Zero-scored | Over-spec. | Scenario items | Rebuild + oracle |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| RXB-1 | old | plain | passed, 428 s | 70 | P07 P08 P09 P12 P14 | 45 | — | passed, 12/12 |
| RXB-3 | old | plain | passed, 597 s | 72 | P12 P14 | 45 | — | passed, 12/12 |
| RXB-5 | old | plain | passed, 632 s | 68 | P07 P08 P09 P12 P14 | 42 | — | passed, 12/12 |
| RXB-2 | old | seeded | passed, 472 s | 66 | P07 P08 P10 P11 P12 P14 | 40 | S3 S4 no | — |
| RXB-4 | old | seeded | passed, 421 s | 70 | P07 P08 P12 P14 | 37 | S3 no | — |
| RXB-6 | old | seeded | passed, 711 s | 69 | P07 P08 P14 | 35 | S6 no | — |
| RXC-1 | old | conflict | passed, 718 s | 71 | P07 P08 P12 P14 | 42 | all yes | — |
| RXC-2 | old | conflict | passed, 549 s | 72 | P07 P08 P12 P14 | 47 | all yes | — |
| RXF-1 | new | plain | passed, 382 s | 81 | none | 17 | — | passed, 12/12 |
| RXF-4 | new | plain | passed, 572 s | 81 | P14 | 22 | — | passed, 12/12 |
| RXF-7 | new | plain | Incomplete, 812 s | — | — | — | — | — |
| RXF-2 | new | seeded | passed, 490 s | 79 | P09 P14 | 20 | S6 no | — |
| RXF-5 | new | seeded | passed, 408 s | 77 | P14 | 27 | all yes | — |
| RXF-3 | new | conflict | passed, 430 s | 79 | P09 P14 | 25 | C4 no | — |
| RXF-6 | new | conflict | passed, 499 s | 78 | P09 P14 | 20 | all yes | — |
| RXT-1 | new | 3-minute budget | Incomplete, 198 s | — | — | — | — | — |

## Method

- **Scenarios** (`experiments/recursive-apply-demo`):
  - `plain` is RA-D2's published A2.
  - `seeded` admits a fourth loan against a documented limit of 3. The tests no longer check it, and LENDING.md ("agreed with the library board") says 21 days against 14 days in the README, code and tests.
  - `conflict` has only a LENDING.md with 21 days that claims no decision.
  - A scripted owner answers loan-period questions with 21 days and all other questions neutrally.
  - Plain runs are adopted by a scripted commit. Apply then rebuilds the realization from the extracted model alone, with the documented build and test checks, and the [library oracle](../../experiments/library-playground/README.md) judges the result with the M2 policy.
- **Assessment:** an Opus agent that took no part in the runs scored every model against the fixed rubric `extract-checklist.md`:
  - 46 obligations of the human M2, at 0, 1 or 2 points each
  - over-specifications in six categories
  - the seeded and conflict expectations

  Scoring RX-3 twice gave 67 both times with the same zero-scored set; the count of over-specifications differed (40 against 49). The score was identical in the one re-scored run; the over-specification count is only a rough measure. The assessor belongs to the same model family as the authors.
- **Execution:** sequential, in WSL Ubuntu, with each attempt's own container store label. Sessions for Apply ran in parallel on the same machine, so durations are indicative only. Claude Code reaped the first driver for low Windows memory during RXB-3. Its rebuild ran to its end unattended, and the oracle was run afterwards. Its rebuild time is therefore not reported.
- **Builds:**
  - The old build is Main `f15ea723`.
  - The new build is `cc89d18b` of PR #507. PR #507 merged as `30bc6e87` after two Main merges. Those merges change no Extract code, Extract text or `internal/run` prompt or tool.
  - The correction above comes after these runs.

## Separate judgments

- **Source checks:** PR #507 passed `go vet`, the tests of `internal/run`, `apply`, `adapters` and `bootstrap`, `markitect check` and the architecture check (see the PR). This record's PR changes one guidance sentence and the runtime contract; the Extract tests pass.
- **Main integration:** PR #507 is on Main (`30bc6e87`) and was read back with `markitect check`. This record's PR is merged after its own checks.
- **Owner acceptance:** pending. The owner decides whether the extracted model is adoptable and what comes next.
