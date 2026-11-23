# Agent

One kind of agent of the way of working: a fresh session that runs exactly one skill. Its document is what the skill is built from, and whoever starts the agent reads here how to start it. The skill is named `markitect-` and the agent's name, such as `markitect-review` for `review.md`; the agent's folder in the context is named like its document, such as `context/review/`. Take it for every skill of the way of working. Take an [observer](observer.md) instead for watching one source on a schedule; the agent `observe` runs it. The document is the agent's usual way of working, as a [role](../core/role.md) is a person's: what it does outside it is a deviation.

## Task
The one task, in one sentence, on the first line. Below it, the steps the agent takes, in order and complete enough to build its skill from. A step that a process, a step or a rule already states links there instead of repeating it. Example: `Carry out one story in the state ready.`, then `1. Claim the story ...`.

## Started by
Who starts it, as links: a role, a person's session, another agent, or a routine. Example: `[Norman](norman.md) or the [Housekeeper](housekeeper.md)`.

## Asks
`yes` when the agent talks with a human during its session, `no` when it runs alone. Example: `no`.

## Model
The model and the effort it runs with, one line per provider: the provider, then a model name as that provider knows it, then the effort, such as `Claude: Opus, effort high` and `Codex: Sol, effort high`. A single line `as the session` for an agent that has no model of its own and takes that of the session it runs in, or `as the observer says` for the agent that runs observers, which takes the Model of the observer it runs. Whoever starts the agent reads it here, so a change here holds from the next start, without a build. Example: `Claude: Sonnet, effort high`, then `Codex: Luna, effort high`.

## Changes
What the agent writes, one item per line: plan items and their sections, findings, folders of `.markitect/`, branches, code, documents of the spec. It leaves everything else as it is. Example: `the story's State, Claimed by and Result`.

## Schedule (optional)
When it runs without a person starting it: `daily`, `weekly`, or the occasion. Leave it out for an agent that runs only when someone starts it. Example: `daily, in the Housekeeper's run`.
