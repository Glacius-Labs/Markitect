# A working day with delegated realization

Date: 16 October 2026. Desired user experience developed in the [owner discussion](../sources/working-day-discussion-20261016.md). The owner asked to preserve the scenario and assess its prerequisites. The concrete interaction choices below are assistant proposals, not additional accepted decisions or delivery commitments.

The person changes and approves intent. The organization realizes it, handles ordinary implementation and review work, and requests a decision when its authority or the model is insufficient. The person can inspect progress without coordinating every Agent exchange.

This develops the [organization and trust reflection](organization-and-trust-20261016.md). The [Apply concept](../../design/canonical-model/manager-apply-concept.md) owns realization behavior. The [concept register](../register.md) remains the owner of decisions and status.

## Morning model change

The owner changes the library's admission limit from five open loans to three. Existing loans remain intact and returnable. Draft editing does not itself start realization. An explicit instruction starts Apply against a selected model revision and the existing realization.

A short acknowledgement states the intended change, selected revision, scope and existing execution authority. It should not request permissions already granted. The first concept reconciles the full selected local model, including reviewed no-change results; it does not yet promise that only changed Packages need assessment.

The root delegates outcomes to the responsible Managers. Lending owns borrowing behavior, while related membership, catalog and reminder obligations receive appropriate assessment. The first strategy permits one active writing task. Work can advance through ready tasks when another task is waiting; the experience does not require concurrent writers.

The owner sees a compact status such as:

> In progress: lower the admission limit to three. Lending is updating its realization; related return and reminder behavior is being assessed. No owner decision is currently needed.

The task tree, actual prompts, candidate differences and evidence remain available on demand. The overview derives from the same history and must distinguish observed progress from an Agent's prediction.

## A question requiring the owner

Suppose a broader library also supports renewals and the model does not say whether a member already above the new limit may renew. This is a hypothetical extension: the authored M1 explicitly excludes renewals. The five-to-three change in the existing example does not require inventing this ambiguity.

If a binding answer already exists, Managers use it. If a necessary decision is missing, the responsible Manager routes a concise question through its parent:

> Decision needed: may members above the new limit renew existing loans? We suggest allowing renewals under the existing renewal conditions, because the new limit concerns additional borrowing. The answer affects renewal behavior and its checks. Unrelated assessment can continue.

The question includes the source of the ambiguity, recommendation, alternatives and consequences, affected work and why it exceeds the current task's authority. Silence is not approval. A required unresolved question prevents aggregate completion, even if other work finishes.

An answer changing intended behavior is recorded in the canonical model through a separate authorized model operation. Apply cannot edit its read-only model. A new Apply binds the updated model and reassesses; old candidates and evidence remain inspectable but do not automatically count as current approval. An answer that only points to an existing rule can instead supply recorded context without changing intent.

## Ordinary review stays inside the organization

Lending implements the change. An independent reviewer examines the actual candidate against its obligations. If a new eligibility check accidentally blocks returns for suspended members, the finding goes back to Lending. Correction, renewed checks and independent review follow within the configured finite limits.

The owner can see “review found a return regression; correction in progress,” but need not answer. Parents coordinate cross-area dependencies and integration. They cannot conceal an incomplete child result in a positive summary.

Interruption is warranted when a decision exceeds authority, a required model ambiguity remains, work cannot converge, or a limit prevents further progress. The request should explain what would unblock the work. An ordinary failed attempt that the organization can resolve within its mandate is visible history rather than a demand for owner attention.

## Result and further steering

A completion report states what changed, the model and exact candidate assessed, required review/check results, remaining limitations and publication outcome. For example:

> Applied and read back: additional borrowing is refused when three loans are already open. Existing loans remain intact and returnable. Boundary cases and connected behavior were checked. A return regression found during review was corrected. No required findings remain open in the assessed scope. These artifacts have not been deployed.

Applied artifacts, a merged branch, a release and an observed production deployment are distinct outcomes. The first Apply does not implicitly merge or deploy. A later delivery workflow would need explicit responsibilities and existing authority for those actions.

