# Three ways to main

Type: [Rule](../../core/rule.md)

## Applies to
[Every role](../team/README.md)

## Rule
There are three ways to `main`. The work is written straight to `main`, without a branch and without a review: drops, reports and checkpoints, findings, features and stories with their claims and states, and briefings; so are additions that only write down what follows from what is decided, a statement of what should hold that a person decided in so many words, which the Project manager enters as it was said, and the modules an extraction wrote, once the person has decided them. Everything else that says what should hold changes through a planning: the Architect writes it with a person, a Reviewer checks it against the levels above, and it goes to `main` at the end of the planning, before anything is built ([the model runs ahead](../plan/the-model-runs-ahead.md)). Artifacts change through a story: on the story's branch, checked by a Reviewer in every round. So nobody plans a change to the plan, and nothing is built that the model does not say.

Writing to `main` is a commit on `main`, pushed; when the push is refused, the writer fetches, puts the commit on top and pushes again, and for a claim first checks again who was first. What belongs to building a story, such as a watch's trial run or the report of the Reviewer that confirms it, lies on the story's branch and reaches `main` with its merge. Whoever works on a story's branch writes the rest of the work to `main` from a working folder of its own for `main`, such as a second worktree, and never carries uncommitted changes from one branch to the other. Nobody checks pull requests by hand: the way of working demands a fresh Reviewer per round, and observes itself, so reviews are not the only thread quality hangs on.
