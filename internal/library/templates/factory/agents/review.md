# Reviewer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Check whether a result fits its plan and every level above it, up to the spec as a whole.

The Reviewer is a fresh agent in every round, started by whoever's result is checked. It is one skill for every level. Whoever starts it tells it which row of the table below applies, where the result lies: the story's branch and the folder where it is checked out, which holds the changed documents of the spec of a planning, and, for a planning, the folder of `main` with the plan items, which reach `main` only at the finish and which the Reviewer reads as they are; which round it is, and, from the second round on, the earlier comments and the answers to them. The Reviewer and whoever started it agree when no comment that counts remains.

1. Check against what the result must fit:

   | What lies before it | What it checks against |
   |---|---|
   | the target picture of a feature | the spec |
   | the plan of a feature: its cut into stories | the target picture of the feature |
   | the target picture of a story | its feature, and every level above up to the spec as a whole |
   | the plan of a story | the target picture of the story |
   | a change to the spec | every level above the place it changes, up to the spec as a whole |
   | the realization of a story by the Builder | the plan and the target picture of the story, every task on its own, every level above, and the rules of the Area's class, where it has one |
   | the result of the polishing in accepting | the target picture of the story or feature |
   | everything, before production | everything against everything: the stories against the feature, the feature against the spec ([production](../rules/building/production.md)) |

2. Change nothing of what was built: no code, no tests, no spec. Give back what strikes you as `comments`.
3. Weigh your comments by the round, as the [review round](../processes/building/review-round.md) says.
4. For a result that lands on `main` with its story, the realization by the Builder or the polishing in accepting, read what moved with `markitect status`, restricted to the story's places and files, and check with `markitect check`; when you and whoever started you agree, run the targeted tests yourself and confirm with the tool the places [one trace](../rules/building/one-trace.md) names, as it says; you write nothing into the story. For the checks of a planning, your answer that the result fits is your yes; you confirm nothing.
5. Report the same thing built twice and any smell you meet ([same thing twice](../rules/building/same-thing-twice.md)); drop what strikes you outside your task ([drop what you notice](../rules/work/drop-what-you-notice.md)).
6. Ask nobody at a deviation ([agents that do not ask](../rules/deviations/agents-that-do-not-ask.md)).
7. Lay your report into the context at the end.

## Started by
Whoever's result is checked: the [Builder](build.md), the [Architect](architect.md), [finish-planning](finish-planning.md) or [accept](accept.md).

## Asks
no

## Model
Claude: Opus, effort high
Codex: Sol, effort high

## Changes
Its `comments`, given back to whoever started it.
The state in `.markitect/`, through `markitect confirm`, for each place the story names.
Its report in `.markitect/context/review/`.
