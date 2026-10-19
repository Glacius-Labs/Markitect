# Apply stability after the #492 findings — 19 and 20 October 2026

**Scope:** the second iteration on Apply, after the [#492 evidence](apply-evidence-haiku-20261019.md). Three findings of that report were changed and then tested again in real runs on the library example: model defects, no idempotent no-change, and costly cross-area repairs. Source checks, these runs, Main integration and owner acceptance are reported separately. No owner acceptance is recorded.

## Verdict

On the final build, Apply ran stable on the library:
- Three consecutive Haiku runs of M1 → A1 → M2 → A2 each verified both Applies in one Verify round, with the oracle at 12/12 for A1 and A2 and no blocking finding.
- A Sonnet run of the same scenario gave the same result in 10 minutes instead of 17.
- Drift was repaired with only the necessary changes.
- A model contradiction and a boundary gap ended the run within one to three minutes with a model-defect report.
- A conforming stand stayed unchanged.

The evidence covers one example and two models of one family.

## Changes

| PR | Change | Why |
| --- | --- | --- |
| #515 | Root's tool `model_defect` ends the run Incomplete without publishing ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)). The mandate states that the model always holds, that contradictions and boundary gaps go up instead of being decided, and that a conforming stand stays unchanged. | RA-C1 continued on a run-only answer, RA-Q1 decided boundary gaps itself, and RA-N2 changed a conforming stand. |
| #518 | Apply asks the owner nothing mid-run; `model_defect` is Root's one way out (owner decision, DEC-063 amendment). | RA-Q2 asked a boundary gap as an ordinary question and continued with the answer. |
| #520 | A parent's foundation is project scaffolding, never a realization of its children's Definitions. A rule several children need is provided by its owning child in a usable part. | Root repeatedly put library-owned input rules into its root package (RA-H1, RA-C1, RA-Q1, RA-H6); the late repair cost RA-H6's deadline. |

## Runs

Total time covers the whole scenario, including preparation, both Applies of a two-Apply run, and the scripted owner step. All runs use Haiku `claude-haiku-5-5` at high effort unless marked Sonnet, with the harness and setup of the [#492 record](apply-evidence-haiku-20261019.md). For #515 and #518, the tested build `7005a4bd` has the same Apply code as Main after #518. For #520, the tested build `4cf58479` has the same Apply code as Main after #520; its second commit changes only wording.

| Run | Build | Scenario | Outcome | Rounds | Total time | Oracle |
| --- | --- | --- | --- | --- | --- | --- |
| RA-C2 | `6e759baa` (#515) | model contradiction | model defect reported by Root, run ended | 0 | 71 s | – |
| RA-Q2 | `6e759baa` (#515) | no bound owner decisions | Verified, but the boundary gap was asked as an ordinary owner question (led to #518) | 1 | 672 s | 12/12 |
| RA-R4 | `6e759baa` (#515) | drift: limit bug + unregistered file | Verified; only the two drifted files changed | 1 | 375 s | 12/12 |
| RA-C3 | `7005a4bd` (#518) | model contradiction | model defect, run ended | 0 | 83 s | – |
| RA-Q3 | `7005a4bd` (#518) | no bound owner decisions | model defect: boundary gap (due date after 9999-12-31), run ended | 0 | 177 s | – |
| RA-N3 | `7005a4bd` (#518) | conforming stand (RA-H4's A2) | Verified, no change | 1 | 346 s | m2 12/12 |
| RA-R5 | `7005a4bd` (#518) | drift: limit bug + unregistered file | Verified | 1 | 373 s | 12/12 |
| RA-H4, RA-H5 | `7005a4bd` (#518) | M1 → A2, two runs in parallel | both Applies Verified | 2+1, 1+1 | 1202 s, 990 s | 12/12, 12/12 |
| RA-H6 | `7005a4bd` (#518) | M1 → A2, parallel | A1 Verified; A2 Incomplete at the shared deadline after three blocking placement findings (led to #520) | 1, 1 | 1516 s | A1 12/12 |
| **RA-H7, RA-H8, RA-H9** | `4cf58479` (#520) | M1 → A2, sequential | **both Applies Verified, one round each, no blocking finding** | 1+1 each | 1064, 1043, 1017 s | 12/12, 12/12 each |
| **RA-S1 (Sonnet)** | `4cf58479` (#520) | M1 → A2 | both Applies Verified, one round each | 1+1 | 604 s | 12/12, 12/12 |
| **RA-R6** | `4cf58479` (#520) | drift: limit bug + unregistered file | Verified; only the two drifted files changed | 1 | 341 s | 12/12 |

In each two-Apply run, A2 assessed under the m1 profile failed the limit cases (10/12), as expected, because the published behavior followed the model change.

## Remaining limits

- **Narrow evidence.** n = 3 for the Haiku two-Apply run on the final build, on one small example, with the bound owner decisions of the demonstration. The other test projects (Roombook, Readinglog) are deferred by the delivery plan.
- **Parallel load stretches time.** Two parallel Haiku runs on one machine made A1 take up to 15 minutes. RA-H6 then missed the shared 26-minute window for both Applies. Sequential runs took 17 to 18 minutes.
- **The demonstration's owner decisions** settle two boundary gaps of the library model. Under DEC-063 these belong in the model; without them, Apply now stops with a model defect (RA-Q3), which is the intended behavior.
- **Verify is same-family review.** It caught a real defect (an empty request id accepted, RA-H4). It did not catch Root's placement in RA-H6's A1, only later in A2.

## Evidence kinds, kept apart

- **Source checks:** `go vet` and `go test ./internal/...` were green, `markitect check` and the doc links passed, and each PR had an independent review.
- **Real runs:** the table above. Run directories and transcripts are outside the repository.
- **Main integration:** #515, #518 and #520 are merged.
- **Owner acceptance:** none recorded.
