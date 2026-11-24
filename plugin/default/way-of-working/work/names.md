# Names in the work

Type: [Rule](../../core/rule.md)

## Applies to
[The work](README.md), [every role](../team/README.md)

## Rule
A name made from a title is the title in lowercase English words joined by hyphens. A finding is a document in the module `findings`, named after its title. In the module `plan`, each module of the model has a module of the same path; there lie its features and its stories side by side, each a document named after its title. A drop in the module `drops` is named by date, time and subject, `YYYY-MM-DD-HHMM-SUBJECT.md`; a [report](../observing/report.md) is named the same way, in the module of its role below `reports`, its subject being the file name of the story or feature it worked on, or, for a session or run without one, what it did, such as `daily-run`, `triage` or `start-of-work`. A briefing lies in `briefings/daily`, `briefings/weekly` or a further meeting's module, named by its date, `YYYY-MM-DD.md`, or for the weekly briefing and a weekly meeting by its week, `YYYY-Www.md`; a briefing written on request lies in `briefings/on-request`, named by date and time, `YYYY-MM-DD-HHMM.md`. A story's branch is named like the story's path below `plan` without `.md`; it is cut from `main` when the first role starts work on it; building and polishing in accepting work on it, each in a working folder of that branch, and whoever starts on it when it exists first takes the current `main` into it.

## Example
The story "Renew a loan" of the feature "Renew online" in the module `lending` lies at `plan/lending/renew-a-loan.md`, beside its feature `plan/lending/renew-online.md`, and its branch is `lending/renew-a-loan`.
