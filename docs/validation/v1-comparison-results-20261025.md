# V1 comparison results, Phase A: Markitect against one agent with ordinary documentation

Date: 25 October 2026. This record reports Phase A of the [V1 comparison protocol](v1-comparison-protocol-20261024.md), approved at `fa6ef99720b73a948d6d20aa43cb6e877f006bd2`, with assessment revision R = `74b7d7df946ee35b8c746dff9aefeeb13405b1f2` and frozen V1 source F = `e11e6cff883ba16da2cddab8d54c0367cae8a5d8`. Phase B (the protocol's addendum) is read by its own rule and reported separately.

## Result

**Under the protocol as approved, no quality advantage is shown. Under the seed list the coordinator substituted after the first S4 run had been observed, the reading is a tendency in Markitect's favor, not established.** Neither reading establishes that Markitect V1 is better than one agent, and Markitect cost about 9 times the time and 24 times the tokens.

| S4 criterion C6 ("no seeded line remains") | M units | B units | One-sided Fisher p | Conforming runs M / B | S4 M / B | Section 8 reading |
| --- | --- | --- | --- | --- | --- | --- |
| **16-line list, as approved** (`9a8c9e54…`) | 3 of 6 (95 % CI 12–88 %) | 3 of 6 (12–88 %) | 0.717 | 9 / 9 of 12 (43–95 %) | 0/3, 0/3 | **No quality advantage shown** |
| 10-line list, post-hoc (`37d14c6a…`, coordinator decision) | 6 of 6 (54–100 %) | 4 of 6 (22–96 %) | 0.227 | 12 / 10 of 12 (74–100 %, 52–98 %) | 3/3, 1/3 | Tendency for M, not established |

S1, S2 and S3 are 3 of 3 conforming in both arms under either list (95 % CI 29–100 %). In S4, 3 of 3 is 29–100 %, 1 of 3 is 1–91 % and 0 of 3 (both arms under the 16-line list) is 0–71 %. The 11-line list gives the same outcome as the 10-line list in every run. Neither reading has an unsupported success, a false blocker or an error type only M shows; under the 16-line list both arms share the one error type, a remaining seeded line.

**The post-hoc change.** The protocol allows, after the first counting run has been assessed, only the repair of an assessor infrastructure failure, and section 7 excludes post-hoc metrics. Replacing the seed list is outside that rule. The coordinator decided it after step 4 reported M-S4 r1 and recorded it as a disclosed correction of a defective predeclared input. The five lines it dropped are neutral test helpers, which the blind classifier also classed as neutral, and they are exactly the five lines M-S4 r1 (and B-S4 r1) had kept. The full sequence is under Deviations.

What decides the post-hoc reading:

- **Behavior was identical.** All 24 candidates pass the oracle's 12 cases with discrimination, all six probes, and their own build and tests under both limit profiles. All six S4 repairs, in both arms, build a program byte-identical to the unseeded base (`f8f41e09…`).
- **The whole difference is one comment line.** In two of B's three S4 runs, the seeded code comment `// Notice identifies one loan that was open and overdue on the assessment date.` (`realization/lending/reminders/overdue.go`, line 9) remained after a correct repair of code, tests and README. M restored it in all three runs, and B in its first, to the exact wording of the S4 base (`// Notice identifies one currently open loan overdue on the assessment date.`), which M's chain-1 S1 run wrote and the S2 base carried. The rubric's R2 excludes code comments, so only C6 could catch this line; one blind reviewer named it as stale without scoring it, the other did not mention it. The kept sentence is still literally true of the repaired code's output; calling it defect-bearing is the blind classifier's judgment.
- **Cost.** Per run, M averaged 584 s and 7.42 M tokens; B 67 s and 0.31 M tokens. M output tokens alone were 348 k in total against B's 64 k (5.4×); 91–93 % of both arms' input tokens were cached. Section 8 counts equal quality at higher cost against the claim, and that is the reading under the approved list.
- **Nothing else in section 8's list against Markitect occurred:** no unsupported success, no false blocker, no unjustified change, every strict no-change met, no technical failure. M made two changes the reviewers classed as *optional* (added test coverage in M-r1-S4 and M-r2-S4), where B made none, and changed 310 lines in S2–S4 against B's 123.

**Conclusion.** On this small greenfield Library suite, one Codex agent given ordinary documentation produced the same behavior in every run at about one ninth of the time and one twenty-fourth of the tokens. Under the protocol as approved the arms tie, which counts against the claim given the cost. Markitect's only measured edge, under a criterion changed after the first S4 result, is the removal of a stale seeded comment in a repair task. This does not establish that Markitect V1 is better than one agent. As the protocol stated in advance (section 14), it removes the Library suite as evidence for the thesis without refuting it for larger projects; Phase B and a larger project address that.

## Execution

- **Environment.** Go runtime image `sha256:1271a363…0ba9`, Codex CLI 0.162.0, model `gpt-6-sol`, effort high, the same switches in both arms, normal account login (fingerprint unchanged before and after every B run; M per step-4 records). B ran in the Manager-session hardening with the six-Manager aggregate ceiling (6 CPUs, 12 GiB, 1,152 processes). Actual CPU, memory and process use was not measured in either arm. Tools ran from a clean clone at R; packet and prompt bytes were verified before the first run. [`analyze.py`](../../tools/comparison/analyze.py), which computes the readings, is not part of R: it was committed at 21:36:48 UTC on 24 October (`ff361763`), before the first counting run at 21:40:20, and is unchanged since.
- **Calibration** (section 5), from 21:38 UTC on the retained A1 candidate: profile m1 oracle 12/12 with discrimination, probes 6/6, own checks pass; profile m2 fails the limit cases (E04 admission and the paired fourth-loan witness), as expected for an M1 candidate. No probe needed correction.
- **S4 base and controls.** Base: M-c1-S2 (`f0a94c4d`), the first M S2 candidate passing C2–C4. Approved seed: the overdue assessment includes loans returned after the assessment date, in four files (code with its comment, its unit test, the end-to-end test and one README sentence; 16 added lines). Positive control: the base passes. Negative control on the seeded tree: oracle 11/12, failing only `E10-overdue` at the returned-loan assertion, discrimination confirmed, probes 6/6, own checks pass, all 16 seeded lines present. M and B fixtures are identical in all 15 paths outside `.markitect/` and `requirements.md`.
- **Order** as section 7, one attempt per run, no replacement, no infrastructure failure, no abort. Repetition 1 ran 24 October 21:40–22:37 UTC; repetition 2 from 22:38 to 25 October 05:31, with a pause from 23:06 to 05:06 (below); repetition 3 05:31–06:29. Every engine use held `NATIVE-RUN.lock`; no writer run and assessment overlapped.
- **Contamination audit** over every command and web item in all 24 writer transcripts (M: the Apply's own transcript directory): no web access, no Git remote or GitHub use, no search for `library-oracle`, `probes` or the contract identifier. The one raw pattern hit (M-c1-S1) is a heredoc that writes the program's own source, which contains the contract identifier from the requirements; it is not a search term. Section 8's reading needs no recomputation.
- **Hygiene.** No residue after any run, no login secret in B's outputs, no Codex stream error in B's transcripts, no build residue in any census, no protected input changed, no hit of the scope greps (clock, network, file writes) in any candidate, no human intervention during a run. In B-c1-S1, B-c2-S1 and B-r2-S4, Codex's own command policy rejected one `rm -f library` ("rm -f style commands are not permitted"); no binary was left in any captured tree.

## Every run

Order is execution order. Time: M Apply seconds, B session seconds. Tokens: M provider usage summed over the Apply's 18 fresh threads; B input plus output tokens of the run. A resumed `codex exec` reports the thread's running total, so B's S2 and S3 values are that total minus the previous step's (the raw totals are kept in the ledger). Cmds / edits: B's command executions and file changes. Files: comparable files changed against the start (M registry files in brackets). Seed: seeded lines remaining under the 16 / 11 / 10-line lists. C6 under the 16-line list fails for every S4 run.

| # | Run | Candidate | Time | Tokens | Msgs | Cmds / edits | Files | Review | Seed | Conforming (10-line) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | M-c1-S1 | `ae37be37` | 591 | 6,954,140 | 21 | | 15 (+6) | clean, R3b 6 | | yes |
| 2 | B-c1-S1 | `d7be33dd` | 163 | 447,856 | 1 | 15 / 5 | 6 | clean, R3b 6 | | yes |
| 3 | M-c1-S2 | `f0a94c4d` | 501 | 6,269,431 | 18 | | 2 | clean, R3b 6 | | yes |
| 4 | B-c1-S2 | `91dd7d14` | 32 | 322,990 | 1 | 10 / 1 | 3 | clean, R3b 6 | | yes |
| 5 | M-c1-S3 | `4136b583` | 536 | 7,586,186 | 20 | | 0 | clean, R3b 6 | | yes (no change) |
| 6 | B-c1-S3 | `f399a8a5` | 18 | 287,407 | 1 | 7 / 0 | 0 | clean, R3b 6 | | yes (no change) |
| 7 | M-r1-S4 | `91db7011` | 526 | 8,966,223 | 33 | | 5 | clean, R3b 6, 1 optional | 5 / 0 / 0 | yes |
| 8 | B-r1-S4 | `65509b39` | 47 | 246,644 | 1 | 19 / 1 | 4 | clean, R3b 6 | 5 / 0 / 0 | yes |
| 9 | B-c2-S1 | `dcd42c47` | 141 | 328,036 | 1 | 11 / 4 | 5 | clean, R3b 6 | | yes |
| 10 | M-c2-S1 | `eaa753b6` | 612 | 8,001,934 | 21 | | 15 (+6) | clean, R3b 6 | | yes |
| 11 | B-c2-S2 | `c66b7b41` | 31 | 257,319 | 1 | 9 / 1 | 3 | clean, R3b 6 | | yes |
| 12 | M-c2-S2 | `f5a66799` | 562 | 6,781,200 | 23 | | 4 | clean, R3b 6 | | yes |
| 13 | B-c2-S3 | `eec83aa0` | 19 | 275,728 | 1 | 6 / 0 | 0 | clean, R3b 6 | | yes (no change) |
| 14 | M-c2-S3 | `1e6eb575` | 565 | 7,071,151 | 18 | | 0 | clean, R3b 6 | | yes (no change) |
| 15 | B-r2-S4 | `490f268b` | 49 | 282,329 | 1 | 17 / 1 | 4 | clean, R3b 6 | 6 / 1 / 1 | **no: C6** |
| 16 | M-r2-S4 | `2e8a681c` | 564 | 7,622,473 | 22 | | 5 | clean, R3b 6, 1 optional | 3 / 0 / 0 | yes |
| 17 | M-c3-S1 | `058d1261` | 826 | 8,565,779 | 22 | | 13 (+6) | clean, R3b 6 | | yes |
| 18 | B-c3-S1 | `6bc12a80` | 168 | 335,220 | 1 | 14 / 4 | 5 | clean, R3b 6 | | yes |
| 19 | M-c3-S2 | `f4561987` | 578 | 7,068,228 | 19 | | 4 | clean, R3b 6 | | yes |
| 20 | B-c3-S2 | `cbd39779` | 52 | 425,491 | 1 | 13 / 2 | 3 | clean, R3b 6 | | yes |
| 21 | M-c3-S3 | `f4561987` | 558 | 6,710,569 | 19 | | 0 | clean, R3b 6 | | yes (no change) |
| 22 | B-c3-S3 | `025e5d28` | 34 | 304,364 | 1 | 11 / 0 | 0 | clean, R3b 6 | | yes (no change) |
| 23 | M-r3-S4 | `ef1d1327` | 585 | 7,482,404 | 22 | | 4 | clean, R3b 6 | 3 / 0 / 0 | yes |
| 24 | B-r3-S4 | `d624a8fa` | 44 | 216,820 | 1 | 18 / 1 | 4 | clean, R3b 6 | 4 / 1 / 1 | **no: C6** |

"Clean" means no confirmed critical rubric finding: no reviewer reported a critical item (R1, R2, R3a, R4, unjustified R5), so no confirmation review was needed, and the mechanical citation check found nothing to validate. No reviewer found an invented requirement (R7). Every S3 result has exactly its input tree; M-c3-S3 returned the input commit itself, the other five are new commits. M-r2-S4's reviewer also called a test added inside a file classed *required* optional, which accounts for that run's 126 changed lines.

## Secondary outcomes

- **Per criterion:** C1–C5 held in all 24 runs; C6 failed in B-r2-S4 and B-r3-S4 under the 10- and 11-line lists, and in all six S4 runs under the 16-line list.
- **S4:** E10 passed in all six repairs. Seeded lines remaining, 16 / 11 / 10-line lists: M 5/0/0, 3/0/0, 3/0/0; B 5/0/0, 6/1/1, 4/1/1. The 16-line list differs from the other two in every S4 run, because five of its lines are neutral test helpers a correct repair may keep. No run changes its outcome between the 11- and the 10-line list.
- **Unnecessary changes (S2–S4):** R5 classes M 14 required, 8 repair, 2 optional, 0 unjustified; B 13 required, 8 repair, 0 optional, 0 unjustified. Comparable lines changed M 310, B 123.
- **Strict no-change (S3):** met by all six runs; M's registry census was empty too.
- **Time including setup and cleanup:** M packets 504–830 s, B runs 22–172 s.
- **Tokens:** M 89.08 M in total (88.73 M input, 91 % cached; 0.35 M output); B 3.73 M (3.67 M input, 93 % cached; 0.06 M output).
- **M-specific:** no Verify Reject branch and no correction branch in any Apply, so no S4 run is a P5 proof for step 4. Findings per Apply (Assessment / Implement / Verify) ranged 5–15 / 11–18 / 7–17; every Run had 6 Sessions, 6 handoffs and 14 closures. Notes written (`append_note`) per Apply: chain 1 S1–S3 0, 0, 0; chain 2 1, 2, 0; chain 3 1, 1, 0; S4 r1–r3 2, 6, 2. Questions between Managers (`ask`) per Apply ranged 0–8, with 8 in M-r1-S4, which accounts for much of its 33 messages.
- **Human attention:** none during runs. Operator actions were starts, the shared-lock handovers between this session and step 4, and the one relaunch below.

## Deviations and incidents, in sequence

1. **Continuation (approved before any run).** Within a chain, S2 and S3 continue in each arm's normal way (M keeps the chain's state directory, B resumes its Codex thread), deviating from the original brief's fresh sessions; recorded in the protocol's approval record.
2. **C6 seed list for S4, changed post hoc, outside the protocol's change rule.** The predeclared criterion was "no seeded line remains" over the approved 16-line list (`9a8c9e54…`). Sequence (24 October, UTC):
   - 22:33 M-S4 r1 ended; at 22:34 step 4 reported it with 5 of 16 lines kept, all five neutral test helpers, and proposed scoring only the defect-bearing lines.
   - 22:34:51–22:35:42 B-S4 r1 ran, its writer unaware of any list.
   - 22:35:09 this session recorded the same 11-line split as a proposal (`seeded-lines-rule.proposed.json`, `836a5fa9…`), derived from step 4's five named helpers; 22:35:18 the coordinator decided on the 11 defect-bearing lines, and step 4 froze them at 22:35:42 (`29f230a3…`).
   - 22:36:17–22:37:40 this session assessed M-S4 r1 and B-S4 r1. The seed check records the remaining lines whatever list is chosen; both runs keep 0 lines of the 11 and of the 10.
   - 22:36:56 a reviewer blind to every candidate and to both lists classified the seed patch from the patch, the base and the model rule; its 10-line set (`37d14c6a…`) agrees with the 11-line split except the line `if openOnDate && loan.DueDate < assessmentDate {`, which only uses the variable set on a seeded line and which it named as ambiguous. At 22:38 the coordinator made this set final.

   The coordinator recorded the change as a disclosed correction of a defective predeclared input, made after M-S4 r1 was observed, applying identically to both arms, with the 16- and 11-line results reported beside it. The decision messages are retained verbatim; the blind classifier's verdict is in step 4's run log, and its prompt and raw output are not retained.
3. **Pause in repetition 2.** After B-c2-S3 (ended 23:05:43 UTC), the step-4 session ended before its launch of M-S3 c2 took effect; no Apply process started. Step 4 relaunched M-S3 c2 at 05:06:44 UTC from the frozen packet. This is its first and only attempt, not a replacement; the order was kept.
4. **Repeated blind review of B-c2-S3.** A session restart stopped the first reviewer before it answered; a fresh reviewer reviewed the same token. No review output was discarded.
5. **Harness classifier unavailable during one review.** While the blind reviewer of M-c3-S3 ran, the agent harness reported its safety classifier unavailable. The review is read-only, its one claim of substance (an empty diff) matches the census, and its citation check passed.

## Limits

The protocol's section 14 applies in full: one small greenfield project, one model and effort, one provider, one machine, and N = 3 per arm and scenario. Behavior hit the expected ceiling, so the arms could only differ in hygiene within a seeded repair. Further limits:

- The S4 base is M's own S2 output (section 9). Every repair that removed the seeded comment, all three of M's and B's first, restored M's original wording exactly.
- Blinded trees carry arm hints: B's READMEs refer to `requirements.md`, and reviewers saw the requirements under names such as `requirements-m1.md`.
- Human authoring effort is unavailable for both arms. B's prompt carried the duties Markitect's harness gives; base prompts differ by design.
- The analyst was not blinded; every reading here is computed by `analyze.py` from the ledgers.

## Retained evidence

Root `C:\Users\Consiliari\Documents\Codex\Markitect-V1-Comparison\e11e6cff883b\`, outside the repository:

- `ledger.json` (24 rows, 10-line C6, SHA-256 `215ebbd0…`), `ledger-16line.json` (the same rows with C6 under the approved list, `b7715383…`), `analysis-phase-a.json` and `analysis-phase-a-16line.json` (the outputs of `analyze.py`), `b-tokens-per-run.json`, and the operator record `LEDGER.md`;
- `runs\B-*\`: fixtures, `result.json`, transcripts, Codex session files, raw trees, capture bundles, `files.sha256`;
- `assess\<run>\`: assessment, oracle and probe reports, own-check logs, census, seed check; `blind\` and `blind-key\`; `reviews\` (each review and its citation check);
- `s4\`: fixtures, the three seed lists, the proposed rule, the same-tree check and `c6-decision-record.md` with the decision messages verbatim;
- `calibration\`, `contamination-audit.json` with its script, `seed-count.py`.

M evidence stays under step 4's `Markitect-V1-Requalification\e11e6cff\packets\m-*\attempt-01\` (`run-summary.json`, Apply report, transcripts, candidate bundles).
