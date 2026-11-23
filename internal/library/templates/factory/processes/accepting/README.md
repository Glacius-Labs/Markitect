# Accepting

Type: [Process](../../../../packages/company/process.md)

## Trigger
A story has landed on `main` and stands in `acceptance`; or nothing new is there, and whole features wait to be looked at. [The Assistant](../../agents/assistant.md) starts [accept](../../agents/accept.md) for the person who accepts.

## Steps
1. [Take](take.md)
2. [Polish](polish.md)
3. [Review](review.md), in rounds
4. [Accept](accept.md)

## Result
The story is `done`; a feature whose stories are all done is accepted as a whole, and only then may it go into [production](../../rules/building/production.md). Friction and anything fundamental went back into the circle. Or the acceptance failed within its rounds, and the story is back in `draft`, as after an [abort](../../rules/building/abort.md).

## Responsible
The person who accepts, in the role whose document says so; by default the [Admin](../../roles/admin.md).

## Involved
The agent [accept](../../agents/accept.md) works with that person; a fresh [Reviewer](../../agents/review.md) checks the result of the polishing in each round.
