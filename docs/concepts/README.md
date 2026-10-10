# Concept record

This is the home of Markitect's product-level decisions, clarifications, promises, assumptions, concepts, future enhancements, ideas and open questions. Each is kept with its origin and status. The record exists so that owner decisions are not lost in chats, and so that each new session or reviewer does not have to derive them again. In this record, "owner" means Markitect's product owner, not the owner of an adopting project.

| Document | Content |
|---|---|
| [Register](register.md) | All entries, with ID, status, origin and links |
| [Owner product description, 9 October 2026](sources/2026-10-09-owner-product-description.md) | The owner's original description of the idea, verbatim |
| [Idea review, 10 October 2026](sources/2026-10-10-idea-review.md) | Owner correction, confirmed summary, review points and owner dispositions, verbatim where quoted |

## Before assessing or redirecting Markitect

Read [Markitect in brief](../vision.md#markitect-in-brief), [Common misreadings](../vision.md#common-misreadings) and the [register](register.md) first. This applies to people and agents alike.

Accepted and clarified entries are the current basis. Do not reopen them without new evidence. If you have such evidence, add an open question that cites it and leave the entry unchanged until the owner decides.

## How this record relates to other documents

| Owner | Holds |
|---|---|
| [Vision](../vision.md) | The accepted thesis as readable prose. It links here for individual entries. |
| This record | Individual decisions, promises, assumptions, concepts, enhancements, ideas and open questions, with status and origin |
| [Roadmap](../implementation-plan.md) and [Product Readiness backlog](../work-items/product-readiness/backlog.yaml) | Scheduling and status of implementation work. A scheduled enhancement's work item links back to its entry here. |
| [Architecture](../architecture.md) and the workflow guides | Implemented behavior |
| [Measurement](../measurement.md) | Evaluation procedure |
| [Strategy sources](../strategy/README.md) | Original strategy bundles from earlier phases, kept with their provenance |
| [Design records](../design/) | Proposals, assessments and design evidence from different stages |

Summaries in the vision and other documents may restate entries. If a summary differs from an entry, the entry and its source govern; correct the summary.

## Entry types

| Prefix | Type | Meaning |
|---|---|---|
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
| Target | A promise to validate |
| Hypothesis, Owner experience, Owner observation | The evidence status of an assumption |
| Open | Not yet answered |
| Superseded | Replaced by a newer entry; it links to its successor |
| Rejected | Declined, with the reason |

This record does not track whether work is scheduled or implemented. When an enhancement is scheduled or delivered, add a link to its work item or to the document that now owns the behavior. The roadmap, backlog and architecture own that state.

## Adding and changing entries

1. **Preserve the source.** Add owner statements verbatim, in their original language, as a dated file under `sources/`. Any context added there must be labeled as a recorder's note. In the register, mark an interpretation as such.
2. **One entry per fact.** Link to the owning document instead of copying its content. The register gives the decision and its origin; the vision, architecture, roadmap and measurement keep their own subjects.
3. **Never overwrite silently.** A change of meaning gets a new entry, or the old entry is marked Superseded with a link to its successor. Keep the old entry.
4. **Only the owner decides.** An agent proposal enters as an `IDEA`, an `OQ` or a part of an entry that is explicitly labeled as a proposal. It stays that way until the owner decides. An agent's agreement, review or report is not acceptance.
5. **Recording is not scheduling.** An entry does not authorize implementation. The roadmap and backlog own scheduling and status.
6. **Keep summaries in step.** When an entry changes, also correct any summary of it, for example in the vision's brief or its table of misreadings.
7. **Ideas move on.** When an idea is evaluated, it becomes a decision, an enhancement or a rejected entry. Its ID stays in the cross-references.

Write entries in English. Quotes in a source keep their original language.
