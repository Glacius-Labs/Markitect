# Apply trial series with Haiku — 20 October 2026

**Scope:** the first real runs of the prepared [Apply trial cases](../../experiments/apply-trials/README.md) (#523), on the Apply of Main `a0928a1b` (DEC-056 to DEC-063, before DEC-064). There were 64 runs: 13 base cases with repeats, 4 escalation variants, and the generated fleet projects for width and depth. Every Execute and Verify turn used Haiku `claude-haiku-5-5` at high effort. Real runs, source checks, Main integration and owner acceptance are reported separately. No owner acceptance is recorded.

## Verdict

- **Repair is reliable.** All 53 injected artifact faults were repaired in 53 of 53 cases, almost always in one Verify round. The faults covered code drift, a passing test of wrong behavior, contradicting README and configuration, a rule rebuilt outside its owner, a registration in the wrong shard, and up to 8 faults spread over 8 Managers. Each fault was fixed by the owner of its area. Of the 43 faults whose repair could be checked this way, 42 also got a test that fails when the fault returns; one list-order repair had none. The rebuilt availability rule was replaced by a call to the owner's realization in both runs (DEC-060).
- **Building and changing work.**
  - Model-only Apply verified in 8 of 8 runs (1 to 10 Managers, depth up to 3), with every behavior check passing in 7. In the eighth, a depot result crashes on an amount written with 5000 leading zeros.
  - The model change 0.1.0 → 0.2.0 on the actual earlier result verified in 2 of 2 runs, with old data files still usable.
  - Valid but unnecessary restrictions were realized exactly, in 2 of 2 runs, and never relaxed or escalated.
- **Escalation depends on the kind of defect.**
  - Contradictions in the model ended the run through `model_defect` in 6 of 7 runs. The reports named both Definitions, after 1 to 3 minutes.
  - A gap without a natural default (a value referred to but never given) was reported in 3 of 3 runs, in about a minute.
  - A gap with a natural default (what `issue` does beyond the stock: refusal fits the rest of the model) was never reported (0 of 7) and was published as a refusal.
- **No-change holds on Apply's own results, not on hand-written stands.**
  - A second Apply on a Verified result changed nothing in 2 of 3 runs.
  - On conforming stands written by hand, Managers extended the tests in almost every run where the tests did not cover every modeled boundary. With thorough tests, 1 of 3 stands stayed unchanged and the other two got small edits.
  - Code changes on conforming stands were rare. Most of them fixed real edge failures of the hand-written references (see below).

## Setup

| Item | Value |
| --- | --- |
| Product | Main `a0928a1b7a95`, CLI sha256 `bce54d44…`, runtime images built by `experiments/recursive-apply-demo/ra_build.sh` plus the Python example image |
| Agents | Claude Code 2.1.301, `claude-haiku-5-5`, high effort, every Execute and Verify turn |
| Owner request | `Apply the committed model.`, no owner decisions |
| Harness | [run/](../../experiments/apply-trials/run/): `ra_bind.sh` and `bind_profiles.py`, packets from `materialize.py`, assessment by `assess.py` inside the Python image after each run |
| Machine | WSL Ubuntu, 16 cores, 30 GB; up to 3 runs in parallel. In the last hour (S-QV and two D-N runs) another session's Apply runs shared the machine |
| Trial setup | Branch `dev/apply-trial-runs`. Each run froze its setup commit (`bad0be23` to `87a84cf3`); all runs were re-assessed with the final checks of `87a84cf3` |
| Evidence | Run directories, transcripts and retained work volumes under WSL `~/mk-tr`, outside the repository |

## Results

Times are wall-clock seconds of the whole Apply, median and range. "Met" means the expected outcome: Verified with publication `applied` or `no_change`, or a `model_defect` stop with nothing published.

| Case | Question | Runs | Met | Faults repaired | Rounds | Time s |
| --- | --- | --- | --- | --- | --- | --- |
| S-D | Code drift, one fault caught only by the model | 2 | 2 | 4/4 | 1, 1 | 266 (265–267) |
| S-A | Code, test, README and configuration contradict each other | 2 | 2 | 8/8 | 2, 1 | 361 (301–421) |
| D-M | Six faults in five areas | 2 | 2 | 12/12 | 1, 1 | 368 (354–383) |
| Fleet -M*k* | 1 to 8 faults in different ledgers, width 1 to 8, depth 1 to 3 | 9 | 9 | 29/29 | 1 each | 432 (318–791) |
| S-G, D-G, fleet -G | Model-only Apply | 8 | 8 | – | 1 to 2 | 576 (408–929) |
| S-C | Model change on the actual S-G result | 2 | 2 | – | 1, 1 | 422 (400–445) |
| S-O | Valid overspecification | 2 | 2 | – | 1, 1 | 443 (412–475) |
| S-X, S-XG | Contradiction in the model (reference / model-only start) | 5, 2 | 4, 2 | – | stop | 142 (64–312) |
| S-QV | Gap without a natural default | 3 | 3 | – | stop | 61 (59–64) |
| S-Q, S-QR | Gap with a natural default (model-only / reference start) | 5, 2 | 0, 0 | – | 1 to 2 | 434 (264–500) |
| S-F, D-F | Follow-up Apply on a Verified result | 2, 2 | 1, 1 | – | 0 to 1 | 337 (170–481) |
| S-N, D-N | Hand-written conforming stand | 3, 4 | 1, 0 | – | 1 to 2 | 435 (296–924) |
| Fleet -N, -TN | Generated conforming stand, thin / thorough tests | 6, 3 | 0, 1 | – | 1 each | 365 (315–674) |

**Fleet scaling** (one run each):

| Size | Managers | Faults repaired | Repair s | Model-only s (rounds) |
| --- | --- | --- | --- | --- |
| 1 ledger | 3 | 1 | 318 | 462 (1) |
| 2 ledgers | 4 | 2 | 332 | – |
| 4 ledgers | 6 | 1 / 4 | 483 / 337 | 589 (1) |
| 8 ledgers | 10 | 1 / 4 / 8 | 450 / 454 / 791 | 832 (2) |
| 4 ledgers, depth 2 | 7 | 4 | 362 | – |
| 4 ledgers, depth 3 | 8 | 4 | 432 | 891 (2) |

Execute turns grew with the number of Managers (3 to 27 per Verified run); no run reached its budget or deadline.

## Findings for decisions

1. **Gaps with a natural default are decided silently (0 of 7).** In S-Q the model keeps "never negative" but no longer says what an `issue` beyond the stock does. Every run chose a refusal, published it, and no reviewer named the gap. The same gap on the reference start (S-QR) was not reported either. A missing value (S-QV) was reported every time. DEC-063 expects both kinds to stop the run. Either that expectation needs a mechanism that looks for undecided boundary behavior, or the owner accepts that a default which the rest of the model suggests is the Manager's choice.
2. **A model defect seen in Verify does not stop publication.** In one S-X run both reviewers described the contradiction and wrote that it belongs to the parent, but filed it as non-blocking; the round was green and the stand was published. Only Root's Execute turn can end the run with `model_defect`.
3. **Model statements that point outside the model age badly.** The 0.2.0 model promises that data written by "the realization of model version 0.1.0" stays usable. In the follow-up run after S-C (S-F, first run) that realization existed only in Git history. Root reported the unstated format as a boundary gap, which is defensible, and the run stopped. The same follow-up on the 0.1.0 result stayed unchanged.
4. **No-change on a stand Apply did not write.**
   - Thin tests make Managers add tests even when nothing blocks: the fleet stands with thin tests changed in 6 of 6 runs, by 26 to 134 added lines. With thorough tests the changes fell to 0 to 19 lines.
   - Reviewers read "meaningful tests for every modeled command" as a command-line test per command. That reading made two D-N rounds blocking.
   - Unnecessary code changes were rare: refusing a repeated option, a defensive change in the stock code, and accepting option values that start with a hyphen.
   - Whether test coverage beyond the model's statements is part of conformance is an owner question. Until it is answered, the first Apply on an existing project will extend its tests.
5. **Verify finds real edge failures.**
   - On the stands I had called conforming, Verify found real failures that the model forbids ("expected failures never crash"): an unusable data path, a deeply nested data file, an ASCII-only stdout, numbers with 5000 leading zeros, and names starting with a hyphen. Managers repaired them.
   - Verify missed one such failure in a depot result that Apply had built itself (D-G, second run).

## Test-setup corrections during the series

The first S-N runs showed that the stockroom reference was not fully conforming. The references, validation overlays and fleet generator were hardened in three steps, and the edge-robustness check now covers each failure found (`bad0be23`, `1d73c38a`, `8652adb6`). Two test-setup defects were fixed:
- a race in the fleet cache key (`a65b9cb5`);
- reports that could not be written when a check detail quoted an undecodable byte (`33090033`).

One check was stricter than the model and was relaxed: it required refusing a name with an undecodable command-line byte (`a54ee737`). These corrections are in the trial setup only; the Apply under test was the same in every run.

## Limits

- One model family (Haiku); 1 to 5 runs per case. Rates are indications, not estimates.
- One machine with up to three runs in parallel, which stretches time. The S-QV and last two D-N runs shared the machine with another session.
- The checks are black-box and follow the model; the evaluator judged findings, defect reports and the purpose of changes by reading them.
- The fleet projects are generated and similar per ledger; they measure organization size, not domain difficulty.
- DEC-064 changes the strategy, the defect path and the start-stand check. These results describe the Apply before it.

## Evidence kinds, kept apart

- **Source checks:** `experiments/apply-trials/selfcheck.py` passed for every project on the final setup, as did `markitect check` on the repository.
- **Real runs:** the tables above. Run evidence stays outside the repository.
- **Main integration:** the trial setup and this record are integrated through the delivering pull request.
- **Owner acceptance:** none recorded.
