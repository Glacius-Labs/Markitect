# Fixed tasks

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [the Recorder](../../agents/record.md)

## Rule
Some stories always have the same tasks. They are created with these tasks and are `ready` at once, in every Area and every class: their spec is already clear, so no human plans them before and none checks them before `main`.

| For | Fixed tasks | Done when |
|---|---|---|
| A bug with a clear spec | reproduce; find the cause; fix; a test for it | the deviation is gone |
| A smell analysis | collect evidence; list possible causes, "false alarm" included; check each; determine place and reach: what depends on it, the effort, the risk of fixing and of leaving it; a report with a recommendation | the report lies in the context |
| Maintenance | update or repair; check that everything still runs | it runs again or is current |

A story with fixed tasks has nothing to accept: it is `done` when its condition holds. Its Target picture is that condition, and its Plan one sentence naming what the tasks work on. Its Target changes name the places its files realize, as for every story ([one trace](../building/one-trace.md)): for a bug, the document that already says what is right, `unchanged`, as the Recorder writes it; for maintenance, the documents its files realize; a smell analysis changes no file and names none. When a fixed task turns out not to suffice, that is a finding of type Factory.
