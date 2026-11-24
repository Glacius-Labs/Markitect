# Dependencies are set explicitly

Type: [Rule](../../core/rule.md)

## Applies to
[Story](story.md), [the Architect](../team/architect.md), [the Project manager](../team/project-manager.md)

## Rule
Whoever creates a story sets its dependencies, every time and explicitly: either its predecessors, or `none`. A story without its section Depends on is not finished. Whoever plans a story that changes a document another open story changes too sees that story, which `markitect show` lists for the document, also when it lies in the plan of another module, and sets the order explicitly: the story depends on the other when it builds on what the other builds, and not otherwise.
