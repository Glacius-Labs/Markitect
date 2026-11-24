# Feature

An idea we want to build that is larger than one story. A feature plans the work: it names the stories that realize the idea, in their order, says who took it, and ends when it is accepted as a whole. What should hold stands in the model, not here. A feature is anything one person can take and work out on their own. A [story](story.md) needs no feature: a bug or a maintenance job is simply a story in the plan of its module.

## State
Type: `open`, `in progress`, `acceptance`, `done`

`open` until someone claims it, `in progress` once claimed, `acceptance` when none of its stories is open any more, each `done` or `cancelled`, no finding is attached to it any more, and it waits to be accepted as a whole, `done` when it is accepted. A feature in `acceptance` goes back to `in progress` when accepting it brings a further story. Every activity that works on a feature says which state it sets. Example: `in progress`.

## Owner (optional)
Type: [Person](../../core/person.md)

The person who took it. Left out while the feature is open. Claiming the feature is writing yourself here on `main` before you start ([claiming](claiming.md)). Example: `[Ada](../team/ada.md)`.

## Origin
Type: [Finding](../triage/finding.md), [Report](../observing/report.md)

Where it came from: the closed finding, or the report. Example: `[Members want to renew online](../../findings/renew-online.md)`.

## Stories
Type: [Story](story.md)

Its stories in their order, one per line, with what each depends on and which of them make the first version; `not planned yet` while the feature is `open` and nobody has planned it. Example: `1. [Renew a loan](renew-a-loan.md), depends on none, first version`.

## Acceptance
What the person who accepts looks at when all its stories are done, and who that is; `not planned yet` while the feature is `open`. Example: `Renewing from the reminder mail to the new due date; the Admin.`
