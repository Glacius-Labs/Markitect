# Finish planning

Type: [Agent](../../../packages/factory/agent.md)

## Task
Close a planning: have the plan checked, report on it, set the states, and bring the plan to `main`.

It runs at the end of a planning session, in that session, and does what the step [finish](../processes/planning/finish.md) of planning says, in its order:

1. Have a fresh [Reviewer](review.md) check the plan. When it does not fit, go back to the planning.
2. Lay the report on the course of the planning into `.markitect/context/architect/` ([one per session](../rules/work/one-per-session.md)), with any coupling the cut showed.
3. Set the states the step names, and release the claim of a planned story ([claiming](../rules/plan/claiming.md)).
4. Bring the plan to `main`.
5. Cut the branch of each story whose Target changes name a change to the standing spec, and put that change onto it, never onto `main` ([spec change](../rules/building/spec-change-lands-with-realization.md)).
6. Tell the person that the plan is secured and the chat can be archived, and send them back to [Norman](norman.md).

## Started by
The planning session of the [Architect](architect.md).

## Asks
no

## Model
as the session

## Changes
The State and Claimed by of the planned story.
The plan in `.markitect/plan/`, on `main`.
The branches of the stories that carry spec changes, with those changes.
The planning's report in `.markitect/context/architect/`.
