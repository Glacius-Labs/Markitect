# Experience selection and runtime feedback

Date: 16 October 2026. The owner proposed using logs from running systems to expose discrepancies and asked to preserve the discussion of experience storage and selection. [Original messages](../sources/experience-discussion-20261016.md#experience-selection-and-runtime-observations) record that request. The mechanisms below are assistant proposals, not implemented capabilities or additions to the first Apply scope.

The [experience reflection](experience-without-implicit-rules-20261016.md) distinguishes observation, explanation, experience and binding rules. This note applies that distinction to two connected flows: supplying useful lessons to a new task, and turning operational signals into investigated findings and possible lessons.

## Store short lessons separately from their evidence

Keep detailed execution history and operational evidence retrievable. Store a short versioned experience note separately, outside canonical Definition roots. A small index contains explicit associations and lifecycle status; it does not need the full underlying transcripts.

| Note content | Purpose |
| --- | --- |
| Identity, revision and title | Refer to an exact lesson version and preserve earlier versions. |
| Responsible areas, model references and relevant artifacts | Find lessons connected to a task, including lessons from neighboring areas. |
| Observation and explanation | Distinguish recorded events from their supported diagnosis. |
| Suggested investigation | State what a future task should examine rather than prescribing a new rule. |
| Applicability and limits | Explain when the lesson helps, its assumptions and counterexamples. |
| Evidence references | Reach original inputs, candidate, incident, review and checks on demand. |
| Status and later feedback | Mark a lesson current, restricted or superseded, with reasons and links. |

These are proposed record contents, not a selected storage format or new Core Kinds. The existing history could own evidence references while a rebuildable index supports selection. An original observation remains tied to its original version. A lesson can remain useful across later commits if its conditions still apply; a different SHA alone does not invalidate it. A relevant model or architecture change can require reassessment.

## Select context with explicit reasons

Start with task scope: affected Definitions, responsible areas and artifacts. Select current lessons with explicit connections to those subjects, including cross-area connections. Matching the Manager name alone is too narrow. Label why each lesson was selected and let the Manager assess its applicability to the current task.

Initially supply a small set of short summaries, perhaps three to five as an experimental starting point. For additional matches, expose a compact topic index and targeted retrieval of notes and original evidence. A context limit must not imply that all relevant risks have been examined; omitted material remains discoverable. Binding requirements retain their own context section and are not displaced by optional experience hints.

A summary might read: “A previous lending candidate reused eligibility after a status change. This task changes eligibility handling. Check whether decisions survive a relevant state transition. The earlier diagnosis concerned a cache; it does not prohibit caching.” Include the lesson revision, source links and reason for selection.

Record which lesson revisions were supplied. Later findings can show that a selected lesson was helpful, misleading or irrelevant, or that a useful lesson was missed. Start with explicit associations and inspectable selection. Semantic retrieval or learned ranking remains an option if concrete missed cases justify it, not a prerequisite.

## Turn operational signals into bounded investigations

The proposed flow is:

```text
Operational events -> grouped observations -> suspicion case
-> responsible Manager investigation -> supported finding
-> reviewed correction when authorized -> reviewed experience note
```

No stage automatically proves the next. A signal can end as a logging defect, an expected outcome, an unresolved question or a finding about a different deployed version. An experience can also be recorded from a confirmed diagnosis while a repair remains pending; its status must retain that limitation.

Deterministic preprocessing can group repeated signatures, count occurrences and select representative cases. A specifically defined check can flag a structured invariant violation. Neither mechanism claims to interpret arbitrary business prose or discover every new failure pattern. Preserve counts, time window, grouping criteria, evidence pointers and sampling limits; combining events must not erase distinct failures.

Managers receive a compact case rather than an unbounded log stream. Useful evidence includes event and correlation identities, environment, deployed software version, operation, relevant state and recorded outcome. Link the deployed artifact/build manifest to its source and model basis, then route through the corresponding responsibility mapping. Missing or ambiguous bindings remain explicit; current ownership or current intent must not silently replace the historical basis.

Remove secrets and unnecessary personal data before agent access. Treat log messages as untrusted observations, never instructions or execution authority. Investigators can request additional bounded evidence where access permits it. Missing or expired evidence is a visible limit, not reconstructed history.

The investigation checks whether the event reflects a real behavioral discrepancy, an ordering issue, faulty instrumentation or a version mismatch. Logs alone may not expose missing behavior or omitted checks. Metrics, state inspection and targeted probes can supplement them when the question requires it. Silence in the logs does not establish correctness.

Confirmed artifact defects can enter the existing scoped work, review, verification and guarded Apply method under existing authorization. Deployment and production data repair need their own authority and outcomes. An investigation must not change the model to make observed faulty behavior conform. No monitoring service, alert schedule or automatic production action is established by this discussion.

## Library walkthrough from incident to lesson

This hypothetical scenario uses [M1](../../design/canonical-model/library-example/README.md) as read at `a078d72566939ee7f227f64f3649e15f6e872618`. M1 requires a callable local implementation and no production deployment. For this future-use illustration, assume such an implementation is running and emits useful events; that does not change M1's required delivery or claim an actual runtime result.

The governing [eligibility rule](../../design/canonical-model/library-example/m1/model/members/eligibility.yaml) prohibits new borrowing after suspension and preserves return rights. [Borrow](../../design/canonical-model/library-example/m1/model/lending/borrow.yaml) requires refusal without observable state changes. [Return](../../design/canonical-model/library-example/m1/model/lending/return.yaml) remains permitted for a suspended member.

| Stage | Illustrative case |
| --- | --- |
| Events | An active member borrows copy A. A status-change operation completes the suspension. A later request for copy B is recorded as accepted and a new loan is recorded. Use pseudonymous identifiers and link the relevant operations. |
| Suspicion | A defined monitor flags the apparent contradiction and groups matching cases. It supplies counts, representative event references, the deployed build/model binding and the suspected rule. Event timestamps alone do not establish the required order or state at admission. |
| Routing | Lending receives the case, with membership context available. The common parent coordinates any needed cross-area work; recipients do not gain write rights to each other's artifacts. |
| Investigation | Inspect correlated state transitions and the deployed candidate. Reproduce the ordered sequence on controlled inputs, without repeating writes against the live system. Check whether suspension completed before admission and whether the logged outcome matches actual state. If the basis is ambiguous, keep the diagnosis unresolved. |
| Explanation | Suppose reproduction and code inspection show that lending reused an eligibility value captured before suspension. Record that supported cause separately from the original event. A faulty log or a different deployed policy would require a different conclusion. |
| Correction | Lending prepares a candidate that enforces the existing rule. Fresh review and checks cover refusal without side effects, preserved existing loans, allowed return and subsequent reactivation. The model is unchanged. |
| Operational follow-through | Guarded Apply establishes the reviewed artifact result. Any authorized deployment has a separate record. Checking the corrected sequence on the new version provides bounded evidence; lack of new alerts alone does not prove resolution. Existing erroneous loans require a separately authorized reconciliation decision rather than silent deletion. |
| Experience | Create a reviewed note linked to members/eligibility and lending/borrow: investigate state-dependent permissions across operation sequences. Retain the incident, diagnosis, correction and actual verification outcome, including anything still pending. Do not promote this into a universal cache ban. |
| Next task | A later lending optimization receives the short lesson because it touches eligibility handling. The Manager checks whether its implementation has the same risk and opens the evidence when needed. Record the lesson revision supplied and any later feedback. |

The resulting observation is the accepted loan after suspension, the explanation is the established stale eligibility mechanism, and the experience is a scoped investigation hint. The binding business rule already existed. Learning improves future attention without silently creating a new architectural restriction.

Return to [product reflections](README.md).
