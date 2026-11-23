# Finish

Type: [Step](../../../../packages/company/step.md)

## Description
The planning is closed, in this order. A fresh Reviewer checks the plan against the target of its own level ([Reviewer](../../agents/review.md)). The report on the course of the planning goes into the context. A planned story is set to `ready` and its claim released; after a feature was planned, its new stories stay `draft` until each is planned, and the feature keeps its Owner and `in progress`. The plan goes to `main`. Then each story whose Target changes name a change to the standing spec gets its branch, cut from `main` and named as [names in the work](../../rules/work/names.md) says, and the change goes onto it ([spec change](../../rules/building/spec-change-lands-with-realization.md)). The person goes back to Norman, and the chat can be archived.

## Performed by
[finish-planning](../../agents/finish-planning.md), in the planning session; a fresh [Reviewer](../../agents/review.md) checks.
