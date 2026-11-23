# Straight to main

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The agents](../../agents/README.md)

## Rule
Agents write the work straight to `main`, without a pull request or review: drops and reports into the context, findings, claims and states in the plan, briefings, the plan from a finished planning, additions to the spec that follow from what is decided, and the parts of the spec an extraction wrote, once the person has decided them. Artifacts reach `main` only through a Reviewer, and any other change to the spec lands with its realization ([spec change](../building/spec-change-lands-with-realization.md)). Writing to `main` is a commit on `main`, pushed; when the push is refused, the agent fetches, puts its commit on top and pushes again, and for a claim first checks again who was first. What belongs to building a story, such as an observer's trial run, lies on the story's branch and reaches `main` with its merge. An agent that works on a story's branch writes the rest of the work to `main` from a working folder of its own for `main`, such as a second worktree, and never carries uncommitted changes from one branch to the other. Nobody checks pull requests by hand: the way of working demands a fresh Reviewer per round, and observes itself, so reviews are not the only thread quality hangs on.
