# Project rules

Read README.md, BACKLOG.md, STATIONS.json and QUALITY.md first. All requirements are
visible from the start. `.study/station.json` says which wave of backlog items is
released right now. Implement only the released items and what they depend on; do not
start later waves. When the released work is merged and checked, stop and report.

## How to work

- Choose the design, the split into tasks, the tools and how you review yourself.
- Work on feature branches. Keep existing behavior and stored data working.
- Add meaningful tests for the risks of each change. Run them and fix what fails.
- A failing command, an error or a question the project files can answer is not a
  reason to stop: find the cause, fix it and rerun the affected checks.
- Keep code, tests and documentation consistent with each other.
- Run `git diff --cached --check` right before every commit.
- Merge checked work into `main` and verify the merged result on `main`.
- Keep the work of other agents and check every conflict resolution.

## Work log

Keep `WORKLOG.md` up to date: items, assumptions, branches and commits, the checks you
ran and their results, failures and fixes, reviews, what is merged, what is open and the
next step. After a break or a context change, compare it with the real Git state, files
and test results before you continue.

## Teamwork in wave S3

S3 must be done by real parallel teamwork: at least two agents work on independent
groups of items, their work overlaps in time, and both contributions are merged.
Planning a team or asking for a helper is not enough. Record in `TEAMWORK.md` who did
what (agent identity, model and effort if known), start and end times, commits and merge
commits, reviews and conflicts. If you could not observe something, say so.

## Reporting

At the end of each wave, say plainly what is done, what is still open and what blocked
you. Leave unfinished changes in place and describe them honestly. Do not report checks
you did not run. Passing checks are evidence, not acceptance by the project owner.

## Boundaries

Work only inside this repository. Never push or publish anything, never change global
settings, and do not look at credentials. Use the configured model and settings for
yourself and for any helpers.
