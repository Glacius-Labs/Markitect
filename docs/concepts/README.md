# Concept record

This is the home of Markitect's long-term vision, product-level decisions, clarifications, promises, assumptions, concepts, future enhancements, ideas and open questions. Each is kept with its origin and status. The record exists so that owner decisions are not lost in chats, and so that each new session or reviewer does not have to derive them again. In this record, "owner" means Markitect's product owner, not the owner of an adopting project.

| Document | Content |
|---|---|
| [Concept 2.0](v2-concept.md) | The target, accepted on 28 October 2026 ([DEC-086](register.md#dec-086-concept-20-is-the-target)): the redesign of DEC-079 to DEC-085 in one readable order; being built in the [delivery plan](../implementation-plan.md#concept-20-build)'s steps T1 to T7 |
| [V1 concept 1.0](v1-concept.md) | What Main implemented until step T1, and the record of V1; the target until 28 October 2026: scope, principles, organization, Run, workflow, harness, acceptance and decisions D1–D18 ([DEC-068](register.md#dec-068-v1-concept-10)) |
| [Register](register.md) | All entries with ID, status, origin and links, and the mapping of earlier collections |
| [Origin and rationale](../vision.md#origin-and-rationale) | The owner's originating project, causal sequence, problems, concepts and assumptions, connected to the existing register; a section of the vision |
| [Sources](sources/README.md) | Earlier owner statements kept verbatim with provenance, including the origin account, product description, idea review and snapshots of the Concepts and Ideas chats' collections |

## Before assessing or redirecting Markitect

Read [Markitect in brief](../vision.md#markitect-in-brief), [Common misreadings](../vision.md#common-misreadings) and the [register](register.md) first. This applies to people and agents alike.

Accepted and clarified entries are the current basis. Do not reopen them without new evidence. If you have such evidence, add an open question that cites it and leave the entry unchanged until the owner decides.

## How this record relates to other documents

The [documentation map](../README.md#canonical-status-and-evidence-owners) names the owner of every other subject, from the current design documents to the original strategy bundles in the archive. The [vision](../vision.md) holds the accepted thesis as readable prose and links here for individual entries.

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

1. **Record the decision, not the conversation.** A register entry states what the owner decided and when, in plain words. It needs no verbatim quote and no source file, and other documents do not attribute their rules to the owner. The files under `sources/` stay as the record of earlier decisions.
2. **One entry per fact.** Link to the owning document instead of copying its content. The register gives the decision and its origin; the vision, architecture, roadmap and measurement keep their own subjects.
3. **Never overwrite silently.** A change of meaning gets a new entry, or the old entry is marked Superseded with a link to its successor. Keep the old entry.
4. **Only the owner decides.** An agent proposal enters as an `IDEA`, an `OQ` or a part of an entry that is explicitly labeled as a proposal. It stays that way until the owner decides. An agent's agreement, review or report is not acceptance.
5. **Recording is not scheduling.** An entry does not authorize implementation. The delivery plan and explicit package dispatches own scheduling; candidate evidence lives in PRs.
6. **Keep summaries in step.** When an entry changes, also correct any summary of it, for example in the vision's brief or its table of misreadings.
7. **Ideas move on.** When an idea is evaluated, it becomes a decision, an enhancement or a rejected entry. Its ID stays in the cross-references.

Write entries, and all other documentation, in English ([DEC-069](register.md#dec-069-documentation-is-written-in-english)). Quotes in an existing source keep their original language; as noted in the DEC-069 scope notes, add an English translation when a source serves as a reference.
