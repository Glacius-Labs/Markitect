# The types of findings

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Finding](../../../../packages/factory/finding.md), [the Triager](../../agents/triage.md), [the Recorder](../../agents/record.md)

## Rule
Every finding carries one of nine types, ordered by what it points at. The type decides its usual response, and its folder, with one exception: a Requirement or Question whose answer follows from what is decided goes to `ready`, the second of [the three exits](three-exits.md). There is no separate check whether something may enter the plan without a decision.

| Points at | Type | Means | Folder | Usual response |
|---|---|---|---|---|
| the spec | Requirement | a statement of what should hold that is missing in the spec or stands there differently | `pending` | work it out and examine it, then decide; taken over, it changes the spec |
| | Idea | a raw proposal for something new | `pending` | never taken over without consent; work it out, then place it in a feature, make it a new one, or park it |
| | Question | unclear or contradictory, needs an answer | `pending` | someone in particular answers; the answer goes into the context and often becomes a Requirement; when it blocks work, its importance is at least `high`, so it is shown daily |
| the product | Bug | does not do what it should | `ready` | a story with fixed tasks; only its priority is open, from likelihood and impact |
| | Risk | works today, may go wrong | `pending` | mitigate (into the plan), observe (an observer keeps an eye on it) or accept (close it with the reason) |
| | Smell | a sign that something is not clean or does not fit: a detour, a flow the structure prevents, a name that cannot be found | `ready` | a story for analysis at once; an existing one only gains a reference; how often decides its priority |
| the plan | Planning | order, scope, date or person | `pending` | adjust the plan; a blocker first |
| the way of working | Factory | the way of working itself catches | `pending` | change an agent, a rule, an observer, or add a type |
| none of these | Unclassified | fits no type | `pending` | the people say in the daily briefing what it is, and the Recorder enters the type; when none fits, it becomes a finding of type Factory: a type is missing |

`pending` needs a human, `ready` is released for work, `closed` is done. The Triager files into `closed` only noise and duplicates; in the end every finding lands in `closed`, with its outcome.
