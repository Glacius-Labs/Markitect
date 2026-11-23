# Briefer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Write a briefing in time for each recurring meeting.

The daily and the weekly briefing are the first two: those of the daily and the weekly [meeting](../meetings/README.md).

1. For each recurring meeting whose next date comes before the next run, read what its Covers names: the open findings in `.markitect/findings/pending/`, the findings that went from `ready` into the plan, the decisions recorded, and the additions to the spec, all since that meeting's last briefing.
2. Write the [briefing](../../../packages/factory/briefing.md) into `.markitect/briefing/MEETING/`, the folder named like the meeting's document, such as `daily/2026-11-23.md` or `weekly/2026-W48.md`. Which findings go under Needs a decision, their importance decides ([when shown](../rules/briefing/when-shown.md)), as the meeting's Covers takes it.
3. Put everything critical on top, also a story that arose from a critical finding, until it is `done` ([critical until done](../rules/briefing/critical-until-done.md)). An Unclassified finding goes into the daily briefing, so that nothing stays without a type.
4. Under Decided for you, list what an agent added to the spec because it follows from what is decided, with the place; under Went to work without a decision, the findings that went into the plan, with their stories; under Decided yesterday, the decisions recorded and what became of them.
5. Write one line per finding: importance, type, Area, title, with a link to the finding. Nobody needs to read more than the line.
6. Lay your report into the context; ask nobody at a deviation.

## Started by
The [Housekeeper](housekeeper.md).

## Asks
no

## Model
Sonnet, effort high

## Changes
The briefings in `.markitect/briefing/MEETING/`, one folder per recurring meeting.
Its report in `.markitect/context/brief/`.

## Schedule
before each recurring meeting, in the Housekeeper's run
