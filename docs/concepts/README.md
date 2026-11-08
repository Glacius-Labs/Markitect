# Concept record

This folder holds the product's one concept and the owner's decisions in force, so that owner decisions are not lost in chats and each new session or reviewer does not have to derive them again. "Owner" means Markitect's product owner, not the owner of an adopting project.

| Document | Content |
|---|---|
| [Concept 4.0](v4-concept.md) | The one description of the product, which Main builds: the spec, the tool that checks it and records what was verified, the skills and roles, the harness, and how sure each guarantee is |
| [Register](register.md) | The decisions in force, grouped by topic, with promises, assumptions, concepts, enhancements, open questions and ideas |
| [Sources](sources/README.md) | The few verbatim owner statements that pages on Main still cite |

Earlier concepts (V1 concept 1.0, concepts 2.0 and 3.0), the plans that built concept 4.0, the full register before its consolidation and the other owner sources are in the [archive](../archive.md#consolidated-on-8-november-2026).

## Before assessing or redirecting Markitect

Read [Markitect in brief](../vision.md#markitect-in-brief), [Common misreadings](../vision.md#common-misreadings) and the [register](register.md) first. This applies to people and agents alike. Do not reopen an entry without new evidence; if you have such evidence, add an open question that cites it and leave the entry unchanged until the owner decides.

If a summary elsewhere differs from an entry, the entry governs; correct the summary.

## Entry types

| Prefix | Type | Meaning |
|---|---|---|
| `DEC` | Decision | The owner has decided something or confirmed how the idea is to be read |
| `PRM` | Promise | A target outcome Markitect is built to deliver; to be validated, not assumed |
| `ASM` | Assumption | Something the idea relies on, with its evidence |
| `CPT` | Concept | A design concept that shapes the method |
| `ENH` | Enhancement | An agreed direction for product work, not yet scheduled |
| `OQ` | Open question | A question that blocks or shapes a decision |
| `IDEA`, `VIS` | Idea, long-term vision | Raised by the owner, or proposed and marked as a proposal; recorded and not planned |

The register lists only what holds. An entry says what holds, since when, and which earlier entries it consolidates; it does not track whether work is scheduled or built. The delivery plan and PRs own that.

## Adding and changing entries

1. **Record the decision, not the conversation.** An entry states what the owner decided and when, in plain words. It needs no verbatim quote and no source file, and other documents do not attribute their rules to the owner.
2. **One entry per fact.** Link to the owning document instead of copying its content.
3. **Never change meaning silently.** A new decision gets a new entry, the next free number of its type. When it replaces an entry, the old entry leaves the register and the new one says what it consolidates or supersedes; the full text of every earlier entry stays at its permalink in the archive. A restatement that keeps the meaning, as in a consolidation, keeps the ID and an anchor for every ID it absorbs, so old links still land.
4. **Only the owner decides.** An agent proposal enters as an `OQ` or an explicitly labeled proposal, and stays that way until the owner decides. An agent's agreement, review or report is not acceptance.
5. **Recording is not scheduling.** An entry does not authorize implementation; the delivery plan and explicit dispatches do.
6. **Keep summaries in step.** When an entry changes, correct concept 4.0 and any summary of it, for example in the vision's brief or its table of misreadings, in the same change.
7. **History moves out.** When an entry is superseded or only records a step on the way, it leaves the register in the change that supersedes it.

Write entries, and all other documentation, in English ([DEC-069](register.md#dec-069-documentation-is-written-in-english)).
