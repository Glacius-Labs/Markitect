# Start of work

Type: [Activity](../../core/activity.md)

## Trigger
A person begins their work on the project, in a new session with the Assistant: "I want to start. What do we do?"

## Steps
1. Fetch the state of `main`. Know whom you serve: the person whose document in [the team](../team/README.md) matches the identity this session runs under; when none matches, ask, and offer to enter them. Show the person their [views](../views/README.md) where the project has built them; else tell what they would show directly from the work.
2. When today's daily briefing is not on `main`, say so and offer to start the [daily run](daily-run.md). Never start it unasked.
3. Tell, in this order and in sentences, not in numbers: what is critical ([when shown](../briefing/when-shown.md)); what has happened since the person's last session, from the briefings and the reports since then: what landed, what was aborted, what was added to the model because it follows from what is decided, and what went to work without a decision, each open to objection; how far each feature in progress is, which of its stories are built and which are not yet; what waits for this person; what the others do, as far as it touches this person; which briefings lie ready, today's daily one and any meeting's due today. Then propose in one sentence where to begin.
4. Propose by what the person's role usually does: an aborted build to reopen, the next story to plan, among them the `draft` stories in the person's features and in the plans of the modules the person is responsible for, the oldest item in `acceptance`, the findings that wait for a decision, the model itself while the project has no modules or no plan yet, an open feature that nobody has taken yet, an accepted feature that waits to be [released](../accepting/release.md), what concerns the way of working. Show the owner of a claimed feature the findings attached to it, the `ready` ones whose outcome links the feature, until the Architect closes them in that feature's planning ([finding](../triage/finding.md)); propose that planning when they wait. Show each person the stories they do themselves ([commitments](../plan/commitments.md)), and when they say one is done, set it `done` and release the claim.
5. For the chosen work, start the fitting role and brief it, as [starting an agent](../team/starting-an-agent.md) says: the [Architect](../team/architect.md) or the [Acceptance helper](../team/acceptance-helper.md) in a session of its own, to which the person switches; a [Builder](../team/builder.md) is never started from the person's session: for a story that is `ready`, start the [daily run](daily-run.md), which starts its Builder and does not depend on this session. Every `draft` story goes to the Architect.
6. When the person decides on a finding, objects to something, or decides something new, drop the decision into the context, naming the finding by its link where there is one ([decide](../briefing/decide.md)). Then offer to start the [daily run](daily-run.md) at once, so that the decision is carried out today and the person can go on with it; say that a run starts several agents. Offer the same when the person hands over material, such as a transcript: you do not judge what is in it, you only say that the next run takes it up and that it can run now.
7. When a person comes back from planning or from accepting, or a run you started has carried out what was decided, which shows on `main` long before its Builders are through, look at what was finished and say in a sentence how it ended; propose to start the run for a story now `ready`, then the next planning.
8. Lead by the role documents: when the person's work lies outside their role's usual way, say so in one sentence and ask for the reason ([the course of a deviation](../deviations/course.md)). At `critical`, go on only after a yes.
9. Leave a [report](../observing/report.md) in the context at the end ([one per session](../observing/one-per-session.md)).

## Result
The person knows what is critical, what happened and what waits for them, and the work they chose is under way with the fitting role. Their decisions lie in the context; the stories they did themselves are `done`. Nothing the session started depends on it staying open.

## Participants
The [Assistant](../team/assistant.md).
