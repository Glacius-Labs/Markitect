# Feature

An idea we want to build, with its own target picture, below one Area of the plan. It has an end: its acceptance. Ownership lies with the feature; its stories are what gets carried out. Take it for anything one person can take and work out on their own. Do not take it for work without a feature: bugs, maintenance and smell analyses are [stories](story.md) of the Area's standing feature `housekeeping`. Its document lies in `.markitect/plan/`, in a folder of its own below its Area's folder, beside its stories. What lasts of its target picture reaches the spec through its stories' Target changes.

## State
On the first line one of `open`, `in progress`, `acceptance`, `done`: `open` until someone claims it, `in progress` once claimed, `acceptance` when its stories are all `done` and it waits to be accepted as a whole, `done` when it is accepted. Example: `in progress`.

## Owner
The person who owns it, as a link, or `none` while it is open. Claiming the feature is writing yourself here on `main` before you start. Example: `[Ada](../../../../docs/spec/factory/people/ada.md)`.

## Origin
Where it came from: a link to the finding in `.markitect/findings/closed/`, or to the report in the context; for the standing feature `housekeeping`, a link to its Area's document. Example: `[Members want to renew online](../../../findings/closed/renew-online.md)`.

## Target picture
What should hold when the feature is done, like an interface or a responsibility, not a specification; and what belongs into the first version for a milestone. It fits the spec. Example: `A member renews a loan without a visit. First version: one renewal per loan.`

## Stories
The stories in their order, as links, each with what it depends on. Example: `1. [Renew a loan](renew-a-loan.md), depends on none`.

## Acceptance
The task for accepting the feature as a whole, as a check box, and who does it: whoever accepts, as the role documents say; `none` for the standing feature `housekeeping`, which has no end. Example: `- [ ] Polish the interface of this feature and accept it`.
