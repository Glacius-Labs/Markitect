---
name: markitect-cleanup
description: Clean up the code without changing what the project does. The markitect-builder agent refactors in a worktree of its own, and the markitect-reviewer agent verifies again what that put back on the list. The work comes back to this branch when it leaves nothing worse. Use when the user asks to clean up or refactor a Markitect project.
disable-model-invocation: true
allowed-tools: Bash(markitect status), Bash(markitect status *)
---
Run `markitect status` and keep the Definitions it lists: the work must not add to them. Then start the markitect-builder agent with the message "Clean up.", and add what the user asked to focus on, if anything. Change no file and record no verdict yourself.

In a Git repository, start it in a worktree of its own (the Agent tool's worktree isolation), so that the user can go on working here. The worktree starts from the last commit of this branch, so it holds no uncommitted change: if `git status` shows any, tell the user they are not part of the cleanup and ask whether to commit them first. In a folder that is not a Git repository, start it here and ask the user to leave the folder alone until it ends.

The builder refactors, starts the reviewer to verify what that put back on the list, and repeats until nothing it can settle is left, or the list has not got shorter for three rounds. Wait for it. It ends by naming its folder and branch.

When it has ended, run `markitect status --repo <its folder>`. Commit everything in its folder in one commit on its branch, `.markitect/review.sum` included. Then:

- **If status there lists no Definition that was not listed here before,** merge its branch into the current branch here, then remove its worktree and branch. If Git refuses, because of uncommitted changes in the way or a conflict, abort the merge, leave the branch as it is and tell the user.
- **Otherwise,** leave the work on its branch and tell the user what the cleanup put on the list, and ask whether to merge it as it stands.

Push nothing. In a folder that is not a Git repository, commit nothing and say that the work is ready to commit.

Then tell the user in a line or two what was cleaned up, whether it was merged, and what status still lists.
