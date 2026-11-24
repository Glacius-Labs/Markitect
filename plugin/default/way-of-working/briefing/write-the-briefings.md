# Write the briefings

Type: [Activity](../../core/activity.md)

## Trigger
Daily for the daily briefing, weekly for the weekly one, and before each further recurring meeting that takes place before the next daily run. Or a person asks for one, at any time: [the Assistant](../team/assistant.md) starts it then, for what happened since the last briefing of any kind. The [Housekeeper](../team/housekeeper.md) starts it in the [daily run](../day/daily-run.md), after the Project manager, so that the briefing shows what was recorded since the last one.

## Steps
1. Write the daily briefing once a day: when today's lies on `main` already, leave it as it is. Read for each briefing what it holds: the open findings in `pending`, the findings that went from `ready` into the plan, the decisions recorded, and the additions to the model, all since that briefing's last issue.
2. Write each [briefing](briefing.md), named as [names in the work](../work/names.md) says. Begin with Where we stand, from the plan and the reports since that briefing's last issue: what landed, what is in progress and who works on it, what this run started, what is blocked, and for each feature in progress which of its stories are built and which are not yet; a few lines. Which findings go under Needs a decision, their importance decides ([when shown](when-shown.md)); for a further meeting, what that meeting covers.
3. Put everything critical on top, also a story that arose from a critical finding, until it is `done` ([critical until done](critical-until-done.md)). An Unclassified finding goes into the daily briefing, so that nothing stays without a type.
4. Sort the findings the Project manager carried out, attached ones included, by their outcome ([finding](../triage/finding.md)): under Decided for you, those added to the model without a decision, with the place; under Went to work without a decision, those that went into the plan or were attached to a claimed feature without one, with their stories or the feature; under Decided yesterday, those decided, and what became of them.
5. Write one line per finding: importance, type, module, title, with a link to the finding. Nobody needs to read more than the line.
6. Leave a [report](../observing/report.md) of the run in the context.

## Result
The daily briefing of the day lies on `main`, the weekly one once a week, and one for each further meeting that is due, each beginning with where we stand; and a briefing on request, when someone asked for one.

## Participants
The [Briefer](../team/briefer.md).
