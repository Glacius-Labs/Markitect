# Priorities

Type: [Rule](../../core/rule.md)

## Applies to
[Story](story.md), [the Project manager](../team/project-manager.md)

## Rule
Whoever creates a story sets its priority: the Project manager by its own judgment, from the importance of its finding; the Architect in a planning, with the person, `normal` unless the person says otherwise. New context, a meeting or a person through their agent changes it. A bug story gets at least `normal`, so it runs without a human. A smell story starts at `low`; each further reference raises it: from the second `normal`, from the fourth `high`. So a single smell waits, and only the pile-up starts the analysis. A smell story of [deviations](../deviations/course.md) takes the higher of the deviation's level and its pile-up.

## Example
A third finding points at the same smell: with three references the smell story stands at `normal`, and the Analyst starts in the next run.
