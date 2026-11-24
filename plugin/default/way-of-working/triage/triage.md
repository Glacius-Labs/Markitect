# Triage

Type: [Activity](../../core/activity.md)

The new reports are sifted: related statements become one finding per topic, and each finding is classified and sent on its way.

## Participants
[Triager](../team/triager.md)

## Trigger
Reports came into the context since the last triage. The Housekeeper starts it in the [daily run](../day/daily-run.md).

## Steps
1. Read the reports that came into the context since the checkpoint, which says up to which state of `main` was sifted ([checkpoints](../work/checkpoints.md)). Look into a report's source only when something is unclear.
2. When a report leaves out something relevant, note that the watch is to be improved; the triage does not make up for it.
3. Bundle related statements into one finding per topic, and a recurring finding into one against the concept a level higher ([recurring findings](recurring-findings.md)). A report without anything noteworthy gives no finding ([what is no finding](not-a-finding.md)).
4. Hold the statements aimed at the model against the model; where the artifacts deviate from a document, look first at the stories that name it, which `markitect show` lists. Send each finding through the exit that fits ([the three exits](three-exits.md)). For the second exit, write the place it follows from into the finding's context.
5. Write each [finding](finding.md): what was noticed, without a solution; where it comes from; in which context; its kind ([the types of findings](finding-types.md)); its importance with the reason; its module, and, when the module is unclear, the model as a whole; for a Bug and a Risk, likelihood and impact.
6. Treat a deviation as material like any other: its importance is its level, and it is filed as a Smell that references the smell story for this person and this kind of deviation ([the course of a deviation](../deviations/course.md)). When its reasons are good and pile up, it is a finding of type Process. A `critical` deviation gives two findings: the Smell, filed at `high`, which asks why it could happen, and the damage itself, a Risk filed `pending` at `critical`, which asks what is to be done now. Where a person said yes to the deviation before it happened, that was their decision: only the Smell is filed. The smell story of an agent that deviated with no person behind it is the one for its role.
7. Set each finding to the state its kind says, `pending` or `ready`, except the second exit and a finding that was decided where it arose, which are `ready` ([the types of findings](finding-types.md)). For a finding of importance `critical`, send the message that [when shown](../briefing/when-shown.md) asks for. Close only noise and duplicates, each with its outcome.
8. Write the findings and the checkpoint in one step. When the run breaks off, the checkpoint does not move.
9. Leave a [report](../observing/report.md) of the run. It names in one line each statement that gave no finding, and why, so that whoever reported it can find out.

## Result
Every new report is sifted. Related statements became one finding per topic, and each finding has its kind, its importance and its module and is `pending`, `ready` or `closed`. The checkpoint moved, and the report of the run lies in the context.
