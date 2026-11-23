# Straight to main

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The agents](../../agents/README.md)

## Rule
Agents write the work straight to `main`, without a pull request or review: drops and reports into the context, findings, claims and states in the plan, briefings, the plan from a finished planning, and additions to the spec that follow from what is decided. Artifacts reach `main` only through a Reviewer, and any other change to the spec lands with its realization ([spec change](../building/spec-change-lands-with-realization.md)). Writing to `main` is a commit on `main`, pushed; when the push is refused, the agent fetches, puts its commit on top and pushes again, and for a claim first checks again who was first. Nobody checks pull requests by hand: the way of working demands a fresh Reviewer per round, and observes itself, so reviews are not the only thread quality hangs on.
