# Briefing

What reaches the humans: short, ordered by importance, one line per finding with a link to it. Nobody has to read more than the line. The Briefer writes one in time for each recurring [meeting](meeting.md), into `.markitect/briefing/MEETING/`, the folder named like the meeting's document; the daily and the weekly briefing are the first two, such as `daily/2026-11-23.md` and `weekly/2026-W48.md`. Take it only for that. Do not take it to decide: people decide in conversation, and the Recorder carries out what they decided.

## Needs a decision
The findings in `pending` that wait for people, critical first, then by importance; which importance goes into which briefing, the meeting's Covers and the rules of the way of working say. One line each: importance, type, Area, the finding's title as a link. `none` when nothing waits. Example: `1. [high] Requirement · Lending · Members want to renew online`.

## Decided for you
What an agent added to the spec because it follows from what is decided, each with the place and the finding. A finding reverses it. `none` when nothing was added. Example: `Lending period: renewals follow the same period [finding]`.

## Went to work without a decision
The findings that went from `ready` into the plan since the last briefing, each with its story. Anyone can object. `none` when there are none. Example: `Bug · Lending · Due date shown in the wrong time zone → housekeeping, ready`.

## Decided yesterday
The decisions recorded since the last briefing, and what became of each. `none` when there were none. Example: `Idea "reading lists": accepted, now a feature in the Area catalog`.
