# Milestones go to main

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The plan](README.md), [the dashboards](../../dashboards/README.md)

## Rule
A milestone is a step after which something on `main` is different that interests a human. Whoever reaches it writes the new state to `main` in the same step; this table is the one place that says who sets which state:

| Milestone | What changes | Who writes it |
|---|---|---|
| Feature claimed | its Owner; feature `open` to `in progress` | the person's session |
| Plan of a story secured | story `draft` to `ready`, claim released | the planning session, with [finish-planning](../../agents/finish-planning.md) |
| Work on a story begun | story `ready` to `in progress`, claimed | the Builder or the Analyst |
| Story landed on `main` | story to `acceptance`, or `done` when there is nothing to accept; claim released | the Builder, after agreeing with the Reviewer |
| Smell analysis finished | story to `done`, claim released | the Analyst |
| Build aborted | story back to `draft`, claim released; a report in the context, linked from the story | the Builder |
| Story accepted | story to `done`, claim released | the session of whoever accepts |
| Last story of a feature done | feature to `acceptance` | the session that sets that story `done` |
| Feature accepted | feature to `done` | the session of whoever accepts |
| Housekeeper's run finished | new findings, recorded decisions, briefings | the Housekeeper |
