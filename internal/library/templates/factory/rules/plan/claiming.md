# Claiming

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The plan](README.md), [the agents](../../agents/README.md)

## Rule
Whoever starts to work on an item of the plan claims it first: writes themselves on `main` into the feature's Owner or the story's Claimed by, before they begin. Whoever is first on `main` has it; the other session sees this and stops. This holds for everyone: the Architect with its person, the Builder, the Analyst and the QA. Whoever claimed a story releases the claim when its session ends its work on the story, by writing `none` into Claimed by in the same step as the story's new state. A feature's Owner stays: ownership lies with the feature. So everyone sees on the dashboards what is going on, and a crashed session shows as a claim that stands still.

## Example
Two persons want the feature "Renew online". The first claim on `main` wins; the second person's agent says so and proposes something else. When the planning of its first story is finished, the story's claim goes back to `none`, and a Builder can claim it.
