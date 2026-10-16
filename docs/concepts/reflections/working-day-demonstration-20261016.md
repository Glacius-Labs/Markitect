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

## Overall acceptance and repeatability

The proposed initial threshold is three complete series of all four scenarios, using a fresh execution state for each case and the same fixed product/profile basis. Each case starts from its specified recorded baseline; within the recovery case, the interrupted state must survive the restart. Do not replace recovery with a new clean assignment.

Every series must satisfy all four scenarios without disallowed rescue work. Record failed attempts rather than selecting only successful runs. After a product correction or acceptance-relevant setup change, start a new acceptance series and retain the earlier failures. Three series are a proposed practical threshold, not a statistically established reliability target or an accepted project rule.

For the budget scenario, an honest bounded incomplete result is a passing test of limit handling and remains an incomplete assignment. For the other scenarios, safe stopping is useful evidence but does not substitute for the required successful continuation and final verified/applied result. Unauthorized model changes, stale publication, inherited approval without required reassessment, budget resets or unaccounted concurrent writers fail the relevant case regardless of whether the final code appears correct.

The bounded conclusion after passing is: in these library scenarios, the owner could concentrate on intent and decisions while Markitect organized ordinary development and handled the selected difficulties. Broader reliability, long-term hygiene, operational monitoring, economic advantage and freedom from human code review across arbitrary projects remain separate claims requiring further evidence.

Return to [product reflections](README.md).
