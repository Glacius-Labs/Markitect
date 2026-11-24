# Release

Type: [Activity](../../core/activity.md)

A person brings what is accepted into production. Agents prepare and check; the release itself is a human's.

## Participants
The person who releases, in the role whose document says so; by default the [Admin](../team/admin.md).
The [Assistant](../team/assistant.md) prepares; a fresh [Reviewer](../team/reviewer.md) checks.

## Trigger
A person says they want to release; `main` can always be released ([off until accepted](off-until-accepted.md)). Whichever agent hears it sends the person to the Assistant.

## Steps
1. Fetch `main` and say what a release would contain: the features accepted since the last release, which are switched on, and the stories without a feature that are `done`; and, apart from these, what lies on `main` switched off, the landed stories that are not accepted yet ([off until accepted](off-until-accepted.md)).
2. When something that is not accepted can be reached by users, because it was built without its switch or the switch is on, say so in one sentence and ask whether to wait. Releasing it anyway is a [critical](../deviations/critical.md) deviation: go on only after a yes, and drop the reason ([the course of a deviation](../deviations/course.md)).
3. Start a fresh Reviewer, who checks everything against everything: the stories against their feature, the features against the model ([review](../review/review.md)). What it finds is dropped into the context; whether it holds up the release, the person decides.
4. The person releases, the way the project's model says the project is released. An agent releases only when the person tells it to in so many words; that is a critical deviation like any other, and after the yes it goes on.
5. Leave a [report](../observing/report.md) in the context: what was released, when, and what the Reviewer found.

## Result
What was accepted is in production, released by a person; the report says what went out. Or the person decided to wait, and nothing went out.

## Notes
How the project is released, where to and with which steps, belongs to the project and stands in its own model, in the module for its infrastructure ([a module for infrastructure](../plan/infrastructure-modules.md)); it is entered when the project is set up ([set up the project](../set-up.md)).
