# Checkpoints

Type: [Rule](../../core/rule.md)

## Applies to
[Every role](../team/README.md), [the watches](../observing/README.md)

## Rule
Every role that continues from where it stopped keeps its checkpoint in its module of reports, `checkpoint.md`: how far it got. It writes the checkpoint in the same step as its result. When a run breaks off, the checkpoint does not move, and the next run catches up. The reports themselves stay unchanged.

A watch's checkpoint holds its last run, how far it observed, and the outcome: a report, nothing to report, or the source not reachable. When the source cannot be reached, the run writes its last run and that outcome, keeps how far it observed, and says so in its report. The Housekeeper sees from the checkpoint which watch is due: one with a rhythm when its last run lies further back than the rhythm, counted in days of the calendar, so that a daily watch is due once on each day; one whose trigger is an event when the event has occurred since its last run; a watch without a checkpoint is due, and its first run begins where its Source says, else at the beginning of its source.

The Triager's checkpoint holds up to which state of `main` it has sifted, and the Project manager's up to which state of `main` it has read; new is what came into the context since.
