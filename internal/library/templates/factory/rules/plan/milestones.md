# Milestones go to main

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The plan](README.md), [the dashboards](../../dashboards/README.md)

## Rule
A milestone is a step after which something on `main` is different that interests a human. Whoever reaches it writes the new state to `main` in the same step; this table is the one place that says who sets which state:

| Milestone | What changes | Who writes it |
|---|---|---|
| Feature claimed | its Owner; feature `open` to `in progress` | the person's session |
| Story claimed for planning | its Claimed by | the Architect, for its person |
| Plan of a story secured | story `draft` to `ready`, claim released | the planning session, with [finish-planning](../../agents/finish-planning.md) |
| Planning ended without finish-planning | claim released, state unchanged | the Architect |
| Work on a story begun | story `ready` to `in progress`, claimed | the Builder or the Analyst |
| Story landed on `main` | story to `acceptance`, or `done` when there is nothing to accept; claim released | the Builder, after agreeing with the Reviewer |
| Smell analysis finished | story to `done`, claim released | the Analyst |
| Build aborted | story back to `draft`, claim released; a report in the context, linked from the story | the Builder |
| Story accepted | story to `done`, claim released | the session of whoever accepts |
| Acceptance failed within its rounds | story back to `draft`, claim released; a report linked from the story | [accept](../../agents/accept.md) |
| Commitment written | a story `in progress`, claimed by the person who does it | the Recorder ([commitments](commitments.md)) |
| Commitment done | story to `done`, claim released | the session the person says it in |
| Observer's story taken | claimed in `draft` while it is planned; `ready` once planned, then `in progress`; `done` after the merge, claim released; back to `draft` when the observer is dropped | [create-observer](../../agents/create-observer.md), which plans and realizes it in one session |
| Last story of a feature done | feature to `acceptance`; never the standing feature `housekeeping`, which has no end | the session that sets that story `done`: the Builder, the Analyst or whoever accepts |
| Feature accepted | feature to `done` | the session of whoever accepts |
| Housekeeper's run finished | new findings, recorded decisions, briefings | the Housekeeper |
