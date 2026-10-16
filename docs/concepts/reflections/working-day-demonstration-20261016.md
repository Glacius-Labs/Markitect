# Demonstrating a usable working day

Date: 16 October 2026. Proposed acceptance procedure preserved at the [owner's request](../sources/working-day-discussion-20261016.md#preserve-the-working-day-demonstration). This develops the [first usable version boundary](first-usable-version-20261016.md#usable-working-day). It is not an executed result, an accepted release gate or evidence of general reliability.

The question is whether the owner can remain responsible for intent and decisions while Markitect organizes implementation, review, correction and continuation. Correct artifacts alone do not establish a usable working day if the owner must keep scheduling Agents or reconstructing lost work.

## Fix the starting point and expectations

The prerequisite is the separate [technical-core demonstration](first-usable-version-20261016.md#technical-core): real Managers realize library M1 as A1, then apply M2's change from five permitted open loans to three against that actual implementation to produce A2. This procedure starts only after that prerequisite has been demonstrated; this document does not claim it has already happened.

Before execution, record:

- The exact model, starting artifacts and registry, Markitect build, execution profile and configuration.
- The assignment, mandates, shared budget, time limits and protected review/integration capacity.
- Independently authored business expectations and the allowed owner interactions.
- The intended fault or interruption, its insertion point and the evidence required for each scenario.

Use four separate scenarios from a recorded baseline. Use real Manager and reviewer execution for the working-day evidence. Controlled fault injection makes the scenario reproducible; fake-adapter tests can support implementation checks but cannot substitute for this demonstration. Injection must preserve the normal candidate and review contracts, including content bindings.

Expected behavior is fixed before the run and checked against the actual resulting artifacts. Agent-written tests and reports are evidence to inspect, not their own acceptance authority. Preserve all attempts, including failures and manual interventions.

## Four scenarios

| Scenario | Setup | Passing behavior |
| --- | --- | --- |
| Question and continuation | Prepare a new, explicitly scoped one-time loan-renewal extension. Fix its other rules in advance, but reserve the decision whether overdue loans may be renewed for the owner. Arrange for useful independent candidate work to exist before the answer. | The organization identifies the missing authority, asks a concrete question with a proposed model edit, binds the answer to that question, observes the actual authorized model change and continues the same assignment in a successor Apply. Suitable work survives; previous approval does not. Required assessment covers the current result. |
| Internal correction | Introduce a controlled candidate defect that permits another loan when the current limit has already been reached, before independent review. | The reviewer independently identifies the violation, the responsible Manager corrects it, and renewed assessment verifies the correction and relevant existing behavior. The owner does not explain the defect, schedule review or repair the code. |
| Interruption and recovery | Interrupt execution after candidate work exists but before its completion is unambiguously confirmed. Record the actual injection point. | On resumption, Markitect reconciles observed execution and artifact state before allowing another writer. It preserves useful work and consumed budget, avoids blind duplicate effects and competing writers, and continues through required assessment. If activity remains uncertain, it stays visibly blocked rather than pretending recovery succeeded. Such a safe block alone does not pass the complete recovery scenario. |
| Insufficient budget | Give the assignment deliberately insufficient unprotected capacity to finish. Keep completion reserves and limits explicit. | Markitect stops within the configured bounds, retains candidates and findings, and reports the unfinished work and any concrete request for additional capacity. It neither spends protected verification capacity on extra implementation nor waives checks or resets consumption. |

Run the cases separately so that failures have an identifiable cause. The renewal case is a new extension: the existing [library M1](../../design/canonical-model/library-example/README.md) excludes renewals and must not be portrayed as ambiguous. The prepared extension explicitly changes that scope. An unresolved business decision belongs to the recorded question/assignment; it must not be treated as permission to invent final model semantics.

## Follow the question through to the result

An example question in the prepared renewal scenario is:

> **Decision needed:** May an overdue loan be renewed?
>
> **Recommendation:** No.
>
> **Proposed model change:** Renewal is permitted only when the request date is on or before the existing due date. Refusal leaves the loan unchanged.
>
> **Work impact:** The eligibility condition waits for this decision. Completed independent work is retained.

The actual question also identifies its source basis and exact proposed model patch. The owner answers that specific question with “Accept the recommendation.” Record these as separate observable steps:

1. The answer is associated with the correct question and source revision.
2. The authorized patch actually changes the model. An answer alone does not imply a successful edit.
3. A successor Apply captures the actual new model while retaining the assignment identity and consumed budget.
4. Existing work receives a current suitability judgment: Keep, Adapt, Replace or Unresolved. Record the retained candidate identity and its use so that rebuilding everything cannot masquerade as reuse.
5. Required reviews and checks assess the current inputs and candidate. Earlier approvals cannot authorize the new result.
6. Guarded publication and read-back establish which artifacts were actually applied.

For the first version, the owner or calling coding agent may apply the exact authorized patch outside artifact Apply and invoke continuation. That explicit convenience limit does not permit recreating child assignments, manually assigning reviewers or transferring candidate files by hand.

Independent behavior checks include a permitted timely renewal, refusal of an overdue renewal without state change, refusal of a second renewal and preservation of existing borrowing and return rules. The prepared scenario must define the other renewal semantics, including the new due date, before execution; the example question is not a complete feature specification.

## Allowed human intervention

Record each intervention with its reason and effect, not only a total count. Required owner decisions are expected behavior; routine management performed by the owner is a failure of this acceptance objective.

| Allowed first-version interaction | Intervention that fails the affected scenario |
| --- | --- |
| Start an assignment and inspect its compact status. | Remind Managers which routine step to take next or manually route their work. |
| Answer a genuinely reserved business question. | Re-explain a rule already available in the selected model to rescue the run. |
| Apply the exact authorized model patch. | Repair implementation or tests manually so the scenario can finish. |
| Invoke supported continuation or restart. | Recreate child assignments, select reviewers or reconstruct lost candidates. |
| Explicitly grant additional budget when requested. | Reset the budget ledger, bypass checks or silently enlarge authority. |

Any requested budget extension starts a recorded continuation; it does not turn the original insufficient-budget outcome into a completed assignment. Fault injection by the evaluator is separately recorded as experiment setup and is not an owner intervention. Subsequent evaluator coaching or repair is not setup.

Independent evaluation of the final result is part of this demonstration, not a requirement that the owner review every routine production diff. No unrecorded human assistance is allowed behind the scenes.

## Status and evidence

An observer should be able to answer the following from the compact status and linked evidence without reconstructing Agent transcripts:

- What is finished, and what remains?
- Which owner decision is needed, and what depends on it?
- What is blocked or uncertain?
- What budget has been consumed, allocated, protected and left available?
- What is locally checked, checked as a whole and actually applied?

Applied artifacts do not establish merge, release, deployment or correct production operation. Preserve those boundaries in the report.

A small result record per scenario is sufficient: fixed starting basis, injection details, observed events and candidate references, owner interactions, budget/time consumption, independent behavior results and a pass/fail judgment against each criterion. Evidence must distinguish what was actually observed from an Agent's claim. A convenient status report cannot replace the underlying evidence.

## A small independent assessment

This proposal answers the [owner's follow-up](../sources/working-day-discussion-20261016.md#keep-independent-assessment-small). Reuse the proposed [external Playground](../../design/canonical-model/manager-apply-playground.md) and [independent library evaluator](../../design/canonical-model/library-example/evaluation-contract.md). Those documents own the runner, interface, isolation, frozen inputs and business-case families. Extend them with the four working-day cases when implementing this proposal; do not build a second orchestration system or duplicate their specifications here. These designs do not yet establish an implemented evaluator or a passed experiment.

### Three complementary evidence sources

| Evidence | Smallest useful check | Boundary |
| --- | --- | --- |
| Actual library behavior | Build the captured artifact tree and call its public JSON Lines interface from the existing proposed external Python standard-library evaluator. Assert returned values and observable state before/after operations against independently authored expectations. | Generated tests and self-reports are not the oracle. An API trace establishes only the behavior exercised; retain the existing source inspection for obligations it cannot observe. |
| Actual artifact and execution history | Compare model patches, candidate/source identities and consistent Host history exports. Cross-check important transitions against the runner's actual invocations, provider/session lifecycle observations and publication read-back. | A Host event is stronger than an Agent's narrative but is still produced by the system under test. A history entry saying "verified" or "reconciled" is not sufficient by itself. Hashes bind bytes, not correctness. |
| A short independent judgment | Inspect the question, its authority/model effect, the review finding and the compact final status using a fixed checklist. Record reasons and links. | Semantic clarity and usefulness cannot be established by checking that fields are nonempty. A separate Agent may assist, but its approval alone does not establish acceptance. |

Use one frozen evidence bundle per scenario: source/model snapshots, relevant history export, external observations, behavior traces, owner-intervention record and verdict. Build from captured artifacts after writers are quiescent. Keep evaluator inputs outside worker write scope and use the Playground's declared enforcement; hashes alone are not isolation. Reuse its consistent history-export mechanism rather than copying a live database or adding another event store.

The independent evaluator is an experiment role, not another Manager. It can invoke the public assignment/answer/resume operations and the declared injection, but cannot route internal work, fabricate review findings, edit the result or rescue execution. Requirements and the public interface are visible to workers; assessment data must introduce no hidden business obligations.

### Concrete checks for the four scenarios

| Scenario | Independent observations and assertions |
| --- | --- |
| Question and continuation | Check the real question/answer association and the actual before/after model patch against the approved edit. Require the same assignment, a new run bound to the new model, preserved consumption and a retained candidate present in the successor's starting inputs. Check new required assessments against the actual current subject. Run timely/overdue/second-renewal cases and unchanged library cases. The renewal operation and remaining semantics require a separately frozen extension to the public contract; do not pretend M1's ten-operation interface already supports them. |
| Internal correction | Capture the defective candidate and show that the external limit case fails for the intended reason before correction. On the corrected artifact, the same case must pass: after three open loans, a fourth request is refused, the count remains three and the requested copy stays available. Inspect the independently produced review finding, its candidate binding, actual correction delta and renewed review. The external evaluator's pre-check remains outside the reviewer's input; otherwise it demonstrates following a supplied finding, not independent detection. If another check finds the defect before the reviewer, record that useful result but do not count it as evidence of reviewer detection. |
| Interruption and recovery | Record the controlled stop point and actual process/session identities outside the product history. Retain the interrupted workspace and state. Before another writer starts, require evidence of the previous writer's terminal state or effective loss of write authority, plus reconciliation of candidate/publication state. Check observed dispatch/write-authority intervals, retained candidate bytes and monotonically accounted consumption across resumption; then assess the final result. A finished artifact alone cannot reveal overlapping writers. If the chosen environment cannot establish these observations, mark the recovery criterion unproven. Evidence is limited to the injected interruption point, not every possible crash window. |
| Insufficient budget | Compare actual captured invocations with the frozen accounting unit and declared reservations/settlements. Check that new work stops at the applicable admission boundary, protected completion capacity is not allocated to ordinary implementation, and restart/successor does not reset consumption. Read back retained work and verify incomplete status and remaining need. Observe through confirmed quiescence, not merely a short quiet interval. Session/step accounting is not proof of a monetary spending ceiling unless actual cost accounting supports that claim. |

Reuse all applicable baseline business cases; only add cases for the new extension. For example, the borrowing-limit witness uses normal member/copy registration and three successful loans, records observable state, attempts the fourth loan and compares state again. A refusal message without unchanged state fails. This is a small concrete trace, not a reference implementation of the library.

For process checks, assert a few required relationships rather than a complete prescribed timeline: answer before the authorized model transition, current input/candidate bindings for assessment, no new writer while prior authority is unresolved, and no reset of shared consumption. Do not require a particular delegation order when several valid orders exist. Reuse product exports as data without importing product scheduling, approval or accounting code as the independent expected-result implementation.

### Keep human judgment bounded

Use three short questions for the evaluator's inspection:

1. Did the owner question identify a genuinely reserved decision, offer a concrete authorized change and describe its effect on ongoing work?
2. Did the independent review finding identify the seeded violation and require an actual correction, with evidence tied to the relevant candidate?
3. Does the compact status accurately convey decisions, blockers, budget and the distinction between local checks, aggregate verification and applied artifacts?

Confirm fresh reviewer execution from actual session/context preparation and candidate bindings. A different role label or session ID alone does not prove relevant context or fresh execution; even fresh execution cannot prove absence of correlated reasoning errors. The seeded defect provides a bounded demonstration of detection.

Record this evaluation after capture, without coaching the active organization. A one-time independent inspection of the acceptance procedure, its expectations and resulting evidence is compatible with removing routine owner code review. It does not require another continuing team, scoring model or full transcript reading for every run. A contradiction discovered during inspection remains a finding; it cannot be dismissed because mechanical checks passed.

### Small implementation and checks of the checker

The proposed addition is four explicit scenario drivers plus a few assertion functions and data files in the planned Playground, reusing its existing proposed manifest, capture and report shapes. Each driver contains preparation, public invocations, the declared injection and assertions. No scenario DSL, generic fault-injection platform, dashboard, duplicate Manager loop or new approval authority is needed. Select one concrete recovery interruption point first and disclose the untested windows.

Before relying on the added assertions, exercise a handful of controlled negative evidence copies outside real execution: a mismatched model/candidate binding, an old approval presented for the successor, missing writer-termination/fencing evidence and reset budget consumption. Each must prevent a pass for its intended reason. Reuse the library evaluator's existing proposed wrong-result checks, including mutation on refusal. These checks establish that the checker discriminates selected failures; they are not real working-day runs.

For every criterion report `passed`, `failed` or `not proven`, with its evidence reference. Missing or inconsistent evidence prevents overall acceptance, but distinguish a deficient capture/evaluator from an observed product defect. Re-evaluating a frozen bundle should require no new Agent execution. Bind the evaluator and case hashes for the repeatability series as well as the product/profile; preserve earlier results if the evaluator itself is corrected.

## Overall acceptance and repeatability

The proposed initial threshold is three complete series of all four scenarios, using a fresh execution state for each case and the same fixed product/profile basis. Each case starts from its specified recorded baseline; within the recovery case, the interrupted state must survive the restart. Do not replace recovery with a new clean assignment.

Every series must satisfy all four scenarios without disallowed rescue work. Record failed attempts rather than selecting only successful runs. After a product correction or acceptance-relevant setup change, start a new acceptance series and retain the earlier failures. Three series are a proposed practical threshold, not a statistically established reliability target or an accepted project rule.

For the budget scenario, an honest bounded incomplete result is a passing test of limit handling and remains an incomplete assignment. For the other scenarios, safe stopping is useful evidence but does not substitute for the required successful continuation and final verified/applied result. Unauthorized model changes, stale publication, inherited approval without required reassessment, budget resets or unaccounted concurrent writers fail the relevant case regardless of whether the final code appears correct.

The bounded conclusion after passing is: in these library scenarios, the owner could concentrate on intent and decisions while Markitect organized ordinary development and handled the selected difficulties. Broader reliability, long-term hygiene, operational monitoring, economic advantage and freedom from human code review across arbitrary projects remain separate claims requiring further evidence.

Return to [product reflections](README.md).
