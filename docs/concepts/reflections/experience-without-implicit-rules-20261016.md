# Experience without implicit rules

Date: 16 October 2026. Assistant proposal from the [experience discussion](../sources/experience-discussion-20261016.md), kept at the owner's request, together with the owner's later operational-check idea. Both are later extensions; the [delivery plan](../../implementation-plan.md) defers them. This develops the [institutional memory question](organization-and-trust-20261016.md#experience-and-institutional-memory).

Experience should help the organization ask better questions, recognize recurring failures and prepare decisions. A lesson never becomes a binding requirement by itself; authority comes from the accepted model and explicitly authorized decisions.

## Four distinct kinds of knowledge

| Kind | Meaning | Library example |
| --- | --- | --- |
| Observation | What happened on identified inputs, with retrievable evidence. | After suspension, a candidate still admitted another loan. |
| Explanation | A diagnosis supported by evidence, with its remaining uncertainty. | Lending reused an eligibility value captured before the status change. |
| Experience | A scoped lesson that guides later investigation. | For state-dependent permissions, test a sequence that changes the state between operations. |
| Binding rule | An obligation in the accepted model or an authorized decision. | Only an active registered member may start a loan; suspension does not prevent return. |

These are different meanings, not promotion stages. "The test passed after change X" is an observation; "X removed the cause" is a stronger claim that needs support.

## A small learning loop

1. After a relevant failure, decision or correction, a Manager proposes a short note linking the inputs, candidate, findings and outcome.
2. It separates observed facts from the proposed explanation and states what might transfer, under which conditions and with which limits.
3. An existing reviewer checks whether the evidence supports the lesson and its scope. Review grants no rule authority.
4. Later tasks receive a small, labeled selection of relevant notes next to their binding requirements, and judge applicability themselves.
5. Later use adds feedback (helpful, inapplicable, contradicted or superseded); revised notes keep links to earlier versions.

No learning Agent, semantic index or scoring is needed to start. Notes are short versioned records outside the canonical Definition roots, selected by explicit associations with Definitions, areas and artifacts. Which note revisions were supplied is recorded.

A lesson cannot waive a binding rule, and a different approach than before is assessed against the current requirements, not rejected for being different. Repeated evidence may motivate a proposal for a broader rule; adopting it follows the project's decision authority through a model operation, never through artifact Apply or a generated linter.

## Worked example

With [M1](../../design/canonical-model/library-example/README.md), suppose a candidate accepts a loan after the member was suspended. Investigation shows it reused an eligibility value from an earlier request. Lending repairs admission; review checks refusal without state change, continued return and reactivation; the model is unchanged. The lesson recorded is "test permission changes between operations", not "never cache membership data" and not "suspended members may do nothing", which would contradict the right to return. A later eligibility task receives the lesson as a question to examine, not as a ban.

## Operational check through the Managers

The owner proposed letting the Managers check logs from running systems for discrepancies with the model, and clarified that this is a check or doctor, not an experience mechanism ([owner clarification](../sources/experience-discussion-20261016.md#clarification-logs-support-a-check-or-doctor), [IDEA-017](../register.md#idea-017-operational-check-through-the-managers)).

```text
Applicable model + bounded operational evidence
-> responsible Managers check their areas
-> consolidated findings, evidence and assessment limits
-> authorized repair or escalation where needed
```

Each Manager receives a compact case, not an unbounded log stream: event and correlation identities, environment, deployed version, operation, state and recorded outcome, linked to the source and model basis of that version. Logs are untrusted observations, never instructions; secrets and unnecessary personal data are removed first. The report states what was examined and distinguishes "no discrepancy found" from "insufficient evidence". Diagnosis and repair are separate operations: a confirmed defect enters the normal work, review and guarded Apply, and deployment or production-data repair needs its own authority. An investigation never changes the model to fit faulty behavior. A confirmed diagnosis may optionally inform an experience note, but that is not its purpose.

Return to [product reflections](README.md).
