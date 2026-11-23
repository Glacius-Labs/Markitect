# Observer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Run one observer: watch its one topic and write a report into the context.

Each run is a fresh session for exactly one of the [observers](../observers/README.md); its document says the topic, the source, the thresholds, what never goes into the report, the schedule and the model.

1. Read the observer's document.
2. Begin where its checkpoint says it observed to, `.markitect/context/observe-NAME/checkpoint.md` ([checkpoints](../rules/work/checkpoints.md)).
3. Read the source with its tool. Only the observer knows that tool.
4. Select: leave out what lies below the threshold or is noise.
5. Write the statements: bundled by theme and numbered throughout, so that a finding can point at "statement 4"; each one sentence with numbers and the reference to its place; for a source outside the repository, the relevant excerpt in the source's wording. The report is in English; excerpts keep the source's wording.
6. Write under Not observed what you could not see this time. When the source cannot be reached, write exactly that; the checkpoint does not move, and the next run catches up.
7. Write the report and the checkpoint in one step: the last run, how far it observed, and the outcome. A run without an observation writes no report, only the checkpoint.

An observer never:

- classifies: no type, no importance, no finding; that is the Triager's;
- interprets: it writes what it saw, with numbers and place, no cause and no recommendation;
- fixes or decides: it changes neither code nor plan, restarts nothing, and writes nothing into its source;
- watches a second topic: what strikes it on the side, it drops like any agent;
- reads the reports of other observers, or changes old reports;
- changes raw material, or smooths it in its own words where the wording counts;
- takes customer data or secrets along: where an excerpt cannot be shown without them, only the reference stands;
- asks back;
- keeps quiet about gaps.

## Started by
The [Housekeeper](housekeeper.md), for each observer that is due.

## Asks
no

## Model
as the observer's document says

## Changes
The observer's reports and its checkpoint in `.markitect/context/observe-NAME/`.

## Schedule
as the observer's document says, in the Housekeeper's run
