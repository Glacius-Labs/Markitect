---
name: markitect-architect
description: Architect a Markitect project with its owner - the first design that walks the template's areas, an idea of the owner, the owner's choice between prepared proposals - and run the architect job that changes the spec for a report, a back-fill or an idea. Use when the owner wants to design, change or extend the spec, start a project, or choose between proposals, or when triage switches a report to architect.
---
The spec is the project's statement of what should hold: an ordinary folder of Markdown documents, each of a type, whose path `.markitect/spec` names. Only the owner makes new decisions; everything that follows from what was decided is written without asking. A conversation never runs as a job, and a job asks no one.

## In a conversation with the owner

What a conversation changes lands on a branch of its own with the role `owner`. Lay out the before-state as the [frame of a job](../../jobs.md) does, and check the branch with `markitect-next status --step owner --before BEFORE` before you hand it on: the owner may change everything but verdicts. A spec change lands with its realization, so the branch goes to the `markitect-next:markitect-build` skill, not straight to the target.

**The first design.** In a folder without `.markitect/`, run `markitect-next init`, then `markitect-next apply software`, or `markitect-next apply company` for a business. `markitect-next check` then lists each empty section: the work to do. Walk the areas of the template with the owner in its order, depth first, finishing an area with everything below it before the next: for a software project purpose, context, domain, interfaces, quality, architecture. Put the questions of `markitect-next questions` that bear on each area. For each area:

1. **Propose** in a short table with the columns What I propose, Why and What else would do, each row as what the owner will notice of it. Ask only what the owner alone can decide, one question at a time, each with your proposal.
2. **The owner settles it:** they confirm, change, or leave it to you; what they leave to you, you decide and name at the end.
3. **Write it now:** documents with `markitect-next new TYPE PATH TITLE`, one thing per document, every statement as strong as it can be, each fact once and linked where it is needed. Run `markitect-next check`, and read the area against what is settled: `markitect-next topic AREA`.
4. **Commit it** on the branch, so the owner sees the spec grow.

The conversation ends with the spec on its branch and the build job starting on it. What stays open becomes a report with Source `owner, DAY`.

**An idea of the owner.** Write it as a report with Source `owner, DAY`: its place, what is wanted as the finding, and no solution. Then run the architect job below for it. Its branch stays unbuilt until the owner takes it; the `markitect-next:markitect-briefing` skill shows how it would change the spec.

**The owner's choice.** For an escalated report, show its proposals with their effects and the recommendation, and let the owner choose. On a branch of its own, write the choice and the owner's comment in its Actions below the proposals, with a line `branch markitect/NAME` first, and the decision as a document of the Decision type, with why and the alternatives weighed. Then hand the branch to the build job.

**Past the spec.** A change this conversation makes beyond the spec, such as to code, becomes a report "past the spec" with Source `owner, DAY`, the diff and the commit messages as evidence; triage sorts it.

## As a job

The architect job runs in the [frame of a job](../../jobs.md). The job is `architect`, on the branch the report's Actions name. Start `markitect-next:architect` with its task and the report or idea:

- **a report switched to architect,** whose answer follows from what was decided;
- **a back-fill** of a report "past the spec" that fills a gap, with the diff and the commits it names;
- **an owner's idea,** prepared unbuilt;
- **the first spec of an existing project,** stated from every artifact.

Write the reports of its answer. Then hand the branch to the `markitect-next:markitect-build` skill, so that the counter-checker derives counter-checks from the new text and the reviewer confirms before it lands; an idea's branch waits for the owner instead. After a first spec of an existing project, the owner settles its reports in a conversation, and a full review by the `markitect-next:markitect-review` skill fills the state.
