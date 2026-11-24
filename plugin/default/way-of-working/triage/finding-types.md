# The types of findings

Type: [Rule](../../core/rule.md)

## Applies to
[Finding](finding.md), [the Triager](../team/triager.md), [the Project manager](../team/project-manager.md)

## Rule
Every finding carries one of nine types, ordered by what it points at. The type decides its usual response, and how it is filed, with one exception: a Requirement or Question whose answer follows from what is decided goes to `ready`, the second of [the three exits](three-exits.md). A finding that was decided where it arose, whatever its type, goes to `ready` as well, with the decision in its context: no human has to decide it a second time. A finding of importance `critical` goes to `pending`, whatever its type: what is critical never goes to work without a human. For a critical deviation that is the damage itself; the Smell that asks why it could happen is filed at `high` and goes to its analysis as always ([the course of a deviation](../deviations/course.md)). There is no separate check whether something may enter the plan without a decision.

| Points at | Type | Means | Filed as | Usual response |
|---|---|---|---|---|
| the model | Requirement | a statement of what should hold that is missing in the model or stands there differently | `pending` | work it out and examine it, then decide; taken over, it changes the model |
| | Idea | a raw proposal for something new | `pending` | never taken over without consent; work it out, then place it in a feature, make it a new one, or park it |
| | Question | unclear or contradictory, needs an answer | `pending` | someone in particular answers; the answer goes into the context and often becomes a Requirement; when it blocks work, its importance is at least `high`, so it is shown daily |
| the product | Bug | does not do what it should | `ready` | a story with fixed tasks; only its priority is open, from its importance |
| | Risk | works today, may go wrong | `pending` | mitigate (into the plan), observe (a watch keeps an eye on it) or accept (close it with the reason) |
| | Smell | a sign that something is not clean or does not fit: a detour, a flow the structure prevents, a name that cannot be found | `ready` | a story for analysis at once; an existing one only gains a reference; how often decides its priority |
| the plan | Planning | order, scope, date or person | `pending` | adjust the plan; a blocker first |
| the way of working | Process | the way of working itself catches | `pending` | change a role, a process, a rule or a watch, or add a type |
| none of these | Unclassified | fits no type | `pending` | the people say in the daily briefing what it is, and the Project manager enters the type; when none fits, it becomes a finding of type Process: a type is missing |

`pending` needs a human, `ready` is released for work, `closed` is done. The Triager files into `closed` only noise and duplicates; in the end every finding lands in `closed`, with its outcome.
