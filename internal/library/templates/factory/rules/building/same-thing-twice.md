# The same thing twice is a smell

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The Builder](../../agents/build.md), [the Reviewer](../../agents/review.md), [duplication](../../observers/duplication.md)

## Rule
When the same thing is built in several places, that is a smell: it is built once and used from there. The observer duplication looks for it regularly; Builders and Reviewers report it when they notice it.
