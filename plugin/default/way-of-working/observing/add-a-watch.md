# Add a watch

Type: [Activity](../../core/activity.md)

## Trigger
There is a topic worth watching again and again in one source, which no watch covers yet: one watch, one topic. A finding of type Process leads to a story for it, in the plan of this module, and it is planned and built like everything else. When a report only misses something, the existing watch is improved instead, and when an existing watch covers the topic, it is extended, both through a story as well. Which watches a project needs, its operation shows.

## Steps
1. While planning the story, the [Architect](../team/architect.md) asks the person, one after another, and writes the answers into the watch's document:

   | Question | What for |
   |---|---|
   | Which one topic, in one sentence? | when an existing watch covers it, that one is extended |
   | What is it called, in English and speaking? | when no name can be found, the topic is not clean yet |
   | Where does the source lie, and with which tool is it read? | inside the repository or outside; which access the person who starts the Housekeeper needs |
   | What is a statement worth, what is noise? | the thresholds |
   | What must never go into the report? | customer data, secrets; what an excerpt looks like |
   | How does it know how far it has observed? | a commit, a run or a file |
   | How often? | its Trigger, a rhythm |

2. From those answers the Architect writes, while planning, the watch's document in this module, of type [Watch](watch.md), and its line in this module's [README](README.md); both go to `main` with the planning ([the model runs ahead](../plan/the-model-runs-ahead.md)), and the story names both under Realizes.
3. The [Builder](../team/builder.md) does a trial run of the watch, as [observe](observe.md) says, in the working folder of the story's branch: as part of building, the trial's checkpoint, and its report when it observed something, lie on that branch and reach `main` with the story's merge; the Builder's own report says what the trial read ([straight to main](../work/straight-to-main.md)).
4. The [Reviewer](../team/reviewer.md) of the build confirms the watch's document and the README as [one trace](../review/one-trace.md) says for a document that no file realizes.
5. The story's last task names the trial's outcome and where it lies, so the story ends in `acceptance`, never directly in `done`, and the person accepts what that task names ([accepting](../accepting/accepting.md)).

## Result
The watch stands in this module, has run once on trial, and the person has accepted what the trial left; from then on the daily run starts it when it is due. When the person drops the watch instead, they end the acceptance as failed; what removes it from `main`, where the daily run would keep starting it, is the decision to give the story up and a story that removes its document ([reopen after a finding](../planning/reopen-after-finding.md)).

## Participants
The [Architect](../team/architect.md) while planning, the [Builder](../team/builder.md) while building.
