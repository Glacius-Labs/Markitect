# The same thing twice is a smell

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The Builder](../../agents/build.md), [the Reviewer](../../agents/review.md), [duplication](../../observers/duplication.md)

## Rule
When every module builds the same thing, that is a smell. Either a helper module comes out of it, or the core is enriched with it. The observer duplication looks for it regularly; Builders and Reviewers report it when they notice it.
