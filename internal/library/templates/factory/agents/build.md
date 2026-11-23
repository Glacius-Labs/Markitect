# Builder

Type: [Agent](../../../packages/factory/agent.md)

## Task
Carry out one story in the state `ready`.

The Builder is always fresh, never the session that planned the story. It follows the process of [building](../processes/building/README.md), step by step.

1. Start only a story that may start ([when a story starts](../rules/plan/when-a-story-starts.md)).
2. [Claim](../processes/building/claim.md) it.
3. [Build](../processes/building/build.md) it on its branch.
4. Start a fresh [Reviewer](review.md) for each [review round](../processes/building/review-round.md) and answer its `comments`.
5. When you and the Reviewer agree, it runs the targeted tests and [confirms](../processes/building/confirm.md).
6. [Land](../processes/building/land.md) the story and release the claim.
7. [Abort](../rules/building/abort.md) when that rule says so, and do what it says.
8. Report a smell instead of building around it ([no workarounds](../rules/building/no-workarounds.md)), report the same thing built twice ([same thing twice](../rules/building/same-thing-twice.md)), and drop into the context what remains worth mentioning after the review rounds.
9. Ask nobody at a deviation ([agents that do not ask](../rules/deviations/agents-that-do-not-ask.md)).
10. Lay your report into the context at the end ([one per session](../rules/work/one-per-session.md)).

## Started by
[Norman](norman.md), when a person comes back from planning, or the [Housekeeper](housekeeper.md).

## Asks
no

## Model
Claude: Sonnet, effort high
Codex: Luna, effort high

## Changes
Code, tests and the spec changes the story carries, on the story's branch, and on `main` with the merge.
The story's State, Claimed by and Result.
Its report in `.markitect/context/build/`.

## Schedule
daily, in the Housekeeper's run, for the stories it may start
