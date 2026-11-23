# Names in the work

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The work folder](README.md), [the agents](../../agents/README.md)

## Rule
A name made from a title is the title in lowercase English words joined by hyphens. A finding is named after its title, in its folder of `.markitect/findings/`. In `.markitect/plan/`, an Area's folder is named like the Area's document in the spec without `.md`; a feature's folder is named after its title and holds the feature as `feature.md` and each story as a file named after the story's title; the standing feature's folder is `housekeeping/`. A drop in `.markitect/context/capture/` is named by date, time and subject, `YYYY-MM-DD-HHMM-SUBJECT.md`; a report is named as its [type](../../../../packages/factory/report.md) says. A briefing lies in its meeting's folder, named by the meeting's date, `YYYY-MM-DD.md`, or for a weekly meeting by its week, `YYYY-Www.md`. A story's branch is named like the story's path below `.markitect/plan/` without `.md`.

## Example
The story "Renew a loan" of the feature "Renew online" in the Area whose document is `lending.md` lies at `.markitect/plan/lending/renew-online/renew-a-loan.md`, and its branch is `lending/renew-online/renew-a-loan`.
