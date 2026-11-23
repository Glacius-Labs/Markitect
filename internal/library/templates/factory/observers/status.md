# status

Type: [Observer](../../../packages/factory/observer.md)

## Observes
What moved between the spec and the files since it was last confirmed: what `markitect status` lists.

## Source
The output of `markitect status` on `main`, run locally. Its checkpoint holds the last list; observed to: that list. The first run records the list and reports nothing, so a project whose files are not mapped yet needs no switch.

## Worth a statement
Every entry that is new against the last list: a document changed or suspect, a file unexplained, a piece of evidence that changed. What stood on the last list already is no statement.

## Never in the report
Customer data and secrets; causes and recommendations.

## Schedule
daily

## Model
Claude: Sonnet, effort high
Codex: Luna, effort high
