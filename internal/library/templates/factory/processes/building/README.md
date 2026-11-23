# Building

Type: [Process](../../../../packages/company/process.md)

## Trigger
A story is `ready` and all its predecessors are `done` ([when a story starts](../../rules/plan/when-a-story-starts.md)). [Norman](../../agents/norman.md) starts a Builder when a person comes back from planning, or the [Housekeeper](../../agents/housekeeper.md) does in its daily run.

## Steps
1. [Claim](claim.md)
2. [Build](build.md)
3. [Review round](review-round.md), repeated
4. [Confirm](confirm.md)
5. [Land](land.md)

## Result
The story is on `main` together with the spec changes it carries and the Reviewer's `confirm`, and stands in `acceptance`, or `done` when there is nothing to accept. No human checked it before `main`. Or the build was [aborted](../../rules/building/abort.md): the story is back in `draft` with a report in the context.

## Responsible
The [Builder](../../agents/build.md).

## Involved
A fresh [Reviewer](../../agents/review.md) per round. The rules: [abort](../../rules/building/abort.md), [tests](../../rules/building/tests.md), [report a smell, do not build around it](../../rules/building/no-workarounds.md).
