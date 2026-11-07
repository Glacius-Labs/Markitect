---
name: markitect-schedule
description: Set up Markitect routines that run on their own through Claude Code's scheduling - a review of every Definition, a bug hunt, or a cleanup. Each run works on a branch of its own and lands on the source branch only when it leaves nothing worse, then reports. Use when the user asks to schedule, automate or set up routines for Markitect work.
disable-model-invocation: true
---
Markitect schedules nothing itself: Claude Code's own scheduling runs the routines, and this skill sets them up with the user. Each run is a session of its own with nobody to answer it.

1. **Which routines, and when.** Ask the user, and propose:
   - **review**, nightly: every Definition verified again, the safety net for what `markitect status` cannot see, as `/markitect-review --all` does;
   - **bughunt**, weekly: a reviewer hunts beyond the list, the builder repairs, the reviewer verifies again, as `/markitect-bughunt` does;
   - **cleanup**, weekly: the builder refactors without changing behaviour and the reviewer verifies again, as `/markitect-cleanup` does.
2. **Where it lands.** Settle the source branch, by default the repository's default branch. A run works on a branch of its own and lands on the source branch only when it leaves nothing worse: `markitect status` lists nothing new, and the project's tests pass. It lands by a push to the remote when the source branch tracks one, otherwise on the local branch, which Git refuses while that branch is checked out in a folder; then the run leaves its branch and says how to land it. Tell the user this and ask them to confirm, since a routine then pushes without asking.
3. **The prompts.** For each routine, take [routine.md](routine.md) and replace `{{NAME}}` with its name, `{{SOURCE}}` with the source branch, `{{FOLDER}}` with this project's main folder (the parent of `git rev-parse --path-format=absolute --git-common-dir`), and `{{WORK}}` with the text of its own file: [review.md](review.md), [bughunt.md](bughunt.md) or [cleanup.md](cleanup.md).
4. **Schedule them** with what this Claude Code offers:
   - **In the desktop app:** a scheduled task for each, created with the scheduled-task tools, from a session in this project, so that the run gets this project's agents and hooks. It runs on this machine while the app is open, with this machine's `markitect`.
   - **Otherwise:** a cloud routine with `/schedule`. It runs on a fresh clone of the repository from GitHub, so its environment must install `markitect`, for example by downloading the binary of the newest release from https://github.com/Glacius-Labs/Markitect/releases, and it lands only by a push.
   - Not `/loop` or a session's own schedule: they end with the session.

   A run that stops to ask for a permission waits until someone answers. Say so, and suggest a permission mode for the task that does not ask, or rules that allow `markitect`, Git and the project's test command.
5. **Report** in a few lines what you set up: each routine's name, schedule, source branch and where it lands, and how to pause or remove it.
