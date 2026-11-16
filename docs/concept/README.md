# Concept

**You architect the project in Markdown. Markitect's tool checks the spec and says what is left to verify, an ordinary agent started by your own session builds the project to the spec, and a fresh reviewer verifies each Definition and records it.**

This page is the one description of the product, and the target that Main builds ([DEC-098](../decisions.md#dec-098-one-concept-describes-the-product)). The [decisions](../decisions.md) list what it rests on, the [roadmap](../roadmap.md) the next steps, and the [vision](../vision.md) why.

## The aim

Make one clean and simple version work reliably on Sonnet ([DEC-009](../decisions.md#dec-009-how-markitect-is-judged)): the spec as it stands gets applied reliably, with no human babysitting, review or other slowing down, so that people concentrate on architecture, which agents can't do by themselves. Every choice below is judged by that.

**The problem.** When agents build a project, trust and overview no longer come with the work, and nothing in the project states what it should be; the [vision](../vision.md#why-markitect-exists) gives the reasoning. Markitect supplies that statement: one spec with authority, and every file accounted for in the registry ([§2](what-markitect-is.md#the-spec)). The spec is how a change is made, so it cannot be ignored while work goes through it. On it, Markitect answers four questions:

| Question | Markitect's answer |
|---|---|
| Was something forgotten? | `status` lists what a change reaches, from the spec and the record, not from memory ([§3](tools.md#how-status-works)). Every Definition keeps a verdict at the inputs it was verified at, and a full review is `/markitect-review --all` |
| Was it understood correctly? | A fresh reviewer verifies every Definition a change reaches ([§5](what-build-does.md)) and records its verdict ([§6](record.md)) |
| Does the spec say what was meant? | Only the owner answers spec against intent, from walking every area of a new spec with the session as it is written, and from its hand-over: the Scenarios told as stories and a table of what the session decided beside what else would do ([§8](other-skills.md)). `vet` ([§4](is-the-spec-good.md)) and the audit support that. A reading view of the spec is an idea, not built |
| How does the project stand, and how is it organized? | Follow the spec, not the code: `list`, `doc` and `vet` show its structure, and it moves only when the owner accepts a change ([§3](tools.md)). `/markitect-tour`, `/markitect-catchup` and `/markitect-explain` walk a person through it, tell them what changed since they last looked and explain a part of it ([§8](other-skills.md)) |

**The way of working:**

1. A person architects the project in an ordinary Claude Code session, with the architect skill: the session asks, proposes and writes the spec, and hands a large draft to the architect agent. The person goes through it with the session, or reads the Markdown diff of a change, and the session commits it with the person's word.
2. The person runs the build skill. Agents started by the person's own session apply the spec as it stands, in a worktree of their own, so the person can go on architecting. They ask nothing about the spec, and nobody has to watch.
3. The spec stays as it was while they work: the hook keeps the agents that build out of it.
4. When `markitect status` lists nothing, or only `spec-problem`s for the person, the session commits the work with the record of what was verified and brings it to the person's branch. The person reads the verdicts when they want to, and answers only when the spec has a problem.
5. A bug hunt, a cleanup and a full review also run on a schedule, each on a branch of its own.

## The other pages

2. [What Markitect is](what-markitect-is.md)
3. [The tools](tools.md)
4. [Is the spec good?](is-the-spec-good.md)
5. [What build does](what-build-does.md)
6. [The record](record.md)
7. [How sure we are](how-sure-we-are.md)
8. [Other skills](other-skills.md)
9. [The harness](harness.md)
10. [Drivers](drivers.md)
11. [Providers](providers.md)
12. [Schedules and continuous work](schedules-and-continuous-work.md)
13. [What is shown and what is not](what-is-shown.md)