The owner should eventually be able to stop work, change priority or submit revised intent. Each action needs an observable acknowledgement and result. A request to stop is not proof that a provider process has stopped. Revised intent cannot silently replace a running operation's model binding; the safe initial behavior is to retain the old run and start a new bound attempt with fresh assessment.

| Information | Expected owner involvement |
| --- | --- |
| Progress, normal findings and bounded correction | Available to inspect; no response required. |
| Completed outcomes, costs where known and remaining limits | Concise completion or requested periodic report. |
| Required decision, exhausted mandate or unresolved execution state | Targeted interruption with the needed decision and affected scope. |

The product should allow the owner to be available for judgment without requiring continuous availability for routine coordination. Wall-clock examples in the discussion were illustrative, not a completion-time promise.

## Prerequisites against the current concept

Assessment basis: committed design at `bc53b58b2319d91581ea98457acb4a9534d95bfe`, especially the [Apply concept](../../design/canonical-model/manager-apply-concept.md), [Go architecture outline](../../design/canonical-model/manager-apply-go-architecture.md) and [implementation plan](../../design/canonical-model/manager-apply-implementation-plan.md). Architecture work is ongoing. This assessment concerns the written contracts, not a fresh audit of product code or a claim about implementation progress.

The central realization mechanisms are already described: Package responsibility, bounded context, artifact ownership, independent local and integration review, aggregate verification, finite budgets, needs-decision outcomes, durable history, interrupted-run reconciliation and stale-input refusal. They need concrete contracts, implementation and evidence; they should not be presented as missing ideas.

| Area | Already described | Smallest remaining specification for the desired day |
| --- | --- | --- |
| Starting a job and its mandate | Explicit Apply instruction, fixed model/configuration, artifact-only target, path ownership and finite budgets. | State how the owner selects and releases a draft revision, sees the effective mandate and changes it. A compact view of existing configuration can suffice; no new authorization framework is implied. |
| Owner questions and answers | Open questions, needs-decision outcomes, parent routing and separate model changes. | Define a durable question identity, affected work, answer authority, pending/resolved state and answer binding. Distinguish context clarification from new intent and show the resulting continuation or new Apply. |
| Stopping and changing work | Cancellation/session ports, historical attempts, recovery and stale refusal. | Define user-visible stop acknowledgement, actual stopped/uncertain result and how revised intent supersedes an older request. Start with a new run; automatic reuse and live reprioritization can wait. |
| Attention and progress | Concise reports, task tree, event timeline, usage and waiting distinctions. | Define which events interrupt, which are grouped into reports, and how unresolved waiting becomes visible without repeated alerts. Deadlines, budget treatment while waiting and the responsible next actor must be explicit. A CLI report or small view is enough initially. |
| Completion beyond artifacts | Guarded publication and read-back; merge/release/deploy expressly excluded from implicit Apply. | Keep the first promise at applied artifacts. Specify later delivery and operational verification separately if the desired mandate includes running software. |

These are gaps in the interaction contract or deliberately later scope, not reasons to replace the Manager architecture. Existing run, task, question and journal records should carry them where possible. A new cockpit, experience database, organization layer or Government mechanism is not required to demonstrate this day.

## What would establish that it works

The planned real M1 Apply and subsequent M2 Apply remain the first evidence. Inspect actual code, behavior, tests, documentation and history, including preserved behavior. Passing generated tests alone is insufficient.

After that base works, exercise a small owner-interaction scenario: a required question appears, independent work remains correctly bounded, an answer is recorded at the right authority, and the resulting run cannot publish against an obsolete model. Add interruption/recovery and an exhausted review budget. The result should make clear what the owner must decide without requiring transcript archaeology.

Across later changes, measure missed obligations, unnecessary questions, owner attention and correction outcomes. These observations determine whether routine supervision can safely decrease. Experience selection and operational doctor checks remain separate extensions, not prerequisites for the initial Apply demonstration.

Return to [product reflections](README.md).
