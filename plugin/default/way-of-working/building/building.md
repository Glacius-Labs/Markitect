# Building

Type: [Activity](../../core/activity.md)

## Trigger
A story is `ready` and all its predecessors have landed ([when a story starts](../plan/when-a-story-starts.md)). The [Housekeeper](../team/housekeeper.md) starts the Builder, in the [daily run](../day/daily-run.md), by priority; a person who wants a story built now has [the Assistant](../team/assistant.md) start the run. When a Builder has finished, the Housekeeper starts the stories its story freed.

## Steps
1. **Start only what may start.** The Builder starts only a story that may start ([when a story starts](../plan/when-a-story-starts.md)). When it may not, it claims nothing and ends with its report.
2. **Claim.** Before it begins, the Builder writes itself into the story's Claimed by and sets it `in progress`, on `main` ([claiming](../plan/claiming.md)). When another session was first, it stops.
3. **Build.** The Builder carries out the story on the story's branch, named as [names in the work](../work/names.md) says, task by task; what is to be accepted is left to whoever accepts. It cuts the branch from `main`; when the branch exists already, from an earlier attempt, it first takes the current `main` into it. It builds what the documents its story names say; what a document of another open story says is not its to build, and where it shares a document with one, its own part is what its sentence under Realizes says ([the model runs ahead](../plan/the-model-runs-ahead.md)). It does not ask. Within the model it decides itself and takes the simple or common solution, simple in the sense of the concept. For a story that adds a watch, [add a watch](../observing/add-a-watch.md) says what it writes and how the trial run goes.
4. **Review round**, repeated. Before each round the Builder runs `markitect check` on the model and the work it changed, looks at what changed on the story's branch, and settles what they show. Then a fresh [Reviewer](../team/reviewer.md) checks the realization, as [review](../review/review.md) says, and gives back comments, which the Builder answers, until they agree. Without agreement within the rounds that process allows, the Builder [aborts](abort.md).
5. **Confirm.** When Builder and Reviewer agree, the Reviewer runs the targeted tests itself and confirms each place in the model that the story names ([one trace](../review/one-trace.md)). The confirmation is the Reviewer's yes, no extra stop.
6. **Land.** Right before the merge, the Builder fetches `main`; when the story was given up meanwhile, it stops as for a lost claim ([claiming](../plan/claiming.md)). When taking `main` in changed a file the Reviewer has confirmed, one more review round confirms the result before it lands. Otherwise it writes what changed into the story's Result and brings the story to `main` as one merge, whose commit names the story. It sets the story to `acceptance`, or to `done` when there is nothing to accept, and releases its claim. When the story is `done`, was the last open story of its feature, and no finding is attached to the feature, it sets the feature to `acceptance`. No human checks before `main`.

## Result
What the story built is on `main`, with the Reviewer's confirmation, and stands in `acceptance`, or `done` when there is nothing to accept. No human checked it before `main`. Or the build was [aborted](abort.md): the story is back in `draft`, with a report in the context that the story links.

## Participants
The [Builder](../team/builder.md).
A fresh [Reviewer](../team/reviewer.md) per round.

## Notes
The Builder [aborts](abort.md) when that rule says so, and does what it says. It reports a smell instead of building around it ([no workarounds](no-workarounds.md)), reports the same thing built twice ([same thing twice](same-thing-twice.md)), and drops into the context what remains worth mentioning after the review rounds ([drop what you notice](../observing/drop-what-you-notice.md)). At a deviation it asks nobody ([agents that do not ask](../deviations/agents-that-do-not-ask.md)). At the end it leaves its [report](../observing/report.md) in the context ([one per session](../observing/one-per-session.md)).
