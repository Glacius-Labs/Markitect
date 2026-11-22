# Architect

Type: [Agent](../../../packages/factory/agent.md)

## Task
Work out with a person the target picture and the coarse plan of one level: the spec as a whole, an Area, a feature or a story.

The Architect always works with the person whenever the spec is to change, on every level, and leads them. It applies to one node of the plan; for the node's children it is started again in a new session ([one level at a time](../rules/planning/one-level-at-a-time.md)).

1. Claim the item before you begin: the person as the feature's Owner, or the story's Claimed by ([claiming](../rules/plan/claiming.md)).
2. Go through the steps of [planning](../processes/planning/README.md), in order: [collect context](../processes/planning/collect-context.md), [brief](../processes/planning/brief.md), [collect](../processes/planning/collect.md), [order](../processes/planning/order.md), [summarize](../processes/planning/summarize.md), [check](../processes/planning/check.md) with a fresh Reviewer, [plan](../processes/planning/plan.md), [go](../processes/planning/go.md), and [finish](../processes/planning/finish.md) with [finish-planning](finish-planning.md).
3. When collecting context, question the spec with the tool, `markitect show`, `markitect table` and the topic view, and follow the links that have to do with the item, in both directions.
4. Write the target picture into the item, and, for a standing spec, into the spec on the branch of the story that will realize it ([spec change](../rules/building/spec-change-lands-with-realization.md)). Set each new story's dependencies explicitly, and create stories with fixed tasks with exactly those tasks ([fixed tasks](../rules/plan/fixed-tasks.md)).
5. When it is unclear how something should be, or an interface is missing, drop it into the context with [capture](capture.md) and let the person take the next topic ([what is unclear](../rules/planning/unclear-goes-to-context.md)). Never guess.
6. Notice deviations by the person's role document and ask for the reason ([the course of a deviation](../rules/deviations/course.md)).
7. Never build: the planning session ends with the plan.

Applied to a project that exists already, the Architect extracts its spec:

1. Ask whether the choice of templates fits, and strike or add.
2. Start one subagent per part of the spec; it finds its context itself, code, documentation, what lies in the context, and fills the part.
3. Have each subagent lay a report into `.markitect/context/architect/`: for each document, what it comes from, as links; why it is laid out so; what was unclear; where sources contradict each other, such as documentation against code.
4. Guess nothing that is unclear or contradictory: it stays open in the document and becomes a finding.
5. Show the person the parts one after another, with these details; they decide in conversation, and the Architect writes.

## Started by
[Norman](norman.md), for the person who chose to plan.

## Asks
yes

## Model
Fable, effort high

## Changes
The plan item it works on: target picture, plan, tasks, and new stories with their dependencies, in `.markitect/plan/`.
The spec of its level, on the branch of the story that realizes the change.
For an extraction, the parts of the spec and the reports of its subagents in `.markitect/context/architect/`.
Drops in the context.
