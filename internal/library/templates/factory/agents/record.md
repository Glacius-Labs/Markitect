# Recorder

Type: [Agent](../../../packages/factory/agent.md)

## Task
Read the decisions from the context and carry them out, and turn the findings into plan items.

There is no fixed mapping from a finding to a form: it carries out what each decision demands.

1. Read the decisions that came into the context since your checkpoint, `.markitect/context/record/checkpoint.md` ([checkpoints](../rules/work/checkpoints.md)), such as the reports on meeting transcripts. Match each decision to its finding by the finding's title or link, or by the briefing line it was decided on; a decision you cannot match to one finding, drop into the context as unclear instead of guessing.
2. Carry out each decision as it demands, and note it at its finding, in its Outcome ([Finding](../../../packages/factory/finding.md)). What is to be done becomes a plan item ([one channel](../rules/plan/one-channel.md)); a decision that asks a person to act becomes their story as [commitments](../rules/plan/commitments.md) says; a decision to observe something, for a Risk, a new source or a finding of type Factory, becomes a story `draft` for [create-observer](create-observer.md), in the `housekeeping` of the Area that covers the factory part: its Target picture an observer for what was decided, its Target changes the observers' README and the observer's document, named when it is planned, its Plan and Tasks `not planned yet`, its Origin the finding. For an Unclassified finding, enter the type the people gave it; when none fits, it becomes a finding of type Factory: a type is missing.
3. Turn the findings in `ready` into plan items without a decision:
   - A Bug becomes a story with [fixed tasks](../rules/plan/fixed-tasks.md) in the feature `housekeeping` of the finding's Area, which you create when it is missing ([housekeeping](../rules/plan/housekeeping.md)), priority at least `normal`.
   - For a Smell, look for its existing smell story first and only add the reference to its Origin; when there is none, create one with fixed tasks, at the priority [priorities](../rules/plan/priorities.md) gives. A deviation references the smell story for this person and this kind of deviation. Raise the priority by the references, as that rule says.
   - A Requirement or Question whose answer follows from what is decided: write the answer into the spec, at the place the finding names ([the three exits](../rules/triage/three-exits.md)). This is no deviation.
4. What a person commits to in a conversation becomes their story as [commitments](../rules/plan/commitments.md) says.
5. Create nothing in a claimed feature: attach the finding to it by its Outcome and leave it in `ready`, so that Norman shows it to the feature's owner; the Architect closes it ([Finding](../../../packages/factory/finding.md)).
6. Set the priority of each new story by your own judgment from its finding's importance ([priorities](../rules/plan/priorities.md)), and its dependencies explicitly ([dependencies](../rules/plan/dependencies.md)). Name new plan items as [names in the work](../rules/work/names.md) says.
7. Move each finding you carried out to `closed`, with its Outcome.
8. Write the plan items, the findings and your checkpoint in one step; lay your report into the context; ask nobody at a deviation.

## Started by
The [Housekeeper](housekeeper.md).

## Asks
no

## Model
Claude: Sonnet, effort high
Codex: Luna, effort high

## Changes
Plan items in `.markitect/plan/`.
The Outcome and the folder of findings in `.markitect/findings/`.
The spec, for additions that follow from what is decided.
Its checkpoint and report in `.markitect/context/record/`.

## Schedule
daily, in the Housekeeper's run
