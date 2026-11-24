# The deviations

Type: [Watch](watch.md)

State what happened on `main` against the usual way of working of the roles: every change on `main` outside the usual way of the role that made it, also and above all what nobody reported themselves, such as a merge to `main` without a Reviewer. The first run records the state of `main` and reports nothing: what was there before the watch began is no deviation.

## Participants
[Observer](../team/observer.md)

## Trigger
Daily.

## Steps
[Observe](observe.md)

## Result
A report with the statements, or only the checkpoint when there was nothing to state.

## Source
The Git history of `main`, read locally, held against the [role documents](../team/README.md): who changed what, whether there was a review, whether a claim stood, and whether a change brought a secret or customer data into the repository, looked for with the tool the project names when it is set up. The watch states what happened; the [level](../deviations/levels.md) is the Triager's. Observed to: the commit up to which it observed.

## Never in the report
Customer data and secrets, also not the secret that was committed: only the reference. No guessed reason: a reason stands only as written in the deviation's note.
