# Briefing

What reaches the humans: short, ordered by importance, one line per finding with a link to it. Nobody has to read more than the line. The Briefer writes the daily and the weekly briefing, which follow from the importances, into `.markitect/briefing/daily/` and `.markitect/briefing/weekly/`, such as `daily/2026-11-23.md` and `weekly/2026-W48.md`; a recurring [meeting](meeting.md) beyond them gets its own, in a folder named like the meeting's document. Take it only for that. Do not take it to decide: people decide in conversation, and the Recorder carries out what they decided.

## Needs a decision
The findings in `pending` that wait for people, critical first, then by importance; which importance goes into which briefing, the rules of the way of working say, and for a further meeting its Covers. One line each: importance, type, Area, the finding's title as a link. `none` when nothing waits. Example: `1. [high] Requirement · Lending · Members want to renew online`.

## Decided for you
What an agent added to the spec because it follows from what is decided, each with the place and the finding. A finding reverses it. `none` when nothing was added. Example: `Lending period: renewals follow the same period [finding]`.

## Went to work without a decision
The findings that went from `ready` into the plan since the last briefing, each with its story. Anyone can object. `none` when there are none. Example: `Bug · Lending · Due date shown in the wrong time zone → housekeeping, ready`.

## Decided yesterday
The decisions recorded since the last briefing, and what became of each. `none` when there were none. Example: `Idea "reading lists": accepted, now a feature in the Area catalog`.
