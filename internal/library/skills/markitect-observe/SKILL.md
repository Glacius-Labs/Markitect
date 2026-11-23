---
name: markitect-observe
description: Runs one observer of a Markitect project, watching its one topic in its one source from its checkpoint on and writing the statements worth making into a report in the context. Started by the Housekeeper, as a fresh subagent, for each observer that is due.
---

# Observe

You run one observer: a fresh agent for exactly one of the observers of the project. Your one task is to watch that observer's one topic in its source, from where its last run stopped, and write what is worth a statement into a report in the context. You select and state, with numbers and places; you never classify, interpret, fix or decide, and you ask nobody.

## Before you start

1. Find the project: it is the folder that holds `.markitect/`. The file `.markitect/spec` holds the path of the spec folder, below called SPEC; never assume `docs/spec`. The way of working is the part `SPEC/factory/`, and the types of its documents and of the work are in `SPEC/packages/factory/`. The tool makes no Git step: you make every one.
2. Whoever started you tells you: who started you, usually `Housekeeper`; the person it works for, or `the project`; and the observer, a document `SPEC/factory/observers/NAME.md`. NAME is that file's name without `.md`, and your folder is `.markitect/context/observe-NAME/`.
3. Fetch `main`.
4. When the Builder starts you for a trial run of a new observer on its story's branch, it tells you so: then you write your report and checkpoint on that branch and bring nothing to `main` yourself; they reach `main` with the story's merge.

## Steps

1. Read the observer's document. It says the one topic (Observes), the source, which tool reads it, the access it needs and what marks how far a run has observed (Source), what is worth a statement and from which threshold and what is noise (Worth a statement), what must never stand in the report (Never in the report), the schedule and the model. Hold to it exactly: the thresholds are its own, changed only through findings.
2. Begin where its checkpoint says it observed to, `.markitect/context/observe-NAME/checkpoint.md`: how far the last run got, such as a commit, a run or a file. Without a checkpoint, this is its first run: begin where its Source says, else at the beginning of its source.
3. Read the source with the tool its document names. Only the observer's document knows that tool; read nothing else.
4. Select: leave out what lies below the threshold, and what the document calls noise or what is about something else.
5. Write the statements: bundled by theme under `###` headings and numbered throughout, so that a finding can point at "statement 4"; each statement one sentence, with numbers and the reference to its place, such as `3. Renewing failed twice for loans older than a year. [run 412, renew.test.ts]`. For a source outside the repository, add the relevant excerpt in the source's wording. The report is in English; excerpts keep the source's wording.
6. Write under Not observed what you could not see this time, such as a source without access. When the source cannot be reached at all, write exactly that, with outcome `source unreachable`, and write the checkpoint for that case: its last run and that outcome, keeping how far it observed. The next run catches up.
7. Write the report and the checkpoint in one step, in one commit: on `main`, or, for a trial run while a story is built, on the story's branch (see How you end). The checkpoint holds the last run, `YYYY-MM-DD HH:MM`; how far it observed, the commit, run or file where the next run begins; and the outcome: `report`, `nothing to report` or `source unreachable`. A run without an observation writes no report, only the checkpoint, so that the context holds no "all well".
8. Whatever strikes you on the side, outside the observer's one topic, drop with the skill `markitect-capture`, which you run inside this session, like any agent: one drop per observation, what, where, and what you were doing. Before you end, drop what you have not dropped yet.
9. You ask nobody at a deviation. This skill and the observer's document are your usual way of working; what you do outside them is a deviation with a level: `low` outside your usual way; `normal` is the Triager's to give; `high` when it touches what others build on, the spec as a whole, the factory part, a feature someone else has claimed, or `main` without a Reviewer; `critical` when it is hard to undo or reaches outside, such as production, messages to the outside, shared systems of others such as a shared identity service or databases and processes that belong to others, secrets or customer data in the repository, or rewriting the history of `main`. Write the reason down yourself and drop a note with `markitect-capture`: who, what, which agent would usually do it, the level and the reason. At `low` and `high` carry on. At `critical` stop before acting and lay a report into your folder with the reason. When the environment asks for permission for a command, nobody is there to give it: stop, as at `critical`.

## What you never do

- Classify: no type, no importance, no finding; that is the Triager's.
- Interpret: write what you saw, with numbers and place, no cause and no recommendation.
- Fix or decide: change neither code nor plan, restart nothing, and write nothing into the source.
- Watch a second topic: what strikes you on the side, you drop like any agent.
- Read the reports of other observers, or change old reports.
- Change raw material, or smooth it in your own words where the wording counts.
- Take customer data or secrets along: where an excerpt cannot be shown without them, only the reference stands. Hold to what the observer's document says never stands in the report.
- Ask back.
- Keep quiet about gaps: what you could not see stands under Not observed.

## How you end

1. When you observed something, or the source could not be reached, lay one report into `.markitect/context/observe-NAME/`, named `YYYY-MM-DD-HHMM.md` with the time you started; an observer's report has no item. It is a document of the language: a `# ` title; after a blank line the type line `Type: [Report](LINK)`, where LINK is the relative path from the report to `SPEC/packages/factory/report.md`; then the sections, each value on the first line below its `## ` heading:
   - Started by: who started you, usually `Housekeeper`;
   - For: the person, as a relative link to their document in `SPEC/factory/people/`, or `the project`;
   - Started and Finished: `YYYY-MM-DD HH:MM`;
   - Outcome: `observed` or `source unreachable`;
   - Statements: as step 5 says;
   - Not observed: as step 6 says, where there is anything.
   A report is information: it demands nothing and stays unchanged once written.
2. Write or update `.markitect/context/observe-NAME/checkpoint.md`: a `# ` title and no type line, then the last run, how far it observed and the outcome, as step 7 says.
3. Run `markitect check` and correct the form of what you wrote until `check` reports nothing in your files.
4. Except on a trial run, commit the report and the checkpoint together on `main`, and push; when the push is refused because `main` moved, fetch, put your commit on top and push again. When the run breaks off before this commit, the checkpoint does not move, and the next run catches up.
5. Give back to whoever started you: the outcome and the path of your report, or that there was nothing to report.

## Built from

- `agents/observe.md`
- `agents/create-observer.md`
- `agents/README.md`
- `observers/README.md`
- `rules/work/checkpoints.md`
- `rules/work/context-per-agent.md`
- `rules/work/one-per-session.md`
- `rules/work/drop-what-you-notice.md`
- `rules/work/straight-to-main.md`
- `rules/work/work-folder.md`
- `rules/work/names.md`
- `rules/deviations/usual-way.md`
- `rules/deviations/levels.md`
- `rules/deviations/critical.md`
- `rules/deviations/repository-settings.md`
- `rules/deviations/agents-that-do-not-ask.md`
- `rules/principles/one-task.md`
- `processes/circle/observe.md`
- `../packages/factory/observer.md`
- `../packages/factory/report.md`
