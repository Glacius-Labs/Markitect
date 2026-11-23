---
name: capture
description: Drops material unchanged into the context of a Markitect project, such as a transcript a person hands over, a deviation, a merge conflict or anything an agent noticed outside its task. Every agent runs it inside its own session, at any time.
---

# Capture

You drop material unchanged into the context of the project. Capture is the skill every agent has: you run inside the session that calls you, with its model, at any time. Your one task is to put what a person says or hands over, or what the calling agent noticed outside its task, into `.markitect/context/capture/` without changing it, confirm that in one sentence, and let the calling session carry on.

## Before you start

1. Find the project: it is the folder that holds `.markitect/`. You need nothing from the spec for a drop.
2. Make sure you know what only the calling session knows right now: who said or noticed it, when, and what was being worked on. You write it into the drop.

## Steps

1. Take what the person says or hands over, such as a transcript, or what the calling agent noticed outside its task: a smell, a bug, a risk, a contradiction between documents, a merge conflict, a workaround it took, a step of the way of working that got in its way. Make one drop per observation; several observations are several drops.
2. Raw material that lies nowhere else, such as transcripts, conversations and messages, you keep unchanged and hidden in `.markitect/context/capture/.raw/`, one file per piece, named like the drop that points to it. Do not copy what lies elsewhere, such as logs or monitoring: point to it.
3. Write the drop into `.markitect/context/capture/`, named `YYYY-MM-DD-HHMM-SUBJECT.md`: the date and time now, and a short English subject in lowercase words joined by hyphens. A drop has a `# ` title and no type line. Below the title write:
   - the wording, unchanged: the person's words, or what the agent noticed in its own words, with where it is;
   - who said or noticed it, and when;
   - what was being worked on;
   - the pointer to the raw material in `.raw/`, or to where the material lies, where there is one.
4. For a deviation, write: who, what, which role or agent would usually do it, the level (`low`, `normal`, `high` or `critical`), and the reason in the person's words; for an agent that asks nobody, the reason it wrote down itself.
5. For a merge conflict, write which files, and which two stories.
6. Never take customer data or secrets into the repository: doing so is a `critical` deviation. Where something cannot be shown without them, keep only the pointer to where it lies.
7. Write straight to `main`: commit the drop and its raw material, and nothing else, on `main`, and push. When the calling session works on another branch, bring only these files to `main` and leave that branch as it is. When the push is refused because `main` moved, fetch, put your commit on top of the new `main` and push again.
8. Confirm in one sentence that it is dropped, and carry on with the calling session's task. When what you dropped obviously hurries, say so in that sentence: whoever dropped it may run the Housekeeper again at once.

## What you never do

- Investigate, fix or classify what you dropped: no type, no importance, no finding. The observer dropped-material makes the report from it and the Triager the finding.
- Change or smooth the wording of raw material, or of what a person said.
- Copy what already lies elsewhere.
- Take customer data or secrets into the repository.
- Commit anything else along with the drop.

## How you end

The drop and its raw material lie on `main` in `.markitect/context/capture/`, and the calling session has your one sentence. You write no report of your own: you are part of the calling session, and its report covers it.

## Built from

- `agents/capture.md`
- `agents/README.md`
- `rules/work/drop-what-you-notice.md`
- `rules/work/context-per-agent.md`
- `rules/work/straight-to-main.md`
- `rules/work/work-folder.md`
- `rules/deviations/course.md`
- `rules/deviations/critical.md`
- `rules/building/merge-conflicts.md`
- `rules/briefing/when-shown.md`
- `processes/circle/observe.md`
