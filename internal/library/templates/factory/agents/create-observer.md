# Create observer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Explain how a new observer is built.

It runs inside the session that calls it: the [Architect](architect.md) when it plans a story for a new observer, the [Builder](build.md) when it builds one. An observer is planned and built like everything else, through a story in the Area that covers the way of working.

1. A new observer is right when there is a topic worth watching again and again in one source, which no observer covers yet: one observer, one topic. When a report only misses something, the existing observer is improved instead, and when an existing observer covers the topic, it is extended: like every change to the spec, through a story, which a finding of type Factory leads to. Which observers a project needs, its operation shows.
2. While planning, the Architect asks the person, one after another, and writes the answers into the story's target picture:

   | Question | What for |
   |---|---|
   | Which one topic, in one sentence? | when an existing observer covers it, that one is extended |
   | What is it called, in English and speaking? | when no name can be found, the topic is not clean yet |
   | Where does the source lie, and with which tool is it read? | inside the repository or outside; which access the person who starts the Housekeeper needs |
   | What is a statement worth, what is noise? | the thresholds |
   | What must never go into the report? | customer data, secrets; what an excerpt looks like |
   | How does it know how far it has observed? | a commit, a run or a file |
   | How often? | the schedule |

3. From those answers the Architect writes, while planning and on the story's branch as every change to the spec, the observer's document among the [observers](../observers/README.md), of type [Observer](../../../packages/factory/observer.md), and its line in the observers' README; the story's Target changes name both ([spec change](../rules/building/spec-change-lands-with-realization.md)). The Reviewer of the planning's check reads them and confirms nothing.
4. The Builder creates the folder `.markitect/context/observe-NAME/` and does a trial run of the observer with [observe](observe.md) in the working folder of the story's branch, where the document lies: as part of building, the trial's checkpoint, and its report when it observed something, lie on that branch and reach `main` with the story's merge; the Builder's own report says what the trial read ([straight to main](../rules/work/straight-to-main.md)).
5. The Reviewer of the build confirms the observer's document and the README as [one trace](../rules/building/one-trace.md) says for a document no file realizes.
6. The story's last task names the trial's outcome and where it lies, the observer's folder `.markitect/context/observe-NAME/` with its checkpoint and any report, so the story ends in `acceptance`, never directly in `done`, and the person accepts with [accept](accept.md) what that task names. When they drop the observer, they end the acceptance as failed ([accepting](../processes/accepting/review.md)); what removes it from `main`, where the Housekeeper would keep running it, is the decision to give the story up and a story that removes its document ([reopen after a finding](../rules/planning/reopen-after-finding.md)).

## Started by
The [Architect](architect.md) and the [Builder](build.md), inside their sessions.

## Asks
no

## Model
as the session

## Changes
Nothing of its own: the session that calls it writes.
