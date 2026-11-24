# Claiming

Type: [Rule](../../core/rule.md)

## Applies to
[The plan](README.md), [every role](../team/README.md)

## Rule
Whoever starts to work on an item of the plan claims it first: writes themselves on `main` into the feature's Owner or the story's Claimed by, before they begin. Whoever is first on `main` has it; the other session sees this and stops. A decision on a claimed story, such as giving it up, is carried out on `main` like any state; the session that holds the claim sees it at its next fetch of `main` and stops the same way. This holds for everyone: the Architect with its person, the Builder, the Analyst and the person who accepts. Whoever claimed a story releases the claim when its session ends its work on the story, by taking itself out of Claimed by in the same step as the story's new state. A feature's Owner stays: ownership lies with the feature. So everyone sees in the views what is going on, and a crashed session shows as a claim that stands still, which the daily run releases after a day ([the daily run](../day/daily-run.md)).

## Example
Two persons want the feature "Renew online". The first claim on `main` wins; the second person's agent says so and proposes something else. When the planning of its first story is finished, the story's claim is released, and a Builder can claim it.
