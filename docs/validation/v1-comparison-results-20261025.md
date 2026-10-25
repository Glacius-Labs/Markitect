# V1 comparison results, Phase A: Markitect against one agent with ordinary documentation

Date: 25 October 2026. This record reports Phase A of the [V1 comparison protocol](v1-comparison-protocol-20261024.md), approved at `fa6ef99720b73a948d6d20aa43cb6e877f006bd2`, with assessment revision R = `74b7d7df946ee35b8c746dff9aefeeb13405b1f2` and frozen V1 source F = `e11e6cff883ba16da2cddab8d54c0367cae8a5d8`. Phase B (the protocol's addendum) is read by its own rule and reported separately.

## Result

**Predeclared reading (section 8): a tendency in Markitect's favor, not established.** One-sided Fisher exact p = 0.227 on the independent units, above the 0.05 the rule requires.

| | M (Markitect Apply) | B (one Codex agent) |
| --- | --- | --- |
| Conforming units (3 chains + 3 S4 runs) | 6 of 6 (95 % CI 54–100 %) | 4 of 6 (22–96 %) |
| Conforming runs | 12 of 12 (74–100 %) | 10 of 12 (52–98 %) |
| Per scenario S1 / S2 / S3 / S4 | 3/3, 3/3, 3/3, 3/3 | 3/3, 3/3, 3/3, 1/3 |
| Error types | none | stale seeded line (2 runs) |
| Mean time per run | 584 s (Apply) | 67 s (session) |
| Mean tokens per run | 7.42 M | 0.58 M |
| Host or input messages | 258 in total, 18–33 per Apply | 12, one per run |

What the difference is, and how much weight it carries:

- **Behavior was identical.** All 24 candidates pass the oracle's 12 cases with discrimination, all six probes, and their own build and tests under both limit profiles. All six S4 repairs, in both arms, build a program byte-identical to the unseeded base (`f8f41e09…`).
- **The whole difference is one comment line.** In two of B's three S4 runs, the seeded code comment `// Notice identifies one loan that was open and overdue on the assessment date.` (`realization/lending/reminders/overdue.go`, line 9) remained after a correct repair of code, tests and README. M removed or rewrote it in all three runs. One blind reviewer named the comment as stale without scoring it; the other did not mention it.
- **The criterion that decides it changed after M-S4 r1 was observed** (see Deviations). Under the originally predeclared 16-line seed list, every S4 run of both arms fails C6, which gives M 3 of 6 and B 3 of 6 units: **no quality advantage shown**. Under the 11-line list the outcome equals the 10-line list in every run. The tendency therefore rests on a post-hoc criterion and a single comment, and should be read that way.
- **Cost.** M took about 8.8 times the time and 12.8 times the tokens of B for the same behavior. Section 8 counts equal quality at higher cost against the claim; the predeclared reading is a tendency rather than a tie only because of the comment line above.
- **Nothing in section 8's list against Markitect occurred:** no unsupported success, no false blocker, no unjustified change, every strict no-change met, no technical failure. M made two changes the reviewers classed as *optional* (added test coverage in M-r1-S4 and M-r2-S4), where B made none, and changed 310 lines in S2–S4 against B's 123.

**Conclusion.** On this small greenfield Library suite, one Codex agent given ordinary documentation produced the same behavior in every run at about one ninth of the time and one thirteenth of the tokens. Markitect's only measured edge is the removal of a stale seeded comment in a repair task. This does not establish that Markitect V1 is better than one agent; the predeclared rule reads it as a tendency, and with the original seed list it is a tie. As the protocol stated in advance (section 14), this removes the Library suite as evidence for the thesis without refuting it for larger projects; Phase B and a larger project address that.

## Execution

- **Environment.** Go runtime image `sha256:1271a363…0ba9`, Codex CLI 0.162.0, model `gpt-6-sol`, effort high, the same switches in both arms, normal account login (fingerprint unchanged before and after every B run; M per step-4 records). B ran in the Manager-session hardening with the six-Manager aggregate ceiling (6 CPUs, 12 GiB, 1,152 processes). Tools ran from a clean clone at R; packet and prompt bytes were verified before the first run.
- **Calibration** (section 5) on the retained A1 candidate: profile m1 oracle 12/12 with discrimination, probes 6/6, own checks pass; profile m2 fails the limit cases (E04 admission and the paired fourth-loan witness), as expected for an M1 candidate. No probe needed correction.
- **S4 base and controls.** Base: M-c1-S2 (`f0a94c4d`), the first M S2 candidate passing C2–C4. Approved seed: the overdue assessment includes loans returned after the assessment date, in code, its unit test and one README sentence (16 added lines). Positive control: the base passes. Negative control on the seeded tree: oracle 11/12, failing only `E10-overdue` at the returned-loan assertion, discrimination confirmed, probes 6/6, own checks pass, all 16 seeded lines present. M and B fixtures are identical in all 15 paths outside `.markitect/` and `requirements.md`.
- **Order** as section 7, one attempt per run, no replacement, no infrastructure failure, no abort. Repetition 1 ran 24 October 21:40–22:37 UTC; repetition 2 from 22:38 to 25 October 05:31, with a pause from 23:06 to 05:06 (below); repetition 3 05:31–06:29. Every engine use held `NATIVE-RUN.lock`; no writer run and assessment overlapped.
- **Contamination audit** over every command and web item in all 24 writer transcripts (M: the Apply's own transcript directory): no web access, no Git remote or GitHub use, no search for `library-oracle`, `probes` or the contract identifier. The one raw pattern hit (M-c1-S1) is a heredoc that writes the program's own source, which contains the contract identifier from the requirements; it is not a search term. Section 8's reading needs no recomputation.
- **Hygiene.** No residue after any run, no login secret in B's outputs, no Codex stream error in B's transcripts, no build residue in any census, no protected input changed, no human intervention during a run.

## Every run

Order is execution order. Time: M Apply seconds, B session seconds. Tokens: M provider usage summed over the Apply's 18 threads; B input plus output tokens of the turn. Files: comparable files changed against the start (M registry files in brackets). Seed: seeded lines remaining under the 16 / 11 / 10-line lists.

| # | Run | Candidate | Time | Tokens | Msgs | Files | Review | Seed | Conforming |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | M-c1-S1 | `ae37be37` | 591 | 6,954,140 | 21 | 15 (+6) | clean, R3b 6 | | yes |
| 2 | B-c1-S1 | `d7be33dd` | 163 | 447,856 | 1 | 6 | clean, R3b 6 | | yes |
| 3 | M-c1-S2 | `f0a94c4d` | 501 | 6,269,431 | 18 | 2 | clean, R3b 6 | | yes |
| 4 | B-c1-S2 | `91dd7d14` | 32 | 770,846 | 1 | 3 | clean, R3b 6 | | yes |
| 5 | M-c1-S3 | `4136b583` | 536 | 7,586,186 | 20 | 0 | clean, R3b 6 | | yes (no change) |
| 6 | B-c1-S3 | `f399a8a5` | 18 | 1,058,253 | 1 | 0 | clean, R3b 6 | | yes (no change) |
| 7 | M-r1-S4 | `91db7011` | 526 | 8,966,223 | 33 | 5 | clean, 1 optional | 5 / 0 / 0 | yes |
| 8 | B-r1-S4 | `65509b39` | 47 | 246,644 | 1 | 4 | clean | 5 / 0 / 0 | yes |
| 9 | B-c2-S1 | `dcd42c47` | 141 | 328,036 | 1 | 5 | clean, R3b 6 | | yes |
| 10 | M-c2-S1 | `eaa753b6` | 612 | 8,001,934 | 21 | 15 (+6) | clean, R3b 6 | | yes |
| 11 | B-c2-S2 | `c66b7b41` | 31 | 585,355 | 1 | 3 | clean, R3b 6 | | yes |
| 12 | M-c2-S2 | `f5a66799` | 562 | 6,781,200 | 23 | 4 | clean, R3b 6 | | yes |
| 13 | B-c2-S3 | `eec83aa0` | 19 | 861,083 | 1 | 0 | clean, R3b 6 | | yes (no change) |
| 14 | M-c2-S3 | `1e6eb575` | 565 | 7,071,151 | 18 | 0 | clean, R3b 6 | | yes (no change) |
| 15 | B-r2-S4 | `490f268b` | 49 | 282,329 | 1 | 4 | clean | 6 / 1 / 1 | **no: C6** |
| 16 | M-r2-S4 | `2e8a681c` | 564 | 7,622,473 | 22 | 5 | clean, 1 optional | 3 / 0 / 0 | yes |
| 17 | M-c3-S1 | `058d1261` | 826 | 8,565,779 | 22 | 13 (+6) | clean, R3b 6 | | yes |
| 18 | B-c3-S1 | `6bc12a80` | 168 | 335,220 | 1 | 5 | clean, R3b 6 | | yes |
| 19 | M-c3-S2 | `f4561987` | 578 | 7,068,228 | 19 | 4 | clean, R3b 6 | | yes |
| 20 | B-c3-S2 | `cbd39779` | 52 | 760,711 | 1 | 3 | clean, R3b 6 | | yes |
| 21 | M-c3-S3 | `f4561987` | 558 | 6,710,569 | 19 | 0 | clean, R3b 6 | | yes (no change) |
| 22 | B-c3-S3 | `025e5d28` | 34 | 1,065,075 | 1 | 0 | clean, R3b 6 | | yes (no change) |
| 23 | M-r3-S4 | `ef1d1327` | 585 | 7,482,404 | 22 | 4 | clean | 3 / 0 / 0 | yes |
| 24 | B-r3-S4 | `d624a8fa` | 44 | 216,820 | 1 | 4 | clean | 4 / 1 / 1 | **no: C6** |

"Clean" means no confirmed critical rubric finding: no reviewer reported a critical item (R1, R2, R3a, R4, unjustified R5), so no confirmation review was needed, and the mechanical citation check found nothing to validate. R3b was 6 of 6 in every run. No reviewer found an invented requirement (R7). Every S3 result has exactly its input tree; M-c3-S3 returned the input commit itself, the other five are new commits.

## Secondary outcomes

- **Per criterion:** C1–C5 held in all 24 runs; C6 failed in B-r2-S4 and B-r3-S4 only.
- **S4:** E10 passed in all six repairs. Seeded lines remaining, 16 / 11 / 10-line lists: M 5/0/0, 3/0/0, 3/0/0; B 5/0/0, 6/1/1, 4/1/1. The 16-line list differs from the other two in every S4 run: under it all six runs fail C6, because five of its lines are neutral test helpers a correct repair may keep. No run changes its outcome between the 11- and the 10-line list.
- **Unnecessary changes (S2–S4):** R5 classes M 14 required, 8 repair, 2 optional, 0 unjustified; B 13 required, 8 repair, 0 optional, 0 unjustified. Comparable lines changed M 310, B 123.
- **Strict no-change (S3):** met by all six runs; M's registry census was empty too.
- **Time including setup and cleanup:** M packets 504–830 s, B runs 22–172 s.
- **M-specific:** no Verify Reject branch and no correction branch in any Apply, so no S4 run is a P5 proof for step 4. Findings per Apply (Assessment / Implement / Verify) ranged 5–15 / 11–18 / 7–17; every Run had 6 Sessions, 6 handoffs and 14 closures. Notes content is not analysed here.
- **Human attention:** none during runs. Operator actions were starts, the shared-lock handovers between this session and step 4, and the one relaunch below.

## Deviations and incidents, in sequence

1. **Continuation (approved before any run).** Within a chain, S2 and S3 continue in each arm's normal way (M keeps the chain's state directory, B resumes its Codex thread), deviating from the original brief's fresh sessions; recorded in the protocol's approval record.
2. **C6 seed list for S4, changed post hoc.** The predeclared criterion was "no seeded line remains" over the approved 16-line list (`9a8c9e54…`). After M-S4 r1 was observed (step 4's own count showed 5 of 16 lines remaining, all of them neutral test helpers), and while B-S4 r1 was running, the coordinator decided at 22:35 UTC to score only the 11 defect-bearing lines (`29f230a3…`). An independent reviewer, blind to every candidate and to both lists, then classified the seed patch, and its 10-line set (`37d14c6a…`) became the final criterion at 22:38 UTC. It agreed with the 11-line split except for the line `if openOnDate && loan.DueDate < assessmentDate {`, which only uses the variable set on a seeded line and which it named as ambiguous. Both decisions came before this session assessed either S4 r1 candidate and before any later S4 run. The change applies identically to both arms, and the 16- and 11-line results are reported beside it for every run. The reason given was that the 16-line list included five neutral helper lines any correct repair may keep. The reading under each list is stated in the Result section.
3. **Pause in repetition 2.** After B-c2-S3 (ended 23:06 UTC), the step-4 session ended before its launch of M-S3 c2 took effect; no Apply process started. Step 4 relaunched M-S3 c2 at 05:06:44 UTC from the frozen packet. This is its first and only attempt, not a replacement; the order was kept.
4. **Repeated blind review of B-c2-S3.** A session restart stopped the first reviewer before it answered; a fresh reviewer reviewed the same token. No review output was discarded.

## Limits

The protocol's section 14 applies in full: one small greenfield project, one model and effort, one provider, one machine, and N = 3 per arm and scenario. Behavior hit the expected ceiling, so the comparison could only separate the arms on hygiene in a seeded repair. Human authoring effort is unavailable for both arms. B's prompt carried the duties Markitect's harness gives; base prompts differ by design. The analyst was not blinded; every reading here is computed by [`analyze.py`](../../tools/comparison/analyze.py) from the ledger.

## Retained evidence

Root `C:\Users\Consiliari\Documents\Codex\Markitect-V1-Comparison\e11e6cff883b\`, outside the repository: `ledger.json` (24 rows, SHA-256 `97012d26…`) and the operator record `LEDGER.md`; `analysis-phase-a.json` (the output of `analyze.py`); `runs\B-*\` (fixtures, `result.json`, transcripts, Codex session files, raw trees, capture bundles, `files.sha256`); `assess\<run>\` (assessment, oracle and probe reports, own-check logs, census, seed check); `blind\` and `blind-key\`; `reviews\` (each review and its citation check); `s4\` (fixtures, the three seed lists, same-tree check); `calibration\`; `contamination-audit.json` with its script; `seed-count.py`. M evidence stays under step 4's `Markitect-V1-Requalification\e11e6cff\packets\m-*\attempt-01\` (`run-summary.json`, Apply report, transcripts, candidate bundles).
