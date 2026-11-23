# Finish planning

Type: [Agent](../../../packages/factory/agent.md)

## Task
Close a planning: have the plan checked, report on it, set the state, and bring the plan to `main`.

It runs at the end of a planning session, in that session, as step [finish](../processes/planning/finish.md) of planning.

1. Have a fresh [Reviewer](review.md) check the plan against the target of its own level. When it does not fit, go back to the planning.
2. Lay the report on the course of the planning into `.markitect/context/architect/` ([one per session](../rules/work/one-per-session.md)), with any coupling the cut showed.
3. Set the planned story to `ready` and release its claim ([claiming](../rules/plan/claiming.md)).
4. Put every change to the standing spec that a story's Target changes name, what the spec says for the whole or for an Area, onto the branch of that story, never onto `main`; when it concerns several stories, each carries the part it realizes ([spec change](../rules/building/spec-change-lands-with-realization.md)).
5. Bring the plan to `main`.
6. Rebuild the dashboards for the person.
7. Tell the person that the plan is secured and the chat can be archived, and send them back to [Norman](norman.md).

## Started by
The planning session of the [Architect](architect.md).

## Asks
no

## Model
as the session

## Changes
The story's State and Claimed by.
The plan in `.markitect/plan/`, on `main`.
The branches of the stories that carry spec changes, with those changes.
The planning's report in `.markitect/context/architect/`.
