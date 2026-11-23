# Finish

Type: [Step](../../../../packages/company/step.md)

## Description
The planning is closed, in this order. A fresh Reviewer checks the plan, and the changed documents of the spec, against the target of its own level ([Reviewer](../../agents/review.md)). The report on the course of the planning goes into the context. A planned story is set to `ready` and its claim released; after a feature was planned, its new stories stay `draft` until each is planned, and the feature keeps its Owner and `in progress`. The plan goes to `main`, with the findings the planning closed, and without the changed documents of the spec. Then each story whose Target changes name a change to the spec gets its branch, named as [names in the work](../../rules/work/names.md) says: cut from `main`, or, when it exists for any reason, after taking the current `main` into it. The changed documents that story carries are committed there and replace what an earlier planning put there; a change the story no longer carries is removed, and the Builder treats code already on the branch as a start it may keep or drop within the spec ([spec change](../../rules/building/spec-change-lands-with-realization.md)). The person goes back to Norman, and the chat can be archived.

## Performed by
[finish-planning](../../agents/finish-planning.md), in the planning session; a fresh [Reviewer](../../agents/review.md) checks.
