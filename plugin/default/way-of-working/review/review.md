# Review

Type: [Activity](../../core/activity.md)

## Trigger
A result is to be checked. Whoever's result it is starts a fresh Reviewer for each round: the [Builder](../team/builder.md), the [Architect](../team/architect.md) in the check and at the finish of a planning, the [Acceptance helper](../team/acceptance-helper.md), or the [Assistant](../team/assistant.md) before a [release](../accepting/release.md). They tell it which row of the table below applies; where the result lies: the story's branch and the folder where it is checked out, which holds the changed documents of the model of a planning, and, for a planning, the folder of `main` with the plan items, which reach `main` only at the finish and which the Reviewer reads as they are; which round it is; and, from the second round on, the earlier comments and the answers to them.

## Steps
1. Check the result against what it must fit:

   | What lies before it | What it checks against |
   |---|---|
   | the plan of a feature: its cut into stories | the change to the model the feature is for |
   | the plan of a story | the places in the model the story realizes |
   | a change to the model | every level above the place it changes, up to the model as a whole |
   | the realization of a story by the Builder | the plan of the story and the places in the model it realizes, every task on its own, every level above, and the rules that hold for the module, without what a story that has not landed still has to build ([the model runs ahead](../plan/the-model-runs-ahead.md)) |
   | the result of the polishing in accepting | the places in the model the story realizes |
   | everything, before production | everything against everything: the stories against the feature, the feature against the model ([production](../accepting/production.md)) |

2. Change nothing of what was built. Give back what strikes you as comments.
3. Weigh the comments by the round: there are at most five rounds, and from the third on, nitpicks and remarks of little value no longer count. The Reviewer and whoever started it agree when no comment that counts remains.
4. For a result that lands on `main` with its story, the realization by the Builder or the polishing in accepting: read what changed on the story's branch, restricted to the story's places and files, check with `markitect check`, and look through the change for secrets and customer data: with either in it, nothing lands ([what is critical](../deviations/critical.md)). On agreement, run the targeted tests yourself and confirm the documents the story names, in your report, as [one trace](one-trace.md) says; write nothing into the story. For the checks of a planning, the answer that the result fits is the yes; nothing is confirmed.
5. Report the same thing built twice and any smell you meet ([same thing twice](../building/same-thing-twice.md)); drop what strikes you outside the task ([drop what you notice](../observing/drop-what-you-notice.md)).
6. Leave a [report](../observing/report.md) in the context at the end. The report that confirms is written on the story's branch and lands with the story.

## Result
Whoever started the Reviewer has its comments, or its yes. For a result that lands on `main`, each place the story names is confirmed, with the files that realize it and the test that proves it.

## Participants
The [Reviewer](../team/reviewer.md).
