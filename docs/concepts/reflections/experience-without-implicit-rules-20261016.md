# Experience without implicit rules

Date: 16 October 2026. Assistant proposal from the experience discussion, recorded at the [owner's request](../sources/experience-discussion-20261016.md). This develops the [organizational memory question](organization-and-trust-20261016.md#experience-and-institutional-memory). It is a design reflection, not an implemented learning mechanism or a new canonical language contract.

Experience should help the organization ask better questions, recognize recurring failures and prepare decisions. A useful lesson does not automatically become a binding requirement. Authority continues to come from the accepted model and expressly authorized decisions within their stated scope.

## Four distinct kinds of knowledge

| Kind | Meaning | Illustrative library example |
| --- | --- | --- |
| Observation | What happened on identified inputs, with retrievable evidence. | After suspension, a particular candidate still admitted another loan. |
| Explanation | A diagnosis supported by evidence, with uncertainty where it remains. | Lending reused an eligibility value captured before the status change. |
| Experience | A scoped lesson that guides future investigation or choice. | For state-dependent permissions, examine a sequence that changes the state between operations. |
| Binding rule | An obligation in the accepted model or an authorized decision, with explicit scope. | Only an active registered member may start a new loan; suspension does not prevent return. |

These are different meanings, not four automatic promotion stages. An observation may have competing explanations. A useful experience may never justify a general rule. A binding rule may already exist before the failure and supply the criterion by which it is recognized.

“The test passed after change X” is an observation. “X removed the complete cause” is a stronger conclusion needing support. Recording the first must not silently assert the second.

## A small learning loop

1. A Manager proposes a short experience note after a relevant failure, decision or correction, linking the actual inputs, candidate, findings and outcome.
2. It separates observed facts from the proposed explanation and identifies what might transfer to another task.
3. An existing reviewer checks whether the evidence supports that lesson and its proposed scope. Reviewing a lesson does not grant it rule authority.
4. Later tasks receive a small relevant selection alongside source references. Their Managers assess applicability to current conditions.
5. Later use adds feedback: helpful, inapplicable, contradicted or superseded, with reasons. Original event evidence remains intact; revised lessons retain links to earlier versions.

No new learning Agent is required for the first version. Existing Managers, reviews and run history could support short versioned notes outside canonical Definition roots. A full semantic index, scoring system or automatic model training is not a prerequisite.

## Contents and use of an experience note

The follow-up [storage, selection and runtime feedback reflection](experience-selection-and-runtime-feedback-20261016.md) develops note indexing, bounded context delivery and investigations of observations from running systems. It includes a library incident traced through to a reusable lesson.

A note should answer five questions:

- What happened, and what evidence establishes it?
- Which explanation is supported, and what remains uncertain?
- What should a later task investigate or consider?
- Under which conditions is the lesson relevant?
- What counterexamples, contradictory findings or open doubts limit it?

Identity, author, date, source revision and candidate references can come from the existing history. Start with stated evidence and limitations rather than an unsupported numerical confidence score. These are proposed note contents, not a selected serialized schema.

Supply binding requirements and experience hints in distinct context sections. For example: “An earlier candidate reused eligibility across a member-status change. Check whether this task also has state-dependent admission decisions; inspect the cited diagnosis before applying its lesson.” A role should receive enough original context to question the explanation.

A reviewer can report a violation of a current rule. A different approach from the one used before needs assessment against current requirements, not rejection merely because it differs. A lesson cannot waive a binding rule either. If an architecture or model change invalidates its assumptions, retain the old evidence but revise or retire the lesson for future selection. Old success is not current approval.

## Decisions and deliberate rule changes

A recorded choice can apply to one candidate, one area under specified conditions, or the whole project. Preserve the scope actually authorized. Repeating a choice does not expand its authority. Repeated evidence may motivate a proposal for a broader rule, with reasons, tradeoffs and intended scope.

Adoption follows the project's existing decision authority, including any explicitly delegated authority. No manual confirmation of every experience note is implied. Model changes remain separate from artifact Apply; incorporating a lesson into the model requires the corresponding authorized operation.

A regression test that checks an existing rule can be part of its authorized realization. A new linter that forbids a previously permitted approach introduces an additional constraint and needs an appropriate decision. Agent-generated checks and guidance must not become a hidden route for promoting hints into policy.

## Worked library example

This is a hypothetical walkthrough using the current [M1 model](../../design/canonical-model/library-example/README.md), inspected at documentation commit `2b82ba5a7b7b3f28073857c7e7a8a36e06619c87`. No failing implementation, run or test result is claimed. M1 leaves the callable interface open and requires no UI, so this example assumes no browser or HTTP API.

The existing [eligibility rule](../../design/canonical-model/library-example/m1/model/members/eligibility.yaml) requires lending to enforce active membership when admitting a loan. [Changing status](../../design/canonical-model/library-example/m1/model/members/set-member-status.yaml) preserves existing loans. [Borrowing](../../design/canonical-model/library-example/m1/model/lending/borrow.yaml) must refuse suspension without observable state changes, while [return](../../design/canonical-model/library-example/m1/model/lending/return.yaml) remains allowed. These requirements precede the imagined incident.

| Step | Hypothetical event and its meaning |
| --- | --- |
| Establish state | Register an active member and two available copies. Borrow the first copy successfully, remaining below the M1 limit. |
| Change state | Suspend the member. The existing loan and due date remain unchanged. |
| Discover failure | Attempt to borrow the second copy. Suppose the candidate accepts it. Record exact inputs and before/after state as the observation. |
| Investigate | Inspect the lending path. Suppose it reused an eligibility value from the first request. That is the supported explanation only if candidate inspection and a controlled reproduction establish it. |
| Repair and review | Lending corrects admission within its scope. Review checks both refusal of the second loan without state changes and continued ability to return the first. Checks also cover restoration to active status, with other borrowing conditions still applying. The model is unchanged. |
| Capture experience | Record that isolated happy-path tests missed a permission change between operations. Suggest checking relevant state transitions on later changes. Cite the incident, diagnosis, reviewed correction and test evidence if such a run is actually performed. |
| Reuse | A later task involving eligibility receives the note as a question to examine. It does not automatically receive a ban on caching or a prescribed implementation pattern. |

An overgeneralization would be “Never cache membership data.” Another would be “Suspended members may perform no operations,” which contradicts the existing right to return. The experience supports neither rule. A future implementation with valid freshness or invalidation behavior can satisfy the model and address the same risk differently.

If the organization later wants a mandatory cache policy, it must propose that additional constraint through the appropriate authority. Evidence of this incident alone does not authorize it. If a future model deliberately permits some borrowing by suspended members, the current model governs the new case and this old lesson must be reassessed.

The useful learning is therefore a better investigation habit, backed by a concrete failure and its limits. The existing business rule remains the basis of correctness. The organization can become more experienced without silently narrowing future implementation freedom.

Return to [product reflections](README.md).
