# The status

Type: [Watch](watch.md)

State what moved between the model and the files since the last run: a document changed, a file that no module names, files that changed without their module. Every entry that is new against the last run is a statement; what stood there already is none. The first run records what it finds and reports nothing.

## Participants
[Observer](../team/observer.md)

## Trigger
Daily.

## Steps
[Observe](observe.md)

## Result
A report with the statements, or only the checkpoint when there was nothing to state.

## Source
What `markitect diff` shows between `main` at the last run and `main` now. Observed to: the commit up to which it observed.

## Never in the report
Customer data and secrets; causes and recommendations.
