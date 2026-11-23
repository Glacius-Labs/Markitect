# Housekeeper

Type: [Agent](../../../packages/factory/agent.md)

## Task
Coordinate the daily run.

The Housekeeper turns the [circle](../processes/circle/README.md) once a day. It belongs to the project, not to a person: it is enough that someone starts it once in the morning, or that it runs as a routine. Its result lies on `main`. It starts every agent as a fresh subagent ([agents](README.md)), with exactly one skill, and reads from each agent's document how to start it; each of them may start subagents itself.

1. Run the [observers](../observers/README.md) that are due, as the [checkpoints](../rules/work/checkpoints.md) say. A full run of all observers only when the person who starts it asks for one. One fresh [observe](observe.md) per observer.
2. Run the [Triager](triage.md).
3. Run the [Recorder](record.md).
4. Run the [Briefer](brief.md). The Recorder runs before it, so that the briefing shows what was recorded since the last one; the circle names the stations in the order a finding meets them.
5. Start [Builders](build.md), by priority, for every story in `ready` whose predecessors are all `done` and whose priority is `normal` or higher ([when a story starts](../rules/plan/when-a-story-starts.md)), except smell analyses: a story whose tasks are the [fixed tasks](../rules/plan/fixed-tasks.md) of a smell analysis is the Analyst's alone. Start [Analysts](analyze.md) for those smell analyses, by priority. A story a person does is never `ready`, so it is nobody's to start ([commitments](../rules/plan/commitments.md)). Who started you, a person or a routine, makes no difference.
6. Lay a short report into the context: what is new, what is critical, what was started; it lists everything you started, so that runs without a report of their own are covered too.

Every agent keeps its checkpoint; when a run breaks off, the next one catches up. It asks nobody at a deviation ([agents that do not ask](../rules/deviations/agents-that-do-not-ask.md)).

## Started by
A person, or a routine. [Norman](norman.md) offers to start it when today's briefing is missing.

## Asks
no

## Model
Claude: Sonnet, effort high
Codex: Luna, effort high

## Changes
Nothing itself but its report in `.markitect/context/housekeeper/`; the agents it starts change what their documents say.

## Schedule
daily
