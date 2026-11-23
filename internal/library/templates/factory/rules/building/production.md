# Production

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Accepting](../../processes/accepting/README.md), [the Reviewer](../../agents/review.md)

## Rule
A feature may go into production only once it is accepted. Before that, a Reviewer checks everything against everything: the stories against the feature, the feature against the spec. The release itself stays a human's, given the way the project releases. Going into production otherwise is a [critical](../deviations/critical.md) deviation.
