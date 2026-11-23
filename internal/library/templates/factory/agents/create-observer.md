# Create observer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Add a new observer, with a person.

It works on a story, like every change to the spec: a story for a new observer, which the Recorder writes for a Risk decided "observe", for a new source or for a finding of type Factory. Claim the story first ([claiming](../rules/plan/claiming.md)) and work on its branch ([names in the work](../rules/work/names.md)). When a report only misses something, the existing observer is improved and no new one is added. Which observers a project needs, its operation shows.

1. Ask, one after another:

   | Question | What for |
   |---|---|
   | Which one topic, in one sentence? | when an existing observer covers it, that one is extended |
   | What is it called, in English and speaking? | when no name can be found, the topic is not clean yet |
   | Where does the source lie, and with which tool is it read? | inside the repository or outside; which access the person who starts the Housekeeper needs |
   | What is a statement worth, what is noise? | the thresholds |
   | What must never go into the report? | customer data, secrets; what an excerpt looks like |
   | How does it know how far it has observed? | a commit, a run or a file |
   | How often? | the schedule |

2. Write the observer's document among the [observers](../observers/README.md) on the story's branch, of type [Observer](../../../packages/factory/observer.md), link it from that folder's README, and name both in the story's Target changes.
3. Create its folder `.markitect/context/observe-NAME/` for its reports and its checkpoint.
4. Do a trial run on the branch: show the person the first report before the observer counts.
5. Once the person is content, have a fresh [Reviewer](review.md) check the change to the spec, in rounds as in [building](../processes/building/review-round.md), and bring the branch to `main` as one merge whose commit names the story; the Reviewer confirms the places the story names. Set the story `done` and release its claim. When the person drops the observer, the story goes back to `draft` with your report, as after an [abort](../rules/building/abort.md).
6. Lay your report into the context at the end ([one per session](../rules/work/one-per-session.md)).

A new observer changes the way of working: usually an Admin does it ([the usual way](../rules/deviations/usual-way.md)).

## Started by
A person, usually through [Norman](norman.md).

## Asks
yes

## Model
as the session

## Changes
A new observer document in the spec, and the observers' README, on the story's branch and through a Reviewer on `main`.
The folder `.markitect/context/observe-NAME/`.
Its report in `.markitect/context/create-observer/`.
The State and Claimed by of the story it worked on.
