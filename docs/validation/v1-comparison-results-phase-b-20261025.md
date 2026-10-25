# V1 comparison results, Phase B: longitudinal Library changes

Date: 25 October 2026. This record reports Phase B of the [V1 comparison protocol](v1-comparison-protocol-20261024.md#addendum-phase-b), approved at `4708c1b55aa90cdb6b95ecc3c4ec04d1e9c8b77e` with the N = 5 option and amended before any Phase B data by [B8](v1-comparison-protocol-20261024.md#b8-pre-data-amendment-arm-m-runs-phase-b-on-f-25-october-2026). Phase B is read by its own rule (B5) and never pooled with [Phase A](v1-comparison-results-20261025.md).

## Result

**No quality advantage shown** (B5, read as section 8). Each arm conformed in all 5 of its units, the chains' T1–T4 segments (95 % CI 48–100 %). Both arms conformed in all 20 counted runs (83–100 %), and the one-sided Fisher exact p is 1.0. Per step T1, T2, T3 and T4, both arms were 5 of 5. Neither arm showed an error type, an unsupported success or a false blocker.

| Per counted run (T1–T4) | M (Markitect Apply on F′) | B (one Codex agent) |
| --- | --- | --- |
| Conforming | 20 of 20 | 20 of 20 |
| Time | 587–810 s, mean 665 s | 25–109 s, mean 63 s |
| Tokens | 6.5–11.0 M, mean 8.0 M | 0.48–1.36 M, mean 0.87 M |
| Output tokens, total | 546 k | 82 k |
| Messages | 408 in total, 18–28 per Apply | 20, one per run |

- **Behavior was identical.** Every candidate of both arms passed every carried and new hidden case of its step: 21 at T1, 27 at T2, 26 at T3 and T4. That includes the composition check, overdue assessment after a renewal. Every candidate also passed its own build and tests under both limit profiles. Both arms' T3 runs removed member suspension from code, tests and documentation, and no review found stale suspension behavior or documentation. Every T4 met strict no-change; M's registry census was empty too.
- **Cost.** M took about 10.5 times the time and 9.2 times the tokens of B per counted run, and 6.7 times the output tokens. B's per-run tokens grow along a chain because a resumed thread carries its history. Section 8 counts equal quality at higher cost against the claim.
- **Change size.** M changed more: 2,979 comparable lines in the counted runs against B's 1,466, about twice as many at every step. M's reviewers classed 10 changes as *optional* (added tests or a comment edit), B's none. M also changed 25 registry files across the counted runs, which its accounting requires.

**Conclusion.** Phase B was designed to find where one agent loses track over successive cross-area changes: short-loan copies, renewals and the removal of suspension, followed by an unchanged re-application. It found nothing of the kind. One Codex agent continuing its own thread met every hidden obligation in every step of five chains, as did Markitect on F′, at about one tenth of the time and tokens. On this suite and with this model the measured result is a tie at a large cost difference. That counts against the claim that Markitect V1 is better than one agent; it does not test larger projects, which Phase C addresses.

## Execution

- **Sources.** Arm B ran with the Phase A runner, prompt and image (`sha256:1271a363…0ba9`) and the reviewed Phase B materials at `43d0365e`. Arm M ran on F′ = `528ef346` (B8), with its own runtime image `sha256:dd0ddd70…` built from F′, so the shared-image condition does not hold for Phase B (B8 limit). M's chains 1–3 continued from their Phase A S3 results and state directories, produced on F; chains 4 and 5 ran every step on F′.
- **Order.** Arm B ran all 26 runs on 25 October from 06:57 to 07:52 UTC. Arm M ran all 26 from 09:08 to 14:15 UTC. B8 records why the counterbalanced pairs were given up. Each arm kept the chain order: chains 1–3 T1–T4, then chains 4 and 5 with S1–S3 first.
- **One attempt per run.** No run was replaced, and no writer run failed.
- **Fixtures.** B's fixture for each step was its own previous capture plus one commit replacing `requirements.md` with that step's rendering (m3 `e6454eaa…`, m4 `fd5fa5e1…`, m5 `058372a0…`). M's was its own previous candidate plus one neutral commit replacing `.markitect/model/` wholesale with the materialized snapshot (manifests m3 `5f23c592…`, m4 `ba1b6ac0…`, m5 `7b93c978…`). T4 is the T3 result unchanged in both arms.
- **Assessment.** T steps were assessed with the Phase B assessor at `43d0365e` (`phase_b_cases.py` through the oracle's transport, with the Library oracle at F). The extra chains' S steps used the Phase A assessor at R. Each candidate also got its own build and tests, the census, the scope greps and one blinded review with a citation check. No reviewer reported a critical finding in any of the 52 runs, so no confirmation review was needed.
- **Hygiene.** No residue, no protected input changed, no build residue in any final candidate, no scope-grep hit, and no human intervention during a run. Step 4's own review noted a go-build binary in intermediate commits of chain 2; it is not in any final candidate.

## Every run

Time: M Apply seconds, B session seconds. Tokens: M provider usage summed over the Apply's threads; B per-run input plus output tokens (the resumed thread's total minus the previous step's). Files: comparable files changed against the start (M registry files in brackets). S rows are the extra chains' S1–S3, additional observations that are not counted (condition 1).

| Run | M candidate | M s | M tokens | M msgs | M files | M conf. | B candidate | B s | B tokens | B files | B conf. |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| c1-T1 | `aa2280db` | 718 | 7,549,219 | 20 | 8 (+0) | yes | `b8f864d6` | 75 | 793,824 | 3 | yes |
| c1-T2 | `0c81ea71` | 810 | 9,157,134 | 22 | 8 (+1) | yes | `46751d5f` | 88 | 990,663 | 3 | yes |
| c1-T3 | `551743f9` | 681 | 7,293,294 | 18 | 9 (+1) | yes | `d4764966` | 56 | 1,363,380 | 3 | yes |
| c1-T4 | `d439291c` | 634 | 7,108,813 | 18 | 0 (+0) | yes | `bf39b4a4` | 25 | 637,971 | 0 | yes |
| c2-T1 | `1788e344` | 687 | 8,704,941 | 25 | 9 (+1) | yes | `f9897ce7` | 60 | 542,118 | 3 | yes |
| c2-T2 | `dc601f9e` | 694 | 8,352,941 | 18 | 9 (+4) | yes | `d75edc2a` | 75 | 691,757 | 3 | yes |
| c2-T3 | `5e21594d` | 788 | 10,957,076 | 28 | 8 (+3) | yes | `80bf22b6` | 58 | 964,606 | 3 | yes |
| c2-T4 | `1e30decf` | 671 | 7,196,705 | 18 | 0 (+0) | yes | `7c401e0f` | 27 | 607,240 | 0 | yes |
| c3-T1 | `3944c5dd` | 711 | 7,719,409 | 22 | 7 (+0) | yes | `f47e9c52` | 82 | 737,719 | 3 | yes |
| c3-T2 | `80d961bf` | 600 | 7,576,204 | 24 | 5 (+1) | yes | `7c2c8fe8` | 87 | 709,926 | 3 | yes |
| c3-T3 | `deca0d03` | 625 | 7,698,348 | 18 | 7 (+2) | yes | `ccfacf68` | 68 | 1,317,654 | 3 | yes |
| c3-T4 | `deca0d03` | 592 | 6,520,878 | 18 | 0 (+0) | yes | `9026f430` | 26 | 479,818 | 0 | yes |
| c4-S1 | `ef69dc6c` | 999 | 8,705,447 | 22 | 13 (+6) | yes | `7269006d` | 161 | 379,976 | 5 | yes |
| c4-S2 | `a7cc22c2` | 590 | 6,921,935 | 19 | 4 (+0) | yes | `e0ea8df2` | 96 | 524,771 | 3 | yes |
| c4-S3 | `a784444e` | 599 | 6,468,316 | 18 | 0 (+0) | yes | `f6b83a72` | 29 | 298,272 | 0 | yes |
| c4-T1 | `961924ed` | 647 | 7,874,789 | 19 | 8 (+1) | yes | `b1453134` | 76 | 739,041 | 3 | yes |
| c4-T2 | `05d62743` | 636 | 9,645,018 | 24 | 6 (+1) | yes | `bb0862b5` | 88 | 932,070 | 3 | yes |
| c4-T3 | `12e497f2` | 725 | 9,816,754 | 22 | 7 (+3) | yes | `ff4bbf9a` | 61 | 1,320,090 | 3 | yes |
| c4-T4 | `12e497f2` | 587 | 7,259,353 | 18 | 0 (+0) | yes | `c66325a0` | 28 | 676,929 | 0 | yes |
| c5-S1 | `d8fbef6e` | 694 | 7,247,725 | 21 | 14 (+6) | yes | `ec04e6cd` | 243 | 425,498 | 5 | yes |
| c5-S2 | `e14b5843` | 605 | 7,453,427 | 19 | 4 (+0) | yes | `243cb06c` | 57 | 469,155 | 3 | yes |
| c5-S3 | `5e0c201b` | 570 | 6,275,587 | 18 | 0 (+0) | yes | `014c8f78` | 33 | 342,612 | 0 | yes |
| c5-T1 | `805df4da` | 638 | 7,367,997 | 22 | 8 (+2) | yes | `01756639` | 78 | 845,379 | 3 | yes |
| c5-T2 | `b069ebab` | 645 | 7,855,181 | 18 | 6 (+2) | yes | `24c51e78` | 109 | 1,115,843 | 3 | yes |
| c5-T3 | `2e732a9d` | 628 | 7,700,869 | 18 | 10 (+3) | yes | `c4b2e5c5` | 72 | 1,232,582 | 3 | yes |
| c5-T4 | `ba82345a` | 587 | 7,145,575 | 18 | 0 (+0) | yes | `0d122d6b` | 26 | 686,901 | 0 | yes |

## Secondary outcomes

- **Hidden families:** every family passed in every run, so neither arm missed an obligation. That covers T1 loan classes, T2 renewals (including the suspended-member refusal at M4 and the composition check) and T3 no member status, plus the carried E01–E12, the witness and the Phase A probes.
- **Stale obligations at T3:** none in either arm; the blind reviews found no documentation or test that still describes suspension as existing behavior.
- **Regressions of earlier families:** none in either arm.
- **Unnecessary changes (counted runs):** R5 classes M 105 required, 10 optional, 0 unjustified; B 45 required, 0 optional, 0 unjustified. Comparable lines changed: M 2,979, B 1,466. In the extra S steps, B-c4-S2 made two optional changes (a byte-wise reader and its test) and M-c4-S2 one.
- **R3b test topics:** 6 of 6 except M-c2-T3, M-c3-T3, B-c1-T1, B-c5-T3 and B-c5-T4 (5 of 6). At T3 and T4 the reviewers disagreed whether suspension tests apply once suspension is removed.
- **Extra chains, S1–S3 (not counted):** all 12 runs conform in both arms, in line with Phase A.
- **M-specific:** no Verify Reject branch and no correction branch in any Apply. Findings per Apply (Assessment / Implement / Verify) ranged 2–15 / 11–18 / 6–18. Notes written (`append_note`) per Apply: 1–11, across all 26. Questions between Managers (`ask`): 0–4.

## Deviations and incidents, in sequence

1. **B8 pre-data amendment (before any Phase B run).** On F the Apply preflight refused M's T3 input, because the registry links to the deleted Definition were invalid; this is recorded as a finding of F. Arm M therefore ran Phase B on F′, which also contains #696, and arm B ran first, so the counterbalanced pairs were given up. F′ was recorded in B8 only after M-c1-T1 had started, so the predeclared timing of that record was missed.
2. **Arm-B driver path error (B-c1-T1, 06:57 UTC).** The first driver version stopped after the run, because Windows Python could not open `/c/...` paths. The session itself completed normally and remains the run's only attempt. The corrected driver never re-runs an existing attempt.
3. **Assessor infrastructure failures (arm M, 14:19 and 14:22 UTC).** The rule, per the coordinator: a repeat is allowed only for a failure shown to be assessor infrastructure. The same rule holds for arm B. Both attempts stay in the record with their causes. If a failure cannot be clearly separated from the candidate, the first result counts.
   - At 14:19 all 26 first M assessments failed inside the assessor's scratch repository before any candidate was exported, built or run. The cause was a carriage return on the start commit read by the operator script.
   - At 14:22 the repeat for M-c1-T1 failed when the hidden-case build could not write to the assessor's own fresh volume ("permission denied"). The candidate binary never ran, and its own build and tests passed in the same assessment.
   - All 26 repeats then ran cleanly.
   - Arm B had no failed or repeated assessment in Phase A or Phase B.
   - The evidence is retained under `phase-b\infra-failures\`.

## Limits

- Phase A's section 14 applies.
- The B8 limits also apply: arms on different sources, different runtime images, and time-separated rather than counterbalanced.
- The suite is still the small Library project. Behavior again hit the ceiling in both arms, so Phase B separated the arms only on cost and change size.
- The blind reviewers interpreted the suspension test topic (R3b item 5) after its removal inconsistently; it is secondary.
- The analyst was not blinded; the reading is computed by [`analyze.py`](../../tools/comparison/analyze.py) `--phase B` from the ledger.

## Retained evidence

Root `C:\Users\Consiliari\Documents\Codex\Markitect-V1-Comparison\e11e6cff883b\phase-b\`:

- `ledger-phase-b.json`, combining `ledger-m.json` (built by `build-ledger-m.py`) and `ledger-b.json` (built by `build-ledger-b.py`), plus `analysis-phase-b.json`.
- `runs\B-*` and `..\runs\B-c4-S*`, `..\runs\B-c5-S*` for arm B's runs.
- `assess\`, `..\assess\M-c4-S*`, `..\assess\M-c5-S*` and `..\assess\B-c*-S*` for the assessments.
- `reviews\` for the reviews and their citation checks.
- `snapshots\` for M's model snapshots with their manifests, and `infra-failures\` for the failure evidence.
- The operator scripts: `drive-b-chain.frozen.sh`, `drive-b-s-chain.frozen.sh`, `assess-m-chain.sh` and `save-review.sh`.

M evidence stays under step 4's `Markitect-V1-Requalification\528ef346\packets\m-*\attempt-01\`.
