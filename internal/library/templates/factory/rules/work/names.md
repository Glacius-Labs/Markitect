# Names in the work

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The work folder](README.md), [the agents](../../agents/README.md)

## Rule
A name made from a title is the title in lowercase English words joined by hyphens. A finding is named after its title, in its folder of `.markitect/findings/`. In `.markitect/plan/`, an Area's folder is named like the Area's document in the spec without `.md`; a feature's folder is named after its title and holds the feature as `feature.md` and each story as a file named after the story's title; the standing feature's folder is `housekeeping/`. A drop in `.markitect/context/capture/` is named by date, time and subject, `YYYY-MM-DD-HHMM-SUBJECT.md`; a report is named as its [type](../../../../packages/factory/report.md) says, its subject being its item's name: a story's file name, a feature's folder name. A briefing lies in `daily/`, `weekly/` or a further meeting's folder, named by its date, `YYYY-MM-DD.md`, or for the weekly briefing and a weekly meeting by its week, `YYYY-Www.md`. A story's branch is named like the story's path below `.markitect/plan/` without `.md`; it is cut from `main` when the planning first changes the spec for the story, or else when the first agent starts work on it; planning, building and polishing in accepting all work on it, each in a working folder of that branch, and whoever starts on it when it exists first takes the current `main` into it.

## Example
The story "Renew a loan" of the feature "Renew online" in the Area whose document is `lending.md` lies at `.markitect/plan/lending/renew-online/renew-a-loan.md`, and its branch is `lending/renew-online/renew-a-loan`.
