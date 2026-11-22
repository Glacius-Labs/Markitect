# Milestones go to main

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The plan](README.md), [the dashboards](../../dashboards/README.md)

## Rule
A milestone is a step after which something on `main` is different that interests a human. Whoever reaches it writes the new state to `main` in the same step:

| Milestone | What changes | Who writes it |
|---|---|---|
| Feature claimed | its Owner | the person's session |
| Plan of a story secured | story `draft` to `ready` | the planning session |
| Work on a story begun | story `ready` to `in progress` | the Builder or the Analyst |
| Story landed on `main` | story to `acceptance`, or `done` when there is nothing to accept | the Builder, after agreeing with the Reviewer |
| Build aborted | a report in the context, linked from the story | the Builder |
| Story or feature accepted | story to `done`; feature accepted | the session of whoever accepts |
| Housekeeper's run finished | new findings, recorded decisions, briefing | the Housekeeper |
