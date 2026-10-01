# Intentional consistency conflict

This standalone fixture declares two different owners for `release-approval` under the opt-in functional `owner` predicate. Run `markitect check --repo examples/consistency-conflict`; the expected result is `consistency.conflict`, with both source locations and resource owners. The declarations are illustrative, not a policy claim about an adopting project.
