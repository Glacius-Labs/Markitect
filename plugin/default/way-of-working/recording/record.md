# Record

Type: [Activity](../../core/activity.md)

## Trigger
Decisions came into the context, or findings are `ready`. The [Housekeeper](../team/housekeeper.md) starts it in the [daily run](../day/daily-run.md), after the triage.

## Steps
1. Read the decisions that came into the context since the checkpoint ([checkpoints](../work/checkpoints.md)), such as the reports on meeting transcripts. Match each decision to its finding by the finding's title or link, or by the briefing line it was decided on; a decision that points at a finding but cannot be matched to one is dropped into the context as unclear instead of guessed. A decision that names no finding was decided where it arose: the Triager has filed it as a finding of its own, `ready` with the decision in its context ([what is no finding](../triage/not-a-finding.md)), and it is carried out in step 3.
2. Carry out each decision as it demands, and note it at its finding, in its outcome ([finding](../triage/finding.md)).
   - What is to be done becomes a plan item ([one channel](../plan/one-channel.md)): a feature `open` whose Stories and Acceptance are `not planned yet`, or a story `draft` whose Plan and Tasks are `not planned yet` and which names under Realizes the place the finding names, as far as it is known. The planning completes both.
   - A decision that says in so many words what should hold: write the sentence into the model, at the place the decision names or where it plainly belongs, in place of what it contradicts there, and note in the finding's outcome the place, the sentence and what it replaced. Create a story `draft` for it that names that document and the sentence under Realizes; its planning checks that the sentence fits the levels above and says how it is built. When it is unclear where the sentence belongs or what it replaces, do not guess: create only the story, and its planning writes the sentence.
   - A decision that asks a person to act becomes their story, as [commitments](../plan/commitments.md) says.
   - A decision to observe something, for a Risk, a new source or a finding of type Process, becomes a story `draft` like any other, in the plan of the module [observing](../observing/README.md) ([add a watch](../observing/add-a-watch.md)): it realizes the README of that module and the watch's document, named when it is planned; its plan and tasks are `not planned yet`, its origin is the finding.
   - A decision to give up a story: carry it out as [giving up a story](../plan/giving-up-a-story.md) says. Set the story `cancelled`, release its claim, and write into its Result that it was given up, why, with a link to the decision; hand over or mark the documents only it realizes, and settle the stories that depend on it. When it was the last open story of its feature, set the feature to `acceptance`.
   - For an Unclassified finding, enter the type the people gave it; when none fits, it becomes a finding of type Process: a type is missing.
3. Turn the findings in `ready` into plan items without a further decision:
   - A finding that was decided where it arose: carry out the decision in its context as in step 2, and note `decided` as its outcome.
   - A Bug becomes a story with [fixed tasks](../plan/fixed-tasks.md) in the plan of the finding's module, priority at least `normal`.
   - For a Smell, look for its existing smell story first and only add the reference to its origin; when there is none, create one with fixed tasks, at the priority [priorities](../plan/priorities.md) gives. A deviation references the smell story for this person and this kind of deviation. Raise the priority by the references, as that rule says.
   - A Requirement or Question whose answer follows from what is decided: write the answer into the model, at the place the finding names ([the three exits](../triage/three-exits.md)). This is no deviation. Create a story with [fixed tasks](../plan/fixed-tasks.md) for it that names the document and the sentence under Realizes, so that the addition counts as not built until the story has landed ([the model runs ahead](../plan/the-model-runs-ahead.md)).
4. What a person commits to in a conversation becomes their story, as [commitments](../plan/commitments.md) says.
5. Create nothing in a claimed feature: attach the finding to it by its outcome and leave it `ready`, so that the Assistant shows it to the feature's owner; the Architect closes it ([finding](../triage/finding.md)).
6. Set the priority of each new story by your own judgment from its finding's importance ([priorities](../plan/priorities.md)), and its dependencies explicitly ([dependencies](../plan/dependencies.md)). Name new plan items as [names in the work](../work/names.md) says.
7. Close each finding that was carried out, with its outcome. A `critical` finding stays `ready` after its decision and is closed only when the stories that arose from it are `done` or `cancelled` ([critical until done](../briefing/critical-until-done.md)).
8. Write the plan items, the findings and the checkpoint in one step. Leave a [report](../observing/report.md) of the run in the context.

## Result
Every decision is noted at its finding and carried out; every finding that was `ready` stands in the plan or in the model; each of them is closed with its outcome, except those attached to a claimed feature, which wait there for its planning. The checkpoint moved, and the report of the run lies in the context.

## Participants
The [Project manager](../team/project-manager.md).
