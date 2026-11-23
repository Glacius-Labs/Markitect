# The plan is a tree

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The plan](README.md)

## Rule
The plan is a tree of four levels. An Area lasts and has an owner and, optionally, a class. A feature is an idea we want to build, with its own target picture, and ends with its acceptance. A story is complete in itself once its predecessors are done. A task is a step of a story; a planned story always has tasks. Ownership lies with the feature; carried out is the story. Usually a Developer follows one Area. The items lie in `.markitect/plan/`: a folder per Area, in it a folder per feature, in it a file per story. An item's place in the tree is its identity; it has no number.

## Example
`.markitect/plan/lending/renew-online/renew-a-loan.md` is the story "Renew a loan" of the feature "Renew online" in the Area Lending.
