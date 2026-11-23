# Land

Type: [Step](../../../../packages/company/step.md)

## Description
Right before the merge, the Builder fetches `main`; when the story was given up meanwhile, it stops as for a lost claim ([claiming](../../rules/plan/claiming.md)). Otherwise it writes what changed into the story's Result and brings the story to `main` as one merge, whose commit names the story; spec changes on the branch go along. It sets the story to `acceptance`, or to `done` when there is nothing to accept, releases its claim ([claiming](../../rules/plan/claiming.md)), sets what the [milestones](../../rules/plan/milestones.md) name for the feature, and lays its report into the context. No human checks before `main`.

## Performed by
The [Builder](../../agents/build.md).
