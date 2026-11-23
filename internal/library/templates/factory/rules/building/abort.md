# Abort

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The Builder](../../agents/build.md), [building](../../processes/building/README.md), [accept](../../agents/accept.md)

## Rule
The Builder stops when the spec turns out to be the problem, when the structure resists, when a fix needs a change to the spec, or when Builder and Reviewer do not agree within the rounds a [review round](../../processes/building/review-round.md) allows. It lays a report with the reason into the context, sets the story back to `draft`, releases its claim ([claiming](../plan/claiming.md)) and links the report in the story's Result. So nobody restarts it on their own: a person reopens it with the Architect, a story with fixed tasks too. The next day the report stands as a finding. A risky bug needs no rule of its own: when it resists, the Builder aborts. When a story is not accepted within the rounds of accepting, or the person who accepts ends the acceptance as failed ([review](../../processes/accepting/review.md)), the build has failed too: accept does the same.

## Example
Renewing a loan would need the catalog to count copies differently than the spec says. The Builder aborts with a report instead of bending the catalog.
