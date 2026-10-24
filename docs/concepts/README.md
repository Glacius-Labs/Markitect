# Concept record

This is the home of Markitect's long-term vision, product-level decisions, clarifications, promises, assumptions, concepts, future enhancements, ideas and open questions. Each is kept with its origin and status. The record exists so that owner decisions are not lost in chats, and so that each new session or reviewer does not have to derive them again. In this record, "owner" means Markitect's product owner, not the owner of an adopting project.

| Document | Content |
|---|---|
| [V1 concept 1.0](v1-concept.md) | The consolidated V1 target: scope, principles, organization, Run, workflow, harness, acceptance and decisions D1–D18 ([DEC-068](register.md#dec-068-v1-concept-10)) |
| [Register](register.md) | All entries with ID, status, origin and links, and the mapping of earlier collections |
| [Origin and rationale](origin-and-rationale.md) | The owner's originating project, causal sequence, problems, concepts and assumptions, connected to the existing register |
| [Product reflections](reflections/README.md) | Dated discussions of future use, organizational ideas and trust, with owner wording and assistant interpretation kept distinct |
| [Sources](sources/README.md) | Owner statements kept verbatim with provenance, including the origin account, product description, idea review and snapshots of the Concepts and Ideas chats' collections |

## Before assessing or redirecting Markitect

Read [Markitect in brief](../vision.md#markitect-in-brief), [Common misreadings](../vision.md#common-misreadings) and the [register](register.md) first. This applies to people and agents alike.

Accepted and clarified entries are the current basis. Do not reopen them without new evidence. If you have such evidence, add an open question that cites it and leave the entry unchanged until the owner decides.

## How this record relates to other documents

| Owner | Holds |
|---|---|
| [Vision](../vision.md) | The accepted thesis as readable prose. It links here for individual entries. |
| This record | Individual decisions, promises, assumptions, concepts, enhancements, ideas and open questions, with status and origin |
| [Delivery plan](../implementation-plan.md) | Current delivery scope and coordination. Chats/PRs record candidate progress and exact evidence. |
| [Architecture](../architecture.md) and the workflow guides | Implemented behavior |
| [Measurement](../guides/measurement.md) | Evaluation procedure |
| [Strategy sources](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/strategy/README.md) | Original strategy bundles from earlier phases, kept with their provenance |
| [Design records](../design/README.md) | Proposals, assessments and design evidence from different stages |

Summaries in the vision and other documents may restate entries. If a summary differs from an entry, the entry and its source govern; correct the summary.

## Entry types

| Prefix | Type | Meaning |
|---|---|---|
| `VIS` | Long-term vision | A picture of where the owner wants Markitect to go, beyond the current thesis. Planning follows the decisions that govern it. |
| `DEC` | Decision, clarification or endorsed direction | The owner has decided something, confirmed how the idea is to be read, or endorsed a direction whose plan is still pending. The status says which. |
| `PRM` | Promise | A target outcome Markitect is built to deliver. It is to be validated, not assumed. |
| `ASM` | Assumption or hypothesis | Something the idea relies on, with its evidence status. |
| `CPT` | Concept | A design concept that shapes the method. |
| `ENH` | Future enhancement | An agreed direction for product work, not yet scheduled. |
| `IDEA` | Idea | A suggestion that has not been evaluated yet. |
| `OQ` | Open question | A question that blocks or shapes a decision. |

## Status vocabulary

| Status | Meaning |
|---|---|
| Accepted | Decided by the owner |
| Clarified | The owner confirmed this reading of the idea |
| Endorsed; plan pending (or design pending) | The owner supports the direction; the plan or design is still open |
| Accepted direction | Agreed product direction |
| Accepted problem; approach open | The problem is agreed; the solution is still open |
| Proposed | Suggested by an agent or reviewer; the owner has not evaluated it yet |
| Owner idea | Raised by the owner; not yet evaluated |
| Owner statement | Said by the owner, possibly relayed by another chat; the entry names the relay |
| Parked | Recorded under the planning hold in DEC-011; not planned |
| Target | A promise to validate |
| Hypothesis, Owner experience, Owner observation | The evidence status of an assumption |
| Open | Not yet answered |
| Superseded | Replaced by a newer entry; it links to its successor |
| Rejected | Declined, with the reason |

This record does not track whether work is scheduled or implemented. When an enhancement is scheduled or delivered, add a link to its work item or to the document that now owns the behavior. The delivery plan, candidate PRs and architecture own their respective scopes.

## Adding and changing entries

1. **Preserve the source.** Add owner statements verbatim, in their original language, as a dated file under `sources/`. Any context added there must be labeled as a recorder's note. In the register, mark an interpretation as such.
2. **One entry per fact.** Link to the owning document instead of copying its content. The register gives the decision and its origin; the vision, architecture, roadmap and measurement keep their own subjects.
3. **Never overwrite silently.** A change of meaning gets a new entry, or the old entry is marked Superseded with a link to its successor. Keep the old entry.
4. **Only the owner decides.** An agent proposal enters as an `IDEA`, an `OQ` or a part of an entry that is explicitly labeled as a proposal. It stays that way until the owner decides. An agent's agreement, review or report is not acceptance.
5. **Recording is not scheduling.** An entry does not authorize implementation. The delivery plan and explicit package dispatches own scheduling; candidate evidence lives in PRs.
6. **Keep summaries in step.** When an entry changes, also correct any summary of it, for example in the vision's brief or its table of misreadings.
7. **Ideas move on.** When an idea is evaluated, it becomes a decision, an enhancement or a rejected entry. Its ID stays in the cross-references.

Write entries, and all other documentation, in English ([DEC-069](register.md#dec-069-documentation-is-written-in-english)). Under the record rules above, quotes in a source keep their original language; as noted in the DEC-069 scope notes, add an English translation when a source serves as a reference.
