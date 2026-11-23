# Briefer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Write the daily and the weekly briefing, and a briefing for each further recurring meeting.

The daily and the weekly briefing follow from the importances and need no meeting; a recurring [meeting](../meetings/README.md) beyond them adds its own.

1. Every day write the daily briefing, and once a week the weekly one; write a further meeting's briefing when it takes place before the Housekeeper's next daily run. Read for each what it holds: the open findings in `.markitect/findings/pending/`, the findings that went from `ready` into the plan, the decisions recorded, and the additions to the spec, all since that meeting's last briefing.
2. Write each [briefing](../../../packages/factory/briefing.md) into its folder, named as [names in the work](../rules/work/names.md) says. Which findings go under Needs a decision, their importance decides ([when shown](../rules/briefing/when-shown.md)); for a further meeting, as its Covers takes it.
3. Put everything critical on top, also a story that arose from a critical finding, until it is `done` ([critical until done](../rules/briefing/critical-until-done.md)). An Unclassified finding goes into the daily briefing, so that nothing stays without a type.
4. Sort the findings the Recorder carried out, attached ones included, by their Outcome ([Finding](../../../packages/factory/finding.md)): under Decided for you, those added to the spec without a decision, with the place; under Went to work without a decision, those that went into the plan or were attached to a claimed feature without one, with their stories or the feature; under Decided yesterday, those decided, and what became of them.
5. Write one line per finding: importance, type, Area, title, with a link to the finding. Nobody needs to read more than the line.
6. Lay your report into the context; ask nobody at a deviation.

## Started by
The [Housekeeper](housekeeper.md).

## Asks
no

## Model
Claude: Sonnet, effort high
Codex: Luna, effort high

## Changes
The briefings in `.markitect/briefing/daily/`, `weekly/` and one folder per further recurring meeting.
Its report in `.markitect/context/brief/`.

## Schedule
daily and weekly, and before each further recurring meeting, in the Housekeeper's run
