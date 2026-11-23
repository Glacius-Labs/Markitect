---
name: markitect-finish-planning
description: Closes a planning session of a Markitect project after the person's go, has the plan checked by a fresh Reviewer, lays the report, sets the states, brings the plan to main and commits and pushes the changed documents on the stories' branches. Runs inside the Architect's planning session as its last step.
---

# Finish planning

You close a planning. You run at the end of a planning session, inside that session, as its step Finish, after the person has given their go; you know what the planning worked out because you are that session. Your one task is to have the plan checked, report on it, set the states, and bring the plan to `main`, with every change to the standing spec on the branch of the story that realizes it.

## Before you start

1. Find the project: it is the folder that holds `.markitect/`. The file `.markitect/spec` holds the path of the spec folder, below called SPEC; never assume `docs/spec`. The way of working is the part `SPEC/factory/`, and the types are in `SPEC/packages/factory/`. Run `markitect` in the project folder, or name it with `-C PROJECT`. The tool makes no Git step: you make every one.
2. Have at hand from the planning: the person; the node planned; the plan items written in `.markitect/plan/`; for each story, its Target changes; the changed documents of the spec, which the Architect wrote on each story's branch, in a working folder of that branch, such as a second worktree; and any coupling the cut showed.

## Steps

1. Have the plan and the changed documents of the spec checked: start a fresh subagent that runs exactly the skill `markitect-review`, with the model and effort that `SPEC/factory/agents/review.md` names under Model on the line of the provider you run on, read at that moment. Give it: `finish-planning` as who started it; the person; the item; and what lies before it: the plan of the story, or for a feature its cut into stories, to be checked against the target of its own level, the target picture of that story or feature, and the changed documents of the spec; where the result lies: the plan items in the folder of `main` of this planning session, which reach `main` only at the finish and which it reads as they are, and the changed documents on each story's branch and the folder where it is checked out; and, when this is not the first check, the number of the round and the earlier comments with the answers. When it does not fit, go back to the planning: the Architect works on it from the step where it went wrong, and you run again after the person's next go.
2. Lay the report on the course of the planning into `.markitect/context/architect/` (see How you end), with any coupling the cut showed: stories that could only be cut with many dependencies.
3. Set the states the finish step names, and release the claim of a planned story: a planned story goes from `draft` to `ready`, with `none` written into its Claimed by in the same step as its new state. After a feature was planned, its new stories stay `draft` until each is planned, except stories with fixed tasks, which are `ready` already, and the feature keeps its Owner and `in progress`.
4. Bring the plan to `main` from the folder of `main`: the plan items with their states and claims, the findings the planning closed, and the report, in one commit on `main`; push. When the push is refused, fetch, put your commit on top and push again. Run `markitect check` before and correct the form of what you wrote in `.markitect/` until `check` reports nothing in your files.
5. Then, for each story whose Target changes name a change to the spec, commit the changed documents that story carries on its branch, named like the story's path below `.markitect/plan/` without `.md`, such as `lending/renew-online/renew-a-loan`, in that branch's working folder where the Architect wrote them, never on `main`, and never carrying uncommitted changes from one branch to the other; they replace what an earlier planning put there, and a change the story no longer carries is removed. The Builder treats code already on the branch as a start it may keep or drop within the spec. Run `markitect check` and correct what you wrote until `check` reports nothing there; push the branch. When a change concerns several stories, each carries on its branch the part it realizes. When two stories change the same place, each carries its part; the conflict shows when they are merged. So the spec on `main` is never ahead of a planned story; the Builder continues on that branch, and spec and code go to `main` together.
6. Tell the person that the plan is secured and the chat can be archived, and send them back to the Assistant.
7. Whatever strikes you outside your task, drop with the skill `markitect-capture`, which you run inside this session, one drop per observation: what, where, and what you were doing.

## What you never do

- Put a change to the standing spec onto `main`.
- Set a story `ready` without the Reviewer's check, or leave its claim standing.
- Build, or start a Builder.

## How you end

The report lies in `.markitect/context/architect/`, named `YYYY-MM-DD-HHMM-ITEM.md`, with the time the planning session started and the planned item's name: a story's file name without `.md`, a feature's folder name. It is a document of the language: a `# ` title; after a blank line the type line `Type: [Report](LINK)`, where LINK is the relative path from the report to `SPEC/packages/factory/report.md`; then the sections, each value on the first line below its `## ` heading:

- Started by: `Assistant`, which started the planning session;
- For: the person, as a relative link to their document in `SPEC/factory/people/`;
- Started and Finished: `YYYY-MM-DD HH:MM`;
- Outcome: `planned`;
- Item: the planned item, as a relative link;
- Statements: the course of the planning, the coupling the cut showed, and what was dropped as unclear, bundled by theme under `###` headings and numbered throughout, each one sentence with the reference to its place.

A report is information: it demands nothing and stays unchanged once written. The plan is on `main`, a planned story is `ready` with its claim released, and every change to the standing spec lies on the branch of the story that realizes it. The person goes back to the Assistant.

## Built from

- `agents/finish-planning.md`
- `agents/README.md`
- `agents/architect.md`
- `agents/review.md`
- `processes/planning/finish.md`
- `processes/planning/README.md`
- `rules/building/spec-change-lands-with-realization.md`
- `rules/plan/claiming.md`
- `rules/plan/milestones.md`
- `rules/work/one-per-session.md`
- `rules/work/straight-to-main.md`
- `rules/work/drop-what-you-notice.md`
- `rules/work/work-folder.md`
- `rules/work/names.md`
- `rules/plan/fixed-tasks.md`
- `../packages/factory/report.md`
- `../packages/factory/story.md`
