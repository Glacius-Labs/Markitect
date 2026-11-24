# Abort

Type: [Rule](../../core/rule.md)

## Applies to
[The Builder](../team/builder.md), [building](building.md)

## Rule
The Builder stops when the model turns out to be the problem, when the structure resists, when a fix needs a change to the model, or when Builder and Reviewer do not agree within the rounds a [review](../review/review.md) allows. It lays a report with the reason into the context, sets the story back to `draft`, releases its claim ([claiming](../plan/claiming.md)) and links the report in the story's Result. So nobody restarts it on their own: a person reopens it with the Architect, a story with fixed tasks too. The report is reason enough to reopen it, the same day; the triage makes a finding from it only when the story still lies untouched in `draft` at the next run. It stops the same way before a task that would be a [critical](../deviations/critical.md) act, such as something in production: stopping there is no deviation, and its report says which task a person has to do. A risky bug needs no rule of its own: when it resists, the Builder aborts. An acceptance never ends in an abort: what a person does not like there becomes a further story ([accepting](../accepting/accepting.md)).

## Example
Renewing a loan would need the catalog to count copies differently than the model says. The Builder aborts with a report instead of bending the catalog.
