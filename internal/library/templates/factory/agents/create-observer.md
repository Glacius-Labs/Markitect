# Create observer

Type: [Agent](../../../packages/factory/agent.md)

## Task
Add a new observer, with a person, on its story.

An observer is added like every change to the spec, through a story: the Recorder writes it, in `draft`, when "observe" was decided for a Risk, a new source or a finding of type Factory ([Recorder](record.md)), and Norman proposes it to the person. When a report only misses something, the existing observer is improved and no new one is added. Which observers a project needs, its operation shows. This agent plans and realizes the story in one session, through the states the [milestones](../rules/plan/milestones.md) name for it.

1. Claim the story ([claiming](../rules/plan/claiming.md)) and work on its branch ([names in the work](../rules/work/names.md)).
2. Ask, one after another:

   | Question | What for |
   |---|---|
   | Which one topic, in one sentence? | when an existing observer covers it, that one is extended |
   | What is it called, in English and speaking? | when no name can be found, the topic is not clean yet |
   | Where does the source lie, and with which tool is it read? | inside the repository or outside; which access the person who starts the Housekeeper needs |
   | What is a statement worth, what is noise? | the thresholds |
   | What must never go into the report? | customer data, secrets; what an excerpt looks like |
   | How does it know how far it has observed? | a commit, a run or a file |
   | How often? | the schedule |

3. Complete the story: its target picture, its Target changes, the observer's document and the observers' README, its Plan and Tasks; then set it as the milestones say.
4. Write the observer's document among the [observers](../observers/README.md) on the story's branch, of type [Observer](../../../packages/factory/observer.md), and link it from that folder's README. Create its folder `.markitect/context/observe-NAME/` for its reports and its checkpoint.
5. Do a trial run on the branch: show the person the first report before the observer counts.
6. Once the person is content, run `markitect check` and have a fresh [Reviewer](review.md) check the change to the spec, in rounds as in [building](../processes/building/review-round.md); it confirms the observer's document and the README as [one trace](../rules/building/one-trace.md) says for a document no file realizes. Bring the branch to `main` as one merge whose commit names the story, set the story `done` and release its claim.
7. When the person drops the observer, the story goes back to `draft` with your report, as after an [abort](../rules/building/abort.md).
8. Lay your report into the context at the end ([one per session](../rules/work/one-per-session.md)).

A new observer changes the way of working: usually an Admin does it ([the usual way](../rules/deviations/usual-way.md)).

## Started by
[Norman](norman.md), for the person who takes the story.

## Asks
yes

## Model
Claude: Sonnet, effort high
Codex: Luna, effort high

## Changes
The story's sections, State and Claimed by.
A new observer document in the spec, and the observers' README, on the story's branch and through a Reviewer on `main`.
The folder `.markitect/context/observe-NAME/`, with the trial run's report and checkpoint.
Its report in `.markitect/context/create-observer/`.
