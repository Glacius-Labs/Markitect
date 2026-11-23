---
name: markitect-create-observer
description: Explains how a new observer of a Markitect project is planned and built through its story, which questions the target picture answers, what the Architect writes on the story's branch, how the Builder runs the trial and how its report is accepted. Runs inside the Architect's session when it plans such a story, and inside the Builder's when it builds one.
---

# Create observer

You explain how a new observer is built. You run inside the session that calls you, with its model: the Architect's, when it plans a story for a new observer, or the Builder's, when it builds one. You are no agent of your own: you claim nothing, start no Reviewer, merge nothing and write no report; the session that calls you does all of that as its own skill says. An observer is planned and built like everything else, through a story in the Area that covers the way of working.

## Before you start

1. Find the project: it is the folder that holds `.markitect/`. The file `.markitect/spec` holds the path of the spec folder, below called SPEC; never assume `docs/spec`. The observers are the documents in `SPEC/factory/observers/`, listed in its README; their type is `SPEC/packages/factory/observer.md`. Run `markitect` in the project folder, or name it with `-C PROJECT`; a document you name to it is a path below SPEC, with or without `.md`.
2. Read the existing observers, each one's Observes, so that you recognize when one of them covers the topic already.

## Steps

1. Make sure a new observer is right. It is right when there is a topic worth watching again and again in one source, which no observer covers yet: one observer, one topic. When a report only misses something, the existing observer is improved instead, and when an existing observer covers the topic, it is extended: like every change to the spec, through a story, which a finding of type Factory leads to. Which observers a project needs, its operation shows.
2. While planning, the Architect asks the person, one question after another, and writes the answers into the story's target picture:

   | Question | What for |
   |---|---|
   | Which one topic, in one sentence? | when an existing observer covers it, that one is extended |
   | What is it called, in English and speaking? | when no name can be found, the topic is not clean yet |
   | Where does the source lie, and with which tool is it read? | inside the repository or outside; which access the person who starts the Housekeeper needs |
   | What is a statement worth, what is noise? | the thresholds |
   | What must never go into the report? | customer data, secrets; what an excerpt looks like |
   | How does it know how far it has observed? | a commit, a run or a file |
   | How often? | the schedule |

3. From those answers the Architect writes, while planning and on the story's branch, as every change to the spec, the observer's document among the observers, `SPEC/factory/observers/NAME.md`, NAME being its speaking English name in lowercase words joined by hyphens, of type Observer, with its Observes, Source, Worth a statement, Never in the report, Schedule and Model, each value on the first line below its heading; and its line in the observers' README, `SPEC/factory/observers/README.md`. The story's Target changes name both. The Reviewer of the planning's check reads them and confirms nothing.
4. The Builder creates the folder `.markitect/context/observe-NAME/` and does a trial run of the observer in the working folder of the story's branch, where the document lies: a fresh subagent that runs exactly the skill `markitect-observe` for the new observer, with the model and effort the observer's document names under Model on the line of the provider it runs on, told that this is a trial run on the story's branch. As part of building, the trial's checkpoint, and its report when it observed something, lie on that branch and reach `main` with the story's merge; the Builder's own report says what the trial read.
5. The Reviewer of the build confirms the observer's document and the README as a document no file realizes: `skip`, with `-m` giving that reason.
6. The story's last task names the trial's outcome and where it lies, the observer's folder `.markitect/context/observe-NAME/` with its checkpoint and any report, so the story ends in `acceptance`, never directly in `done`, and the person accepts with the skill `markitect-accept` what that task names. When they drop the observer, they end the acceptance as failed. That does not remove the observer from `main`, where the Housekeeper would keep running it: what removes it is the decision to give the story up and a story that removes its document.

## What you never do

- Add a second topic to an observer, or a new observer for what an existing one covers.
- Act as an agent of your own: claim, review, merge or report; the session that calls you does that.
- Let the story go to `done` without the person accepting what its last task names.

## How you end

You change nothing of your own: the session that calls you writes, and goes on with its own skill.

## Built from

- `agents/create-observer.md`
- `agents/README.md`
- `agents/architect.md`
- `agents/build.md`
- `agents/observe.md`
- `agents/accept.md`
- `observers/README.md`
- `rules/building/one-trace.md`
- `processes/accepting/review.md`
- `rules/planning/reopen-after-finding.md`
- `rules/building/spec-change-lands-with-realization.md`
- `rules/work/straight-to-main.md`
- `rules/principles/one-task.md`
- `rules/principles/speaking-names.md`
- `rules/work/context-per-agent.md`
- `rules/work/names.md`
- `../packages/factory/observer.md`
