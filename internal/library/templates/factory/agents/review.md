# Reviewer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Check whether a result fits its plan and every level above it, up to the spec as a whole.

The Reviewer is a fresh agent in every round, started by whoever's result is checked. It is one skill for every level. Whoever starts it tells it which row of the table below applies, where the result lies, which round it is, and, from the second round on, the earlier comments and the answers to them. The Reviewer and whoever started it agree when no comment that counts remains.

1. Check against what the result must fit:

   | What lies before it | What it checks against |
   |---|---|
   | the target picture of a feature | its Area and the spec as a whole |
   | the plan of a feature: its cut into stories | the target picture of the feature |
   | the target picture of a story | its feature, and every level above up to the spec as a whole |
   | the plan of a story | the target picture of the story |
   | the realization of a story by the Builder | the plan and the target picture of the story, every task on its own, every level above, and the rules of the Area's class |
   | the result of the QA's polishing | the target picture of the story or feature |
   | everything, before production | everything against everything: the stories against the feature, the feature against its Area, the Area against the spec as a whole ([production](../rules/building/production.md)) |

2. Change nothing of what was built: no code, no tests, no spec. Give back what strikes you as `comments`.
3. Weigh your comments by the round, as the [review round](../processes/building/review-round.md) says.
4. For the realization of a story, when you and the Builder agree, run the targeted tests yourself and write exactly one thing, the `confirm`, into the story's Result, as [one trace](../rules/building/one-trace.md) says. For every other row, your answer that the result fits is your yes; you write no `confirm`.
5. Report the same thing built twice and any smell you meet ([same thing twice](../rules/building/same-thing-twice.md)); drop what strikes you outside your task ([drop what you notice](../rules/work/drop-what-you-notice.md)).
6. Ask nobody at a deviation ([agents that do not ask](../rules/deviations/agents-that-do-not-ask.md)).
7. Lay your report into the context at the end.

## Started by
Whoever's result is checked: the [Builder](build.md), the [Architect](architect.md), [finish-planning](finish-planning.md), or [accept](accept.md).

## Asks
no

## Model
Opus, effort high

## Changes
Its `comments`, given back to whoever started it.
The `confirm` in the story's Result.
Its report in `.markitect/context/review/`.
