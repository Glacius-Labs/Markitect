# The daily run

Type: [Activity](../../core/activity.md)

## Trigger
At least once a day: a person starts it, or a routine does. It may run again at any time, and then does only what is new, since every station continues from its checkpoint ([checkpoints](../work/checkpoints.md)). [The Assistant](../team/assistant.md) offers to start it when today's briefing is missing, and again when a person has decided or dropped something that should not wait until tomorrow; it never starts it unasked. The run is a session of its own, whoever starts it: it does not depend on a person's session, so what it started goes on when that person leaves.

## Steps
1. Run the watches that are due, as the [checkpoints](../work/checkpoints.md) say: one fresh [Observer](../team/observer.md) per watch ([observe](../observing/observe.md)). A full run of all watches only when the person who starts it asks for one.
2. Run the [triage](../triage/triage.md).
3. Run the [record](../recording/record.md).
4. Release what stands still: a story `in progress` that an agent claimed more than a day ago, with no commit on its branch since, goes back to `ready` and loses its claim; the report of the run says so. Then start [Builders](../team/builder.md), by priority, for every story in `ready` whose predecessors have all landed and whose priority is `normal` or higher ([when a story starts](../plan/when-a-story-starts.md)), except smell analyses: a story whose tasks are the [fixed tasks](../plan/fixed-tasks.md) of a smell analysis is the Analyst's alone. Start [Analysts](../team/analyst.md) for those, by priority. A story a person does is never `ready`, so it is nobody's to start ([commitments](../plan/commitments.md)). A story for an addition that this run wrote into the model waits for the first run of the next day, so that a briefing has shown the addition and a person could object before it is built. Who started the run, a person or a routine, makes no difference.
5. Have the [briefings written](../briefing/write-the-briefings.md). The record runs before it, so that the briefing shows what was recorded since the last one, and the Builders are started before it, so that it says what was started. The daily briefing is written once a day: a later run on the same day leaves it as it is.
6. When a Builder or an Analyst finishes, start what its story freed: the stories that may start now, by the same rule as in step 4, until nothing may start any more.
7. Leave a short [report](../observing/report.md) in the context: what is new, what is critical, what was started and how it ended; it lists everything that was started, so that runs without a report of their own are covered too.

## Result
On `main` lie the new reports of the watches, the new findings, the recorded decisions and today's briefing; every story that may start without a human has had a Builder or an Analyst, also the stories that became free during the run; and the report of the run says what is new, what is critical, what was started and how it ended.

## Participants
The [Housekeeper](../team/housekeeper.md).
The [Observer](../team/observer.md), the [Triager](../team/triager.md), the [Project manager](../team/project-manager.md), the [Briefer](../team/briefer.md), [Builders](../team/builder.md) and [Analysts](../team/analyst.md), each started fresh.

## Notes
The circle names the stations in the order a finding meets them. Every role keeps its checkpoint; when a run breaks off, the next one catches up. At a deviation, the Housekeeper asks nobody ([agents that do not ask](../deviations/agents-that-do-not-ask.md)).
