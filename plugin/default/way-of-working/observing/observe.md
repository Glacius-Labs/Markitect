# Observe

Type: [Activity](../../core/activity.md)

## Trigger
A watch is due: its last run lies further back than the rhythm in its Trigger says, or it has never run ([checkpoints](../work/checkpoints.md)). The [Housekeeper](../team/housekeeper.md) starts one fresh Observer per watch that is due, in the [daily run](../day/daily-run.md).

## Steps
1. Read the watch's document.
2. Begin where the watch's checkpoint says it observed to, or, on a first run, where the [checkpoints](../work/checkpoints.md) rule says.
3. Read the source with its tool. Only the watch knows that tool.
4. Select: leave out what lies below the threshold or is noise.
5. Write the statements: bundled by theme and numbered throughout, so that a finding can point at "statement 4"; each one sentence with numbers and the reference to its place; for a source outside the repository, the relevant excerpt in the source's wording. The report is in English; excerpts keep the source's wording.
6. Write under Not observed what could not be seen this time. When the source cannot be reached, write exactly that, and write the checkpoint as the [checkpoints](../work/checkpoints.md) rule says for that case; the next run catches up.
7. Write the report and the checkpoint in one step, to `main`, or, for a trial run while a story is built, on the story's branch ([add a watch](add-a-watch.md)): the last run, how far it observed, and the outcome. A run without an observation writes no report, only the checkpoint.

## Result
The [report](report.md) of the run lies in the context, with its statements and what was not observed, and the watch's checkpoint says how far it observed. A run that observed nothing moved only the checkpoint.

## Participants
The [Observer](../team/observer.md).
