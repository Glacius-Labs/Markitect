# Review

Type: [Step](../../../../packages/company/step.md)

## Description
The result of the polishing goes through a Reviewer and automatically to `main`: the result, not every single detail. Accepting has rounds like building: a fresh Reviewer per round, at most five rounds, and from the third on only weightier comments count. When the story is not accepted after five rounds, the build has failed: it flows back as a finding, with state and report as after an [abort](../../rules/building/abort.md).

## Performed by
A fresh [Reviewer](../../agents/review.md) per round, started by [accept](../../agents/accept.md).
