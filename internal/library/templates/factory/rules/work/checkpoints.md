# Checkpoints

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The agents](../../agents/README.md), [the observers](../../observers/README.md)

## Rule
Every agent that continues from where it stopped keeps its checkpoint in its context folder, `checkpoint.md`: how far it got. It writes the checkpoint in the same step as its result. When a run breaks off, the checkpoint does not move, and the next run catches up. The reports themselves stay unchanged.

An observer's checkpoint holds its last run, how far it observed, and the outcome: a report, nothing to report, or the source not reachable. When the source cannot be reached, the run writes its last run and that outcome, keeps how far it observed, and says so in its report. The Housekeeper sees from the checkpoint which observer is due; an observer without a checkpoint is due, and its first run begins where its Source says, else at the beginning of its source.

The Triager's checkpoint holds up to which state of `main` it has sifted, and the Recorder's up to which state of `main` it has read; new is what came into the context since.
