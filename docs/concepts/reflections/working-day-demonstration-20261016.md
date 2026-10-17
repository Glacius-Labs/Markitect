# Demonstrating a usable working day

Date: 16 October 2026. Assistant proposal, kept at the [owner's request](../sources/working-day-discussion-20261016.md#preserve-the-working-day-demonstration) together with the owner's emphasis on four scenarios, passing criteria, allowed human interventions and repeatability. The [delivery plan](../../implementation-plan.md) schedules these scenarios after the technical core; they are follow-up evidence, not a gate of the first Apply.

The question is whether the owner can stay responsible for intent and decisions while Markitect organizes implementation, review, correction and continuation. Correct artifacts alone do not show a usable working day if the owner still has to schedule Agents or reconstruct lost work.

## Fix the starting point and expectations

The prerequisite is the [technical core](../../design/canonical-model/manager-apply-implementation-plan.md#the-target): real Managers realize M1 as A1, then reconcile A1 with M2. Before each scenario, record:

- the exact model, starting artifacts and registry, Markitect build, execution profile and configuration;
- the assignment, mandates, caps and completion reserve;
- independently authored expectations and the allowed owner interactions;
- the intended fault or interruption, where it is inserted and the evidence each criterion needs.

Each scenario starts from a recorded baseline and uses real Manager and reviewer execution. Controlled fault injection keeps it reproducible and must preserve the normal candidate and review contracts. Expected behavior is fixed before the run. Agent-written tests and reports are evidence to inspect, not acceptance. All attempts are preserved, including failures and interventions.

## Four scenarios

| Scenario | Setup | Passing behavior |
| --- | --- | --- |
| Question and continuation | A new, explicitly scoped one-time loan-renewal extension. Its other rules are fixed in advance; whether overdue loans may be renewed is reserved for the owner. Useful independent work exists before the answer. | The organization recognizes the missing authority, asks a concrete question with a proposed model edit, binds the answer to it, observes the actual model change and continues the same assignment in a continuation run. Suitable work survives; earlier approval does not. |
| Internal correction | A controlled candidate defect that permits another loan at the current limit, inserted before independent review. | The reviewer finds the violation independently, the responsible Manager corrects it, and renewed assessment verifies the correction and existing behavior. The owner neither explains the defect nor schedules review or repair. |
| Interruption and recovery | Execution stops after candidate work exists but before its completion is confirmed. | On resumption Markitect reconciles execution and artifact state before another writer starts, keeps useful work and consumption, avoids duplicate effects and finishes through required assessment. A safe block alone does not pass. |
| Insufficient budget | Deliberately insufficient capacity for writing work, with an explicit completion reserve. | Markitect stops within its caps, keeps candidates and findings and reports the unfinished work and any concrete extension request. It neither spends the reserve on implementation nor waives checks or resets consumption. |

The renewal case is a new extension: M1 excludes renewals and must not be portrayed as ambiguous. An unresolved business decision belongs to the recorded question, never to invented model semantics.

## Follow the question through to the result

> **Decision needed:** May an overdue loan be renewed?
>
> **Recommendation:** No.
>
> **Proposed model change:** Renewal is permitted only when the request date is on or before the existing due date. Refusal leaves the loan unchanged.
>
> **Work impact:** The eligibility condition waits for this decision. Completed independent work is retained.

The actual question also names its source basis and exact patch. The owner answers "Accept the recommendation." Each step is observed separately: the answer is bound to the right question version; the run finishes Incomplete awaiting the change; the patch actually changes the model and is committed; a continuation captures the new model with the same assignment and consumption; retained work gets a Keep, Adapt, Replace or Unresolved judgment; required reviews and checks assess the current candidate; guarded publication and read-back establish what was applied. In the first version the owner or calling agent applies the patch outside artifact Apply.

## Allowed human intervention

| Allowed first-version interaction | Intervention that fails the scenario |
| --- | --- |
| Start an assignment and inspect its status. | Remind Managers of their next routine step or route their work. |
| Answer a genuinely reserved business question. | Re-explain a rule already in the model to rescue the run. |
| Apply the exact authorized model patch. | Repair implementation or tests by hand. |
| Invoke supported continuation or restart. | Recreate child tasks, select reviewers or reconstruct lost candidates. |
| Grant additional capacity when asked. | Reset the ledger, bypass checks or silently widen authority. |

Every intervention is recorded with its reason and effect. Fault injection by the evaluator is setup, not an owner intervention; later coaching or repair by the evaluator is not setup.

## Status and evidence

The compact status and its evidence must answer, without reading Agent transcripts: what is finished and what remains; which decision is needed and what depends on it; what is blocked or uncertain; what capacity is consumed, reserved and left; what is locally checked, checked as a whole and actually applied. Applied artifacts are not merged, released or deployed.

## A small independent assessment

The [Playground](../../design/canonical-model/manager-apply-playground.md) and the [library evaluator](../../design/canonical-model/library-example/evaluation-contract.md) are extended with four scenario drivers; no second orchestration system is built. Three kinds of evidence complement each other: actual library behavior through the public interface, actual artifacts and execution history cross-checked against the runner's own observations, and a short independent judgment with a fixed checklist. The evaluator is an experiment role: it can invoke public operations and the declared injection but cannot route work, write findings or repair results.

| Scenario | Independent observations |
| --- | --- |
| Question and continuation | The real question and answer, the actual model patch, the same assignment with a new run, preserved consumption, retained work in the continuation's inputs and current assessments; timely, overdue and second renewal cases plus unchanged library cases. |
| Internal correction | The defective candidate fails the external limit case for the intended reason; the corrected artifact refuses the fourth loan after three with unchanged state. The review finding is independent of the evaluator's pre-check. |
| Interruption and recovery | The recorded stop point and process identities, evidence that the previous writer ended or lost write authority before another started, retained candidate bytes and monotonic consumption. If the environment cannot show this, the criterion is not proven. |
| Insufficient budget | Actual invocations against the declared reservations, refusal of writing work at the reserve boundary, no reset across restart, retained work and an honest Incomplete result. |

Process checks assert a few relationships rather than a fixed timeline: the answer precedes the model transition, assessments bind current inputs, no new writer starts while earlier authority is unresolved, and consumption is never reset. Before relying on them, a few negative evidence copies (a mismatched binding, an old approval on a continuation, missing writer termination, a reset ledger) must each prevent a pass. Every criterion reports passed, failed or not proven with its evidence.

## Keep expectations subordinate to the model

This answers the [owner's concern about a second specification](../sources/working-day-discussion-20261016.md#avoid-a-second-specification). The canonical model owns business intent; tests and expectation sets are realizations of it with provenance. Keeping them current is delegated work, not another document the owner synchronizes. A failing test is evidence to investigate, never authority over a newer authorized model.

Routine project checks read structured values such as the loan limit from the selected model snapshot and are reviewed against the model; when meaning changes, the responsible realization updates the affected checks with fresh review. Only a deliberate experiment that evaluates Markitect itself freezes its expectations, such as the five and three witnesses of the library evaluator. Each check references the obligation it assesses, reusing existing assessment and registry metadata rather than a manual requirements catalogue.

## Acceptance and repeatability

The proposed threshold is three complete series of all four scenarios with fresh state and the same product and profile. Every series must pass without disallowed rescue work; failed attempts are kept, and a product correction starts a new series. For the budget scenario, an honest bounded Incomplete result passes. Unauthorized model changes, stale publication, inherited approval, budget resets or unaccounted concurrent writers fail regardless of how the final code looks.

The bounded conclusion after passing: in these library scenarios, the owner could concentrate on intent and decisions while Markitect organized ordinary development and handled the selected difficulties. Broader reliability, long-term hygiene, economic advantage and freedom from code review on arbitrary projects remain separate claims.

## Deferred ideas

The owner adopted the previously deferred duty to examine consequential task premises and act within existing authority on 17 October 2026, as part of [technical risk derivation](../../design/canonical-model/manager-apply-concept.md#technical-risk-derivation) ([DEC-045](../register.md#dec-045-managers-and-reviewers-derive-technical-risks)). Its investigation, correction and evidence use the existing Manager/reviewer process; the remaining ideas below stay deferred.

These were discussed on 16 October and wait until the scenarios run:

- Escalations are judged against the information available at the time, and decision quality is evaluated beyond counting questions, including decisions made without asking.
- Feedback improves context and mandates without silently becoming policy.
- Integration cases are derived from intended outcomes and selected by distinct composed obligations, with coverage traced to check references ([ENH-003](../register.md#enh-003-test-to-statement-binding-and-test-review)).
- After a change, semantic checks are reassessed selectively by changed effects; the first Apply reconciles fully instead.

## Where the detailed rules went

The evening discussion of 16 October added detailed runtime rules here. They now live in the contracts that own them; the [original text](https://github.com/Glacius-Labs/Markitect/blob/3ebe3c6e6d283b998020350334e24a7303b089b7/docs/concepts/reflections/working-day-demonstration-20261016.md) remains in Git history.

| Topic | Now owned by |
| --- | --- |
| <a id="reduce-shared-misunderstandings-without-more-reviewers"></a><a id="select-deeper-checks-with-two-questions"></a><a id="keep-budget-pressure-from-weakening-assessment"></a><a id="select-consequential-silent-decisions-from-obligations-and-observable-effects"></a>Model-first discriminating cases, deeper checks and evidence under budget pressure | [Execution: evidence need](../../design/canonical-model/manager-apply-execution.md#evidence-need-survives-budget-pressure) |
| <a id="record-required-evidence-through-existing-findings"></a><a id="reassess-open-findings-in-a-successor-apply"></a><a id="keep-current-finding-context-small-without-losing-history"></a><a id="select-finding-context-from-recorded-responsibility"></a><a id="detect-incorrect-or-incomplete-finding-ownership-locally"></a>Findings, their resolution, carry-over into a continuation, context and ownership; the four applicability judgments became three finding states | [Execution: evidence need](../../design/canonical-model/manager-apply-execution.md#evidence-need-survives-budget-pressure) |
| <a id="keep-parent-coordination-bounded-and-prevent-forwarding-loops"></a>Bounded parent coordination | [Concept: tasks and delegation](../../design/canonical-model/manager-apply-concept.md#tasks-and-delegation) |
| <a id="release-a-dependency-from-an-accepted-delivery"></a><a id="correct-an-accepted-delivery-without-discarding-downstream-work"></a><a id="gate-active-work-by-recorded-delivery-dependencies"></a>Accepted deliveries and dependent work; exposure gating was dropped for the first version | [Execution: accepted deliveries](../../design/canonical-model/manager-apply-execution.md#accepted-deliveries-and-dependent-work), [north star](../../design/canonical-model/manager-realization-north-star.md#simplified-for-the-first-version) |
| <a id="record-supplied-inputs-rather-than-every-file-read"></a><a id="shape-useful-input-packages-without-unnecessary-artifact-exposure"></a><a id="give-reviewers-a-reproducible-obligation-context-with-explicit-limits"></a>Supplied inputs, input packages and reviewer context | [Concept: context packages](../../design/canonical-model/manager-apply-concept.md#context-packages) |
| <a id="detect-relationship-gaps-from-concrete-boundary-discrepancies"></a>Relationship gaps at Package boundaries | [Concept: checking effects across boundaries](../../design/canonical-model/manager-apply-concept.md#checking-effects-across-package-boundaries) |
| <a id="detect-consequential-context-gaps-during-ordinary-work"></a><a id="prefer-authorized-action-over-exhaustive-certainty"></a>Context gaps and authorized action | [Technical risk derivation](../../design/canonical-model/manager-apply-concept.md#technical-risk-derivation) |
| <a id="assess-questions-using-the-information-available-at-the-time"></a><a id="improve-context-and-mandates-without-turning-feedback-into-policy"></a><a id="evaluate-better-decisions-including-decisions-made-without-questions"></a>Decision quality and feedback | [Deferred ideas](#deferred-ideas) |
| <a id="derive-integration-cases-from-intended-outcomes-before-mapping-responsibilities"></a><a id="select-integration-cases-by-distinct-composed-obligations"></a><a id="derive-integration-coverage-from-check-references-and-current-review-evidence"></a><a id="select-semantic-check-reassessment-from-changed-effects-and-actual-use"></a>Integration cases, their coverage and selective reassessment | [Concept: review responsibilities](../../design/canonical-model/manager-apply-concept.md#distinct-review-responsibilities) and [deferred ideas](#deferred-ideas) |

Return to [product reflections](README.md).
