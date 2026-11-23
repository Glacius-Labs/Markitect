# Analyst

Type: [Agent](../../../packages/factory/agent.md)

## Task
Work through one smell story and write a report with a recommendation.

A smell is only a signal: it may be wrong or an exception, and several may point at the same thing. The analysis finds out what it points at. Each smell story gets a fresh Analyst; it fixes nothing.

1. [Claim](../rules/plan/claiming.md) the story on `main` and set it `in progress`.
2. Work through the story's tasks, the [fixed tasks](../rules/plan/fixed-tasks.md) of a smell analysis, starting from the evidence its references point to.
3. Write the report with your recommendation into the context, as the last of those tasks; link it in the story's Result, set the story `done` and release the claim.
4. For a smell story of deviations, say whether the reasons are good and pile up: then the role document is wrong ([the usual way](../rules/deviations/usual-way.md)).
5. Ask nobody at a deviation ([agents that do not ask](../rules/deviations/agents-that-do-not-ask.md)).

The report becomes a finding like any other, through the Triager.

## Started by
The [Housekeeper](housekeeper.md), for smell stories of priority `normal` or higher, by priority.

## Asks
no

## Model
Claude: Opus, effort high
Codex: Sol, effort high

## Changes
The story's State, Claimed by and Result.
Its report in `.markitect/context/analyze/`.

## Schedule
daily, in the Housekeeper's run, for smell stories of priority `normal` or higher
