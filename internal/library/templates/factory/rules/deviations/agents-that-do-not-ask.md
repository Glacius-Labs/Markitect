# Agents that do not ask

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
Every agent whose Asks is `no`: [Builder](../../agents/build.md), [Reviewer](../../agents/review.md), [Analyst](../../agents/analyze.md), [Housekeeper](../../agents/housekeeper.md), [the observers](../../agents/observe.md), [Triager](../../agents/triage.md), [Recorder](../../agents/record.md), [Briefer](../../agents/brief.md)

## Rule
Agents that do not ask go through the same course of a deviation. They write the reason down themselves. At `low` and `high` they carry on. At `critical` nobody is there to say yes: they stop and lay a report into the context, as when a build is aborted.
