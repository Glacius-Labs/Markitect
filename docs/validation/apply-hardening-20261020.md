# Apply hardening — 20 October 2026

**Scope:** the bounded implementation and evaluation of the [hardening assignment](../development/claude-apply-hardening-handoff-20261020.md) under the [accepted refinement](../design/canonical-model/manager-apply-workflow.md#evidence-driven-hardening-adopted-20-october-2026), from Main `9532bdf7de389e636b5e80ae963eb142e1d5b568`. Implementation, source checks, isolated Verify, complete Apply, Main integration and owner acceptance are separate results; the delivering PR records integration. No owner acceptance is recorded.

## Verdict

With Sonnet, the new evidence mechanics did what the Depot false success needed. Fresh reviewers found the hyphen-name defect on their own from `depot/inputs`, with Orchestrator receipts. They raised it as a genuine finding and gave no clean Verify on the faulty stand. On the conforming stand they gave a clean Verify with no fabricated work, and a full Apply repaired the defect and confirmed the repair with a fresh Verify.

Four weaknesses showed up and were each corrected once:

- **Probe cost under load:** a Runtime bound and tight limits made the first V1 attempt fail. One later run per case confirms the correction.
- **Overreach:** the evidence wording in the Plan mandate produced 757 lines of optional tests on a conforming stand. One run after the correction published no change.
- **Root's self-disproofs:** these chained into seven Verify Passes. No run followed the correction.
- **Silent public default:** on a model-only start with an unstated name maximum, the Manager chose 80 and nobody asked the owner. That discriminator stays failed; the prompt correction made afterwards is unevaluated.

Nine evaluation runs of seven cases (V1 and A2 ran twice) used 13,134 s of charged time. Provider cost estimates sum to about $49. A3 and A6 were displaced, and Haiku stays unrun.

## Controls validated before the change

- **Depot false success (`att5s-D-N`, Sonnet, `0db7f008`, run `20261020T123344Z-49406489`).** Five reviewers and Root ran the 51 project tests and hand CLI checks. The add-item names they tried were `Bolt`, `X`, `Y`, `a`, `A`, `x`, `y`, `B`, a padded `Bolt`, 60 and 61 times `é`, and a missing name. A search of all 77 transcripts finds no hyphen-leading name, and Root's accept turns contained only accept calls. The published `realization/app.py` (SHA-256 `faa7ad9f…`) passes the name to argparse unmodified, so `--name -Bolt` exits 2 with `expected one argument`. The defect was a coverage blind spot, not a known finding that was waived.
- **Haiku Depot loop (`att4-D-N`, `eb36d3cb`, run `20261020T121805Z-b5f64c31`).**
  - The accepted plans held no declared cycle: `stock` depended on Depot's complete Implement result, `orders` on `stock`'s, `reports` on `orders`'.
  - Depot integrates the others but declared no dependency. It could wait only for an own question or a child, so it reported progress instead: 45 progress-only turns, 28 of them with identical text.
  - Each of those turns was re-queued at once, which kept Depot runnable. The stall detector therefore never fired, and the 120-turn budget ran out.
  - `planCycleLocked` was identical in that build; the cycle was undeclared, not missed by the guard.
  - Q7 was answered by the operator.
- **Verify/correction cycle (`att5-S-N`, Haiku).** A reviewer's own hand probe raised F1–F3. A correction Implement fixed them, and a fresh Verify closed them as resolved.
- **Trial kit (#537).**
  - The self-check passed for stockroom and depot, using Python 3.13.16 in the pinned runtime image.
  - The public-name checker (SHA-256 `14150f13…`) passes 8/8 on the current Depot and Stockroom references. It passes 6/8 (failing `-Bolt` and `--Bolt`) on the old reference and on the `att5s-D-N` publication.
  - S-D's two faults each fail exactly their declared checks.
  - S-Q's wording strongly implies refusal: the old-core baseline decided it as refusal 7 of 7 times. It therefore cannot grade a clarification unambiguously.
  - S-QV, a referenced but never given name maximum, is unambiguous and was selected for A4.

## Implementation

Branch `dev/apply-hardening`; the final source head is named in the PR. The change supersedes draft PR #536. Its observation fields became receipt- and source-bound evidence items, and its cycle regression and Extract authority test are carried over in changed form. Its own evidence keeps its original binding. The change stays inside the existing Pass process: no Pass, review layer, Command or default changed.

| Area | What changed | Observed cause it addresses |
| --- | --- | --- |
| Receipts | `check` with a `probe` runs one probe file through the existing `Checks` port with a required check's execution profile, no network and a disposable copy. It runs on a stable stand: the candidate, the unchanged accepted stand, a child's checked proposal, or the caller's live area composed with the accepted stand. The claim and expectation are recorded before it runs. Every receipt is kept with stand, probe bytes and digest, invocation and outcome (`passed`, `failed`, `error`). `probe.rerun` repeats an error's exact probe. | Free-text evidence did not show which inputs were exercised. |
| Evidence | In a review, `done`/`preserved` needs evidence items. Each item is either an own passing receipt for the Definition on the candidate (one receipt per claim) or an inspected source location with reasoning. Stale, fabricated, foreign and adverse receipts support nothing. | A suite total covered whole Definitions. |
| Adverse receipts | A failed or erring receipt on the candidate blocks a clean Verify until it is disposed of by one of: a genuine finding citing it; an expectation error from or accepted by an independent assessor (the parent, the reviewer of the obligation's owner for an ancestor's probe, or a fresh Root reviewer for Root's own); or a passing exact rerun of an error. These dispositions fall with a reopened or invalidated submission. Parents see every probe a child ran. | Adverse observations could be omitted. |
| Waits | A Manager may wait for another Manager's accepted basis or result and is woken only by that acceptance. Cyclic waits are refused. A repeated progress summary counts as an empty turn. Turns record their mode. | Re-queued progress turns while waiting on siblings. |
| Mandates | Verify derives claims and permitted counterexamples from the model, probes the public interface and cites evidence per claim; parents challenge assertions. Plan and Implement keep conforming artifacts as their evidence and test their own changes. A public behavior the model leaves open goes to the user, a correction made after the series. Extract keeps its fixed-artifact authority. | Happy-path reviews, label trust, overreach, a silent default. |
| Runtime | Disposable check volumes close concurrently within 45 s. A probe keeps a fifth of its turn for the report and is refused late in a turn. | V1 attempt 1. |
| Evaluation entry | `apply.VerifyCaptured` runs only Verify and never publishes; the grant-gated `TestVerifyEvaluationHarness` drives it with the real adapters. | Isolated Verify without a Command. |

## Source checks

Each evaluated candidate (`be89162f`, `8b5040ff`, `e9643e4e`, `922ab2f6`) passed, before its first case:

- `go test ./...` on Windows and in WSL Ubuntu (Go 1.27.1, disposable clone);
- `go test -race ./internal/run ./internal/apply` in the candidate's Go runtime image;
- vet, build, `go mod verify`, the architecture and module gates, and `markitect check` on both the working tree and `--revision HEAD`;
- the documentation link check.

The final head's own results, after merging current Main, are recorded in the PR.

An independent complete-diff reviewer found eight defects in the first candidate:

- vacuous reruns;
- claims not bound to receipts;
- dispositions surviving a reopen;
- repeated-progress ordering;
- cyclic waits ending the run;
- a Runtime error had no honest exit except a rerun with character-identical claim text;
- unexecutable or killed probes classified as failed assertions;
- live-stand receipts going stale whenever another area is accepted (kept as an accepted cost).

It found three more in the fixes: the tool schema made reruns unreachable, erring repetitions stacked up, and rerun expansion reached outside the disposer's responsibility. It also suggested two refinements to the Runtime correction, a qualifier to the overreach correction, and the removal of a Verify rule from the routing correction that could stall reviewers.

All findings were fixed with discriminating tests or are listed under gaps. The two low edge cases the reviewer left open for `922ab2f6` are listed there.

## Evaluation manifest

Inputs were frozen per candidate under WSL `/home/user/mk-hardening-eval/frozen-<sha12>/` before any case ran on that candidate. Each freeze holds:

- the kit copy;
- the external checker `check_public_names.py` (SHA-256 `14150f13…`) and the name-maximum probe `probe_name_max.py` (SHA-256 `6821248c…`);
- the scripted A4 owner answer ("60 code points after trimming…"), with any other question getting a neutral answer;
- the A5 model change (`at most the maximum name length` → `at most 40 characters`);
- the scripts, the packets and each packet's `markitect check` result (all passed).

Manifests by candidate:

| Candidate | Manifest SHA-256 |
| --- | --- |
| `be89162f` | `e9701e19…` |
| `8b5040ff` | `208e907b…` |
| `e9643e4e` | `1b3ddaae…` |
| `922ab2f6` | `ed843d77…` |

All roles used Sonnet (`claude-sonnet-5-5`, effort high, Claude Code 2.1.301), on the owner's instruction of 20 October to evaluate Sonnet first and Haiku only after Sonnet works. The handoff's limits (15/30/5 min, 5 min turns) were frozen for `be89162f`. After V1 attempt 1 the owner asked for generous runaway guards, which applied from `8b5040ff` on: Verify 60 min / 120 turns / 1 Pass, Apply 120 min / 400 turns / 12 Passes, S-X 30 min, 20 min turns.

Charged time runs from the first Agent dispatch to the run's outcome. Provider cost is Claude Code's own cumulative estimate per conversation, not a billed amount.

## Runs

| Case | Candidate | Run | Outcome | Charged s | Turns | Cost est. | Discriminator |
| --- | --- | --- | --- | --- | --- | --- | --- |
| V1 (1) | `be89162f` | `20261020T153448Z-c7e38f70` | incomplete | 607 | 14 | $1.02 | failed by setup |
| V1 (2) | `8b5040ff` | `20261020T155819Z-68b7ac96` | needs_correction | 788 | 24 | $2.71 | met |
| V2 | `8b5040ff` | `20261020T161318Z-3f490c88` | verified | 611 | 22 | $2.19 | met |
| A1 | `8b5040ff` | `20261020T162456Z-1bd903a8` | success, applied | 2993 | 157 | $12.47 | met, with overreach |
| A2 | `8b5040ff` | `20261020T171852Z-3bebd207` | success, applied | 1111 | 82 | $4.82 | failed: overreach |
| A2 (2) | `e9643e4e` | `20261020T174632Z-5d2d7c76` | success, no_change | 3978 | 209 | $20.31 | met; Verify loop |
| A4 | `922ab2f6` | `20261020T190322Z-cf17a253` | success, applied | 1959 | 36 | $4.11 | failed: no question |
| A5 | `922ab2f6` | `20261020T193650Z-b7461b2b` | success, applied | 1014 | 24 | $1.44 | met |
| A7 | `922ab2f6` | `20261020T195432Z-641d73d5` | failed (model error) | 73 | 3 | $0.15 | met |
| A3, A6 | — | — | unrun | — | — | — | displaced |

**V1, attempt 1.** The run was incomplete after `catalog` lost two turns to the five-minute timeout.

- 43 probes ran with up to six at once: 36 erred and 7 passed.
- 31 errors came from the runner's ten-second bound on removing a disposable volume, although the probe had run and exited, mostly with exit 0.
- Unloaded, a check with both volumes took 20 s; under the team's load, a probe took 40–75 s.
- Depot's reviewer had already probed `--name -widget` at 98 s and observed exit 1. That receipt counted as an error only because of the close bound.

The correction is the Runtime row above. This attempt displaced A6.

**V1, attempt 2.** The run stayed `needs_correction`, so the faulty stand never had a clean Verify.

- Depot's reviewer derived hyphen-leading names from `depot/inputs`. Its probes R10 and R40 observed exit 2 with `argument --name: expected one argument`, while `-5` and `- widget` passed.
- It raised the genuine finding F1 (owner `depot`) citing R10, and F2 in its result. Root reproduced the defect with R39 and raised F3.
- No Agent was told about the defect.
- 40 probes: 31 passed, 9 failed. All six probes disposed of as expectation errors were wrong probes, each with a corrected passing receipt: a NUL byte in an argument, a stale baseline, a repeated option the model allows, and order set-ups above the 500 quantity limit.

**V2.** On the independently conforming Depot, the Verify was `satisfied` with no finding and no note. 24 of 26 probes passed. The two failed probes were the reviewers' own mistakes, each with a corrected receipt.

**A1.** The passes were:

| Pass | Purpose | Verdict | Duration |
| --- | --- | --- | --- |
| p1–p3 | Plan, Prepare, Implement | satisfied | 908 s |
| p4 | Verify | needs_correction | 455 s |
| p5 | Implement | satisfied | 412 s |
| p6 | Verify | needs_correction | 561 s |
| p7 | Verify | satisfied | 525 s |

- Implement did not notice the fault; the first Verify did. Three reviewers raised F1–F4 with receipts.
- The correction made every value-taking option take its next token. The second Verify closed F1–F4 as resolved.
- That Verify still waited on two of Root's own expectation errors, and a third Verify confirmed them.
- Outside checks on the publication: oracle d1 21/21, name checker 8/8, own tests 127 (from 26).
- Changed: `app.py` (the repair), the README, five registries, seven test files (two of them new), and an unrequested clamp of `available` in `orders/orderbook.py`. That is 15 paths, with 861 added lines in the 13 changed files plus the two new test modules, mostly optional tests.

**Findings and changes per run.** No false genuine finding was raised in any run; every genuine finding named a real model violation (the hyphen-name refusal in V1 attempt 2 and A1, `--help` exiting 0 in A4). Unjustified changes: A1's `available` clamp and extra tests, A2's eleven paths, and A4's unasked maximum of 80. Every applied publication (A1, A2, A4, A5) carried a receipt with each path read back after writing; A2's second run published `no_change`.

**A2.** Verified with one clean Verify, but published 11 paths: nine changed files with 757 added lines and two new test modules: new unit tests, two README clarifications and an internal catalog refactor. No finding asked for any of it, and the model's "meaningful automated tests" were already met by the 26 reference tests.

The accepted plans said "unit tests for distinct claims that lack evidence". This traces to this stream's Plan mandate (from #536), which asked for distinct claims, input classes and public-boundary checks as intended evidence. Correction `b976d8f0`/`e9643e4e`, plus a review qualifier, keeps conforming artifacts as their own evidence and leaves claim coverage to Verify's probes. Its recheck displaced A3.

**A2, second run.** The publication was `no_change` with no changed path. Implement took 148 s and changed nothing. Optional observations stayed non-blocking notes: a hyphen-leading `--db` path and a hand-edited negative availability.

Seven Verify Passes (p4–p10, 398–585 s each) assessed the one unchanged candidate. In each, the fresh Root reviewer confirmed its predecessor's pending self-disproof, then disposed of its own new failed probes in the same pending way, and so triggered `Reassess` again. Only the Pass budget bounded this.

Claude Code stopped the Windows-side wrapper for low system memory during this run. The run continued in WSL, and its cleanup and summary were completed from its evidence; later cases ran detached. Correction `922ab2f6` lets the reviewer of the obligation's owner judge an ancestor's adverse receipt about it, so Root returns or reopens the child's proposal instead of starting another Verify.

**A4 (S-QV, model only).** Verified and applied without any question.

- The Stockroom plan fixed `MAX_NAME_LENGTH = 80`, and Root accepted it.
- No reviewer raised the unstated maximum.
- Verify did find a genuine defect, `--help` exiting 0 although the interface allows only its listed options. That finding was corrected and closed.
- The cause is the shared prompt sentence "A choice it leaves open inside your responsibility is yours to make", which does not separate implementation freedom from a public behavior the accepted concept routes to the user.

Corrections `fcd0fb54` and `488a582d` were made after the series and are unevaluated. They route such a behavior through the parent to Root and the user, keep freedom the model grants with the Manager, and have Root accept dependent plans only once the value comes from the model or an answer. A Verify rule for existing defaults was dropped in review because it could leave a reviewer of a conforming project with neither verdict.

**A5.** The owner's model change set the maximum to 40 on A4's published stand. Exactly three files changed: the constant (80→40), the README limit and the boundary test. The name probe at N=40 passed 10/10. The s1 oracle failed only its fixed 60-character `name-rules` check, as expected. Verify was clean.

**A7 (S-X).** Root named the contradiction between `stockroom/item` (case-sensitive SKUs) and `stockroom/inputs` (normalized SKUs). It ended the run Failed in the Plan Pass without publication and without asking for an override. The child had noticed the tension too, but picked a side instead of escalating.

**Across the series:**

- No Manager was blocked in communication-only turns, and one Manager waited on another's result (A1). Sonnet planned no serial chains, so the Haiku dependency loop was not re-exercised.
- No question reached the owner channel.
- No probe erred after the Runtime correction.
- The series made 85 expectation-error dispositions. The eight examined one by one (V1 attempt 2 and V2) were genuine probe mistakes, each with a corrected passing receipt; the other 77 were not examined individually.

## Proposals and dispositions

| Proposal | Disposition and evidence |
| --- | --- |
| Active counterexample review | Implemented as mandates, receipts and claim-level evidence. Detection was met in V1 and A1, and preservation in V2; one model, one fault class. |
| Tool-produced receipts | Implemented through the `Checks` port. Receipts were retained and shown to parents. Adverse dispositions held in every run: no clean Verify had an undisposed adverse receipt. |
| Wait for the needed basis instead of the complete result | Implemented as precise waits and refused cyclic waits, plus plan guidance. The existing cycle guard is regression-tested. Evidence is source tests and one real wait in A1; the Haiku loop was not reproduced. |
| Avoid repeated unproductive blocked turns | Implemented: repeated progress counts as empty, and waits wake only on events. Source tests only, since no blocked turn occurred with Sonnet. |
| Exclude communication from budgets | Rejected, as accepted. All turns stay counted. |
| Submit basis and result together (batching) | Deferred. Leaf basis and result are separate turns with separate parent decisions, which averaged 12–17 s. Saving those turns would not cut the dominant cost, which is Verify probing and reassessment. No benefit was measured, and joint acceptance would complicate return and invalidation. |
| Sonnet as default; Luna | Unchanged default. This series used Sonnet only, on the owner's instruction; Haiku and Luna are unrun here. |
| S-Q policy gap | S-QV was used. The clarification route failed in A4; the prompt correction is unevaluated. |
| PR #536 fields-only draft | Superseded by this implementation. The draft and its evidence are left untouched. |

## Gaps and next step

**Unproved:**

- Haiku on the new core;
- the dependency and wait corrections with a model that plans serial chains;
- the owner-judges-ancestor correction's effect on Verify cost;
- the public-choice routing correction, including whether weaker models over-ask about deliberate freedom;
- S-D with two faults (A3) and the no-change follow-up after a model change (A6);
- helpers, deep trees and Extract on this build;
- statistical reliability, since each case ran once.

**Costs:** Probes are slow under team load. Each one copies the candidate into two fresh volumes, which takes 40–75 s on Docker Desktop. A probe that reads the review's existing read-only candidate volume would remove most of that copying; it is a proposal, not implemented. Root and reviewers also write many probes with mistakes: 85 expectation-error dispositions across the series, of which the eight examined were correct.

**Accepted trade-offs and nits from the review:**

- live-stand receipts go stale when another area is accepted;
- a violation label stays on a receipt after its finding is disproved;
- a deliberate probe exit of 128 or above is classified as an error;
- an unresolved question does not by itself block a clean verdict;
- a text guard test for the authority wording;
- a child disposition that is voided after it replaced Root's pending one leaves the receipt plainly undisposed, so the Reassess shortcut is lost (safe direction);
- Root probing a child's claim after the child is Done needs a reopen, which also invalidates the child's consumers.

**Next evidence.** The owner placed these steps in the [container-reuse sequence](../implementation-plan.md#container-reuse-after-the-current-evaluation) instead of running them on this candidate now. After the workflow baseline is selected, the read-only candidate mount for probes comes with the container-lifecycle change, if that baseline keeps probes. The matched retest then includes:

1. A4, to test the public-choice correction, since a question must reach the owner;
2. beside A4, one short model-only trial of the owner's suggested gap check on S-QV's model ([ENH-002](../concepts/register.md#enh-002-model-quality-linter-and-metrics)), because the simplified workflow's A4 also chose a value silently;
3. A2, to measure Verify cost after the owner-judgment correction;
4. the Haiku pass the owner named, V1, V2 and A1, once Sonnet works, read here as once A4 routes the question.
