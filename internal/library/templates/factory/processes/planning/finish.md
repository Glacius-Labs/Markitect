# Finish

Type: [Step](../../../../packages/company/step.md)

## Description
The planning is closed, in this order. A fresh Reviewer checks the plan against the target of its own level ([Reviewer](../../agents/review.md)). The report on the course of the planning goes into the context. A planned story is set to `ready` and its claim released; after a feature was planned, its new stories stay `draft` until each is planned, and the feature keeps its Owner and `in progress`. The plan goes to `main`, with the findings the planning closed. Then each story whose Target changes name a change to the standing spec gets its branch, cut from `main` and named as [names in the work](../../rules/work/names.md) says, and the change goes onto it; when the branch exists from an aborted build, it first takes in the current `main`, and the new change replaces the old one there, which is removed when the story no longer changes the standing spec, while the Builder treats the code on it as a start it may keep or drop within the spec ([spec change](../../rules/building/spec-change-lands-with-realization.md)). The person goes back to Norman, and the chat can be archived.

## Performed by
[finish-planning](../../agents/finish-planning.md), in the planning session; a fresh [Reviewer](../../agents/review.md) checks.
