---
name: markitect-cleanup
description: Clean up the code without changing what the project does. The markitect-builder agent refactors in a worktree of its own, and the markitect-reviewer agent verifies again what that put back on the list; a large project is cleaned up in parallel, per Package. The work comes back to this branch when it leaves nothing worse. Use when the user asks to clean up or refactor a Markitect project.
disable-model-invocation: true
allowed-tools: Bash(markitect status), Bash(markitect status *), Bash(markitect config strategy.cleanup)
---
Run `markitect status` and keep the Definitions it lists: the work must not add to them. Run `markitect config strategy.cleanup` for how to clean up: `sequential`, `parallel`, `auto` or the name of a custom workflow. With `auto`, choose: parallel when this is a Git repository and the spec has more than about 30 Definitions in at least two Packages (`markitect status --json` counts them per Package), otherwise sequential. Parallel and custom workflows need Git: in a folder that is not a Git repository, work sequentially. Tell the user in a line how you work and why. Change no file and record no verdict yourself.

In a Git repository the work starts from the last commit of this branch, so it holds no uncommitted change: if `git status` shows any, tell the user they are not part of the cleanup and ask whether to commit them first.

**Sequential.** Start the markitect-builder agent with the message "Clean up.", and add what the user asked to focus on, if anything. In a Git repository, start it in a worktree of its own (the Agent tool's worktree isolation), and in the background where the Agent tool can, so that the user can go on designing, here or in another session. In a folder that is not a Git repository, start it here and ask the user to leave the folder alone until it ends. The builder refactors, starts the reviewer to verify what that put back on the list, and repeats until nothing it can settle is left, or the list has not got shorter for three rounds. Wait for it to end. It ends by naming its folder and branch.

**Parallel, or a custom workflow.** Note the current branch (`git branch --show-current`): the run comes back to it. Then enter a worktree of your own with the EnterWorktree tool, named `markitect-cleanup` with the date and time: the run works on its branch, and Claude Code branches every builder's worktree from it. Then start the workflow with the Workflow tool, `markitect-parallel` for parallel or the custom workflow by its name, with the args `{"mode": "cleanup", "into": "<the branch you noted>", "focus": "<what the user asked to focus on, or empty>"}`. It has one builder clean up each Package, each in a worktree of its own with its own reviewer, in waves along the spec's dependencies, Root last, and merges each strand into your worktree's branch. Tell the user that it runs, and stay in the worktree until the workflow has ended. Then leave the worktree with ExitWorktree and keep it: its folder and branch hold the work.

**When the work has ended,** run `markitect status --repo <its folder>`. If its worktree is gone, because nothing changed, run `markitect status` here instead, skip the commit and the merge, and report. Commit everything in its folder in one commit on its branch, `.markitect/review.sum` included. Then:

- **If status there lists no Definition that was not listed here before,** merge its branch into the current branch here, then remove its worktree and branch. If Git refuses, because of uncommitted changes in the way or a conflict, abort the merge, leave the branch as it is and tell the user.
- **Otherwise,** leave the work on its branch and tell the user what the cleanup put on the list, and ask whether to merge it as it stands.

Push nothing. In a folder that is not a Git repository, commit nothing and say that the work is ready to commit.

Then tell the user in a line or two what was cleaned up, whether it was merged, and what status still lists. A workflow returns the strands it could not merge, each with its branch: name them.
