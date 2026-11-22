# Land

Type: [Step](../../../../packages/company/step.md)

## Description
The Builder writes what changed into the story's Result and brings the story to `main` as one merge, whose commit names the story; spec changes on the branch go along. It sets the story to `acceptance`, or to `done` when there is nothing to accept, and lays its report into the context. No human checks before `main`.

## Performed by
The [Builder](../../agents/build.md).
