# Scope of the first usable organization

Date: 16 October 2026. Assistant recommendation preserved at the [owner's request](../sources/working-day-discussion-20261016.md#scope-the-first-usable-version). This is input to consolidation, not an accepted release plan or additional backlog commitment. “First usable version” here means a product delivery, not a change to the completed canonical language v1.

The first version should be a small organization that performs delegated work reliably within its limits. Occasional manual operation is acceptable. Requiring the owner to schedule Agents, arrange reviews or reconstruct failed attempts would miss the intended working life.

The proposed promise is: the owner releases a model revision and a bounded assignment; Markitect organizes realization and assessment, raises necessary questions, records the owner's answers and returns either a verified result or an understandable incomplete outcome. The [working-day reflection](desired-working-day-20261016.md) develops the interaction; the [consistency review](concept-consistency-review-20261016.md) identifies contracts to consolidate.

## Essential capabilities

| Capability | Smallest useful delivery |
| --- | --- |
| A clear assignment | One local project, a fixed model/configuration basis, explicit authority and a shared budget. A CLI entry point is sufficient. |
| Autonomous delegation | Package-based Managers, a default Manager plus independent reviewer team, sequential writing and internal coordination of dependencies. |
| Complete assessment and publication | Local and integration reviews, required checks, final verification, guarded artifact/registry publication and read-back. |
| Questions and continuation | A concrete source-bound question and proposed edit, an unambiguously associated owner answer, then continuation or a successor Apply bound to changed intent. |
| Bounded consumption and honest failure | Finite attempts and execution time, protected completion capacity, progress assessment after repeated failure, and controlled stopping. |
| Inspectability and recovery | Compact status, open decisions, retained candidates and evidence, and reconciliation of actual execution state before resuming interrupted work. |

Keep the initial configuration simple: one organization strategy, one standard team shape and supported execution profiles. Existing responsibility and review boundaries remain in force even when execution is sequential.

### An explicit initial convenience limit

The organization must formulate the question, preserve the answer and continue correctly. An authorized model edit may initially be performed by the owner or the calling coding agent, outside artifact Apply. Markitect then acquires the actual new model and continues the assignment using retained work under a new run, without inheriting previous approval.

This leaves a visible manual step but does not require rebuilding the organization or reconstructing its work. Automatic application of the exact authorized model diff is an early follow-up convenience. Pending or failed model edits must remain visible; recording an answer does not imply the change or continuation happened.

## Deliberately deferred extensions

| Later capability | What the first version provides instead |
| --- | --- |
| Cockpit, live visualization and push notifications | Readable status and accessible questions, with details on demand. No requirement to inspect full Agent transcripts. |
| Many team variants and arbitrary role composition | One working standard team preserving independent review. |
| Parallel writing | Sequential candidate work and explicit integration. |
| Automatic reuse of prior approvals or selective review skipping | Retained work, current suitability assessment and renewed required checks/reviews. |
| Experience storage and intelligent selection | Retrievable execution history that later experience notes can cite. |
| Operational doctor and continuous monitoring | Realization and assessment of governed project artifacts. |
| Automatic merge, release and deployment | An explicit completion boundary at applied artifacts, with no claim that they are running in production. |
| Government and autonomous model-policy decisions | Local decisions within a mandate; new business meaning remains with the authorized owner. |
| Sophisticated budget optimization | One shared account, bounded allocations and protected review/integration capacity. |

Deferring these capabilities does not authorize ignoring obligations already present in a project's selected model. The first supported scope and operation target must be explicit rather than presenting an unsupported obligation as completed.

## Reliability boundaries that must remain from the start

Do not defer fixed input bindings, artifact ownership, independent review, publication guards or honest outcomes. In particular, the organization must not silently change the model, issue its own unreviewed approval, publish stale results, lose retained work or retry indefinitely. Unknown activity is reconciled before another writer takes over.

These are foundations of the method. Adding them later would change the core workflow, whereas adding a richer UI or additional operation can build on the existing boundaries. Limited initial capability is compatible with the concept; weakened authority or evidence is not.

## Two distinct demonstrations

### Technical core

The existing [implementation plan](../../design/canonical-model/manager-apply-implementation-plan.md#the-actual-target) remains the owner of the primary sequence: fully author library M1, run real Managers to produce A1, inspect its actual behavior/code/tests/documentation and evidence, change the model to M2, then Apply to that existing A1 and inspect A2. Retain the planned independent expectations and fixed product/profile basis.

This establishes bounded evidence for initial realization and a subsequent model-driven change. Provider startup alone, fake-adapter tests or repairing only synthetic drift cannot replace it.

### Usable working day

The [concrete library acceptance procedure](working-day-demonstration-20261016.md) develops the four scenarios below, their passing criteria, allowed human interventions and proposed repeatability threshold.

After the base works, demonstrate the interaction and failure paths in small separate scenarios:

- A necessary question receives an authorized answer; the recorded model change leads to the correct continuation or successor without losing existing work or carrying old approval forward.
- An independent review finding leads to internal correction and renewed assessment without unnecessary owner intervention.
- An interruption is reconciled and work resumes without blind duplicate execution or an uncertain writer being handed over.
- Exhausted budget produces a retained, understandable incomplete result and, where useful, a concrete request for additional authority. It does not waive checks or reset consumption.

Do not overload one library run with all these situations or invent ambiguity in rules M1 already resolves. Separate cases make the cause of failure and the evidence easier to assess. These are recommended product acceptance scenarios, not results already observed or a promise of general reliability from one success.

The practical first-version criterion is that the owner may occasionally operate Markitect but should not have to direct its routine work. Convenience can grow later; responsibility, independent assessment and honest results must carry the first usable organization.

Return to [product reflections](README.md).
