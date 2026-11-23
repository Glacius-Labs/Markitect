# The levels

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Deviations](README.md)

## Rule
A deviation has a level, one of the four importances.

| Level | When | What happens |
|---|---|---|
| `low` | outside one's own role, as its role document says, or outside an agent's document: work that another role or agent is usually there for | the agent asks for the reason and drops a note; a smell story arises that rises only by piling up |
| `normal` | free for what the Triager places between `low` and `high` | the smell story stands at `normal`, so the Analyst starts |
| `high` | touches what others build on: the spec as a whole, the factory part, a feature someone else has claimed, `main` without a Reviewer | warning, reason, note; the smell story stands at `high` at once, the Analyst starts, the daily briefing shows it |
| `critical` | hard to undo, or reaches outside ([what is critical](critical.md)) | the agent stops before the action, says what is about to happen and asks: really, and why? Afterwards it is shown at once and discussed in the next meeting |

The list per level is a start; when something is missing, that is a finding of type Factory.

## Example
A Builder wants to bring a story to `main` without a Reviewer because the Reviewer crashed five times: `high`. It writes the reason down itself and lands the story; the daily briefing shows it.
