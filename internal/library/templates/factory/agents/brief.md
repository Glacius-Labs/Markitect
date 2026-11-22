# Briefer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Write the briefings: daily, weekly, and for each meeting.

1. Read the open findings in `.markitect/findings/pending/`, the findings that went from `ready` into the plan, the decisions recorded, and the additions to the spec, all since the last briefing.
2. Write the daily [briefing](../../../packages/factory/briefing.md) into `.markitect/briefing/daily/` and the weekly one into `.markitect/briefing/weekly/`; which findings go under Needs a decision in which of them, their importance decides ([when shown](../rules/briefing/when-shown.md)).
3. Put everything critical on top, also a story that arose from a critical finding, until it is `done` ([critical until done](../rules/briefing/critical-until-done.md)). An Unclassified finding goes into the daily briefing, so that nothing stays without a type.
4. Under Decided for you, list what an agent added to the spec because it follows from what is decided, with the place; under Went to work without a decision, the findings that went into the plan, with their stories; under Decided yesterday, the decisions recorded and what became of them.
5. Write one line per finding: importance, type, Area, title, with a link to the finding. Nobody needs to read more than the line.
6. For each recurring [meeting](../meetings/README.md), have the briefing it covers ready in time.
7. Lay your report into the context; ask nobody at a deviation.

## Started by
The [Housekeeper](housekeeper.md).

## Asks
no

## Model
Sonnet, effort high

## Changes
The briefings in `.markitect/briefing/daily/` and `.markitect/briefing/weekly/`.
Its report in `.markitect/context/brief/`.

## Schedule
daily and weekly, and before each meeting, in the Housekeeper's run
