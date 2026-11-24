# The hotspots

Type: [Watch](watch.md)

State where development jams in the structure. The last 14 days against the 14 before: for each module of the model, how many stories touched the files that realize it; files touched by three or more stories; files changed again within 3 days of their merge; parallel changes, the same file changed on both sides of a merge, as an approximation of merge conflicts, which needs real merge commits. Generated files and generated lists do not count. The thresholds are starting values.

## Participants
[Observer](../team/observer.md)

## Trigger
Weekly.

## Steps
[Observe](observe.md)

## Result
A report with the statements, or only the checkpoint when there was nothing to state.

## Source
The Git history of `main`, read locally, and the modules of the model. Observed to: the commit up to which it observed.

## Never in the report
Customer data and secrets; causes and recommendations.
