# Story

A piece of work that is complete in itself once its predecessors are built: afterwards something can be used, and the next story adds to it. A story is what gets carried out, by a Builder with Reviewers, by an Analyst, or by a person. Every piece of work in the plan is a story, also a bug, a maintenance job or a smell analysis ([fixed tasks](fixed-tasks.md)). What should hold when it is done stands in the model, not here: the story names the places it realizes. A story may belong to a [feature](feature.md), which then names it; it needs none. An idea nobody has decided on is no story but a [finding](../triage/finding.md). Cleaning up and changing behavior are never mixed in one story ([cleaning up stands apart](../planning/cleanup-apart.md)).

## State
Type: `draft`, `ready`, `in progress`, `acceptance`, `done`, `cancelled`

One of six states, named neutrally, since a story may also be an analysis:

| State | Means | Who acts |
|---|---|---|
| `draft` | created, not planned yet or being planned, or set back by an aborted build | the Architect with a person; a story with fixed tasks is created `ready` and comes here only after an abort |
| `ready` | no human has anything left to conceive | waits to be started |
| `in progress` | someone builds it, analyses it or does it | Builder and Reviewer, an Analyst, or a person |
| `acceptance` | the result is on `main` and waits to be accepted | the person who accepts |
| `done` | built, and accepted where there was something to accept | nobody |
| `cancelled` | given up: nobody wants it any more ([giving up a story](giving-up-a-story.md)) | nobody |

A story is open while it is neither `done` nor `cancelled`. A story with nothing to accept goes from `in progress` straight to `done`. An aborted build sets it back to `draft` ([abort](../building/abort.md)). It starts only when it is `ready` and all its predecessors have landed, in `acceptance` or `done`; a `cancelled` predecessor has not ([when a story starts](when-a-story-starts.md)). Every activity that works on a story says which state it sets. Example: `ready`.

## Priority
Type: `critical`, `high`, `normal`, `low`

How urgent it is ([priorities](priorities.md)). The order within a feature stands in the feature. Example: `normal`.

## Depends on
Type: [Story](story.md)

The stories it waits for, one per line, or `none`. Whoever creates the story sets it, always ([dependencies](dependencies.md)). Example: `[Show due dates](show-due-dates.md)`.

## Claimed by (optional)
Type: [Actor](../../core/actor.md)

Who works on it now: a person, or the agent that fills the role at work. Left out while nobody does. Claiming the story is writing yourself here on `main` before you start ([claiming](claiming.md)). Example: `[Builder agent](../team/builder-agent.md)`.

## Origin
Type: [Finding](../triage/finding.md), [Report](../observing/report.md), [Feature](feature.md)

Where it came from: the finding or the report, or its feature for a story cut from one. A smell story lists every finding that references it, one link per line. Example: `[Renew online](renew-online.md)`.

## Realizes
Every place in the model that this story realizes, each as a link with one sentence on how the story changes it, or `unchanged` when it changes nothing there. These documents say what should hold when the story is done; the story has no target beside them. When no document of the model covers its files, something is missing in the model, and its planning adds it first. `none` only for a story that changes no file, such as a smell analysis. A story cut in a feature's planning holds the places as far as they are known; its own planning completes them. The model says so already when the story is created; until the story has landed, the documents it changes are ahead of the artifacts ([the model runs ahead](the-model-runs-ahead.md)). When the story only adds to a document that other stories realize too, the sentence says which part is its own. Example: `[Lending period](../../lending/period.md): a renewal extends it once.`

## Plan
How it will be realized, coarse, in concepts and building blocks: no file names, no code; `not planned yet` while the story waits in `draft` for its planning. Example: `The loan gains a renewal; the member's page offers it.`

## Tasks
The steps in which it is built, in order, as a plain list; `not planned yet` while the story waits in `draft`. A story with fixed tasks has exactly those. How far the work is, the State alone says. Example: `1. Add the renewal to the loan. 2. Offer it on the member's page.`

## Acceptance (optional)
What the person who accepts looks at, tries and accepts. Left out when there is nothing to accept; then the story is `done` once it has landed. Example: `Renewing a loan on the member's page.`

## Result (optional)
What changed, written by the Builder, or, after an abort, a link to the report; for a story given up, that it was given up, why, and a link to the decision. Nothing else: the Reviewer's confirmation does not stand here ([one trace](../review/one-trace.md)). Example: `Loans can be renewed once; the member's page offers it.`
