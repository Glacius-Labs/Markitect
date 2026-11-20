# Concept

**You architect the project in a typed, declarative specification language in Markdown. Jobs realize it, check it and measure how well the checking works; the tool computes what is built and verified; and you are asked only for the decisions that are yours.**

This page is the one description of the product, and the target that Main builds ([DEC-098](../decisions.md#dec-098-one-concept-describes-the-product)). The [decisions](../decisions.md) list what it rests on, the [roadmap](../roadmap.md) the next steps, and the [vision](../vision.md) why.

## The aim

Make one clean and simple version that works reliably ([DEC-120](../decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)): the spec as it stands is realized and checked without anyone reading the work, so that people concentrate on architecture and on the decisions only they can make.

**The problem.** When agents build a project, trust and overview no longer come with the work, and nothing in the project states what it should be; the [vision](../vision.md#why-markitect-exists) gives the reasoning. Markitect supplies that statement: one spec with authority, and every file of the project mapped to a document of it or ignored ([§2](what-markitect-is.md)). On it, Markitect answers four questions:

| Question | Markitect's answer |
|---|---|
| Was something forgotten? | `status` computes from the spec and the state what is changed, suspect or unexplained, not from anyone's memory ([§6](the-state.md)). Spot checks and rare full reviews check confirmed documents again ([§7](trust.md)) |
| Was it understood correctly? | A fresh reviewer confirms each document a change reaches. Counter-checks, derived from the spec alone before anything is built, catch a misreading that builder and reviewer share, and control samples measure how much the reviewer finds ([§7](trust.md)) |
| Does the spec say what was meant? | Only use shows it. Every finding names its place and the property it violates, "correct" among them ([§5](is-the-spec-good.md)), and a new decision goes to the owner as a choice between prepared proposals ([§9](reports-and-actions.md)). The spec itself is read and questioned regularly |
| How does the project stand, and what changed since I last looked? | The tool shows and Claude explains, with a citation for each statement: the briefing, explanations from the spec, and a comparison of two states ([§10](overview.md)) |

**The way of working:**

1. The owner designs the project with an agent, in a conversation. The first design walks every area of the template; later the owner drops in ideas, or designs along when they want.
2. Everything that calls for a change becomes a report: an idea, a bug report, a customer's wish, a log, a finding of a job. Triage sorts each one: a repair, a spec change that follows from what was decided, or a new decision for the owner.
3. Every change is a job on a branch of its own. For what `status` lists, a counter-checker derives checks from the spec, a builder realizes, and a reviewer confirms. The branch lands by one rule when nothing gets worse, a spec change together with its realization.
4. Routines keep the project in shape: spot checks, control samples, bug hunt, cleanup, improve, and reading the spec.
5. The owner looks into the briefing: what to decide, what was decided for them, incidents, and how the factory runs.

## The other pages

2. [What Markitect is](what-markitect-is.md)
3. [The language](the-language.md)
4. [What goes in a spec](what-goes-in-a-spec.md)
5. [Is the spec good?](is-the-spec-good.md)
6. [The state](the-state.md)
7. [Trust](trust.md)
8. [The factory](the-factory.md)
9. [Reports and actions](reports-and-actions.md)
10. [Overview](overview.md)
11. [The tool](the-tool.md)
12. [Jobs and runners](jobs-and-runners.md)
13. [What is shown and what is not](what-is-shown.md)
