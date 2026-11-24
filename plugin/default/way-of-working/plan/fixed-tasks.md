# Fixed tasks

Type: [Rule](../../core/rule.md)

## Applies to
[Story](story.md), [the Project manager](../team/project-manager.md)

## Rule
Some stories always have the same tasks. They are created with these tasks and are `ready` at once, in every module: their model is already clear, so no human plans them before and none checks them before `main`.

| For | Fixed tasks | Done when |
|---|---|---|
| A bug with a clear model | reproduce; find the cause; fix; a test for it | the deviation is gone |
| An addition that follows from what is decided | check whether the artifacts follow it already; build what is missing; a test for it | the artifacts follow the addition |
| A smell analysis | collect evidence; list possible causes, "false alarm" included; check each; determine place and reach: what depends on it, the effort, the risk of fixing and of leaving it; a report with a recommendation | the report lies in the context |
| Maintenance | update or repair; check that everything still runs | it runs again or is current |

A story with fixed tasks has nothing to accept: it is `done` when its condition holds. Its Plan is one sentence naming what the tasks work on. Under Realizes it names the places its files realize, as every story does ([one trace](../review/one-trace.md)): for a bug, the document that already says what is right, `unchanged`, as the Project manager writes it; for an addition, the document with the sentence that was added; for maintenance, the documents its files realize; a smell analysis changes no file and names none. When a fixed task turns out not to suffice, that is a finding of type Process.
