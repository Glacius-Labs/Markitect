# Dependencies are set explicitly

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [the Architect](../../agents/architect.md), [the Recorder](../../agents/record.md)

## Rule
Whoever creates a story sets its dependencies, every time and explicitly: either its predecessors, or `none`. A story without its section Depends on is not finished.
