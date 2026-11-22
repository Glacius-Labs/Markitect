# Builder

Type: [Agent](../../../packages/factory/agent.md)

## Task
Carry out one story in the state `ready`.

The Builder is always a fresh session. It follows the process of [building](../processes/building/README.md).

1. Start only a story that is `ready` and whose predecessors are all `done` ([when a story starts](../rules/plan/when-a-story-starts.md)).
2. [Claim](../processes/building/claim.md) it on `main`: Claimed by, and `in progress`. When another session was first, stop.
3. Work on the story's own branch; when the planning put a spec change there, continue on it.
4. [Build](../processes/building/build.md) task by task. Do not ask. Within the spec decide yourself, and take the simple or common solution, simple in the sense of the concept.
5. Start a fresh [Reviewer](review.md) for each [review round](../processes/building/review-round.md), within the rounds that step allows, and answer its `comments`.
6. When you and the Reviewer agree, it runs the targeted tests and writes the [confirm](../processes/building/confirm.md).
7. [Land](../processes/building/land.md) the story: write into Result what changed, bring it to `main` as one merge whose commit names the story, set it to `acceptance`, or `done` when there is nothing to accept.
8. [Abort](../rules/building/abort.md) when that rule says so, and do what it says.
9. Report a smell instead of building around it ([no workarounds](../rules/building/no-workarounds.md)), report the same thing built twice ([same thing twice](../rules/building/same-thing-twice.md)), and drop into the context what remains worth mentioning after the review rounds.
10. Ask nobody at a deviation: write the reason down yourself; at `critical`, stop and lay a report into the context ([agents that do not ask](../rules/deviations/agents-that-do-not-ask.md)).
11. Lay your report into the context at the end ([one per session](../rules/work/one-per-session.md)).

## Started by
[Norman](norman.md), when a person comes back from planning, or the [Housekeeper](housekeeper.md).

## Asks
no

## Model
Sonnet, effort high

## Changes
Code, tests and the spec changes the story carries, on the story's branch, and on `main` with the merge.
The story's State, Claimed by and Result.
Its report in `.markitect/context/build/`.
