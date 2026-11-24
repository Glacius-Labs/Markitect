# When a story starts

Type: [Rule](../../core/rule.md)

## Applies to
[Story](story.md), [the Assistant](../team/assistant.md), [the Housekeeper](../team/housekeeper.md)

## Rule
A story starts only when it is `ready` and all its predecessors have landed, that is, stand in `acceptance` or are `done`; only the state counts, and a planning session never builds itself. So a chain of stories does not wait for a person to accept: what is built is on `main`, and the next story builds on it. What the acceptance of a predecessor still changes reaches its successor through `main`, like any other change. A blocked story is simply one whose predecessor has not landed; it needs no state of its own. When it waits for a human, that human's commitment is its predecessor. A `cancelled` predecessor has not landed: whoever gives a story up says what becomes of the stories that wait for it ([giving up a story](giving-up-a-story.md)). Without a human, only a story of priority `normal` or higher starts.

## Example
"Renew a loan" is `ready` with priority `high`, but its predecessor "Show due dates" is `in progress`: the Housekeeper leaves it. The watch on [the plan items](../observing/watch-plan-items.md) reports when it stays blocked.
