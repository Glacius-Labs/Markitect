# A spec change lands with its realization

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [finish-planning](../../agents/finish-planning.md), [the Builder](../../agents/build.md)

## Rule
When a planning changes the standing spec, the change lies on the branch of the story that realizes it, not on `main`. When it concerns several stories, each carries the part it realizes. The Builder continues on that branch, and spec and code go to `main` together, so the spec on `main` is never ahead of a planned story. The plan itself goes to `main` at once. When two stories change the same place of the spec, each carries its part on its branch; the conflict shows when they are merged, and whoever resolves it leaves a note, as with every merge conflict. An addition that only writes down what follows from what is decided stands on `main` without a build: it adds nothing new.

## Example
The feature "Renew online" changes the spec's lending period. The story "Renew a loan" carries that change on its branch and brings it to `main` together with the code.
