# Simplified Apply workflow comparison — 20 October 2026

**Status: experimental comparison candidate, not for integration.** The owner commissioned this separate session to implement and evaluate an Apply without the global Plan and Prepare Passes, next to the running [hardening stream](apply-hardening-20261020.md). It changes no accepted concept: the [accepted workflow](../design/canonical-model/manager-apply-workflow.md) keeps Plan, Prepare and Implement until the owner decides. Branches `dev/apply-simplified-passes` (C1, C1′, this report, draft PR #545) and `dev/apply-simplified-passes-922` (C2). Implementation, source checks, isolated Verify, complete Apply and owner acceptance are reported separately. No owner acceptance is recorded.

## Hypothesis

Apply runs Model check and File check, then **Implement**, then an independent **Verify**. Findings start a new Implement and a fresh Verify. A clean Verify leads to guarded publication with exact read-back. The hypothesis: this is at least as reliable and complete as Plan → Prepare → Implement → Verify on the same frozen inputs.

Planning and preparation do not disappear: each Manager plans and prepares its own part inside Implement, through the existing base process:

1. settle its own approach, obligations and dependencies;
2. produce the basis its dependent children need;
3. have that basis, including files and registry, checked technically and accepted semantically by its parent;
4. only then release its children;
5. stay available for questions, review and coordinate its direct children's contributions;
6. integrate accepted results and submit its complete responsibility for acceptance.

No new mandatory sub-process replaces the removed Passes under another name. Parents do not plan or implement for their children, and every Manager writes only in its own area.

### Design account (working method)

1. **Observation.** The [first Pass-process report](apply-pass-process-20261020.md) records substantial coordination overhead: each Pass costs every Manager a basis and a result plus parent decisions, three times before Verify. Haiku also exhausted its budget on serial dependencies. Plan's read-only result and Prepare's scaffolding are both re-examined in Implement anyway.
2. **Challenged assumption.** That a whole-tree planning and preparation stage is needed before realization, so that dependencies and foundations are agreed globally before anyone builds.
3. **What changes and what stays.** Only the Apply composition and the Implement mandate change; the Pass engine, its gates, receipts, findings, dispositions, ownership, limits and publication stay. The Implement basis now carries a Manager's approach together with the foundation its children build on, so the parent still reviews an approach before its children start. Dependencies that were declared in plans are now declared by the waits Managers report while they work. A waited-for accepted basis or result is bound to the waiter's work at its accepted version. Its revision or reopening explicitly resets that work, which restarts only on the newly accepted version. Unchanged valid bases stay.
4. **Observable improvement.** On the same frozen cases and model: no additional missed defect, no lost preservation, no new ownership conflict or omission, and question, contradiction and dependency handling at least as correct as the reference. Only then are fewer Passes, Turns or less time a gain. Fewer Passes alone or one green run prove nothing.

## Candidate

| | Reference (hardening) | Comparison |
| --- | --- | --- |
| Product SHA | `8b5040ff294018dbd9d3c9d3c52742013106a980` (reviewed, gated); later `922ab2f601f12703b913663ebfe6d9c2bb01f9ee` for A4, A5 and A7 | C1 `e814c373251b9198e20b2edc87e9255f40c3c1f8` and C1′ `5cd22e58598143df00fefebcad6ad9c99bd2981d` on `dev/apply-simplified-passes` (parent `8b5040ff`); C2 `205bf5243d69f20e0c96f196b2d29f03ba99b2bc` on `dev/apply-simplified-passes-922` (parent `922ab2f6`) |
| Work Passes | Plan (read-only, plan payload), Prepare, Implement | Implement |
| Correction purposes | plan, prepare, implement | implement |
| Dependencies | declared in accepted plans, start conditions of later Passes; ad-hoc waits | declared by waits, bound to the accepted version |

Changed source, besides the composition (`internal/apply/apply.go`):

- **Implement mandate.** It now holds the duties of the former Plan and Prepare mandates that concern a Manager's own part: per obligation the evidence the reference's Plan mandate asks for on the respective base (on C1 claims, input classes and public-boundary evidence; on C2 the method and intended evidence, with existing conforming artifacts as their own evidence); interfaces, needs and outputs; how a parent coordinates its children; and a basis that holds the approach and the foundation children build on. It also says how to declare a dependency. The Verify mandate and guidance are unchanged.
- **Bound dependencies (`internal/run/bindings.go`, opt-in `PassSpec.BindWaits`, set only on Apply's Implement).** Without plans, nothing would invalidate work built on another Manager's revised basis through a wait. This change keeps the invalidation guarantee; it is not a separate improvement.
  - The accepted target of a wait, or one already accepted when the wait is reported, is bound to the event the waiter works toward: its basis or its result. Before, an already accepted target was a refusal; now it is a turn report kind `bind`. It is recorded as `dependency.bound` with proposal and stand.
  - Only Managers outside the waiter's own line bind. Ancestors and descendants are already related by the parent-first and integration order.
  - A binding that would close a cycle among all start, basis and result events, whether or not they already happened, is refused or skipped (`dependency.unbound`).
  - A revision or reopening of a bound submission resets a waiter whose basis built on it, like a declared consumer; its restart waits for the new version and records it as rebound. A waiter whose result alone built on it keeps its basis and children and integrates again: its submitted result is invalidated, or its accepted one reopened with that result's consumers.
  - Verify and Extract bind nothing, so an isolated Verify runs exactly the reference's Verify code path.
- **Self-model and guides.** `product/run/confirmed-run-history`, its generated view, the Apply registry entries, the run/apply guides, the runtime contract and the CLI help line describe the candidate. The workflow, architecture, contributing and project-workflow pages mark it as an unadopted experiment.

The independent complete-diff review of the first draft found one blocking defect. The bindings were not cycle-checked across already completed events, and every binding acted as a start condition. As a result, a parent bound to its own child's result, or two siblings bound to each other's bases, would deadlock on the next reset. Its re-review found that a binding serving only the result still reset the waiter's unchanged basis and subtree. The design above corrects both, and regression tests cover these scenarios plus the cycle check. The review also flagged two mandate sentences as going beyond a minimal change. They are kept because they replace the parent's plan review: "submit your basis once it serves your children instead of holding it back", and "a leaf's basis is its approach and what it has realized so far". Without them a Manager could offer its first basis only after complete realization, and no parent would see an approach before the children build.

The engine's `Plans` capability and the `plan`/`prepare` purposes stay as generic engine features with their tests. Removing them belongs to an adoption, not to this comparison.

**Comparability.** Same kit, packets, oracles, owner answers, profiles, images, model and limits as the reference. The deliberate differences are the composition, the Implement mandate and the Implement-only wait binding. All three follow from the workflow change; no other product change was made. V1 and V2 therefore exercise unchanged Verify code: they replicate the reference on a different host binary rather than test the simplification, so their main value is the same-model variance of the reference cases.

## Frozen comparison plan

Frozen on 20 October 2026, 17:45Z, before any provider work of this session. It was agreed with the hardening session, which ran the reference on the same inputs. Evaluation directory: `/home/user/mk-simplified-eval/` in WSL Ubuntu, container label prefix `cmp-`.

| Binding | Value |
| --- | --- |
| Candidate | `e814c373251b9198e20b2edc87e9255f40c3c1f8`; CLI SHA-256 `2229cb89…44717`, Verify harness `3a32d3e5…3f5` (built from the candidate) |
| Reference | `8b5040ff294018dbd9d3c9d3c52742013106a980`, the hardening session's reviewed and gated evaluation candidate |
| Images | the reference build's own: runtime `sha256:3d773344…de394`, agent `sha256:fc865635…7aa`, Python example `sha256:81a0f8bc…32f`. Claude Code 2.1.301, agentd `development`. The candidate changes no image content. Its own rebuild had different IDs (npm layer not cached) and is kept under tags `-own`. |
| Model | `claude-sonnet-5-5`, effort high, every Execute and Verify role, the existing dedicated runtime token |
| Inputs | the reference's frozen kit copied verbatim: kit digest `973a1413…4d3c` at the copy, manifest `208e907b…fa4d`. The attempts' bindings show `fde27e3e…da2c`, because the verification's own `git status` refreshed the packets' `.git/index` stat caches; packet heads, trees and every other file are identical. Packets V1 `49c7a616`, V2 `f8173a9c`, A1 `8660dc87`, A2 `876a9940`, A3 `551e34e4`, A4 (S-QV) `59280816`, A7 `95d55288`; scripted A4 owner answer `85b73b90…4c0` with neutral answers to every other question; the A5 model change "at most the maximum name length" → "at most 40 characters" |
| Oracles | `check_public_names.py` `14150f13…f03b` (8 cases), `probe_name_max.py` `6821248c…5baa`, the kit's `oracle.py`, `assess.py`, the project's own tests |
| Case limits | taken over unchanged from the reference (owner instruction of 20 October): Verify 60 min, 120 Turns, 1 Pass; Apply 120 min, 400 Turns, 12 Passes; S-X 30 min; every Turn at most 20 min |
| Series limits | at most 9 provider-backed cases and 4 h summed charged time, first Agent dispatch to outcome. Repetitions and any Haiku case count. |
| Harness | the reference scripts with only paths, label prefix and one addition changed. The addition: the series remainder can shorten, never lengthen, a case's deadline, and such a cut is recorded. |

**Order and rules.** V1, V2, A1, A2, A4, A7, A3, then A5 from this session's own A4 publication (never from the reference's). A6 is not planned; the reference displaced it too.

- A case starts only while at least 30 minutes of the series remain. A cut by the series remainder is recorded as series exhaustion, not as a workflow result.
- Cases run one at a time and never while a reference case runs on the shared Docker engine.
- No repetition until green. A failed case keeps its result, and a documented infrastructure repeat counts as a case.
- **Haiku stays unrun.** The reference has no Haiku case on this build, and the owner made Haiku conditional on Sonnet working.

**Amendment before the first provider run (20 October, about 20:10Z).** The reference series ended at that time and showed that its candidate moved during the series:

- V1-b, V2, A1 and A2 ran on `8b5040ff`;
- A2-b ran on `e9643e4e`, a mandate fix after A2 overreached;
- A4, A5 and A7 ran on `922ab2f6`, which also lets the owner's reviewer judge an ancestor's adverse receipt;
- A3 and A6 were not run.

The packet trees of all three freezes are byte-identical, so the inputs are equivalent and only the builds differ. Each comparison therefore uses the reference's own base:

- **C1** = `e814c373` (base `8b5040ff`) runs V1, V2, A1 and A2.
- **C2** = the same simplification applied to `922ab2f601f12703b913663ebfe6d9c2bb01f9ee` runs A4, A7 and A5. C2 carries the reference's mandate fixes into the merged Implement mandate, word for word where the text corresponds, so that the comparison still varies only the workflow. C2 gets its own independent review and gates, built only in gaps between cases.
- **Remaining budget, in this order:** A2 on C2 against A2-b (confounded by `922ab2f6`'s receipt change, named as such), then A3 without a reference.
- **Excluded:** later reference commits such as `fcd0fb54`, which no provider run evaluated.

**Correction before any Implement ran (C1′).** The review of C2 found a gap in C1's merged Implement mandate. It had dropped the Plan mandate's sentence "Existing artifacts and tests that already meet an obligation are its evidence as they are", which the reference's Plan Pass carries on `8b5040ff` as a counterweight to optional changes. C1′ restores that sentence and changes nothing else. A1 and A2 run on C1′. V1 and V2 use only the unchanged Verify mandate, so they are equally valid on C1 and C1′.

| Build | Product SHA | CLI / Verify harness SHA-256 | Images (always the reference build's own) | Gates |
| --- | --- | --- | --- | --- |
| C1 | `e814c373` | `2229cb89…44717` / `3a32d3e5…3f5` | `8b5040ff`: `3d773344…`, `fc865635…`, `81a0f8bc…` | Windows and Linux full suites, race for run/apply, complete-diff review |
| C1′ | `5cd22e58` | `e0df5827…dde9` / `695dfb11…4e3` | as C1 | Linux full suite, race; Windows apply tests |
| C2 | `205bf524` (code identical to the reviewed `6e2bd5a3`) | `8737b948…c45d` / `8c5279eb…54ed` | `922ab2f6`: `714fb4a3…`, `a012340d…`, `37a02a44…` | Windows and Linux full suites, race, delta review |

**Expected outcomes** (the reference manifest's):

- V1: a genuine, owner-bound finding of the hyphen-name refusal with receipts; no clean Verify.
- V2: clean, no fabricated mandatory work.
- A1: the fault repaired; d1 oracle 21/21 and name checker 8/8; fresh Verify; findings closed.
- A2: no-change publication.
- A3: both faults repaired; s1 21/21.
- A4: the question routed to Root and the user; the scripted answer permits completion; s1 21/21.
- A5: exactly the public change; name maximum N=40 10/10; s1 without its hard-coded name rule 20/20.
- A7: Root reports the model error; the run ends Failed without publication.

## Results

Every case: Sonnet 5.5, effort high, the same packet tree as its reference, one run unless stated otherwise. "Charged" runs from the first Agent dispatch to the outcome. Cost is the provider's own cumulative estimate per conversation, not a billed amount. Reference values come from the hardening session's attempts under `/home/user/mk-hardening-eval/runs/`, read only. "Simplified" is this candidate. Evidence lives under `/home/user/mk-simplified-eval/runs/<attempt>/` (binding, events, transcripts, summary, outside oracles).

| Case (build pair) | Run | Outcome and publication | Discriminator | Passes | Turns (work/coord./review) | Charged s | Cost $ | Changed paths |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| V1 faulty Depot, Verify only (`8b5040ff` / C1 `e814c373`) | reference V1-b `…155819Z-68b7ac96` | Incomplete: Verify needs correction | met: 3 genuine findings on the hyphen-name refusal with receipts | 1 | 24 (–/–/24) | 788 | 2.71 | – |
| | simplified V1 `…195955Z-5ecd5bf8` | Incomplete: Verify needs correction | met: 5 genuine findings (catalog, depot, Root) with receipts R5/R16/R25/R29 | 1 | 25 (–/–/25) | 681 | 2.72 | – |
| V2 conforming Depot, Verify only | reference V2 `…161318Z-3f490c88` | Verified, clean | met | 1 | 22 | 611 | 2.19 | – |
| | simplified V2 `…202007Z-6c51bb38` | Verified, clean | met | 1 | 22 | 531 | 2.14 | – |
| A1 faulty Depot (`8b5040ff` / C1′ `5cd22e58`) | reference A1 `…162456Z-1bd903a8` | Verified, applied | met: d1 21/21, names 8/8, 4 findings closed as resolved | 7 (plan, prepare, implement, verify NC, implement, verify NC, verify) | 157 (49/35/73) | 2,993 | 12.47 | 15, incl. 5 registry shards and 2 new test files |
| | simplified A1 `…203032Z-bc713e26` | Verified, applied | met: Implement repaired the fault itself; d1 21/21, names 8/8, own tests 29 OK; fresh Verify clean | 2 (implement, verify) | 40 (12/7/21) | 514 | 3.13 | 3: the fix, its CLI test, one optional reports test file |
| A2 conforming Depot | reference A2 `…171852Z-3bebd207` | Verified, **applied** | **not met**: 11 paths: 757 inserted lines of optional tests and edits in 9 tracked files plus 2 new test files | 4 | 82 | 1,111 | 4.82 | 11 |
| | simplified A2 `…204135Z-d941c1a7` | Verified, **no change** | met: no-change receipt, d1 21/21, names 8/8 | 2 | 39 (12/5/22) | 978 | 2.84 | 0 |
| A4 model-only Stockroom S-QV (`922ab2f6` / C2 `205bf524`) | reference A4 `…190322Z-cf17a253` | Verified, applied | **not met**: name maximum defaulted to 80 without a question; s1 20/21, name maximum 60 6/10, names checker fails | 6 | 36 (15/9/12) | 1,959 | 4.11 | new realization and registry |
| | simplified A4 attempt 1 `…210017Z-f86d5de4` | Incomplete after 102 s | infrastructure: the run evidence shows the Orchestrator's context cancelled and the turn's container end not observed, so the run correctly refused certainty. The cockpit session confirmed and reproduced the cause: its `pkill -f "markitect cockpit"`, sent through PowerShell, lost its quotes and hit every markitect process. | – | 2 | 102 | 0.03 | – |
| | simplified A4-b `…211052Z-f08d3eb3` | Verified, applied | **not met**: name maximum defaulted to 100 without a question; s1 20/21, name maximum 60 6/10, names checker fails. Verify found hyphen names and a 5000-digit quantity refused; the correction Implement repaired both; fresh Verify closed all 8 findings as resolved. | 4 (implement, verify NC, implement, verify) | 24 (8/4/12) | 1,418 | 2.64 | new realization and registry |
| A5 owner sets maximum 40 (from each run's own A4) | reference A5 `…193650Z-b7461b2b` | Verified, applied | met: name maximum 40 10/10; s1 20/21 with only the hard-coded 60 rule failing, as expected | 4 | 24 | 1,014 | 1.44 | 3 |
| | simplified A5 `…213643Z-d1e6fef1` | Verified, applied | met: identical oracle picture | 2 | 12 (3/3/6) | 566 | 1.08 | 3 (+7/−8) |
| A7 S-X contradiction | reference A7 `…195432Z-641d73d5` | Failed: Root model error, nothing published | met | 0 | 3 | 73 | 0.15 | 0 |
| | simplified A7 `…210909Z-a64eb400` | Failed: Root model error after stockroom's escalation, nothing published | met | 0 | 3 | 42 | 0.10 | 0 |
| A2 conforming Depot on C2 (nearest reference A2-b on `e9643e4e`) | reference A2-b `…174632Z-5d2d7c76` | Verified, no change | met, after 7 Verify Passes, 6 of them needing correction | 10 | 209 (36/21/152) | 3,978 | 20.31 | 0 |
| | simplified A2-c2 `…215140Z-9f056e6c` | **Incomplete**: run deadline reached during the fifth Pass; nothing published | **not met**. The first Implement preserved the stand. Verify then demanded more test evidence and ended in a bookkeeping block; see below. | 5 (implement, verify NC, verify NC, implement, verify unsealed) | 159 (24/15/120) | 6,580 | 13.84 | none published; the unpublished correction added tests only |

**Series totals.** Simplified: 9 cases (including the lost A4 attempt), 11,411 s charged, about $28.52 estimated. Reference: 9 cases, 13,133 s. A3, displaced here by the A4 infrastructure repeat, and A6 ran in neither series. Haiku did not run. No series cut occurred.

**What the cases exercised.**
- **Detection (V1).** Both detect the defect themselves on the unchanged Verify code.
- **Repair, finding closure and fresh Verify.**
  - A1: the simplified Implement repaired the fault directly, so Verify had nothing left to close.
  - A4-b: the full cycle of a genuine finding, owner correction in a new Implement, and fresh Verify with resolved closure ran without help.
- **Preservation.**
  - A2 on C1′: no optional change, where the reference on the same base published 757 lines of optional tests.
  - A2-c2 on C2: see below.
- **Model gap (A4).** Both workflows default a materially open public value instead of routing it to the owner. The reference's later fix `fcd0fb54` targets that, independently of this workflow, and was not part of either evaluated build.
- **Contradiction (A7).** Correct in both.
- **Dependencies.** No work Pass of either series needed a cross-Manager wait. Waits between other Managers occurred only between reviewers in Verify (reference A1 one, simplified V1 two), where nothing binds. No run recorded a binding. The binding mechanism is therefore proved only by source tests: rebinding, reset, result-only reopening, the result gate and cycle refusal.
- **Conflicts or omissions from the missing Plan and Prepare.** None observed: no File check or ownership failure, no stalled or cyclic dependency, no Manager missing from a Pass. The model-only A4-b produced a complete Stockroom in one Implement. Its only oracle failure, the defaulted name maximum, is shared with the reference.

**A2 on C2.** The first Implement preserved the conforming Depot without any change, which confirms the preservation result of C1′ on the second base. The extra cost came from Verify, whose code C2 shares unchanged with the reference `922ab2f6`:

1. **First Verify (p2).** Root disposed of one of its own adverse probes as an expectation error, which needs a fresh assessment.
2. **Second Verify (p3).** The reviewers used mutation probes and judged the conforming tests insufficient as "meaningful tests for every modeled command". They raised 15 blocking findings, all missing test evidence; the program was correct.
3. **Correction Implement (p4).** It added discriminating tests and changed no program code.
4. **Third Verify (p5).** It raised further genuine test-evidence findings (F16 by stock, with surviving mutants R90 and R101; F17 and F18), and two Verify bookkeeping issues blocked it:
   - a finding Root had raised in p3 kept the orders child from reporting satisfied, although Root's own rerun showed it resolved;
   - oversized mutation probes timed out at the 180 s check deadline and could only be disposed of by an impossible passing rerun.

   Root escalated both to the user. This case has no owner channel, the same as the reference's A2, so Root could not submit and the run ended at its deadline. The outcome came about 10 minutes before the configured 23:51:40Z, for a reason not determined; WSL and Windows clocks agree afterwards. It had no effect, because the run was only waiting for an unanswered owner question.

The reference session confirmed both issues as defects of the shared Verify engine. It fixed them on its own branch `fix/verify-finding-and-probe-deadlocks` (heads `b1283e25`, then `95396f08`). Neither fix is part of a candidate evaluated here.

On C1 and C1′, V2 and A2 judged the same stand clean. That variance in Verify's judgment of test adequacy, and its cost, are therefore not effects of the workflow change. The nearest reference, A2-b on `e9643e4e`, preserved the stand only after 7 Verify Passes, 6 of them needing correction, and $20.31.

## Assessment

**Reliability and completeness first.** In every same-base pair, the simplified workflow reached at least the reference's result.

- **Better than the reference:** A1 and A2 on `8b5040ff`. The fault was repaired without a correction round, and the conforming Depot was preserved without optional changes. The reference traced its own A2 overreach to the planning duty to name intended public-boundary evidence, which Managers then implemented (`b976d8f0`). The interpretation that a separate Plan Pass made that commitment binding, while the merged mandate did not, is plausible but rests on single runs.
- **Equal to the reference:** V1, V2, A5 and A7.
- **Equally failed:** A4. Both missed the clarification, defaulting the name maximum (80 and 100).
- **Worse than the nearest reference:** A2-c2 ended Incomplete, while A2-b, on a different base, was Verified. The cause lies in the shared Verify engine, not in the workflow; see above.
- **New weaknesses:** none attributable to the simplification. The A4 infrastructure loss and the Verify deadlocks of A2-c2 come from outside the workflow change: another session's process kill, and Verify bookkeeping shared unchanged with the reference.

These are single runs per case. They show that the simplified workflow can work on these inputs, not that it is more reliable in general.

**What is not proved.** Plan and Prepare existed above all to agree on contracts across Managers and to lay foundations before dependent work starts. None of the evaluated cases needed that: the Depot's children work independently on Root's basis, and Stockroom has a single Package. The declared-wait binding that replaces plan dependencies has only source-test evidence. So has explicit invalidation of a revised basis, with its rebinding, result reopening, result gate and cycle refusal. Deep trees, helpers, Extract and Haiku are also unproved.

**Effort and time second.** Where results matched, the simplified runs needed fewer Passes and Turns and less time and cost:

| Case | Charged time, simplified vs reference | Cost |
| --- | --- | --- |
| A1 | 514 vs 2,993 s | $3.13 vs $12.47 |
| A2 | 978 vs 1,111 s | $2.84 vs $4.82 |
| A4 | 1,418 vs 1,959 s (excluding the lost first attempt) | $2.64 vs $4.11 |
| A5 | 566 vs 1,014 s | $1.08 vs $1.44 |
| A7 | 42 vs 73 s | $0.10 vs $0.15 |

V1 and V2 run identical Verify code and differ only by run variance. Part of the A1 saving comes from Implement repairing the fault before Verify, which one run cannot attribute to the workflow alone.

**Recommendation: improve, then adopt.** Do not keep the old workflow on this evidence. It cost more Passes, Turns and optional changes for no observed gain in correctness. Do not adopt the simplified workflow unchanged either, because the risk its removal of Plan and Prepare creates is exactly the one this series did not exercise. Before adoption:

1. Run one bounded multi-Manager case with a real contract dependency, for example Library lending → catalog. A missing foundation, an unbound dependency or a deadlock there would refute the simplification.
2. Have the owner decide on DEC-064 and the accepted workflow; this branch only labels them as an experiment.
3. Address the shared weaknesses this series exposed independently of the workflow:
   - clarification of materially open public values (the reference's `fcd0fb54`, unevaluated);
   - the two Verify bookkeeping deadlocks found in A2-c2: findings raised by an ancestor in an earlier Pass that block a child's satisfied result, and Runtime-error receipts of probes that cannot finish in the check deadline.

If adopted, remove the unused `Plans` engine path and its tests together.
