# When a story starts

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [Norman](../../agents/norman.md), [the Housekeeper](../../agents/housekeeper.md)

## Rule
A story starts only when it is `ready` and all its predecessors are `done`; only the state counts, and a planning session never builds itself; the one agent that plans and realizes a story in one session is create-observer, for an observer's story ([milestones](milestones.md)). A blocked story is simply one whose predecessor is still open; it needs no state of its own. When it waits for a human, that human's commitment is its predecessor. Without a human, only a story of priority `normal` or higher starts.

## Example
"Renew a loan" is `ready` with priority `high`, but its predecessor "Show due dates" is `in progress`: the Housekeeper leaves it. The observer [plan-items](../../observers/plan-items.md) reports when it stays blocked.
