---
name: markitect-create-observer
description: Adds a new observer to a Markitect project together with a person, asking for its one topic, name, source, thresholds, exclusions, checkpoint and schedule, writing its document and doing a trial run. Started by a person, usually through Norman, on a new source, a Risk decided to be observed, or a finding of type Factory.
---

# Create observer

You add a new observer, with a person, in a session of your own that the person switches to, usually started by Norman; you have no model of your own and take that of the session you run in. You start on a new source, on a Risk about which "observe" was decided, which reaches you as the story of the person who does it, or on a finding of type Factory. Your one task is to work out the observer with the person, write its document, create its folder and show the person a first report before the observer counts. When a report only misses something, the existing observer is improved and no new one is added; which observers a project needs, its operation shows.

## Before you start

1. Find the project: it is the folder that holds `.markitect/`. The file `.markitect/spec` holds the path of the spec folder, below called SPEC; never assume `docs/spec`. The observers are the documents in `SPEC/factory/observers/`, listed in its README; their type is `SPEC/packages/factory/observer.md`. Run `markitect` in the project folder, or name it with `-C PROJECT`; a document you name to it is a path below SPEC, with or without `.md`. The tool makes no Git step: you make every one.
2. Whoever started you tells you: who started you, usually `Norman`; for whom, the person; and the item, the story you work on, where there is one: a story in a feature `housekeeping`, `in progress` and claimed by the person.
3. Fetch `main`. Tell the person first what is critical: every finding with importance `critical` in `.markitect/findings/pending/` and `ready/`, and every story of priority `critical` that is not `done`. Read the existing observers, each one's Observes, so that you recognize when one of them covers the topic already.
4. A new observer changes the way of working, which usually an Admin does. Read the person's roles from their document in `SPEC/factory/people/`. When the person is no Admin, that is a deviation of level `high`, since it touches the Area of class Factory, which others build on: say so in one sentence, ask for the reason, carry on after the answer, and drop a note with the skill `capture`, which you run inside this session: who, what, which role would be usual, the level, and the reason in the person's words.

## Steps

1. Ask the person, one question after another, and wait for each answer:

   | Question | What for |
   |---|---|
   | Which one topic, in one sentence? | when an existing observer covers it, that one is extended instead |
   | What is it called, in English and speaking? | when no name can be found, the topic is not clean yet |
   | Where does the source lie, and with which tool is it read? | inside the repository or outside; which access the person who starts the Housekeeper needs |
   | What is a statement worth, what is noise? | the thresholds |
   | What must never go into the report? | customer data, secrets; what an excerpt looks like without them |
   | How does it know how far it has observed? | a commit, a run or a file |
   | How often? | the schedule |

   One observer watches one topic: a second topic is a second observer. When an existing observer covers the topic, stop here: that one is extended instead, which changes the way of working like any other change, through a finding of type Factory; drop it with `capture`. When no name can be found, the topic is not clean yet: work on the topic with the person, or stop.
2. Write the observer's document, `SPEC/factory/observers/NAME.md`, NAME being the speaking English name in lowercase words joined by hyphens: create it with `markitect new packages/factory/observer factory/observers/NAME NAME` and fill its sections, each value on the first line below its heading:
   - Observes: the one topic, in one sentence;
   - Source: where the source lies and with which tool it is read, inside the repository or outside it; the access the person who starts the Housekeeper needs; and what marks how far a run has observed, a commit, a run or a file, where the next run begins;
   - Worth a statement: what the observer states and from which threshold, and what is noise and stays out; the thresholds are starting values, changed through findings;
   - Never in the report: what must never stand in a report, and how an excerpt looks without it; customer data and secrets never do, and where an excerpt cannot be shown without them, the report holds only the reference;
   - Schedule: `daily`, `weekly`, or every so many days;
   - Model: the model and the effort it runs with, as the person chooses, such as the ones the other observers name.
   An observer selects and states, with numbers and places; it never classifies, interprets, fixes or decides: write nothing into its document that asks for that. Link the document from the list in `SPEC/factory/observers/README.md`, with its topic in a few words. Run `markitect check` and correct what you wrote until `check` reports nothing there.
3. Create its folder `.markitect/context/observe-NAME/` for its reports and its checkpoint.
4. Do a trial run: start a fresh subagent that runs exactly the skill `observe`, with the model and effort the new observer's document names under Model; give it the new observer's document as its item, the person as who it works for, and that you started it. Tell it that this is a trial run: it writes its report and checkpoint into your working folder only. Show the person the first report, or that the run observed nothing, before the observer counts. When the person is not content, change the document with them and run again.
5. Once the person is content with the trial run, bring the observer's document, the README line, the folder and the trial run's report and checkpoint to `main` together, in one commit, and push; until then all of them stay off `main`, and when the person drops the observer, they are dropped with it. When the push is refused, fetch, put your commit on top and push again. From then on the Housekeeper runs it when it is due. When you worked on a story, set it `done` and release its claim by writing `none` into its Claimed by, on `main`.
6. Whatever strikes you outside your task, drop with `capture`, one drop per observation: what, where, and what you were doing.

## What you never do

- Add a second topic to an observer, or a new observer for what an existing one covers.
- Let an observer count before the person has seen its first report.
- Write customer data or secrets into the observer's document or its excerpts.

## How you end

1. Lay one short report into `.markitect/context/create-observer/`, named `YYYY-MM-DD-HHMM-ITEM.md` with the time you started and the story's file name without `.md`, or `YYYY-MM-DD-HHMM.md` without a story. It is a document of the language: a `# ` title; after a blank line the type line `Type: [Report](LINK)`, where LINK is the relative path from the report to `SPEC/packages/factory/report.md`; then the sections, each value on the first line below its `## ` heading: Started by, who started you, usually `Norman`; For, the person as a relative link to their document in `SPEC/factory/people/`; Started and Finished, `YYYY-MM-DD HH:MM`; Outcome, a word or two such as `observer added` or `dropped`; Item, the story as a relative link, where there is one; and Statements, where you have any, bundled by theme under `###` headings and numbered throughout. A report is information: it demands nothing and stays unchanged once written.
2. Run `markitect check` and correct the form of what you wrote until `check` reports nothing in your files.
3. Commit the report on `main`, with the story's new state where there is one, and push; when the push is refused, fetch, put your commit on top and push again.
4. Tell the person whether the observer counts from now on, and send them back to Norman.

## Built from

- `agents/create-observer.md`
- `agents/README.md`
- `agents/observe.md`
- `observers/README.md`
- `rules/deviations/usual-way.md`
- `rules/deviations/levels.md`
- `rules/deviations/course.md`
- `rules/principles/one-task.md`
- `rules/principles/speaking-names.md`
- `rules/principles/improves-itself.md`
- `rules/work/context-per-agent.md`
- `rules/work/drop-what-you-notice.md`
- `rules/work/one-per-session.md`
- `rules/work/names.md`
- `rules/work/work-folder.md`
- `rules/plan/commitments.md`
- `rules/plan/claiming.md`
- `rules/briefing/when-shown.md`
- `../packages/factory/report.md`
- `rules/work/straight-to-main.md`
- `agents/norman.md`
- `agents/record.md`
- `../packages/factory/observer.md`
