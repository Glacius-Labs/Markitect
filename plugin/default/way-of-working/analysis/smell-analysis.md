# Smell analysis

Type: [Activity](../../core/activity.md)

## Trigger
A smell story is `ready` with priority `normal` or higher. The [Housekeeper](../team/housekeeper.md) starts one fresh Analyst per such story, by priority, in the [daily run](../day/daily-run.md).

## Steps
1. [Claim](../plan/claiming.md) the story on `main` and set it `in progress`.
2. Work through the story's tasks, the [fixed tasks](../plan/fixed-tasks.md) of a smell analysis, starting from the evidence its references point to.
3. Write the report with the recommendation into the context, as the last of those tasks; link it in the story's Result, set the story `done` and release the claim.
4. For a smell story of deviations, say whether the reasons are good and pile up: then the role document is wrong ([the usual way](../deviations/usual-way.md)).

## Result
The story is `done`, and its Result links the report with the recommendation in the context. Nothing was fixed; the next triage makes a finding from the report.

## Participants
The [Analyst](../team/analyst.md).
