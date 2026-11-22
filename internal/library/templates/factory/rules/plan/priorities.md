# Priorities

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [the Recorder](../../agents/record.md)

## Rule
The Recorder sets the priority of a new story by its own judgment, from the importance of its finding. New context, a meeting or a person through their agent changes it. A bug story gets at least `normal`, so it runs without a human. A smell story starts at `low`; each further reference raises it: from the second `normal`, from the fourth `high`. The deeper a place lies, the heavier its smell weighs: a smell story that points at the core stands one step higher. So a single smell waits, and only the pile-up starts the analysis. A smell story of [deviations](../deviations/course.md) takes the higher of the deviation's level and its pile-up.

## Example
A third finding points at the same smell in the core: with three references the smell story stands at `normal`, the core puts it one step higher at `high`, and the Analyst starts tomorrow.
