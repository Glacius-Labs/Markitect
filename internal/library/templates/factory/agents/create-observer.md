# Create observer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Add a new observer, with a person.

It starts on a new source, on a Risk about which "observe" was decided, which reaches it as the story of the person who does it ([commitments](../rules/plan/commitments.md)), or on a finding of type Factory. When a report only misses something, the existing observer is improved and no new one is added. Which observers a project needs, its operation shows.

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

2. Write the observer's document among the [observers](../observers/README.md), of type [Observer](../../../packages/factory/observer.md), and link it from that folder's README.
3. Create its folder `.markitect/context/observe-NAME/` for its reports and its checkpoint.
4. Do a trial run: show the person the first report before the observer counts. The trial run writes its report and checkpoint into your working folder only.
5. Once the person is content with the trial run, bring the observer's document, the README line, the folder and the trial run's report and checkpoint to `main` together; until then all of them stay off `main`, and when the person drops the observer, they are dropped with it. When you worked on a story, set it `done` and release its claim.
6. Lay your report into the context at the end ([one per session](../rules/work/one-per-session.md)).

A new observer changes the way of working: usually an Admin does it ([the usual way](../rules/deviations/usual-way.md)).

## Started by
A person, usually through [Norman](norman.md).

## Asks
yes

## Model
as the session

## Changes
A new observer document in the spec, and the observers' README.
The folder `.markitect/context/observe-NAME/`.
Its report in `.markitect/context/create-observer/`.
The State and Claimed by of the story it worked on.
