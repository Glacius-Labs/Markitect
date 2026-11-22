# Finish planning

Type: [Agent](../../../packages/factory/agent.md)

## Task
Close a planning: have the plan checked, report on it, set the state, and bring the plan to `main`.

It runs at the end of a planning session, in that session, as step [finish](../processes/planning/finish.md) of planning.

1. Have a fresh [Reviewer](review.md) check the plan against the target picture of its own level. When it does not fit, go back to the planning.
2. Lay the report on the course of the planning into `.markitect/context/architect/` ([one per session](../rules/work/one-per-session.md)), with any coupling the cut showed.
3. Set the planned story to `ready`.
4. Put every change to the standing spec, what it says for the whole or for an Area, onto the branch of the story that realizes it, never onto `main`; when it concerns several stories, each carries the part it realizes ([spec change](../rules/building/spec-change-lands-with-realization.md)).
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
The story's State.
The plan in `.markitect/plan/`, on `main`.
The branches of the stories that carry spec changes.
The planning's report in `.markitect/context/architect/`.
