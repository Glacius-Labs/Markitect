# Checkpoints

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The agents](../../agents/README.md), [the observers](../../observers/README.md)

## Rule
Every agent that continues from where it stopped keeps its checkpoint in its context folder, `checkpoint.md`: how far it got. It writes the checkpoint in the same step as its result. When a run breaks off, the checkpoint does not move, and the next run catches up. An observer's checkpoint holds its last run, how far it observed, and the outcome: a report, nothing to report, or the source not reachable; the Housekeeper sees from it which observer is due. The Triager's holds up to which state of `main` it has sifted; new is what came into the context since. The reports themselves stay unchanged.
