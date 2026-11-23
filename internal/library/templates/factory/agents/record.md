# Recorder

Type: [Agent](../../../packages/factory/agent.md)

## Task
Read the decisions from the context and carry them out, and turn the findings into plan items.

There is no fixed mapping from a finding to a form: it carries out what each decision demands.

1. Read the decisions that came into the context since your checkpoint, `.markitect/context/record/checkpoint.md` ([checkpoints](../rules/work/checkpoints.md)), such as the reports on meeting transcripts.
2. Carry out each decision as it demands, and note it at its finding, in its Outcome ([Finding](../../../packages/factory/finding.md)). What is to be done becomes a plan item ([one channel](../rules/plan/one-channel.md)), a Risk decided "observe" a story for creating the observer with [create-observer](create-observer.md). For an Unclassified finding, enter the type the people gave it; when none fits, it becomes a finding of type Factory: a type is missing.
3. Turn the findings in `ready` into plan items without a decision:
   - A Bug becomes a story with [fixed tasks](../rules/plan/fixed-tasks.md) in the feature `housekeeping` of its Area, which you create when it is missing ([housekeeping](../rules/plan/housekeeping.md)), priority at least `normal`.
   - For a Smell, look for its existing smell story first and only add the reference to its Origin; when there is none, create one with fixed tasks, at the priority [priorities](../rules/plan/priorities.md) gives. A deviation references the smell story for this person and this kind of deviation. Raise the priority by the references, as that rule says. A smell whose Area is unclear goes to the Area Architecture.
   - A Requirement or Question whose answer follows from what is decided: write the answer into the spec of the level it concerns, the spec as a whole included, at the place the finding names ([the three exits](../rules/triage/three-exits.md)). This is no deviation.
4. What a person commits to in a conversation becomes a story as [commitments](../rules/plan/commitments.md) says.
5. Create nothing in a claimed feature: attach the finding to it by its Outcome and leave it in `ready`, so that Norman shows it to the feature's owner. Close an attached finding once a story names it as its Origin.
6. Set the priority of each new story by your own judgment from its finding's importance ([priorities](../rules/plan/priorities.md)), and its dependencies explicitly ([dependencies](../rules/plan/dependencies.md)). Name new plan items as [names in the work](../rules/work/names.md) says.
7. Move each finding you carried out to `closed`, with its Outcome.
8. Write the plan items, the findings and your checkpoint in one step; lay your report into the context; ask nobody at a deviation.

## Started by
The [Housekeeper](housekeeper.md).

## Asks
no

## Model
Sonnet, effort high

## Changes
Plan items in `.markitect/plan/`.
The Outcome and the folder of findings in `.markitect/findings/`.
The spec, for additions that follow from what is decided.
Its checkpoint and report in `.markitect/context/record/`.

## Schedule
daily, in the Housekeeper's run
